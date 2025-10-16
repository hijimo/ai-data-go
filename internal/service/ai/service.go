package ai

import (
	"context"

	"genkit-ai-service/internal/model"
)

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

	// ChatStream 流式对话（预留接口）
	// 支持流式返回 AI 生成的内容
	// 参数:
	//   ctx: 上下文，用于控制请求生命周期
	//   req: 对话请求，包含用户消息和可选参数
	// 返回:
	//   <-chan model.StreamChunk: 流式响应通道
	//   error: 错误信息
	ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error)

	// AbortChat 中止对话
	// 取消正在进行的对话请求
	// 参数:
	//   ctx: 上下文
	//   sessionID: 要中止的会话ID
	// 返回:
	//   error: 错误信息，如果会话不存在或已完成则返回错误
	AbortChat(ctx context.Context, sessionID string) error
}
