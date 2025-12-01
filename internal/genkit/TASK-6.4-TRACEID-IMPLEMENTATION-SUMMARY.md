# TASK-6.4 TraceID 追踪实现总结

## 任务概述

为 Genkit Client 添加 TraceID 追踪功能，确保在整个 AI 调用链路中可以追踪请求。

## 实现内容

### 1. TraceID 自动追踪

#### 1.1 非流式生成（Generate）

**修改文件**: `internal/genkit/client.go`

**实现内容**:

- 在 `Generate` 方法中，所有日志记录自动包含 TraceID
- TraceID 通过 `logger.InfoContext(ctx, ...)` 自动从 context 提取
- 无需手动提取或传递 TraceID

**关键日志点**:

1. 开始生成内容
2. 选择模型提供商
3. 初始化提供商
4. 生成内容成功/失败

**日志字段增强**:

```go
logger.InfoContext(ctx, "生成内容成功", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "durationMs":       duration.Milliseconds(),  // 新增：毫秒级耗时
    "promptTokens":     result.Usage.PromptTokens,
    "completionTokens": result.Usage.CompletionTokens,
    "totalTokens":      result.Usage.TotalTokens,
    "responseLen":      len(result.Text),
})
// traceId 会自动从 context 中提取并添加到日志
```

#### 1.2 流式生成（GenerateStream）

**修改文件**: `internal/genkit/client.go`

**实现内容**:

- 在 `GenerateStream` 方法中，所有日志记录自动包含 TraceID
- 流式调用的每个阶段都会记录 TraceID
- 支持追踪首字节时间（TTFB）

**关键日志点**:

1. 开始流式生成内容
2. 选择模型提供商
3. 初始化提供商
4. 收到首个响应块（TTFB）
5. 流式生成完成/失败

**日志字段增强**:

```go
logger.InfoContext(ctx, "流式生成完成", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "durationMs":       duration.Milliseconds(),  // 新增：毫秒级耗时
    "ttfb":             ttfb.String(),
    "ttfbMs":           ttfb.Milliseconds(),      // 新增：毫秒级 TTFB
    "chunkCount":       chunkCount,
    "totalContentLen":  len(totalContent),
    "promptTokens":     usage.PromptTokens,
    "completionTokens": usage.CompletionTokens,
    "totalTokens":      usage.TotalTokens,
})
// traceId 会自动从 context 中提取并添加到日志
```

### 2. TraceID 基础设施（已存在）

#### 2.1 中间件层

**文件**: `internal/api/middleware/logger.go`

**功能**:

- 从 HTTP 请求头提取 `X-Trace-ID`
- 如果不存在，自动生成 TraceID
- 将 TraceID 注入到 Context
- 在响应头中返回 TraceID

#### 2.2 Context 管理

**文件**: `internal/api/middleware/context.go`

**功能**:

- 提供 `SetTraceID` 和 `GetTraceID` 方法
- 使用字符串键 `"traceId"` 确保跨包兼容
- 高性能的 TraceID 生成算法

#### 2.3 日志记录器

**文件**: `internal/logger/logger.go`

**功能**:

- `InfoContext`、`ErrorContext` 等方法自动从 context 提取 TraceID
- TraceID 自动添加到日志的 `fields.traceId` 字段
- 支持 JSON 和文本格式输出

## TraceID 流转链路

```
HTTP 请求（X-Trace-ID 头）
  ↓
Logger 中间件
  ├─ 提取或生成 TraceID
  ├─ 注入到 Context
  └─ 设置响应头
  ↓
Handler 层（Context 传递）
  ↓
Service 层（Context 传递）
  ↓
AI Service 层（Context 传递）
  ↓
Genkit Client
  ├─ Generate/GenerateStream
  ├─ 所有日志自动包含 TraceID
  └─ 性能指标记录
  ↓
日志文件（JSON 格式）
  └─ fields.traceId: "trace-xxx"
```

## 日志示例

### 非流式生成日志

```json
{
  "timestamp": "2025-12-01T10:30:00Z",
  "level": "INFO",
  "message": "开始生成内容",
  "fields": {
    "traceId": "trace-1733051400-a3f9k2-b8c1d4",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "promptLen": 150
  }
}

{
  "timestamp": "2025-12-01T10:30:00Z",
  "level": "INFO",
  "message": "选择模型提供商",
  "fields": {
    "traceId": "trace-1733051400-a3f9k2-b8c1d4",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "provider": "azureopenai",
    "model": "gpt-4"
  }
}

{
  "timestamp": "2025-12-01T10:30:02Z",
  "level": "INFO",
  "message": "生成内容成功",
  "fields": {
    "traceId": "trace-1733051400-a3f9k2-b8c1d4",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "model": "gpt-4",
    "duration": "2.5s",
    "durationMs": 2500,
    "promptTokens": 150,
    "completionTokens": 300,
    "totalTokens": 450,
    "responseLen": 1024
  }
}
```

### 流式生成日志

```json
{
  "timestamp": "2025-12-01T10:30:00Z",
  "level": "INFO",
  "message": "开始流式生成内容",
  "fields": {
    "traceId": "trace-1733051400-b4e8d3-c9f2a1",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "promptLen": 150
  }
}

{
  "timestamp": "2025-12-01T10:30:00Z",
  "level": "INFO",
  "message": "收到首个响应块",
  "fields": {
    "traceId": "trace-1733051400-b4e8d3-c9f2a1",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "model": "gpt-4",
    "ttfb": "500ms"
  }
}

{
  "timestamp": "2025-12-01T10:30:03Z",
  "level": "INFO",
  "message": "流式生成完成",
  "fields": {
    "traceId": "trace-1733051400-b4e8d3-c9f2a1",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "model": "gpt-4",
    "duration": "3.2s",
    "durationMs": 3200,
    "ttfb": "500ms",
    "ttfbMs": 500,
    "chunkCount": 25,
    "totalContentLen": 2048,
    "promptTokens": 150,
    "completionTokens": 400,
    "totalTokens": 550
  }
}
```

