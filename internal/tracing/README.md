# 性能追踪模块

本模块提供基于 OpenTelemetry 的分布式追踪功能，支持 Jaeger 导出器。

## 功能特性

- ✅ OpenTelemetry 集成
- ✅ OTLP 导出器支持（兼容 Jaeger 0.14+）
- ✅ Flow 执行追踪
- ✅ 服务层方法追踪
- ✅ 数据库操作追踪
- ✅ 外部服务调用追踪
- ✅ 自定义 Span 属性
- ✅ 上下文信息自动提取
- ✅ 灵活的采样策略
- ✅ 完整的单元测试

## 快速开始

### 1. 初始化追踪提供者

```go
package main

import (
    "context"
    "log"
    "genkit-ai-service/internal/tracing"
)

func main() {
    // 创建配置
    config := &tracing.Config{
        Enabled:        true,
        ServiceName:    "genkit-ai-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        OTLPEndpoint:   "localhost:4317", // Jaeger OTLP gRPC 端口
        SamplingRate:   0.1,               // 10% 采样
    }

    // 初始化追踪提供者
    tp, err := tracing.NewTracerProvider(config)
    if err != nil {
        log.Fatalf("初始化追踪提供者失败: %v", err)
    }
    defer tp.Shutdown(context.Background())

    // 应用启动...
}
```

### 2. 追踪 Flow 执行

```go
func (f *Flow) Execute(ctx context.Context, input Input) (Output, error) {
    var output Output
    
    err := tracing.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
        // Flow 执行逻辑
        result, err := f.buildContext(ctx, input)
        if err != nil {
            return err
        }
        output = result
        return nil
    })
    
    return output, err
}
```

### 3. 追踪服务层方法

```go
func (s *ContextService) BuildContext(ctx context.Context, req BuildContextRequest) (*ContextResult, error) {
    var result *ContextResult
    
    err := tracing.TraceService(ctx, "ContextService", "BuildContext", func(ctx context.Context) error {
        // 服务逻辑
        res, err := s.doBuildContext(ctx, req)
        if err != nil {
            return err
        }
        result = res
        return nil
    })
    
    return result, err
}
```

### 4. 追踪数据库操作

```go
func (r *MemoryRepository) SearchByVector(ctx context.Context, sessionID string, embedding Vector) ([]*Memory, error) {
    var memories []*Memory
    
    err := tracing.TraceRepository(ctx, "MemoryRepository", "SearchByVector", func(ctx context.Context) error {
        // 数据库查询
        return r.db.WithContext(ctx).
            Where("session_id = ?", sessionID).
            Find(&memories).Error
    })
    
    return memories, err
}
```

### 5. 追踪外部服务调用

```go
func (s *AIService) Generate(ctx context.Context, prompt string) (string, error) {
    var response string
    
    err := tracing.TraceExternalCall(ctx, "OpenAI", "ChatCompletion", func(ctx context.Context) error {
        // 调用外部 API
        resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
            Model: "gpt-4",
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: prompt},
            },
        })
        if err != nil {
            return err
        }
        response = resp.Choices[0].Message.Content
        return nil
    })
    
    return response, err
}
```

## 高级用法

### 添加自定义属性

```go
func processData(ctx context.Context, data Data) error {
    // 添加自定义属性
    tracing.AddSpanAttributes(ctx,
        attribute.String("data.id", data.ID),
        attribute.Int("data.size", len(data.Content)),
        attribute.String("data.type", data.Type),
    )
    
    // 处理数据...
    return nil
}
```

### 添加事件

```go
func processSteps(ctx context.Context) error {
    // 步骤 1
    tracing.AddSpanEvent(ctx, "step1.started")
    if err := step1(); err != nil {
        return err
    }
    tracing.AddSpanEvent(ctx, "step1.completed")
    
    // 步骤 2
    tracing.AddSpanEvent(ctx, "step2.started")
    if err := step2(); err != nil {
        return err
    }
    tracing.AddSpanEvent(ctx, "step2.completed")
    
    return nil
}
```

### 记录 Token 使用

```go
func generateResponse(ctx context.Context, prompt string) (string, error) {
    ctx, span := tracing.TraceAIGeneration(ctx, "gpt-4", len(prompt))
    defer span.End()
    
    // 生成响应
    response, usage, err := ai.Generate(prompt)
    if err != nil {
        tracing.SetSpanError(span, err)
        return "", err
    }
    
    // 记录 Token 使用
    tracing.AddTokenUsage(ctx, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
    tracing.SetSpanSuccess(span)
    
    return response, nil
}
```

### 记录上下文指标

```go
func buildContext(ctx context.Context, req Request) (*Context, error) {
    ctx, span := tracing.StartSpan(ctx, "buildContext")
    defer span.End()
    
    // 构建上下文
    context, err := doBuil dContext(req)
    if err != nil {
        tracing.SetSpanError(span, err)
        return nil, err
    }
    
    // 记录指标
    tracing.AddContextMetrics(ctx, context.TotalTokens, context.QualityScore, context.Strategy)
    tracing.SetSpanSuccess(span)
    
    return context, nil
}
```

### 手动创建 Span

```go
func complexOperation(ctx context.Context) error {
    // 创建父 Span
    ctx, parentSpan := tracing.StartSpan(ctx, "complexOperation")
    defer parentSpan.End()
    
    // 子操作 1
    ctx, span1 := tracing.StartSpan(ctx, "subOperation1")
    if err := subOp1(ctx); err != nil {
        tracing.SetSpanError(span1, err)
        span1.End()
        return err
    }
    tracing.SetSpanSuccess(span1)
    span1.End()
    
    // 子操作 2
    ctx, span2 := tracing.StartSpan(ctx, "subOperation2")
    if err := subOp2(ctx); err != nil {
        tracing.SetSpanError(span2, err)
        span2.End()
        return err
    }
    tracing.SetSpanSuccess(span2)
    span2.End()
    
    tracing.SetSpanSuccess(parentSpan)
    return nil
}
```

