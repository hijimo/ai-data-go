# TraceID 全链路追踪设计文档

## 概述

本设计实现轻量级的全链路追踪功能，通过 TraceID 关联一次请求的所有日志和操作。采用最小化改动原则，确保高性能、向后兼容，并为未来升级到 OpenTelemetry 预留扩展空间。

### 设计目标

- **轻量级**：最小化系统改动，避免引入重型追踪框架
- **高性能**：追踪功能的性能开销可忽略不计（< 1ms）
- **向后兼容**：不破坏现有 API 和功能
- **易用性**：开发人员无需大量代码改动即可使用
- **可扩展**：为未来升级到 OpenTelemetry 预留空间

### 核心原则

1. **Context 传递**：使用 Go Context 在整个调用链中传递 TraceID
2. **中间件注入**：在 HTTP 中间件层统一处理 TraceID 生成和注入
3. **自动化集成**：日志和响应自动包含 TraceID，无需手动处理
4. **优雅降级**：TraceID 缺失时系统继续正常工作

## 架构设计

### 整体架构

```
客户端请求
    ↓
[Logger中间件] ← 生成/提取 TraceID
    ↓
[Context注入] ← 将 TraceID 注入到 Context
    ↓
[Handler层] ← 传递 Context 到服务层
    ↓
[Service层] ← 使用 Context 记录日志
    ↓
[响应构建] ← 从 Context 提取 TraceID 注入响应
    ↓
客户端响应（包含 TraceID）
```

### 数据流

1. **请求入口**：Logger中间件检查 `X-Trace-ID` 请求头
2. **TraceID 生成**：如果不存在则生成新的 TraceID
3. **Context 注入**：将 TraceID 存入 Context
4. **日志记录**：日志系统自动从 Context 提取 TraceID
5. **响应返回**：响应头和响应体都包含 TraceID

## 组件设计

### 1. TraceID 生成器

**位置**：`internal/middleware/logger.go`

**功能**：生成唯一的 TraceID

**格式**：`trace-{timestamp}-{random}`

- `timestamp`：Unix 时间戳（秒）
- `random`：6位随机字符串（a-z0-9）

**实现示例**：

```go
func generateTraceID() string {
    timestamp := time.Now().Unix()
    random := generateRandomString(6)
    return fmt.Sprintf("trace-%d-%s", timestamp, random)
}
```

**设计决策**：

- 使用可识别的前缀 "trace-" 便于日志过滤和未来扩展
- 包含时间戳便于时间范围查询
- 随机字符串确保唯一性
- 总长度约 20 字符，性能和可读性平衡

### 2. Context 键定义

**位置**：`internal/middleware/context.go`（新建）

**功能**：定义 Context 中存储 TraceID 的键

**实现**：

```go
package middleware

type contextKey string

const (
    TraceIDKey contextKey = "traceId"
)

// SetTraceID 将 TraceID 注入到 Context
func SetTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 Context 提取 TraceID
func GetTraceID(ctx context.Context) string {
    if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
        return traceID
    }
    return ""
}
```

**设计决策**：

- 使用自定义类型 `contextKey` 避免键冲突
- 提供封装函数而不是直接暴露键，便于未来修改
- GetTraceID 返回空字符串而不是 error，简化调用方代码

### 3. Logger 中间件增强

**位置**：`internal/middleware/logger.go`

**功能**：处理 TraceID 的生成、提取和注入

**修改点**：

1. 检查请求头 `X-Trace-ID`
2. 生成或使用现有 TraceID
3. 注入到 Context
4. 设置响应头 `X-Trace-ID`

**实现流程**：

```go
func LoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取或生成 TraceID
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = generateTraceID()
        }
        
        // 2. 注入到 Context
        ctx := SetTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        // 3. 设置响应头
        c.Header("X-Trace-ID", traceID)
        
        // 4. 记录请求日志（包含 TraceID）
        logger.InfoContext(ctx, "HTTP请求", ...)
        
        // 5. 处理请求
        c.Next()
        
        // 6. 记录响应日志（包含 TraceID）
        logger.InfoContext(ctx, "HTTP响应", ...)
    }
}
```

