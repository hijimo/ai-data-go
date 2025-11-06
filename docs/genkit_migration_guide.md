# Genkit 会话管理模块数据库迁移指南

## 概述

本迁移为 Genkit 会话管理模块创建了三个核心表，用于实现三层记忆架构。**向量数据存储在 Qdrant 向量数据库中**，PostgreSQL 仅存储元数据。

## 架构说明

### 数据存储分离

- **PostgreSQL**：存储记忆的元数据（ID、内容、重要性、访问统计等）
- **Qdrant**：存储向量嵌入和执行相似度搜索

这种分离架构的优势：

1. 专业化：使用专门的向量数据库获得更好的检索性能
2. 可扩展性：向量数据库可以独立扩展
3. 灵活性：可以轻松切换向量数据库提供商
4. 成本优化：向量数据可以使用云服务，按需付费

## 新增表结构

### 1. conversation_memories（会话记忆表）

存储长期记忆的元数据。

**主要字段：**

- `id`: 记忆唯一标识符（UUID）
- `tenant_id`: 租户ID
- `session_id`: 会话ID
- `memory_type`: 记忆类型（short_term, long_term, summary）
- `content`: 记忆内容
- `token_count`: Token数量
- `importance`: 重要性评分（0-1）
- `access_count`: 访问次数
- `last_access_at`: 最后访问时间
- `expires_at`: 过期时间

**注意**：向量数据存储在 Qdrant 中，使用 `memory_id` 作为关联键。

**索引：**

- 租户+会话复合索引
- 记忆类型索引
- 过期时间索引

### 2. conversation_contexts（会话上下文配置表）

存储会话的上下文管理配置。

**主要字段：**

- `id`: 配置唯一标识符（UUID）
- `tenant_id`: 租户ID
- `session_id`: 会话ID（唯一）
- `max_tokens`: 最大Token数量
- `strategy`: 上下文策略（auto, short, full）
- `include_summary`: 是否包含摘要
- `include_long_term`: 是否包含长期记忆
- `short_term_window`: 短期记忆窗口大小
- `total_messages`: 总消息数
- `total_tokens_used`: 总Token使用量

**约束：**

- session_id 唯一约束
- max_tokens 范围：100-128000
- short_term_window 范围：1-100

### 3. conversation_summaries（会话摘要表）

存储会话的摘要信息。

**主要字段：**

- `id`: 摘要唯一标识符（UUID）
- `tenant_id`: 租户ID
- `session_id`: 会话ID
- `summary_type`: 摘要类型（incremental, full）
- `content`: 摘要内容
- `token_count`: Token数量
- `message_count`: 包含的消息数量
- `quality_score`: 质量评分（0-1）
- `compression_rate`: 压缩率（0-1）
- `key_topics`: 关键主题数组
- `previous_summary_id`: 前一个摘要ID

## Qdrant 向量数据库配置

### Collection 结构

为每个租户创建独立的 collection：

```
Collection 名称: memories_{tenant_id}
向量维度: 1536 (OpenAI text-embedding-ada-002)
距离度量: Cosine
```

### Payload 结构

每个向量点包含以下 payload：

```json
{
  "memory_id": "uuid",
  "tenant_id": "uuid",
  "session_id": "uuid",
  "memory_type": "short_term|long_term|summary",
  "importance": 0.8,
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

### 索引配置

Qdrant 会自动为以下字段创建索引：

- `tenant_id` (keyword)
- `session_id` (keyword)
- `memory_type` (keyword)

## 执行迁移

### 方法1：使用完整迁移脚本

```bash
# 执行所有迁移（包括 Genkit 模块）
go run scripts/init_migration.go
```

### 方法2：单独测试 Genkit 迁移

```bash
# 测试迁移（不回滚）
go run scripts/test_genkit_migration.go

