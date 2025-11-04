# Qdrant 向量服务使用指南

## 概述

本向量服务集成了 Google AI 嵌入模型和 Qdrant 向量数据库，提供完整的文本向量化、存储和检索能力，并实现了严格的多租户数据隔离。

## 核心特性

- ✅ **Google AI 嵌入模型**: 使用 text-embedding-004 模型生成 768 维向量
- ✅ **Qdrant 向量数据库**: 高性能向量存储和检索
- ✅ **多租户隔离**: 基于 tenant_id 的严格数据隔离
- ✅ **批量处理**: 支持批量向量生成和存储
- ✅ **重试机制**: 内置指数退避重试策略
- ✅ **性能优化**: 针对多租户场景的索引优化

## 多租户隔离机制

### 设计原则

根据 Qdrant 官方多租户最佳实践，本服务采用 **单集合多租户** 架构：

1. **所有租户共享一个集合**：`conversation_memories`
2. **通过 payload 字段隔离**：每个向量都包含 `tenant_id` 字段
3. **查询时强制过滤**：所有搜索操作都必须包含 `tenant_id` 过滤条件
4. **索引优化**：配置 `payload_m=16, m=0` 为每个租户构建独立索引

### 安全保证

- ❌ **不可能跨租户访问**：所有查询都强制包含 tenant_id 过滤
- ✅ **自动租户隔离**：API 层面自动添加租户过滤条件
- ✅ **审计日志**：记录所有向量操作的租户信息

## 配置

### 环境变量

在 `.env` 文件中配置以下变量：

```env
# Google AI 嵌入模型
GEMINI_API_KEY=your_google_ai_api_key

# Qdrant 配置
QDRANT_ENDPOINT=https://your-cluster.qdrant.io
QDRANT_ACCESS_KEY=your_qdrant_api_key
QDRANT_CLUSTER_ID=your_cluster_id
```

### 服务配置

```go
import (
    "genkit-ai-service/internal/service"
    "genkit-ai-service/internal/logger"
    "os"
)

config := &service.VectorServiceConfig{
    Provider:       service.EmbeddingProviderGoogleAI,
    APIKey:         os.Getenv("GEMINI_API_KEY"),
    Model:          "text-embedding-004",
    Dimension:      768,
    BatchSize:      100,
    MaxRetries:     3,
    QdrantEndpoint: os.Getenv("QDRANT_ENDPOINT"),
    QdrantAPIKey:   os.Getenv("QDRANT_ACCESS_KEY"),
    CollectionName: "conversation_memories",
}

log := logger.NewLogger()
vectorService, err := service.NewVectorService(config, log)
if err != nil {
    log.Fatal("创建向量服务失败", logger.Fields{"error": err})
}
```

## 使用示例

### 1. 初始化集合

首次使用前，需要确保集合存在：

```go
ctx := context.Background()

// 确保集合存在（如果不存在则自动创建）
err := vectorService.EnsureCollection(ctx)
if err != nil {
    log.Fatal("初始化集合失败", logger.Fields{"error": err})
}
```

### 2. 生成并存储向量

```go
import "github.com/google/uuid"

// 租户ID和会话ID
tenantID := uuid.MustParse("tenant-uuid-here")
sessionID := uuid.MustParse("session-uuid-here")

// 要向量化的文本
text := "用户询问：如何使用 Qdrant 实现多租户隔离？"

// 生成向量
vector, err := vectorService.GenerateEmbedding(ctx, text)
if err != nil {
    log.Error("生成向量失败", logger.Fields{"error": err})
    return
}

// 存储向量
err = vectorService.StoreVector(ctx, &service.StoreVectorRequest{
    PointID:   uuid.New().String(),
    TenantID:  tenantID,
    SessionID: sessionID,
    Content:   text,
    Vector:    vector,
    Metadata: map[string]interface{}{
        "message_type": "user_query",
        "timestamp":    time.Now().Unix(),
        "importance":   0.8,
    },
})

if err != nil {
    log.Error("存储向量失败", logger.Fields{"error": err})
    return
}

log.Info("向量存储成功")
```

### 3. 批量存储向量

```go
// 准备多个文本
texts := []string{
    "第一条对话内容",
    "第二条对话内容",
    "第三条对话内容",
}

// 批量生成向量
vectors, err := vectorService.GenerateEmbeddings(ctx, texts)
if err != nil {
    log.Error("批量生成向量失败", logger.Fields{"error": err})
    return
}

// 构建批量存储请求
requests := make([]*service.StoreVectorRequest, len(texts))
for i, text := range texts {
    requests[i] = &service.StoreVectorRequest{
        PointID:   uuid.New().String(),
        TenantID:  tenantID,
        SessionID: sessionID,
        Content:   text,
        Vector:    vectors[i],
        Metadata: map[string]interface{}{
            "index": i,
        },
    }
}

// 批量存储
err = vectorService.StoreVectors(ctx, requests)
if err != nil {
    log.Error("批量存储向量失败", logger.Fields{"error": err})
    return
}

log.Info("批量向量存储成功", logger.Fields{"count": len(texts)})
```

