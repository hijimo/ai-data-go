package database

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/logger"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis 客户端包装
type RedisClient struct {
	client *redis.Client
	logger logger.Logger
}

// NewRedisClient 创建新的 Redis 客户端
func NewRedisClient(cfg config.RedisConfig, log logger.Logger) (*RedisClient, error) {
	if !cfg.Enabled {
		log.Info("Redis 已禁用，跳过连接")
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("无法连接到 Redis: %w", err)
	}

	log.Info("Redis 连接成功", logger.Fields{
		"host": cfg.Host,
		"port": cfg.Port,
		"db":   cfg.DB,
	})

	return &RedisClient{
		client: client,
		logger: log,
	}, nil
}

// GetClient 获取原始 Redis 客户端
func (r *RedisClient) GetClient() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Set 设置键值对
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取键值
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("Redis 客户端未初始化")
	}
	return r.client.Get(ctx, key).Result()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}
	return r.client.Exists(ctx, keys...).Result()
}

// Del 删除键
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}
	return r.client.Del(ctx, keys...).Err()
}

// TTL 获取键的剩余生存时间
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}
	return r.client.TTL(ctx, key).Result()
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// IsEnabled 检查 Redis 是否已启用
func (r *RedisClient) IsEnabled() bool {
	return r != nil && r.client != nil
}
