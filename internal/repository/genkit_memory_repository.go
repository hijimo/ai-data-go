package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// GenkitMemoryRepository Genkit会话记忆数据访问接口
type GenkitMemoryRepository interface {
	// Create 创建记忆
	Create(ctx context.Context, memory *model.ConversationMemory) error

	// GetByID 根据ID获取记忆
	GetByID(ctx context.Context, id string) (*model.ConversationMemory, error)

	// SearchByVector 向量相似度搜索（单会话）
	SearchByVector(
		ctx context.Context,
		sessionID string,
		embedding pgvector.Vector,
		topK int,
		minSimilarity float32,
	) ([]*model.ConversationMemory, error)

	// SearchByVectorCrossSessions 跨会话向量搜索（同租户内）
	SearchByVectorCrossSessions(
		ctx context.Context,
		tenantID string,
		embedding pgvector.Vector,
		topK int,
		minSimilarity float32,
	) ([]*model.ConversationMemory, error)

	// SearchByVectorWithFilters 带过滤条件的向量搜索
	SearchByVectorWithFilters(
		ctx context.Context,
		sessionID string,
		embedding pgvector.Vector,
		filters *MemorySearchFilters,
	) ([]*model.ConversationMemory, error)

	// UpdateAccessStats 更新访问统计
	UpdateAccessStats(ctx context.Context, id string) error

	// BatchUpdateAccessStats 批量更新访问统计
	BatchUpdateAccessStats(ctx context.Context, ids []string) error

	// DeleteByStrategy 按策略删除记忆
	DeleteByStrategy(
		ctx context.Context,
		tenantID string,
		strategy string,
		mode string,
		batchSize int,
	) (int, error)

	// GetExpiredMemories 获取过期记忆
	GetExpiredMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error)

	// GetLowQualityMemories 获取低质量记忆
	GetLowQualityMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error)

	// GetUnusedMemories 获取长期未使用的记忆
	GetUnusedMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error)

	// GetAllMemoriesForCleanup 获取所有待清理的记忆
	GetAllMemoriesForCleanup(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error)

	// SoftDeleteBatch 批量软删除记忆
	SoftDeleteBatch(ctx context.Context, ids []string) (int, error)

	// HardDeleteBatch 批量硬删除记忆
	HardDeleteBatch(ctx context.Context, ids []string) (int, error)

	// CountBySession 统计会话的记忆数量
	CountBySession(ctx context.Context, sessionID string) (int, error)

	// CountByTenant 统计租户的记忆数量
	CountByTenant(ctx context.Context, tenantID string) (int, error)

	// Update 更新记忆
	Update(ctx context.Context, memory *model.ConversationMemory) error

	// SoftDelete 软删除记忆
	SoftDelete(ctx context.Context, id string) error

	// HardDelete 硬删除记忆
	HardDelete(ctx context.Context, id string) error

	// SearchByContent 全文搜索记忆（降级方案）
	SearchByContent(ctx context.Context, sessionID, query string, topK int) ([]*model.ConversationMemory, error)
}

// MemorySearchFilters 记忆搜索过滤条件
type MemorySearchFilters struct {
	TopK          int
	MinSimilarity float32
	MemoryTypes   []string
	TimeRangeDays int
	MinImportance *float32
}

// MemoryCleanupFilters 记忆清理过滤条件
type MemoryCleanupFilters struct {
	TenantID  string
	SessionID string
	Strategy  string
	BatchSize int
}

// genkitMemoryRepository Genkit会话记忆数据访问实现
type genkitMemoryRepository struct {
	db *gorm.DB
}

// NewGenkitMemoryRepository 创建Genkit会话记忆数据访问实例
func NewGenkitMemoryRepository(db *gorm.DB) GenkitMemoryRepository {
	return &genkitMemoryRepository{
		db: db,
	}
}

// Create 创建记忆
func (r *genkitMemoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return fmt.Errorf("创建记忆失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取记忆
func (r *genkitMemoryRepository) GetByID(ctx context.Context, id string) (*model.ConversationMemory, error) {
	var memory model.ConversationMemory
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&memory).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("记忆不存在")
		}
		return nil, fmt.Errorf("查询记忆失败: %w", err)
	}

	return &memory, nil
}

