package model

// ChatRequest 对话请求
type ChatRequest struct {
	// 用户消息内容
	Message string `json:"message" validate:"required" example:"你好，请介绍一下你自己"`
	// 会话ID（可选）
	SessionID string `json:"sessionId,omitempty" example:"session-123456"`
	// AI高级参数（可选）
	Options *ChatOptions `json:"options,omitempty"`
}

// ChatOptions AI高级参数
type ChatOptions struct {
	// 温度值，控制输出的随机性（0-2）
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.7"`
	// 最大token数
	MaxTokens *int `json:"maxTokens,omitempty" validate:"omitempty,gt=0" example:"2048"`
	// Top-P采样参数（0-1）
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.9"`
	// Top-K采样参数
	TopK *int `json:"topK,omitempty" validate:"omitempty,gt=0" example:"40"`
}

// AbortRequest 中止对话请求
type AbortRequest struct {
	// 会话ID（必填）
	SessionID string `json:"sessionId" validate:"required" example:"session-123456"`
}
