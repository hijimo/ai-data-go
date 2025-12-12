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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/firebase/genkit/go/ai"
)

// ModelGenerator 处理 Azure OpenAI 生成请求
type ModelGenerator struct {
	client          *http.Client
	retryableClient *RetryableHTTPClient
	baseURL         string
	apiKey          string
	apiVersion      string
	modelName       string

	// 请求构建
	messages   []Message
	tools      []Tool
	toolChoice any
	config     map[string]any

	// 调试选项
	enableDebugLog bool

	// 错误跟踪
	err error
}

// NewModelGenerator 创建一个新的 ModelGenerator 实例
func NewModelGenerator(client *http.Client, baseURL, apiKey, apiVersion, modelName string) *ModelGenerator {
	return &ModelGenerator{
		client:     client,
		baseURL:    baseURL,
		apiKey:     apiKey,
		apiVersion: apiVersion,
		modelName:  modelName,
	}
}

// NewModelGeneratorWithRetry 创建一个支持重试的 ModelGenerator 实例
func NewModelGeneratorWithRetry(retryableClient *RetryableHTTPClient, baseURL, apiKey, apiVersion, modelName string) *ModelGenerator {
	return &ModelGenerator{
		retryableClient: retryableClient,
		baseURL:         baseURL,
		apiKey:          apiKey,
		apiVersion:      apiVersion,
		modelName:       modelName,
	}
}

// WithMessages 添加消息到请求
func (g *ModelGenerator) WithMessages(messages []*ai.Message) *ModelGenerator {
	if g.err != nil {
		return g
	}

	azMessages, err := convertMessages(messages)
	if err != nil {
		g.err = err
		return g
	}

	g.messages = azMessages
	return g
}

// WithTools 添加工具到请求
func (g *ModelGenerator) WithTools(tools []*ai.ToolDefinition) *ModelGenerator {
	if g.err != nil {
		return g
	}

	azTools := convertTools(tools)
	g.tools = azTools
	return g
}

// WithConfig 添加配置参数
// 支持以下配置参数：
// - temperature: float64 - 采样温度（0-2）
// - max_tokens/maxTokens: int - 生成的最大 token 数
// - top_p/topP: float64 - 核采样参数
// - frequency_penalty/frequencyPenalty: float64 - 频率惩罚（-2.0 到 2.0）
// - presence_penalty/presencePenalty: float64 - 存在惩罚（-2.0 到 2.0）
// - stop: []string - 停止序列
// - user: string - 用户标识符
func (g *ModelGenerator) WithConfig(config any) *ModelGenerator {
	if g.err != nil {
		return g
	}

	if config == nil {
		return g
	}

	// 将配置转换为 map[string]any
	configMap, err := convertConfigToMap(config)
	if err != nil {
		g.err = NewRequestError("配置格式无效", err)
		return g
	}

	g.config = configMap
	return g
}

// WithDebugLog 设置是否启用调试日志
func (g *ModelGenerator) WithDebugLog(enable bool) *ModelGenerator {
	g.enableDebugLog = enable
	return g
}

// buildRequestURL 构建请求 URL，包含 api-version 参数
func (g *ModelGenerator) buildRequestURL() string {
	// 使用 Responses API 端点
	return fmt.Sprintf("%s/openai/responses?api-version=%s", g.baseURL, g.apiVersion)
}

// buildRequestBody 构建请求体，使用 input 字段而非 messages 字段
func (g *ModelGenerator) buildRequestBody(stream bool) (*ResponsesRequest, error) {
	req := &ResponsesRequest{
		Model:  g.modelName,
		Input:  g.messages,
		Stream: stream,
		Tools:  g.tools,
	}

	// 应用配置参数
	if g.config != nil {
		applyConfig(req, g.config)
	}

	// 如果有工具且没有设置 tool_choice，使用默认值
	if len(g.tools) > 0 && g.toolChoice != nil {
		req.ToolChoice = g.toolChoice
	}

	return req, nil
}

// Generate 执行生成请求
func (g *ModelGenerator) Generate(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// 检查是否有错误
	if g.err != nil {
		return nil, g.err
	}

	// 验证必需字段
	if len(g.messages) == 0 {
		return nil, NewRequestError("消息列表不能为空", nil)
	}

	// 确定是否使用流式模式
	isStreaming := cb != nil

	if isStreaming {
		// 流式模式
		return g.generateStreaming(ctx, cb)
	}

	// 非流式模式
	return g.generateNonStreaming(ctx)
}

