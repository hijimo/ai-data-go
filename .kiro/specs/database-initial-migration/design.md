# 设计文档

## 概述

本设计文档描述了如何将现有的分散数据库迁移整合为统一的初始迁移基线，并确保ORM模型定义与数据库结构完全一致。该设计遵循PostgreSQL最佳实践，使用UUID作为主键类型，并提供完整的迁移管理机制。

## 架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     应用启动层                                │
│                  (cmd/server/main.go)                        │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   迁移管理层                                  │
│         (internal/database/migrations/)                      │
│  ┌──────────────────┐  ┌──────────────────┐                │
│  │ MigrationManager │  │ InitialMigration │                │
│  └──────────────────┘  └──────────────────┘                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   数据库连接层                                │
│              (internal/database/postgres.go)                 │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   PostgreSQL数据库                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │ 认证表   │  │ 会话表   │  │ 审计表   │                  │
│  └──────────┘  └──────────┘  └──────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### 迁移执行流程

```text
应用启动
   │
   ▼
加载配置
   │
   ▼
连接数据库
   │
   ▼
检查UUID扩展
   │
   ▼
创建MigrationManager
   │
   ▼
注册InitialMigration
   │
   ▼
执行迁移（Up方法）
   │
   ├─ 创建tenants表
   ├─ 创建users表
   ├─ 创建refresh_tokens表
   ├─ 创建email_verification_tokens表
   ├─ 创建auth_audit表
   ├─ 创建chat_sessions表
   ├─ 创建chat_messages表
   └─ 创建chat_summaries表
   │
   ▼
验证表结构
   │
   ▼
应用正常启动
```

## 组件和接口

### 1. InitialMigration 结构

```go
// InitialMigration 初始迁移结构
type InitialMigration struct {
    db *gorm.DB
}

// NewInitialMigration 创建初始迁移实例
func NewInitialMigration(db *gorm.DB) *InitialMigration

// Up 执行迁移（创建表）
func (m *InitialMigration) Up() error

// Down 回滚迁移（删除表）
func (m *InitialMigration) Down() error

// Name 返回迁移名称
func (m *InitialMigration) Name() string
```

**职责：**

- 定义完整的数据库表结构
- 按正确顺序创建所有表
- 创建必要的索引和约束
- 添加表和列的注释
- 提供回滚功能

### 2. MigrationManager 更新

```go
// MigrationManager 迁移管理器
type MigrationManager struct {
    db         *gorm.DB
    migrations []Migration
}

// RegisterInitialMigration 注册初始迁移
func (m *MigrationManager) RegisterInitialMigration()

// RunAllMigrations 执行所有迁移
func (m *MigrationManager) RunAllMigrations() error

// RunInitialMigration 单独执行初始迁移
func RunInitialMigration(db *gorm.DB) error
```

**职责：**

- 管理所有迁移的注册和执行
- 确保迁移按正确顺序执行
- 提供错误处理和日志记录

### 3. 数据模型更新

#### Tenant 模型

```go
type Tenant struct {
    ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    Name        string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
    Description string         `gorm:"type:text" json:"description"`
    IsActive    bool           `gorm:"default:true" json:"isActive"`
    CreatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}
```

#### User 模型

```go
type User struct {
    ID                uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    TenantID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenantId"`
    Username          string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_tenant_username" json:"username"`
    Email             string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_tenant_email" json:"email"`
    PasswordHash      string         `gorm:"type:varchar(255);not null" json:"-"`
    Role              string         `gorm:"type:varchar(50);default:'user'" json:"role"`
    IsActive          bool           `gorm:"default:true" json:"isActive"`
    EmailVerified     bool           `gorm:"default:false" json:"emailVerified"`
    FailedLoginCount  int            `gorm:"default:0" json:"failedLoginCount"`
    LastFailedLoginAt *time.Time     `json:"lastFailedLoginAt,omitempty"`
    CreatedAt         time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt         time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    DeletedAt         gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
    
    Tenant            Tenant         `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"tenant,omitempty"`
}
```

#### RefreshToken 模型

```go
type RefreshToken struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
    Token     string         `gorm:"type:varchar(500);not null;uniqueIndex" json:"token"`
    ExpiresAt time.Time      `gorm:"not null;index" json:"expiresAt"`
    CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    RevokedAt *time.Time     `json:"revokedAt,omitempty"`
    
    User      User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}
