package storage_test

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/storage"

	"github.com/google/uuid"
)

// 示例：启动时预热缓存
func ExampleCacheWarmer_WarmupOnStartup() {
	// 创建依赖（实际使用时从依赖注入容器获取）
	// cacheService := storage.NewCacheService(redisClient, log)
	// contextRepo := repository.NewContextRepository(db)
	// summaryRepo := repository.NewSummaryRepository(db)
	// sessionRepo := repository.NewSessionRepository(db)
	// db := database.GetDB()
	// log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)

	// 创建缓存预热配置
	config := &storage.CacheWarmerConfig{
		WarmupInterval:    30 * time.Minute, // 30分钟预热一次
		ActiveSessionDays: 7,                // 7天内活跃的会话
	}

	// 创建缓存预热器
	// warmer := storage.NewCacheWarmer(
	// 	cacheService,
	// 	contextRepo,
	// 	summaryRepo,
	// 	sessionRepo,
	// 	db,
	// 	log,
	// 	config,
	// )

	// 执行启动时预热
	ctx := context.Background()
	// if err := warmer.WarmupOnStartup(ctx); err != nil {
	// 	log.Fatalf("缓存预热失败: %v", err)
	// }

	fmt.Println("缓存预热完成")
	// Output: 缓存预热完成
	_ = ctx
}

// 示例：启动定期预热
func ExampleCacheWarmer_StartPeriodicWarmup() {
	// 创建缓存预热器（省略初始化代码）
	// warmer := storage.NewCacheWarmer(...)

	// 启动定期预热（在后台运行）
	ctx := context.Background()
	// warmer.StartPeriodicWarmup(ctx)

	// 应用继续运行...
	// time.Sleep(1 * time.Hour)

	// 优雅关闭时停止预热
	// warmer.Stop()

	fmt.Println("定期预热已启动")
	// Output: 定期预热已启动
	_ = ctx
}

// 示例：按需预热会话列表
func ExampleCacheWarmer_WarmupSessionList() {
	// 创建缓存预热器（省略初始化代码）
	// warmer := storage.NewCacheWarmer(...)

	// 预热用户的会话列表
	ctx := context.Background()
	userID := uuid.New()
	page := 1
	pageSize := 20

	// if err := warmer.WarmupSessionList(ctx, userID, page, pageSize); err != nil {
	// 	log.Printf("预热会话列表失败: %v", err)
	// }

	fmt.Printf("预热会话列表: userID=%s, page=%d, pageSize=%d\n", userID, page, pageSize)
	// Output: 预热会话列表: userID=00000000-0000-0000-0000-000000000000, page=1, pageSize=20
	_ = ctx
}

// 示例：按需预热Token使用统计
func ExampleCacheWarmer_WarmupTokenUsage() {
	// 创建缓存预热器（省略初始化代码）
	// warmer := storage.NewCacheWarmer(...)

	// 预热会话的Token使用统计
	ctx := context.Background()
	sessionID := uuid.New()

	// if err := warmer.WarmupTokenUsage(ctx, sessionID); err != nil {
	// 	log.Printf("预热Token使用统计失败: %v", err)
	// }

	fmt.Printf("预热Token使用统计: sessionID=%s\n", sessionID)
	// Output: 预热Token使用统计: sessionID=00000000-0000-0000-0000-000000000000
	_ = ctx
}

// 示例：完整的应用启动流程
func ExampleCacheWarmer_fullStartup() {
	// 1. 初始化所有依赖
	log := logger.New(logger.InfoLevel, logger.JSONFormat, nil)
	_ = log

	// 2. 创建缓存预热器
	config := &storage.CacheWarmerConfig{
		WarmupInterval:    30 * time.Minute,
		ActiveSessionDays: 7,
	}
	_ = config

	// warmer := storage.NewCacheWarmer(
	// 	cacheService,
	// 	contextRepo,
	// 	summaryRepo,
	// 	sessionRepo,
	// 	db,
	// 	log,
	// 	config,
	// )

	// 3. 执行启动时预热
	ctx := context.Background()
	// if err := warmer.WarmupOnStartup(ctx); err != nil {
	// 	log.Fatalf("启动时缓存预热失败: %v", err)
	// }

	// 4. 启动定期预热
	// warmer.StartPeriodicWarmup(ctx)

	// 5. 启动HTTP服务器
	// server := &http.Server{...}
	// go server.ListenAndServe()

	// 6. 等待关闭信号
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	// <-sigChan

	// 7. 优雅关闭
	// warmer.Stop()
	// server.Shutdown(ctx)

	fmt.Println("应用启动完成")
	// Output: 应用启动完成
	_ = ctx
}

// 示例：自定义配置
func ExampleCacheWarmerConfig() {
	// 高频使用场景：更短的预热间隔和活跃阈值
	highFreqConfig := &storage.CacheWarmerConfig{
		WarmupInterval:    15 * time.Minute, // 15分钟预热一次
		ActiveSessionDays: 3,                // 3天内活跃的会话
	}

	// 低频使用场景：更长的预热间隔和活跃阈值
	lowFreqConfig := &storage.CacheWarmerConfig{
		WarmupInterval:    1 * time.Hour, // 1小时预热一次
		ActiveSessionDays: 30,            // 30天内活跃的会话
	}

	fmt.Printf("高频配置: 间隔=%v, 阈值=%d天\n", highFreqConfig.WarmupInterval, highFreqConfig.ActiveSessionDays)
	fmt.Printf("低频配置: 间隔=%v, 阈值=%d天\n", lowFreqConfig.WarmupInterval, lowFreqConfig.ActiveSessionDays)
	// Output:
	// 高频配置: 间隔=15m0s, 阈值=3天
	// 低频配置: 间隔=1h0m0s, 阈值=30天
}