// SearchByVector 向量相似度搜索（单会话）
func (r *genkitMemoryRepository) SearchByVector(
	ctx context.Context,
	sessionID string,
	embedding pgvector.Vector,
	topK int,
	minSimilarity float32,
) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	// 使用余弦相似度搜索
	// <=> 是 pgvector 的余弦距离操作符
	// 1 - 余弦距离 = 余弦相似度
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Where("is_deleted = ?", false).
		Where("(1 - (embedding <=> ?)) >= ?", embedding, minSimilarity).
		Order(gorm.Expr("embedding <=> ?", embedding)).
		Limit(topK).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	return memories, nil
}

// SearchByVectorCrossSessions 跨会话向量搜索（同租户内）
func (r *genkitMemoryRepository) SearchByVectorCrossSessions(
	ctx context.Context,
	tenantID string,
	embedding pgvector.Vector,
	topK int,
	minSimilarity float32,
) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	// 跨会话检索，但限制在同一租户内
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("is_deleted = ?", false).
		Where("(1 - (embedding <=> ?)) >= ?", embedding, minSimilarity).
		Order(gorm.Expr("embedding <=> ?", embedding)).
		Limit(topK).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("跨会话向量检索失败: %w", err)
	}

	return memories, nil
}

// SearchByVectorWithFilters 带过滤条件的向量搜索
func (r *genkitMemoryRepository) SearchByVectorWithFilters(
	ctx context.Context,
	sessionID string,
	embedding pgvector.Vector,
	filters *MemorySearchFilters,
) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	query := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Where("is_deleted = ?", false).
		Where("(1 - (embedding <=> ?)) >= ?", embedding, filters.MinSimilarity)

	// 应用记忆类型过滤
	if len(filters.MemoryTypes) > 0 {
		query = query.Where("memory_type IN ?", filters.MemoryTypes)
	}

	// 应用时间范围过滤
	if filters.TimeRangeDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -filters.TimeRangeDays)
		query = query.Where("created_at >= ?", cutoffTime)
	}

	// 应用重要性过滤
	if filters.MinImportance != nil {
		query = query.Where("importance >= ?", *filters.MinImportance)
	}

	err := query.
		Order(gorm.Expr("embedding <=> ?", embedding)).
		Limit(filters.TopK).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("带过滤条件的向量检索失败: %w", err)
	}

	return memories, nil
}

// UpdateAccessStats 更新访问统计
func (r *genkitMemoryRepository) UpdateAccessStats(ctx context.Context, id string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"access_count":   gorm.Expr("access_count + 1"),
			"last_access_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("更新访问统计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("记忆不存在或已删除")
	}

	return nil
}

// BatchUpdateAccessStats 批量更新访问统计
func (r *genkitMemoryRepository) BatchUpdateAccessStats(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Updates(map[string]interface{}{
			"access_count":   gorm.Expr("access_count + 1"),
			"last_access_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("批量更新访问统计失败: %w", result.Error)
	}

	return nil
}

// DeleteByStrategy 按策略删除记忆
func (r *genkitMemoryRepository) DeleteByStrategy(
	ctx context.Context,
	tenantID string,
	strategy string,
	mode string,
	batchSize int,
) (int, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("is_deleted = ?", false)

	// 根据策略添加过滤条件
	switch strategy {
	case "expired":
		query = query.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now())
	case "low_quality":
		query = query.Where("importance < ? AND access_count < ?", 0.3, 2)
	case "unused":
		cutoff := time.Now().AddDate(0, 0, -90)
		query = query.Where("last_access_at < ? OR (last_access_at IS NULL AND created_at < ?)", cutoff, cutoff)
	case "all":
		// 不添加额外过滤条件
	default:
		return 0, fmt.Errorf("不支持的清理策略: %s", strategy)
	}

	query = query.Limit(batchSize)

	// 根据模式执行删除
	var result *gorm.DB
	if mode == "soft" {
		// 软删除
		result = query.Update("is_deleted", true)
	} else if mode == "hard" {
		// 硬删除
		result = query.Delete(&model.ConversationMemory{})
	} else {
		return 0, fmt.Errorf("不支持的删除模式: %s", mode)
	}

	if result.Error != nil {
		return 0, fmt.Errorf("删除记忆失败: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// GetExpiredMemories 获取过期记忆
func (r *genkitMemoryRepository) GetExpiredMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", filters.TenantID).
		Where("is_deleted = ?", false).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now())

	// 如果指定了会话ID，则只清理该会话的记忆
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}

	err := query.
		Limit(filters.BatchSize).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("获取过期记忆失败: %w", err)
	}

	return memories, nil
}

