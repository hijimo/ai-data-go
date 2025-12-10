# Azure AI Plugin 错误处理文档

## 概述

Azure AI Plugin 实现了完整的错误处理机制，包括配置错误、请求错误、网络错误、API 错误和解析错误。所有错误都使用统一的 `AzureAIError` 类型进行包装，提供清晰的错误分类和详细的错误信息。

## 错误类型

### AzureAIError 结构

```go
type AzureAIError struct {
    // Type 错误类型：config, request, network, api, parse
    Type string
    
    // Code HTTP 状态码或错误代码
    Code string
    
    // Message 错误消息
    Message string
    
    // Details 错误详情
    Details any
    
    // Err 原始错误
    Err error
}
```

### 错误类型分类

#### 1. 配置错误 (config)

**使用场景**：
- 缺少必需的配置参数（API Key、Base URL）
- 配置参数格式无效
- 配置值超出允许范围

**创建方法**：
```go
err := NewConfigError("缺少 API Key", map[string]string{"field": "APIKey"})
```

**示例**：
```go
if a.APIKey == "" {
    return NewConfigError("APIKey 是必需的", map[string]string{
        "field": "APIKey",
        "hint": "请在初始化时提供有效的 Azure OpenAI API Key",
    })
}
```

#### 2. 请求错误 (request)

**使用场景**：
- 请求参数无效
- 消息格式错误
- 序列化失败
- 工具定义格式错误

**创建方法**：
```go
err := NewRequestError("消息列表不能为空", nil)
```

**示例**：
```go
if len(messages) == 0 {
    return nil, NewRequestError("至少需要一条消息", nil)
}

reqJSON, err := json.Marshal(reqBody)
if err != nil {
    return nil, NewRequestError("序列化请求体失败", err)
}
```

#### 3. 网络错误 (network)

**使用场景**：
- HTTP 请求发送失败
- 连接超时
- DNS 解析失败
- TLS 握手失败
- 读取响应体失败

**创建方法**：
```go
err := NewNetworkError("发送 HTTP 请求失败", originalErr)
```

**示例**：
```go
httpResp, err := client.Do(httpReq)
if err != nil {
    return nil, NewNetworkError("发送 HTTP 请求失败", err)
}

respBody, err := io.ReadAll(httpResp.Body)
if err != nil {
    return nil, NewNetworkError("读取响应体失败", err)
}
```

#### 4. API 错误 (api)

**使用场景**：
- HTTP 状态码非 2xx
- Azure OpenAI API 返回错误
- 认证失败（401）
- 速率限制（429）
- 服务器错误（5xx）

**创建方法**：
```go
err := NewAPIError("401", "未授权", errorDetails)
```

**示例**：
```go
if httpResp.StatusCode != http.StatusOK {
    var errResp ErrorResponse
    if err := json.Unmarshal(respBody, &errResp); err == nil {
        return NewAPIError(
            fmt.Sprintf("%d", httpResp.StatusCode),
            errResp.Error.Message,
            errResp.Error,
        )
    }
    return NewAPIError(
        fmt.Sprintf("%d", httpResp.StatusCode),
        fmt.Sprintf("API 请求失败: %s", string(respBody)),
        nil,
    )
}
```

#### 5. 解析错误 (parse)

**使用场景**：
- JSON 解析失败
- 响应格式不符合预期
- 缺少必需字段
- 类型转换失败

**创建方法**：
```go
err := NewParseError("解析响应 JSON 失败", originalErr)
```

**示例**：
```go
var azResp ResponsesResponse
if err := json.Unmarshal(respBody, &azResp); err != nil {
    return nil, NewParseError("解析响应 JSON 失败", err)
}

if len(azResp.Choices) == 0 {
    return nil, NewParseError("响应中没有选项", nil)
}
```

## 错误处理方法

### Error() 方法

实现 `error` 接口，返回格式化的错误消息：

