# Azure AI Genkit Provider

Azure AI Genkit Provider 是一个为 Genkit Go 框架提供 Azure OpenAI 服务集成的插件。该插件使用 Azure OpenAI 的 Responses API (`/openai/responses`) 而非传统的 chat/completions 端点。

## 目录

- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [详细示例](#详细示例)
  - [基本使用](#基本使用)
  - [流式响应](#流式响应)
  - [工具调用](#工具调用)
  - [文本嵌入](#文本嵌入)
  - [错误处理](#错误处理)
- [配置选项](#配置选项)
- [API 端点](#api-端点)
- [重试和超时](#重试和超时)
- [性能优化](#性能优化)
- [安全注意事项](#安全注意事项)
- [示例代码](#示例代码)
- [常见问题](#常见问题)

## 特性

- ✅ 支持文本生成（使用 Responses API）
- ✅ 支持流式响应（Server-Sent Events）
- ✅ 支持工具调用（Function Calling）
- ✅ 支持多模态输入（文本和图像）
- ✅ 支持文本嵌入（批量处理）
- ✅ 完整的错误处理和分类
- ✅ 自动重试机制（指数退避）
- ✅ 灵活的超时配置
- ✅ 连接池优化

## 安装

```bash
go get genkit-ai-service/internal/genkit/plugins/azure
```

## 快速开始

### 环境变量设置

首先，设置必需的环境变量：

```bash
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_BASE_URL="https://your-resource.openai.azure.com"
```

### 最简单的使用方式

```go
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
    
    // 创建并初始化 Azure AI Provider
    plugin := &azure.AzureAI{
        APIKey:  os.Getenv("AZURE_OPENAI_API_KEY"),
        BaseURL: os.Getenv("AZURE_OPENAI_BASE_URL"),
    }
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
        log.Fatal(err)
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

## 详细示例

### 基本使用

#### 1. 简单文本生成

```go
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
```

#### 2. 多轮对话

```go
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
            ai.NewTextPart("学习 Go 语言是个很好的选择！我建议从官方文档开始..."),
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
```

#### 3. 使用系统提示

```go
messages := []*ai.Message{
    {
        Role: ai.RoleSystem,
        Content: []*ai.Part{
            ai.NewTextPart("你是一个专业的技术文档写作助手。"),
        },
    },
    {
        Role: ai.RoleUser,
        Content: []*ai.Part{
            ai.NewTextPart("请解释什么是 RESTful API。"),
        },
    },
}
```

#### 4. 配置生成参数

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: messages,
    Config: map[string]any{
        "temperature": 0.7,
        "max_tokens":  100,
        "top_p":       0.9,
    },
}, nil)
```

#### 5. 多模态输入（文本 + 图像）

```go
model := plugin.DefineModel("azure", "gpt-4-vision", ai.ModelOptions{
    Label:    "GPT-4 Vision",
    Supports: &azure.Multimodal,
})

resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("这张图片里有什么？"),
                ai.NewMediaPart("https://example.com/image.jpg", "image/jpeg"),
            },
        },
    },
}, nil)
```

### 流式响应

#### 1. 基本流式响应

```go
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
```

#### 2. 收集流式内容

```go
var fullContent strings.Builder

resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    for _, part := range chunk.Content {
        if part.IsText() {
            fullContent.WriteString(part.Text)
            fmt.Print(part.Text) // 同时实时显示
        }
    }
    return nil
})

fmt.Printf("\n完整内容: %s\n", fullContent.String())
```

#### 3. 流式响应中处理工具调用

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: messages,
    Tools:    tools,
}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    // 处理文本内容
    for _, part := range chunk.Content {
        if part.IsText() {
            fmt.Print(part.Text)
        }
    }
    
    // 检查工具调用
    if chunk.Message != nil {
        for _, part := range chunk.Message.Content {
            if part.IsToolRequest() {
                fmt.Printf("\n[工具调用] %s\n", part.ToolRequest.Name)
            }
        }
    }
    
    return nil
})
```

### 工具调用

#### 1. 定义和使用工具

```go
// 定义工具
tools := []*ai.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "获取指定城市的当前天气信息",
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

// 第一轮：模型决定调用工具
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
}, nil)

// 检查工具调用
var toolCall *ai.ToolRequest
for _, part := range resp.Message.Content {
    if part.IsToolRequest() {
        toolCall = part.ToolRequest
        break
    }
}

// 执行工具
result := executeGetWeather(toolCall.Input)

// 第二轮：返回工具结果
messages := []*ai.Message{
    {
        Role: ai.RoleUser,
        Content: []*ai.Part{
            ai.NewTextPart("北京今天天气怎么样？"),
        },
    },
    resp.Message, // 包含工具调用的助手消息
    {
        Role: ai.RoleTool,
        Content: []*ai.Part{
            ai.NewToolResponsePart(&ai.ToolResponse{
                Name:   toolCall.Name,
                Ref:    toolCall.Ref,
                Output: result,
            }),
        },
    },
}

finalResp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: messages,
    Tools:    tools,
}, nil)
```

#### 2. 多个工具调用

```go
// 定义多个工具
tools := []*ai.ToolDefinition{
    {Name: "get_weather", ...},
    {Name: "get_time", ...},
    {Name: "calculate", ...},
}

// 模型可能同时调用多个工具
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("请告诉我北京的天气和当前时间。"),
            },
        },
    },
    Tools: tools,
}, nil)

// 处理所有工具调用
for _, part := range resp.Message.Content {
    if part.IsToolRequest() {
        result := executeToolByName(part.ToolRequest.Name, part.ToolRequest.Input)
        // 添加工具响应...
    }
}
```

#### 3. 强制工具调用

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: messages,
    Tools:    tools,
    Config: map[string]any{
        "tool_choice": map[string]any{
            "type": "function",
            "function": map[string]any{
                "name": "get_weather",
            },
        },
    },
}, nil)
```

### 文本嵌入

#### 1. 单个文档嵌入

```go
embedder := plugin.DefineEmbedder("azure", "text-embedding-ada-002", &ai.EmbedderOptions{
    Label:      "Azure OpenAI - text-embedding-ada-002",
    Dimensions: 1536,
})

resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {
            Content: []*ai.Part{
                ai.NewTextPart("Hello, this is a test document."),
            },
        },
    },
})

fmt.Printf("向量维度: %d\n", len(resp.Embeddings[0].Embedding))
```

#### 2. 批量文档嵌入

```go
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {Content: []*ai.Part{ai.NewTextPart("First document")}},
        {Content: []*ai.Part{ai.NewTextPart("Second document")}},
        {Content: []*ai.Part{ai.NewTextPart("Third document")}},
    },
})

for i, embedding := range resp.Embeddings {
    fmt.Printf("文档 %d 向量维度: %d\n", i+1, len(embedding.Embedding))
}
```

### 错误处理

#### 1. 基本错误处理

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        fmt.Printf("错误类型: %s\n", azErr.Type)
        fmt.Printf("错误代码: %s\n", azErr.Code)
        fmt.Printf("错误消息: %s\n", azErr.Message)
    }
}
```

#### 2. 错误类型判断

```go
if err != nil {
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        switch azErr.Type {
        case azure.ErrorTypeConfig:
            fmt.Println("配置错误 - 检查 APIKey 和 BaseURL")
        case azure.ErrorTypeNetwork:
            fmt.Println("网络错误 - 检查网络连接")
        case azure.ErrorTypeAPI:
            fmt.Printf("API 错误 - 状态码: %s\n", azErr.Code)
            if azErr.Code == "429" {
                fmt.Println("请求过于频繁，请稍后重试")
            }
        case azure.ErrorTypeParse:
            fmt.Println("解析错误 - 可能是 API 响应格式变化")
        }
    }
}
```

#### 3. 实现重试逻辑

```go
maxRetries := 3
var resp *ai.ModelResponse
var err error