// GetLowQualityMemories 获取低质量记忆
func (r *genkitMemoryRepository) GetLowQualityMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", filters.TenantID).
		Where("is_deleted = ?", false).
		Where("importance < ? AND access_count < ?", 0.3, 2)

	// 如果指定了会话ID，则只清理该会话的记忆
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}

	err := query.
		Limit(filters.BatchSize).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("获取低质量记忆失败: %w", err)
	}

	return memories, nil
}

// GetUnusedMemories 获取长期未使用的记忆
func (r *genkitMemoryRepository) GetUnusedMemories(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	// 默认90天未访问
	cutoff := time.Now().AddDate(0, 0, -90)

	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", filters.TenantID).
		Where("is_deleted = ?", false).
		Where("last_access_at < ? OR (last_access_at IS NULL AND created_at < ?)", cutoff, cutoff)

	// 如果指定了会话ID，则只清理该会话的记忆
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}

	err := query.
		Limit(filters.BatchSize).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("获取未使用记忆失败: %w", err)
	}

	return memories, nil
}

// GetAllMemoriesForCleanup 获取所有待清理的记忆
func (r *genkitMemoryRepository) GetAllMemoriesForCleanup(ctx context.Context, filters *MemoryCleanupFilters) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", filters.TenantID).
		Where("is_deleted = ?", false)

	// 如果指定了会话ID，则只清理该会话的记忆
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}

	err := query.
		Limit(filters.BatchSize).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("获取所有记忆失败: %w", err)
	}

	return memories, nil
}

// SoftDeleteBatch 批量软删除记忆
func (r *genkitMemoryRepository) SoftDeleteBatch(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return 0, fmt.Errorf("批量软删除记忆失败: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// HardDeleteBatch 批量硬删除记忆
func (r *genkitMemoryRepository) HardDeleteBatch(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Delete(&model.ConversationMemory{})

	if result.Error != nil {
		return 0, fmt.Errorf("批量硬删除记忆失败: %w", result.Error)
	}

	return int(result.RowsAffected), nil
}

// CountBySession 统计会话的记忆数量
func (r *genkitMemoryRepository) CountBySession(ctx context.Context, sessionID string) (int, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("session_id = ? AND is_deleted = ?", sessionID, false).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("统计会话记忆数量失败: %w", err)
	}

	return int(count), nil
}

// CountByTenant 统计租户的记忆数量
func (r *genkitMemoryRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("tenant_id = ? AND is_deleted = ?", tenantID, false).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("统计租户记忆数量失败: %w", err)
	}

	return int(count), nil
}

// Update 更新记忆
func (r *genkitMemoryRepository) Update(ctx context.Context, memory *model.ConversationMemory) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", memory.ID, false).
		Save(memory)

	if result.Error != nil {
		return fmt.Errorf("更新记忆失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("记忆不存在或已删除")
	}

	return nil
}

// SoftDelete 软删除记忆
func (r *genkitMemoryRepository) SoftDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&model.ConversationMemory{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("软删除记忆失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("记忆不存在或已删除")
	}

	return nil
}

// HardDelete 硬删除记忆
func (r *genkitMemoryRepository) HardDelete(ctx context.Context, id string) error {
	// 解析UUID
	memoryID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("无效的记忆ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Where("id = ?", memoryID).
		Delete(&model.ConversationMemory{})

	if result.Error != nil {
		return fmt.Errorf("硬删除记忆失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("记忆不存在")
	}

	return nil
}

// SearchByContent 全文搜索记忆（降级方案）
// 当向量服务不可用时，使用简单的文本匹配作为降级方案
func (r *genkitMemoryRepository) SearchByContent(
	ctx context.Context,
	sessionID string,
	query string,
	topK int,
) ([]*model.ConversationMemory, error) {
	var memories []*model.ConversationMemory

	// 使用 ILIKE 进行不区分大小写的模糊匹配
	// 注意：这是一个简单的降级方案，性能不如向量检索
	// 实际生产环境应该使用 PostgreSQL 的全文搜索功能（tsvector, tsquery）
	searchPattern := "%" + query + "%"

	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Where("is_deleted = ?", false).
		Where("content ILIKE ?", searchPattern).
		Order("importance DESC, created_at DESC").
		Limit(topK).
		Find(&memories).Error

	if err != nil {
		return nil, fmt.Errorf("全文搜索失败: %w", err)
	}

	return memories, nil
}
