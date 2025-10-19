# 多租户用户管理与 JWT 身份认证设计文档

## 概述

本文档描述了多租户用户管理与 JWT 身份认证系统的详细设计方案。该系统基于 Go 1.25、Gin 框架和 GORM ORM，提供完整的多租户隔离、用户认证、JWT token 管理和基于角色的访问控制（RBAC）功能。

### 设计目标

- **多租户隔离**：采用单库单表 + 行级隔离策略，确保租户数据完全隔离
- **安全认证**：使用 JWT access token + refresh token 双 token 机制，支持 token 刷新和撤销
- **灵活授权**：实现 RBAC 权限模型，支持角色和权限范围（scopes）
- **高安全性**：密码使用 bcrypt 哈希、refresh token 轮换、短生命周期 access token
- **可扩展性**：预留 OAuth/SSO、MFA、审计日志等扩展接口
- **易于集成**：与现有 Gin 路由和中间件体系无缝集成

### 技术栈

- **语言**：Go 1.25
- **Web 框架**：标准库 net/http（当前项目使用）
- **ORM**：GORM v1.31.0
- **数据库**：PostgreSQL
- **JWT 库**：golang-jwt/jwt v5
- **密码哈希**：bcrypt（golang.org/x/crypto/bcrypt）
- **UUID**：google/uuid v1.6.0

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端应用                                │
│                  (Web / Mobile / Desktop)                       │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP/HTTPS
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway / Router                        │
│                    (http.ServeMux + 中间件)                      │
├─────────────────────────────────────────────────────────────────┤
│  中间件链：                                                        │
│  1. Recovery (恢复 panic)                                        │
│  2. Logger (请求日志)                                             │
│  3. CORS (跨域处理)                                               │
│  4. TenantIdentifier (租户识别) ← 新增                            │
│  5. JWTAuth (JWT 验证) ← 新增                                     │
│  6. RBACAuthorizer (权限验证) ← 新增                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Handler 层                                │
├─────────────────────────────────────────────────────────────────┤
│  认证相关：                                                        │
│  - AuthHandler (登录、注册、刷新、注销)                            │
│  - TenantHandler (租户管理)                                       │
│  - UserHandler (用户管理)                                         │
│                                                                  │
│  业务相关：                                                        │
│  - ChatHandler (聊天)                                            │
│  - SessionHandler (会话)                                         │
│  - HealthHandler (健康检查)                                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Service 层                                │
├─────────────────────────────────────────────────────────────────┤
│  - AuthService (认证服务)                                         │
│  - TenantService (租户服务)                                       │
│  - UserService (用户服务)                                         │
│  - TokenService (Token 管理服务)                                  │
│  - AuditService (审计日志服务)                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Repository 层                               │
├─────────────────────────────────────────────────────────────────┤
│  - TenantRepository (租户数据访问)                                │
│  - UserRepository (用户数据访问)                                  │
│  - RefreshTokenRepository (Token 数据访问)                        │
│  - AuditRepository (审计日志数据访问)                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PostgreSQL 数据库                            │
├─────────────────────────────────────────────────────────────────┤
│  表：                                                             │
│  - tenants (租户表)                                               │
│  - users (用户表)                                                 │
│  - refresh_tokens (刷新令牌表)                                    │
│  - auth_audit (审计日志表)                                        │
└─────────────────────────────────────────────────────────────────┘
```

### 租户识别策略

系统支持多种租户识别方式，按优先级顺序：

1. **请求头识别**：`X-Tenant-ID` 或 `X-Tenant-Domain`
2. **子域名识别**：从 `Host` 头提取子域名（如 `tenant1.api.example.com`）
3. **URL 路径识别**：从路径提取（如 `/api/v1/tenants/{tenant_id}/...`）
4. **Cookie 识别**：从 Cookie 中读取 `tenant_id`
5. **JWT Claims**：从已验证的 JWT token 中提取 `tid` claim

### 数据隔离策略

采用**单库单表 + 行级隔离**模式：

- 所有租户数据存储在同一数据库的同一张表中
- 每条记录包含 `tenant_id` 字段标识所属租户
- 通过中间件自动注入租户过滤条件
- 使用 GORM Scopes 实现自动租户隔离

**优点**：

- 部署和维护简单
- 易于跨租户数据分析和聚合
- 成本效益高

**安全措施**：

- 中间件层强制注入租户过滤
- Repository 层二次验证租户 ID
- 数据库层唯一索引约束（tenant_id + 业务键）

## 组件与接口设计

### 数据模型（Model 层）

#### Tenant 模型

```go
// Tenant 租户模型
type Tenant struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Name      string         `gorm:"type:varchar(255);not null" json:"name"`
    Domain    string         `gorm:"type:varchar(255)" json:"domain"`
    Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    Status    bool           `gorm:"default:true" json:"status"` // true=启用, false=禁用
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by"`
    IsDeleted bool           `gorm:"default:false" json:"is_deleted"`
}
```

#### User 模型

```go
// User 用户模型
type User struct {
    ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_tenant_email" json:"tenant_id"`
    Email         string         `gorm:"type:varchar(320);not null;index:idx_tenant_email" json:"email"`
    EmailVerified bool           `gorm:"default:false" json:"email_verified"`
    Phone         string         `gorm:"type:varchar(20)" json:"phone"`
    PasswordHash  string         `gorm:"type:text;not null" json:"-"` // 不返回给客户端
    DisplayName   string         `gorm:"type:varchar(255)" json:"display_name"`
    IsActive      bool           `gorm:"default:true" json:"is_active"`
    IsAdmin       bool           `gorm:"default:false" json:"is_admin"`
    Roles         datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"roles"` // ["user", "admin"]
    Meta          datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"meta"`
    LastLoginAt   *time.Time     `json:"last_login_at"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    CreatedBy     *uuid.UUID     `gorm:"type:uuid" json:"created_by"`
    IsDeleted     bool           `gorm:"default:false" json:"is_deleted"`
    
    // 关联
    Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}
```

#### RefreshToken 模型

```go
// RefreshToken 刷新令牌模型
type RefreshToken struct {
    ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
    TenantID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
    TokenHash  string     `gorm:"type:text;not null;uniqueIndex" json:"-"` // 存储 token 的 SHA256 哈希
    Revoked    bool       `gorm:"default:false;index" json:"revoked"`
    CreatedAt  time.Time  `json:"created_at"`
    ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
    ReplacedBy *uuid.UUID `gorm:"type:uuid" json:"replaced_by"` // 轮换时指向新 token
    
    // 关联
    User   User   `gorm:"foreignKey:UserID" json:"-"`
    Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}
```

#### AuthAudit 模型

```go
// AuthAudit 认证审计日志模型
type AuthAudit struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID  *uuid.UUID     `gorm:"type:uuid;index" json:"tenant_id"`
    UserID    *uuid.UUID     `gorm:"type:uuid;index" json:"user_id"`
    Event     string         `gorm:"type:varchar(64);not null;index" json:"event"` // login, logout, refresh, revoke, failed_login
    IP        string         `gorm:"type:inet" json:"ip"`
    UserAgent string         `gorm:"type:text" json:"user_agent"`
    Meta      datatypes.JSON `gorm:"type:jsonb" json:"meta"`
    CreatedAt time.Time      `gorm:"index" json:"created_at"`
}
```

### JWT Claims 结构

```go
// JWTClaims 自定义 JWT Claims
type JWTClaims struct {
    jwt.RegisteredClaims
    TenantID uuid.UUID `json:"tid"`           // 租户 ID
    Roles    []string  `json:"roles"`         // 用户角色
    Scopes   []string  `json:"scopes"`        // 权限范围
}

// 标准 Claims 说明：
// - iss (Issuer): 签发者，如 "genkit-ai-service"
// - sub (Subject): 用户 ID (uuid.UUID)
// - aud (Audience): 受众，如 "genkit-api"
// - exp (Expiration): 过期时间（Unix 时间戳）
// - iat (IssuedAt): 签发时间（Unix 时间戳）
// - jti (JWT ID): JWT 唯一标识符
```

### Repository 接口

#### TenantRepository

```go
type TenantRepository interface {
    // Create 创建租户
    Create(ctx context.Context, tenant *Tenant) error
    
    // GetByID 根据 ID 获取租户
    GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
    
    // GetByDomain 根据域名获取租户
    GetByDomain(ctx context.Context, domain string) (*Tenant, error)
    
    // Update 更新租户
    Update(ctx context.Context, tenant *Tenant) error
    
    // Delete 软删除租户
    Delete(ctx context.Context, id uuid.UUID) error
    
    // List 列出租户（支持分页）
    List(ctx context.Context, page, pageSize int) ([]*Tenant, int64, error)
}
```

#### UserRepository

```go
type UserRepository interface {
    // Create 创建用户
    Create(ctx context.Context, user *User) error
    
    // GetByID 根据 ID 获取用户
    GetByID(ctx context.Context, tenantID, userID uuid.UUID) (*User, error)
    
    // GetByEmail 根据邮箱获取用户
    GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error)
    
    // Update 更新用户
    Update(ctx context.Context, user *User) error
    
    // Delete 软删除用户
    Delete(ctx context.Context, tenantID, userID uuid.UUID) error
    
    // List 列出租户下的用户（支持分页）
    List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*User, int64, error)
    
    // UpdateLastLogin 更新最后登录时间
    UpdateLastLogin(ctx context.Context, tenantID, userID uuid.UUID) error
}
```

#### RefreshTokenRepository

```go
type RefreshTokenRepository interface {
    // Create 创建刷新令牌
    Create(ctx context.Context, token *RefreshToken) error
    
    // GetByTokenHash 根据 token 哈希获取
    GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
    
    // Revoke 撤销令牌
    Revoke(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error
    
    // RevokeAllByUser 撤销用户的所有令牌
    RevokeAllByUser(ctx context.Context, tenantID, userID uuid.UUID) error
    
    // DeleteExpired 删除过期的令牌
    DeleteExpired(ctx context.Context) error
}
```

#### AuditRepository

```go
type AuditRepository interface {
    // Create 创建审计日志
    Create(ctx context.Context, audit *AuthAudit) error
    
    // List 列出审计日志（支持过滤和分页）
    List(ctx context.Context, filter AuditFilter, page, pageSize int) ([]*AuthAudit, int64, error)
}

type AuditFilter struct {
    TenantID  *uuid.UUID
    UserID    *uuid.UUID
    Event     string
    StartTime *time.Time
    EndTime   *time.Time
}
```

### Service 接口

#### AuthService

```go
type AuthService interface {
    // Register 用户注册
    Register(ctx context.Context, req RegisterRequest) (*User, error)
    
    // Login 用户登录
    Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
    
    // RefreshToken 刷新访问令牌
    RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
    
    // Logout 用户注销
    Logout(ctx context.Context, refreshToken string) error
    
    // ChangePassword 修改密码
    ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

type RegisterRequest struct {
    TenantID    uuid.UUID `json:"tenant_id" validate:"required"`
    Email       string    `json:"email" validate:"required,email"`
    Password    string    `json:"password" validate:"required,min=8"`
    DisplayName string    `json:"display_name"`
    Phone       string    `json:"phone"`
}

type LoginRequest struct {
    TenantID uuid.UUID `json:"tenant_id"` // 可选，可从其他方式识别
    Email    string    `json:"email" validate:"required,email"`
    Password string    `json:"password" validate:"required"`
}

type LoginResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresIn    int64     `json:"expires_in"` // 秒
    TokenType    string    `json:"token_type"` // "Bearer"
    User         *User     `json:"user"`
}
```

#### TokenService

```go
type TokenService interface {
    // GenerateAccessToken 生成访问令牌
    GenerateAccessToken(user *User) (string, error)
    
    // GenerateRefreshToken 生成刷新令牌
    GenerateRefreshToken(user *User) (string, *RefreshToken, error)
    
    // ValidateAccessToken 验证访问令牌
    ValidateAccessToken(tokenString string) (*JWTClaims, error)
    
    // ValidateRefreshToken 验证刷新令牌
    ValidateRefreshToken(ctx context.Context, tokenString string) (*RefreshToken, error)
    
    // RevokeRefreshToken 撤销刷新令牌
    RevokeRefreshToken(ctx context.Context, tokenString string) error
    
    // HashToken 计算 token 的哈希值
    HashToken(token string) string
}
```

#### TenantService

```go
type TenantService interface {
    // Create 创建租户
    Create(ctx context.Context, req CreateTenantRequest) (*Tenant, error)
    
    // Get 获取租户
    Get(ctx context.Context, id uuid.UUID) (*Tenant, error)
    
    // Update 更新租户
    Update(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) (*Tenant, error)
    
    // Delete 删除租户
    Delete(ctx context.Context, id uuid.UUID) error
    
    // List 列出租户
    List(ctx context.Context, page, pageSize int) ([]*Tenant, int64, error)
    
    // GetByDomain 根据域名获取租户
    GetByDomain(ctx context.Context, domain string) (*Tenant, error)
}
```

#### UserService

```go
type UserService interface {
    // Create 创建用户
    Create(ctx context.Context, req CreateUserRequest) (*User, error)
    
    // Get 获取用户
    Get(ctx context.Context, tenantID, userID uuid.UUID) (*User, error)
    
    // Update 更新用户
    Update(ctx context.Context, tenantID, userID uuid.UUID, req UpdateUserRequest) (*User, error)
    
    // Delete 删除用户
    Delete(ctx context.Context, tenantID, userID uuid.UUID) error
    
    // List 列出用户
    List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*User, int64, error)
}
```

### 中间件设计

#### TenantIdentifier 中间件

```go
// TenantIdentifier 租户识别中间件
func TenantIdentifier(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var tenantID uuid.UUID
        var err error
        
        // 1. 从请求头识别
        if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
            tenantID, err = uuid.Parse(tid)
        }
        
        // 2. 从子域名识别
        if err != nil || tenantID == uuid.Nil {
            host := r.Host
            // 提取子域名逻辑
            // ...
        }
        
        // 3. 从 URL 路径识别
        // ...
        
        // 4. 从 Cookie 识别
        // ...
        
        if tenantID == uuid.Nil {
            http.Error(w, "Tenant not identified", http.StatusBadRequest)
            return
        }
        
        // 将租户 ID 注入上下文
        ctx := context.WithValue(r.Context(), "tenant_id", tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

#### JWTAuth 中间件

```go
// JWTAuth JWT 认证中间件
func JWTAuth(tokenService TokenService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 从 Authorization 头提取 token
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", http.StatusUnauthorized)
                return
            }
            
            // 验证 Bearer 格式
            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
                return
            }
            
            // 验证 token
            claims, err := tokenService.ValidateAccessToken(parts[1])
            if err != nil {
                http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
                return
            }
            
            // 将用户信息注入上下文
            ctx := r.Context()
            ctx = context.WithValue(ctx, "user_id", claims.Subject)
            ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
            ctx = context.WithValue(ctx, "roles", claims.Roles)
            ctx = context.WithValue(ctx, "scopes", claims.Scopes)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### RBACAuthorizer 中间件

