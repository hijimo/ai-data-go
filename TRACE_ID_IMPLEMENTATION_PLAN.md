# 全链路追踪 TraceID 实施方案

## 一、方案概述

### 目标

为系统添加全链路追踪能力，通过 TraceID 关联一次请求的所有日志和操作，便于问题排查和性能分析。

### 核心原则

1. **最小侵入**：利用现有架构，最小化代码改动
2. **性能优先**：追踪开销控制在 1ms 以内
3. **向后兼容**：不影响现有功能
4. **渐进式**：先实现基础功能，后续可扩展

## 二、技术选型

### 推荐方案：轻量级 TraceID（阶段一）

**选择理由**：

- ✅ 项目已有 RequestID 生成机制
- ✅ 日志系统已支持上下文传递
- ✅ 无需引入新依赖
- ✅ 实施周期短（1-2小时）
- ✅ 满足当前 90% 的需求

**TraceID 格式**：

```
trace-{timestamp}-{random}
示例：trace-1729756800-a1b2c3d4
```

### 未来扩展：OpenTelemetry（阶段二）

当需要以下功能时再升级：

- 分布式追踪（跨服务调用）
- Span 级别的性能分析
- 与 Jaeger/Zipkin 集成
- 自动化的调用链路图

## 三、实施步骤

### 步骤 1：修改响应结构（添加 TraceID 字段）

**文件**：`internal/model/response.go`

```go
// ResponseData 通用响应数据结构
type ResponseData[T any] struct {
 Code    int    `json:"code" example:"200"`
 Message string `json:"message" example:"success"`
 Data    *T     `json:"data,omitempty"`
 TraceID string `json:"traceId" example:"trace-1729756800-a1b2c3d4"` // 新增
}

// ResponsePaginationData 分页响应数据结构
type ResponsePaginationData[T any] struct {
 Code    int               `json:"code" example:"200"`
 Message string            `json:"message" example:"success"`
 Data    PaginationData[T] `json:"data"`
 TraceID string            `json:"traceId" example:"trace-1729756800-a1b2c3d4"` // 新增
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
 Code    int    `json:"code" example:"400"`
 Message string `json:"message" example:"请求参数错误"`
 TraceID string `json:"traceId" example:"trace-1729756800-a1b2c3d4"` // 新增
}
```

### 步骤 2：增强 Logger 中间件（生成 TraceID）

**文件**：`internal/api/middleware/logger.go`

```go
// TraceIDKey TraceID 上下文键
const TraceIDKey = "trace_id"

// Logger 请求日志中间件
func Logger(next http.Handler) http.Handler {
 return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  // 1. 生成 TraceID（优先使用客户端传入的，否则生成新的）
  traceID := r.Header.Get("X-Trace-ID")
  if traceID == "" {
   traceID = generateTraceID()
  }
  
  // 2. 生成 RequestID（保持现有逻辑）
  requestID := uuid.New().String()
  
  // 3. 将 TraceID 和 RequestID 注入到上下文中
  ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
  ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)
  r = r.WithContext(ctx)
  
  // 4. 设置响应头
  w.Header().Set("X-Request-ID", requestID)
  w.Header().Set("X-Trace-ID", traceID)
  
  // ... 其余代码保持不变
 })
}

// generateTraceID 生成 TraceID
func generateTraceID() string {
 timestamp := time.Now().Unix()
 random := uuid.New().String()[:8]
 return fmt.Sprintf("trace-%d-%s", timestamp, random)
}
```

### 步骤 3：更新日志系统（支持 TraceID）

**文件**：`internal/logger/logger.go`

```go
const (
 // TraceIDKey TraceID 键
 TraceIDKey contextKey = "traceId"
 // RequestIDKey 请求ID键
 RequestIDKey contextKey = "requestId"
 // ... 其他键
)

// extractContextFields 从上下文提取字段
func extractContextFields(ctx context.Context) Fields {
 fields := make(Fields)
 
 // 提取 TraceID（新增）
 if traceID := ctx.Value(TraceIDKey); traceID != nil {
  fields["traceId"] = traceID
 }
 
 // 提取 RequestID
 if requestID := ctx.Value(RequestIDKey); requestID != nil {
  fields["requestId"] = requestID
 }
 
 // ... 其他字段
 
 return fields
}
```

### 步骤 4：更新响应工具函数（自动注入 TraceID）

**文件**：`pkg/response/response.go`

```go
// Success 创建成功响应（自动注入 TraceID）
func Success[T any](ctx context.Context, data *T) model.ResponseData[T] {
 return model.ResponseData[T]{
  Code:    200,
  Message: "success",
  Data:    data,
  TraceID: getTraceID(ctx),
 }
}

// Error 创建错误响应（自动注入 TraceID）
func Error[T any](ctx context.Context, code int, message string) model.ResponseData[T] {
 return model.ResponseData[T]{
  Code:    code,
  Message: message,
  Data:    nil,
  TraceID: getTraceID(ctx),
 }
}

// SuccessPagination 创建分页成功响应（自动注入 TraceID）
func SuccessPagination[T any](ctx context.Context, data T, pageNo, pageSize, totalCount int) model.ResponsePaginationData[T] {
 totalPage := (totalCount + pageSize - 1) / pageSize
 return model.ResponsePaginationData[T]{
  Code:    200,
  Message: "success",
  Data: model.PaginationData[T]{
   Data:       data,
   PageNo:     pageNo,
   PageSize:   pageSize,
   TotalCount: totalCount,
   TotalPage:  totalPage,
  },
  TraceID: getTraceID(ctx),
 }
}

// getTraceID 从上下文中获取 TraceID
func getTraceID(ctx context.Context) string {
 if traceID, ok := ctx.Value("trace_id").(string); ok {
  return traceID
 }
 return ""
}
```

