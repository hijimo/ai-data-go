# Genkit 会话管理模块快速参考

## 快速开始

### 1. 执行迁移

```bash
# 方式1：使用迁移脚本
go run scripts/genkit_session_migrate.go up

# 方式2：使用 make 命令（如果配置了）
make migrate-genkit-session

# 方式3：在代码中执行
# 参见 scripts/genkit_session_migrate.go
```

### 2. 创建向量索引

```bash
# 确保表中有至少 100 条记录
go run scripts/genkit_session_migrate.go create-vector-index
```

### 3. 回滚迁移

```bash
go run scripts/genkit_session_migrate.go down
```

## 数据模型速查

### ConversationMemory（对话记忆）

```go
memory := &model.ConversationMemory{
    TenantID:   tenantID,
    SessionID:  sessionID,
    MemoryType: model.MemoryTypeLongTerm,  // short_term, long_term, summary
    Content:    "记忆内容",
    Embedding:  vector,                     // pgvector.Vector
    TokenCount: 100,
    Importance: 0.8,                        // 0-1
}
```

### ConversationContext（上下文配置）

```go
context := &model.ConversationContext{
    TenantID:        tenantID,
    SessionID:       sessionID,
    MaxTokens:       4000,
    Strategy:        model.ContextStrategyAuto,  // auto, short, full
    IncludeSummary:  true,
    IncludeLongTerm: true,
    ShortTermWindow: 10,
}
```

### ConversationSummary（对话摘要）

```go
summary := &model.ConversationSummary{
    TenantID:     tenantID,
    SessionID:    sessionID,
    SummaryType:  model.SummaryTypeIncremental,  // incremental, full
    Content:      "摘要内容",
    TokenCount:   50,
    MessageCount: 20,
    KeyTopics:    []string{"主题1", "主题2"},
}
```

## 常用查询

### 查询会话的长期记忆

```go
var memories []model.ConversationMemory
db.Where("session_id = ? AND memory_type = ? AND is_deleted = ?", 
    sessionID, model.MemoryTypeLongTerm, false).
    Order("created_at DESC").
    Limit(10).
    Find(&memories)
```

### 向量相似度搜索

```go
var memories []model.ConversationMemory
db.Where("session_id = ? AND is_deleted = ?", sessionID, false).
    Where("(1 - (embedding <=> ?)) >= ?", queryVector, 0.7).
    Order(gorm.Expr("embedding <=> ?", queryVector)).
    Limit(5).
    Find(&memories)
```

### 获取最新摘要

```go
var summary model.ConversationSummary
db.Where("session_id = ? AND is_deleted = ?", sessionID, false).
    Order("created_at DESC").
    First(&summary)
```

### 获取上下文配置

```go
var context model.ConversationContext
db.Where("session_id = ? AND is_deleted = ?", sessionID, false).
    First(&context)
```

## 常量定义

### 记忆类型

```go
model.MemoryTypeShortTerm  // "short_term"
model.MemoryTypeLongTerm   // "long_term"
model.MemoryTypeSummary    // "summary"
```

### 上下文策略

```go
model.ContextStrategyAuto   // "auto"
model.ContextStrategyShort  // "short"
model.ContextStrategyFull   // "full"
```

### 摘要类型

```go
model.SummaryTypeIncremental  // "incremental"
model.SummaryTypeFull         // "full"
```

## 索引说明

### conversation_memories 表索引

- `idx_memories_tenant_session`: (tenant_id, session_id) - 快速查询租户会话的记忆
- `idx_memories_type`: memory_type - 按类型过滤
- `idx_memories_expires`: expires_at - 查找过期记忆
- `idx_memories_created`: created_at DESC - 按时间排序
- `idx_memories_embedding`: 向量索引 - 语义检索

### conversation_contexts 表索引

- `idx_contexts_tenant`: tenant_id - 租户过滤
- `idx_contexts_session`: session_id - 会话查询（唯一）

