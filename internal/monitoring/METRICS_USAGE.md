# 监控指标使用指南

## 概述

本模块提供了完整的监控指标收集功能，支持 Flow 执行、Token 使用、缓存性能等多个维度的指标记录。

## 核心功能

### 1. Flow 执行指标

记录 Genkit Flow 的执行情况，包括执行次数、成功率和执行时间。

#### 使用示例

```go
import (
    "time"
    "genkit-ai-service/internal/monitoring"
)

// 记录 Flow 执行
func executeFlow(flowName string) error {
    startTime := time.Now()
    
    // 执行 Flow 逻辑
    err := doSomething()
    
    // 记录执行时间
    duration := time.Since(startTime)
    monitoring.RecordFlowDuration(flowName, duration)
    
    // 记录执行结果
    if err != nil {
        monitoring.RecordFlowExecution(flowName, "error")
        return err
    }
    
    monitoring.RecordFlowExecution(flowName, "success")
    return nil
}

// 获取 Flow 指标
func getFlowStats(flowName string) {
    metrics := monitoring.GetMetrics()
    flowMetrics := metrics.GetFlowMetrics(flowName)
    
    fmt.Printf("Flow: %s\n", flowMetrics.FlowName)
    fmt.Printf("执行次数: %d\n", flowMetrics.Executions)
    fmt.Printf("成功次数: %d\n", flowMetrics.Successes)
    fmt.Printf("错误次数: %d\n", flowMetrics.Errors)
    fmt.Printf("成功率: %.2f%%\n", flowMetrics.SuccessRate)
    fmt.Printf("平均执行时间: %.2fms\n", flowMetrics.AvgDuration)
    fmt.Printf("P95 执行时间: %.2fms\n", flowMetrics.P95Duration)
    fmt.Printf("P99 执行时间: %.2fms\n", flowMetrics.P99Duration)
}
```

### 2. Token 使用量指标

记录 AI 模型的 Token 使用情况，支持按租户统计。

#### 使用示例

```go
// 记录 Token 使用
func recordAIUsage(tenantID string, response AIResponse) {
    monitoring.RecordTokenUsage(
        tenantID,
        response.PromptTokens,
        response.CompletionTokens,
    )
}

// 获取 Token 使用统计
func getTokenStats() {
    metrics := monitoring.GetMetrics()
    tokenMetrics := metrics.GetTokenMetrics()
    
    fmt.Printf("Prompt Token 总数: %d\n", tokenMetrics.PromptTokens)
    fmt.Printf("Completion Token 总数: %d\n", tokenMetrics.CompletionTokens)
    fmt.Printf("总 Token 数: %d\n", tokenMetrics.TotalTokens)
    
    // 按租户查看
    for tenantID, usage := range tokenMetrics.ByTenant {
        fmt.Printf("租户 %s: %d tokens\n", tenantID, usage)
    }
}
```

### 3. 缓存性能指标

记录缓存的命中和未命中情况，用于优化缓存策略。

#### 使用示例

```go
// 在缓存服务中使用
func (s *CacheService) Get(key string) (interface{}, error) {
    value, err := s.redis.Get(ctx, key).Result()
    
    if err == redis.Nil {
        // 缓存未命中
        monitoring.RecordCacheMiss(key)
        return nil, ErrCacheMiss
    }
    
    if err != nil {
        return nil, err
    }
    
    // 缓存命中
    monitoring.RecordCacheHit(key)
    return value, nil
}

// 获取缓存统计
func getCacheStats() {
    metrics := monitoring.GetMetrics()
    cacheMetrics := metrics.GetCacheMetrics()
    
    fmt.Printf("缓存命中次数: %d\n", cacheMetrics.Hits)
    fmt.Printf("缓存未命中次数: %d\n", cacheMetrics.Misses)
    fmt.Printf("缓存命中率: %.2f%%\n", cacheMetrics.HitRate)
}
```

## 在 Genkit Flow 中集成

### Flow 中间件模式

```go
// 创建 Flow 监控中间件
func MonitorFlow(flowName string, fn func() error) error {
    startTime := time.Now()
    
    err := fn()
    
    duration := time.Since(startTime)
    monitoring.RecordFlowDuration(flowName, duration)
    
    if err != nil {
        monitoring.RecordFlowExecution(flowName, "error")
        return err
    }
    
    monitoring.RecordFlowExecution(flowName, "success")
    return nil
}

// 在 Flow 中使用
func contextBuildFlow(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
    var output ContextBuildOutput
    
    err := MonitorFlow("contextBuildFlow", func() error {
        // Flow 逻辑
        result, err := buildContext(ctx, input)
        if err != nil {
            return err
        }
        output = result
        return nil
    })
    
    return output, err
}
```

### 完整的 Flow 示例

```go
func chatGenerateFlow(ctx context.Context, input ChatGenerateInput) (ChatGenerateOutput, error) {
    startTime := time.Now()
    flowName := "chatGenerateFlow"
    
    // 执行 Flow 逻辑
    output, err := generateChat(ctx, input)
    
    // 记录执行时间
    duration := time.Since(startTime)
    monitoring.RecordFlowDuration(flowName, duration)
    
    // 记录执行结果
    if err != nil {
        monitoring.RecordFlowExecution(flowName, "error")
        return ChatGenerateOutput{}, err
    }
    
    monitoring.RecordFlowExecution(flowName, "success")
    
    // 记录 Token 使用
    if output.TokenUsage.TotalTokens > 0 {
        tenantID := getTenantIDFromContext(ctx)
        monitoring.RecordTokenUsage(
            tenantID,
            output.TokenUsage.PromptTokens,
            output.TokenUsage.CompletionTokens,
        )
    }
    
    return output, nil
}
```

