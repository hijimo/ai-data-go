# 认证服务 (Auth Service)

本目录包含多租户用户管理与 JWT 身份认证系统的服务层实现。

## 组件

### TokenService

TokenService 负责 JWT Access Token 和 Refresh Token 的生成、验证和管理。

#### 功能特性

- **Access Token 生成和验证**
  - 使用 JWT (JSON Web Token) 格式
  - 签名算法：HS256 (HMAC-SHA256)
  - 默认有效期：60 分钟（可配置）
  - 包含用户 ID、租户 ID、角色和权限范围

- **Refresh Token 生成和验证**
  - 使用 UUID 作为不透明令牌
  - 存储 SHA256 哈希值到数据库
  - 默认有效期：30 天（可配置）
  - 支持 Token 轮换机制

- **Token 撤销**
  - 支持单个 Token 撤销
  - 支持撤销用户的所有 Token
  - 记录 Token 轮换关系

#### 使用示例

```go
package main

import (
    "genkit-ai-service/internal/config"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/repository"
    "genkit-ai-service/internal/service/auth"
)

func main() {
    // 加载配置
    cfg, _ := config.Load()
    
    // 创建 Repository
    tokenRepo := repository.NewRefreshTokenRepository(db)
    
    // 创建 TokenService
    tokenService := auth.NewTokenService(&cfg.Auth, tokenRepo)
    
    // 生成 Access Token
    user := &model.User{
        ID:       "user-id",
        TenantID: "tenant-id",
        Email:    "user@example.com",
        Roles:    datatypes.JSON([]byte(`["user"]`)),
    }
    
    accessToken, err := tokenService.GenerateAccessToken(user)
    if err != nil {
        // 处理错误
    }
    
    // 验证 Access Token
    claims, err := tokenService.ValidateAccessToken(accessToken)
    if err != nil {
        // Token 无效或已过期
    }
    
    // 生成 Refresh Token
    refreshTokenString, refreshToken, err := tokenService.GenerateRefreshToken(user)
    if err != nil {
        // 处理错误
    }
    
    // 验证 Refresh Token
    validatedToken, err := tokenService.ValidateRefreshToken(ctx, refreshTokenString)
    if err != nil {
        // Token 无效、已撤销或已过期
    }
    
    // 撤销 Refresh Token
    err = tokenService.RevokeRefreshToken(ctx, refreshTokenString)
    if err != nil {
        // 处理错误
    }
}
```

#### JWT Claims 结构

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    TenantID string   `json:"tid"`    // 租户 ID
    Roles    []string `json:"roles"`  // 用户角色列表
    Scopes   []string `json:"scopes"` // 权限范围列表
}
```

**标准 Claims：**

- `iss` (Issuer): 签发者
- `sub` (Subject): 用户 ID
- `aud` (Audience): 受众
- `exp` (Expiration): 过期时间
- `iat` (IssuedAt): 签发时间
- `jti` (JWT ID): JWT 唯一标识符

**自定义 Claims：**

- `tid`: 租户 ID
- `roles`: 用户角色数组（如 ["user", "admin"]）
- `scopes`: 权限范围数组（如 ["chat:read", "chat:write"]）

#### 角色和权限

系统支持以下预定义角色：

- **admin**: 管理员，拥有所有权限
  - chat:read, chat:write, chat:delete
  - session:read, session:write, session:delete
  - user:read, user:write, user:delete
  - tenant:read, tenant:write

- **moderator**: 协调员，拥有部分管理权限
  - chat:read, chat:write, chat:delete
  - session:read, session:write
  - user:read

- **user**: 普通用户，基本权限
  - chat:read, chat:write
  - session:read, session:write

## 配置

在 `.env` 文件中配置以下环境变量：

```bash
# JWT 配置
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-min-32-chars
JWT_ISSUER=genkit-ai-service
JWT_AUDIENCE=genkit-api
ACCESS_TOKEN_TTL=60m
REFRESH_TOKEN_TTL=720h  # 30 天

# 密码配置
BCRYPT_COST=12
PASSWORD_MIN_LENGTH=8

