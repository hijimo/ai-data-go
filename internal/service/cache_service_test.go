package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 创建测试用的 Redis 实例
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestCacheService_SetAndGet(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 测试数据
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testData := TestData{
		Name:  "test",
		Value: 123,
	}

	// 设置缓存
	err := cache.Set(ctx, "test:key", testData, 1*time.Minute)
	assert.NoError(t, err)

	// 获取缓存
	var result TestData
	err = cache.Get(ctx, "test:key", &result)
	assert.NoError(t, err)
	assert.Equal(t, testData.Name, result.Name)
	assert.Equal(t, testData.Value, result.Value)
}

func TestCacheService_GetNotFound(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "nonexistent:key", &result)
	assert.ErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheService_Delete(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 设置缓存
	err := cache.Set(ctx, "test:key", "value", 1*time.Minute)
	assert.NoError(t, err)

	// 验证存在
	exists, err := cache.Exists(ctx, "test:key")
	assert.NoError(t, err)
	assert.True(t, exists)

	// 删除缓存
	err = cache.Delete(ctx, "test:key")
	assert.NoError(t, err)

	// 验证已删除
	exists, err = cache.Exists(ctx, "test:key")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCacheService_DeletePattern(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 设置多个缓存
	err := cache.Set(ctx, "session:123:context", "value1", 1*time.Minute)
	assert.NoError(t, err)

	err = cache.Set(ctx, "session:123:summary", "value2", 1*time.Minute)
	assert.NoError(t, err)

	err = cache.Set(ctx, "session:456:context", "value3", 1*time.Minute)
	assert.NoError(t, err)

	// 删除匹配模式的缓存
	err = cache.DeletePattern(ctx, "session:123:*")
	assert.NoError(t, err)

	// 验证 session:123 的缓存已删除
	exists, err := cache.Exists(ctx, "session:123:context")
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = cache.Exists(ctx, "session:123:summary")
	assert.NoError(t, err)
	assert.False(t, exists)

	// 验证 session:456 的缓存仍然存在
	exists, err = cache.Exists(ctx, "session:456:context")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCacheService_Increment(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 第一次增加
	result, err := cache.Increment(ctx, "counter:key", 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	// 第二次增加
	result, err = cache.Increment(ctx, "counter:key", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), result)

	// 第三次增加
	result, err = cache.Increment(ctx, "counter:key", 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(16), result)
}

func TestCacheService_TTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 设置缓存，TTL 为 1 分钟
	err := cache.Set(ctx, "test:key", "value", 1*time.Minute)
	assert.NoError(t, err)

	// 获取 TTL
	ttl, err := cache.TTL(ctx, "test:key")
	assert.NoError(t, err)
	assert.Greater(t, ttl, 50*time.Second) // 应该接近 1 分钟
	assert.LessOrEqual(t, ttl, 1*time.Minute)
}

func TestCacheService_Expire(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	cache := NewCacheService(client, "test")
	ctx := context.Background()

	// 设置缓存，TTL 为 1 分钟
	err := cache.Set(ctx, "test:key", "value", 1*time.Minute)
	assert.NoError(t, err)

	// 修改 TTL 为 2 分钟
	err = cache.Expire(ctx, "test:key", 2*time.Minute)
	assert.NoError(t, err)

	// 验证新的 TTL
	ttl, err := cache.TTL(ctx, "test:key")
	assert.NoError(t, err)
	assert.Greater(t, ttl, 1*time.Minute+50*time.Second) // 应该接近 2 分钟
	assert.LessOrEqual(t, ttl, 2*time.Minute)
}

func TestCacheService_HashQuery(t *testing.T) {
	cache := NewCacheService(nil, "test")

	// 测试相同的查询生成相同的哈希
	query1 := "这是一个测试查询"
	hash1 := cache.HashQuery(query1)
	hash2 := cache.HashQuery(query1)
	assert.Equal(t, hash1, hash2)

	// 测试不同的查询生成不同的哈希
	query2 := "这是另一个测试查询"
	hash3 := cache.HashQuery(query2)
	assert.NotEqual(t, hash1, hash3)

	// 验证哈希长度（MD5 哈希应该是 32 个字符）
	assert.Len(t, hash1, 32)
}

func TestCacheService_NilClient(t *testing.T) {
	// 测试 Redis 客户端为 nil 的情况（Redis 未启用）
	cache := NewCacheService(nil, "test")
	ctx := context.Background()

	// Get 应该返回 ErrCacheNotFound
	var result string
	err := cache.Get(ctx, "test:key", &result)
	assert.ErrorIs(t, err, ErrCacheNotFound)

	// Set 应该静默失败（不返回错误）
	err = cache.Set(ctx, "test:key", "value", 1*time.Minute)
	assert.NoError(t, err)

	// Delete 应该静默失败（不返回错误）
	err = cache.Delete(ctx, "test:key")
	assert.NoError(t, err)

	// DeletePattern 应该静默失败（不返回错误）
	err = cache.DeletePattern(ctx, "test:*")
	assert.NoError(t, err)

	// Exists 应该返回 false
	exists, err := cache.Exists(ctx, "test:key")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Expire 应该静默失败（不返回错误）
	err = cache.Expire(ctx, "test:key", 1*time.Minute)
	assert.NoError(t, err)
}