## API 端点示例

### 监控指标查询接口

```go
// Handler: 获取 Flow 指标
func (h *MonitoringHandler) HandleGetFlowMetrics(c *gin.Context) {
    flowName := c.Param("flowName")
    
    metrics := monitoring.GetMetrics()
    flowMetrics := metrics.GetFlowMetrics(flowName)
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "获取 Flow 指标成功",
        "data":    flowMetrics,
    })
}

// Handler: 获取 Token 使用统计
func (h *MonitoringHandler) HandleGetTokenMetrics(c *gin.Context) {
    metrics := monitoring.GetMetrics()
    tokenMetrics := metrics.GetTokenMetrics()
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "获取 Token 使用统计成功",
        "data":    tokenMetrics,
    })
}

// Handler: 获取缓存统计
func (h *MonitoringHandler) HandleGetCacheMetrics(c *gin.Context) {
    metrics := monitoring.GetMetrics()
    cacheMetrics := metrics.GetCacheMetrics()
    
    c.JSON(200, gin.H{
        "code":    200,
        "message": "获取缓存统计成功",
        "data":    cacheMetrics,
    })
}
```

## 最佳实践

### 1. 命名规范

- Flow 名称使用驼峰命名：`contextBuildFlow`、`chatGenerateFlow`
- 缓存键使用冒号分隔：`context:session-id`、`summary:session-id`

### 2. 性能考虑

- 指标记录操作是线程安全的，可以在并发环境中使用
- 每个指标类型保留最近 1000 条记录，避免内存溢出
- 使用读写锁优化并发性能

### 3. 监控告警

结合告警系统使用：

```go
// 检查 Flow 性能
func checkFlowPerformance() {
    metrics := monitoring.GetMetrics()
    flowMetrics := metrics.GetFlowMetrics("contextBuildFlow")
    
    // P95 超过 500ms 触发告警
    if flowMetrics.P95Duration > 500 {
        alert.Send("contextBuildFlow P95 延迟过高: %.2fms", flowMetrics.P95Duration)
    }
    
    // 错误率超过 10% 触发告警
    if flowMetrics.SuccessRate < 90 {
        alert.Send("contextBuildFlow 成功率过低: %.2f%%", flowMetrics.SuccessRate)
    }
}

// 检查缓存性能
func checkCachePerformance() {
    metrics := monitoring.GetMetrics()
    cacheMetrics := metrics.GetCacheMetrics()
    
    // 命中率低于 70% 触发告警
    if cacheMetrics.HitRate < 70 {
        alert.Send("缓存命中率过低: %.2f%%", cacheMetrics.HitRate)
    }
}
```

### 4. 定期重置

对于长期运行的服务，可以定期重置指标以避免数据过时：

```go
// 每天凌晨重置指标
func scheduleMetricsReset() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := monitoring.GetMetrics()
        
        // 保存当前快照
        snapshot := metrics.GetSnapshot()
        saveSnapshot(snapshot)
        
        // 重置指标
        metrics.Reset()
    }
}
```

## 数据结构

### FlowMetrics

```go
type FlowMetrics struct {
    FlowName    string  // Flow 名称
    Executions  int64   // 执行次数
    Successes   int64   // 成功次数
    Errors      int64   // 错误次数
    SuccessRate float64 // 成功率（百分比）
    AvgDuration float64 // 平均执行时间（毫秒）
    P95Duration float64 // P95 执行时间（毫秒）
    P99Duration float64 // P99 执行时间（毫秒）
}
```

### TokenMetrics

```go
type TokenMetrics struct {
    PromptTokens     int64            // Prompt Token 总数
    CompletionTokens int64            // Completion Token 总数
    TotalTokens      int64            // 总 Token 数
    ByTenant         map[string]int64 // 按租户统计
}
```

### CacheMetrics

```go
type CacheMetrics struct {
    Hits    int64   // 缓存命中次数
    Misses  int64   // 缓存未命中次数
    HitRate float64 // 缓存命中率（百分比）
}
```

## 注意事项

1. **线程安全**：所有指标记录方法都是线程安全的，可以在并发环境中使用
2. **内存管理**：每个指标类型保留最近 1000 条记录，自动清理旧数据
3. **性能影响**：指标记录操作非常轻量，对系统性能影响极小
4. **数据持久化**：当前实现是内存存储，重启后数据会丢失。如需持久化，可以定期导出到数据库或时序数据库

## 未来扩展

### Prometheus 集成

```go
// TODO: 添加 Prometheus 指标导出
import "github.com/prometheus/client_golang/prometheus"

var (
    flowExecutions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "flow_executions_total",
            Help: "Total number of flow executions",
        },
        []string{"flow_name", "status"},
    )
    
    flowDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "flow_duration_seconds",
            Help:    "Flow execution duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"flow_name"},
    )
)
```

### Grafana 仪表板

可以基于这些指标创建 Grafana 仪表板，实时监控系统性能。

## 相关文档

- [告警系统使用指南](./ALERTS_USAGE.md)
- [性能优化指南](../../docs/PERFORMANCE_GUIDE.md)
- [监控最佳实践](../../docs/MONITORING_GUIDE.md)
