# 依赖注入和服务生命周期管理

## 概述

本文档描述了 Genkit AI Service 的依赖注入架构和服务生命周期管理策略。

## 架构层次

系统采用分层架构，依赖关系自上而下：

```
Handler 层 (HTTP 请求处理)
    ↓
Service 层 (业务逻辑)
    ↓
Repository 层 (数据访问)
    ↓
Database 层 (数据持久化)
```

## 依赖注入流程

### 1. 配置加载

```go
cfg, err := config.Load()
```

从环境变量和配置文件加载应用配置。

### 2. 基础设施初始化

#### 2.1 日志系统

```go
log := logger.New(logLevel, logFormat, os.Stdout)
```

#### 2.2 数据库连接

```go
db, err := initDatabase(cfg, log)
defer db.Close()
```

- 创建 PostgreSQL 连接
- 执行数据库迁移
- 配置连接池参数

#### 2.3 Genkit 客户端

```go
genkitClient, err := initGenkit(cfg, log)
```

- 初始化 AI 模型客户端
- 配置默认参数

### 3. Repository 层初始化

```go
// 获取 GORM 数据库实例
gormDB := db.GetDB()

// 创建 Repository 实例
sessionRepo := repository.NewSessionRepository(gormDB)
messageRepo := repository.NewMessageRepository(gormDB)
summaryRepo := repository.NewSummaryRepository(gormDB)
```

Repository 层负责数据访问，依赖于数据库连接。

### 4. Service 层初始化

```go
// 会话服务
sessionService := session.NewSessionService(sessionRepo, messageRepo)

// 摘要服务
summaryService := session.NewSummaryService(
    summaryRepo, 
    messageRepo, 
    sessionRepo, 
    aiService, 
    cfg, 
    log
)

// 消息服务
messageService := session.NewMessageService(
    gormDB, 
    sessionRepo, 
    messageRepo, 
    aiService, 
    log
)
```

Service 层实现业务逻辑，依赖于：

- Repository 层（数据访问）
- AI Service（AI 功能）
- Config（配置）
- Logger（日志）

### 5. Handler 层初始化

```go
// 创建 Handler 实例
sessionHandler := handler.NewSessionHandler(sessionService, log)
messageHandler := handler.NewMessageHandler(messageService, log)
```

Handler 层处理 HTTP 请求，依赖于：

- Service 层（业务逻辑）
- Logger（日志）

### 6. 路由注册

```go
serveMux := http.NewServeMux()

// 注册会话管理路由
routes.RegisterSessionRoutes(serveMux, sessionHandler, messageHandler)

// 注册其他路由
routes.RegisterProviderRoutes(serveMux, providerHandler)
```

## 服务生命周期管理

### 启动流程

1. **配置加载** → 2. **日志初始化** → 3. **数据库连接** → 4. **Genkit 客户端初始化**
5. **Repository 初始化** → 6. **Service 初始化** → 7. **Handler 初始化** → 8. **路由注册**
9. **中间件应用** → 10. **HTTP 服务器启动**

### 关闭流程

系统支持优雅关闭（Graceful Shutdown）：

```go
// 监听系统信号
shutdown := make(chan os.Signal, 1)
signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

// 等待关闭信号
<-shutdown

// 创建关闭超时上下文
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 优雅关闭 HTTP 服务器
server.Shutdown(ctx)

// 关闭数据库连接
db.Close()
```

关闭顺序：

1. **停止接受新请求**
2. **等待现有请求完成**（最多 30 秒）
3. **关闭数据库连接**
4. **清理其他资源**

### 错误处理

系统采用渐进式降级策略：

- **数据库不可用**：会话管理功能不可用，但模型提供商 API 仍可用
- **Genkit 客户端不可用**：AI 对话功能不可用，但其他功能仍可用
- **模型数据加载失败**：服务启动失败（必需功能）

## 依赖关系图

