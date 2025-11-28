package bailian

// BailianRequest 百炼 API 请求结构
// 由于百炼完全兼容 OpenAI API，这里的结构与 OpenAI 保持一致
type BailianRequest struct {
	// Model 模型名称（如 qwen-plus, qwen-max, qwen-turbo）
	Model string `json:"model"`
	
	// Messages 消息列表
	Messages []Message `json:"messages"`
	
	// Temperature 温度参数，控制随机性 (0-2)
	Temperature *float64 `json:"temperature,omitempty"`
	
	// MaxTokens 最大生成 token 数
	MaxTokens *int `json:"max_tokens,omitempty"`
	
	// TopP 核采样参数
	TopP *float64 `json:"top_p,omitempty"`
	
	// Stream 是否流式输出
	Stream bool `json:"stream,omitempty"`
	
	// StreamOptions 流式输出选项
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

// Message 消息结构
type Message struct {
	// Role 角色（system, user, assistant）
	Role string `json:"role"`
	
	// Content 消息内容
	Content string `json:"content"`
}

// StreamOptions 流式输出选项
type StreamOptions struct {
	// IncludeUsage 是否在流式输出中包含 token 使用统计
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// BailianResponse 百炼 API 响应结构
type BailianResponse struct {
	// ID 请求ID
	ID string `json:"id"`
	
	// Object 对象类型
	Object string `json:"object"`
	
	// Created 创建时间戳
	Created int64 `json:"created"`
	
	// Model 使用的模型
	Model string `json:"model"`
	
	// Choices 生成的选项列表
	Choices []Choice `json:"choices"`
	
	// Usage token 使用统计
	Usage *Usage `json:"usage,omitempty"`
}

// Choice 生成选项
type Choice struct {
	// Index 选项索引
	Index int `json:"index"`
	
	// Message 生成的消息
	Message *Message `json:"message,omitempty"`
	
	// Delta 流式输出的增量消息
	Delta *Message `json:"delta,omitempty"`
	
	// FinishReason 结束原因
	FinishReason *string `json:"finish_reason,omitempty"`
}

// Usage token 使用统计
type Usage struct {
	// PromptTokens 输入 token 数
	PromptTokens int `json:"prompt_tokens"`
	
	// CompletionTokens 输出 token 数
	CompletionTokens int `json:"completion_tokens"`
	
	// TotalTokens 总 token 数
	TotalTokens int `json:"total_tokens"`
}

// BailianStreamChunk 百炼流式响应块
type BailianStreamChunk struct {
	// ID 请求ID
	ID string `json:"id"`
	
	// Object 对象类型
	Object string `json:"object"`
	
	// Created 创建时间戳
	Created int64 `json:"created"`
	
	// Model 使用的模型
	Model string `json:"model"`
	
	// Choices 生成的选项列表
	Choices []Choice `json:"choices"`
	
	// Usage token 使用统计（仅在最后一个 chunk 中包含）
	Usage *Usage `json:"usage,omitempty"`
}

// BailianError 百炼 API 错误响应
type BailianError struct {
	// Error 错误详情
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	// Message 错误消息
	Message string `json:"message"`
	
	// Type 错误类型
	Type string `json:"type"`
	
	// Code 错误代码
	Code string `json:"code,omitempty"`
}
