# 设计文档

## 概述

本功能旨在增强系统的审计追踪能力，通过在所有数据表中添加 `created_by_name` 字段来冗余记录创建人的显示名称。同时，确保 `created_by` 和 `created_by_name` 字段始终从 JWT 令牌中自动获取，而不是由外部请求传入，以提高数据的安全性和一致性。

### 设计目标

1. **提升查询性能**：通过冗余存储创建人显示名称，避免频繁关联用户表查询
2. **增强数据安全**：强制从 JWT 令牌中提取创建者信息，防止客户端伪造
3. **保持向后兼容**：新字段允许为 NULL，不影响现有数据和功能
4. **简化审计追踪**：在审计日志和数据查询中直接显示创建人名称

### 核心原则

- **安全优先**：创建者信息必须从服务端 JWT 令牌中提取，不信任客户端传入的值
- **数据一致性**：`created_by` 和 `created_by_name` 必须来自同一个 JWT 令牌
- **向后兼容**：新字段允许为 NULL，现有数据保持不变
- **最小侵入**：尽量减少对现有代码的修改，保持代码结构清晰

## 架构设计

### 整体架构

```
┌─────────────────┐
│   客户端请求    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  JWT 中间件     │ ← 解析 JWT，提取 Claims
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Handler 层     │ ← 接收请求，调用服务层
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Service 层     │ ← 从 Context 获取 JWT Claims
│                 │   提取 Subject (用户ID)
│                 │   提取 DisplayName (显示名称)
│                 │   设置 created_by 和 created_by_name
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Repository 层   │ ← 执行数据库操作
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   数据库        │
└─────────────────┘
```

### 数据流程

1. **用户发起创建请求**（创建租户/用户/会话）
2. **JWT 中间件解析令牌**，将 Claims 存入 Context
3. **Handler 层接收请求**，调用服务层方法
4. **Service 层从 Context 获取 JWT Claims**
   - 提取 `Subject` 字段作为用户ID
   - 提取 `DisplayName` 字段作为显示名称
5. **Service 层设置创建者信息**
   - 将用户ID设置为 `created_by` 字段
   - 将显示名称设置为 `created_by_name` 字段
   - 忽略客户端请求中的 `created_by` 和 `created_by_name` 值
6. **Repository 层执行数据库插入**
7. **返回创建结果**

## 组件和接口

### 1. JWT Claims 扩展

#### 当前结构

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    TenantID string   `json:"tid"`
    Roles    []string `json:"roles"`
    Scopes   []string `json:"scopes"`
}
```

#### 扩展后结构

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    TenantID    string   `json:"tid"`
    DisplayName string   `json:"displayName"` // 新增字段
    Roles       []string `json:"roles"`
    Scopes      []string `json:"scopes"`
}
```

#### 设计决策

- **字段名称**：使用 `DisplayName` 而不是 `Name`，与 User 模型保持一致
- **JSON 标签**：使用 `displayName` 驼峰命名，符合前端命名规范
- **必填性**：该字段为可选，如果用户没有设置显示名称，则为空字符串

### 2. 数据模型更新

#### 2.1 Tenant 模型

```go
type Tenant struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Name      string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
    // ... 其他字段 ...
    CreatedBy     *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
    CreatedByName *string    `gorm:"type:varchar(255)" json:"createdByName"` // 新增字段
    // ... 其他字段 ...
}
```

#### 2.2 User 模型

