// Package flows 提供 Flow 错误处理工具
package flows

import (
	"context"
	stdErrors "errors"
	"fmt"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
)

// FlowError Flow 执行错误
type FlowError struct {
	FlowName string
	Step     string
	Err      error
}

// Error 实现 error 接口
func (e *FlowError) Error() string {
	if e.Step != "" {
		return fmt.Sprintf("Flow '%s' 在步骤 '%s' 失败: %v", e.FlowName, e.Step, e.Err)
	}
	return fmt.Sprintf("Flow '%s' 失败: %v", e.FlowName, e.Err)
}

// Unwrap 返回原始错误
func (e *FlowError) Unwrap() error {
	return e.Err
}

// NewFlowError 创建 Flow 错误
func NewFlowError(flowName, step string, err error) *FlowError {
	return &FlowError{
		FlowName: flowName,
		Step:     step,
		Err:      err,
	}
}

// HandleFlowError 处理 Flow 错误
// 将各种错误类型转换为 AppError
func HandleFlowError(ctx context.Context, flowName string, err error) error {
	if err == nil {
		return nil
	}

	// 记录错误日志
	logger.ErrorContext(ctx, "Flow 执行失败", logger.Fields{
		"flow_name": flowName,
		"error":     err,
	})

	// 如果已经是 AppError，直接返回
	var appErr *errors.AppError
	if stdErrors.As(err, &appErr) {
		return appErr
	}

	// 如果是 FlowError，提取原始错误
	var flowErr *FlowError
	if stdErrors.As(err, &flowErr) {
		// 记录步骤信息
		logger.ErrorContext(ctx, "Flow 步骤失败", logger.Fields{
			"flow_name": flowErr.FlowName,
			"step":      flowErr.Step,
			"error":     flowErr.Err,
		})
		
		// 递归处理原始错误
		return HandleFlowError(ctx, flowName, flowErr.Err)
	}

	// 检查是否是上下文取消
	if stdErrors.Is(err, context.Canceled) {
		return errors.NewContextCancelledError()
	}

	// 检查是否是上下文超时
	if stdErrors.Is(err, context.DeadlineExceeded) {
		return errors.NewServiceUnavailableError("Flow 执行超时")
	}

	// 默认返回内部错误
	return errors.NewInternalError(err)
}

// WrapFlowStep 包装 Flow 步骤执行
// 自动捕获错误并添加步骤信息
func WrapFlowStep(flowName, step string, fn func() error) error {
	err := fn()
	if err != nil {
		return NewFlowError(flowName, step, err)
	}
	return nil
}

// RecoverFlowPanic 恢复 Flow panic
// 应该在 Flow 函数开始时使用 defer 调用
func RecoverFlowPanic(ctx context.Context, flowName string, errPtr *error) {
	if r := recover(); r != nil {
		// 记录 panic 日志
		logger.ErrorContext(ctx, "Flow panic", logger.Fields{
			"flow_name": flowName,
			"panic":     r,
		})

		// 将 panic 转换为错误
		var err error
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("%s", v)
		default:
			err = fmt.Errorf("panic: %v", r)
		}

		// 设置错误
		*errPtr = errors.NewInternalError(err)
	}
}

// ValidateFlowInput 验证 Flow 输入
// 返回验证错误
func ValidateFlowInput(ctx context.Context, flowName string, validator func() error) error {
	err := validator()
	if err != nil {
		logger.WarnContext(ctx, "Flow 输入验证失败", logger.Fields{
			"flow_name": flowName,
			"error":     err,
		})
		return errors.NewValidationError(err.Error())
	}
	return nil
}

// RetryableError 可重试错误接口
type RetryableError interface {
	error
	IsRetryable() bool
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否实现了 RetryableError 接口
	var retryable RetryableError
	if stdErrors.As(err, &retryable) {
		return retryable.IsRetryable()
	}

	// 检查是否是 AppError
	var appErr *errors.AppError
	if stdErrors.As(err, &appErr) {
		// 根据错误码判断是否可重试
		return isRetryableErrorCode(appErr.Code)
	}

	// 默认不可重试
	return false
}

// isRetryableErrorCode 判断错误码是否可重试
func isRetryableErrorCode(code int) bool {
	switch code {
	case errors.CodeAIServiceError,
		errors.CodeServiceUnavailable,
		errors.CodeContextCancelled,
		errors.CodeVectorGenerationFailed,
		errors.CodeVectorSearchFailed,
		errors.CodeChatGenerationFailed,
		errors.CodeChatStreamFailed:
		return true
	default:
		return false
	}
}

// ErrorWithRetry 带重试信息的错误
type ErrorWithRetry struct {
	Err        error
	Retryable  bool
	RetryAfter int // 建议重试延迟（秒）
}

// Error 实现 error 接口
func (e *ErrorWithRetry) Error() string {
	return e.Err.Error()
}

// Unwrap 返回原始错误
func (e *ErrorWithRetry) Unwrap() error {
	return e.Err
}

// IsRetryable 实现 RetryableError 接口
func (e *ErrorWithRetry) IsRetryable() bool {
	return e.Retryable
}

// NewRetryableError 创建可重试错误
func NewRetryableError(err error, retryAfter int) *ErrorWithRetry {
	return &ErrorWithRetry{
		Err:        err,
		Retryable:  true,
		RetryAfter: retryAfter,
	}
}

// NewNonRetryableError 创建不可重试错误
func NewNonRetryableError(err error) *ErrorWithRetry {
	return &ErrorWithRetry{
		Err:       err,
		Retryable: false,
	}
}
