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
	"os"

	"genkit-ai-service/internal/genkit/plugins/azure"

	"github.com/firebase/genkit/go/ai"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取配置
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	baseURL := os.Getenv("AZURE_OPENAI_BASE_URL")

	if apiKey == "" || baseURL == "" {
		log.Fatal("请设置 AZURE_OPENAI_API_KEY 和 AZURE_OPENAI_BASE_URL 环境变量")
	}

	// 示例 1: 最简单的使用方式
	example1_SimpleGeneration(ctx, apiKey, baseURL)

	// 示例 2: 多轮对话
	example2_MultiTurnConversation(ctx, apiKey, baseURL)

	// 示例 3: 使用系统提示
	example3_SystemPrompt(ctx, apiKey, baseURL)

	// 示例 4: 配置生成参数
	example4_ConfigureParameters(ctx, apiKey, baseURL)

	// 示例 5: 多模态输入（文本 + 图像）
	example5_MultimodalInput(ctx, apiKey, baseURL)
}

// 示例 1: 最简单的使用方式
func example1_SimpleGeneration(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 1: 最简单的使用方式 ===")

	// 创建 Azure AI 插件
	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}

	// 初始化插件
	plugin.Init(ctx)

	// 定义模型
	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 生成响应
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("你好！请用一句话介绍你自己。"),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 2: 多轮对话
func example2_MultiTurnConversation(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 2: 多轮对话 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 构建对话历史
	messages := []*ai.Message{
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("我想学习 Go 语言，你有什么建议吗？"),
			},
		},
		{
			Role: ai.RoleModel,
			Content: []*ai.Part{
				ai.NewTextPart("学习 Go 语言是个很好的选择！我建议从官方文档开始，然后通过实践项目来巩固知识。"),
			},
		},
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("有什么好的实践项目推荐吗？"),
			},
		},
	}

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 3: 使用系统提示
func example3_SystemPrompt(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 3: 使用系统提示 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 使用系统提示定义 AI 的行为
	messages := []*ai.Message{
		{
			Role: ai.RoleSystem,
			Content: []*ai.Part{
				ai.NewTextPart("你是一个专业的技术文档写作助手，擅长用简洁清晰的语言解释复杂的技术概念。"),
			},
		},
		{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				ai.NewTextPart("请解释什么是 RESTful API。"),
			},
		},
	}

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: messages,
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}

// 示例 4: 配置生成参数
func example4_ConfigureParameters(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 4: 配置生成参数 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 配置生成参数
	temperature := 0.7
	maxTokens := 100
	topP := 0.9

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("写一首关于春天的短诗。"),
				},
			},
		},
		Config: map[string]any{
			"temperature": temperature,
			"max_tokens":  maxTokens,
			"top_p":       topP,
		},
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("响应: %s\n", resp.Message.Content[0].Text)
	fmt.Printf("使用的 tokens: %d\n\n", resp.Usage.TotalTokens)
}

// 示例 5: 多模态输入（文本 + 图像）
func example5_MultimodalInput(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 5: 多模态输入（文本 + 图像）===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	// 使用支持多模态的模型
	model := plugin.DefineModel("azure", "gpt-4-vision", ai.ModelOptions{
		Label:    "GPT-4 Vision",
		Supports: &azure.Multimodal,
	})

	// 构建包含文本和图像的消息
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("这张图片里有什么？请详细描述。"),
					ai.NewMediaPart("https://example.com/image.jpg", "image/jpeg"),
				},
			},
		},
	}, nil)

	if err != nil {
		log.Fatalf("生成失败: %v", err)
	}

	fmt.Printf("响应: %s\n\n", resp.Message.Content[0].Text)
}
