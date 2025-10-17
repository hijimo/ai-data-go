package model

// ChatResponse 对话响应
type ChatResponse struct {
	// 会话ID
	SessionID string `json:"sessionId" example:"session-123456"`
	// AI生成的消息内容
	Message string `json:"message" example:"你好！我是一个 AI 助手..."`
	// 使用的模型名称
	Model string `json:"model" example:"gemini-1.5-flash"`
	// Token使用情况
	Usage *Usage `json:"usage,omitempty"`
}

// Usage Token使用情况
type Usage struct {
	// 提示词token数
	PromptTokens int `json:"promptTokens" example:"10"`
	// 生成内容token数
	CompletionTokens int `json:"completionTokens" example:"50"`
	// 总token数
	TotalTokens int `json:"totalTokens" example:"60"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	// 内容片段
	Content string `json:"content"`
	// 是否完成
	Done bool `json:"done"`
	// 错误信息
	Error error `json:"error,omitempty"`
}