```go
// RBACAuthorizer RBAC 授权中间件
func RBACAuthorizer(requiredRoles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles, ok := r.Context().Value("roles").([]string)
            if !ok {
                http.Error(w, "Roles not found in context", http.StatusForbidden)
                return
            }
            
            // 检查是否有所需角色
            hasRole := false
            for _, required := range requiredRoles {
                for _, role := range roles {
                    if role == required {
                        hasRole = true
                        break
                    }
                }
                if hasRole {
                    break
                }
            }
            
            if !hasRole {
                http.Error(w, "Insufficient permissions", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Handler 设计

#### AuthHandler

```go
type AuthHandler struct {
    authService  AuthService
    auditService AuditService
    logger       logger.Logger
}

// HandleRegister 处理用户注册
// @Summary 用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 201 {object} response.ResponseData[User]
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request)

// HandleLogin 处理用户登录
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} response.ResponseData[LoginResponse]
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request)

// HandleRefresh 处理 token 刷新
// @Summary 刷新访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "刷新请求"
// @Success 200 {object} response.ResponseData[LoginResponse]
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request)

// HandleLogout 处理用户注销
// @Summary 用户注销
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "注销请求"
// @Success 200 {object} response.ResponseData[any]
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request)
```

## 数据库设计

### 表结构与索引

#### tenants 表

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL COMMENT '租户名称',
    domain VARCHAR(255) COMMENT '租户域名，用于子域识别',
    metadata JSONB COMMENT '租户元数据',
    status BOOLEAN DEFAULT TRUE COMMENT '租户状态：true=启用，false=禁用',
    created_at TIMESTAMP DEFAULT now() COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT now() COMMENT '更新时间',
    created_by UUID COMMENT '创建者用户ID',
    is_deleted BOOLEAN DEFAULT FALSE COMMENT '软删除标记'
);

-- 索引
CREATE INDEX idx_tenants_domain ON tenants(domain) WHERE domain IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(status) WHERE NOT is_deleted;
CREATE INDEX idx_tenants_created_at ON tenants(created_at);
```

