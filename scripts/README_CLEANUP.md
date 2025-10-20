# 数据库迁移记录清理工具

## 概述

`cleanup_migration_records.go` 是一个用于检查和清理数据库中旧迁移记录表的工具。该工具会扫描常见的迁移记录表名，并提供安全的删除功能。

## 背景

在数据库迁移整合过程中，可能存在以下旧的迁移记录表：

- `schema_migrations` - 常见的迁移记录表
- `migrations` - 通用迁移表名
- `gorm_migrations` - GORM 迁移记录表
- `db_migrations` - 数据库迁移表
- `_migrations` - 下划线前缀的迁移表

这些表通常由旧的迁移工具或框架创建，在整合为统一的初始迁移后，这些表已不再需要。

## 使用方法

### 1. 检查模式（推荐首次使用）

使用 `--dry-run` 参数仅检查不执行删除：

```bash
go run scripts/cleanup_migration_records.go --dry-run
```

这将显示：

- 发现的迁移记录表
- 每个表的记录数
- 将要执行的 SQL 语句（但不实际执行）

### 2. 交互式删除

不带参数运行，会在删除前要求确认：

```bash
go run scripts/cleanup_migration_records.go
```

程序会：

1. 显示发现的迁移记录表
2. 显示每个表的记录数
3. 要求用户输入 `yes` 确认删除

### 3. 强制删除

使用 `--force` 参数跳过确认直接删除：

```bash
go run scripts/cleanup_migration_records.go --force
```

**警告**: 此模式会立即删除所有发现的迁移记录表，请谨慎使用！

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--dry-run` | 仅检查不执行删除操作 | false |
| `--force` | 强制删除，不需要确认 | false |

## 输出示例

### Dry Run 模式

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

### 交互式删除

```
=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

发现 1 个迁移记录表:
  - schema_migrations
    记录数: 5

警告: 此操作将删除上述迁移记录表
这些表通常用于跟踪数据库迁移历史
删除后将无法回滚到之前的迁移状态

确认删除这些表吗? (yes/no): yes

开始清理迁移记录表...
删除表 schema_migrations... 成功

=== 清理完成 ===
成功删除: 1 个表

验证清理结果...
✓ 所有迁移记录表已成功清理
```

### 无迁移表

```
=== 数据库迁移记录清理工具 ===
数据库: postgres@localhost:5432/genkit_ai_service

✓ 未发现任何迁移记录表
数据库状态良好，无需清理
```

## 安全特性

1. **Dry Run 模式**: 默认提供检查模式，避免误删除
2. **交互式确认**: 删除前需要用户明确确认
3. **CASCADE 删除**: 使用 CASCADE 确保正确处理依赖关系
4. **验证机制**: 删除后验证表是否真正被删除
5. **详细日志**: 显示每个操作的详细信息

## 注意事项

1. **备份数据**: 在执行删除操作前，建议先备份数据库
2. **生产环境**: 在生产环境使用前，请先在测试环境验证
3. **业务数据**: 该工具只删除迁移记录表，不会影响业务数据表
4. **权限要求**: 需要有删除表的数据库权限
5. **不可恢复**: 删除操作不可恢复，请谨慎操作

## 环境要求

- Go 1.18 或更高版本
- PostgreSQL 数据库
- 正确配置的 `.env` 文件或环境变量

## 配置

工具使用与主应用相同的数据库配置，从以下环境变量读取：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=genkit_ai_service
DB_SSLMODE=disable
```

## 故障排除

### 连接失败

如果出现数据库连接失败：

1. 检查 `.env` 文件是否存在且配置正确
2. 确认数据库服务正在运行
3. 验证数据库连接参数（主机、端口、用户名、密码）

### 删除失败

如果某些表删除失败：

1. 检查是否有其他进程正在使用这些表
2. 确认数据库用户有足够的权限
3. 查看错误信息了解具体原因

### 表仍然存在

如果删除后表仍然存在：

1. 可能有外键约束阻止删除（工具使用 CASCADE 应该能处理）
2. 检查数据库日志了解详细信息
3. 尝试手动执行 `DROP TABLE` 命令

## 相关文档

- [数据库迁移指南](../docs/database-migration-guide.md)
- [初始迁移设计文档](../.kiro/specs/database-initial-migration/design.md)
- [迁移任务列表](../.kiro/specs/database-initial-migration/tasks.md)

## 更新日志

### 2025-01-20

- 初始版本
- 支持检查和删除常见的迁移记录表
- 提供 dry-run 和 force 模式
- 添加交互式确认机制
