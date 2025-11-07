# 性能追踪模块

本模块提供基于 OpenTelemetry 的分布式追踪功能，支持 Jaeger 作为追踪后端。

## 功能特性

- ✅ 集成 OpenTelemetry
- ✅ 支持 Jaeger 导出器
- ✅ Flow 执行追踪
- ✅ 数据库查询追踪
- ✅ 向量检索追踪
- ✅ AI 生成追踪
- ✅ 缓存操作追踪
- ✅ 嵌套 span 支持
- ✅ 自动提取上下文信息（session_id, tenant_id, user_id）
- ✅ 可配置采样率
- ✅ 支持禁用追踪（NoOp 模式）

## 环境变量配置

在 `.env` 文件中添加以下配置：

```bash
# 追踪配置
TRACING_ENABLED=true
TRACING_SERVICE_NAME=genkit-ai-service
TRACING_SERVICE_VERSION=1.0.0
TRACING_ENVIRONMENT=production
OTLP_ENDPOINT=localhost:4318
TRACING_SAMPLING_RATE=1.0
```

### 配置说明

- `TRACING_ENABLED`: 是否启用追踪（true/false）
- `TRACING_SERVICE_NAME`: 服务名称
- `TRACING_SERVICE_VERSION`: 服务版本
- `TRACING_ENVIRONMENT`: 环境（development/staging/production）
- `OTLP_ENDPOINT`: OTLP 端点地址（支持 Jaeger v1.35+、Tempo 等）
- `TRACING_SAMPLING_RATE`: 采样率（0.0-1.0）
  - 1.0 = 100% 采样（所有请求都追踪）
  - 0.5 = 50% 采样
  - 0.0 = 不采样

## 使用方法

### 1. 初始化追踪器

在应用启动时初始化追踪器：

```go
package main

import (
    "context"
    "log"
    
    "genkit-ai-service/internal/tracing"
)

func main() {
    // 初始化追踪器
    tracer, err := tracing.InitTracer()
    if err != nil {
        log.Fatalf("初始化追踪器失败: %v", err)
    }
    defer tracer.Shutdown(context.Background())
    
    // 使用追踪器
    // ...
}
```

### 2. 追踪 Flow 执行

在 Flow 中使用追踪：

```go
func (s *ContextService) BuildContext(ctx context.Context, req BuildContextRequest) (*ContextResult, error) {
    return tracer.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
        // Flow 执行逻辑
        // ...
        return nil
    })
}
```

### 3. 追踪数据库查询

```go
err := tracing.TraceDBQuery(ctx, "get_recent_messages",
    "SELECT * FROM conversation_messages WHERE session_id = $1",
    func(ctx context.Context) error {
        // 执行数据库查询
        messages, err := s.messageRepo.GetRecentMessages(ctx, sessionID, 10)
        return err
    },
)
```

### 4. 追踪向量检索

```go
err := tracing.TraceVectorSearch(ctx, sessionID, 5, func(ctx context.Context) error {
    // 执行向量检索
    memories, err := s.memoryRepo.SearchByVector(ctx, sessionID, embedding, 5, 0.7)
    return err
})
```

### 5. 追踪 AI 生成

```go
err := tracing.TraceAIGeneration(ctx, "gemini-2.5-flash", promptTokens, func(ctx context.Context) error {
    // 调用 AI 生成
    response, err := s.aiService.Generate(ctx, prompt)
    return err
})
```

### 6. 追踪缓存操作

```go
err := tracing.TraceCacheOperation(ctx, "get", cacheKey, func(ctx context.Context) error {
    // 从缓存获取数据
    err := s.cache.Get(ctx, cacheKey, &result)
    return err
})
```

### 7. 创建自定义 Span

