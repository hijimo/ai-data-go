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

// LoginResponse 登录响应（用于 Swagger）
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

// AuthAuditPaginationData 审计日志分页数据（用于 Swagger）
type AuthAuditPaginationData struct {
	// 数据
	Data []AuthAuditItem `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// AuthAuditListResponse 审计日志列表响应（用于 Swagger）
type AuthAuditListResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"查询审计日志成功"`
	// 分页数据
	Data AuthAuditPaginationData `json:"data"`
}

// TenantPaginationData 租户分页数据（用于 Swagger）
type TenantPaginationData struct {
	// 数据
	Data []Tenant `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// TenantListResponse 租户列表响应（用于 Swagger）
type TenantListResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"获取租户列表成功"`
	// 分页数据
	Data TenantPaginationData `json:"data"`
}

// UserPaginationData 用户分页数据（用于 Swagger）
type UserPaginationData struct {
	// 数据
	Data []User `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// UserListResponse 用户列表响应（用于 Swagger）
type UserListResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"获取用户列表成功"`
	// 分页数据
	Data UserPaginationData `json:"data"`
}

// SessionPaginationData 会话分页数据（用于 Swagger）
type SessionPaginationData struct {
	// 数据
	Data []SessionResponse `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// SessionListResponse 会话列表响应（用于 Swagger）
type SessionListResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"获取会话列表成功"`
	// 分页数据
	Data SessionPaginationData `json:"data"`
}

// MessageDetailPaginationData 消息详情分页数据（用于 Swagger）
type MessageDetailPaginationData struct {
	// 数据
	Data []MessageDetailResponse `json:"data"`
	// 当前页码
	PageNo int `json:"pageNo" example:"1"`
	// 每页大小
	PageSize int `json:"pageSize" example:"10"`
	// 总记录数
	TotalCount int `json:"totalCount" example:"100"`
	// 总页数
	TotalPage int `json:"totalPage" example:"10"`
}

// MessageDetailListResponse 消息详情列表响应（用于 Swagger）
type MessageDetailListResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"获取消息列表成功"`
	// 分页数据
	Data MessageDetailPaginationData `json:"data"`
}

// UserDataResponse 用户数据响应（用于 Swagger）
type UserDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 用户数据
	Data *User `json:"data"`
}

// TenantDataResponse 租户数据响应（用于 Swagger）
type TenantDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 租户数据
	Data *Tenant `json:"data"`
}

// CreateTenantWithAdminData 创建租户并自动生成管理员的响应数据（用于 Swagger）
type CreateTenantWithAdminData struct {
	// 租户信息
	Tenant *Tenant `json:"tenant"`
	// 管理员用户信息
	AdminUser *User `json:"adminUser"`
	// 管理员初始密码（仅在创建时返回）
	AdminPassword string `json:"adminPassword" example:"Xy9#mK2$pL5@qR8!"`
}

// CreateTenantWithAdminResponse 创建租户并自动生成管理员的响应（用于 Swagger）
type CreateTenantWithAdminResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"租户创建成功"`
	// 响应数据
	Data *CreateTenantWithAdminData `json:"data"`
}

// LoginDataResponse 登录数据响应（用于 Swagger）
type LoginDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"登录成功"`
	// 登录数据
	Data *LoginResponse `json:"data"`
}

// SessionDataResponse 会话数据响应（用于 Swagger）
type SessionDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 会话数据
	Data *SessionResponse `json:"data"`
}

// AnyDataResponse 任意数据响应（用于 Swagger）
type AnyDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 数据
	Data interface{} `json:"data,omitempty"`
}

// MessageResponseData 消息响应数据（用于 Swagger）
type MessageResponseData struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 消息响应数据
	Data *MessageResponse `json:"data"`
}

// MessageDetailDataResponse 消息详情数据响应（用于 Swagger）
type MessageDetailDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 消息详情数据
	Data *MessageDetailResponse `json:"data"`
}

// ChatResponseData 对话响应数据（用于 Swagger）
type ChatResponseData struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 对话响应数据
	Data *ChatResponse `json:"data"`
}

// ProviderListDataResponse 提供商列表数据响应（用于 Swagger）
type ProviderListDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 提供商列表数据
	Data []Provider `json:"data"`
}

// ProviderDataResponse 提供商数据响应（用于 Swagger）
type ProviderDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 提供商数据
	Data *Provider `json:"data"`
}

// ModelListDataResponse 模型列表数据响应（用于 Swagger）
type ModelListDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 模型列表数据
	Data []Model `json:"data"`
}

// ModelDataResponse 模型数据响应（用于 Swagger）
type ModelDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 模型数据
	Data *Model `json:"data"`
}

// ParameterRuleListDataResponse 参数规则列表数据响应（用于 Swagger）
type ParameterRuleListDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 参数规则列表数据
	Data []ParameterRule `json:"data"`
}

// MetricsDataResponse 指标数据响应（用于 Swagger）
// 注意：MetricsSnapshot 类型来自 internal/monitoring 包
type MetricsDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 指标数据
	Data interface{} `json:"data"` // 使用 interface{} 避免循环依赖
}

// AlertListDataResponse 告警列表数据响应（用于 Swagger）
// 注意：Alert 类型来自 internal/monitoring 包
type AlertListDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 告警列表数据
	Data interface{} `json:"data"` // 使用 interface{} 避免循环依赖
}

// HealthDataResponse 健康检查数据响应（用于 Swagger）
type HealthDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"200"`
	// 响应信息
	Message string `json:"message" example:"操作成功"`
	// 健康检查数据
	Data map[string]interface{} `json:"data"`
}