## 使用方式

### 1. 客户端提供 TraceID

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "X-Trace-ID: trace-custom-123456" \
  -H "Authorization: Bearer {token}" \
  -d '{"message": "Hello"}'
```

响应头：

```
X-Trace-ID: trace-custom-123456
```

### 2. 自动生成 TraceID

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "Authorization: Bearer {token}" \
  -d '{"message": "Hello"}'
```

响应头：

```
X-Trace-ID: trace-1733051400-a3f9k2-b8c1d4
```

## 日志查询

### 按 TraceID 查询

```bash
# 查询特定 TraceID 的所有日志
grep "trace-1733051400-a3f9k2-b8c1d4" logs/app-2025-12-01.log

# 使用 jq 查询 JSON 日志
cat logs/app-2025-12-01.log | \
  jq 'select(.fields.traceId == "trace-1733051400-a3f9k2-b8c1d4")'

# 追踪完整调用链
grep "trace-1733051400-a3f9k2-b8c1d4" logs/app-2025-12-01.log | \
  jq -r '[.timestamp, .level, .message, .fields.duration // ""] | @tsv'
```

## 性能指标

所有日志都包含以下性能指标：

### 通用指标

- `duration`: 总耗时（字符串格式，如 "2.5s"）
- `durationMs`: 总耗时（毫秒，如 2500）

### 流式特有指标

- `ttfb`: 首字节时间（字符串格式，如 "500ms"）
- `ttfbMs`: 首字节时间（毫秒，如 500）
- `chunkCount`: 响应块数量
- `totalContentLen`: 总内容长度

### Token 使用指标

- `promptTokens`: 提示词 Token 数
- `completionTokens`: 生成内容 Token 数
- `totalTokens`: 总 Token 数

## 技术优势

### 1. 零侵入性

- 无需修改业务代码
- 无需手动传递 TraceID
- 通过 Context 自动传递

### 2. 高性能

- TraceID 生成使用对象池优化
- 日志记录异步处理
- 最小化性能开销

### 3. 完整追踪

- 覆盖整个调用链路
- 包含所有关键操作
- 支持错误追踪

### 4. 易于查询

- 结构化日志（JSON 格式）
- 统一的字段命名
- 支持日志聚合系统

## 与其他系统集成

### OpenTelemetry

TraceID 可以与 OpenTelemetry 的 Trace ID 关联：

```go
// 在 tracing 包中
span.SetAttributes(attribute.String("trace.id", traceID))
```

### 日志聚合系统

支持集成到：

- ELK Stack（Elasticsearch + Logstash + Kibana）
- Grafana Loki
- Splunk
- Datadog

## 验收标准

✅ **已完成**:

1. ✅ 记录提供商选择日志（包含 TraceID）
2. ✅ 记录 API 调用耗时（duration、durationMs）
3. ✅ 记录 Token 使用统计（promptTokens、completionTokens、totalTokens）
4. ✅ 记录错误详情（包含 TraceID）
5. ✅ 所有日志自动包含 TraceID

⏳ **待完成**（可选）:

- [ ] 添加 TraceID 追踪（已在 TASK-6.4 中完成基础实现）
- [ ] 确保敏感信息脱敏（API 密钥已脱敏，需要审查其他敏感信息）

## 相关文档

- [TraceID 追踪快速参考](./TRACEID_TRACKING_QUICK_REF.md)
- [Logger 中间件](../api/middleware/logger.go)
- [Context 管理](../api/middleware/context.go)
- [日志记录器](../logger/logger.go)

## 测试建议

### 1. 单元测试

测试 TraceID 在不同场景下的传递：

```go
func TestGenerateWithTraceID(t *testing.T) {
    ctx := context.Background()
    ctx = context.WithValue(ctx, "traceId", "test-trace-123")
    
    // 调用 Generate
    result, err := client.Generate(ctx, tenantID, modelName, prompt, nil)
    
    // 验证日志中包含 TraceID
    // ...
}
```

### 2. 集成测试

测试完整的 HTTP 请求链路：

```bash
# 测试脚本
#!/bin/bash

TRACE_ID="trace-test-$(date +%s)"

# 发送请求
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "X-Trace-ID: $TRACE_ID" \
  -H "Authorization: Bearer {token}" \
  -d '{"message": "Hello"}'

# 查询日志
sleep 1
grep "$TRACE_ID" logs/app-$(date +%Y-%m-%d).log
```

### 3. 性能测试

验证 TraceID 追踪不会显著影响性能：

```bash
# 压力测试
ab -n 1000 -c 10 \
  -H "X-Trace-ID: trace-perf-test" \
  -H "Authorization: Bearer {token}" \
  http://localhost:8080/api/v1/chat/sessions/{id}/messages
```

## 总结

TraceID 追踪功能已成功集成到 Genkit Client 中，实现了：

1. **自动追踪**：所有 AI 调用自动包含 TraceID
2. **完整链路**：覆盖从 HTTP 请求到 AI 响应的完整链路
3. **性能监控**：记录耗时、TTFB、Token 使用等关键指标
4. **易于查询**：结构化日志，支持按 TraceID 快速查询
5. **零侵入**：无需修改业务代码，通过 Context 自动传递

该功能为系统的可观测性提供了强有力的支持，便于问题排查、性能分析和用户行为追踪。
