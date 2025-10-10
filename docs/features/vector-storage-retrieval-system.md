# 向量存储与检索系统实现文档

## 概述

本文档描述了AI知识管理平台中向量存储与检索系统的实现。该系统提供了统一的向量存储接口、多厂商Embedding API集成以及高性能的向量检索功能。

## 系统架构

### 核心组件

1. **VectorProvider接口** - 向量存储的统一抽象接口
2. **EmbeddingProvider接口** - 文本向量化的统一接口
3. **向量管理器** - 管理多个向量存储提供商
4. **向量化管理器** - 管理多个向量化提供商
5. **向量服务** - 集成向量化和存储的高级服务

### 架构图

```
┌─────────────────┐    ┌─────────────────┐
│   VectorService │    │  Integration    │
│                 │    │     Layer       │
└─────────┬───────┘    └─────────────────┘
          │
    ┌─────┴─────┐
    │           │
┌───▼────┐  ┌──▼─────┐
│Vector  │  │Embedding│
│Manager │  │Manager  │
└───┬────┘  └──┬─────┘
    │          │
┌───▼────┐  ┌──▼─────┐
│Vector  │  │Embedding│
│Provider│  │Provider │
└────────┘  └────────┘
```

## 实现详情

### 1. VectorProvider抽象接口

#### 核心接口定义

```go
type VectorProvider interface {
    // 索引管理
    CreateIndex(ctx context.Context, req *CreateIndexRequest) error
    DeleteIndex(ctx context.Context, indexName string) error
    IndexExists(ctx context.Context, indexName string) (bool, error)
    
    // 向量操作
    InsertVectors(ctx context.Context, indexName string, vectors []Vector) error
    BatchInsertVectors(ctx context.Context, indexName string, vectors []Vector, batchSize int) error
    Search(ctx context.Context, indexName string, req *SearchRequest) ([]SearchResult, error)
    DeleteVectors(ctx context.Context, indexName string, ids []string) error
    UpdateVectors(ctx context.Context, indexName string, vectors []Vector) error
    GetVector(ctx context.Context, indexName string, id string) (*Vector, error)
    
    // 统计和健康检查
    GetStats(ctx context.Context, indexName string) (*IndexStats, error)
    HealthCheck(ctx context.Context) error
    Close() error
}
```

#### 支持的向量存储后端

- **ADB-PG** - 阿里云AnalyticDB PostgreSQL（已实现）
- **Pinecone** - 预留接口
- **Weaviate** - 预留接口
- **Chroma** - 预留接口
- **Milvus** - 预留接口

#### 配置管理

```go
type Config struct {
    Provider ProviderType           `json:"provider"`
    Settings map[string]interface{} `json:"settings"`
}
```

支持运行时切换不同的向量存储提供商，便于未来迁移和扩展。

### 2. ADB-PG向量适配器

#### 实现特性

- **连接管理** - 支持连接池配置和自动重连
- **索引创建** - 支持HNSW和IVF索引类型
- **距离度量** - 支持余弦、欧几里得、点积距离
- **批量操作** - 优化的批量插入和更新
- **元数据过滤** - 基于JSONB的灵活过滤
- **统计信息** - 实时的索引统计和性能监控

#### 配置示例

```go
config := &Config{
    Provider: ProviderADBPG,
    Settings: map[string]interface{}{
        "host":           "localhost",
        "port":           5432,
        "database":       "vectors",
        "username":       "user",
        "password":       "pass",
        "ssl_mode":       "require",
        "max_open_conns": 25,
        "max_idle_conns": 5,
    },
}
```

### 3. Embedding API集成

#### 支持的提供商

- **OpenAI** - text-embedding-ada-002等模型
- **Azure OpenAI** - Azure部署的OpenAI模型
- **千问** - 阿里云千问向量化模型
- **百川** - 百川智能向量化模型
- **智谱** - 智谱AI向量化模型

#### 统一接口

```go
type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    GetDimension() int
    GetModelName() string
    HealthCheck(ctx context.Context) error
}
```

#### 异步处理

实现了异步向量化处理器，支持：

- 批量文本处理
- 任务状态跟踪
- 进度监控
- 错误处理和重试

### 4. 工具函数

#### VectorUtils类

提供了丰富的向量操作工具：

```go
// 向量归一化
func (u *VectorUtils) NormalizeVector(vector []float32) []float32

// 相似度计算
func (u *VectorUtils) CosineSimilarity(a, b []float32) float32
func (u *VectorUtils) EuclideanDistance(a, b []float32) float32
func (u *VectorUtils) DotProduct(a, b []float32) float32

// 向量验证
func (u *VectorUtils) ValidateVector(vector []float32, expectedDim int) error

// 批量处理
func (u *VectorUtils) BatchVectors(vectors []Vector, batchSize int) [][]Vector

// 结果排序和过滤
func (u *VectorUtils) SortSearchResults(results []SearchResult, ascending bool)
func (u *VectorUtils) FilterSearchResults(results []SearchResult, filters map[string]interface{}) []SearchResult
```

### 5. 错误处理

#### 错误分类

- **连接错误** - 数据库连接失败
- **索引错误** - 索引不存在或创建失败
- **向量错误** - 向量格式错误或操作失败
- **搜索错误** - 搜索参数错误或执行失败
- **配置错误** - 配置验证失败

