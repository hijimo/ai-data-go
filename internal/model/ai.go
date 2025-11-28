package model

// ChatResponse 对话响应
// @Description AI对话的响应数据
// @name ChatResponse
type ChatResponse struct {
	// 会话ID
	// @Description 会话的唯一标识符
	// @Example session-123456
	SessionID string `json:"sessionId" example:"session-123456"`
	// AI生成的消息内容
	// @Description AI生成的回复文本
	// @Example 你好！我是一个 AI 助手...
	Message string `json:"message" example:"你好！我是一个 AI 助手..."`
	// 使用的模型名称
	// @Description 实际使用的AI模型名称（可能是会话默认模型或请求中指定的模型）
	// @Example gemini-1.5-flash
	Model string `json:"model" example:"gemini-1.5-flash"`
	// Token使用情况
	// @Description 本次对话的token使用统计
	Usage *Usage `json:"usage,omitempty"`
}

// Usage Token使用情况
// @Description AI模型的token使用统计信息
// @name Usage
type Usage struct {
	// 提示词token数
	// @Description 输入提示词（包括历史消息）消耗的token数量
	// @Example 10
	PromptTokens int `json:"promptTokens" example:"10"`
	// 生成内容token数
	// @Description AI生成的回复内容消耗的token数量
	// @Example 50
	CompletionTokens int `json:"completionTokens" example:"50"`
	// 总token数
	// @Description 本次对话总共消耗的token数量（promptTokens + completionTokens）
	// @Example 60
	TotalTokens int `json:"totalTokens" example:"60"`
}

// StreamChunk 流式响应块（兼容旧格式，建议使用 TencentCloudStreamMessage）
// 已废弃：请使用 TencentCloudStreamMessage 以获得完整的流式输出支持
type StreamChunk struct {
	// 会话ID
	SessionID string `json:"sessionId,omitempty"`
	// 内容片段
	Content string `json:"content"`
	// 是否完成
	Done bool `json:"done"`
	// 使用的模型名称（仅在完成时提供）
	Model string `json:"model,omitempty"`
	// Token使用情况（仅在完成时提供）
	Usage *Usage `json:"usage,omitempty"`
	// 错误信息
	Error error `json:"error,omitempty"`
}
