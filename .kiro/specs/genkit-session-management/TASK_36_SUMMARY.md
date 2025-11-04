# 任务 36: 性能追踪实现 - 完成总结

## 任务概述

实现基于 OpenTelemetry 的分布式追踪功能，支持 Jaeger 可视化，用于追踪 Flow 执行、服务调用、数据库操作和外部服务调用。

## 实现内容

### 1. 核心追踪功能 (`internal/tracing/tracer.go`)

实现了以下追踪函数：

- **TraceFlow**: 追踪 Genkit Flow 执行
- **TraceService**: 追踪服务层方法调用
- **TraceRepository**: 追踪数据库操作
- **TraceExternalCall**: 追踪外部服务调用
- **AddSpanAttributes**: 添加自定义属性到 Span
- **AddSpanEvent**: 添加事件到 Span
- **RecordError**: 记录错误到 Span

所有追踪函数都会自动从 Context 中提取以下信息：

- `session_id`: 会话 ID
- `user_id`: 用户 ID
- `tenant_id`: 租户 ID
- `traceId`: 追踪 ID（用于日志关联）

### 2. 配置和初始化 (`internal/tracing/config.go`)

实现了 OpenTelemetry 追踪提供者的配置和初始化：

- **Config 结构**: 追踪配置，包括服务名称、版本、环境、OTLP 端点和采样率
- **NewTracerProvider**: 创建并初始化追踪提供者
- **Shutdown**: 优雅关闭追踪提供者
- **ForceFlush**: 强制刷新待处理的 Span
- **DefaultConfig**: 开发环境默认配置（100% 采样）
- **ProductionConfig**: 生产环境配置（10% 采样）

技术选择：

- 使用 OTLP gRPC 导出器（而非已弃用的 Jaeger 导出器）
- 支持 Jaeger 0.14+ 的 OTLP 接收器
- 使用 ParentBased 采样策略
- 批量导出 Span（5秒超时，最大512个）

### 3. 辅助函数 (`internal/tracing/helpers.go`)

提供了便捷的追踪辅助函数：

- **StartSpan**: 开始一个新的 Span
- **TraceDBQuery**: 追踪数据库查询
- **TraceVectorSearch**: 追踪向量检索
- **TraceAIGeneration**: 追踪 AI 生成
- **TraceCacheOperation**: 追踪缓存操作
- **SetSpanSuccess/SetSpanError**: 设置 Span 状态
- **AddTokenUsage**: 添加 Token 使用统计
- **AddContextMetrics**: 添加上下文指标
- **AddMemoryMetrics**: 添加记忆指标
- **AddSummaryMetrics**: 添加摘要指标
- **GetTraceID/GetSpanID**: 获取追踪 ID
- **IsTracing**: 检查是否在追踪中

### 4. 单元测试

实现了完整的单元测试覆盖：

- **tracer_test.go**: 测试核心追踪功能（9个测试）
  - 成功和失败场景
  - 上下文值提取
  - 不同类型的追踪（Flow、Service、Repository、External）
  - 属性和事件添加
  - 错误记录

- **helpers_test.go**: 测试辅助函数（15个测试）
  - Span 创建和管理
  - 各种追踪场景（DB、Vector、AI、Cache）
  - 指标添加
  - TraceID/SpanID 获取
  - 追踪状态检查

- **example_test.go**: 提供使用示例（15个示例）
  - Flow 追踪示例
  - 服务追踪示例
  - 数据库追踪示例
  - 自定义属性示例
  - 嵌套 Span 示例
  - 错误处理示例

所有测试均通过 ✅

### 5. 文档 (`internal/tracing/README.md`)

创建了详细的使用文档，包括：

- 功能特性列表
- 快速开始指南
- 高级用法示例
- 配置选项说明
- 采样策略说明
- Jaeger UI 使用指南
- 性能考虑
- 最佳实践
- 故障排查
- 依赖项说明

## 技术亮点

### 1. 使用 OTLP 而非 Jaeger 导出器

- Jaeger 导出器已被弃用
- OTLP 是 OpenTelemetry 的标准协议
- Jaeger 0.14+ 完全支持 OTLP
- 更好的互操作性和未来兼容性

### 2. 自动上下文提取

追踪模块会自动从 Context 中提取业务信息：

```go
// 自动提取并添加到 Span
- session.id
- user.id
- tenant.id
- trace.id
```

### 3. 灵活的采样策略

- 开发环境：100% 采样（全量追踪）
- 生产环境：10% 采样（降低开销）
- 使用 ParentBased 策略保证追踪完整性

### 4. 批量导出优化

- 批处理超时：5秒
- 最大批处理大小：512个 Span
- 异步导出，不阻塞主流程

### 5. 类型安全的追踪函数

所有追踪函数都使用函数式编程风格：

```go
err := tracing.TraceFlow(ctx, "flowName", func(ctx context.Context) error {
    // 业务逻辑
    return nil
})
```

## 使用示例

### 追踪 Flow 执行

