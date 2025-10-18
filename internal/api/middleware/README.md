# HTTP 中间件

本包提供了一组标准的 HTTP 中间件，用于处理日志记录、错误恢复、CORS 和用户身份验证。

## 中间件列表

### 1. Logger 中间件

记录所有 HTTP 请求的详细信息，包括请求方法、路径、耗时、状态码等。

**功能特性：**

- 自动生成唯一的请求 ID
- 将请求 ID 注入到上下文和响应头中
- 记录请求开始和完成的日志
- 记录请求耗时（毫秒级）
- 支持结构化日志输出

**使用示例：**

```go
import "genkit-ai-service/internal/api/middleware"

// 应用日志中间件
handler := middleware.Logger(yourHandler)
```

**日志输出示例：**

```json
{
  "timestamp": "2025-10-16T01:23:53Z",
  "level": "INFO",
  "message": "HTTP request started",
  "fields": {
    "method": "POST",
    "path": "/api/v1/chat",
    "query": "",
    "remoteAddr": "192.168.1.100:54321",
    "requestId": "e1a19b11-1fe5-4ca2-919e-1656f51ab5e6",
    "userAgent": "Mozilla/5.0..."
  }
}
```

### 2. Recovery 中间件

捕获并恢复 panic，防止服务器崩溃，并返回标准的错误响应。

**功能特性：**

- 捕获所有 panic 错误
- 记录详细的堆栈信息
- 返回标准的 JSON 错误响应
- 防止服务器崩溃

**使用示例：**

```go
import "genkit-ai-service/internal/api/middleware"

// 应用恢复中间件
handler := middleware.Recovery(yourHandler)
```

**错误响应示例：**

```json
{
  "code": 500,
  "message": "内部错误",
  "data": null
}
```

### 3. UserContext 中间件

从请求头中提取用户身份信息并存入上下文，用于实现多用户隔离和权限验证。

**功能特性：**

- 从 `X-User-ID` 请求头提取用户 ID
- 将用户 ID 存入请求上下文
- 自动验证用户身份，未提供用户 ID 时返回 401 错误
- 提供便捷的上下文访问函数

**使用示例：**

```go
import "genkit-ai-service/internal/api/middleware"

// 应用用户上下文中间件
handler := middleware.UserContext(yourHandler)

// 在处理器中获取用户 ID
func yourHandler(w http.ResponseWriter, r *http.Request) {
    // 方式1：安全获取（推荐）
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        // 处理未找到用户 ID 的情况
        return
    }
    
    // 方式2：直接获取（确保已经过中间件处理）
    userID := middleware.MustGetUserID(r.Context())
    
    // 使用 userID 进行业务处理
}
```

**请求示例：**

```bash
curl -X GET http://localhost:8080/api/v1/sessions \
  -H "X-User-ID: user-123"
```

**错误响应示例（未提供用户 ID）：**

```json
{
  "code": 401,
  "message": "未提供用户身份信息",
  "data": null
}
```

### 4. CORS 中间件

处理跨域资源共享（CORS）请求，支持预检请求和自定义配置。

**功能特性：**

- 支持自定义允许的来源、方法、头部
- 自动处理 OPTIONS 预检请求
- 支持凭证（credentials）
- 可配置缓存时间

**使用示例：**

```go
import "genkit-ai-service/internal/api/middleware"

// 使用默认配置
cors := middleware.DefaultCORS()
handler := cors.Handler(yourHandler)

// 或自定义配置
cors := &middleware.CORS{
    AllowOrigins:     []string{"http://example.com", "http://test.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           86400, // 24小时
}
handler := cors.Handler(yourHandler)
```

**默认 CORS 配置：**

- AllowOrigins: `["*"]` - 允许所有来源
- AllowMethods: `["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"]`
- AllowHeaders: `["Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"]`
- ExposeHeaders: `["X-Request-ID"]`
- AllowCredentials: `false`
- MaxAge: `86400` (24小时)

## 中间件链式使用

推荐按以下顺序应用中间件：

```go
import (
    "net/http"
    "genkit-ai-service/internal/api/middleware"
)

func setupMiddleware(handler http.Handler) http.Handler {
    // 1. 首先应用恢复中间件（最外层）
    handler = middleware.Recovery(handler)
    
    // 2. 然后应用日志中间件
    handler = middleware.Logger(handler)
    
    // 3. 应用 CORS 中间件
    cors := middleware.DefaultCORS()
    handler = cors.Handler(handler)
    
    // 4. 应用用户上下文中间件（需要身份验证的路由）
    // 注意：可以选择性地应用到特定路由
    handler = middleware.UserContext(handler)
    
    return handler
}

// 使用
mux := http.NewServeMux()
mux.HandleFunc("/api/v1/chat", chatHandler)

server := &http.Server{
    Addr:    ":8080",
    Handler: setupMiddleware(mux),
}
```

## 测试

运行中间件测试：

```bash
# 运行所有中间件测试
go test -v ./internal/api/middleware/...

# 运行特定中间件测试
go test -v ./internal/api/middleware/... -run TestLogger
go test -v ./internal/api/middleware/... -run TestRecovery
go test -v ./internal/api/middleware/... -run TestCORS
go test -v ./internal/api/middleware/... -run TestUserContext
```

## 注意事项

1. **中间件顺序很重要**：Recovery 应该在最外层，以捕获所有 panic
2. **请求 ID**：Logger 中间件会自动生成请求 ID 并注入到上下文中，后续处理器可以通过 `logger.RequestIDKey` 获取
3. **用户身份验证**：UserContext 中间件应该应用到需要身份验证的路由上，可以选择性地应用而不是全局应用
4. **CORS 配置**：生产环境中应该明确指定允许的来源，而不是使用 `*`
5. **性能考虑**：日志中间件会记录每个请求，在高并发场景下注意日志输出的性能影响

## 扩展

如果需要添加新的中间件，请遵循以下模式：

```go
func YourMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 前置处理
        
        // 调用下一个处理器
        next.ServeHTTP(w, r)
        
        // 后置处理
    })
}
```
