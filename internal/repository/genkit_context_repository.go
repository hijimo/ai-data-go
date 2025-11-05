package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
	"github.com/google/uuid"
)

// GenkitContextRepository Genkit会话上下文数据访问接口
type GenkitContextRepository interface {
	// Create 创建上下文配置
	Create(ctx context.Context, context *model.ConversationContext) error

	// GetBySessionID 根据会话ID获取上下文配置
	GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error)

	// Update 更新上下文配置
	Update(ctx context.Context, context *model.ConversationContext) error

	// UpdateFields 更新指定字段
	UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error

	// GetLatestSummary 获取最新摘要
	GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error)

	// UpdateTokenUsage 更新Token使用统计
	UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error

	// IncrementMessageCount 增加消息计数
	IncrementMessageCount(ctx context.Context, sessionID string) error

	// UpdateLastSummary 更新最后摘要信息
	UpdateLastSummary(ctx context.Context, sessionID string, summaryID string) error

	// GetByTenantID 获取租户的所有上下文配置
	GetByTenantID(ctx context.Context, tenantID string, page, pageSize int) ([]*model.ConversationContext, int64, error)

	// Delete 删除上下文配置
	Delete(ctx context.Context, sessionID string) error

	// GetSummaryByID 根据ID获取摘要
	GetSummaryByID(ctx context.Context, summaryID string) (*model.ConversationSummary, error)

	// CreateSummary 创建摘要
	CreateSummary(ctx context.Context, summary *model.ConversationSummary) error

	// GetSummariesBySession 获取会话的所有摘要
	GetSummariesBySession(ctx context.Context, sessionID string, limit int) ([]*model.ConversationSummary, error)

	// CountSummariesBySession 统计会话的摘要数量
	CountSummariesBySession(ctx context.Context, sessionID string) (int, error)
}

// genkitContextRepository Genkit会话上下文数据访问实现
type genkitContextRepository struct {
	db *gorm.DB
}

// NewGenkitContextRepository 创建Genkit会话上下文数据访问实例
func NewGenkitContextRepository(db *gorm.DB) GenkitContextRepository {
	return &genkitContextRepository{
		db: db,
	}
}

// Create 创建上下文配置
func (r *genkitContextRepository) Create(ctx context.Context, context *model.ConversationContext) error {
	if err := r.db.WithContext(ctx).Create(context).Error; err != nil {
		return fmt.Errorf("创建上下文配置失败: %w", err)
	}
	return nil
}

// GetBySessionID 根据会话ID获取上下文配置
func (r *genkitContextRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	var context model.ConversationContext

	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("无效的会话ID: %w", err)
	}

	err = r.db.WithContext(ctx).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		First(&context).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("上下文配置不存在")
		}
		return nil, fmt.Errorf("查询上下文配置失败: %w", err)
	}

	return &context, nil
}

// Update 更新上下文配置
func (r *genkitContextRepository) Update(ctx context.Context, context *model.ConversationContext) error {
	result := r.db.WithContext(ctx).
		Where("session_id = ? AND is_deleted = ?", context.SessionID, false).
		Save(context)

	if result.Error != nil {
		return fmt.Errorf("更新上下文配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// UpdateFields 更新指定字段
func (r *genkitContextRepository) UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error {
	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("更新上下文配置字段失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// GetLatestSummary 获取最新摘要
func (r *genkitContextRepository) GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary

	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("无效的会话ID: %w", err)
	}

	err = r.db.WithContext(ctx).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Order("created_at DESC").
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("摘要不存在")
		}
		return nil, fmt.Errorf("查询最新摘要失败: %w", err)
	}

	return &summary, nil
}

// UpdateTokenUsage 更新Token使用统计
func (r *genkitContextRepository) UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error {
	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		UpdateColumn("total_tokens_used", gorm.Expr("total_tokens_used + ?", tokens))

	if result.Error != nil {
		return fmt.Errorf("更新Token使用统计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// IncrementMessageCount 增加消息计数
func (r *genkitContextRepository) IncrementMessageCount(ctx context.Context, sessionID string) error {
	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		UpdateColumn("total_messages", gorm.Expr("total_messages + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("增加消息计数失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// UpdateLastSummary 更新最后摘要信息
func (r *genkitContextRepository) UpdateLastSummary(ctx context.Context, sessionID string, summaryID string) error {
	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	summaryUUID, err := uuid.Parse(summaryID)
	if err != nil {
		return fmt.Errorf("无效的摘要ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Updates(map[string]interface{}{
			"last_summary_id": summaryUUID,
			"last_summary_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return fmt.Errorf("更新最后摘要信息失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// GetByTenantID 获取租户的所有上下文配置
func (r *genkitContextRepository) GetByTenantID(ctx context.Context, tenantID string, page, pageSize int) ([]*model.ConversationContext, int64, error) {
	var contexts []*model.ConversationContext
	var total int64

	// 解析UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("无效的租户ID: %w", err)
	}

	// 构建查询
	query := r.db.WithContext(ctx).Model(&model.ConversationContext{}).
		Where("tenant_id = ? AND is_deleted = ?", tenantUUID, false)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计上下文配置总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = query.
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&contexts).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询上下文配置列表失败: %w", err)
	}

	return contexts, total, nil
}

// Delete 删除上下文配置
func (r *genkitContextRepository) Delete(ctx context.Context, sessionID string) error {
	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("删除上下文配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("上下文配置不存在或已删除")
	}

	return nil
}

// GetSummaryByID 根据ID获取摘要
func (r *genkitContextRepository) GetSummaryByID(ctx context.Context, summaryID string) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary

	// 解析UUID
	summaryUUID, err := uuid.Parse(summaryID)
	if err != nil {
		return nil, fmt.Errorf("无效的摘要ID: %w", err)
	}

	err = r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", summaryUUID, false).
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("摘要不存在")
		}
		return nil, fmt.Errorf("查询摘要失败: %w", err)
	}

	return &summary, nil
}

// CreateSummary 创建摘要
func (r *genkitContextRepository) CreateSummary(ctx context.Context, summary *model.ConversationSummary) error {
	if err := r.db.WithContext(ctx).Create(summary).Error; err != nil {
		return fmt.Errorf("创建摘要失败: %w", err)
	}
	return nil
}

// GetSummariesBySession 获取会话的所有摘要
func (r *genkitContextRepository) GetSummariesBySession(ctx context.Context, sessionID string, limit int) ([]*model.ConversationSummary, error) {
	var summaries []*model.ConversationSummary

	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("无效的会话ID: %w", err)
	}

	query := r.db.WithContext(ctx).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err = query.Find(&summaries).Error
	if err != nil {
		return nil, fmt.Errorf("查询会话摘要列表失败: %w", err)
	}

	return summaries, nil
}

// CountSummariesBySession 统计会话的摘要数量
func (r *genkitContextRepository) CountSummariesBySession(ctx context.Context, sessionID string) (int, error) {
	var count int64

	// 解析UUID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, fmt.Errorf("无效的会话ID: %w", err)
	}

	err = r.db.WithContext(ctx).
		Model(&model.ConversationSummary{}).
		Where("session_id = ? AND is_deleted = ?", sessionUUID, false).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("统计会话摘要数量失败: %w", err)
	}

	return int(count), nil
}
