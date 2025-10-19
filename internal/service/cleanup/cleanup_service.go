package cleanup

import (
	"context"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
)

// CleanupService 数据库清理服务接口
type CleanupService interface {
	// Start 启动清理任务
	Start(ctx context.Context)

	// Stop 停止清理任务
	Stop()

	// CleanExpiredTokens 清理过期的 Refresh Token
	CleanExpiredTokens(ctx context.Context) error

	// CleanExpiredVerificationTokens 清理过期的邮箱验证令牌
	CleanExpiredVerificationTokens(ctx context.Context) error
}

// cleanupService 清理服务实现
type cleanupService struct {
	refreshTokenRepo      repository.RefreshTokenRepository
	verificationTokenRepo repository.EmailVerificationRepository
	logger                logger.Logger
	ticker                *time.Ticker
	stopChan              chan struct{}
	interval              time.Duration
}

// CleanupConfig 清理服务配置
type CleanupConfig struct {
	// TokenCleanupInterval Token 清理间隔，默认 1 小时
	TokenCleanupInterval time.Duration
}

// NewCleanupService 创建清理服务实例
func NewCleanupService(
	refreshTokenRepo repository.RefreshTokenRepository,
	verificationTokenRepo repository.EmailVerificationRepository,
	logger logger.Logger,
	config CleanupConfig,
) CleanupService {
	// 设置默认清理间隔
	interval := config.TokenCleanupInterval
	if interval == 0 {
		interval = 1 * time.Hour
	}

	return &cleanupService{
		refreshTokenRepo:      refreshTokenRepo,
		verificationTokenRepo: verificationTokenRepo,
		logger:                logger,
		interval:              interval,
		stopChan:              make(chan struct{}),
	}
}

// Start 启动清理任务
func (s *cleanupService) Start(ctx context.Context) {
	s.logger.Info("启动数据库清理服务", map[string]interface{}{
		"interval": s.interval.String(),
	})

	// 立即执行一次清理
	if err := s.CleanExpiredTokens(ctx); err != nil {
		s.logger.Error("初始清理 Refresh Token 失败", map[string]interface{}{
			"error": err.Error(),
		})
	}
	
	if err := s.CleanExpiredVerificationTokens(ctx); err != nil {
		s.logger.Error("初始清理验证令牌失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 创建定时器
	s.ticker = time.NewTicker(s.interval)

	// 启动后台清理任务
	go s.runCleanupLoop(ctx)
}

// Stop 停止清理任务
func (s *cleanupService) Stop() {
	s.logger.Info("停止数据库清理服务", nil)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	close(s.stopChan)
}

// runCleanupLoop 运行清理循环
func (s *cleanupService) runCleanupLoop(ctx context.Context) {
	for {
		select {
		case <-s.ticker.C:
			// 定时执行清理
			if err := s.CleanExpiredTokens(ctx); err != nil {
				s.logger.Error("定时清理 Refresh Token 失败", map[string]interface{}{
					"error": err.Error(),
				})
			}
			
			if err := s.CleanExpiredVerificationTokens(ctx); err != nil {
				s.logger.Error("定时清理验证令牌失败", map[string]interface{}{
					"error": err.Error(),
				})
			}

		case <-s.stopChan:
			// 收到停止信号
			s.logger.Info("清理服务已停止", nil)
			return

		case <-ctx.Done():
			// 上下文取消
			s.logger.Info("清理服务上下文已取消", nil)
			return
		}
	}
}

// CleanExpiredTokens 清理过期的 Refresh Token
func (s *cleanupService) CleanExpiredTokens(ctx context.Context) error {
	s.logger.Info("开始清理过期的 Refresh Token", nil)

	startTime := time.Now()

	// 调用 repository 删除过期 token
	err := s.refreshTokenRepo.DeleteExpired(ctx)
	if err != nil {
		s.logger.Error("清理过期 Token 失败", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	duration := time.Since(startTime)
	s.logger.Info("清理过期 Token 完成", map[string]interface{}{
		"duration": duration.String(),
	})

	return nil
}

// CleanExpiredVerificationTokens 清理过期的邮箱验证令牌
func (s *cleanupService) CleanExpiredVerificationTokens(ctx context.Context) error {
	s.logger.Info("开始清理过期的邮箱验证令牌", nil)

	startTime := time.Now()

	// 调用 repository 删除过期验证令牌
	err := s.verificationTokenRepo.DeleteExpired(ctx)
	if err != nil {
		s.logger.Error("清理过期验证令牌失败", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	duration := time.Since(startTime)
	s.logger.Info("清理过期验证令牌完成", map[string]interface{}{
		"duration": duration.String(),
	})

	return nil
}
