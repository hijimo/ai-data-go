# 认证系统数据库迁移

本文档说明如何使用认证系统的数据库迁移。

## 概述

认证系统包含以下数据表：

1. **tenants** - 租户表，存储多租户系统中的租户信息
2. **users** - 用户表，存储租户下的用户账户信息
3. **refresh_tokens** - 刷新令牌表，存储用户的 Refresh Token 信息
4. **auth_audit** - 认证审计日志表，记录所有身份认证相关的操作

## 数据模型

### Tenant（租户）

- ID: 租户唯一标识符（UUID）
- Name: 租户名称
- Domain: 租户域名，用于子域名识别
- Metadata: 租户元数据（JSON格式）
- Status: 租户状态（启用/禁用）
- 时间戳：CreatedAt, UpdatedAt
- 软删除标记：IsDeleted

### User（用户）

- ID: 用户唯一标识符（UUID）
- TenantID: 所属租户ID
- Email: 用户邮箱（租户内唯一）
- PasswordHash: 密码哈希值（bcrypt）
- DisplayName: 显示名称
- Roles: 用户角色列表（JSON格式）
- IsActive: 账户是否激活
- IsAdmin: 是否为管理员
- LastLoginAt: 最后登录时间
- 时间戳：CreatedAt, UpdatedAt
- 软删除标记：IsDeleted

### RefreshToken（刷新令牌）

- ID: 令牌唯一标识符（UUID）
- UserID: 用户ID
- TenantID: 租户ID
- TokenHash: Token 的 SHA256 哈希值
- Revoked: 是否已撤销
- ExpiresAt: 过期时间
- ReplacedBy: 轮换时指向新 token 的 ID
- 时间戳：CreatedAt

### AuthAudit（审计日志）

- ID: 审计日志唯一标识符（UUID）
- TenantID: 租户ID
- UserID: 用户ID
- Event: 事件类型（login, logout, refresh, revoke, failed_login）
- IP: 客户端IP地址
- UserAgent: 用户代理字符串
- Meta: 事件元数据（JSON格式）
- 时间戳：CreatedAt

## 使用方法

### 1. 仅运行认证迁移

```go
import (
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/database/migrations"
)

// 连接数据库
db, err := database.NewPostgresDatabase(config).Connect(ctx)
if err != nil {
    log.Fatal(err)
}

// 运行认证迁移
if err := migrations.RunAuthMigrations(db.GetDB()); err != nil {
    log.Fatal(err)
}
```

### 2. 运行所有迁移

```go
// 运行所有迁移（包括认证和会话管理）
if err := migrations.RunAllMigrations(db.GetDB()); err != nil {
    log.Fatal(err)
}
```

### 3. 使用迁移管理器

```go
manager := migrations.NewMigrationManager(db.GetDB())

// 注册迁移
manager.Register(migrations.NewAuthMigration(db.GetDB()))

// 执行迁移
if err := manager.Up(); err != nil {
    log.Fatal(err)
}

// 回滚迁移
if err := manager.Down(); err != nil {
    log.Fatal(err)
}
```

## 数据库兼容性

### PostgreSQL（推荐）

- 使用 UUID 类型存储 ID
- 使用 JSONB 类型存储 JSON 数据
- 使用 INET 类型存储 IP 地址
- 支持表和列注释
- 自动生成 UUID（gen_random_uuid()）

### SQLite（测试环境）

- 使用 VARCHAR(36) 存储 UUID
- 使用 JSON 类型存储 JSON 数据
- 使用 VARCHAR(45) 存储 IP 地址
- 不支持表和列注释
- 需要应用层生成 UUID

### MySQL（兼容）

- 使用 VARCHAR(36) 存储 UUID
- 使用 JSON 类型存储 JSON 数据
- 使用 VARCHAR(45) 存储 IP 地址
- 支持列注释
- 需要应用层生成 UUID

## 索引说明

### Tenant 表索引

- `idx_tenants_domain`: 域名查询索引
- `idx_tenants_status`: 状态过滤索引
- `idx_tenants_created_at`: 时间排序索引

### User 表索引

- `idx_tenant_email`: 租户+邮箱复合索引（唯一约束）
- `idx_users_tenant_id`: 租户用户查询索引
- `idx_users_email`: 邮箱查询索引
- `idx_users_is_active`: 活跃状态过滤索引
- `idx_users_created_at`: 时间排序索引

### RefreshToken 表索引

- `idx_token_hash`: Token 哈希唯一索引
- `idx_user_tokens`: 用户令牌查询索引
- `idx_tenant_tokens`: 租户令牌查询索引
- `idx_revoked`: 撤销状态过滤索引
- `idx_expires`: 过期时间查询索引

### AuthAudit 表索引

- `idx_tenant_audit`: 租户审计日志查询索引
- `idx_user_audit`: 用户审计日志查询索引
- `idx_event`: 事件类型过滤索引
- `idx_created_at`: 时间排序索引

## 注意事项

1. **UUID 生成**：在 PostgreSQL 中，ID 字段会自动生成 UUID。在其他数据库中，需要在应用层使用 `google/uuid` 包生成 UUID。

2. **租户隔离**：所有用户相关的查询都必须包含 `tenant_id` 过滤条件，确保数据隔离。

3. **软删除**：使用 `is_deleted` 字段实现软删除，查询时需要过滤已删除的记录。

4. **密码安全**：密码必须使用 bcrypt 算法哈希后存储，永远不要存储明文密码。

5. **Token 安全**：Refresh Token 必须使用 SHA256 哈希后存储，原始 Token 只在生成时返回给客户端。

6. **审计日志**：所有认证相关的操作都应该记录审计日志，包括成功和失败的操作。

## 测试

运行迁移测试：

```bash
# 运行所有认证迁移测试
go test -v ./internal/database/migrations -run TestAuthMigration

# 运行特定测试
go test -v ./internal/database/migrations -run TestAuthMigration_Up
go test -v ./internal/database/migrations -run TestRunAuthMigrations
```

## 相关文件

- `internal/model/auth.go` - 认证相关的数据模型定义
- `internal/model/jwt.go` - JWT Claims 结构定义
- `internal/database/migrations/auth_migration.go` - 认证迁移实现
- `internal/database/migrations/auth_migration_test.go` - 迁移测试
- `internal/database/migrations/migration_manager.go` - 迁移管理器
