# 平台管理员租户系统设计文档

## 概述

本文档描述了多租户系统中平台管理员租户功能的详细设计。该系统通过引入系统级租户和平台管理员角色，解决了多租户系统初始化时的循环依赖问题，并提供了完整的租户生命周期管理能力。

### 核心目标

1. 实现系统自举（Bootstrap）：在系统首次启动时自动创建平台租户和平台管理员
2. 支持租户类型区分：区分平台租户（system）和业务租户（tenant）
3. 实现分级权限管理：支持平台管理员、租户管理员和普通用户三级角色
4. 提供租户生命周期管理：创建、启用、禁用、删除租户
5. 自动化租户初始化：创建租户时自动生成租户管理员账户

### 设计原则

- **安全优先**：所有操作都需要严格的权限验证
- **租户隔离**：确保业务租户之间的数据完全隔离
- **可审计性**：记录所有关键操作的审计日志
- **向后兼容**：与现有的认证和授权系统无缝集成
- **配置灵活**：支持通过环境变量自定义初始化参数

## 架构设计

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      应用层 (API Layer)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 租户管理API  │  │ 用户管理API  │  │ 认证授权API  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    中间件层 (Middleware)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ JWT认证中间件│  │ RBAC权限中间件│  │ 租户上下文   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    服务层 (Service Layer)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 租户服务     │  │ 用户服务     │  │ 初始化服务   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                  数据访问层 (Repository Layer)               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 租户仓储     │  │ 用户仓储     │  │ 审计仓储     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    数据层 (Database Layer)                   │
│              PostgreSQL (Tenants, Users, Audit)              │
└─────────────────────────────────────────────────────────────┘
```

### 角色层次结构

```
┌─────────────────────────────────────────────────────────────┐
│                      平台管理员                              │
│                   (system_admin)                             │
│  - 所属租户：平台租户 (type: system)                         │
│  - 权限范围：所有租户                                        │
│  - 可执行操作：                                              │
│    * 创建/删除/启用/禁用业务租户                             │
│    * 查看所有租户列表                                        │
│    * 跨租户访问数据                                          │
└─────────────────────────────────────────────────────────────┘
                            │
                            ├─────────────────────────────────┐
                            │                                 │
┌───────────────────────────▼─────┐   ┌───────────────────────▼─────┐
│        租户管理员                │   │        租户管理员            │
│      (tenant_admin)              │   │      (tenant_admin)          │
│  - 所属租户：业务租户 A          │   │  - 所属租户：业务租户 B      │
│  - 权限范围：本租户内            │   │  - 权限范围：本租户内        │
│  - 可执行操作：                  │   │  - 可执行操作：              │
│    * 创建/删除/启用/禁用用户     │   │    * 创建/删除/启用/禁用用户 │
│    * 查看本租户用户列表          │   │    * 查看本租户用户列表      │
└──────────────────────────────────┘   └──────────────────────────────┘
                │                                  │
                ├──────────────┐                   ├──────────────┐
                │              │                   │              │
┌───────────────▼───┐  ┌───────▼───────┐  ┌───────▼───────┐  ┌──▼──────────┐
│   普通用户        │  │   普通用户    │  │   普通用户    │  │  普通用户   │
│    (user)         │  │    (user)     │  │    (user)     │  │   (user)    │
│  - 租户 A         │  │  - 租户 A     │  │  - 租户 B     │  │  - 租户 B   │
│  - 基本业务权限   │  │  - 基本业务权限│  │  - 基本业务权限│  │ - 基本业务权限│
└───────────────────┘  └───────────────┘  └───────────────┘  └─────────────┘
```

## 数据模型设计

### 数据库表结构

#### 1. Tenants 表（租户表）

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    domain VARCHAR(255),
    type VARCHAR(32) NOT NULL DEFAULT 'tenant',  -- 新增字段
    metadata JSONB,
    status BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    is_deleted BOOLEAN DEFAULT false,
    
    -- 索引
    CONSTRAINT idx_tenant_type CHECK (type IN ('system', 'tenant')),
    CONSTRAINT unique_system_tenant UNIQUE (type) WHERE type = 'system'
);

CREATE INDEX idx_tenants_type ON tenants(type);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_domain ON tenants(domain);
```

**字段说明：**

- `type`: 租户类型，"system" 表示平台租户，"tenant" 表示业务租户
- `unique_system_tenant`: 确保只能有一个平台租户存在

#### 2. Users 表（用户表）

