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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

// TestNewModelGenerator 测试 ModelGenerator 的创建
func TestNewModelGenerator(t *testing.T) {
	client := &http.Client{}
	baseURL := "https://test.openai.azure.com"
	apiKey := "test-key"
	apiVersion := "2025-04-01-preview"
	modelName := "gpt-4"

	gen := NewModelGenerator(client, baseURL, apiKey, apiVersion, modelName)

	if gen == nil {
		t.Fatal("NewModelGenerator 返回 nil")
	}

	if gen.client != client {
		t.Error("client 未正确设置")
	}

	if gen.baseURL != baseURL {
		t.Errorf("baseURL = %s, 期望 %s", gen.baseURL, baseURL)
	}

	if gen.apiKey != apiKey {
		t.Errorf("apiKey = %s, 期望 %s", gen.apiKey, apiKey)
	}

	if gen.apiVersion != apiVersion {
		t.Errorf("apiVersion = %s, 期望 %s", gen.apiVersion, apiVersion)
	}

	if gen.modelName != modelName {
		t.Errorf("modelName = %s, 期望 %s", gen.modelName, modelName)
	}
}

// TestWithMessages 测试 WithMessages 方法
func TestWithMessages(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("Hello"),
			},
		},
	}

	result := gen.WithMessages(messages)

	if result != gen {
		t.Error("WithMessages 应该返回相同的 generator 实例")
	}

	if len(gen.messages) != 1 {
		t.Errorf("messages 长度 = %d, 期望 1", len(gen.messages))
	}

	if gen.messages[0].Role != "user" {
		t.Errorf("消息角色 = %s, 期望 user", gen.messages[0].Role)
	}
}

// TestWithTools 测试 WithTools 方法
func TestWithTools(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{"type": "string"},
				},
			},
		},
	}

	result := gen.WithTools(tools)

	if result != gen {
		t.Error("WithTools 应该返回相同的 generator 实例")
	}

	if len(gen.tools) != 1 {
		t.Errorf("tools 长度 = %d, 期望 1", len(gen.tools))
	}

	if gen.tools[0].Type != "function" {
		t.Errorf("工具类型 = %s, 期望 function", gen.tools[0].Type)
	}

	if gen.tools[0].Function.Name != "get_weather" {
		t.Errorf("工具名称 = %s, 期望 get_weather", gen.tools[0].Function.Name)
	}
}

// TestWithConfig_MapConfig 测试使用 map 配置
func TestWithConfig_MapConfig(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	config := map[string]any{
		"temperature": 0.7,
		"max_tokens":  100,
		"top_p":       0.9,
	}

	result := gen.WithConfig(config)

	if result != gen {
		t.Error("WithConfig 应该返回相同的 generator 实例")
	}

	if gen.config == nil {
		t.Fatal("config 不应该为 nil")
	}

	if temp, ok := gen.config["temperature"].(float64); !ok || temp != 0.7 {
		t.Errorf("temperature = %v, 期望 0.7", gen.config["temperature"])
	}

	// max_tokens 可能是 int 或 float64，取决于输入
	maxTokensVal := gen.config["max_tokens"]
	var maxTokens int
	switch v := maxTokensVal.(type) {
	case int:
		maxTokens = v
	case float64:
		maxTokens = int(v)
	default:
		t.Errorf("max_tokens 类型错误: %T", maxTokensVal)
		return
	}

	if maxTokens != 100 {
		t.Errorf("max_tokens = %v, 期望 100", maxTokens)
	}
}

// TestWithConfig_NilConfig 测试 nil 配置
func TestWithConfig_NilConfig(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	result := gen.WithConfig(nil)

	if result != gen {
		t.Error("WithConfig 应该返回相同的 generator 实例")
	}

	if gen.config != nil {
		t.Error("config 应该为 nil")
	}
}

