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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: azure-ai-genkit-provider, Property 1: API 版本参数正确性**
// 属性 1: API 版本参数正确性
// 对于任何指定了 API 版本的请求，构建的 URL 应该包含正确的 api-version 查询参数
// 验证需求: 1.4
func TestProperty_APIVersionParameterCorrectness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("所有请求都包含正确的 api-version 查询参数",
		prop.ForAll(
			func(apiVersion string, modelName string) bool {
				// 跳过空字符串，因为空字符串会使用默认版本
				if apiVersion == "" {
					return true
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试 URL 构建
					"https://test.openai.azure.com",
					"test-api-key",
					apiVersion,
					modelName,
				)

				// 构建请求 URL
				url := generator.buildRequestURL()

				// 验证 URL 包含正确的 api-version 参数
				expectedParam := "api-version=" + apiVersion
				if !strings.Contains(url, expectedParam) {
					t.Logf("URL 不包含预期的 api-version 参数")
					t.Logf("预期参数: %s", expectedParam)
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证 URL 使用 Responses API 端点
				if !strings.Contains(url, "/openai/responses") {
					t.Logf("URL 不包含 /openai/responses 端点")
					t.Logf("实际 URL: %s", url)
					return false
				}

				return true
			},
			// 生成有效的 API 版本字符串
			gen.OneConstOf(
				"2025-04-01-preview",
				"2024-12-01-preview",
				"2024-10-01-preview",
				"2024-08-01-preview",
				"2024-06-01",
				"2024-05-01-preview",
			),
			// 生成模型名称
			gen.OneConstOf(
				"gpt-4",
				"gpt-4-turbo",
				"gpt-35-turbo",
				"gpt-4o",
				"gpt-4o-mini",
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 2: 默认 API 版本**
// 属性 2: 默认 API 版本
// 对于任何未指定 API 版本的请求，构建的 URL 应该包含默认的 api-version=2025-04-01-preview
// 验证需求: 1.5
func TestProperty_DefaultAPIVersion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("未指定 API 版本时使用默认版本",
		prop.ForAll(
			func(modelName string) bool {
				// 创建 AzureAI 插件实例，不指定 API 版本
				plugin := &AzureAI{
					APIKey:   "test-api-key",
					BaseURL:  "https://test.openai.azure.com",
					Provider: "azure",
					// APIVersion 留空，应该使用默认值
				}

				// 初始化插件（这会设置默认 API 版本）
				ctx := context.Background()
				plugin.Init(ctx)

				// 验证插件的 APIVersion 字段被设置为默认值
				if plugin.APIVersion != DefaultAPIVersion {
					t.Logf("插件的 APIVersion 不是默认值")
					t.Logf("预期: %s", DefaultAPIVersion)
					t.Logf("实际: %s", plugin.APIVersion)
					return false
				}

				// 验证默认版本是 2025-04-01-preview
				if plugin.APIVersion != "2025-04-01-preview" {
					t.Logf("默认 API 版本不是 2025-04-01-preview")
					t.Logf("实际: %s", plugin.APIVersion)
					return false
				}

				// 创建 ModelGenerator 使用插件的 API 版本
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试 URL 构建
					plugin.BaseURL,
					plugin.APIKey,
					plugin.APIVersion,
					modelName,
				)

				// 构建请求 URL
				url := generator.buildRequestURL()

				// 验证 URL 包含默认的 api-version 参数
				expectedParam := "api-version=" + DefaultAPIVersion
				if !strings.Contains(url, expectedParam) {
					t.Logf("URL 不包含默认的 api-version 参数")
					t.Logf("预期参数: %s", expectedParam)
					t.Logf("实际 URL: %s", url)
					return false
				}

				return true
			},
			// 生成模型名称
			gen.OneConstOf(
				"gpt-4",
				"gpt-4-turbo",
				"gpt-35-turbo",
				"gpt-4o",
				"gpt-4o-mini",
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 6: 系统消息位置**
// 属性 6: 系统消息位置
// 对于任何包含系统消息的消息列表，转换后的 input 数组中系统消息应该位于第一个位置
// 验证需求: 2.5, 5.2
func TestProperty_SystemMessagePosition(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("系统消息始终位于 input 数组的开头",
		prop.ForAll(
			func(systemText string, userTexts []string) bool {
				// 构建包含系统消息和用户消息的消息列表
				var messages []*ai.Message

				// 添加系统消息
				messages = append(messages, &ai.Message{
					Role:    ai.RoleSystem,
					Content: []*ai.Part{ai.NewTextPart(systemText)},
				})

				// 添加用户消息
				for _, userText := range userTexts {
					messages = append(messages, &ai.Message{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart(userText)},
					})
				}

				// 转换消息
				azMessages, err := convertMessages(messages)
				if err != nil {
					t.Logf("转换消息失败: %v", err)
					return false
				}

				// 验证至少有一条消息
				if len(azMessages) == 0 {
					t.Logf("转换后的消息列表为空")
					return false
				}

				// 验证第一条消息是系统消息
				if azMessages[0].Role != "system" {
					t.Logf("第一条消息不是系统消息")
					t.Logf("预期角色: system")
					t.Logf("实际角色: %s", azMessages[0].Role)
					return false
				}

				// 验证系统消息的内容
				if azMessages[0].Content != systemText {
					t.Logf("系统消息内容不匹配")
					t.Logf("预期内容: %s", systemText)
					t.Logf("实际内容: %v", azMessages[0].Content)
					return false
				}

				// 验证后续消息不是系统消息
				for i := 1; i < len(azMessages); i++ {
					if azMessages[i].Role == "system" {
						t.Logf("在位置 %d 发现额外的系统消息", i)
						return false
					}
				}

				return true
			},
			// 生成系统消息文本（非空字符串）
			gen.Identifier(),
			// 生成用户消息文本数组（1-3条消息）
			gen.SliceOfN(
				3,
				gen.Identifier(),
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 15: 助手消息工具调用**
// 属性 15: 助手消息工具调用
// 对于任何包含工具调用的助手消息，转换后应该包含 tool_calls 数组
// 验证需求: 5.3
func TestProperty_AssistantMessageToolCalls(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("助手消息转换后包含正确的工具调用",
		prop.ForAll(
			func(textContent string, toolCallCount int) bool {
				// 限制工具调用数量以避免过大的测试用例
				if toolCallCount < 0 || toolCallCount > 5 {
					return true // 跳过无效输入
				}

				// 至少需要一个内容（文本或工具调用）
				if textContent == "" && toolCallCount == 0 {
					return true // 跳过空消息
				}

				// 构建助手消息
				var parts []*ai.Part

				// 添加文本内容（如果有）
				if textContent != "" {
					parts = append(parts, ai.NewTextPart(textContent))
				}

				// 添加工具调用
				for i := 0; i < toolCallCount; i++ {
					toolRequest := &ai.ToolRequest{
						Ref:  fmt.Sprintf("call_%d", i),
						Name: fmt.Sprintf("tool_%d", i),
						Input: map[string]any{
							"param1": fmt.Sprintf("value_%d", i),
							"param2": i,
						},
					}
					parts = append(parts, ai.NewToolRequestPart(toolRequest))
				}

				assistantMessage := &ai.Message{
					Role:    ai.RoleModel,
					Content: parts,
				}

				// 转换消息
				azMsg, err := convertAssistantMessage(assistantMessage)
				if err != nil {
					t.Logf("转换助手消息失败: %v", err)
					return false
				}

				// 验证角色
				if azMsg.Role != "assistant" {
					t.Logf("消息角色不正确")
					t.Logf("预期: assistant")
					t.Logf("实际: %s", azMsg.Role)
					return false
				}

				// 验证文本内容
				if textContent != "" {
					if azMsg.Content == nil {
						t.Logf("助手消息应该包含文本内容")
						return false
					}
					contentStr, ok := azMsg.Content.(string)
					if !ok {
						t.Logf("助手消息内容应该是字符串类型")
						t.Logf("实际类型: %T", azMsg.Content)
						return false
					}
					if contentStr != textContent {
						t.Logf("文本内容不匹配")
						t.Logf("预期: %s", textContent)
						t.Logf("实际: %s", contentStr)
						return false
					}
				}

				// 验证工具调用数量
				if len(azMsg.ToolCalls) != toolCallCount {
					t.Logf("工具调用数量不匹配")
					t.Logf("预期: %d", toolCallCount)
					t.Logf("实际: %d", len(azMsg.ToolCalls))
					return false
				}

				// 验证每个工具调用的结构
				for i, toolCall := range azMsg.ToolCalls {
					// 验证 ID
					expectedID := fmt.Sprintf("call_%d", i)
					if toolCall.ID != expectedID {
						t.Logf("工具调用 %d 的 ID 不匹配", i)
						t.Logf("预期: %s", expectedID)
						t.Logf("实际: %s", toolCall.ID)
						return false
					}

					// 验证类型
					if toolCall.Type != "function" {
						t.Logf("工具调用 %d 的类型不正确", i)
						t.Logf("预期: function")
						t.Logf("实际: %s", toolCall.Type)
						return false
					}

					// 验证函数名称
					expectedName := fmt.Sprintf("tool_%d", i)
					if toolCall.Function.Name != expectedName {
						t.Logf("工具调用 %d 的函数名称不匹配", i)
						t.Logf("预期: %s", expectedName)
						t.Logf("实际: %s", toolCall.Function.Name)
						return false
					}

					// 验证参数是有效的 JSON
					if toolCall.Function.Arguments == "" {
						t.Logf("工具调用 %d 的参数为空", i)
						return false
					}

					// 尝试解析参数 JSON
					var args map[string]any
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						t.Logf("工具调用 %d 的参数不是有效的 JSON: %v", i, err)
						return false
					}

					// 验证参数内容
					if param1, ok := args["param1"].(string); !ok || param1 != fmt.Sprintf("value_%d", i) {
						t.Logf("工具调用 %d 的 param1 参数不匹配", i)
						return false
					}

					// param2 可能是 float64（JSON 数字默认类型）
					if param2, ok := args["param2"].(float64); !ok || int(param2) != i {
						t.Logf("工具调用 %d 的 param2 参数不匹配", i)
						return false
					}
				}

				return true
			},
			// 生成文本内容（可能为空）
			gen.OneConstOf("", "这是助手的回复", "Let me help you with that", "好的，我来调用工具"),
			// 生成工具调用数量（0-5）
			gen.IntRange(0, 5),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 16: 工具响应 ID 保留**
// 属性 16: 工具响应 ID 保留
// 对于任何工具响应，转换后的 tool_call_id 应该与原始工具调用的 ID 相同
// 验证需求: 5.4, 6.3
func TestProperty_ToolResponseIDPreservation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("工具响应转换后保留原始工具调用 ID",
		prop.ForAll(
			func(toolResponseCount int) bool {
				// 限制工具响应数量以避免过大的测试用例
				if toolResponseCount < 1 || toolResponseCount > 5 {
					return true // 跳过无效输入
				}

				// 构建工具响应消息
				var parts []*ai.Part

				// 存储生成的 ID 用于验证
				expectedIDs := make([]string, toolResponseCount)

				// 添加工具响应
				for i := 0; i < toolResponseCount; i++ {
					toolID := fmt.Sprintf("call_%d_%d", i, i*100)
					expectedIDs[i] = toolID

					toolResponse := &ai.ToolResponse{
						Ref:  toolID,
						Name: fmt.Sprintf("tool_%d", i),
						Output: map[string]any{
							"result":  fmt.Sprintf("result_%d", i),
							"status":  "success",
							"value":   i * 10,
							"message": fmt.Sprintf("工具 %d 执行成功", i),
						},
					}
					parts = append(parts, ai.NewToolResponsePart(toolResponse))
				}

				toolMessage := &ai.Message{
					Role:    ai.RoleTool,
					Content: parts,
				}

				// 转换工具响应消息
				azMsgs, err := convertToolResponses(toolMessage)
				if err != nil {
					t.Logf("转换工具响应失败: %v", err)
					return false
				}

				// 验证消息数量
				if len(azMsgs) != toolResponseCount {
					t.Logf("工具响应消息数量不匹配")
					t.Logf("预期: %d", toolResponseCount)
					t.Logf("实际: %d", len(azMsgs))
					return false
				}

				// 验证每个工具响应消息
				for i, azMsg := range azMsgs {
					// 验证角色
					if azMsg.Role != "tool" {
						t.Logf("工具响应 %d 的角色不正确", i)
						t.Logf("预期: tool")
						t.Logf("实际: %s", azMsg.Role)
						return false
					}

					// 验证 ToolCallID 是否保留
					if azMsg.ToolCallID != expectedIDs[i] {
						t.Logf("工具响应 %d 的 ToolCallID 不匹配", i)
						t.Logf("预期: %s", expectedIDs[i])
						t.Logf("实际: %s", azMsg.ToolCallID)
						return false
					}

					// 验证内容不为空
					if azMsg.Content == nil {
						t.Logf("工具响应 %d 的内容为空", i)
						return false
					}

					// 验证内容是字符串类型
					contentStr, ok := azMsg.Content.(string)
					if !ok {
						t.Logf("工具响应 %d 的内容应该是字符串类型", i)
						t.Logf("实际类型: %T", azMsg.Content)
						return false
					}

					// 验证内容是有效的 JSON
					var output map[string]any
					if err := json.Unmarshal([]byte(contentStr), &output); err != nil {
						t.Logf("工具响应 %d 的内容不是有效的 JSON: %v", i, err)
						return false
					}

					// 验证输出内容
					if result, ok := output["result"].(string); !ok || result != fmt.Sprintf("result_%d", i) {
						t.Logf("工具响应 %d 的 result 字段不匹配", i)
						return false
					}

					if status, ok := output["status"].(string); !ok || status != "success" {
						t.Logf("工具响应 %d 的 status 字段不匹配", i)
						return false
					}

					// value 可能是 float64（JSON 数字默认类型）
					if value, ok := output["value"].(float64); !ok || int(value) != i*10 {
						t.Logf("工具响应 %d 的 value 字段不匹配", i)
						return false
					}
				}

				return true
			},
			// 生成工具响应数量（1-5）
			gen.IntRange(1, 5),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 14: 用户消息内容转换**
// 属性 14: 用户消息内容转换
// 对于任何包含文本和媒体的用户消息，转换后应该包含对应数量的 content parts
// 验证需求: 5.1
func TestProperty_UserMessageContentConversion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("用户消息转换后包含正确数量的 content parts",
		prop.ForAll(
			func(textCount int, imageCount int) bool {
				// 限制范围以避免过大的测试用例
				if textCount < 0 || textCount > 5 {
					return true // 跳过无效输入
				}
				if imageCount < 0 || imageCount > 3 {
					return true // 跳过无效输入
				}

				// 至少需要一个内容部分
				if textCount == 0 && imageCount == 0 {
					return true // 跳过空消息
				}

				// 构建用户消息
				var parts []*ai.Part

				// 添加文本部分
				for i := 0; i < textCount; i++ {
					parts = append(parts, ai.NewTextPart(fmt.Sprintf("文本内容 %d", i)))
				}

				// 添加图像部分
				for i := 0; i < imageCount; i++ {
					// 使用有效的图像 URL
					imageURL := fmt.Sprintf("https://example.com/image%d.jpg", i)
					parts = append(parts, ai.NewMediaPart("image/jpeg", imageURL))
				}

				userMessage := &ai.Message{
					Role:    ai.RoleUser,
					Content: parts,
				}

				// 转换消息
				azMsg, err := convertUserMessage(userMessage)
				if err != nil {
					t.Logf("转换用户消息失败: %v", err)
					return false
				}

				// 验证角色
				if azMsg.Role != "user" {
					t.Logf("消息角色不正确")
					t.Logf("预期: user")
					t.Logf("实际: %s", azMsg.Role)
					return false
				}

				// 计算预期的 content parts 数量
				expectedPartsCount := textCount + imageCount

				// 如果只有一个文本部分，内容应该是字符串
				if textCount == 1 && imageCount == 0 {
					if _, ok := azMsg.Content.(string); !ok {
						t.Logf("单个文本消息应该是字符串格式")
						t.Logf("实际类型: %T", azMsg.Content)
						return false
					}
					return true
				}

				// 否则内容应该是 ContentPart 数组
				contentParts, ok := azMsg.Content.([]ContentPart)
				if !ok {
					t.Logf("多部分消息内容应该是 []ContentPart 类型")
					t.Logf("实际类型: %T", azMsg.Content)
					return false
				}

				// 验证 content parts 数量
				if len(contentParts) != expectedPartsCount {
					t.Logf("content parts 数量不匹配")
					t.Logf("预期: %d (文本: %d, 图像: %d)", expectedPartsCount, textCount, imageCount)
					t.Logf("实际: %d", len(contentParts))
					return false
				}

				// 验证每个 content part 的类型
				textPartsFound := 0
				imagePartsFound := 0

				for _, part := range contentParts {
					switch part.Type {
					case "text":
						textPartsFound++
						if part.Text == "" {
							t.Logf("文本 part 的内容为空")
							return false
						}
					case "image_url":
						imagePartsFound++
						if part.ImageURL == nil || part.ImageURL.URL == "" {
							t.Logf("图像 part 的 URL 为空")
							return false
						}
					default:
						t.Logf("未知的 content part 类型: %s", part.Type)
						return false
					}
				}

				// 验证文本和图像部分的数量
				if textPartsFound != textCount {
					t.Logf("文本 parts 数量不匹配")
					t.Logf("预期: %d", textCount)
					t.Logf("实际: %d", textPartsFound)
					return false
				}

				if imagePartsFound != imageCount {
					t.Logf("图像 parts 数量不匹配")
					t.Logf("预期: %d", imageCount)
					t.Logf("实际: %d", imagePartsFound)
					return false
				}

				return true
			},
			// 生成文本部分数量（0-5）
			gen.IntRange(0, 5),
			// 生成图像部分数量（0-3）
			gen.IntRange(0, 3),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 4: 多模态内容转换**
// 属性 4: 多模态内容转换
// 对于任何包含文本和图像的消息，转换后的 Azure OpenAI 格式应该包含对应的 text 和 image_url 内容部分
// 验证需求: 2.3
func TestProperty_MultimodalContentConversion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("包含文本和图像的消息转换后包含对应的 content parts",
		prop.ForAll(
			func(textCount int, imageCount int) bool {
				// 限制范围以避免过大的测试用例
				if textCount < 0 || textCount > 5 {
					return true // 跳过无效输入
				}
				if imageCount < 0 || imageCount > 3 {
					return true // 跳过无效输入
				}

				// 必须同时包含文本和图像（这是多模态的定义）
				if textCount == 0 || imageCount == 0 {
					return true // 跳过非多模态消息
				}

				// 构建包含文本和图像的用户消息
				var parts []*ai.Part

				// 添加文本部分
				for i := 0; i < textCount; i++ {
					parts = append(parts, ai.NewTextPart(fmt.Sprintf("文本内容 %d", i)))
				}

				// 添加图像部分（使用不同格式）
				for i := 0; i < imageCount; i++ {
					var imageURL string
					switch i % 3 {
					case 0:
						// HTTP URL
						imageURL = fmt.Sprintf("https://example.com/image%d.jpg", i)
					case 1:
						// HTTPS URL
						imageURL = fmt.Sprintf("https://example.com/image%d.png", i)
					case 2:
						// Base64 data URL
						imageURL = fmt.Sprintf("data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA%d", i)
					}
					parts = append(parts, ai.NewMediaPart("image/jpeg", imageURL))
				}

				userMessage := &ai.Message{
					Role:    ai.RoleUser,
					Content: parts,
				}

				// 转换消息
				azMsg, err := convertUserMessage(userMessage)
				if err != nil {
					t.Logf("转换用户消息失败: %v", err)
					return false
				}

				// 验证角色
				if azMsg.Role != "user" {
					t.Logf("消息角色不正确")
					t.Logf("预期: user")
					t.Logf("实际: %s", azMsg.Role)
					return false
				}

				// 多模态消息的内容应该是 ContentPart 数组
				contentParts, ok := azMsg.Content.([]ContentPart)
				if !ok {
					t.Logf("多模态消息内容应该是 []ContentPart 类型")
					t.Logf("实际类型: %T", azMsg.Content)
					return false
				}

				// 验证 content parts 总数
				expectedTotalParts := textCount + imageCount
				if len(contentParts) != expectedTotalParts {
					t.Logf("content parts 总数不匹配")
					t.Logf("预期: %d (文本: %d, 图像: %d)", expectedTotalParts, textCount, imageCount)
					t.Logf("实际: %d", len(contentParts))
					return false
				}

				// 统计各类型的 content parts
				textPartsFound := 0
				imagePartsFound := 0

				for _, part := range contentParts {
					switch part.Type {
					case "text":
						textPartsFound++
						// 验证文本内容不为空
						if part.Text == "" {
							t.Logf("文本 part 的内容为空")
							return false
						}
						// 验证文本内容格式正确
						if !strings.HasPrefix(part.Text, "文本内容 ") {
							t.Logf("文本 part 的内容格式不正确: %s", part.Text)
							return false
						}

					case "image_url":
						imagePartsFound++
						// 验证 ImageURL 不为空
						if part.ImageURL == nil {
							t.Logf("图像 part 的 ImageURL 为空")
							return false
						}
						// 验证 URL 不为空
						if part.ImageURL.URL == "" {
							t.Logf("图像 part 的 URL 为空")
							return false
						}
						// 验证 URL 格式（http/https 或 data:）
						url := part.ImageURL.URL
						if !strings.HasPrefix(url, "http://") &&
							!strings.HasPrefix(url, "https://") &&
							!strings.HasPrefix(url, "data:image/") {
							t.Logf("图像 URL 格式不正确: %s", url)
							return false
						}

					default:
						t.Logf("未知的 content part 类型: %s", part.Type)
						return false
					}
				}

				// 验证文本和图像部分的数量
				if textPartsFound != textCount {
					t.Logf("文本 parts 数量不匹配")
					t.Logf("预期: %d", textCount)
					t.Logf("实际: %d", textPartsFound)
					return false
				}

				if imagePartsFound != imageCount {
					t.Logf("图像 parts 数量不匹配")
					t.Logf("预期: %d", imageCount)
					t.Logf("实际: %d", imagePartsFound)
					return false
				}

				// 验证 content parts 的顺序（文本在前，图像在后）
				// 这是根据我们构建消息的方式
				for i := 0; i < textCount; i++ {
					if contentParts[i].Type != "text" {
						t.Logf("位置 %d 应该是文本 part，实际是 %s", i, contentParts[i].Type)
						return false
					}
				}
				for i := textCount; i < textCount+imageCount; i++ {
					if contentParts[i].Type != "image_url" {
						t.Logf("位置 %d 应该是图像 part，实际是 %s", i, contentParts[i].Type)
						return false
					}
				}

				return true
			},
			// 生成文本部分数量（1-5，确保至少有1个）
			gen.IntRange(1, 5),
			// 生成图像部分数量（1-3，确保至少有1个）
			gen.IntRange(1, 3),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 5: 工具定义转换**
// 属性 5: 工具定义转换
// 对于任何 Genkit 工具定义，转换后的 Azure OpenAI 格式应该包含 type="function" 和正确的 function 字段
// 验证需求: 2.4, 6.1
func TestProperty_ToolDefinitionConversion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Genkit 工具定义转换为 Azure OpenAI 格式后包含正确的结构",
		prop.ForAll(
			func(toolCount int) bool {
				// 限制工具数量以避免过大的测试用例
				if toolCount < 1 || toolCount > 10 {
					return true // 跳过无效输入
				}

				// 构建 Genkit 工具定义列表
				var genkitTools []*ai.ToolDefinition

				for i := 0; i < toolCount; i++ {
					tool := &ai.ToolDefinition{
						Name:        fmt.Sprintf("tool_%d", i),
						Description: fmt.Sprintf("这是工具 %d 的描述", i),
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"param1": map[string]any{
									"type":        "string",
									"description": fmt.Sprintf("工具 %d 的参数 1", i),
								},
								"param2": map[string]any{
									"type":        "integer",
									"description": fmt.Sprintf("工具 %d 的参数 2", i),
								},
								"param3": map[string]any{
									"type":        "boolean",
									"description": fmt.Sprintf("工具 %d 的参数 3", i),
								},
							},
							"required": []string{"param1", "param2"},
						},
					}
					genkitTools = append(genkitTools, tool)
				}

				// 转换工具定义
				azureTools := convertTools(genkitTools)

				// 验证转换后的工具数量
				if len(azureTools) != toolCount {
					t.Logf("转换后的工具数量不匹配")
					t.Logf("预期: %d", toolCount)
					t.Logf("实际: %d", len(azureTools))
					return false
				}

				// 验证每个工具的结构
				for i, azureTool := range azureTools {
					// 验证 Type 字段必须是 "function"
					if azureTool.Type != "function" {
						t.Logf("工具 %d 的 Type 字段不正确", i)
						t.Logf("预期: function")
						t.Logf("实际: %s", azureTool.Type)
						return false
					}

					// 验证 Function 字段存在
					if azureTool.Function.Name == "" {
						t.Logf("工具 %d 的 Function.Name 为空", i)
						return false
					}

					// 验证函数名称匹配
					expectedName := fmt.Sprintf("tool_%d", i)
					if azureTool.Function.Name != expectedName {
						t.Logf("工具 %d 的函数名称不匹配", i)
						t.Logf("预期: %s", expectedName)
						t.Logf("实际: %s", azureTool.Function.Name)
						return false
					}

					// 验证描述匹配
					expectedDesc := fmt.Sprintf("这是工具 %d 的描述", i)
					if azureTool.Function.Description != expectedDesc {
						t.Logf("工具 %d 的描述不匹配", i)
						t.Logf("预期: %s", expectedDesc)
						t.Logf("实际: %s", azureTool.Function.Description)
						return false
					}

					// 验证 Parameters 字段存在且不为空
					if azureTool.Function.Parameters == nil {
						t.Logf("工具 %d 的 Parameters 为空", i)
						return false
					}

					// 验证 Parameters 包含正确的结构
					params := azureTool.Function.Parameters

					// 验证 type 字段
					if paramType, ok := params["type"].(string); !ok || paramType != "object" {
						t.Logf("工具 %d 的 Parameters type 不正确", i)
						return false
					}

					// 验证 properties 字段存在
					properties, ok := params["properties"].(map[string]any)
					if !ok {
						t.Logf("工具 %d 的 Parameters properties 字段类型不正确", i)
						return false
					}

					// 验证包含预期的参数
					expectedParams := []string{"param1", "param2", "param3"}
					for _, paramName := range expectedParams {
						if _, exists := properties[paramName]; !exists {
							t.Logf("工具 %d 的 Parameters 缺少参数: %s", i, paramName)
							return false
						}
					}

					// 验证 required 字段存在
					required, ok := params["required"].([]string)
					if !ok {
						// 可能是 []any 类型，尝试转换
						if requiredAny, ok := params["required"].([]any); ok {
							required = make([]string, len(requiredAny))
							for j, v := range requiredAny {
								if str, ok := v.(string); ok {
									required[j] = str
								}
							}
						} else {
							t.Logf("工具 %d 的 Parameters required 字段类型不正确", i)
							return false
						}
					}

					// 验证 required 包含预期的必需参数
					expectedRequired := map[string]bool{"param1": true, "param2": true}
					for _, reqParam := range required {
						if !expectedRequired[reqParam] {
							t.Logf("工具 %d 的 required 包含意外的参数: %s", i, reqParam)
							return false
						}
						delete(expectedRequired, reqParam)
					}

					// 验证所有必需参数都被包含
					if len(expectedRequired) > 0 {
						t.Logf("工具 %d 的 required 缺少必需参数", i)
						for param := range expectedRequired {
							t.Logf("缺少: %s", param)
						}
						return false
					}
				}

				return true
			},
			// 生成工具数量（1-10）
			gen.IntRange(1, 10),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 17: 图像格式支持**
// 属性 17: 图像格式支持
// 对于任何 URL 或 base64 格式的图像，转换后都应该生成有效的 image_url 内容部分
// 验证需求: 5.5
func TestProperty_ImageFormatSupport(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("URL 和 base64 格式的图像都能正确转换",
		prop.ForAll(
			func(imageType string, imageIndex int) bool {
				// 限制索引范围
				if imageIndex < 0 || imageIndex > 100 {
					return true // 跳过无效输入
				}

				var imageURL string
				var expectedPrefix string

				// 根据类型生成不同格式的图像
				switch imageType {
				case "http":
					imageURL = fmt.Sprintf("http://example.com/image%d.jpg", imageIndex)
					expectedPrefix = "http://"
				case "https":
					imageURL = fmt.Sprintf("https://example.com/image%d.png", imageIndex)
					expectedPrefix = "https://"
				case "base64_png":
					// 生成一个简单的 base64 编码图像（1x1 透明 PNG）
					imageURL = fmt.Sprintf("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==%d", imageIndex)
					expectedPrefix = "data:image/png;base64,"
				case "base64_jpeg":
					// 生成一个简单的 base64 编码图像（JPEG 格式）
					imageURL = fmt.Sprintf("data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA%d", imageIndex)
					expectedPrefix = "data:image/jpeg;base64,"
				case "base64_gif":
					// 生成一个简单的 base64 编码图像（GIF 格式）
					imageURL = fmt.Sprintf("data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7%d", imageIndex)
					expectedPrefix = "data:image/gif;base64,"
				default:
					// 未知类型，跳过
					return true
				}

				// 创建包含图像的用户消息
				userMessage := &ai.Message{
					Role: ai.RoleUser,
					Content: []*ai.Part{
						ai.NewMediaPart("image/jpeg", imageURL),
					},
				}

				// 转换消息
				azMsg, err := convertUserMessage(userMessage)
				if err != nil {
					t.Logf("转换用户消息失败: %v", err)
					t.Logf("图像类型: %s", imageType)
					t.Logf("图像 URL: %s", imageURL)
					return false
				}

				// 验证角色
				if azMsg.Role != "user" {
					t.Logf("消息角色不正确")
					t.Logf("预期: user")
					t.Logf("实际: %s", azMsg.Role)
					return false
				}

				// 验证内容是 ContentPart 数组
				contentParts, ok := azMsg.Content.([]ContentPart)
				if !ok {
					t.Logf("消息内容应该是 []ContentPart 类型")
					t.Logf("实际类型: %T", azMsg.Content)
					return false
				}

				// 验证有一个 content part
				if len(contentParts) != 1 {
					t.Logf("应该有 1 个 content part")
					t.Logf("实际数量: %d", len(contentParts))
					return false
				}

				// 验证 content part 类型是 image_url
				if contentParts[0].Type != "image_url" {
					t.Logf("content part 类型应该是 image_url")
					t.Logf("实际类型: %s", contentParts[0].Type)
					return false
				}

				// 验证 ImageURL 不为空
				if contentParts[0].ImageURL == nil {
					t.Logf("ImageURL 不应该为空")
					return false
				}

				// 验证 URL 不为空
				if contentParts[0].ImageURL.URL == "" {
					t.Logf("ImageURL.URL 不应该为空")
					return false
				}

				// 验证 URL 包含预期的前缀
				if !strings.HasPrefix(contentParts[0].ImageURL.URL, expectedPrefix) {
					t.Logf("ImageURL.URL 应该以 %s 开头", expectedPrefix)
					t.Logf("实际 URL: %s", contentParts[0].ImageURL.URL)
					return false
				}

				// 对于 URL 格式，验证完整的 URL
				if imageType == "http" || imageType == "https" {
					if contentParts[0].ImageURL.URL != imageURL {
						t.Logf("ImageURL.URL 不匹配")
						t.Logf("预期: %s", imageURL)
						t.Logf("实际: %s", contentParts[0].ImageURL.URL)
						return false
					}
				}

				// 对于 base64 格式，验证包含 base64 数据
				if strings.HasPrefix(imageType, "base64_") {
					if !strings.Contains(contentParts[0].ImageURL.URL, "base64,") {
						t.Logf("base64 图像 URL 应该包含 'base64,'")
						t.Logf("实际 URL: %s", contentParts[0].ImageURL.URL)
						return false
					}
				}

				return true
			},
			// 生成不同类型的图像格式
			gen.OneConstOf(
				"http",
				"https",
				"base64_png",
				"base64_jpeg",
				"base64_gif",
			),
			// 生成图像索引（用于生成不同的 URL）
			gen.IntRange(0, 100),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 18: 工具调用解析**
// 属性 18: 工具调用解析
// 对于任何包含工具调用的 API 响应，解析后应该提取出工具名称、参数和 ID
// 验证需求: 6.2
func TestProperty_ToolCallParsing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("API 响应中的工具调用能够正确解析",
		prop.ForAll(
			func(toolCallCount int) bool {
				// 限制工具调用数量以避免过大的测试用例
				if toolCallCount < 1 || toolCallCount > 5 {
					return true // 跳过无效输入
				}

				// 构建包含工具调用的 Azure OpenAI 响应
				var toolCalls []ToolCall
				expectedToolData := make(map[string]map[string]any) // toolID -> {name, args}

				for i := 0; i < toolCallCount; i++ {
					toolID := fmt.Sprintf("call_%d_%d", i, i*100)
					toolName := fmt.Sprintf("tool_%d", i)

					// 构建工具参数
					args := map[string]any{
						"param1": fmt.Sprintf("value_%d", i),
						"param2": i * 10,
						"param3": i%2 == 0,
						"nested": map[string]any{
							"key1": fmt.Sprintf("nested_value_%d", i),
							"key2": i * 5,
						},
					}

					// 序列化参数为 JSON 字符串
					argsJSON, err := json.Marshal(args)
					if err != nil {
						t.Logf("序列化工具参数失败: %v", err)
						return false
					}

					toolCall := ToolCall{
						ID:   toolID,
						Type: "function",
						Function: FunctionCall{
							Name:      toolName,
							Arguments: string(argsJSON),
						},
					}

					toolCalls = append(toolCalls, toolCall)

					// 保存预期数据用于验证
					expectedToolData[toolID] = map[string]any{
						"name": toolName,
						"args": args,
					}
				}

				// 构建 Azure OpenAI 响应消息
				azMsg := &Message{
					Role:      "assistant",
					Content:   "我将调用这些工具来帮助你",
					ToolCalls: toolCalls,
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil,
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 使用 convertResponseMessage 方法解析响应消息
				genkitMsg, err := generator.convertResponseMessage(azMsg)
				if err != nil {
					t.Logf("转换响应消息失败: %v", err)
					return false
				}

				// 验证消息角色
				if genkitMsg.Role != ai.RoleModel {
					t.Logf("消息角色不正确")
					t.Logf("预期: %v", ai.RoleModel)
					t.Logf("实际: %v", genkitMsg.Role)
					return false
				}

				// 统计解析出的工具调用
				parsedToolCalls := 0
				for _, part := range genkitMsg.Content {
					if !part.IsToolRequest() {
						continue
					}

					parsedToolCalls++
					toolReq := part.ToolRequest

					// 验证工具调用 ID 存在于预期数据中
					expectedData, exists := expectedToolData[toolReq.Ref]
					if !exists {
						t.Logf("解析出的工具调用 ID 不在预期数据中: %s", toolReq.Ref)
						return false
					}

					// 验证工具名称
					expectedName := expectedData["name"].(string)
					if toolReq.Name != expectedName {
						t.Logf("工具名称不匹配")
						t.Logf("工具 ID: %s", toolReq.Ref)
						t.Logf("预期名称: %s", expectedName)
						t.Logf("实际名称: %s", toolReq.Name)
						return false
					}

					// 验证参数
					expectedArgs := expectedData["args"].(map[string]any)

					// 首先将 Input 断言为 map[string]any
					inputMap, ok := toolReq.Input.(map[string]any)
					if !ok {
						t.Logf("工具 %s 的 Input 类型不是 map[string]any", toolReq.Ref)
						return false
					}

					// 验证 param1
					if param1, ok := inputMap["param1"].(string); !ok {
						t.Logf("工具 %s 的 param1 类型不正确", toolReq.Ref)
						return false
					} else if param1 != expectedArgs["param1"].(string) {
						t.Logf("工具 %s 的 param1 值不匹配", toolReq.Ref)
						t.Logf("预期: %s", expectedArgs["param1"])
						t.Logf("实际: %s", param1)
						return false
					}

					// 验证 param2 (JSON 解析后可能是 float64)
					if param2, ok := inputMap["param2"].(float64); !ok {
						t.Logf("工具 %s 的 param2 类型不正确", toolReq.Ref)
						return false
					} else if int(param2) != expectedArgs["param2"].(int) {
						t.Logf("工具 %s 的 param2 值不匹配", toolReq.Ref)
						t.Logf("预期: %d", expectedArgs["param2"])
						t.Logf("实际: %d", int(param2))
						return false
					}

					// 验证 param3
					if param3, ok := inputMap["param3"].(bool); !ok {
						t.Logf("工具 %s 的 param3 类型不正确", toolReq.Ref)
						return false
					} else if param3 != expectedArgs["param3"].(bool) {
						t.Logf("工具 %s 的 param3 值不匹配", toolReq.Ref)
						t.Logf("预期: %v", expectedArgs["param3"])
						t.Logf("实际: %v", param3)
						return false
					}

					// 验证嵌套对象
					if nested, ok := inputMap["nested"].(map[string]any); !ok {
						t.Logf("工具 %s 的 nested 类型不正确", toolReq.Ref)
						return false
					} else {
						expectedNested := expectedArgs["nested"].(map[string]any)

						// 验证 nested.key1
						if key1, ok := nested["key1"].(string); !ok {
							t.Logf("工具 %s 的 nested.key1 类型不正确", toolReq.Ref)
							return false
						} else if key1 != expectedNested["key1"].(string) {
							t.Logf("工具 %s 的 nested.key1 值不匹配", toolReq.Ref)
							return false
						}

						// 验证 nested.key2 (JSON 解析后可能是 float64)
						if key2, ok := nested["key2"].(float64); !ok {
							t.Logf("工具 %s 的 nested.key2 类型不正确", toolReq.Ref)
							return false
						} else if int(key2) != expectedNested["key2"].(int) {
							t.Logf("工具 %s 的 nested.key2 值不匹配", toolReq.Ref)
							return false
						}
					}
				}

				// 验证解析出的工具调用数量
				if parsedToolCalls != toolCallCount {
					t.Logf("解析出的工具调用数量不匹配")
					t.Logf("预期: %d", toolCallCount)
					t.Logf("实际: %d", parsedToolCalls)
					return false
				}

				return true
			},
			// 生成工具调用数量（1-5）
			gen.IntRange(1, 5),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 3: Responses API 端点使用**
// 属性 3: Responses API 端点使用
// 对于任何模型生成请求，构建的 URL 应该包含 /openai/responses 路径而非 /chat/completions
// 验证需求: 2.2, 9.1
func TestProperty_ResponsesAPIEndpointUsage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("所有模型生成请求都使用 /openai/responses 端点",
		prop.ForAll(
			func(modelName string, apiVersion string) bool {
				// 跳过空字符串
				if modelName == "" || apiVersion == "" {
					return true
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试 URL 构建
					"https://test.openai.azure.com",
					"test-api-key",
					apiVersion,
					modelName,
				)

				// 构建请求 URL
				url := generator.buildRequestURL()

				// 验证 URL 包含 /openai/responses 端点
				if !strings.Contains(url, "/openai/responses") {
					t.Logf("URL 不包含 /openai/responses 端点")
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证 URL 不包含 /chat/completions 端点（旧的端点）
				if strings.Contains(url, "/chat/completions") {
					t.Logf("URL 不应该包含 /chat/completions 端点")
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证 URL 不包含 /completions 端点（更旧的端点）
				if strings.Contains(url, "/completions") && !strings.Contains(url, "/openai/responses") {
					t.Logf("URL 不应该包含 /completions 端点")
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证 URL 格式正确（包含 base URL）
				if !strings.HasPrefix(url, "https://test.openai.azure.com") {
					t.Logf("URL 应该以 base URL 开头")
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证 URL 包含 api-version 参数
				if !strings.Contains(url, "api-version=") {
					t.Logf("URL 应该包含 api-version 参数")
					t.Logf("实际 URL: %s", url)
					return false
				}

				// 验证完整的端点路径格式
				expectedPattern := "/openai/responses?api-version="
				if !strings.Contains(url, expectedPattern) {
					t.Logf("URL 应该包含正确的端点路径格式")
					t.Logf("预期模式: %s", expectedPattern)
					t.Logf("实际 URL: %s", url)
					return false
				}

				return true
			},
			// 生成模型名称
			gen.OneConstOf(
				"gpt-4",
				"gpt-4-turbo",
				"gpt-4-turbo-preview",
				"gpt-35-turbo",
				"gpt-35-turbo-16k",
				"gpt-4o",
				"gpt-4o-mini",
				"gpt-4-vision-preview",
			),
			// 生成 API 版本
			gen.OneConstOf(
				"2025-04-01-preview",
				"2024-12-01-preview",
				"2024-10-01-preview",
				"2024-08-01-preview",
				"2024-06-01",
				"2024-05-01-preview",
				"2024-02-15-preview",
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 28: Input 字段使用**
// 属性 28: Input 字段使用
// 对于任何请求，序列化的 JSON 应该包含 input 字段而非 messages 字段
// 验证需求: 9.2
func TestProperty_InputFieldUsage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("请求序列化后使用 input 字段而非 messages 字段",
		prop.ForAll(
			func(messageCount int, includeTools bool) bool {
				// 限制消息数量以避免过大的测试用例
				if messageCount < 1 || messageCount > 10 {
					return true // 跳过无效输入
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求体构建
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 构建消息列表
				var messages []*ai.Message

				// 添加系统消息
				messages = append(messages, &ai.Message{
					Role:    ai.RoleSystem,
					Content: []*ai.Part{ai.NewTextPart("你是一个有帮助的助手")},
				})

				// 添加用户消息
				for i := 0; i < messageCount; i++ {
					messages = append(messages, &ai.Message{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart(fmt.Sprintf("用户消息 %d", i))},
					})
				}

				// 设置消息
				generator.WithMessages(messages)

				// 如果需要，添加工具
				if includeTools {
					tools := []*ai.ToolDefinition{
						{
							Name:        "get_weather",
							Description: "获取天气信息",
							InputSchema: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{
										"type":        "string",
										"description": "城市名称",
									},
								},
								"required": []string{"location"},
							},
						},
					}
					generator.WithTools(tools)
				}

				// 构建请求体（非流式）
				reqBody, err := generator.buildRequestBody(false)
				if err != nil {
					t.Logf("构建请求体失败: %v", err)
					return false
				}

				// 序列化请求体为 JSON
				jsonBytes, err := json.Marshal(reqBody)
				if err != nil {
					t.Logf("序列化请求体失败: %v", err)
					return false
				}

				jsonStr := string(jsonBytes)

				// 验证 JSON 包含 "input" 字段
				if !strings.Contains(jsonStr, `"input"`) {
					t.Logf("JSON 不包含 'input' 字段")
					t.Logf("实际 JSON: %s", jsonStr)
					return false
				}

				// 验证 JSON 不包含 "messages" 字段
				// 注意：我们需要确保不是在其他上下文中出现 "messages"
				// 例如在错误消息或描述中
				if strings.Contains(jsonStr, `"messages":[`) || strings.Contains(jsonStr, `"messages": [`) {
					t.Logf("JSON 不应该包含 'messages' 字段作为顶级字段")
					t.Logf("实际 JSON: %s", jsonStr)
					return false
				}

				// 解析 JSON 以验证结构
				var parsedReq map[string]any
				if err := json.Unmarshal(jsonBytes, &parsedReq); err != nil {
					t.Logf("解析 JSON 失败: %v", err)
					return false
				}

				// 验证 input 字段存在
				inputField, exists := parsedReq["input"]
				if !exists {
					t.Logf("解析后的 JSON 不包含 'input' 字段")
					t.Logf("JSON 字段: %v", parsedReq)
					return false
				}

				// 验证 input 是数组类型
				inputArray, ok := inputField.([]any)
				if !ok {
					t.Logf("'input' 字段应该是数组类型")
					t.Logf("实际类型: %T", inputField)
					return false
				}

				// 验证 input 数组包含正确数量的消息
				expectedMessageCount := messageCount + 1 // +1 for system message
				if len(inputArray) != expectedMessageCount {
					t.Logf("'input' 数组的消息数量不正确")
					t.Logf("预期: %d", expectedMessageCount)
					t.Logf("实际: %d", len(inputArray))
					return false
				}

				// 验证 messages 字段不存在
				if _, exists := parsedReq["messages"]; exists {
					t.Logf("解析后的 JSON 不应该包含 'messages' 字段")
					t.Logf("JSON 字段: %v", parsedReq)
					return false
				}

				// 验证其他必需字段存在
				if _, exists := parsedReq["model"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'model' 字段")
					return false
				}

				// 如果包含工具，验证 tools 字段存在
				if includeTools {
					if _, exists := parsedReq["tools"]; !exists {
						t.Logf("包含工具时，解析后的 JSON 应该包含 'tools' 字段")
						return false
					}
				}

				// 验证 input 数组中的第一条消息是系统消息
				if len(inputArray) > 0 {
					firstMsg, ok := inputArray[0].(map[string]any)
					if !ok {
						t.Logf("input 数组的第一个元素应该是对象类型")
						return false
					}

					role, exists := firstMsg["role"]
					if !exists {
						t.Logf("input 数组的第一个消息应该包含 'role' 字段")
						return false
					}

					if role != "system" {
						t.Logf("input 数组的第一个消息应该是系统消息")
						t.Logf("实际角色: %v", role)
						return false
					}
				}

				return true
			},
			// 生成消息数量（1-10）
			gen.IntRange(1, 10),
			// 是否包含工具
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 24: Temperature 参数传递**
// 属性 24: Temperature 参数传递
// 对于任何包含 temperature 配置的请求，该参数应该出现在 API 请求体中
// 验证需求: 8.1
func TestProperty_TemperatureParameterPassing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("temperature 参数正确传递到请求体",
		prop.ForAll(
			func(temperature float64) bool {
				// 限制 temperature 范围为有效值（0-2）
				if temperature < 0 || temperature > 2 {
					return true // 跳过无效输入
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求体构建
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 构建消息列表
				messages := []*ai.Message{
					{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart("测试消息")},
					},
				}

				// 设置消息
				generator.WithMessages(messages)

				// 设置包含 temperature 的配置
				config := map[string]any{
					"temperature": temperature,
				}
				generator.WithConfig(config)

				// 构建请求体（非流式）
				reqBody, err := generator.buildRequestBody(false)
				if err != nil {
					t.Logf("构建请求体失败: %v", err)
					return false
				}

				// 验证 Temperature 字段存在
				if reqBody.Temperature == nil {
					t.Logf("请求体中 Temperature 字段为空")
					return false
				}

				// 验证 Temperature 值正确
				if *reqBody.Temperature != temperature {
					t.Logf("Temperature 值不匹配")
					t.Logf("预期: %f", temperature)
					t.Logf("实际: %f", *reqBody.Temperature)
					return false
				}

				// 序列化请求体为 JSON 以验证序列化后的格式
				jsonBytes, err := json.Marshal(reqBody)
				if err != nil {
					t.Logf("序列化请求体失败: %v", err)
					return false
				}

				jsonStr := string(jsonBytes)

				// 验证 JSON 包含 temperature 字段
				if !strings.Contains(jsonStr, `"temperature"`) {
					t.Logf("JSON 不包含 'temperature' 字段")
					t.Logf("实际 JSON: %s", jsonStr)
					return false
				}

				// 解析 JSON 以验证值
				var parsedReq map[string]any
				if err := json.Unmarshal(jsonBytes, &parsedReq); err != nil {
					t.Logf("解析 JSON 失败: %v", err)
					return false
				}

				// 验证 temperature 字段存在
				tempField, exists := parsedReq["temperature"]
				if !exists {
					t.Logf("解析后的 JSON 不包含 'temperature' 字段")
					t.Logf("JSON 字段: %v", parsedReq)
					return false
				}

				// 验证 temperature 值（JSON 解析后可能是 float64）
				tempValue, ok := tempField.(float64)
				if !ok {
					t.Logf("temperature 字段应该是 float64 类型")
					t.Logf("实际类型: %T", tempField)
					return false
				}

				// 验证值匹配（使用小的误差范围，因为浮点数比较）
				epsilon := 0.0001
				if tempValue < temperature-epsilon || tempValue > temperature+epsilon {
					t.Logf("temperature 值不匹配")
					t.Logf("预期: %f", temperature)
					t.Logf("实际: %f", tempValue)
					return false
				}

				return true
			},
			// 生成有效的 temperature 值（0-2）
			gen.Float64Range(0, 2),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 25: MaxTokens 参数传递**
// 属性 25: MaxTokens 参数传递
// 对于任何包含 max_tokens 配置的请求，该参数应该出现在 API 请求体中
// 验证需求: 8.2
func TestProperty_MaxTokensParameterPassing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("max_tokens 参数正确传递到请求体",
		prop.ForAll(
			func(maxTokens int, useSnakeCase bool) bool {
				// 限制 maxTokens 范围为有效值（1-32000）
				// Azure OpenAI 的最大 token 限制通常在这个范围内
				if maxTokens < 1 || maxTokens > 32000 {
					return true // 跳过无效输入
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求体构建
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 构建消息列表
				messages := []*ai.Message{
					{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart("测试消息")},
					},
				}

				// 设置消息
				generator.WithMessages(messages)

				// 设置包含 max_tokens 的配置
				// 测试两种命名方式：max_tokens（snake_case）和 maxTokens（camelCase）
				var config map[string]any
				if useSnakeCase {
					config = map[string]any{
						"max_tokens": maxTokens,
					}
				} else {
					config = map[string]any{
						"maxTokens": maxTokens,
					}
				}
				generator.WithConfig(config)

				// 构建请求体（非流式）
				reqBody, err := generator.buildRequestBody(false)
				if err != nil {
					t.Logf("构建请求体失败: %v", err)
					return false
				}

				// 验证 MaxTokens 字段存在
				if reqBody.MaxTokens == nil {
					t.Logf("请求体中 MaxTokens 字段为空")
					t.Logf("配置: %v", config)
					return false
				}

				// 验证 MaxTokens 值正确
				if *reqBody.MaxTokens != maxTokens {
					t.Logf("MaxTokens 值不匹配")
					t.Logf("预期: %d", maxTokens)
					t.Logf("实际: %d", *reqBody.MaxTokens)
					return false
				}

				// 序列化请求体为 JSON 以验证序列化后的格式
				jsonBytes, err := json.Marshal(reqBody)
				if err != nil {
					t.Logf("序列化请求体失败: %v", err)
					return false
				}

				jsonStr := string(jsonBytes)

				// 验证 JSON 包含 max_tokens 字段（注意：JSON 序列化使用 snake_case）
				if !strings.Contains(jsonStr, `"max_tokens"`) {
					t.Logf("JSON 不包含 'max_tokens' 字段")
					t.Logf("实际 JSON: %s", jsonStr)
					return false
				}

				// 解析 JSON 以验证值
				var parsedReq map[string]any
				if err := json.Unmarshal(jsonBytes, &parsedReq); err != nil {
					t.Logf("解析 JSON 失败: %v", err)
					return false
				}

				// 验证 max_tokens 字段存在
				maxTokensField, exists := parsedReq["max_tokens"]
				if !exists {
					t.Logf("解析后的 JSON 不包含 'max_tokens' 字段")
					t.Logf("JSON 字段: %v", parsedReq)
					return false
				}

				// 验证 max_tokens 值（JSON 解析后可能是 float64）
				var maxTokensValue int
				switch v := maxTokensField.(type) {
				case float64:
					maxTokensValue = int(v)
				case int:
					maxTokensValue = v
				default:
					t.Logf("max_tokens 字段应该是数字类型")
					t.Logf("实际类型: %T", maxTokensField)
					return false
				}

				// 验证值匹配
				if maxTokensValue != maxTokens {
					t.Logf("max_tokens 值不匹配")
					t.Logf("预期: %d", maxTokens)
					t.Logf("实际: %d", maxTokensValue)
					return false
				}

				// 验证其他必需字段存在
				if _, exists := parsedReq["model"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'model' 字段")
					return false
				}

				if _, exists := parsedReq["input"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'input' 字段")
					return false
				}

				return true
			},
			// 生成有效的 max_tokens 值（1-32000）
			// 使用常见的 token 限制值
			gen.OneConstOf(
				1, 10, 50, 100, 256, 512, 1000, 1024, 2000, 2048,
				4000, 4096, 8000, 8192, 16000, 16384, 32000,
			),
			// 是否使用 snake_case 命名（true: max_tokens, false: maxTokens）
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 26: TopP 参数传递**
// 属性 26: TopP 参数传递
// 对于任何包含 top_p 配置的请求，该参数应该出现在 API 请求体中
// 验证需求: 8.3
func TestProperty_TopPParameterPassing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("top_p 参数正确传递到请求体",
		prop.ForAll(
			func(topP float64, useSnakeCase bool) bool {
				// 限制 topP 范围为有效值（0-1）
				// Azure OpenAI 的 top_p 参数范围是 0 到 1
				if topP < 0 || topP > 1 {
					return true // 跳过无效输入
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求体构建
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 构建消息列表
				messages := []*ai.Message{
					{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart("测试消息")},
					},
				}

				// 设置消息
				generator.WithMessages(messages)

				// 设置包含 top_p 的配置
				// 测试两种命名方式：top_p（snake_case）和 topP（camelCase）
				var config map[string]any
				if useSnakeCase {
					config = map[string]any{
						"top_p": topP,
					}
				} else {
					config = map[string]any{
						"topP": topP,
					}
				}
				generator.WithConfig(config)

				// 构建请求体（非流式）
				reqBody, err := generator.buildRequestBody(false)
				if err != nil {
					t.Logf("构建请求体失败: %v", err)
					return false
				}

				// 验证 TopP 字段存在
				if reqBody.TopP == nil {
					t.Logf("请求体中 TopP 字段为空")
					t.Logf("配置: %v", config)
					return false
				}

				// 验证 TopP 值正确
				if *reqBody.TopP != topP {
					t.Logf("TopP 值不匹配")
					t.Logf("预期: %f", topP)
					t.Logf("实际: %f", *reqBody.TopP)
					return false
				}

				// 序列化请求体为 JSON 以验证序列化后的格式
				jsonBytes, err := json.Marshal(reqBody)
				if err != nil {
					t.Logf("序列化请求体失败: %v", err)
					return false
				}

				jsonStr := string(jsonBytes)

				// 验证 JSON 包含 top_p 字段（注意：JSON 序列化使用 snake_case）
				if !strings.Contains(jsonStr, `"top_p"`) {
					t.Logf("JSON 不包含 'top_p' 字段")
					t.Logf("实际 JSON: %s", jsonStr)
					return false
				}

				// 解析 JSON 以验证值
				var parsedReq map[string]any
				if err := json.Unmarshal(jsonBytes, &parsedReq); err != nil {
					t.Logf("解析 JSON 失败: %v", err)
					return false
				}

				// 验证 top_p 字段存在
				topPField, exists := parsedReq["top_p"]
				if !exists {
					t.Logf("解析后的 JSON 不包含 'top_p' 字段")
					t.Logf("JSON 字段: %v", parsedReq)
					return false
				}

				// 验证 top_p 值（JSON 解析后是 float64）
				topPValue, ok := topPField.(float64)
				if !ok {
					t.Logf("top_p 字段应该是 float64 类型")
					t.Logf("实际类型: %T", topPField)
					return false
				}

				// 验证值匹配（使用小的误差范围，因为浮点数比较）
				epsilon := 0.0001
				if topPValue < topP-epsilon || topPValue > topP+epsilon {
					t.Logf("top_p 值不匹配")
					t.Logf("预期: %f", topP)
					t.Logf("实际: %f", topPValue)
					return false
				}

				// 验证其他必需字段存在
				if _, exists := parsedReq["model"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'model' 字段")
					return false
				}

				if _, exists := parsedReq["input"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'input' 字段")
					return false
				}

				return true
			},
			// 生成有效的 top_p 值（0-1）
			// 使用常见的 top_p 值
			gen.OneConstOf(
				0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.75,
				0.8, 0.85, 0.9, 0.95, 0.99, 1.0,
			),
			// 是否使用 snake_case 命名（true: top_p, false: topP）
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 27: Map 配置支持**
// 属性 27: Map 配置支持
// 对于任何 map[string]any 格式的配置，应该能够被正确解析并应用到请求中
// 验证需求: 8.4
func TestProperty_MapConfigurationSupport(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("map[string]any 格式的配置能够正确解析并应用",
		prop.ForAll(
			func(temperature float64, maxTokens int, topP float64, includeStop bool, includeUser bool) bool {
				// 限制参数范围为有效值
				if temperature < 0 || temperature > 2 {
					return true // 跳过无效输入
				}
				if maxTokens < 1 || maxTokens > 32000 {
					return true // 跳过无效输入
				}
				if topP < 0 || topP > 1 {
					return true // 跳过无效输入
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求体构建
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 构建消息列表
				messages := []*ai.Message{
					{
						Role:    ai.RoleUser,
						Content: []*ai.Part{ai.NewTextPart("测试消息")},
					},
				}

				// 设置消息
				generator.WithMessages(messages)

				// 构建 map[string]any 格式的配置
				config := map[string]any{
					"temperature": temperature,
					"maxTokens":   maxTokens, // 使用 camelCase
					"top_p":       topP,      // 使用 snake_case
				}

				// 可选参数
				if includeStop {
					config["stop"] = []string{"停止词1", "停止词2", "END"}
				}

				if includeUser {
					config["user"] = "test_user_123"
				}

				// 设置配置
				generator.WithConfig(config)

				// 验证没有错误
				if generator.err != nil {
					t.Logf("设置配置时发生错误: %v", generator.err)
					return false
				}

				// 构建请求体（非流式）
				reqBody, err := generator.buildRequestBody(false)
				if err != nil {
					t.Logf("构建请求体失败: %v", err)
					return false
				}

				// 验证 Temperature 参数
				if reqBody.Temperature == nil {
					t.Logf("Temperature 参数未被应用")
					return false
				}
				if *reqBody.Temperature != temperature {
					t.Logf("Temperature 值不匹配")
					t.Logf("预期: %f", temperature)
					t.Logf("实际: %f", *reqBody.Temperature)
					return false
				}

				// 验证 MaxTokens 参数
				if reqBody.MaxTokens == nil {
					t.Logf("MaxTokens 参数未被应用")
					return false
				}
				if *reqBody.MaxTokens != maxTokens {
					t.Logf("MaxTokens 值不匹配")
					t.Logf("预期: %d", maxTokens)
					t.Logf("实际: %d", *reqBody.MaxTokens)
					return false
				}

				// 验证 TopP 参数
				if reqBody.TopP == nil {
					t.Logf("TopP 参数未被应用")
					return false
				}
				epsilon := 0.0001
				if *reqBody.TopP < topP-epsilon || *reqBody.TopP > topP+epsilon {
					t.Logf("TopP 值不匹配")
					t.Logf("预期: %f", topP)
					t.Logf("实际: %f", *reqBody.TopP)
					return false
				}

				// 验证 Stop 参数（如果包含）
				if includeStop {
					if reqBody.Stop == nil {
						t.Logf("Stop 参数未被应用")
						return false
					}
					expectedStop := []string{"停止词1", "停止词2", "END"}
					if len(reqBody.Stop) != len(expectedStop) {
						t.Logf("Stop 数组长度不匹配")
						t.Logf("预期: %d", len(expectedStop))
						t.Logf("实际: %d", len(reqBody.Stop))
						return false
					}
					for i, stop := range expectedStop {
						if reqBody.Stop[i] != stop {
							t.Logf("Stop[%d] 值不匹配", i)
							t.Logf("预期: %s", stop)
							t.Logf("实际: %s", reqBody.Stop[i])
							return false
						}
					}
				}

				// 验证 User 参数（如果包含）
				if includeUser {
					if reqBody.User != "test_user_123" {
						t.Logf("User 值不匹配")
						t.Logf("预期: test_user_123")
						t.Logf("实际: %s", reqBody.User)
						return false
					}
				}

				// 序列化请求体为 JSON 以验证序列化后的格式
				jsonBytes, err := json.Marshal(reqBody)
				if err != nil {
					t.Logf("序列化请求体失败: %v", err)
					return false
				}

				// 解析 JSON 以验证所有字段都正确序列化
				var parsedReq map[string]any
				if err := json.Unmarshal(jsonBytes, &parsedReq); err != nil {
					t.Logf("解析 JSON 失败: %v", err)
					return false
				}

				// 验证必需字段存在
				if _, exists := parsedReq["model"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'model' 字段")
					return false
				}

				if _, exists := parsedReq["input"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'input' 字段")
					return false
				}

				// 验证配置参数在 JSON 中存在
				if _, exists := parsedReq["temperature"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'temperature' 字段")
					return false
				}

				if _, exists := parsedReq["max_tokens"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'max_tokens' 字段")
					return false
				}

				if _, exists := parsedReq["top_p"]; !exists {
					t.Logf("解析后的 JSON 应该包含 'top_p' 字段")
					return false
				}

				// 验证可选参数
				if includeStop {
					if _, exists := parsedReq["stop"]; !exists {
						t.Logf("解析后的 JSON 应该包含 'stop' 字段")
						return false
					}
				}

				if includeUser {
					if _, exists := parsedReq["user"]; !exists {
						t.Logf("解析后的 JSON 应该包含 'user' 字段")
						return false
					}
				}

				// 验证配置参数的值类型正确
				if tempField, ok := parsedReq["temperature"].(float64); !ok {
					t.Logf("temperature 字段应该是 float64 类型")
					t.Logf("实际类型: %T", parsedReq["temperature"])
					return false
				} else if tempField < temperature-epsilon || tempField > temperature+epsilon {
					t.Logf("JSON 中的 temperature 值不匹配")
					return false
				}

				// 验证 max_tokens 值
				var maxTokensValue int
				switch v := parsedReq["max_tokens"].(type) {
				case float64:
					maxTokensValue = int(v)
				case int:
					maxTokensValue = v
				default:
					t.Logf("max_tokens 字段应该是数字类型")
					t.Logf("实际类型: %T", parsedReq["max_tokens"])
					return false
				}
				if maxTokensValue != maxTokens {
					t.Logf("JSON 中的 max_tokens 值不匹配")
					return false
				}

				// 验证 top_p 值
				if topPField, ok := parsedReq["top_p"].(float64); !ok {
					t.Logf("top_p 字段应该是 float64 类型")
					t.Logf("实际类型: %T", parsedReq["top_p"])
					return false
				} else if topPField < topP-epsilon || topPField > topP+epsilon {
					t.Logf("JSON 中的 top_p 值不匹配")
					return false
				}

				return true
			},
			// 生成有效的 temperature 值（0-2）
			gen.Float64Range(0, 2),
			// 生成有效的 max_tokens 值（1-32000）
			gen.OneConstOf(
				1, 10, 50, 100, 256, 512, 1000, 1024, 2000, 2048,
				4000, 4096, 8000, 8192, 16000, 16384, 32000,
			),
			// 生成有效的 top_p 值（0-1）
			gen.Float64Range(0, 1),
			// 是否包含 stop 参数
			gen.Bool(),
			// 是否包含 user 参数
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 29: 认证头设置**
// 属性 29: 认证头设置
// 对于任何 HTTP 请求，应该包含 api-key 认证头
// 验证需求: 9.3
func TestProperty_AuthenticationHeaderSetting(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("所有 HTTP 请求都包含 api-key 认证头",
		prop.ForAll(
			func(apiKey string, modelName string) bool {
				// 跳过空字符串
				if apiKey == "" || modelName == "" {
					return true
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil, // client 不需要，因为我们只测试请求头设置
					"https://test.openai.azure.com",
					apiKey,
					"2025-04-01-preview",
					modelName,
				)

				// 创建一个 HTTP 请求
				url := generator.buildRequestURL()
				httpReq, err := http.NewRequest("POST", url, nil)
				if err != nil {
					t.Logf("创建 HTTP 请求失败: %v", err)
					return false
				}

				// 调用 setRequestHeaders 设置请求头
				generator.setRequestHeaders(httpReq)

				// 验证 api-key 头存在
				apiKeyHeader := httpReq.Header.Get("api-key")
				if apiKeyHeader == "" {
					t.Logf("HTTP 请求缺少 api-key 认证头")
					return false
				}

				// 验证 api-key 头的值正确
				if apiKeyHeader != apiKey {
					t.Logf("api-key 认证头的值不匹配")
					t.Logf("预期: %s", apiKey)
					t.Logf("实际: %s", apiKeyHeader)
					return false
				}

				// 验证 Content-Type 头存在
				contentType := httpReq.Header.Get("Content-Type")
				if contentType == "" {
					t.Logf("HTTP 请求缺少 Content-Type 头")
					return false
				}

				// 验证 Content-Type 是 application/json
				if contentType != "application/json" {
					t.Logf("Content-Type 头的值不正确")
					t.Logf("预期: application/json")
					t.Logf("实际: %s", contentType)
					return false
				}

				// 验证 User-Agent 头存在
				userAgent := httpReq.Header.Get("User-Agent")
				if userAgent == "" {
					t.Logf("HTTP 请求缺少 User-Agent 头")
					return false
				}

				// 验证 User-Agent 包含插件标识
				if !strings.Contains(userAgent, "genkit-azure-plugin") {
					t.Logf("User-Agent 头应该包含插件标识")
					t.Logf("实际: %s", userAgent)
					return false
				}

				// 验证请求头的数量（应该至少有这三个）
				if len(httpReq.Header) < 3 {
					t.Logf("HTTP 请求头数量不足")
					t.Logf("预期至少: 3 (api-key, Content-Type, User-Agent)")
					t.Logf("实际: %d", len(httpReq.Header))
					return false
				}

				return true
			},
			// 生成 API Key（使用各种格式）
			gen.OneConstOf(
				"test-api-key-123",
				"sk-1234567890abcdef",
				"azure-key-abc123xyz",
				"api_key_with_underscores",
				"APIKEY123456789",
				"key-with-dashes-and-numbers-123",
				"very-long-api-key-with-many-characters-1234567890abcdefghijklmnopqrstuvwxyz",
			),
			// 生成模型名称
			gen.OneConstOf(
				"gpt-4",
				"gpt-4-turbo",
				"gpt-35-turbo",
				"gpt-4o",
				"gpt-4o-mini",
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 30: 响应格式解析**
// 属性 30: 响应格式解析
// 对于任何符合 Azure OpenAI 响应格式的 JSON，应该能够被正确解析为 Genkit ModelResponse
// 验证需求: 9.4
func TestProperty_ResponseFormatParsing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Azure OpenAI 响应格式能够正确解析为 Genkit ModelResponse",
		prop.ForAll(
			func(responseID string, modelName string, contentText string, includeToolCalls bool, includeUsage bool) bool {
				// 跳过空字符串
				if responseID == "" || modelName == "" {
					return true
				}

				// 构建 Azure OpenAI 响应
				azResp := &ResponsesResponse{
					ID:      responseID,
					Object:  "chat.completion",
					Created: 1234567890,
					Model:   modelName,
				}

				// 构建消息内容
				message := Message{
					Role:    "assistant",
					Content: contentText,
				}

				// 如果包含工具调用，添加工具调用
				if includeToolCalls {
					toolCalls := []ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"北京","unit":"celsius"}`,
							},
						},
						{
							ID:   "call_2",
							Type: "function",
							Function: FunctionCall{
								Name:      "get_time",
								Arguments: `{"timezone":"Asia/Shanghai"}`,
							},
						},
					}
					message.ToolCalls = toolCalls
				}

				// 构建 Choice
				choice := Choice{
					Index:        0,
					Message:      message,
					FinishReason: "stop",
				}

				azResp.Choices = []Choice{choice}

				// 如果包含使用统计，添加 Usage
				if includeUsage {
					azResp.Usage = Usage{
						PromptTokens:     100,
						CompletionTokens: 50,
						TotalTokens:      150,
					}
				}

				// 添加系统指纹
				azResp.SystemFingerprint = "fp_123456"

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil,
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					modelName,
				)

				// 使用 convertToModelResponse 方法解析响应
				genkitResp, err := generator.convertToModelResponse(azResp)
				if err != nil {
					t.Logf("解析响应失败: %v", err)
					return false
				}

				// 验证响应不为空
				if genkitResp == nil {
					t.Logf("解析后的响应为空")
					return false
				}

				// 验证消息不为空
				if genkitResp.Message == nil {
					t.Logf("解析后的消息为空")
					return false
				}

				// 验证消息角色
				if genkitResp.Message.Role != ai.RoleModel {
					t.Logf("消息角色不正确")
					t.Logf("预期: %v", ai.RoleModel)
					t.Logf("实际: %v", genkitResp.Message.Role)
					return false
				}

				// 验证消息内容
				// 注意：如果既没有文本内容也没有工具调用，消息内容可能为空
				if contentText == "" && !includeToolCalls {
					// 这是一个边缘情况，但仍然是有效的响应
					// 跳过此测试用例
					return true
				}

				if len(genkitResp.Message.Content) == 0 {
					t.Logf("消息内容为空")
					t.Logf("contentText: %s", contentText)
					t.Logf("includeToolCalls: %v", includeToolCalls)
					return false
				}

				// 如果有文本内容，验证文本部分
				if contentText != "" {
					hasTextPart := false
					for _, part := range genkitResp.Message.Content {
						if part.IsText() {
							hasTextPart = true
							if part.Text != contentText {
								t.Logf("文本内容不匹配")
								t.Logf("预期: %s", contentText)
								t.Logf("实际: %s", part.Text)
								return false
							}
							break
						}
					}
					if !hasTextPart {
						t.Logf("消息应该包含文本部分")
						return false
					}
				}

				// 如果包含工具调用，验证工具调用部分
				if includeToolCalls {
					toolCallCount := 0
					for _, part := range genkitResp.Message.Content {
						if part.IsToolRequest() {
							toolCallCount++
						}
					}

					// 应该有 2 个工具调用
					if toolCallCount != 2 {
						t.Logf("工具调用数量不匹配")
						t.Logf("预期: 2")
						t.Logf("实际: %d", toolCallCount)
						return false
					}

					// 验证工具调用的详细信息
					toolCallsFound := make(map[string]bool)
					for _, part := range genkitResp.Message.Content {
						if part.IsToolRequest() {
							toolReq := part.ToolRequest
							toolCallsFound[toolReq.Name] = true

							// 验证工具调用 ID 不为空
							if toolReq.Ref == "" {
								t.Logf("工具调用 ID 为空")
								return false
							}

							// 验证工具调用参数
							if toolReq.Input == nil {
								t.Logf("工具调用参数为空")
								return false
							}

							// 验证参数是 map 类型
							inputMap, ok := toolReq.Input.(map[string]any)
							if !ok {
								t.Logf("工具调用参数应该是 map[string]any 类型")
								t.Logf("实际类型: %T", toolReq.Input)
								return false
							}

							// 根据工具名称验证参数
							switch toolReq.Name {
							case "get_weather":
								if location, ok := inputMap["location"].(string); !ok || location != "北京" {
									t.Logf("get_weather 的 location 参数不正确")
									return false
								}
								if unit, ok := inputMap["unit"].(string); !ok || unit != "celsius" {
									t.Logf("get_weather 的 unit 参数不正确")
									return false
								}
							case "get_time":
								if timezone, ok := inputMap["timezone"].(string); !ok || timezone != "Asia/Shanghai" {
									t.Logf("get_time 的 timezone 参数不正确")
									return false
								}
							}
						}
					}

					// 验证找到了所有预期的工具调用
					if !toolCallsFound["get_weather"] {
						t.Logf("未找到 get_weather 工具调用")
						return false
					}
					if !toolCallsFound["get_time"] {
						t.Logf("未找到 get_time 工具调用")
						return false
					}
				}

				// 验证 FinishReason
				if genkitResp.FinishReason != ai.FinishReasonStop {
					t.Logf("FinishReason 不正确")
					t.Logf("预期: %v", ai.FinishReasonStop)
					t.Logf("实际: %v", genkitResp.FinishReason)
					return false
				}

				// 如果包含使用统计，验证 Usage
				if includeUsage {
					if genkitResp.Usage == nil {
						t.Logf("响应应该包含 Usage 信息")
						return false
					}

					if genkitResp.Usage.InputTokens != 100 {
						t.Logf("InputTokens 不匹配")
						t.Logf("预期: 100")
						t.Logf("实际: %d", genkitResp.Usage.InputTokens)
						return false
					}

					if genkitResp.Usage.OutputTokens != 50 {
						t.Logf("OutputTokens 不匹配")
						t.Logf("预期: 50")
						t.Logf("实际: %d", genkitResp.Usage.OutputTokens)
						return false
					}

					if genkitResp.Usage.TotalTokens != 150 {
						t.Logf("TotalTokens 不匹配")
						t.Logf("预期: 150")
						t.Logf("实际: %d", genkitResp.Usage.TotalTokens)
						return false
					}
				}

				// 验证 Custom 字段包含系统指纹
				if genkitResp.Custom == nil {
					t.Logf("响应应该包含 Custom 字段")
					return false
				}

				customMap, ok := genkitResp.Custom.(map[string]any)
				if !ok {
					t.Logf("Custom 字段应该是 map[string]any 类型")
					t.Logf("实际类型: %T", genkitResp.Custom)
					return false
				}

				if fingerprint, ok := customMap["system_fingerprint"].(string); !ok || fingerprint != "fp_123456" {
					t.Logf("system_fingerprint 不匹配")
					t.Logf("预期: fp_123456")
					if ok {
						t.Logf("实际: %s", fingerprint)
					} else {
						t.Logf("实际: 字段不存在或类型不正确")
					}
					return false
				}

				// 验证模型名称
				if modelField, ok := customMap["model"].(string); !ok || modelField != modelName {
					t.Logf("model 字段不匹配")
					t.Logf("预期: %s", modelName)
					if ok {
						t.Logf("实际: %s", modelField)
					} else {
						t.Logf("实际: 字段不存在或类型不正确")
					}
					return false
				}

				// 验证响应 ID
				if idField, ok := customMap["id"].(string); !ok || idField != responseID {
					t.Logf("id 字段不匹配")
					t.Logf("预期: %s", responseID)
					if ok {
						t.Logf("实际: %s", idField)
					} else {
						t.Logf("实际: 字段不存在或类型不正确")
					}
					return false
				}

				return true
			},
			// 生成响应 ID
			gen.OneConstOf(
				"chatcmpl-123",
				"chatcmpl-456789",
				"resp-abc123",
				"response-xyz-789",
				"cmpl-test-001",
			),
			// 生成模型名称
			gen.OneConstOf(
				"gpt-4",
				"gpt-4-turbo",
				"gpt-35-turbo",
				"gpt-4o",
				"gpt-4o-mini",
			),
			// 生成内容文本
			gen.OneConstOf(
				"这是一个测试响应",
				"Hello, how can I help you?",
				"我可以帮你查询天气信息",
				"",
				"Let me call some tools to help you",
			),
			// 是否包含工具调用
			gen.Bool(),
			// 是否包含使用统计
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 20: Token 使用统计提取**
// 属性 20: Token 使用统计提取
// 对于任何包含 usage 字段的响应，应该正确提取 prompt_tokens、completion_tokens 和 total_tokens
// 验证需求: 7.1
func TestProperty_TokenUsageStatisticsExtraction(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("响应中的 token 使用统计能够正确提取",
		prop.ForAll(
			func(promptTokens int, completionTokens int) bool {
				// 限制 token 数量范围为有效值（1-100000）
				if promptTokens < 1 || promptTokens > 100000 {
					return true // 跳过无效输入
				}
				if completionTokens < 1 || completionTokens > 100000 {
					return true // 跳过无效输入
				}

				// 计算总 token 数
				totalTokens := promptTokens + completionTokens

				// 构建包含 Usage 信息的 Azure OpenAI 响应
				azResp := &ResponsesResponse{
					ID:      "chatcmpl-test-123",
					Object:  "chat.completion",
					Created: 1234567890,
					Model:   "gpt-4",
					Choices: []Choice{
						{
							Index: 0,
							Message: Message{
								Role:    "assistant",
								Content: "这是一个测试响应",
							},
							FinishReason: "stop",
						},
					},
					Usage: Usage{
						PromptTokens:     promptTokens,
						CompletionTokens: completionTokens,
						TotalTokens:      totalTokens,
					},
					SystemFingerprint: "fp_test_123",
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil,
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 使用 convertToModelResponse 方法解析响应
				genkitResp, err := generator.convertToModelResponse(azResp)
				if err != nil {
					t.Logf("解析响应失败: %v", err)
					return false
				}

				// 验证响应不为空
				if genkitResp == nil {
					t.Logf("解析后的响应为空")
					return false
				}

				// 验证 Usage 字段存在
				if genkitResp.Usage == nil {
					t.Logf("响应应该包含 Usage 信息")
					return false
				}

				// 验证 InputTokens（对应 prompt_tokens）
				if genkitResp.Usage.InputTokens != promptTokens {
					t.Logf("InputTokens 不匹配")
					t.Logf("预期: %d", promptTokens)
					t.Logf("实际: %d", genkitResp.Usage.InputTokens)
					return false
				}

				// 验证 OutputTokens（对应 completion_tokens）
				if genkitResp.Usage.OutputTokens != completionTokens {
					t.Logf("OutputTokens 不匹配")
					t.Logf("预期: %d", completionTokens)
					t.Logf("实际: %d", genkitResp.Usage.OutputTokens)
					return false
				}

				// 验证 TotalTokens
				if genkitResp.Usage.TotalTokens != totalTokens {
					t.Logf("TotalTokens 不匹配")
					t.Logf("预期: %d", totalTokens)
					t.Logf("实际: %d", genkitResp.Usage.TotalTokens)
					return false
				}

				// 验证 TotalTokens 等于 InputTokens + OutputTokens
				if genkitResp.Usage.TotalTokens != genkitResp.Usage.InputTokens+genkitResp.Usage.OutputTokens {
					t.Logf("TotalTokens 应该等于 InputTokens + OutputTokens")
					t.Logf("InputTokens: %d", genkitResp.Usage.InputTokens)
					t.Logf("OutputTokens: %d", genkitResp.Usage.OutputTokens)
					t.Logf("TotalTokens: %d", genkitResp.Usage.TotalTokens)
					t.Logf("预期 TotalTokens: %d", genkitResp.Usage.InputTokens+genkitResp.Usage.OutputTokens)
					return false
				}

				// 验证所有 token 数量都是正数
				if genkitResp.Usage.InputTokens <= 0 {
					t.Logf("InputTokens 应该是正数")
					t.Logf("实际: %d", genkitResp.Usage.InputTokens)
					return false
				}

				if genkitResp.Usage.OutputTokens <= 0 {
					t.Logf("OutputTokens 应该是正数")
					t.Logf("实际: %d", genkitResp.Usage.OutputTokens)
					return false
				}

				if genkitResp.Usage.TotalTokens <= 0 {
					t.Logf("TotalTokens 应该是正数")
					t.Logf("实际: %d", genkitResp.Usage.TotalTokens)
					return false
				}

				// 验证 TotalTokens 大于等于 InputTokens 和 OutputTokens
				if genkitResp.Usage.TotalTokens < genkitResp.Usage.InputTokens {
					t.Logf("TotalTokens 应该大于等于 InputTokens")
					return false
				}

				if genkitResp.Usage.TotalTokens < genkitResp.Usage.OutputTokens {
					t.Logf("TotalTokens 应该大于等于 OutputTokens")
					return false
				}

				// 验证响应的其他字段也正确设置
				if genkitResp.Message == nil {
					t.Logf("响应应该包含 Message")
					return false
				}

				if genkitResp.FinishReason != ai.FinishReasonStop {
					t.Logf("FinishReason 不正确")
					t.Logf("预期: %v", ai.FinishReasonStop)
					t.Logf("实际: %v", genkitResp.FinishReason)
					return false
				}

				// 验证 Custom 字段包含响应元数据
				if genkitResp.Custom == nil {
					t.Logf("响应应该包含 Custom 字段")
					return false
				}

				customMap, ok := genkitResp.Custom.(map[string]any)
				if !ok {
					t.Logf("Custom 字段应该是 map[string]any 类型")
					return false
				}

				// 验证 Custom 字段包含必要的元数据
				if _, exists := customMap["id"]; !exists {
					t.Logf("Custom 字段应该包含 id")
					return false
				}

				if _, exists := customMap["model"]; !exists {
					t.Logf("Custom 字段应该包含 model")
					return false
				}

				return true
			},
			// 生成有效的 prompt_tokens 值（1-100000）
			// 使用常见的 token 数量范围
			gen.OneConstOf(
				1, 5, 10, 20, 50, 100, 150, 200, 250, 300, 400, 500,
				600, 700, 800, 900, 1000, 1500, 2000, 2500, 3000,
				4000, 5000, 6000, 7000, 8000, 10000, 15000, 20000,
				25000, 30000, 40000, 50000, 75000, 100000,
			),
			// 生成有效的 completion_tokens 值（1-100000）
			// 使用常见的 token 数量范围
			gen.OneConstOf(
				1, 5, 10, 20, 30, 40, 50, 75, 100, 150, 200, 250,
				300, 400, 500, 600, 700, 800, 900, 1000, 1200, 1500,
				2000, 2500, 3000, 4000, 5000, 6000, 7000, 8000,
				10000, 15000, 20000, 25000, 30000, 40000, 50000,
				75000, 100000,
			),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 22: 系统指纹存储**
// 属性 22: 系统指纹存储
// 对于任何包含 system_fingerprint 的响应，该值应该被存储在响应的 Custom 字段中
// 验证需求: 7.3
func TestProperty_SystemFingerprintStorage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("响应中的 system_fingerprint 能够正确存储到 Custom 字段",
		prop.ForAll(
			func(systemFingerprint string, includeFingerprint bool) bool {
				// 跳过空字符串（当不包含指纹时）
				if !includeFingerprint {
					systemFingerprint = ""
				}

				// 如果包含指纹但字符串为空，跳过
				if includeFingerprint && systemFingerprint == "" {
					return true
				}

				// 构建 Azure OpenAI 响应
				azResp := &ResponsesResponse{
					ID:      "chatcmpl-test-123",
					Object:  "chat.completion",
					Created: 1234567890,
					Model:   "gpt-4",
					Choices: []Choice{
						{
							Index: 0,
							Message: Message{
								Role:    "assistant",
								Content: "这是一个测试响应",
							},
							FinishReason: "stop",
						},
					},
					Usage: Usage{
						PromptTokens:     100,
						CompletionTokens: 50,
						TotalTokens:      150,
					},
				}

				// 如果包含指纹，设置 SystemFingerprint 字段
				if includeFingerprint {
					azResp.SystemFingerprint = systemFingerprint
				}

				// 创建 ModelGenerator 实例
				generator := NewModelGenerator(
					nil,
					"https://test.openai.azure.com",
					"test-api-key",
					"2025-04-01-preview",
					"gpt-4",
				)

				// 使用 convertToModelResponse 方法解析响应
				genkitResp, err := generator.convertToModelResponse(azResp)
				if err != nil {
					t.Logf("解析响应失败: %v", err)
					return false
				}

				// 验证响应不为空
				if genkitResp == nil {
					t.Logf("解析后的响应为空")
					return false
				}

				// 验证 Custom 字段存在
				if genkitResp.Custom == nil {
					t.Logf("响应应该包含 Custom 字段")
					return false
				}

				// 将 Custom 字段转换为 map
				customMap, ok := genkitResp.Custom.(map[string]any)
				if !ok {
					t.Logf("Custom 字段应该是 map[string]any 类型")
					t.Logf("实际类型: %T", genkitResp.Custom)
					return false
				}

				// 如果包含指纹，验证 system_fingerprint 字段存在且值正确
				if includeFingerprint {
					fingerprintField, exists := customMap["system_fingerprint"]
					if !exists {
						t.Logf("Custom 字段应该包含 system_fingerprint")
						t.Logf("Custom 字段内容: %v", customMap)
						return false
					}

					// 验证 system_fingerprint 的值
					fingerprintStr, ok := fingerprintField.(string)
					if !ok {
						t.Logf("system_fingerprint 应该是字符串类型")
						t.Logf("实际类型: %T", fingerprintField)
						return false
					}

					if fingerprintStr != systemFingerprint {
						t.Logf("system_fingerprint 值不匹配")
						t.Logf("预期: %s", systemFingerprint)
						t.Logf("实际: %s", fingerprintStr)
						return false
					}

					// 验证 system_fingerprint 不为空
					if fingerprintStr == "" {
						t.Logf("system_fingerprint 不应该为空字符串")
						return false
					}
				} else {
					// 如果不包含指纹，验证 system_fingerprint 字段不存在或为空
					if fingerprintField, exists := customMap["system_fingerprint"]; exists {
						// 如果字段存在，它应该是空字符串或不存在
						if fingerprintStr, ok := fingerprintField.(string); ok && fingerprintStr != "" {
							t.Logf("当响应不包含 system_fingerprint 时，Custom 字段不应该包含非空的 system_fingerprint")
							t.Logf("实际值: %s", fingerprintStr)
							return false
						}
					}
				}

				// 验证 Custom 字段还包含其他必要的元数据
				// 验证 id 字段存在
				if idField, exists := customMap["id"]; !exists {
					t.Logf("Custom 字段应该包含 id")
					return false
				} else {
					if idStr, ok := idField.(string); !ok || idStr != "chatcmpl-test-123" {
						t.Logf("id 字段值不正确")
						return false
					}
				}

				// 验证 model 字段存在
				if modelField, exists := customMap["model"]; !exists {
					t.Logf("Custom 字段应该包含 model")
					return false
				} else {
					if modelStr, ok := modelField.(string); !ok || modelStr != "gpt-4" {
						t.Logf("model 字段值不正确")
						return false
					}
				}

				// 验证响应的其他字段也正确设置
				if genkitResp.Message == nil {
					t.Logf("响应应该包含 Message")
					return false
				}

				if genkitResp.Usage == nil {
					t.Logf("响应应该包含 Usage")
					return false
				}

				if genkitResp.FinishReason != ai.FinishReasonStop {
					t.Logf("FinishReason 不正确")
					t.Logf("预期: %v", ai.FinishReasonStop)
					t.Logf("实际: %v", genkitResp.FinishReason)
					return false
				}

				return true
			},
			// 生成各种格式的 system_fingerprint 值
			gen.OneConstOf(
				"fp_123456",
				"fp_abcdef123456",
				"fingerprint_test_001",
				"fp_xyz789",
				"fp_44709d6fcb",
				"fp_2f57f81c52",
				"fp_eeff13170a",
				"system_fp_12345678",
				"fp_a1b2c3d4e5f6",
				"fp_test_fingerprint_long_string_12345",
			),
			// 是否包含 system_fingerprint
			gen.Bool(),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}

// **Feature: azure-ai-genkit-provider, Property 19: 工具调用响应关联**
// 属性 19: 工具调用响应关联
// 对于任何工具调用和对应的工具响应，它们应该通过相同的 ID 关联
// 验证需求: 6.4
func TestProperty_ToolCallResponseAssociation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("工具调用和工具响应通过相同的 ID 正确关联",
		prop.ForAll(
			func(toolCount int) bool {
				// 限制工具数量以避免过大的测试用例
				if toolCount < 1 || toolCount > 5 {
					return true // 跳过无效输入
				}

				// 第一步：创建包含工具调用的助手消息
				var assistantParts []*ai.Part
				toolIDs := make([]string, toolCount)

				for i := 0; i < toolCount; i++ {
					toolID := fmt.Sprintf("call_%d_%d", i, i*100)
					toolIDs[i] = toolID

					toolRequest := &ai.ToolRequest{
						Ref:  toolID,
						Name: fmt.Sprintf("tool_%d", i),
						Input: map[string]any{
							"param1": fmt.Sprintf("value_%d", i),
							"param2": i * 10,
						},
					}
					assistantParts = append(assistantParts, ai.NewToolRequestPart(toolRequest))
				}

				assistantMessage := &ai.Message{
					Role:    ai.RoleModel,
					Content: assistantParts,
				}

				// 转换助手消息（包含工具调用）
				azAssistantMsg, err := convertAssistantMessage(assistantMessage)
				if err != nil {
					t.Logf("转换助手消息失败: %v", err)
					return false
				}

				// 验证工具调用包含正确的 ID
				if len(azAssistantMsg.ToolCalls) != toolCount {
					t.Logf("工具调用数量不匹配")
					t.Logf("预期: %d", toolCount)
					t.Logf("实际: %d", len(azAssistantMsg.ToolCalls))
					return false
				}

				// 提取工具调用的 ID
				extractedToolIDs := make(map[string]bool)
				for _, toolCall := range azAssistantMsg.ToolCalls {
					extractedToolIDs[toolCall.ID] = true
				}

				// 验证所有预期的工具 ID 都存在
				for _, expectedID := range toolIDs {
					if !extractedToolIDs[expectedID] {
						t.Logf("工具调用缺少预期的 ID: %s", expectedID)
						return false
					}
				}

				// 第二步：创建对应的工具响应消息
				var toolResponseParts []*ai.Part

				for i := 0; i < toolCount; i++ {
					// 使用相同的 ID 创建工具响应
					toolResponse := &ai.ToolResponse{
						Ref:  toolIDs[i], // 使用相同的 ID
						Name: fmt.Sprintf("tool_%d", i),
						Output: map[string]any{
							"result":  fmt.Sprintf("result_%d", i),
							"status":  "success",
							"value":   i * 20,
							"message": fmt.Sprintf("工具 %d 执行成功", i),
						},
					}
					toolResponseParts = append(toolResponseParts, ai.NewToolResponsePart(toolResponse))
				}

				toolMessage := &ai.Message{
					Role:    ai.RoleTool,
					Content: toolResponseParts,
				}

				// 转换工具响应消息
				azToolMsgs, err := convertToolResponses(toolMessage)
				if err != nil {
					t.Logf("转换工具响应失败: %v", err)
					return false
				}

				// 验证工具响应消息数量
				if len(azToolMsgs) != toolCount {
					t.Logf("工具响应消息数量不匹配")
					t.Logf("预期: %d", toolCount)
					t.Logf("实际: %d", len(azToolMsgs))
					return false
				}

				// 第三步：验证工具响应的 ToolCallID 与原始工具调用的 ID 匹配
				for i, azToolMsg := range azToolMsgs {
					// 验证角色
					if azToolMsg.Role != "tool" {
						t.Logf("工具响应 %d 的角色不正确", i)
						t.Logf("预期: tool")
						t.Logf("实际: %s", azToolMsg.Role)
						return false
					}

					// 验证 ToolCallID 存在
					if azToolMsg.ToolCallID == "" {
						t.Logf("工具响应 %d 的 ToolCallID 为空", i)
						return false
					}

					// 验证 ToolCallID 是否在原始工具调用的 ID 列表中
					if !extractedToolIDs[azToolMsg.ToolCallID] {
						t.Logf("工具响应 %d 的 ToolCallID 不在原始工具调用的 ID 列表中", i)
						t.Logf("ToolCallID: %s", azToolMsg.ToolCallID)
						t.Logf("原始工具调用 IDs: %v", toolIDs)
						return false
					}

					// 验证内容不为空
					if azToolMsg.Content == nil {
						t.Logf("工具响应 %d 的内容为空", i)
						return false
					}

					// 验证内容是字符串类型
					contentStr, ok := azToolMsg.Content.(string)
					if !ok {
						t.Logf("工具响应 %d 的内容应该是字符串类型", i)
						t.Logf("实际类型: %T", azToolMsg.Content)
						return false
					}

					// 验证内容是有效的 JSON
					var output map[string]any
					if err := json.Unmarshal([]byte(contentStr), &output); err != nil {
						t.Logf("工具响应 %d 的内容不是有效的 JSON: %v", i, err)
						return false
					}
				}

				// 第四步：验证完整的关联关系
				// 创建一个映射：工具调用 ID -> 工具名称
				toolCallIDToName := make(map[string]string)
				for i, toolCall := range azAssistantMsg.ToolCalls {
					toolCallIDToName[toolCall.ID] = fmt.Sprintf("tool_%d", i)
				}

				// 验证每个工具响应都能关联到对应的工具调用
				for i, azToolMsg := range azToolMsgs {
					// 通过 ToolCallID 查找对应的工具名称
					expectedToolName, exists := toolCallIDToName[azToolMsg.ToolCallID]
					if !exists {
						t.Logf("工具响应 %d 的 ToolCallID 无法关联到任何工具调用", i)
						t.Logf("ToolCallID: %s", azToolMsg.ToolCallID)
						return false
					}

					// 验证工具响应的内容包含正确的工具名称信息
					// （虽然 Azure OpenAI 的工具响应消息本身不包含工具名称，
					// 但我们可以通过 ToolCallID 关联到原始的工具调用）
					_ = expectedToolName // 用于验证关联关系存在

					// 验证工具响应的输出内容
					contentStr := azToolMsg.Content.(string)
					var output map[string]any
					json.Unmarshal([]byte(contentStr), &output)

					// 验证输出包含预期的字段
					if _, ok := output["result"]; !ok {
						t.Logf("工具响应 %d 的输出缺少 result 字段", i)
						return false
					}

					if _, ok := output["status"]; !ok {
						t.Logf("工具响应 %d 的输出缺少 status 字段", i)
						return false
					}

					if _, ok := output["value"]; !ok {
						t.Logf("工具响应 %d 的输出缺少 value 字段", i)
						return false
					}
				}

				// 第五步：验证双向关联
				// 确保每个工具调用都有对应的工具响应
				toolResponseIDs := make(map[string]bool)
				for _, azToolMsg := range azToolMsgs {
					toolResponseIDs[azToolMsg.ToolCallID] = true
				}

				for _, toolCall := range azAssistantMsg.ToolCalls {
					if !toolResponseIDs[toolCall.ID] {
						t.Logf("工具调用 %s 没有对应的工具响应", toolCall.ID)
						return false
					}
				}

				return true
			},
			// 生成工具数量（1-5）
			gen.IntRange(1, 5),
		),
	)

	// 运行至少 100 次迭代
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	properties.TestingRun(t, params)
}
