# 熔断器使用指南

## 概述

熔断器（Circuit Breaker）是一种用于防止系统级联故障的设计模式。当外部服务（如AI服务）频繁失败时，熔断器会自动"打开"，拒绝后续请求，避免浪费资源。经过一段时间后，熔断器会进入"半开"状态，允许少量请求通过以测试服务是否恢复。如果测试成功，熔断器会"关闭"，恢复正常服务。

## 熔断器状态

熔断器有三种状态：

### 1. Closed（关闭）

- **行为**：正常执行所有请求
- **转换条件**：当失败次数达到阈值时，转换为 Open 状态

### 2. Open（打开）

- **行为**：拒绝所有请求，立即返回错误
- **转换条件**：超时后自动转换为 HalfOpen 状态

### 3. HalfOpen（半开）

- **行为**：允许有限数量的请求通过以测试服务
- **转换条件**：
  - 如果连续成功达到阈值，转换为 Closed 状态
  - 如果任何请求失败，立即转换回 Open 状态

## 配置参数

```go
type CircuitBreakerConfig struct {
    // MaxFailures 触发熔断的最大失败次数
    MaxFailures int
    
    // Timeout 熔断器打开后的超时时间，超时后进入半开状态
    Timeout time.Duration
    
    // HalfOpenMaxRequests 半开状态下允许的最大请求数
    HalfOpenMaxRequests int
    
    // SuccessThreshold 半开状态下连续成功次数达到此阈值后关闭熔断器
    SuccessThreshold int
    
    // OnStateChange 状态变化回调函数
    OnStateChange func(from, to CircuitState)
}
```

### 默认配置

```go
config := &CircuitBreakerConfig{
    MaxFailures:         5,                // 5次失败后打开熔断器
    Timeout:             30 * time.Second, // 30秒后进入半开状态
    HalfOpenMaxRequests: 3,                // 半开状态允许3个请求
    SuccessThreshold:    2,                // 连续2次成功后关闭熔断器
}
```

## 使用示例

### 1. 基本使用

```go
package main

import (
    "context"
    "time"
    
    "genkit-ai-service/internal/middleware"
)

func main() {
    // 创建熔断器
    config := &middleware.CircuitBreakerConfig{
        MaxFailures:         5,
        Timeout:             30 * time.Second,
        HalfOpenMaxRequests: 3,
        SuccessThreshold:    2,
    }
    
    cb := middleware.NewCircuitBreaker("my-service", config)
    
    // 使用熔断器执行操作
    ctx := context.Background()
    result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
        // 调用外部服务
        return callExternalService()
    })
    
    if err != nil {
        if err == middleware.ErrCircuitBreakerOpen {
            // 熔断器已打开，执行降级逻辑
            return handleDegradation()
        }
        return err
    }
    
    // 处理结果
    processResult(result)
}
```

### 2. 在 Genkit AI 服务中使用

```go
package main

import (
    "context"
    
    "genkit-ai-service/internal/genkit"
)

func main() {
    // 创建基础客户端
    baseClient := genkit.NewClient()
    
    // 初始化
    ctx := context.Background()
    config := &genkit.Config{
        APIKey: "your-api-key",
        Model:  "gemini-pro",
    }
    baseClient.Initialize(ctx, config)
    baseClient.InitializeModel(ctx)
    
    // 创建带熔断器的客户端
    clientWithBreaker := genkit.NewClientWithCircuitBreaker(baseClient, nil)
    
    // 使用客户端（自动带熔断保护）
    result, err := clientWithBreaker.Generate(ctx, "Hello, AI!", nil)
    if err != nil {
        // 处理错误（包括熔断器打开的情况）
        handleError(err)
        return
    }
    
    // 处理结果
    processResult(result)
}
```

### 3. 使用熔断器管理器

```go
package main

import (
    "context"
    
    "genkit-ai-service/internal/middleware"
)

func main() {
    // 创建熔断器管理器
    manager := middleware.NewCircuitBreakerManager()
    
    // 为不同的服务创建熔断器
    aiServiceBreaker := manager.GetOrCreate("ai-service", nil)
    vectorServiceBreaker := manager.GetOrCreate("vector-service", nil)
    
    // 使用熔断器
    ctx := context.Background()
    
    // AI 服务调用
    aiResult, err := aiServiceBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
        return callAIService()
    })
    
    // 向量服务调用
    vectorResult, err := vectorServiceBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
        return callVectorService()
    })
    
    // 获取所有熔断器的统计信息
    stats := manager.GetAllStats()
    for _, stat := range stats {
        fmt.Printf("熔断器: %s, 状态: %s, 失败次数: %d\n", 
            stat.Name, stat.State, stat.FailureCount)
    }
}
```

