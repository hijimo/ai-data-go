# 向量服务 (Vector Service)

## 概述

向量服务提供文本向量化能力，支持 Google AI 和 OpenAI 两种嵌入模型提供商。该服务用于将文本转换为向量表示，以支持语义检索和相似度计算。

## 功能特性

- **多提供商支持**: 支持 Google AI 和 OpenAI 嵌入模型
- **批量处理**: 自动分批处理大量文本，优化性能
- **重试机制**: 内置指数退避重试策略，提高可靠性
- **类型安全**: 使用 pgvector.Vector 类型，与 PostgreSQL pgvector 扩展兼容
- **性能监控**: 记录详细的执行日志和性能指标

## 接口定义

### VectorService

```go
type VectorService interface {
    // 生成单个文本的向量
    GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error)
    
    // 批量生成文本向量
    GenerateEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error)
    
    // 获取向量维度
    GetEmbeddingDimension() int
}
```

## 配置

### VectorServiceConfig

```go
type VectorServiceConfig struct {
    // 嵌入模型提供商 ("google" 或 "openai")
    Provider EmbeddingProvider
    
    // API 密钥
    APIKey string
    
    // 模型名称
    Model string
    
    // 向量维度
    Dimension int
    
    // 批量处理大小
    BatchSize int
    
    // 最大重试次数
    MaxRetries int
}
```

### 默认值

| 配置项 | Google AI 默认值 | OpenAI 默认值 |
|--------|-----------------|---------------|
| Model | text-embedding-004 | text-embedding-3-small |
| Dimension | 768 | 1536 |
| BatchSize | 100 | 100 |
| MaxRetries | 3 | 3 |

## 使用示例

### 创建向量服务

#### Google AI

```go
import (
    "genkit-ai-service/internal/service"
    "genkit-ai-service/internal/logger"
)

// 创建配置
config := &service.VectorServiceConfig{
    Provider:  service.EmbeddingProviderGoogleAI,
    APIKey:    "your-google-api-key",
    Model:     "text-embedding-004",
    Dimension: 768,
}

// 创建服务
log := logger.NewLogger()
vectorService, err := service.NewVectorService(config, log)
if err != nil {
    log.Fatal("创建向量服务失败", logger.Fields{"error": err})
}
```

#### OpenAI

```go
config := &service.VectorServiceConfig{
    Provider:  service.EmbeddingProviderOpenAI,
    APIKey:    "your-openai-api-key",
    Model:     "text-embedding-3-small",
    Dimension: 1536,
}

vectorService, err := service.NewVectorService(config, log)
```

### 生成单个向量

```go
ctx := context.Background()
text := "这是一段需要向量化的文本"

vector, err := vectorService.GenerateEmbedding(ctx, text)
if err != nil {
    log.Error("生成向量失败", logger.Fields{"error": err})
    return
}

log.Info("向量生成成功", logger.Fields{
    "dimension": vectorService.GetEmbeddingDimension(),
})
```

### 批量生成向量

```go
texts := []string{
    "第一段文本",
    "第二段文本",
    "第三段文本",
    // ... 更多文本
}

vectors, err := vectorService.GenerateEmbeddings(ctx, texts)
if err != nil {
    log.Error("批量生成向量失败", logger.Fields{"error": err})
    return
}

log.Info("批量向量生成成功", logger.Fields{
    "totalTexts":   len(texts),
    "totalVectors": len(vectors),
})
```

### 存储向量到数据库

```go
import (
    "genkit-ai-service/internal/model"
    "github.com/google/uuid"
)

// 生成向量
text := "需要存储的对话内容"
vector, err := vectorService.GenerateEmbedding(ctx, text)
if err != nil {
    return err
}

// 创建记忆对象
memory := &model.ConversationMemory{
    TenantID:   tenantID,
    SessionID:  sessionID,
    MemoryType: model.MemoryTypeLongTerm,
    Content:    text,
    Embedding:  vector,
    TokenCount: len(text) / 4, // 粗略估算
    Importance: 0.8,
}

// 保存到数据库
err = db.Create(memory).Error
```

## 性能优化

### 批量处理

向量服务自动将大量文本分批处理，以优化性能和避免 API 限制：

