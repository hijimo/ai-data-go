# 向量存储迁移总结：从 pgvector 到 Qdrant

## 变更概述

本次更新将向量存储方案从 PostgreSQL 的 pgvector 扩展迁移到 Qdrant 在线向量数据库服务。

## 变更原因

1. **专业化**：Qdrant 是专门的向量数据库，提供更好的检索性能和更丰富的功能
2. **可扩展性**：向量数据库可以独立于主数据库进行扩展
3. **云原生**：使用在线服务减少运维负担
4. **灵活性**：更容易切换或升级向量数据库
5. **成本优化**：按需付费，避免过度配置

## 文件变更清单

### 1. 需求文档 (.kiro/specs/genkit-session-management/requirements.md)

**变更内容：**

- 在术语表中添加 Qdrant 定义
- 更新需求 2（数据模型设计）：
  - 移除 embedding 字段要求
  - 添加 Qdrant collection 创建要求
  - 添加 Qdrant payload 配置要求
- 更新需求 12（长期记忆检索）：将 pgvector 替换为 Qdrant
- 更新需求 13（记忆存储）：添加 Qdrant 存储步骤
- 更新需求 23（数据隔离策略）：添加 Qdrant filter 要求
- 更新需求 29（健康检查）：将向量服务改为 Qdrant
- 更新需求 32（降级策略）：将向量服务改为 Qdrant

### 2. 设计文档 (.kiro/specs/genkit-session-management/design.md)

**变更内容：**

- 更新技术栈：将 "PostgreSQL 15+ with pgvector" 改为 "PostgreSQL 15+" 和 "Qdrant（在线服务）"
- 更新架构图：将 pgvector 替换为 Qdrant
- 更新数据模型：
  - 移除 conversation_memories 表的 embedding 字段
  - 移除向量索引创建语句
  - 添加注释说明向量存储在 Qdrant
- 移除 Go 模型中的 pgvector.Vector 类型
- 添加完整的 Qdrant 设计章节：
  - Collection 结构设计
  - Payload 结构定义
  - 索引配置
  - VectorService 接口定义
  - QdrantClient 实现示例

### 3. 数据库迁移 (internal/database/migrations/genkit_session_management_migration.go)

**变更内容：**

- 移除 pgvector 扩展启用逻辑（保留方法但返回 nil）
- 移除 conversation_memories 表的 embedding 字段定义
- 移除向量索引创建语句
- 更新表注释，说明向量存储在 Qdrant

### 4. Go 模型 (internal/model/genkit_session.go)

**变更内容：**

- 移除 pgvector-go 包导入
- 移除 ConversationMemory 模型的 Embedding 字段
- 添加注释说明向量存储在 Qdrant

### 5. 测试脚本 (scripts/test_genkit_migration.go)

**变更内容：**

- 移除 pgvector 扩展验证
- 移除向量索引验证
- 添加注释说明使用 Qdrant

### 6. 迁移指南 (docs/genkit_migration_guide.md)

**完全重写，新增内容：**

- 架构说明：数据存储分离的优势
- Qdrant 配置章节
- Collection 和 Payload 结构说明
- 多租户隔离在 Qdrant 层面的实现
- Qdrant 验证方法
- Qdrant 性能优化建议

### 7. 新增文档 (docs/qdrant_migration_summary.md)

本文档，总结所有变更。

## 依赖变更

### 移除的依赖

```go
github.com/pgvector/pgvector-go
```

### 需要添加的依赖

```go
github.com/qdrant/go-client  // Qdrant Go 客户端
```

## 数据库 Schema 变更

### conversation_memories 表

**移除字段：**

- `embedding vector(1536)` - 向量数据现在存储在 Qdrant

**移除索引：**

- `idx_memories_embedding` - 向量索引现在在 Qdrant 中

## Qdrant 配置要求

### 环境变量

需要添加以下环境变量：

```bash
QDRANT_HOST=your-qdrant-host.com
QDRANT_PORT=6333
QDRANT_API_KEY=your-api-key
QDRANT_USE_TLS=true
```

### Collection 命名规范

- 格式：`memories_{tenant_id}`
- 示例：`memories_123e4567-e89b-12d3-a456-426614174000`

### 向量配置

- 维度：1536（OpenAI text-embedding-ada-002）
- 距离度量：Cosine
- 索引类型：HNSW（默认）

## 迁移状态

### ✅ 已完成