现有表结构已经支持所需字段，无需修改：

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    email VARCHAR(320) NOT NULL,
    email_verified BOOLEAN DEFAULT false,
    phone VARCHAR(20),
    password_hash TEXT NOT NULL,
    display_name VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_admin BOOLEAN DEFAULT false,
    roles JSONB,  -- 存储角色数组，如 ["system_admin"], ["tenant_admin"], ["user"]
    meta JSONB,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    is_deleted BOOLEAN DEFAULT false,
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMP,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE (tenant_id, email)
);
```

**角色定义：**

- `["system_admin"]`: 平台管理员角色
- `["tenant_admin"]`: 租户管理员角色
- `["user"]`: 普通用户角色

### Go 数据模型

#### Tenant 模型扩展

```go
type Tenant struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Name      string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
    Domain    string         `gorm:"type:varchar(255)" json:"domain"`
    Type      string         `gorm:"type:varchar(32);not null;default:'tenant';index" json:"type"` // 新增
    Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    Status    bool           `gorm:"default:true" json:"status"`
    CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"createdBy"`
    IsDeleted bool           `gorm:"default:false" json:"isDeleted"`
}

// 租户类型常量
const (
    TenantTypeSystem   = "system"   // 平台租户
    TenantTypeBusiness = "tenant"   // 业务租户
)
```

#### User 模型（无需修改）

现有的 User 模型已经支持所需的所有字段，包括 `Roles` 和 `IsAdmin`。

#### 角色常量定义

```go
// 角色常量
const (
    RoleSystemAdmin = "system_admin"  // 平台管理员
    RoleTenantAdmin = "tenant_admin"  // 租户管理员
    RoleUser        = "user"          // 普通用户
)
```

## 组件和接口设计

### 1. 系统初始化服务 (Bootstrap Service)

#### 接口定义

```go
type BootstrapService interface {
    // Initialize 初始化平台租户和平台管理员
    Initialize(ctx context.Context) error
    
    // IsInitialized 检查系统是否已初始化
    IsInitialized(ctx context.Context) (bool, error)
}
```

#### 实现逻辑

```go
type bootstrapService struct {
    tenantRepo repository.TenantRepository
    userRepo   repository.UserRepository
    config     *BootstrapConfig
}

type BootstrapConfig struct {
    AdminEmail       string // 从环境变量 PLATFORM_ADMIN_EMAIL 读取
    AdminPassword    string // 从环境变量 PLATFORM_ADMIN_PASSWORD 读取
    AdminDisplayName string // 从环境变量 PLATFORM_ADMIN_NAME 读取
}

func (s *bootstrapService) Initialize(ctx context.Context) error {
    // 1. 检查是否已存在平台租户
    // 2. 如果不存在，创建平台租户
    // 3. 创建平台管理员用户
    // 4. 记录初始化日志
}
```

**初始化流程：**

1. 检查是否存在 `type = 'system'` 的租户
2. 如果不存在，创建平台租户：
   - Name: "Platform"
   - Domain: "system.local"
   - Type: "system"
   - Status: true
3. 创建平台管理员用户：
   - Email: 从配置读取（默认 "<admin@system.local>"）
   - Password: 从配置读取（如未设置则随机生成）
   - DisplayName: 从配置读取（默认 "Platform Admin"）
   - Roles: ["system_admin"]
   - IsAdmin: true
4. 记录初始化信息到日志

### 2. 租户服务扩展 (Tenant Service)

#### 接口扩展

```go
type TenantService interface {
    // 现有方法
    Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error)
    Get(ctx context.Context, id string) (*model.Tenant, error)
    Update(ctx context.Context, id string, req UpdateTenantRequest) (*model.Tenant, error)
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, page, pageSize int) ([]*model.Tenant, int64, error)
    GetByDomain(ctx context.Context, domain string) (*model.Tenant, error)
    
    // 新增方法
    CreateWithAdmin(ctx context.Context, req CreateTenantWithAdminRequest) (*CreateTenantWithAdminResponse, error)
    GetByType(ctx context.Context, tenantType string) ([]*model.Tenant, error)
    EnableTenant(ctx context.Context, id string) error
    DisableTenant(ctx context.Context, id string) error
}
```

#### 请求/响应结构

```go
// CreateTenantRequest 扩展
type CreateTenantRequest struct {
    Name      string                 `json:"name" validate:"required,min=1,max=255"`
    Domain    string                 `json:"domain" validate:"omitempty,max=255"`
    Type      string                 `json:"type" validate:"omitempty,oneof=system tenant"` // 新增
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedBy *string                `json:"createdBy"`
}

// CreateTenantWithAdminRequest 创建租户并自动生成管理员
type CreateTenantWithAdminRequest struct {
    TenantName        string                 `json:"tenantName" validate:"required"`
    TenantDomain      string                 `json:"tenantDomain" validate:"required"`
    TenantMetadata    map[string]interface{} `json:"tenantMetadata"`
    AdminEmail        string                 `json:"adminEmail" validate:"omitempty,email"`
    AdminDisplayName  string                 `json:"adminDisplayName"`
}

// CreateTenantWithAdminResponse 创建租户响应
type CreateTenantWithAdminResponse struct {
    Tenant        *model.Tenant `json:"tenant"`
    AdminUser     *model.User   `json:"adminUser"`
    AdminPassword string        `json:"adminPassword"` // 初始密码
}
```

#### 核心业务逻辑

**CreateWithAdmin 方法：**

