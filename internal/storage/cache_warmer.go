package storage

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CacheWarmer 缓存预热器
type CacheWarmer struct {
	cache             CacheService
	contextRepo       repository.ContextRepository
	summaryRepo       repository.SummaryRepository
	sessionRepo       repository.SessionRepository
	db                *gorm.DB
	logger            logger.Logger
	stopChan          chan struct{}
	warmupInterval    time.Duration
	activeSessionDays int // 活跃会话的天数阈值
}

// CacheWarmerConfig 缓存预热配置
type CacheWarmerConfig struct {
	WarmupInterval    time.Duration // 定期预热间隔
	ActiveSessionDays int           // 活跃会话的天数阈值（默认7天）
}

// NewCacheWarmer 创建缓存预热器实例
func NewCacheWarmer(
	cache CacheService,
	contextRepo repository.ContextRepository,
	summaryRepo repository.SummaryRepository,
	sessionRepo repository.SessionRepository,
	db *gorm.DB,
	log logger.Logger,
	config *CacheWarmerConfig,
) *CacheWarmer {
	if config == nil {
		config = &CacheWarmerConfig{
			WarmupInterval:    30 * time.Minute, // 默认30分钟
			ActiveSessionDays: 7,                // 默认7天
		}
	}

	return &CacheWarmer{
		cache:             cache,
		contextRepo:       contextRepo,
		summaryRepo:       summaryRepo,
		sessionRepo:       sessionRepo,
		db:                db,
		logger:            log,
		stopChan:          make(chan struct{}),
		warmupInterval:    config.WarmupInterval,
		activeSessionDays: config.ActiveSessionDays,
	}
}

// WarmupOnStartup 启动时预热缓存
// 预热活跃会话的上下文配置和摘要
func (w *CacheWarmer) WarmupOnStartup(ctx context.Context) error {
	if !w.cache.IsEnabled() {
		w.logger.WarnContext(ctx, "缓存服务未启用，跳过预热")
		return nil
	}

	w.logger.InfoContext(ctx, "开始启动时缓存预热")
	startTime := time.Now()

	// 预热活跃会话
	sessionCount, err := w.warmupActiveSessions(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "预热活跃会话失败", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("预热活跃会话失败: %w", err)
	}

	duration := time.Since(startTime)
	w.logger.InfoContext(ctx, "启动时缓存预热完成", logger.Fields{
		"session_count": sessionCount,
		"duration_ms":   duration.Milliseconds(),
	})

	return nil
}

// warmupActiveSessions 预热活跃会话
// 活跃会话定义：最近N天内有更新的会话
func (w *CacheWarmer) warmupActiveSessions(ctx context.Context) (int, error) {
	// 计算活跃会话的时间阈值
	activeThreshold := time.Now().AddDate(0, 0, -w.activeSessionDays)

	// 查询活跃会话
	var activeSessions []model.ChatSession
	err := w.db.WithContext(ctx).
		Where("updated_at >= ? AND is_deleted = ?", activeThreshold, false).
		Order("updated_at DESC").
		Limit(100). // 限制预热数量，避免启动时间过长
		Find(&activeSessions).Error

	if err != nil {
		return 0, fmt.Errorf("查询活跃会话失败: %w", err)
	}

	w.logger.InfoContext(ctx, "找到活跃会话", logger.Fields{
		"count":             len(activeSessions),
		"active_threshold":  activeThreshold.Format(time.RFC3339),
		"threshold_days":    w.activeSessionDays,
	})

	// 预热每个活跃会话的数据
	successCount := 0
	for _, session := range activeSessions {
		if err := w.warmupSession(ctx, session.ID); err != nil {
			w.logger.WarnContext(ctx, "预热会话失败", logger.Fields{
				"session_id": session.ID.String(),
				"error":      err.Error(),
			})
			continue
		}
		successCount++
	}

	return successCount, nil
}