```go
type User struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"tenantId"`
    // ... 其他字段 ...
    CreatedBy     *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
    CreatedByName *string    `gorm:"type:varchar(255)" json:"createdByName"` // 新增字段
    // ... 其他字段 ...
}
```

#### 2.3 ChatSession 模型

```go
type ChatSession struct {
    ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID   uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
    // ... 其他字段 ...
    CreatedBy     uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
    CreatedByName *string   `gorm:"type:varchar(255)" json:"createdByName"` // 新增字段
    // ... 其他字段 ...
}
```

#### 设计决策

- **字段类型**：使用 `*string` 指针类型，允许为 NULL
- **GORM 标签**：`gorm:"type:varchar(255)"` 明确指定数据库类型
- **JSON 标签**：使用 `createdByName` 驼峰命名
- **注释**：添加中文注释说明字段用途："创建者显示名称"

### 3. 服务层实现

#### 3.1 通用辅助函数

在 `internal/service/auth/helpers.go` 中添加辅助函数：

```go
// GetCreatorInfoFromContext 从 Context 中获取创建者信息
// 返回：用户ID指针、显示名称指针
func GetCreatorInfoFromContext(ctx context.Context) (*uuid.UUID, *string) {
    claims, ok := GetJWTClaimsFromContext(ctx)
    if !ok {
        return nil, nil
    }
    
    // 解析用户ID
    var userIDPtr *uuid.UUID
    if claims.Subject != "" {
        userID, err := uuid.Parse(claims.Subject)
        if err == nil {
            userIDPtr = &userID
        }
    }
    
    // 获取显示名称
    var displayNamePtr *string
    if claims.DisplayName != "" {
        displayNamePtr = &claims.DisplayName
    }
    
    return userIDPtr, displayNamePtr
}
```

#### 设计决策

- **统一函数**：创建统一的辅助函数，避免在每个服务中重复代码
- **返回指针**：返回指针类型，与模型字段类型保持一致
- **错误处理**：如果 JWT Claims 不存在或解析失败，返回 nil
- **位置选择**：放在 `auth` 包中，因为其他服务（如 session）也需要使用

#### 3.2 租户服务更新

在 `TenantService.Create` 方法中：

```go
func (s *tenantService) Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error) {
    // ... 现有验证逻辑 ...
    
    // 从 Context 获取创建者信息（替换现有逻辑）
    createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)
    
    tenant := &model.Tenant{
        ID:            uuid.New(),
        Name:          req.Name,
        Domain:        req.Domain,
        Type:          tenantType,
        Status:        true,
        CreatedBy:     createdByUUID,     // 从 JWT 获取
        CreatedByName: createdByName,     // 从 JWT 获取
        IsDeleted:     false,
    }
    
    // ... 其余逻辑 ...
}
```

在 `TenantService.CreateWithAdmin` 方法中：

```go
func (s *tenantService) CreateWithAdmin(ctx context.Context, req CreateTenantWithAdminRequest) (*CreateTenantWithAdminResponse, error) {
    // ... 现有验证逻辑 ...
    
    // 从 Context 获取创建者信息
    createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)
    
    // 创建租户
    tenant := &model.Tenant{
        ID:            uuid.New(),
        Name:          req.TenantName,
        Domain:        req.TenantDomain,
        Type:          model.TenantTypeBusiness,
        Status:        true,
        CreatedBy:     createdByUUID,
        CreatedByName: createdByName,
        IsDeleted:     false,
    }
    
    // ... 保存租户 ...
    
    // 创建管理员用户时也需要设置创建者信息
    adminUser := &model.User{
        ID:            uuid.New(),
        TenantID:      tenant.ID,
        Email:         adminEmail,
        PasswordHash:  passwordHash,
        DisplayName:   adminDisplayName,
        IsActive:      true,
        IsAdmin:       true,
        CreatedBy:     createdByUUID,
        CreatedByName: createdByName,
        IsDeleted:     false,
    }
    
    // ... 其余逻辑 ...
}
```

#### 3.3 用户服务更新

在 `UserService.Create` 方法中：

```go
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*model.User, error) {
    // ... 现有验证逻辑 ...
    
    // 从 Context 获取创建者信息（替换现有逻辑）
    createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)
    
    user := &model.User{
        ID:            uuid.New(),
        TenantID:      tenantUUID,
        Email:         req.Email,
        PasswordHash:  passwordHash,
        DisplayName:   req.DisplayName,
        Phone:         req.Phone,
        IsActive:      true,
        IsAdmin:       req.IsAdmin,
        CreatedBy:     createdByUUID,     // 从 JWT 获取
        CreatedByName: createdByName,     // 从 JWT 获取
        IsDeleted:     false,
    }
    
    // ... 其余逻辑 ...
}
```

#### 设计决策

- **移除请求参数**：从 `CreateUserRequest` 中移除 `CreatedBy` 字段（如果存在）
- **强制使用 JWT**：不再允许客户端传入创建者信息
- **保持一致性**：所有创建操作都使用相同的逻辑

#### 3.4 会话服务更新

在 `SessionService.CreateSession` 方法中：

```go
func (s *sessionService) CreateSession(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error) {
    userUUID, err := uuid.Parse(userID)
    if err != nil {
        return nil, errors.NewBadRequestError("用户ID格式无效")
    }
    
    // 从 Context 获取创建者信息
    createdByUUID, createdByName := auth.GetCreatorInfoFromContext(ctx)
    
    // 如果无法从 JWT 获取，使用当前用户ID
    if createdByUUID == nil {
        createdByUUID = &userUUID
    }
    
    session := &model.ChatSession{
        UserID:        userUUID,
        Title:         req.Title,
        ModelName:     req.ModelName,
        SystemPrompt:  req.SystemPrompt,
        Temperature:   req.Temperature,
        TopP:          req.TopP,
        CreatedBy:     *createdByUUID,    // 从 JWT 获取
        CreatedByName: createdByName,     // 从 JWT 获取
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
        MessageCount:  0,
        IsPinned:      false,
        IsArchived:    false,
        IsDeleted:     false,
    }
    
    // ... 其余逻辑 ...
}
```

#### 设计决策

- **跨包调用**：会话服务需要导入 `auth` 包来使用辅助函数
- **回退逻辑**：如果无法从 JWT 获取创建者ID，使用当前用户ID作为回退
- **字段类型差异**：ChatSession 的 `CreatedBy` 是非指针类型，需要解引用

### 4. 认证服务更新

#### 4.1 JWT 令牌生成

在 `AuthService.Login` 方法中，生成 JWT 令牌时添加 `DisplayName`：

```go
func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    // ... 现有验证逻辑 ...
    
    // 生成 JWT Claims
    claims := &model.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID.String(),
            Issuer:    "genkit-ai-service",
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
        },
        TenantID:    user.TenantID.String(),
        DisplayName: user.DisplayName,  // 新增字段
        Roles:       roles,
        Scopes:      scopes,
    }
    
    // ... 生成令牌 ...
}
```

#### 4.2 JWT 令牌刷新

在 `AuthService.RefreshToken` 方法中，刷新令牌时也需要包含 `DisplayName`：

```go
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error) {
    // ... 现有验证逻辑 ...
    
    // 重新查询用户信息以获取最新的 DisplayName
    user, err := s.userRepo.GetByIDOnly(ctx, oldToken.UserID.String())
    if err != nil {
        return nil, errors.New("用户不存在")
    }
    
    // 生成新的 Access Token
    claims := &model.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID.String(),
            Issuer:    "genkit-ai-service",
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
        },
        TenantID:    user.TenantID.String(),
        DisplayName: user.DisplayName,  // 新增字段
        Roles:       roles,
        Scopes:      scopes,
    }
    
    // ... 其余逻辑 ...
}
```

#### 设计决策

- **实时获取**：每次生成令牌时都从数据库获取最新的 `DisplayName`
- **刷新同步**：刷新令牌时重新查询用户信息，确保 `DisplayName` 是最新的
- **空值处理**：如果用户没有设置 `DisplayName`，则为空字符串

## 数据库迁移

### 迁移文件结构

创建新的迁移文件：`internal/db/migrations/add_created_by_name_migration.go`

```go
package migrations

