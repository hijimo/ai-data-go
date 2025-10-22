# 邮箱全局唯一性迁移指南

## 概述

本次迁移将用户邮箱从"租户内唯一"改为"全局唯一"，简化登录流程。

## 变更内容

### 1. 数据库变更

- **删除索引**: `idx_tenant_email` (租户+邮箱联合唯一索引)
- **新增索引**: `idx_users_email_unique` (邮箱全局唯一索引)
- **更新注释**: 邮箱字段注释从"租户内唯一"改为"全局唯一"

### 2. 代码变更

#### UserRepository 接口

新增方法：

```go
// GetByEmailOnly 仅根据邮箱获取用户（不需要租户ID）
GetByEmailOnly(ctx context.Context, email string) (*model.User, error)
```

#### LoginRequest 结构

**变更前**:

```go
type LoginRequest struct {
    TenantID string `json:"tenantId"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

**变更后**:

```go
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

#### 登录逻辑

- 使用 `GetByEmailOnly` 方法通过邮箱查找用户
- 从用户记录中获取租户ID，无需客户端提供
- 简化了登录流程，提升用户体验

## 执行迁移

### 前提条件

1. 确保数据库中没有重复的邮箱地址
2. 备份数据库（推荐）

### 检查重复邮箱

在执行迁移前，运行以下SQL检查是否存在重复邮箱：

```sql
SELECT email, COUNT(*) as count
FROM users
WHERE is_deleted = false
GROUP BY email
HAVING COUNT(*) > 1;
```

如果有重复邮箱，需要先手动处理。

### 执行迁移命令

```bash
go run scripts/migrate_email_unique.go
```

### 迁移输出示例

```
=== 邮箱全局唯一性迁移工具 ===

📡 正在连接数据库...
✅ 数据库连接成功

🔄 开始执行邮箱全局唯一性迁移...

✅ 邮箱全局唯一性迁移成功完成！

变更内容：
  - 删除了 idx_tenant_email 索引（租户+邮箱联合唯一）
  - 创建了 idx_users_email_unique 索引（邮箱全局唯一）
  - 更新了邮箱字段的注释

💡 提示：现在用户登录时只需要提供邮箱和密码，不再需要租户ID
```

## API 变更

### 登录接口

**端点**: `POST /auth/login`

**变更前的请求体**:

```json
{
  "tenantId": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "password": "password123"
}
```

**变更后的请求体**:

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应**: 保持不变

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "accessToken": "eyJhbGc...",
    "refreshToken": "550e8400-...",
    "expiresIn": 3600,
    "tokenType": "Bearer",
    "user": {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "tenantId": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      ...
    }
  }
}
```

## 回滚

如果需要回滚到租户内唯一，可以使用以下SQL：

```sql
BEGIN;

-- 删除全局唯一索引
DROP INDEX IF EXISTS idx_users_email_unique;

-- 恢复租户+邮箱联合唯一索引
CREATE UNIQUE INDEX idx_tenant_email ON users(tenant_id, email) WHERE NOT is_deleted;

-- 恢复列注释
COMMENT ON COLUMN users.email IS '用户邮箱地址（租户内唯一）';

COMMIT;
```

## 注意事项

1. **邮箱唯一性**: 迁移后，同一个邮箱不能在多个租户中注册
2. **现有数据**: 如果现有数据中有重复邮箱，迁移会失败
3. **注册流程**: 注册时仍需提供租户ID，只有登录时不需要
4. **向后兼容**: 旧的 `GetByEmail` 方法仍然保留，用于需要租户隔离的场景

## 优势

1. **简化登录**: 用户只需记住邮箱和密码
2. **更好的用户体验**: 不需要知道租户ID
3. **符合常规**: 大多数应用都是使用邮箱作为唯一标识
4. **安全性**: 邮箱全局唯一可以防止账户混淆

## 测试

### 测试登录

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### 测试重复邮箱注册

尝试在不同租户注册相同邮箱，应该返回错误：

```bash
# 第一次注册（成功）
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-1",
    "email": "test@example.com",
    "password": "password123"
  }'

# 第二次注册相同邮箱（失败）
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-2",
    "email": "test@example.com",
    "password": "password123"
  }'
```

预期响应：

```json
{
  "code": 400,
  "message": "邮箱已被注册"
}
```

## 相关文件

- `internal/repository/user_repository.go` - 新增 GetByEmailOnly 方法
- `internal/service/auth/auth_service.go` - 更新登录和注册逻辑
- `internal/api/handler/auth_handler.go` - 更新 LoginRequest 结构
- `internal/model/auth.go` - 更新邮箱字段索引定义
- `internal/database/migrations/email_unique_migration.go` - 迁移脚本
- `scripts/migrate_email_unique.go` - 迁移执行工具