```

#### EmailVerificationToken 模型

```go
type EmailVerificationToken struct {
    ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
    Token     string     `gorm:"type:varchar(500);not null;uniqueIndex" json:"token"`
    ExpiresAt time.Time  `gorm:"not null;index" json:"expiresAt"`
    CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UsedAt    *time.Time `json:"usedAt,omitempty"`
    
    User      User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}
```

#### AuthAudit 模型

```go
type AuthAudit struct {
    ID        uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID    *uuid.UUID        `gorm:"type:uuid;index" json:"userId,omitempty"`
    TenantID  *uuid.UUID        `gorm:"type:uuid;index" json:"tenantId,omitempty"`
    Action    string            `gorm:"type:varchar(100);not null;index" json:"action"`
    Resource  string            `gorm:"type:varchar(255)" json:"resource"`
    Status    string            `gorm:"type:varchar(50);not null" json:"status"`
    IPAddress string            `gorm:"type:varchar(45)" json:"ipAddress"`
    UserAgent string            `gorm:"type:text" json:"userAgent"`
    Details   datatypes.JSON    `gorm:"type:jsonb" json:"details"`
    CreatedAt time.Time         `gorm:"default:CURRENT_TIMESTAMP;index" json:"createdAt"`
}
```

#### ChatSession 模型

```go
type ChatSession struct {
    ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
    TenantID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenantId"`
    Title       string         `gorm:"type:varchar(500)" json:"title"`
    ModelName   string         `gorm:"type:varchar(100)" json:"modelName"`
    IsActive    bool           `gorm:"default:true" json:"isActive"`
    CreatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP;index" json:"createdAt"`
    UpdatedAt   time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
    
    User        User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
    Tenant      Tenant         `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"tenant,omitempty"`
}
```

#### ChatMessage 模型

```go
type ChatMessage struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    SessionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"sessionId"`
    Role      string         `gorm:"type:varchar(50);not null" json:"role"`
    Content   string         `gorm:"type:text;not null" json:"content"`
    Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP;index" json:"createdAt"`
    
    Session   ChatSession    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"session,omitempty"`
}
```

#### ChatSummary 模型

```go
type ChatSummary struct {
    ID        uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    SessionID uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex" json:"sessionId"`
    Summary   string      `gorm:"type:text" json:"summary"`
    CreatedAt time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
    
    Session   ChatSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"session,omitempty"`
}
```

## 数据模型

### 表结构关系图

```text
┌─────────────┐
│   tenants   │
└──────┬──────┘
       │
       │ 1:N
       │
       ▼
┌─────────────┐         ┌──────────────────┐
│    users    │◄────────│ refresh_tokens   │
└──────┬──────┘    1:N  └──────────────────┘
       │
       │ 1:N            ┌──────────────────────────┐
       ├────────────────│ email_verification_tokens│
       │                └──────────────────────────┘
       │
       │ 1:N            ┌──────────────┐
       ├────────────────│  auth_audit  │
       │                └──────────────┘
       │
       │ 1:N
       ▼
┌──────────────┐
│chat_sessions │
└──────┬───────┘
       │
       │ 1:N            ┌──────────────┐
       ├────────────────│chat_messages │
       │                └──────────────┘
       │
       │ 1:1            ┌──────────────┐
       └────────────────│chat_summaries│
                        └──────────────┘
```

### 表详细定义

#### 1. tenants 表（租户表）

| 字段名      | 类型                      | 可空 | 默认值              | 说明         |
|-------------|---------------------------|------|---------------------|--------------|
| id          | UUID                      | NO   | gen_random_uuid()   | 租户ID（主键）|
| name        | VARCHAR(255)              | NO   | -                   | 租户名称     |
| domain      | VARCHAR(255)              | YES  | -                   | 租户域名     |
| metadata    | JSONB                     | YES  | -                   | 租户元数据   |
| status      | BOOLEAN                   | YES  | true                | 租户状态     |
| created_at  | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 创建时间     |
| updated_at  | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 更新时间     |
| created_by  | VARCHAR(36)               | YES  | -                   | 创建者ID     |
| is_deleted  | BOOLEAN                   | YES  | false               | 软删除标记   |

**索引：**

- PRIMARY KEY: tenants_pkey (id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, name)

#### 2. users 表（用户表）

