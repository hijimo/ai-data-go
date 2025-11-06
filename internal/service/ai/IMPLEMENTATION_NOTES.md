# 向量嵌入服务实施说明

## 实施状态

✅ **已完成**：向量嵌入服务的核心接口和实现已完成

## 已实现的功能

1. ✅ **服务接口定义** (`vector_service.go`)
   - `GenerateEmbedding`: 单个文本向量生成接口
   - `GenerateBatchEmbeddings`: 批量文本向量生成接口

2. ✅ **服务实现** (`vector_service_impl.go`)
   - 完整的向量生成逻辑
   - 指数退避重试机制
   - 批量处理优化
   - 向量维度验证
   - 上下文取消支持

3. ✅ **配置管理**
   - `VectorServiceConfig`: 服务配置结构
   - `RetryConfig`: 重试策略配置
   - 默认值设置

4. ✅ **测试覆盖**
   - 单元测试 (`vector_service_test.go`)
   - 集成测试 (`vector_service_integration_test.go`)
   - 输入验证测试
   - 批量处理逻辑测试
   - 重试机制测试

5. ✅ **文档**
   - 详细的使用文档 (`VECTOR_SERVICE_README.md`)
   - API说明
   - 配置说明
   - 性能优化建议
   - 监控指标建议

## 当前实现说明

### Genkit API 集成

当前实现使用了 Genkit Go SDK 的嵌入API：

```go
// 初始化Genkit
g := genkit.Init(ctx,
    genkit.WithPlugins(&googlegenai.GoogleAI{
        APIKey: config.APIKey,
    }),
)

// 创建embedder
embedder := googlegenai.GoogleAIEmbedder(g, config.EmbedderModel)

// 调用embedder生成向量
req := &ai.EmbedRequest{
    Input: []*ai.Document{
        ai.DocumentFromText(text, nil),
    },
}
resp, err := embedder.Embed(ctx, req)
```

**注意事项**：

- Genkit实例需要正确初始化Google AI插件
- 模型名称格式：`text-embedding-004`（不需要googleai/前缀）
- 向量维度：1536（与Qdrant配置匹配）
- 使用`googlegenai.GoogleAIEmbedder`创建embedder实例

### 重试机制

实现了指数退避重试策略：

- 最大重试次数：3次
- 初始延迟：100ms
- 最大延迟：5s
- 退避倍数：2.0

### 批量处理

- 默认批量大小：10个文本/批次
- 自动分批处理大量文本
- 每个批次独立重试

## 集成指南

### 1. 初始化服务

```go
config := &ai.VectorServiceConfig{
    EmbedderModel: "text-embedding-004",
    VectorDim:     1536,
    BatchSize:     10,
    APIKey:        os.Getenv("GOOGLE_AI_API_KEY"),
}

vectorSvc, err := ai.NewVectorService(config)
if err != nil {
    log.Fatal(err)
}
```

### 2. 在记忆服务中使用

```go
type MemoryService struct {
    vectorSvc ai.VectorService
    qdrant    storage.QdrantClient
    // ...
}

func (s *MemoryService) StoreMemory(ctx context.Context, content string) error {
    // 生成向量
    embedding, err := s.vectorSvc.GenerateEmbedding(ctx, content)
    if err != nil {
        return fmt.Errorf("生成向量失败: %w", err)
    }
    
    // 存储到Qdrant
    err = s.qdrant.UpsertVector(ctx, &storage.UpsertVectorRequest{
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

### 3. 在记忆检索中使用

```go
func (s *MemoryService) SearchMemories(ctx context.Context, query string) ([]*Memory, error) {
    // 生成查询向量
    queryEmbedding, err := s.vectorSvc.GenerateEmbedding(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("生成查询向量失败: %w", err)
    }
    
    // 执行向量检索
    results, err := s.qdrant.SearchVectors(ctx, &storage.SearchVectorRequest{
        TenantID:    tenantID,
        SessionID:   &sessionID,
        QueryVector: queryEmbedding,
        TopK:        5,
        MinScore:    0.7,
    })
    
    return results, err
}
```

## 依赖注入配置

在 `cmd/server/main.go` 中配置依赖注入：

```go
// 初始化向量服务
vectorConfig := &ai.VectorServiceConfig{
    EmbedderModel: config.Genkit.EmbedderModel,
    VectorDim:     1536,
    BatchSize:     10,
    APIKey:        config.Genkit.APIKey,
}

vectorSvc, err := ai.NewVectorService(vectorConfig)
if err != nil {
    log.Fatal("初始化向量服务失败:", err)
}

// 创建记忆服务
memoryService := service.NewMemoryService(
    memoryRepo,
    vectorSvc,
    qdrantClient,
    logger,
)
```

## 环境变量配置

在 `.env` 文件中添加：

```bash
# Google AI API配置
GOOGLE_AI_API_KEY=your-api-key-here

# 向量服务配置
EMBEDDER_MODEL=text-embedding-004
VECTOR_DIM=1536
BATCH_SIZE=10
```

## 测试

### 运行单元测试

```bash
go test ./internal/service/ai/... -v
```

### 运行集成测试

需要设置Google AI API密钥：

```bash
export GOOGLE_AI_API_KEY="your-api-key"
go test ./internal/service/ai/... -tags=integration -v
```

## 性能考虑

1. **批量处理**：对于多个文本，使用`GenerateBatchEmbeddings`可以显著提高性能
2. **异步处理**：非关键路径的向量生成可以异步执行
3. **缓存**：频繁查询的文本向量可以缓存
4. **速率限制**：注意Google AI API的速率限制

## 监控指标

建议监控以下指标：

- `vector_generation_total`: 向量生成总次数
- `vector_generation_duration_ms`: 向量生成延迟
- `vector_generation_errors_total`: 向量生成失败次数
- `vector_batch_size`: 批量大小分布
- `vector_generation_retries_total`: 重试次数

## 下一步工作

1. ✅ 向量嵌入服务实现完成
2. ⏭️ 实现Token管理器（任务6）
3. ⏭️ 实现ContextService（任务7）
4. ⏭️ 实现MemoryService（任务8）
5. ⏭️ 实现SummaryService（任务9）

## 相关文档

- [向量服务使用文档](./VECTOR_SERVICE_README.md)
- [Qdrant向量数据库文档](../../storage/QDRANT_README.md)
- [Genkit会话管理设计文档](../../../.kiro/specs/genkit-session-management/design.md)
- [任务列表](../../../.kiro/specs/genkit-session-management/tasks.md)

## 注意事项

1. **API密钥安全**：不要将API密钥提交到版本控制系统
2. **向量维度**：必须与Qdrant配置的1536维匹配
3. **模型名称**：使用完整的模型名称格式 `googleai/text-embedding-004`
4. **错误处理**：实现了自动重试，但仍需要处理最终失败的情况
5. **上下文取消**：支持通过context取消长时间运行的操作

## 更新日志

### 2024-01-XX

- ✅ 实现向量嵌入服务接口
- ✅ 实现单个文本向量生成
- ✅ 实现批量文本向量生成
- ✅ 添加指数退避重试机制
- ✅ 添加向量维度验证
- ✅ 添加批量处理优化
- ✅ 编写单元测试
- ✅ 编写集成测试
- ✅ 编写使用文档
