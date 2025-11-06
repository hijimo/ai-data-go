package repository

import (
	"context"

	"github.com/google/uuid"

	"genkit-ai-service/internal/model"
)

// MemoryRepository 记忆数据访问接口
type MemoryRepository interface {
	// Create 创建记忆记录
	Create(ctx context.Context, memory *model.ConversationMemory) error

	// GetByID 根据ID获取记忆
	GetByID(ctx context.Context, tenantID, memoryID uuid.UUID) (*model.ConversationMemory, error)

	// SearchByVector 向量相似度搜索（会话内）
	// 注意：此方法仅返回元数据，实际向量检索由 VectorService 完成
	SearchByVector(ctx context.Context, tenantID, sessionID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error)

	// SearchByVectorCrossSessions 跨会话向量搜索（租户内）
	// 注意：此方法仅返回元数据，实际向量检索由 VectorService 完成
	SearchByVectorCrossSessions(ctx context.Context, tenantID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error)

	// UpdateAccessStats 更新访问统计
	UpdateAccessStats(ctx context.Context, tenantID, memoryID uuid.UUID) error

	// DeleteByStrategy 按策略删除记忆
	DeleteByStrategy(ctx context.Context, tenantID uuid.UUID, strategy DeleteStrategy, mode DeleteMode) (int64, error)

	// GetExpiredMemories 获取过期记忆
	GetExpiredMemories(ctx context.Context, tenantID uuid.UUID, limit int) ([]*model.ConversationMemory, error)

	// BatchCreate 批量创建记忆记录
	BatchCreate(ctx context.Context, memories []*model.ConversationMemory) error

	// GetBySessionID 获取会话的所有记忆
	GetBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID, memoryType string, limit int) ([]*model.ConversationMemory, error)

	// SoftDelete 软删除记忆
	SoftDelete(ctx context.Context, tenantID, memoryID uuid.UUID) error

	// HardDelete 硬删除记忆
	HardDelete(ctx context.Context, tenantID, memoryID uuid.UUID) error
}

// DeleteStrategy 删除策略
type DeleteStrategy string

const (
	// DeleteStrategyExpired 删除过期记忆
	DeleteStrategyExpired DeleteStrategy = "expired"
	// DeleteStrategyLowQuality 删除低质量记忆（重要性低于0.3且访问次数少于2）
	DeleteStrategyLowQuality DeleteStrategy = "low_quality"
	// DeleteStrategyUnused 删除未使用记忆（90天未访问）
	DeleteStrategyUnused DeleteStrategy = "unused"
	// DeleteStrategyAll 删除所有记忆
	DeleteStrategyAll DeleteStrategy = "all"
)

// DeleteMode 删除模式
type DeleteMode string

const (
	// DeleteModeSoft 软删除
	DeleteModeSoft DeleteMode = "soft"
	// DeleteModeHard 硬删除
	DeleteModeHard DeleteMode = "hard"
)