// TestBuildRequestURL 测试 URL 构建
func TestBuildRequestURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		apiVersion string
		want       string
	}{
		{
			name:       "默认 API 版本",
			baseURL:    "https://test.openai.azure.com",
			apiVersion: "2025-04-01-preview",
			want:       "https://test.openai.azure.com/openai/responses?api-version=2025-04-01-preview",
		},
		{
			name:       "自定义 API 版本",
			baseURL:    "https://custom.openai.azure.com",
			apiVersion: "2024-12-01",
			want:       "https://custom.openai.azure.com/openai/responses?api-version=2024-12-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewModelGenerator(&http.Client{}, tt.baseURL, "key", tt.apiVersion, "gpt-4")
			got := gen.buildRequestURL()

			if got != tt.want {
				t.Errorf("buildRequestURL() = %s, 期望 %s", got, tt.want)
			}
		})
	}
}

// TestBuildRequestBody 测试请求体构建
func TestBuildRequestBody(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	// 添加消息
	gen.messages = []Message{
		{Role: "user", Content: "Hello"},
	}

	// 添加配置
	gen.config = map[string]any{
		"temperature": 0.8,
		"max_tokens":  150,
	}

	req, err := gen.buildRequestBody(false)
	if err != nil {
		t.Fatalf("buildRequestBody() 错误: %v", err)
	}

	if req.Model != "gpt-4" {
		t.Errorf("Model = %s, 期望 gpt-4", req.Model)
	}

	if len(req.Input) != 1 {
		t.Errorf("Input 长度 = %d, 期望 1", len(req.Input))
	}

	if req.Stream != false {
		t.Error("Stream 应该为 false")
	}

	if req.Temperature == nil || *req.Temperature != 0.8 {
		t.Errorf("Temperature = %v, 期望 0.8", req.Temperature)
	}

	if req.MaxTokens == nil || *req.MaxTokens != 150 {
		t.Errorf("MaxTokens = %v, 期望 150", req.MaxTokens)
	}
}

// TestBuildRequestBody_WithStream 测试流式请求体构建
func TestBuildRequestBody_WithStream(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	gen.messages = []Message{
		{Role: "user", Content: "Hello"},
	}

	req, err := gen.buildRequestBody(true)
	if err != nil {
		t.Fatalf("buildRequestBody() 错误: %v", err)
	}

	if req.Stream != true {
		t.Error("Stream 应该为 true")
	}
}

// TestApplyConfig 测试配置应用
func TestApplyConfig(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]any
		check  func(*testing.T, *ResponsesRequest)
	}{
		{
			name: "Temperature 参数",
			config: map[string]any{
				"temperature": 0.7,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.Temperature == nil || *req.Temperature != 0.7 {
					t.Errorf("Temperature = %v, 期望 0.7", req.Temperature)
				}
			},
		},
		{
			name: "MaxTokens 参数（float64）",
			config: map[string]any{
				"max_tokens": 100.0,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.MaxTokens == nil || *req.MaxTokens != 100 {
					t.Errorf("MaxTokens = %v, 期望 100", req.MaxTokens)
				}
			},
		},
		{
			name: "MaxTokens 参数（int）",
			config: map[string]any{
				"maxTokens": 200,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.MaxTokens == nil || *req.MaxTokens != 200 {
					t.Errorf("MaxTokens = %v, 期望 200", req.MaxTokens)
				}
			},
		},
		{
			name: "TopP 参数",
			config: map[string]any{
				"top_p": 0.9,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.TopP == nil || *req.TopP != 0.9 {
					t.Errorf("TopP = %v, 期望 0.9", req.TopP)
				}
			},
		},
		{
			name: "FrequencyPenalty 参数",
			config: map[string]any{
				"frequency_penalty": 0.5,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.FrequencyPenalty == nil || *req.FrequencyPenalty != 0.5 {
					t.Errorf("FrequencyPenalty = %v, 期望 0.5", req.FrequencyPenalty)
				}
			},
		},
		{
			name: "PresencePenalty 参数",
			config: map[string]any{
				"presencePenalty": 0.3,
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.PresencePenalty == nil || *req.PresencePenalty != 0.3 {
					t.Errorf("PresencePenalty = %v, 期望 0.3", req.PresencePenalty)
				}
			},
		},
		{
			name: "Stop 参数",
			config: map[string]any{
				"stop": []string{"END", "STOP"},
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if len(req.Stop) != 2 {
					t.Errorf("Stop 长度 = %d, 期望 2", len(req.Stop))
				}
			},
		},
		{
			name: "User 参数",
			config: map[string]any{
				"user": "test-user",
			},
			check: func(t *testing.T, req *ResponsesRequest) {
				if req.User != "test-user" {
					t.Errorf("User = %s, 期望 test-user", req.User)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ResponsesRequest{}
			applyConfig(req, tt.config)
			tt.check(t, req)
		})
	}
}

