package ai

import (
	"context"

	"genkit-ai-service/internal/model"
)

// ConversationContextService 会话上下文服务接口
// 用于构建包含历史消息的对话上下文，实现"记忆"功能
// 这个接口定义在 ai 包中以避免循环依赖
type ConversationContextService interface {
	// BuildConversationHistory 构建对话历史
	// 根据会话ID获取历史消息，用于多轮对话
	// 参数:
	//   ctx: 上下文
	//   sessionID: 会话ID
	//   maxMessages: 最大消息数量（0 表示使用默认值）
	// 返回:
	//   []*model.ChatHistoryMessage: 历史消息列表
	//   error: 错误信息
	BuildConversationHistory(ctx context.Context, sessionID string, maxMessages int) ([]*model.ChatHistoryMessage, error)
}

// AIService AI 服务接口
// 定义了 AI 对话服务的核心功能
type AIService interface {
	// Chat 发起对话
	// 处理用户的对话请求，调用 AI 模型生成响应
	// 参数:
	//   ctx: 上下文，用于控制请求生命周期
	//   req: 对话请求，包含用户消息和可选参数
	// 返回:
	//   *model.ChatResponse: 对话响应，包含 AI 生成的消息
	//   error: 错误信息
	Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error)

	// ChatStream 流式对话
	// 支持流式返回 AI 生成的内容，使用腾讯云流格式
	// 参数:
	//   ctx: 上下文，用于控制请求生命周期
	//   req: 对话请求，包含用户消息和可选参数
	// 返回:
	//   <-chan *model.TencentCloudStreamMessage: 流式响应通道（腾讯云格式）
	//   error: 错误信息
	ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan *model.TencentCloudStreamMessage, error)

	// AbortChat 中止对话
	// 取消正在进行的对话请求
	// 参数:
	//   ctx: 上下文
	//   messageID: 要中止的消息ID
	// 返回:
	//   error: 错误信息，如果消息不存在或已完成则返回错误
	AbortChat(ctx context.Context, messageID string) error
}
