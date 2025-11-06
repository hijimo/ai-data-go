// internal/service/memory_service.go
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"genkit-ai-service/internal/model"
)

// MemoryService 记忆管理服务接口
// 负责长期记忆的存储、检索和清理
type MemoryService interface {
	// SearchMemories 检索记忆
	// 基于向量相似度检索相关的历史对话记忆
	// 参数:
	//   - ctx: 上下文（包含租户信息）
	//   - req: 检索请求
	// 返回:
	//   - []*MemorySearchResult: 检索结果列表
	//   - error: 错误信息
	SearchMemories(ctx context.Context, req *SearchMemoriesRequest) ([]*MemorySearchResult, error)

	// StoreMemory 存储记忆
	// 将对话消息转换为长期记忆并存储
	// 参数:
	//   - ctx: 上下文（包含租户信息）
	//   - req: 存储请求
	// 返回:
	//   - *model.ConversationMemory: 创建的记忆记录
	//   - error: 错误信息
	StoreMemory(ctx context.Context, req *StoreMemoryRequest) (*model.ConversationMemory, error)

	// CleanupMemories 清理记忆
	// 根据策略清理过期或低质量的记忆
	// 参数:
	//   - ctx: 上下文（包含租户信息）
	//   - req: 清理请求
	// 返回:
	//   - *CleanupResult: 清理结果
	//   - error: 错误信息
	CleanupMemories(ctx context.Context, req *CleanupMemoriesRequest) (*CleanupResult, error)

	// UpdateMemoryAccess 更新记忆访问统计
	// 记录记忆被访问的次数和时间
	// 参数:
	//   - ctx: 上下文（包含租户信息）
	//   - tenantID: 租户ID
	//   - memoryID: 记忆ID
	// 返回:
	//   - error: 错误信息
	UpdateMemoryAccess(ctx context.Context, tenantID, memoryID uuid.UUID) error
}

// SearchMemoriesRequest 检索记忆请求
type SearchMemoriesRequest struct {
	SessionID            uuid.UUID // 会话ID（必须）
	Query                string    // 查询文本（必须）
	TopK                 int       // 返回结果数量（默认：5）
	MinSimilarity        float32   // 最小相似度（0-1，默认：0.7）
	TimeRangeDays        int       // 时间范围（天数，0表示不限制）
	MemoryTypes          []string  // 记忆类型过滤（可选）
	IncludeCrossSessions bool      // 是否包含跨会话检索（默认：false）
}

// MemorySearchResult 记忆检索结果
type MemorySearchResult struct {
	Memory     *model.ConversationMemory // 记忆记录
	Similarity float32                   // 相似度分数（0-1）
	Score      float32                   // 综合评分（相似度 × 重要性）
}

// StoreMemoryRequest 存储记忆请求
type StoreMemoryRequest struct {
	SessionID      uuid.UUID              // 会话ID（必须）
	MessageIDs     []uuid.UUID            // 消息ID列表（可选，用于关联消息）
	MemoryType     string                 // 记忆类型（必须）
	Content        string                 // 记忆内容（必须）
	Importance     float32                // 重要性评分（0-1，默认：0.5）
	ExpirationDays int                    // 过期天数（0表示不过期）
	Metadata       map[string]interface{} // 元数据（可选）
}

// CleanupMemoriesRequest 清理记忆请求
type CleanupMemoriesRequest struct {
	SessionID uuid.UUID // 会话ID（可选，为空则清理租户所有记忆）
	Strategy  string    // 清理策略：expired, low_quality, unused, all
	Mode      string    // 清理模式：soft（软删除）, hard（硬删除）
	BatchSize int       // 批量处理大小（默认：100）
	Execute   bool      // 是否执行删除（false时仅预览）
}

// CleanupResult 清理结果
type CleanupResult struct {
	CleanedCount int             // 清理数量
	FreedSpace   int64           // 释放空间（字节）
	Details      []CleanupDetail // 清理详情
	Preview      bool            // 是否为预览模式
}

// CleanupDetail 清理详情
type CleanupDetail struct {
	MemoryID   uuid.UUID // 记忆ID
	Reason     string    // 清理原因
	Size       int64     // 记忆大小（字节）
	CreatedAt  time.Time // 创建时间
	LastAccess time.Time // 最后访问时间
}
