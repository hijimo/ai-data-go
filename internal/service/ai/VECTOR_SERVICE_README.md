# 向量嵌入服务 (Vector Service)

## 概述

向量嵌入服务提供文本到向量的转换功能，用于支持语义搜索和记忆检索。该服务集成了 Google Genkit 的嵌入模型，支持单个文本和批量文本的向量生成。

## 功能特性

- ✅ 单个文本向量生成
- ✅ 批量文本向量生成
- ✅ 自动重试机制（指数退避）
- ✅ 批量处理优化
- ✅ 向量维度验证
- ✅ 上下文取消支持

## 技术规格

### 向量维度

- **维度**: 1536
- **模型**: `googleai/text-embedding-004`
- **距离度量**: Cosine（余弦相似度）

### 性能参数

- **默认批量大小**: 10个文本/批次
- **最大重试次数**: 3次
- **初始重试延迟**: 100ms
- **最大重试延迟**: 5s
- **退避倍数**: 2.0

## 使用方法

### 1. 创建服务实例

```go
import (
    "context"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
    "genkit-ai-service/internal/service/ai"
)

// 初始化 Genkit
ctx := context.Background()
g := genkit.Init(ctx,
    genkit.WithPlugins(&googlegenai.GoogleAI{
        APIKey: "your-api-key",
    }),
)

// 创建向量服务（使用默认配置）
vectorSvc, err := ai.NewVectorService(g, nil)
if err != nil {
    log.Fatal(err)
}

// 或使用自定义配置
config := &ai.VectorServiceConfig{
    EmbedderModel: "googleai/text-embedding-004",
    VectorDim:     1536,
    BatchSize:     20,
}
vectorSvc, err := ai.NewVectorService(g, config)
```

### 2. 生成单个文本向量

```go
ctx := context.Background()
text := "这是一段需要生成向量的文本"

embedding, err := vectorSvc.GenerateEmbedding(ctx, text)
if err != nil {
    log.Printf("生成向量失败: %v", err)
    return
}

fmt.Printf("向量维度: %d\n", len(embedding))
// 输出: 向量维度: 1536
```

### 3. 批量生成向量

```go
ctx := context.Background()
texts := []string{
    "第一段文本",
    "第二段文本",
    "第三段文本",
}

embeddings, err := vectorSvc.GenerateBatchEmbeddings(ctx, texts)
if err != nil {
    log.Printf("批量生成向量失败: %v", err)
    return
}

fmt.Printf("生成了 %d 个向量\n", len(embeddings))
// 输出: 生成了 3 个向量
```

### 4. 在记忆存储中使用

```go
// 存储记忆时生成向量
func (s *MemoryService) StoreMemory(ctx context.Context, content string) error {
    // 1. 生成向量
    embedding, err := s.vectorSvc.GenerateEmbedding(ctx, content)
    if err != nil {
        return fmt.Errorf("生成向量失败: %w", err)
    }
    
    // 2. 存储到 Qdrant
    err = s.qdrantClient.UpsertVector(ctx, &storage.UpsertVectorRequest{
        TenantID:   tenantID,
        MemoryID:   memoryID,
        SessionID:  sessionID,
        MemoryType: "long_term",
        Vector:     embedding,
        Importance: 0.8,
    })
    
    return err
}
```

### 5. 在记忆检索中使用

```go
// 检索记忆时生成查询向量
func (s *MemoryService) SearchMemories(ctx context.Context, query string) ([]*Memory, error) {
    // 1. 生成查询向量
    queryEmbedding, err := s.vectorSvc.GenerateEmbedding(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("生成查询向量失败: %w", err)
    }
    
    // 2. 执行向量检索
    results, err := s.qdrantClient.SearchVectors(ctx, &storage.SearchVectorRequest{
        TenantID:    tenantID,
        SessionID:   &sessionID,
        QueryVector: queryEmbedding,
        TopK:        5,
        MinScore:    0.7,
    })
    
    return results, err
}
```

## 配置说明

### VectorServiceConfig

```go
type VectorServiceConfig struct {
    // 嵌入模型名称
    // 默认: "googleai/text-embedding-004"
    EmbedderModel string
    
    // 向量维度
    // 默认: 1536
    VectorDim int
    
    // 批量处理大小
    // 默认: 10
    BatchSize int
}
```

### RetryConfig

```go
type RetryConfig struct {
    // 最大重试次数
    // 默认: 3
    MaxRetries int
    
    // 初始重试延迟
    // 默认: 100ms
    InitialDelay time.Duration
    
    // 最大重试延迟
    // 默认: 5s
    MaxDelay time.Duration
    
    // 退避倍数
    // 默认: 2.0
    Multiplier float64
}
```

