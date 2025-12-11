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
	"strings"

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

	// 示例 1: 基本流式响应
	example1_BasicStreaming(ctx, apiKey, baseURL)

	// 示例 2: 流式响应中收集完整内容
	example2_CollectStreamContent(ctx, apiKey, baseURL)

	// 示例 3: 流式响应中处理工具调用
	example3_StreamingWithTools(ctx, apiKey, baseURL)

	// 示例 4: 流式响应中的错误处理
	example4_StreamingErrorHandling(ctx, apiKey, baseURL)
}

// 示例 1: 基本流式响应
func example1_BasicStreaming(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 1: 基本流式响应 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 使用回调函数处理流式响应
	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("请写一个关于人工智能的简短介绍。"),
				},
			},
		},
	}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		// 实时打印每个数据块
		for _, part := range chunk.Content {
			if part.IsText() {
				fmt.Print(part.Text)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("流式生成失败: %v", err)
	}

	fmt.Printf("\n\n完成！总共使用 %d tokens\n\n", resp.Usage.TotalTokens)
}

// 示例 2: 流式响应中收集完整内容
func example2_CollectStreamContent(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 2: 流式响应中收集完整内容 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 用于收集完整内容
	var fullContent strings.Builder
	chunkCount := 0

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("列出学习编程的 5 个步骤。"),
				},
			},
		},
	}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		chunkCount++
		for _, part := range chunk.Content {
			if part.IsText() {
				fullContent.WriteString(part.Text)
				fmt.Print(part.Text) // 实时显示
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("流式生成失败: %v", err)
	}

	fmt.Printf("\n\n收到 %d 个数据块\n", chunkCount)
	fmt.Printf("完整内容长度: %d 字符\n", fullContent.Len())
	fmt.Printf("使用的 tokens: %d\n\n", resp.Usage.TotalTokens)
}

// 示例 3: 流式响应中处理工具调用
func example3_StreamingWithTools(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 3: 流式响应中处理工具调用 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 定义工具
	tools := []*ai.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的天气信息",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{
						"type":        "string",
						"description": "城市名称",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("北京今天天气怎么样？"),
				},
			},
		},
		Tools: tools,
	}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		// 处理文本内容
		for _, part := range chunk.Content {
			if part.IsText() {
				fmt.Print(part.Text)
			}
		}

		// 检查是否有工具调用
		if chunk.Message != nil {
			for _, part := range chunk.Message.Content {
				if part.IsToolRequest() {
					fmt.Printf("\n[工具调用] %s(%v)\n",
						part.ToolRequest.Name,
						part.ToolRequest.Input)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("流式生成失败: %v", err)
	}

	// 检查最终响应中的工具调用
	if resp.Message != nil {
		for _, part := range resp.Message.Content {
			if part.IsToolRequest() {
				fmt.Printf("\n最终工具调用: %s\n", part.ToolRequest.Name)
			}
		}
	}

	fmt.Println("\n完成！\n")
}

// 示例 4: 流式响应中的错误处理
func example4_StreamingErrorHandling(ctx context.Context, apiKey, baseURL string) {
	fmt.Println("=== 示例 4: 流式响应中的错误处理 ===")

	plugin := &azure.AzureAI{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	plugin.Init(ctx)

	model := plugin.DefineModel("azure", "gpt-4", ai.ModelOptions{
		Label: "GPT-4",
	})

	// 模拟在回调中遇到错误
	maxChunks := 5
	chunkCount := 0

	resp, err := model.Generate(ctx, &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: ai.RoleUser,
				Content: []*ai.Part{
					ai.NewTextPart("请写一篇长文章介绍机器学习。"),
				},
			},
		},
	}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		chunkCount++

		// 打印数据块
		for _, part := range chunk.Content {
			if part.IsText() {
				fmt.Print(part.Text)
			}
		}

		// 模拟：在收到一定数量的数据块后停止
		if chunkCount >= maxChunks {
			fmt.Printf("\n\n[达到最大数据块限制 %d，停止接收]\n", maxChunks)
			return fmt.Errorf("达到最大数据块限制")
		}

		return nil
	})

	if err != nil {
		fmt.Printf("流式生成被中断: %v\n", err)
		// 注意：即使回调返回错误，resp 可能仍包含部分数据
		if resp != nil && resp.Message != nil {
			fmt.Printf("已接收的部分内容长度: %d 字符\n", len(resp.Message.Content[0].Text))
		}
	} else {
		fmt.Printf("\n\n完成！使用的 tokens: %d\n", resp.Usage.TotalTokens)
	}

	fmt.Println()
}