for attempt := 1; attempt <= maxRetries; attempt++ {
    resp, err = model.Generate(ctx, req, nil)
    if err == nil {
        break
    }
    
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        // 只重试可重试的错误
        if azErr.Type == azure.ErrorTypeAPI && 
           (azErr.Code == "429" || azErr.Code == "500" || azErr.Code == "503") {
            if attempt < maxRetries {
                time.Sleep(time.Second * time.Duration(attempt))
                continue
            }
        }
    }
    break
}
```

## 配置选项

### AzureAI 结构体

```go
type AzureAI struct {
    // 必需配置
    APIKey  string  // Azure OpenAI API 密钥
    BaseURL string  // Azure OpenAI 资源的基础 URL
    
    // 可选配置
    APIVersion    string         // API 版本，默认: "2025-04-01-preview"
    Provider      string         // 插件标识符，默认: "azure"
    RetryConfig   *RetryConfig   // 重试配置
    TimeoutConfig *TimeoutConfig // 超时配置
}
```

**必需参数**：
- `APIKey`: Azure OpenAI API 密钥
- `BaseURL`: Azure OpenAI 资源的基础 URL，格式为 `https://{resource-name}.openai.azure.com`

**可选参数**：
- `APIVersion`: API 版本，默认为 `2025-04-01-preview`
- `Provider`: 插件标识符，默认为 `azure`
- `RetryConfig`: 重试配置（见下文）
- `TimeoutConfig`: 超时配置（见下文）

