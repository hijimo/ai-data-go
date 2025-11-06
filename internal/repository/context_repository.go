package repository

import (
	"context"

	"genkit-ai-service/internal/model"
)

// ContextRepository 上下文配置仓储接口
type ContextRepository interface {
	// Create 创建上下文配置
	Create(ctx context.Context, context *model.ConversationContext) error

	// GetBySessionID 根据会话ID获取上下文配置
	GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error)

	// Update 更新上下文配置
	Update(ctx context.Context, context *model.ConversationContext) error

	// GetLatestSummary 获取最新摘要
	GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error)

	// UpdateTokenUsage 更新Token使用统计
	UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error
}