## 配置选项

### 开发环境配置

```go
config := tracing.DefaultConfig()
// 默认配置：
// - Enabled: false
// - ServiceName: "genkit-ai-service"
// - ServiceVersion: "1.0.0"
// - Environment: "development"
// - OTLPEndpoint: "localhost:4317"
// - SamplingRate: 1.0 (100% 采样)
```

### 生产环境配置

```go
config := tracing.ProductionConfig("jaeger.prod.example.com:4317")
// 生产配置：
// - Enabled: true
// - ServiceName: "genkit-ai-service"
// - ServiceVersion: "1.0.0"
// - Environment: "production"
// - OTLPEndpoint: 指定的端点
// - SamplingRate: 0.1 (10% 采样)
```

### 自定义配置

```go
config := &tracing.Config{
    Enabled:        true,
    ServiceName:    "my-service",
    ServiceVersion: "2.0.0",
    Environment:    "staging",
    OTLPEndpoint:   "jaeger.staging.example.com:4317",
    SamplingRate:   0.5, // 50% 采样
}
```

## 采样策略

采样率 (SamplingRate) 决定了多少比例的请求会被追踪：

- `1.0`: 100% 采样，追踪所有请求（适合开发环境）
- `0.1`: 10% 采样，追踪 10% 的请求（适合生产环境）
- `0.01`: 1% 采样，追踪 1% 的请求（适合高流量生产环境）

采样器使用 ParentBased 策略，如果父 Span 被采样，则子 Span 也会被采样。

## 上下文信息自动提取

追踪模块会自动从 Context 中提取以下信息并添加到 Span：

- `session.id`: 会话 ID
- `user.id`: 用户 ID
- `tenant.id`: 租户 ID
- `trace.id`: 追踪 ID（用于日志关联）

确保在 Context 中设置这些值：

```go
ctx = context.WithValue(ctx, "session_id", sessionID)
ctx = context.WithValue(ctx, "user_id", userID)
ctx = context.WithValue(ctx, "tenant_id", tenantID)
ctx = context.WithValue(ctx, "traceId", traceID)
```

## Jaeger UI

### 启动 Jaeger

使用 Docker 启动 Jaeger（支持 OTLP）：

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

端口说明：

- `16686`: Jaeger UI
- `4317`: OTLP gRPC 接收器
- `4318`: OTLP HTTP 接收器

### 访问 UI

打开浏览器访问：<http://localhost:16686>

在 UI 中可以：

- 查看服务列表
- 搜索追踪记录
- 查看追踪详情和时间线
- 分析性能瓶颈
- 查看依赖关系图

## 性能考虑

1. **采样率**: 生产环境建议使用较低的采样率（如 10%）以减少性能开销
2. **批处理**: 追踪数据会批量发送到 Jaeger，默认批处理超时为 5 秒
3. **异步导出**: Span 数据异步导出，不会阻塞主流程
4. **资源限制**: 默认最大批处理大小为 512 个 Span

## 最佳实践

1. **命名规范**:
   - Flow: 使用 Flow 名称，如 "contextBuildFlow"
   - Service: 使用 "ServiceName.MethodName" 格式
   - Repository: 使用 "RepositoryName.Operation" 格式
   - External: 使用 "external.ServiceName.Operation" 格式

2. **属性添加**:
   - 添加有意义的属性帮助调试
   - 避免添加敏感信息（密码、令牌等）
   - 使用标准的属性键（如 db.system, http.method 等）

3. **错误处理**:
   - 始终记录错误到 Span
   - 使用 RecordError 或 SetSpanError
   - 设置正确的状态码

4. **Span 生命周期**:
   - 使用 defer span.End() 确保 Span 被关闭
   - 在函数返回前设置最终状态
   - 避免创建过多的细粒度 Span

5. **上下文传递**:
   - 始终传递 Context 到下游函数
   - 不要创建新的 Context，使用返回的 Context
   - 确保 Context 包含必要的信息

## 故障排查

### 追踪数据未显示在 Jaeger

1. 检查 Jaeger 是否运行：`docker ps | grep jaeger`
2. 检查端点配置是否正确
3. 检查采样率是否过低
4. 查看应用日志是否有错误

### 性能问题

1. 降低采样率
2. 检查批处理配置
3. 确保 Jaeger 有足够的资源
4. 考虑使用 Jaeger Agent 而不是直接连接 Collector

### 内存泄漏

1. 确保所有 Span 都调用了 End()
2. 检查是否有 goroutine 泄漏
3. 使用 pprof 分析内存使用

## 依赖项

```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
)
```

注意：我们使用 OTLP 导出器而不是已弃用的 Jaeger 导出器。OTLP 是 OpenTelemetry 的标准协议，Jaeger 0.14+ 完全支持。

## 相关文档

- [OpenTelemetry Go 文档](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [需求文档](../../.kiro/specs/genkit-session-management/requirements.md) - 需求 28
- [设计文档](../../.kiro/specs/genkit-session-management/design.md) - 性能追踪设计

## 示例项目

完整的示例代码请参考：

- Flow 追踪: `internal/genkit/flows/`
- Service 追踪: `internal/service/`
- Repository 追踪: `internal/repository/`