**设计决策**：

- 在中间件层统一处理，避免在每个 Handler 中重复代码
- 优先使用客户端提供的 TraceID，支持跨服务追踪
- 同时设置响应头和响应体，提供多种获取方式

### 4. 日志系统集成

**位置**：`pkg/logger/logger.go`

**功能**：自动从 Context 提取 TraceID 并添加到日志字段

**修改点**：

1. 在日志记录函数中添加 TraceID 提取逻辑
2. 将 TraceID 作为结构化字段输出

**实现示例**：

```go
func InfoContext(ctx context.Context, msg string, args ...any) {
    fields := extractContextFields(ctx)
    logger.With(fields...).Info(msg, args...)
}

func extractContextFields(ctx context.Context) []any {
    fields := make([]any, 0, 2)
    
    // 提取 TraceID
    if traceID := middleware.GetTraceID(ctx); traceID != "" {
        fields = append(fields, "traceId", traceID)
    }
    
    return fields
}
```

**设计决策**：

- 使用 `*Context` 系列函数（InfoContext、ErrorContext 等）
- TraceID 缺失时不报错，保持日志系统稳定性
- 使用结构化日志字段，便于日志查询和分析
- 所有日志函数统一调用 `extractContextFields`，确保一致性

### 5. 响应结构更新

**位置**：`internal/handler/response.go`

**功能**：在所有响应结构中添加 TraceID 字段

**修改内容**：

```go
// ResponseData 通用响应数据结构
type ResponseData[T any] struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    TraceID string `json:"traceId,omitempty"`
    Data    *T     `json:"data,omitempty"`
}

// PaginationData 分页数据结构
type PaginationData[T any] struct {
    Data       T   `json:"data"`
    PageNo     int `json:"pageNo"`
    PageSize   int `json:"pageSize"`
    TotalCount int `json:"totalCount"`
    TotalPage  int `json:"totalPage"`
}

// ResponsePaginationData 分页响应数据结构
type ResponsePaginationData[T any] struct {
    Code    int               `json:"code"`
    Message string            `json:"message"`
    TraceID string            `json:"traceId,omitempty"`
    Data    PaginationData[T] `json:"data"`
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    TraceID string `json:"traceId,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

**设计决策**：

- 使用 `omitempty` 标签，TraceID 为空时不输出该字段
- 在所有响应类型中统一添加 TraceID 字段，保持一致性
- 字段名使用 `traceId`（驼峰命名），符合 JSON 命名规范

### 6. 响应工具函数增强

**位置**：`internal/handler/response.go`

**功能**：提供便捷的响应构建函数，自动注入 TraceID

**新增函数**：

```go
// SuccessWithContext 返回成功响应（带 Context）
func SuccessWithContext[T any](ctx context.Context, data *T) ResponseData[T] {
    return ResponseData[T]{
        Code:    http.StatusOK,
        Message: "success",
        TraceID: middleware.GetTraceID(ctx),
        Data:    data,
    }
}

// SuccessWithMessageContext 返回带自定义消息的成功响应（带 Context）
func SuccessWithMessageContext[T any](ctx context.Context, message string, data *T) ResponseData[T] {
    return ResponseData[T]{
        Code:    http.StatusOK,
        Message: message,
        TraceID: middleware.GetTraceID(ctx),
        Data:    data,
    }
}

// PaginationSuccessContext 返回分页成功响应（带 Context）
func PaginationSuccessContext[T any](ctx context.Context, data T, pageNo, pageSize, totalCount int) ResponsePaginationData[T] {
    totalPage := (totalCount + pageSize - 1) / pageSize
    return ResponsePaginationData[T]{
        Code:    http.StatusOK,
        Message: "success",
        TraceID: middleware.GetTraceID(ctx),
        Data: PaginationData[T]{
            Data:       data,
            PageNo:     pageNo,
            PageSize:   pageSize,
            TotalCount: totalCount,
            TotalPage:  totalPage,
        },
    }
}

// ErrorResponseContext 返回错误响应（带 Context）
func ErrorResponseContext(ctx context.Context, code int, message string, err error) ErrorResponse {
    resp := ErrorResponse{
        Code:    code,
        Message: message,
        TraceID: middleware.GetTraceID(ctx),
    }
    if err != nil {
        resp.Error = err.Error()
    }
    return resp
}
```

