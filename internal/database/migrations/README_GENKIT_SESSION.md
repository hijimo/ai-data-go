# Genkit 会话管理模块数据库迁移

## 概述

本迁移为 Genkit 会话管理模块创建必要的数据库表和索引，支持三层记忆架构（短期、长期、摘要）和向量检索功能。

## 创建的表

### 1. conversation_memories

存储对话记忆，支持向量检索。

**字段说明：**

- `id`: UUID 主键，自动生成
- `tenant_id`: 租户ID，外键关联 tenants 表
- `session_id`: 会话ID，外键关联 conversation_sessions 表
- `memory_type`: 记忆类型（short_term, long_term, summary）
- `content`: 记忆内容
- `embedding`: 向量嵌入（1536维，用于语义检索）
- `token_count`: Token 数量
- `importance`: 重要性评分（0-1）
- `access_count`: 访问次数
- `last_access_at`: 最后访问时间
- `metadata`: 元数据（JSONB）
- `expires_at`: 过期时间
- `is_deleted`: 软删除标记
- `created_at`: 创建时间
- `updated_at`: 更新时间

**索引：**

- `idx_memories_tenant_session`: (tenant_id, session_id) 复合索引
- `idx_memories_type`: memory_type 索引
- `idx_memories_expires`: expires_at 索引
- `idx_memories_created`: created_at 降序索引
- `idx_memories_embedding`: 向量索引（IVFFlat 算法，需要数据后创建）

### 2. conversation_contexts

存储会话的上下文配置。

**字段说明：**

- `id`: UUID 主键，自动生成
- `tenant_id`: 租户ID
- `session_id`: 会话ID（唯一）
- `max_tokens`: 最大 Token 数
- `strategy`: 上下文策略（auto, short, full）
- `include_summary`: 是否包含摘要
- `include_long_term`: 是否包含长期记忆
- `short_term_window`: 短期记忆窗口大小
- `last_summary_id`: 最后一次摘要ID
- `last_summary_at`: 最后一次摘要时间
- `total_messages`: 总消息数
- `total_tokens_used`: 总 Token 使用量
- `is_deleted`: 软删除标记
- `created_at`: 创建时间
- `updated_at`: 更新时间

**索引：**

- `idx_contexts_tenant`: tenant_id 索引
- `idx_contexts_session`: session_id 索引

### 3. conversation_summaries

存储对话摘要。

**字段说明：**

- `id`: UUID 主键，自动生成
- `tenant_id`: 租户ID
- `session_id`: 会话ID
- `summary_type`: 摘要类型（incremental, full）
- `content`: 摘要内容
- `token_count`: Token 数量
- `message_count`: 消息数量
- `start_message_id`: 起始消息ID
- `end_message_id`: 结束消息ID
- `quality_score`: 质量评分（0-1）
- `compression_rate`: 压缩率（0-1）
- `key_topics`: 关键主题数组
- `previous_summary_id`: 上一个摘要ID
- `is_deleted`: 软删除标记
- `created_at`: 创建时间
- `updated_at`: 更新时间

**索引：**

- `idx_summaries_tenant_session`: (tenant_id, session_id) 复合索引
- `idx_summaries_created`: created_at 降序索引
- `idx_summaries_session_latest`: (session_id, created_at DESC) 复合索引

## 使用方法

### 方法1：使用迁移脚本

```bash
# 执行迁移
go run scripts/genkit_session_migrate.go up

# 回滚迁移
go run scripts/genkit_session_migrate.go down

# 创建向量索引（需要表中有至少100条记录）
go run scripts/genkit_session_migrate.go create-vector-index
```

### 方法2：在代码中使用

```go
package main

import (
    "genkit-ai-service/internal/database"
    "genkit-ai-service/internal/database/migrations"
)

func main() {
    // 连接数据库
    db, err := database.NewPostgresDB(cfg)
    if err != nil {
        panic(err)
    }

    // 创建迁移实例
    migration := migrations.NewGenkitSessionMigration(db)

    // 执行迁移
    if err := migration.Up(); err != nil {
        panic(err)
    }

    // 记录迁移状态
    if err := migrations.RecordMigrationStatus(db, migration.GetName()); err != nil {
        log.Printf("警告: 记录迁移状态失败: %v", err)
    }
}
```

### 方法3：集成到统一迁移管理器

在 `RunAllMigrations` 函数中注册：

```go
func RunAllMigrations(db *gorm.DB) error {
    manager := NewMigrationManager(db)
    
    // 注册初始迁移
    manager.RegisterInitialMigration()
    
    // 注册 Genkit 会话管理迁移
    manager.Register(NewGenkitSessionMigration(db))
    
    // 注册其他迁移...
    
    return manager.Up()
}
```

## 前置条件

### 1. PostgreSQL 版本要求

- PostgreSQL 13+ （支持 `gen_random_uuid()` 函数）
- 如果使用较旧版本，需要启用 `pgcrypto` 扩展

### 2. pgvector 扩展

迁移会自动启用 pgvector 扩展。如果手动安装，执行：

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### 3. 依赖表

以下表必须已存在：

- `tenants`: 租户表
- `conversation_sessions`: 会话表

这些表通常由初始迁移创建。

## 向量索引说明

### 为什么不在迁移时创建向量索引？

