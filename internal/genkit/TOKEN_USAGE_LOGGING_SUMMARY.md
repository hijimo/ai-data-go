# Token 使用统计日志记录实现总结

## 概述

本次实现完善了 Genkit 多模型支持中的 Token 使用统计日志记录功能，确保所有 AI 调用都能详细记录 Token 使用情况，便于监控、分析和成本控制。

## 实现内容

### 1. Genkit Client 层（internal/genkit/client.go）

#### 非流式调用（Generate 方法）

已实现完整的 Token 使用统计记录：

```go
logger.InfoContext(ctx, "生成内容成功", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "promptTokens":     result.Usage.PromptTokens,      // 提示词 token 数
    "completionTokens": result.Usage.CompletionTokens,  // 生成内容 token 数
    "totalTokens":      result.Usage.TotalTokens,       // 总 token 数
    "responseLen":      len(result.Text),
})
```

#### 流式调用（GenerateStream 方法）

已实现完整的 Token 使用统计记录：

```go
logger.InfoContext(ctx, "流式生成完成", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "ttfb":             ttfb.String(),
    "chunkCount":       chunkCount,
    "totalContentLen":  len(totalContent),
    "promptTokens":     usage.PromptTokens,      // 提示词 token 数
    "completionTokens": usage.CompletionTokens,  // 生成内容 token 数
    "totalTokens":      usage.TotalTokens,       // 总 token 数
})
```

### 2. AI Service 层（internal/service/ai/genkit_service.go）

#### 非流式对话（Chat 方法）

改进前：

```go
s.logger.InfoContext(sessionCtx, "对话请求处理完成", logger.Fields{
    "sessionId": sessionID,
    "model":     result.Model,
    "duration":  duration.String(),
    "tokens":    response.Usage,  // 整个对象，不便于查询
})
```

改进后：

```go
logFields := logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "modelName": modelName,
    "model":     result.Model,
    "duration":  duration.String(),
}

// 添加 Token 使用统计（展开为单独字段）
if response.Usage != nil {
    logFields["promptTokens"] = response.Usage.PromptTokens
    logFields["completionTokens"] = response.Usage.CompletionTokens
    logFields["totalTokens"] = response.Usage.TotalTokens
}

s.logger.InfoContext(sessionCtx, "对话请求处理完成", logFields)
```

#### 流式对话（ChatStream 方法）

改进前：

```go
s.logger.InfoContext(sessionCtx, "流式对话请求处理完成", logger.Fields{
    "sessionId": sessionID,
    "model":     lastModel,
    "duration":  duration.String(),
    "tokens":    lastUsage,  // 整个对象，不便于查询
})
```

改进后：

```go
logFields := logger.Fields{
    "sessionId": sessionID,
    "tenantId":  tenantID,
    "modelName": modelName,
    "model":     lastModel,
    "duration":  duration.String(),
}

// 添加 Token 使用统计（展开为单独字段）
if lastUsage != nil {
    logFields["promptTokens"] = lastUsage.PromptTokens
    logFields["completionTokens"] = lastUsage.CompletionTokens
    logFields["totalTokens"] = lastUsage.TotalTokens
}

s.logger.InfoContext(sessionCtx, "流式对话请求处理完成", logFields)
```

## 改进优势

### 1. 更好的可查询性

将 Token 使用统计展开为单独的字段后，可以更方便地进行日志查询和分析：

```bash
# 查询总 token 使用量超过 1000 的请求
grep "totalTokens" app.log | jq 'select(.fields.totalTokens > 1000)'

# 统计某个租户的 token 使用情况
grep "tenantId.*test-tenant-id" app.log | jq '.fields.totalTokens' | awk '{sum+=$1} END {print sum}'

# 查询某个模型的平均 token 使用量
grep "modelName.*gpt-4" app.log | jq '.fields.totalTokens' | awk '{sum+=$1; count++} END {print sum/count}'
```

### 2. 更好的监控集成

展开的字段可以直接被日志聚合系统（如 ELK、Splunk、Datadog）识别和索引：

- 可以创建基于 `promptTokens`、`completionTokens`、`totalTokens` 的仪表板
- 可以设置基于 token 使用量的告警
- 可以按租户、模型、时间段等维度进行统计分析

### 3. 更完整的上下文信息

每条日志都包含了完整的上下文信息：

- `tenantId`: 租户ID，用于多租户隔离和成本分摊
- `modelName`: 模型名称，用于区分不同的模型配置
- `model`: 实际使用的模型，用于了解底层提供商
- `duration`: 请求耗时，用于性能分析
- `promptTokens`: 输入 token 数
- `completionTokens`: 输出 token 数
- `totalTokens`: 总 token 数

