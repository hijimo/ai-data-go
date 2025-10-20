# 迁移记录清理脚本使用示例

## 快速开始

### 1. 首次使用 - 检查模式

在首次使用时，建议先使用 `--dry-run` 模式检查数据库中是否存在旧的迁移记录表：

```bash
go run scripts/cleanup_migration_records.go --dry-run
```

**示例输出：**

```
=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

发现 2 个迁移记录表:
  - schema_migrations
    记录数: 5
  - migrations
    记录数: 3

--- Dry Run 模式 ---
以下表将被删除（实际未执行）:
  DROP TABLE IF EXISTS schema_migrations CASCADE;
  DROP TABLE IF EXISTS migrations CASCADE;

提示: 移除 --dry-run 参数以执行实际删除操作
```

### 2. 交互式删除（推荐）

确认需要删除后，运行不带参数的命令，脚本会要求确认：

```bash
go run scripts/cleanup_migration_records.go
```

**交互过程：**

```
=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

发现 2 个迁移记录表:
  - schema_migrations
    记录数: 5
  - migrations
    记录数: 3

警告: 此操作将删除上述迁移记录表
这些表通常用于跟踪数据库迁移历史
删除后将无法回滚到之前的迁移状态

确认删除这些表吗? (yes/no): yes

开始清理迁移记录表...
删除表 schema_migrations... 成功
删除表 migrations... 成功

=== 清理完成 ===
成功删除: 2 个表

验证清理结果...
✓ 所有迁移记录表已成功清理
```

### 3. 强制删除（自动化场景）

在自动化脚本或 CI/CD 流程中，可以使用 `--force` 参数跳过确认：

```bash
go run scripts/cleanup_migration_records.go --force
```

**警告：** 此模式会立即删除所有发现的迁移记录表，请谨慎使用！

## 常见场景

### 场景 1：数据库中没有迁移记录表

```bash
$ go run scripts/cleanup_migration_records.go --dry-run

=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

✓ 未发现任何迁移记录表
数据库状态良好，无需清理
```

### 场景 2：用户取消删除操作

```bash
$ go run scripts/cleanup_migration_records.go

=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

发现 1 个迁移记录表:
  - schema_migrations
    记录数: 5

警告: 此操作将删除上述迁移记录表
这些表通常用于跟踪数据库迁移历史
删除后将无法回滚到之前的迁移状态

确认删除这些表吗? (yes/no): no

操作已取消
```

### 场景 3：部分表删除失败

```bash
$ go run scripts/cleanup_migration_records.go --force

=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

发现 2 个迁移记录表:
  - schema_migrations
    记录数: 5
  - migrations
    记录数: 3

开始清理迁移记录表...
删除表 schema_migrations... 成功
删除表 migrations... 失败: 执行 SQL 失败: pq: permission denied

=== 清理完成 ===
成功删除: 1 个表
删除失败: 1 个表

验证清理结果...
⚠ 仍有 1 个表未能删除:
  - migrations
```

## 集成到部署流程

### 在初始化脚本中使用

```bash
#!/bin/bash
# deploy.sh

echo "检查旧的迁移记录表..."
go run scripts/cleanup_migration_records.go --dry-run

echo "清理旧的迁移记录表..."
go run scripts/cleanup_migration_records.go --force

echo "执行新的初始迁移..."
go run scripts/init_migration.go
```

### 在 Makefile 中使用

```makefile
.PHONY: clean-migrations
clean-migrations:
 @echo "清理旧的迁移记录表..."
 @go run scripts/cleanup_migration_records.go --force

.PHONY: check-migrations
check-migrations:
 @echo "检查迁移记录表..."
 @go run scripts/cleanup_migration_records.go --dry-run

.PHONY: reset-db
reset-db: clean-migrations
 @echo "重置数据库..."
 @go run scripts/reset_db.go
```

## 安全建议

1. **生产环境使用前务必备份数据库**

```bash
# PostgreSQL 备份示例
pg_dump -h localhost -U postgres -d genkit_ai_service > backup_$(date +%Y%m%d_%H%M%S).sql
```

2. **先在测试环境验证**

在生产环境使用前，先在测试环境或开发环境验证脚本行为。

3. **使用 dry-run 模式预览**

始终先使用 `--dry-run` 模式查看将要删除的表。

4. **检查数据库权限**

确保数据库用户有删除表的权限：

```sql
-- 检查当前用户权限
SELECT * FROM information_schema.table_privileges 
WHERE grantee = current_user;
```

## 故障排除

### 问题：连接数据库失败

**解决方案：**

1. 检查 `.env` 文件是否存在且配置正确
2. 确认数据库服务正在运行
3. 验证数据库连接参数

```bash
# 测试数据库连接
psql -h localhost -U postgres -d genkit_ai_service -c "SELECT 1"
```

### 问题：权限不足

**错误信息：** `permission denied`

**解决方案：**

```sql
-- 授予删除表权限
GRANT DROP ON ALL TABLES IN SCHEMA public TO your_user;
```

### 问题：表被其他进程锁定

**错误信息：** `table is locked`

**解决方案：**

```sql
-- 查看锁定的进程
SELECT * FROM pg_locks WHERE relation = 'schema_migrations'::regclass;

-- 终止锁定的进程（谨慎使用）
SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
WHERE datname = 'genkit_ai_service' AND pid <> pg_backend_pid();
```

## 相关文档

- [清理脚本详细文档](./README_CLEANUP.md)
- [数据库迁移指南](../docs/database-migration-guide.md)
- [初始迁移设计](../.kiro/specs/database-initial-migration/design.md)

## 支持的迁移记录表

脚本会检查并清理以下常见的迁移记录表：

- `schema_migrations` - 最常见的迁移记录表
- `migrations` - 通用迁移表名
- `gorm_migrations` - GORM 框架使用的迁移表
- `db_migrations` - 数据库迁移工具使用的表
- `_migrations` - 下划线前缀的迁移表

如果你的项目使用了其他名称的迁移记录表，可以修改 `scripts/cleanup_migration_records.go` 中的 `commonMigrationTables` 变量。
