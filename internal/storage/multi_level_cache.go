package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
)

// 本地缓存项
type localCacheItem struct {
	value      interface{}
	expiration time.Time
	version    string
}

// 多级缓存配置
type MultiLevelCacheConfig struct {
	// 本地缓存最大条目数
	LocalMaxEntries int
	// 本地缓存默认TTL
	LocalDefaultTTL time.Duration
	// 是否启用本地缓存
	EnableLocalCache bool
	// 缓存版本（用于全局失效）
	CacheVersion string
	// 空值缓存TTL（防止缓存穿透）
	NullValueTTL time.Duration
	// 随机过期时间范围（防止缓存雪崩）
	ExpirationJitter time.Duration
}

// MultiLevelCache 多级缓存实现
type MultiLevelCache struct {
	// L1: 本地内存缓存
	localCache map[string]*localCacheItem
	localMutex sync.RWMutex
	
	// L2: Redis 缓存
	redisCache CacheService
	
	// 配置
	config MultiLevelCacheConfig
	
	// 日志
	logger logger.Logger
	
	// 统计信息
	stats struct {
		sync.RWMutex
		l1Hits   int64
		l1Misses int64
		l2Hits   int64
		l2Misses int64
	}
}

// NewMultiLevelCache 创建多级缓存实例
func NewMultiLevelCache(redisCache CacheService, config MultiLevelCacheConfig, log logger.Logger) *MultiLevelCache {
	if config.LocalMaxEntries == 0 {
		config.LocalMaxEntries = 1000
	}
	if config.LocalDefaultTTL == 0 {
		config.LocalDefaultTTL = 5 * time.Minute
	}
	if config.NullValueTTL == 0 {
		config.NullValueTTL = 1 * time.Minute
	}
	if config.ExpirationJitter == 0 {
		config.ExpirationJitter = 30 * time.Second
	}
	if config.CacheVersion == "" {
		config.CacheVersion = "v1"
	}

	mlc := &MultiLevelCache{
		localCache: make(map[string]*localCacheItem),
		redisCache: redisCache,
		config:     config,
		logger:     log,
	}

	// 启动本地缓存清理协程
	if config.EnableLocalCache {
		go mlc.startLocalCacheCleanup()
	}

	return mlc
}

// Get 获取缓存值（多级查询）
func (m *MultiLevelCache) Get(ctx context.Context, key string, dest interface{}) error {
	versionedKey := m.buildVersionedKey(key)

	// L1: 尝试从本地缓存获取
	if m.config.EnableLocalCache {
		if value, found := m.getFromLocal(versionedKey); found {
			m.recordL1Hit()
			m.logger.DebugContext(ctx, "L1缓存命中", logger.Fields{"key": key})
			return m.unmarshalValue(value, dest)
		}
		m.recordL1Miss()
	}

	// L2: 尝试从 Redis 获取
	err := m.redisCache.Get(ctx, versionedKey, dest)
	if err == nil {
		m.recordL2Hit()
		m.logger.DebugContext(ctx, "L2缓存命中", logger.Fields{"key": key})
		
		// 回填到本地缓存
		if m.config.EnableLocalCache {
			m.setToLocal(versionedKey, dest, m.config.LocalDefaultTTL)
		}
		return nil
	}

	m.recordL2Miss()
	
	// 检查是否是缓存穿透保护的空值
	if m.isNullValue(dest) {
		return &CacheKeyNotFoundError{Key: key}
	}

	return err
}

