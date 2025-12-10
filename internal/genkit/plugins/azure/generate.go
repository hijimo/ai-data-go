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
	client     *http.Client
	baseURL    string
	apiKey     string
	apiVersion string
	modelName  string

	// 请求构建
	messages   []Message
	tools      []Tool
	toolChoice any
	config     map[string]any

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

	// 发送请求
	httpResp, err := g.client.Do(httpReq)
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

	// 发送请求
	httpResp, err := g.client.Do(httpReq)
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
func (g *ModelGenerator) parseStreamingResponse(ctx context.Context, body io.Reader, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// 用于聚合最终响应
	var aggregatedContent string
	var aggregatedToolCalls []ToolCall
	var finishReason string
	var responseID string
	var model string
	var created int64
	var systemFingerprint string

	// 创建 SSE 解析器
	scanner := newSSEScanner(body)

	// 逐行读取流式数据
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		if line == "" || line[0] == ':' {
			continue
		}

		// 解析 SSE 数据行
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]

			// 检查结束标记
			if data == "[DONE]" {
				break
			}

			// 解析 JSON 数据块
			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return nil, NewParseError(fmt.Sprintf("解析流式数据块失败: %v", err), err)
			}

			// 保存元数据（从第一个块）
			if responseID == "" {
				responseID = chunk.ID
				model = chunk.Model
				created = chunk.Created
				systemFingerprint = chunk.SystemFingerprint
			}

			// 处理每个选项
			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]

				// 聚合内容
				if choice.Delta.Content != "" {
					aggregatedContent += choice.Delta.Content

					// 调用回调函数
					if cb != nil {
						chunkResp := &ai.ModelResponseChunk{
							Content: []*ai.Part{ai.NewTextPart(choice.Delta.Content)},
						}
						if err := cb(ctx, chunkResp); err != nil {
							return nil, fmt.Errorf("回调函数执行失败: %w", err)
						}
					}
				}

				// 聚合工具调用
				if len(choice.Delta.ToolCalls) > 0 {
					aggregatedToolCalls = append(aggregatedToolCalls, choice.Delta.ToolCalls...)

					// 为工具调用创建回调
					if cb != nil {
						for _, toolCall := range choice.Delta.ToolCalls {
							// 解析参数
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

				// 保存完成原因
				if choice.FinishReason != "" {
					finishReason = choice.FinishReason
				}
			}
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		return nil, NewNetworkError("读取流式响应失败", err)
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
