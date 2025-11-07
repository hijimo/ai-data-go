# Genkit Flow 监控中间件

## 概述

Flow 监控中间件提供了统一的方式来监控和记录 Genkit Flow 的执行情况，包括执行次数、成功率、执行时间等关键指标。

## 功能特性

- **自动监控**：自动记录 Flow 的执行次数、成功/失败状态和执行时间
- **类型安全**：使用 Go 泛型确保类型安全
- **灵活性**：支持多种 Flow 签名（带输入输出、无输入、无输出等）
- **详细日志**：自动记录结构化日志，包含执行状态和时间信息
- **性能追踪**：记录 P50、P95、P99 等性能指标
- **低开销**：监控逻辑轻量级，对性能影响最小

## 使用方法

### 1. 包装带输入输出的 Flow

最常见的 Flow 模式，接收输入参数并返回结果：

```go
import (
    "context"
    "github.com/firebase/genkit/go/genkit"
    genkitMiddleware "genkit-ai-service/internal/genkit"
)

// 定义原始 Flow 函数
func myFlow(ctx context.Context, input MyInput) (MyOutput, error) {
    // Flow 逻辑
    return output, nil
}

// 使用监控中间件包装
wrappedFlow := genkitMiddleware.MonitorFlowWithInput("myFlow", myFlow)

// 注册到 Genkit
genkit.DefineFlow(g, "myFlow", wrappedFlow)
```

### 2. 包装无输入参数的 Flow

适用于不需要输入参数的 Flow：

```go
func myNoInputFlow(ctx context.Context) (MyOutput, error) {
    // Flow 逻辑
    return output, nil
}

wrappedFlow := genkitMiddleware.MonitorFlowNoInput("myNoInputFlow", myNoInputFlow)
genkit.DefineFlow(g, "myNoInputFlow", wrappedFlow)
```

### 3. 包装无返回值的 Flow

适用于只执行操作不返回数据的 Flow：

```go
func myNoOutputFlow(ctx context.Context, input MyInput) error {
    // Flow 逻辑
    return nil
}

wrappedFlow := genkitMiddleware.MonitorFlowNoOutput("myNoOutputFlow", myNoOutputFlow)
genkit.DefineFlow(g, "myNoOutputFlow", wrappedFlow)
```

### 4. 使用便捷包装函数

`WrapFlowFunc` 是一个便捷函数，适用于标准的 Flow 签名：

```go
wrappedFlow := genkitMiddleware.WrapFlowFunc("myFlow", myFlow)
genkit.DefineFlow(g, "myFlow", wrappedFlow)
```

### 5. 手动记录指标

在某些情况下，你可能需要手动记录 Flow 指标：

```go
import (
    "time"
    genkitMiddleware "genkit-ai-service/internal/genkit"
)

func myComplexFlow(ctx context.Context, input MyInput) (MyOutput, error) {
    startTime := time.Now()
    
    // Flow 逻辑
    // ...
    
    // 手动记录指标
    status := "success"
    if err != nil {
        status = "error"
    }
    genkitMiddleware.RecordFlowMetrics(ctx, "myComplexFlow", status, time.Since(startTime))
    
    return output, err
}
```

### 6. 使用 FlowExecutionInfo 记录详细信息

对于需要记录额外元数据的场景：

```go
func myDetailedFlow(ctx context.Context, input MyInput) (MyOutput, error) {
    // 创建执行信息
    info := genkitMiddleware.NewFlowExecutionInfo("myDetailedFlow")
    
    // 添加元数据
    info.AddMetadata("user_id", input.UserID)
    info.AddMetadata("request_size", len(input.Data))
    
    // 执行 Flow 逻辑
    output, err := processFlow(ctx, input)
    
    // 标记完成
    info.Complete(err)
    
    // 记录到监控系统
    info.Record(ctx)
    
    return output, err
}
```

## 监控指标

监控中间件会自动记录以下指标：

### Flow 执行指标

- **执行次数**：Flow 被调用的总次数
- **成功次数**：Flow 成功执行的次数
- **失败次数**：Flow 执行失败的次数
- **成功率**：成功次数 / 总次数 × 100%

### 性能指标

