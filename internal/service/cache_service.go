package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheNotFound 缓存未找到错误
var ErrCacheNotFound = errors.New("缓存未找到")

// CacheService 缓存服务接口
type CacheService interface {
	// Get 获取缓存
	Get(ctx context.Context, key string, dest interface{}) error

	// Set 设置缓存
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, keys ...string) error

	// DeletePattern 按模式删除缓存
	DeletePattern(ctx context.Context, pattern string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Increment 增加计数器
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Expire 设置过期时间
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// TTL 获取剩余生存时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// HashQuery 对查询字符串进行哈希
	HashQuery(query string) string
}

// cacheServiceImpl 缓存服务实现
type cacheServiceImpl struct {
	client    *redis.Client
	namespace string
}

// NewCacheService 创建新的缓存服务实例
func NewCacheService(client *redis.Client, namespace string) CacheService {
	if namespace == "" {
		namespace = "genkit"
	}
	return &cacheServiceImpl{
		client:    client,
		namespace: namespace,
	}
}

// Get 获取缓存
func (s *cacheServiceImpl) Get(ctx context.Context, key string, dest interface{}) error {
	if s.client == nil {
		return ErrCacheNotFound
	}

	fullKey := s.buildKey(key)

	data, err := s.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheNotFound
		}
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("反序列化缓存失败: %w", err)
	}

	return nil
}

// Set 设置缓存
func (s *cacheServiceImpl) Set(
	ctx context.Context,
	key string,
	value interface{},
	ttl time.Duration,
) error {
	if s.client == nil {
		return nil // Redis 未启用，静默失败
	}

	fullKey := s.buildKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	if err := s.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (s *cacheServiceImpl) Delete(ctx context.Context, keys ...string) error {
	if s.client == nil {
		return nil // Redis 未启用，静默失败
	}

	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = s.buildKey(key)
	}

	if err := s.client.Del(ctx, fullKeys...).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	return nil
}

// DeletePattern 按模式删除缓存
func (s *cacheServiceImpl) DeletePattern(ctx context.Context, pattern string) error {
	if s.client == nil {
		return nil // Redis 未启用，静默失败
	}

	fullPattern := s.buildKey(pattern)

	// 使用 SCAN 命令遍历匹配的键
	iter := s.client.Scan(ctx, 0, fullPattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描缓存失败: %w", err)
	}

	// 批量删除
	if len(keys) > 0 {
		// 注意：这里不需要再次调用 buildKey，因为 SCAN 返回的已经是完整的键
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("批量删除缓存失败: %w", err)
		}
	}

	return nil
}

// Exists 检查缓存是否存在
func (s *cacheServiceImpl) Exists(ctx context.Context, key string) (bool, error) {
	if s.client == nil {
		return false, nil
	}

	fullKey := s.buildKey(key)

	count, err := s.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	return count > 0, nil
}

// Increment 增加计数器
func (s *cacheServiceImpl) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	if s.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}

	fullKey := s.buildKey(key)

	result, err := s.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("增加计数器失败: %w", err)
	}

	return result, nil
}

// Expire 设置过期时间
func (s *cacheServiceImpl) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if s.client == nil {
		return nil // Redis 未启用，静默失败
	}

	fullKey := s.buildKey(key)

	if err := s.client.Expire(ctx, fullKey, ttl).Err(); err != nil {
		return fmt.Errorf("设置过期时间失败: %w", err)
	}

	return nil
}

// TTL 获取剩余生存时间
func (s *cacheServiceImpl) TTL(ctx context.Context, key string) (time.Duration, error) {
	if s.client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}

	fullKey := s.buildKey(key)

	ttl, err := s.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("获取剩余生存时间失败: %w", err)
	}

	return ttl, nil
}

// HashQuery 对查询字符串进行哈希
func (s *cacheServiceImpl) HashQuery(query string) string {
	hash := md5.Sum([]byte(query))
	return hex.EncodeToString(hash[:])
}

// buildKey 构建完整的缓存键
func (s *cacheServiceImpl) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", s.namespace, key)
}
