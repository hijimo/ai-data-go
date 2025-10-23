---
inclusion: always
---

# 多租户访问控制规范

## 适用范围

本规范适用于**所有涉及租户数据的操作接口**，包括但不限于：

- 租户管理接口（创建、查询、更新、删除租户）
- 用户管理接口（创建、查询、更新、删除用户）
- 业务数据接口（所有涉及租户数据的CRUD操作）
- 会话管理接口（聊天会话、消息等）
- 配置管理接口（租户配置、用户配置等）

### 例外接口

以下接口**不需要**进行租户权限验证，但仍需要适当的安全控制：

#### 公开接口

- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/refresh` - 刷新令牌
- `POST /api/v1/auth/logout` - 用户登出
- `POST /api/v1/auth/verify-email` - 邮箱验证
- `POST /api/v1/auth/forgot-password` - 忘记密码
- `POST /api/v1/auth/reset-password` - 重置密码

#### 系统接口

- `GET /health` - 健康检查
- `GET /api/v1/health` - API健康检查
- `GET /metrics` - 监控指标（需要特殊认证）
- `GET /swagger/*` - API文档

#### 静态资源

- `GET /api/v1/public/*` - 公开资源访问

**重要提示**：即使是例外接口，也必须实施适当的安全措施，如速率限制、输入验证、防止暴力破解等。

## 核心原则

### 铁则：权限隔离

**这是系统安全的基石，任何情况下都不得违反！**

**平台管理员（system_admin）**：

- 可以查看、修改、删除所有租户的数据
- 可以创建新租户
- 可以管理所有租户下的用户
- 可以启用/禁用任何租户

**租户管理员（tenant_admin）**：

- 只能查看、修改自己租户的数据
- 不能访问其他租户的数据
- 只能管理自己租户下的用户
- 不能创建新租户
- 不能启用/禁用租户

## 租户管理接口规范

### 创建租户

- **路径**: `POST /api/v1/tenants`
- **权限**: 仅平台管理员（system_admin）
- **功能**: 创建租户时自动生成租户管理员账户
- **返回**: 租户信息 + 管理员信息 + 管理员初始密码

### 更新租户

- **路径**: `PUT /api/v1/tenants/{id}`
- **权限**: 平台管理员和租户管理员
- **字段权限**:
  - 租户管理员：只能修改 `name`（租户名称）
  - 平台管理员：可以修改所有字段（name, domain, metadata, status）
- **访问控制**:
  - 租户管理员只能更新自己的租户
  - 平台管理员可以更新任何租户

### 启用/禁用租户

- **路径**: `PATCH /api/v1/tenants/{id}/status`
- **权限**: 仅平台管理员（system_admin）
- **功能**: 启用或禁用租户，影响该租户下所有用户的访问权限

### 查询租户

- **路径**: `GET /api/v1/tenants` 和 `GET /api/v1/tenants/{id}`
- **权限**: 平台管理员和租户管理员
- **访问控制**:
  - 租户管理员只能查看自己的租户
  - 平台管理员可以查看所有租户

### 删除租户

- **路径**: `DELETE /api/v1/tenants/{id}`
- **权限**: 仅平台管理员（system_admin）
- **功能**: 软删除租户

## 用户管理接口规范

### 创建用户

- **路径**: `POST /api/v1/users`
- **权限**: 平台管理员和租户管理员
- **参数**: `tenantId` 为可选参数
- **访问控制**:
  - 租户管理员：必须在自己的租户下创建用户（tenantId 必须匹配或为空时自动使用当前租户）
  - 平台管理员：可以在任何租户下创建用户（可指定任意 tenantId）

### 更新用户

- **路径**: `PUT /api/v1/users/{id}`
- **权限**: 平台管理员和租户管理员
- **访问控制**:
  - 租户管理员：只能更新自己租户下的用户
  - 平台管理员：可以更新任何租户下的用户

### 更新用户状态

- **路径**: `PATCH /api/v1/users/{id}/status`
- **权限**: 平台管理员和租户管理员
- **功能**: 启用或禁用用户
- **访问控制**:
  - 租户管理员：只能更新自己租户下的用户状态
  - 平台管理员：可以更新任何租户下的用户状态

### 查询用户

- **路径**: `GET /api/v1/users` 和 `GET /api/v1/users/{id}`
- **权限**: 平台管理员和租户管理员
- **参数**: `tenantId` 为可选参数（仅列表接口）
- **访问控制**:
  - 租户管理员：只能查看自己租户下的用户
  - 平台管理员：可以查看任何租户下的用户（可通过 tenantId 过滤）

### 删除用户

- **路径**: `DELETE /api/v1/users/{id}`
- **权限**: 平台管理员和租户管理员
- **访问控制**:
  - 租户管理员：只能删除自己租户下的用户
  - 平台管理员：可以删除任何租户下的用户

## 实现要点

### 权限检查顺序

1. 检查用户是否已认证
2. 检查用户角色（system_admin 或 tenant_admin）
3. 如果是租户管理员，验证目标资源是否属于当前租户
4. 如果是平台管理员，允许访问所有资源

### 上下文信息

从请求上下文中获取：

- `user_id`: 当前用户ID
- `tenant_id`: 当前用户所属租户ID
- `roles`: 当前用户角色列表

### 错误处理

- 未认证：返回 401 Unauthorized
- 权限不足：返回 403 Forbidden
- 资源不存在：返回 404 Not Found
- 跨租户访问：返回 403 Forbidden（租户管理员尝试访问其他租户资源）

## 安全注意事项

1. **永远不要信任客户端传入的 tenantId**：始终使用上下文中的租户ID进行权限验证
2. **先验证权限，再执行操作**：避免信息泄露
3. **记录所有管理操作**：用于审计和追踪
4. **敏感操作需要二次确认**：如删除租户、禁用租户等
5. **密码和令牌不得出现在日志中**：保护用户隐私

## 标准实现模式

### 模式1：中间件层角色验证

在路由配置中使用中间件进行基础的角色验证：

```go
// 仅平台管理员可访问
router.POST("/api/v1/tenants", 
    middleware.RequireSystemAdmin()(tenantHandler.HandleCreate))

// 平台管理员或租户管理员可访问
router.GET("/api/v1/users", 
    middleware.RequireTenantAdmin()(userHandler.HandleList))
```

### 模式2：服务层租户隔离验证

在服务层实现细粒度的租户访问权限验证：

```go
// 示例：获取用户详情
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    // 1. 查询用户信息
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 2. 获取当前用户的JWT声明
    claims := middleware.GetJWTClaims(ctx)
    
    // 3. 平台管理员可以访问所有用户
    if hasRole(claims, model.RoleSystemAdmin) {
        return user, nil
    }
    
    // 4. 租户管理员只能访问自己租户的用户
    if hasRole(claims, model.RoleTenantAdmin) {
        if user.TenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的用户")
        }
        return user, nil
    }
    
    // 5. 其他角色拒绝访问
    return nil, errors.NewForbiddenError("权限不足")
}
```

### 模式3：字段级权限控制

对于更新操作，实现字段级别的权限控制：

```go
// 示例：更新租户信息
func (s *TenantService) Update(ctx context.Context, tenantID string, req UpdateTenantRequest) (*model.Tenant, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员的特殊处理
    if hasRole(claims, model.RoleTenantAdmin) {
        // 验证租户ID匹配
        if tenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：无法修改其他租户")
        }
        
        // 只允许修改 name 字段
        if req.Domain != nil || req.Metadata != nil || req.Status != nil {
            return nil, errors.NewForbiddenError("权限不足：租户管理员只能修改租户名称")
        }
    }
    
    // 平台管理员可以修改所有字段
    return s.repo.Update(ctx, tenantID, req)
}
```

### 模式4：列表查询的租户过滤

在列表查询中根据角色自动过滤租户数据：

```go
// 示例：获取用户列表
func (s *UserService) List(ctx context.Context, tenantID *string, pageNo, pageSize int) ([]*model.User, int64, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能查看自己租户的用户
    if hasRole(claims, model.RoleTenantAdmin) {
        // 强制使用当前租户ID，忽略客户端传入的参数
        currentTenantID := claims.TenantID
        return s.repo.ListByTenant(ctx, &currentTenantID, pageNo, pageSize)
    }
    
    // 平台管理员可以查看所有租户或指定租户的用户
    if hasRole(claims, model.RoleSystemAdmin) {
        return s.repo.ListByTenant(ctx, tenantID, pageNo, pageSize)
    }
    
    return nil, 0, errors.NewForbiddenError("权限不足")
}
```

### 模式5：创建操作的租户ID处理

在创建操作中自动处理租户ID：

```go
// 示例：创建用户
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能在自己的租户下创建用户
    if hasRole(claims, model.RoleTenantAdmin) {
        // 如果提供了 tenantId，验证是否匹配
        if req.TenantID != "" && req.TenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：只能在当前租户下创建用户")
        }
        // 强制使用当前租户ID
        req.TenantID = claims.TenantID
    }
    
    // 平台管理员必须明确指定租户ID
    if hasRole(claims, model.RoleSystemAdmin) {
        if req.TenantID == "" {
            return nil, errors.NewBadRequestError("平台管理员必须指定租户ID")
        }
    }
    
    // 创建用户
    user := &model.User{
        TenantID: req.TenantID,
        Email:    req.Email,
        // ... 其他字段
    }
    return s.repo.Create(ctx, user)
}
```

## 权限验证检查清单

在实现任何涉及租户数据的接口时，请使用此检查清单确保正确实现权限控制：

### 设计阶段

- [ ] 确认接口是否需要租户权限验证（参考"例外接口"列表）
- [ ] 确定接口的角色要求（system_admin、tenant_admin 或两者）
- [ ] 确定是否需要字段级权限控制
- [ ] 设计审计日志记录策略

### 实现阶段

#### 路由层

- [ ] 配置了适当的认证中间件（JWT验证）
- [ ] 配置了适当的角色中间件（RequireSystemAdmin 或 RequireTenantAdmin）
- [ ] 路由路径不包含 tenantId（除非是特殊的嵌套资源）

#### Handler层

- [ ] 从请求中正确提取参数
- [ ] 调用服务层方法进行业务处理
- [ ] 正确处理服务层返回的错误
- [ ] 返回标准格式的响应

#### 服务层（关键！）

- [ ] 从上下文中获取JWT声明（用户ID、租户ID、角色）
- [ ] 实现了平台管理员的全局访问逻辑
- [ ] 实现了租户管理员的租户隔离验证
- [ ] 对于查询操作：验证目标资源是否属于当前租户
- [ ] 对于创建操作：自动设置或验证租户ID
- [ ] 对于更新操作：验证目标资源是否属于当前租户
- [ ] 对于删除操作：验证目标资源是否属于当前租户
- [ ] 对于列表操作：根据角色自动过滤租户数据
- [ ] 权限验证失败时返回明确的错误信息
- [ ] 记录权限验证失败的审计日志

#### 仓储层

- [ ] 提供按租户ID过滤的查询方法
- [ ] 所有查询都包含软删除过滤（is_deleted=false）
- [ ] 使用参数化查询防止SQL注入

### 测试阶段

- [ ] 测试平台管理员访问自己租户的资源
- [ ] 测试平台管理员访问其他租户的资源
- [ ] 测试租户管理员访问自己租户的资源
- [ ] 测试租户管理员尝试访问其他租户的资源（应返回403）
- [ ] 测试普通用户尝试访问管理接口（应返回403）
- [ ] 测试未认证用户访问接口（应返回401）
- [ ] 测试边界情况（不存在的资源、无效的ID等）

### 安全审查

- [ ] 不信任客户端传入的 tenantId 参数
- [ ] 先验证权限，再执行数据库操作
- [ ] 权限验证失败时不泄露敏感信息
- [ ] 记录所有权限验证失败的尝试
- [ ] 敏感信息（密码、令牌）不出现在日志中

## 常见错误和解决方案

### 错误1：忘记在服务层验证租户ID

**错误示例**：

```go
// ❌ 危险！只在中间件验证角色，未在服务层验证租户ID
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    return s.repo.FindByID(ctx, userID)  // 租户管理员可能访问其他租户的用户
}
```

**正确示例**：

```go
// ✅ 在服务层验证租户ID
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    claims := middleware.GetJWTClaims(ctx)
    if !hasRole(claims, model.RoleSystemAdmin) && user.TenantID != claims.TenantID {
        return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的用户")
    }
    
    return user, nil
}
```

### 错误2：信任客户端传入的租户ID

**错误示例**：

```go
// ❌ 危险！直接使用客户端传入的租户ID
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
    user := &model.User{
        TenantID: req.TenantID,  // 租户管理员可能在其他租户下创建用户
        Email:    req.Email,
    }
    return s.repo.Create(ctx, user)
}
```

**正确示例**：

```go
// ✅ 验证并强制使用正确的租户ID
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能在自己的租户下创建用户
    if hasRole(claims, model.RoleTenantAdmin) {
        if req.TenantID != "" && req.TenantID != claims.TenantID {
            return nil, errors.NewForbiddenError("权限不足：只能在当前租户下创建用户")
        }
        req.TenantID = claims.TenantID  // 强制使用当前租户ID
    }
    
    user := &model.User{
        TenantID: req.TenantID,
        Email:    req.Email,
    }
    return s.repo.Create(ctx, user)
}
```

### 错误3：列表查询未过滤租户

**错误示例**：

```go
// ❌ 危险！返回所有用户
func (s *UserService) List(ctx context.Context, pageNo, pageSize int) ([]*model.User, int64, error) {
    return s.repo.List(ctx, pageNo, pageSize)  // 租户管理员可能看到其他租户的用户
}
```

**正确示例**：

```go
// ✅ 根据角色过滤租户
func (s *UserService) List(ctx context.Context, tenantID *string, pageNo, pageSize int) ([]*model.User, int64, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能查看自己租户的用户
    if hasRole(claims, model.RoleTenantAdmin) {
        currentTenantID := claims.TenantID
        return s.repo.ListByTenant(ctx, &currentTenantID, pageNo, pageSize)
    }
    
    // 平台管理员可以查看所有或指定租户的用户
    return s.repo.ListByTenant(ctx, tenantID, pageNo, pageSize)
}
```

### 错误4：权限验证后仍泄露信息

**错误示例**：

```go
// ❌ 先查询数据，再验证权限，可能泄露资源是否存在
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err  // 如果是 "not found"，泄露了资源不存在的信息
    }
    
    claims := middleware.GetJWTClaims(ctx)
    if !hasRole(claims, model.RoleSystemAdmin) && user.TenantID != claims.TenantID {
        return nil, errors.NewForbiddenError("权限不足")
    }
    
    return user, nil
}
```

**正确示例**：

```go
// ✅ 权限验证失败时统一返回403，不泄露资源是否存在
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        // 对于租户管理员，即使资源不存在也返回403
        claims := middleware.GetJWTClaims(ctx)
        if !hasRole(claims, model.RoleSystemAdmin) {
            return nil, errors.NewForbiddenError("权限不足")
        }
        return nil, err
    }
    
    claims := middleware.GetJWTClaims(ctx)
    if !hasRole(claims, model.RoleSystemAdmin) && user.TenantID != claims.TenantID {
        return nil, errors.NewForbiddenError("权限不足")
    }
    
    return user, nil
}
```

## 审计日志要求

所有权限验证失败的尝试都必须记录审计日志，包括：

```go
// 记录权限验证失败
logger.WarnContext(ctx, "权限验证失败", 
    "event", "permission_denied",
    "reason", "尝试访问其他租户的数据",
    "user_id", claims.Subject,
    "user_tenant_id", claims.TenantID,
    "target_resource_type", "user",
    "target_resource_id", userID,
    "target_tenant_id", user.TenantID,
    "ip", getClientIP(r),
    "user_agent", r.UserAgent(),
)
```

**审计日志字段说明**：

- `event`: 事件类型（permission_denied, unauthorized_access等）
- `reason`: 失败原因的简短描述
- `user_id`: 操作者的用户ID
- `user_tenant_id`: 操作者所属的租户ID
- `target_resource_type`: 目标资源类型（user, tenant, session等）
- `target_resource_id`: 目标资源ID
- `target_tenant_id`: 目标资源所属的租户ID
- `ip`: 客户端IP地址
- `user_agent`: 客户端User-Agent

## 测试要求

每个接口都必须测试以下场景：

1. ✅ 平台管理员访问自己租户的资源
2. ✅ 平台管理员访问其他租户的资源
3. ✅ 租户管理员访问自己租户的资源
4. ❌ 租户管理员尝试访问其他租户的资源（应返回403）
5. ❌ 普通用户尝试访问管理接口（应返回403）
6. ❌ 未认证用户访问接口（应返回401）

## 总结

多租户访问控制是系统安全的基石。遵循本规范，确保：

1. **中间件层**：验证用户身份和基本角色
2. **服务层**：实施细粒度的租户隔离验证（关键！）
3. **仓储层**：提供按租户过滤的查询方法
4. **审计日志**：记录所有权限验证失败的尝试

**记住**：永远不要信任客户端传入的 tenantId，始终在服务层验证租户访问权限！
