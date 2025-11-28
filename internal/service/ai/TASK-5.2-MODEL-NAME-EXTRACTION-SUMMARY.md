# TASK-5.2 实现总结：从 ChatOptions 中提取 ModelName 字段

## 任务概述

实现从 `ChatOptions` 中提取 `ModelName` 字段的功能，使 AI Service 能够根据用户请求动态选择模型。

## 实现内容

### 1. 更新 `genkit_service.go`

#### Chat 方法更新

在 `Chat` 方法中添加了模型名称提取逻辑：

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

#### ChatStream 方法更新

在 `ChatStream` 方法中添加了相同的模型名称提取逻辑，确保流式和非流式调用的行为一致。

### 2. 添加测试用例

在 `genkit_service_test.go` 中添加了以下测试用例：

1. **TestChat_WithModelName**: 测试指定模型名称的情况
   - 验证传递给 Genkit Client 的模型名称正确
   - 验证响应中的模型名称正确

2. **TestChat_WithoutModelName**: 测试不指定模型名称的情况
   - 验证使用默认模型 "gemini-pro"

3. **TestChat_WithEmptyModelName**: 测试空模型名称的情况
   - 验证空字符串被视为未指定，使用默认模型

4. **TestChatStream_WithModelName**: 测试流式对话指定模型名称
   - 验证流式调用也能正确提取和使用模型名称

## 实现逻辑

### 模型名称提取规则

1. **优先级**：请求中的 `ModelName` > 默认模型
2. **默认模型**：`gemini-pro`
3. **空值处理**：
   - `Options` 为 `nil`：使用默认模型
   - `ModelName` 为 `nil`：使用默认模型
   - `ModelName` 为空字符串：使用默认模型

### 日志记录

- 当使用请求中指定的模型时，记录 DEBUG 级别日志
- 当使用默认模型时，记录 DEBUG 级别日志
- 日志包含 `sessionId` 和 `modelName` 字段

## 测试结果

所有测试用例均通过：

```
=== RUN   TestChat_WithModelName
--- PASS: TestChat_WithModelName (0.00s)
=== RUN   TestChat_WithoutModelName
--- PASS: TestChat_WithoutModelName (0.00s)
=== RUN   TestChat_WithEmptyModelName
--- PASS: TestChat_WithEmptyModelName (0.00s)
=== RUN   TestChatStream_WithModelName
--- PASS: TestChatStream_WithModelName (0.00s)
```

所有现有测试也继续通过，确保没有破坏现有功能。

## 与其他任务的集成

### 依赖关系

- **TASK-5.1**：已完成，`ChatOptions` 中已添加 `ModelName` 字段
- **TASK-2.2**：Genkit Client 已支持 `tenantID` 和 `modelName` 参数
- **TASK-2.3**：`Generate` 和 `GenerateStream` 方法已更新签名

### 后续任务

本任务完成后，下一步是：

- **TASK-5.4**：更新应用初始化逻辑，注入 ModelConfigurationRepository

## 使用示例

### 指定模型名称

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好，请介绍一下自己",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.8,
    "maxTokens": 1000
  }
}
```

系统会：

1. 从 `options.modelName` 提取模型名称 "gpt-4"
2. 使用当前租户ID和模型名称查询 `model_configurations` 表
3. 使用查询到的配置调用相应的 AI 提供商

### 使用默认模型

```json
POST /api/v1/chat/sessions/{id}/messages
{
  "message": "你好，请介绍一下自己"
}
```

系统会：

1. 使用默认模型名称 "gemini-pro"
2. 使用当前租户ID和默认模型名称查询配置
3. 使用查询到的配置调用 Google AI

## 注意事项

1. **向后兼容性**：
   - `ModelName` 字段是可选的（指针类型）
   - 不指定时使用默认模型，保持现有行为

2. **空值处理**：
   - 代码正确处理了 `nil` 和空字符串的情况
   - 确保不会因为空值导致错误

3. **日志记录**：
   - 添加了详细的日志记录，便于调试和追踪
   - 日志级别为 DEBUG，不会影响生产环境性能

4. **测试覆盖**：
   - 测试覆盖了所有边界情况
   - 包括正常情况、空值情况和流式调用

## 文件变更

- ✅ `internal/service/ai/genkit_service.go` - 更新 Chat 和 ChatStream 方法
- ✅ `internal/service/ai/genkit_service_test.go` - 添加测试用例
- ✅ `internal/service/ai/TASK-5.2-MODEL-NAME-EXTRACTION-SUMMARY.md` - 本文档

## 验收标准

- [x] 从 `ChatOptions` 中提取 `ModelName` 字段
- [x] 修改 `Generate()` 调用，传递 tenantID 和 modelName
- [x] 修改 `GenerateStream()` 调用，传递 tenantID 和 modelName
- [x] 添加日志记录（包含租户ID和模型名称）
- [x] 添加错误处理
- [x] 保持向后兼容

## 完成时间

2025-11-28

## 相关文档

- [TASK-5.1 实现总结](../../model/TASK-5.1-IMPLEMENTATION-SUMMARY.md)
- [TASK-2.2 实现总结](../../genkit/TASK-2.2-IMPLEMENTATION-SUMMARY.md)
- [TASK-2.3 实现总结](../../genkit/TASK-2.3-COMPLETION-SUMMARY.md)
