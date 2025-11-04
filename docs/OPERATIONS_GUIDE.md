# Genkit 会话管理模块运维指南

## 概述

本文档提供 Genkit 会话管理模块的日常运维指南，包括监控、故障排查、性能优化、备份恢复等操作。

## 目录

- [日常监控](#日常监控)
- [性能优化](#性能优化)
- [故障排查](#故障排查)
- [备份和恢复](#备份和恢复)
- [扩容和缩容](#扩容和缩容)
- [安全运维](#安全运维)
- [日志管理](#日志管理)
- [告警配置](#告警配置)

## 日常监控

### 1. 关键指标

#### 服务健康指标

| 指标 | 说明 | 正常范围 | 告警阈值 |
|-----|------|---------|---------|
| 服务可用性 | 服务是否正常运行 | 100% | < 99.9% |
| 响应时间 | API 平均响应时间 | < 200ms | > 1000ms |
| 错误率 | 请求错误比例 | < 0.1% | > 1% |
| QPS | 每秒请求数 | - | - |

#### 资源使用指标

| 指标 | 说明 | 正常范围 | 告警阈值 |
|-----|------|---------|---------|
| CPU 使用率 | Pod CPU 使用率 | < 70% | > 85% |
| 内存使用率 | Pod 内存使用率 | < 80% | > 90% |
| 磁盘使用率 | 存储使用率 | < 70% | > 85% |
| 网络带宽 | 网络流量 | - | - |

#### 业务指标

| 指标 | 说明 | 正常范围 | 告警阈值 |
|-----|------|---------|---------|
| Token 使用量 | 每日 Token 消耗 | - | 超过配额 90% |
| 会话数量 | 活跃会话数 | - | - |
| 记忆存储量 | 长期记忆数量 | - | - |
| 缓存命中率 | Redis 缓存命中率 | > 80% | < 60% |

### 2. 监控工具

#### Prometheus 查询

```promql
# 服务可用性
up{job="genkit-service"}

# 请求速率
rate(genkit_flow_executions_total[5m])

# 错误率
rate(genkit_flow_executions_total{status="error"}[5m]) / 
rate(genkit_flow_executions_total[5m])

# 响应时间 P95
histogram_quantile(0.95, 
  rate(genkit_flow_duration_seconds_bucket[5m]))

# CPU 使用率
rate(container_cpu_usage_seconds_total{
  pod=~"genkit-service-.*"
}[5m])

# 内存使用
container_memory_usage_bytes{
  pod=~"genkit-service-.*"
}

# Token 使用量
sum(rate(genkit_token_usage_total[1h])) by (tenant_id)

# 缓存命中率
rate(genkit_cache_hits_total[5m]) / 
(rate(genkit_cache_hits_total[5m]) + 
 rate(genkit_cache_misses_total[5m]))
```

#### Grafana 仪表板

创建 Grafana 仪表板监控关键指标：

```json
{
  "dashboard": {
    "title": "Genkit Service Dashboard",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(genkit_flow_executions_total[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(genkit_flow_executions_total{status=\"error\"}[5m])"
          }
        ]
      },
      {
        "title": "Response Time P95",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(genkit_flow_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "CPU Usage",
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{pod=~\"genkit-service-.*\"}[5m])"
          }
        ]
      }
    ]
  }
}
```

### 3. 健康检查

#### 服务健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 详细健康检查
curl http://localhost:8080/health/detailed

# 响应示例
{
  "status": "healthy",
  "timestamp": "2024-01-01T10:00:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "latency": "5ms"
    },
    "redis": {
      "status": "healthy",
      "latency": "2ms"
    },
    "ai_service": {
      "status": "healthy",
      "latency": "100ms"
    }
  }
}
```

#### 数据库健康检查

```bash
# 检查数据库连接
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c "SELECT 1"

# 检查数据库大小
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "SELECT pg_size_pretty(pg_database_size('genkit_prod'))"

# 检查活跃连接数
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "SELECT count(*) FROM pg_stat_activity"

# 检查慢查询
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "SELECT query, calls, total_time, mean_time 
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10"
```

#### Redis 健康检查

```bash
# 检查 Redis 连接
kubectl exec -n genkit redis-0 -- redis-cli ping

# 检查内存使用
kubectl exec -n genkit redis-0 -- redis-cli info memory

# 检查键数量
kubectl exec -n genkit redis-0 -- redis-cli dbsize

# 检查缓存命中率
kubectl exec -n genkit redis-0 -- redis-cli info stats | grep keyspace
```

## 性能优化

### 1. 数据库优化

#### 索引优化

```sql
-- 查看缺失的索引
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats
WHERE schemaname = 'public'
  AND n_distinct > 100
  AND correlation < 0.1;

-- 创建复合索引
CREATE INDEX CONCURRENTLY idx_memories_session_similarity 
ON conversation_memories(session_id, (1 - (embedding <=> '[0,0,...]'::vector)));

-- 创建部分索引
CREATE INDEX CONCURRENTLY idx_memories_active 
ON conversation_memories(session_id, created_at) 
WHERE is_deleted = false;

-- 重建索引
REINDEX INDEX CONCURRENTLY idx_memories_session_similarity;
```

#### 查询优化

```sql
-- 分析查询计划
EXPLAIN ANALYZE
SELECT * FROM conversation_memories
WHERE session_id = 'xxx'
  AND is_deleted = false
ORDER BY created_at DESC
LIMIT 10;

-- 更新统计信息
ANALYZE conversation_memories;

-- 清理死元组
VACUUM ANALYZE conversation_memories;
```

#### 连接池优化

```go
// 配置连接池
db.SetMaxOpenConns(50)      // 最大打开连接数
db.SetMaxIdleConns(10)      // 最大空闲连接数
db.SetConnMaxLifetime(1 * time.Hour)  // 连接最大生命周期
db.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大生命周期
```

### 2. 缓存优化

#### Redis 配置优化

```bash
# 设置最大内存
kubectl exec -n genkit redis-0 -- \
  redis-cli config set maxmemory 2gb

# 设置淘汰策略
kubectl exec -n genkit redis-0 -- \
  redis-cli config set maxmemory-policy allkeys-lru

# 启用持久化
kubectl exec -n genkit redis-0 -- \
  redis-cli config set save "900 1 300 10 60 10000"
```

#### 缓存策略优化

```go
// 多级缓存
type MultiLevelCache struct {
    local  *LocalCache  // 本地缓存（内存）
    remote *RedisCache  // 远程缓存（Redis）
}

func (c *MultiLevelCache) Get(key string) (interface{}, error) {
    // 1. 先查本地缓存
    if value, ok := c.local.Get(key); ok {
        return value, nil
    }
    
    // 2. 查 Redis
    value, err := c.remote.Get(key)
    if err == nil {
        // 写入本地缓存
        c.local.Set(key, value, 5*time.Minute)
        return value, nil
    }
    
    return nil, ErrCacheNotFound
}

// 缓存预热
func (c *CacheWarmer) WarmupActiveSessions() error {
    sessions, err := c.sessionRepo.GetRecentActive(ctx, 100)
    if err != nil {
        return err
    }
    
    for _, session := range sessions {
        // 预加载上下文
        context, _ := c.contextService.BuildContext(ctx, session.ID)
        c.cache.Set(fmt.Sprintf("context:%s", session.ID), context, 10*time.Minute)
    }
    
    return nil
}
```

### 3. 向量检索优化

#### 索引优化

```sql
-- 创建 IVFFlat 索引
CREATE INDEX CONCURRENTLY idx_memories_embedding_ivfflat
ON conversation_memories 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- 或使用 HNSW 索引（更快但占用更多内存）
CREATE INDEX CONCURRENTLY idx_memories_embedding_hnsw
ON conversation_memories 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- 设置查询参数
SET ivfflat.probes = 10;  -- IVFFlat
SET hnsw.ef_search = 40;  -- HNSW
```

#### 批量向量生成

```go
// 批量生成向量
func (s *VectorService) GenerateBatchEmbeddings(
    ctx context.Context,
    texts []string,
) ([]pgvector.Vector, error) {
    // 批量调用 API
    embeddings, err := s.client.GenerateEmbeddings(ctx, texts)
    if err != nil {
        return nil, err
    }
    
    // 转换为 pgvector.Vector
    vectors := make([]pgvector.Vector, len(embeddings))
    for i, emb := range embeddings {
        vectors[i] = pgvector.NewVector(emb)
    }
    
    return vectors, nil
}
```

### 4. 应用层优化

#### 并发控制

```go
// 使用 worker pool 控制并发
type WorkerPool struct {
    workers   int
    taskQueue chan Task
}

func NewWorkerPool(workers int) *WorkerPool {
    pool := &WorkerPool{
        workers:   workers,
        taskQueue: make(chan Task, workers*2),
    }
    
    for i := 0; i < workers; i++ {
        go pool.worker()
    }
    
    return pool
}

func (p *WorkerPool) worker() {
    for task := range p.taskQueue {
        task.Execute()
    }
}
```

#### 请求合并

```go
// 合并相同的请求
type RequestBatcher struct {
    mu       sync.Mutex
    batches  map[string]*Batch
    interval time.Duration
}

func (b *RequestBatcher) Add(key string, req Request) <-chan Response {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    batch, ok := b.batches[key]
    if !ok {
        batch = NewBatch()
        b.batches[key] = batch
        
        // 延迟执行
        time.AfterFunc(b.interval, func() {
            b.executeBatch(key, batch)
        })
    }
    
    return batch.Add(req)
}
```

## 故障排查

### 1. 服务不可用

#### 症状

- 健康检查失败
- API 请求超时或返回 5xx 错误
- Pod 频繁重启

#### 排查步骤

```bash
# 1. 检查 Pod 状态
kubectl get pods -n genkit
kubectl describe pod <pod-name> -n genkit

# 2. 查看日志
kubectl logs <pod-name> -n genkit --tail=100
kubectl logs <pod-name> -n genkit --previous  # 查看上一个容器的日志

# 3. 检查资源使用
kubectl top pods -n genkit

# 4. 检查事件
kubectl get events -n genkit --sort-by='.lastTimestamp'

# 5. 进入容器调试
kubectl exec -it <pod-name> -n genkit -- /bin/sh
```

#### 常见原因和解决方案

**原因1：数据库连接失败**

```bash
# 检查数据库服务
kubectl get svc postgres-service -n genkit

# 测试连接
kubectl run -it --rm debug --image=postgres:14 --restart=Never -n genkit -- \
  psql -h postgres-service -U genkit_user -d genkit_prod

# 解决方案：
# - 检查数据库密码是否正确
# - 检查网络策略
# - 增加数据库连接数
```

**原因2：内存不足**

```bash
# 查看内存使用
kubectl top pod <pod-name> -n genkit

# 解决方案：
# - 增加内存限制
kubectl set resources deployment genkit-service \
  --limits=memory=1Gi -n genkit

# - 或增加副本数分散负载
kubectl scale deployment genkit-service --replicas=5 -n genkit
```

**原因3：配置错误**

```bash
# 检查配置
kubectl get configmap genkit-config -n genkit -o yaml
kubectl get secret genkit-secret -n genkit -o yaml

# 解决方案：
# - 更新配置
kubectl edit configmap genkit-config -n genkit

# - 重启 Pod 使配置生效
kubectl rollout restart deployment genkit-service -n genkit
```

### 2. 性能下降

#### 症状

- API 响应时间增加
- 吞吐量下降
- CPU 或内存使用率高

#### 排查步骤

```bash
# 1. 查看性能指标
kubectl top pods -n genkit

# 2. 分析慢查询
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "SELECT query, calls, total_time, mean_time 
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10"

# 3. 检查缓存命中率
kubectl exec -n genkit redis-0 -- \
  redis-cli info stats | grep keyspace_hits

# 4. 分析日志
kubectl logs deployment/genkit-service -n genkit | \
  grep "duration_ms" | \
  awk '{print $NF}' | \
  sort -n | \
  tail -20
```

#### 优化措施

```bash
# 1. 增加副本数
kubectl scale deployment genkit-service --replicas=5 -n genkit

# 2. 启用 HPA
kubectl autoscale deployment genkit-service \
  --cpu-percent=70 \
  --min=3 \
  --max=10 \
  -n genkit

# 3. 优化数据库
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c "VACUUM ANALYZE"

# 4. 清理缓存
kubectl exec -n genkit redis-0 -- redis-cli FLUSHDB
```

### 3. 数据不一致

#### 症状

- 查询结果不符合预期
- 数据丢失或重复
- 缓存和数据库数据不一致

#### 排查步骤

```bash
# 1. 检查数据库数据
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "SELECT count(*) FROM conversation_memories WHERE is_deleted = false"

# 2. 检查缓存数据
kubectl exec -n genkit redis-0 -- redis-cli keys "context:*" | wc -l

# 3. 对比数据
# 编写脚本对比数据库和缓存的数据一致性
```

#### 解决方案

```bash
# 1. 清理缓存
kubectl exec -n genkit redis-0 -- redis-cli FLUSHDB

# 2. 重建索引
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -c \
  "REINDEX DATABASE genkit_prod"

# 3. 数据修复
# 运行数据修复脚本
kubectl exec -n genkit postgres-0 -- \
  psql -U genkit_user -d genkit_prod -f /scripts/fix_data.sql
```
