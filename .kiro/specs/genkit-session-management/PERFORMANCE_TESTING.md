# 性能测试文档

## 概述

本文档描述了Genkit会话管理模块的性能测试策略、测试用例和性能指标。

## 测试目标

### 核心性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 上下文构建延迟 | < 100ms | 构建完整上下文的时间 |
| 向量检索延迟 | < 50ms | Top-5向量相似度搜索 |
| 缓存读取延迟 | < 1ms | 从缓存读取数据 |
| 缓存写入延迟 | < 2ms | 向缓存写入数据 |
| 并发处理能力 | > 1000 req/s | 单实例并发请求处理 |
| 内存使用 | < 500MB | 正常负载下的内存占用 |

### 性能要求

1. **响应时间**
   - P50 < 50ms
   - P95 < 200ms
   - P99 < 500ms

2. **吞吐量**
   - 单实例 > 1000 QPS
   - 向量检索 > 500 QPS

3. **资源使用**
   - CPU使用率 < 70%
   - 内存使用 < 500MB
   - 数据库连接 < 50

## 测试文件结构

```
internal/
├── service/
│   └── performance_benchmark_test.go      # 服务层性能测试
├── repository/
│   └── vector_performance_benchmark_test.go  # 向量检索性能测试
└── storage/
    └── cache_performance_benchmark_test.go   # 缓存性能测试

scripts/
└── run_performance_tests.sh               # 性能测试运行脚本
```

## 测试用例

### 1. 服务层性能测试

#### 1.1 上下文构建性能

**测试用例：**

- `BenchmarkContextService_BuildContext` - 基础上下文构建
- `BenchmarkContextService_BuildContext_WithLongTerm` - 包含长期记忆
- `BenchmarkContextService_BuildContext_WithSummary` - 包含摘要
- `BenchmarkContextService_BuildContext_Parallel` - 并发构建

**测试数据：**

- 10条短期消息
- 5条长期记忆
- 1个对话摘要
- 最大Token: 4000

**性能目标：**

- 基础构建: < 10ms
- 包含长期记忆: < 50ms
- 包含摘要: < 30ms
- 并发性能: > 1000 ops/s

#### 1.2 上下文优化性能

**测试用例：**

- `BenchmarkContextService_OptimizeContext` - 上下文优化
- `BenchmarkOptimizeContext_Strategy_Balanced` - 平衡策略
- `BenchmarkOptimizeContext_Strategy_Aggressive` - 激进策略
- `BenchmarkOptimizeContext_Strategy_Conservative` - 保守策略

**测试数据：**

- 50条消息
- 20条记忆
- Token超限场景（8000 -> 3000）

**性能目标：**

- 优化时间: < 20ms
- 内存分配: < 100KB

#### 1.3 记忆服务性能

**测试用例：**

- `BenchmarkMemoryService_StoreMemory` - 存储记忆
- `BenchmarkMemoryService_SearchMemories` - 搜索记忆
- `BenchmarkMemoryService_SearchMemories_CrossSessions` - 跨会话搜索
- `BenchmarkMemoryService_CleanupMemories` - 清理记忆

**性能目标：**

- 存储: < 20ms
- 搜索: < 50ms
- 跨会话搜索: < 100ms
- 清理: < 50ms

### 2. 向量检索性能测试

#### 2.1 向量相似度搜索

**测试用例：**

- `BenchmarkMemoryRepository_SearchByVector` - Top-5检索
- `BenchmarkMemoryRepository_SearchByVector_TopK10` - Top-10检索
- `BenchmarkMemoryRepository_SearchByVector_TopK50` - Top-50检索
- `BenchmarkMemoryRepository_SearchByVectorCrossSessions` - 跨会话检索

**测试数据规模：**

- 小规模: 100条记忆
- 中规模: 200条记忆
- 大规模: 500条记忆
- 跨会话: 10个会话，每个20条

**性能目标：**

- Top-5: < 10ms
- Top-10: < 20ms
- Top-50: < 50ms
- 跨会话: < 100ms

#### 2.2 CRUD操作性能

**测试用例：**

