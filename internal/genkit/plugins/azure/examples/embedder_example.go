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

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"genkit-ai-service/internal/genkit/plugins/azure"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取配置
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	baseURL := os.Getenv("AZURE_OPENAI_BASE_URL")

	if apiKey == "" || baseURL == "" {
		log.Fatal("请设置 AZURE_OPENAI_API_KEY 和 AZURE_OPENAI_BASE_URL 环境变量")
	}

	// 创建 Genkit 实例
	g := genkit.New(ctx, nil)

	// 创建并初始化 Azure AI 插件
	azurePlugin := &azure.AzureAI{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		APIVersion: "2025-04-01-preview",
		Provider:   "azure",
	}

	// 注册插件
	g.RegisterPlugin(azurePlugin)

	// 定义嵌入器
	embedder := azurePlugin.DefineEmbedder("azure", "text-embedding-ada-002", &ai.EmbedderOptions{
		Label:      "Azure OpenAI - text-embedding-ada-002",
		Dimensions: 1536,
		Supports: &ai.EmbedderSupports{
			Input:        []string{"text"},
			Multilingual: true,
		},
	})

	fmt.Println("嵌入器定义成功:", embedder.Name())

	// 示例 1: 单个文档嵌入
	fmt.Println("\n=== 示例 1: 单个文档嵌入 ===")
	singleDocResp, err := embedder.Embed(ctx, &ai.EmbedRequest{
		Input: []*ai.Document{
			{
				Content: []*ai.Part{
					ai.NewTextPart("Hello, this is a test document for embedding."),
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("单个文档嵌入失败: %v", err)
	}

	fmt.Printf("嵌入向量维度: %d\n", len(singleDocResp.Embeddings[0].Embedding))
	fmt.Printf("前 5 个向量值: %v\n", singleDocResp.Embeddings[0].Embedding[:5])

	// 示例 2: 批量文档嵌入
	fmt.Println("\n=== 示例 2: 批量文档嵌入 ===")
	batchResp, err := embedder.Embed(ctx, &ai.EmbedRequest{
		Input: []*ai.Document{
			{
				Content: []*ai.Part{
					ai.NewTextPart("First document about artificial intelligence."),
				},
			},
			{
				Content: []*ai.Part{
					ai.NewTextPart("Second document about machine learning."),
				},
			},
			{
				Content: []*ai.Part{
					ai.NewTextPart("Third document about natural language processing."),
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("批量文档嵌入失败: %v", err)
	}

	fmt.Printf("嵌入数量: %d\n", len(batchResp.Embeddings))
	for i, embedding := range batchResp.Embeddings {
		fmt.Printf("文档 %d 向量维度: %d, 前 3 个值: %v\n",
			i+1, len(embedding.Embedding), embedding.Embedding[:3])
	}

	// 示例 3: 多部分文档嵌入
	fmt.Println("\n=== 示例 3: 多部分文档嵌入 ===")
	multiPartResp, err := embedder.Embed(ctx, &ai.EmbedRequest{
		Input: []*ai.Document{
			{
				Content: []*ai.Part{
					ai.NewTextPart("This is the first part. "),
					ai.NewTextPart("This is the second part. "),
					ai.NewTextPart("This is the third part."),
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("多部分文档嵌入失败: %v", err)
	}

	fmt.Printf("嵌入向量维度: %d\n", len(multiPartResp.Embeddings[0].Embedding))
	fmt.Printf("前 5 个向量值: %v\n", multiPartResp.Embeddings[0].Embedding[:5])

	fmt.Println("\n所有示例执行成功！")
}
