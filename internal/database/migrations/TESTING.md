# 数据库迁移测试指南

## 概述

本文档说明如何测试数据库迁移功能。由于迁移使用了 PostgreSQL 特定的功能（如 UUID 类型、JSONB 等），需要使用真实的 PostgreSQL 数据库进行测试。

## 单元测试

单元测试位于 `internal/database/migrate_test.go`，使用 SQLite 内存数据库进行基本的功能测试。由于 SQLite 不支持 PostgreSQL 的特定功能，这些测试主要验证迁移函数可以被正确调用。

运行单元测试：

```bash
go test ./internal/database -short
```

## 集成测试

集成测试需要真实的 PostgreSQL 数据库。

### 准备测试环境

1. 启动 PostgreSQL 数据库（使用 Docker）：

```bash
docker run --name postgres-test \
  -e POSTGRES_USER=testuser \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  -d postgres:15
```

2. 配置环境变量：

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=testuser
export DB_PASSWORD=testpass
export DB_NAME=testdb
export DB_SSLMODE=disable
```

### 运行集成测试

运行完整的集成测试（包括数据库迁移）：

```bash
go test ./internal/database -v
```

### 手动测试迁移

1. 启动服务：

```bash
go run cmd/server/main.go
```

服务启动时会自动执行数据库迁移，日志中会显示：

```
开始执行数据库迁移...
数据库迁移完成 migrations=[chat_sessions chat_messages chat_summaries]
```

2. 验证表结构：

连接到 PostgreSQL 数据库并检查表：

```sql
-- 查看所有表
\dt

-- 查看 chat_sessions 表结构
\d chat_sessions

-- 查看 chat_messages 表结构
\d chat_messages

-- 查看 chat_summaries 表结构
\d chat_summaries

-- 查看索引
\di
```

### 验证迁移结果

迁移成功后，应该创建以下表和索引：

#### 表

- `chat_sessions` - 会话表
- `chat_messages` - 消息表
- `chat_summaries` - 摘要表

#### 索引

**chat_sessions 表：**

- `idx_user_sessions` - (user_id, updated_at DESC)
- `idx_pinned` - (is_pinned, updated_at DESC)
- `idx_archived` - (is_archived)
- `idx_deleted` - (is_deleted)

**chat_messages 表：**

- `idx_session_messages` - (session_id, sequence ASC)
- `idx_created` - (created_at DESC)

**chat_summaries 表：**

- `idx_session_summary` - (session_id, created_at DESC)

## 回滚迁移

如果需要回滚迁移（删除所有表），可以使用迁移管理器的 Down 方法：

```go
import (
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/database/migrations"
)

// 获取数据库连接
db := database.NewPostgresDatabase(config)
// ... 连接数据库

// 创建迁移管理器
manager := migrations.NewMigrationManager(db.GetDB())
manager.Register(migrations.NewSessionMigration(db.GetDB()))

// 回滚迁移
if err := manager.Down(); err != nil {
    log.Fatal(err)
}
```

## 故障排除

### 迁移失败

如果迁移失败，检查以下内容：

1. **数据库连接**：确保数据库连接配置正确
2. **权限**：确保数据库用户有创建表和索引的权限
3. **PostgreSQL 版本**：确保使用 PostgreSQL 12 或更高版本（支持 `gen_random_uuid()`）
4. **日志**：查看详细的错误日志以确定具体问题

### 索引创建失败

如果索引创建失败，可能是因为：

1. 索引已存在（迁移会检查索引是否存在）
2. 表中已有数据且违反了索引约束
3. 数据库资源不足

### 清理测试数据

测试完成后，清理 Docker 容器：

```bash
docker stop postgres-test
docker rm postgres-test
```

## 最佳实践

1. **始终在测试环境中先测试迁移**
2. **备份生产数据库后再执行迁移**
3. **使用事务确保迁移的原子性**
4. **记录迁移日志以便追踪问题**
5. **为每个迁移编写回滚脚本**
