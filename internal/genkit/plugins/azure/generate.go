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
func (g *ModelGenerator) WithConfig(config any) *ModelGenerator {
	// TODO: 实现配置处理逻辑
	return g
}

// Generate 执行生成请求
func (g *ModelGenerator) Generate(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// TODO: 实现生成逻辑
	return nil, NewRequestError("not implemented", nil)
}
