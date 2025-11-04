# Genkit 会话管理中间件使用指南

## 概述

本文档介绍如何使用 Genkit 会话管理系统的权限验证和审计日志中间件。

## 中间件列表

### 1. JWT 认证中间件 (`JWTAuth`)

验证 JWT token 并将用户信息注入上下文。

```go
router.Use(middleware.JWTAuth(tokenService, blacklistService))
```

### 2. 租户识别中间件 (`TenantIdentifier`)

从请求中识别租户并注入上下文。

```go
config := middleware.TenantIdentifierConfig{
    Strategy:   "header",
    TenantRepo: tenantRepo,
}
router.Use(middleware.TenantIdentifier(config))
```

### 3. RBAC 中间件

#### RequireSystemAdmin - 要求平台管理员权限

```go
router.Use(middleware.RequireSystemAdmin())
```

#### RequireTenantAdmin - 要求租户管理员或平台管理员权限

```go
router.Use(middleware.RequireTenantAdmin())
```

#### RequireTenantAccess - 要求访问特定租户权限

```go
router.Use(middleware.RequireTenantAccess())
```

### 4. Genkit 会话认证中间件

#### RequireGenkitSessionAccess - 验证会话访问权限

```go
config := middleware.GenkitSessionAuthConfig{
    AuditRepo:   auditRepo,
    SessionRepo: sessionRepo,
    UserRepo:    userRepo,
}
router.Use(middleware.RequireGenkitSessionAccess(config))
```

#### RequireGenkitMemoryAccess - 验证记忆访问权限

```go
router.Use(middleware.RequireGenkitMemoryAccess(config))
```

### 5. 审计日志中间件

记录所有请求的审计日志。

```go
auditConfig := middleware.AuditConfig{
    AuditRepo: auditRepo,
    EnabledEvents: []string{
        "session_create",
        "session_delete",
        "permission_denied",
    },
    ExcludedPaths: []string{
        "/health",
        "/metrics",
        "/api/v1/public/*",
    },
}
router.Use(middleware.AuditMiddleware(auditConfig))
```

## 完整的路由配置示例

```go
package router

import (
    "genkit-ai-service/internal/api/handler"
    "genkit-ai-service/internal/api/middleware"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/service/auth"
    
    "github.com/go-chi/chi/v5"
)

func SetupGenkitRoutes(
    r chi.Router,
    tokenService auth.TokenService,
    blacklistService auth.TokenBlacklistService,
    auditRepo repository.AuditRepository,
    sessionRepo repository.SessionRepository,
    userRepo repository.UserRepository,
    tenantRepo repository.TenantRepository,
    handlers *handler.Handlers,
) {
    // 全局中间件
    r.Use(middleware.JWTAuth(tokenService, blacklistService))
    r.Use(middleware.TenantIdentifier(middleware.TenantIdentifierConfig{
        Strategy:   "header",
        TenantRepo: tenantRepo,
    }))
    
    // 审计日志中间件
    r.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        AuditRepo: auditRepo,
        ExcludedPaths: []string{"/health", "/metrics"},
    }))
    
    // Genkit 会话路由组
    r.Route("/api/v1/genkit", func(r chi.Router) {
        // 会话管理路由
        r.Route("/sessions", func(r chi.Router) {
            // 列表和创建不需要会话级别的权限验证
            r.With(middleware.RequireTenantAdmin()).Get("/", handlers.Session.List)
            r.With(middleware.RequireTenantAdmin()).Post("/", handlers.Session.Create)
            
            // 具体会话操作需要会话级别的权限验证
            sessionAuthConfig := middleware.GenkitSessionAuthConfig{
                AuditRepo:   auditRepo,
                SessionRepo: sessionRepo,
                UserRepo:    userRepo,
            }
            
            r.Route("/{sessionId}", func(r chi.Router) {
                r.Use(middleware.RequireGenkitSessionAccess(sessionAuthConfig))
                
                r.Get("/", handlers.Session.Get)
                r.Put("/", handlers.Session.Update)
                r.Delete("/", handlers.Session.Delete)
                
                // 会话消息
                r.Get("/messages", handlers.Message.List)
                r.Post("/messages", handlers.Message.Create)
                
                // 会话上下文
                r.Post("/context", handlers.Context.Build)
                
                // 会话摘要
                r.Post("/summaries", handlers.Summary.Generate)
                r.Get("/summaries", handlers.Summary.List)
            })
        })
        
        // 记忆管理路由
        r.Route("/memories", func(r chi.Router) {
            r.With(middleware.RequireTenantAdmin()).Get("/", handlers.Memory.List)
            r.With(middleware.RequireTenantAdmin()).Post("/", handlers.Memory.Create)
            
            memoryAuthConfig := middleware.GenkitSessionAuthConfig{
                AuditRepo:   auditRepo,
                SessionRepo: sessionRepo,
                UserRepo:    userRepo,
            }
            
            r.Route("/{memoryId}", func(r chi.Router) {
                r.Use(middleware.RequireGenkitMemoryAccess(memoryAuthConfig))
                
                r.Get("/", handlers.Memory.Get)
                r.Put("/", handlers.Memory.Update)
                r.Delete("/", handlers.Memory.Delete)
            })
        })
        
        // 对话生成路由
        r.Route("/chat", func(r chi.Router) {
            sessionAuthConfig := middleware.GenkitSessionAuthConfig{
                AuditRepo:   auditRepo,
                SessionRepo: sessionRepo,
                UserRepo:    userRepo,
            }
            
            r.With(middleware.RequireGenkitSessionAccess(sessionAuthConfig)).
                Post("/", handlers.Chat.Generate)
        })
    })
}
```