### 4. 向量相似度搜索（多租户隔离）

```go
// 搜索查询
queryText := "如何实现多租户隔离？"

// 执行搜索（自动进行租户隔离）
results, err := vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
    TenantID:       tenantID,        // 必需：租户ID
    SessionID:      &sessionID,      // 可选：限制在特定会话内搜索
    QueryText:      queryText,       // 查询文本（自动生成向量）
    Limit:          5,               // 返回前5个结果
    ScoreThreshold: 0.7,             // 相似度阈值
    Filter: map[string]interface{}{ // 额外过滤条件
        "message_type": "user_query",
    },
})

if err != nil {
    log.Error("向量搜索失败", logger.Fields{"error": err})
    return
}

// 处理搜索结果
for i, result := range results {
    log.Info("搜索结果", logger.Fields{
        "rank":    i + 1,
        "score":   result.Score,
        "content": result.Content,
        "pointID": result.PointID,
    })
}
```

### 5. 使用向量直接搜索

```go
// 如果已经有向量，可以直接搜索
queryVector, err := vectorService.GenerateEmbedding(ctx, queryText)
if err != nil {
    return err
}

results, err := vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
    TenantID:       tenantID,
    QueryVector:    queryVector,  // 直接提供向量
    Limit:          10,
    ScoreThreshold: 0.75,
})
```

### 6. 跨会话搜索

```go
// 不指定 SessionID，在整个租户范围内搜索
results, err := vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
    TenantID:       tenantID,
    QueryText:      "相关问题",
    Limit:          10,
    ScoreThreshold: 0.7,
})
```

### 7. 删除向量

```go
// 删除单个向量
pointID := "point-id-to-delete"
err := vectorService.DeleteVector(ctx, tenantID, pointID)
if err != nil {
    log.Error("删除向量失败", logger.Fields{"error": err})
    return
}

// 根据条件批量删除
err = vectorService.DeleteVectorsByFilter(ctx, tenantID, map[string]interface{}{
    "session_id": sessionID.String(),
})
if err != nil {
    log.Error("批量删除向量失败", logger.Fields{"error": err})
    return
}
```

## 与 Repository 层集成

### 存储记忆时自动生成向量

```go
// internal/repository/memory_repository_impl.go

func (r *memoryRepositoryImpl) Create(ctx context.Context, memory *model.ConversationMemory) error {
    // 生成向量
    vector, err := r.vectorService.GenerateEmbedding(ctx, memory.Content)
    if err != nil {
        return fmt.Errorf("生成向量失败: %w", err)
    }

    // 存储到 Qdrant
    err = r.vectorService.StoreVector(ctx, &service.StoreVectorRequest{
        PointID:   memory.ID.String(),
        TenantID:  memory.TenantID,
        SessionID: memory.SessionID,
        Content:   memory.Content,
        Vector:    vector,
        Metadata: map[string]interface{}{
            "memory_type": memory.MemoryType,
            "importance":  memory.Importance,
            "token_count": memory.TokenCount,
        },
    })

    if err != nil {
        return fmt.Errorf("存储向量失败: %w", err)
    }

    // 存储到 PostgreSQL（不包含向量）
    return r.db.Create(memory).Error
}
```

### 向量检索

```go
func (r *memoryRepositoryImpl) SearchByVector(
    ctx context.Context,
    tenantID uuid.UUID,
    sessionID uuid.UUID,
    queryText string,
    topK int,
    minSimilarity float32,
) ([]*model.ConversationMemory, error) {
    // 执行向量搜索
    results, err := r.vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
        TenantID:       tenantID,
        SessionID:      &sessionID,
        QueryText:      queryText,
        Limit:          topK,
        ScoreThreshold: minSimilarity,
    })

    if err != nil {
        return nil, err
    }

    // 根据搜索结果从数据库加载完整记忆
    memoryIDs := make([]uuid.UUID, len(results))
    for i, result := range results {
        memoryIDs[i] = uuid.MustParse(result.PointID)
    }

    var memories []*model.ConversationMemory
    err = r.db.Where("id IN ?", memoryIDs).Find(&memories).Error
    if err != nil {
        return nil, err
    }

    return memories, nil
}
```

## 性能优化

### 1. 批量操作

```go
// ✅ 推荐：批量处理
texts := []string{"text1", "text2", "text3", ...}
vectors, err := vectorService.GenerateEmbeddings(ctx, texts)

requests := make([]*service.StoreVectorRequest, len(texts))
// ... 构建请求
vectorService.StoreVectors(ctx, requests)

// ❌ 不推荐：逐个处理
for _, text := range texts {
    vector, _ := vectorService.GenerateEmbedding(ctx, text)
    vectorService.StoreVector(ctx, &service.StoreVectorRequest{...})
}
```

### 2. 异步处理