// TestConvertConfigToMap 测试配置转换
func TestConvertConfigToMap(t *testing.T) {
	tests := []struct {
		name    string
		config  any
		wantErr bool
	}{
		{
			name:    "nil 配置",
			config:  nil,
			wantErr: false,
		},
		{
			name: "map 配置",
			config: map[string]any{
				"temperature": 0.7,
			},
			wantErr: false,
		},
		{
			name: "结构体配置",
			config: struct {
				Temperature float64 `json:"temperature"`
			}{
				Temperature: 0.8,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertConfigToMap(tt.config)

			if tt.wantErr && err == nil {
				t.Error("期望错误但没有返回错误")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("不期望错误但返回了错误: %v", err)
			}

			if tt.config == nil && result != nil {
				t.Error("nil 配置应该返回 nil")
			}
		})
	}
}

// TestChainedCalls 测试链式调用
func TestChainedCalls(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("Hello"),
			},
		},
	}

	tools := []*ai.ToolDefinition{
		{
			Name:        "test_tool",
			Description: "测试工具",
		},
	}

	config := map[string]any{
		"temperature": 0.7,
	}

	// 测试链式调用
	result := gen.WithMessages(messages).WithTools(tools).WithConfig(config)

	if result != gen {
		t.Error("链式调用应该返回相同的 generator 实例")
	}

	if len(gen.messages) == 0 {
		t.Error("messages 应该被设置")
	}

	if len(gen.tools) == 0 {
		t.Error("tools 应该被设置")
	}

	if gen.config == nil {
		t.Error("config 应该被设置")
	}
}

// TestErrorPropagation 测试错误传播
func TestErrorPropagation(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	// 设置一个会导致错误的消息
	invalidMessages := []*ai.Message{
		{
			Role:    ai.RoleUser,
			Content: []*ai.Part{}, // 空内容会导致错误
		},
	}

	result := gen.WithMessages(invalidMessages)

	if result != gen {
		t.Error("即使有错误，WithMessages 也应该返回相同的 generator 实例")
	}

	if gen.err == nil {
		t.Error("应该设置错误")
	}

	// 后续调用应该被跳过
	gen.WithTools([]*ai.ToolDefinition{{Name: "test"}})
	gen.WithConfig(map[string]any{"temperature": 0.7})

	// 错误应该保持不变
	if gen.err == nil {
		t.Error("错误应该被保留")
	}
}

// TestMapFinishReason 测试 finish_reason 映射
func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   ai.FinishReason
	}{
		{
			name:   "stop",
			reason: "stop",
			want:   ai.FinishReasonStop,
		},
		{
			name:   "length",
			reason: "length",
			want:   ai.FinishReasonLength,
		},
		{
			name:   "content_filter",
			reason: "content_filter",
			want:   ai.FinishReasonBlocked,
		},
		{
			name:   "tool_calls",
			reason: "tool_calls",
			want:   ai.FinishReasonStop,
		},
		{
			name:   "unknown",
			reason: "unknown_reason",
			want:   ai.FinishReasonOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapFinishReason(tt.reason)
			if got != tt.want {
				t.Errorf("mapFinishReason(%s) = %v, 期望 %v", tt.reason, got, tt.want)
			}
		})
	}
}

