package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/logger"
)

// TokenBlacklistService Token 黑名单服务接口
type TokenBlacklistService interface {
	// AddToBlacklist 将 token 加入黑名单
	AddToBlacklist(ctx context.Context, token string, expiresAt time.Time) error
	
	// IsBlacklisted 检查 token 是否在黑名单中
	IsBlacklisted(ctx context.Context, token string) (bool, error)
	
	// RemoveFromBlacklist 从黑名单中移除 token（通常不需要，因为会自动过期）
	RemoveFromBlacklist(ctx context.Context, token string) error
	
	// CleanupExpired 清理已过期的黑名单条目（Redis 会自动处理，此方法用于兼容性）
	CleanupExpired(ctx context.Context) error
}

// tokenBlacklistService Token 黑名单服务实现
type tokenBlacklistService struct {
	redis  *database.RedisClient
	logger logger.Logger
}

// NewTokenBlacklistService 创建新的 Token 黑名单服务
func NewTokenBlacklistService(redis *database.RedisClient, log logger.Logger) TokenBlacklistService {
	return &tokenBlacklistService{
		redis:  redis,
		logger: log,
	}
}

// hashToken 计算 token 的哈希值（用于存储）
func (s *tokenBlacklistService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// getBlacklistKey 获取黑名单键名
func (s *tokenBlacklistService) getBlacklistKey(tokenHash string) string {
	return fmt.Sprintf("token:blacklist:%s", tokenHash)
}

// AddToBlacklist 将 token 加入黑名单
func (s *tokenBlacklistService) AddToBlacklist(ctx context.Context, token string, expiresAt time.Time) error {
	// 如果 Redis 未启用，记录警告但不返回错误
	if !s.redis.IsEnabled() {
		s.logger.Warn("Redis 未启用，无法将 token 加入黑名单")
		return nil
	}

	tokenHash := s.hashToken(token)
	key := s.getBlacklistKey(tokenHash)
	
	// 计算 TTL（token 过期时间 - 当前时间）
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// token 已经过期，不需要加入黑名单
		s.logger.Debug("token 已过期，无需加入黑名单")
		return nil
	}

	// 存储到 Redis，值为撤销时间戳
	if err := s.redis.Set(ctx, key, time.Now().Unix(), ttl); err != nil {
		s.logger.Error("将 token 加入黑名单失败", logger.Fields{"error": err})
		return fmt.Errorf("将 token 加入黑名单失败: %w", err)
	}

	s.logger.Info("token 已加入黑名单", logger.Fields{"ttl": ttl})
	return nil
}

// IsBlacklisted 检查 token 是否在黑名单中
func (s *tokenBlacklistService) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	// 如果 Redis 未启用，返回 false（不阻止访问）
	if !s.redis.IsEnabled() {
		return false, nil
	}

	tokenHash := s.hashToken(token)
	key := s.getBlacklistKey(tokenHash)
	
	// 检查键是否存在
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		s.logger.Error("检查 token 黑名单状态失败", logger.Fields{"error": err})
		// 发生错误时，为了安全起见，假设 token 未被列入黑名单
		// 这样可以避免因 Redis 故障导致所有请求被拒绝
		return false, nil
	}

	return exists > 0, nil
}

// RemoveFromBlacklist 从黑名单中移除 token
func (s *tokenBlacklistService) RemoveFromBlacklist(ctx context.Context, token string) error {
	// 如果 Redis 未启用，返回 nil
	if !s.redis.IsEnabled() {
		return nil
	}

	tokenHash := s.hashToken(token)
	key := s.getBlacklistKey(tokenHash)
	
	if err := s.redis.Del(ctx, key); err != nil {
		s.logger.Error("从黑名单中移除 token 失败", logger.Fields{"error": err})
		return fmt.Errorf("从黑名单中移除 token 失败: %w", err)
	}

	s.logger.Info("token 已从黑名单中移除")
	return nil
}

// CleanupExpired 清理已过期的黑名单条目
// 注意：Redis 会自动清理过期的键，此方法主要用于兼容性
func (s *tokenBlacklistService) CleanupExpired(ctx context.Context) error {
	// Redis 会自动处理过期键，无需手动清理
	s.logger.Debug("Redis 自动处理过期键，无需手动清理")
	return nil
}