import (
    "gorm.io/gorm"
)

type AddCreatedByNameMigration struct {
    db *gorm.DB
}

func NewAddCreatedByNameMigration(db *gorm.DB) *AddCreatedByNameMigration {
    return &AddCreatedByNameMigration{db: db}
}

func (m *AddCreatedByNameMigration) Up() error {
    return m.db.Transaction(func(tx *gorm.DB) error {
        // 为 tenants 表添加 created_by_name 字段
        if err := tx.Exec(`
            ALTER TABLE tenants 
            ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255)
        `).Error; err != nil {
            return err
        }
        
        // 为 users 表添加 created_by_name 字段
        if err := tx.Exec(`
            ALTER TABLE users 
            ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255)
        `).Error; err != nil {
            return err
        }
        
        // 为 chat_sessions 表添加 created_by_name 字段
        if err := tx.Exec(`
            ALTER TABLE chat_sessions 
            ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255)
        `).Error; err != nil {
            return err
        }
        
        // 添加字段注释
        if err := tx.Exec(`
            COMMENT ON COLUMN tenants.created_by_name IS '创建者显示名称'
        `).Error; err != nil {
            return err
        }
        
        if err := tx.Exec(`
            COMMENT ON COLUMN users.created_by_name IS '创建者显示名称'
        `).Error; err != nil {
            return err
        }
        
        if err := tx.Exec(`
            COMMENT ON COLUMN chat_sessions.created_by_name IS '创建者显示名称'
        `).Error; err != nil {
            return err
        }
        
        return nil
    })
}

