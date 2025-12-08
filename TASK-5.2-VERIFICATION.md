# TASK-5.2 验证报告：修改 AI Service 传递租户和模型参数

## 任务状态

✅ **已完成** - 所有验收标准均已满足

## 验收标准完成情况

### 1. ✅ 从上下文获取当前租户ID

**实现位置**: `internal/service/ai/genkit_service.go`

**Chat 方法实现**:

```go
// 从上下文中获取租户ID
claims, ok := authservice.GetJWTClaimsFromContext(ctx)
if !ok || claims == nil {
    s.logger.ErrorContext(ctx, "无法从上下文获取JWT Claims", logger.Fields{
        "sessionId": sessionID,
    })
    return nil, errors.NewUnauthorizedError("身份认证信息缺失")
}

tenantID := claims.TenantID
if tenantID == "" {
    s.logger.ErrorContext(ctx, "JWT Claims 中缺少租户ID", logger.Fields{
        "sessionId": sessionID,
        "userId":    claims.Subject,
    })
    return nil, errors.NewUnauthorizedError("租户信息缺失")
}
```

**ChatStream 方法实现**:

```go
// 从上下文中获取租户ID
claims, ok := authservice.GetJWTClaimsFromContext(ctx)
if !ok || claims == nil {
    s.logger.ErrorContext(ctx, "无法从上下文获取JWT Claims", logger.Fields{
        "sessionId": sessionID,
    })
    return nil, errors.NewUnauthorizedError("身份认证信息缺失")
}

tenantID := claims.TenantID
if tenantID == "" {
    s.logger.ErrorContext(ctx, "JWT Claims 中缺少租户ID", logger.Fields{
        "sessionId": sessionID,
        "userId":    claims.Subject,
    })
    return nil, errors.NewUnauthorizedError("租户信息缺失")
}
```

**验证结果**:

- ✅ 从上下文正确获取 JWT Claims
- ✅ 提取租户ID（claims.TenantID）
- ✅ 验证租户ID不为空
- ✅ 租户ID缺失时返回适当的错误

### 2. ✅ 从 `ChatOptions` 中提取 `ModelName` 字段

**实现位置**: `internal/service/ai/genkit_service.go`

**Chat 方法实现**:

```go
// 从请求中获取模型名称
// 如果请求中指定了模型名称，使用它；否则使用默认模型
modelName := "gemini-pro" // 默认模型
if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
    modelName = *req.Options.ModelName
    s.logger.DebugContext(ctx, "使用请求中指定的模型", logger.Fields{
        "sessionId": sessionID,
        "modelName": modelName,
    })
} else {
    s.logger.DebugContext(ctx, "使用默认模型", logger.Fields{
        "sessionId": sessionId,
        "modelName": modelName,
    })
}
```

**ChatStream 方法实现**:

```go
// 从请求中获取模型名称
// 如果请求中指定了模型名称，使用它；否则使用默认模型
modelName := "gemini-pro" // 默认模型
if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
    modelName = *req.Options.ModelName
    s.logger.DebugContext(ctx, "使用请求中指定的模型", logger.Fields{
        "sessionId": sessionID,
        "modelName": modelName,
    })
} else {
    s.logger.DebugContext(ctx, "使用默认模型", logger.Fields{
        "sessionId": sessionID,
        "modelName": modelName,
    })
}
```

**验证结果**:

- ✅ 检查 req.Options 是否为 nil
- ✅ 检查 req.Options.ModelName 是否为 nil
- ✅ 检查模型名称是否为空字符串
- ✅ 提供默认模型名称（gemini-pro）
- ✅ 记录使用的模型名称

### 3. ✅ 修改 `Generate()` 调用，传递 tenantID 和 modelName

**实现位置**: `internal/service/ai/genkit_service.go`

**实现代码**:

```go
// 调用 Genkit 生成响应
result, err := s.client.Generate(sessionCtx, tenantID, modelName, req.Message, options)
if err != nil {
    // 检查是否是上下文取消错误
    if sessionCtx.Err() == context.Canceled {
        s.logger.WarnContext(ctx, "对话请求被取消", logger.Fields{
            "sessionId": sessionID,
            "tenantId":  tenantID,
            "modelName": modelName,
            "error":     err.Error(),
        })
        return nil, errors.NewContextCancelledError()
    }

    s.logger.ErrorContext(ctx, "AI 生成失败", logger.Fields{
        "sessionId": sessionID,
        "tenantId":  tenantID,
        "modelName": modelName,
        "message":   req.Message,
        "error":     err.Error(),
    })
    return nil, errors.NewAIServiceError(err)
}
```

**验证结果**:

- ✅ 传递 sessionCtx（会话上下文）
- ✅ 传递 tenantID（租户ID）
- ✅ 传递 modelName（模型名称）
- ✅ 传递 req.Message（用户消息）
- ✅ 传递 options（生成选项）

### 4. ✅ 修改 `GenerateStream()` 调用，传递 tenantID 和 modelName

**实现位置**: `internal/service/ai/genkit_service.go`

**实现代码**:

```go
// 调用 Genkit 流式生成
genkitStream, err := s.client.GenerateStream(sessionCtx, tenantID, modelName, req.Message, options)
if err != nil {
    s.logger.ErrorContext(ctx, "启动流式生成失败", logger.Fields{
        "sessionId": sessionID,
        "tenantId":  tenantID,
        "modelName": modelName,
        "message":   req.Message,
        "error":     err.Error(),
    })
    return nil, errors.NewAIServiceError(err)
}
```

**验证结果**:

- ✅ 传递 sessionCtx（会话上下文）
- ✅ 传递 tenantID（租户ID）
- ✅ 传递 modelName（模型名称）
- ✅ 传递 req.Message（用户消息）
- ✅ 传递 options（生成选项）

### 5. ✅ 添加日志记录（包含租户ID和模型名称）

**实现位置**: `internal/service/ai/genkit_service.go`

**日志记录示例**:

1. **租户ID获取日志**:

```go
s.logger.DebugContext(ctx, "从上下文获取租户ID", logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "userId":    claims.Subject,
})
```

2. **模型选择日志**:

```go
s.logger.DebugContext(ctx, "使用请求中指定的模型", logger.Fields{
    "sessionId": sessionID,
    "modelName": modelName,
})
```

3. **成功完成日志（Chat）**:

```go
s.logger.InfoContext(sessionCtx, "对话请求处理完成", logger.Fields{
    "sessionId":        sessionID,
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            result.Model,
    "duration":         duration.String(),
    "promptTokens":     response.Usage.PromptTokens,
    "completionTokens": response.Usage.CompletionTokens,
    "totalTokens":      response.Usage.TotalTokens,
})
```

4. **成功完成日志（ChatStream）**:

```go
s.logger.InfoContext(sessionCtx, "流式对话请求处理完成", logger.Fields{
    "sessionId":        sessionID,
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            lastModel,
    "duration":         duration.String(),
    "promptTokens":     lastUsage.PromptTokens,
    "completionTokens": lastUsage.CompletionTokens,
    "totalTokens":      lastUsage.TotalTokens,
})
```

5. **错误日志**:

```go
s.logger.ErrorContext(ctx, "AI 生成失败", logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "modelName": modelName,
    "message":   req.Message,
    "error":     err.Error(),
})
```

**验证结果**:

- ✅ 所有关键操作都有日志记录
- ✅ 日志包含租户ID（tenantId）
- ✅ 日志包含模型名称（modelName）
- ✅ 日志包含会话ID（sessionId）
- ✅ 日志包含性能指标（duration）
- ✅ 日志包含 Token 使用统计
- ✅ 错误日志包含详细的错误信息

### 6. ✅ 添加错误处理

**实现位置**: `internal/service/ai/genkit_service.go`

**错误处理场景**:

1. **JWT Claims 缺失**:

```go
claims, ok := authservice.GetJWTClaimsFromContext(ctx)
if !ok || claims == nil {
    s.logger.ErrorContext(ctx, "无法从上下文获取JWT Claims", logger.Fields{
        "sessionId": sessionID,
    })
    return nil, errors.NewUnauthorizedError("身份认证信息缺失")
}
```