# 测试迁移并回滚
TEST_ROLLBACK=true go run scripts/test_genkit_migration.go
```

### 方法3：在应用启动时自动执行

迁移已注册到 `RunAllMigrations` 函数中，会在应用启动时自动执行。

## Go 模型定义

新增了三个 Go 模型：

1. **ConversationMemory** - `internal/model/genkit_session.go`
2. **ConversationContext** - `internal/model/genkit_session.go`
3. **ConversationSummary** - `internal/model/genkit_session.go`

所有模型都使用 UUID 作为主键，并包含租户隔离字段。

**注意**：`ConversationMemory` 模型不包含 `Embedding` 字段，因为向量数据存储在 Qdrant 中。

## Qdrant 配置

### 环境变量

```bash
QDRANT_HOST=your-qdrant-host.com
QDRANT_PORT=6333
QDRANT_API_KEY=your-api-key
QDRANT_USE_TLS=true
```

### 配置文件示例

```yaml
# config/config.yaml
qdrant:
  host: your-qdrant-host.com
  port: 6333
  api_key: ${QDRANT_API_KEY}
  use_tls: true
  timeout: 30s
```

## 多租户隔离

### PostgreSQL 层面

所有表都包含 `tenant_id` 字段，并设置了外键约束：

```sql
CONSTRAINT fk_memories_tenant 
FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
```

**重要提示：**

- 所有查询必须包含 `tenant_id` 过滤条件
- 使用 `WHERE is_deleted = false` 过滤软删除记录
- 索引已针对租户隔离查询进行优化

### Qdrant 层面

- 每个租户使用独立的 collection
- Collection 命名：`memories_{tenant_id}`
- 在 payload 中存储 `tenant_id` 作为额外保障
- 使用 filter 确保跨会话搜索时的租户隔离

## 验证迁移

执行迁移后，可以通过以下 SQL 验证：

```sql
-- 检查表是否创建
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN ('conversation_memories', 'conversation_contexts', 'conversation_summaries');

-- 检查外键约束
SELECT 
    tc.table_name, 
    tc.constraint_name, 
    tc.constraint_type
FROM information_schema.table_constraints tc
WHERE tc.table_name IN ('conversation_memories', 'conversation_contexts', 'conversation_summaries')
AND tc.constraint_type = 'FOREIGN KEY';
```

### 验证 Qdrant

```bash
# 使用 Qdrant API 检查 collection
curl -X GET "https://your-qdrant-host.com:6333/collections" \
  -H "api-key: your-api-key"
```

## 回滚迁移

如果需要回滚迁移：

```go
migration := migrations.NewGenkitSessionManagementMigration(db)
err := migration.Down()
```

这将删除所有三个表及其相关的索引和约束。

**注意**：回滚不会删除 Qdrant 中的 collection，需要手动清理。

## 注意事项

1. **Qdrant 连接**：确保 Qdrant 服务可访问且 API 密钥正确
2. **向量维度**：当前使用 1536 维（OpenAI text-embedding-ada-002），如需更改需同步更新
3. **Collection 管理**：首次使用时需要为每个租户创建 collection
4. **数据同步**：确保 PostgreSQL 和 Qdrant 中的数据保持一致
5. **备份**：执行迁移前建议备份数据库

## 性能优化建议

### PostgreSQL 优化

1. **查询优化**：
   - 始终使用 `WHERE is_deleted = false` 过滤
   - 利用复合索引进行租户+会话查询
   - 使用 `LIMIT` 限制结果数量

2. **数据清理**：
   - 定期清理过期记忆（`expires_at < NOW()`）
   - 归档历史数据以保持表大小可控

### Qdrant 优化

1. **Collection 配置**：
   - 根据数据量调整 HNSW 参数
   - 使用适当的 `ef_construct` 和 `m` 值

2. **查询优化**：
   - 使用 filter 减少搜索范围
   - 合理设置 `top_k` 值
   - 利用 payload 索引加速过滤

3. **批量操作**：
   - 批量插入向量以提高性能
   - 使用异步操作处理大量数据

## 相关文档

- [设计文档](../.kiro/specs/genkit-session-management/design.md)
- [需求文档](../.kiro/specs/genkit-session-management/requirements.md)
- [任务列表](../.kiro/specs/genkit-session-management/tasks.md)
- [Qdrant 官方文档](https://qdrant.tech/documentation/)
