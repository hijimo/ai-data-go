package model

import (
	"fmt"
	"net/http"
)

// 错误码常量定义
const (
	// 通用错误 (10xxx)
	ErrCodeSuccess       = 10000 // 成功
	ErrCodeBadRequest    = 10101 // 请求参数错误
	ErrCodeUnauthorized  = 10102 // 未授权
	ErrCodeForbidden     = 10103 // 禁止访问
	ErrCodeNotFound      = 10104 // 资源不存在
	ErrCodeInternalError = 10201 // 内部错误
	ErrCodeValidation    = 10301 // 参数验证失败

	// 会话管理错误 (30xxx)
	ErrCodeSessionNotFound      = 30104 // 会话不存在
	ErrCodeSessionExpired       = 30301 // 会话已过期
	ErrCodeSessionAccessDenied  = 30302 // 无权访问会话
	ErrCodeSessionCreateFailed  = 30401 // 会话创建失败
	ErrCodeSessionUpdateFailed  = 30402 // 会话更新失败
	ErrCodeSessionDeleteFailed  = 30403 // 会话删除失败

	// 上下文管理错误 (40xxx)
	ErrCodeContextBuildFailed   = 40201 // 上下文构建失败
	ErrCodeTokenExceeded        = 40302 // Token 超出限制
	ErrCodeContextNotFound      = 40104 // 上下文不存在
	ErrCodeContextInvalid       = 40303 // 上下文无效

	// 记忆管理错误 (50xxx)
	ErrCodeMemoryNotFound           = 50104 // 记忆不存在
	ErrCodeVectorGenerationFailed   = 50201 // 向量生成失败
	ErrCodeMemoryStoreFailed        = 50401 // 记忆存储失败
	ErrCodeMemoryRetrieveFailed     = 50402 // 记忆检索失败
	ErrCodeMemoryDeleteFailed       = 50403 // 记忆删除失败

	// AI 服务错误 (60xxx)
	ErrCodeAIServiceTimeout     = 60201 // AI 服务超时
	ErrCodeAIServiceError       = 60202 // AI 服务错误
	ErrCodeQuotaExceeded        = 60301 // 配额超出限制
	ErrCodeModelNotAvailable    = 60302 // 模型不可用
	ErrCodeProviderNotFound     = 60104 // 提供商不存在
	ErrCodeModelNotFound        = 60105 // 模型不存在
	ErrCodeStreamingFailed      = 60403 // 流式响应失败
)

// 错误消息常量
const (
	MsgSuccess                  = "成功"
	MsgBadRequest               = "请求参数错误"
	MsgUnauthorized             = "未授权"
	MsgForbidden                = "禁止访问"
	MsgNotFound                 = "资源不存在"
	MsgInternalError            = "内部错误"
	MsgValidation               = "参数验证失败"
	MsgSessionNotFound          = "会话不存在"
	MsgSessionExpired           = "会话已过期"
	MsgSessionAccessDenied      = "无权访问会话"
	MsgSessionCreateFailed      = "会话创建失败"
	MsgSessionUpdateFailed      = "会话更新失败"
	MsgSessionDeleteFailed      = "会话删除失败"
	MsgContextBuildFailed       = "上下文构建失败"
	MsgTokenExceeded            = "Token 超出限制"
	MsgContextNotFound          = "上下文不存在"
	MsgContextInvalid           = "上下文无效"
	MsgMemoryNotFound           = "记忆不存在"
	MsgVectorGenerationFailed   = "向量生成失败"
	MsgMemoryStoreFailed        = "记忆存储失败"
	MsgMemoryRetrieveFailed     = "记忆检索失败"
	MsgMemoryDeleteFailed       = "记忆删除失败"
	MsgAIServiceTimeout         = "AI 服务超时"
	MsgAIServiceError           = "AI 服务错误"
	MsgQuotaExceeded            = "配额超出限制"
	MsgModelNotAvailable        = "模型不可用"
	MsgProviderNotFound         = "提供商不存在"
	MsgModelNotFound            = "模型不存在"
	MsgStreamingFailed          = "流式响应失败"
)