1. **数据库 Schema 更新**
   - ✅ 移除 conversation_memories 表的 embedding 字段
   - ✅ 移除向量索引
   - ✅ 更新表注释
   - ✅ 迁移已成功执行并验证

2. **文档更新**
   - ✅ 需求文档更新
   - ✅ 设计文档更新
   - ✅ 迁移指南更新
   - ✅ Go 模型更新

3. **数据库表验证**
   - ✅ conversation_memories 表已创建（14 列）
   - ✅ conversation_contexts 表已创建（15 列）
   - ✅ conversation_summaries 表已创建（16 列）

### 🔄 待实现

以下是需要实现的新组件：

#### 1. Qdrant 客户端 (internal/storage/qdrant_client.go)

```go
- NewQdrantClient() - 创建客户端
- CreateCollection() - 创建 collection
- Upsert() - 插入/更新向量
- Search() - 搜索相似向量
- Delete() - 删除向量
- UpdatePayload() - 更新 payload
```

#### 2. 向量服务 (internal/service/vector_service.go)

```go
- StoreVector() - 存储向量
- SearchVectors() - 搜索向量
- DeleteVector() - 删除向量
- UpdatePayload() - 更新元数据
```

#### 3. 配置管理 (internal/config/qdrant.go)

```go
- QdrantConfig 结构体
- 配置加载和验证
```

#### 4. 健康检查 (internal/health/qdrant_check.go)

```go
- CheckQdrantHealth() - 检查 Qdrant 连接状态
```

## 迁移步骤

### 对于新部署

1. 配置 Qdrant 连接信息
2. 运行数据库迁移
3. 应用会自动为新租户创建 collection

### 对于现有部署（如果之前使用了 pgvector）

1. **数据迁移**：

   ```sql
   -- 导出现有向量数据
   SELECT id, tenant_id, session_id, embedding 
   FROM conversation_memories 
   WHERE is_deleted = false;
   ```

2. **导入到 Qdrant**：
   - 为每个租户创建 collection
   - 批量导入向量数据
   - 验证数据完整性

3. **Schema 更新**：

   ```sql
   -- 移除 embedding 字段
   ALTER TABLE conversation_memories DROP COLUMN embedding;
   
   -- 移除向量索引
   DROP INDEX IF EXISTS idx_memories_embedding;
   ```

4. **验证**：
   - 测试向量检索功能
   - 验证多租户隔离
   - 性能测试

## 性能对比

### pgvector

- **优势**：数据在同一数据库，减少网络延迟
- **劣势**：
  - 向量索引占用 PostgreSQL 资源
  - 扩展性受限于 PostgreSQL
  - 索引维护影响主库性能

### Qdrant

- **优势**：
  - 专业向量数据库，检索性能更好
  - 独立扩展，不影响主库
  - 丰富的过滤和搜索功能
  - 云服务，减少运维负担
- **劣势**：
  - 增加网络延迟（可通过缓存缓解）
  - 需要额外的服务配置

## 测试建议

### 单元测试

- Qdrant 客户端连接测试
- Collection 创建和删除测试
- 向量插入和搜索测试
- Payload 更新测试

### 集成测试

- 端到端记忆存储和检索测试
- 多租户隔离测试
- 性能基准测试
- 故障恢复测试

### 性能测试

- 向量检索延迟（P50, P95, P99）
- 批量插入性能
- 并发查询性能
- 大规模数据集测试

## 监控指标

需要添加的监控指标：

- `qdrant_connection_status` - Qdrant 连接状态
- `qdrant_search_duration_seconds` - 搜索延迟
- `qdrant_upsert_duration_seconds` - 插入延迟
- `qdrant_collection_size` - Collection 大小
- `qdrant_api_errors_total` - API 错误总数

## 回滚计划

如果需要回滚到 pgvector：

1. 保留 Qdrant 中的数据作为备份
2. 重新添加 embedding 字段到 PostgreSQL
3. 从 Qdrant 导出向量数据
4. 导入到 PostgreSQL
5. 重新创建向量索引
6. 更新应用代码

## 总结

本次迁移将向量存储从 pgvector 迁移到 Qdrant，带来以下好处：

✅ 更好的检索性能
✅ 独立的扩展能力
✅ 减少主数据库负担
✅ 更灵活的配置选项
✅ 云原生架构

需要注意的是，这需要额外的 Qdrant 服务配置和网络连接管理。建议在生产环境部署前进行充分的性能测试和故障演练。
