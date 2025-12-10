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
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
)

var (
	// BasicText 描述仅支持文本的模型能力
	BasicText = ai.ModelSupports{
		Multiturn:  true,
		Tools:      true,
		SystemRole: true,
		Media:      false,
	}

	// Multimodal 描述支持多模态的模型能力
	Multimodal = ai.ModelSupports{
		Multiturn:  true,
		Tools:      true,
		SystemRole: true,
		Media:      true,
		ToolChoice: true,
	}
)

// AzureAI 是一个提供 Azure OpenAI 服务集成的 Genkit 插件
// 它使用 Azure OpenAI 的 Responses API (/openai/responses) 而非传统的 chat/completions 端点
type AzureAI struct {
	// mu 保护并发访问客户端和初始化状态
	mu sync.Mutex

	// initted 跟踪插件是否已初始化
	initted bool

	// httpClient 用于发送 HTTP 请求的客户端
	httpClient *http.Client

	// APIKey Azure OpenAI API 密钥
	// 必需：用于认证
	APIKey string

	// BaseURL Azure OpenAI 资源的基础 URL
	// 必需：格式为 https://{resource-name}.openai.azure.com
	BaseURL string

	// APIVersion Azure OpenAI API 版本
	// 可选：默认为 "2025-04-01-preview"
	APIVersion string

	// Provider 插件的唯一标识符
	// 用作模型名称的前缀（例如 "azure/gpt-4"）
	// 应为小写并与 Name() 方法匹配
	Provider string
}

// Init 实现 genkit.Plugin 接口
// 初始化插件并创建 HTTP 客户端
func (a *AzureAI) Init(ctx context.Context) []api.Action {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initted {
		panic("azure.Init already called")
	}

	// 验证必需配置
	if a.APIKey == "" {
		panic("azure: APIKey is required")
	}
	if a.BaseURL == "" {
		panic("azure: BaseURL is required")
	}

	// 设置默认 API 版本
	if a.APIVersion == "" {
		a.APIVersion = DefaultAPIVersion
	}

	// 设置默认 Provider
	if a.Provider == "" {
		a.Provider = "azure"
	}

	// 创建 HTTP 客户端
	a.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	a.initted = true
	return []api.Action{}
}

// Name 实现 genkit.Plugin 接口
// 返回插件名称
func (a *AzureAI) Name() string {
	return a.Provider
}

// DefineModel 在注册表中定义一个模型
func (a *AzureAI) DefineModel(provider, id string, opts ai.ModelOptions) ai.Model {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initted {
		panic("AzureAI.Init not called")
	}

	return ai.NewModel(api.NewName(provider, id), &opts, func(
		ctx context.Context,
		input *ai.ModelRequest,
		cb func(context.Context, *ai.ModelResponseChunk) error,
	) (*ai.ModelResponse, error) {
		// 使用输入配置响应生成器
		generator := NewModelGenerator(a.httpClient, a.BaseURL, a.APIKey, a.APIVersion, id).
			WithMessages(input.Messages).
			WithConfig(input.Config).
			WithTools(input.Tools)

		// 生成响应
		resp, err := generator.Generate(ctx, input, cb)
		if err != nil {
			return nil, err
		}

		return resp, nil
	})
}

// DefineEmbedder 定义一个嵌入器
func (a *AzureAI) DefineEmbedder(provider, name string, embedOpts *ai.EmbedderOptions) ai.Embedder {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initted {
		panic("AzureAI.Init not called")
	}

	return ai.NewEmbedder(api.NewName(provider, name), embedOpts, func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
		return generateEmbeddings(ctx, a.httpClient, a.BaseURL, a.APIKey, a.APIVersion, name, req)
	})
}

// IsDefinedEmbedder 报告指定的嵌入器是否由此插件定义
func (a *AzureAI) IsDefinedEmbedder(g *genkit.Genkit, name string) bool {
	return genkit.LookupEmbedder(g, name) != nil
}

// Embedder 返回指定名称的嵌入器
// 如果嵌入器未定义，则返回 nil
func (a *AzureAI) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return genkit.LookupEmbedder(g, name)
}

// Model 返回指定名称的模型
// 如果模型未定义，则返回 nil
func (a *AzureAI) Model(g *genkit.Genkit, name string) ai.Model {
	return genkit.LookupModel(g, name)
}

// IsDefinedModel 报告指定的模型是否由此插件定义
func (a *AzureAI) IsDefinedModel(g *genkit.Genkit, name string) bool {
	return genkit.LookupModel(g, name) != nil
}

// ListActions 列出插件提供的所有操作
func (a *AzureAI) ListActions(ctx context.Context) []api.ActionDesc {
	// Azure OpenAI 不提供模型列表 API，需要手动定义模型
	return []api.ActionDesc{}
}

// ResolveAction 解析指定类型和名称的操作
func (a *AzureAI) ResolveAction(atype api.ActionType, name string) api.Action {
	switch atype {
	case api.ActionTypeModel:
		if model := a.DefineModel(a.Provider, name, ai.ModelOptions{
			Label:    fmt.Sprintf("%s - %s", a.Provider, name),
			Stage:    ai.ModelStageStable,
			Versions: []string{},
			Supports: &Multimodal,
		}); model != nil {
			return model.(api.Action)
		}
	}

	return nil
}
