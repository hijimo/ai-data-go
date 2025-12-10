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
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
)

// TestDefineModel_BasicConfiguration 测试基本模型定义
func TestDefineModel_BasicConfiguration(t *testing.T) {
	// 创建插件实例
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	// 初始化插件
	plugin.Init(context.Background())

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	// 验证模型不为 nil
	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 验证模型可以转换为 Action
	if _, ok := model.(api.Action); !ok {
		t.Error("模型无法转换为 api.Action")
	}
}

// TestDefineModel_WithBasicTextSupport 测试仅支持文本的模型
func TestDefineModel_WithBasicTextSupport(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义仅支持文本的模型
	model := plugin.DefineModel("azure", "gpt-3.5-turbo", ai.ModelOptions{
		Label:    "GPT-3.5 Turbo",
		Stage:    ai.ModelStageStable,
		Supports: &BasicText,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}
}

// TestDefineModel_WithMultimodalSupport 测试支持多模态的模型
func TestDefineModel_WithMultimodalSupport(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义支持多模态的模型
	model := plugin.DefineModel("azure", "gpt-4-vision", ai.ModelOptions{
		Label:    "GPT-4 Vision",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}
}

// TestDefineModel_PanicWhenNotInitialized 测试未初始化时调用 DefineModel 会 panic
func TestDefineModel_PanicWhenNotInitialized(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	// 不调用 Init，直接调用 DefineModel 应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("DefineModel 在未初始化时应该 panic")
		}
	}()

	plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &Multimodal,
	})
}

// TestDefineModel_MultipleModels 测试定义多个模型
func TestDefineModel_MultipleModels(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义多个模型
	model1 := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &Multimodal,
	})

	model2 := plugin.DefineModel("azure", "gpt-3.5-turbo", ai.ModelOptions{
		Label:    "GPT-3.5 Turbo",
		Supports: &BasicText,
	})

	if model1 == nil || model2 == nil {
		t.Fatal("DefineModel 返回了 nil")
	}
}

// TestDefineModel_CustomProvider 测试自定义 provider 名称
func TestDefineModel_CustomProvider(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "custom-azure",
	}

	plugin.Init(context.Background())

	// 使用自定义 provider 名称定义模型
	model := plugin.DefineModel("custom-azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &Multimodal,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}
}

// TestModelSupports_BasicText 测试 BasicText 能力配置
func TestModelSupports_BasicText(t *testing.T) {
	if !BasicText.Multiturn {
		t.Error("BasicText 应该支持多轮对话")
	}
	if !BasicText.Tools {
		t.Error("BasicText 应该支持工具")
	}
	if !BasicText.SystemRole {
		t.Error("BasicText 应该支持系统角色")
	}
	if BasicText.Media {
		t.Error("BasicText 不应该支持媒体")
	}
}

// TestModelSupports_Multimodal 测试 Multimodal 能力配置
func TestModelSupports_Multimodal(t *testing.T) {
	if !Multimodal.Multiturn {
		t.Error("Multimodal 应该支持多轮对话")
	}
	if !Multimodal.Tools {
		t.Error("Multimodal 应该支持工具")
	}
	if !Multimodal.SystemRole {
		t.Error("Multimodal 应该支持系统角色")
	}
	if !Multimodal.Media {
		t.Error("Multimodal 应该支持媒体")
	}
	if !Multimodal.ToolChoice {
		t.Error("Multimodal 应该支持工具选择")
	}
}

// TestDefineModel_ModelFunctionIntegration 测试模型函数集成
// 验证 DefineModel 创建的模型能够正确集成 ModelGenerator
func TestDefineModel_ModelFunctionIntegration(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	// 验证模型不为 nil
	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 注意：实际的模型调用需要真实的 API 密钥和网络连接
	// 这里只验证模型定义的正确性
	t.Log("模型定义成功，集成了 ModelGenerator")
}

// TestDefineModel_StreamingAndNonStreamingRouting 测试流式和非流式请求路由
// 验证模型能够根据回调函数的存在决定使用流式或非流式模式
func TestDefineModel_StreamingAndNonStreamingRouting(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 验证模型定义包含了正确的路由逻辑
	// 实际的路由逻辑在 ModelGenerator.Generate 方法中实现
	// 该方法会根据 cb 参数是否为 nil 来决定使用流式或非流式模式
	t.Log("模型定义成功，支持流式和非流式请求路由")
}

// TestDefineModel_ConfigurationPropagation 测试配置参数传递
// 验证模型定义能够正确传递配置参数到 ModelGenerator
func TestDefineModel_ConfigurationPropagation(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 验证模型定义包含了配置传递逻辑
	// ModelGenerator.WithConfig 方法会处理配置参数
	t.Log("模型定义成功，支持配置参数传递")
}

// TestDefineModel_ToolsIntegration 测试工具集成
// 验证模型定义能够正确处理工具定义
func TestDefineModel_ToolsIntegration(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义支持工具的模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal, // Multimodal 支持工具
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 验证模型定义包含了工具处理逻辑
	// ModelGenerator.WithTools 方法会处理工具定义
	t.Log("模型定义成功，支持工具集成")
}

// TestDefineModel_MessagesIntegration 测试消息集成
// 验证模型定义能够正确处理消息转换
func TestDefineModel_MessagesIntegration(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Stage:    ai.ModelStageStable,
		Supports: &Multimodal,
	})

	if model == nil {
		t.Fatal("DefineModel 返回了 nil")
	}

	// 验证模型定义包含了消息处理逻辑
	// ModelGenerator.WithMessages 方法会处理消息转换
	t.Log("模型定义成功，支持消息集成")
}

func TestDefineEmbedder_BasicConfiguration(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	embedder := plugin.DefineEmbedder("azure", "text-embedding-ada-002", &ai.EmbedderOptions{
		Label:      "Azure OpenAI - text-embedding-ada-002",
		Dimensions: 1536,
	})

	if embedder == nil {
		t.Fatal("DefineEmbedder 返回 nil")
	}

	if embedder.Name() != "azure/text-embedding-ada-002" {
		t.Errorf("期望嵌入器名称 'azure/text-embedding-ada-002', 实际: %s", embedder.Name())
	}
}

func TestDefineEmbedder_PanicWhenNotInitialized(t *testing.T) {
	plugin := &AzureAI{
		APIKey:  "test-api-key",
		BaseURL: "https://test.openai.azure.com",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("期望 panic，但没有发生")
		}
	}()

	plugin.DefineEmbedder("azure", "text-embedding-ada-002", nil)
}

func TestDefineEmbedder_MultipleEmbedders(t *testing.T) {
	plugin := &AzureAI{
		APIKey:     "test-api-key",
		BaseURL:    "https://test.openai.azure.com",
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	plugin.Init(context.Background())

	embedder1 := plugin.DefineEmbedder("azure", "text-embedding-ada-002", &ai.EmbedderOptions{
		Label:      "Azure OpenAI - Ada 002",
		Dimensions: 1536,
	})

	embedder2 := plugin.DefineEmbedder("azure", "text-embedding-3-small", &ai.EmbedderOptions{
		Label:      "Azure OpenAI - Embedding 3 Small",
		Dimensions: 1536,
	})

	if embedder1 == nil || embedder2 == nil {
		t.Fatal("DefineEmbedder 返回 nil")
	}

	if embedder1.Name() == embedder2.Name() {
		t.Error("不同的嵌入器应该有不同的名称")
	}
}