- `BenchmarkMemoryRepository_Create` - 创建记忆
- `BenchmarkMemoryRepository_GetByID` - 根据ID查询
- `BenchmarkMemoryRepository_UpdateAccessStats` - 更新统计
- `BenchmarkMemoryRepository_DeleteByStrategy` - 批量删除

**性能目标：**

- 创建: < 5ms
- 查询: < 2ms
- 更新: < 3ms
- 删除: < 10ms

### 3. 缓存性能测试

#### 3.1 不同数据大小的缓存性能

**测试用例：**

- `BenchmarkCacheService_Set_Small` - 小数据写入（< 100B）
- `BenchmarkCacheService_Set_Medium` - 中等数据写入（~1KB）
- `BenchmarkCacheService_Set_Large` - 大数据写入（~10KB）
- `BenchmarkCacheService_Get_Small/Medium/Large` - 对应的读取测试

**性能目标：**

- 小数据读写: < 0.5ms
- 中等数据读写: < 1ms
- 大数据读写: < 5ms

#### 3.2 并发缓存性能

**测试用例：**

- `BenchmarkCacheService_Parallel_Set` - 并发写入
- `BenchmarkCacheService_Parallel_Get` - 并发读取
- `BenchmarkCacheService_Parallel_Mixed` - 混合读写

**性能目标：**

- 并发写入: > 10000 ops/s
- 并发读取: > 50000 ops/s
- 混合操作: > 20000 ops/s

#### 3.3 缓存命中率测试

**测试用例：**

- `BenchmarkCacheService_HitRate` - 50%命中率场景
- `BenchmarkContextService_CacheHitRate` - 实际使用场景

**性能目标：**

- 缓存命中: < 0.5ms
- 缓存未命中: < 50ms（含数据库查询）

### 4. 大规模数据性能测试

#### 4.1 大规模上下文

**测试用例：**

- `BenchmarkContextService_LargeContext` - 100条消息
- `BenchmarkMemoryService_LargeVectorSearch` - 50条检索结果
- `BenchmarkContextService_TokenOptimization` - 大规模Token优化

**测试数据：**

- 100条消息
- 50条记忆
- Token: 15000 -> 4000

**性能目标：**

- 大规模构建: < 200ms
- 大规模检索: < 100ms
- Token优化: < 50ms

## 运行性能测试

### 方式1：使用脚本运行

```bash
# 运行所有性能测试
chmod +x scripts/run_performance_tests.sh
./scripts/run_performance_tests.sh
```

### 方式2：手动运行

```bash
# 运行服务层性能测试
go test -bench=. -benchmem -benchtime=3s ./internal/service

# 运行向量检索性能测试
go test -bench=. -benchmem -benchtime=3s ./internal/repository

# 运行缓存性能测试
go test -bench=. -benchmem -benchtime=3s ./internal/storage

# 运行特定测试
go test -bench=BenchmarkContextService_BuildContext -benchmem ./internal/service

# 生成CPU性能分析
go test -bench=. -cpuprofile=cpu.prof ./internal/service
go tool pprof cpu.prof

# 生成内存性能分析
go test -bench=. -memprofile=mem.prof ./internal/service
go tool pprof mem.prof
```

### 方式3：对比测试

```bash
# 运行基准测试并保存结果
go test -bench=. -benchmem ./internal/service > old.txt

# 修改代码后再次运行
go test -bench=. -benchmem ./internal/service > new.txt

# 使用benchcmp对比（需要安装）
go install golang.org/x/tools/cmd/benchcmp@latest
benchcmp old.txt new.txt
```

## 性能分析工具

### 1. pprof性能分析

```bash
# CPU性能分析
go test -bench=BenchmarkContextService_BuildContext \
  -cpuprofile=cpu.prof ./internal/service
go tool pprof -http=:8080 cpu.prof

# 内存性能分析
go test -bench=BenchmarkContextService_BuildContext \
  -memprofile=mem.prof ./internal/service
go tool pprof -http=:8080 mem.prof

# 阻塞分析
go test -bench=BenchmarkContextService_BuildContext \
  -blockprofile=block.prof ./internal/service
go tool pprof -http=:8080 block.prof
```

### 2. trace追踪分析