向量索引（IVFFlat）需要表中有足够的数据才能有效创建。建议：

1. 先执行迁移创建表结构
2. 插入至少 100 条记录
3. 然后创建向量索引

### 创建向量索引

```bash
# 使用脚本
go run scripts/genkit_session_migrate.go create-vector-index

# 或手动执行 SQL
CREATE INDEX idx_memories_embedding 
    ON conversation_memories 
    USING ivfflat (embedding vector_cosine_ops) 
    WITH (lists = 100)
    WHERE is_deleted = FALSE AND embedding IS NOT NULL;
```

### 向量索引参数说明

- `lists = 100`: IVFFlat 算法的聚类数量
  - 建议值：sqrt(记录数)
  - 对于 10,000 条记录，使用 100
  - 对于 100,000 条记录，使用 316
  - 对于 1,000,000 条记录，使用 1000

- `vector_cosine_ops`: 使用余弦相似度
  - 其他选项：`vector_l2_ops`（欧氏距离）、`vector_ip_ops`（内积）

## 数据模型

Go 数据模型定义在 `internal/model/genkit_session.go`：

```go
// ConversationMemory 对话记忆模型
type ConversationMemory struct {
    ID           uuid.UUID
    TenantID     uuid.UUID
    SessionID    uuid.UUID
    MemoryType   string
    Content      string
    Embedding    pgvector.Vector
    TokenCount   int
    Importance   float32
    AccessCount  int
    LastAccessAt *time.Time
    Metadata     map[string]interface{}
    ExpiresAt    *time.Time
    IsDeleted    bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// ConversationContext 对话上下文配置模型
type ConversationContext struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    SessionID       uuid.UUID
    MaxTokens       int
    Strategy        string
    IncludeSummary  bool
    IncludeLongTerm bool
    ShortTermWindow int
    LastSummaryID   *uuid.UUID
    LastSummaryAt   *time.Time
    TotalMessages   int
    TotalTokensUsed int64
    IsDeleted       bool
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// ConversationSummary 对话摘要模型
type ConversationSummary struct {
    ID                uuid.UUID
    TenantID          uuid.UUID
    SessionID         uuid.UUID
    SummaryType       string
    Content           string
    TokenCount        int
    MessageCount      int
    StartMessageID    *uuid.UUID
    EndMessageID      *uuid.UUID
    QualityScore      *float64
    CompressionRate   *float64
    KeyTopics         []string
    PreviousSummaryID *uuid.UUID
    IsDeleted         bool
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

## 多租户隔离

所有表都包含 `tenant_id` 字段，确保数据隔离：

1. **数据库层**：所有查询必须包含 `tenant_id` 过滤
2. **索引优化**：复合索引包含 `tenant_id`
3. **外键约束**：级联删除确保数据一致性

## 性能优化建议

### 1. 定期清理过期记忆

```sql
DELETE FROM conversation_memories 
WHERE is_deleted = FALSE 
  AND expires_at IS NOT NULL 
  AND expires_at < NOW();
```

### 2. 监控向量索引性能

```sql
-- 查看索引大小
SELECT pg_size_pretty(pg_relation_size('idx_memories_embedding'));

-- 查看索引使用情况
SELECT * FROM pg_stat_user_indexes 
WHERE indexrelname = 'idx_memories_embedding';
```

### 3. 定期重建向量索引

当数据量增长较大时，考虑重建索引：

```sql
DROP INDEX idx_memories_embedding;
CREATE INDEX idx_memories_embedding 
    ON conversation_memories 
    USING ivfflat (embedding vector_cosine_ops) 
    WITH (lists = 316)  -- 根据数据量调整
    WHERE is_deleted = FALSE AND embedding IS NOT NULL;
```

## 故障排查

### 问题1：pgvector 扩展不存在

**错误信息：**

```
ERROR: type "vector" does not exist
```

**解决方案：**

```bash
# 安装 pgvector
# Ubuntu/Debian
sudo apt-get install postgresql-15-pgvector

# macOS
brew install pgvector

# 然后在数据库中启用
psql -d your_database -c "CREATE EXTENSION vector;"
```

### 问题2：外键约束失败

**错误信息：**

```
ERROR: insert or update on table "conversation_memories" violates foreign key constraint
```

**解决方案：**
确保依赖表已存在：

```bash
# 先运行初始迁移
go run scripts/init_migration.go

# 然后运行 Genkit 会话管理迁移
go run scripts/genkit_session_migrate.go up
```

### 问题3：向量索引创建失败

**错误信息：**

```
ERROR: 数据量不足，无法创建向量索引
```

**解决方案：**
等待表中有至少 100 条记录后再创建向量索引。

## 相关文档

- [Genkit 会话管理需求文档](../../../.kiro/specs/genkit-session-management/requirements.md)
- [Genkit 会话管理设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [pgvector 文档](https://github.com/pgvector/pgvector)
- [PostgreSQL UUID 文档](https://www.postgresql.org/docs/current/datatype-uuid.html)

## 版本历史

- v1.0.0 (2025-10-29): 初始版本
  - 创建 conversation_memories 表
  - 创建 conversation_contexts 表
  - 创建 conversation_summaries 表
  - 支持 pgvector 向量检索
  - 支持多租户隔离