1. 验证请求参数
2. 创建业务租户（type = "tenant"）
3. 生成租户管理员账户：
   - Email: 使用请求中的 adminEmail，如果为空则使用 "admin@{tenant_domain}"
   - Password: 随机生成 16 位强密码
   - Roles: ["tenant_admin"]
   - IsAdmin: true
4. 返回租户信息和管理员初始密码
5. 记录审计日志

### 3. 权限验证中间件 (RBAC Middleware)

#### 中间件接口

```go
// RequireSystemAdmin 要求平台管理员权限
func RequireSystemAdmin() gin.HandlerFunc

// RequireTenantAdmin 要求租户管理员或平台管理员权限
func RequireTenantAdmin() gin.HandlerFunc

// RequireTenantAccess 要求访问特定租户的权限
func RequireTenantAccess() gin.HandlerFunc
```

#### 权限验证逻辑

**RequireSystemAdmin:**

```go
func RequireSystemAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 JWT Token 中获取用户角色
        // 2. 检查角色是否包含 "system_admin"
        // 3. 如果不包含，返回 403 Forbidden
        // 4. 如果包含，继续处理请求
    }
}
```

**RequireTenantAdmin:**

```go
func RequireTenantAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 JWT Token 中获取用户角色
        // 2. 检查角色是否包含 "system_admin" 或 "tenant_admin"
        // 3. 如果都不包含，返回 403 Forbidden
        // 4. 如果包含，继续处理请求
    }
}
```

**RequireTenantAccess:**

```go
func RequireTenantAccess() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 JWT Token 中获取用户的 tenant_id 和角色
        // 2. 从请求路径或参数中获取目标 tenant_id
        // 3. 如果用户是 system_admin，允许访问
        // 4. 如果用户的 tenant_id 与目标 tenant_id 匹配，允许访问
        // 5. 否则返回 403 Forbidden
    }
}
```

### 4. JWT Token 结构扩展

#### Token Claims 结构

```go
type JWTClaims struct {
    UserID    string   `json:"userId"`
    TenantID  string   `json:"tenantId"`
    Email     string   `json:"email"`
    Roles     []string `json:"roles"`      // 新增：用户角色列表
    IsAdmin   bool     `json:"isAdmin"`    // 现有字段
    jwt.RegisteredClaims
}
```

#### Token 生成逻辑

在用户登录成功后，生成 JWT Token 时需要包含：

- `userId`: 用户 ID
- `tenantId`: 用户所属租户 ID
- `email`: 用户邮箱
- `roles`: 用户角色数组（从数据库 User.Roles 字段读取）
- `isAdmin`: 是否为管理员标记

**示例 Token Payload:**

```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "tenantId": "660e8400-e29b-41d4-a716-446655440000",
  "email": "admin@system.local",
  "roles": ["system_admin"],
  "isAdmin": true,
  "exp": 1735689600,
  "iat": 1735603200
}
```

### 5. 数据库迁移服务

#### 迁移接口

```go
type Migration interface {
    // Up 执行迁移
    Up() error
    
    // Down 回滚迁移
    Down() error
    
    // Version 返回迁移版本号
    Version() string
}
```

#### 平台租户迁移

```go
type PlatformTenantMigration struct {
    db *gorm.DB
}

func (m *PlatformTenantMigration) Up() error {
    // 1. 添加 type 字段到 tenants 表
    // 2. 创建 type 字段索引
    // 3. 添加唯一约束确保只有一个 system 类型租户
    // 4. 更新现有租户的 type 为 'tenant'
}

func (m *PlatformTenantMigration) Down() error {
    // 1. 删除唯一约束
    // 2. 删除 type 字段索引
    // 3. 删除 type 字段
}
```

**SQL 迁移脚本：**

```sql
-- Up Migration
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS type VARCHAR(32) NOT NULL DEFAULT 'tenant';
CREATE INDEX IF NOT EXISTS idx_tenants_type ON tenants(type);
ALTER TABLE tenants ADD CONSTRAINT unique_system_tenant UNIQUE (type) WHERE type = 'system';
ALTER TABLE tenants ADD CONSTRAINT check_tenant_type CHECK (type IN ('system', 'tenant'));

-- Down Migration
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS check_tenant_type;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS unique_system_tenant;
DROP INDEX IF EXISTS idx_tenants_type;
ALTER TABLE tenants DROP COLUMN IF EXISTS type;
```

## API 接口设计

### 1. 平台管理员 API

#### 1.1 创建租户（带管理员）

**端点:** `POST /api/v1/platform/tenants`

**权限:** 需要 `system_admin` 角色

**请求体:**

```json
{
  "tenantName": "Acme Corporation",
  "tenantDomain": "acme.example.com",
  "tenantMetadata": {
    "industry": "Technology",
    "size": "Medium"
  },
  "adminEmail": "admin@acme.example.com",
  "adminDisplayName": "Acme Admin"
}
```

**响应:**

