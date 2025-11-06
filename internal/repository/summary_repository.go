package repository

import (
	"context"

	"github.com/google/uuid"

	"genkit-ai-service/internal/model"
)

// SummaryRepository 摘要数据访问接口
type SummaryRepository interface {
	// Create 创建摘要
	Create(ctx context.Context, summary *model.ConversationSummary) error

	// GetByID 根据ID获取摘要
	GetByID(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error)

	// GetLatestBySessionID 获取会话最新摘要
	GetLatestBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (*model.ConversationSummary, error)

	// ListBySessionID 获取会话摘要列表
	ListBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error)

	// Update 更新摘要
	Update(ctx context.Context, summary *model.ConversationSummary) error

	// SoftDelete 软删除摘要
	SoftDelete(ctx context.Context, tenantID, summaryID uuid.UUID) error

	// HardDelete 硬删除摘要
	HardDelete(ctx context.Context, tenantID, summaryID uuid.UUID) error

	// GetByType 根据类型获取摘要列表
	GetByType(ctx context.Context, tenantID, sessionID uuid.UUID, summaryType string, limit int) ([]*model.ConversationSummary, error)

	// CountBySessionID 统计会话摘要数量
	CountBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (int64, error)
}
