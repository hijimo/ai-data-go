# Azure AI Genkit Provider

Azure AI Genkit Provider 是一个为 Genkit Go 框架提供 Azure OpenAI 服务集成的插件。该插件使用 Azure OpenAI 的 Responses API (`/openai/responses`) 而非传统的 chat/completions 端点。

## 特性

- ✅ 支持文本生成（使用 Responses API）
- ✅ 支持流式响应
- ✅ 支持工具调用（Function Calling）
- ✅ 支持多模态输入（文本和图像）
- ✅ 支持文本嵌入
- ✅ 完整的错误处理和重试机制

## 安装

```bash
go get github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
    ctx := context.Background()
    
    // 创建 Genkit 实例
    g := genkit.New(ctx, nil)
    
    // 初始化 Azure AI Provider
    azurePlugin := &azure.AzureAI{
        APIKey:     "your-api-key",
        BaseURL:    "https://your-resource.openai.azure.com",
        APIVersion: "2025-04-01-preview", // 可选，默认值
        Provider:   "azure",                // 可选，默认值
    }
    
    // 注册插件
    g.RegisterPlugin(azurePlugin)
    
    // 定义模型
    model := azurePlugin.DefineModel("azure", "gpt-4", azure.ModelOptions{
        Label:    "GPT-4",
        Supports: &azure.Multimodal,
    })
    
    // 使用模型
    resp, err := model.Generate(ctx, &genkit.GenerateRequest{
        Messages: []*genkit.Message{
            {Role: "user", Content: []*genkit.Part{genkit.NewTextPart("你好！")}},
        },
    }, nil)
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

### 流式响应

```go
resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *genkit.ModelResponseChunk) error {
    for _, part := range chunk.Content {
        if part.IsText() {
            fmt.Print(part.Text)
        }
    }
    return nil
})
```

### 工具调用

```go
tools := []*genkit.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "获取当前天气",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{"type": "string"},
            },
            "required": []string{"location"},
        },
    },
}

resp, err := model.Generate(ctx, &genkit.GenerateRequest{
    Messages: messages,
    Tools:    tools,
}, nil)
```

### 多模态输入

```go
resp, err := model.Generate(ctx, &genkit.GenerateRequest{
    Messages: []*genkit.Message{
        {
            Role: "user",
            Content: []*genkit.Part{
                genkit.NewTextPart("这张图片里有什么？"),
                genkit.NewMediaPart("https://example.com/image.jpg", "image/jpeg"),
            },
        },
    },
}, nil)
```

### 文本嵌入

```go
embedder := azurePlugin.DefineEmbedder("azure", "text-embedding-ada-002", nil)

resp, err := embedder.Embed(ctx, &genkit.EmbedRequest{
    Input: []*genkit.Document{
        {Content: []*genkit.Part{genkit.NewTextPart("Hello, world!")}},
    },
})
```

## 配置选项

### AzureAI 结构体

- `APIKey` (必需): Azure OpenAI API 密钥
- `BaseURL` (必需): Azure OpenAI 资源的基础 URL，格式为 `https://{resource-name}.openai.azure.com`
- `APIVersion` (可选): API 版本，默认为 `2025-04-01-preview`
- `Provider` (可选): 插件标识符，默认为 `azure`

### 模型配置

支持的配置参数：

- `temperature`: 采样温度（0-2）
- `max_tokens`: 生成的最大 token 数
- `top_p`: 核采样参数
- `frequency_penalty`: 频率惩罚（-2.0 到 2.0）
- `presence_penalty`: 存在惩罚（-2.0 到 2.0）
- `stop`: 停止序列

## API 端点

该插件使用以下 Azure OpenAI API 端点：

- **文本生成**: `{baseURL}/openai/responses?api-version={apiVersion}`
- **文本嵌入**: `{baseURL}/openai/embeddings?api-version={apiVersion}`

注意：使用 Responses API 而非传统的 `/chat/completions` 端点。

## 错误处理

插件提供了详细的错误类型：

- `ConfigError`: 配置错误（缺少必需参数等）
- `RequestError`: 请求构建错误
- `NetworkError`: 网络连接错误
- `APIError`: API 返回的错误
- `ParseError`: 响应解析错误

示例：

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    if azErr, ok := err.(*azure.AzureAIError); ok {
        fmt.Printf("错误类型: %s\n", azErr.Type)
        fmt.Printf("错误代码: %s\n", azErr.Code)
        fmt.Printf("错误消息: %s\n", azErr.Message)
    }
}
```

## 重试机制

插件内置了指数退避重试策略，自动处理以下错误：

- 429 Too Many Requests（速率限制）
- 5xx 服务器错误

## 性能优化

- 使用 HTTP 连接池减少连接建立开销
- 支持批量嵌入请求
- 流式响应减少首字节时间

## 安全注意事项

- 不要在代码中硬编码 API Key，使用环境变量或密钥管理服务
- 注意 Azure OpenAI 的数据处理政策
- 不要在日志中记录完整的请求/响应（可能包含敏感信息）

## 许可证

Apache License 2.0

## 贡献

欢迎提交 Issue 和 Pull Request！