```json
{
  "code": 200,
  "message": "租户创建成功",
  "data": {
    "tenant": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Acme Corporation",
      "domain": "acme.example.com",
      "type": "tenant",
      "status": true,
      "createdAt": "2025-01-20T10:00:00Z"
    },
    "adminUser": {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "email": "admin@acme.example.com",
      "displayName": "Acme Admin",
      "roles": ["tenant_admin"],
      "isAdmin": true
    },
    "adminPassword": "Xy9#mK2$pL5@qR8!"
  }
}
```

#### 1.2 列出所有租户

**端点:** `GET /api/v1/platform/tenants`

**权限:** 需要 `system_admin` 角色

**查询参数:**

- `page`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10，最大 100）
- `type`: 租户类型过滤（可选，"system" 或 "tenant"）

**响应:**

```json
{
  "code": 200,
  "message": "获取租户列表成功",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Platform",
        "domain": "system.local",
        "type": "system",
        "status": true
      },
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "Acme Corporation",
        "domain": "acme.example.com",
        "type": "tenant",
        "status": true
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

#### 1.3 启用/禁用租户

**端点:** `PATCH /api/v1/platform/tenants/{tenantId}/status`

**权限:** 需要 `system_admin` 角色

**请求体:**

```json
{
  "status": true
}
```

**响应:**

```json
{
  "code": 200,
  "message": "租户状态更新成功",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "Acme Corporation",
    "status": true
  }
}
```

#### 1.4 删除租户

**端点:** `DELETE /api/v1/platform/tenants/{tenantId}`

**权限:** 需要 `system_admin` 角色

**响应:**

```json
{
  "code": 200,
  "message": "租户删除成功",
  "data": null
}
```

**注意:**

- 不允许删除平台租户（type = "system"）
- 执行软删除，设置 `is_deleted = true`

### 2. 租户管理员 API

#### 2.1 创建用户

**端点:** `POST /api/v1/tenants/{tenantId}/users`

**权限:** 需要 `tenant_admin` 或 `system_admin` 角色

**请求体:**

```json
{
  "email": "user@acme.example.com",
  "password": "SecurePassword123!",
  "displayName": "John Doe",
  "phone": "+1234567890",
  "roles": ["user"]
}
```

**响应:**

```json
{
  "code": 200,
  "message": "用户创建成功",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "tenantId": "660e8400-e29b-41d4-a716-446655440001",
    "email": "user@acme.example.com",
    "displayName": "John Doe",
    "roles": ["user"],
    "isActive": true,
    "createdAt": "2025-01-20T10:30:00Z"
  }
}
```

#### 2.2 列出租户用户

**端点:** `GET /api/v1/tenants/{tenantId}/users`

**权限:** 需要 `tenant_admin` 或 `system_admin` 角色

**查询参数:**

- `page`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10，最大 100）

**响应:**

```json
{
  "code": 200,
  "message": "获取用户列表成功",
  "data": {
    "data": [
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "email": "admin@acme.example.com",
        "displayName": "Acme Admin",
        "roles": ["tenant_admin"],
        "isAdmin": true,
        "isActive": true
      },
      {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "email": "user@acme.example.com",
        "displayName": "John Doe",
        "roles": ["user"],
        "isAdmin": false,
        "isActive": true
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

#### 2.3 禁用/启用用户

**端点:** `PATCH /api/v1/tenants/{tenantId}/users/{userId}/status`

**权限:** 需要 `tenant_admin` 或 `system_admin` 角色

**请求体:**

```json
{
  "isActive": false
}
```

**响应:**

```json
{
  "code": 200,
  "message": "用户状态更新成功",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "email": "user@acme.example.com",
    "isActive": false
  }
}
```

**副作用:** 禁用用户时，系统会自动撤销该用户的所有有效 Refresh Token。

### 3. 认证 API 扩展

#### 3.1 登录

**端点:** `POST /api/v1/auth/login`

**请求体:**

```json
{
  "email": "admin@system.local",
  "password": "AdminPassword123!"
}
```

**响应:**

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "550e8400-e29b-41d4-a716-446655440000",
    "expiresIn": 3600,
    "user": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "tenantId": "550e8400-e29b-41d4-a716-446655440000",
      "email": "admin@system.local",
      "displayName": "Platform Admin",
      "roles": ["system_admin"],
      "isAdmin": true
    }
  }
}
```

**登录验证逻辑扩展:**

1. 验证邮箱和密码
2. 检查用户所属租户的状态（`tenant.status`）
3. 如果租户被禁用，拒绝登录并返回错误
4. 检查用户的激活状态（`user.is_active`）
5. 如果用户被禁用，拒绝登录并返回错误
6. 生成包含角色信息的 JWT Token
7. 创建 Refresh Token
8. 更新最后登录时间
9. 记录登录审计日志

## 错误处理

### 错误代码定义

```go
const (
    // 租户相关错误
    ErrTenantNotFound        = "TENANT_NOT_FOUND"
    ErrTenantDisabled        = "TENANT_DISABLED"
    ErrTenantAlreadyExists   = "TENANT_ALREADY_EXISTS"
    ErrCannotDeleteSystemTenant = "CANNOT_DELETE_SYSTEM_TENANT"
    ErrSystemTenantExists    = "SYSTEM_TENANT_EXISTS"
    
    // 用户相关错误
    ErrUserNotFound          = "USER_NOT_FOUND"
    ErrUserDisabled          = "USER_DISABLED"
    ErrUserAlreadyExists     = "USER_ALREADY_EXISTS"
    
    // 权限相关错误
    ErrInsufficientPermission = "INSUFFICIENT_PERMISSION"
    ErrUnauthorized          = "UNAUTHORIZED"
    ErrForbidden             = "FORBIDDEN"
    
    // 初始化相关错误
    ErrSystemAlreadyInitialized = "SYSTEM_ALREADY_INITIALIZED"
    ErrInitializationFailed     = "INITIALIZATION_FAILED"
)
```

### 错误响应格式

```json
{
  "code": 403,
  "message": "权限不足：需要平台管理员权限",
  "data": null
}
```

### 常见错误场景

1. **租户被禁用时登录:**
   - HTTP 状态码: 403
   - 错误消息: "租户已被禁用，无法登录"

2. **非平台管理员尝试创建租户:**
   - HTTP 状态码: 403
   - 错误消息: "权限不足：需要平台管理员权限"

3. **租户管理员尝试访问其他租户数据:**
   - HTTP 状态码: 403
   - 错误消息: "权限不足：无法访问其他租户的数据"

4. **尝试删除平台租户:**
   - HTTP 状态码: 400
   - 错误消息: "不允许删除平台租户"

5. **尝试创建第二个平台租户:**
   - HTTP 状态码: 400
   - 错误消息: "平台租户已存在，不能创建多个"

## 测试策略

### 单元测试

#### 1. 服务层测试

**BootstrapService 测试:**

- 测试首次初始化成功创建平台租户和管理员
- 测试重复初始化不会创建重复数据
- 测试配置参数正确应用
- 测试初始化失败时的回滚

**TenantService 测试:**

- 测试创建业务租户
- 测试创建租户时自动生成管理员
- 测试不允许创建多个平台租户
- 测试启用/禁用租户
- 测试删除租户（软删除）
- 测试不允许删除平台租户

**UserService 测试:**

- 测试在租户内创建用户
- 测试邮箱唯一性验证（租户级别）
- 测试禁用用户时撤销 Token

#### 2. 中间件测试

**RBAC 中间件测试:**

- 测试 RequireSystemAdmin 正确验证平台管理员权限
- 测试 RequireTenantAdmin 正确验证租户管理员权限
- 测试 RequireTenantAccess 正确验证租户访问权限
- 测试权限不足时返回 403
- 测试未认证时返回 401

#### 3. 仓储层测试

**TenantRepository 测试:**

- 测试按类型查询租户
- 测试唯一约束（只能有一个平台租户）
- 测试软删除

**UserRepository 测试:**

- 测试租户隔离查询
- 测试邮箱唯一性（租户级别）

### 集成测试

#### 1. 系统初始化流程测试

```go
func TestSystemBootstrap(t *testing.T) {
    // 1. 启动应用
    // 2. 验证平台租户已创建
    // 3. 验证平台管理员已创建
    // 4. 验证可以使用平台管理员登录
}
```

#### 2. 租户生命周期测试

```go
func TestTenantLifecycle(t *testing.T) {
    // 1. 使用平台管理员登录
    // 2. 创建业务租户
    // 3. 验证租户管理员已自动创建
    // 4. 使用租户管理员登录
    // 5. 创建普通用户
    // 6. 禁用租户
    // 7. 验证租户下所有用户无法登录
    // 8. 启用租户
    // 9. 验证用户可以正常登录
    // 10. 删除租户
}
```

#### 3. 权限验证测试

```go
func TestPermissionValidation(t *testing.T) {
    // 1. 创建两个业务租户 A 和 B
    // 2. 在租户 A 中创建租户管理员和普通用户
    // 3. 在租户 B 中创建租户管理员和普通用户
    // 4. 验证租户 A 的管理员无法访问租户 B 的数据
    // 5. 验证平台管理员可以访问所有租户的数据
    // 6. 验证普通用户只能访问自己的数据
}
```

#### 4. 审计日志测试

```go
func TestAuditLogging(t *testing.T) {
    // 1. 执行各种操作（创建租户、创建用户、禁用租户等）
    // 2. 验证审计日志正确记录
    // 3. 验证日志包含必要的信息（操作者、操作类型、时间戳等）
}
```

### 端到端测试

#### 场景 1: 新系统部署

1. 部署应用到全新环境
2. 验证系统自动初始化
3. 使用平台管理员登录
4. 创建第一个业务租户
5. 使用租户管理员登录
6. 创建普通用户
7. 使用普通用户登录并访问业务功能

#### 场景 2: 租户管理

1. 平台管理员创建多个租户
2. 验证每个租户的管理员可以独立管理自己的用户
3. 验证租户之间的数据隔离
4. 禁用某个租户
5. 验证该租户下的用户无法登录
6. 重新启用租户
7. 验证用户可以正常登录

#### 场景 3: 权限边界测试

1. 尝试使用租户管理员访问平台管理 API
2. 验证返回 403 错误
3. 尝试使用租户 A 的管理员访问租户 B 的数据
4. 验证返回 403 错误
5. 使用平台管理员访问所有租户的数据
6. 验证都能成功访问

## 安全考虑

### 1. 密码安全

- **平台管理员初始密码:**
  - 如果通过环境变量提供，必须满足强密码要求（至少 12 字符，包含大小写字母、数字和特殊字符）
  - 如果未提供，系统自动生成 16 位随机强密码
  - 初始密码记录到日志中（仅在初始化时）
  - 建议首次登录后立即修改密码

- **租户管理员初始密码:**
  - 自动生成 16 位随机强密码
  - 包含大小写字母、数字和特殊字符
  - 通过 API 响应返回给平台管理员
  - 建议通过安全渠道传递给租户管理员

### 2. Token 安全

- **JWT Token:**
  - 使用 HS256 算法签名
  - 包含用户角色信息，避免每次请求查询数据库
  - 设置合理的过期时间（建议 1 小时）
  - 不在 Token 中存储敏感信息

- **Refresh Token:**
  - 存储 SHA256 哈希值，不存储明文
  - 支持撤销机制
  - 禁用用户时自动撤销所有 Token

### 3. 权限控制

- **最小权限原则:**
  - 每个角色只拥有必要的权限
  - 租户管理员无法访问其他租户的数据
  - 普通用户只能访问自己的数据

- **租户隔离:**
  - 所有数据查询都必须包含租户 ID 过滤
  - 在仓储层强制执行租户隔离
  - 平台管理员可以跨租户访问，但需要明确指定目标租户

### 4. 审计和监控

- **审计日志:**
  - 记录所有敏感操作（创建/删除租户、创建/禁用用户等）
  - 包含操作者、操作类型、目标对象、时间戳
  - 日志不可修改，只能追加

- **异常监控:**
  - 监控失败的权限验证尝试
  - 监控异常的跨租户访问尝试
  - 监控平台管理员的所有操作

### 5. 数据保护

- **软删除:**
  - 租户和用户删除使用软删除
  - 保留数据用于审计和恢复
  - 定期清理过期的软删除数据

- **敏感数据:**
  - 密码使用 bcrypt 哈希存储
  - Refresh Token 使用 SHA256 哈希存储
  - 不在日志中记录密码和 Token 明文

## 配置管理

### 环境变量

```bash
# 平台管理员配置
PLATFORM_ADMIN_EMAIL=admin@system.local          # 平台管理员邮箱
PLATFORM_ADMIN_PASSWORD=                         # 平台管理员初始密码（留空则自动生成）
PLATFORM_ADMIN_NAME=Platform Admin               # 平台管理员显示名称

