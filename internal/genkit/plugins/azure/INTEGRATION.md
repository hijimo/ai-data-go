# Azure AI Genkit Provider 集成指南

## 概述

Azure AI Genkit Provider 已成功集成到现有的 Genkit AI Service 项目中。该插件提供了对 Azure OpenAI 服务的原生支持，使用 Azure OpenAI 的 Responses API (`/openai/responses`) 而非传统的 chat/completions 端点。

## 集成位置

插件代码位于：`internal/genkit/plugins/azure/`

集成点：
- `internal/genkit/client.go` - Genkit 客户端，负责初始化和管理提供商
- `cmd/server/main.go` - 服务器主程序，负责初始化服务

## 提供商类型

系统现在支持两种 Azure OpenAI 集成方式：

### 1. `azure` - 原生 Azure 插件（推荐）

使用原生 Azure AI 插件，支持 Azure OpenAI Responses API。

**特点：**
- 使用 `/openai/responses` 端点
- 支持最新的 Azure OpenAI API 版本
- 原生支持 Azure 特性（如系统指纹、自定义请求头等）
- 支持重试和超时配置
- 更好的错误处理

**配置示例：**
```json
{
  "modelProvider": "azure",
  "model": "gpt-4",
  "baseUrl": "https://your-resource.openai.azure.com",
  "queryParams": {
    "azureApiVersion": "2025-04-01-preview"
  }
}
```

### 2. `azureopenai` - 兼容模式（向后兼容）

使用 OpenAI 兼容插件 + Azure BaseURL 的方式。

**特点：**
- 使用传统的 `/chat/completions` 端点
- 向后兼容现有配置
- 适用于不支持 Responses API 的场景

**配置示例：**
```json
{
  "modelProvider": "azureopenai",
  "model": "gpt-4",
  "baseUrl": "https://your-resource.openai.azure.com/openai/deployments/gpt-4",
  "queryParams": {
    "azureApiVersion": "2024-12-01-preview"
  }
}
```

## 使用方法

### 1. 通过模型配置管理接口

使用模型配置管理 API 创建 Azure 模型配置：

```bash
POST /api/v1/model-configurations
Content-Type: application/json
Authorization: Bearer {token}

{
  "tenantId": "your-tenant-id",
  "modelName": "gpt-4-azure",
  "modelProvider": "azure",
  "model": "gpt-4",
  "baseUrl": "https://your-resource.openai.azure.com",
  "apiKey": "your-api-key",
  "queryParams": {
    "azureApiVersion": "2025-04-01-preview"
  },
  "isEnabled": true
}
```

### 2. 通过代码直接使用

```go
package main

import (
    "context"
    "fmt"
    
    "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/genkit/plugins/azure"
    "github.com/firebase/genkit/go/genkit"
)

func main() {
    ctx := context.Background()
    
    // 创建 Azure AI 插件
    azurePlugin := &azure.AzureAI{
        APIKey:     "your-api-key",
        BaseURL:    "https://your-resource.openai.azure.com",
        APIVersion: "2025-04-01-preview",
        Provider:   "azure",
    }
    
    // 初始化插件
    azurePlugin.Init(ctx)
    
    // 创建 Genkit 实例
    g := genkit.Init(ctx,
        genkit.WithPlugins(azurePlugin),
        genkit.WithDefaultModel("azure/gpt-4"),
    )
    
    // 定义模型
    model := azurePlugin.DefineModel("azure", "gpt-4", azure.Multimodal)
    
    // 使用模型
    resp, err := model.Generate(ctx, &genkit.GenerateRequest{
        Messages: []*genkit.Message{
            {Role: "user", Content: []*genkit.Part{genkit.NewTextPart("Hello!")}},
        },
    }, nil)
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

## 配置参数

### 必需参数

- `modelProvider`: 提供商类型，设置为 `"azure"`
- `model`: 模型名称（如 `"gpt-4"`, `"gpt-4-turbo"`, `"gpt-4o"` 等）
- `baseUrl`: Azure OpenAI 资源的基础 URL
- `apiKey`: Azure OpenAI API 密钥

### 可选参数

通过 `queryParams` 字段配置：

- `azureApiVersion`: API 版本（默认：`"2025-04-01-preview"`）
- `azureOrganization`: Azure 组织 ID
- `customHeaders`: 自定义请求头（JSON 对象）

### 超时和重试配置

插件使用默认的超时和重试配置：

**超时配置：**
- 非流式请求：30 秒
- 流式请求：60 秒
- 连接超时：10 秒
- TLS 握手超时：10 秒

**重试配置：**
- 最大重试次数：3 次
- 初始退避时间：1 秒
- 最大退避时间：30 秒
- 退避倍数：2
- 可重试的 HTTP 状态码：429, 500, 502, 503, 504

## 支持的功能

### 1. 文本生成

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {Role: ai.RoleUser, Content: []*ai.Part{ai.NewTextPart("Hello!")}},
    },
}, nil)
```

### 2. 流式响应

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {Role: ai.RoleUser, Content: []*ai.Part{ai.NewTextPart("Hello!")}},
    },
}, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
    fmt.Print(chunk.Text())
    return nil
})
```

### 3. 工具调用

```go
tools := []*ai.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "Get current weather",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{"type": "string"},
            },
            "required": []string{"location"},
        },
    },
}

resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: messages,
    Tools:    tools,
}, nil)
```

### 4. 多模态输入

```go
resp, err := model.Generate(ctx, &ai.ModelRequest{
    Messages: []*ai.Message{
        {
            Role: ai.RoleUser,
            Content: []*ai.Part{
                ai.NewTextPart("What's in this image?"),
                ai.NewMediaPart("image/jpeg", "https://example.com/image.jpg"),
            },
        },
    },
}, nil)
```

### 5. 文本嵌入

```go
embedder := azurePlugin.DefineEmbedder("azure", "text-embedding-ada-002", nil)

resp, err := embedder.Embed(ctx, &ai.EmbedRequest{
    Documents: []*ai.Document{
        ai.DocumentFromText("Hello, world!", nil),
    },
}, nil)
```

## 错误处理

插件提供了详细的错误处理：

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    // 检查错误类型
    if azureErr, ok := err.(*azure.AzureAIError); ok {
        switch azureErr.Type {
        case "config":
            // 配置错误
        case "network":
            // 网络错误
        case "api":
            // API 错误
            fmt.Printf("API Error: %s (Code: %s)\n", azureErr.Message, azureErr.Code)
        case "parse":
            // 响应解析错误
        }
    }
}
```

## 监控和日志

插件会自动记录详细的日志信息：

- 请求开始和完成
- API 调用详情（脱敏后的 API Key）
- 错误信息和重试尝试
- Token 使用统计
- 响应时间

日志示例：
```
INFO  初始化 Azure AI 提供商（原生插件） provider=azure model=gpt-4
INFO  Azure AI 提供商初始化成功（原生插件） provider=azure fullModelName=azure/gpt-4 apiVersion=2025-04-01-preview
```

## 迁移指南

### 从 `azureopenai` 迁移到 `azure`

1. 更新模型配置中的 `modelProvider` 字段：
   ```json
   {
     "modelProvider": "azure"  // 原来是 "azureopenai"
   }
   ```

2. 更新 `baseUrl`（如果需要）：
   - 原生插件会自动处理端点路径
   - 不需要在 URL 中包含 `/openai/deployments/{deployment-name}`

3. 更新 API 版本（如果需要）：
   ```json
   {
     "queryParams": {
       "azureApiVersion": "2025-04-01-preview"
     }
   }
   ```

4. 测试配置：
   ```bash
   POST /api/v1/model-configurations/{id}/validate
   ```

## 故障排除

### 1. 配置验证失败

**问题：** 创建模型配置时返回 400 错误

**解决方案：**
- 检查 `baseUrl` 格式是否正确
- 确保 API Key 有效
- 验证 API 版本是否支持

### 2. 请求超时

**问题：** 请求经常超时

**解决方案：**
- 检查网络连接
- 增加超时时间（通过自定义 `TimeoutConfig`）
- 检查 Azure 服务状态

### 3. 重试失败

**问题：** 请求失败后没有重试

**解决方案：**
- 检查错误类型（只有特定错误会重试）
- 查看日志中的重试信息
- 自定义重试配置（通过 `RetryConfig`）

## 性能优化

### 1. 连接池

插件使用 HTTP 连接池来提高性能：
- 最大空闲连接数：100
- 每个主机的最大空闲连接数：10
- 空闲连接超时：90 秒

### 2. 缓存

Genkit 客户端会缓存已初始化的提供商实例：
- 缓存键：`{tenantId}_{modelName}`
- 自动清理：配置更新时
- 手动清理：调用 `ClearCache()` 方法

### 3. 批量嵌入

嵌入请求支持批量处理：
- 建议批量大小：10-100 个文本
- 减少 API 调用次数
- 提高吞吐量

## 安全考虑

### 1. API Key 管理

- API Key 在数据库中加密存储
- 日志中自动脱敏
- 不在错误信息中暴露

### 2. 数据隐私

- 遵循 Azure OpenAI 的数据处理政策
- 不发送敏感个人信息
- 考虑使用 Azure 的私有端点

### 3. 访问控制

- 基于租户的隔离
- 角色基础的访问控制（RBAC）
- 审计日志记录

## 参考资料

- [Azure OpenAI 官方文档](https://learn.microsoft.com/azure/ai-services/openai/)
- [Azure OpenAI Responses API](https://learn.microsoft.com/azure/ai-services/openai/reference)
- [Genkit 官方文档](https://firebase.google.com/docs/genkit)
- [插件 README](./README.md)
- [错误处理文档](./ERROR_HANDLING.md)
- [重试和超时文档](./RETRY_AND_TIMEOUT.md)

## 更新日志

### 2025-01-10
- ✅ 完成插件集成到现有项目
- ✅ 添加 `azure` 提供商类型支持
- ✅ 保留 `azureopenai` 向后兼容
- ✅ 更新文档和示例

### 2025-01-09
- ✅ 完成插件核心功能开发
- ✅ 添加重试和超时机制
- ✅ 完成文档和示例

## 支持

如有问题或建议，请：
1. 查看文档和示例
2. 检查日志信息
3. 提交 Issue 或 Pull Request