```bash
# 生成trace文件
go test -bench=BenchmarkContextService_BuildContext \
  -trace=trace.out ./internal/service

# 查看trace
go tool trace trace.out
```

### 3. 内存逃逸分析

```bash
# 查看内存逃逸
go test -gcflags="-m -m" ./internal/service 2>&1 | grep "escapes to heap"
```

## 性能优化建议

### 1. 上下文构建优化

- **缓存策略**：缓存构建结果，TTL 5分钟
- **并行处理**：并行获取消息、记忆和摘要
- **延迟加载**：按需加载长期记忆和摘要
- **批量操作**：批量生成向量，减少API调用

### 2. 向量检索优化

- **索引优化**：使用IVFFlat或HNSW索引
- **查询优化**：设置合理的相似度阈值
- **结果缓存**：缓存热门查询结果
- **分区策略**：按租户或时间分区

### 3. 缓存优化

- **多级缓存**：本地缓存 + Redis
- **预热策略**：启动时预热热门数据
- **淘汰策略**：LRU淘汰不常用数据
- **压缩存储**：大数据使用压缩

### 4. 数据库优化

- **连接池**：合理配置连接池大小
- **索引优化**：为常用查询添加索引
- **批量操作**：使用批量插入和更新
- **读写分离**：读操作使用只读副本

## 性能监控

### 关键指标监控

```go
// Prometheus指标
- genkit_context_build_duration_seconds
- genkit_vector_search_duration_seconds
- genkit_cache_hit_rate
- genkit_token_usage_total
- genkit_memory_usage_bytes
```

### 告警规则

```yaml
# 响应时间告警
- alert: HighContextBuildLatency
  expr: histogram_quantile(0.95, genkit_context_build_duration_seconds) > 0.2
  for: 5m
  annotations:
    summary: "上下文构建延迟过高"

# 缓存命中率告警
- alert: LowCacheHitRate
  expr: rate(genkit_cache_hits_total[5m]) / rate(genkit_cache_requests_total[5m]) < 0.7
  for: 10m
  annotations:
    summary: "缓存命中率过低"
```

## 性能测试报告

### 报告内容

1. **测试环境**
   - 硬件配置
   - 软件版本
   - 测试时间

2. **测试结果**
   - 各项基准测试结果
   - 性能指标对比
   - 资源使用情况

3. **性能分析**
   - 性能瓶颈识别
   - 热点函数分析
   - 内存分配分析

4. **优化建议**
   - 具体优化方案
   - 预期性能提升
   - 实施优先级

### 报告示例

```
========================================
性能测试报告
========================================

测试时间: 2024-01-15 10:00:00
Go版本: go1.21.0
硬件: 8 Core CPU, 16GB RAM

关键性能指标:
1. 上下文构建: 8.5ms (目标: <10ms) ✓
2. 向量检索: 35ms (目标: <50ms) ✓
3. 缓存读取: 0.8ms (目标: <1ms) ✓
4. 并发处理: 1200 QPS (目标: >1000 QPS) ✓

内存使用:
- 平均: 320MB
- 峰值: 450MB
- 分配次数: 1.2M/s

性能瓶颈:
1. 向量生成API调用 (30%)
2. 数据库查询 (25%)
3. JSON序列化 (15%)

优化建议:
1. 实施向量批量生成
2. 增加数据库连接池
3. 使用更快的序列化库
```

## 持续性能监控

### 1. 每次发布前

- 运行完整性能测试套件
- 对比上一版本性能
- 确保无性能退化

### 2. 定期性能测试

- 每周运行性能测试
- 记录性能趋势
- 及时发现性能问题

### 3. 生产环境监控

- 实时监控关键指标
- 设置性能告警
- 定期性能审查

## 总结

性能测试是确保系统质量的重要环节。通过系统的性能测试和持续优化，我们可以：

1. **验证性能目标**：确保系统满足性能要求
2. **识别瓶颈**：找出性能瓶颈并优化
3. **防止退化**：及时发现性能退化
4. **指导优化**：为性能优化提供数据支持

定期运行性能测试，持续监控和优化，确保系统始终保持高性能。
