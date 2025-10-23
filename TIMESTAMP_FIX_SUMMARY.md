# 租户时间戳字段修复总结

## 问题

创建租户时，以下字段没有被正确记录：

- `created_at` - 创建时间
- `updated_at` - 更新时间  
- `created_by` - 创建者用户ID

## 根本原因

### 1. 数据库层面

数据库表定义中 `created_at` 和 `updated_at` 字段缺少 `DEFAULT CURRENT_TIMESTAMP` 约束，导致 GORM 在插入记录时这些字段为 NULL。

### 2. 应用层面

服务层代码在创建租户时没有从 JWT claims 中获取当前用户ID来设置 `created_by` 字段。

## 解决方案

### 修复内容

#### 1. 数据库迁移（新增）

- **文件**: `internal/database/migrations/fix_timestamps_migration.go`
- **功能**:
  - 为 `tenants` 和 `users` 表的时间戳字段添加 `DEFAULT CURRENT_TIMESTAMP`
  - 为已存在但时间戳为空的记录设置当前时间

#### 2. 迁移管理器（更新）

- **文件**: `internal/database/migrations/migration_manager.go`
- **修改**: 在 `RunAllMigrations` 中注册时间戳修复迁移

#### 3. 租户服务（更新）

- **文件**: `internal/service/auth/tenant_service.go`
- **修改**:
  - `Create` 方法：从 JWT claims 自动获取当前用户ID设置 `created_by`
  - `CreateWithAdmin` 方法：同样从 JWT claims 获取创建者ID

#### 4. 执行脚本（新增）

- **文件**: `scripts/fix_timestamps.go` - Go 执行脚本
- **文件**: `fix_timestamps.sh` - Shell 包装脚本
- **文件**: `test_timestamp_fix.sh` - 验证脚本

#### 5. 文档（新增）

- **文件**: `TIMESTAMP_FIX_README.md` - 详细修复指南
- **文件**: `TIMESTAMP_FIX_SUMMARY.md` - 本文档

## 使用方法

### 1. 执行数据库修复（必须）

```bash
# 方式1：使用 shell 脚本
./fix_timestamps.sh

# 方式2：直接运行 Go 脚本
go run scripts/fix_timestamps.go
```

### 2. 验证修复

```bash
# 运行验证脚本
./test_timestamp_fix.sh
```

### 3. 测试创建租户

```bash
# 使用 API 创建租户
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试租户",
    "domain": "test.example.com"
  }'
```

检查返回结果应包含：

```json
{
  "code": 200,
  "message": "创建租户成功",
  "data": {
    "id": "...",
    "name": "测试租户",
    "domain": "test.example.com",
    "createdAt": "2025-01-23T10:30:00Z",  // ✅ 自动生成
    "updatedAt": "2025-01-23T10:30:00Z",  // ✅ 自动生成
    "createdBy": "550e8400-e29b-41d4-a716-446655440000"  // ✅ 从 JWT 获取
  }
}
```

## 技术细节

### 数据库修改

```sql
-- 为时间戳字段添加默认值
ALTER TABLE tenants 
  ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP,
  ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

-- 更新已存在的空值记录
UPDATE tenants 
SET created_at = CURRENT_TIMESTAMP 
WHERE created_at IS NULL;

UPDATE tenants 
SET updated_at = CURRENT_TIMESTAMP 
WHERE updated_at IS NULL;
```

### 代码修改

```go
// 在创建租户时自动获取当前用户ID
var createdByUUID *uuid.UUID
if claims, ok := GetJWTClaimsFromContext(ctx); ok {
    userID, err := uuid.Parse(claims.Subject)
    if err == nil {
        createdByUUID = &userID
    }
}

tenant := &model.Tenant{
    ID:        uuid.New(),
    Name:      req.Name,
    Domain:    req.Domain,
    Type:      tenantType,
    Status:    true,
    CreatedBy: createdByUUID,  // ✅ 自动设置
    IsDeleted: false,
}
```

## 影响范围

### 已修复

- ✅ 新创建的租户会自动记录时间戳和创建者
- ✅ 已存在的空值记录会被更新为当前时间
- ✅ 用户表也同样修复

### 注意事项

- ⚠️ 已存在记录的时间戳会被设置为修复脚本运行时的时间（不是实际创建时间）
- ⚠️ Bootstrap 初始化创建的平台租户可能 `created_by` 为空（正常现象）
- ⚠️ 必须先运行数据库修复脚本，否则新记录仍然不会有时间戳

## 回滚方案

如果需要回滚数据库修改：

```go
migration := migrations.NewFixTimestampsMigration(db)
err := migration.Down()
```

这会移除时间戳字段的默认值约束。

## 相关文件清单

### 新增文件

- `internal/database/migrations/fix_timestamps_migration.go`
- `scripts/fix_timestamps.go`
- `fix_timestamps.sh`
- `test_timestamp_fix.sh`
- `TIMESTAMP_FIX_README.md`
- `TIMESTAMP_FIX_SUMMARY.md`

### 修改文件

- `internal/database/migrations/migration_manager.go`
- `internal/service/auth/tenant_service.go`

## 后续建议

1. **立即执行修复脚本**，确保数据库表结构正确
2. **运行验证脚本**，确认修复成功
3. **测试创建租户**，验证时间戳和创建者字段正常记录
4. **考虑添加单元测试**，确保未来不会再出现类似问题
5. **更新初始迁移文件**，确保新环境部署时就有正确的表结构

## 完成状态

- ✅ 数据库迁移脚本已创建
- ✅ 服务层代码已修复
- ✅ 执行脚本已创建
- ✅ 验证脚本已创建
- ✅ 文档已完善
- ⏳ 等待执行数据库修复脚本
- ⏳ 等待验证修复结果