// Set 设置缓存值（写入所有层级）
func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	versionedKey := m.buildVersionedKey(key)
	
	// 添加随机抖动防止缓存雪崩
	jitteredExpiration := m.addJitter(expiration)

	// L2: 写入 Redis
	err := m.redisCache.Set(ctx, versionedKey, value, jitteredExpiration)
	if err != nil {
		m.logger.ErrorContext(ctx, "写入Redis缓存失败", logger.Fields{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	// L1: 写入本地缓存
	if m.config.EnableLocalCache {
		localTTL := m.config.LocalDefaultTTL
		if expiration > 0 && expiration < localTTL {
			localTTL = expiration
		}
		m.setToLocal(versionedKey, value, localTTL)
	}

	m.logger.DebugContext(ctx, "缓存已设置", logger.Fields{
		"key":        key,
		"expiration": jitteredExpiration.String(),
	})

	return nil
}

// SetNullValue 设置空值（防止缓存穿透）
func (m *MultiLevelCache) SetNullValue(ctx context.Context, key string) error {
	versionedKey := m.buildVersionedKey(key)
	nullMarker := map[string]interface{}{"__null__": true}
	
	// 使用较短的TTL存储空值
	err := m.redisCache.Set(ctx, versionedKey, nullMarker, m.config.NullValueTTL)
	if err != nil {
		return err
	}

	if m.config.EnableLocalCache {
		m.setToLocal(versionedKey, nullMarker, m.config.NullValueTTL)
	}

	m.logger.DebugContext(ctx, "缓存空值已设置（防穿透）", logger.Fields{"key": key})
	return nil
}

// Delete 删除缓存（所有层级）
func (m *MultiLevelCache) Delete(ctx context.Context, keys ...string) error {
	versionedKeys := make([]string, len(keys))
	for i, key := range keys {
		versionedKeys[i] = m.buildVersionedKey(key)
	}

	// L1: 从本地缓存删除
	if m.config.EnableLocalCache {
		m.deleteFromLocal(versionedKeys...)
	}

	// L2: 从 Redis 删除
	return m.redisCache.Delete(ctx, versionedKeys...)
}

// DeletePattern 按模式删除缓存
func (m *MultiLevelCache) DeletePattern(ctx context.Context, pattern string) error {
	versionedPattern := m.buildVersionedKey(pattern)
	
	// L1: 清空本地缓存（简单实现：清空所有）
	if m.config.EnableLocalCache {
		m.clearLocal()
	}

	// L2: 从 Redis 按模式删除
	return m.redisCache.DeletePattern(ctx, versionedPattern)
}

// InvalidateVersion 使整个缓存版本失效
func (m *MultiLevelCache) InvalidateVersion(ctx context.Context) error {
	// 更新版本号
	m.config.CacheVersion = m.generateNewVersion()
	
	// 清空本地缓存
	if m.config.EnableLocalCache {
		m.clearLocal()
	}

	m.logger.InfoContext(ctx, "缓存版本已失效", logger.Fields{
		"new_version": m.config.CacheVersion,
	})

	return nil
}

// GetStats 获取缓存统计信息
func (m *MultiLevelCache) GetStats() map[string]interface{} {
	m.stats.RLock()
	defer m.stats.RUnlock()

	l1Total := m.stats.l1Hits + m.stats.l1Misses
	l2Total := m.stats.l2Hits + m.stats.l2Misses

	var l1HitRate, l2HitRate float64
	if l1Total > 0 {
		l1HitRate = float64(m.stats.l1Hits) / float64(l1Total) * 100
	}
	if l2Total > 0 {
		l2HitRate = float64(m.stats.l2Hits) / float64(l2Total) * 100
	}

	m.localMutex.RLock()
	localSize := len(m.localCache)
	m.localMutex.RUnlock()

	return map[string]interface{}{
		"l1_hits":      m.stats.l1Hits,
		"l1_misses":    m.stats.l1Misses,
		"l1_hit_rate":  fmt.Sprintf("%.2f%%", l1HitRate),
		"l2_hits":      m.stats.l2Hits,
		"l2_misses":    m.stats.l2Misses,
		"l2_hit_rate":  fmt.Sprintf("%.2f%%", l2HitRate),
		"local_size":   localSize,
		"local_max":    m.config.LocalMaxEntries,
		"version":      m.config.CacheVersion,
	}
}

// 本地缓存操作
func (m *MultiLevelCache) getFromLocal(key string) (interface{}, bool) {
	m.localMutex.RLock()
	defer m.localMutex.RUnlock()

	item, found := m.localCache[key]
	if !found {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.expiration) {
		return nil, false
	}

	// 检查版本是否匹配
	if item.version != m.config.CacheVersion {
		return nil, false
	}

	return item.value, true
}

func (m *MultiLevelCache) setToLocal(key string, value interface{}, ttl time.Duration) {
	m.localMutex.Lock()
	defer m.localMutex.Unlock()

	// 检查是否超过最大条目数
	if len(m.localCache) >= m.config.LocalMaxEntries {
		// 简单的LRU：删除一个随机项
		for k := range m.localCache {
			delete(m.localCache, k)
			break
		}
	}

	m.localCache[key] = &localCacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
		version:    m.config.CacheVersion,
	}
}

func (m *MultiLevelCache) deleteFromLocal(keys ...string) {
	m.localMutex.Lock()
	defer m.localMutex.Unlock()

	for _, key := range keys {
		delete(m.localCache, key)
	}
}

func (m *MultiLevelCache) clearLocal() {
	m.localMutex.Lock()
	defer m.localMutex.Unlock()

	m.localCache = make(map[string]*localCacheItem)
}

// 本地缓存清理协程
func (m *MultiLevelCache) startLocalCacheCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredLocal()
	}
}

func (m *MultiLevelCache) cleanupExpiredLocal() {
	m.localMutex.Lock()
	defer m.localMutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for key, item := range m.localCache {
		if now.After(item.expiration) || item.version != m.config.CacheVersion {
			delete(m.localCache, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		m.logger.Debug("本地缓存清理完成", logger.Fields{
			"expired_count": expiredCount,
			"remaining":     len(m.localCache),
		})
	}
}

// 辅助方法
func (m *MultiLevelCache) buildVersionedKey(key string) string {
	return fmt.Sprintf("%s:%s", m.config.CacheVersion, key)
}

func (m *MultiLevelCache) addJitter(expiration time.Duration) time.Duration {
	if expiration == 0 || m.config.ExpirationJitter == 0 {
		return expiration
	}

	jitter := time.Duration(rand.Int63n(int64(m.config.ExpirationJitter)))
	return expiration + jitter
}

func (m *MultiLevelCache) generateNewVersion() string {
	hash := sha256.Sum256([]byte(time.Now().String()))
	return "v_" + hex.EncodeToString(hash[:8])
}

func (m *MultiLevelCache) isNullValue(value interface{}) bool {
	if m, ok := value.(map[string]interface{}); ok {
		if _, exists := m["__null__"]; exists {
			return true
		}
	}
	return false
}

func (m *MultiLevelCache) unmarshalValue(value interface{}, dest interface{}) error {
	// 序列化再反序列化以确保类型正确
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// 统计方法
func (m *MultiLevelCache) recordL1Hit() {
	m.stats.Lock()
	m.stats.l1Hits++
	m.stats.Unlock()
}

func (m *MultiLevelCache) recordL1Miss() {
	m.stats.Lock()
	m.stats.l1Misses++
	m.stats.Unlock()
}

func (m *MultiLevelCache) recordL2Hit() {
	m.stats.Lock()
	m.stats.l2Hits++
	m.stats.Unlock()
}

func (m *MultiLevelCache) recordL2Miss() {
	m.stats.Lock()
	m.stats.l2Misses++
	m.stats.Unlock()
}
