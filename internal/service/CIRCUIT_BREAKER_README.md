# 熔断器实现文档

## 概述

熔断器（Circuit Breaker）是一种用于保护系统免受级联故障影响的设计模式。当外部服务频繁失败时，熔断器会自动"打开"，阻止进一步的请求，从而避免资源浪费和系统过载。

本实现提供了完整的熔断器功能，包括三种状态管理、自动恢复机制和与降级策略的集成。

## 核心特性

### 1. 三种状态管理

熔断器实现了标准的三状态模型：

- **Closed（关闭）**：正常状态，所有请求都会被执行
- **Open（打开）**：熔断状态，所有请求都会被拒绝
- **HalfOpen（半开）**：测试状态，允许部分请求通过以测试服务是否恢复

### 2. 状态转换规则

```
Closed --[失败次数达到阈值]--> Open
Open --[超时后]--> HalfOpen
HalfOpen --[连续成功达到阈值]--> Closed
HalfOpen --[任何失败]--> Open
```

### 3. 可配置参数

- `MaxFailures`: 触发熔断的最大失败次数
- `Timeout`: 熔断器打开后的超时时间
- `HalfOpenMaxRequests`: 半开状态下允许的最大请求数
- `HalfOpenSuccessThreshold`: 半开状态下关闭熔断器所需的连续成功次数
- `ResetTimeout`: 关闭状态下重置失败计数的超时时间

## 使用方法

### 基本使用

```go
package main

import (
    "context"
    "time"
    "genkit-ai-service/internal/service"
    "genkit-ai-service/internal/logger"
)

func main() {
    log := logger.NewLogger()
    
    // 创建熔断器配置
    config := service.CircuitBreakerConfig{
        MaxFailures:              5,
        Timeout:                  30 * time.Second,
        HalfOpenMaxRequests:      3,
        HalfOpenSuccessThreshold: 2,
        ResetTimeout:             60 * time.Second,
    }
    
    // 创建熔断器
    cb := service.NewCircuitBreaker("my-service", config, log)
    
    // 使用熔断器执行函数
    ctx := context.Background()
    err := cb.Execute(ctx, func() error {
        // 调用外部服务
        return callExternalService()
    })
    
    if err != nil {
        // 处理错误（可能是服务错误或熔断器错误）
        log.ErrorContext(ctx, "服务调用失败", logger.Fields{
            "error": err.Error(),
        })
    }
}
```

### 使用熔断器管理器

```go
// 创建熔断器管理器
manager := service.NewCircuitBreakerManager(log)

// 获取或创建熔断器
cb := manager.GetOrCreate("ai-service", config)

// 执行受保护的调用
err := cb.Execute(ctx, func() error {
    return aiService.Generate(ctx, sessionID, query)
})
```

### 集成降级策略

```go
// 创建受保护的 AI 服务
protectedAI := service.NewProtectedAIService(
    aiService,
    cbManager,
    degradationService,
    log,
)

// 调用服务（自动处理熔断和降级）
response, err := protectedAI.Generate(ctx, sessionID, userQuery)
```

### 使用中间件模式

```go
// 创建熔断器中间件
middleware := service.NewCircuitBreakerMiddleware(manager, log)

// 包装函数调用
err := middleware.Wrap(ctx, "my-service", config, func() error {
    return doSomething()
})

// 包装函数调用并提供降级
err := middleware.WrapWithFallback(
    ctx,
    "my-service",
    config,
    func() error {
        return primaryService()
    },
    func() error {
        return fallbackService()
    },
)
```

## 配置建议

### AI 服务配置

```go
config := service.CircuitBreakerConfig{
    MaxFailures:              5,  // AI 服务可能偶尔失败
    Timeout:                  30 * time.Second,  // 给予足够的恢复时间
    HalfOpenMaxRequests:      3,  // 谨慎测试恢复
    HalfOpenSuccessThreshold: 2,  // 需要多次成功才能确认恢复
    ResetTimeout:             60 * time.Second,  // 较长的重置时间
}
```

### 向量服务配置

```go
config := service.CircuitBreakerConfig{
    MaxFailures:              3,  // 向量服务应该更稳定
    Timeout:                  20 * time.Second,  // 较短的恢复时间
    HalfOpenMaxRequests:      2,  // 快速测试
    HalfOpenSuccessThreshold: 2,  // 确认恢复
    ResetTimeout:             45 * time.Second,
}
```

### 数据库服务配置

```go
config := service.CircuitBreakerConfig{
    MaxFailures:              10,  // 数据库应该非常稳定
    Timeout:                  10 * time.Second,  // 快速恢复
    HalfOpenMaxRequests:      5,  // 允许更多测试请求
    HalfOpenSuccessThreshold: 3,  // 需要更多成功确认
    ResetTimeout:             30 * time.Second,
}
```

## 监控和管理

### 获取熔断器状态

```go
// 获取当前状态
state := cb.GetState()
fmt.Printf("当前状态: %s\n", state.String())

// 获取统计信息
stats := cb.GetStats()
fmt.Printf("失败次数: %d\n", stats.FailureCount)
fmt.Printf("成功次数: %d\n", stats.SuccessCount)
fmt.Printf("最后失败时间: %s\n", stats.LastFailureTime)
```

### 获取所有熔断器状态

```go
// 获取所有熔断器的统计信息
allStats := manager.GetAllStats()
for _, stats := range allStats {
    fmt.Printf("熔断器: %s, 状态: %s, 失败次数: %d\n",
        stats.Name, stats.State, stats.FailureCount)
}
```

### 手动重置熔断器