```go
ctx, span := tracer.StartSpan(ctx, "custom_operation")
defer span.End()

// 添加自定义属性
span.SetAttributes(
    attribute.String("custom.key", "value"),
    attribute.Int("custom.count", 42),
)

// 执行操作
err := doSomething(ctx)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### 8. 嵌套 Span

```go
err := tracer.TraceFlow(ctx, "parentFlow", func(ctx context.Context) error {
    // 步骤 1
    ctx, span1 := tracer.StartSpan(ctx, "step1")
    // ... 执行步骤 1
    span1.End()
    
    // 步骤 2
    err := tracing.TraceOperation(ctx, "step2", func(ctx context.Context) error {
        // ... 执行步骤 2
        return nil
    })
    
    return err
})
```

## 上下文信息自动提取

追踪器会自动从 context 中提取以下信息并添加到 span：

- `session_id`: 会话 ID
- `tenant_id`: 租户 ID
- `user_id`: 用户 ID
- `request_id`: 请求 ID
- `trace_id`: 追踪 ID

确保在 context 中设置这些值：

```go
ctx = context.WithValue(ctx, "session_id", sessionID)
ctx = context.WithValue(ctx, "tenant_id", tenantID)
ctx = context.WithValue(ctx, "user_id", userID)
```

## 追踪后端部署

### 使用 Docker 运行 Jaeger（推荐）

Jaeger v1.35+ 支持 OTLP 协议：

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

端口说明：

- `16686`: Jaeger UI
- `4318`: OTLP HTTP 接收器

### 访问 Jaeger UI

打开浏览器访问：<http://localhost:16686>

### 使用 Grafana Tempo（可选）

```bash
docker run -d --name tempo \
  -p 4318:4318 \
  -p 3200:3200 \
  grafana/tempo:latest \
  -config.file=/etc/tempo.yaml
```

### 使用 OpenTelemetry Collector（可选）

如果需要更灵活的配置，可以使用 OpenTelemetry Collector 作为中间层：

```bash
docker run -d --name otel-collector \
  -p 4318:4318 \
  otel/opentelemetry-collector:latest
```

## 性能考虑

### 采样策略

在生产环境中，建议使用适当的采样率以减少性能开销：

- **开发环境**: 1.0（100% 采样）
- **测试环境**: 1.0（100% 采样）
- **生产环境**: 0.1-0.3（10%-30% 采样）

### 禁用追踪

如果不需要追踪功能，可以设置 `TRACING_ENABLED=false`，系统会使用 NoOp 追踪器，不会产生任何性能开销。

## 最佳实践

### 1. 为关键操作添加追踪

优先为以下操作添加追踪：

- Flow 执行
- 数据库查询
- 外部服务调用（AI、向量数据库）
- 缓存操作
- 长时间运行的操作

### 2. 添加有意义的属性

```go
span.SetAttributes(
    attribute.String("operation.type", "vector_search"),
    attribute.Int("result.count", len(results)),
    attribute.Float64("similarity.threshold", 0.7),
)
```

### 3. 记录错误信息

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    span.SetAttributes(
        attribute.String("error.type", "database_error"),
        attribute.String("error.details", err.Error()),
    )
}
```

### 4. 使用有意义的 Span 名称

- ✅ 好的命名: `contextBuildFlow`, `db.get_recent_messages`, `vector.search`
- ❌ 不好的命名: `operation1`, `query`, `search`

### 5. 避免过度追踪

不要为每个小函数都创建 span，这会产生大量开销。只追踪关键路径和重要操作。

## 故障排查

### 追踪数据未显示在 Jaeger

1. 检查 Jaeger 是否正常运行
2. 检查 `JAEGER_ENDPOINT` 配置是否正确
3. 检查采样率是否设置为 0
4. 检查网络连接

### 性能问题

1. 降低采样率
2. 减少 span 数量
3. 检查 Jaeger 性能

### 内存占用高

1. 降低采样率
2. 配置 Jaeger 批处理参数
3. 增加 Jaeger 导出间隔

## 与监控指标的关系

追踪（Tracing）和监控指标（Metrics）是互补的：

- **监控指标**: 提供聚合的统计数据（请求数、延迟、错误率）
- **追踪**: 提供单个请求的详细执行路径

建议同时使用两者以获得完整的可观测性。

## 参考资料

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/)