func (m *AddCreatedByNameMigration) Down() error {
    return m.db.Transaction(func(tx *gorm.DB) error {
        // 回滚：删除 created_by_name 字段
        if err := tx.Exec(`
            ALTER TABLE tenants DROP COLUMN IF EXISTS created_by_name
        `).Error; err != nil {
            return err
        }
        
        if err := tx.Exec(`
            ALTER TABLE users DROP COLUMN IF EXISTS created_by_name
        `).Error; err != nil {
            return err
        }
        
        if err := tx.Exec(`
            ALTER TABLE chat_sessions DROP COLUMN IF EXISTS created_by_name
        `).Error; err != nil {
            return err
        }
        
        return nil
    })
}

func (m *AddCreatedByNameMigration) Name() string {
    return "add_created_by_name"
}
```

### 迁移注册

在迁移管理器中注册新迁移：

```go
// 在 internal/db/migrations/manager.go 中
func (m *Manager) RegisterMigrations() {
    // ... 现有迁移 ...
    m.Register(NewAddCreatedByNameMigration(m.db))
}
```

### 设计决策

- **事务执行**：所有 DDL 操作在一个事务中执行，确保原子性
- **幂等性**：使用 `IF NOT EXISTS` 和 `IF EXISTS`，支持重复执行
- **回滚支持**：提供 Down 方法，支持迁移回滚
- **字段注释**：添加数据库注释，提高可维护性
- **NULL 允许**：新字段允许为 NULL，不影响现有数据

## 错误处理

### 错误场景

1. **JWT Claims 不存在**
   - 场景：用户未登录或令牌无效
   - 处理：`GetCreatorInfoFromContext` 返回 nil
   - 影响：`created_by` 和 `created_by_name` 字段为 NULL

2. **用户ID 解析失败**
   - 场景：JWT Claims 中的 Subject 格式无效
   - 处理：返回 nil，不设置创建者信息
   - 影响：`created_by` 和 `created_by_name` 字段为 NULL

3. **DisplayName 为空**
   - 场景：用户未设置显示名称
   - 处理：`created_by_name` 设置为 NULL
   - 影响：仅 `created_by` 有值，`created_by_name` 为 NULL

4. **数据库迁移失败**
   - 场景：迁移过程中发生错误
   - 处理：事务回滚，保持数据库原状
   - 影响：系统继续使用旧结构

### 错误处理策略

- **优雅降级**：如果无法获取创建者信息，允许字段为 NULL，不阻塞业务流程
- **日志记录**：记录无法获取创建者信息的情况，便于排查问题
- **事务保护**：数据库迁移使用事务，确保原子性
- **向后兼容**：新字段允许为 NULL，不影响现有功能

## 测试策略

### 单元测试

#### 1. JWT Claims 测试

```go
func TestJWTClaims_WithDisplayName(t *testing.T) {
    claims := &model.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject: "user-id-123",
        },
        TenantID:    "tenant-id-456",
        DisplayName: "张三",
        Roles:       []string{"user"},
    }
    
    // 验证 DisplayName 字段
    assert.Equal(t, "张三", claims.DisplayName)
}
```

#### 2. 辅助函数测试

```go
func TestGetCreatorInfoFromContext(t *testing.T) {
    // 测试正常情况
    ctx := context.WithValue(context.Background(), "jwt_claims", &model.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject: "user-id-123",
        },
        DisplayName: "张三",
    })
    
    userID, displayName := GetCreatorInfoFromContext(ctx)
    assert.NotNil(t, userID)
    assert.NotNil(t, displayName)
    assert.Equal(t, "张三", *displayName)
    
    // 测试 Claims 不存在
    emptyCtx := context.Background()
    userID, displayName = GetCreatorInfoFromContext(emptyCtx)
    assert.Nil(t, userID)
    assert.Nil(t, displayName)
}
```

#### 3. 服务层测试

```go
func TestTenantService_Create_WithCreatorInfo(t *testing.T) {
    // 准备测试数据
    ctx := context.WithValue(context.Background(), "jwt_claims", &model.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject: "user-id-123",
        },
        DisplayName: "张三",
    })
    
    req := CreateTenantRequest{
        Name:   "测试租户",
        Domain: "test.example.com",
    }
    
    // 执行创建
    tenant, err := tenantService.Create(ctx, req)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, tenant.CreatedBy)
    assert.NotNil(t, tenant.CreatedByName)
    assert.Equal(t, "张三", *tenant.CreatedByName)
}