2. **租户ID缺失**:

```go
tenantID := claims.TenantID
if tenantID == "" {
    s.logger.ErrorContext(ctx, "JWT Claims 中缺少租户ID", logger.Fields{
        "sessionId": sessionID,
        "userId":    claims.Subject,
    })
    return nil, errors.NewUnauthorizedError("租户信息缺失")
}
```

3. **上下文取消错误（Chat）**:

```go
if sessionCtx.Err() == context.Canceled {
    s.logger.WarnContext(ctx, "对话请求被取消", logger.Fields{
        "sessionId": sessionID,
        "tenantId":  tenantID,
        "modelName": modelName,
        "error":     err.Error(),
    })
    return nil, errors.NewContextCancelledError()
}
```

4. **AI 生成失败（Chat）**:

```go
s.logger.ErrorContext(ctx, "AI 生成失败", logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "modelName": modelName,
    "message":   req.Message,
    "error":     err.Error(),
})
return nil, errors.NewAIServiceError(err)
```

5. **流式生成启动失败（ChatStream）**:

```go
s.logger.ErrorContext(ctx, "启动流式生成失败", logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "modelName": modelName,
    "message":   req.Message,
    "error":     err.Error(),
})
return nil, errors.NewAIServiceError(err)
```

6. **流式生成错误（ChatStream）**:

```go
if chunk.Error != nil {
    // 检查是否是上下文取消错误
    if sessionCtx.Err() == context.Canceled {
        s.logger.WarnContext(ctx, "流式对话请求被取消", logger.Fields{
            "sessionId":  sessionID,
            "tenantId":   tenantID,
            "modelName":  modelName,
            "chunkCount": len(fullContent),
        })
    } else {
        s.logger.ErrorContext(ctx, "流式生成出错", logger.Fields{
            "sessionId":  sessionID,
            "tenantId":   tenantID,
            "modelName":  modelName,
            "chunkCount": len(fullContent),
            "error":      chunk.Error.Error(),
            "errorType":  fmt.Sprintf("%T", chunk.Error),
        })
    }
    // ... 发送错误消息
}
```

**验证结果**:

- ✅ 处理 JWT Claims 缺失错误
- ✅ 处理租户ID缺失错误
- ✅ 处理上下文取消错误
- ✅ 处理 AI 生成失败错误
- ✅ 处理流式生成错误
- ✅ 所有错误都有详细的日志记录
- ✅ 返回适当的错误类型

### 7. ✅ 保持向后兼容

**向后兼容性验证**:

1. **默认模型支持**:

```go
modelName := "gemini-pro" // 默认模型
if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
    modelName = *req.Options.ModelName
}
```

- ✅ 如果不提供 modelName，使用默认模型
- ✅ 现有的 API 请求不需要修改

2. **Options 可选**:

```go
if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
    // 使用指定的模型
}
```

- ✅ req.Options 可以为 nil
- ✅ req.Options.ModelName 可以为 nil
- ✅ 模型名称可以为空字符串

3. **现有测试通过**:

```bash
$ go test ./internal/service/ai/... -v
PASS
ok      genkit-ai-service/internal/service/ai   0.688s
```

- ✅ 所有现有测试通过
- ✅ 没有破坏现有功能

## 测试验证

### 单元测试

```bash
$ go test ./internal/service/ai/... -v
=== RUN   TestChat_Success
--- PASS: TestChat_Success (0.00s)
=== RUN   TestChat_WithOptions
--- PASS: TestChat_WithOptions (0.00s)
=== RUN   TestChat_WithExistingSession
--- PASS: TestChat_WithExistingSession (0.00s)
=== RUN   TestChat_ContextCancelled
--- PASS: TestChat_ContextCancelled (0.00s)
...
PASS
ok      genkit-ai-service/internal/service/ai   0.688s
```

### 日志输出验证

测试日志显示正确记录了租户ID和模型名称：

```json
{
  "timestamp": "2025-12-07T12:41:30Z",
  "level": "INFO",
  "message": "对话请求处理完成",
  "fields": {
    "sessionId": "e4912f6f-cb9a-4953-8af4-76c53fcd8f50",
    "tenantId": "test-tenant-id",
    "modelName": "gemini-pro",
    "model": "test-model",
    "duration": "143.125µs",
    "promptTokens": 10,
    "completionTokens": 20,
    "totalTokens": 30
  }
}
```