// TestConvertResponseMessage_TextContent 测试文本内容转换
func TestConvertResponseMessage_TextContent(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	azMsg := &Message{
		Role:    "assistant",
		Content: "Hello, how can I help you?",
	}

	msg, err := gen.convertResponseMessage(azMsg)
	if err != nil {
		t.Fatalf("convertResponseMessage() 错误: %v", err)
	}

	if msg.Role != ai.RoleModel {
		t.Errorf("Role = %v, 期望 %v", msg.Role, ai.RoleModel)
	}

	if len(msg.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(msg.Content))
	}

	if !msg.Content[0].IsText() {
		t.Error("Content[0] 应该是文本类型")
	}

	if msg.Content[0].Text != "Hello, how can I help you?" {
		t.Errorf("Text = %s, 期望 'Hello, how can I help you?'", msg.Content[0].Text)
	}
}

// TestConvertResponseMessage_ToolCalls 测试工具调用转换
func TestConvertResponseMessage_ToolCalls(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	azMsg := &Message{
		Role: "assistant",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "get_weather",
					Arguments: `{"location":"San Francisco"}`,
				},
			},
		},
	}

	msg, err := gen.convertResponseMessage(azMsg)
	if err != nil {
		t.Fatalf("convertResponseMessage() 错误: %v", err)
	}

	if len(msg.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(msg.Content))
	}

	if !msg.Content[0].IsToolRequest() {
		t.Error("Content[0] 应该是工具请求类型")
	}

	toolReq := msg.Content[0].ToolRequest
	if toolReq.Ref != "call_123" {
		t.Errorf("Ref = %s, 期望 'call_123'", toolReq.Ref)
	}

	if toolReq.Name != "get_weather" {
		t.Errorf("Name = %s, 期望 'get_weather'", toolReq.Name)
	}

	inputMap, ok := toolReq.Input.(map[string]any)
	if !ok {
		t.Fatalf("Input 应该是 map[string]any 类型")
	}

	if location, ok := inputMap["location"].(string); !ok || location != "San Francisco" {
		t.Errorf("Input location = %v, 期望 'San Francisco'", inputMap["location"])
	}
}

// TestConvertResponseMessage_InvalidJSON 测试无效 JSON 参数
func TestConvertResponseMessage_InvalidJSON(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	azMsg := &Message{
		Role: "assistant",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "get_weather",
					Arguments: `{invalid json}`,
				},
			},
		},
	}

	_, err := gen.convertResponseMessage(azMsg)
	if err == nil {
		t.Error("期望错误但没有返回错误")
	}
}

// TestConvertToModelResponse 测试完整响应转换
func TestConvertToModelResponse(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	azResp := &ResponsesResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		SystemFingerprint: "fp_123",
	}

	resp, err := gen.convertToModelResponse(azResp)
	if err != nil {
		t.Fatalf("convertToModelResponse() 错误: %v", err)
	}

	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	if resp.Usage == nil {
		t.Fatal("Usage 不应该为 nil")
	}

	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, 期望 10", resp.Usage.InputTokens)
	}

	if resp.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, 期望 5", resp.Usage.OutputTokens)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, 期望 15", resp.Usage.TotalTokens)
	}

	if resp.FinishReason != ai.FinishReasonStop {
		t.Errorf("FinishReason = %v, 期望 %v", resp.FinishReason, ai.FinishReasonStop)
	}

	if resp.Custom == nil {
		t.Fatal("Custom 不应该为 nil")
	}

	customMap, ok := resp.Custom.(map[string]any)
	if !ok {
		t.Fatal("Custom 应该是 map[string]any 类型")
	}

	if customMap["id"] != "chatcmpl-123" {
		t.Errorf("Custom id = %v, 期望 'chatcmpl-123'", customMap["id"])
	}

	if customMap["model"] != "gpt-4" {
		t.Errorf("Custom model = %v, 期望 'gpt-4'", customMap["model"])
	}

	if customMap["system_fingerprint"] != "fp_123" {
		t.Errorf("Custom system_fingerprint = %v, 期望 'fp_123'", customMap["system_fingerprint"])
	}
}

