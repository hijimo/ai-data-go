package storage

import (
	"context"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
)

// CacheOptimizer 缓存优化器
type CacheOptimizer struct {
	cache  *MultiLevelCache
	logger logger.Logger

	// 布隆过滤器（简化实现）
	bloomFilter map[string]bool
	bloomMutex  sync.RWMutex

	// 热点数据追踪
	hotKeys map[string]*keyStats
	hotMutex sync.RWMutex
}

type keyStats struct {
	accessCount int64
	lastAccess  time.Time
}

// NewCacheOptimizer 创建缓存优化器
func NewCacheOptimizer(cache *MultiLevelCache, log logger.Logger) *CacheOptimizer {
	optimizer := &CacheOptimizer{
		cache:       cache,
		logger:      log,
		bloomFilter: make(map[string]bool),
		hotKeys:     make(map[string]*keyStats),
	}

	// 启动热点数据追踪清理
	go optimizer.startHotKeysCleanup()

	return optimizer
}

// GetWithProtection 带保护的获取缓存
func (o *CacheOptimizer) GetWithProtection(ctx context.Context, key string, dest interface{}, loader func() (interface{}, error)) error {
	// 1. 布隆过滤器检查（防止缓存穿透）
	if !o.mightExist(key) {
		o.logger.DebugContext(ctx, "布隆过滤器拦截", logger.Fields{"key": key})
		
		// 尝试加载数据
		value, err := loader()
		if err != nil {
			return err
		}

		// 如果数据存在，添加到布隆过滤器并缓存
		if value != nil {
			o.addToBloomFilter(key)
			return o.cache.Set(ctx, key, value, 10*time.Minute)
		}

		// 数据不存在，缓存空值
		return o.cache.SetNullValue(ctx, key)
	}

	// 2. 尝试从缓存获取
	err := o.cache.Get(ctx, key, dest)
	if err == nil {
		// 记录热点访问
		o.recordAccess(key)
		return nil
	}

	// 3. 缓存未命中，使用单飞模式加载
	value, err := o.singleFlight(ctx, key, loader)
	if err != nil {
		return err
	}

	// 4. 判断是否为热点数据
	ttl := 10 * time.Minute
	if o.isHotKey(key) {
		// 热点数据使用更长的TTL
		ttl = 30 * time.Minute
		o.logger.DebugContext(ctx, "检测到热点数据", logger.Fields{
			"key": key,
			"ttl": ttl.String(),
		})
	}

	// 5. 设置缓存
	if value != nil {
		o.addToBloomFilter(key)
		if err := o.cache.Set(ctx, key, value, ttl); err != nil {
			return err
		}
		// 将值复制到dest
		return o.cache.Get(ctx, key, dest)
	}

	// 6. 数据不存在，缓存空值
	return o.cache.SetNullValue(ctx, key)
}

// 单飞模式实现（防止缓存击穿）
var (
	singleFlightMutex sync.Mutex
	singleFlightCalls = make(map[string]*singleFlightCall)
)

type singleFlightCall struct {
	wg    sync.WaitGroup
	value interface{}
	err   error
}

func (o *CacheOptimizer) singleFlight(ctx context.Context, key string, loader func() (interface{}, error)) (interface{}, error) {
	singleFlightMutex.Lock()

	// 检查是否已有相同的调用在进行
	if call, exists := singleFlightCalls[key]; exists {
		singleFlightMutex.Unlock()
		o.logger.DebugContext(ctx, "等待单飞调用完成", logger.Fields{"key": key})
		call.wg.Wait()
		return call.value, call.err
	}

	// 创建新的调用
	call := &singleFlightCall{}
	call.wg.Add(1)
	singleFlightCalls[key] = call
	singleFlightMutex.Unlock()

	// 执行加载
	o.logger.DebugContext(ctx, "执行单飞调用", logger.Fields{"key": key})
	call.value, call.err = loader()
	call.wg.Done()

	// 清理
	singleFlightMutex.Lock()
	delete(singleFlightCalls, key)
	singleFlightMutex.Unlock()

	return call.value, call.err
}

// 布隆过滤器操作（简化实现）
func (o *CacheOptimizer) mightExist(key string) bool {
	o.bloomMutex.RLock()
	defer o.bloomMutex.RUnlock()
	return o.bloomFilter[key]
}