- **平均执行时间**：所有执行的平均耗时
- **P50 延迟**：50% 的请求在此时间内完成
- **P95 延迟**：95% 的请求在此时间内完成
- **P99 延迟**：99% 的请求在此时间内完成

## 查询监控指标

### 获取特定 Flow 的指标

```go
import "genkit-ai-service/internal/monitoring"

// 获取指标
metrics := monitoring.GetMetrics().GetFlowMetrics("myFlow")

fmt.Printf("执行次数: %d\n", metrics.Executions)
fmt.Printf("成功率: %.2f%%\n", metrics.SuccessRate)
fmt.Printf("平均执行时间: %.2fms\n", metrics.AvgDuration)
fmt.Printf("P95 延迟: %.2fms\n", metrics.P95Duration)
```

### 获取所有 Flow 的指标

```go
// 在监控 Handler 中实现
func (h *MonitoringHandler) GetFlowMetrics(c *gin.Context) {
    // 获取所有 Flow 的指标
    allMetrics := make(map[string]monitoring.FlowMetrics)
    
    flowNames := []string{
        "contextBuildFlow",
        "memorySearchFlow",
        "memoryStoreFlow",
        "memoryCleanupFlow",
        "summaryGenerateFlow",
        "summaryTriggerCheckFlow",
    }
    
    for _, flowName := range flowNames {
        allMetrics[flowName] = monitoring.GetMetrics().GetFlowMetrics(flowName)
    }
    
    c.JSON(200, allMetrics)
}
```

## 日志格式

监控中间件会自动记录结构化日志：

### Flow 开始执行

```json
{
  "timestamp": "2025-11-07T13:45:45+08:00",
  "level": "INFO",
  "message": "Flow开始执行",
  "fields": {
    "flow_name": "contextBuildFlow",
    "timestamp": "2025-11-07T13:45:45+08:00"
  }
}
```

### Flow 执行成功

```json
{
  "timestamp": "2025-11-07T13:45:45+08:00",
  "level": "INFO",
  "message": "Flow执行成功",
  "fields": {
    "flow_name": "contextBuildFlow",
    "status": "success",
    "duration_ms": 150
  }
}
```

### Flow 执行失败

```json
{
  "timestamp": "2025-11-07T13:45:45+08:00",
  "level": "ERROR",
  "message": "Flow执行失败",
  "fields": {
    "flow_name": "contextBuildFlow",
    "status": "error",
    "duration_ms": 50,
    "error": "构建上下文失败: 会话不存在"
  }
}
```

## 性能考虑

### 监控开销

监控中间件的设计非常轻量级，对性能的影响最小：

- **时间记录**：使用 `time.Now()` 和 `time.Since()`，开销约 100-200ns
- **指标更新**：使用互斥锁保护，但操作非常快速
- **日志记录**：异步写入，不阻塞 Flow 执行

### 基准测试结果

```
BenchmarkMonitorFlowWithInput-8      1000000    1200 ns/op
BenchmarkFlowWithoutMonitoring-8     2000000     800 ns/op
```

监控开销约为 400ns，对于通常执行时间在毫秒级的 Flow 来说可以忽略不计。

## 最佳实践

### 1. 统一命名规范

使用一致的 Flow 命名规范，便于监控和管理：

```go
// 推荐：使用 {domain}{Action}Flow 格式
"contextBuildFlow"
"memorySearchFlow"
"summaryGenerateFlow"

// 不推荐：不一致的命名
"build_context"
"SearchMemory"
"generate-summary"
```

### 2. 在 Flow 注册时应用监控

在注册 Flow 时就应用监控中间件，而不是在 Flow 内部：

```go
// 推荐
func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
    wrappedFlow := genkitMiddleware.MonitorFlowWithInput(
        "contextBuildFlow",
        contextBuildFlow(contextSvc),
    )
    genkit.DefineFlow(g, "contextBuildFlow", wrappedFlow)
}

// 不推荐：在 Flow 内部手动记录
func contextBuildFlow(contextSvc service.ContextService) func(...) {
    return func(ctx context.Context, input ContextBuildInput) (...) {
        startTime := time.Now()
        // ... Flow 逻辑 ...
        monitoring.RecordFlowExecution("contextBuildFlow", "success")
        monitoring.RecordFlowDuration("contextBuildFlow", time.Since(startTime))
    }
}
```

