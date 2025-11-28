package model

// ChatRequest 对话请求
// @Description 发送对话消息的请求体
type ChatRequest struct {
	// 用户消息内容
	// @Description 用户发送的消息文本内容
	// @Example 你好，请介绍一下你自己
	Message string `json:"message" validate:"required" example:"你好，请介绍一下你自己"`
	// 消息ID（可选，用于继续对话）
	// @Description 可选的消息ID，用于继续之前的对话
	// @Example 550e8400-e29b-41d4-a716-446655440000
	MessageID string `json:"messageId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	// AI高级参数（可选）
	// @Description 可选的AI模型配置参数，包括模型名称、温度、最大token数等。如果指定了 modelName，系统会使用该模型；否则使用会话的默认模型。
	Options *ChatOptions `json:"options,omitempty"`
}

// ChatOptions AI高级参数
// @Description AI模型的高级配置参数，所有字段都是可选的
type ChatOptions struct {
	// 模型名称（可选，用于指定使用的模型）
	// @Description 指定要使用的AI模型名称，如 "gpt-4"、"gemini-pro"、"qwen-turbo" 等。系统会根据当前租户ID和模型名称从 model_configurations 表中查询配置。如果不指定，将使用会话的默认模型。
	// @Example gpt-4
	ModelName *string `json:"modelName,omitempty" validate:"omitempty,min=1,max=128" example:"gpt-4"`
	// 温度值，控制输出的随机性（0-2）
	// @Description 控制生成文本的随机性。值越高，输出越随机；值越低，输出越确定。范围：0.0-2.0
	// @Example 0.7
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.7"`
	// 最大token数
	// @Description 生成内容的最大token数量。实际生成的token数可能少于此值。
	// @Example 2048
	MaxTokens *int `json:"maxTokens,omitempty" validate:"omitempty,gt=0" example:"2048"`
	// Top-P采样参数（0-1）
	// @Description 核采样参数，控制生成文本的多样性。值越小，输出越集中；值越大，输出越多样。范围：0.0-1.0
	// @Example 0.9
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.9"`
	// Top-K采样参数
	// @Description 限制每步采样时考虑的token数量。值越小，输出越集中。
	// @Example 40
	TopK *int `json:"topK,omitempty" validate:"omitempty,gt=0" example:"40"`
}

// AbortRequest 中止对话请求
type AbortRequest struct {
	// 消息ID（必填）
	MessageID string `json:"messageId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// CreateSessionRequest 创建会话请求
// @Description 创建新对话会话的请求体
type CreateSessionRequest struct {
	// 会话标题
	// @Description 会话的显示标题
	// @Example 我的第一个会话
	Title string `json:"title" validate:"required,max=255" example:"我的第一个会话"`
	// 模型名称
	// @Description 会话使用的AI模型名称，如 "gpt-4"、"gemini-pro"、"qwen-turbo" 等。系统会根据当前租户ID和模型名称从 model_configurations 表中查询配置。
	// @Example gpt-4
	ModelName string `json:"modelName" validate:"required,max=128" example:"gpt-4"`
	// 系统提示词（可选）
	// @Description 可选的系统级提示词，用于设定AI的角色和行为
	// @Example 你是一个有帮助的AI助手
	SystemPrompt string `json:"systemPrompt,omitempty" example:"你是一个有帮助的AI助手"`
	// 温度参数（可选，0-2）
	// @Description 控制生成文本的随机性，范围：0.0-2.0
	// @Example 0.7
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.7"`
	// TopP参数（可选，0-1）
	// @Description 核采样参数，范围：0.0-1.0
	// @Example 0.9
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.9"`
	// 元数据（可选）
	// @Description 可选的自定义元数据
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
}

// UpdateSessionRequest 更新会话请求
// @Description 更新现有会话配置的请求体，所有字段都是可选的
type UpdateSessionRequest struct {
	// 会话标题（可选）
	// @Description 更新会话的显示标题
	// @Example 更新后的标题
	Title *string `json:"title,omitempty" validate:"omitempty,max=255" example:"更新后的标题"`
	// 系统提示词（可选）
	// @Description 更新系统级提示词
	// @Example 你是一个专业的编程助手
	SystemPrompt *string `json:"systemPrompt,omitempty" example:"你是一个专业的编程助手"`
	// 温度参数（可选，0-2）
	// @Description 更新温度参数，范围：0.0-2.0
	// @Example 0.8
	Temperature *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2" example:"0.8"`
	// TopP参数（可选，0-1）
	// @Description 更新核采样参数，范围：0.0-1.0
	// @Example 0.95
	TopP *float64 `json:"topP,omitempty" validate:"omitempty,gte=0,lte=1" example:"0.95"`
	// 模型名称（可选）
	// @Description 更新会话使用的AI模型名称。系统会根据当前租户ID和新的模型名称从 model_configurations 表中查询配置。
	// @Example gpt-4-turbo
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
// @Description 向指定会话发送消息的请求体
type SendMessageRequest struct {
	// 会话ID
	// @Description 目标会话的唯一标识符
	// @Example 550e8400-e29b-41d4-a716-446655440000
	SessionID string `json:"sessionId" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 消息内容
	// @Description 用户发送的消息文本内容
	// @Example 你好，请介绍一下你自己
	Message string `json:"message" validate:"required" example:"你好，请介绍一下你自己"`
	// AI高级参数（可选）
	// @Description 可选的AI模型配置参数。如果指定了 modelName，系统会根据当前租户ID和模型名称从数据库查询配置并使用该模型；否则使用会话的默认模型。支持动态切换不同的AI提供商（Google AI、Azure OpenAI、阿里云百炼等）。
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