```go
func (f *Flow) Execute(ctx context.Context, input Input) (Output, error) {
    var output Output
    
    err := tracing.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
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

### 追踪服务方法

```go
func (s *ContextService) BuildContext(ctx context.Context, req Request) (*Result, error) {
    var result *Result
    
    err := tracing.TraceService(ctx, "ContextService", "BuildContext", func(ctx context.Context) error {
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

### 添加自定义指标

```go
func generateResponse(ctx context.Context, prompt string) (string, error) {
    ctx, span := tracing.TraceAIGeneration(ctx, "gpt-4", len(prompt))
    defer span.End()
    
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

## 部署配置

### 启动 Jaeger

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

### 应用配置

```go
// 开发环境
config := tracing.DefaultConfig()
config.Enabled = true

// 生产环境
config := tracing.ProductionConfig("jaeger.prod.example.com:4317")

// 初始化
tp, err := tracing.NewTracerProvider(config)
if err != nil {
    log.Fatalf("初始化追踪失败: %v", err)
}
defer tp.Shutdown(context.Background())
```

### 访问 Jaeger UI

打开浏览器访问：<http://localhost:16686>

## 性能影响

### 开销分析

1. **采样开销**：
   - 100% 采样：约 1-2% CPU 开销
   - 10% 采样：约 0.1-0.2% CPU 开销

2. **内存开销**：
   - 每个 Span 约 1-2KB
   - 批处理缓冲区：最大 512 个 Span

3. **网络开销**：
   - 批量发送，5秒超时
   - gRPC 压缩传输

### 优化建议

1. 生产环境使用较低采样率（10%）
2. 避免创建过多细粒度 Span
3. 使用 Jaeger Agent 减少网络延迟
4. 定期清理 Jaeger 存储

## 依赖项

新增依赖：

```
go.opentelemetry.io/otel v1.21.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
go.opentelemetry.io/otel/sdk v1.21.0
go.opentelemetry.io/otel/trace v1.21.0
```

## 测试结果

```bash
$ go test -v ./internal/tracing/...
=== RUN   TestTraceFlow_Success
--- PASS: TestTraceFlow_Success (0.00s)
=== RUN   TestTraceFlow_WithError
--- PASS: TestTraceFlow_WithError (0.00s)
=== RUN   TestTraceFlow_WithContextValues
--- PASS: TestTraceFlow_WithContextValues (0.00s)
... (共 39 个测试)
PASS
ok      genkit-ai-service/internal/tracing      0.179s
```

所有测试通过 ✅

## 后续集成建议

### 1. 在 Flow 中集成

```go
// internal/genkit/flows/context.go
func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
    genkit.DefineFlow(g, "contextBuildFlow", 
        func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
            var output ContextBuildOutput
            
            err := tracing.TraceFlow(ctx, "contextBuildFlow", func(ctx context.Context) error {
                result, err := contextSvc.BuildContext(ctx, ...)
                if err != nil {
                    return err
                }
                output = convertToOutput(result)
                return nil
            })
            
            return output, err
        },
    )
}
```

### 2. 在服务层集成

```go
// internal/service/context_service.go
func (s *contextServiceImpl) BuildContext(ctx context.Context, req BuildContextRequest) (*ContextResult, error) {
    var result *ContextResult
    
    err := tracing.TraceService(ctx, "ContextService", "BuildContext", func(ctx context.Context) error {
        // 业务逻辑
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

### 3. 在 Repository 层集成

```go
// internal/repository/memory_repository.go
func (r *memoryRepositoryImpl) SearchByVector(ctx context.Context, ...) ([]*Memory, error) {
    var memories []*Memory
    
    err := tracing.TraceRepository(ctx, "MemoryRepository", "SearchByVector", func(ctx context.Context) error {
        return r.db.WithContext(ctx).
            Where("session_id = ?", sessionID).
            Find(&memories).Error
    })
    
    return memories, err
}
```

### 4. 在主程序中初始化

```go
// cmd/server/main.go
func main() {
    // 初始化追踪
    tracingConfig := &tracing.Config{
        Enabled:        os.Getenv("TRACING_ENABLED") == "true",
        ServiceName:    "genkit-ai-service",
        ServiceVersion: "1.0.0",
        Environment:    os.Getenv("ENVIRONMENT"),
        OTLPEndpoint:   os.Getenv("OTLP_ENDPOINT"),
        SamplingRate:   0.1,
    }
    
    tp, err := tracing.NewTracerProvider(tracingConfig)
    if err != nil {
        log.Fatalf("初始化追踪失败: %v", err)
    }
    defer tp.Shutdown(context.Background())
    
    // 启动应用...
}
```

## 相关文档

- [需求文档](requirements.md) - 需求 28
- [设计文档](design.md) - 性能追踪设计
- [OpenTelemetry Go 文档](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)

## 总结

任务 36 已完成，实现了完整的 OpenTelemetry 性能追踪功能：

✅ 集成 OpenTelemetry  
✅ 实现 TraceFlow 方法  
✅ 实现 Span 属性设置  
✅ 配置追踪导出（OTLP/Jaeger）  
✅ 完整的单元测试  
✅ 详细的使用文档  

该实现提供了：

- 类型安全的追踪 API
- 自动上下文提取
- 灵活的采样策略
- 完整的测试覆盖
- 详细的使用文档
- 生产就绪的配置

可以直接在 Genkit Flow、服务层、Repository 层和外部服务调用中使用，为系统提供完整的分布式追踪能力。