```go
func (e *AzureAIError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Type, e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}
```

**输出示例**：
- 简单错误：`[config] invalid_config: 缺少 API Key`
- 带原始错误：`[network] network_error: 连接失败 (caused by: dial tcp: timeout)`

### Unwrap() 方法

支持 Go 1.13+ 的错误链功能：

```go
func (e *AzureAIError) Unwrap() error {
    return e.Err
}
```

**使用示例**：
```go
err := NewNetworkError("网络错误", originalErr)

// 使用 errors.Unwrap 获取原始错误
unwrapped := errors.Unwrap(err)

// 使用 errors.Is 检查错误类型
if errors.Is(err, originalErr) {
    // 处理特定错误
}
```

## 错误处理最佳实践

### 1. 配置验证

在插件初始化时验证所有必需配置：

```go
func (a *AzureAI) Init(ctx context.Context) error {
    if a.APIKey == "" {
        return NewConfigError("APIKey 是必需的", map[string]string{
            "field": "APIKey",
        })
    }
    if a.BaseURL == "" {
        return NewConfigError("BaseURL 是必需的", map[string]string{
            "field": "BaseURL",
        })
    }
    return nil
}
```

### 2. 请求构建

在构建请求时验证参数：

```go
func (g *ModelGenerator) Generate(ctx context.Context, req *ai.ModelRequest, cb func(...) error) (*ai.ModelResponse, error) {
    if len(g.messages) == 0 {
        return nil, NewRequestError("消息列表不能为空", nil)
    }
    
    reqJSON, err := json.Marshal(reqBody)
    if err != nil {
        return nil, NewRequestError("序列化请求体失败", err)
    }
    
    // ... 继续处理
}
```

### 3. 网络请求

处理所有可能的网络错误：

```go
httpResp, err := client.Do(httpReq)
if err != nil {
    return nil, NewNetworkError("发送 HTTP 请求失败", err)
}
defer httpResp.Body.Close()

respBody, err := io.ReadAll(httpResp.Body)
if err != nil {
    return nil, NewNetworkError("读取响应体失败", err)
}
```

### 4. API 响应

检查 HTTP 状态码并解析 API 错误：

```go
if httpResp.StatusCode != http.StatusOK {
    return nil, parseAPIError(httpResp.StatusCode, respBody)
}
```

### 5. 响应解析

验证响应格式并处理解析错误：

```go
var azResp ResponsesResponse
if err := json.Unmarshal(respBody, &azResp); err != nil {
    return nil, NewParseError("解析响应 JSON 失败", err)
}

if len(azResp.Choices) == 0 {
    return nil, NewParseError("响应中没有选项", nil)
}
```

## 错误恢复策略

### 重试策略

对于临时性错误（如网络超时、429 速率限制），应实现重试机制：

```go
func (g *ModelGenerator) generateWithRetry(ctx context.Context, maxRetries int) (*ai.ModelResponse, error) {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        resp, err := g.generate(ctx)
        if err == nil {
            return resp, nil
        }
        
        // 检查是否应该重试
        if azErr, ok := err.(*AzureAIError); ok {
            if azErr.Type == "api" && (azErr.Code == "429" || azErr.Code >= "500") {
                // 指数退避
                time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
                lastErr = err
                continue
            }
        }
        
        // 不可重试的错误，直接返回
        return nil, err
    }
    
    return nil, lastErr
}
```

### 降级策略

在流式响应失败时，可以回退到非流式模式：

```go
func (g *ModelGenerator) Generate(ctx context.Context, req *ai.ModelRequest, cb func(...) error) (*ai.ModelResponse, error) {
    if cb != nil {
        // 尝试流式响应
        resp, err := g.generateStream(ctx, req, cb)
        if err != nil {
            // 如果流式失败，回退到非流式
            if azErr, ok := err.(*AzureAIError); ok && azErr.Type == "network" {
                return g.generateNonStream(ctx, req)
            }
            return nil, err
        }
        return resp, nil
    }
    
    return g.generateNonStream(ctx, req)
}
```

