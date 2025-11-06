package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/logger"

	"github.com/redis/go-redis/v9"
)

// CacheServiceImpl 缓存服务实现
type CacheServiceImpl struct {
	redis  *database.RedisClient
	logger logger.Logger
}

// NewCacheService 创建新的缓存服务实例
func NewCacheService(redis *database.RedisClient, log logger.Logger) CacheService {
	return &CacheServiceImpl{
		redis:  redis,
		logger: log,
	}
}

// Get 获取缓存值并反序列化到 dest
func (s *CacheServiceImpl) Get(ctx context.Context, key string, dest interface{}) error {
	if !s.IsEnabled() {
		return fmt.Errorf("缓存服务未启用")
	}

	value, err := s.redis.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return &CacheKeyNotFoundError{Key: key}
		}
		s.logger.ErrorContext(ctx, "获取缓存失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	// 反序列化 JSON 数据
	if err := json.Unmarshal([]byte(value), dest); err != nil {
		s.logger.ErrorContext(ctx, "反序列化缓存失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return fmt.Errorf("反序列化缓存失败: %w", err)
	}

	s.logger.DebugContext(ctx, "缓存命中", logger.Fields{
		"key": key,
	})

	return nil
}

// GetString 获取字符串类型的缓存值
func (s *CacheServiceImpl) GetString(ctx context.Context, key string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("缓存服务未启用")
	}

	value, err := s.redis.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return "", &CacheKeyNotFoundError{Key: key}
		}
		s.logger.ErrorContext(ctx, "获取缓存失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return "", fmt.Errorf("获取缓存失败: %w", err)
	}

	s.logger.DebugContext(ctx, "缓存命中", logger.Fields{
		"key": key,
	})

	return value, nil
}

// Set 设置缓存值（自动序列化为 JSON）
func (s *CacheServiceImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !s.IsEnabled() {
		return fmt.Errorf("缓存服务未启用")
	}

	// 序列化为 JSON
	data, err := json.Marshal(value)
	if err != nil {
		s.logger.ErrorContext(ctx, "序列化缓存失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	err = s.redis.Set(ctx, key, data, expiration)
	if err != nil {
		s.logger.ErrorContext(ctx, "设置缓存失败", logger.Fields{
			"key":        key,
			"expiration": expiration.String(),
			"error":      err.Error(),
		})
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	s.logger.DebugContext(ctx, "缓存已设置", logger.Fields{
		"key":        key,
		"expiration": expiration.String(),
	})

	return nil
}

// SetString 设置字符串类型的缓存值
func (s *CacheServiceImpl) SetString(ctx context.Context, key string, value string, expiration time.Duration) error {
	if !s.IsEnabled() {
		return fmt.Errorf("缓存服务未启用")
	}

	err := s.redis.Set(ctx, key, value, expiration)
	if err != nil {
		s.logger.ErrorContext(ctx, "设置缓存失败", logger.Fields{
			"key":        key,
			"expiration": expiration.String(),
			"error":      err.Error(),
		})
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	s.logger.DebugContext(ctx, "缓存已设置", logger.Fields{
		"key":        key,
		"expiration": expiration.String(),
	})

	return nil
}

// Delete 删除缓存键
func (s *CacheServiceImpl) Delete(ctx context.Context, keys ...string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("缓存服务未启用")
	}

	if len(keys) == 0 {
		return nil
	}

	err := s.redis.Del(ctx, keys...)
	if err != nil {
		s.logger.ErrorContext(ctx, "删除缓存失败", logger.Fields{
			"keys":  keys,
			"error": err.Error(),
		})
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	s.logger.DebugContext(ctx, "缓存已删除", logger.Fields{
		"keys":  keys,
		"count": len(keys),
	})

	return nil
}

// DeletePattern 按模式删除缓存键
func (s *CacheServiceImpl) DeletePattern(ctx context.Context, pattern string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("缓存服务未启用")
	}

	client := s.redis.GetClient()
	if client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	// 使用 SCAN 命令查找匹配的键
	var cursor uint64
	var deletedCount int64

	for {
		var keys []string
		var err error

		// 扫描匹配的键
		keys, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.logger.ErrorContext(ctx, "扫描缓存键失败", logger.Fields{
				"pattern": pattern,
				"error":   err.Error(),
			})
			return fmt.Errorf("扫描缓存键失败: %w", err)
		}

		// 删除找到的键
		if len(keys) > 0 {
			err = s.redis.Del(ctx, keys...)
			if err != nil {
				s.logger.ErrorContext(ctx, "批量删除缓存失败", logger.Fields{
					"pattern": pattern,
					"keys":    keys,
					"error":   err.Error(),
				})
				return fmt.Errorf("批量删除缓存失败: %w", err)
			}
			deletedCount += int64(len(keys))
		}

		// 如果 cursor 为 0，表示扫描完成
		if cursor == 0 {
			break
		}
	}

	s.logger.InfoContext(ctx, "按模式删除缓存完成", logger.Fields{
		"pattern":       pattern,
		"deleted_count": deletedCount,
	})

	return nil
}

// Exists 检查缓存键是否存在
func (s *CacheServiceImpl) Exists(ctx context.Context, keys ...string) (int64, error) {
	if !s.IsEnabled() {
		return 0, fmt.Errorf("缓存服务未启用")
	}

	if len(keys) == 0 {
		return 0, nil
	}

	count, err := s.redis.Exists(ctx, keys...)
	if err != nil {
		s.logger.ErrorContext(ctx, "检查缓存键存在性失败", logger.Fields{
			"keys":  keys,
			"error": err.Error(),
		})
		return 0, fmt.Errorf("检查缓存键存在性失败: %w", err)
	}

	return count, nil
}

// Increment 增量操作
func (s *CacheServiceImpl) Increment(ctx context.Context, key string, value int64) (int64, error) {
	if !s.IsEnabled() {
		return 0, fmt.Errorf("缓存服务未启用")
	}

	client := s.redis.GetClient()
	if client == nil {
		return 0, fmt.Errorf("Redis 客户端未初始化")
	}

	newValue, err := client.IncrBy(ctx, key, value).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "增量操作失败", logger.Fields{
			"key":   key,
			"value": value,
			"error": err.Error(),
		})
		return 0, fmt.Errorf("增量操作失败: %w", err)
	}

	s.logger.DebugContext(ctx, "增量操作完成", logger.Fields{
		"key":       key,
		"increment": value,
		"new_value": newValue,
	})

	return newValue, nil
}

