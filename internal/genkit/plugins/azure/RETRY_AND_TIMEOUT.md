# Azure AI Plugin - 重试和超时机制

本文档描述了 Azure AI Genkit Provider 插件的重试和超时机制。

## 概述

插件实现了以下功能：

1. **指数退避重试策略** - 针对 429 和 5xx 错误自动重试
2. **HTTP 客户端超时配置** - 配置各种超时参数
3. **请求上下文超时控制** - 支持基于上下文的超时

## 重试机制

### 默认重试配置

插件使用以下默认重试配置：

```go
retryConfig := &RetryConfig{
    MaxRetries:         3,                    // 最大重试次数
    InitialBackoff:     1 * time.Second,      // 初始退避时间
    MaxBackoff:         32 * time.Second,     // 最大退避时间
    BackoffMultiplier:  2.0,                  // 退避时间倍数
    RetryableStatusCodes: map[int]bool{
        http.StatusTooManyRequests:     true, // 429
        http.StatusInternalServerError: true, // 500
        http.StatusBadGateway:          true, // 502
        http.StatusServiceUnavailable:  true, // 503
        http.StatusGatewayTimeout:      true, // 504
    },
}
```

### 指数退避算法

重试使用指数退避算法，退避时间按以下公式计算：

```
backoff = InitialBackoff * (BackoffMultiplier ^ attempt)
```

例如，使用默认配置：
- 第 1 次重试：1 秒
- 第 2 次重试：2 秒
- 第 3 次重试：4 秒
- 第 4 次重试：8 秒

退避时间不会超过 `MaxBackoff`（默认 32 秒）。

### 可重试的错误

以下 HTTP 状态码会触发自动重试：

- **429 Too Many Requests** - 速率限制
- **500 Internal Server Error** - 服务器内部错误
- **502 Bad Gateway** - 网关错误
- **503 Service Unavailable** - 服务不可用
- **504 Gateway Timeout** - 网关超时

其他状态码（如 400、401、403、404）不会触发重试。

## 超时配置

### 默认超时配置

插件使用以下默认超时配置：

```go
timeoutConfig := &TimeoutConfig{
    RequestTimeout:      30 * time.Second,  // 单个请求超时
    StreamTimeout:       60 * time.Second,  // 流式请求超时
    DialTimeout:         10 * time.Second,  // 连接超时
    TLSHandshakeTimeout: 10 * time.Second,  // TLS 握手超时
    IdleConnTimeout:     90 * time.Second,  // 空闲连接超时
}
```

### 超时类型说明

- **RequestTimeout** - 整个请求的最大执行时间（包括重试）
- **StreamTimeout** - 流式请求的最大执行时间
- **DialTimeout** - 建立 TCP 连接的超时时间
- **TLSHandshakeTimeout** - TLS 握手的超时时间
- **IdleConnTimeout** - 连接池中空闲连接的保持时间

## 使用示例

### 基本使用（使用默认配置）

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
    
    // 创建 Azure AI Provider（使用默认重试和超时配置）
    azurePlugin := &azure.AzureAI{
        APIKey:     "your-api-key",
        BaseURL:    "https://your-resource.openai.azure.com",
        APIVersion: "2025-04-01-preview",
        Provider:   "azure",
    }
    
    // 初始化插件
    g := genkit.New(ctx, nil)
    g.RegisterPlugin(azurePlugin)
    
    // 定义模型
    model := azurePlugin.DefineModel("azure", "gpt-4", azure.ModelOptions{
        Label: "GPT-4",
        Supports: &azure.Multimodal,
    })
    
    // 使用模型（自动重试和超时）
    resp, err := model.Generate(ctx, &genkit.GenerateRequest{
        Messages: []*genkit.Message{
            {Role: "user", Content: []*genkit.Part{genkit.NewTextPart("Hello!")}},
        },
    }, nil)
    
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

### 自定义重试配置

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
    ctx := context.Background()
    
    // 自定义重试配置
    retryConfig := &azure.RetryConfig{
        MaxRetries:         5,                    // 增加重试次数
        InitialBackoff:     500 * time.Millisecond, // 减少初始退避时间
        MaxBackoff:         60 * time.Second,     // 增加最大退避时间
        BackoffMultiplier:  2.0,
        RetryableStatusCodes: map[int]bool{
            http.StatusTooManyRequests:     true,
            http.StatusInternalServerError: true,
            http.StatusBadGateway:          true,
            http.StatusServiceUnavailable:  true,
            http.StatusGatewayTimeout:      true,
        },
    }
    
    // 创建 Azure AI Provider
    azurePlugin := &azure.AzureAI{
        APIKey:      "your-api-key",
        BaseURL:     "https://your-resource.openai.azure.com",
        APIVersion:  "2025-04-01-preview",
        Provider:    "azure",
        RetryConfig: retryConfig, // 使用自定义重试配置
    }
    
    // 初始化并使用...
    g := genkit.New(ctx, nil)
    g.RegisterPlugin(azurePlugin)
}
```

### 自定义超时配置

```go
package main

