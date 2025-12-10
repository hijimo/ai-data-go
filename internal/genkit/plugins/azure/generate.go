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
	_ = cb // 流式模式将在后续任务中实现

	// TODO: 实现实际的 HTTP 请求和响应处理
	// 这将在后续任务中实现
	return nil, NewRequestError("生成逻辑尚未实现", nil)
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
