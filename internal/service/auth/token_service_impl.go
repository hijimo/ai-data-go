package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// tokenServiceImpl TokenService 接口的实现
type tokenServiceImpl struct {
	config    *config.AuthConfig
	tokenRepo repository.RefreshTokenRepository
}

// NewTokenService 创建 TokenService 实例
// 参数：
//   - cfg: 认证配置
//   - tokenRepo: RefreshToken 仓储
// 返回：
//   - TokenService: TokenService 接口实例
func NewTokenService(cfg *config.AuthConfig, tokenRepo repository.RefreshTokenRepository) TokenService {
	return &tokenServiceImpl{
		config:    cfg,
		tokenRepo: tokenRepo,
	}
}

// GenerateAccessToken 生成访问令牌（JWT）
func (s *tokenServiceImpl) GenerateAccessToken(user *model.User) (string, error) {
	if user == nil {
		return "", errors.New("用户信息不能为空")
	}

	now := time.Now()

	// 提取用户角色
	roles := extractRoles(user.Roles)

	// 构建 JWT Claims
	claims := model.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.JWTIssuer,
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{s.config.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		TenantID: user.TenantID,
		Roles:    roles,
		Scopes:   generateScopes(roles), // 根据角色生成权限范围
	}

	// 创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名并返回
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("生成 JWT token 失败: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken 验证访问令牌
func (s *tokenServiceImpl) ValidateAccessToken(tokenString string) (*model.JWTClaims, error) {
	if tokenString == "" {
		return nil, errors.New("token 不能为空")
	}

	// 解析 token
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名算法: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析 token 失败: %w", err)
	}

	// 提取 claims
	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的 token")
	}

	// 验证必需的 claims
	if claims.Subject == "" {
		return nil, errors.New("token 缺少用户 ID (sub)")
	}

	if claims.TenantID == "" {
		return nil, errors.New("token 缺少租户 ID (tid)")
	}

	// 验证过期时间
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token 已过期")
	}

	return claims, nil
}

// HashToken 计算 token 的哈希值
func (s *tokenServiceImpl) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// extractRoles 从 JSONB 中提取角色列表
func extractRoles(rolesJSON datatypes.JSON) []string {
	if rolesJSON == nil || len(rolesJSON) == 0 {
		return []string{"user"} // 默认角色
	}

	var roles []string
	if err := json.Unmarshal(rolesJSON, &roles); err != nil {
		return []string{"user"} // 解析失败返回默认角色
	}

	if len(roles) == 0 {
		return []string{"user"} // 空列表返回默认角色
	}

	return roles
}

// GenerateRefreshToken 生成刷新令牌
func (s *tokenServiceImpl) GenerateRefreshToken(user *model.User) (string, *model.RefreshToken, error) {
	if user == nil {
		return "", nil, errors.New("用户信息不能为空")
	}

	// 生成随机 UUID 作为 token
	tokenString := uuid.New().String()

	// 计算 token 哈希值
	tokenHash := s.HashToken(tokenString)

	// 创建 RefreshToken 记录
	refreshToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TenantID:  user.TenantID,
		TokenHash: tokenHash,
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.config.RefreshTokenTTL),
	}

	// 保存到数据库
	if err := s.tokenRepo.Create(context.Background(), refreshToken); err != nil {
		return "", nil, fmt.Errorf("保存 refresh token 失败: %w", err)
	}

	return tokenString, refreshToken, nil
}

// ValidateRefreshToken 验证刷新令牌
func (s *tokenServiceImpl) ValidateRefreshToken(ctx context.Context, tokenString string) (*model.RefreshToken, error) {
	if tokenString == "" {
		return nil, errors.New("token 不能为空")
	}

	// 计算 token 哈希值
	tokenHash := s.HashToken(tokenString)

	// 从数据库查询
	refreshToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("token 不存在或已失效: %w", err)
	}

	// 验证是否已撤销
	if refreshToken.Revoked {
		return nil, errors.New("token 已被撤销")
	}

	// 验证是否过期
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, errors.New("token 已过期")
	}

	return refreshToken, nil
}

// RevokeRefreshToken 撤销刷新令牌
func (s *tokenServiceImpl) RevokeRefreshToken(ctx context.Context, tokenString string) error {
	if tokenString == "" {
		return errors.New("token 不能为空")
	}

	// 计算 token 哈希值
	tokenHash := s.HashToken(tokenString)

	// 从数据库查询
	refreshToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("token 不存在: %w", err)
	}

	// 撤销 token（不设置 replacedBy）
	if err := s.tokenRepo.Revoke(ctx, refreshToken.ID, nil); err != nil {
		return fmt.Errorf("撤销 token 失败: %w", err)
	}

	return nil
}

// generateScopes 根据角色生成权限范围
func generateScopes(roles []string) []string {
	scopesMap := make(map[string]bool)

	for _, role := range roles {
		switch role {
		case "admin":
			// 管理员拥有所有权限
			scopesMap["chat:read"] = true
			scopesMap["chat:write"] = true
			scopesMap["chat:delete"] = true
			scopesMap["session:read"] = true
			scopesMap["session:write"] = true
			scopesMap["session:delete"] = true
			scopesMap["user:read"] = true
			scopesMap["user:write"] = true
			scopesMap["user:delete"] = true
			scopesMap["tenant:read"] = true
			scopesMap["tenant:write"] = true
		case "moderator":
			// 协调员拥有部分管理权限
			scopesMap["chat:read"] = true
			scopesMap["chat:write"] = true
			scopesMap["chat:delete"] = true
			scopesMap["session:read"] = true
			scopesMap["session:write"] = true
			scopesMap["user:read"] = true
		case "user":
			// 普通用户基本权限
			scopesMap["chat:read"] = true
			scopesMap["chat:write"] = true
			scopesMap["session:read"] = true
			scopesMap["session:write"] = true
		}
	}

	// 转换为切片
	scopes := make([]string, 0, len(scopesMap))
	for scope := range scopesMap {
		scopes = append(scopes, scope)
	}

	return scopes
}