```go
// 重置单个熔断器
cb.Reset(ctx)

// 重置所有熔断器
manager.ResetAll(ctx)
```

## 与降级策略集成

熔断器与降级服务紧密集成，当熔断器打开时，自动执行降级策略：

```go
// AI 服务降级流程
executeErr := circuitBreaker.Execute(ctx, func() error {
    response, err = aiService.Generate(ctx, sessionID, userQuery)
    return err
})

if executeErr != nil {
    // 熔断器打开或服务失败，执行降级
    degradationResult, _ := degradation.DegradeAIService(ctx, sessionID, userQuery)
    return degradationResult.Response, nil
}
```

## 最佳实践

### 1. 为不同服务使用不同的熔断器

```go
// 不要这样做
cb := manager.GetOrCreate("all-services", config)

// 应该这样做
aiCB := manager.GetOrCreate("ai-service", aiConfig)
vectorCB := manager.GetOrCreate("vector-service", vectorConfig)
dbCB := manager.GetOrCreate("database", dbConfig)
```

### 2. 根据服务特性调整配置

- 稳定的服务：更高的 `MaxFailures`，更短的 `Timeout`
- 不稳定的服务：更低的 `MaxFailures`，更长的 `Timeout`
- 关键服务：更谨慎的半开状态配置

### 3. 记录熔断器状态变化

熔断器会自动记录所有状态变化，确保日志级别正确配置：

```go
log.InfoContext(ctx, "熔断器状态变更", logger.Fields{
    "circuit_breaker": name,
    "old_state":       oldState.String(),
    "new_state":       newState.String(),
})
```

### 4. 监控熔断器指标

定期检查熔断器统计信息，及时发现问题：

```go
// 定期检查
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        stats := manager.GetAllStats()
        for _, stat := range stats {
            if stat.State == "Open" {
                // 发送告警
                alertService.Send("熔断器打开", stat)
            }
        }
    }
}()
```

### 5. 提供手动控制接口

在管理 API 中提供熔断器控制接口：

```go
// GET /api/v1/admin/circuit-breakers
func ListCircuitBreakers(c *gin.Context) {
    stats := manager.GetAllStats()
    c.JSON(200, stats)
}

// POST /api/v1/admin/circuit-breakers/:name/reset
func ResetCircuitBreaker(c *gin.Context) {
    name := c.Param("name")
    if cb, exists := manager.Get(name); exists {
        cb.Reset(c.Request.Context())
        c.JSON(200, gin.H{"message": "熔断器已重置"})
    } else {
        c.JSON(404, gin.H{"error": "熔断器不存在"})
    }
}
```

## 测试

### 单元测试

```go
func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
    log := logger.NewLogger()
    config := CircuitBreakerConfig{
        MaxFailures: 3,
        Timeout:     1 * time.Second,
    }
    cb := NewCircuitBreaker("test", config, log)
    
    // 执行失败的请求
    for i := 0; i < 3; i++ {
        cb.Execute(context.Background(), func() error {
            return errors.New("error")
        })
    }
    
    // 验证熔断器已打开
    assert.Equal(t, StateOpen, cb.GetState())
}
```

### 集成测试

```go
func TestProtectedService_WithCircuitBreaker(t *testing.T) {
    // 设置测试环境
    manager := NewCircuitBreakerManager(log)
    protectedService := NewProtectedAIService(
        mockAIService,
        manager,
        mockDegradation,
        log,
    )
    
    // 模拟服务失败
    mockAIService.SetFailure(true)
    
    // 多次调用触发熔断
    for i := 0; i < 5; i++ {
        protectedService.Generate(ctx, sessionID, query)
    }
    
    // 验证熔断器已打开
    cb, _ := manager.Get("ai-service")
    assert.Equal(t, StateOpen, cb.GetState())
    
    // 验证降级策略被调用
    assert.True(t, mockDegradation.WasCalled())
}
```

## 性能考虑

### 1. 锁竞争

熔断器使用读写锁来保护状态，在高并发场景下可能存在锁竞争。建议：

- 为不同的服务使用不同的熔断器实例
- 避免在熔断器内部执行耗时操作

### 2. 内存使用

每个熔断器实例占用的内存很小（< 1KB），可以安全地创建多个实例。

### 3. 性能开销

熔断器的性能开销非常小：

- 关闭状态：几乎无开销（仅读锁）
- 打开状态：立即返回错误（仅读锁）
- 半开状态：轻微开销（写锁）

## 故障排查

### 问题：熔断器频繁打开

**可能原因**：

- `MaxFailures` 设置过低
- 外部服务确实不稳定
- 超时时间设置不合理

**解决方案**：

- 增加 `MaxFailures` 值
- 检查外部服务状态
- 调整超时配置

### 问题：熔断器无法恢复

**可能原因**：

- `Timeout` 设置过长
- 半开状态配置过于严格
- 服务仍然不可用

**解决方案**：

- 减少 `Timeout` 值
- 降低 `HalfOpenSuccessThreshold`
- 检查服务恢复情况
- 考虑手动重置

### 问题：降级策略未执行

**可能原因**：

- 未正确集成降级服务
- 降级服务本身失败

**解决方案**：

- 检查降级服务配置
- 添加降级服务的错误处理
- 提供多级降级策略

## 相关文档

- [降级服务文档](./DEGRADATION_SERVICE_README.md)
- [错误处理文档](../genkit/flows/ERROR_HANDLING_README.md)
- [监控指标文档](../monitoring/README.md)

## 更新日志

### v1.0.0 (2025-01-01)

- 初始实现
- 支持三种状态管理
- 集成降级策略
- 提供熔断器管理器
- 完整的测试覆盖