// generateNonStreaming 执行非流式生成请求
func (g *ModelGenerator) generateNonStreaming(ctx context.Context) (*ai.ModelResponse, error) {
	// 构建请求体
	reqBody, err := g.buildRequestBody(false)
	if err != nil {
		return nil, err
	}

	// 序列化请求体
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewRequestError("序列化请求体失败", err)
	}

	// 构建请求 URL
	url := g.buildRequestURL()

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, NewRequestError("创建 HTTP 请求失败", err)
	}

	// 设置请求头
	g.setRequestHeaders(httpReq)

	// 打印详细的请求日志（curl 格式）
	g.logRequest(httpReq, reqJSON)

	// 发送请求（使用支持重试的客户端）
	var httpResp *http.Response
	if g.retryableClient != nil {
		httpResp, err = g.retryableClient.Do(httpReq)
	} else {
		httpResp, err = g.client.Do(httpReq)
	}
	if err != nil {
		return nil, NewNetworkError("发送 HTTP 请求失败", err)
	}
	defer httpResp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, NewNetworkError("读取响应体失败", err)
	}

	// 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		return nil, g.handleErrorResponse(httpResp.StatusCode, respBody)
	}

	// 解析响应
	var azResp ResponsesResponse
	if err := json.Unmarshal(respBody, &azResp); err != nil {
		return nil, NewParseError("解析响应 JSON 失败", err)
	}

	// 转换为 Genkit ModelResponse
	modelResp, err := g.convertToModelResponse(&azResp)
	if err != nil {
		return nil, err
	}

	return modelResp, nil
}

// setRequestHeaders 设置 HTTP 请求头
func (g *ModelGenerator) setRequestHeaders(req *http.Request) {
	// 设置认证头（api-key）
	req.Header.Set("api-key", g.apiKey)

	// 设置内容类型
	req.Header.Set("Content-Type", "application/json")

	// 设置 User-Agent
	req.Header.Set("User-Agent", "genkit-azure-plugin/1.0")
}

// handleErrorResponse 处理错误响应
func (g *ModelGenerator) handleErrorResponse(statusCode int, body []byte) error {
	// 尝试解析错误响应
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return NewAPIError(
			fmt.Sprintf("%d", statusCode),
			errResp.Error.Message,
			errResp.Error,
		)
	}

	// 如果无法解析，返回通用错误
	return NewAPIError(
		fmt.Sprintf("%d", statusCode),
		fmt.Sprintf("API 请求失败: %s", string(body)),
		nil,
	)
}

// convertToModelResponse 将 Azure OpenAI 响应转换为 Genkit ModelResponse
func (g *ModelGenerator) convertToModelResponse(azResp *ResponsesResponse) (*ai.ModelResponse, error) {
	if len(azResp.Choices) == 0 {
		return nil, NewParseError("响应中没有选项", nil)
	}

	// 使用第一个选项
	choice := azResp.Choices[0]

	// 转换消息内容
	message, err := g.convertResponseMessage(&choice.Message)
	if err != nil {
		return nil, err
	}

	// 映射 finish_reason
	finishReason := mapFinishReason(choice.FinishReason)

	// 构建响应
	resp := &ai.ModelResponse{
		Message: message,
		Usage: &ai.GenerationUsage{
			InputTokens:  azResp.Usage.PromptTokens,
			OutputTokens: azResp.Usage.CompletionTokens,
			TotalTokens:  azResp.Usage.TotalTokens,
		},
		FinishReason: finishReason,
		Custom: map[string]any{
			"id":                 azResp.ID,
			"model":              azResp.Model,
			"created":            azResp.Created,
			"system_fingerprint": azResp.SystemFingerprint,
		},
	}

	return resp, nil
}

// convertResponseMessage 转换响应消息
func (g *ModelGenerator) convertResponseMessage(azMsg *Message) (*ai.Message, error) {
	msg := &ai.Message{
		Role: ai.RoleModel,
	}

	// 转换文本内容
	if azMsg.Content != nil {
		switch content := azMsg.Content.(type) {
		case string:
			if content != "" {
				msg.Content = append(msg.Content, ai.NewTextPart(content))
			}
		case []any:
			// 处理 ContentPart 数组
			for _, item := range content {
				if partMap, ok := item.(map[string]any); ok {
					if partType, ok := partMap["type"].(string); ok && partType == "text" {
						if text, ok := partMap["text"].(string); ok && text != "" {
							msg.Content = append(msg.Content, ai.NewTextPart(text))
						}
					}
				}
			}
		}
	}

	// 转换工具调用
	if len(azMsg.ToolCalls) > 0 {
		for _, toolCall := range azMsg.ToolCalls {
			// 解析参数 JSON
			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return nil, NewParseError(fmt.Sprintf("解析工具调用参数失败: %v", err), err)
			}

			msg.Content = append(msg.Content, ai.NewToolRequestPart(&ai.ToolRequest{
				Ref:   toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: args,
			}))
		}
	}

	return msg, nil
}