#### users 表

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE COMMENT '所属租户ID',
    email VARCHAR(320) NOT NULL COMMENT '用户邮箱',
    email_verified BOOLEAN DEFAULT FALSE COMMENT '邮箱是否已验证',
    phone VARCHAR(20) COMMENT '手机号码',
    password_hash TEXT NOT NULL COMMENT '密码哈希值（bcrypt）',
    display_name VARCHAR(255) COMMENT '显示名称',
    is_active BOOLEAN DEFAULT TRUE COMMENT '账户是否激活',
    is_admin BOOLEAN DEFAULT FALSE COMMENT '是否为管理员',
    roles JSONB DEFAULT '[]'::jsonb COMMENT '用户角色列表，如 ["user","admin"]',
    meta JSONB DEFAULT '{}'::jsonb COMMENT '用户元数据',
    last_login_at TIMESTAMP COMMENT '最后登录时间',
    created_at TIMESTAMP DEFAULT now() COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT now() COMMENT '更新时间',
    created_by UUID COMMENT '创建者用户ID',
    is_deleted BOOLEAN DEFAULT FALSE COMMENT '软删除标记',
    
    CONSTRAINT uq_tenant_email UNIQUE(tenant_id, email)
);

-- 索引
CREATE INDEX idx_users_tenant_id ON users(tenant_id) WHERE NOT is_deleted;
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE NOT is_deleted;
CREATE INDEX idx_users_created_at ON users(created_at);
```

#### refresh_tokens 表

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE COMMENT '用户ID',
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE COMMENT '租户ID',
    token_hash TEXT NOT NULL COMMENT 'Refresh Token 的 SHA256 哈希值',
    revoked BOOLEAN DEFAULT FALSE COMMENT '是否已撤销',
    created_at TIMESTAMP DEFAULT now() COMMENT '创建时间',
    expires_at TIMESTAMP NOT NULL COMMENT '过期时间',
    replaced_by UUID COMMENT '轮换时指向新 token 的 ID',
    
    CONSTRAINT uq_token_hash UNIQUE(token_hash)
);

-- 索引
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_tenant_id ON refresh_tokens(tenant_id);
CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(revoked);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

#### auth_audit 表

```sql
CREATE TABLE auth_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID COMMENT '租户ID',
    user_id UUID COMMENT '用户ID',
    event VARCHAR(64) NOT NULL COMMENT '事件类型：login, logout, refresh, revoke, failed_login',
    ip INET COMMENT '客户端IP地址',
    user_agent TEXT COMMENT '用户代理字符串',
    meta JSONB COMMENT '事件元数据',
    created_at TIMESTAMP DEFAULT now() COMMENT '事件发生时间'
);