// warmupSession 预热单个会话的缓存
func (w *CacheWarmer) warmupSession(ctx context.Context, sessionID uuid.UUID) error {
	// 1. 预热上下文配置
	if err := w.warmupContextConfig(ctx, sessionID); err != nil {
		// 上下文配置可能不存在，记录调试日志但不返回错误
		w.logger.DebugContext(ctx, "预热上下文配置失败（可能不存在）", logger.Fields{
			"session_id": sessionID.String(),
			"error":      err.Error(),
		})
	}

	// 2. 预热最新摘要
	if err := w.warmupLatestSummary(ctx, sessionID); err != nil {
		// 摘要可能不存在，不视为错误
		w.logger.DebugContext(ctx, "预热摘要失败（可能不存在）", logger.Fields{
			"session_id": sessionID.String(),
			"error":      err.Error(),
		})
	}

	return nil
}

// warmupContextConfig 预热上下文配置
func (w *CacheWarmer) warmupContextConfig(ctx context.Context, sessionID uuid.UUID) error {
	// 构建缓存键
	cacheKey := fmt.Sprintf("context:config:%s", sessionID.String())

	// 检查缓存是否已存在
	exists, err := w.cache.Exists(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	if exists > 0 {
		// 缓存已存在，跳过
		return nil
	}

	// 从数据库查询上下文配置
	contextConfig, err := w.contextRepo.GetBySessionID(ctx, sessionID.String())
	if err != nil {
		// 上下文配置可能不存在（会话刚创建时），这是正常的，不视为错误
		w.logger.DebugContext(ctx, "上下文配置不存在，跳过预热", logger.Fields{
			"session_id": sessionID.String(),
			"error":      err.Error(),
		})
		return nil
	}

	// 设置缓存（TTL: 5分钟）
	if err := w.cache.Set(ctx, cacheKey, contextConfig, 5*time.Minute); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	w.logger.DebugContext(ctx, "预热上下文配置成功", logger.Fields{
		"session_id": sessionID.String(),
		"cache_key":  cacheKey,
	})

	return nil
}

// warmupLatestSummary 预热最新摘要
func (w *CacheWarmer) warmupLatestSummary(ctx context.Context, sessionID uuid.UUID) error {
	// 构建缓存键
	cacheKey := fmt.Sprintf("summary:latest:%s", sessionID.String())

	// 检查缓存是否已存在
	exists, err := w.cache.Exists(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	if exists > 0 {
		// 缓存已存在，跳过
		return nil
	}

	// 从数据库查询最新摘要
	// 注意：需要获取租户ID，这里我们从会话中获取
	var session model.ChatSession
	if err := w.db.WithContext(ctx).
		Select("id").
		Where("id = ? AND is_deleted = ?", sessionID, false).
		First(&session).Error; err != nil {
		return fmt.Errorf("查询会话失败: %w", err)
	}

	// 查询最新摘要（使用 contextRepo 的方法）
	summary, err := w.contextRepo.GetLatestSummary(ctx, sessionID.String())
	if err != nil {
		return fmt.Errorf("查询最新摘要失败: %w", err)
	}

	if summary == nil {
		// 摘要不存在，不是错误
		return nil
	}

	// 设置缓存（TTL: 1小时）
	if err := w.cache.Set(ctx, cacheKey, summary, 1*time.Hour); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	w.logger.DebugContext(ctx, "预热最新摘要成功", logger.Fields{
		"session_id": sessionID.String(),
		"summary_id": summary.ID.String(),
		"cache_key":  cacheKey,
	})

	return nil
}

// StartPeriodicWarmup 启动定期预热
// 在后台定期预热活跃会话的缓存
func (w *CacheWarmer) StartPeriodicWarmup(ctx context.Context) {
	if !w.cache.IsEnabled() {
		w.logger.WarnContext(ctx, "缓存服务未启用，跳过定期预热")
		return
	}

	w.logger.InfoContext(ctx, "启动定期缓存预热", logger.Fields{
		"interval_minutes": w.warmupInterval.Minutes(),
	})

	// 启动定期预热协程
	go w.periodicWarmupLoop(ctx)
}

// periodicWarmupLoop 定期预热循环
func (w *CacheWarmer) periodicWarmupLoop(ctx context.Context) {
	ticker := time.NewTicker(w.warmupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 执行预热
			w.logger.DebugContext(ctx, "开始定期缓存预热")
			startTime := time.Now()

			sessionCount, err := w.warmupActiveSessions(ctx)
			if err != nil {
				w.logger.ErrorContext(ctx, "定期预热失败", logger.Fields{
					"error": err.Error(),
				})
			} else {
				duration := time.Since(startTime)
				w.logger.InfoContext(ctx, "定期缓存预热完成", logger.Fields{
					"session_count": sessionCount,
					"duration_ms":   duration.Milliseconds(),
				})
			}

		case <-w.stopChan:
			// 停止预热
			w.logger.InfoContext(ctx, "停止定期缓存预热")
			return

		case <-ctx.Done():
			// 上下文取消
			w.logger.InfoContext(ctx, "上下文取消，停止定期缓存预热")
			return
		}
	}
}

// Stop 停止定期预热
func (w *CacheWarmer) Stop() {
	close(w.stopChan)
}

// WarmupSessionList 预热会话列表缓存
// 用于预热用户的会话列表
func (w *CacheWarmer) WarmupSessionList(ctx context.Context, userID uuid.UUID, page, pageSize int) error {
	if !w.cache.IsEnabled() {
		return nil
	}

	// 构建缓存键
	cacheKey := fmt.Sprintf("session:list:%s:%d:%d", userID.String(), page, pageSize)

	// 检查缓存是否已存在
	exists, err := w.cache.Exists(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	if exists > 0 {
		// 缓存已存在，跳过
		return nil
	}

	// 从数据库查询会话列表
	sessions, total, err := w.sessionRepo.GetByUserID(ctx, userID.String(), page, pageSize, nil)
	if err != nil {
		return fmt.Errorf("查询会话列表失败: %w", err)
	}

	// 构建缓存数据
	cacheData := map[string]interface{}{
		"sessions": sessions,
		"total":    total,
	}

	// 设置缓存（TTL: 10分钟）
	if err := w.cache.Set(ctx, cacheKey, cacheData, 10*time.Minute); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	w.logger.DebugContext(ctx, "预热会话列表成功", logger.Fields{
		"user_id":       userID.String(),
		"page":          page,
		"page_size":     pageSize,
		"session_count": len(sessions),
		"cache_key":     cacheKey,
	})

	return nil
}

// WarmupTokenUsage 预热Token使用统计缓存
func (w *CacheWarmer) WarmupTokenUsage(ctx context.Context, sessionID uuid.UUID) error {
	if !w.cache.IsEnabled() {
		return nil
	}

	// 构建缓存键
	cacheKey := fmt.Sprintf("token:usage:%s", sessionID.String())

	// 检查缓存是否已存在
	exists, err := w.cache.Exists(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	if exists > 0 {
		// 缓存已存在，跳过
		return nil
	}

	// 从数据库查询上下文配置（包含Token使用统计）
	contextConfig, err := w.contextRepo.GetBySessionID(ctx, sessionID.String())
	if err != nil {
		// 上下文配置可能不存在，这是正常的，不视为错误
		return nil
	}

	// 构建Token使用统计数据
	tokenUsage := map[string]interface{}{
		"total_tokens_used": contextConfig.TotalTokensUsed,
		"total_messages":    contextConfig.TotalMessages,
	}

	// 设置缓存（TTL: 5分钟）
	if err := w.cache.Set(ctx, cacheKey, tokenUsage, 5*time.Minute); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	w.logger.DebugContext(ctx, "预热Token使用统计成功", logger.Fields{
		"session_id": sessionID.String(),
		"cache_key":  cacheKey,
	})

	return nil
}