## 日志示例

### 非流式调用日志

```json
{
  "timestamp": "2025-11-30T00:44:36Z",
  "level": "INFO",
  "message": "对话请求处理完成",
  "fields": {
    "sessionId": "67c91c7a-f2d2-4059-93f6-1246cc0ae457",
    "tenantId": "test-tenant-id",
    "modelName": "gemini-pro",
    "model": "test-model",
    "duration": "143.584µs",
    "promptTokens": 10,
    "completionTokens": 20,
    "totalTokens": 30
  }
}
```

### 流式调用日志

```json
{
  "timestamp": "2025-11-30T00:45:07Z",
  "level": "INFO",
  "message": "流式对话请求处理完成",
  "fields": {
    "sessionId": "c5d765ca-ff23-4091-bbfa-e9926e142cf6",
    "tenantId": "test-tenant-id",
    "modelName": "gemini-pro",
    "model": "test-model",
    "duration": "189.833µs",
    "promptTokens": 10,
    "completionTokens": 20,
    "totalTokens": 30
  }
}
```

## 测试验证

所有相关测试都已通过：

```bash
$ go test ./internal/service/ai/
ok      genkit-ai-service/internal/service/ai   0.656s
```

测试覆盖：

- ✅ 非流式对话的 Token 统计记录
- ✅ 流式对话的 Token 统计记录
- ✅ 指定模型名称的场景
- ✅ 使用默认模型的场景
- ✅ 错误场景的处理

## 使用建议

### 1. 成本监控

可以基于 `totalTokens` 字段计算 API 调用成本：

```bash
# 计算某个租户的每日成本（假设每 1000 tokens 成本为 $0.002）
grep "tenantId.*tenant-123" app.log | \
  jq '.fields.totalTokens' | \
  awk '{sum+=$1} END {print "Total tokens:", sum, "Cost: $" sum/1000*0.002}'
```

### 2. 性能分析

结合 `duration` 和 `totalTokens` 分析性能：

```bash
# 查找 token 使用量大且耗时长的请求
grep "totalTokens" app.log | \
  jq 'select(.fields.totalTokens > 1000 and .fields.duration > "1s")'
```

### 3. 模型使用统计

统计不同模型的使用情况：

```bash
# 统计各模型的调用次数和 token 使用量
grep "modelName" app.log | \
  jq -r '[.fields.modelName, .fields.totalTokens] | @tsv' | \
  awk '{model[$1]++; tokens[$1]+=$2} END {for(m in model) print m, model[m], tokens[m]}'
```

## 后续优化建议

1. **添加 Token 使用率指标**：
   - 计算 `completionTokens / promptTokens` 比率
   - 识别异常的 token 使用模式

2. **添加成本估算**：
   - 根据不同模型的定价，实时计算成本
   - 在日志中添加 `estimatedCost` 字段

3. **添加 Token 使用趋势分析**：
   - 按时间段统计 token 使用趋势
   - 识别 token 使用量的异常增长

4. **添加租户级别的 Token 配额**：
   - 为每个租户设置 token 使用配额
   - 超过配额时发出告警或限制调用

## 相关文件

- `internal/genkit/client.go` - Genkit 客户端实现
- `internal/service/ai/genkit_service.go` - AI 服务实现
- `internal/service/ai/genkit_service_test.go` - 测试文件
- `internal/genkit/config.go` - 配置和类型定义
- `internal/model/ai.go` - AI 相关的数据模型

## 完成状态

✅ **任务已完成**

- [x] 记录提供商选择日志
- [x] 记录 API 调用耗时
- [x] 记录 Token 使用统计（promptTokens, completionTokens, totalTokens）
- [x] 确保日志格式便于查询和分析
- [x] 所有测试通过

## 验收标准

根据 TASK-6.4 的验收标准：

- [x] 记录提供商选择日志 - 已在 `client.go` 中实现
- [x] 记录 API 调用耗时 - 已在所有方法中记录 `duration`
- [x] 记录 Token 使用统计 - 已完整实现并展开为单独字段
- [ ] 记录错误详情 - 已有基本实现，可进一步完善
- [ ] 添加 TraceID 追踪 - 待实现（需要集成分布式追踪系统）
- [ ] 确保敏感信息脱敏 - 已在 `client.go` 中实现 API Key 脱敏

## 总结

本次实现完善了 Token 使用统计的日志记录功能，确保所有 AI 调用都能详细记录 Token 使用情况。通过将 Token 统计展开为单独的字段，大大提高了日志的可查询性和可分析性，为后续的成本监控、性能优化和使用分析奠定了基础。