### 模型配置

在 `Generate` 请求中可以通过 `Config` 字段配置以下参数：

```go
Config: map[string]any{
    "temperature":        0.7,    // 采样温度（0-2）
    "max_tokens":         100,    // 生成的最大 token 数
    "top_p":              0.9,    // 核采样参数（0-1）
    "frequency_penalty":  0.0,    // 频率惩罚（-2.0 到 2.0）
    "presence_penalty":   0.0,    // 存在惩罚（-2.0 到 2.0）
    "stop":               []string{"END"}, // 停止序列
}
```

**参数说明**：
- `temperature`: 控制输出的随机性，值越高越随机
- `max_tokens`: 限制生成的最大 token 数
- `top_p`: 核采样，控制输出的多样性
- `frequency_penalty`: 降低重复词汇的频率
- `presence_penalty`: 鼓励谈论新话题
- `stop`: 遇到这些字符串时停止生成

## API 端点

该插件使用以下 Azure OpenAI API 端点：

- **文本生成**: `{baseURL}/openai/responses?api-version={apiVersion}`
- **文本嵌入**: `{baseURL}/openai/embeddings?api-version={apiVersion}`

**重要说明**：
- 使用 Responses API (`/openai/responses`) 而非传统的 `/chat/completions` 端点
- 请求体使用 `input` 字段而非 `messages` 字段（符合 Responses API 规范）
- 默认 API 版本为 `2025-04-01-preview`

## 重试和超时

### 重试配置

插件内置了指数退避重试策略，自动处理以下错误：

- 429 Too Many Requests（速率限制）
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout

**默认重试配置**：

```go
RetryConfig: &azure.RetryConfig{
    MaxRetries:        3,
    InitialBackoff:    1 * time.Second,
    MaxBackoff:        30 * time.Second,
    BackoffMultiplier: 2.0,
}
```

**自定义重试配置**：

```go
plugin := &azure.AzureAI{
    APIKey:  apiKey,
    BaseURL: baseURL,
    RetryConfig: &azure.RetryConfig{
        MaxRetries:        5,                      // 增加重试次数
        InitialBackoff:    500 * time.Millisecond, // 减少初始退避时间
        MaxBackoff:        60 * time.Second,       // 增加最大退避时间
        BackoffMultiplier: 2.0,
    },
}
```

**禁用重试**：

```go
plugin := &azure.AzureAI{
    APIKey:  apiKey,
    BaseURL: baseURL,
    RetryConfig: &azure.RetryConfig{
        MaxRetries: 0, // 设置为 0 禁用重试
    },
}
```

### 超时配置

**默认超时配置**：

```go
TimeoutConfig: &azure.TimeoutConfig{
    RequestTimeout:      30 * time.Second,  // 普通请求超时
    StreamTimeout:       60 * time.Second,  // 流式请求超时
    DialTimeout:         10 * time.Second,  // 连接超时
    TLSHandshakeTimeout: 10 * time.Second,  // TLS 握手超时
    IdleConnTimeout:     90 * time.Second,  // 空闲连接超时
}
```

