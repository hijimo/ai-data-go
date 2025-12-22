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

// ============================================================================
// 乐享知识库相关响应类型（用于 Swagger）
// ============================================================================

// LexiangSpaceResponse 乐享知识库响应
// @name LexiangSpaceResponse
type LexiangSpaceResponse struct {
	ID                 string `json:"id" example:"space_123"`
	Name               string `json:"name" example:"我的知识库"`
	Logo               string `json:"logo,omitempty"`
	VisibleType        int    `json:"visibleType" example:"0"`
	ManagerInheritType string `json:"managerInheritType" example:"manager"`
	MemberInheritType  string `json:"memberInheritType" example:"viewer"`
	TeamID             string `json:"teamId" example:"team_123"`
	RootEntryID        string `json:"rootEntryId" example:"entry_root_123"`
}

// LexiangSpaceItem 乐享知识库列表项
// @name LexiangSpaceItem
type LexiangSpaceItem struct {
	ID          string `json:"id" example:"space_123"`
	Name        string `json:"name" example:"我的知识库"`
	Logo        string `json:"logo,omitempty"`
	RootEntryID string `json:"rootEntryId" example:"entry_root_123"`
}

// LexiangEntryResponse 乐享知识节点响应
// @name LexiangEntryResponse
type LexiangEntryResponse struct {
	ID                string `json:"id" example:"entry_123"`
	Name              string `json:"name" example:"文档.pdf"`
	EntryType         string `json:"entryType" example:"file"`
	HasChildren       bool   `json:"hasChildren" example:"false"`
	CreatedAt         string `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt         string `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
	MemberInheritType string `json:"memberInheritType" example:"viewer"`
	DownloadURL       string `json:"downloadUrl,omitempty"`
}

// LexiangEntryItem 乐享知识节点列表项
// @name LexiangEntryItem
type LexiangEntryItem struct {
	ID          string `json:"id" example:"entry_123"`
	Name        string `json:"name" example:"文档.pdf"`
	EntryType   string `json:"entryType" example:"file"`
	HasChildren bool   `json:"hasChildren" example:"false"`
}

// LexiangUploadSignResponse 乐享上传签名响应
// @name LexiangUploadSignResponse
type LexiangUploadSignResponse struct {
	State     string `json:"state" example:"upload_state_xxx"`
	Key       string `json:"key" example:"path/to/file"`
	Bucket    string `json:"bucket" example:"bucket-name"`
	Region    string `json:"region" example:"ap-guangzhou"`
	UploadURL string `json:"uploadUrl" example:"https://bucket.cos.region.myqcloud.com/path/to/file"`
}

