package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-knowledge-platform/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RedisClient Redis客户端实例
var RedisClient *redis.Client

// NewRedisClient 创建新的Redis客户端
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	RedisClient = client
	logrus.Info("Redis连接成功")
	return client, nil
}

// CacheManager 缓存管理器
type CacheManager struct {
	client *redis.Client
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(client *redis.Client) *CacheManager {
	return &CacheManager{
		client: client,
	}
}

// Set 设置缓存
func (c *CacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// Get 获取缓存
func (c *CacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("缓存不存在")
		}
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("反序列化缓存数据失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *CacheManager) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

// Exists 检查缓存是否存在
func (c *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("检查缓存存在性失败: %w", err)
	}
	return count > 0, nil
}

// SetWithTTL 设置带TTL的缓存
func (c *CacheManager) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl)
}

// GetTTL 获取缓存剩余TTL
func (c *CacheManager) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("获取缓存TTL失败: %w", err)
	}
	return ttl, nil
}

// Increment 递增计数器
func (c *CacheManager) Increment(ctx context.Context, key string) (int64, error) {
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("递增计数器失败: %w", err)
	}
	return count, nil
}

// IncrementWithExpire 递增计数器并设置过期时间
func (c *CacheManager) IncrementWithExpire(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	pipe := c.client.TxPipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, expiration)
	
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("递增计数器并设置过期时间失败: %w", err)
	}
	
	return incrCmd.Val(), nil
}

// HealthCheck Redis健康检查
func (c *CacheManager) HealthCheck(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis健康检查失败: %w", err)
	}
	return nil
}

// Close 关闭Redis连接
func (c *CacheManager) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}