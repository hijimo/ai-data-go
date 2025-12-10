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
	"encoding/json"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

// TestConvertMessages_SystemMessage 测试系统消息转换
func TestConvertMessages_SystemMessage(t *testing.T) {
	messages := []*ai.Message{
		{
			Role:    ai.RoleSystem,
			Content: []*ai.Part{ai.NewTextPart("You are a helpful assistant.")},
		},
		{
			Role:    ai.RoleUser,
			Content: []*ai.Part{ai.NewTextPart("Hello")},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// 验证系统消息在第一个位置
	if result[0].Role != "system" {
		t.Errorf("expected first message role to be 'system', got '%s'", result[0].Role)
	}

	if result[0].Content != "You are a helpful assistant." {
		t.Errorf("expected system message content to be 'You are a helpful assistant.', got '%s'", result[0].Content)
	}
}

// TestConvertMessages_UserTextOnly 测试纯文本用户消息
func TestConvertMessages_UserTextOnly(t *testing.T) {
	messages := []*ai.Message{
		{
			Role:    ai.RoleUser,
			Content: []*ai.Part{ai.NewTextPart("Hello, how are you?")},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	if result[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", result[0].Role)
	}

	// 单个文本应该是字符串格式
	if content, ok := result[0].Content.(string); !ok {
		t.Errorf("expected content to be string, got %T", result[0].Content)
	} else if content != "Hello, how are you?" {
		t.Errorf("expected content 'Hello, how are you?', got '%s'", content)
	}
}

// TestConvertMessages_UserMultimodal 测试多模态用户消息
func TestConvertMessages_UserMultimodal(t *testing.T) {
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("What's in this image?"),
				ai.NewMediaPart("image/jpeg", "https://example.com/image.jpg"),
			},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	// 多模态内容应该是 ContentPart 数组
	contentParts, ok := result[0].Content.([]ContentPart)
	if !ok {
		t.Fatalf("expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(contentParts))
	}

	// 验证文本部分
	if contentParts[0].Type != "text" {
		t.Errorf("expected first part type 'text', got '%s'", contentParts[0].Type)
	}
	if contentParts[0].Text != "What's in this image?" {
		t.Errorf("expected text 'What's in this image?', got '%s'", contentParts[0].Text)
	}

	// 验证图像部分
	if contentParts[1].Type != "image_url" {
		t.Errorf("expected second part type 'image_url', got '%s'", contentParts[1].Type)
	}
	if contentParts[1].ImageURL == nil {
		t.Fatal("expected ImageURL to be non-nil")
	}
	if contentParts[1].ImageURL.URL != "https://example.com/image.jpg" {
		t.Errorf("expected image URL 'https://example.com/image.jpg', got '%s'", contentParts[1].ImageURL.URL)
	}
}

// TestConvertMessages_AssistantWithToolCalls 测试包含工具调用的助手消息
func TestConvertMessages_AssistantWithToolCalls(t *testing.T) {
	toolInput := map[string]any{
		"location": "San Francisco",
	}

	messages := []*ai.Message{
		{
			Role: ai.RoleModel,
			Content: []*ai.Part{
				ai.NewTextPart("Let me check the weather for you."),
				ai.NewToolRequestPart(&ai.ToolRequest{
					Name:  "get_weather",
					Ref:   "call_123",
					Input: toolInput,
				}),
			},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	if result[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", result[0].Role)
	}

	// 验证文本内容
	if result[0].Content != "Let me check the weather for you." {
		t.Errorf("expected content 'Let me check the weather for you.', got '%v'", result[0].Content)
	}

	// 验证工具调用
	if len(result[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result[0].ToolCalls))
	}

	toolCall := result[0].ToolCalls[0]
	if toolCall.ID != "call_123" {
		t.Errorf("expected tool call ID 'call_123', got '%s'", toolCall.ID)
	}
	if toolCall.Type != "function" {
		t.Errorf("expected tool call type 'function', got '%s'", toolCall.Type)
	}
	if toolCall.Function.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", toolCall.Function.Name)
	}

	// 验证参数 JSON
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		t.Fatalf("failed to unmarshal arguments: %v", err)
	}
	if args["location"] != "San Francisco" {
		t.Errorf("expected location 'San Francisco', got '%v'", args["location"])
	}
}

// TestConvertMessages_ToolResponse 测试工具响应消息
func TestConvertMessages_ToolResponse(t *testing.T) {
	toolOutput := map[string]any{
		"temperature": 72,
		"condition":   "sunny",
	}

	messages := []*ai.Message{
		{
			Role: ai.RoleTool,
			Content: []*ai.Part{
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   "get_weather",
					Ref:    "call_123",
					Output: toolOutput,
				}),
			},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	if result[0].Role != "tool" {
		t.Errorf("expected role 'tool', got '%s'", result[0].Role)
	}

	if result[0].ToolCallID != "call_123" {
		t.Errorf("expected tool call ID 'call_123', got '%s'", result[0].ToolCallID)
	}

	// 验证内容是 JSON 字符串
	contentStr, ok := result[0].Content.(string)
	if !ok {
		t.Fatalf("expected content to be string, got %T", result[0].Content)
	}

	var output map[string]any
	if err := json.Unmarshal([]byte(contentStr), &output); err != nil {
		t.Fatalf("failed to unmarshal content: %v", err)
	}

	if output["temperature"].(float64) != 72 {
		t.Errorf("expected temperature 72, got %v", output["temperature"])
	}
	if output["condition"] != "sunny" {
		t.Errorf("expected condition 'sunny', got '%v'", output["condition"])
	}
}

// TestConvertMessages_MultipleToolResponses 测试多个工具响应
func TestConvertMessages_MultipleToolResponses(t *testing.T) {
	messages := []*ai.Message{
		{
			Role: ai.RoleTool,
			Content: []*ai.Part{
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   "tool1",
					Ref:    "call_1",
					Output: map[string]any{"result": "A"},
				}),
				ai.NewToolResponsePart(&ai.ToolResponse{
					Name:   "tool2",
					Ref:    "call_2",
					Output: map[string]any{"result": "B"},
				}),
			},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	// 每个工具响应应该生成一个独立的消息
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	if result[0].ToolCallID != "call_1" {
		t.Errorf("expected first tool call ID 'call_1', got '%s'", result[0].ToolCallID)
	}
	if result[1].ToolCallID != "call_2" {
		t.Errorf("expected second tool call ID 'call_2', got '%s'", result[1].ToolCallID)
	}
}

// TestConvertMessages_Base64Image 测试 base64 编码的图像
func TestConvertMessages_Base64Image(t *testing.T) {
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewMediaPart("image/png", "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="),
			},
		},
	}

	result, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages failed: %v", err)
	}

	contentParts, ok := result[0].Content.([]ContentPart)
	if !ok {
		t.Fatalf("expected content to be []ContentPart, got %T", result[0].Content)
	}

	if len(contentParts) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(contentParts))
	}

	if contentParts[0].Type != "image_url" {
		t.Errorf("expected type 'image_url', got '%s'", contentParts[0].Type)
	}

	if contentParts[0].ImageURL == nil {
		t.Fatal("expected ImageURL to be non-nil")
	}

	if !containsString(contentParts[0].ImageURL.URL, "data:image/png;base64") {
		t.Errorf("expected base64 data URL, got '%s'", contentParts[0].ImageURL.URL)
	}
}