```go
// 自动分批处理 1000 个文本
texts := make([]string, 1000)
// ... 填充文本

// 服务会自动分成 10 个批次（每批 100 个）
vectors, err := vectorService.GenerateEmbeddings(ctx, texts)
```

### 重试策略

服务内置指数退避重试机制：

- 第 1 次重试：等待 1 秒
- 第 2 次重试：等待 2 秒
- 第 3 次重试：等待 3 秒

### 异步处理

对于非关键路径的向量生成，建议使用异步处理：

```go
// 异步生成向量
go func() {
    vector, err := vectorService.GenerateEmbedding(context.Background(), text)
    if err != nil {
        log.Error("异步向量生成失败", logger.Fields{"error": err})
        return
    }
    
    // 更新数据库
    db.Model(&memory).Update("embedding", vector)
}()
```

## 错误处理

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| "配置不能为空" | 未提供配置 | 创建并传入有效配置 |
| "API 密钥不能为空" | 未设置 API 密钥 | 在配置中设置有效的 API 密钥 |
| "文本不能为空" | 传入空字符串 | 确保文本内容不为空 |
| "生成向量失败" | API 调用失败 | 检查网络连接和 API 密钥 |
| "找不到嵌入器" | 模型名称错误 | 使用正确的模型名称 |

### 错误处理示例

```go
vector, err := vectorService.GenerateEmbedding(ctx, text)
if err != nil {
    // 记录错误
    log.Error("向量生成失败", logger.Fields{
        "error": err,
        "text":  text,
    })
    
    // 根据错误类型采取不同措施
    if strings.Contains(err.Error(), "API 密钥") {
        // API 密钥问题
        return fmt.Errorf("请检查 API 密钥配置")
    }
    
    if strings.Contains(err.Error(), "已达最大重试次数") {
        // 网络或服务问题
        return fmt.Errorf("服务暂时不可用，请稍后重试")
    }
    
    return err
}
```

## 监控和日志

### 日志级别

- **Info**: 正常操作（初始化成功、向量生成成功）
- **Warn**: 重试操作
- **Error**: 失败操作（达到最大重试次数）

### 日志字段

```go
// 单个向量生成
logger.Fields{
    "textLength": len(text),
    "dimension":  dimension,
    "duration":   duration,
    "attempt":    attempt,
}

// 批量向量生成
logger.Fields{
    "totalTexts":   totalTexts,
    "totalVectors": len(vectors),
    "duration":     duration,
    "avgPerText":   avgDuration,
    "batchNum":     batchNum,
    "batchSize":    batchSize,
}
```

## 最佳实践

### 1. 选择合适的模型

- **Google AI text-embedding-004**: 适合中文和多语言场景，维度较小（768）
- **OpenAI text-embedding-3-small**: 适合英文场景，性价比高（1536 维度）
- **OpenAI text-embedding-3-large**: 最高质量，但成本较高（3072 维度）

### 2. 批量处理

```go
// ✅ 推荐：批量处理
texts := []string{"text1", "text2", "text3", ...}
vectors, err := vectorService.GenerateEmbeddings(ctx, texts)

// ❌ 不推荐：逐个处理
for _, text := range texts {
    vector, err := vectorService.GenerateEmbedding(ctx, text)
    // ...
}
```

### 3. 缓存向量

```go
// 检查缓存
cacheKey := fmt.Sprintf("vector:%s", hash(text))
var vector pgvector.Vector
if err := cache.Get(ctx, cacheKey, &vector); err == nil {
    return vector, nil
}

// 生成并缓存
vector, err := vectorService.GenerateEmbedding(ctx, text)
if err == nil {
    cache.Set(ctx, cacheKey, vector, 24*time.Hour)
}
```

### 4. 错误重试

服务已内置重试机制，但对于关键操作，可以在应用层再次重试：

```go
var vector pgvector.Vector
var err error

for i := 0; i < 3; i++ {
    vector, err = vectorService.GenerateEmbedding(ctx, text)
    if err == nil {
        break
    }
    time.Sleep(time.Duration(i+1) * time.Second)
}
```

### 5. 监控性能