-- 索引
CREATE INDEX idx_auth_audit_tenant_id ON auth_audit(tenant_id);
CREATE INDEX idx_auth_audit_user_id ON auth_audit(user_id);
CREATE INDEX idx_auth_audit_event ON auth_audit(event);
CREATE INDEX idx_auth_audit_created_at ON auth_audit(created_at);
```

### 数据库迁移

使用 GORM AutoMigrate 或自定义迁移脚本：

```go
// 迁移函数
func MigrateAuthTables(db *gorm.DB) error {
    return db.AutoMigrate(
        &Tenant{},
        &User{},
        &RefreshToken{},
        &AuthAudit{},
    )
}
```

## 安全设计

### 密码安全

#### 密码哈希

使用 bcrypt 算法，cost factor 设置为 12：

```go
import "golang.org/x/crypto/bcrypt"

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

// VerifyPassword 验证密码
func VerifyPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

#### 密码策略

- 最小长度：8 字符
- 建议包含：大小写字母、数字、特殊字符
- 不存储明文密码
- 密码修改需验证旧密码

### Token 安全

#### Access Token

- **类型**：JWT（JSON Web Token）
- **签名算法**：HS256（HMAC-SHA256）
- **生命周期**：60 分钟
- **存储位置**：客户端内存或 sessionStorage（不推荐 localStorage）
- **传输方式**：HTTP Authorization 头（Bearer Token）

