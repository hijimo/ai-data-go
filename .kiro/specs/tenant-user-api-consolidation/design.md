# 租户和用户管理接口合并优化设计文档

## 概述

本设计文档描述了租户和用户管理接口的合并优化方案。当前系统存在功能重复的接口：

- 租户管理：`/api/v1/platform/tenants` 和 `/api/v1/tenants`
- 用户管理：`/api/v1/users` 和 `/api/v1/tenants/{tenantId}/users`

本次优化将合并这些接口，提供统一的API设计，同时严格遵循多租户访问控制规范。

### 设计目标

1. **简化API结构**：移除重复接口，提供单一、清晰的API路由
2. **统一权限控制**：在所有接口中实施一致的权限验证逻辑
3. **保持向后兼容**：确保现有功能不受影响
4. **提升可维护性**：减少代码重复，降低维护成本
5. **增强安全性**：严格执行租户隔离和权限控制

### 核心设计原则

**权限隔离铁则**：

- 平台管理员（system_admin）：可以访问、修改、删除所有租户的数据
- 租户管理员（tenant_admin）：只能访问、修改自己租户的数据，不能访问其他租户的数据
- 普通用户：不能访问管理接口

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                           │
│                    (路由 + 中间件)                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ├─── JWT认证中间件
                              ├─── RBAC权限中间件
                              └─── 租户访问控制中间件
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                             │
┌───────▼────────┐                          ┌────────▼────────┐
│  租户管理Handler │                          │  用户管理Handler │
│  (合并后)       │                          │  (合并后)        │
└───────┬────────┘                          └────────┬────────┘
        │                                             │
┌───────▼────────┐                          ┌────────▼────────┐
│  租户服务层     │                          │  用户服务层      │
└───────┬────────┘                          └────────┬────────┘
        │                                             │
┌───────▼────────┐                          ┌────────▼────────┐
│  租户仓储层     │                          │  用户仓储层      │
└───────┬────────┘                          └────────┬────────┘
        │                                             │
        └─────────────────┬───────────────────────────┘
                          │
                  ┌───────▼────────┐
                  │   PostgreSQL    │
                  └─────────────────┘
