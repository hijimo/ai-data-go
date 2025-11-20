# 模型配置模块数据库迁移指南

## 概述

本文档提供了模型配置模块数据库迁移的完整指南，包括执行步骤、验证方法和故障排除。

## 快速开始

### 1. 前置条件

确保以下条件已满足：

- ✅ PostgreSQL 数据库已安装并运行
- ✅ 数据库中已存在 `tenants` 表
- ✅ 数据库中已存在 `users` 表
- ✅ `.env` 文件已正确配置数据库连接信息
- ✅ 环境变量 `ENCRYPTION_SECRET_KEY` 已配置（32字节）

### 2. 执行迁移

```bash
# 方法1: 使用专用迁移脚本（推荐）
go run scripts/run_model_config_migration.go

# 方法2: 使用通用迁移管理器
go run scripts/init_migration.go
```

### 3. 验证迁移

```bash
# 验证表结构、索引和约束
go run scripts/verify_model_config_table.go

# 功能测试（可选）
go run scripts/test_model_config_table.go
```

## 详细说明

### 迁移内容

#### 1. 表结构

创建 `model_configurations` 表，包含以下字段：

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | 主键 |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | 租户ID |
| name | VARCHAR(255) | NOT NULL | 配置名称 |
| model | VARCHAR(255) | NOT NULL | 模型标识 |
| model_provider | VARCHAR(50) | NOT NULL, CHECK | 模型提供商 |
| base_url | VARCHAR(500) | NULL | API基础URL |
| api_key | TEXT | NOT NULL | API密钥（加密） |
| query_params | JSONB | NULL | 查询参数 |
| is_enabled | BOOLEAN | NOT NULL, DEFAULT true | 是否启用 |
| is_deleted | BOOLEAN | NOT NULL, DEFAULT false | 软删除标记 |
| created_by | UUID | NOT NULL, FK -> users(id) | 创建者 |
| created_at | TIMESTAMP WITH TIME ZONE | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_by | UUID | NULL, FK -> users(id) | 更新者 |
| updated_at | TIMESTAMP WITH TIME ZONE | NULL | 更新时间 |
| deleted_by | UUID | NULL, FK -> users(id) | 删除者 |
| deleted_at | TIMESTAMP WITH TIME ZONE | NULL | 删除时间 |

#### 2. 索引

| 索引名称 | 类型 | 列 | 说明 |
|---------|------|-----|------|
| idx_model_configs_tenant_provider | 复合索引 | (tenant_id, model_provider) | 租户+提供商查询 |
| idx_model_configs_deleted | 单列索引 | is_deleted | 软删除过滤 |
| idx_model_configs_enabled | 部分索引 | is_enabled WHERE is_deleted = false | 启用状态查询 |
| idx_model_configs_tenant_id | 部分索引 | tenant_id WHERE is_deleted = false | 租户查询 |
| idx_model_configs_created_at | 单列索引 | created_at DESC | 时间排序 |

#### 3. 约束

**外键约束**:

- `fk_model_configurations_tenant`: tenant_id -> tenants(id) ON DELETE CASCADE
- `fk_model_configurations_created_by`: created_by -> users(id)
- `fk_model_configurations_updated_by`: updated_by -> users(id)
- `fk_model_configurations_deleted_by`: deleted_by -> users(id)

**CHECK 约束**:

- `chk_model_provider`: 确保 model_provider 为有效值
  - 有效值: openai, anthropic, googlegenai, azureopenai, bianlian, custom_openai

### 迁移脚本说明

#### 1. run_model_config_migration.go

**功能**:

- 连接数据库
- 执行模型配置迁移
- 记录迁移状态
- 验证表结构
- 验证索引
- 验证外键约束

**使用方法**:

```bash
go run scripts/run_model_config_migration.go
```

**输出示例**:

```
=== 模型配置模块数据库迁移工具 ===

✅ 数据库连接成功
   数据库: postgres@localhost:5432/ai_service

📦 开始执行模型配置迁移...
✅ 模型配置迁移执行成功

✅ 迁移状态已记录

🔍 验证表结构...
✅ 表结构验证通过

🔍 验证索引...
✅ 索引验证通过

🔍 验证外键约束...
✅ 外键约束验证通过

🎉 所有验证通过！模型配置模块迁移完成！
```

#### 2. verify_model_config_table.go

**功能**:

- 详细验证表结构
- 检查所有列的类型和约束
- 验证所有索引
- 验证所有外键约束
- 验证 CHECK 约束
- 显示表统计信息

**使用方法**:

```bash
go run scripts/verify_model_config_table.go
```

#### 3. test_model_config_table.go

**功能**:

- 插入测试记录
- 查询测试记录
- 更新测试记录
- 软删除测试
- 验证外键约束
- 验证 CHECK 约束
- 自动清理测试数据

**使用方法**:

```bash
go run scripts/test_model_config_table.go
```

**注意**: 此脚本需要数据库中至少有一个租户和用户。

## 验证清单

执行迁移后，请确认以下项目：

### 表结构验证

- [ ] 表 `model_configurations` 已创建
- [ ] 所有16个列已正确创建
- [ ] 列的数据类型正确
- [ ] 列的可空性正确
- [ ] 默认值已正确设置

