package flows

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	pkgErrors "genkit-ai-service/pkg/errors"
)

func TestNewFlowError(t *testing.T) {
	err := errors.New("原始错误")
	flowErr := NewFlowError("testFlow", "step1", err)

	assert.Equal(t, "testFlow", flowErr.FlowName)
	assert.Equal(t, "step1", flowErr.Step)
	assert.Equal(t, err, flowErr.Err)
	assert.Contains(t, flowErr.Error(), "testFlow")
	assert.Contains(t, flowErr.Error(), "step1")
}

func TestFlowErrorUnwrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	flowErr := NewFlowError("testFlow", "step1", originalErr)

	unwrapped := errors.Unwrap(flowErr)
	assert.Equal(t, originalErr, unwrapped)
}

func TestHandleFlowError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		inputErr      error
		expectedType  interface{}
		expectedCode  int
	}{
		{
			name:         "nil 错误",
			inputErr:     nil,
			expectedType: nil,
		},
		{
			name:         "AppError",
			inputErr:     pkgErrors.NewBadRequestError("测试"),
			expectedType: &pkgErrors.AppError{},
			expectedCode: pkgErrors.CodeBadRequest,
		},
		{
			name:         "FlowError 包装的 AppError",
			inputErr:     NewFlowError("testFlow", "step1", pkgErrors.NewNotFoundError("资源不存在")),
			expectedType: &pkgErrors.AppError{},
			expectedCode: pkgErrors.CodeNotFound,
		},
		{
			name:         "Context Canceled",
			inputErr:     context.Canceled,
			expectedType: &pkgErrors.AppError{},
			expectedCode: pkgErrors.CodeContextCancelled,
		},
		{
			name:         "Context DeadlineExceeded",
			inputErr:     context.DeadlineExceeded,
			expectedType: &pkgErrors.AppError{},
			expectedCode: pkgErrors.CodeServiceUnavailable,
		},
		{
			name:         "普通错误",
			inputErr:     errors.New("普通错误"),
			expectedType: &pkgErrors.AppError{},
			expectedCode: pkgErrors.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleFlowError(ctx, "testFlow", tt.inputErr)

			if tt.expectedType == nil {
				assert.Nil(t, result)
			} else {
				assert.IsType(t, tt.expectedType, result)
				if appErr, ok := result.(*pkgErrors.AppError); ok {
					assert.Equal(t, tt.expectedCode, appErr.Code)
				}
			}
		})
	}
}

func TestWrapFlowStep(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		expectError bool
		expectStep  bool
	}{
		{
			name: "成功执行",
			fn: func() error {
				return nil
			},
			expectError: false,
		},
		{
			name: "执行失败",
			fn: func() error {
				return errors.New("步骤失败")
			},
			expectError: true,
			expectStep:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapFlowStep("testFlow", "step1", tt.fn)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectStep {
					var flowErr *FlowError
					assert.True(t, errors.As(err, &flowErr))
					assert.Equal(t, "testFlow", flowErr.FlowName)
					assert.Equal(t, "step1", flowErr.Step)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRecoverFlowPanic(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		fn          func()
		expectError bool
	}{
		{
			name: "无 panic",
			fn: func() {
				// 正常执行
			},
			expectError: false,
		},
		{
			name: "panic 字符串",
			fn: func() {
				panic("测试 panic")
			},
			expectError: true,
		},
		{
			name: "panic 错误",
			fn: func() {
				panic(errors.New("测试错误"))
			},
			expectError: true,
		},
		{
			name: "panic 其他类型",
			fn: func() {
				panic(123)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			func() {
				defer RecoverFlowPanic(ctx, "testFlow", &err)
				tt.fn()
			}()

			if tt.expectError {
				assert.Error(t, err)
				var appErr *pkgErrors.AppError
				assert.True(t, errors.As(err, &appErr))
				assert.Equal(t, pkgErrors.CodeInternalError, appErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFlowInput(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		validator   func() error
		expectError bool
	}{
		{
			name: "验证通过",
			validator: func() error {
				return nil
			},
			expectError: false,
		},
		{
			name: "验证失败",
			validator: func() error {
				return errors.New("验证失败")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFlowInput(ctx, "testFlow", tt.validator)

			if tt.expectError {
				assert.Error(t, err)
				var appErr *pkgErrors.AppError
				assert.True(t, errors.As(err, &appErr))
				assert.Equal(t, pkgErrors.CodeValidationError, appErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil 错误",
			err:      nil,
			expected: false,
		},
		{
			name:     "可重试的 AppError - AI 服务错误",
			err:      pkgErrors.NewAIServiceError(errors.New("AI 服务失败")),
			expected: true,
		},
		{
			name:     "可重试的 AppError - 服务不可用",
			err:      pkgErrors.NewServiceUnavailableError(""),
			expected: true,
		},
		{
			name:     "不可重试的 AppError - BadRequest",
			err:      pkgErrors.NewBadRequestError(""),
			expected: false,
		},
		{
			name:     "不可重试的 AppError - NotFound",
			err:      pkgErrors.NewNotFoundError(""),
			expected: false,
		},
		{
			name:     "RetryableError 接口 - 可重试",
			err:      NewRetryableError(errors.New("测试"), 5),
			expected: true,
		},
		{
			name:     "RetryableError 接口 - 不可重试",
			err:      NewNonRetryableError(errors.New("测试")),
			expected: false,
		},
		{
			name:     "普通错误",
			err:      errors.New("普通错误"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorWithRetry(t *testing.T) {
	originalErr := errors.New("原始错误")

	t.Run("可重试错误", func(t *testing.T) {
		err := NewRetryableError(originalErr, 5)
		assert.True(t, err.IsRetryable())
		assert.Equal(t, 5, err.RetryAfter)
		assert.Equal(t, originalErr, errors.Unwrap(err))
	})

	t.Run("不可重试错误", func(t *testing.T) {
		err := NewNonRetryableError(originalErr)
		assert.False(t, err.IsRetryable())
		assert.Equal(t, originalErr, errors.Unwrap(err))
	})
}

func TestIsRetryableErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected bool
	}{
		{"AI 服务错误", pkgErrors.CodeAIServiceError, true},
		{"服务不可用", pkgErrors.CodeServiceUnavailable, true},
		{"上下文取消", pkgErrors.CodeContextCancelled, true},
		{"向量生成失败", pkgErrors.CodeVectorGenerationFailed, true},
		{"向量检索失败", pkgErrors.CodeVectorSearchFailed, true},
		{"对话生成失败", pkgErrors.CodeChatGenerationFailed, true},
		{"流式对话失败", pkgErrors.CodeChatStreamFailed, true},
		{"请求参数错误", pkgErrors.CodeBadRequest, false},
		{"未授权", pkgErrors.CodeUnauthorized, false},
		{"资源不存在", pkgErrors.CodeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
