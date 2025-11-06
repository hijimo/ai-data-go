package service

import (
	"context"

	"genkit-ai-service/internal/model"
)

// TokenManager Token管理器接口
// 负责计算和估算AI模型的Token使用量
type TokenManager interface {
	// CalculateTokens 计算文本的Token数量
	// 参数：
	//   - ctx: 上下文
	//   - text: 要计算的文本
	//   - modelName: 模型名称（可选，用于选择合适的编码器）
	// 返回：
	//   - int: Token数量
	//   - error: 错误信息
	CalculateTokens(ctx context.Context, text string, modelName string) (int, error)

	// CalculateContextTokens 计算上下文的总Token数量
	// 包括短期消息、长期记忆和摘要的Token总和
	// 参数：
	//   - ctx: 上下文
	//   - messages: 短期消息列表
	//   - memories: 长期记忆列表（可选）
	//   - summary: 摘要内容（可选）
	// 返回：
	//   - int: 总Token数量
	//   - error: 错误信息
	CalculateContextTokens(
		ctx context.Context,
		messages []*model.ChatMessage,
		memories []*model.ConversationMemory,
		summary *model.ConversationSummary,
	) (int, error)

	// EstimateTokens 快速估算文本的Token数量
	// 使用简单的启发式方法，速度快但精度较低
	// 参数：
	//   - text: 要估算的文本
	// 返回：
	//   - int: 估算的Token数量
	EstimateTokens(text string) int

	// CalculateMessagesTokens 计算消息列表的Token数量
	// 参数：
	//   - ctx: 上下文
	//   - messages: 消息列表
	//   - modelName: 模型名称
	// 返回：
	//   - int: Token数量
	//   - error: 错误信息
	CalculateMessagesTokens(ctx context.Context, messages []*model.ChatMessage, modelName string) (int, error)
}
