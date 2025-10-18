package model

// ChatRequest 对话请求
type ChatRequest struct {
	// 用户消息内容
	Message string `json:"message" validate:"required" example:"你好，请介绍一下你自己"`
	// 消息ID（可选，用于继续对话）
	MessageID string `json:"messageId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
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
	// 消息ID（必填）
	MessageID string `json:"messageId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	// 会话标题
	Title string `json:"title" validate:"required,max=255" example:"我的第一个会话"`
	// 模型名称
	ModelName string `json:"modelName" validate:"required,max=128" example:"gpt-4"`
	// 系统提示词（可选）
	SystemPrompt string `json:"systemPrompt,omitempty" example:"你是一个有帮助的AI助手"`
	// 温度参数（可选，0-2）
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.7"`
	// TopP参数（可选，0-1）
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.9"`
	// 元数据（可选）
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// ListSessionsRequest 获取会话列表请求
type ListSessionsRequest struct {
	// 页码
	PageNo int `json:"pageNo" validate:"required,min=1" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" validate:"required,min=1,max=100" example:"20"`
	// 是否置顶（可选）
	IsPinned *bool `json:"isPinned,omitempty" example:"true"`
	// 是否归档（可选）
	IsArchived *bool `json:"isArchived,omitempty" example:"false"`
	// 模型名称（可选）
	ModelName string `json:"modelName,omitempty" example:"gpt-4"`
}

// UpdateSessionRequest 更新会话请求
type UpdateSessionRequest struct {
	// 会话标题（可选）
	Title *string `json:"title,omitempty" validate:"omitempty,max=255" example:"更新后的标题"`
	// 系统提示词（可选）
	SystemPrompt *string `json:"systemPrompt,omitempty" example:"你是一个专业的编程助手"`
	// 温度参数（可选，0-2）
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.8"`
	// TopP参数（可选，0-1）
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.95"`
	// 模型名称（可选）
	ModelName *string `json:"modelName,omitempty" validate:"omitempty,max=128" example:"gpt-4-turbo"`
}

// SearchSessionsRequest 搜索会话请求
type SearchSessionsRequest struct {
	// 搜索关键词
	Keyword string `json:"keyword" validate:"required" example:"AI"`
	// 页码
	PageNo int `json:"pageNo" validate:"required,min=1" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" validate:"required,min=1,max=100" example:"20"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 消息内容
	Message string `json:"message" validate:"required" example:"你好，请介绍一下你自己"`
	// AI高级参数（可选）
	Options *ChatOptions `json:"options,omitempty"`
}

// GetMessagesRequest 获取消息历史请求
type GetMessagesRequest struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 页码
	PageNo int `json:"pageNo" validate:"required,min=1" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" validate:"required,min=1,max=100" example:"50"`
}

// AbortMessageRequest 中止消息生成请求
type AbortMessageRequest struct {
	// 消息ID
	MessageID string `json:"messageId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}