**自定义超时配置**：

```go
plugin := &azure.AzureAI{
    APIKey:  apiKey,
    BaseURL: baseURL,
    TimeoutConfig: &azure.TimeoutConfig{
        RequestTimeout: 60 * time.Second,  // 增加请求超时
        StreamTimeout:  120 * time.Second, // 增加流式超时
    },
}
```

**使用上下文超时**：

```go
// 创建带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 使用带超时的上下文
resp, err := model.Generate(ctx, req, nil)
if err == context.DeadlineExceeded {
    fmt.Println("请求超时")
}
```

## 错误处理

### 错误类型

插件提供了详细的错误分类：

```go
const (
    ErrorTypeConfig  = "config"   // 配置错误
    ErrorTypeRequest = "request"  // 请求构建错误
    ErrorTypeNetwork = "network"  // 网络连接错误
    ErrorTypeAPI     = "api"      // API 返回的错误
    ErrorTypeParse   = "parse"    // 响应解析错误
)
```

### AzureAIError 结构

```go
type AzureAIError struct {
    Type    string // 错误类型
    Code    string // HTTP 状态码或错误代码
    Message string // 错误消息
    Details any    // 详细错误信息
    Err     error  // 原始错误
}
```

### 错误处理示例

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        switch azErr.Type {
        case azure.ErrorTypeConfig:
            // 配置错误 - 需要修复配置
            fmt.Printf("配置错误: %s\n", azErr.Message)
            
        case azure.ErrorTypeNetwork:
            // 网络错误 - 可以重试
            fmt.Printf("网络错误: %s\n", azErr.Message)
            
        case azure.ErrorTypeAPI:
            // API 错误 - 根据状态码决定
            fmt.Printf("API 错误 (状态码 %s): %s\n", azErr.Code, azErr.Message)
            if azErr.Code == "429" {
                fmt.Println("请求过于频繁，请稍后重试")
            }
            
        case azure.ErrorTypeParse:
            // 解析错误 - 可能是 API 响应格式变化
            fmt.Printf("解析错误: %s\n", azErr.Message)
        }
    }
}
```

## 性能优化

### 连接池

插件使用 HTTP 连接池来减少连接建立开销：

```go
// 默认连接池配置
Transport: &http.Transport{
    MaxIdleConns:        100,  // 最大空闲连接数
    MaxIdleConnsPerHost: 10,   // 每个主机的最大空闲连接数
    IdleConnTimeout:     90 * time.Second,
}
```

### 批量处理

嵌入请求支持批量处理，减少 API 调用次数：

```go
// 一次请求处理多个文档
resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Input: []*ai.Document{
        {Content: []*ai.Part{ai.NewTextPart("Doc 1")}},
        {Content: []*ai.Part{ai.NewTextPart("Doc 2")}},
        {Content: []*ai.Part{ai.NewTextPart("Doc 3")}},
        // ... 建议批量大小：10-100 个文档
    },
})
```

### 流式响应

使用流式响应可以减少首字节时间，提升用户体验：

```go
// 流式响应立即开始返回数据
resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    // 实时处理数据块
    return nil
})
```

### 性能建议

1. **复用插件实例**：避免为每个请求创建新的插件实例
2. **使用连接池**：保持默认的连接池配置，除非有特殊需求
3. **批量嵌入**：将多个文档合并到一个请求中
4. **流式响应**：对于长文本生成，使用流式响应提升体验
5. **合理设置超时**：根据实际需求调整超时时间

## 安全注意事项

### API Key 管理

**不要在代码中硬编码 API Key**：

```go
// ❌ 错误做法
plugin := &azure.AzureAI{
    APIKey: "sk-abc123...",  // 不要硬编码
}

