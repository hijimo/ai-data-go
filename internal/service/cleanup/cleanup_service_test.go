package cleanup

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&model.RefreshToken{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// TestCleanupService_CleanExpiredTokens 测试清理过期 Token
func TestCleanupService_CleanExpiredTokens(t *testing.T) {
	// 1. 设置测试数据库
	db := setupTestDB(t)

	// 2. 创建 repository 和 service
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	log := logger.NewTestLogger()
	config := CleanupConfig{
		TokenCleanupInterval: 1 * time.Hour,
	}
	cleanupService := NewCleanupService(refreshTokenRepo, log, config)

	ctx := context.Background()

	// 3. 创建测试数据
	now := time.Now()
	
	// 创建一个已过期的 token
	expiredToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		TenantID:  uuid.New().String(),
		TokenHash: "expired_token_hash",
		Revoked:   false,
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour), // 1 小时前过期
	}
	if err := refreshTokenRepo.Create(ctx, expiredToken); err != nil {
		t.Fatalf("创建过期 token 失败: %v", err)
	}

	// 创建一个未过期的 token
	validToken := &model.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		TenantID:  uuid.New().String(),
		TokenHash: "valid_token_hash",
		Revoked:   false,
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour), // 1 小时后过期
	}
	if err := refreshTokenRepo.Create(ctx, validToken); err != nil {
		t.Fatalf("创建有效 token 失败: %v", err)
	}

	// 4. 执行清理
	if err := cleanupService.CleanExpiredTokens(ctx); err != nil {
		t.Fatalf("清理过期 token 失败: %v", err)
	}

	// 5. 验证结果
	// 过期的 token 应该被删除
	_, err := refreshTokenRepo.GetByTokenHash(ctx, expiredToken.TokenHash)
	if err == nil {
		t.Error("过期的 token 应该被删除")
	}

	// 有效的 token 应该仍然存在
	retrievedToken, err := refreshTokenRepo.GetByTokenHash(ctx, validToken.TokenHash)
	if err != nil {
		t.Fatalf("获取有效 token 失败: %v", err)
	}
	if retrievedToken.ID != validToken.ID {
		t.Error("有效的 token 应该仍然存在")
	}
}

// TestCleanupService_StartStop 测试启动和停止清理服务
func TestCleanupService_StartStop(t *testing.T) {
	// 1. 设置测试数据库
	db := setupTestDB(t)

	// 2. 创建 repository 和 service
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	log := logger.NewTestLogger()
	config := CleanupConfig{
		TokenCleanupInterval: 100 * time.Millisecond, // 使用短间隔进行测试
	}
	cleanupService := NewCleanupService(refreshTokenRepo, log, config)

	ctx := context.Background()

	// 3. 启动清理服务
	cleanupService.Start(ctx)

	// 4. 等待一段时间确保至少执行一次清理
	time.Sleep(200 * time.Millisecond)

	// 5. 停止清理服务
	cleanupService.Stop()

	// 测试通过（没有 panic 或错误）
}

// TestNewCleanupService_DefaultInterval 测试默认清理间隔
func TestNewCleanupService_DefaultInterval(t *testing.T) {
	// 1. 设置测试数据库
	db := setupTestDB(t)

	// 2. 创建 repository 和 service（不设置间隔）
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	log := logger.NewTestLogger()
	config := CleanupConfig{
		TokenCleanupInterval: 0, // 不设置，应该使用默认值
	}
	cleanupService := NewCleanupService(refreshTokenRepo, log, config)

	// 3. 验证服务创建成功
	if cleanupService == nil {
		t.Error("清理服务不应该为 nil")
	}
	
	// 注意：由于 interval 是私有字段，我们无法直接验证
	// 但可以通过启动服务来间接验证其正常工作
	ctx := context.Background()
	cleanupService.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cleanupService.Stop()
}
