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

package azure_test

import (
	"context"
	"fmt"

	"genkit-ai-service/internal/genkit/plugins/azure"
)

// ExampleAzureAI_Init 演示如何初始化 Azure AI 插件
func ExampleAzureAI_Init() {
	// 创建 Azure AI 插件实例
	plugin := &azure.AzureAI{
		APIKey:     "your-api-key",
		BaseURL:    "https://your-resource.openai.azure.com",
		APIVersion: "2025-04-01-preview", // 可选，默认为此版本
		Provider:   "azure",              // 可选，默认为 "azure"
	}

	// 初始化插件
	ctx := context.Background()
	plugin.Init(ctx)

	// 获取插件名称
	fmt.Println(plugin.Name())
	// Output: azure
}

// ExampleAzureAI_Init_minimal 演示使用最少配置初始化插件
func ExampleAzureAI_Init_minimal() {
	// 只提供必需的配置
	plugin := &azure.AzureAI{
		APIKey:  "your-api-key",
		BaseURL: "https://your-resource.openai.azure.com",
	}

	ctx := context.Background()
	plugin.Init(ctx)

	// 插件会自动使用默认值
	// - APIVersion: "2025-04-01-preview"
	// - Provider: "azure"

	fmt.Println(plugin.Name())
	// Output: azure
}