// mapFinishReason 映射 Azure OpenAI 的 finish_reason 到 Genkit FinishReason
func mapFinishReason(reason string) ai.FinishReason {
	switch reason {
	case "stop":
		return ai.FinishReasonStop
	case "length":
		return ai.FinishReasonLength
	case "content_filter":
		return ai.FinishReasonBlocked
	case "tool_calls":
		// 工具调用完成也视为正常停止
		return ai.FinishReasonStop
	default:
		return ai.FinishReasonOther
	}
}

// convertConfigToMap 将配置转换为 map[string]any
func convertConfigToMap(config any) (map[string]any, error) {
	if config == nil {
		return nil, nil
	}

	// 如果已经是 map，直接返回
	if m, ok := config.(map[string]any); ok {
		return m, nil
	}

	// 尝试通过 JSON 序列化/反序列化转换
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("反序列化配置失败: %w", err)
	}

	return result, nil
}

// applyConfig 将配置参数应用到请求
func applyConfig(req *ResponsesRequest, config map[string]any) {
	// Temperature
	if v, ok := config["temperature"].(float64); ok {
		req.Temperature = &v
	}

	// MaxTokens - 支持多种命名方式
	if v, ok := config["max_tokens"].(float64); ok {
		maxTokens := int(v)
		req.MaxTokens = &maxTokens
	} else if v, ok := config["maxTokens"].(float64); ok {
		maxTokens := int(v)
		req.MaxTokens = &maxTokens
	} else if v, ok := config["max_tokens"].(int); ok {
		req.MaxTokens = &v
	} else if v, ok := config["maxTokens"].(int); ok {
		req.MaxTokens = &v
	}

	// TopP
	if v, ok := config["top_p"].(float64); ok {
		req.TopP = &v
	} else if v, ok := config["topP"].(float64); ok {
		req.TopP = &v
	}

	// FrequencyPenalty
	if v, ok := config["frequency_penalty"].(float64); ok {
		req.FrequencyPenalty = &v
	} else if v, ok := config["frequencyPenalty"].(float64); ok {
		req.FrequencyPenalty = &v
	}

	// PresencePenalty
	if v, ok := config["presence_penalty"].(float64); ok {
		req.PresencePenalty = &v
	} else if v, ok := config["presencePenalty"].(float64); ok {
		req.PresencePenalty = &v
	}

	// Stop
	if v, ok := config["stop"].([]string); ok {
		req.Stop = v
	} else if v, ok := config["stop"].([]any); ok {
		// 处理 []any 类型
		var stop []string
		for _, s := range v {
			if str, ok := s.(string); ok {
				stop = append(stop, str)
			}
		}
		if len(stop) > 0 {
			req.Stop = stop
		}
	}

	// User
	if v, ok := config["user"].(string); ok {
		req.User = v
	}
}

// generateStreaming 执行流式生成请求
func (g *ModelGenerator) generateStreaming(ctx context.Context, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// 构建请求体（stream=true）
	reqBody, err := g.buildRequestBody(true)
	if err != nil {
		return nil, err
	}

	// 序列化请求体
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewRequestError("序列化请求体失败", err)
	}

	// 构建请求 URL
	url := g.buildRequestURL()

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, NewRequestError("创建 HTTP 请求失败", err)
	}

	// 设置请求头
	g.setRequestHeaders(httpReq)

	// 打印详细的请求日志（curl 格式）
	g.logRequest(httpReq, reqJSON)

	// 发送请求（使用支持重试的客户端）
	var httpResp *http.Response
	if g.retryableClient != nil {
		httpResp, err = g.retryableClient.Do(httpReq)
	} else {
		httpResp, err = g.client.Do(httpReq)
	}
	if err != nil {
		return nil, NewNetworkError("发送 HTTP 请求失败", err)
	}
	defer httpResp.Body.Close()

	// 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		// 读取错误响应
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, g.handleErrorResponse(httpResp.StatusCode, respBody)
	}

	// 解析流式响应
	return g.parseStreamingResponse(ctx, httpResp.Body, cb)
}