## 错误处理

### 常见错误

1. **文本为空**

   ```go
   _, err := vectorSvc.GenerateEmbedding(ctx, "")
   // 错误: "文本不能为空"
   ```

2. **向量维度不匹配**

   ```go
   // 如果返回的向量维度不是1536
   // 错误: "向量维度不匹配: 期望 1536, 实际 XXX"
   ```

3. **API调用失败**

   ```go
   // 网络错误、API密钥无效等
   // 错误: "重试3次后仍然失败: ..."
   ```

### 错误处理示例

```go
embedding, err := vectorSvc.GenerateEmbedding(ctx, text)
if err != nil {
    // 记录错误
    logger.ErrorContext(ctx, "生成向量失败",
        "error", err,
        "text_length", len(text),
    )
    
    // 根据错误类型处理
    if strings.Contains(err.Error(), "文本不能为空") {
        return errors.NewBadRequestError("请提供有效的文本内容")
    }
    
    if strings.Contains(err.Error(), "重试") {
        return errors.NewServiceUnavailableError("向量服务暂时不可用，请稍后重试")
    }
    
    return errors.NewInternalError(err)
}
```

## 性能优化

### 1. 批量处理

对于多个文本，使用批量接口可以显著提高性能：

```go
// ❌ 不推荐：逐个生成
for _, text := range texts {
    embedding, _ := vectorSvc.GenerateEmbedding(ctx, text)
    embeddings = append(embeddings, embedding)
}

// ✅ 推荐：批量生成
embeddings, err := vectorSvc.GenerateBatchEmbeddings(ctx, texts)
```

### 2. 异步处理

对于非关键路径的向量生成，可以使用异步处理：

```go
// 异步生成向量
go func() {
    ctx := context.Background()
    embedding, err := vectorSvc.GenerateEmbedding(ctx, content)
    if err != nil {
        logger.WarnContext(ctx, "异步生成向量失败", "error", err)
        return
    }
    
    // 存储向量
    s.storeVector(ctx, memoryID, embedding)
}()
```

### 3. 缓存向量

对于频繁查询的文本，可以缓存其向量：

```go
// 尝试从缓存获取
cacheKey := fmt.Sprintf("embedding:%s", hash(text))
var embedding []float32
if err := cache.Get(ctx, cacheKey, &embedding); err == nil {
    return embedding, nil
}

// 生成新向量
embedding, err := vectorSvc.GenerateEmbedding(ctx, text)
if err != nil {
    return nil, err
}

// 缓存向量（TTL: 1小时）
cache.Set(ctx, cacheKey, embedding, time.Hour)
```

## 监控指标

建议监控以下指标：

```go
// 向量生成次数
metrics.IncrementCounter("vector_generation_total")

// 向量生成延迟
metrics.RecordDuration("vector_generation_duration_ms", duration)

// 向量生成失败次数
metrics.IncrementCounter("vector_generation_errors_total")

// 批量大小分布
metrics.RecordHistogram("vector_batch_size", len(texts))

// 重试次数
metrics.IncrementCounter("vector_generation_retries_total")
```

## 测试

### 单元测试

```bash
go test ./internal/service/ai/... -v
```

### 集成测试

需要配置真实的 Google AI API 密钥：

```bash
export GOOGLE_AI_API_KEY="your-api-key"
go test ./internal/service/ai/... -tags=integration -v
```

## 依赖项

- `github.com/firebase/genkit/go/genkit` - Genkit Go SDK
- `github.com/firebase/genkit/go/ai` - Genkit AI 接口
- `github.com/firebase/genkit/go/plugins/googlegenai` - Google AI 插件

## 相关文档

- [Genkit 会话管理模块设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [Qdrant 向量数据库使用指南](../../storage/QDRANT_README.md)
- [记忆管理服务文档](./memory_service.md)

## 注意事项

1. **API 配额**: Google AI API 有速率限制，请注意控制调用频率
2. **文本长度**: 过长的文本可能导致 API 调用失败，建议限制在 8000 字符以内
3. **向量维度**: 必须使用 1536 维向量以匹配 Qdrant 配置
4. **上下文取消**: 支持通过 context 取消长时间运行的操作
5. **错误重试**: 自动重试机制可以处理临时性错误，但不会重试客户端错误

## 更新日志

### v1.0.0 (2024-01-XX)

- ✅ 实现单个文本向量生成
- ✅ 实现批量文本向量生成
- ✅ 添加指数退避重试机制
- ✅ 添加向量维度验证
- ✅ 添加批量处理优化
- ✅ 添加单元测试