```go
// 对于非关键路径的向量生成，使用异步处理
go func() {
    vector, err := vectorService.GenerateEmbedding(context.Background(), text)
    if err != nil {
        log.Error("异步向量生成失败", logger.Fields{"error": err})
        return
    }

    vectorService.StoreVector(context.Background(), &service.StoreVectorRequest{
        // ... 请求参数
    })
}()
```

### 3. 索引优化

集合创建时已自动配置多租户优化：

```go
// 自动配置（无需手动设置）
HnswConfig: &qdrant.HnswConfigDiff{
    PayloadM: 16,  // 为每个租户构建独立索引
    M:        0,   // 禁用全局索引
}
```

## 监控和日志

### 日志级别

- **Info**: 正常操作（初始化成功、向量生成成功、搜索成功）
- **Warn**: 重试操作、索引创建失败（非致命）
- **Error**: 失败操作（达到最大重试次数、存储失败、搜索失败）

### 关键日志字段

```go
// 向量生成
logger.Fields{
    "textLength": len(text),
    "dimension":  dimension,
    "duration":   duration,
    "attempt":    attempt,
}

// 向量存储
logger.Fields{
    "pointID":   pointID,
    "tenantID":  tenantID,
    "sessionID": sessionID,
}

// 向量搜索
logger.Fields{
    "tenantID":    tenantID,
    "resultCount": len(results),
    "queryText":   queryText,
}
```

## 安全最佳实践

### 1. 租户ID验证

```go
// ✅ 始终验证租户ID
if tenantID == uuid.Nil {
    return fmt.Errorf("租户ID不能为空")
}

// ✅ 从JWT中获取租户ID，不信任客户端传入
claims := middleware.GetJWTClaims(ctx)
tenantID := uuid.MustParse(claims.TenantID)
```

### 2. 强制租户过滤

```go
// ✅ 所有搜索操作都必须包含租户ID
results, err := vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
    TenantID: tenantID,  // 必需字段
    // ...
})

// ❌ 永远不要允许跨租户搜索
// 服务层已经强制要求 TenantID，无法绕过
```

### 3. 审计日志

```go
// 记录所有向量操作
log.InfoContext(ctx, "向量操作", logger.Fields{
    "operation": "search",
    "tenantID":  tenantID,
    "userID":    userID,
    "sessionID": sessionID,
})
```

## 故障排查

### 问题：集合创建失败

**症状**: `EnsureCollection` 返回错误

**可能原因**:

1. Qdrant 端点配置错误
2. API 密钥无效
3. 网络连接问题

**解决步骤**:

1. 检查 `QDRANT_ENDPOINT` 和 `QDRANT_ACCESS_KEY`
2. 测试网络连接：`curl -H "api-key: YOUR_KEY" https://your-endpoint/collections`
3. 查看 Qdrant 日志

### 问题：搜索结果为空

**症状**: `SearchVectors` 返回空结果

**可能原因**:

1. 租户ID不匹配
2. 相似度阈值过高
3. 向量未正确存储

**解决步骤**:

1. 验证租户ID是否正确
2. 降低 `ScoreThreshold`（如 0.5）
3. 检查向量是否已存储：查看 Qdrant 控制台

### 问题：跨租户数据泄露

**症状**: 担心租户A能看到租户B的数据

**解决方案**:

- ✅ **不可能发生**：所有搜索操作都强制包含 `tenant_id` 过滤
- ✅ **代码层面保证**：`SearchVectors` 方法强制要求 `TenantID` 参数
- ✅ **Qdrant 层面隔离**：查询时自动添加 `tenant_id` 过滤条件

## 测试

### 单元测试

```go
func TestVectorServiceMultiTenancy(t *testing.T) {
    // 创建两个租户
    tenant1 := uuid.New()
    tenant2 := uuid.New()

    // 租户1存储向量
    vectorService.StoreVector(ctx, &service.StoreVectorRequest{
        PointID:  "point1",
        TenantID: tenant1,
        Content:  "租户1的数据",
        Vector:   vector1,
    })

    // 租户2存储向量
    vectorService.StoreVector(ctx, &service.StoreVectorRequest{
        PointID:  "point2",
        TenantID: tenant2,
        Content:  "租户2的数据",
        Vector:   vector2,
    })

    // 租户1搜索，应该只能看到自己的数据
    results, _ := vectorService.SearchVectors(ctx, &service.SearchVectorRequest{
        TenantID:  tenant1,
        QueryText: "数据",
        Limit:     10,
    })

    // 验证结果只包含租户1的数据
    for _, result := range results {
        assert.Equal(t, tenant1.String(), result.Metadata["tenant_id"])
    }
}
```

## 参考资料

- [Qdrant 多租户指南](https://qdrant.tech/documentation/guides/multiple-partitions/)
- [Google AI Embeddings API](https://ai.google.dev/docs/embeddings_guide)
- [Qdrant Go Client](https://github.com/qdrant/go-client)
