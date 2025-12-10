# 任务 12 实现总结：添加重试和超时机制

## 完成时间
2025-12-10

## 任务描述
实现指数退避重试策略（针对 429 和 5xx 错误）、配置 HTTP 客户端超时、实现请求上下文超时控制。

## 实现内容

### 1. 创建重试机制 (`retry.go`)

#### RetryConfig 结构
- `MaxRetries`: 最大重试次数（默认 3 次）
- `InitialBackoff`: 初始退避时间（默认 1 秒）
- `MaxBackoff`: 最大退避时间（默认 32 秒）
- `BackoffMultiplier`: 退避时间倍数（默认 2.0）
- `RetryableStatusCodes`: 可重试的 HTTP 状态码映射

#### 可重试的状态码
- 429 Too Many Requests
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout

#### RetryableHTTPClient
- 包装标准 HTTP 客户端
- 实现自动重试逻辑
- 支持指数退避算法
- 支持上下文取消

#### 指数退避算法
```
backoff = InitialBackoff * (BackoffMultiplier ^ attempt)
```
退避时间不超过 MaxBackoff。

### 2. 创建超时配置

#### TimeoutConfig 结构
- `RequestTimeout`: 单个请求超时（默认 30 秒）
- `StreamTimeout`: 流式请求超时（默认 60 秒）
- `DialTimeout`: 连接超时（默认 10 秒）
- `TLSHandshakeTimeout`: TLS 握手超时（默认 10 秒）
- `IdleConnTimeout`: 空闲连接超时（默认 90 秒）

#### NewHTTPClientWithTimeout
创建配置了超时参数的 HTTP 客户端。

#### WithTimeout 辅助函数
为请求添加超时上下文。

### 3. 更新 AzureAI 插件

#### 新增字段
- `retryableClient`: 支持重试的 HTTP 客户端
- `RetryConfig`: 重试配置（可选）
- `TimeoutConfig`: 超时配置（可选）

#### Init 方法更新
- 设置默认超时配置
- 设置默认重试配置
- 创建配置了超时的 HTTP 客户端
- 创建支持重试的 HTTP 客户端包装器

### 4. 更新 ModelGenerator

#### 新增字段
- `retryableClient`: 支持重试的 HTTP 客户端

#### 新增方法
- `NewModelGeneratorWithRetry`: 创建支持重试的生成器

#### 更新请求发送逻辑
- 优先使用 `retryableClient`（如果可用）
- 回退到标准 `client`（向后兼容）
- 同时支持流式和非流式请求

### 5. 更新嵌入器

#### generateEmbeddings 函数更新
- 接受 `interface{}` 类型的客户端参数
- 支持 `RetryableHTTPClient` 和 `http.Client`
- 使用类型断言选择正确的客户端

### 6. 测试覆盖

#### 重试配置测试
- `TestRetryConfig_shouldRetry`: 测试重试判断逻辑
- `TestRetryConfig_calculateBackoff`: 测试退避时间计算

#### 重试客户端测试
- `TestRetryableHTTPClient_Do_Success`: 测试成功请求
- `TestRetryableHTTPClient_Do_RetryOn429`: 测试 429 错误重试
- `TestRetryableHTTPClient_Do_RetryOn500`: 测试 500 错误重试
- `TestRetryableHTTPClient_Do_NoRetryOn400`: 测试 400 错误不重试
- `TestRetryableHTTPClient_Do_MaxRetriesExceeded`: 测试超过最大重试次数
- `TestRetryableHTTPClient_Do_ContextCancellation`: 测试上下文取消
- `TestRetryableHTTPClient_Do_ExponentialBackoff`: 测试指数退避

#### 超时配置测试
- `TestNewHTTPClientWithTimeout`: 测试创建超时客户端
- `TestWithTimeout`: 测试添加超时上下文

#### 基准测试
- `BenchmarkRetryableHTTPClient_Do`: 重试客户端性能测试

### 7. 文档

创建了 `RETRY_AND_TIMEOUT.md` 文档，包含：
- 重试机制详细说明
- 超时配置详细说明
- 使用示例（基本、自定义配置、上下文超时等）
- 错误处理指南
- 最佳实践
- 故障排查指南

## 技术亮点

### 1. 指数退避算法
使用标准的指数退避算法，避免在服务器压力大时加剧问题。

### 2. 上下文感知
在重试过程中持续检查上下文状态，支持及时取消。

### 3. 灵活配置
支持自定义重试和超时配置，满足不同场景需求。

### 4. 向后兼容
保留了对标准 HTTP 客户端的支持，不破坏现有代码。

### 5. 类型安全
使用类型断言安全地处理不同类型的客户端。

## 测试结果