// TestConvertToModelResponse_NoChoices 测试没有选项的响应
func TestConvertToModelResponse_NoChoices(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	azResp := &ResponsesResponse{
		ID:      "chatcmpl-123",
		Choices: []Choice{},
	}

	_, err := gen.convertToModelResponse(azResp)
	if err == nil {
		t.Error("期望错误但没有返回错误")
	}
}

// TestSetRequestHeaders 测试请求头设置
func TestSetRequestHeaders(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "test-api-key", "2025-04-01-preview", "gpt-4")

	req, _ := http.NewRequest("POST", "https://test.openai.azure.com/openai/responses", nil)
	gen.setRequestHeaders(req)

	// 检查 api-key 头
	if apiKey := req.Header.Get("api-key"); apiKey != "test-api-key" {
		t.Errorf("api-key header = %s, 期望 'test-api-key'", apiKey)
	}

	// 检查 Content-Type 头
	if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type header = %s, 期望 'application/json'", contentType)
	}

	// 检查 User-Agent 头
	if userAgent := req.Header.Get("User-Agent"); userAgent != "genkit-azure-plugin/1.0" {
		t.Errorf("User-Agent header = %s, 期望 'genkit-azure-plugin/1.0'", userAgent)
	}
}

// TestHandleErrorResponse 测试错误响应处理
func TestHandleErrorResponse(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantCode   string
	}{
		{
			name:       "结构化错误响应",
			statusCode: 400,
			body:       []byte(`{"error":{"message":"Invalid request","type":"invalid_request_error","code":"invalid_request"}}`),
			wantCode:   "400",
		},
		{
			name:       "非结构化错误响应",
			statusCode: 500,
			body:       []byte(`Internal Server Error`),
			wantCode:   "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.handleErrorResponse(tt.statusCode, tt.body)
			if err == nil {
				t.Fatal("期望错误但没有返回错误")
			}

			azErr, ok := err.(*AzureAIError)
			if !ok {
				t.Fatalf("错误类型 = %T, 期望 *AzureAIError", err)
			}

			if azErr.Code != tt.wantCode {
				t.Errorf("错误代码 = %s, 期望 %s", azErr.Code, tt.wantCode)
			}

			if azErr.Type != "api" {
				t.Errorf("错误类型 = %s, 期望 'api'", azErr.Type)
			}
		})
	}
}

// TestSSEScanner 测试 SSE 扫描器
func TestSSEScanner(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "单行数据",
			input:    "data: {\"test\":\"value\"}\n",
			expected: []string{"data: {\"test\":\"value\"}"},
		},
		{
			name:     "多行数据",
			input:    "data: line1\ndata: line2\ndata: line3\n",
			expected: []string{"data: line1", "data: line2", "data: line3"},
		},
		{
			name:     "带空行",
			input:    "data: line1\n\ndata: line2\n",
			expected: []string{"data: line1", "", "data: line2"},
		},
		{
			name:     "带注释",
			input:    ": comment\ndata: line1\n",
			expected: []string{": comment", "data: line1"},
		},
		{
			name:     "结束标记",
			input:    "data: line1\ndata: [DONE]\n",
			expected: []string{"data: line1", "data: [DONE]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := newSSEScanner(bytes.NewReader([]byte(tt.input)))
			var lines []string

			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("扫描错误: %v", err)
			}

			if len(lines) != len(tt.expected) {
				t.Fatalf("行数 = %d, 期望 %d", len(lines), len(tt.expected))
			}

			for i, line := range lines {
				if line != tt.expected[i] {
					t.Errorf("行 %d = %q, 期望 %q", i, line, tt.expected[i])
				}
			}
		})
	}
}