**向后兼容**：保留原有函数，TraceID 字段为空

```go
// Success 返回成功响应（向后兼容）
func Success[T any](data *T) ResponseData[T] {
    return ResponseData[T]{
        Code:    http.StatusOK,
        Message: "success",
        Data:    data,
    }
}
```

**设计决策**：

- 新增 `*Context` 系列函数，接受 Context 参数
- 保留原有函数签名，确保向后兼容
- 自动从 Context 提取 TraceID，Handler 无需手动处理
- 统一的函数命名规范（后缀 `Context`）

## 数据模型

### Context 数据结构

```go
// Context 中存储的数据
type ContextData struct {
    TraceID string  // 追踪ID
}

// Context 键类型
type contextKey string

const (
    TraceIDKey contextKey = "traceId"
)
```

### TraceID 格式规范

**格式**：`trace-{timestamp}-{random}`

**示例**：`trace-1704067200-a3f9k2`

**字段说明**：

- `trace-`：固定前缀，标识这是一个追踪ID
- `{timestamp}`：Unix 时间戳（秒），10位数字
- `{random}`：6位随机字符串，字符集 [a-z0-9]

**长度**：约 20-22 字符

**唯一性保证**：

- 时间戳确保时间维度唯一
- 随机字符串确保同一秒内的唯一性
- 组合概率碰撞率 < 1/10^9

## 接口设计

### HTTP 请求头

**请求头**：

- `X-Trace-ID`（可选）：客户端提供的 TraceID

**响应头**：

- `X-Trace-ID`（必需）：服务端返回的 TraceID

### API 响应格式

**普通响应**：

```json
{
  "code": 200,
  "message": "success",
  "traceId": "trace-1704067200-a3f9k2",
  "data": {
    "id": "123",
    "name": "example"
  }
}
```

**分页响应**：

```json
{
  "code": 200,
  "message": "success",
  "traceId": "trace-1704067200-a3f9k2",
  "data": {
    "data": [...],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 100,
    "totalPage": 10
  }
}
```

**错误响应**：

```json
{
  "code": 400,
  "message": "请求参数错误",
  "traceId": "trace-1704067200-a3f9k2",
  "error": "invalid email format"
}
```

## 错误处理

### 错误场景与处理策略

| 场景 | 处理策略 | 影响 |
|------|---------|------|
| Context 中无 TraceID | 返回空字符串 | 日志和响应中 TraceID 为空，不影响业务 |
| TraceID 生成失败 | 使用降级方案（简化格式） | 确保系统继续运行 |
| 客户端提供无效 TraceID | 使用客户端提供的值 | 保持追踪链路完整性 |
| 日志系统提取 TraceID 失败 | 跳过 TraceID 字段 | 日志正常记录，仅缺少 TraceID |
| 响应构建时 Context 为 nil | TraceID 字段为空 | 响应正常返回 |

### 降级方案

**TraceID 生成降级**：

```go
func generateTraceID() string {
    defer func() {
        if r := recover(); r != nil {
            // 降级方案：使用简化格式
            return fmt.Sprintf("trace-fallback-%d", time.Now().UnixNano())
        }
    }()
    
    // 正常生成逻辑
    timestamp := time.Now().Unix()
    random := generateRandomString(6)
    return fmt.Sprintf("trace-%d-%s", timestamp, random)
}
```

**设计原则**：

- **优雅降级**：任何错误都不应导致请求失败
- **非侵入式**：TraceID 功能异常不影响业务逻辑
- **可观测性**：降级事件应记录到监控系统

## 性能优化

### 性能目标