### 索引验证

- [ ] `idx_model_configs_tenant_provider` 已创建
- [ ] `idx_model_configs_deleted` 已创建
- [ ] `idx_model_configs_enabled` 已创建（部分索引）
- [ ] `idx_model_configs_tenant_id` 已创建（部分索引）
- [ ] `idx_model_configs_created_at` 已创建

### 约束验证

- [ ] 主键约束已设置
- [ ] 外键约束 `fk_model_configurations_tenant` 已创建
- [ ] 外键约束 `fk_model_configurations_created_by` 已创建
- [ ] 外键约束 `fk_model_configurations_updated_by` 已创建
- [ ] 外键约束 `fk_model_configurations_deleted_by` 已创建
- [ ] CHECK 约束 `chk_model_provider` 已创建

### 功能验证

- [ ] 可以插入记录
- [ ] 可以查询记录
- [ ] 可以更新记录
- [ ] 软删除功能正常
- [ ] 外键约束生效
- [ ] CHECK 约束生效

## 故障排除

### 问题1: 连接数据库失败

**错误信息**:

```
❌ 连接数据库失败: connection refused
```

**解决方案**:

1. 检查 PostgreSQL 是否正在运行
2. 检查 `.env` 文件中的数据库配置
3. 确认数据库主机和端口正确

### 问题2: 外键约束创建失败

**错误信息**:

```
ERROR: relation "tenants" does not exist
```

**解决方案**:

1. 确保已执行初始迁移（创建 tenants 和 users 表）
2. 运行 `go run scripts/init_migration.go`

### 问题3: 迁移已执行

**错误信息**:

```
迁移 model_configuration_migration 已执行，跳过
```

**说明**: 这不是错误，表示迁移已经执行过了。如需重新执行：

1. 删除迁移记录：

```sql
DELETE FROM schema_migrations WHERE name = 'model_configuration_migration';
```

2. 删除表（谨慎操作）：

```sql
DROP TABLE IF EXISTS model_configurations CASCADE;
```

3. 重新执行迁移

### 问题4: CHECK 约束违反

**错误信息**:

```
ERROR: new row for relation "model_configurations" violates check constraint "chk_model_provider"
```

**解决方案**:
确保 `model_provider` 字段的值为以下之一：

- openai
- anthropic
- googlegenai
- azureopenai
- bianlian
- custom_openai

## 回滚方案

### 方法1: 使用迁移管理器

```go
migration := migrations.NewModelConfigurationMigration(db)
if err := migration.Down(); err != nil {
    log.Fatal(err)
}
```

### 方法2: 手动执行 SQL

```sql
-- 删除表（会级联删除所有相关数据）
DROP TABLE IF EXISTS model_configurations CASCADE;

-- 删除迁移记录
DELETE FROM schema_migrations WHERE name = 'model_configuration_migration';
```

**警告**: 回滚操作会删除所有模型配置数据，请谨慎操作！

## 性能优化建议

### 1. 索引使用

- 查询特定租户的配置时，使用 `tenant_id` 过滤
- 查询特定提供商的配置时，使用 `(tenant_id, model_provider)` 复合索引
- 始终包含 `is_deleted = false` 条件以利用部分索引

### 2. 查询优化

**推荐**:

```sql
-- 利用部分索引
SELECT * FROM model_configurations 
WHERE tenant_id = ? AND is_deleted = false;

-- 利用复合索引
SELECT * FROM model_configurations 
WHERE tenant_id = ? AND model_provider = ? AND is_deleted = false;
```

**不推荐**:

```sql
-- 全表扫描
SELECT * FROM model_configurations;

-- 未使用索引
SELECT * FROM model_configurations WHERE name LIKE '%test%';
```

### 3. 批量操作

对于批量插入或更新，使用事务：

```go
tx := db.Begin()
for _, config := range configs {
    if err := tx.Create(&config).Error; err != nil {
        tx.Rollback()
        return err
    }
}
tx.Commit()
```

## 监控建议

### 1. 表大小监控

```sql
SELECT 
    pg_size_pretty(pg_table_size('model_configurations')) as table_size,
    pg_size_pretty(pg_indexes_size('model_configurations')) as index_size,
    pg_size_pretty(pg_total_relation_size('model_configurations')) as total_size;
```

### 2. 索引使用情况

```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'model_configurations'
ORDER BY idx_scan DESC;
```

### 3. 慢查询监控

在 PostgreSQL 配置中启用慢查询日志：

```
log_min_duration_statement = 1000  # 记录超过1秒的查询
```

## 相关文档

- [需求文档](./requirements.md)
- [设计文档](./design.md)
- [任务列表](./tasks.md)
- [迁移验证报告](./MIGRATION_VERIFICATION.md)
- [Task 9 完成总结](./TASK_9_SUMMARY.md)

## 支持

如遇到问题，请：

1. 查看本文档的故障排除部分
2. 检查迁移验证报告
3. 查看数据库日志
4. 联系开发团队

## 更新日志

### 2025-11-20

- ✅ 初始迁移创建
- ✅ 表结构验证通过
- ✅ 索引验证通过
- ✅ 约束验证通过
- ✅ 功能测试通过