#### Refresh Token

- **类型**：不透明令牌（随机生成的 UUID）
- **生命周期**：7-30 天（可配置）
- **存储位置**：
  - 服务端：数据库（存储哈希值）
  - 客户端：HttpOnly Cookie（推荐）或安全存储
- **轮换策略**：一次性使用，使用后立即替换
- **撤销机制**：数据库标记 `revoked=true`

#### Token 生成流程

```go
// 生成 Access Token
func (s *tokenService) GenerateAccessToken(user *User) (string, error) {
    now := time.Now()
    claims := JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "genkit-ai-service",
            Subject:   user.ID.String(),
            Audience:  jwt.ClaimStrings{"genkit-api"},
            ExpiresAt: jwt.NewNumericDate(now.Add(60 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        TenantID: user.TenantID,
        Roles:    extractRoles(user.Roles),
        Scopes:   []string{"chat:read", "chat:write"},
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.jwtSecret))
}

// 生成 Refresh Token
func (s *tokenService) GenerateRefreshToken(user *User) (string, *RefreshToken, error) {
    tokenString := uuid.New().String()
    tokenHash := s.HashToken(tokenString)
    
    refreshToken := &RefreshToken{
        UserID:    user.ID,
        TenantID:  user.TenantID,
        TokenHash: tokenHash,
        ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 天
    }
    
    if err := s.tokenRepo.Create(context.Background(), refreshToken); err != nil {
        return "", nil, err
    }
    
    return tokenString, refreshToken, nil
}
```