```go
startTime := time.Now()
vectors, err := vectorService.GenerateEmbeddings(ctx, texts)
duration := time.Since(startTime)

// 记录性能指标
metrics.RecordVectorGeneration(len(texts), duration)

// 如果性能下降，发出告警
if duration > 10*time.Second {
    log.Warn("向量生成性能下降", logger.Fields{
        "duration":   duration,
        "totalTexts": len(texts),
    })
}
```

## 与其他服务集成

### 与 MemoryRepository 集成

```go
// 存储记忆时生成向量
func (s *memoryService) StoreMemory(ctx context.Context, req StoreMemoryRequest) error {
    // 生成向量
    vector, err := s.vectorService.GenerateEmbedding(ctx, req.Content)
    if err != nil {
        return fmt.Errorf("生成向量失败: %w", err)
    }
    
    // 创建记忆
    memory := &model.ConversationMemory{
        TenantID:   req.TenantID,
        SessionID:  req.SessionID,
        Content:    req.Content,
        Embedding:  vector,
        MemoryType: model.MemoryTypeLongTerm,
    }
    
    // 保存到数据库
    return s.memoryRepo.Create(ctx, memory)
}
```

### 与 ContextService 集成

```go
// 构建上下文时检索相关记忆
func (s *contextService) BuildContext(ctx context.Context, req BuildContextRequest) error {
    // 生成查询向量
    queryVector, err := s.vectorService.GenerateEmbedding(ctx, req.UserQuery)
    if err != nil {
        log.Warn("生成查询向量失败，跳过长期记忆检索", logger.Fields{"error": err})
        // 继续执行，不中断流程
    } else {
        // 执行向量检索
        memories, err := s.memoryRepo.SearchByVector(ctx, req.SessionID, queryVector, 5, 0.7)
        if err != nil {
            log.Warn("向量检索失败", logger.Fields{"error": err})
        }
        // 使用检索到的记忆
    }
    
    return nil
}
```

## 测试

### 单元测试

```go
func TestVectorService(t *testing.T) {
    // 使用测试配置
    config := &service.VectorServiceConfig{
        Provider:   service.EmbeddingProviderGoogleAI,
        APIKey:     os.Getenv("TEST_GOOGLE_API_KEY"),
        Dimension:  768,
        BatchSize:  10,
        MaxRetries: 2,
    }
    
    log := logger.NewLogger()
    vectorService, err := service.NewVectorService(config, log)
    require.NoError(t, err)
    
    // 测试单个向量生成
    vector, err := vectorService.GenerateEmbedding(context.Background(), "测试文本")
    assert.NoError(t, err)
    assert.Equal(t, 768, vectorService.GetEmbeddingDimension())
}
```

### 集成测试

```go
func TestVectorServiceIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 测试批量处理
    texts := make([]string, 50)
    for i := range texts {
        texts[i] = fmt.Sprintf("测试文本 %d", i)
    }
    
    vectors, err := vectorService.GenerateEmbeddings(context.Background(), texts)
    assert.NoError(t, err)
    assert.Equal(t, len(texts), len(vectors))
}
```

## 故障排查

### 问题：向量生成失败

**症状**: 调用 `GenerateEmbedding` 返回错误

**可能原因**:

1. API 密钥无效或过期
2. 网络连接问题
3. API 配额用尽
4. 模型名称错误

**解决步骤**:

1. 检查 API 密钥是否正确
2. 测试网络连接
3. 查看 API 使用配额
4. 验证模型名称

### 问题：批量处理性能差

**症状**: 批量生成向量耗时过长

**可能原因**:

1. 批量大小设置不当
2. 网络延迟高
3. API 限流

**解决步骤**:

1. 调整 `BatchSize` 配置
2. 使用异步处理
3. 实施缓存策略

### 问题：向量维度不匹配

**症状**: 数据库插入失败，提示向量维度不匹配

**可能原因**:

1. 配置的维度与实际模型输出不符
2. 数据库表定义的维度不正确

**解决步骤**:

1. 确认模型的实际输出维度
2. 更新配置中的 `Dimension` 值
3. 修改数据库表定义

## 参考资料

- [Google AI Embeddings API](https://ai.google.dev/docs/embeddings_guide)
- [OpenAI Embeddings API](https://platform.openai.com/docs/guides/embeddings)
- [pgvector 文档](https://github.com/pgvector/pgvector)
- [Genkit Go SDK](https://firebase.google.com/docs/genkit-go)