| 字段名                 | 类型                      | 可空 | 默认值              | 说明             |
|------------------------|---------------------------|------|---------------------|------------------|
| id                     | UUID                      | NO   | gen_random_uuid()   | 用户ID（主键）   |
| tenant_id              | UUID                      | NO   | -                   | 租户ID（外键）   |
| email                  | VARCHAR(320)              | NO   | -                   | 邮箱地址         |
| email_verified         | BOOLEAN                   | YES  | false               | 邮箱是否验证     |
| phone                  | VARCHAR(20)               | YES  | -                   | 手机号码         |
| password_hash          | TEXT                      | NO   | -                   | 密码哈希         |
| display_name           | VARCHAR(255)              | YES  | -                   | 显示名称         |
| is_active              | BOOLEAN                   | YES  | true                | 是否激活         |
| is_admin               | BOOLEAN                   | YES  | false               | 是否管理员       |
| roles                  | JSONB                     | YES  | -                   | 角色列表         |
| meta                   | JSONB                     | YES  | -                   | 用户元数据       |
| last_login_at          | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 最后登录时间     |
| created_at             | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 创建时间         |
| updated_at             | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 更新时间         |
| created_by             | VARCHAR(36)               | YES  | -                   | 创建者ID         |
| is_deleted             | BOOLEAN                   | YES  | false               | 软删除标记       |
| failed_login_attempts  | BIGINT                    | YES  | 0                   | 失败登录次数     |
| locked_until           | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 锁定截止时间     |

**索引：**

- PRIMARY KEY: users_pkey (id)
- UNIQUE INDEX: idx_tenant_email (tenant_id, email) - 租户内邮箱唯一

**外键：**

- FOREIGN KEY: fk_users_tenant (tenant_id) REFERENCES tenants(id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, tenant_id, email, password_hash)

#### 3. refresh_tokens 表（刷新令牌表）

| 字段名       | 类型                      | 可空 | 默认值              | 说明             |
|--------------|---------------------------|------|---------------------|------------------|
| id           | UUID                      | NO   | gen_random_uuid()   | 令牌ID（主键）   |
| user_id      | UUID                      | NO   | -                   | 用户ID（外键）   |
| tenant_id    | UUID                      | NO   | -                   | 租户ID（外键）   |
| token_hash   | TEXT                      | NO   | -                   | 令牌哈希值       |
| revoked      | BOOLEAN                   | YES  | false               | 是否已撤销       |
| created_at   | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 创建时间         |
| expires_at   | TIMESTAMP WITH TIME ZONE  | NO   | -                   | 过期时间         |
| replaced_by  | VARCHAR(36)               | YES  | -                   | 替换令牌ID       |

**索引：**

- PRIMARY KEY: refresh_tokens_pkey (id)
- UNIQUE INDEX: idx_token_hash (token_hash) - 令牌哈希唯一
- INDEX: idx_user_tokens (user_id) - 用户令牌查询
- INDEX: idx_tenant_tokens (tenant_id) - 租户令牌查询
- INDEX: idx_expires (expires_at) - 过期时间查询
- INDEX: idx_revoked (revoked) - 撤销状态查询

**外键：**

- FOREIGN KEY: fk_refresh_tokens_user (user_id) REFERENCES users(id)
- FOREIGN KEY: fk_refresh_tokens_tenant (tenant_id) REFERENCES tenants(id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, user_id, tenant_id, token_hash, expires_at)

#### 4. email_verification_tokens 表（邮箱验证令牌表）

| 字段名       | 类型                      | 可空 | 默认值              | 说明             |
|--------------|---------------------------|------|---------------------|------------------|
| id           | UUID                      | NO   | gen_random_uuid()   | 令牌ID（主键）   |
| user_id      | UUID                      | NO   | -                   | 用户ID（外键）   |
| tenant_id    | UUID                      | NO   | -                   | 租户ID（外键）   |
| token        | VARCHAR(64)               | NO   | -                   | 验证令牌         |
| email        | VARCHAR(320)              | NO   | -                   | 待验证邮箱       |
| used         | BOOLEAN                   | YES  | false               | 是否已使用       |
| created_at   | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 创建时间         |
| expires_at   | TIMESTAMP WITH TIME ZONE  | NO   | -                   | 过期时间         |

**索引：**

- PRIMARY KEY: email_verification_tokens_pkey (id)
- UNIQUE INDEX: idx_verification_token (token) - 令牌唯一
- INDEX: idx_user_verification (user_id) - 用户验证令牌查询
- INDEX: idx_tenant_verification (tenant_id) - 租户验证令牌查询
- INDEX: idx_verification_expires (expires_at) - 过期时间查询
- INDEX: idx_used (used) - 使用状态查询

**外键：**

