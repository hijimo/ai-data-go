// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestAzureAIError_Error 测试 AzureAIError 的 Error 方法
func TestAzureAIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AzureAIError
		expected string
	}{
		{
			name: "配置错误",
			err: &AzureAIError{
				Type:    "config",
				Code:    "invalid_config",
				Message: "缺少 API Key",
				Details: map[string]string{"field": "APIKey"},
			},
			expected: "[config] invalid_config: 缺少 API Key",
		},
		{
			name: "网络错误带原始错误",
			err: &AzureAIError{
				Type:    "network",
				Code:    "network_error",
				Message: "连接超时",
				Err:     errors.New("dial tcp: timeout"),
			},
			expected: "[network] network_error: 连接超时 (caused by: dial tcp: timeout)",
		},
		{
			name: "API 错误",
			err: &AzureAIError{
				Type:    "api",
				Code:    "401",
				Message: "未授权",
				Details: map[string]string{"error": "invalid_api_key"},
			},
			expected: "[api] 401: 未授权",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestAzureAIError_Unwrap 测试 AzureAIError 的 Unwrap 方法
func TestAzureAIError_Unwrap(t *testing.T) {
	originalErr := errors.New("原始错误")
	azErr := &AzureAIError{
		Type:    "network",
		Code:    "network_error",
		Message: "网络错误",
		Err:     originalErr,
	}

	unwrapped := azErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// 测试没有原始错误的情况
	azErr2 := &AzureAIError{
		Type:    "config",
		Code:    "invalid_config",
		Message: "配置错误",
	}

	if azErr2.Unwrap() != nil {
		t.Errorf("Unwrap() should return nil when Err is nil")
	}
}

// TestNewConfigError 测试 NewConfigError 函数
func TestNewConfigError(t *testing.T) {
	message := "缺少必需的配置"
	details := map[string]string{"field": "APIKey"}

	err := NewConfigError(message, details)

	if err.Type != "config" {
		t.Errorf("Type = %v, want config", err.Type)
	}
	if err.Code != "invalid_config" {
		t.Errorf("Code = %v, want invalid_config", err.Code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	// 注意：map 不能直接比较，只能比较是否为 nil
	if err.Details == nil {
		t.Errorf("Details should not be nil")
	}
	if err.Err != nil {
		t.Errorf("Err should be nil")
	}
}

// TestNewRequestError 测试 NewRequestError 函数
func TestNewRequestError(t *testing.T) {
	message := "无效的请求格式"
	originalErr := errors.New("json: invalid syntax")

	err := NewRequestError(message, originalErr)

	if err.Type != "request" {
		t.Errorf("Type = %v, want request", err.Type)
	}
	if err.Code != "invalid_request" {
		t.Errorf("Code = %v, want invalid_request", err.Code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

// TestNewNetworkError 测试 NewNetworkError 函数
func TestNewNetworkError(t *testing.T) {
	message := "网络连接失败"
	originalErr := errors.New("connection refused")

	err := NewNetworkError(message, originalErr)

	if err.Type != "network" {
		t.Errorf("Type = %v, want network", err.Type)
	}
	if err.Code != "network_error" {
		t.Errorf("Code = %v, want network_error", err.Code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

// TestNewAPIError 测试 NewAPIError 函数
func TestNewAPIError(t *testing.T) {
	code := "429"
	message := "请求过于频繁"
	details := map[string]any{
		"retry_after": 60,
		"limit":       100,
	}

	err := NewAPIError(code, message, details)

	if err.Type != "api" {
		t.Errorf("Type = %v, want api", err.Type)
	}
	if err.Code != code {
		t.Errorf("Code = %v, want %v", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	// 注意：map 不能直接比较，只能比较是否为 nil
	if err.Details == nil {
		t.Errorf("Details should not be nil")
	}
}

// TestNewParseError 测试 NewParseError 函数
func TestNewParseError(t *testing.T) {
	message := "解析响应失败"
	originalErr := errors.New("unexpected end of JSON input")

	err := NewParseError(message, originalErr)

	if err.Type != "parse" {
		t.Errorf("Type = %v, want parse", err.Type)
	}
	if err.Code != "parse_error" {
		t.Errorf("Code = %v, want parse_error", err.Code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

// TestErrorChaining 测试错误链
func TestErrorChaining(t *testing.T) {
	// 创建一个错误链
	originalErr := errors.New("底层错误")
	networkErr := NewNetworkError("网络错误", originalErr)

	// 使用 errors.Unwrap 遍历错误链
	unwrapped := errors.Unwrap(networkErr)
	if unwrapped != originalErr {
		t.Errorf("errors.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

// TestErrorTypes 测试不同类型的错误
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		createError  func() error
		expectedType string
		expectedCode string
	}{
		{
			name:         "配置错误",
			createError:  func() error { return NewConfigError("配置错误", nil) },
			expectedType: "config",
			expectedCode: "invalid_config",
		},
		{
			name:         "请求错误",
			createError:  func() error { return NewRequestError("请求错误", nil) },
			expectedType: "request",
			expectedCode: "invalid_request",
		},
		{
			name:         "网络错误",
			createError:  func() error { return NewNetworkError("网络错误", nil) },
			expectedType: "network",
			expectedCode: "network_error",
		},
		{
			name:         "API错误",
			createError:  func() error { return NewAPIError("500", "API错误", nil) },
			expectedType: "api",
			expectedCode: "500",
		},
		{
			name:         "解析错误",
			createError:  func() error { return NewParseError("解析错误", nil) },
			expectedType: "parse",
			expectedCode: "parse_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()
			azErr, ok := err.(*AzureAIError)
			if !ok {
				t.Fatalf("错误类型不是 *AzureAIError")
			}

			if azErr.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", azErr.Type, tt.expectedType)
			}
			if azErr.Code != tt.expectedCode {
				t.Errorf("Code = %v, want %v", azErr.Code, tt.expectedCode)
			}
		})
	}
}

// TestErrorFormatting 测试错误格式化
func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "简单错误",
			err:  NewConfigError("缺少 API Key", nil),
			contains: []string{
				"[config]",
				"invalid_config",
				"缺少 API Key",
			},
		},
		{
			name: "带原始错误的错误",
			err:  NewNetworkError("连接失败", errors.New("timeout")),
			contains: []string{
				"[network]",
				"network_error",
				"连接失败",
				"caused by",
				"timeout",
			},
		},
		{
			name: "API 错误",
			err:  NewAPIError("401", "未授权", map[string]string{"error": "invalid_key"}),
			contains: []string{
				"[api]",
				"401",
				"未授权",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("Error string %q should contain %q", errStr, substr)
				}
			}
		})
	}
}

// TestErrorDetails 测试错误详情
func TestErrorDetails(t *testing.T) {
	details := map[string]any{
		"field":  "temperature",
		"value":  3.0,
		"max":    2.0,
		"reason": "value exceeds maximum",
	}

	azErr := NewConfigError("配置值超出范围", details)

	detailsMap, ok := azErr.Details.(map[string]any)
	if !ok {
		t.Fatalf("Details 不是 map[string]any 类型")
	}

	if detailsMap["field"] != "temperature" {
		t.Errorf("Details[field] = %v, want temperature", detailsMap["field"])
	}
	if detailsMap["value"] != 3.0 {
		t.Errorf("Details[value] = %v, want 3.0", detailsMap["value"])
	}
}

// TestErrorWrapping 测试错误包装
func TestErrorWrapping(t *testing.T) {
	// 创建一个底层错误
	baseErr := fmt.Errorf("底层错误: %w", errors.New("原始错误"))

	// 包装成网络错误
	networkErr := NewNetworkError("网络请求失败", baseErr)

	// 验证错误消息包含所有层级
	errMsg := networkErr.Error()
	if !strings.Contains(errMsg, "网络请求失败") {
		t.Error("错误消息应包含网络错误描述")
	}
	if !strings.Contains(errMsg, "底层错误") {
		t.Error("错误消息应包含底层错误描述")
	}

	// 验证可以解包到原始错误
	unwrapped := errors.Unwrap(networkErr)
	if unwrapped != baseErr {
		t.Error("应该能够解包到底层错误")
	}
}