// TestConvertTools 测试工具定义转换
func TestConvertTools(t *testing.T) {
	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			Name:        "calculate",
			Description: "Perform a calculation",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type": "string",
					},
				},
			},
		},
	}

	result := convertTools(tools)

	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}

	// 验证第一个工具
	if result[0].Type != "function" {
		t.Errorf("expected type 'function', got '%s'", result[0].Type)
	}
	if result[0].Function.Name != "get_weather" {
		t.Errorf("expected name 'get_weather', got '%s'", result[0].Function.Name)
	}
	if result[0].Function.Description != "Get the current weather" {
		t.Errorf("expected description 'Get the current weather', got '%s'", result[0].Function.Description)
	}
	if result[0].Function.Parameters == nil {
		t.Error("expected parameters to be non-nil")
	}

	// 验证第二个工具
	if result[1].Function.Name != "calculate" {
		t.Errorf("expected name 'calculate', got '%s'", result[1].Function.Name)
	}
}

// TestConvertTools_Empty 测试空工具列表
func TestConvertTools_Empty(t *testing.T) {
	result := convertTools(nil)
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}

	result = convertTools([]*ai.ToolDefinition{})
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}
}

// TestConvertMessages_EmptyMessages 测试空消息列表
func TestConvertMessages_EmptyMessages(t *testing.T) {
	_, err := convertMessages(nil)
	if err == nil {
		t.Error("expected error for nil messages")
	}

	_, err = convertMessages([]*ai.Message{})
	if err == nil {
		t.Error("expected error for empty messages")
	}
}

// TestConvertMessages_InvalidImageURL 测试无效的图像 URL
func TestConvertMessages_InvalidImageURL(t *testing.T) {
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewMediaPart("image/jpeg", "invalid-url"),
			},
		},
	}

	_, err := convertMessages(messages)
	if err == nil {
		t.Error("expected error for invalid image URL")
	}
}

// 辅助函数
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