# 平台租户配置
PLATFORM_TENANT_NAME=Platform                    # 平台租户名称
PLATFORM_TENANT_DOMAIN=system.local              # 平台租户域名

# 密码策略
MIN_PASSWORD_LENGTH=8                            # 最小密码长度
REQUIRE_PASSWORD_COMPLEXITY=true                 # 是否要求密码复杂度
AUTO_GENERATED_PASSWORD_LENGTH=16                # 自动生成密码的长度

# Token 配置
JWT_SECRET=your-secret-key                       # JWT 签名密钥
JWT_EXPIRATION=3600                              # JWT 过期时间（秒）
REFRESH_TOKEN_EXPIRATION=2592000                 # Refresh Token 过期时间（秒，默认 30 天）
```

### 配置加载

```go
type Config struct {
    // 平台管理员配置
    PlatformAdmin struct {
        Email       string
        Password    string
        DisplayName string
    }
    
    // 平台租户配置
    PlatformTenant struct {
        Name   string
        Domain string
    }
    
    // 密码策略
    PasswordPolicy struct {
        MinLength              int
        RequireComplexity      bool
        AutoGeneratedLength    int
    }
    
    // Token 配置
    JWT struct {
        Secret              string
        Expiration          int
        RefreshExpiration   int
    }
}

