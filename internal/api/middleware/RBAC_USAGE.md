# RBAC 权限验证中间件使用指南

## 概述

RBAC（基于角色的访问控制）中间件提供了三个主要的权限验证中间件，用于保护 API 端点：

1. **RequireSystemAdmin** - 要求平台管理员权限
2. **RequireTenantAdmin** - 要求租户管理员或平台管理员权限
3. **RequireTenantAccess** - 要求访问特定租户的权限

## 角色定义

系统支持以下角色（定义在 `internal/model/auth.go`）：

- `system_admin` - 平台管理员，拥有所有租户的管理权限
- `tenant_admin` - 租户管理员，仅拥有所属租户内的管理权限
- `user` - 普通用户，拥有基本业务权限

## 中间件使用

### 1. RequireSystemAdmin

要求用户具有平台管理员权限。适用于平台级别的管理操作。

**使用场景：**

- 创建/删除租户
- 查看所有租户列表
- 启用/禁用租户
- 跨租户的管理操作

**示例：**

```go
// 使用标准库 http.Handler
handler := middleware.RequireSystemAdmin()(yourHandler)

// 在路由中使用
mux.Handle("/api/v1/platform/tenants", 
    middleware.JWTAuth(tokenService, blacklistService)(
        middleware.RequireSystemAdmin()(
            http.HandlerFunc(platformHandler.CreateTenant),
        ),
    ),
)
```

**权限验证逻辑：**

- 检查 JWT Token 中的 roles 字段是否包含 `system_admin`
- 如果不包含，返回 403 Forbidden 错误
- 如果包含，继续处理请求

### 2. RequireTenantAdmin

要求用户具有租户管理员或平台管理员权限。适用于租户级别的管理操作。

**使用场景：**

- 在租户内创建/删除用户
- 查看租户用户列表
- 启用/禁用租户内的用户
- 管理租户内的资源

**示例：**

```go
// 使用标准库 http.Handler
handler := middleware.RequireTenantAdmin()(yourHandler)

// 在路由中使用
mux.Handle("/api/v1/tenants/{tenantId}/users", 
    middleware.JWTAuth(tokenService, blacklistService)(
        middleware.RequireTenantAdmin()(
            http.HandlerFunc(userHandler.CreateUser),
        ),
    ),
)
```

**权限验证逻辑：**

- 检查 JWT Token 中的 roles 字段是否包含 `tenant_admin` 或 `system_admin`
- 如果都不包含，返回 403 Forbidden 错误
- 如果包含任意一个，继续处理请求

### 3. RequireTenantAccess

验证用户是否有权访问目标租户的数据。平台管理员可以访问所有租户，其他用户只能访问自己所属的租户。

**使用场景：**

- 访问特定租户的数据
- 确保租户隔离
- 防止跨租户数据访问

**示例：**

```go
// 使用标准库 http.Handler
handler := middleware.RequireTenantAccess()(yourHandler)

// 在路由中使用
mux.Handle("/api/v1/tenants/{tenantId}/users", 
    middleware.JWTAuth(tokenService, blacklistService)(
        middleware.RequireTenantAccess()(
            http.HandlerFunc(userHandler.ListUsers),
        ),
    ),
)
```

**权限验证逻辑：**

1. 从 URL 路径中提取目标租户 ID（支持格式：`/api/v1/tenants/{tenantId}/...`）
2. 如果路径中没有租户 ID，尝试从查询参数 `tenantId` 获取
3. 如果用户是 `system_admin`，允许访问所有租户
4. 如果用户的 `tenant_id` 与目标租户 ID 匹配，允许访问
5. 否则返回 403 Forbidden 错误

**支持的 URL 格式：**

- `/api/v1/tenants/{tenantId}/users`
- `/api/v1/tenants/{tenantId}/sessions`
- `/api/v1/platform/tenants/{tenantId}/...`
- 查询参数：`/api/v1/data?tenantId={tenantId}`

## 中间件组合使用

可以组合多个中间件来实现复杂的权限控制：

```go
// 示例：租户用户管理 API
// 1. 需要 JWT 认证
// 2. 需要租户管理员权限
// 3. 需要访问目标租户的权限
mux.Handle("/api/v1/tenants/{tenantId}/users", 
    middleware.JWTAuth(tokenService, blacklistService)(
        middleware.RequireTenantAdmin()(
            middleware.RequireTenantAccess()(
                http.HandlerFunc(userHandler.CreateUser),
            ),
        ),
    ),
)
```

**中间件执行顺序：**

1. JWTAuth - 验证 JWT Token，提取用户信息
2. RequireTenantAdmin - 验证用户角色
3. RequireTenantAccess - 验证租户访问权限
4. 业务处理器 - 执行实际的业务逻辑

## 错误响应

所有 RBAC 中间件使用统一的错误响应格式：

### 401 Unauthorized（未授权）

当 JWT Token 缺失或无效时返回：

```json
{
  "code": 401,
  "message": "身份认证信息缺失",
  "data": null
}
```

### 403 Forbidden（权限不足）

当用户权限不足时返回：

