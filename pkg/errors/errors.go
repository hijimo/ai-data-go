package errors

import "fmt"

// 错误码常量定义
const (
	// CodeSuccess 成功
	CodeSuccess = 200

	// 客户端错误 4xx
	CodeBadRequest      = 400 // 请求参数错误
	CodeUnauthorized    = 401 // 未授权
	CodeForbidden       = 403 // 禁止访问
	CodeNotFound        = 404 // 资源不存在
	CodeValidationError = 422 // 参数验证失败

	// 服务器错误 5xx
	CodeInternalError      = 500 // 内部错误
	CodeServiceUnavailable = 503 // 服务不可用
	CodeAIServiceError     = 550 // AI 服务错误
	CodeContextCancelled   = 551 // 上下文已取消
	
	// 模型提供商相关错误 560-569
	CodeProviderNotFound = 560 // 提供商不存在
	CodeModelNotFound    = 561 // 模型不存在
	CodeLoadDataError    = 562 // 数据加载错误

	// 会话相关错误 570-579
	CodeSessionNotFound      = 570 // 会话不存在
	CodeSessionAccessDenied  = 571 // 无权访问会话
	CodeSessionAlreadyExists = 572 // 会话已存在

	// 消息相关错误 580-589
	CodeMessageNotFound     = 580 // 消息不存在
	CodeMessageAccessDenied = 581 // 无权访问消息
	CodeMessageSendFailed   = 582 // 消息发送失败

	// 摘要相关错误 590-599
	CodeSummaryGenerationFailed = 590 // 摘要生成失败
)

// 错误消息常量
const (
	MsgSuccess             = "成功"
	MsgBadRequest          = "请求参数错误"
	MsgUnauthorized        = "未授权"
	MsgForbidden           = "禁止访问"
	MsgNotFound            = "资源不存在"
	MsgValidationError     = "参数验证失败"
	MsgInternalError       = "内部错误"
	MsgServiceUnavailable  = "服务不可用"
	MsgAIServiceError      = "AI 服务错误"
	MsgContextCancelled    = "请求已取消"
	MsgProviderNotFound    = "提供商不存在"
	MsgModelNotFound       = "模型不存在"
	MsgLoadDataError       = "数据加载失败"
	MsgSessionNotFound          = "会话不存在"
	MsgSessionAccessDenied      = "无权访问会话"
	MsgMessageNotFound          = "消息不存在"
	MsgMessageAccessDenied      = "无权访问消息"
	MsgMessageSendFailed        = "消息发送失败"
	MsgSummaryGenerationFailed  = "摘要生成失败"
)

// AppError 自定义应用错误类型
type AppError struct {
	Code    int    // 错误码
	Message string // 错误消息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误，支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装现有错误
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义的错误构造函数

// NewBadRequestError 创建请求参数错误
func NewBadRequestError(message string) *AppError {
	if message == "" {
		message = MsgBadRequest
	}
	return New(CodeBadRequest, message)
}

// NewValidationError 创建参数验证错误
func NewValidationError(message string) *AppError {
	if message == "" {
		message = MsgValidationError
	}
	return New(CodeValidationError, message)
}

// NewNotFoundError 创建资源不存在错误
func NewNotFoundError(message string) *AppError {
	if message == "" {
		message = MsgNotFound
	}
	return New(CodeNotFound, message)
}

// NewInternalError 创建内部错误
func NewInternalError(err error) *AppError {
	return Wrap(CodeInternalError, MsgInternalError, err)
}

// NewAIServiceError 创建 AI 服务错误
func NewAIServiceError(err error) *AppError {
	return Wrap(CodeAIServiceError, MsgAIServiceError, err)
}

// NewContextCancelledError 创建上下文取消错误
func NewContextCancelledError() *AppError {
	return New(CodeContextCancelled, MsgContextCancelled)
}

// NewServiceUnavailableError 创建服务不可用错误
func NewServiceUnavailableError(message string) *AppError {
	if message == "" {
		message = MsgServiceUnavailable
	}
	return New(CodeServiceUnavailable, message)
}

// NewProviderNotFoundError 创建提供商不存在错误
func NewProviderNotFoundError(providerID string) *AppError {
	message := MsgProviderNotFound
	if providerID != "" {
		message = fmt.Sprintf("提供商 '%s' 不存在", providerID)
	}
	return New(CodeProviderNotFound, message)
}

// NewModelNotFoundError 创建模型不存在错误
func NewModelNotFoundError(modelID string) *AppError {
	message := MsgModelNotFound
	if modelID != "" {
		message = fmt.Sprintf("模型 '%s' 不存在", modelID)
	}
	return New(CodeModelNotFound, message)
}

// NewLoadDataError 创建数据加载错误
func NewLoadDataError(err error) *AppError {
	return Wrap(CodeLoadDataError, MsgLoadDataError, err)
}

// NewSessionNotFoundError 创建会话不存在错误
func NewSessionNotFoundError(sessionID string) *AppError {
	message := MsgSessionNotFound
	if sessionID != "" {
		message = fmt.Sprintf("会话 '%s' 不存在", sessionID)
	}
	return New(CodeSessionNotFound, message)
}

// NewSessionAccessDeniedError 创建会话访问拒绝错误
func NewSessionAccessDeniedError() *AppError {
	return New(CodeSessionAccessDenied, MsgSessionAccessDenied)
}

// NewMessageNotFoundError 创建消息不存在错误
func NewMessageNotFoundError(messageID string) *AppError {
	message := MsgMessageNotFound
	if messageID != "" {
		message = fmt.Sprintf("消息 '%s' 不存在", messageID)
	}
	return New(CodeMessageNotFound, message)
}

// NewMessageAccessDeniedError 创建消息访问拒绝错误
func NewMessageAccessDeniedError() *AppError {
	return New(CodeMessageAccessDenied, MsgMessageAccessDenied)
}

// NewMessageSendFailedError 创建消息发送失败错误
func NewMessageSendFailedError(err error) *AppError {
	return Wrap(CodeMessageSendFailed, MsgMessageSendFailed, err)
}

// NewSummaryGenerationFailedError 创建摘要生成失败错误
func NewSummaryGenerationFailedError(err error) *AppError {
	return Wrap(CodeSummaryGenerationFailed, MsgSummaryGenerationFailed, err)
}