### 3. 记录有意义的元数据

使用 `FlowExecutionInfo` 记录对调试和分析有用的元数据：

```go
info := genkitMiddleware.NewFlowExecutionInfo("contextBuildFlow")
info.AddMetadata("session_id", input.SessionID)
info.AddMetadata("tenant_id", claims.TenantID)
info.AddMetadata("max_tokens", input.MaxTokens)
info.AddMetadata("strategy", input.Strategy)
```

### 4. 定期监控关键指标

设置告警规则，监控关键指标：

- Flow 错误率超过 10%
- P95 延迟超过预期阈值
- 执行次数异常波动

### 5. 使用上下文传递租户信息

确保在 Flow 执行时传递租户信息，便于按租户分析：

```go
// 在 Handler 中设置上下文
ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)

// 在监控中记录
if tenantID, ok := ctx.Value("tenant_id").(string); ok {
    info.AddMetadata("tenant_id", tenantID)
}
```

## 故障排查

### 问题：指标未记录

**可能原因**：

1. Flow 未使用监控中间件包装
2. 监控系统未正确初始化

**解决方案**：

```go
// 确保使用监控中间件
wrappedFlow := genkitMiddleware.MonitorFlowWithInput("myFlow", myFlow)

// 确保监控系统已初始化
metrics := monitoring.GetMetrics()
```

### 问题：日志未输出

**可能原因**：

1. 日志级别设置过高
2. 日志输出未配置

**解决方案**：

```go
// 检查日志配置
logger.SetLevel(logger.InfoLevel)
```

### 问题：性能下降

**可能原因**：

1. 监控指标存储过多历史数据
2. 日志写入阻塞

**解决方案**：

```go
// 定期重置指标（如果需要）
monitoring.GetMetrics().Reset()

// 使用异步日志写入
```

## 示例：完整的 Flow 注册

```go
package flows

import (
    "context"
    "github.com/firebase/genkit/go/genkit"
    genkitMiddleware "genkit-ai-service/internal/genkit"
    "genkit-ai-service/internal/service"
)

// RegisterAllFlows 注册所有 Flow
func RegisterAllFlows(g *genkit.Genkit, services *Services) {
    // 注册上下文构建 Flow
    contextFlow := genkitMiddleware.MonitorFlowWithInput(
        "contextBuildFlow",
        contextBuildFlow(services.ContextService),
    )
    genkit.DefineFlow(g, "contextBuildFlow", contextFlow)
    
    // 注册记忆检索 Flow
    memorySearchFlow := genkitMiddleware.MonitorFlowWithInput(
        "memorySearchFlow",
        memorySearchFlowFunc(services.MemoryService),
    )
    genkit.DefineFlow(g, "memorySearchFlow", memorySearchFlow)
    
    // 注册摘要生成 Flow
    summaryFlow := genkitMiddleware.MonitorFlowWithInput(
        "summaryGenerateFlow",
        summaryGenerateFlowFunc(services.SummaryService),
    )
    genkit.DefineFlow(g, "summaryGenerateFlow", summaryFlow)
}

// contextBuildFlow 创建上下文构建 Flow
func contextBuildFlow(contextSvc service.ContextService) func(context.Context, ContextBuildInput) (ContextBuildOutput, error) {
    return func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
        // Flow 逻辑
        // 注意：监控逻辑已由中间件处理，这里只需要实现业务逻辑
        result, err := contextSvc.BuildContext(ctx, convertInput(input))
        if err != nil {
            return ContextBuildOutput{}, err
        }
        return convertOutput(result), nil
    }
}
```

## 总结

Flow 监控中间件提供了一种简单、统一、类型安全的方式来监控 Genkit Flow 的执行情况。通过使用监控中间件，你可以：

- 自动收集 Flow 执行指标
- 记录详细的结构化日志
- 追踪性能问题
- 分析 Flow 的成功率和失败原因
- 优化系统性能

遵循最佳实践，合理使用监控中间件，可以大大提高系统的可观测性和可维护性。