import (
    "context"
    "time"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
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
        APIKey:        "your-api-key",
        BaseURL:       "https://your-resource.openai.azure.com",
        APIVersion:    "2025-04-01-preview",
        Provider:      "azure",
        TimeoutConfig: timeoutConfig, // 使用自定义超时配置
    }
    
    // 初始化并使用...
    g := genkit.New(ctx, nil)
    g.RegisterPlugin(azurePlugin)
}
```

### 使用上下文超时

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
    // 创建带超时的上下文（10 秒）
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // 创建 Azure AI Provider
    azurePlugin := &azure.AzureAI{
        APIKey:     "your-api-key",
        BaseURL:    "https://your-resource.openai.azure.com",
        APIVersion: "2025-04-01-preview",
        Provider:   "azure",
    }
    
    g := genkit.New(context.Background(), nil)
    g.RegisterPlugin(azurePlugin)
    
    model := azurePlugin.DefineModel("azure", "gpt-4", azure.ModelOptions{
        Label: "GPT-4",
        Supports: &azure.Multimodal,
    })
    
    // 使用带超时的上下文
    resp, err := model.Generate(ctx, &genkit.GenerateRequest{
        Messages: []*genkit.Message{
            {Role: "user", Content: []*genkit.Part{genkit.NewTextPart("Hello!")}},
        },
    }, nil)
    
    if err != nil {
        if err == context.DeadlineExceeded {
            fmt.Println("请求超时")
        } else {
            fmt.Printf("请求失败: %v\n", err)
        }
        return
    }
    
    fmt.Println(resp.Message.Content[0].Text)
}
```

### 禁用重试

```go
package main

import (
    "context"
    
    "github.com/firebase/genkit/go/genkit"
    "github.com/your-org/genkit-ai-service/internal/genkit/plugins/azure"
)

func main() {
    ctx := context.Background()
    
    // 禁用重试
    retryConfig := &azure.RetryConfig{
        MaxRetries: 0, // 设置为 0 禁用重试
    }
    
    // 创建 Azure AI Provider
    azurePlugin := &azure.AzureAI{
        APIKey:      "your-api-key",
        BaseURL:     "https://your-resource.openai.azure.com",
        APIVersion:  "2025-04-01-preview",
        Provider:    "azure",
        RetryConfig: retryConfig,
    }
    
    // 初始化并使用...
    g := genkit.New(ctx, nil)
    g.RegisterPlugin(azurePlugin)
}
```

## 错误处理

### 重试失败

当所有重试都失败后，插件会返回一个包含重试信息的错误：

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    // 检查是否是网络错误
    if azErr, ok := err.(*azure.AzureAIError); ok {
        if azErr.Type == "network" {
            fmt.Printf("网络错误: %s\n", azErr.Message)
            // 可能包含 "请求失败，已重试 N 次" 的信息
        }
    }
}
```

### 超时错误

当请求超时时，会返回上下文超时错误：

```go
resp, err := model.Generate(ctx, req, nil)
if err != nil {
    if err == context.DeadlineExceeded {
        fmt.Println("请求超时")
    } else if err == context.Canceled {
        fmt.Println("请求被取消")
    }
}
```

## 最佳实践

### 1. 根据场景调整重试配置

- **生产环境**：使用默认配置或增加重试次数
- **开发环境**：可以减少重试次数和退避时间以加快调试
- **批量处理**：增加重试次数和最大退避时间

### 2. 合理设置超时时间

- **简单查询**：使用较短的超时时间（10-30 秒）
- **复杂生成**：使用较长的超时时间（60-120 秒）
- **流式响应**：使用更长的超时时间（120-300 秒）

### 3. 使用上下文超时

对于用户交互场景，建议使用上下文超时来控制请求时间：

```go
// 为每个请求设置独立的超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := model.Generate(ctx, req, nil)
```

### 4. 监控重试和超时

在生产环境中，建议监控以下指标：

- 重试次数和频率
- 超时发生的频率
- 平均响应时间
- 错误类型分布

### 5. 处理速率限制

对于 429 错误（速率限制），插件会自动重试。但如果频繁遇到速率限制，建议：

- 增加初始退避时间
- 实现请求队列
- 使用多个 API 密钥进行负载均衡

## 技术细节

### 重试流程

1. 发送 HTTP 请求
2. 检查响应状态码
3. 如果状态码可重试且未超过最大重试次数：
   - 计算退避时间（指数退避）
   - 等待退避时间
   - 重新发送请求
4. 如果所有重试都失败，返回最后一次的响应或错误

### 上下文取消

在重试过程中，插件会检查上下文是否已取消：

- 在每次重试前检查上下文
- 在退避等待期间检查上下文
- 如果上下文已取消，立即返回错误

### 连接池管理

插件使用 HTTP 连接池来提高性能：

- `MaxIdleConns`: 100 - 最大空闲连接数
- `MaxIdleConnsPerHost`: 10 - 每个主机的最大空闲连接数
- `IdleConnTimeout`: 90 秒 - 空闲连接超时时间

## 故障排查

### 问题：请求总是超时

**可能原因**：
- 超时时间设置过短
- 网络延迟较高
- Azure OpenAI 服务响应慢

**解决方案**：
- 增加 `RequestTimeout`
- 检查网络连接
- 联系 Azure 支持

### 问题：频繁遇到 429 错误

**可能原因**：
- 请求速率超过配额
- 多个客户端共享同一个 API 密钥

**解决方案**：
- 增加退避时间
- 实现请求限流
- 升级 Azure OpenAI 配额

### 问题：重试次数过多

**可能原因**：
- Azure OpenAI 服务不稳定
- 网络连接不稳定

**解决方案**：
- 检查 Azure 服务状态
- 检查网络连接
- 考虑使用备用区域

## 参考资料

- [Azure OpenAI Service Documentation](https://learn.microsoft.com/azure/cognitive-services/openai/)
- [HTTP Retry Best Practices](https://docs.microsoft.com/azure/architecture/best-practices/retry-service-specific)
- [Go Context Package](https://pkg.go.dev/context)