| 操作 | 目标延迟 | 内存开销 |
|------|---------|---------|
| TraceID 生成 | < 1ms | 24 bytes |
| Context 注入 | < 0.1ms | 指针传递，忽略不计 |
| 日志字段添加 | < 0.1ms | 24 bytes |
| 响应字段赋值 | < 0.1ms | 24 bytes |
| 总体开销（1000 QPS） | < 1ms/req | < 100KB/s |

### 优化策略

**1. TraceID 生成优化**

```go
// 使用对象池减少内存分配
var stringBuilderPool = sync.Pool{
    New: func() interface{} {
        return &strings.Builder{}
    },
}

func generateTraceID() string {
    sb := stringBuilderPool.Get().(*strings.Builder)
    defer func() {
        sb.Reset()
        stringBuilderPool.Put(sb)
    }()
    
    sb.WriteString("trace-")
    sb.WriteString(strconv.FormatInt(time.Now().Unix(), 10))
    sb.WriteByte('-')
    sb.WriteString(generateRandomString(6))
    
    return sb.String()
}
```

**2. Context 传递优化**

- 使用指针传递 Context，避免值拷贝
- TraceID 字符串在 Context 中只存储一次

**3. 日志系统优化**

- 预分配字段切片容量
- 避免不必要的字符串拼接

**4. 响应构建优化**

- 使用结构体字面量直接赋值
- 避免中间变量

## 测试策略

### 单元测试

**1. TraceID 生成测试**

- 测试 TraceID 格式正确性
- 测试唯一性（并发生成 10000 个 TraceID）
- 测试性能（生成 10000 个 TraceID 的耗时）

**2. Context 操作测试**

- 测试 SetTraceID 和 GetTraceID
- 测试 Context 为 nil 的情况
- 测试 TraceID 不存在的情况

**3. 中间件测试**

- 测试客户端提供 TraceID 的情况
- 测试客户端未提供 TraceID 的情况
- 测试响应头设置
- 测试 Context 注入

**4. 日志系统测试**

- 测试日志包含 TraceID
- 测试 Context 中无 TraceID 的情况
- 测试结构化日志输出格式

**5. 响应工具函数测试**

- 测试各种响应函数的 TraceID 注入
- 测试向后兼容的函数
- 测试 Context 为 nil 的情况

### 集成测试

**1. 端到端追踪测试**

```go
func TestEndToEndTracing(t *testing.T) {
    // 1. 发送带 X-Trace-ID 的请求
    req := httptest.NewRequest("GET", "/api/v1/users", nil)
    req.Header.Set("X-Trace-ID", "trace-test-123")
    
    // 2. 执行请求
    resp := executeRequest(req)
    
    // 3. 验证响应头包含 TraceID
    assert.Equal(t, "trace-test-123", resp.Header.Get("X-Trace-ID"))
    
    // 4. 验证响应体包含 TraceID
    var body ResponseData[any]
    json.Unmarshal(resp.Body.Bytes(), &body)
    assert.Equal(t, "trace-test-123", body.TraceID)
    
    // 5. 验证日志包含 TraceID
    logs := getLogEntries()
    assert.Contains(t, logs[0], "traceId=trace-test-123")
}
```

**2. 性能测试**

```go
func BenchmarkTraceIDGeneration(b *testing.B) {
    for i := 0; i < b.N; i++ {
        generateTraceID()
    }
}

func BenchmarkRequestWithTracing(b *testing.B) {
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest("GET", "/api/v1/health", nil)
        executeRequest(req)
    }
}
```

**3. 并发测试**

```go
func TestConcurrentTraceIDGeneration(t *testing.T) {
    const goroutines = 100
    const iterations = 100
    
    traceIDs := make(map[string]bool)
    mu := sync.Mutex{}
    wg := sync.WaitGroup{}
    
    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < iterations; j++ {
                traceID := generateTraceID()
                mu.Lock()
                assert.False(t, traceIDs[traceID], "重复的 TraceID")
                traceIDs[traceID] = true
                mu.Unlock()
            }
        }()
    }
    
    wg.Wait()
    assert.Equal(t, goroutines*iterations, len(traceIDs))
}
```