### 防护措施

#### 1. CSRF 防护

- Refresh Token 使用 HttpOnly + SameSite Cookie
- 关键操作需要 CSRF Token

#### 2. XSS 防护

- 输入验证和输出编码
- Content-Security-Policy 头
- Access Token 不存储在 localStorage

#### 3. 暴力破解防护

- 登录失败次数限制（如 5 次/15 分钟）
- 账户临时锁定机制
- 审计日志记录失败尝试

#### 4. Token 泄露防护

- 短生命周期 Access Token
- Refresh Token 轮换
- Token 撤销机制
- 异常登录检测（IP、设备变化）

## 错误处理

### 错误码定义

```go
const (
    // 认证相关错误
    ErrInvalidCredentials    = "AUTH_001" // 凭证无效
    ErrUserNotFound          = "AUTH_002" // 用户不存在
    ErrUserInactive          = "AUTH_003" // 用户未激活
    ErrTenantDisabled        = "AUTH_004" // 租户已禁用
    ErrTokenExpired          = "AUTH_005" // Token 已过期
    ErrTokenInvalid          = "AUTH_006" // Token 无效
    ErrTokenRevoked          = "AUTH_007" // Token 已撤销
    ErrInsufficientPermission = "AUTH_008" // 权限不足
    
    // 用户管理错误
    ErrEmailAlreadyExists    = "USER_001" // 邮箱已存在
    ErrWeakPassword          = "USER_002" // 密码强度不足
    ErrInvalidEmail          = "USER_003" // 邮箱格式无效
    
    // 租户相关错误
    ErrTenantNotFound        = "TENANT_001" // 租户不存在
    ErrTenantNotIdentified   = "TENANT_002" // 无法识别租户
)
```

### 错误响应格式

遵循项目现有的响应格式：

```go
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    *struct {
        ErrorCode string `json:"error_code"`
        Details   string `json:"details"`
    } `json:"data,omitempty"`
}
```

## 测试策略

### 单元测试

#### Repository 层测试