### 4. 状态变化监控

```go
config := &middleware.CircuitBreakerConfig{
    MaxFailures:         5,
    Timeout:             30 * time.Second,
    HalfOpenMaxRequests: 3,
    SuccessThreshold:    2,
    OnStateChange: func(from, to middleware.CircuitState) {
        // 记录状态变化
        logger.Info("熔断器状态变化", logger.Fields{
            "from": from.String(),
            "to":   to.String(),
        })
        
        // 发送告警
        switch to {
        case middleware.StateOpen:
            alertService.SendAlert("熔断器已打开")
        case middleware.StateClosed:
            alertService.SendAlert("熔断器已关闭，服务恢复")
        case middleware.StateHalfOpen:
            alertService.SendAlert("熔断器进入半开状态")
        }
    },
}
```

## 监控和管理

### 获取统计信息

```go
// 获取单个熔断器的统计信息
stats := circuitBreaker.GetStats()
fmt.Printf("名称: %s\n", stats.Name)
fmt.Printf("状态: %s\n", stats.State)
fmt.Printf("失败次数: %d\n", stats.FailureCount)
fmt.Printf("成功次数: %d\n", stats.SuccessCount)
fmt.Printf("最后失败时间: %s\n", stats.LastFailureTime)
```

### 手动重置熔断器

```go
// 重置熔断器（管理员操作）
circuitBreaker.Reset()
```

### 检查熔断器状态

```go
state := circuitBreaker.GetState()
switch state {
case middleware.StateClosed:
    fmt.Println("熔断器关闭，服务正常")
case middleware.StateOpen:
    fmt.Println("熔断器打开，服务不可用")
case middleware.StateHalfOpen:
    fmt.Println("熔断器半开，正在测试服务")
}
```

## 最佳实践

### 1. 合理配置参数

- **MaxFailures**: 根据服务的稳定性设置，通常 3-10 次
- **Timeout**: 根据服务恢复时间设置，通常 10-60 秒
- **HalfOpenMaxRequests**: 通常 1-5 个请求
- **SuccessThreshold**: 通常 1-3 次成功

### 2. 结合降级策略

```go
result, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
    return callExternalService()
})

if err != nil {
    if err == middleware.ErrCircuitBreakerOpen {
        // 熔断器打开，执行降级逻辑
        return degradationService.GetCachedResponse()
    }
    return err
}
```

### 3. 监控告警

- 监控熔断器状态变化
- 记录失败次数和失败率
- 设置告警阈值
- 定期检查熔断器健康状态

### 4. 日志记录

```go
config := &middleware.CircuitBreakerConfig{
    OnStateChange: func(from, to middleware.CircuitState) {
        logger.Warn("熔断器状态变化", logger.Fields{
            "from":      from.String(),
            "to":        to.String(),
            "timestamp": time.Now(),
        })
    },
}
```

## 常见问题

### Q1: 熔断器打开后如何恢复？

A: 熔断器会在配置的 Timeout 时间后自动进入半开状态，允许少量请求通过。如果这些请求成功，熔断器会自动关闭。

### Q2: 如何手动重置熔断器？

A: 调用 `circuitBreaker.Reset()` 方法可以手动重置熔断器到关闭状态。

### Q3: 熔断器适用于哪些场景？

A: 熔断器适用于调用外部服务的场景，如：

- AI 服务调用
- 数据库查询
- 第三方 API 调用
- 微服务间调用

### Q4: 如何选择合适的配置参数？

A: 根据服务特性选择：

- 稳定的服务：较高的 MaxFailures（如 10）
- 不稳定的服务：较低的 MaxFailures（如 3）
- 快速恢复的服务：较短的 Timeout（如 10s）
- 慢速恢复的服务：较长的 Timeout（如 60s）

## 性能影响

熔断器的性能开销非常小：

- 状态检查：O(1) 时间复杂度
- 内存占用：每个熔断器约 100 字节
- 并发安全：使用读写锁，读操作不阻塞

## 相关文档

- [降级服务文档](../service/DEGRADATION_SERVICE_README.md)
- [错误处理文档](../model/ERRORS_USAGE.md)
- [监控指标文档](../monitoring/METRICS_USAGE.md)