// TestSSEScanner_EmptyInput 测试空输入
func TestSSEScanner_EmptyInput(t *testing.T) {
	scanner := newSSEScanner(bytes.NewReader([]byte("")))

	if scanner.Scan() {
		t.Error("空输入不应该有数据")
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("不应该有错误: %v", err)
	}
}

// TestSSEScanner_LargeInput 测试大输入
func TestSSEScanner_LargeInput(t *testing.T) {
	// 创建一个大的输入（超过缓冲区大小）
	var input bytes.Buffer
	for i := 0; i < 1000; i++ {
		input.WriteString(fmt.Sprintf("data: line %d\n", i))
	}

	scanner := newSSEScanner(&input)
	count := 0

	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("扫描错误: %v", err)
	}

	if count != 1000 {
		t.Errorf("行数 = %d, 期望 1000", count)
	}
}

// TestParseStreamingResponse_SimpleText 测试简单文本流式响应
func TestParseStreamingResponse_SimpleText(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	// 模拟流式响应数据
	streamData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}
data: [DONE]
`

	var chunks []*ai.ModelResponseChunk
	callback := func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		chunks = append(chunks, chunk)
		return nil
	}

	resp, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), callback)
	if err != nil {
		t.Fatalf("parseStreamingResponse() 错误: %v", err)
	}

	// 验证回调被调用了 3 次（3 个内容块）
	if len(chunks) != 3 {
		t.Errorf("回调调用次数 = %d, 期望 3", len(chunks))
	}

	// 验证最终响应
	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	if len(resp.Message.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(resp.Message.Content))
	}

	if !resp.Message.Content[0].IsText() {
		t.Error("Content[0] 应该是文本类型")
	}

	expectedText := "Hello world!"
	if resp.Message.Content[0].Text != expectedText {
		t.Errorf("Text = %q, 期望 %q", resp.Message.Content[0].Text, expectedText)
	}

	if resp.FinishReason != ai.FinishReasonStop {
		t.Errorf("FinishReason = %v, 期望 %v", resp.FinishReason, ai.FinishReasonStop)
	}

	// 验证元数据
	if resp.Custom == nil {
		t.Fatal("Custom 不应该为 nil")
	}

	customMap, ok := resp.Custom.(map[string]any)
	if !ok {
		t.Fatal("Custom 应该是 map[string]any 类型")
	}

	if customMap["id"] != "chatcmpl-123" {
		t.Errorf("Custom id = %v, 期望 'chatcmpl-123'", customMap["id"])
	}

	if customMap["model"] != "gpt-4" {
		t.Errorf("Custom model = %v, 期望 'gpt-4'", customMap["model"])
	}
}

// TestParseStreamingResponse_WithToolCalls 测试带工具调用的流式响应
func TestParseStreamingResponse_WithToolCalls(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	// 模拟带工具调用的流式响应
	streamData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"id":"call_123","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"San Francisco\"}"}}]},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}
data: [DONE]
`

	var chunks []*ai.ModelResponseChunk
	callback := func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		chunks = append(chunks, chunk)
		return nil
	}

	resp, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), callback)
	if err != nil {
		t.Fatalf("parseStreamingResponse() 错误: %v", err)
	}

	// 验证回调被调用了 1 次（1 个工具调用）
	if len(chunks) != 1 {
		t.Errorf("回调调用次数 = %d, 期望 1", len(chunks))
	}

	// 验证最终响应
	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	if len(resp.Message.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(resp.Message.Content))
	}

	if !resp.Message.Content[0].IsToolRequest() {
		t.Error("Content[0] 应该是工具请求类型")
	}

	toolReq := resp.Message.Content[0].ToolRequest
	if toolReq.Ref != "call_123" {
		t.Errorf("Ref = %s, 期望 'call_123'", toolReq.Ref)
	}

	if toolReq.Name != "get_weather" {
		t.Errorf("Name = %s, 期望 'get_weather'", toolReq.Name)
	}

	if resp.FinishReason != ai.FinishReasonStop {
		t.Errorf("FinishReason = %v, 期望 %v", resp.FinishReason, ai.FinishReasonStop)
	}
}

