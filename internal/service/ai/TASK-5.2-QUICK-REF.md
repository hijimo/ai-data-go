# TASK-5.2 快速参考：从 ChatOptions 提取 ModelName

## 任务状态

✅ **已完成** - 2025-11-28

## 实现内容

### 1. 模型名称提取逻辑

在 `Chat` 和 `ChatStream` 方法中添加：

```go
// 从请求中获取模型名称
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

### 2. 调用 Genkit Client

```go
// 非流式调用
result, err := s.client.Generate(sessionCtx, tenantID, modelName, req.Message, options)

// 流式调用
genkitStream, err := s.client.GenerateStream(sessionCtx, tenantID, modelName, req.Message, options)
```

## 验收标准

- [x] 从上下文获取当前租户ID
- [x] 从 `ChatOptions` 中提取 `ModelName` 字段
- [x] 修改 `Generate()` 调用，传递 tenantID 和 modelName
- [x] 修改 `GenerateStream()` 调用，传递 tenantID 和 modelName
- [x] 添加日志记录（包含租户ID和模型名称）
- [x] 添加错误处理
- [x] 保持向后兼容

## 测试用例

1. **TestChat_WithModelName** - 指定模型名称
2. **TestChat_WithoutModelName** - 使用默认模型
3. **TestChat_WithEmptyModelName** - 空模型名称
4. **TestChatStream_WithModelName** - 流式调用指定模型

## API 使用示例

### 指定模型

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好",
  "options": {
    "modelName": "gpt-4"
  }
}
```

### 使用默认模型

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好"
}
```

## 相关文件

- `internal/service/ai/genkit_service.go` - 主要实现
- `internal/service/ai/genkit_service_test.go` - 测试用例
- `internal/model/request.go` - ChatOptions 定义

## 下一步

继续 TASK-5.4：更新应用初始化逻辑
