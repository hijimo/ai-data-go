# Genkit 会话管理监控指南

本文档描述了 Genkit 会话管理模块的监控指标和使用方法。

## 概述

Genkit 监控模块提供了全面的性能指标收集和监控功能，基于 Prometheus 实现，支持以下监控场景：

- Flow 执行监控
- Token 使用监控
- 缓存命中率监控
- 向量检索性能监控
- AI 服务调用监控
- 会话健康度监控

## 监控指标

### 1. Flow 执行指标

#### genkit_flow_executions_total

Flow 执行次数计数器

**标签**：

- `flow_name`: Flow 名称（如 contextBuildFlow, chatGenerateFlow）
- `status`: 执行状态（success, error）
- `tenant_id`: 租户 ID

**示例**：

```promql
# 查询每个 Flow 的总执行次数
sum(genkit_flow_executions_total) by (flow_name)

# 查询错误率
rate(genkit_flow_executions_total{status="error"}[5m]) / rate(genkit_flow_executions_total[5m])
```

#### genkit_flow_duration_seconds

Flow 执行时间直方图

**标签**：

- `flow_name`: Flow 名称
- `tenant_id`: 租户 ID

**分桶**：0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0 秒

**示例**：

```promql
# 查询 P95 执行时间
histogram_quantile(0.95, rate(genkit_flow_duration_seconds_bucket[5m]))

# 查询平均执行时间
rate(genkit_flow_duration_seconds_sum[5m]) / rate(genkit_flow_duration_seconds_count[5m])
```

#### genkit_flow_errors_total

Flow 错误次数计数器

**标签**：

- `flow_name`: Flow 名称
- `error_type`: 错误类型（timeout, permission, validation, not_found, ai_service, database, cache, vector_service, quota_exceeded, unknown）
- `tenant_id`: 租户 ID

**示例**：

```promql
# 查询各类错误的分布
sum(genkit_flow_errors_total) by (error_type)

# 查询特定租户的错误率
rate(genkit_flow_errors_total{tenant_id="tenant-123"}[5m])
```

### 2. Token 使用指标

#### genkit_token_usage_total

Token 使用量计数器

**标签**：

- `tenant_id`: 租户 ID
- `token_type`: Token 类型（prompt, completion, total）
- `flow_name`: Flow 名称

**示例**：

```promql
# 查询租户的总 Token 使用量
sum(genkit_token_usage_total{tenant_id="tenant-123"})

# 查询每个 Flow 的 Token 使用量
sum(genkit_token_usage_total) by (flow_name, token_type)
```

#### genkit_session_token_usage

会话 Token 使用量仪表盘

**标签**：

- `session_id`: 会话 ID
- `tenant_id`: 租户 ID

**示例**：

```promql
# 查询当前活跃会话的 Token 使用量
genkit_session_token_usage

# 查询 Token 使用量最高的会话
topk(10, genkit_session_token_usage)
```

### 3. 缓存指标

#### genkit_cache_hits_total / genkit_cache_misses_total

缓存命中和未命中计数器

**标签**：

- `cache_type`: 缓存类型（context, vector, summary, sessions, tokens, quota）
- `tenant_id`: 租户 ID

**示例**：

```promql
# 计算缓存命中率
sum(rate(genkit_cache_hits_total[5m])) / (sum(rate(genkit_cache_hits_total[5m])) + sum(rate(genkit_cache_misses_total[5m])))

# 查询各类缓存的命中率
sum(rate(genkit_cache_hits_total[5m])) by (cache_type) / (sum(rate(genkit_cache_hits_total[5m])) by (cache_type) + sum(rate(genkit_cache_misses_total[5m])) by (cache_type))
```

### 4. 上下文构建指标

#### genkit_context_build_tokens

上下文构建的 Token 数量直方图

**标签**：

- `session_id`: 会话 ID
- `tenant_id`: 租户 ID

**分桶**：100, 500, 1000, 2000, 4000, 8000, 16000

**示例**：

```promql
# 查询平均上下文大小
avg(genkit_context_build_tokens)

# 查询 P95 上下文大小
histogram_quantile(0.95, rate(genkit_context_build_tokens_bucket[5m]))
```

#### genkit_context_quality_score

上下文质量评分直方图

**标签**：

- `session_id`: 会话 ID
- `tenant_id`: 租户 ID

**分桶**：0.1 到 1.0，步长 0.1

**示例**：

```promql
# 查询平均质量评分
avg(genkit_context_quality_score)

# 查询质量评分分布
histogram_quantile(0.5, rate(genkit_context_quality_score_bucket[5m]))
```

### 5. 向量检索指标

#### genkit_vector_search_duration_seconds

向量检索执行时间直方图

**标签**：

- `tenant_id`: 租户 ID

**分桶**：0.01, 0.05, 0.1, 0.2, 0.5, 1.0 秒

#### genkit_vector_search_results

向量检索结果数量直方图