// TestParseStreamingResponse_CallbackError 测试回调错误
func TestParseStreamingResponse_CallbackError(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	streamData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}
data: [DONE]
`

	expectedErr := fmt.Errorf("callback error")
	callback := func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		return expectedErr
	}

	_, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), callback)
	if err == nil {
		t.Fatal("期望错误但没有返回错误")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("callback error")) {
		t.Errorf("错误消息 = %v, 应该包含 'callback error'", err)
	}
}

// TestParseStreamingResponse_InvalidJSON 测试无效 JSON
func TestParseStreamingResponse_InvalidJSON(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	streamData := `data: {invalid json}
data: [DONE]
`

	_, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), nil)
	if err == nil {
		t.Fatal("期望错误但没有返回错误")
	}

	azErr, ok := err.(*AzureAIError)
	if !ok {
		t.Fatalf("错误类型 = %T, 期望 *AzureAIError", err)
	}

	if azErr.Type != "parse" {
		t.Errorf("错误类型 = %s, 期望 'parse'", azErr.Type)
	}
}

// TestParseStreamingResponse_EmptyStream 测试空流
func TestParseStreamingResponse_EmptyStream(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	streamData := `data: [DONE]
`

	resp, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), nil)
	if err != nil {
		t.Fatalf("parseStreamingResponse() 错误: %v", err)
	}

	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	// 空流应该返回空内容
	if len(resp.Message.Content) != 0 {
		t.Errorf("Content 长度 = %d, 期望 0", len(resp.Message.Content))
	}
}

// TestParseStreamingResponse_NoCallback 测试没有回调
func TestParseStreamingResponse_NoCallback(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	streamData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":"stop"}]}
data: [DONE]
`

	// 不提供回调函数
	resp, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), nil)
	if err != nil {
		t.Fatalf("parseStreamingResponse() 错误: %v", err)
	}

	// 即使没有回调，也应该正确聚合响应
	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	if len(resp.Message.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(resp.Message.Content))
	}

	expectedText := "Hello world"
	if resp.Message.Content[0].Text != expectedText {
		t.Errorf("Text = %q, 期望 %q", resp.Message.Content[0].Text, expectedText)
	}
}

// TestParseStreamingResponse_MultipleChoices 测试多个选项（只使用第一个）
func TestParseStreamingResponse_MultipleChoices(t *testing.T) {
	gen := NewModelGenerator(&http.Client{}, "https://test.openai.azure.com", "key", "2025-04-01-preview", "gpt-4")

	// 模拟多个选项的响应（虽然实际上 Azure OpenAI 通常只返回一个）
	streamData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"First"},"finish_reason":null},{"index":1,"delta":{"role":"assistant","content":"Second"},"finish_reason":null}]}
data: [DONE]
`

	resp, err := gen.parseStreamingResponse(context.Background(), bytes.NewReader([]byte(streamData)), nil)
	if err != nil {
		t.Fatalf("parseStreamingResponse() 错误: %v", err)
	}

	// 应该只使用第一个选项
	if resp.Message == nil {
		t.Fatal("Message 不应该为 nil")
	}

	if len(resp.Message.Content) != 1 {
		t.Fatalf("Content 长度 = %d, 期望 1", len(resp.Message.Content))
	}

	// 应该是第一个选项的内容
	if resp.Message.Content[0].Text != "First" {
		t.Errorf("Text = %q, 期望 'First'", resp.Message.Content[0].Text)
	}
}
