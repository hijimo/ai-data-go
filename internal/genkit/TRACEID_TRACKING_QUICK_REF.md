# TraceID 追踪快速参考

## 概述

TraceID 追踪功能已集成到 Genkit Client 中，用于在整个 AI 调用链路中追踪请求。TraceID 会自动从 HTTP 请求的上下文中提取，并在所有日志记录中包含。

## TraceID 流转链路

```
HTTP 请求
  ↓
Logger 中间件（提取或生成 TraceID）
  ↓
Context（注入 TraceID）
  ↓
Handler 层
  ↓
Service 层
  ↓
AI Service 层
  ↓
Genkit Client（自动记录 TraceID）
  ↓
所有日志记录（自动包含 TraceID）
```

## TraceID 格式

TraceID 格式：`trace-{timestamp}-{nanoHex}{random}`

示例：`trace-1704067200-a3f9k2-b8c1d4`

- `timestamp`: Unix 时间戳（秒）
- `nanoHex`: 纳秒时间戳的十六进制表示（后6位）
- `random`: 6位随机十六进制字符串

## 自动追踪的操作

### 1. 非流式生成

```go
result, err := client.Generate(ctx, tenantID, modelName, prompt, options)
```

**自动记录的日志（包含 TraceID）：**

- 开始生成内容
- 选择模型提供商
- 初始化提供商
- 生成内容成功/失败

**日志字段：**

```json
{
  "timestamp": "2025-12-01T10:30:00Z",
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

### 2. 流式生成

```go
streamChan, err := client.GenerateStream(ctx, tenantID, modelName, prompt, options)
```

**自动记录的日志（包含 TraceID）：**

- 开始流式生成内容
- 选择模型提供商
- 初始化提供商
- 收到首个响应块（TTFB）
- 流式生成完成/失败

**日志字段：**

```json
{
  "timestamp": "2025-12-01T10:30:05Z",
  "level": "INFO",
  "message": "流式生成完成",
  "fields": {
    "traceId": "trace-1733051405-b4e8d3-c9f2a1",
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

客户端可以在 HTTP 请求头中提供 TraceID：

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "X-Trace-ID: trace-custom-123456" \
  -H "Authorization: Bearer {token}" \
  -d '{"message": "Hello"}'
```

系统会使用客户端提供的 TraceID，并在响应头中返回：

```
X-Trace-ID: trace-custom-123456
```

### 2. 自动生成 TraceID

如果客户端未提供 TraceID，系统会自动生成：

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{id}/messages \
  -H "Authorization: Bearer {token}" \
  -d '{"message": "Hello"}'
```

响应头会包含自动生成的 TraceID：

```
X-Trace-ID: trace-1733051400-a3f9k2-b8c1d4
```

## 日志查询

### 1. 按 TraceID 查询所有日志

```bash
# 查询特定 TraceID 的所有日志
grep "trace-1733051400-a3f9k2-b8c1d4" logs/app-2025-12-01.log
```

### 2. 按 TraceID 查询 JSON 格式日志

```bash
# 使用 jq 查询 JSON 日志
cat logs/app-2025-12-01.log | jq 'select(.fields.traceId == "trace-1733051400-a3f9k2-b8c1d4")'
```

### 3. 追踪完整调用链

```bash
# 查看特定 TraceID 的完整调用链
grep "trace-1733051400-a3f9k2-b8c1d4" logs/app-2025-12-01.log | \
  jq -r '[.timestamp, .level, .message, .fields.duration // ""] | @tsv'
```

输出示例：

```
2025-12-01T10:30:00Z    INFO    开始生成内容
2025-12-01T10:30:00Z    INFO    选择模型提供商
2025-12-01T10:30:00Z    INFO    初始化提供商
2025-12-01T10:30:02Z    INFO    生成内容成功    2.5s
```

## 监控指标

所有带有 TraceID 的日志都包含以下关键指标：

### 性能指标

- `duration`: 总耗时（字符串格式）
- `durationMs`: 总耗时（毫秒）
- `ttfb`: 首字节时间（仅流式）
- `ttfbMs`: 首字节时间（毫秒，仅流式）

### Token 使用指标

- `promptTokens`: 提示词 Token 数
- `completionTokens`: 生成内容 Token 数
- `totalTokens`: 总 Token 数

### 流式指标

- `chunkCount`: 响应块数量
- `totalContentLen`: 总内容长度

## 错误追踪

当发生错误时，TraceID 会自动包含在错误日志中：

```json
{
  "timestamp": "2025-12-01T10:30:00Z",
  "level": "ERROR",
  "message": "获取模型实例失败",
  "fields": {
    "traceId": "trace-1733051400-a3f9k2-b8c1d4",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "modelName": "gpt-4",
    "error": "获取模型配置失败: record not found"
  }
}
```

## 与其他追踪系统集成

### OpenTelemetry 集成

TraceID 可以与 OpenTelemetry 的 Trace ID 关联：

```go
// 在 tracing 包中，TraceID 会自动添加到 span 属性
span.SetAttributes(attribute.String("trace.id", traceID))
```

### 日志聚合系统

TraceID 字段可以在日志聚合系统（如 ELK、Loki）中用于：

1. **按 TraceID 分组**：查看单个请求的完整调用链
2. **性能分析**：统计不同 TraceID 的耗时分布
3. **错误分析**：快速定位特定请求的错误原因
4. **用户行为追踪**：结合 userID 和 TraceID 分析用户操作

## 最佳实践

### 1. 客户端传递 TraceID

建议客户端在发起请求时生成并传递 TraceID，以便：

- 关联前端和后端日志
- 追踪跨服务调用
- 支持分布式追踪

### 2. 日志查询优化

在日志聚合系统中为 `traceId` 字段建立索引，提高查询性能。

### 3. TraceID 保留策略

建议保留 TraceID 日志至少 30 天，用于：

- 问题排查
- 性能分析
- 用户行为分析

### 4. 敏感信息脱敏

确保 TraceID 日志中不包含敏感信息：

- API 密钥已自动脱敏
- 用户输入内容仅记录长度，不记录完整内容
- 响应内容仅记录长度，不记录完整内容

## 相关文档

- [Logger 中间件实现](../api/middleware/logger.go)
- [Context 管理](../api/middleware/context.go)
- [日志记录器](../logger/logger.go)
- [分布式追踪](../tracing/tracer.go)

## 更新历史

- 2025-12-01: 完成 TraceID 追踪功能集成
  - 在 Generate 和 GenerateStream 方法中自动记录 TraceID
  - 所有日志记录自动包含 TraceID
  - 添加性能指标（duration、ttfb）
  - 添加 Token 使用统计
