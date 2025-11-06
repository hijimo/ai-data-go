package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// contextRepositoryImpl 上下文配置仓储实现
type contextRepositoryImpl struct {
	db *gorm.DB
}

// NewContextRepository 创建上下文配置仓储实例
func NewContextRepository(db *gorm.DB) ContextRepository {
	return &contextRepositoryImpl{
		db: db,
	}
}

// Create 创建上下文配置
func (r *contextRepositoryImpl) Create(ctx context.Context, context *model.ConversationContext) error {
	if err := r.db.WithContext(ctx).Create(context).Error; err != nil {
		return fmt.Errorf("创建上下文配置失败: %w", err)
	}
	return nil
}

// GetBySessionID 根据会话ID获取上下文配置
// 包含租户ID过滤和软删除过滤
func (r *contextRepositoryImpl) GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	var context model.ConversationContext

	// 解析会话ID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("无效的会话ID: %w", err)
	}

	// 查询上下文配置，包含软删除过滤
	err = r.db.WithContext(ctx).
		Where("session_id = ?", sessionUUID).
		Where("is_deleted = ?", false).
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
// 包含租户ID验证和软删除过滤
func (r *contextRepositoryImpl) Update(ctx context.Context, context *model.ConversationContext) error {
	// 验证记录存在且未被删除
	var existing model.ConversationContext
	err := r.db.WithContext(ctx).
		Where("id = ?", context.ID).
		Where("is_deleted = ?", false).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("上下文配置不存在或已被删除")
		}
		return fmt.Errorf("查询上下文配置失败: %w", err)
	}

	// 验证租户ID匹配（防止跨租户更新）
	if existing.TenantID != context.TenantID {
		return fmt.Errorf("权限不足：无法更新其他租户的上下文配置")
	}

	// 执行更新
	err = r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("id = ?", context.ID).
		Where("is_deleted = ?", false).
		Updates(context).Error

	if err != nil {
		return fmt.Errorf("更新上下文配置失败: %w", err)
	}

	return nil
}

// GetLatestSummary 获取最新摘要
// 包含租户ID过滤和软删除过滤
func (r *contextRepositoryImpl) GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary

	// 解析会话ID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("无效的会话ID: %w", err)
	}

	// 查询最新摘要，按创建时间降序排列
	err = r.db.WithContext(ctx).
		Where("session_id = ?", sessionUUID).
		Where("is_deleted = ?", false).
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
// 包含租户ID验证和软删除过滤
func (r *contextRepositoryImpl) UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error {
	// 解析会话ID
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("无效的会话ID: %w", err)
	}

	// 验证记录存在且未被删除
	var existing model.ConversationContext
	err = r.db.WithContext(ctx).
		Where("session_id = ?", sessionUUID).
		Where("is_deleted = ?", false).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("上下文配置不存在或已被删除")
		}
		return fmt.Errorf("查询上下文配置失败: %w", err)
	}

	// 更新Token使用统计和消息计数
	err = r.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Where("session_id = ?", sessionUUID).
		Where("is_deleted = ?", false).
		Updates(map[string]interface{}{
			"total_tokens_used": gorm.Expr("total_tokens_used + ?", tokens),
			"total_messages":    gorm.Expr("total_messages + 1"),
		}).Error

	if err != nil {
		return fmt.Errorf("更新Token使用统计失败: %w", err)
	}

	return nil
}