# 登录安全配置
MAX_LOGIN_ATTEMPTS=5
LOGIN_ATTEMPT_WINDOW=15m

# Token 策略
ENABLE_REFRESH_ROTATION=true

# 租户识别策略
TENANT_IDENTIFY_STRATEGY=header
```

## 安全注意事项

1. **JWT Secret**: 生产环境必须使用强随机密钥（至少 32 个字符）
2. **Token 存储**:
   - Access Token 应存储在内存或 sessionStorage
   - Refresh Token 应存储在 HttpOnly Cookie
3. **Token 生命周期**:
   - Access Token 短期有效（60 分钟）
   - Refresh Token 长期有效（30 天）
4. **Token 轮换**: 启用 Refresh Token 轮换以提高安全性
5. **哈希存储**: Refresh Token 在数据库中以 SHA256 哈希值存储

## 测试

运行单元测试：

```bash
go test -v ./internal/service/auth
```

### TenantService

TenantService 负责租户的创建、查询、更新和删除等管理功能。

#### 功能特性

- **租户管理**
  - 创建租户（支持域名和元数据）
  - 查询租户（按 ID 或域名）
  - 更新租户信息
  - 软删除租户
  - 分页列出租户

- **租户状态验证**
  - 验证租户是否存在
  - 验证租户是否启用
  - 防止域名重复

#### 使用示例

```go
// 创建 TenantService
tenantRepo := repository.NewTenantRepository(db)
tenantService := auth.NewTenantService(tenantRepo)

// 创建租户
tenant, err := tenantService.Create(ctx, auth.CreateTenantRequest{
    Name:   "示例公司",
    Domain: "example.com",
    Metadata: map[string]interface{}{
        "industry": "technology",
        "size":     "medium",
    },
})

// 获取租户
tenant, err := tenantService.Get(ctx, tenantID)

// 根据域名获取租户
tenant, err := tenantService.GetByDomain(ctx, "example.com")

// 更新租户
name := "新公司名称"
status := false
tenant, err := tenantService.Update(ctx, tenantID, auth.UpdateTenantRequest{
    Name:   &name,
    Status: &status,
})

// 列出租户
tenants, total, err := tenantService.List(ctx, 1, 10)

// 删除租户
err := tenantService.Delete(ctx, tenantID)
```

### UserService

UserService 负责用户的创建、查询、更新和删除等管理功能，所有操作都包含租户隔离。

#### 功能特性

- **用户管理**
  - 创建用户（自动密码加密）
  - 查询用户（按 ID）
  - 更新用户信息
  - 软删除用户
  - 分页列出租户下的用户

- **租户隔离**
  - 所有操作自动包含租户过滤
  - 防止跨租户数据访问
  - 邮箱在租户内唯一

- **密码安全**
  - 密码强度验证
  - 使用 bcrypt 加密存储
  - 不返回密码哈希值

#### 使用示例

```go
// 创建 UserService
userRepo := repository.NewUserRepository(db)
tenantRepo := repository.NewTenantRepository(db)
userService := auth.NewUserService(userRepo, tenantRepo)

// 创建用户
user, err := userService.Create(ctx, auth.CreateUserRequest{
    TenantID:    tenantID,
    Email:       "user@example.com",
    Password:    "SecurePass123!",
    DisplayName: "张三",
    Phone:       "13800138000",
    IsAdmin:     false,
    Roles:       []string{"user"},
})

// 获取用户
user, err := userService.Get(ctx, tenantID, userID)

// 更新用户
displayName := "李四"
isActive := true
user, err := userService.Update(ctx, tenantID, userID, auth.UpdateUserRequest{
    DisplayName: &displayName,
    IsActive:    &isActive,
    Roles:       []string{"user", "moderator"},
})

// 列出租户下的用户
users, total, err := userService.List(ctx, tenantID, 1, 10)

// 删除用户
err := userService.Delete(ctx, tenantID, userID)
```

## 下一步

- 实现认证中间件（JWT 验证、租户识别、RBAC）
- 实现 Handler 层（API 端点）
- 添加审计日志功能
