package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// memoryRepository 记忆数据访问实现
type memoryRepository struct {
	db *gorm.DB
}

// NewMemoryRepository 创建记忆数据访问实例
func NewMemoryRepository(db *gorm.DB) MemoryRepository {
	return &memoryRepository{
		db: db,
	}
}

// Create 创建记忆记录
func (r *memoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return fmt.Errorf("创建记忆记录失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取记忆
func (r *memoryRepository) GetByID(ctx context.Context, tenantID, memoryID uuid.UUID) (*model.ConversationMemory, error) {
	var memory model.ConversationMemory
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", memoryID, tenantID, false).
		First(&memory).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("查询记忆失败: %w", err)
	}

	return &memory, nil
}

// SearchByVector 向量相似度搜索（会话内）
// 注意：此方法仅返回元数据，实际向量检索由 VectorService 完成
func (r *memoryRepository) SearchByVector(ctx context.Context, tenantID, sessionID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error) {
	if len(memoryIDs) == 0 {
		return []*model.ConversationMemory{}, nil
	}

	var memories []*model.ConversationMemory
	err := r.db.WithContext(ctx).
		Where("id IN ? AND tenant_id = ? AND session_id = ? AND is_deleted = ?", memoryIDs, tenantID, sessionID, false).
		Order("importance DESC, created_at DESC").
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("查询记忆元数据失败: %w", err)
	}

	return memories, nil
}

// SearchByVectorCrossSessions 跨会话向量搜索（租户内）
// 注意：此方法仅返回元数据，实际向量检索由 VectorService 完成
func (r *memoryRepository) SearchByVectorCrossSessions(ctx context.Context, tenantID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error) {
	if len(memoryIDs) == 0 {
		return []*model.ConversationMemory{}, nil
	}

	var memories []*model.ConversationMemory
	err := r.db.WithContext(ctx).
		Where("id IN ? AND tenant_id = ? AND is_deleted = ?", memoryIDs, tenantID, false).
		Order("importance DESC, created_at DESC").
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("查询跨会话记忆元数据失败: %w", err)
	}

	return memories, nil
}

// UpdateAccessStats 更新访问统计
func (r *memoryRepository) UpdateAccessStats(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", memoryID, tenantID, false).
		Updates(map[string]interface{}{
			"access_count":   gorm.Expr("access_count + ?", 1),
			"last_access_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("更新访问统计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteByStrategy 按策略删除记忆
func (r *memoryRepository) DeleteByStrategy(ctx context.Context, tenantID uuid.UUID, strategy DeleteStrategy, mode DeleteMode) (int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ConversationMemory{}).
		Where("tenant_id = ?", tenantID)

	// 根据策略构建查询条件
	switch strategy {
	case DeleteStrategyExpired:
		// 删除已过期的记忆
		query = query.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now())
		if mode == DeleteModeSoft {
			query = query.Where("is_deleted = ?", false)
		}

	case DeleteStrategyLowQuality:
		// 删除低质量记忆（重要性低于0.3且访问次数少于2）
		query = query.Where("importance < ? AND access_count < ?", 0.3, 2)
		if mode == DeleteModeSoft {
			query = query.Where("is_deleted = ?", false)
		}

	case DeleteStrategyUnused:
		// 删除90天未访问的记忆
		ninetyDaysAgo := time.Now().AddDate(0, 0, -90)
		query = query.Where("(last_access_at IS NULL AND created_at < ?) OR (last_access_at IS NOT NULL AND last_access_at < ?)",
			ninetyDaysAgo, ninetyDaysAgo)
		if mode == DeleteModeSoft {
			query = query.Where("is_deleted = ?", false)
		}

	case DeleteStrategyAll:
		// 删除所有记忆
		if mode == DeleteModeSoft {
			query = query.Where("is_deleted = ?", false)
		}

	default:
		return 0, fmt.Errorf("不支持的删除策略: %s", strategy)
	}

	var result *gorm.DB
	if mode == DeleteModeSoft {
		// 软删除
		result = query.Update("is_deleted", true)
	} else {
		// 硬删除
		result = query.Delete(&model.ConversationMemory{})
	}

	if result.Error != nil {
		return 0, fmt.Errorf("按策略删除记忆失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetExpiredMemories 获取过期记忆
func (r *memoryRepository) GetExpiredMemories(ctx context.Context, tenantID uuid.UUID, limit int) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND expires_at IS NOT NULL AND expires_at < ? AND is_deleted = ?",
			tenantID, time.Now(), false).
		Order("expires_at ASC").
		Limit(limit).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("查询过期记忆失败: %w", err)
	}

	return memories, nil
}

// BatchCreate 批量创建记忆记录
func (r *memoryRepository) BatchCreate(ctx context.Context, memories []*model.ConversationMemory) error {
	if len(memories) == 0 {
		return nil
	}

	// 使用批量插入，每批最多1000条
	batchSize := 1000
	for i := 0; i < len(memories); i += batchSize {
		end := i + batchSize
		if end > len(memories) {
			end = len(memories)
		}

		batch := memories[i:end]
		if err := r.db.WithContext(ctx).Create(&batch).Error; err != nil {
			return fmt.Errorf("批量创建记忆记录失败（批次 %d-%d）: %w", i, end, err)
		}
	}

	return nil
}

// GetBySessionID 获取会话的所有记忆
func (r *memoryRepository) GetBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID, memoryType string, limit int) ([]*model.ConversationMemory, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND session_id = ? AND is_deleted = ?", tenantID, sessionID, false)

	// 如果指定了记忆类型，添加过滤条件
	if memoryType != "" {
		query = query.Where("memory_type = ?", memoryType)
	}

	// 按创建时间倒序排列
	query = query.Order("created_at DESC")

	// 如果指定了限制数量
	if limit > 0 {
		query = query.Limit(limit)
	}

	var memories []*model.ConversationMemory
	err := query.Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("查询会话记忆失败: %w", err)
	}

	return memories, nil
}

// SoftDelete 软删除记忆
func (r *memoryRepository) SoftDelete(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", memoryID, tenantID, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("软删除记忆失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// HardDelete 硬删除记忆
func (r *memoryRepository) HardDelete(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", memoryID, tenantID).
		Delete(&model.ConversationMemory{})

	if result.Error != nil {
		return fmt.Errorf("硬删除记忆失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