// parseStreamingResponse 解析流式响应
// 支持 Azure Responses API 的 SSE 事件格式
func (g *ModelGenerator) parseStreamingResponse(ctx context.Context, body io.Reader, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	if g.enableDebugLog {
		fmt.Println("\n" + repeatString("=", 80))
		fmt.Println("开始解析流式响应")
		fmt.Println(repeatString("=", 80))
		if cb != nil {
			fmt.Println("回调函数: 已提供")
		} else {
			fmt.Println("回调函数: nil（警告！）")
		}
	}

	// 用于聚合最终响应
	var aggregatedContent string
	var aggregatedToolCalls []ToolCall
	var responseID string
	var model string
	var created int64
	var usage *Usage
	var eventCount int

	// 创建 SSE 解析器
	scanner := newSSEScanner(body)

	// 逐行读取流式数据
	for scanner.Scan() {
		line := scanner.Text()

		// 记录原始 SSE 行（用于调试）
		if g.enableDebugLog {
			fmt.Printf("[原始SSE] %s\n", line)
		}

		// 跳过空行和注释
		if line == "" || line[0] == ':' {
			continue
		}

		// 解析 SSE 事件行
		// Azure Responses API 使用 "event: " 和 "data: " 格式
		if len(line) > 6 && line[:6] == "event:" {
			// 事件类型行，跳过（我们从 data 中获取类型）
			continue
		}

		// 解析 SSE 数据行
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]

			// 检查结束标记
			if data == "[DONE]" {
				if g.enableDebugLog {
					fmt.Println("[SSE] 收到结束标记 [DONE]")
				}
				break
			}

			// 记录原始 JSON 数据
			if g.enableDebugLog {
				fmt.Printf("[原始JSON] %s\n", data)
			}

			// 解析 JSON 数据块
			var event ResponsesStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				if g.enableDebugLog {
					fmt.Printf("[错误] 解析事件失败: %v\n", err)
					fmt.Printf("[错误] 原始数据: %s\n", data)
				}
				return nil, NewParseError(fmt.Sprintf("解析流式数据块失败: %v", err), err)
			}

			eventCount++
			if g.enableDebugLog {
				fmt.Printf("\n[事件 #%d] 类型: %s\n", eventCount, event.Type)
				// 打印事件的所有字段
				if event.ItemID != "" {
					fmt.Printf("  ItemID: %s\n", event.ItemID)
				}
				if event.OutputIndex > 0 {
					fmt.Printf("  OutputIndex: %d\n", event.OutputIndex)
				}
				if event.ContentIndex > 0 {
					fmt.Printf("  ContentIndex: %d\n", event.ContentIndex)
				}
				if len(event.Delta) > 0 {
					fmt.Printf("  Delta (raw): %s\n", string(event.Delta))
				}
			}

			// 处理不同类型的事件
			switch event.Type {
			case "response.created":
				// 响应创建事件，保存元数据
				if event.Response != nil {
					responseID = event.Response.ID
					model = event.Response.Model
					created = event.Response.Created
				}

			case "response.in_progress":
				// 响应进行中事件，暂不处理
				continue

			case "response.output_item.added":
				// 输出项添加事件
				if g.enableDebugLog {
					fmt.Printf("收到 response.output_item.added 事件: output_index=%d\n", event.OutputIndex)
				}
				continue

			case "response.content_part.added":
				// 内容部分添加事件
				if g.enableDebugLog {
					fmt.Printf("收到 response.content_part.added 事件: output_index=%d, content_index=%d\n",
						event.OutputIndex, event.ContentIndex)
				}
				continue

			case "response.content_part.delta":
				// 内容部分增量事件
				if g.enableDebugLog {
					fmt.Printf("收到 response.content_part.delta 事件: item_id=%s, output_index=%d, content_index=%d\n",
						event.ItemID, event.OutputIndex, event.ContentIndex)
				}

				// 解析 delta 字段（可能是字符串或结构体）
				if len(event.Delta) > 0 {
					// 尝试解析为 ResponseDelta 结构体
					var delta ResponseDelta
					if err := json.Unmarshal(event.Delta, &delta); err == nil {
						// 成功解析为结构体
						if delta.Text != "" {
							aggregatedContent += delta.Text

							if g.enableDebugLog {
								fmt.Printf("  Delta文本长度=%d, 聚合总长度=%d\n", len(delta.Text), len(aggregatedContent))
								fmt.Printf("  准备调用回调函数，cb是否为nil: %v\n", cb == nil)
							}

							// 调用回调函数实时发送文本
							if cb != nil {
								chunkResp := &ai.ModelResponseChunk{
									Content: []*ai.Part{ai.NewTextPart(delta.Text)},
								}
								if g.enableDebugLog {
									fmt.Printf("  [Azure插件] 调用回调函数，文本长度=%d\n", len(delta.Text))
								}
								if err := cb(ctx, chunkResp); err != nil {
									return nil, fmt.Errorf("回调函数执行失败: %w", err)
								}
								if g.enableDebugLog {
									fmt.Printf("  [Azure插件] 回调函数执行成功\n")
								}
							} else {
								if g.enableDebugLog {
									fmt.Printf("  [警告] 回调函数为 nil，无法发送增量文本\n")
								}
							}
						}
					} else {
						// 尝试解析为字符串
						var deltaStr string
						if err := json.Unmarshal(event.Delta, &deltaStr); err == nil {
							if deltaStr != "" {
								aggregatedContent += deltaStr

								if g.enableDebugLog {
									fmt.Printf("  Delta字符串长度=%d, 聚合总长度=%d\n", len(deltaStr), len(aggregatedContent))
									fmt.Printf("  准备调用回调函数，cb是否为nil: %v\n", cb == nil)
								}

								// 调用回调函数实时发送文本
								if cb != nil {
									chunkResp := &ai.ModelResponseChunk{
										Content: []*ai.Part{ai.NewTextPart(deltaStr)},
									}
									if g.enableDebugLog {
										fmt.Printf("  [Azure插件] 调用回调函数，文本长度=%d\n", len(deltaStr))
									}
									if err := cb(ctx, chunkResp); err != nil {
										return nil, fmt.Errorf("回调函数执行失败: %w", err)
									}
									if g.enableDebugLog {
										fmt.Printf("  [Azure插件] 回调函数执行成功\n")
									}
								} else {
									if g.enableDebugLog {
										fmt.Printf("  [警告] 回调函数为 nil，无法发送增量文本\n")
									}
								}
							}
						} else {
							if g.enableDebugLog {
								fmt.Printf("  警告：无法解析 delta 字段: %s\n", string(event.Delta))
							}
						}
					}
				}

			case "response.content_part.done":
				// 内容部分完成事件
				// 根据官方文档，Responses API 在流式模式下直接发送完整文本，没有 delta 事件
				if g.enableDebugLog {
					fmt.Printf("收到 response.content_part.done 事件: item_id=%s, output_index=%d, content_index=%d\n",
						event.ItemID, event.OutputIndex, event.ContentIndex)
					if event.Part != nil {
						fmt.Printf("  Part.Type=%s, Text长度=%d\n", event.Part.Type, len(event.Part.Text))
					}
				}

				if event.Part != nil && event.Part.Type == "output_text" && event.Part.Text != "" {
					// 聚合文本内容
					aggregatedContent += event.Part.Text

					if g.enableDebugLog {
						fmt.Printf("  聚合文本，当前总长度=%d\n", len(aggregatedContent))
						fmt.Printf("  准备调用回调函数，cb是否为nil: %v\n", cb == nil)
						fmt.Printf("  文本内容: %s\n", event.Part.Text)
					}

					// 调用回调函数实时发送文本
					if cb != nil {
						chunkResp := &ai.ModelResponseChunk{
							Content: []*ai.Part{ai.NewTextPart(event.Part.Text)},
						}
						if g.enableDebugLog {
							fmt.Printf("  [Azure插件] 调用回调函数，文本长度=%d\n", len(event.Part.Text))
						}
						if err := cb(ctx, chunkResp); err != nil {
							return nil, fmt.Errorf("回调函数执行失败: %w", err)
						}
						if g.enableDebugLog {
							fmt.Printf("  [Azure插件] 回调函数执行成功\n")
						}
					} else {
						if g.enableDebugLog {
							fmt.Printf("  [警告] 回调函数为 nil，无法发送文本块\n")
						}
					}
				} else {
					if g.enableDebugLog {
						if event.Part == nil {
							fmt.Printf("  跳过：Part 为 nil\n")
						} else if event.Part.Type != "output_text" {
							fmt.Printf("  跳过：Part.Type=%s (不是 output_text)\n", event.Part.Type)
						} else if event.Part.Text == "" {
							fmt.Printf("  跳过：Text 为空\n")
						}
					}
				}

				// 处理函数调用
				if event.Part != nil && event.Part.Type == "function_call" && event.Part.FunctionCall != nil {
					toolCall := ToolCall{
						ID:   fmt.Sprintf("call_%s_%d", event.ItemID, event.ContentIndex),
						Type: "function",
						Function: FunctionCall{
							Name:      event.Part.FunctionCall.Name,
							Arguments: event.Part.FunctionCall.Arguments,
						},
					}
					aggregatedToolCalls = append(aggregatedToolCalls, toolCall)

					// 为工具调用创建回调
					if cb != nil {
						var args map[string]any
						if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
							return nil, NewParseError(fmt.Sprintf("解析工具调用参数失败: %v", err), err)
						}

						chunkResp := &ai.ModelResponseChunk{
							Content: []*ai.Part{
								ai.NewToolRequestPart(&ai.ToolRequest{
									Ref:   toolCall.ID,
									Name:  toolCall.Function.Name,
									Input: args,
								}),
							},
						}
						if err := cb(ctx, chunkResp); err != nil {
							return nil, fmt.Errorf("回调函数执行失败: %w", err)
						}
					}
				}

			case "response.output_text.done":
				// 输出文本完成事件（可选的额外事件）
				// 通常 content_part.done 已经包含了文本，这里可以跳过
				continue

			case "response.output_item.done":
				// 输出项完成事件，暂不处理
				continue

			case "response.completed":
				// 响应完成事件（包含完整的 output 和 usage）
				if g.enableDebugLog {
					fmt.Printf("收到 response.completed 事件\n")
				}

				if event.Response != nil {
					// 更新元数据
					if responseID == "" {
						responseID = event.Response.ID
					}
					if model == "" {
						model = event.Response.Model
					}
					if created == 0 {
						created = event.Response.Created
					}

					// 获取 token 使用统计
					if event.Response.Usage != nil {
						usage = event.Response.Usage
						if g.enableDebugLog {
							fmt.Printf("  Usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d\n",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
						}
					}

					if g.enableDebugLog {
						fmt.Printf("  Response ID=%s, Model=%s\n", responseID, model)
					}
				}

			case "response.failed":
				// 响应失败事件
				if event.Response != nil && event.Response.Error != nil {
					return nil, NewAPIError(
						event.Response.Error.Code,
						event.Response.Error.Message,
						event.Response.Error,
					)
				}
				return nil, NewAPIError("response_failed", "响应生成失败", nil)

			case "response.incomplete":
				// 响应不完整事件
				if event.Response != nil && event.Response.IncompleteDetails != nil {
					return nil, NewAPIError(
						"response_incomplete",
						fmt.Sprintf("响应不完整: %s", event.Response.IncompleteDetails.Reason),
						event.Response.IncompleteDetails,
					)
				}
				return nil, NewAPIError("response_incomplete", "响应不完整", nil)

			case "response.reasoning_summary_part.done":
				// 推理摘要完成事件，暂不处理
				continue

			default:
				// 未知事件类型，记录但继续处理
				if g.enableDebugLog {
					fmt.Printf("未知的事件类型: %s\n", event.Type)
				}
			}
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		return nil, NewNetworkError("读取流式响应失败", err)
	}

	if g.enableDebugLog {
		fmt.Println("\n" + repeatString("=", 80))
		fmt.Printf("流式响应解析完成\n")
		fmt.Printf("总事件数: %d\n", eventCount)
		fmt.Printf("聚合文本长度: %d\n", len(aggregatedContent))
		fmt.Printf("工具调用数: %d\n", len(aggregatedToolCalls))
		fmt.Println(repeatString("=", 80) + "\n")
	}

	// 构建最终响应
	message := &ai.Message{
		Role: ai.RoleModel,
	}

	// 添加文本内容
	if aggregatedContent != "" {
		message.Content = append(message.Content, ai.NewTextPart(aggregatedContent))
	}

	// 添加工具调用
	for _, toolCall := range aggregatedToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return nil, NewParseError(fmt.Sprintf("解析工具调用参数失败: %v", err), err)
		}

		message.Content = append(message.Content, ai.NewToolRequestPart(&ai.ToolRequest{
			Ref:   toolCall.ID,
			Name:  toolCall.Function.Name,
			Input: args,
		}))
	}

	// 构建响应
	resp := &ai.ModelResponse{
		Message:      message,
		FinishReason: ai.FinishReasonStop, // Azure Responses API 不提供 finish_reason
		Custom: map[string]any{
			"id":      responseID,
			"model":   model,
			"created": created,
		},
	}

	// 添加 token 使用统计
	if usage != nil {
		resp.Usage = &ai.GenerationUsage{
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
		}
	}

	return resp, nil
}