### 测试覆盖率目标

- 单元测试覆盖率：> 90%
- 集成测试覆盖率：> 80%
- 关键路径覆盖率：100%

## 向后兼容性

### 兼容性保证

**1. API 接口兼容**

- 所有现有 API 接口继续正常工作
- 响应结构新增 `traceId` 字段，使用 `omitempty` 标签
- 旧版本客户端可以忽略新增字段

**2. 函数签名兼容**

- 保留所有原有响应工具函数
- 新增 `*Context` 系列函数，不影响现有代码
- Handler 可以渐进式迁移到新函数

**3. 日志系统兼容**

- 新增 `*Context` 系列日志函数
- 保留原有日志函数，继续可用
- 旧代码的日志不包含 TraceID，但不影响功能

### 迁移策略

**阶段 1：基础设施部署**

- 部署中间件和 Context 工具函数
- 部署响应结构更新
- 部署日志系统更新
- **影响**：所有请求自动获得 TraceID，但 Handler 暂未使用

**阶段 2：渐进式迁移**

- 逐步更新 Handler 使用 `*Context` 响应函数
- 优先迁移核心业务接口
- 低优先级接口可以延后迁移
- **影响**：部分接口响应包含 TraceID

**阶段 3：全面覆盖**

- 完成所有 Handler 的迁移
- 更新所有日志调用为 `*Context` 版本
- **影响**：所有接口和日志都包含 TraceID

**阶段 4：清理优化**

- 评估是否废弃旧版本函数
- 优化性能和代码结构
- **影响**：代码更简洁，性能更优

### 回滚方案

如果出现问题，可以快速回滚：

**1. 中间件回滚**

- 注释掉 TraceID 生成和注入逻辑
- 保留中间件其他功能

**2. 响应结构回滚**

- TraceID 字段使用 `omitempty`，为空时不影响客户端
- 可以直接回滚代码版本

**3. 日志系统回滚**

- 移除 TraceID 提取逻辑
- 日志系统恢复原状

## 可扩展性设计

### 未来扩展方向

**1. 升级到 OpenTelemetry**

当前设计为升级预留空间：

```go
// 当前实现
type contextKey string
const TraceIDKey contextKey = "traceId"

// 未来可以扩展为
type contextKey string
const (
    TraceIDKey contextKey = "traceId"
    SpanIDKey  contextKey = "spanId"
    ParentIDKey contextKey = "parentId"
)

// 或直接使用 OpenTelemetry 的 Context
import "go.opentelemetry.io/otel/trace"
```

**2. 分布式追踪**

支持跨服务追踪：

- 当前 TraceID 格式已支持跨服务传递
- 未来可以添加 SpanID 和 ParentID
- 可以集成 Jaeger 或 Zipkin

**3. 采样策略**

当前全量追踪，未来可以添加采样：

```go
// 采样配置
type SamplingConfig struct {
    SampleRate float64  // 采样率 0.0-1.0
    AlwaysSample []string  // 总是采样的路径
}

// 在中间件中判断是否采样
if shouldSample(c.Request.URL.Path, config) {
    traceID = generateTraceID()
    // ... 追踪逻辑
}
```

**4. 追踪数据存储**

未来可以将追踪数据存储到专门的系统：

- 时序数据库（InfluxDB、TimescaleDB）
- 分布式追踪系统（Jaeger、Zipkin）
- 日志聚合系统（ELK、Loki）

### 扩展接口设计

**预留扩展点**：

```go
// Tracer 接口（未来可以实现）
type Tracer interface {
    StartSpan(ctx context.Context, name string) (context.Context, Span)
    InjectTraceID(ctx context.Context, traceID string) context.Context
    ExtractTraceID(ctx context.Context) string
}

// Span 接口（未来可以实现）
type Span interface {
    SetAttribute(key string, value interface{})
    SetStatus(code int, message string)
    End()
}
```

当前实现可以视为 `Tracer` 接口的简化版本。
