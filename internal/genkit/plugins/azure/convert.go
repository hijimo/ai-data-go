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
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
)

// convertMessages 将 Genkit 消息转换为 Azure OpenAI 消息格式
// 系统消息会被放置在 input 数组的开头
func convertMessages(messages []*ai.Message) ([]Message, error) {
	if len(messages) == 0 {
		return nil, NewRequestError("至少需要一条消息", nil)
	}

	var result []Message
	var systemMessages []Message
	var otherMessages []Message

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		switch msg.Role {
		case ai.RoleSystem:
			// 系统消息：提取文本内容
			text := extractText(msg.Content)
			if text != "" {
				systemMessages = append(systemMessages, Message{
					Role:    "system",
					Content: text,
				})
			}

		case ai.RoleUser:
			// 用户消息：处理文本和多模态内容
			azMsg, err := convertUserMessage(msg)
			if err != nil {
				return nil, err
			}
			otherMessages = append(otherMessages, azMsg)

		case ai.RoleModel:
			// 助手消息：处理文本内容和工具调用
			azMsg, err := convertAssistantMessage(msg)
			if err != nil {
				return nil, err
			}
			otherMessages = append(otherMessages, azMsg)

		case ai.RoleTool:
			// 工具响应消息：转换为 tool 角色消息
			toolMsgs, err := convertToolResponses(msg)
			if err != nil {
				return nil, err
			}
			otherMessages = append(otherMessages, toolMsgs...)

		default:
			return nil, NewRequestError(fmt.Sprintf("不支持的消息角色: %s", msg.Role), nil)
		}
	}

	// 系统消息必须放在开头
	result = append(result, systemMessages...)
	result = append(result, otherMessages...)

	return result, nil
}

// extractText 从 Part 数组中提取所有文本内容
func extractText(parts []*ai.Part) string {
	var texts []string
	for _, part := range parts {
		if part.IsText() || part.IsData() {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// convertUserMessage 转换用户消息
// 如果只有文本，返回字符串；如果有多模态内容，返回 ContentPart 数组
func convertUserMessage(msg *ai.Message) (Message, error) {
	if len(msg.Content) == 0 {
		return Message{}, NewRequestError("用户消息不能为空", nil)
	}

	// 检查是否只有纯文本
	hasOnlyText := true
	for _, part := range msg.Content {
		if !part.IsText() && !part.IsData() {
			hasOnlyText = false
			break
		}
	}

	// 如果只有文本，返回字符串格式
	if hasOnlyText && len(msg.Content) == 1 {
		return Message{
			Role:    "user",
			Content: msg.Content[0].Text,
		}, nil
	}

	// 否则构建 ContentPart 数组
	var contentParts []ContentPart
	for _, part := range msg.Content {
		if part.IsText() || part.IsData() {
			contentParts = append(contentParts, ContentPart{
				Type: "text",
				Text: part.Text,
			})
		} else if part.IsMedia() {
			// 处理图像
			if part.IsImage() {
				imageURL, err := convertImagePart(part)
				if err != nil {
					return Message{}, err
				}
				contentParts = append(contentParts, ContentPart{
					Type:     "image_url",
					ImageURL: imageURL,
				})
			}
			// 其他媒体类型暂不支持，跳过
		}
	}

	if len(contentParts) == 0 {
		return Message{}, NewRequestError("用户消息必须包含至少一个文本或图像内容", nil)
	}

	return Message{
		Role:    "user",
		Content: contentParts,
	}, nil
}

// convertImagePart 转换图像 Part 为 ImageURL
func convertImagePart(part *ai.Part) (*ImageURL, error) {
	if !part.IsImage() {
		return nil, NewRequestError("Part 不是图像类型", nil)
	}

	// part.Text 包含 URL 或 base64 编码的图像
	url := part.Text

	// 验证 URL 格式
	if url == "" {
		return nil, NewRequestError("图像 URL 不能为空", nil)
	}

	// 支持 http/https URL 和 data: URL (base64)
	if !strings.HasPrefix(url, "http://") &&
		!strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "data:image/") {
		return nil, NewRequestError("图像 URL 必须是 http/https URL 或 base64 data URL", nil)
	}

	return &ImageURL{
		URL: url,
	}, nil
}

// convertAssistantMessage 转换助手消息
func convertAssistantMessage(msg *ai.Message) (Message, error) {
	azMsg := Message{
		Role: "assistant",
	}

	// 提取文本内容
	text := extractText(msg.Content)
	if text != "" {
		azMsg.Content = text
	}

	// 提取工具调用
	toolCalls := extractToolCalls(msg.Content)
	if len(toolCalls) > 0 {
		azMsg.ToolCalls = toolCalls
	}

	// 助手消息必须包含内容或工具调用
	if azMsg.Content == nil && len(azMsg.ToolCalls) == 0 {
		return Message{}, NewRequestError("助手消息必须包含文本内容或工具调用", nil)
	}

	return azMsg, nil
}

// extractToolCalls 从 Part 数组中提取工具调用
func extractToolCalls(parts []*ai.Part) []ToolCall {
	var toolCalls []ToolCall

	for _, part := range parts {
		if !part.IsToolRequest() {
			continue
		}

		req := part.ToolRequest
		if req == nil {
			continue
		}

		// 序列化参数为 JSON 字符串
		argsJSON, err := json.Marshal(req.Input)
		if err != nil {
			// 如果序列化失败，使用空对象
			argsJSON = []byte("{}")
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:   req.Ref,
			Type: "function",
			Function: FunctionCall{
				Name:      req.Name,
				Arguments: string(argsJSON),
			},
		})
	}

	return toolCalls
}

// convertToolResponses 转换工具响应消息
// 每个工具响应都会生成一个独立的 tool 角色消息
func convertToolResponses(msg *ai.Message) ([]Message, error) {
	var messages []Message

	for _, part := range msg.Content {
		if !part.IsToolResponse() {
			continue
		}

		resp := part.ToolResponse
		if resp == nil {
			continue
		}

		// 序列化输出为 JSON 字符串
		var content string
		if resp.Output != nil {
			outputJSON, err := json.Marshal(resp.Output)
			if err != nil {
				return nil, NewRequestError(fmt.Sprintf("序列化工具输出失败: %v", err), err)
			}
			content = string(outputJSON)
		} else {
			content = "{}"
		}

		messages = append(messages, Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: resp.Ref,
		})
	}

	if len(messages) == 0 {
		return nil, NewRequestError("工具消息必须包含至少一个工具响应", nil)
	}

	return messages, nil
}

// convertTools 将 Genkit 工具定义转换为 Azure OpenAI 工具格式
func convertTools(tools []*ai.ToolDefinition) []Tool {
	if len(tools) == 0 {
		return nil
	}

	var result []Tool
	for _, tool := range tools {
		if tool == nil {
			continue
		}

		result = append(result, Tool{
			Type: "function",
			Function: FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return result
}