// parseStreamingResponseLegacy 解析传统格式的流式响应（向后兼容）
func (g *ModelGenerator) parseStreamingResponseLegacy(ctx context.Context, body io.Reader, cb func(context.Context, *ai.ModelResponseChunk) error, firstChunk *StreamChunk, scanner *sseScanner) (*ai.ModelResponse, error) {
	// 用于聚合最终响应
	var aggregatedContent string
	var aggregatedToolCalls []ToolCall
	var finishReason string
	var responseID string
	var model string
	var created int64
	var systemFingerprint string

	// 处理第一个块
	if firstChunk != nil {
		responseID = firstChunk.ID
		model = firstChunk.Model
		created = firstChunk.Created
		systemFingerprint = firstChunk.SystemFingerprint

		if len(firstChunk.Choices) > 0 {
			choice := firstChunk.Choices[0]

			if choice.Delta.Content != "" {
				aggregatedContent += choice.Delta.Content

				if cb != nil {
					chunkResp := &ai.ModelResponseChunk{
						Content: []*ai.Part{ai.NewTextPart(choice.Delta.Content)},
					}
					if err := cb(ctx, chunkResp); err != nil {
						return nil, fmt.Errorf("回调函数执行失败: %w", err)
					}
				}
			}

			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}

	// 继续读取剩余的数据
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || line[0] == ':' {
			continue
		}

		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]

			if data == "[DONE]" {
				break
			}

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return nil, NewParseError(fmt.Sprintf("解析流式数据块失败: %v", err), err)
			}

			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]

				if choice.Delta.Content != "" {
					aggregatedContent += choice.Delta.Content

					if cb != nil {
						chunkResp := &ai.ModelResponseChunk{
							Content: []*ai.Part{ai.NewTextPart(choice.Delta.Content)},
						}
						if err := cb(ctx, chunkResp); err != nil {
							return nil, fmt.Errorf("回调函数执行失败: %w", err)
						}
					}
				}

				if len(choice.Delta.ToolCalls) > 0 {
					aggregatedToolCalls = append(aggregatedToolCalls, choice.Delta.ToolCalls...)

					if cb != nil {
						for _, toolCall := range choice.Delta.ToolCalls {
							var args map[string]any
							if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
								return nil, NewParseError(fmt.Sprintf("解析工具调用参数失败: %v", err), err)
							}

							chunkResp := &ai.ModelResponseChunk{
								Content: []*ai.Part{
									ai.NewToolRequestPart(&ai.ToolRequest{
										Ref:   toolCall.ID,
										Name:  toolCall.Function.Name,
										Input: args,
									}),
								},
							}
							if err := cb(ctx, chunkResp); err != nil {
								return nil, fmt.Errorf("回调函数执行失败: %w", err)
							}
						}
					}
				}

				if choice.FinishReason != "" {
					finishReason = choice.FinishReason
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewNetworkError("读取流式响应失败", err)
	}

	message := &ai.Message{
		Role: ai.RoleModel,
	}

	if aggregatedContent != "" {
		message.Content = append(message.Content, ai.NewTextPart(aggregatedContent))
	}

	for _, toolCall := range aggregatedToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return nil, NewParseError(fmt.Sprintf("解析工具调用参数失败: %v", err), err)
		}

		message.Content = append(message.Content, ai.NewToolRequestPart(&ai.ToolRequest{
			Ref:   toolCall.ID,
			Name:  toolCall.Function.Name,
			Input: args,
		}))
	}

	resp := &ai.ModelResponse{
		Message:      message,
		FinishReason: mapFinishReason(finishReason),
		Custom: map[string]any{
			"id":                 responseID,
			"model":              model,
			"created":            created,
			"system_fingerprint": systemFingerprint,
		},
	}

	return resp, nil
}