- 使用 SQLite 内存数据库或 testcontainers
- 测试 CRUD 操作
- 测试租户隔离逻辑
- 测试并发安全性

#### Service 层测试

- Mock Repository 依赖
- 测试业务逻辑
- 测试错误处理
- 测试边界条件

#### Middleware 测试

- 测试租户识别逻辑
- 测试 JWT 验证
- 测试权限检查
- 测试错误场景

### 集成测试

- 完整的登录流程测试
- Token 刷新流程测试
- 注销流程测试
- 多租户隔离验证
- 并发请求测试

### 安全测试

- SQL 注入测试
- XSS 攻击测试
- CSRF 攻击测试
- Token 重放攻击测试
- 暴力破解测试

## 配置管理

### 环境变量

在现有 `Config` 结构中添加认证相关配置：

```go
// AuthConfig 认证配置
type AuthConfig struct {
    JWTSecret              string        // JWT 签名密钥
    JWTIssuer              string        // JWT 签发者
    JWTAudience            string        // JWT 受众
    AccessTokenTTL         time.Duration // Access Token 生命周期
    RefreshTokenTTL        time.Duration // Refresh Token 生命周期
    BcryptCost             int           // Bcrypt cost factor
    MaxLoginAttempts       int           // 最大登录尝试次数
    LoginAttemptWindow     time.Duration // 登录尝试时间窗口
    PasswordMinLength      int           // 密码最小长度
    EnableRefreshRotation  bool          // 是否启用 Refresh Token 轮换
    TenantIdentifyStrategy string        // 租户识别策略：header, subdomain, path, cookie
}

// 加载认证配置
config.Auth = AuthConfig{
    JWTSecret:              getEnv("JWT_SECRET", ""),
    JWTIssuer:              getEnv("JWT_ISSUER", "genkit-ai-service"),
    JWTAudience:            getEnv("JWT_AUDIENCE", "genkit-api"),
    AccessTokenTTL:         getEnvDuration("ACCESS_TOKEN_TTL", 60*time.Minute),
    RefreshTokenTTL:        getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
    BcryptCost:             getEnvInt("BCRYPT_COST", 12),
    MaxLoginAttempts:       getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
    LoginAttemptWindow:     getEnvDuration("LOGIN_ATTEMPT_WINDOW", 15*time.Minute),
    PasswordMinLength:      getEnvInt("PASSWORD_MIN_LENGTH", 8),
    EnableRefreshRotation:  getEnvBool("ENABLE_REFRESH_ROTATION", true),
    TenantIdentifyStrategy: getEnv("TENANT_IDENTIFY_STRATEGY", "header"),
}
```

### .env 示例

```bash
# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_ISSUER=genkit-ai-service
JWT_AUDIENCE=genkit-api
ACCESS_TOKEN_TTL=60m
REFRESH_TOKEN_TTL=720h  # 30 天

# 密码配置
BCRYPT_COST=12
PASSWORD_MIN_LENGTH=8

# 登录安全
MAX_LOGIN_ATTEMPTS=5
LOGIN_ATTEMPT_WINDOW=15m

# Token 策略
ENABLE_REFRESH_ROTATION=true

# 租户识别
TENANT_IDENTIFY_STRATEGY=header  # header, subdomain, path, cookie
```

## 部署考虑

### 数据库迁移

1. 创建迁移脚本
2. 在部署前执行迁移
3. 验证表结构和索引
4. 创建初始管理员账户

### 密钥管理

- JWT Secret 使用环境变量或密钥管理服务（如 AWS Secrets Manager）
- 定期轮换密钥
- 使用强随机密钥（至少 256 位）

### 监控与告警

- 登录失败率监控
- Token 刷新失败率监控
- 异常 IP 访问告警
- 数据库连接池监控
- API 响应时间监控

### 性能优化

- Refresh Token 表定期清理过期记录
- 审计日志表分区或归档
- Redis 缓存用户信息和权限
- 数据库连接池优化
- JWT 验证缓存（短期）

## 扩展性设计

### OAuth 2.0 / SSO 集成

预留接口支持第三方认证：

```go
type OAuthProvider interface {
    GetAuthURL(state string) string
    ExchangeToken(code string) (*OAuthToken, error)
    GetUserInfo(token string) (*OAuthUserInfo, error)
}
```