## 实现文件

- ✅ `internal/service/ai/genkit_service.go` - AI Service 实现

## API 使用示例

### 示例 1: 使用默认模型

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好，请介绍一下你自己"
}
```

系统行为：

- 从上下文获取租户ID
- 使用默认模型 "gemini-pro"
- 调用 `client.Generate(ctx, tenantID, "gemini-pro", message, options)`

### 示例 2: 指定使用 GPT-4 模型

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好，请介绍一下你自己",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.8
  }
}
```

系统行为：

- 从上下文获取租户ID
- 从 options 中提取模型名称 "gpt-4"
- 调用 `client.Generate(ctx, tenantID, "gpt-4", message, options)`

### 示例 3: 流式调用指定模型

```json
POST /api/v1/chat/sessions/{id}/messages/stream
{
  "message": "请写一篇关于人工智能的文章",
  "options": {
    "modelName": "qwen-turbo",
    "temperature": 0.7,
    "maxTokens": 2000
  }
}
```

系统行为：

- 从上下文获取租户ID
- 从 options 中提取模型名称 "qwen-turbo"
- 调用 `client.GenerateStream(ctx, tenantID, "qwen-turbo", message, options)`

## 数据流

### Chat 方法数据流

```
用户请求
  ↓
Handler: 解析请求参数
  ↓
Service: Chat()
  ↓
1. 从上下文获取 JWT Claims
  ↓
2. 提取租户ID（claims.TenantID）
  ↓
3. 从 req.Options.ModelName 提取模型名称（或使用默认）
  ↓
4. 记录日志（包含 tenantId, modelName）
  ↓
5. 调用 client.Generate(ctx, tenantID, modelName, message, options)
  ↓
6. 处理响应或错误
  ↓
7. 记录完成日志（包含 tenantId, modelName, tokens, duration）
  ↓
Handler: 返回响应
```

### ChatStream 方法数据流

```
用户请求
  ↓
Handler: 设置 SSE 响应头
  ↓
Service: ChatStream()
  ↓
1. 从上下文获取 JWT Claims
  ↓
2. 提取租户ID（claims.TenantID）
  ↓
3. 从 req.Options.ModelName 提取模型名称（或使用默认）
  ↓
4. 记录日志（包含 tenantId, modelName）
  ↓
5. 调用 client.GenerateStream(ctx, tenantID, modelName, message, options)
  ↓
6. 在 goroutine 中处理流式响应
  ↓
7. 转换为腾讯云格式并发送
  ↓
8. 记录完成日志（包含 tenantId, modelName, tokens, duration）
  ↓
Handler: 关闭 SSE 连接
```

## 后续任务

TASK-5.2 已完成，为后续任务奠定了基础：

- **TASK-6.1**: 端到端测试
  - 测试 Google AI 端到端流程
  - 测试 Azure OpenAI 端到端流程
  - 测试百炼端到端流程
  - 测试提供商切换
  - 测试默认提供商逻辑
  - 测试错误场景

## 总结

TASK-5.2 已成功完成，所有验收标准均已满足：

1. ✅ **从上下文获取租户ID**: 正确从 JWT Claims 中提取租户ID
2. ✅ **提取模型名称**: 从 ChatOptions 中提取 ModelName，支持默认值
3. ✅ **修改 Generate 调用**: 传递 tenantID 和 modelName
4. ✅ **修改 GenerateStream 调用**: 传递 tenantID 和 modelName
5. ✅ **添加日志记录**: 所有关键操作都记录了租户ID和模型名称
6. ✅ **添加错误处理**: 完整的错误处理和日志记录
7. ✅ **保持向后兼容**: 支持默认模型，不破坏现有功能

该实现使得系统能够根据租户ID和模型名称动态选择 AI 提供商，为多模型支持功能提供了完整的服务层支持。

## 验证时间

2025-12-07

## 验证人

Kiro AI Assistant