func TestUserService_Create_WithCreatorInfo(t *testing.T) {
    // 类似的测试逻辑
}

func TestSessionService_CreateSession_WithCreatorInfo(t *testing.T) {
    // 类似的测试逻辑
}
```

### 集成测试

#### 1. 端到端测试

```go
func TestCreateTenant_E2E(t *testing.T) {
    // 1. 登录获取 JWT 令牌
    loginResp := login(t, "admin@example.com", "password")
    token := loginResp.AccessToken
    
    // 2. 使用令牌创建租户
    req := CreateTenantRequest{
        Name:   "新租户",
        Domain: "new.example.com",
    }
    
    tenant := createTenant(t, token, req)
    
    // 3. 验证创建者信息
    assert.NotNil(t, tenant.CreatedBy)
    assert.NotNil(t, tenant.CreatedByName)
    assert.Equal(t, "Admin User", *tenant.CreatedByName)
}
```

#### 2. 数据库迁移测试

```go
func TestAddCreatedByNameMigration(t *testing.T) {
    // 1. 执行迁移
    migration := NewAddCreatedByNameMigration(db)
    err := migration.Up()
    assert.NoError(t, err)
    
    // 2. 验证字段存在
    var columnExists bool
    db.Raw(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'tenants' 
            AND column_name = 'created_by_name'
        )
    `).Scan(&columnExists)
    assert.True(t, columnExists)
    
    // 3. 测试回滚
    err = migration.Down()
    assert.NoError(t, err)
    
    // 4. 验证字段已删除
    db.Raw(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'tenants' 
            AND column_name = 'created_by_name'
        )
    `).Scan(&columnExists)
    assert.False(t, columnExists)
}
```

### 测试覆盖目标

- **单元测试覆盖率**：≥ 80%
- **集成测试覆盖率**：≥ 60%
- **关键路径测试**：100%（创建租户、用户、会话）

## 安全考虑

### 1. 防止客户端伪造

**问题**：客户端可能尝试在请求中传入 `created_by` 或 `created_by_name` 字段

**解决方案**：

- 在服务层强制从 JWT Claims 中提取创建者信息
- 忽略请求参数中的 `created_by` 和 `created_by_name` 字段
- 从 `CreateTenantRequest`、`CreateUserRequest` 等结构体中移除这些字段

**代码示例**：

```go
// ❌ 错误做法：信任客户端传入的值
func (s *tenantService) Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error) {
    tenant := &model.Tenant{
        CreatedBy:     req.CreatedBy,     // 危险！
        CreatedByName: req.CreatedByName, // 危险！
    }
}

// ✅ 正确做法：从 JWT 获取
func (s *tenantService) Create(ctx context.Context, req CreateTenantRequest) (*model.Tenant, error) {
    createdByUUID, createdByName := GetCreatorInfoFromContext(ctx)
    tenant := &model.Tenant{
        CreatedBy:     createdByUUID,
        CreatedByName: createdByName,
    }
}
```

### 2. JWT 令牌安全

**问题**：JWT 令牌可能被篡改或伪造

**现有保护措施**：

- JWT 使用签名验证，防止篡改
- 令牌有过期时间，限制有效期
- 使用 HTTPS 传输，防止中间人攻击

**新增考虑**：

- `DisplayName` 字段包含在签名中，无法单独篡改
- 刷新令牌时重新查询数据库，确保 `DisplayName` 是最新的

### 3. 数据一致性

**问题**：`created_by` 和 `created_by_name` 可能不一致

**保护措施**：

- 两个字段必须来自同一个 JWT 令牌
- 使用统一的辅助函数 `GetCreatorInfoFromContext`
- 不允许单独更新这两个字段

### 4. 审计追踪

**增强**：

- 所有创建操作都记录创建者信息
- 审计日志中包含 `created_by_name`，便于追踪
- 即使用户被删除，`created_by_name` 仍然保留

## 性能考虑

### 1. 查询性能提升

**优化点**：

- 避免关联查询：查询租户/用户/会话列表时，不需要 JOIN users 表获取创建者名称
- 减少数据库负载：冗余存储 `created_by_name`，减少查询次数

**性能对比**：

```sql
-- 优化前：需要 JOIN users 表
SELECT t.*, u.display_name as creator_name
FROM tenants t
LEFT JOIN users u ON t.created_by = u.id
WHERE t.is_deleted = false;

