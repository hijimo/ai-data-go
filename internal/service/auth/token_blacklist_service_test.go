package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/logger"
)

func TestTokenBlacklistService(t *testing.T) {
	// 创建测试用的 Redis 客户端（如果 Redis 不可用，测试会跳过）
	cfg := config.RedisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       1, // 使用测试数据库
		Enabled:  true,
	}

	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)

	redisClient, err := database.NewRedisClient(cfg, log)
	if err != nil {
		t.Skipf("跳过测试：无法连接到 Redis: %v", err)
		return
	}
	defer redisClient.Close()

	service := NewTokenBlacklistService(redisClient, log)
	ctx := context.Background()

	t.Run("添加和检查黑名单", func(t *testing.T) {
		token := "test-token-123"
		expiresAt := time.Now().Add(1 * time.Hour)

		// 添加到黑名单
		err := service.AddToBlacklist(ctx, token, expiresAt)
		if err != nil {
			t.Fatalf("添加到黑名单失败: %v", err)
		}

		// 检查是否在黑名单中
		isBlacklisted, err := service.IsBlacklisted(ctx, token)
		if err != nil {
			t.Fatalf("检查黑名单状态失败: %v", err)
		}

		if !isBlacklisted {
			t.Error("token 应该在黑名单中")
		}
	})

	t.Run("检查不在黑名单中的 token", func(t *testing.T) {
		token := "non-existent-token"

		isBlacklisted, err := service.IsBlacklisted(ctx, token)
		if err != nil {
			t.Fatalf("检查黑名单状态失败: %v", err)
		}

		if isBlacklisted {
			t.Error("token 不应该在黑名单中")
		}
	})

	t.Run("从黑名单中移除 token", func(t *testing.T) {
		token := "test-token-to-remove"
		expiresAt := time.Now().Add(1 * time.Hour)

		// 添加到黑名单
		err := service.AddToBlacklist(ctx, token, expiresAt)
		if err != nil {
			t.Fatalf("添加到黑名单失败: %v", err)
		}

		// 从黑名单中移除
		err = service.RemoveFromBlacklist(ctx, token)
		if err != nil {
			t.Fatalf("从黑名单中移除失败: %v", err)
		}

		// 检查是否已移除
		isBlacklisted, err := service.IsBlacklisted(ctx, token)
		if err != nil {
			t.Fatalf("检查黑名单状态失败: %v", err)
		}

		if isBlacklisted {
			t.Error("token 应该已从黑名单中移除")
		}
	})

	t.Run("已过期的 token 不应加入黑名单", func(t *testing.T) {
		token := "expired-token"
		expiresAt := time.Now().Add(-1 * time.Hour) // 已过期

		// 尝试添加到黑名单
		err := service.AddToBlacklist(ctx, token, expiresAt)
		if err != nil {
			t.Fatalf("添加到黑名单失败: %v", err)
		}

		// 检查是否在黑名单中（不应该在）
		isBlacklisted, err := service.IsBlacklisted(ctx, token)
		if err != nil {
			t.Fatalf("检查黑名单状态失败: %v", err)
		}

		if isBlacklisted {
			t.Error("已过期的 token 不应该在黑名单中")
		}
	})
}

func TestTokenBlacklistServiceWithoutRedis(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)

	// 创建未启用 Redis 的服务
	service := NewTokenBlacklistService(nil, log)
	ctx := context.Background()

	t.Run("Redis 未启用时不应报错", func(t *testing.T) {
		token := "test-token"
		expiresAt := time.Now().Add(1 * time.Hour)

		// 添加到黑名单（应该不报错）
		err := service.AddToBlacklist(ctx, token, expiresAt)
		if err != nil {
			t.Fatalf("Redis 未启用时添加到黑名单不应报错: %v", err)
		}

		// 检查黑名单（应该返回 false）
		isBlacklisted, err := service.IsBlacklisted(ctx, token)
		if err != nil {
			t.Fatalf("Redis 未启用时检查黑名单不应报错: %v", err)
		}

		if isBlacklisted {
			t.Error("Redis 未启用时应该返回 false")
		}
	})
}
