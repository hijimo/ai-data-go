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
