package model

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		want     string
	}{
		{
			name: "仅包含消息",
			appError: &AppError{
				Code:    ErrCodeBadRequest,
				Message: MsgBadRequest,
			},
			want: "[10101] 请求参数错误",
		},
		{
			name: "包含详情",
			appError: &AppError{
				Code:    ErrCodeBadRequest,
				Message: MsgBadRequest,
				Details: "缺少必填字段 'name'",
			},
			want: "[10101] 请求参数错误: 缺少必填字段 'name'",
		},
		{
			name: "包含原始错误",
			appError: &AppError{
				Code:    ErrCodeInternalError,
				Message: MsgInternalError,
				Err:     errors.New("数据库连接失败"),
			},
			want: "[10201] 内部错误 (原因: 数据库连接失败)",
		},
		{
			name: "包含详情和原始错误",
			appError: &AppError{
				Code:    ErrCodeSessionCreateFailed,
				Message: MsgSessionCreateFailed,
				Details: "用户 ID: user123",
				Err:     errors.New("数据库写入失败"),
			},
			want: "[30401] 会话创建失败: 用户 ID: user123 (原因: 数据库写入失败)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appError.Error(); got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	appErr := WrapError(ErrCodeInternalError, MsgInternalError, originalErr)

	if unwrapped := appErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("AppError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// 测试没有原始错误的情况
	appErr2 := NewAppError(ErrCodeBadRequest, MsgBadRequest, "")
	if unwrapped := appErr2.Unwrap(); unwrapped != nil {
		t.Errorf("AppError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"成功", ErrCodeSuccess, http.StatusOK},
		{"请求参数错误", ErrCodeBadRequest, http.StatusBadRequest},
		{"未授权", ErrCodeUnauthorized, http.StatusUnauthorized},
		{"禁止访问", ErrCodeForbidden, http.StatusForbidden},
		{"资源不存在", ErrCodeNotFound, http.StatusNotFound},
		{"参数验证失败", ErrCodeValidation, http.StatusBadRequest},
		{"内部错误", ErrCodeInternalError, http.StatusInternalServerError},
		{"会话不存在", ErrCodeSessionNotFound, http.StatusNotFound},
		{"会话已过期", ErrCodeSessionExpired, http.StatusGone},
		{"会话访问拒绝", ErrCodeSessionAccessDenied, http.StatusForbidden},
		{"上下文不存在", ErrCodeContextNotFound, http.StatusNotFound},
		{"Token 超出限制", ErrCodeTokenExceeded, http.StatusBadRequest},
		{"记忆不存在", ErrCodeMemoryNotFound, http.StatusNotFound},
		{"提供商不存在", ErrCodeProviderNotFound, http.StatusNotFound},
		{"模型不存在", ErrCodeModelNotFound, http.StatusNotFound},
		{"配额超出限制", ErrCodeQuotaExceeded, http.StatusTooManyRequests},
		{"AI 服务超时", ErrCodeAIServiceTimeout, http.StatusGatewayTimeout},
		{"模型不可用", ErrCodeModelNotAvailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHTTPStatus(tt.code); got != tt.want {
				t.Errorf("getHTTPStatus(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestNewBadRequestError(t *testing.T) {
	details := "缺少必填字段"
	err := NewBadRequestError(details)

	if err.Code != ErrCodeBadRequest {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeBadRequest)
	}
	if err.Message != MsgBadRequest {
		t.Errorf("Message = %v, want %v", err.Message, MsgBadRequest)
	}
	if err.Details != details {
		t.Errorf("Details = %v, want %v", err.Details, details)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusBadRequest)
	}
}

func TestNewSessionNotFoundError(t *testing.T) {
	sessionID := "session123"
	err := NewSessionNotFoundError(sessionID)

	if err.Code != ErrCodeSessionNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeSessionNotFound)
	}
	if err.Message != MsgSessionNotFound {
		t.Errorf("Message = %v, want %v", err.Message, MsgSessionNotFound)
	}
	expectedDetails := "会话 ID: session123"
	if err.Details != expectedDetails {
		t.Errorf("Details = %v, want %v", err.Details, expectedDetails)
	}
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusNotFound)
	}
}

func TestNewTokenExceededError(t *testing.T) {
	current := 5000
	limit := 4096
	err := NewTokenExceededError(current, limit)

	if err.Code != ErrCodeTokenExceeded {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeTokenExceeded)
	}
	if err.Message != MsgTokenExceeded {
		t.Errorf("Message = %v, want %v", err.Message, MsgTokenExceeded)
	}
	expectedDetails := "当前 Token 数: 5000, 限制: 4096"
	if err.Details != expectedDetails {
		t.Errorf("Details = %v, want %v", err.Details, expectedDetails)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusBadRequest)
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("数据库连接失败")
	err := WrapError(ErrCodeInternalError, MsgInternalError, originalErr)

	if err.Code != ErrCodeInternalError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInternalError)
	}
	if err.Message != MsgInternalError {
		t.Errorf("Message = %v, want %v", err.Message, MsgInternalError)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
	if err.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("HTTPStatus = %v, want %v", err.HTTPStatus, http.StatusInternalServerError)
	}

	// 测试 Unwrap
	if unwrapped := errors.Unwrap(err); unwrapped != originalErr {
		t.Errorf("errors.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewAIServiceError(t *testing.T) {
	originalErr := errors.New("API 调用失败")
	err := NewAIServiceError(originalErr)

	if err.Code != ErrCodeAIServiceError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeAIServiceError)
	}
	if err.Message != MsgAIServiceError {
		t.Errorf("Message = %v, want %v", err.Message, MsgAIServiceError)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

func TestNewMemoryNotFoundError(t *testing.T) {
	// 测试带 ID 的情况
	memoryID := "memory456"
	err := NewMemoryNotFoundError(memoryID)

	if err.Code != ErrCodeMemoryNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeMemoryNotFound)
	}
	expectedDetails := "记忆 ID: memory456"
	if err.Details != expectedDetails {
		t.Errorf("Details = %v, want %v", err.Details, expectedDetails)
	}

	// 测试不带 ID 的情况
	err2 := NewMemoryNotFoundError("")
	if err2.Details != "" {
		t.Errorf("Details = %v, want empty string", err2.Details)
	}
}

func TestNewProviderNotFoundError(t *testing.T) {
	providerID := "openai"
	err := NewProviderNotFoundError(providerID)

	if err.Code != ErrCodeProviderNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeProviderNotFound)
	}
	expectedDetails := "提供商 ID: openai"
	if err.Details != expectedDetails {
		t.Errorf("Details = %v, want %v", err.Details, expectedDetails)
	}
}

func TestNewModelNotFoundError(t *testing.T) {
	modelID := "gpt-4"
	err := NewModelNotFoundError(modelID)

	if err.Code != ErrCodeModelNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeModelNotFound)
	}
	expectedDetails := "模型 ID: gpt-4"
	if err.Details != expectedDetails {
		t.Errorf("Details = %v, want %v", err.Details, expectedDetails)
	}
}
