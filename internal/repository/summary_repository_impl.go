package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// summaryRepository 摘要数据访问实现
type summaryRepository struct {
	db *gorm.DB
}

// NewSummaryRepository 创建摘要数据访问实例
func NewSummaryRepository(db *gorm.DB) SummaryRepository {
	return &summaryRepository{
		db: db,
	}
}

// Create 创建摘要
func (r *summaryRepository) Create(ctx context.Context, summary *model.ConversationSummary) error {
	if err := r.db.WithContext(ctx).Create(summary).Error; err != nil {
		return fmt.Errorf("创建摘要失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取摘要
// 包含租户ID过滤和软删除过滤
func (r *summaryRepository) GetByID(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", summaryID, tenantID, false).
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("查询摘要失败: %w", err)
	}

	return &summary, nil
}

// GetLatestBySessionID 获取会话最新摘要
// 包含租户ID过滤和软删除过滤
func (r *summaryRepository) GetLatestBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND tenant_id = ? AND is_deleted = ?", sessionID, tenantID, false).
		Order("created_at DESC").
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("查询最新摘要失败: %w", err)
	}

	return &summary, nil
}

// ListBySessionID 获取会话摘要列表
// 包含租户ID过滤和软删除过滤
func (r *summaryRepository) ListBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error) {
	query := r.db.WithContext(ctx).
		Where("session_id = ? AND tenant_id = ? AND is_deleted = ?", sessionID, tenantID, false).
		Order("created_at DESC")

	// 如果指定了限制数量
	if limit > 0 {
		query = query.Limit(limit)
	}

	var summaries []*model.ConversationSummary
	err := query.Find(&summaries).Error

	if err != nil {
		return nil, fmt.Errorf("查询会话摘要列表失败: %w", err)
	}

	return summaries, nil
}

// Update 更新摘要
// 包含租户ID验证和软删除过滤
func (r *summaryRepository) Update(ctx context.Context, summary *model.ConversationSummary) error {
	// 验证记录存在且未被删除
	var existing model.ConversationSummary
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", summary.ID, false).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("查询摘要失败: %w", err)
	}

	// 验证租户ID匹配（防止跨租户更新）
	if existing.TenantID != summary.TenantID {
		return fmt.Errorf("权限不足：无法更新其他租户的摘要")
	}

	// 执行更新
	result := r.db.WithContext(ctx).
		Model(&model.ConversationSummary{}).
		Where("id = ? AND is_deleted = ?", summary.ID, false).
		Updates(summary)

	if result.Error != nil {
		return fmt.Errorf("更新摘要失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// SoftDelete 软删除摘要
// 包含租户ID验证
func (r *summaryRepository) SoftDelete(ctx context.Context, tenantID, summaryID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.ConversationSummary{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", summaryID, tenantID, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("软删除摘要失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// HardDelete 硬删除摘要
// 包含租户ID验证
func (r *summaryRepository) HardDelete(ctx context.Context, tenantID, summaryID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", summaryID, tenantID).
		Delete(&model.ConversationSummary{})

	if result.Error != nil {
		return fmt.Errorf("硬删除摘要失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetByType 根据类型获取摘要列表
// 包含租户ID过滤和软删除过滤
func (r *summaryRepository) GetByType(ctx context.Context, tenantID, sessionID uuid.UUID, summaryType string, limit int) ([]*model.ConversationSummary, error) {
	query := r.db.WithContext(ctx).
		Where("session_id = ? AND tenant_id = ? AND summary_type = ? AND is_deleted = ?", sessionID, tenantID, summaryType, false).
		Order("created_at DESC")

	// 如果指定了限制数量
	if limit > 0 {
		query = query.Limit(limit)
	}

	var summaries []*model.ConversationSummary
	err := query.Find(&summaries).Error

	if err != nil {
		return nil, fmt.Errorf("查询指定类型的摘要列表失败: %w", err)
	}

	return summaries, nil
}

// CountBySessionID 统计会话摘要数量
// 包含租户ID过滤和软删除过滤
func (r *summaryRepository) CountBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ConversationSummary{}).
		Where("session_id = ? AND tenant_id = ? AND is_deleted = ?", sessionID, tenantID, false).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("统计会话摘要数量失败: %w", err)
	}

	return count, nil
}