```
┌─────────────────────────────────────────────────────────────┐
│                        Main Function                         │
└─────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────┼─────────────────────┐
        ↓                     ↓                     ↓
   ┌─────────┐         ┌──────────┐         ┌──────────┐
   │ Config  │         │ Database │         │  Genkit  │
   └─────────┘         └──────────┘         └──────────┘
        ↓                     ↓                     ↓
        └─────────────────────┼─────────────────────┘
                              ↓
                    ┌──────────────────┐
                    │   Repositories   │
                    │ - SessionRepo    │
                    │ - MessageRepo    │
                    │ - SummaryRepo    │
                    └──────────────────┘
                              ↓
                    ┌──────────────────┐
                    │    Services      │
                    │ - SessionService │
                    │ - MessageService │
                    │ - SummaryService │
                    └──────────────────┘
                              ↓
                    ┌──────────────────┐
                    │    Handlers      │
                    │ - SessionHandler │
                    │ - MessageHandler │
                    └──────────────────┘
                              ↓
                    ┌──────────────────┐
                    │      Router      │
                    └──────────────────┘
```

## 最佳实践

### 1. 构造函数注入

所有依赖通过构造函数注入，避免使用全局变量：

```go
func NewSessionService(
    sessionRepo repository.SessionRepository,
    messageRepo repository.MessageRepository,
) SessionService {
    return &sessionService{
        sessionRepo: sessionRepo,
        messageRepo: messageRepo,
    }
}
```

### 2. 接口依赖

依赖接口而非具体实现，便于测试和替换：

```go
type SessionService interface {
    CreateSession(ctx context.Context, req *CreateSessionRequest) (*SessionResponse, error)
    // ...
}
```

### 3. 单一职责

每个组件只负责一个职责：

- Repository：数据访问
- Service：业务逻辑
- Handler：HTTP 处理

### 4. 错误传播

错误向上传播，在适当的层级处理：

- Repository：返回数据库错误
- Service：包装业务错误
- Handler：转换为 HTTP 响应

### 5. 上下文传递

使用 `context.Context` 传递请求上下文和取消信号：

```go
func (s *sessionService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*SessionResponse, error) {
    // 使用 ctx 进行数据库操作
    err := s.sessionRepo.Create(ctx, session)
    // ...
}
```

## 测试策略

### 单元测试

使用 mock 对象测试各层：

```go
// 测试 Service 层
mockRepo := &MockSessionRepository{}
service := NewSessionService(mockRepo, mockMessageRepo)
```

### 集成测试

测试完整的依赖注入流程：

```go
// 测试完整的初始化流程
db := setupTestDatabase()
service := initSessionHandlers(db, aiService, cfg, log)
```

## 配置管理

### 环境变量

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=genkit_ai

# Genkit 配置
GENKIT_API_KEY=your-api-key
GENKIT_MODEL=gemini-pro

# 服务配置
SERVER_PORT=8080
LOG_LEVEL=info
```

### 配置文件

配置优先级：环境变量 > 配置文件 > 默认值

## 监控和日志

### 结构化日志

```go
log.Info("会话管理服务初始化成功", logger.Fields{
    "repositories": []string{"SessionRepository", "MessageRepository", "SummaryRepository"},
    "services":     []string{"SessionService", "MessageService", "SummaryService"},
    "handlers":     []string{"SessionHandler", "MessageHandler"},
})
```

### 健康检查

```go
// GET /api/v1/health
{
    "status": "healthy",
    "version": "1.0.0",
    "database": "connected",
    "genkit": "connected"
}
```

## 扩展指南

### 添加新的 Repository

1. 定义接口：`internal/repository/new_repository.go`
2. 实现接口：使用 GORM 操作数据库
3. 在 `initSessionHandlers` 中初始化

### 添加新的 Service

1. 定义接口：`internal/service/new_service.go`
2. 实现业务逻辑
3. 注入所需的 Repository 和其他依赖
4. 在 `initSessionHandlers` 中初始化

### 添加新的 Handler

1. 定义 Handler：`internal/api/handler/new_handler.go`
2. 实现 HTTP 处理方法
3. 注入所需的 Service
4. 在路由中注册

## 参考资料

- [Go 依赖注入最佳实践](https://go.dev/blog/wire)
- [GORM 文档](https://gorm.io/docs/)
- [优雅关闭 HTTP 服务器](https://pkg.go.dev/net/http#Server.Shutdown)
