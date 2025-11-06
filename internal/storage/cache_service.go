package storage

import (
	"context"
	"time"
)

// CacheService 缓存服务接口
type CacheService interface {
	// Get 获取缓存值并反序列化到 dest
	// dest: 目标对象指针，用于接收反序列化后的数据
	// 返回错误，如果键不存在返回 ErrCacheKeyNotFound
	Get(ctx context.Context, key string, dest interface{}) error

	// GetString 获取字符串类型的缓存值
	// 返回值和错误，如果键不存在返回 ErrCacheKeyNotFound
	GetString(ctx context.Context, key string) (string, error)

	// Set 设置缓存值（自动序列化为 JSON）
	// key: 缓存键
	// value: 缓存值（将被序列化为 JSON）
	// expiration: 过期时间，0 表示永不过期
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// SetString 设置字符串类型的缓存值
	// key: 缓存键
	// value: 字符串值
	// expiration: 过期时间
	SetString(ctx context.Context, key string, value string, expiration time.Duration) error

	// Delete 删除缓存键
	// 支持删除多个键
	Delete(ctx context.Context, keys ...string) error

	// DeletePattern 按模式删除缓存键
	// pattern: 匹配模式，支持 * 通配符
	// 例如：DeletePattern(ctx, "user:*") 删除所有以 "user:" 开头的键
	DeletePattern(ctx context.Context, pattern string) error

	// Exists 检查缓存键是否存在
	// 返回存在的键数量
	Exists(ctx context.Context, keys ...string) (int64, error)

	// Increment 增量操作
	// 对数值类型的缓存值进行增量操作
	// 如果键不存在，则初始化为 0 后再增加
	Increment(ctx context.Context, key string, value int64) (int64, error)

	// GetWithNamespace 使用命名空间获取缓存值
	// namespace: 命名空间前缀
	// key: 缓存键
	// dest: 目标对象指针
	GetWithNamespace(ctx context.Context, namespace, key string, dest interface{}) error

	// SetWithNamespace 使用命名空间设置缓存值
	// namespace: 命名空间前缀
	// key: 缓存键
	// value: 缓存值
	// expiration: 过期时间
	SetWithNamespace(ctx context.Context, namespace, key string, value interface{}, expiration time.Duration) error

	// DeleteWithNamespace 使用命名空间删除缓存键
	// namespace: 命名空间前缀
	// keys: 缓存键列表
	DeleteWithNamespace(ctx context.Context, namespace string, keys ...string) error

	// DeleteNamespace 删除整个命名空间下的所有键
	// namespace: 命名空间前缀
	DeleteNamespace(ctx context.Context, namespace string) error

	// TTL 获取键的剩余生存时间
	// 返回剩余时间，如果键不存在返回负值
	TTL(ctx context.Context, key string) (time.Duration, error)

	// IsEnabled 检查缓存服务是否已启用
	IsEnabled() bool
}

// CacheKeyNotFoundError 缓存键不存在错误
type CacheKeyNotFoundError struct {
	Key string
}

func (e *CacheKeyNotFoundError) Error() string {
	return "缓存键不存在: " + e.Key
}

// ErrCacheKeyNotFound 缓存键不存在错误实例
var ErrCacheKeyNotFound = &CacheKeyNotFoundError{}