// GetWithNamespace 使用命名空间获取缓存值
func (s *CacheServiceImpl) GetWithNamespace(ctx context.Context, namespace, key string, dest interface{}) error {
	fullKey := s.buildNamespacedKey(namespace, key)
	return s.Get(ctx, fullKey, dest)
}

// SetWithNamespace 使用命名空间设置缓存值
func (s *CacheServiceImpl) SetWithNamespace(ctx context.Context, namespace, key string, value interface{}, expiration time.Duration) error {
	fullKey := s.buildNamespacedKey(namespace, key)
	return s.Set(ctx, fullKey, value, expiration)
}

// DeleteWithNamespace 使用命名空间删除缓存键
func (s *CacheServiceImpl) DeleteWithNamespace(ctx context.Context, namespace string, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = s.buildNamespacedKey(namespace, key)
	}

	return s.Delete(ctx, fullKeys...)
}

// DeleteNamespace 删除整个命名空间下的所有键
func (s *CacheServiceImpl) DeleteNamespace(ctx context.Context, namespace string) error {
	pattern := s.buildNamespacedKey(namespace, "*")
	return s.DeletePattern(ctx, pattern)
}

// TTL 获取键的剩余生存时间
func (s *CacheServiceImpl) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !s.IsEnabled() {
		return 0, fmt.Errorf("缓存服务未启用")
	}

	ttl, err := s.redis.TTL(ctx, key)
	if err != nil {
		s.logger.ErrorContext(ctx, "获取TTL失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return 0, fmt.Errorf("获取TTL失败: %w", err)
	}

	return ttl, nil
}

// IsEnabled 检查缓存服务是否已启用
func (s *CacheServiceImpl) IsEnabled() bool {
	return s.redis != nil && s.redis.IsEnabled()
}

// buildNamespacedKey 构建带命名空间的键
func (s *CacheServiceImpl) buildNamespacedKey(namespace, key string) string {
	if namespace == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", namespace, key)
}