## 在服务层使用辅助函数

### 获取用户上下文

```go
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
    // 获取用户上下文
    userCtx := middleware.GetUserContext(ctx)
    if userCtx == nil {
        return nil, errors.NewUnauthorizedError("未认证")
    }
    
    // 检查是否为管理员
    if userCtx.IsAdmin() {
        // 管理员逻辑
    }
    
    // 查询会话
    session, err := s.repo.GetByID(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // 验证租户访问权限
    if !userCtx.CanAccessTenant(session.TenantID.String()) {
        return nil, errors.NewForbiddenError("权限不足")
    }
    
    return session, nil
}
```

### 获取会话 ID

```go
func (s *ContextService) BuildContext(ctx context.Context, req *BuildContextRequest) (*ContextResult, error) {
    // 从上下文获取会话 ID
    sessionID, ok := middleware.GetSessionID(ctx)
    if !ok {
        return nil, errors.NewBadRequestError("缺少会话 ID")
    }
    
    // 或者使用 Must 版本（如果确定会话 ID 存在）
    sessionID = middleware.MustGetSessionID(ctx)
    
    // 业务逻辑...
}
```

### 验证权限

```go
func (s *SessionService) UpdateSession(ctx context.Context, sessionID string, req *UpdateRequest) error {
    // 查询会话
    session, err := s.repo.GetByID(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // 验证访问权限
    if err := middleware.ValidateSessionAccess(ctx, session.TenantID.String()); err != nil {
        return err
    }
    
    // 更新逻辑...
}
```

### 手动记录审计事件

```go
func (s *SessionService) ArchiveSession(ctx context.Context, sessionID string) error {
    // 归档会话
    if err := s.repo.Archive(ctx, sessionID); err != nil {
        return err
    }
    
    // 记录审计事件
    middleware.LogAuditEvent(ctx, "session_archived", map[string]interface{}{
        "session_id": sessionID,
        "reason":     "manual_archive",
    }, s.auditRepo)
    
    return nil
}
```

## 权限检查函数

### 检查角色

```go
// 检查是否为平台管理员
if middleware.HasSystemAdminRole(ctx) {
    // 平台管理员逻辑
}

// 检查是否为租户管理员
if middleware.HasTenantAdminRole(ctx) {
    // 租户管理员逻辑
}

// 检查是否为任意管理员
if middleware.HasAdminRole(ctx) {
    // 管理员逻辑
}

// 检查是否具有特定角色
if middleware.HasRole(ctx, "custom_role") {
    // 自定义角色逻辑
}

// 检查是否具有任意一个角色
if middleware.HasAnyRole(ctx, "role1", "role2") {
    // 具有任意一个角色的逻辑
}
```

## 测试中使用

### 设置测试上下文

```go
func TestSessionService(t *testing.T) {
    // 创建测试上下文
    ctx := context.Background()
    
    // 设置用户上下文
    ctx = middleware.SetUserContext(
        ctx,
        "user-id",
        "tenant-id",
        []string{model.RoleTenantAdmin},
    )
    
    // 设置会话 ID
    ctx = context.WithValue(ctx, "session_id", "session-id")
    
    // 运行测试
    result, err := service.GetSession(ctx, "session-id")
    // 断言...
}
```

## 审计事件类型

系统自动识别以下事件类型：

- `session_create` - 创建会话
- `session_read` - 读取会话
- `session_update` - 更新会话
- `session_delete` - 删除会话
- `memory_create` - 创建记忆
- `memory_read` - 读取记忆
- `memory_update` - 更新记忆
- `memory_delete` - 删除记忆
- `summary_create` - 生成摘要
- `summary_read` - 读取摘要
- `context_build` - 构建上下文
- `chat_generate` - 生成对话
- `permission_denied` - 权限拒绝
- `authentication_failed` - 认证失败
- `api_request` - 默认请求

## 最佳实践

### 1. 中间件顺序

推荐的中间件顺序：

1. JWT 认证
2. 租户识别
3. 审计日志
4. RBAC 权限验证
5. 资源级别权限验证（如会话访问）

### 2. 错误处理

- 使用统一的错误类型（`errors.NewUnauthorizedError`, `errors.NewForbiddenError`）
- 区分 401（未认证）和 403（权限不足）
- 记录详细的错误日志

### 3. 性能优化

- 审计日志异步记录
- 避免在中间件中进行复杂的数据库查询
- 使用上下文传递已解析的信息

### 4. 安全性

- 不信任客户端传入的租户 ID
- 始终在服务层验证权限
- 记录所有权限验证失败的尝试

## 故障排查

### 问题：JWT Claims 未找到

**原因**：请求未经过 JWT 认证中间件

**解决**：确保路由配置了 `JWTAuth` 中间件

### 问题：租户 ID 未找到

**原因**：请求未经过租户识别中间件

**解决**：确保路由配置了 `TenantIdentifier` 中间件

### 问题：会话 ID 未找到

**原因**：请求未经过会话认证中间件，或 URL 路径不正确

**解决**：

1. 确保路由配置了 `RequireGenkitSessionAccess` 中间件
2. 检查 URL 路径格式是否正确（如 `/api/v1/sessions/{sessionId}/...`）
3. 或者在查询参数中提供 `sessionId`

### 问题：权限验证失败

**原因**：用户没有足够的权限

**解决**：

1. 检查用户的角色
2. 检查租户 ID 是否匹配
3. 查看审计日志了解详细信息

## 相关文档

- [多租户访问控制规范](.kiro/steering/multi-tenant-access-control.md)
- [任务 30 完成总结](../../.kiro/specs/genkit-session-management/TASK_30_SUMMARY.md)