func LoadConfig() (*Config, error) {
    // 从环境变量加载配置
    // 设置默认值
    // 验证配置有效性
}
```

### 配置验证

在应用启动时验证配置：

1. 验证 JWT_SECRET 已设置且足够长（至少 32 字符）
2. 验证邮箱格式有效
3. 如果提供了初始密码，验证密码强度
4. 验证过期时间配置合理

## 部署和运维

### 初始化流程

#### 首次部署

1. **准备环境变量:**

   ```bash
   export PLATFORM_ADMIN_EMAIL=admin@yourcompany.com
   export PLATFORM_ADMIN_PASSWORD=YourSecurePassword123!
   export PLATFORM_ADMIN_NAME="System Administrator"
   export JWT_SECRET=$(openssl rand -base64 32)
   ```

2. **运行数据库迁移:**

   ```bash
   ./server migrate up
   ```

3. **启动应用:**

   ```bash
   ./server
   ```

4. **验证初始化:**
   - 检查日志确认平台租户和管理员已创建
   - 使用平台管理员账户登录
   - 创建第一个业务租户

#### 升级部署

1. **备份数据库**
2. **运行新的迁移脚本**
3. **重启应用**
4. **验证功能正常**

### 监控指标

#### 关键指标

1. **租户指标:**
   - 活跃租户数量
   - 禁用租户数量
   - 新创建租户数量（按时间段）

2. **用户指标:**
   - 总用户数量
   - 活跃用户数量
   - 各租户的用户数量分布

3. **认证指标:**
   - 登录成功率
   - 登录失败次数
   - Token 刷新频率

4. **权限指标:**
   - 权限验证失败次数
   - 跨租户访问尝试次数

#### 告警规则

1. **安全告警:**
   - 短时间内大量登录失败
   - 异常的跨租户访问尝试
   - 平台管理员账户的异常操作

2. **业务告警:**
   - 租户创建失败
   - 用户创建失败率过高
   - 数据库连接失败

### 日志管理

#### 日志级别

- **INFO:** 正常操作（登录、创建租户、创建用户等）
- **WARN:** 异常但可恢复的情况（登录失败、权限不足等）
- **ERROR:** 错误情况（数据库错误、系统错误等）

#### 日志内容

```json
{
  "level": "INFO",
  "timestamp": "2025-01-20T10:00:00Z",
  "message": "Tenant created successfully",
  "context": {
    "tenantId": "550e8400-e29b-41d4-a716-446655440000",
    "tenantName": "Acme Corporation",
    "createdBy": "660e8400-e29b-41d4-a716-446655440000",
    "createdByRole": "system_admin"
  }
}
```

### 备份和恢复

#### 备份策略

1. **数据库备份:**
   - 每日全量备份
   - 每小时增量备份
   - 保留 30 天的备份数据

2. **配置备份:**
   - 备份环境变量配置
   - 备份应用配置文件

#### 恢复流程

1. 停止应用
2. 恢复数据库
3. 恢复配置文件
4. 启动应用
5. 验证数据完整性

## 实施计划

### 阶段 1: 数据模型和迁移（优先级：高）

**目标:** 扩展数据库表结构以支持租户类型

**任务:**

1. 创建数据库迁移脚本
2. 添加 `type` 字段到 `tenants` 表
3. 添加索引和约束
4. 更新 Go 数据模型
5. 测试迁移脚本

**预计时间:** 1-2 天

### 阶段 2: 系统初始化服务（优先级：高）

**目标:** 实现系统自举功能

**任务:**

1. 创建 BootstrapService 接口和实现
2. 实现配置加载逻辑
3. 实现平台租户和管理员创建逻辑
4. 集成到应用启动流程
5. 编写单元测试

**预计时间:** 2-3 天

### 阶段 3: 租户服务扩展（优先级：高）

**目标:** 支持租户类型和自动创建管理员

**任务:**

1. 扩展 TenantService 接口
2. 实现 CreateWithAdmin 方法
3. 实现租户类型过滤
4. 实现启用/禁用租户
5. 添加平台租户保护逻辑
6. 编写单元测试

**预计时间:** 2-3 天

### 阶段 4: 权限验证中间件（优先级：高）

**目标:** 实现基于角色的权限控制

**任务:**

1. 创建 RBAC 中间件
2. 实现 RequireSystemAdmin
3. 实现 RequireTenantAdmin
4. 实现 RequireTenantAccess
5. 编写中间件测试

**预计时间:** 2 天

### 阶段 5: JWT Token 扩展（优先级：高）

**目标:** 在 Token 中包含角色信息

**任务:**

1. 扩展 JWTClaims 结构
2. 更新 Token 生成逻辑
3. 更新 Token 验证逻辑
4. 更新登录流程
5. 编写测试

**预计时间:** 1-2 天

### 阶段 6: API 接口实现（优先级：中）

**目标:** 实现平台管理和租户管理 API

**任务:**

1. 创建平台管理 API Handler
2. 创建租户管理 API Handler
3. 配置路由和中间件
4. 实现请求验证
5. 编写 API 测试

**预计时间:** 3-4 天

### 阶段 7: 审计日志（优先级：中）

**目标:** 记录所有关键操作

**任务:**

1. 扩展审计日志服务
2. 在关键操作点添加日志记录
3. 实现日志查询接口
4. 编写测试

**预计时间:** 2 天

### 阶段 8: 集成测试和文档（优先级：中）

**目标:** 确保系统整体功能正常

**任务:**

1. 编写集成测试
2. 编写端到端测试
3. 更新 API 文档
4. 编写部署文档
5. 编写运维手册

**预计时间:** 3-4 天

### 总预计时间

**开发时间:** 16-22 天
**测试和文档:** 3-4 天
**总计:** 19-26 天

### 风险和依赖

**风险:**

1. 数据库迁移可能影响现有数据
2. 权限验证逻辑可能影响现有 API
3. Token 结构变化可能影响现有客户端

**缓解措施:**

1. 充分测试迁移脚本，提供回滚方案
2. 保持向后兼容，逐步迁移现有 API
3. 提供 Token 版本控制，支持平滑升级

**依赖:**

1. 需要 PostgreSQL 数据库支持
2. 需要现有的认证和授权系统
3. 需要 GORM 和 Gin 框架

## 附录

### A. 密码生成算法

```go
func GenerateSecurePassword(length int) (string, error) {
    const (
        lowercase = "abcdefghijklmnopqrstuvwxyz"
        uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        digits    = "0123456789"
        special   = "!@#$%^&*"
    )
    
    all := lowercase + uppercase + digits + special
    
    // 确保至少包含每种字符类型
    password := []byte{
        lowercase[rand.Intn(len(lowercase))],
        uppercase[rand.Intn(len(uppercase))],
        digits[rand.Intn(len(digits))],
        special[rand.Intn(len(special))],
    }
    
    // 填充剩余长度
    for i := 4; i < length; i++ {
        password = append(password, all[rand.Intn(len(all))])
    }
    
    // 随机打乱
    rand.Shuffle(len(password), func(i, j int) {
        password[i], password[j] = password[j], password[i]
    })
    
    return string(password), nil
}
```

### B. 角色权限矩阵

| 操作 | system_admin | tenant_admin | user |
|------|--------------|--------------|------|
| 创建租户 | ✓ | ✗ | ✗ |
| 查看所有租户 | ✓ | ✗ | ✗ |
| 启用/禁用租户 | ✓ | ✗ | ✗ |
| 删除租户 | ✓ | ✗ | ✗ |
| 创建用户（本租户） | ✓ | ✓ | ✗ |
| 查看用户（本租户） | ✓ | ✓ | ✗ |
| 启用/禁用用户（本租户） | ✓ | ✓ | ✗ |
| 删除用户（本租户） | ✓ | ✓ | ✗ |
| 跨租户访问数据 | ✓ | ✗ | ✗ |
| 访问业务功能 | ✓ | ✓ | ✓ |

### C. 数据库索引策略

```sql
-- Tenants 表索引
CREATE INDEX idx_tenants_type ON tenants(type);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_domain ON tenants(domain);
CREATE INDEX idx_tenants_created_at ON tenants(created_at);