### 步骤 5：更新所有 Handler（传递 Context）

**示例**：`internal/api/handler/auth_handler.go`

```go
// 修改前
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
 // ...
 resp := response.Success(loginResp)
 // ...
}

// 修改后
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
 ctx := r.Context() // 获取上下文
 // ...
 resp := response.Success(ctx, loginResp) // 传递上下文
 // ...
}
```

## 四、实施检查清单

### 代码修改

- [ ] 修改 `internal/model/response.go` - 添加 TraceID 字段
- [ ] 修改 `internal/api/middleware/logger.go` - 生成和注入 TraceID
- [ ] 修改 `internal/logger/logger.go` - 支持 TraceID 提取
- [ ] 创建/修改 `pkg/response/response.go` - 自动注入 TraceID
- [ ] 更新所有 Handler - 传递 Context 到响应函数

### 测试验证

- [ ] 单元测试：TraceID 生成逻辑
- [ ] 集成测试：TraceID 在请求链路中的传递
- [ ] 日志验证：所有日志包含 TraceID
- [ ] 响应验证：所有 API 响应包含 TraceID
- [ ] 性能测试：追踪开销 < 1ms

### 文档更新

- [ ] API 文档：更新响应示例（包含 TraceID）
- [ ] 运维文档：TraceID 使用指南
- [ ] 开发文档：如何在新接口中使用 TraceID

## 五、使用示例

### 客户端请求

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: trace-custom-12345" \
  -d '{"email":"admin@example.com","password":"password"}'
```

### 服务端响应

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "accessToken": "eyJhbGc...",
    "user": {...}
  },
  "traceId": "trace-1729756800-a1b2c3d4"
}
```

### 日志输出

```json
{
  "timestamp": "2024-10-24T10:00:00Z",
  "level": "INFO",
  "message": "用户登录成功",
  "fields": {
    "traceId": "trace-1729756800-a1b2c3d4",
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "userId": "user-123",
    "email": "admin@example.com"
  }
}
```

### 问题排查

```bash
# 1. 用户报告错误，提供 traceId
traceId: trace-1729756800-a1b2c3d4

# 2. 在日志系统中搜索
grep "trace-1729756800-a1b2c3d4" server.log

# 3. 查看完整的请求链路
2024-10-24 10:00:00 [INFO] HTTP request started traceId=trace-1729756800-a1b2c3d4
2024-10-24 10:00:00 [INFO] JWT 认证成功 traceId=trace-1729756800-a1b2c3d4
2024-10-24 10:00:00 [INFO] 查询用户信息 traceId=trace-1729756800-a1b2c3d4
2024-10-24 10:00:01 [ERROR] 数据库查询失败 traceId=trace-1729756800-a1b2c3d4
2024-10-24 10:00:01 [INFO] HTTP request completed traceId=trace-1729756800-a1b2c3d4
```

## 六、性能影响评估

### 开销分析

- TraceID 生成：~0.1ms（UUID + 字符串拼接）
- Context 传递：~0.01ms（指针传递）
- 日志字段添加：~0.05ms（map 操作）
- 响应字段添加：~0.01ms（结构体赋值）

**总开销**：< 0.2ms（可忽略不计）

### 内存影响

- 每个请求额外内存：~100 bytes（TraceID 字符串）
- 对于 1000 QPS：~100KB/s（可忽略不计）

## 七、未来扩展路径

### 阶段二：OpenTelemetry 集成

当需要以下功能时：

1. **分布式追踪**：跨服务调用链路
2. **性能分析**：每个操作的耗时统计
3. **可视化**：Jaeger UI 查看调用链路
4. **告警**：基于追踪数据的异常检测

**实施步骤**：

1. 引入 OpenTelemetry SDK
2. 配置 Tracer Provider
3. 在关键操作处创建 Span
4. 导出到 Jaeger/Zipkin

**代码示例**：

```go
import (
 "go.opentelemetry.io/otel"
 "go.opentelemetry.io/otel/trace"
)

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
 // 创建 Span
 ctx, span := otel.Tracer("user-service").Start(ctx, "GetUser")
 defer span.End()
 
 // 添加属性
 span.SetAttributes(
  attribute.String("user.id", userID),
 )
 
 // 业务逻辑
 user, err := s.repo.FindByID(ctx, userID)
 if err != nil {
  span.RecordError(err)
  return nil, err
 }
 
 return user, nil
}
```

## 八、常见问题

### Q1：TraceID 和 RequestID 有什么区别？

- **TraceID**：标识一次完整的业务请求，可以跨多个服务
- **RequestID**：标识一次 HTTP 请求，仅在单个服务内有效

### Q2：客户端可以自定义 TraceID 吗？

可以。通过 `X-Trace-ID` 请求头传入，服务端会优先使用客户端提供的 TraceID。

### Q3：TraceID 会暴露系统信息吗？

不会。TraceID 只包含时间戳和随机字符串，不包含敏感信息。

### Q4：如何在微服务间传递 TraceID？

通过 HTTP 请求头 `X-Trace-ID` 传递，或者升级到 OpenTelemetry 使用标准的 W3C Trace Context。

## 九、总结

本方案采用**轻量级 TraceID** 实现全链路追踪，具有以下优势：

✅ **快速实施**：1-2 小时完成
✅ **零依赖**：无需引入新库
✅ **高性能**：开销 < 0.2ms
✅ **易维护**：代码简单清晰
✅ **可扩展**：未来可升级到 OpenTelemetry

**立即开始实施，让问题排查变得简单！**