// AppError 自定义应用错误类型
type AppError struct {
	Code       int    `json:"code"`        // 错误码
	Message    string `json:"message"`     // 错误消息
	Details    string `json:"details,omitempty"` // 错误详情
	HTTPStatus int    `json:"-"`           // HTTP 状态码
	Err        error  `json:"-"`           // 原始错误
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Details != "" {
		if e.Err != nil {
			return fmt.Sprintf("[%d] %s: %s (原因: %v)", e.Code, e.Message, e.Details, e.Err)
		}
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s (原因: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误，支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError 创建新的应用错误
func NewAppError(code int, message, details string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: getHTTPStatus(code),
	}
}

// WrapError 包装现有错误
func WrapError(code int, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// WrapErrorWithDetails 包装现有错误并添加详情
func WrapErrorWithDetails(code int, message, details string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: getHTTPStatus(code),
		Err:        err,
	}
}

// getHTTPStatus 根据错误码获取 HTTP 状态码
func getHTTPStatus(code int) int {
	switch {
	case code == ErrCodeSuccess:
		return http.StatusOK
	case code >= 10101 && code < 10200: // 客户端错误
		switch code {
		case ErrCodeBadRequest:
			return http.StatusBadRequest
		case ErrCodeUnauthorized:
			return http.StatusUnauthorized
		case ErrCodeForbidden:
			return http.StatusForbidden
		case ErrCodeNotFound:
			return http.StatusNotFound
		default:
			return http.StatusBadRequest
		}
	case code >= 10200 && code < 10300: // 服务器错误
		return http.StatusInternalServerError
	case code >= 10301 && code < 10400: // 验证错误
		return http.StatusBadRequest
	case code >= 30000 && code < 40000: // 会话管理错误
		switch code {
		case ErrCodeSessionNotFound:
			return http.StatusNotFound
		case ErrCodeSessionAccessDenied:
			return http.StatusForbidden
		case ErrCodeSessionExpired:
			return http.StatusGone
		default:
			return http.StatusInternalServerError
		}
	case code >= 40000 && code < 50000: // 上下文管理错误
		switch code {
		case ErrCodeContextNotFound:
			return http.StatusNotFound
		case ErrCodeTokenExceeded, ErrCodeContextInvalid:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	case code >= 50000 && code < 60000: // 记忆管理错误
		switch code {
		case ErrCodeMemoryNotFound:
			return http.StatusNotFound
		default:
			return http.StatusInternalServerError
		}
	case code >= 60000 && code < 70000: // AI 服务错误
		switch code {
		case ErrCodeProviderNotFound, ErrCodeModelNotFound:
			return http.StatusNotFound
		case ErrCodeQuotaExceeded:
			return http.StatusTooManyRequests
		case ErrCodeAIServiceTimeout:
			return http.StatusGatewayTimeout
		case ErrCodeModelNotAvailable:
			return http.StatusServiceUnavailable
		default:
			return http.StatusInternalServerError
		}
	default:
		return http.StatusInternalServerError
	}
}

// ============ 通用错误构造函数 ============

// NewBadRequestError 创建请求参数错误
func NewBadRequestError(details string) *AppError {
	return NewAppError(ErrCodeBadRequest, MsgBadRequest, details)
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(details string) *AppError {
	return NewAppError(ErrCodeUnauthorized, MsgUnauthorized, details)
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(details string) *AppError {
	return NewAppError(ErrCodeForbidden, MsgForbidden, details)
}

// NewNotFoundError 创建资源不存在错误
func NewNotFoundError(details string) *AppError {
	return NewAppError(ErrCodeNotFound, MsgNotFound, details)
}

// NewValidationError 创建参数验证错误
func NewValidationError(details string) *AppError {
	return NewAppError(ErrCodeValidation, MsgValidation, details)
}

// NewInternalError 创建内部错误
func NewInternalError(err error) *AppError {
	return WrapError(ErrCodeInternalError, MsgInternalError, err)
}

// ============ 会话管理错误构造函数 ============

// NewSessionNotFoundError 创建会话不存在错误
func NewSessionNotFoundError(sessionID string) *AppError {
	details := ""
	if sessionID != "" {
		details = fmt.Sprintf("会话 ID: %s", sessionID)
	}
	return NewAppError(ErrCodeSessionNotFound, MsgSessionNotFound, details)
}

// NewSessionExpiredError 创建会话已过期错误
func NewSessionExpiredError(sessionID string) *AppError {
	details := ""
	if sessionID != "" {
		details = fmt.Sprintf("会话 ID: %s", sessionID)
	}
	return NewAppError(ErrCodeSessionExpired, MsgSessionExpired, details)
}

// NewSessionAccessDeniedError 创建会话访问拒绝错误
func NewSessionAccessDeniedError() *AppError {
	return NewAppError(ErrCodeSessionAccessDenied, MsgSessionAccessDenied, "")
}

// NewSessionCreateFailedError 创建会话创建失败错误
func NewSessionCreateFailedError(err error) *AppError {
	return WrapError(ErrCodeSessionCreateFailed, MsgSessionCreateFailed, err)
}

// NewSessionUpdateFailedError 创建会话更新失败错误
func NewSessionUpdateFailedError(err error) *AppError {
	return WrapError(ErrCodeSessionUpdateFailed, MsgSessionUpdateFailed, err)
}

// NewSessionDeleteFailedError 创建会话删除失败错误
func NewSessionDeleteFailedError(err error) *AppError {
	return WrapError(ErrCodeSessionDeleteFailed, MsgSessionDeleteFailed, err)
}

// ============ 上下文管理错误构造函数 ============

// NewContextBuildFailedError 创建上下文构建失败错误
func NewContextBuildFailedError(err error) *AppError {
	return WrapError(ErrCodeContextBuildFailed, MsgContextBuildFailed, err)
}

// NewTokenExceededError 创建 Token 超出限制错误
func NewTokenExceededError(current, limit int) *AppError {
	details := fmt.Sprintf("当前 Token 数: %d, 限制: %d", current, limit)
	return NewAppError(ErrCodeTokenExceeded, MsgTokenExceeded, details)
}

// NewContextNotFoundError 创建上下文不存在错误
func NewContextNotFoundError(contextID string) *AppError {
	details := ""
	if contextID != "" {
		details = fmt.Sprintf("上下文 ID: %s", contextID)
	}
	return NewAppError(ErrCodeContextNotFound, MsgContextNotFound, details)
}

// NewContextInvalidError 创建上下文无效错误
func NewContextInvalidError(details string) *AppError {
	return NewAppError(ErrCodeContextInvalid, MsgContextInvalid, details)
}

// ============ 记忆管理错误构造函数 ============

// NewMemoryNotFoundError 创建记忆不存在错误
func NewMemoryNotFoundError(memoryID string) *AppError {
	details := ""
	if memoryID != "" {
		details = fmt.Sprintf("记忆 ID: %s", memoryID)
	}
	return NewAppError(ErrCodeMemoryNotFound, MsgMemoryNotFound, details)
}

// NewVectorGenerationFailedError 创建向量生成失败错误
func NewVectorGenerationFailedError(err error) *AppError {
	return WrapError(ErrCodeVectorGenerationFailed, MsgVectorGenerationFailed, err)
}

// NewMemoryStoreFailedError 创建记忆存储失败错误
func NewMemoryStoreFailedError(err error) *AppError {
	return WrapError(ErrCodeMemoryStoreFailed, MsgMemoryStoreFailed, err)
}

// NewMemoryRetrieveFailedError 创建记忆检索失败错误
func NewMemoryRetrieveFailedError(err error) *AppError {
	return WrapError(ErrCodeMemoryRetrieveFailed, MsgMemoryRetrieveFailed, err)
}

// NewMemoryDeleteFailedError 创建记忆删除失败错误
func NewMemoryDeleteFailedError(err error) *AppError {
	return WrapError(ErrCodeMemoryDeleteFailed, MsgMemoryDeleteFailed, err)
}

// ============ AI 服务错误构造函数 ============

// NewAIServiceTimeoutError 创建 AI 服务超时错误
func NewAIServiceTimeoutError(err error) *AppError {
	return WrapError(ErrCodeAIServiceTimeout, MsgAIServiceTimeout, err)
}

// NewAIServiceError 创建 AI 服务错误
func NewAIServiceError(err error) *AppError {
	return WrapError(ErrCodeAIServiceError, MsgAIServiceError, err)
}

// NewQuotaExceededError 创建配额超出限制错误
func NewQuotaExceededError(details string) *AppError {
	return NewAppError(ErrCodeQuotaExceeded, MsgQuotaExceeded, details)
}

// NewModelNotAvailableError 创建模型不可用错误
func NewModelNotAvailableError(modelID string) *AppError {
	details := ""
	if modelID != "" {
		details = fmt.Sprintf("模型 ID: %s", modelID)
	}
	return NewAppError(ErrCodeModelNotAvailable, MsgModelNotAvailable, details)
}

// NewProviderNotFoundError 创建提供商不存在错误
func NewProviderNotFoundError(providerID string) *AppError {
	details := ""
	if providerID != "" {
		details = fmt.Sprintf("提供商 ID: %s", providerID)
	}
	return NewAppError(ErrCodeProviderNotFound, MsgProviderNotFound, details)
}

// NewModelNotFoundError 创建模型不存在错误
func NewModelNotFoundError(modelID string) *AppError {
	details := ""
	if modelID != "" {
		details = fmt.Sprintf("模型 ID: %s", modelID)
	}
	return NewAppError(ErrCodeModelNotFound, MsgModelNotFound, details)
}

// NewStreamingFailedError 创建流式响应失败错误
func NewStreamingFailedError(err error) *AppError {
	return WrapError(ErrCodeStreamingFailed, MsgStreamingFailed, err)
}