-- 优化后：直接查询
SELECT t.*, t.created_by_name
FROM tenants t
WHERE t.is_deleted = false;
```

**预期提升**：

- 查询时间减少约 20-30%（取决于数据量）
- 数据库负载降低

### 2. 存储开销

**新增存储**：

- 每条记录增加 VARCHAR(255) 字段
- 预估每条记录增加约 50-100 字节（取决于名称长度）

**影响评估**：

- 对于 10 万条记录，增加约 5-10 MB 存储空间
- 存储成本可忽略不计

### 3. 写入性能

**影响**：

- 创建操作增加一个字段的写入
- 性能影响可忽略（< 1%）

**优化措施**：

- 字段允许为 NULL，不需要额外验证
- 不创建额外索引，避免写入开销

### 4. JWT 令牌大小

**影响**：

- JWT 令牌增加 `DisplayName` 字段
- 预估增加 20-50 字节（取决于名称长度）

**评估**：

- 对于典型的 JWT 令牌（约 500 字节），增加约 5-10%
- 对网络传输影响可忽略

## 向后兼容性

### 1. 数据库兼容

**现有数据**：

- 迁移后，现有记录的 `created_by_name` 字段为 NULL
- 不影响现有查询和业务逻辑

**新数据**：

- 迁移后创建的记录会自动填充 `created_by_name`
- 逐步替换旧数据

### 2. API 兼容

**响应格式**：

```json
{
  "id": "uuid",
  "name": "租户名称",
  "createdBy": "user-uuid",
  "createdByName": "张三"  // 新增字段，可能为 null
}
```

**影响评估**：

- 前端可以选择性使用 `createdByName` 字段
- 如果字段为 null，前端可以回退到关联查询或显示 ID

### 3. 代码兼容

**请求结构体**：

- 从 `CreateTenantRequest` 等结构体中移除 `CreatedBy` 字段
- 这是一个破坏性变更，但提高了安全性

**迁移建议**：

- 如果有外部系统调用创建接口，需要更新调用代码
- 移除请求中的 `createdBy` 参数

### 4. 回滚策略

**如果需要回滚**：

1. 执行迁移的 Down 方法，删除 `created_by_name` 字段
2. 恢复服务层代码，移除 `created_by_name` 相关逻辑
3. 恢复 JWT Claims 结构，移除 `DisplayName` 字段

**数据影响**：

- 回滚后，`created_by_name` 字段数据会丢失
- `created_by` 字段不受影响，可以通过关联查询恢复名称

## 实施计划

### 阶段 1：准备阶段

1. **代码审查**
   - 确认所有涉及创建操作的服务
   - 识别需要修改的代码位置

2. **测试环境准备**
   - 准备测试数据库
   - 配置测试环境

### 阶段 2：开发阶段

1. **数据模型更新**（1 天）
   - 更新 Tenant、User、ChatSession 模型
   - 添加 `created_by_name` 字段定义

2. **JWT Claims 扩展**（0.5 天）
   - 更新 JWTClaims 结构体
   - 修改令牌生成和刷新逻辑

3. **服务层实现**（2 天）
   - 创建辅助函数 `GetCreatorInfoFromContext`
   - 更新租户服务
   - 更新用户服务
   - 更新会话服务

4. **数据库迁移**（1 天）
   - 编写迁移脚本
   - 测试迁移和回滚

5. **单元测试**（1 天）
   - 编写辅助函数测试
   - 编写服务层测试

### 阶段 3：测试阶段

1. **集成测试**（1 天）
   - 端到端测试
   - 数据库迁移测试

2. **性能测试**（0.5 天）
   - 查询性能对比
   - JWT 令牌大小测试

3. **安全测试**（0.5 天）
   - 测试客户端伪造防护
   - 测试数据一致性

### 阶段 4：部署阶段

1. **预发布环境部署**（0.5 天）
   - 执行数据库迁移
   - 部署新代码
   - 验证功能

2. **生产环境部署**（0.5 天）
   - 执行数据库迁移
   - 部署新代码
   - 监控系统运行

### 总计时间：约 8 天

## 监控和维护

### 1. 监控指标

**数据完整性监控**：

```sql
-- 监控 created_by 和 created_by_name 的填充率
SELECT 
    'tenants' as table_name,
    COUNT(*) as total_records,
    COUNT(created_by) as has_created_by,
    COUNT(created_by_name) as has_created_by_name,
    ROUND(COUNT(created_by_name)::numeric / COUNT(*)::numeric * 100, 2) as fill_rate
