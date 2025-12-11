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
	"errors"
	"fmt"
	"os"

	"genkit-ai-service/internal/genkit/plugins/azure"

	"github.com/firebase/genkit/go/ai"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取配置
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	baseURL := os.Getenv("AZURE_OPENAI_BASE_URL")

	// 示例 1: 配置错误处理
	example1_ConfigError()

	// 示例 2: 网络错误处理
	example2_NetworkError(ctx)

	// 示例 3: API 错误处理
	example3_APIError(ctx, apiKey, baseURL)

	// 示例 4: 错误类型判断
	example4_ErrorTypeChecking(ctx, apiKey, baseURL)

	// 示例 5: 错误恢复和重试
	example5_ErrorRecovery(ctx, apiKey, baseURL)
}

// 示例 1: 配置错误处理
func example1_ConfigError() {
	fmt.Println("=== 示例 1: 配置错误处理 ===")

	ctx := context.Background()

	// 缺少必需的配置
	plugin := &azure.AzureAI{
		// APIKey 和 BaseURL 都缺失
	}

	// 初始化会失败
	actions := plugin.Init(ctx)
	if len(actions) == 0 {
		fmt.Println("初始化失败：缺少必需的配置")
	}

	// 尝试定义模型也会失败
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Hello"),
				},
			},
		},
	}, nil)

	if err != nil {
		// 检查是否是配置错误
		var azErr *azure.AzureAIError
		if errors.As(err, &azErr) {
			if azErr.Type == azure.ErrorTypeConfig {
				fmt.Printf("配置错误: %s\n", azErr.Message)
				fmt.Printf("错误代码: %s\n", azErr.Code)
			}
		}
	}

	fmt.Println()
}

// 示例 2: 网络错误处理
func example2_NetworkError(ctx context.Context) {
	fmt.Println("=== 示例 2: 网络错误处理 ===")

	// 使用无效的 BaseURL
	plugin := &azure.AzureAI{
		APIKey:  "test-key",
		BaseURL: "https://invalid-domain-that-does-not-exist.com",
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Hello"),
				},
			},
		},
	}, nil)

	if err != nil {
		var azErr *azure.AzureAIError
		if errors.As(err, &azErr) {
			if azErr.Type == azure.ErrorTypeNetwork {
				fmt.Printf("网络错误: %s\n", azErr.Message)
				fmt.Printf("可能的原因:\n")
				fmt.Println("  - 网络连接问题")
				fmt.Println("  - DNS 解析失败")
				fmt.Println("  - 服务器不可达")
			}
		}
	}

	fmt.Println()
}

// 示例 3: API 错误处理
func example3_APIError(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 3: API 错误处理 ===")

	if apiKey == "" || baseURL == "" {
		fmt.Println("跳过：需要设置环境变量")
		fmt.Println()
		return
	}

	// 使用无效的 API Key
	plugin := &azure.AzureAI{
		APIKey:  "invalid-api-key",
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Hello"),
				},
			},
		},
	}, nil)

	if err != nil {
		var azErr *azure.AzureAIError
		if errors.As(err, &azErr) {
			if azErr.Type == azure.ErrorTypeAPI {
				fmt.Printf("API 错误: %s\n", azErr.Message)
				fmt.Printf("HTTP 状态码: %s\n", azErr.Code)

				// 根据状态码提供建议
				switch azErr.Code {
				case "401":
					fmt.Println("建议: 检查 API Key 是否正确")
				case "429":
					fmt.Println("建议: 请求过于频繁，请稍后重试")
				case "500", "502", "503", "504":
					fmt.Println("建议: Azure 服务暂时不可用，请稍后重试")
				}

				// 检查详细错误信息
				if azErr.Details != nil {
					fmt.Printf("详细信息: %v\n", azErr.Details)
				}
			}
		}
	}

	fmt.Println()
}

// 示例 4: 错误类型判断
func example4_ErrorTypeChecking(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 4: 错误类型判断 ===")

	if apiKey == "" || baseURL == "" {
		fmt.Println("跳过：需要设置环境变量")
		fmt.Println()
		return
	}

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 发送一个可能失败的请求
	_, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("Hello"),
				},
			},
		},
	}, nil)

	if err != nil {
		handleError(err)
	} else {
		fmt.Println("请求成功")
	}

	fmt.Println()
}

