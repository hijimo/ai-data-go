# API 路由配置

本包提供了 HTTP 路由配置和中间件集成。

## 功能特性

- 统一的路由配置管理
- 中间件链式应用
- 标准化的 API 端点
- 健康检查端点

## 路由列表

### API 路由

| 方法 | 路径 | 描述 | 处理器 |
|------|------|------|--------|
| POST | `/api/v1/chat` | AI 对话接口 | ChatHandler |
| POST | `/api/v1/chat/abort` | 中止对话接口 | AbortHandler |

### 系统路由

| 方法 | 路径 | 描述 | 处理器 |
|------|------|------|--------|
| GET | `/health` | 健康检查接口 | HealthHandler |

## 中间件

路由器应用了以下中间件（按执行顺序）：

1. **Recovery** - Panic 恢复中间件
   - 捕获并记录 panic 错误
   - 返回标准错误响应
   - 防止服务崩溃

2. **Logger** - 请求日志中间件
   - 记录请求开始和完成
   - 生成唯一的请求 ID
   - 记录请求耗时和状态码

3. **CORS** - 跨域资源共享中间件
   - 处理跨域请求
   - 支持预检请求
   - 可配置允许的来源、方法和头部

## 使用示例

### 创建路由器

```go
package main

import (
    "genkit-ai-service/internal/api"
    "genkit-ai-service/internal/logger"
    "genkit-ai-service/internal/service/ai"
    "genkit-ai-service/internal/service/health"
)

func main() {
    // 创建服务实例
    aiService := ai.NewGenkitService(...)
    healthService := health.NewService(...)
    log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)
    
    // 创建路由器
    router := api.NewRouter(aiService, healthService, log)
    
    // 获取配置好的 HTTP 处理器
    handler := router.Handler()
    
    // 启动 HTTP 服务器
    http.ListenAndServe(":8080", handler)
}
```

### 自定义 CORS 配置

如果需要自定义 CORS 配置，可以在创建路由器后修改：

```go
router := api.NewRouter(aiService, healthService, log)

// 自定义 CORS 配置
router.corsConfig = &middleware.CORS{
    AllowOrigins:     []string{"https://example.com"},
    AllowMethods:     []string{"GET", "POST"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}

handler := router.Handler()
```

## 测试

运行路由器测试：

```bash
go test -v ./internal/api/...
```

## 架构说明

### 路由器结构

```go
type Router struct {
    mux           *http.ServeMux      // HTTP 路由复用器
    chatHandler   *handler.ChatHandler    // 对话处理器
    abortHandler  *handler.AbortHandler   // 中止处理器
    healthHandler *handler.HealthHandler  // 健康检查处理器
    corsConfig    *middleware.CORS        // CORS 配置
}
```

### 中间件应用顺序

请求流程：

```
客户端请求
    ↓
Recovery 中间件（最外层）
    ↓
Logger 中间件
    ↓
CORS 中间件
    ↓
路由匹配
    ↓
处理器执行
    ↓
响应返回
```

## 扩展路由

如果需要添加新的路由，在 `Setup()` 方法中注册：

```go
func (r *Router) Setup() http.Handler {
    // 现有路由
    r.mux.HandleFunc("/api/v1/chat", r.chatHandler.HandleChat)
    r.mux.HandleFunc("/api/v1/chat/abort", r.abortHandler.HandleAbort)
    r.mux.HandleFunc("/health", r.healthHandler.Handle)
    
    // 添加新路由
    r.mux.HandleFunc("/api/v1/new-endpoint", r.newHandler.Handle)
    
    // 应用中间件
    var handler http.Handler = r.mux
    handler = r.corsConfig.Handler(handler)
    handler = middleware.Logger(handler)
    handler = middleware.Recovery(handler)
    
    return handler
}
```

## 注意事项

1. **中间件顺序**：中间件的应用顺序很重要，Recovery 必须在最外层以捕获所有 panic
2. **请求 ID**：Logger 中间件会自动生成请求 ID 并注入到上下文中
3. **CORS 配置**：默认允许所有来源，生产环境应该配置具体的允许来源
4. **错误处理**：所有处理器都应该返回标准的 ResponseData 格式
