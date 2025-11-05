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
	// 追踪ID（用于全链路追踪和问题排查）
	TraceID string `json:"traceId,omitempty" example:"trace-1729756800-a1b2c3d4"`
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
	// 追踪ID（用于全链路追踪和问题排查）
	TraceID string `json:"traceId,omitempty" example:"trace-1729756800-a1b2c3d4"`
}

// ErrorResponse 错误响应结构（用于 Swagger 文档）
// @name ErrorResponse
type ErrorResponse struct {
	// 响应代码
	Code int `json:"code" example:"400"`
	// 响应信息
	Message string `json:"message" example:"请求参数错误"`
	// 追踪ID（用于全链路追踪和问题排查）
	TraceID string `json:"traceId,omitempty" example:"trace-1729756800-a1b2c3d4"`
}

// EmptyData 空数据结构（用于无数据返回的成功响应）
// @name EmptyData
type EmptyData struct{}

// SuccessResponse 成功响应结构（无数据）
// @name SuccessResponse
type SuccessResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 追踪ID（用于全链路追踪和问题排查）
	TraceID string `json:"traceId,omitempty" example:"trace-1729756800-a1b2c3d4"`
}

// MessagePreview 消息预览
// @name MessagePreview
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
// @name SessionResponse
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
// @name Message
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
// @name MessageResponse
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
// @name MessageDetailResponse
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

// LoginResponse 登录响应（用于 Swagger）
// @name LoginResponse
type LoginResponse struct {
	// 访问令牌
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// 刷新令牌
	RefreshToken string `json:"refreshToken" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 过期时间（秒）
	ExpiresIn int64 `json:"expiresIn" example:"3600"`
	// 令牌类型
	TokenType string `json:"tokenType" example:"Bearer"`
	// 用户信息
	User *User `json:"user"`
}

// AuthAuditItem 审计日志项（用于 Swagger）
// @name AuthAuditItem
type AuthAuditItem struct {
	// 审计日志ID
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 租户ID
	TenantID *string `json:"tenantId" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 用户ID
	UserID *string `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 事件类型
	Event string `json:"event" example:"login"`
	// 客户端IP地址
	IP string `json:"ip" example:"192.168.1.1"`
	// 用户代理字符串
	UserAgent string `json:"userAgent" example:"Mozilla/5.0"`
	// 事件元数据
	Meta interface{} `json:"meta"`
	// 事件发生时间
	CreatedAt string `json:"createdAt" example:"2024-01-01T12:00:00Z"`
}

// 类型别名：使用泛型 ResponsePaginationData 替代重复定义
// 这些别名仅用于 Swagger 文档的可读性，实际代码应直接使用泛型类型

// AuthAuditListResponse 审计日志列表响应（用于 Swagger）
// @name AuthAuditListResponse
type AuthAuditListResponse = ResponsePaginationData[[]AuthAuditItem]

// TenantListResponse 租户列表响应（用于 Swagger）
// @name TenantListResponse
type TenantListResponse = ResponsePaginationData[[]Tenant]

// UserListResponse 用户列表响应（用于 Swagger）
// @name UserListResponse
type UserListResponse = ResponsePaginationData[[]User]

// SessionListResponse 会话列表响应（用于 Swagger）
// @name SessionListResponse
type SessionListResponse = ResponsePaginationData[[]SessionResponse]

// MessageDetailListResponse 消息详情列表响应（用于 Swagger）
// @name MessageDetailListResponse
type MessageDetailListResponse = ResponsePaginationData[[]MessageDetailResponse]

// CreateTenantWithAdminData 创建租户并自动生成管理员的响应数据（用于 Swagger）
// @name CreateTenantWithAdminData
type CreateTenantWithAdminData struct {
	// 租户信息
	Tenant *Tenant `json:"tenant"`
	// 管理员用户信息
	AdminUser *User `json:"adminUser"`
	// 管理员初始密码（仅在创建时返回）
	AdminPassword string `json:"adminPassword" example:"Xy9#mK2$pL5@qR8!"`
}

// 类型别名：使用泛型 ResponseData 替代重复定义
// 这些别名仅用于 Swagger 文档的可读性，实际代码应直接使用泛型类型

// CreateTenantWithAdminResponse 创建租户并自动生成管理员的响应（用于 Swagger）
// @name CreateTenantWithAdminResponse
type CreateTenantWithAdminResponse = ResponseData[CreateTenantWithAdminData]

// LoginDataResponse 登录数据响应（用于 Swagger）
// @name LoginDataResponse
type LoginDataResponse = ResponseData[LoginResponse]

// SessionDataResponse 会话数据响应（用于 Swagger）
// @name SessionDataResponse
type SessionDataResponse = ResponseData[SessionResponse]

// MessageResponseData 消息响应数据（用于 Swagger）
// @name MessageResponseData
type MessageResponseData = ResponseData[MessageResponse]

// MessageDetailDataResponse 消息详情数据响应（用于 Swagger）
// @name MessageDetailDataResponse
type MessageDetailDataResponse = ResponseData[MessageDetailResponse]

// ChatResponseData 对话响应数据（用于 Swagger）
// @name ChatResponseData
type ChatResponseData = ResponseData[ChatResponse]

// ProviderListDataResponse 提供商列表数据响应（用于 Swagger）
// @name ProviderListDataResponse
type ProviderListDataResponse = ResponseData[[]Provider]

// ProviderDataResponse 提供商数据响应（用于 Swagger）
// @name ProviderDataResponse
type ProviderDataResponse = ResponseData[Provider]

// ModelListDataResponse 模型列表数据响应（用于 Swagger）
// @name ModelListDataResponse
type ModelListDataResponse = ResponseData[[]Model]

// ModelDataResponse 模型数据响应（用于 Swagger）
// @name ModelDataResponse
type ModelDataResponse = ResponseData[Model]

// ParameterRuleListDataResponse 参数规则列表数据响应（用于 Swagger）
// @name ParameterRuleListDataResponse
type ParameterRuleListDataResponse = ResponseData[[]ParameterRule]

// MetricsDataResponse 指标数据响应（用于 Swagger）
// @name MetricsDataResponse
// 注意：使用 interface{} 避免与 internal/monitoring 包的循环依赖
type MetricsDataResponse = ResponseData[interface{}]

// AlertListDataResponse 告警列表数据响应（用于 Swagger）
// @name AlertListDataResponse
// 注意：使用 interface{} 避免与 internal/monitoring 包的循环依赖
type AlertListDataResponse = ResponseData[interface{}]

// HealthDataResponse 健康检查数据响应（用于 Swagger）
// @name HealthDataResponse
type HealthDataResponse = ResponseData[map[string]interface{}]

// UserDataResponse 用户数据响应（用于 Swagger）
// @name UserDataResponse
type UserDataResponse = ResponseData[User]

// TenantDataResponse 租户数据响应（用于 Swagger）
// @name TenantDataResponse
type TenantDataResponse = ResponseData[Tenant]

// AnyDataResponse 任意数据响应（用于 Swagger）
// @name AnyDataResponse
// 用于返回空数据或任意类型数据的成功响应
type AnyDataResponse = ResponseData[interface{}]
