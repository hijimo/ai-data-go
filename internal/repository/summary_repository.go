package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// SummaryRepository 摘要数据访问接口
type SummaryRepository interface {
	// Create 创建摘要
	Create(ctx context.Context, summary *model.ChatSummary) error

	// GetLatestBySessionID 获取会话的最新摘要
	GetLatestBySessionID(ctx context.Context, sessionID string) (*model.ChatSummary, error)

	// GetBySessionID 获取会话的所有摘要
	GetBySessionID(ctx context.Context, sessionID string) ([]*model.ChatSummary, error)
}

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
func (r *summaryRepository) Create(ctx context.Context, summary *model.ChatSummary) error {
	if err := r.db.WithContext(ctx).Create(summary).Error; err != nil {
		return fmt.Errorf("创建摘要失败: %w", err)
	}
	return nil
}

// GetLatestBySessionID 获取会话的最新摘要
func (r *summaryRepository) GetLatestBySessionID(ctx context.Context, sessionID string) (*model.ChatSummary, error) {
	var summary model.ChatSummary
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		First(&summary).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有摘要时返回 nil 而不是错误
		}
		return nil, fmt.Errorf("查询最新摘要失败: %w", err)
	}

	return &summary, nil
}

// GetBySessionID 获取会话的所有摘要
func (r *summaryRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*model.ChatSummary, error) {
	var summaries []*model.ChatSummary

	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&summaries).Error

	if err != nil {
		return nil, fmt.Errorf("查询会话摘要列表失败: %w", err)
	}

	return summaries, nil
}