```

### 权限控制架构

```
请求流程：
1. 客户端请求 → JWT认证中间件（验证token）
2. JWT认证中间件 → RBAC中间件（验证角色）
3. RBAC中间件 → Handler（业务逻辑）
4. Handler → 服务层（权限二次验证 + 业务逻辑）
5. 服务层 → 仓储层（数据访问）
```

**权限验证层次**：

- **第一层（中间件层）**：验证用户角色是否满足接口的基本权限要求
- **第二层（服务层）**：验证用户是否有权访问特定的租户数据

## 组件和接口设计

### 1. 租户管理接口

#### 1.1 创建租户（带管理员）

**路由**: `POST /api/v1/tenants`  
**权限**: 仅平台管理员（system_admin）  
**中间件**: `RequireSystemAdmin()`

**请求参数**:

```go
type CreateTenantRequest struct {
    TenantName       string                 `json:"tenantName" validate:"required,min=1,max=255"`
    TenantDomain     string                 `json:"tenantDomain" validate:"required,max=255"`
    TenantMetadata   map[string]interface{} `json:"tenantMetadata"`
    AdminEmail       string                 `json:"adminEmail" validate:"omitempty,email"`
    AdminDisplayName string                 `json:"adminDisplayName" validate:"omitempty,max=255"`
}
```

**响应数据**:

```go
type CreateTenantResponse struct {
    Tenant        *model.Tenant `json:"tenant"`
    AdminUser     *model.User   `json:"adminUser"`
    AdminPassword string        `json:"adminPassword"`
}
```

**业务逻辑**:

1. 验证平台管理员权限
2. 验证租户域名唯一性
3. 创建租户（type="tenant", status=true）
4. 生成管理员邮箱（默认为 admin@{tenantDomain}）
5. 生成16位随机强密码
6. 创建租户管理员账户（角色为 tenant_admin）
7. 返回租户信息和管理员初始密码

#### 1.2 获取租户列表

**路由**: `GET /api/v1/tenants`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**查询参数**:

```go
pageNo   int    // 页码，默认1
pageSize int    // 每页大小，默认10，最大100
type     string // 租户类型过滤（可选）：system, tenant
```

**响应数据**:

```go
type TenantListResponse struct {
    Code    int                    `json:"code"`
    Message string                 `json:"message"`
    Data    PaginationData[Tenant] `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色（tenant_admin 或 system_admin）
2. 如果是平台管理员：
   - 返回所有租户列表（可按type过滤）
3. 如果是租户管理员：
   - 只返回当前用户所属的租户信息（单条记录）
   - 忽略type过滤参数

**设计决策**：租户管理员调用列表接口时，只返回自己的租户信息，这样可以保持接口的一致性，避免需要单独的"获取当前租户"接口。

#### 1.3 获取租户详情

**路由**: `GET /api/v1/tenants/{id}`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**响应数据**:

```go
type TenantDataResponse struct {
    Code    int           `json:"code"`
    Message string        `json:"message"`
    Data    *model.Tenant `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 如果是平台管理员：
   - 允许查看任意租户
3. 如果是租户管理员：
   - 验证目标租户ID是否与当前用户的租户ID匹配
   - 不匹配则返回403错误
4. 查询并返回租户信息

#### 1.4 更新租户

**路由**: `PUT /api/v1/tenants/{id}`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**请求参数**:

```go
type UpdateTenantRequest struct {
    Name     *string                `json:"name" validate:"omitempty,min=1,max=255"`
    Domain   *string                `json:"domain" validate:"omitempty,max=255"`
    Metadata map[string]interface{} `json:"metadata"`
    Status   *bool                  `json:"status"`
}
```

**响应数据**:

```go
type TenantDataResponse struct {
    Code    int           `json:"code"`
    Message string        `json:"message"`
    Data    *model.Tenant `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 如果是租户管理员：
   - 验证目标租户ID是否与当前用户的租户ID匹配
   - 只允许修改 `name` 字段
   - 如果请求中包含其他字段（domain, metadata, status），返回403错误
3. 如果是平台管理员：
   - 允许修改所有字段
4. 更新租户信息并返回

**设计决策**：通过字段级权限控制，实现不同角色对同一接口的差异化访问。

#### 1.5 启用/禁用租户

**路由**: `PATCH /api/v1/tenants/{id}/status`  
**权限**: 仅平台管理员  
**中间件**: `RequireSystemAdmin()`

**请求参数**:

```go
type UpdateTenantStatusRequest struct {
    Status bool `json:"status" validate:"required"`
}
```

**响应数据**:

```go
type TenantDataResponse struct {
    Code    int           `json:"code"`
    Message string        `json:"message"`
    Data    *model.Tenant `json:"data"`
}
```

**业务逻辑**:

1. 验证平台管理员权限
2. 验证目标租户存在
3. 不允许禁用平台租户（type="system"）
4. 更新租户状态
5. 返回更新后的租户信息

#### 1.6 删除租户

**路由**: `DELETE /api/v1/tenants/{id}`  
**权限**: 仅平台管理员  
**中间件**: `RequireSystemAdmin()`

**响应数据**:

```go
type AnyDataResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}
```

**业务逻辑**:

1. 验证平台管理员权限
2. 验证目标租户存在
3. 不允许删除平台租户（type="system"）
4. 执行软删除（设置 is_deleted=true）
5. 返回成功响应

### 2. 用户管理接口

#### 2.1 创建用户

**路由**: `POST /api/v1/users`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**请求参数**:

```go
type CreateUserRequest struct {
    TenantID    string                 `json:"tenantId" validate:"omitempty"`  // 可选
    Email       string                 `json:"email" validate:"required,email"`
    Password    string                 `json:"password" validate:"required,min=8"`
    DisplayName string                 `json:"displayName"`
    Phone       string                 `json:"phone"`
    IsAdmin     bool                   `json:"isAdmin"`
    Roles       []string               `json:"roles"`
    Meta        map[string]interface{} `json:"meta"`
}
```

**响应数据**:

```go
type UserDataResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    *model.User `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 处理租户ID：
   - 如果是租户管理员：
     - 如果未提供 tenantId，使用当前用户的租户ID
     - 如果提供了 tenantId，验证是否与当前用户的租户ID匹配
     - 不匹配则返回403错误
   - 如果是平台管理员：
     - 必须提供 tenantId
     - 允许在任意租户下创建用户
3. 验证邮箱唯一性
4. 创建用户并返回

**设计决策**：通过可选的 tenantId 参数，实现单一接口支持两种角色的不同使用场景。

#### 2.2 获取用户列表

**路由**: `GET /api/v1/users`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**查询参数**:

```go
pageNo   int    // 页码，默认1
pageSize int    // 每页大小，默认20，最大100
tenantId string // 租户ID过滤（可选，仅平台管理员可用）
```

**响应数据**:

```go
type UserListResponse struct {
    Code    int                  `json:"code"`
    Message string               `json:"message"`
    Data    PaginationData[User] `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 如果是租户管理员：
   - 忽略 tenantId 参数
   - 只返回当前用户所属租户下的用户列表
3. 如果是平台管理员：
   - 如果提供了 tenantId，返回指定租户下的用户列表
   - 如果未提供 tenantId，返回所有租户下的用户列表
4. 分页查询并返回

#### 2.3 获取用户详情

**路由**: `GET /api/v1/users/{id}`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**响应数据**:

```go
type UserDataResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    *model.User `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 查询用户信息
3. 如果是租户管理员：
   - 验证目标用户的租户ID是否与当前用户的租户ID匹配
   - 不匹配则返回403错误
4. 如果是平台管理员：
   - 允许查看任意租户下的用户
5. 返回用户信息

#### 2.4 更新用户

**路由**: `PUT /api/v1/users/{id}`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**请求参数**:

```go
type UpdateUserRequest struct {
    Email       *string                `json:"email" validate:"omitempty,email"`
    DisplayName *string                `json:"displayName"`
    Phone       *string                `json:"phone"`
    IsActive    *bool                  `json:"isActive"`
    IsAdmin     *bool                  `json:"isAdmin"`
    Roles       []string               `json:"roles"`
    Meta        map[string]interface{} `json:"meta"`
}
```

**响应数据**:

```go
type UserDataResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    *model.User `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 查询目标用户
3. 如果是租户管理员：
   - 验证目标用户的租户ID是否与当前用户的租户ID匹配
   - 不匹配则返回403错误
4. 如果是平台管理员：
   - 允许更新任意租户下的用户
5. 更新用户信息并返回

#### 2.5 更新用户状态

**路由**: `PATCH /api/v1/users/{id}/status`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**请求参数**:

```go
type UpdateUserStatusRequest struct {
    IsActive bool `json:"isActive" validate:"required"`
}
```

**响应数据**:

```go
type UserDataResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    *model.User `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 查询目标用户
3. 如果是租户管理员：
   - 验证目标用户的租户ID是否与当前用户的租户ID匹配
   - 不匹配则返回403错误
4. 如果是平台管理员：
   - 允许更新任意租户下的用户状态
5. 更新用户状态并返回

#### 2.6 删除用户

**路由**: `DELETE /api/v1/users/{id}`  
**权限**: 平台管理员或租户管理员  
**中间件**: `RequireTenantAdmin()`

**响应数据**:

```go
type AnyDataResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}
```

**业务逻辑**:

1. 验证用户角色
2. 查询目标用户
3. 如果是租户管理员：
   - 验证目标用户的租户ID是否与当前用户的租户ID匹配
   - 不匹配则返回403错误
4. 如果是平台管理员：
   - 允许删除任意租户下的用户
5. 执行软删除并返回

## 数据模型

### Tenant 模型

```go
type Tenant struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name      string         `gorm:"type:varchar(255);not null;uniqueIndex"`
    Domain    string         `gorm:"type:varchar(255)"`
    Type      string         `gorm:"type:varchar(32);not null;default:'tenant';index"`
    Metadata  datatypes.JSON `gorm:"type:jsonb"`
    Status    bool           `gorm:"default:true"`
    CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
    UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
    CreatedBy *uuid.UUID     `gorm:"type:uuid"`
    IsDeleted bool           `gorm:"default:false"`
}
```

### User 模型

```go
type User struct {
    ID                  uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID            uuid.UUID      `gorm:"type:uuid;not null;index"`
    Email               string         `gorm:"type:varchar(320);not null;uniqueIndex"`
    EmailVerified       bool           `gorm:"default:false"`
    Phone               string         `gorm:"type:varchar(20)"`
    PasswordHash        string         `gorm:"type:text;not null"`
    DisplayName         string         `gorm:"type:varchar(255)"`
    IsActive            bool           `gorm:"default:true"`
    IsAdmin             bool           `gorm:"default:false"`
    Roles               datatypes.JSON `gorm:"type:jsonb"`
    Meta                datatypes.JSON `gorm:"type:jsonb"`
    LastLoginAt         *time.Time
    CreatedAt           time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
    UpdatedAt           time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
    CreatedBy           *uuid.UUID     `gorm:"type:uuid"`
    IsDeleted           bool           `gorm:"default:false"`
    FailedLoginAttempts int            `gorm:"default:0"`
    LockedUntil         *time.Time
}
```

## 错误处理

### 错误码定义

| 错误码 | HTTP状态码 | 说明 |
|--------|-----------|------|
| 400 | 400 Bad Request | 请求参数错误 |
| 401 | 401 Unauthorized | 未认证 |
| 403 | 403 Forbidden | 权限不足 |
| 404 | 404 Not Found | 资源不存在 |
| 422 | 422 Unprocessable Entity | 参数验证失败 |
| 500 | 500 Internal Server Error | 服务器内部错误 |

### 错误响应格式

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

### 常见错误场景

1. **跨租户访问错误**
   - 场景：租户管理员尝试访问其他租户的数据
   - 错误码：403
   - 错误信息：`"权限不足：无法访问其他租户的数据"`

2. **权限不足错误**
   - 场景：租户管理员尝试执行仅平台管理员可执行的操作
   - 错误码：403
   - 错误信息：`"权限不足：需要平台管理员权限"`

3. **字段权限错误**
   - 场景：租户管理员尝试修改不允许修改的字段
   - 错误码：403
   - 错误信息：`"权限不足：租户管理员只能修改租户名称"`

4. **资源不存在错误**
   - 场景：请求的租户或用户不存在
   - 错误码：404
   - 错误信息：`"租户不存在"` 或 `"用户不存在"`

5. **参数验证错误**
   - 场景：请求参数不符合验证规则
   - 错误码：422
   - 错误信息：包含详细的验证错误信息

## 测试策略

### 单元测试

**测试范围**：

- 服务层业务逻辑
- 权限验证逻辑
- 数据转换逻辑

**测试用例示例**：

1. 平台管理员创建租户成功
2. 租户管理员尝试创建租户失败（权限不足）
3. 租户管理员更新自己租户成功
4. 租户管理员尝试更新其他租户失败
5. 平台管理员更新任意租户成功

### 集成测试

**测试范围**：

- 完整的API请求流程
- 中间件和Handler的集成
- 数据库操作

**测试用例示例**：

1. 端到端创建租户流程
2. 端到端用户管理流程
3. 跨租户访问控制验证
4. 分页查询功能验证

### 权限测试矩阵

| 接口 | 平台管理员 | 租户管理员（自己租户） | 租户管理员（其他租户） | 普通用户 |
|------|-----------|---------------------|---------------------|---------|
| POST /api/v1/tenants | ✅ | ❌ | ❌ | ❌ |
| GET /api/v1/tenants | ✅（所有） | ✅（仅自己） | N/A | ❌ |
| GET /api/v1/tenants/{id} | ✅ | ✅（仅自己） | ❌ | ❌ |
| PUT /api/v1/tenants/{id} | ✅（所有字段） | ✅（仅name） | ❌ | ❌ |
| PATCH /api/v1/tenants/{id}/status | ✅ | ❌ | ❌ | ❌ |
| DELETE /api/v1/tenants/{id} | ✅ | ❌ | ❌ | ❌ |
| POST /api/v1/users | ✅ | ✅（仅自己租户） | ❌ | ❌ |
| GET /api/v1/users | ✅（所有） | ✅（仅自己租户） | N/A | ❌ |
| GET /api/v1/users/{id} | ✅ | ✅（仅自己租户） | ❌ | ❌ |
| PUT /api/v1/users/{id} | ✅ | ✅（仅自己租户） | ❌ | ❌ |
| PATCH /api/v1/users/{id}/status | ✅ | ✅（仅自己租户） | ❌ | ❌ |
| DELETE /api/v1/users/{id} | ✅ | ✅（仅自己租户） | ❌ | ❌ |

## 实施计划

### 阶段1：准备工作

1. **更新Handler层**
   - 修改 `TenantHandler`，移除平台管理员专用逻辑
   - 修改 `UserHandler`，移除租户路径相关逻辑
   - 统一权限验证逻辑

2. **更新服务层**
   - 在 `TenantService` 中实现角色感知的业务逻辑
   - 在 `UserService` 中实现租户ID处理逻辑
   - 添加字段级权限验证

### 阶段2：路由整合

1. **更新路由配置**
   - 移除 `/api/v1/platform/tenants` 相关路由
   - 移除 `/api/v1/tenants/{tenantId}/users` 相关路由
   - 保留并优化 `/api/v1/tenants` 和 `/api/v1/users` 路由

2. **配置中间件**
   - 为租户管理接口配置适当的权限中间件
   - 为用户管理接口配置适当的权限中间件

### 阶段3：代码清理

1. **删除冗余代码**
   - 删除 `PlatformHandler` 及相关代码
   - 删除 `UserHandler` 中的租户路径处理方法
   - 清理未使用的导入和类型定义

2. **更新Swagger文档**
   - 更新所有接口的Swagger注释
   - 添加详细的权限说明
   - 更新请求/响应示例

### 阶段4：测试和验证

1. **单元测试**
   - 编写服务层权限验证测试
   - 编写业务逻辑测试

2. **集成测试**
   - 编写API端到端测试
   - 验证权限控制矩阵

3. **手动测试**
   - 使用不同角色的用户测试所有接口
   - 验证错误处理和边界情况

### 阶段5：文档更新

1. **更新Agent Steering规范**
   - 更新多租户访问控制规范
   - 添加权限控制铁则
   - 提供标准实现模式

2. **更新API文档**
   - 生成最新的Swagger文档
   - 更新接口使用说明

## 关键设计决策

### 决策1：单一接口 vs 多接口

**决策**：采用单一接口设计，通过角色和上下文区分行为

**理由**：

- 减少API端点数量，降低维护成本
- 提供一致的API体验
- 简化客户端集成
- 通过服务层逻辑实现角色差异化

**权衡**：

- 服务层逻辑稍微复杂
- 需要更细致的权限验证

### 决策2：可选 tenantId 参数

**决策**：在用户管理接口中，tenantId 作为可选参数

**理由**：

- 租户管理员无需关心 tenantId，系统自动使用当前租户
- 平台管理员可以明确指定目标租户
- 保持接口简洁性

**权衡**：

- 需要在服务层处理参数默认值
- 需要验证参数合法性

### 决策3：字段级权限控制

**决策**：在更新租户接口中，通过字段级权限控制实现差异化访问

**理由**：

- 避免创建多个相似的接口
- 提供灵活的权限控制
- 符合RESTful设计原则

**权衡**：

- 需要在服务层验证字段权限
- 错误信息需要明确指出权限问题

### 决策4：中间件 + 服务层双重验证

**决策**：在中间件层验证角色，在服务层验证租户访问权限

**理由**：

- 中间件层快速拒绝无权限请求
- 服务层进行细粒度的租户隔离验证
- 分层清晰，职责明确

**权衡**：

- 需要维护两层权限验证逻辑
- 需要确保两层验证的一致性

## 安全考虑

### 1. 租户隔离

**措施**：

- 所有数据查询必须包含租户ID过滤（除平台管理员外）
- 在服务层验证用户是否有权访问目标租户的数据
- 记录所有跨租户访问尝试的审计日志

### 2. 权限提升防护

**措施**：

- 租户管理员不能修改自己的角色为平台管理员
- 租户管理员不能创建平台管理员用户
- 所有角色变更操作需要记录审计日志

### 3. 敏感信息保护

**措施**：

- 密码哈希值不在API响应中返回
- 管理员初始密码只在创建时返回一次
- 建议首次登录后立即修改密码

### 4. 审计日志

**记录内容**：

- 所有租户创建、更新、删除操作
- 所有用户创建、更新、删除操作
- 所有权限验证失败的尝试
- 所有跨租户访问尝试

**日志字段**：

- 操作时间
- 操作者ID和租户ID
- 操作类型
- 目标资源ID
- 操作结果
- 客户端IP和User-Agent

## 性能优化

### 1. 数据库查询优化

**索引策略**：

- `tenants` 表：在 `type`, `status`, `is_deleted` 字段上创建索引
- `users` 表：在 `tenant_id`, `is_active`, `is_deleted` 字段上创建复合索引
- `users` 表：在 `email` 字段上创建唯一索引

**查询优化**：

- 使用分页查询避免大量数据加载
- 在列表查询中只返回必要字段
- 使用数据库层面的软删除过滤

### 2. 缓存策略

**可缓存数据**：

- 租户基本信息（TTL: 5分钟）
- 用户角色信息（TTL: 5分钟）

**缓存失效**：

- 租户或用户更新时，清除相关缓存
- 使用Redis作为缓存存储

### 3. 并发控制

**乐观锁**：

- 使用 `updated_at` 字段实现乐观锁
- 在更新操作中检测并发冲突

## 监控和告警

### 关键指标

1. **API性能指标**
   - 接口响应时间（P50, P95, P99）
   - 接口错误率
   - 接口调用量

2. **权限验证指标**
   - 权限验证失败次数
   - 跨租户访问尝试次数
   - 权限提升尝试次数

3. **业务指标**
   - 租户创建数量
   - 用户创建数量
   - 活跃租户数量

### 告警规则

1. **安全告警**
   - 短时间内多次权限验证失败
   - 跨租户访问尝试
   - 异常的权限提升尝试

2. **性能告警**
   - 接口响应时间超过阈值
   - 接口错误率超过阈值
   - 数据库查询慢查询

3. **业务告警**
   - 租户创建失败率过高
   - 用户创建失败率过高

## 迁移策略

### 向后兼容性

**废弃接口处理**：

- 保留旧接口一段时间（建议3个月）
- 在旧接口响应中添加 `Deprecated` 头
- 在文档中明确标注废弃信息

**客户端迁移指南**：

1. 更新API端点URL
2. 调整请求参数（如添加或移除 tenantId）
3. 更新错误处理逻辑
4. 测试权限控制行为

### 数据迁移

**无需数据迁移**：

- 数据模型保持不变
- 只是接口层面的整合

## Agent Steering规范更新

### 更新多租户访问控制规范

为了确保所有后续开发都遵循统一的权限控制标准，需要更新 `.kiro/steering/multi-tenant-access-control.md` 文件，明确以下内容：

#### 1. 适用范围

**规范适用于所有数据操作接口**，包括但不限于：

- 租户管理接口（创建、查询、更新、删除租户）
- 用户管理接口（创建、查询、更新、删除用户）
- 业务数据接口（所有涉及租户数据的接口）

**例外接口（不需要租户权限验证）**：

- 认证接口：`POST /api/v1/auth/login`, `POST /api/v1/auth/register`, `POST /api/v1/auth/refresh`
- 健康检查接口：`GET /health`, `GET /api/v1/health`
- 公开接口：`GET /api/v1/public/*`
- Swagger文档接口：`GET /swagger/*`

#### 2. 权限控制铁则

**平台管理员（system_admin）权限铁则**：

- ✅ 可以查看所有租户的数据
- ✅ 可以修改所有租户的数据
- ✅ 可以删除所有租户的数据
- ✅ 可以创建新租户
- ✅ 可以启用/禁用任何租户
- ✅ 可以在任意租户下创建用户
- ✅ 可以管理所有租户下的用户

**租户管理员（tenant_admin）权限铁则**：

- ✅ 可以查看自己租户的数据
- ✅ 可以修改自己租户的数据（部分字段受限）
- ✅ 可以管理自己租户下的用户
- ❌ 不能访问其他租户的数据
- ❌ 不能修改其他租户的数据
- ❌ 不能删除其他租户的数据
- ❌ 不能创建新租户
- ❌ 不能启用/禁用租户
- ❌ 不能在其他租户下创建用户

**普通用户（user）权限铁则**：

- ❌ 不能访问管理接口
- ❌ 不能管理租户
- ❌ 不能管理用户

#### 3. 标准实现模式

**模式1：中间件层权限验证**

```go
// 在路由配置中使用中间件
router.Handle("/api/v1/tenants", 
    middleware.RequireSystemAdmin()(tenantHandler.HandleCreate))

router.Handle("/api/v1/users", 
    middleware.RequireTenantAdmin()(userHandler.HandleCreate))
```

**模式2：服务层租户隔离验证**

```go
// 在服务层验证租户访问权限
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    // 1. 查询用户
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 2. 获取当前用户的角色和租户ID
    claims := middleware.GetJWTClaims(ctx)
    
    // 3. 如果是平台管理员，允许访问
    if hasRole(claims, model.RoleSystemAdmin) {
        return user, nil
    }
    
    // 4. 如果是租户管理员，验证租户ID
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

**模式3：字段级权限控制**

```go
// 在服务层验证字段级权限
func (s *TenantService) Update(ctx context.Context, tenantID string, req UpdateTenantRequest) (*model.Tenant, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 如果是租户管理员
    if hasRole(claims, model.RoleTenantAdmin) {
        // 验证租户ID
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

#### 4. 审计日志要求

所有权限验证失败的尝试都必须记录审计日志，包括：

- 操作时间
- 操作者ID和租户ID
- 操作类型和目标资源
- 失败原因
- 客户端IP和User-Agent

```go
// 记录权限验证失败
logger.WarnContext(ctx, "权限验证失败", logger.Fields{
    "event":      "permission_denied",
    "reason":     "尝试访问其他租户的数据",
    "user_id":    claims.Subject,
    "tenant_id":  claims.TenantID,
    "target_id":  targetID,
    "ip":         getClientIP(r),
    "user_agent": r.UserAgent(),
})
```

#### 5. 测试要求

每个数据操作接口都必须测试以下场景：

1. ✅ 平台管理员访问自己租户的资源
2. ✅ 平台管理员访问其他租户的资源
3. ✅ 租户管理员访问自己租户的资源
4. ❌ 租户管理员尝试访问其他租户的资源（应被拒绝）
5. ❌ 普通用户尝试访问管理接口（应被拒绝）

#### 6. 常见错误和解决方案

**错误1：忘记在服务层验证租户ID**

```go
// ❌ 错误示例：只在中间件验证角色，未在服务层验证租户ID
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    return s.repo.FindByID(ctx, userID)  // 危险！租户管理员可能访问其他租户的用户
}

// ✅ 正确示例：在服务层验证租户ID
func (s *UserService) Get(ctx context.Context, userID string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    claims := middleware.GetJWTClaims(ctx)
    if !hasRole(claims, model.RoleSystemAdmin) && user.TenantID != claims.TenantID {
        return nil, errors.NewForbiddenError("权限不足")
    }
    
    return user, nil
}
```

**错误2：信任客户端传入的租户ID**

```go
// ❌ 错误示例：直接使用客户端传入的租户ID
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
    user := &model.User{
        TenantID: req.TenantID,  // 危险！租户管理员可能在其他租户下创建用户
        Email:    req.Email,
    }
    return s.repo.Create(ctx, user)
}

// ✅ 正确示例：验证租户ID
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

**错误3：在列表查询中未过滤租户**

```go
// ❌ 错误示例：返回所有用户
func (s *UserService) List(ctx context.Context, pageNo, pageSize int) ([]*model.User, int64, error) {
    return s.repo.List(ctx, pageNo, pageSize)  // 危险！租户管理员可能看到其他租户的用户
}

// ✅ 正确示例：根据角色过滤租户
func (s *UserService) List(ctx context.Context, pageNo, pageSize int) ([]*model.User, int64, error) {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能查看自己租户的用户
    var tenantID *string
    if !hasRole(claims, model.RoleSystemAdmin) {
        tenantID = &claims.TenantID
    }
    
    return s.repo.ListByTenant(ctx, tenantID, pageNo, pageSize)
}
```

## 总结

本设计通过合并重复接口、统一权限控制、优化API结构，实现了以下目标：

1. **简化API设计**：从4组接口减少到2组接口
2. **统一权限控制**：在所有接口中实施一致的权限验证逻辑
3. **提升可维护性**：减少代码重复，降低维护成本
4. **增强安全性**：严格执行租户隔离和权限控制
5. **保持灵活性**：通过角色和上下文实现差异化访问

关键设计原则是**权限隔离铁则**：平台管理员可以访问所有数据，租户管理员只能访问自己租户的数据。这一原则贯穿整个设计，确保了系统的安全性和数据隔离。
