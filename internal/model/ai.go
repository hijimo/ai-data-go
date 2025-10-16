package model

// ChatResponse 对话响应
type ChatResponse struct {
	// 会话ID
	SessionID string `json:"sessionId"`
	// AI生成的消息内容
	Message string `json:"message"`
	// 使用的模型名称
	Model string `json:"model"`
	// Token使用情况
	Usage *Usage `json:"usage,omitempty"`
}

// Usage Token使用情况
type Usage struct {
	// 提示词token数
	PromptTokens int `json:"promptTokens"`
	// 生成内容token数
	CompletionTokens int `json:"completionTokens"`
	// 总token数
	TotalTokens int `json:"totalTokens"`
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
