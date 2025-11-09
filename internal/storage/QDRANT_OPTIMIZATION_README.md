# Qdrant 向量检索优化指南

## 概述

本文档描述了 Qdrant 向量数据库的性能优化功能，包括批量操作、缓存、异步处理和定期优化。

## 优化功能

### 1. Collection 配置优化

#### 分片配置

根据租户数量调整分片数量，提高并发性能：

```go
config := &storage.QdrantConfig{
    Endpoint:          "https://xxx.cloud.qdrant.io",
    APIKey:            "your-api-key",
    ShardNumber:       8,  // 根据租户数量调整（建议：租户数/100）
    ReplicationFactor: 2,  // 高可用配置
}
```

**建议值**：

- 小型部署（<100租户）：4个分片
- 中型部署（100-1000租户）：8个分片
- 大型部署（>1000租户）：16个分片

#### HNSW 索引优化

调整 HNSW 参数以平衡搜索速度和准确性：

```go
config := &storage.QdrantConfig{
    HnswM:              16,  // 图的连接度（默认：16）
    HnswEfConstruction: 100, // 构建时的搜索深度（默认：100）
}
```

**参数说明**：

- `HnswM`：控制图的连接度
  - 较小值（8-16）：更快的构建速度，较低的内存占用
  - 较大值（32-64）：更高的搜索准确性，更多的内存占用
- `HnswEfConstruction`：构建时的搜索深度
  - 较小值（50-100）：更快的构建速度
  - 较大值（200-400）：更高的索引质量

### 2. 批量向量操作

#### 批量插入

使用批量插入可以显著提高性能（比单条插入快5-10倍）：

```go
// 构建批量请求
reqs := []*storage.UpsertVectorRequest{
    {
        TenantID:   tenantID,
        MemoryID:   memoryID1,
        SessionID:  sessionID,
        MemoryType: "long_term",
        Vector:     vector1,
        Importance: 0.8,
    },
    {
        TenantID:   tenantID,
        MemoryID:   memoryID2,
        SessionID:  sessionID,
        MemoryType: "long_term",
        Vector:     vector2,
        Importance: 0.7,
    },
    // ... 更多请求
}

// 批量插入
err := qdrantClient.BatchUpsertVectors(ctx, reqs)
```

**性能对比**：

- 单条插入：~50ms/条
- 批量插入（100条）：~500ms（平均5ms/条）

### 3. 向量查询结果缓存

#### 启用缓存

创建带缓存的 Qdrant 客户端：

```go
// 创建缓存服务
cacheService := storage.NewCacheService(redisClient)

// 创建带缓存的 Qdrant 客户端
qdrantClient, err := storage.NewQdrantClientWithCache(config, cacheService)
```

#### 缓存策略

- **缓存键**：基于租户ID、会话ID、记忆类型和TopK
- **缓存时间**：30分钟（可配置）
- **缓存失效**：当插入新向量时自动失效相关缓存

**性能提升**：

- 缓存命中：~5ms（比直接查询快20倍）
- 缓存未命中：~100ms（正常查询时间）

### 4. 异步向量生成

#### 创建异步生成器

```go
// 创建向量生成器
vectorGenerator := ai.NewVectorService(genkitClient)

// 创建异步生成器
asyncGenerator := storage.NewAsyncVectorGenerator(
    vectorGenerator,
    qdrantClient,
    &storage.AsyncGeneratorConfig{
        QueueSize:      1000, // 任务队列大小
        WorkerCount:    5,    // 工作协程数量
        BatchQueueSize: 500,  // 批量队列大小
        BatchSize:      50,   // 批量大小
        BatchInterval:  5,    // 批量间隔（秒）
        TaskTimeout:    30,   // 任务超时（秒）
    },
)

// 启动异步生成器
err := asyncGenerator.Start(ctx)
```

#### 提交任务

```go
// 提交单个任务
task := &storage.VectorGenerateTask{
    TenantID:   tenantID,
    MemoryID:   memoryID,
    SessionID:  sessionID,
    MemoryType: "long_term",
    Content:    "用户消息内容",
    Importance: 0.8,
    Callback: func(err error) {
        if err != nil {
            log.Printf("向量生成失败: %v", err)
        } else {
            log.Printf("向量生成成功")
        }
    },
}

err := asyncGenerator.SubmitTask(task)

// 提交批量任务（优先使用批量处理）
err := asyncGenerator.SubmitBatchTask(task)
```

**优势**：

- 非阻塞：不影响主流程响应时间
- 批量处理：自动合并多个任务进行批量处理
- 错误重试：支持自动重试机制
- 队列管理：防止任务堆积

### 5. 定期优化

#### 创建优化器

```go
// 创建优化器
optimizer := storage.NewQdrantOptimizer(
    qdrantClient,
    &storage.OptimizerConfig{
        OptimizationInterval:   24,   // 优化间隔（小时）
        EnableAutoOptimization: true, // 启用自动优化
    },
)

// 启动优化器
err := optimizer.Start(ctx)
```