-- Users 表索引
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_roles ON users USING GIN(roles);
CREATE UNIQUE INDEX idx_users_tenant_email ON users(tenant_id, email) WHERE is_deleted = false;

-- Auth Audit 表索引
CREATE INDEX idx_auth_audit_tenant_id ON auth_audit(tenant_id);
CREATE INDEX idx_auth_audit_user_id ON auth_audit(user_id);
CREATE INDEX idx_auth_audit_event ON auth_audit(event);
CREATE INDEX idx_auth_audit_created_at ON auth_audit(created_at);
```

### D. API 路由规划

```
# 平台管理 API（需要 system_admin 权限）
POST   /api/v1/platform/tenants                    # 创建租户（带管理员）
GET    /api/v1/platform/tenants                    # 列出所有租户
GET    /api/v1/platform/tenants/:id                # 获取租户详情
PATCH  /api/v1/platform/tenants/:id/status         # 启用/禁用租户
DELETE /api/v1/platform/tenants/:id                # 删除租户

# 租户管理 API（需要 tenant_admin 或 system_admin 权限）
POST   /api/v1/tenants/:tenantId/users             # 创建用户
GET    /api/v1/tenants/:tenantId/users             # 列出租户用户
GET    /api/v1/tenants/:tenantId/users/:userId     # 获取用户详情
PATCH  /api/v1/tenants/:tenantId/users/:userId     # 更新用户信息
PATCH  /api/v1/tenants/:tenantId/users/:userId/status  # 启用/禁用用户
DELETE /api/v1/tenants/:tenantId/users/:userId     # 删除用户

# 认证 API（公开）
POST   /api/v1/auth/login                          # 登录
POST   /api/v1/auth/logout                         # 登出
POST   /api/v1/auth/refresh                        # 刷新 Token
```

### E. 参考资料

1. **多租户架构模式:**
   - [Multi-Tenant Data Architecture](https://docs.microsoft.com/en-us/azure/architecture/guide/multitenant/approaches/overview)
   - [SaaS Tenant Isolation Strategies](https://aws.amazon.com/blogs/apn/saas-tenant-isolation-strategies/)

2. **RBAC 设计:**
   - [Role-Based Access Control](https://en.wikipedia.org/wiki/Role-based_access_control)
   - [NIST RBAC Model](https://csrc.nist.gov/projects/role-based-access-control)

3. **JWT 最佳实践:**
   - [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
   - [JWT Security Best Practices](https://curity.io/resources/learn/jwt-best-practices/)

4. **PostgreSQL 多租户:**
   - [PostgreSQL Multi-Tenant Patterns](https://www.citusdata.com/blog/2016/10/03/designing-your-saas-database-for-high-scalability/)