## 错误日志记录

### 日志级别

根据错误类型选择适当的日志级别：

- **配置错误**：ERROR（阻止启动）
- **请求错误**：WARN（客户端错误）
- **网络错误**：ERROR（可能需要重试）
- **API 错误**：WARN（401、403、429）或 ERROR（5xx）
- **解析错误**：ERROR（数据格式问题）

### 日志示例

```go
import "log/slog"

func (g *ModelGenerator) Generate(ctx context.Context, req *ai.ModelRequest, cb func(...) error) (*ai.ModelResponse, error) {
    resp, err := g.generate(ctx, req, cb)
    if err != nil {
        if azErr, ok := err.(*AzureAIError); ok {
            switch azErr.Type {
            case "config":
                slog.ErrorContext(ctx, "配置错误",
                    "error", azErr.Error(),
                    "details", azErr.Details)
            case "network":
                slog.ErrorContext(ctx, "网络错误",
                    "error", azErr.Error(),
                    "code", azErr.Code)
            case "api":
                if azErr.Code >= "500" {
                    slog.ErrorContext(ctx, "API 服务器错误",
                        "error", azErr.Error(),
                        "code", azErr.Code)
                } else {
                    slog.WarnContext(ctx, "API 客户端错误",
                        "error", azErr.Error(),
                        "code", azErr.Code)
                }
            case "parse":
                slog.ErrorContext(ctx, "解析错误",
                    "error", azErr.Error())
            }
        }
        return nil, err
    }
    return resp, nil
}
```

## 测试

错误处理的测试覆盖了以下场景：

1. **错误创建**：测试所有错误类型的创建函数
2. **错误格式化**：测试 Error() 方法的输出格式
3. **错误链**：测试 Unwrap() 方法和错误链功能
4. **错误详情**：测试 Details 字段的使用
5. **错误包装**：测试多层错误包装

运行测试：

```bash
go test -v -run TestError ./internal/genkit/plugins/azure
```

## 常见错误场景

### 1. 认证失败

```
[api] 401: 未授权
Details: {
    "error": {
        "message": "Invalid API key",
        "type": "invalid_request_error",
        "code": "invalid_api_key"
    }
}
```

**解决方案**：检查 API Key 是否正确，是否已过期。

### 2. 速率限制

```
[api] 429: 请求过于频繁
Details: {
    "error": {
        "message": "Rate limit exceeded",
        "type": "rate_limit_error"
    }
}
```

**解决方案**：实现指数退避重试，或降低请求频率。

### 3. 网络超时

```
[network] network_error: 发送 HTTP 请求失败 (caused by: context deadline exceeded)
```

**解决方案**：增加超时时间，或检查网络连接。

### 4. 响应格式错误

```
[parse] parse_error: 解析响应 JSON 失败 (caused by: unexpected end of JSON input)
```

**解决方案**：检查 API 版本是否正确，响应格式是否符合预期。

### 5. 配置缺失

```
[config] invalid_config: APIKey 是必需的
Details: {
    "field": "APIKey",
    "hint": "请在初始化时提供有效的 Azure OpenAI API Key"
}
```

**解决方案**：在初始化插件时提供所有必需的配置参数。

## 总结

Azure AI Plugin 的错误处理机制提供了：

1. **统一的错误类型**：所有错误都使用 `AzureAIError` 包装
2. **清晰的错误分类**：5 种错误类型覆盖所有场景
3. **详细的错误信息**：包含错误类型、代码、消息和详情
4. **错误链支持**：支持 Go 1.13+ 的错误链功能
5. **完整的测试覆盖**：所有错误场景都有对应的测试

通过遵循本文档的最佳实践，可以确保错误被正确处理和报告，提高系统的可靠性和可维护性。
