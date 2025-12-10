# Azure OpenAI 嵌入器使用指南

本文档介绍如何使用 Azure AI Genkit Provider 插件的嵌入器功能。

## 概述

Azure AI 插件提供了与 Azure OpenAI 嵌入服务的集成，支持将文本转换为向量表示。嵌入器使用 Azure OpenAI 的 `/openai/embeddings` 端点。

## 功能特性

- ✅ 支持单个文档嵌入
- ✅ 支持批量文档嵌入（在单个 API 请求中处理多个文档）
- ✅ 支持多部分文档内容自动连接
- ✅ 自动处理 float64 到 float32 的向量转换
- ✅ 完整的错误处理和描述性错误信息
- ✅ 符合 Genkit Embedder 接口

## 快速开始

### 1. 初始化插件

```go
import (
    "context"
    "genkit-ai-service/internal/genkit/plugins/azure"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
)

ctx := context.Background()
g := genkit.New(ctx, nil)

// 创建 Azure AI 插件
azurePlugin := &azure.AzureAI{
    APIKey:     "your-api-key",
    BaseURL:    "https://your-resource.openai.azure.com",
    APIVersion: "2025-04-01-preview",
    Provider:   "azure",
}

// 注册插件
g.RegisterPlugin(azurePlugin)
```

### 2. 定义嵌入器

```go
embedder := azurePlugin.DefineEmbedder("azure", "text-embedding-ada-002", &ai.EmbedderOptions{
    Label:      "Azure OpenAI - text-embedding-ada-002",
    Dimensions: 1536,
    Supports: &ai.EmbedderSupports{
        Input:        []string{"text"},
        Multilingual: true,
    },
})
```

### 3. 使用嵌入器

#### 单个文档嵌入

```go
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {
            Content: []*ai.Part{
                ai.NewTextPart("Hello, this is a test document."),
            },
        },
    },
})

if err != nil {
    log.Fatalf("嵌入失败: %v", err)
}

// 访问嵌入向量
embedding := resp.Embeddings[0].Embedding
fmt.Printf("向量维度: %d\n", len(embedding))
```

#### 批量文档嵌入

```go
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {
            Content: []*ai.Part{
                ai.NewTextPart("First document"),
            },
        },
        {
            Content: []*ai.Part{
                ai.NewTextPart("Second document"),
            },
        },
        {
            Content: []*ai.Part{
                ai.NewTextPart("Third document"),
            },
        },
    },
})

if err != nil {
    log.Fatalf("批量嵌入失败: %v", err)
}

// 处理多个嵌入结果
for i, embedding := range resp.Embeddings {
    fmt.Printf("文档 %d 向量维度: %d\n", i+1, len(embedding.Embedding))
}
```

#### 多部分文档嵌入

文档可以包含多个文本部分，它们会被自动连接：

```go
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {
            Content: []*ai.Part{
                ai.NewTextPart("Part 1: "),
                ai.NewTextPart("Part 2: "),
                ai.NewTextPart("Part 3"),
            },
        },
    },
})

// 所有部分会被连接成一个字符串进行嵌入
```

## 支持的模型

Azure OpenAI 支持以下嵌入模型：

- `text-embedding-ada-002` (1536 维)
- `text-embedding-3-small` (1536 维)
- `text-embedding-3-large` (3072 维)

## API 端点

嵌入器使用以下 Azure OpenAI API 端点：

```
POST {baseURL}/openai/embeddings?api-version={apiVersion}
```

### 请求格式

```json
{
  "model": "text-embedding-ada-002",
  "input": ["text1", "text2", "text3"]
}
```

### 响应格式

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.1, 0.2, 0.3, ...]
    }
  ],
  "model": "text-embedding-ada-002",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

## 错误处理

嵌入器提供详细的错误信息：

```go
resp, err := embedder.Embed(ctx, req)
if err != nil {
    // 检查错误类型
    if azErr, ok := err.(*azure.AzureAIError); ok {
        switch azErr.Type {
        case "request":
            // 请求构建错误
            fmt.Printf("请求错误: %s\n", azErr.Message)
        case "network":
            // 网络错误
            fmt.Printf("网络错误: %s\n", azErr.Message)
        case "api":
            // API 错误
            fmt.Printf("API 错误 [%s]: %s\n", azErr.Code, azErr.Message)
        case "parse":
            // 响应解析错误
            fmt.Printf("解析错误: %s\n", azErr.Message)
        }
    }
}
```

## 性能优化

### 批量处理

为了提高性能和减少 API 调用次数，建议批量处理多个文档：

```go
// ✅ 推荐：批量处理
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: documents, // 10-100 个文档
})

// ❌ 不推荐：逐个处理
for _, doc := range documents {
    resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
        Input: []*ai.Document{doc},
    })
}
```

### 建议的批量大小

- 小文档（< 100 tokens）：50-100 个文档/批次
- 中等文档（100-500 tokens）：20-50 个文档/批次
- 大文档（> 500 tokens）：10-20 个文档/批次

## 常见问题

### Q: 嵌入向量的维度是多少？

A: 取决于使用的模型：
- `text-embedding-ada-002`: 1536 维
- `text-embedding-3-small`: 1536 维
- `text-embedding-3-large`: 3072 维

### Q: 如何处理空文档？

A: 空文档会被转换为空字符串，Azure OpenAI API 会返回相应的嵌入向量。

### Q: 批量嵌入的最大数量是多少？

A: Azure OpenAI API 没有明确的限制，但建议每批次不超过 100 个文档以获得最佳性能。

### Q: 嵌入向量是 float32 还是 float64？

A: Genkit 使用 float32，插件会自动将 Azure OpenAI 返回的 float64 转换为 float32。

## 示例代码

完整的示例代码请参考：
- `examples/embedder_example.go` - 基本使用示例
- `embed_test.go` - 单元测试示例

## 相关文档

- [Azure OpenAI Embeddings API 文档](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference#embeddings)
- [Genkit Embedder 接口文档](https://firebase.google.com/docs/genkit/embedders)
- [Azure AI Provider 设计文档](./design.md)

## 许可证

Copyright 2025 Google LLC

Licensed under the Apache License, Version 2.0