### 多因素认证（MFA）

预留 MFA 相关字段和接口：

```go
type User struct {
    // ... 现有字段
    MFAEnabled bool   `gorm:"default:false" json:"mfa_enabled"`
    MFASecret  string `gorm:"type:text" json:"-"`
}

type MFAService interface {
    EnableMFA(ctx context.Context, userID uuid.UUID) (*MFASetup, error)
    VerifyMFA(ctx context.Context, userID uuid.UUID, code string) error
    DisableMFA(ctx context.Context, userID uuid.UUID) error
}
```

### 权限细粒度控制

支持更细粒度的权限定义：

```go
type Permission struct {
    ID          uuid.UUID `gorm:"type:uuid;primary_key"`
    TenantID    uuid.UUID `gorm:"type:uuid;not null"`
    Resource    string    `gorm:"type:varchar(100);not null"` // chat, session, user
    Action      string    `gorm:"type:varchar(50);not null"`  // read, write, delete
    Description string    `gorm:"type:text"`
}

type RolePermission struct {
    RoleID       string    `gorm:"type:varchar(50);not null"`
    PermissionID uuid.UUID `gorm:"type:uuid;not null"`
}
```

## API 路由设计

### 认证相关路由

```
POST   /api/v1/auth/register          # 用户注册
POST   /api/v1/auth/login             # 用户登录
POST   /api/v1/auth/refresh           # 刷新 Token
POST   /api/v1/auth/logout            # 用户注销
POST   /api/v1/auth/change-password   # 修改密码
GET    /api/v1/auth/me                # 获取当前用户信息
```

### 租户管理路由（需要管理员权限）

```
POST   /api/v1/tenants                # 创建租户
GET    /api/v1/tenants                # 列出租户
GET    /api/v1/tenants/:id            # 获取租户详情
PUT    /api/v1/tenants/:id            # 更新租户
DELETE /api/v1/tenants/:id            # 删除租户
```

### 用户管理路由（需要租户管理员权限）

```
POST   /api/v1/users                  # 创建用户
GET    /api/v1/users                  # 列出用户
GET    /api/v1/users/:id              # 获取用户详情
PUT    /api/v1/users/:id              # 更新用户
DELETE /api/v1/users/:id              # 删除用户
```

### 审计日志路由（需要管理员权限）

```
GET    /api/v1/audit/auth             # 查询认证审计日志
```

## 实现优先级

### Phase 1：核心认证功能（MVP）

1. 数据库表结构和迁移
2. User 和 Tenant 模型
3. Repository 层实现
4. 密码哈希和验证
5. JWT Token 生成和验证
6. 基本的登录、注册、注销功能
7. 租户识别中间件（仅支持请求头）
8. JWT 认证中间件

### Phase 2：Token 管理

1. Refresh Token 生成和存储
2. Token 刷新流程
3. Token 轮换机制
4. Token 撤销功能
5. 过期 Token 清理

### Phase 3：权限控制

1. RBAC 中间件
2. 角色管理
3. 权限验证
4. 审计日志记录

### Phase 4：增强功能

1. 多种租户识别策略
2. 密码修改功能
3. 邮箱验证
4. 登录失败限制
5. 异常登录检测

### Phase 5：扩展功能（可选）

1. OAuth/SSO 集成
2. MFA 支持
3. 细粒度权限控制
4. 性能优化（缓存）
5. 监控和告警

## 总结

本设计文档提供了一个完整的多租户用户管理与 JWT 身份认证系统的技术方案。该方案：

- **安全可靠**：采用业界最佳实践，包括 bcrypt 密码哈希、JWT + Refresh Token 双 token 机制、token 轮换等
- **易于集成**：与现有项目架构无缝集成，遵循项目的分层架构和编码规范
- **可扩展**：预留了 OAuth、MFA、细粒度权限等扩展接口
- **高性能**：采用合理的索引设计和缓存策略
- **易于维护**：清晰的分层架构、完善的错误处理和日志记录

实施时建议按照优先级分阶段进行，先实现核心功能（Phase 1-2），再逐步添加增强功能。