- FOREIGN KEY: fk_email_verification_tokens_user (user_id) REFERENCES users(id)
- FOREIGN KEY: fk_email_verification_tokens_tenant (tenant_id) REFERENCES tenants(id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, user_id, tenant_id, token, email, expires_at)

#### 5. auth_audit 表（认证审计表）

| 字段名       | 类型                      | 可空 | 默认值              | 说明             |
|--------------|---------------------------|------|---------------------|------------------|
| id           | UUID                      | NO   | gen_random_uuid()   | 审计ID（主键）   |
| tenant_id    | UUID                      | YES  | -                   | 租户ID           |
| user_id      | UUID                      | YES  | -                   | 用户ID           |
| event        | VARCHAR(64)               | NO   | -                   | 事件类型         |
| ip           | VARCHAR(45)               | YES  | -                   | IP地址           |
| user_agent   | TEXT                      | YES  | -                   | 用户代理         |
| meta         | JSONB                     | YES  | -                   | 元数据           |
| created_at   | TIMESTAMP WITH TIME ZONE  | YES  | -                   | 创建时间         |

**索引：**

- PRIMARY KEY: auth_audit_pkey (id)
- INDEX: idx_tenant_audit (tenant_id) - 租户审计查询
- INDEX: idx_user_audit (user_id) - 用户审计查询
- INDEX: idx_event (event) - 事件类型查询
- INDEX: idx_created_at (created_at) - 时间范围查询

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, event)

#### 6. chat_sessions 表（聊天会话表）

| 字段名           | 类型                      | 可空 | 默认值              | 说明             |
|------------------|---------------------------|------|---------------------|------------------|
| id               | UUID                      | NO   | gen_random_uuid()   | 会话ID（主键）   |
| user_id          | UUID                      | NO   | -                   | 用户ID           |
| title            | VARCHAR(255)              | NO   | -                   | 会话标题         |
| model_name       | VARCHAR(128)              | NO   | -                   | 模型名称         |
| system_prompt    | TEXT                      | YES  | -                   | 系统提示词       |
| temperature      | NUMERIC                   | YES  | -                   | 温度参数         |
| top_p            | NUMERIC                   | YES  | -                   | Top-P参数        |
| created_by       | UUID                      | NO   | -                   | 创建者ID         |
| created_at       | TIMESTAMP WITH TIME ZONE  | NO   | CURRENT_TIMESTAMP   | 创建时间         |
| updated_at       | TIMESTAMP WITH TIME ZONE  | NO   | CURRENT_TIMESTAMP   | 更新时间         |
| last_message_id  | UUID                      | YES  | -                   | 最后消息ID       |
| message_count    | BIGINT                    | YES  | 0                   | 消息数量         |
| is_pinned        | BOOLEAN                   | YES  | false               | 是否置顶         |
| is_archived      | BOOLEAN                   | YES  | false               | 是否归档         |
| is_deleted       | BOOLEAN                   | YES  | false               | 软删除标记       |
| meta             | JSONB                     | YES  | -                   | 元数据           |

**索引：**

- PRIMARY KEY: chat_sessions_pkey (id)
- INDEX: idx_user_sessions (user_id) - 用户会话查询
- INDEX: idx_pinned (is_pinned) - 置顶会话查询
- INDEX: idx_archived (is_archived) - 归档会话查询
- INDEX: idx_deleted (is_deleted) - 删除状态查询

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, user_id, title, model_name, created_by, created_at, updated_at)

#### 7. chat_messages 表（聊天消息表）

| 字段名       | 类型                      | 可空 | 默认值              | 说明             |
|--------------|---------------------------|------|---------------------|------------------|
| id           | UUID                      | NO   | gen_random_uuid()   | 消息ID（主键）   |
| session_id   | UUID                      | NO   | -                   | 会话ID（外键）   |
| role         | VARCHAR(32)               | NO   | -                   | 角色（user/assistant/system）|
| content      | TEXT                      | NO   | -                   | 消息内容         |
| tokens       | BIGINT                    | YES  | 0                   | Token数量        |
| created_at   | TIMESTAMP WITH TIME ZONE  | NO   | CURRENT_TIMESTAMP   | 创建时间         |
| sequence     | BIGINT                    | NO   | -                   | 消息序号         |
| tool_calls   | JSONB                     | YES  | -                   | 工具调用信息     |
| error        | TEXT                      | YES  | -                   | 错误信息         |
| parent_id    | UUID                      | YES  | -                   | 父消息ID         |
| meta         | JSONB                     | YES  | -                   | 元数据           |