// ✅ 正确做法
plugin := &azure.AzureAI{
    APIKey: os.Getenv("AZURE_OPENAI_API_KEY"),  // 使用环境变量
}
```

### 数据隐私

1. **注意数据处理政策**：了解 Azure OpenAI 的数据处理和存储政策
2. **不要发送敏感信息**：避免在请求中包含个人身份信息（PII）
3. **使用私有端点**：考虑使用 Azure 的私有端点功能

### 日志安全

```go
// ❌ 不要记录完整的请求/响应
log.Printf("Request: %+v", req)  // 可能包含敏感信息

// ✅ 只记录必要的元数据
log.Printf("Request ID: %s, Model: %s", requestID, modelName)
```

### 错误信息

```go
// 确保错误信息不暴露 API Key
if err != nil {
    // 插件已经处理了敏感信息的过滤
    log.Printf("Error: %v", err)
}
```

## 示例代码

完整的示例代码位于 `examples/` 目录：

- **basic_usage.go**: 基本使用示例
  - 简单文本生成
  - 多轮对话
  - 系统提示
  - 配置参数
  - 多模态输入

- **streaming_example.go**: 流式响应示例
  - 基本流式响应
  - 收集流式内容
  - 流式工具调用
  - 错误处理

- **tool_calling_example.go**: 工具调用示例
  - 单个工具调用
  - 多个工具调用
  - 工具调用链
  - 强制工具调用

- **embedder_example.go**: 嵌入示例
  - 单个文档嵌入
  - 批量文档嵌入
  - 多部分文档嵌入

- **error_handling_example.go**: 错误处理示例
  - 配置错误
  - 网络错误
  - API 错误
  - 错误类型判断
  - 错误恢复和重试

- **retry_example.go**: 重试和超时示例
  - 默认配置
  - 自定义重试配置
  - 自定义超时配置
  - 上下文超时
  - 禁用重试

### 运行示例

```bash
# 设置环境变量
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_BASE_URL="https://your-resource.openai.azure.com"

# 运行示例
go run internal/genkit/plugins/azure/examples/basic_usage.go
go run internal/genkit/plugins/azure/examples/streaming_example.go
go run internal/genkit/plugins/azure/examples/tool_calling_example.go
```

## 常见问题

### Q: 为什么使用 Responses API 而不是 Chat Completions API？

A: Responses API 是 Azure OpenAI 的新 API 端点，提供了更好的性能和更灵活的功能。该插件专门为 Responses API 设计。

### Q: 如何处理速率限制（429 错误）？

A: 插件内置了自动重试机制，会自动处理 429 错误。你也可以通过 `RetryConfig` 自定义重试策略。

### Q: 支持哪些模型？

A: 支持所有 Azure OpenAI 部署的模型，包括：
- GPT-4 系列（文本生成）
- GPT-4 Vision（多模态）
- GPT-3.5 Turbo（文本生成）
- text-embedding-ada-002（文本嵌入）

### Q: 如何调试请求和响应？

A: 可以通过检查错误详情来调试：

```go
if err != nil {
    var azErr *azure.AzureAIError
    if errors.As(err, &azErr) {
        fmt.Printf("错误类型: %s\n", azErr.Type)
        fmt.Printf("详细信息: %+v\n", azErr.Details)
    }
}
```

### Q: 流式响应中如何处理错误？

A: 在回调函数中返回错误即可中断流式响应：

```go
resp, err := model.Generate(ctx, req, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    if someCondition {
        return fmt.Errorf("停止接收")
    }
    return nil
})
```

### Q: 如何设置请求超时？

A: 有两种方式：

1. 使用上下文超时：
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

2. 使用插件配置：
```go
plugin := &azure.AzureAI{
    TimeoutConfig: &azure.TimeoutConfig{
        RequestTimeout: 30 * time.Second,
    },
}
```

### Q: 支持并发请求吗？

A: 是的，插件是线程安全的，支持并发请求。HTTP 客户端使用连接池来优化并发性能。

## 相关文档

- [错误处理详细文档](ERROR_HANDLING.md)
- [重试和超时配置](RETRY_AND_TIMEOUT.md)
- [嵌入器使用指南](EMBEDDER_README.md)

## 许可证

Apache License 2.0

## 贡献

欢迎提交 Issue 和 Pull Request！

如果你发现 bug 或有功能建议，请在 GitHub 上创建 Issue。