所有测试通过：
```
=== RUN   TestRetryConfig_shouldRetry
--- PASS: TestRetryConfig_shouldRetry (0.00s)
=== RUN   TestRetryConfig_calculateBackoff
--- PASS: TestRetryConfig_calculateBackoff (0.00s)
=== RUN   TestRetryableHTTPClient_Do_Success
--- PASS: TestRetryableHTTPClient_Do_Success (0.00s)
=== RUN   TestRetryableHTTPClient_Do_RetryOn429
--- PASS: TestRetryableHTTPClient_Do_RetryOn429 (0.03s)
=== RUN   TestRetryableHTTPClient_Do_RetryOn500
--- PASS: TestRetryableHTTPClient_Do_RetryOn500 (0.01s)
=== RUN   TestRetryableHTTPClient_Do_NoRetryOn400
--- PASS: TestRetryableHTTPClient_Do_NoRetryOn400 (0.00s)
=== RUN   TestRetryableHTTPClient_Do_MaxRetriesExceeded
--- PASS: TestRetryableHTTPClient_Do_MaxRetriesExceeded (0.03s)
=== RUN   TestRetryableHTTPClient_Do_ContextCancellation
--- PASS: TestRetryableHTTPClient_Do_ContextCancellation (0.05s)
=== RUN   TestNewHTTPClientWithTimeout
--- PASS: TestNewHTTPClientWithTimeout (0.00s)
=== RUN   TestWithTimeout
--- PASS: TestWithTimeout (0.00s)
=== RUN   TestRetryableHTTPClient_Do_ExponentialBackoff
--- PASS: TestRetryableHTTPClient_Do_ExponentialBackoff (0.70s)
```

## 使用示例

### 基本使用（默认配置）
```go
azurePlugin := &azure.AzureAI{
    APIKey:     "your-api-key",
    BaseURL:    "https://your-resource.openai.azure.com",
    APIVersion: "2025-04-01-preview",
    Provider:   "azure",
    // 使用默认重试和超时配置
}
```

### 自定义重试配置
```go
retryConfig := &azure.RetryConfig{
    MaxRetries:         5,
    InitialBackoff:     500 * time.Millisecond,
    MaxBackoff:         60 * time.Second,
    BackoffMultiplier:  2.0,
    RetryableStatusCodes: map[int]bool{
        http.StatusTooManyRequests:     true,
        http.StatusInternalServerError: true,
    },
}

azurePlugin := &azure.AzureAI{
    APIKey:      "your-api-key",
    BaseURL:     "https://your-resource.openai.azure.com",
    RetryConfig: retryConfig,
}
```

### 自定义超时配置
```go
timeoutConfig := &azure.TimeoutConfig{
    RequestTimeout:      60 * time.Second,
    StreamTimeout:       120 * time.Second,
    DialTimeout:         5 * time.Second,
    TLSHandshakeTimeout: 5 * time.Second,
    IdleConnTimeout:     120 * time.Second,
}

azurePlugin := &azure.AzureAI{
    APIKey:        "your-api-key",
    BaseURL:       "https://your-resource.openai.azure.com",
    TimeoutConfig: timeoutConfig,
}
```

### 使用上下文超时
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := model.Generate(ctx, req, nil)
if err == context.DeadlineExceeded {
    fmt.Println("请求超时")
}
```

## 性能影响

### 重试开销
- 成功请求：无额外开销
- 失败请求：增加退避等待时间
- 最坏情况：3 次重试 + 退避时间（约 7 秒）

### 内存开销
- 每个插件实例增加约 200 字节（配置结构）
- 重试客户端包装器：约 100 字节

### 连接池优化
- 复用 HTTP 连接
- 减少 TLS 握手次数
- 提高并发性能

## 后续改进建议

1. **自适应重试**：根据历史成功率动态调整重试策略
2. **断路器模式**：在服务持续失败时快速失败
3. **重试指标**：收集和暴露重试相关的监控指标
4. **重试回调**：允许用户注册重试事件的回调函数
5. **请求去重**：避免重复发送相同的请求

## 相关文件

- `internal/genkit/plugins/azure/retry.go` - 重试和超时实现
- `internal/genkit/plugins/azure/retry_test.go` - 测试文件
- `internal/genkit/plugins/azure/azure.go` - 插件主文件（已更新）
- `internal/genkit/plugins/azure/generate.go` - 生成器（已更新）
- `internal/genkit/plugins/azure/embed.go` - 嵌入器（已更新）
- `internal/genkit/plugins/azure/RETRY_AND_TIMEOUT.md` - 使用文档

## 验证需求

任务要求：
- ✅ 实现指数退避重试策略（针对 429 和 5xx 错误）
- ✅ 配置 HTTP 客户端超时
- ✅ 实现请求上下文超时控制

所有要求均已实现并通过测试。