**索引：**

- PRIMARY KEY: chat_messages_pkey (id)
- INDEX: idx_session_messages (session_id) - 会话消息查询
- INDEX: idx_created (created_at) - 时间范围查询

**外键：**

- FOREIGN KEY: fk_chat_messages_session (session_id) REFERENCES chat_sessions(id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, session_id, role, content, created_at, sequence)

#### 8. chat_summaries 表（聊天摘要表）

| 字段名           | 类型                      | 可空 | 默认值              | 说明             |
|------------------|---------------------------|------|---------------------|------------------|
| id               | UUID                      | NO   | gen_random_uuid()   | 摘要ID（主键）   |
| session_id       | UUID                      | NO   | -                   | 会话ID（外键）   |
| summary          | TEXT                      | NO   | -                   | 摘要内容         |
| last_message_id  | UUID                      | NO   | -                   | 最后消息ID       |
| token_count      | BIGINT                    | YES  | 0                   | Token数量        |
| created_at       | TIMESTAMP WITH TIME ZONE  | NO   | CURRENT_TIMESTAMP   | 创建时间         |

**索引：**

- PRIMARY KEY: chat_summaries_pkey (id)
- INDEX: idx_session_summary (session_id) - 会话摘要查询

**外键：**

- FOREIGN KEY: fk_chat_summaries_session (session_id) REFERENCES chat_sessions(id)

**约束：**

- PRIMARY KEY (id)
- NOT NULL 约束 (id, session_id, summary, last_message_id, created_at)

## 错误处理

### 错误类型

1. **连接错误**
   - 数据库连接失败
   - 网络超时
   - 认证失败

2. **迁移错误**
   - 表已存在
   - 外键约束冲突
   - 索引创建失败
   - SQL语法错误

3. **配置错误**
   - 缺少必要的环境变量
   - 数据库URL格式错误
   - 不支持的数据库类型

### 错误处理策略

```go
// 迁移执行错误处理
func (m *InitialMigration) Up() error {
    // 使用事务确保原子性
    return m.db.Transaction(func(tx *gorm.DB) error {
        // 1. 启用UUID扩展
        if err := enableUUIDExtension(tx); err != nil {
            return fmt.Errorf("启用UUID扩展失败: %w", err)
        }
        
        // 2. 按顺序创建表
        tables := []struct {
            name string
            sql  string
        }{
            {"tenants", createTenantsTableSQL},
            {"users", createUsersTableSQL},
            // ... 其他表
        }
        
        for _, table := range tables {
            if err := tx.Exec(table.sql).Error; err != nil {
                return fmt.Errorf("创建表 %s 失败: %w", table.name, err)
            }
        }
        
        return nil
    })
}
```

### 错误日志

```go
// 记录详细的错误信息
logger.Error("迁移失败",
    zap.String("migration", m.Name()),
    zap.Error(err),
    zap.String("database", dbConfig.Type),
)
```

## 测试策略

### 单元测试

1. **InitialMigration 测试**
   - 测试Up方法创建所有表
   - 测试Down方法删除所有表
   - 测试表结构正确性
   - 测试索引创建
   - 测试外键约束

2. **MigrationManager 测试**
   - 测试迁移注册
   - 测试迁移执行顺序
   - 测试错误处理

### 集成测试

1. **完整迁移流程测试**
   - 在空数据库上执行迁移
   - 验证所有表都已创建
   - 验证所有索引都已创建
   - 验证所有外键约束都已创建

2. **回滚测试**
   - 执行Up后执行Down
   - 验证所有表都已删除
   - 再次执行Up验证可重复性

3. **数据完整性测试**
   - 测试外键约束生效
   - 测试级联删除
   - 测试唯一约束

### 测试环境

```go
// 使用测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
    dsn := "host=localhost user=test password=test dbname=test_db port=5432 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    require.NoError(t, err)
    
    // 清理测试数据
    t.Cleanup(func() {
        migration := NewInitialMigration(db)
        _ = migration.Down()
    })
    
    return db
}
```

### 测试覆盖率目标

- 单元测试覆盖率：≥ 80%
- 集成测试覆盖率：≥ 90%
- 关键路径测试覆盖率：100%

## 实施计划

### 阶段1：准备工作

1. 备份现有迁移文件
2. 分析现有表结构
3. 确认所有依赖关系

### 阶段2：创建初始迁移

1. 创建 initial_migration.go 文件
2. 实现 Up 方法（创建所有表）
3. 实现 Down 方法（删除所有表）
4. 添加表注释