// LexiangFeedbackItem 乐享反馈列表项
// @name LexiangFeedbackItem
type LexiangFeedbackItem struct {
	ID         string `json:"id" example:"feedback_123"`
	Status     string `json:"status" example:"unprocessed"`
	Type       string `json:"type" example:"kb_content_mistake"`
	Content    string `json:"content" example:"内容有误"`
	CreatedAt  string `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
	OwnerID    string `json:"ownerId" example:"user_123"`
	EntryID    string `json:"entryId" example:"entry_123"`
}

// LexiangSpaceDataResponse 乐享知识库数据响应（用于 Swagger）
// @name LexiangSpaceDataResponse
type LexiangSpaceDataResponse = ResponseData[LexiangSpaceResponse]

// LexiangSpaceListDataResponse 乐享知识库列表数据响应（用于 Swagger）
// @name LexiangSpaceListDataResponse
type LexiangSpaceListDataResponse = ResponseData[[]LexiangSpaceItem]

// LexiangEntryDataResponse 乐享知识节点数据响应（用于 Swagger）
// @name LexiangEntryDataResponse
type LexiangEntryDataResponse = ResponseData[LexiangEntryResponse]

// LexiangEntryListDataResponse 乐享知识节点列表数据响应（用于 Swagger）
// @name LexiangEntryListDataResponse
type LexiangEntryListDataResponse = ResponseData[[]LexiangEntryItem]

// LexiangUploadSignDataResponse 乐享上传签名数据响应（用于 Swagger）
// @name LexiangUploadSignDataResponse
type LexiangUploadSignDataResponse = ResponseData[LexiangUploadSignResponse]

// LexiangFeedbackListDataResponse 乐享反馈列表数据响应（用于 Swagger）
// @name LexiangFeedbackListDataResponse
type LexiangFeedbackListDataResponse = ResponseData[[]LexiangFeedbackItem]

// LexiangEntryContentResponse 乐享线上文档内容响应
// @name LexiangEntryContentResponse
type LexiangEntryContentResponse struct {
	Name        string `json:"name" example:"在线文档"`
	HTMLContent string `json:"htmlContent" example:"<p>文档内容</p>"`
}

// LexiangEntryContentDataResponse 乐享线上文档内容数据响应（用于 Swagger）
// @name LexiangEntryContentDataResponse
type LexiangEntryContentDataResponse = ResponseData[LexiangEntryContentResponse]

// LexiangUploadStateResponse 乐享上传状态响应
// @name LexiangUploadStateResponse
type LexiangUploadStateResponse struct {
	State string `json:"state" example:"upload_state_xxx"`
}

// LexiangUploadStateDataResponse 乐享上传状态数据响应（用于 Swagger）
// @name LexiangUploadStateDataResponse
type LexiangUploadStateDataResponse = ResponseData[LexiangUploadStateResponse]

// LexiangDocFileResponse 乐享附件详情响应
// @name LexiangDocFileResponse
type LexiangDocFileResponse struct {
	ID          string `json:"id" example:"file_123"`
	Name        string `json:"name" example:"附件.pdf"`
	DownloadURL string `json:"downloadUrl" example:"https://example.com/download"`
}

// LexiangDocFileDataResponse 乐享附件详情数据响应（用于 Swagger）
// @name LexiangDocFileDataResponse
type LexiangDocFileDataResponse = ResponseData[LexiangDocFileResponse]

// ============================================================================
// 乐享 AI 问答和搜索响应类型（用于 Swagger）
// ============================================================================

// LexiangReferenceChunk 参考内容段落
// @name LexiangReferenceChunk
type LexiangReferenceChunk struct {
	Content    string `json:"content" example:"这是匹配的内容段落..."`
	TargetID   string `json:"targetId" example:"entry_123"`
	TargetType string `json:"targetType" example:"kb_entry"`
	Title      string `json:"title" example:"知识库使用指南"`
	URL        string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// LexiangReferenceDoc 参考文档来源
// @name LexiangReferenceDoc
type LexiangReferenceDoc struct {
	Title string `json:"title" example:"知识库使用指南"`
	URL   string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// LexiangAdditionalContent 附加内容信息
// @name LexiangAdditionalContent
type LexiangAdditionalContent struct {
	GeneratedQuestion string                  `json:"generatedQuestion,omitempty" example:"如何使用乐享知识库？"`
	ReferenceChunks   []LexiangReferenceChunk `json:"referenceChunks,omitempty"`
	ReferenceDocs     []LexiangReferenceDoc   `json:"referenceDocs,omitempty"`
}

// LexiangAIQAResponse AI问答响应数据
// @name LexiangAIQAResponse
type LexiangAIQAResponse struct {
	Content           string                    `json:"content" example:"知识库是用于存储和管理文档的工具..."`
	AnswerSource      string                    `json:"answerSource" example:"internal"`
	ReasoningContent  string                    `json:"reasoningContent,omitempty" example:"让我思考一下这个问题..."`
	SessionID         string                    `json:"sessionId" example:"session_abc123"`
	AdditionalContent *LexiangAdditionalContent `json:"additionalContent,omitempty"`
}

// LexiangAIQADataResponse 乐享AI问答数据响应（用于 Swagger）
// @name LexiangAIQADataResponse
type LexiangAIQADataResponse = ResponseData[LexiangAIQAResponse]

// LexiangAISearchResultItem AI搜索结果项
// @name LexiangAISearchResultItem
type LexiangAISearchResultItem struct {
	Title   string `json:"title" example:"知识库使用指南"`
	Content string `json:"content" example:"## 如何使用知识库\n\n知识库是..."`
	URL     string `json:"url" example:"https://lexiang.tencent.com/kb/entry/123"`
}

// LexiangAISearchResponse AI搜索响应数据
// @name LexiangAISearchResponse
type LexiangAISearchResponse struct {
	List []LexiangAISearchResultItem `json:"list"`
}

// LexiangAISearchDataResponse 乐享AI搜索数据响应（用于 Swagger）
// @name LexiangAISearchDataResponse
type LexiangAISearchDataResponse = ResponseData[LexiangAISearchResponse]
