package service

import (
	"context"
	"time"

	"genkit-ai-service/internal/logger"
)

// CacheWarmer 缓存预热器
type CacheWarmer struct {
	cache     CacheService
	cacheKeys *CacheKeys
	logger    logger.Logger
}

// NewCacheWarmer 创建新的缓存预热器实例
func NewCacheWarmer(
	cache CacheService,
	cacheKeys *CacheKeys,
	log logger.Logger,
) *CacheWarmer {
	return &CacheWarmer{
		cache:     cache,
		cacheKeys: cacheKeys,
		logger:    log,
	}
}

// WarmupOnStartup 启动时执行缓存预热
func (w *CacheWarmer) WarmupOnStartup(ctx context.Context) error {
	w.logger.Info("开始缓存预热...")

	startTime := time.Now()

	// 注意：实际的预热逻辑需要在具体的服务层实现
	// 这里只提供框架，具体的预热策略由各个服务决定
	// 例如：预热活跃会话、预热常用配置等

	duration := time.Since(startTime)
	w.logger.Info("缓存预热完成", logger.Fields{
		"duration_ms": duration.Milliseconds(),
	})

	return nil
}

// StartPeriodicWarmup 启动定期预热
func (w *CacheWarmer) StartPeriodicWarmup(ctx context.Context, interval time.Duration) {
	w.logger.Info("启动定期缓存预热", logger.Fields{
		"interval": interval.String(),
	})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("停止定期缓存预热")
			return
		case <-ticker.C:
			w.logger.Debug("执行定期缓存预热")
			if err := w.WarmupOnStartup(ctx); err != nil {
				w.logger.Warn("定期预热失败", logger.Fields{
					"error": err.Error(),
				})
			}
		}
	}
}

// InvalidateSession 使会话相关的所有缓存失效
func (w *CacheWarmer) InvalidateSession(ctx context.Context, sessionID string) error {
	w.logger.Debug("使会话缓存失效", logger.Fields{
		"session_id": sessionID,
	})

	// 删除上下文配置缓存
	contextConfigKey := w.cacheKeys.ContextConfigKey(sessionID)
	if err := w.cache.Delete(ctx, contextConfigKey); err != nil {
		w.logger.Warn("删除上下文配置缓存失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
	}

	// 删除摘要缓存
	summaryKey := w.cacheKeys.SummaryKey(sessionID)
	if err := w.cache.Delete(ctx, summaryKey); err != nil {
		w.logger.Warn("删除摘要缓存失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
	}

	// 删除 Token 使用统计缓存
	tokenUsageKey := w.cacheKeys.TokenUsageKey(sessionID)
	if err := w.cache.Delete(ctx, tokenUsageKey); err != nil {
		w.logger.Warn("删除 Token 使用统计缓存失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
	}

	// 删除所有相关的上下文和向量检索缓存
	pattern := w.cacheKeys.SessionPattern(sessionID)
	if err := w.cache.DeletePattern(ctx, pattern); err != nil {
		w.logger.Warn("删除会话相关缓存失败", logger.Fields{
			"session_id": sessionID,
			"pattern":    pattern,
			"error":      err.Error(),
		})
	}

	return nil
}

// InvalidateUser 使用户相关的所有缓存失效
func (w *CacheWarmer) InvalidateUser(ctx context.Context, userID string) error {
	w.logger.Debug("使用户缓存失效", logger.Fields{
		"user_id": userID,
	})

	// 删除会话列表缓存
	sessionListKey := w.cacheKeys.SessionListKey(userID)
	if err := w.cache.Delete(ctx, sessionListKey); err != nil {
		w.logger.Warn("删除会话列表缓存失败", logger.Fields{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	// 删除所有相关缓存
	pattern := w.cacheKeys.UserPattern(userID)
	if err := w.cache.DeletePattern(ctx, pattern); err != nil {
		w.logger.Warn("删除用户相关缓存失败", logger.Fields{
			"user_id": userID,
			"pattern": pattern,
			"error":   err.Error(),
		})
	}

	return nil
}

// InvalidateTenant 使租户相关的所有缓存失效
func (w *CacheWarmer) InvalidateTenant(ctx context.Context, tenantID string) error {
	w.logger.Debug("使租户缓存失效", logger.Fields{
		"tenant_id": tenantID,
	})

	// 删除配额缓存
	dailyQuotaKey := w.cacheKeys.QuotaKey(tenantID, "daily")
	monthlyQuotaKey := w.cacheKeys.QuotaKey(tenantID, "monthly")
	if err := w.cache.Delete(ctx, dailyQuotaKey, monthlyQuotaKey); err != nil {
		w.logger.Warn("删除配额缓存失败", logger.Fields{
			"tenant_id": tenantID,
			"error":     err.Error(),
		})
	}

	// 删除所有相关缓存
	pattern := w.cacheKeys.TenantPattern(tenantID)
	if err := w.cache.DeletePattern(ctx, pattern); err != nil {
		w.logger.Warn("删除租户相关缓存失败", logger.Fields{
			"tenant_id": tenantID,
			"pattern":   pattern,
			"error":     err.Error(),
		})
	}

	return nil
}

// RefreshSessionContext 刷新会话上下文缓存
func (w *CacheWarmer) RefreshSessionContext(ctx context.Context, sessionID string) error {
	w.logger.Debug("刷新会话上下文缓存", logger.Fields{
		"session_id": sessionID,
	})

	// 使缓存失效
	if err := w.InvalidateSession(ctx, sessionID); err != nil {
		w.logger.Warn("使会话缓存失效失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return err
	}

	// 注意：实际的预热逻辑应该由调用方在使缓存失效后按需加载

	return nil
}

// GetCacheStats 获取缓存统计信息
func (w *CacheWarmer) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 这里可以添加更多的统计信息
	// 例如：缓存命中率、缓存大小等
	// 需要根据实际的 Redis 客户端实现来获取这些信息

	stats["status"] = "active"
	stats["timestamp"] = time.Now().Format(time.RFC3339)

	return stats, nil
}