**标签**：

- `tenant_id`: 租户 ID

**分桶**：0, 1, 5, 10, 20, 50

### 6. AI 服务指标

#### genkit_ai_service_calls_total

AI 服务调用次数计数器

**标签**：

- `provider`: 提供商（google, openai 等）
- `model`: 模型名称（gemini-1.5-flash 等）
- `status`: 调用状态（success, error）
- `tenant_id`: 租户 ID

#### genkit_ai_service_duration_seconds

AI 服务调用时间直方图

**标签**：

- `provider`: 提供商
- `model`: 模型名称
- `tenant_id`: 租户 ID

**分桶**：0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0 秒

### 7. 会话健康度指标

#### genkit_session_health_score

会话健康度评分仪表盘

**标签**：

- `session_id`: 会话 ID
- `tenant_id`: 租户 ID

**值范围**：0.0 到 1.0

#### genkit_active_sessions

活跃会话数仪表盘

**标签**：

- `tenant_id`: 租户 ID

### 8. 记忆管理指标

#### genkit_memory_stores_total

记忆存储次数计数器

**标签**：

- `memory_type`: 记忆类型（long_term, short_term）
- `tenant_id`: 租户 ID

#### genkit_memory_cleanups_total

记忆清理次数计数器

**标签**：

- `strategy`: 清理策略（expired, low_quality, unused）
- `mode`: 清理模式（soft, hard）
- `tenant_id`: 租户 ID

### 9. 系统资源指标

#### genkit_database_connections

数据库连接数仪表盘

#### genkit_redis_connections

Redis 连接数仪表盘

## 使用方法

### 1. 初始化监控

```go
import (
    "genkit-ai-service/internal/monitoring"
    "genkit-ai-service/internal/genkit/middleware"
)

// 创建 Genkit 指标收集器
genkitMetrics := monitoring.NewGenkitMetrics()

// 创建 Flow 监控中间件
flowMonitor := middleware.NewFlowMonitor(genkitMetrics)
```

### 2. 监控 Flow 执行

```go
// 方式 1：简单监控
monitoredFlow := flowMonitor.MonitorFlow("contextBuildFlow", func(ctx context.Context) error {
    // Flow 逻辑
    return nil
})

// 执行监控的 Flow
err := monitoredFlow(ctx)

// 方式 2：带额外指标记录
monitoredFlow := flowMonitor.MonitorFlowWithMetrics(
    "contextBuildFlow",
    func(ctx context.Context) (interface{}, error) {
        // Flow 逻辑
        result := buildContext(ctx)
        return result, nil
    },
    func(ctx context.Context, result interface{}, err error) {
        // 记录额外指标
        if err == nil {
            // 记录业务指标
        }
    },
)
```

### 3. 记录 Token 使用

```go
// 记录 Token 使用量
genkitMetrics.RecordTokenUsage(
    tenantID,
    "total",
    "chatGenerateFlow",
    1500,
)

// 更新会话 Token 使用量
genkitMetrics.UpdateSessionTokenUsage(
    sessionID,
    tenantID,
    2000,
)
```

### 4. 监控缓存操作

```go
import "genkit-ai-service/internal/service"

// 包装缓存服务以自动记录指标
cache := service.NewCacheMetricsWrapper(originalCache, genkitMetrics)

// 使用包装后的缓存服务
err := cache.Get(ctx, key, &dest)
// 自动记录缓存命中或未命中
```

### 5. 记录上下文构建指标

```go
// 记录上下文构建
genkitMetrics.RecordContextBuild(
    sessionID,
    tenantID,
    totalTokens,
    qualityScore,
)
```

### 6. 记录向量检索指标

```go
startTime := time.Now()

// 执行向量检索
results, err := vectorSearch(ctx, query)

// 记录指标
duration := time.Since(startTime)
genkitMetrics.RecordVectorSearch(
    tenantID,
    duration,
    len(results),
)
```

### 7. 记录 AI 服务调用

```go
startTime := time.Now()

// 调用 AI 服务
response, err := aiService.Generate(ctx, prompt)

// 记录指标
duration := time.Since(startTime)
status := "success"
if err != nil {
    status = "error"
}

genkitMetrics.RecordAIServiceCall(
    "google",
    "gemini-1.5-flash",
    status,
    tenantID,
    duration,
)
```

### 8. 更新会话健康度

```go
// 计算会话健康度
healthScore := calculateSessionHealth(session)

// 更新指标
genkitMetrics.UpdateSessionHealth(
    sessionID,
    tenantID,
    healthScore,
)
```

## Prometheus 配置

### 1. 暴露指标端点

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

// 注册 Prometheus 处理器
http.Handle("/metrics", promhttp.Handler())

// 启动 HTTP 服务器
go func() {
    log.Fatal(http.ListenAndServe(":9090", nil))
}()
```

### 2. Prometheus 配置文件

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'genkit-service'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
```