FROM tenants
WHERE is_deleted = false
UNION ALL
SELECT 
    'users' as table_name,
    COUNT(*) as total_records,
    COUNT(created_by) as has_created_by,
    COUNT(created_by_name) as has_created_by_name,
    ROUND(COUNT(created_by_name)::numeric / COUNT(*)::numeric * 100, 2) as fill_rate
FROM users
WHERE is_deleted = false
UNION ALL
SELECT 
    'chat_sessions' as table_name,
    COUNT(*) as total_records,
    COUNT(created_by) as has_created_by,
    COUNT(created_by_name) as has_created_by_name,
    ROUND(COUNT(created_by_name)::numeric / COUNT(*)::numeric * 100, 2) as fill_rate
FROM chat_sessions
WHERE is_deleted = false;
```

**性能监控**：

- 监控创建操作的响应时间
- 监控列表查询的响应时间
- 对比迁移前后的性能指标

### 2. 日志记录

**关键日志点**：

```go
// 记录无法获取创建者信息的情况
if createdByUUID == nil || createdByName == nil {
    logger.WarnContext(ctx, "无法从 JWT 获取完整的创建者信息",
        "has_user_id", createdByUUID != nil,
        "has_display_name", createdByName != nil,
    )
}
```

### 3. 数据修复

**修复脚本**（可选）：

```sql
-- 如果需要回填历史数据的 created_by_name
UPDATE tenants t
SET created_by_name = u.display_name
FROM users u
WHERE t.created_by = u.id
  AND t.created_by_name IS NULL
  AND t.is_deleted = false;

UPDATE users u1
SET created_by_name = u2.display_name
FROM users u2
WHERE u1.created_by = u2.id
  AND u1.created_by_name IS NULL
  AND u1.is_deleted = false;

UPDATE chat_sessions cs
SET created_by_name = u.display_name
FROM users u
WHERE cs.created_by = u.id
  AND cs.created_by_name IS NULL
  AND cs.is_deleted = false;
```

**注意**：

- 回填脚本是可选的，不是必需的
- 只在需要完整历史数据时执行
- 执行前做好数据备份

### 4. 维护建议

- **定期检查**：每月检查一次数据完整性
- **性能监控**：持续监控查询性能，确保优化效果
- **日志审查**：定期审查警告日志，发现潜在问题
- **文档更新**：保持文档与代码同步

## 总结

本设计文档详细描述了 `created_by_name` 字段的实现方案，包括：

1. **数据模型扩展**：在 Tenant、User、ChatSession 模型中添加 `created_by_name` 字段
2. **JWT Claims 扩展**：在 JWT 令牌中添加 `DisplayName` 字段
3. **服务层实现**：创建统一的辅助函数，从 JWT 中提取创建者信息
4. **数据库迁移**：提供完整的迁移脚本，支持升级和回滚
5. **安全保护**：强制从 JWT 获取创建者信息，防止客户端伪造
6. **性能优化**：通过冗余存储提升查询性能
7. **向后兼容**：新字段允许为 NULL，不影响现有功能

该方案在保证安全性和数据一致性的前提下，提升了系统的查询性能和审计追踪能力。