#### 手动触发优化

```go
// 立即执行优化
err := optimizer.OptimizeNow(ctx)

// 获取 Collection 统计信息
stats, err := optimizer.GetCollectionStats(ctx)
fmt.Printf("向量数量: %d\n", stats.VectorsCount)
fmt.Printf("已索引向量: %d\n", stats.IndexedVectorsCount)
fmt.Printf("索引进度: %.2f%%\n", stats.IndexingProgress)
fmt.Printf("段数量: %d\n", stats.SegmentsCount)
```

#### 优化效果

- **Compact**：合并小段，减少段数量
- **Reindex**：重建索引，提高查询性能
- **Vacuum**：清理已删除的向量，释放空间

**建议优化频率**：

- 高频写入场景：每12小时
- 中频写入场景：每24小时
- 低频写入场景：每周一次

### 6. 租户级别的 Payload 索引

#### 配置说明

系统自动为 `tenant_id` 字段创建特殊索引（`is_tenant=true`），Qdrant 会针对租户字段进行优化：

```go
// 在 InitializeCollection 中自动创建
err := c.createFieldIndex(ctx, "tenant_id", "keyword", true) // is_tenant=true
```

**优势**：

- 更快的租户过滤速度
- 更好的数据分布
- 支持租户级别的分片策略

## 性能监控

### 获取 Collection 信息

```go
info, err := qdrantClient.GetCollectionInfo(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("状态: %s\n", info.Status)
fmt.Printf("向量数量: %d\n", info.VectorsCount)
fmt.Printf("已索引向量: %d\n", info.IndexedVectorsCount)
fmt.Printf("点数量: %d\n", info.PointsCount)
fmt.Printf("段数量: %d\n", info.SegmentsCount)
fmt.Printf("分片数量: %d\n", info.Config.ShardNumber)
fmt.Printf("副本数量: %d\n", info.Config.ReplicationFactor)
```

### 监控指标

建议监控以下指标：

1. **向量数量**：总向量数和增长趋势
2. **索引进度**：已索引向量 / 总向量
3. **段数量**：段数量过多会影响性能
4. **查询延迟**：P50、P95、P99延迟
5. **缓存命中率**：缓存命中次数 / 总查询次数

## 最佳实践

### 1. 批量操作

- 尽可能使用批量插入（建议批量大小：50-100）
- 避免在循环中单条插入

### 2. 缓存策略

- 为高频查询启用缓存
- 合理设置缓存过期时间
- 及时清理过期缓存

### 3. 异步处理

- 非关键路径的向量生成使用异步处理
- 合理设置队列大小，避免内存溢出
- 监控队列长度，及时调整工作协程数量

### 4. 定期优化

- 根据写入频率调整优化间隔
- 在低峰期执行优化操作
- 监控优化效果，调整优化策略

### 5. 配置调优

- 根据实际租户数量调整分片数量
- 根据查询准确性要求调整 HNSW 参数
- 定期评估配置效果，持续优化

## 故障排查

### 问题1：查询速度慢

**可能原因**：

- 段数量过多
- 索引未完成
- HNSW 参数不合理

**解决方案**：

1. 执行 Collection 优化
2. 等待索引完成
3. 调整 HNSW 参数

### 问题2：内存占用高

**可能原因**：

- 向量数量过多
- HNSW M 参数过大
- 缓存占用过多

**解决方案**：

1. 清理过期向量
2. 降低 HNSW M 参数
3. 调整缓存策略

### 问题3：插入速度慢

**可能原因**：

- 单条插入
- 索引构建中
- 分片数量不足

**解决方案**：

1. 使用批量插入
2. 等待索引完成
3. 增加分片数量

## 配置示例

### 开发环境

```go
config := &storage.QdrantConfig{
    Host:                   "localhost",
    Port:                   6333,
    APIKey:                 "dev-api-key",
    ShardNumber:            2,
    ReplicationFactor:      1,
    HnswM:                  8,
    HnswEfConstruction:     50,
    EnableOptimization:     false,
}
```

### 生产环境

```go
config := &storage.QdrantConfig{
    Endpoint:               "https://xxx.cloud.qdrant.io",
    APIKey:                 "prod-api-key",
    ClusterID:              "prod-cluster",
    ShardNumber:            8,
    ReplicationFactor:      2,
    HnswM:                  16,
    HnswEfConstruction:     100,
    EnableOptimization:     true,
    OptimizationInterval:   24,
}
```

## 参考资料

- [Qdrant 官方文档](https://qdrant.tech/documentation/)
- [HNSW 算法原理](https://arxiv.org/abs/1603.09320)
- [多租户架构最佳实践](https://qdrant.tech/documentation/guides/multiple-partitions/)