func (o *CacheOptimizer) addToBloomFilter(key string) {
	o.bloomMutex.Lock()
	defer o.bloomMutex.Unlock()
	o.bloomFilter[key] = true
}

func (o *CacheOptimizer) clearBloomFilter() {
	o.bloomMutex.Lock()
	defer o.bloomMutex.Unlock()
	o.bloomFilter = make(map[string]bool)
}

// 热点数据追踪
func (o *CacheOptimizer) recordAccess(key string) {
	o.hotMutex.Lock()
	defer o.hotMutex.Unlock()

	if stats, exists := o.hotKeys[key]; exists {
		stats.accessCount++
		stats.lastAccess = time.Now()
	} else {
		o.hotKeys[key] = &keyStats{
			accessCount: 1,
			lastAccess:  time.Now(),
		}
	}
}

func (o *CacheOptimizer) isHotKey(key string) bool {
	o.hotMutex.RLock()
	defer o.hotMutex.RUnlock()

	stats, exists := o.hotKeys[key]
	if !exists {
		return false
	}

	// 定义热点数据：5分钟内访问超过10次
	if time.Since(stats.lastAccess) < 5*time.Minute && stats.accessCount >= 10 {
		return true
	}

	return false
}

func (o *CacheOptimizer) startHotKeysCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		o.cleanupHotKeys()
	}
}

func (o *CacheOptimizer) cleanupHotKeys() {
	o.hotMutex.Lock()
	defer o.hotMutex.Unlock()

	now := time.Now()
	cleanedCount := 0

	for key, stats := range o.hotKeys {
		// 清理10分钟未访问的键
		if now.Sub(stats.lastAccess) > 10*time.Minute {
			delete(o.hotKeys, key)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		o.logger.Debug("热点数据清理完成", logger.Fields{
			"cleaned_count": cleanedCount,
			"remaining":     len(o.hotKeys),
		})
	}
}

// GetHotKeys 获取热点数据列表
func (o *CacheOptimizer) GetHotKeys() []string {
	o.hotMutex.RLock()
	defer o.hotMutex.RUnlock()

	hotKeys := make([]string, 0)
	for key, stats := range o.hotKeys {
		if time.Since(stats.lastAccess) < 5*time.Minute && stats.accessCount >= 10 {
			hotKeys = append(hotKeys, key)
		}
	}

	return hotKeys
}

// PrewarmCache 预热缓存
func (o *CacheOptimizer) PrewarmCache(ctx context.Context, keys []string, loader func(key string) (interface{}, error)) error {
	o.logger.InfoContext(ctx, "开始缓存预热", logger.Fields{
		"key_count": len(keys),
	})

	successCount := 0
	failCount := 0

	for _, key := range keys {
		value, err := loader(key)
		if err != nil {
			o.logger.WarnContext(ctx, "预热失败", logger.Fields{
				"key":   key,
				"error": err.Error(),
			})
			failCount++
			continue
		}

		if value != nil {
			if err := o.cache.Set(ctx, key, value, 30*time.Minute); err != nil {
				o.logger.WarnContext(ctx, "缓存设置失败", logger.Fields{
					"key":   key,
					"error": err.Error(),
				})
				failCount++
				continue
			}
			o.addToBloomFilter(key)
			successCount++
		}
	}

	o.logger.InfoContext(ctx, "缓存预热完成", logger.Fields{
		"success_count": successCount,
		"fail_count":    failCount,
	})

	return nil
}

// GetOptimizationStats 获取优化统计信息
func (o *CacheOptimizer) GetOptimizationStats() map[string]interface{} {
	o.hotMutex.RLock()
	hotKeyCount := len(o.hotKeys)
	o.hotMutex.RUnlock()

	o.bloomMutex.RLock()
	bloomSize := len(o.bloomFilter)
	o.bloomMutex.RUnlock()

	cacheStats := o.cache.GetStats()

	return map[string]interface{}{
		"hot_key_count":   hotKeyCount,
		"bloom_size":      bloomSize,
		"cache_stats":     cacheStats,
		"hot_keys":        o.GetHotKeys(),
	}
}