// handleError 演示如何处理不同类型的错误
func handleError(err error) {
	var azErr *azure.AzureAIError
	if !errors.As(err, &azErr) {
		// 不是 AzureAIError，可能是其他类型的错误
		fmt.Printf("未知错误: %v\n", err)
		return
	}

	// 根据错误类型采取不同的处理策略
	switch azErr.Type {
	case azure.ErrorTypeConfig:
		fmt.Println("配置错误 - 需要修复配置")
		fmt.Printf("  错误: %s\n", azErr.Message)
		fmt.Println("  建议: 检查 APIKey 和 BaseURL 是否正确设置")

	case azure.ErrorTypeRequest:
		fmt.Println("请求构建错误 - 需要修复请求参数")
		fmt.Printf("  错误: %s\n", azErr.Message)
		fmt.Println("  建议: 检查消息格式、工具定义等是否正确")

	case azure.ErrorTypeNetwork:
		fmt.Println("网络错误 - 可以重试")
		fmt.Printf("  错误: %s\n", azErr.Message)
		fmt.Println("  建议: 检查网络连接，稍后重试")

	case azure.ErrorTypeAPI:
		fmt.Println("API 错误 - 根据状态码决定是否重试")
		fmt.Printf("  错误: %s\n", azErr.Message)
		fmt.Printf("  状态码: %s\n", azErr.Code)

		// 判断是否可以重试
		if isRetryableStatusCode(azErr.Code) {
			fmt.Println("  建议: 这是一个可重试的错误，请稍后重试")
		} else {
			fmt.Println("  建议: 这是一个不可重试的错误，需要修复请求")
		}

	case azure.ErrorTypeParse:
		fmt.Println("解析错误 - 可能是 API 响应格式变化")
		fmt.Printf("  错误: %s\n", azErr.Message)
		fmt.Println("  建议: 检查 API 版本是否正确，或联系支持")

	default:
		fmt.Printf("未知错误类型: %s\n", azErr.Type)
		fmt.Printf("  错误: %s\n", azErr.Message)
	}

	// 获取原始错误（如果有）
	if azErr.Err != nil {
		fmt.Printf("  原始错误: %v\n", azErr.Err)
	}
}

// isRetryableStatusCode 判断 HTTP 状态码是否可重试
func isRetryableStatusCode(code string) bool {
	retryableCodes := map[string]bool{
		"429": true, // Too Many Requests
		"500": true, // Internal Server Error
		"502": true, // Bad Gateway
		"503": true, // Service Unavailable
		"504": true, // Gateway Timeout
	}
	return retryableCodes[code]
}

// 示例 5: 错误恢复和重试
func example5_ErrorRecovery(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 5: 错误恢复和重试 ===")

	if apiKey == "" || baseURL == "" {
		fmt.Println("跳过：需要设置环境变量")
		fmt.Println()
		return
	}

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 实现简单的重试逻辑
	maxRetries := 3
	var resp *ai.ModelResponse
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("尝试 %d/%d...\n", attempt, maxRetries)

		resp, err = model.Generate(ctx, &ai.ModelRequest{
			Messages: []*ai.Message{
				{
					Role: ai.RoleUser,
					Content: []*ai.Part{
						ai.NewTextPart("Hello, how are you?"),
					},
				},
			},
		}, nil)

		if err == nil {
			fmt.Println("请求成功！")
			fmt.Printf("响应: %s\n", resp.Message.Content[0].Text)
			break
		}

		// 检查是否应该重试
		var azErr *azure.AzureAIError
		if errors.As(err, &azErr) {
			if azErr.Type == azure.ErrorTypeAPI && isRetryableStatusCode(azErr.Code) {
				fmt.Printf("遇到可重试错误 (状态码: %s)，", azErr.Code)
				if attempt < maxRetries {
					fmt.Println("将重试...")
					// 在实际应用中，这里应该添加退避延迟
					// time.Sleep(time.Second * time.Duration(attempt))
				} else {
					fmt.Println("已达到最大重试次数")
				}
			} else {
				fmt.Printf("遇到不可重试错误: %s\n", azErr.Message)
				break
			}
		} else {
			fmt.Printf("遇到未知错误: %v\n", err)
			break
		}
	}

	if err != nil {
		fmt.Printf("最终失败: %v\n", err)
	}

	fmt.Println()
}