```json
{
  "code": 403,
  "message": "权限不足：需要平台管理员权限",
  "data": null
}
```

或

```json
{
  "code": 403,
  "message": "权限不足：无法访问其他租户的数据",
  "data": null
}
```

## 审计日志

所有权限验证失败的情况都会记录到应用日志中，包含以下信息：

- 事件类型：`permission_denied`
- 失败原因
- 请求路径和方法
- 用户 ID 和租户 ID
- 用户角色
- 客户端 IP 地址
- User-Agent

**日志示例：**

```
WARN 权限验证失败 event=permission_denied reason="需要平台管理员权限" 
path=/api/v1/platform/tenants method=POST user_id=550e8400-e29b-41d4-a716-446655440000 
tenant_id=660e8400-e29b-41d4-a716-446655440001 roles=[user] 
ip=192.168.1.100 user_agent="Mozilla/5.0..."
```

## 辅助函数

### GetJWTClaims

从上下文中获取 JWT Claims：

```go
claims, ok := middleware.GetJWTClaims(ctx)
if !ok {
    // JWT Claims 不存在
}
```

### HasRole

检查用户是否具有指定角色：

```go
if middleware.HasRole(ctx, model.RoleSystemAdmin) {
    // 用户是平台管理员
}
```

### HasAnyRole

检查用户是否具有任意一个指定角色：

```go
if middleware.HasAnyRole(ctx, model.RoleTenantAdmin, model.RoleSystemAdmin) {
    // 用户是租户管理员或平台管理员
}
```

### HasAllRoles

检查用户是否具有所有指定角色：

```go
if middleware.HasAllRoles(ctx, model.RoleTenantAdmin, model.RoleUser) {
    // 用户同时具有租户管理员和普通用户角色
}
```

## 最佳实践

1. **始终在 RBAC 中间件之前使用 JWTAuth 中间件**
   - RBAC 中间件依赖 JWT Token 中的用户信息

2. **按照从严格到宽松的顺序应用中间件**
   - 先验证角色权限（RequireSystemAdmin/RequireTenantAdmin）
   - 再验证租户访问权限（RequireTenantAccess）

3. **在业务逻辑中进行二次验证**
   - 中间件提供基础的权限控制
   - 业务逻辑应该进行更细粒度的权限验证

4. **使用辅助函数简化权限检查**
   - 在处理器中使用 HasRole、HasAnyRole 等函数
   - 避免直接操作上下文

5. **记录审计日志**
   - 所有敏感操作都应该记录审计日志
   - 使用 AuditRepository 记录到数据库

## 示例：完整的 API 路由配置

```go
package api

import (
    "net/http"
    "genkit-ai-service/internal/api/handler"
    "genkit-ai-service/internal/api/middleware"
)

func SetupRoutes(
    mux *http.ServeMux,
    tokenService auth.TokenService,
    blacklistService auth.TokenBlacklistService,
    platformHandler *handler.PlatformHandler,
    userHandler *handler.UserHandler,
) {
    // 平台管理 API - 需要平台管理员权限
    mux.Handle("/api/v1/platform/tenants", 
        middleware.JWTAuth(tokenService, blacklistService)(
            middleware.RequireSystemAdmin()(
                http.HandlerFunc(platformHandler.CreateTenant),
            ),
        ),
    )

    // 租户用户管理 API - 需要租户管理员权限和租户访问权限
    mux.Handle("/api/v1/tenants/{tenantId}/users", 
        middleware.JWTAuth(tokenService, blacklistService)(
            middleware.RequireTenantAdmin()(
                middleware.RequireTenantAccess()(
                    http.HandlerFunc(userHandler.CreateUser),
                ),
            ),
        ),
    )

    // 用户数据访问 API - 只需要租户访问权限
    mux.Handle("/api/v1/tenants/{tenantId}/data", 
        middleware.JWTAuth(tokenService, blacklistService)(
            middleware.RequireTenantAccess()(
                http.HandlerFunc(dataHandler.GetData),
            ),
        ),
    )
}
```

## 注意事项

1. **URL 路径格式**
   - RequireTenantAccess 中间件会自动从 URL 路径中提取租户 ID
   - 支持的格式：`/api/v1/tenants/{tenantId}/...`
   - 如果路径格式不匹配，可以使用查询参数 `tenantId`

2. **平台管理员特权**
   - 平台管理员可以绕过租户隔离限制
   - 在 RequireTenantAccess 中，平台管理员可以访问所有租户

3. **错误处理**
   - 所有权限验证失败都会返回标准的错误响应
   - 客户端应该根据错误码（401/403）进行相应处理

4. **性能考虑**
   - 中间件会在每次请求时进行权限验证
   - 验证逻辑非常轻量，不会显著影响性能
   - 避免在中间件中进行数据库查询

## 相关文档

- [认证系统文档](../../../docs/AUTH_SETUP.md)
- [JWT Token 设计](../../../docs/AUTH_QUICK_REFERENCE.md)
- [租户管理设计](../../../.kiro/specs/platform-admin-tenant/design.md)
- [需求文档](../../../.kiro/specs/platform-admin-tenant/requirements.md)