#### 错误包装

```go
type VectorError struct {
    Op       string // 操作名称
    Provider string // 提供商名称
    Index    string // 索引名称
    Err      error  // 原始错误
}
```

### 6. 集成服务

#### VectorService

提供了高级的集成服务，简化了向量化和存储的使用：

```go
// 索引和存储文档
func (s *VectorService) IndexAndStore(ctx context.Context, 
    vectorProviderName, embeddingProviderName, indexName string, 
    documents []Document) error

// 搜索相似文档
func (s *VectorService) SearchSimilar(ctx context.Context, 
    vectorProviderName, embeddingProviderName, indexName, queryText string, 
    topK int, filters map[string]interface{}) ([]SearchResult, error)

// 批量处理
func (s *VectorService) BatchIndexAndStore(ctx context.Context, 
    vectorProviderName, embeddingProviderName, indexName string, 
    documents []Document, batchSize int) error
```

## 使用示例

### 基本使用

```go
// 创建管理器
vectorFactory := NewDefaultProviderFactory()
vectorManager := NewManager(vectorFactory)

embeddingFactory := NewDefaultEmbeddingProviderFactory()
embeddingManager := NewEmbeddingManager(embeddingFactory)

// 注册提供商
vectorConfig := &Config{
    Provider: ProviderADBPG,
    Settings: map[string]interface{}{
        "host": "localhost",
        "port": 5432,
        "database": "vectors",
        "username": "user",
        "password": "pass",
    },
}
vectorManager.RegisterProvider(ctx, "adbpg", vectorConfig)

embeddingConfig := &EmbeddingConfig{
    Provider: EmbeddingProviderOpenAI,
    Model: "text-embedding-ada-002",
    APIKey: "your-api-key",
    Dimension: 1536,
}
embeddingManager.RegisterProvider("openai", embeddingConfig)

// 创建集成服务
service := NewVectorService(vectorManager, embeddingManager)

// 索引文档
documents := []Document{
    {
        ID: "doc1",
        Content: "这是第一个文档的内容",
        Metadata: map[string]interface{}{
            "category": "技术",
            "author": "张三",
        },
    },
}

err := service.IndexAndStore(ctx, "adbpg", "openai", "my_index", documents)

// 搜索相似文档
results, err := service.SearchSimilar(ctx, "adbpg", "openai", "my_index", 
    "技术文档", 10, map[string]interface{}{
        "category": "技术",
    })
```

### 异步处理

```go
// 创建异步处理器
processor := NewAsyncEmbeddingProcessor(embeddingManager, 4)
processor.Start()
defer processor.Stop()

// 提交任务
texts := []string{"文本1", "文本2", "文本3"}
task, err := processor.SubmitTask("openai", texts)

// 查询任务状态
status, err := processor.GetTask(task.ID)
```

## 性能优化

### 1. 批量操作

- 支持批量向量插入，减少网络往返
- 可配置的批次大小，平衡内存使用和性能
- 异步处理大量文档

### 2. 连接池管理

- 数据库连接池配置
- 连接生命周期管理
- 自动重连机制

### 3. 索引优化

- 支持HNSW和IVF索引类型
- 自动计算索引参数
- 内存使用估算

### 4. 缓存策略

- 向量检索结果缓存
- LLM响应缓存
- 文档元数据缓存

## 监控和可观测性

### 1. 健康检查

- 向量存储连接状态
- 向量化API可用性
- 索引状态监控

### 2. 统计信息

- 向量数量统计
- 索引大小监控
- 查询性能指标

### 3. 错误处理

- 详细的错误分类
- 错误重试机制
- 错误日志记录

## 扩展性

### 1. 提供商扩展

通过工厂模式，可以轻松添加新的向量存储和向量化提供商：

```go
type CustomVectorProvider struct {
    // 自定义实现
}

func (p *CustomVectorProvider) CreateIndex(ctx context.Context, req *CreateIndexRequest) error {
    // 实现索引创建逻辑
}

// 实现其他接口方法...
```

### 2. 配置扩展

支持灵活的配置管理，可以根据需要添加新的配置选项。

### 3. 功能扩展

- 支持更多距离度量算法
- 支持更复杂的过滤条件
- 支持向量压缩和量化

## 测试覆盖

### 1. 单元测试

- 接口实现测试
- 工具函数测试
- 错误处理测试

### 2. 集成测试

- 端到端流程测试
- 多提供商集成测试
- 性能基准测试

### 3. 模拟测试

- Mock提供商实现
- 异步处理测试
- 错误场景测试

## 总结

向量存储与检索系统为AI知识管理平台提供了强大的语义搜索能力。通过统一的接口设计、多提供商支持和丰富的功能特性，该系统能够满足各种规模和场景的需求。

### 主要优势

1. **统一接口** - 屏蔽不同提供商的差异
2. **高性能** - 优化的批量操作和索引策略
3. **可扩展** - 易于添加新的提供商和功能
4. **可靠性** - 完善的错误处理和重试机制
5. **可观测** - 丰富的监控和统计信息

### 未来规划

1. 支持更多向量存储后端
2. 实现向量压缩和量化
3. 添加更多距离度量算法
4. 优化大规模数据处理性能
5. 集成更多向量化模型

该系统为平台的RAG功能、文档检索和智能问答提供了坚实的技术基础。
