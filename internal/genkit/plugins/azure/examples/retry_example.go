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

//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"genkit-ai-service/internal/genkit/plugins/azure"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func main() {
	// 示例 1: 使用默认重试和超时配置
	example1_DefaultConfig()

	// 示例 2: 自定义重试配置
	example2_CustomRetryConfig()

	// 示例 3: 自定义超时配置
	example3_CustomTimeoutConfig()

	// 示例 4: 使用上下文超时
	example4_ContextTimeout()

	// 示例 5: 禁用重试
	example5_DisableRetry()
}

// 示例 1: 使用默认重试和超时配置
func example1_DefaultConfig() {
	fmt.Println("=== 示例 1: 使用默认重试和超时配置 ===")

	ctx := context.Background()

	// 创建 Azure AI Provider（使用默认配置）
	azurePlugin := &azure.AzureAI{
		APIKey:     getEnv("AZURE_OPENAI_API_KEY", "your-api-key"),
		BaseURL:    getEnv("AZURE_OPENAI_BASE_URL", "https://your-resource.openai.azure.com"),
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
		// RetryConfig 和 TimeoutConfig 为 nil，将使用默认配置
	}

	// 初始化插件
	g := genkit.New(ctx, nil)
	g.RegisterPlugin(azurePlugin)

	// 定义模型
	model := azurePlugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &azure.Multimodal,
	})

	// 使用模型（自动重试和超时）
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Hello! Please respond with a short greeting."),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Printf("请求失败: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 2: 自定义重试配置
func example2_CustomRetryConfig() {
	fmt.Println("=== 示例 2: 自定义重试配置 ===")

	ctx := context.Background()

	// 自定义重试配置
	retryConfig := &azure.RetryConfig{
		MaxRetries:        5,                      // 增加重试次数
		InitialBackoff:    500 * time.Millisecond, // 减少初始退避时间
		MaxBackoff:        60 * time.Second,       // 增加最大退避时间
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: map[int]bool{
			http.StatusTooManyRequests:     true, // 429
			http.StatusInternalServerError: true, // 500
			http.StatusBadGateway:          true, // 502
			http.StatusServiceUnavailable:  true, // 503
			http.StatusGatewayTimeout:      true, // 504
		},
	}

	// 创建 Azure AI Provider
	azurePlugin := &azure.AzureAI{
		APIKey:      getEnv("AZURE_OPENAI_API_KEY", "your-api-key"),
		BaseURL:     getEnv("AZURE_OPENAI_BASE_URL", "https://your-resource.openai.azure.com"),
		APIVersion:  "2025-04-01-preview",
		Provider:    "azure",
		RetryConfig: retryConfig, // 使用自定义重试配置
	}

	g := genkit.New(ctx, nil)
	g.RegisterPlugin(azurePlugin)

	model := azurePlugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &azure.Multimodal,
	})

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("What is the capital of France?"),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Printf("请求失败: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 3: 自定义超时配置
func example3_CustomTimeoutConfig() {
	fmt.Println("=== 示例 3: 自定义超时配置 ===")

	ctx := context.Background()

	// 自定义超时配置
	timeoutConfig := &azure.TimeoutConfig{
		RequestTimeout:      60 * time.Second,  // 增加请求超时
		StreamTimeout:       120 * time.Second, // 增加流式超时
		DialTimeout:         5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		IdleConnTimeout:     120 * time.Second,
	}

	// 创建 Azure AI Provider
	azurePlugin := &azure.AzureAI{
		APIKey:        getEnv("AZURE_OPENAI_API_KEY", "your-api-key"),
		BaseURL:       getEnv("AZURE_OPENAI_BASE_URL", "https://your-resource.openai.azure.com"),
		APIVersion:    "2025-04-01-preview",
		Provider:      "azure",
		TimeoutConfig: timeoutConfig, // 使用自定义超时配置
	}

	g := genkit.New(ctx, nil)
	g.RegisterPlugin(azurePlugin)

	model := azurePlugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &azure.Multimodal,
	})

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Explain quantum computing in simple terms."),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Printf("请求失败: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 4: 使用上下文超时
func example4_ContextTimeout() {
	fmt.Println("=== 示例 4: 使用上下文超时 ===")

	// 创建带超时的上下文（10 秒）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建 Azure AI Provider
	azurePlugin := &azure.AzureAI{
		APIKey:     getEnv("AZURE_OPENAI_API_KEY", "your-api-key"),
		BaseURL:    getEnv("AZURE_OPENAI_BASE_URL", "https://your-resource.openai.azure.com"),
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	g := genkit.New(context.Background(), nil)
	g.RegisterPlugin(azurePlugin)

	model := azurePlugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &azure.Multimodal,
	})

	// 使用带超时的上下文
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Tell me a short joke."),
				},
			},
		},
	}, nil)

	if err != nil {
		if err == context.DeadlineExceeded {
			fmt.Println("请求超时")
		} else if err == context.Canceled {
			fmt.Println("请求被取消")
		} else {
			log.Printf("请求失败: %v\n", err)
		}
		return
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 5: 禁用重试
func example5_DisableRetry() {
	fmt.Println("=== 示例 5: 禁用重试 ===")

	ctx := context.Background()

	// 禁用重试
	retryConfig := &azure.RetryConfig{
		MaxRetries: 0, // 设置为 0 禁用重试
	}

	// 创建 Azure AI Provider
	azurePlugin := &azure.AzureAI{
		APIKey:      getEnv("AZURE_OPENAI_API_KEY", "your-api-key"),
		BaseURL:     getEnv("AZURE_OPENAI_BASE_URL", "https://your-resource.openai.azure.com"),
		APIVersion:  "2025-04-01-preview",
		Provider:    "azure",
		RetryConfig: retryConfig,
	}

	g := genkit.New(ctx, nil)
	g.RegisterPlugin(azurePlugin)

	model := azurePlugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label:    "GPT-4",
		Supports: &azure.Multimodal,
	})

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("What is 2+2?"),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Printf("请求失败（不会重试）: %v\n", err)
		return
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