## Grafana 仪表板

### 推荐面板

#### 1. Flow 性能面板

```promql
# Flow 执行次数
sum(rate(genkit_flow_executions_total[5m])) by (flow_name)

# Flow 错误率
sum(rate(genkit_flow_executions_total{status="error"}[5m])) by (flow_name) / sum(rate(genkit_flow_executions_total[5m])) by (flow_name)

# Flow P95 执行时间
histogram_quantile(0.95, sum(rate(genkit_flow_duration_seconds_bucket[5m])) by (flow_name, le))
```

#### 2. Token 使用面板

```promql
# 租户 Token 使用趋势
sum(rate(genkit_token_usage_total[5m])) by (tenant_id)

# Flow Token 使用分布
sum(genkit_token_usage_total) by (flow_name, token_type)
```

#### 3. 缓存性能面板

```promql
# 缓存命中率
sum(rate(genkit_cache_hits_total[5m])) / (sum(rate(genkit_cache_hits_total[5m])) + sum(rate(genkit_cache_misses_total[5m])))

# 各类缓存命中率
sum(rate(genkit_cache_hits_total[5m])) by (cache_type) / (sum(rate(genkit_cache_hits_total[5m])) by (cache_type) + sum(rate(genkit_cache_misses_total[5m])) by (cache_type))
```

#### 4. 系统健康面板

```promql
# 活跃会话数
genkit_active_sessions

# 数据库连接数
genkit_database_connections

# Redis 连接数
genkit_redis_connections
```

## 告警规则

### 推荐告警

```yaml
# alerts.yml
groups:
  - name: genkit_alerts
    interval: 30s
    rules:
      # Flow 错误率过高
      - alert: HighFlowErrorRate
        expr: |
          sum(rate(genkit_flow_executions_total{status="error"}[5m])) by (flow_name) 
          / sum(rate(genkit_flow_executions_total[5m])) by (flow_name) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Flow {{ $labels.flow_name }} 错误率过高"
          description: "Flow {{ $labels.flow_name }} 的错误率为 {{ $value | humanizePercentage }}"

      # Flow 执行时间过长
      - alert: SlowFlowExecution
        expr: |
          histogram_quantile(0.95, sum(rate(genkit_flow_duration_seconds_bucket[5m])) by (flow_name, le)) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Flow {{ $labels.flow_name }} 执行时间过长"
          description: "Flow {{ $labels.flow_name }} 的 P95 执行时间为 {{ $value }}s"

      # 缓存命中率过低
      - alert: LowCacheHitRate
        expr: |
          sum(rate(genkit_cache_hits_total[5m])) by (cache_type) 
          / (sum(rate(genkit_cache_hits_total[5m])) by (cache_type) + sum(rate(genkit_cache_misses_total[5m])) by (cache_type)) < 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "缓存 {{ $labels.cache_type }} 命中率过低"
          description: "缓存 {{ $labels.cache_type }} 的命中率为 {{ $value | humanizePercentage }}"

      # Token 使用量异常
      - alert: HighTokenUsage
        expr: |
          sum(rate(genkit_token_usage_total[5m])) by (tenant_id) > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "租户 {{ $labels.tenant_id }} Token 使用量异常"
          description: "租户 {{ $labels.tenant_id }} 的 Token 使用速率为 {{ $value }}/s"

      # AI 服务调用失败率过高
      - alert: HighAIServiceErrorRate
        expr: |
          sum(rate(genkit_ai_service_calls_total{status="error"}[5m])) by (provider, model) 
          / sum(rate(genkit_ai_service_calls_total[5m])) by (provider, model) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "AI 服务 {{ $labels.provider }}/{{ $labels.model }} 错误率过高"
          description: "错误率为 {{ $value | humanizePercentage }}"
```

## 最佳实践

### 1. 监控粒度

- 在所有关键 Flow 中添加监控
- 记录详细的错误类型以便分析
- 使用租户 ID 标签进行多租户隔离

### 2. 性能优化

- 监控代码的性能开销应小于 1%
- 使用异步方式记录非关键指标
- 定期清理过期的指标数据

### 3. 告警策略

- 设置合理的告警阈值
- 使用分级告警（warning, critical）
- 避免告警疲劳

### 4. 数据保留

- 短期数据（1-7天）：高精度
- 中期数据（7-30天）：中等精度
- 长期数据（30天+）：低精度或聚合数据

## 故障排查

### 指标未更新

1. 检查 Prometheus 是否正常抓取指标
2. 检查指标端点是否可访问
3. 检查是否正确调用了记录方法

### 指标值异常

1. 检查标签值是否正确
2. 检查时间范围是否合理
3. 检查是否有数据丢失

### 性能问题

1. 检查指标数量是否过多
2. 检查标签基数是否过高
3. 考虑使用采样或聚合

## 参考资料

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Prometheus 最佳实践](https://prometheus.io/docs/practices/)
