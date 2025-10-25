# TraceID Context 管理工具

## 概述

本模块提供了 TraceID 生成和 Context 管理功能，用于实现全链路追踪。

## 功能特性

### 1. TraceID 生成

- **格式**: `trace-{timestamp}-{nanoHex}{random}`
- **示例**: `trace-1704067200-a3f9k2b8c1d4`
- **唯一性保证**:
  - 10位 Unix 时间戳（秒）
  - 6位纳秒时间戳的十六进制表示
  - 6位加密随机字符串
- **性能**: < 0.2 微秒/次，远超 1 毫秒的要求

### 2. Context 操作

#### SetTraceID

将 TraceID 注入到 Context 中。

```go
ctx := middleware.SetTraceID(context.Background(), traceID)
```

#### GetTraceID

从 Context 中提取 TraceID，如果不存在则返回空字符串。

```go
traceID := middleware.GetTraceID(ctx)
```

## 性能优化

### 对象池

使用 `sync.Pool` 优化内存分配：

- `stringBuilderPool`: 复用 strings.Builder
- `randomBytesPool`: 复用随机字节切片

### 性能指标

根据基准测试结果：

| 操作 | 耗时 | 内存分配 |
|------|------|---------|
| GenerateTraceID | ~188 ns | 96 B |
| SetTraceID | ~15 ns | 48 B |
| GetTraceID | ~3.6 ns | 0 B |

所有操作的性能都远超设计要求。

## 使用示例

### 基本使用

```go
import "genkit-ai-service/internal/api/middleware"

// 生成 TraceID
traceID := middleware.GenerateTraceID()

// 注入到 Context
ctx := middleware.SetTraceID(context.Background(), traceID)

// 提取 TraceID
retrievedID := middleware.GetTraceID(ctx)
```

### 在 HTTP 中间件中使用

```go
func LoggerMiddleware() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 检查请求头或生成新的 TraceID
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = middleware.GenerateTraceID()
        }
        
        // 注入到 Context
        ctx := middleware.SetTraceID(r.Context(), traceID)
        r = r.WithContext(ctx)
        
        // 设置响应头
        w.Header().Set("X-Trace-ID", traceID)
        
        // 继续处理请求
        next.ServeHTTP(w, r)
    }
}
```

### 在业务逻辑中使用

```go
func ProcessRequest(ctx context.Context) error {
    // 提取 TraceID 用于日志记录
    traceID := middleware.GetTraceID(ctx)
    logger.InfoContext(ctx, "处理请求", logger.Fields{
        "traceId": traceID,
        "action": "process",
    })
    
    // 业务逻辑...
    return nil
}
```

## 设计决策

### 为什么使用纳秒时间戳？

在高并发场景下，同一秒内可能生成大量 TraceID。纳秒时间戳提供了额外的时间维度唯一性。

### 为什么使用 crypto/rand？

`crypto/rand` 提供加密级别的随机数，确保 TraceID 的不可预测性和唯一性。

### 为什么使用对象池？

对象池减少了内存分配和 GC 压力，在高并发场景下显著提升性能。

### 为什么 GetTraceID 返回空字符串而不是 error？

简化调用方代码，避免不必要的错误处理。TraceID 缺失不应该导致业务逻辑失败。

## 测试覆盖

- ✅ TraceID 格式验证
- ✅ TraceID 唯一性测试（10000 个并发生成）
- ✅ Context 注入和提取
- ✅ 边界情况（nil Context、空 TraceID）
- ✅ 性能基准测试

## 未来扩展

当前实现为升级到 OpenTelemetry 预留了扩展空间：

- TraceID 格式使用可识别的前缀 `trace-`
- Context 键使用标准化命名
- 可以轻松添加 SpanID 和 ParentID 支持

## 相关文档

- [设计文档](../../../.kiro/specs/trace-id-implementation/design.md)
- [需求文档](../../../.kiro/specs/trace-id-implementation/requirements.md)
- [任务列表](../../../.kiro/specs/trace-id-implementation/tasks.md)
