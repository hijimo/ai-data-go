package llm

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	// 配置相关错误
	ErrInvalidProviderType = errors.New("无效的提供商类型")
	ErrMissingAPIKey      = errors.New("缺少API密钥")
	ErrInvalidConfig      = errors.New("无效的配置")
	ErrProviderNotFound   = errors.New("提供商未找到")
	
	// 请求相关错误
	ErrInvalidRequest     = errors.New("无效的请求")
	ErrEmptyMessages      = errors.New("消息列表不能为空")
	ErrInvalidModel       = errors.New("无效的模型")
	ErrInvalidParameters  = errors.New("无效的参数")
	
	// 调用相关错误
	ErrAPICallFailed      = errors.New("API调用失败")
	ErrRateLimitExceeded  = errors.New("超出速率限制")
	ErrQuotaExceeded      = errors.New("超出配额")
	ErrModelNotAvailable  = errors.New("模型不可用")
	ErrServiceUnavailable = errors.New("服务不可用")
	ErrTimeout            = errors.New("请求超时")
	
	// 响应相关错误
	ErrInvalidResponse    = errors.New("无效的响应")
	ErrEmptyResponse      = errors.New("空响应")
	ErrStreamClosed       = errors.New("流已关闭")
	
	// 认证相关错误
	ErrUnauthorized       = errors.New("未授权")
	ErrInvalidAPIKey      = errors.New("无效的API密钥")
	ErrPermissionDenied   = errors.New("权限被拒绝")
)

// LLMError LLM错误结构
type LLMError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Type       string                 `json:"type"`
	Provider   ProviderType           `json:"provider"`
	Model      string                 `json:"model,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Underlying error                  `json:"-"`
}

// Error 实现error接口
func (e *LLMError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("[%s] %s: %s (underlying: %v)", e.Provider, e.Code, e.Message, e.Underlying)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
}

// Unwrap 支持errors.Unwrap
func (e *LLMError) Unwrap() error {
	return e.Underlying
}

// Is 支持errors.Is
func (e *LLMError) Is(target error) bool {
	if e.Underlying != nil {
		return errors.Is(e.Underlying, target)
	}
	return false
}

// NewLLMError 创建LLM错误
func NewLLMError(code, message string, provider ProviderType) *LLMError {
	return &LLMError{
		Code:     code,
		Message:  message,
		Provider: provider,
	}
}

// NewLLMErrorWithDetails 创建带详情的LLM错误
func NewLLMErrorWithDetails(code, message string, provider ProviderType, details map[string]interface{}) *LLMError {
	return &LLMError{
		Code:     code,
		Message:  message,
		Provider: provider,
		Details:  details,
	}
}

// WrapLLMError 包装底层错误
func WrapLLMError(code, message string, provider ProviderType, underlying error) *LLMError {
	return &LLMError{
		Code:       code,
		Message:    message,
		Provider:   provider,
		Underlying: underlying,
	}
}

// 错误码常量
const (
	// 配置错误码
	ErrCodeInvalidConfig     = "INVALID_CONFIG"
	ErrCodeMissingAPIKey     = "MISSING_API_KEY"
	ErrCodeInvalidProvider   = "INVALID_PROVIDER"
	
	// 请求错误码
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
	ErrCodeInvalidModel      = "INVALID_MODEL"
	ErrCodeInvalidParameters = "INVALID_PARAMETERS"
	
	// API错误码
	ErrCodeAPICallFailed     = "API_CALL_FAILED"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrCodeQuotaExceeded     = "QUOTA_EXCEEDED"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeModelNotAvailable = "MODEL_NOT_AVAILABLE"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout           = "TIMEOUT"
	
	// 响应错误码
	ErrCodeInvalidResponse   = "INVALID_RESPONSE"
	ErrCodeEmptyResponse     = "EMPTY_RESPONSE"
	ErrCodeStreamClosed      = "STREAM_CLOSED"
)

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		switch llmErr.Code {
		case ErrCodeTimeout, ErrCodeServiceUnavailable, ErrCodeAPICallFailed:
			return true
		case ErrCodeRateLimitExceeded:
			return true // 可以等待后重试
		default:
			return false
		}
	}
	return false
}

// IsQuotaError 判断是否为配额错误
func IsQuotaError(err error) bool {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == ErrCodeQuotaExceeded
	}
	return false
}

// IsAuthError 判断是否为认证错误
func IsAuthError(err error) bool {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == ErrCodeUnauthorized
	}
	return false
}

// IsRateLimitError 判断是否为速率限制错误
func IsRateLimitError(err error) bool {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == ErrCodeRateLimitExceeded
	}
	return false
}