### 阶段3：更新模型定义

1. 更新 internal/model/auth.go
2. 更新 internal/model/session.go
3. 确保所有字段类型正确
4. 添加正确的GORM标签

### 阶段4：更新迁移管理器

1. 在 MigrationManager 中注册初始迁移
2. 实现 RunInitialMigration 函数
3. 更新错误处理逻辑

### 阶段5：创建脚本

1. 创建 scripts/init_migration.go
2. 创建 scripts/reset_db.go
3. 添加使用说明

### 阶段6：集成到应用

1. 在 cmd/server/main.go 中集成迁移调用
2. 添加启动检查逻辑
3. 更新错误处理

### 阶段7：测试

1. 编写单元测试
2. 编写集成测试
3. 在测试环境验证

### 阶段8：文档更新

1. 更新 database-migration-guide.md
2. 添加使用示例
3. 添加常见问题解答

## 配置管理

### 环境变量

```bash
# .env.example
DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
DATABASE_TYPE=postgres
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=user
DATABASE_PASSWORD=password
DATABASE_NAME=dbname
DATABASE_SSLMODE=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m
```

### 配置结构

```go
type DatabaseConfig struct {
    Type            string
    Host            string
    Port            int
    User            string
    Password        string
    Name            string
    SSLMode         string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}
```

## 性能考虑

### 索引策略

1. **主键索引**：所有表的UUID主键自动创建索引
2. **外键索引**：所有外键字段创建索引以提高JOIN性能
3. **查询索引**：根据常见查询模式创建复合索引
4. **时间索引**：created_at 字段创建索引支持时间范围查询

### 连接池配置

```go
// 推荐配置
MaxOpenConns:    25  // 最大打开连接数
MaxIdleConns:    5   // 最大空闲连接数
ConnMaxLifetime: 5m  // 连接最大生命周期
```

### 查询优化

1. 使用预编译语句
2. 避免N+1查询问题
3. 合理使用事务
4. 使用连接池

## 安全考虑

### 1. SQL注入防护

- 使用参数化查询
- 使用GORM的安全API
- 避免字符串拼接SQL

### 2. 密码安全

- 使用bcrypt哈希
- 不在日志中记录密码
- 密码字段使用 `json:"-"` 标签

### 3. 数据访问控制

- 实施租户隔离
- 使用外键约束
- 实施软删除

### 4. 审计日志

- 记录所有认证操作
- 记录敏感数据访问
- 保留足够的审计信息

## 维护和监控

### 迁移版本管理

```go
// 迁移版本信息
const (
    InitialMigrationVersion = "v1.0.0"
    InitialMigrationDate    = "2025-01-20"
)
```

### 监控指标

1. **迁移执行时间**
2. **迁移成功率**
3. **数据库连接状态**
4. **表大小和增长趋势**

### 备份策略

1. 迁移前自动备份
2. 定期全量备份
3. 增量备份策略
4. 备份验证机制

## 回滚计划

### 回滚触发条件

1. 迁移执行失败
2. 数据完整性检查失败
3. 应用启动失败
4. 性能严重下降

### 回滚步骤

```go
// 执行回滚
func rollback(db *gorm.DB) error {
    migration := NewInitialMigration(db)
    if err := migration.Down(); err != nil {
        return fmt.Errorf("回滚失败: %w", err)
    }
    
    // 恢复备份（如果需要）
    if err := restoreBackup(); err != nil {
        return fmt.Errorf("恢复备份失败: %w", err)
    }
    
    return nil
}
```

## 附录

### A. SQL脚本示例

```sql
-- 启用UUID扩展
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 创建tenants表
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

COMMENT ON TABLE tenants IS '租户表';
COMMENT ON COLUMN tenants.id IS '租户ID';
COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.description IS '租户描述';
COMMENT ON COLUMN tenants.is_active IS '是否激活';
```

### B. 常见问题

**Q: 为什么使用UUID而不是自增ID？**
A: UUID提供全局唯一性，适合分布式系统，且不会暴露记录数量信息。

**Q: 如何处理现有数据？**
A: 初始迁移假设在空数据库上执行。如有现有数据，需要单独的数据迁移脚本。

**Q: 迁移失败如何处理？**
A: 使用事务确保原子性，失败时自动回滚。可以通过Down方法手动清理。

**Q: 如何添加新的迁移？**
A: 创建新的迁移文件，在MigrationManager中注册，确保版本号递增。