// logRequest 打印详细的 HTTP 请求日志（curl 格式）
func (g *ModelGenerator) logRequest(req *http.Request, bodyBytes []byte) {
	// 只在启用调试日志时打印
	if !g.enableDebugLog {
		return
	}

	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("Azure OpenAI API 请求详情")
	fmt.Println(repeatString("=", 80))
	fmt.Printf("方法: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Println("\n请求头:")
	for key, values := range req.Header {
		for _, value := range values {
			// 脱敏 api-key header
			if key == "api-key" {
				value = maskAPIKey(value)
			}
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(bodyBytes) > 0 {
		fmt.Println("\n请求体:")
		// 格式化 JSON
		fmt.Println(formatJSON(string(bodyBytes)))
	}

	fmt.Println("\n等效的 curl 命令:")
	fmt.Println(buildCurlCommand(req, bodyBytes))
	fmt.Println(repeatString("=", 80) + "\n")
}

// buildCurlCommand 构建等效的 curl 命令
func buildCurlCommand(req *http.Request, bodyBytes []byte) string {
	var cmd string
	cmd += fmt.Sprintf("curl -X %s '%s'", req.Method, req.URL.String())

	// 添加 headers
	for key, values := range req.Header {
		for _, value := range values {
			// 脱敏 api-key header
			if key == "api-key" {
				value = maskAPIKey(value)
			}
			cmd += fmt.Sprintf(" \\\n  -H '%s: %s'", key, value)
		}
	}

	// 添加 body
	if len(bodyBytes) > 0 {
		// 转义单引号
		body := string(bodyBytes)
		body = replaceString(body, "'", "'\\''")
		cmd += fmt.Sprintf(" \\\n  -d '%s'", body)
	}

	return cmd
}

// maskAPIKey 脱敏 API Key
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 20 {
		return "****"
	}
	return apiKey[:10] + "****" + apiKey[len(apiKey)-6:]
}

// formatJSON 格式化 JSON 字符串
func formatJSON(jsonStr string) string {
	var result string
	indent := 0
	inString := false
	escape := false

	for i, char := range jsonStr {
		if escape {
			result += string(char)
			escape = false
			continue
		}

		switch char {
		case '\\':
			result += string(char)
			escape = true
		case '"':
			result += string(char)
			inString = !inString
		case '{', '[':
			result += string(char)
			if !inString {
				indent++
				if i+1 < len(jsonStr) && jsonStr[i+1] != '}' && jsonStr[i+1] != ']' {
					result += "\n" + repeatString("  ", indent)
				}
			}
		case '}', ']':
			if !inString {
				indent--
				if i > 0 && jsonStr[i-1] != '{' && jsonStr[i-1] != '[' {
					result += "\n" + repeatString("  ", indent)
				}
			}
			result += string(char)
		case ',':
			result += string(char)
			if !inString {
				result += "\n" + repeatString("  ", indent)
			}
		case ':':
			result += string(char)
			if !inString {
				result += " "
			}
		default:
			result += string(char)
		}
	}

	return result
}

// repeatString 重复字符串 n 次
func repeatString(s string, n int) string {
	var result string
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// replaceString 替换字符串中的所有匹配项
func replaceString(s, old, new string) string {
	result := ""
	for {
		idx := indexString(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// indexString 查找子字符串的位置
func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// sseScanner 是一个用于解析 SSE 格式的扫描器
type sseScanner struct {
	reader  io.Reader
	buffer  []byte
	err     error
	current string
}

// newSSEScanner 创建一个新的 SSE 扫描器
func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{
		reader: r,
		buffer: make([]byte, 0, 4096),
	}
}

// Scan 读取下一行
func (s *sseScanner) Scan() bool {
	if s.err != nil {
		return false
	}

	// 读取数据直到找到换行符
	for {
		// 在缓冲区中查找换行符
		for i, b := range s.buffer {
			if b == '\n' {
				// 找到换行符，提取当前行
				s.current = string(s.buffer[:i])
				// 移除已处理的数据（包括换行符）
				s.buffer = s.buffer[i+1:]
				return true
			}
		}

		// 缓冲区中没有换行符，读取更多数据
		tmp := make([]byte, 1024)
		n, err := s.reader.Read(tmp)
		if n > 0 {
			s.buffer = append(s.buffer, tmp[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				// 到达文件末尾，如果缓冲区还有数据，返回它
				if len(s.buffer) > 0 {
					s.current = string(s.buffer)
					s.buffer = nil
					return true
				}
			}
			s.err = err
			return false
		}
	}
}

// Text 返回当前行的文本
func (s *sseScanner) Text() string {
	return s.current
}

// Err 返回扫描过程中的错误
func (s *sseScanner) Err() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}
