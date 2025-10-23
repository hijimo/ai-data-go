# 时间戳字段修复指南

## 问题描述

在创建租户时，`created_at`、`updated_at` 和 `created_by` 字段没有被正确记录。

## 问题原因

1. **数据库层面**：`tenants` 和 `users` 表的 `created_at` 和 `updated_at` 字段缺少 `DEFAULT CURRENT_TIMESTAMP` 约束
2. **应用层面**：创建租户时没有从 JWT claims 中获取当前用户ID来设置 `created_by` 字段

## 解决方案

### 1. 数据库修复（必须执行）

运行以下命令修复数据库表结构：

```bash
./fix_timestamps.sh
```

或者直接运行 Go 脚本：

```bash
go run scripts/fix_timestamps.go
```

这个脚本会：

- 为 `tenants` 表的 `created_at` 和 `updated_at` 字段添加 `DEFAULT CURRENT_TIMESTAMP`
- 为 `users` 表的 `created_at` 和 `updated_at` 字段添加 `DEFAULT CURRENT_TIMESTAMP`
- 为已存在但时间戳为空的记录设置当前时间

### 2. 代码修复（已完成）

已更新以下文件：

#### 新增迁移文件

- `internal/database/migrations/fix_timestamps_migration.go` - 时间戳修复迁移

#### 更新的文件

- `internal/service/auth/tenant_service.go` - 在创建租户时自动从 JWT claims 获取当前用户ID设置 `created_by`
- `internal/database/migrations/migration_manager.go` - 注册时间戳修复迁移

## 验证修复

### 1. 检查数据库表结构

```sql
-- 查看 tenants 表结构
\d tenants

-- 应该看到：
-- created_at | timestamp with time zone | default CURRENT_TIMESTAMP
-- updated_at | timestamp with time zone | default CURRENT_TIMESTAMP
```

### 2. 测试创建租户

```bash
# 使用平台管理员账户登录并创建租户
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试租户",
    "domain": "test.example.com"
  }'
```

检查返回的租户信息，应该包含：

- `createdAt`: 创建时间（自动生成）
- `updatedAt`: 更新时间（自动生成）
- `createdBy`: 创建者用户ID（从 JWT token 中获取）

## 注意事项

1. **必须先运行数据库修复脚本**，否则新创建的记录仍然不会有时间戳
2. **已存在的记录**会被更新为当前时间（如果时间戳为空）
3. **创建者字段**只有在用户已认证的情况下才会被设置
4. 如果是通过 bootstrap 初始化创建的平台租户，`created_by` 可能为空（这是正常的）

## 回滚

如果需要回滚数据库修改：

```go
// 在 Go 代码中执行
migration := migrations.NewFixTimestampsMigration(db)
err := migration.Down()
```

这会移除时间戳字段的默认值约束。

## 相关文件

- 迁移脚本：`internal/database/migrations/fix_timestamps_migration.go`
- 执行脚本：`scripts/fix_timestamps.go`
- Shell 脚本：`fix_timestamps.sh`
- 服务层修复：`internal/service/auth/tenant_service.go`