### conversation_summaries 表索引

- `idx_summaries_tenant_session`: (tenant_id, session_id) - 租户会话摘要
- `idx_summaries_created`: created_at DESC - 时间排序
- `idx_summaries_session_latest`: (session_id, created_at DESC) - 最新摘要

## 性能优化提示

### 1. 向量检索优化

```go
// 使用合适的相似度阈值
minSimilarity := 0.7  // 推荐值：0.6-0.8

// 限制返回数量
topK := 5  // 推荐值：3-10

// 使用索引提示（如果需要）
db.Set("gorm:query_hint", "/*+ IndexScan(conversation_memories idx_memories_embedding) */")
```

### 2. 批量操作

```go
// 批量插入记忆
memories := []*model.ConversationMemory{...}
db.CreateInBatches(memories, 100)

// 批量更新访问统计
db.Model(&model.ConversationMemory{}).
    Where("id IN ?", memoryIDs).
    Updates(map[string]interface{}{
        "access_count": gorm.Expr("access_count + 1"),
        "last_access_at": time.Now(),
    })
```

### 3. 定期清理

```go
// 清理过期记忆
db.Model(&model.ConversationMemory{}).
    Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
    Update("is_deleted", true)

// 清理低质量记忆
db.Model(&model.ConversationMemory{}).
    Where("importance < ? AND access_count < ?", 0.3, 2).
    Where("created_at < ?", time.Now().AddDate(0, 0, -90)).
    Update("is_deleted", true)
```

## 多租户隔离

### 查询时必须包含租户过滤

```go
// ✅ 正确：包含租户过滤
db.Where("tenant_id = ? AND session_id = ?", tenantID, sessionID).
    Find(&memories)

// ❌ 错误：缺少租户过滤
db.Where("session_id = ?", sessionID).
    Find(&memories)
```

### 使用作用域简化租户过滤

```go
// 定义租户作用域
func TenantScope(tenantID uuid.UUID) func(db *gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("tenant_id = ?", tenantID)
    }
}

// 使用作用域
db.Scopes(TenantScope(tenantID)).
    Where("session_id = ?", sessionID).
    Find(&memories)
```

## 故障排查

### 问题：向量索引创建失败

```bash
# 检查数据量
SELECT COUNT(*) FROM conversation_memories 
WHERE is_deleted = FALSE AND embedding IS NOT NULL;

# 如果少于 100 条，等待数据积累后再创建索引
```

### 问题：查询性能慢

```bash
# 检查索引使用情况
EXPLAIN ANALYZE 
SELECT * FROM conversation_memories 
WHERE tenant_id = '...' AND session_id = '...' 
AND is_deleted = FALSE;

# 检查索引是否存在
SELECT indexname FROM pg_indexes 
WHERE tablename = 'conversation_memories';
```

### 问题：向量检索不准确

```go
// 调整相似度阈值
minSimilarity := 0.6  // 降低阈值获取更多结果

// 增加返回数量
topK := 10  // 增加候选数量

// 检查向量维度是否正确
fmt.Printf("向量维度: %d\n", len(memory.Embedding))  // 应该是 1536
```

## 相关文件

- 迁移脚本：`internal/database/migrations/genkit_session_migration.go`
- 数据模型：`internal/model/genkit_session.go`
- 执行脚本：`scripts/genkit_session_migrate.go`
- 详细文档：`internal/database/migrations/README_GENKIT_SESSION.md`
- 测试文件：`internal/database/migrations/genkit_session_migration_test.go`
- 示例代码：`internal/model/genkit_session_example_test.go`

## 下一步

1. ✅ 数据库迁移和模型定义（已完成）
2. ⏭️ Repository 层实现
3. ⏭️ 缓存服务实现
4. ⏭️ 向量服务实现
5. ⏭️ Token 管理服务实现

参见：`.kiro/specs/genkit-session-management/tasks.md`
