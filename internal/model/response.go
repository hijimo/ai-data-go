package model

// ResponseData 通用响应数据结构
// 用于所有非分页接口的标准响应格式
type ResponseData[T any] struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"success"`
	// 响应数据
	Data *T `json:"data,omitempty"`
}

// PaginationData 分页数据结构
type PaginationData[T any] struct {
	// 数据
	Data T `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// ResponsePaginationData 分页响应数据结构
// 用于所有分页列表接口的标准响应格式
type ResponsePaginationData[T any] struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"success"`
	// 分页数据
	Data PaginationData[T] `json:"data"`
}

// ErrorResponse 错误响应结构（用于 Swagger 文档）
type ErrorResponse struct {
	// 响应代码
	Code int `json:"code" example:"400"`
	// 响应信息
	Message string `json:"message" example:"请求参数错误"`
}

// EmptyData 空数据结构（用于无数据返回的成功响应）
type EmptyData struct{}

// SuccessResponse 成功响应结构（无数据）
type SuccessResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
}

// MessagePreview 消息预览
type MessagePreview struct {
	// 消息ID
	ID string `json:"id" example:"msg-123456"`
	// 角色
	Role string `json:"role" example:"user"`
	// 消息内容
	Content string `json:"content" example:"你好"`
	// 创建时间
	CreatedAt string `json:"createdAt" example:"2024-01-01T12:00:00Z"`
}

// SessionResponse 会话响应
type SessionResponse struct {
	// 会话ID
	ID string `json:"id" example:"session-123456"`
	// 用户ID
	UserID string `json:"userId" example:"user-123456"`
	// 会话标题
	Title string `json:"title" example:"我的第一个会话"`
	// 模型名称
	ModelName string `json:"modelName" example:"gpt-4"`
	// 系统提示词
	SystemPrompt string `json:"systemPrompt" example:"你是一个有帮助的AI助手"`
	// 温度参数
	Temperature *float64 `json:"temperature,omitempty" example:"0.7"`
	// TopP参数
	TopP *float64 `json:"topP,omitempty" example:"0.9"`
	// 创建时间
	CreatedAt string `json:"createdAt" example:"2024-01-01T12:00:00Z"`
	// 更新时间
	UpdatedAt string `json:"updatedAt" example:"2024-01-01T12:00:00Z"`
	// 消息数量
	MessageCount int `json:"messageCount" example:"10"`
	// 是否置顶
	IsPinned bool `json:"isPinned" example:"false"`
	// 是否归档
	IsArchived bool `json:"isArchived" example:"false"`
	// 最后一条消息
	LastMessage *MessagePreview `json:"lastMessage,omitempty"`
	// 元数据
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// Message 消息结构
type Message struct {
	// 消息ID
	ID string `json:"id" example:"msg-123456"`
	// 角色
	Role string `json:"role" example:"user"`
	// 消息内容
	Content string `json:"content" example:"你好"`
	// 序列号
	Sequence int `json:"sequence" example:"1"`
	// 创建时间
	CreatedAt string `json:"createdAt" example:"2024-01-01T12:00:00Z"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	// 消息ID
	MessageID string `json:"messageId" example:"msg-123456"`
	// 会话ID
	SessionID string `json:"sessionId" example:"session-123456"`
	// 用户消息
	UserMessage *Message `json:"userMessage"`
	// AI消息
	AIMessage *Message `json:"aiMessage"`
	// 模型名称
	Model string `json:"model" example:"gpt-4"`
	// 使用统计
	Usage *Usage `json:"usage,omitempty"`
}

// MessageDetailResponse 消息详情响应
type MessageDetailResponse struct {
	// 消息ID
	ID string `json:"id" example:"msg-123456"`
	// 会话ID
	SessionID string `json:"sessionId" example:"session-123456"`
	// 角色
	Role string `json:"role" example:"user"`
	// 消息内容
	Content string `json:"content" example:"你好"`
	// Token数量
	Tokens int `json:"tokens" example:"10"`
	// 序列号
	Sequence int `json:"sequence" example:"1"`
	// 创建时间
	CreatedAt string `json:"createdAt" example:"2024-01-01T12:00:00Z"`
	// 工具调用
	ToolCalls map[string]interface{} `json:"toolCalls,omitempty"`
	// 错误信息
	Error string `json:"error,omitempty" example:""`
	// 元数据
	Meta map[string]interface{} `json:"meta,omitempty"`
}
