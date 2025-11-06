package service

import (
	"context"

	"genkit-ai-service/internal/model"
)

// ContextService 上下文服务接口
// 负责构建和优化会话上下文，实现三层记忆架构
type ContextService interface {
	// BuildContext 构建会话上下文
	// 根据会话ID和用户查询，智能构建包含短期记忆、长期记忆和摘要的上下文
	// 参数：
	//   - ctx: 上下文
	//   - req: 构建上下文请求
	// 返回：
	//   - *ContextResult: 构建的上下文结果
	//   - error: 错误信息
	BuildContext(ctx context.Context, req BuildContextRequest) (*ContextResult, error)

	// OptimizeContext 优化上下文Token使用
	// 当上下文Token数量超过限制时，智能裁剪以满足Token预算
	// 参数：
	//   - ctx: 上下文
	//   - req: 优化上下文请求
	// 返回：
	//   - *ContextResult: 优化后的上下文结果
	//   - error: 错误信息
	OptimizeContext(ctx context.Context, req OptimizeContextRequest) (*ContextResult, error)

	// GetContextConfig 获取上下文配置
	// 获取指定会话的上下文配置信息
	// 参数：
	//   - ctx: 上下文
	//   - sessionID: 会话ID
	// 返回：
	//   - *model.ConversationContext: 上下文配置
	//   - error: 错误信息
	GetContextConfig(ctx context.Context, sessionID string) (*model.ConversationContext, error)

	// UpdateContextConfig 更新上下文配置
	// 更新指定会话的上下文配置
	// 参数：
	//   - ctx: 上下文
	//   - sessionID: 会话ID
	//   - config: 新的上下文配置
	// 返回：
	//   - error: 错误信息
	UpdateContextConfig(ctx context.Context, sessionID string, config *model.ConversationContext) error
}

// BuildContextRequest 构建上下文请求
type BuildContextRequest struct {
	// 会话ID
	SessionID string
	// 用户查询（用于向量检索）
	UserQuery string
	// 最大Token数量
	MaxTokens int
	// 上下文策略：auto（自动）、short（仅短期）、full（完整）
	Strategy string
	// 是否包含摘要
	IncludeSummary bool
	// 是否包含长期记忆
	IncludeLongTerm bool
	// 短期记忆窗口大小（最近N条消息）
	ShortTermWindow int
}

// ContextResult 上下文构建结果
type ContextResult struct {
	// 会话ID
	SessionID string
	// 摘要记忆
	Summary *model.ConversationSummary
	// 长期记忆列表
	LongTermMemories []*model.ConversationMemory
	// 短期消息列表
	ShortTermMessages []*model.ChatMessage
	// 总Token数量
	TotalTokens int
	// 使用的策略
	Strategy string
	// 上下文质量评分（0-1）
	QualityScore float64
}

// OptimizeContextRequest 优化上下文请求
type OptimizeContextRequest struct {
	// 原始上下文
	Context *ContextResult
	// 目标Token数量
	TargetTokens int
	// 优化策略：aggressive（激进）、balanced（平衡）、conservative（保守）
	Strategy string
	// 是否保留摘要
	PreserveSummary bool
}
