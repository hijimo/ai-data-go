package service_test

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/service"

	"github.com/redis/go-redis/v9"
)

// ExampleCacheService_basic 演示缓存服务的基本用法
func ExampleCacheService_basic() {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 创建缓存服务
	cache := service.NewCacheService(client, "genkit")
	ctx := context.Background()

	// 设置缓存
	type UserData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := UserData{
		Name:  "张三",
		Email: "zhangsan@example.com",
	}

	_ = cache.Set(ctx, "user:123", user, 5*time.Minute)

	// 获取缓存
	var cachedUser UserData
	_ = cache.Get(ctx, "user:123", &cachedUser)

	fmt.Printf("Name: %s, Email: %s\n", cachedUser.Name, cachedUser.Email)
	// Output: Name: 张三, Email: zhangsan@example.com
}

// ExampleCacheKeys 演示缓存键管理的用法
func ExampleCacheKeys() {
	keys := service.NewCacheKeys()

	// 构建上下文缓存键
	sessionID := "session-123"
	queryHash := "abc123"
	contextKey := keys.ContextKey(sessionID, queryHash)
	fmt.Println("Context Key:", contextKey)

	// 构建摘要缓存键
	summaryKey := keys.SummaryKey(sessionID)
	fmt.Println("Summary Key:", summaryKey)

	// 构建配额缓存键
	tenantID := "tenant-456"
	quotaKey := keys.QuotaKey(tenantID, "daily")
	fmt.Println("Quota Key:", quotaKey)

	// Output:
	// Context Key: context:session-123:abc123
	// Summary Key: summary:session-123:latest
	// Quota Key: quota:tenant-456:daily
}

// ExampleCacheService_withTTL 演示带 TTL 的缓存操作
func ExampleCacheService_withTTL() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := service.NewCacheService(client, "genkit")
	keys := service.NewCacheKeys()
	ctx := context.Background()

	// 使用预定义的 TTL
	sessionID := "session-123"
	summaryData := map[string]interface{}{
		"content":      "这是会话摘要",
		"message_count": 50,
	}

	// 设置摘要缓存，使用预定义的 TTL
	summaryKey := keys.SummaryKey(sessionID)
	summaryTTL := keys.SummaryTTL()
	_ = cache.Set(ctx, summaryKey, summaryData, summaryTTL)

	// 检查 TTL
	ttl, _ := cache.TTL(ctx, summaryKey)
	fmt.Printf("Summary TTL: %v\n", ttl > 0)

	// Output: Summary TTL: true
}

// ExampleCacheService_pattern 演示模式匹配删除
func ExampleCacheService_pattern() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := service.NewCacheService(client, "genkit")
	keys := service.NewCacheKeys()
	ctx := context.Background()

	sessionID := "session-123"

	// 设置多个相关缓存
	_ = cache.Set(ctx, keys.ContextKey(sessionID, "query1"), "data1", 5*time.Minute)
	_ = cache.Set(ctx, keys.ContextKey(sessionID, "query2"), "data2", 5*time.Minute)
	_ = cache.Set(ctx, keys.SummaryKey(sessionID), "summary", 1*time.Hour)

	// 使用模式删除所有会话相关的缓存
	pattern := keys.SessionPattern(sessionID)
	_ = cache.DeletePattern(ctx, pattern)

	fmt.Println("All session caches deleted")
	// Output: All session caches deleted
}

// ExampleCacheWarmer 演示缓存预热器的用法
// 注意：实际使用时需要提供真实的 logger 实现
func ExampleCacheWarmer() {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := service.NewCacheService(client, "genkit")
	keys := service.NewCacheKeys()

	// 在实际使用中，需要传入真实的 logger
	// logger := logger.New(...)
	// warmer := service.NewCacheWarmer(cache, keys, logger)

	ctx := context.Background()

	// 使会话缓存失效（不需要 logger）
	sessionID := "session-123"
	
	// 删除会话相关的所有缓存
	pattern := keys.SessionPattern(sessionID)
	_ = cache.DeletePattern(ctx, pattern)

	fmt.Println("Cache operations completed")
	// Output: Cache operations completed
}

// ExampleCacheService_counter 演示计数器功能
func ExampleCacheService_counter() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	cache := service.NewCacheService(client, "genkit")
	ctx := context.Background()

	// 增加计数器
	count1, _ := cache.Increment(ctx, "api:calls:tenant-123", 1)
	fmt.Printf("Count 1: %d\n", count1)

	count2, _ := cache.Increment(ctx, "api:calls:tenant-123", 5)
	fmt.Printf("Count 2: %d\n", count2)

	// Output:
	// Count 1: 1
	// Count 2: 6
}

// ExampleCacheService_hashQuery 演示查询哈希功能
func ExampleCacheService_hashQuery() {
	cache := service.NewCacheService(nil, "genkit")

	// 对查询进行哈希
	query := "用户想要了解 AI 对话系统的工作原理"
	hash := cache.HashQuery(query)

	fmt.Printf("Query hash length: %d\n", len(hash))
	fmt.Printf("Hash is consistent: %v\n", hash == cache.HashQuery(query))

	// Output:
	// Query hash length: 32
	// Hash is consistent: true
}
