package service

import (
	"context"
	"time"

	"genkit-ai-service/internal/monitoring"
)

// CacheMetricsWrapper 缓存指标包装器
// 用于在缓存操作时自动记录指标
type CacheMetricsWrapper struct {
	cache   CacheService
	metrics *monitoring.GenkitMetrics
}

// NewCacheMetricsWrapper 创建缓存指标包装器
func NewCacheMetricsWrapper(cache CacheService, metrics *monitoring.GenkitMetrics) CacheService {
	return &CacheMetricsWrapper{
		cache:   cache,
		metrics: metrics,
	}
}

// Get 获取缓存（带指标记录）
func (w *CacheMetricsWrapper) Get(ctx context.Context, key string, dest interface{}) error {
	cacheType := getCacheTypeFromKey(key)
	tenantID := getTenantIDFromContext(ctx)

	err := w.cache.Get(ctx, key, dest)

	if err == nil {
		// 缓存命中
		w.metrics.RecordCacheHit(cacheType, tenantID)
	} else if err == ErrCacheNotFound {
		// 缓存未命中
		w.metrics.RecordCacheMiss(cacheType, tenantID)
	}

	return err
}

// Set 设置缓存
func (w *CacheMetricsWrapper) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return w.cache.Set(ctx, key, value, ttl)
}

// Delete 删除缓存
func (w *CacheMetricsWrapper) Delete(ctx context.Context, keys ...string) error {
	return w.cache.Delete(ctx, keys...)
}

// DeletePattern 按模式删除缓存
func (w *CacheMetricsWrapper) DeletePattern(ctx context.Context, pattern string) error {
	return w.cache.DeletePattern(ctx, pattern)
}

// Exists 检查缓存是否存在
func (w *CacheMetricsWrapper) Exists(ctx context.Context, key string) (bool, error) {
	return w.cache.Exists(ctx, key)
}

// Increment 增加缓存值
func (w *CacheMetricsWrapper) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return w.cache.Increment(ctx, key, delta)
}

// getCacheTypeFromKey 从缓存键获取缓存类型
func getCacheTypeFromKey(key string) string {
	// 解析缓存键前缀来确定类型
	// 例如: "context:session-id:query-hash" -> "context"
	//      "vector:session-id:query-hash" -> "vector"
	//      "summary:session-id:latest" -> "summary"

	for i, c := range key {
		if c == ':' {
			return key[:i]
		}
	}

	return "unknown"
}

// getTenantIDFromContext 从上下文获取租户 ID
func getTenantIDFromContext(ctx context.Context) string {
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return "unknown"
}

// CacheMetricsCollector 缓存指标收集器
// 用于定期收集和报告缓存统计信息
type CacheMetricsCollector struct {
	metrics *monitoring.GenkitMetrics
	cache   CacheService
}

// NewCacheMetricsCollector 创建缓存指标收集器
func NewCacheMetricsCollector(metrics *monitoring.GenkitMetrics, cache CacheService) *CacheMetricsCollector {
	return &CacheMetricsCollector{
		metrics: metrics,
		cache:   cache,
	}
}

// StartCollection 开始收集缓存指标
func (c *CacheMetricsCollector) StartCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectMetrics(ctx)
		}
	}
}

// collectMetrics 收集缓存指标
func (c *CacheMetricsCollector) collectMetrics(ctx context.Context) {
	// 这里可以收集更多的缓存统计信息
	// 例如：缓存大小、内存使用、过期键数量等

	// 注意：具体实现取决于使用的缓存系统（Redis、Memcached 等）
	// 这里提供一个基本框架
}

// CacheHitRateCalculator 缓存命中率计算器
type CacheHitRateCalculator struct {
	metrics *monitoring.GenkitMetrics
}

// NewCacheHitRateCalculator 创建缓存命中率计算器
func NewCacheHitRateCalculator(metrics *monitoring.GenkitMetrics) *CacheHitRateCalculator {
	return &CacheHitRateCalculator{
		metrics: metrics,
	}
}

// CalculateHitRate 计算缓存命中率
func (c *CacheHitRateCalculator) CalculateHitRate(cacheType, tenantID string) float64 {
	// 从 Prometheus 指标计算命中率
	// 命中率 = 命中次数 / (命中次数 + 未命中次数)
	return c.metrics.GetCacheHitRate(cacheType, tenantID)
}
