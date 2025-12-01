# Token 使用统计日志 - 快速参考

## 日志字段说明

每次 AI 调用都会记录以下字段：

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `tenantId` | string | 租户ID | `"test-tenant-id"` |
| `modelName` | string | 模型配置名称 | `"gemini-pro"` |
| `model` | string | 实际使用的模型 | `"gemini-1.5-pro"` |
| `duration` | string | 请求耗时 | `"143.584µs"` |
| `promptTokens` | int | 输入 token 数 | `10` |
| `completionTokens` | int | 输出 token 数 | `20` |
| `totalTokens` | int | 总 token 数 | `30` |

## 常用查询示例

### 1. 查询某个租户的 Token 使用情况

```bash
# 查询租户的所有请求
grep "tenantId.*tenant-123" app.log | jq '.fields | {tenantId, modelName, totalTokens}'

# 统计租户的总 token 使用量
grep "tenantId.*tenant-123" app.log | jq '.fields.totalTokens' | awk '{sum+=$1} END {print sum}'
```

### 2. 查询某个模型的使用情况

```bash
# 查询模型的所有请求
grep "modelName.*gpt-4" app.log | jq '.fields | {tenantId, totalTokens, duration}'

# 统计模型的平均 token 使用量
grep "modelName.*gpt-4" app.log | jq '.fields.totalTokens' | awk '{sum+=$1; count++} END {print sum/count}'
```

### 3. 查询高 Token 使用量的请求

```bash
# 查询 token 使用量超过 1000 的请求
grep "totalTokens" app.log | jq 'select(.fields.totalTokens > 1000)'

# 查询 token 使用量最高的 10 个请求
grep "totalTokens" app.log | jq '.fields | {tenantId, modelName, totalTokens}' | sort -k3 -n | tail -10
```

### 4. 查询慢请求

```bash
# 查询耗时超过 1 秒的请求
grep "duration" app.log | jq 'select(.fields.duration | tonumber > 1000000000)'

# 查询 token 使用量大且耗时长的请求
grep "totalTokens" app.log | jq 'select(.fields.totalTokens > 1000 and (.fields.duration | contains("s")))'
```

### 5. 成本估算

```bash
# 计算某个租户的每日成本（假设每 1000 tokens 成本为 $0.002）
grep "tenantId.*tenant-123" app.log | \
  jq '.fields.totalTokens' | \
  awk '{sum+=$1} END {print "Total tokens:", sum, "Cost: $" sum/1000*0.002}'

# 按模型统计成本
grep "modelName" app.log | \
  jq -r '[.fields.modelName, .fields.totalTokens] | @tsv' | \
  awk '{tokens[$1]+=$2} END {for(m in tokens) print m, tokens[m], "$" tokens[m]/1000*0.002}'
```

### 6. 使用趋势分析

```bash
# 按小时统计 token 使用量
grep "totalTokens" app.log | \
  jq -r '[.timestamp[0:13], .fields.totalTokens] | @tsv' | \
  awk '{hour[$1]+=$2} END {for(h in hour) print h, hour[h]}'

# 按租户统计每日 token 使用量
grep "totalTokens" app.log | \
  jq -r '[.timestamp[0:10], .fields.tenantId, .fields.totalTokens] | @tsv' | \
  awk '{key=$1" "$2; tokens[key]+=$3} END {for(k in tokens) print k, tokens[k]}'
```

## 日志示例

### 非流式调用

```json
{
  "timestamp": "2025-11-30T00:44:36Z",
  "level": "INFO",
  "message": "对话请求处理完成",
  "fields": {
    "sessionId": "67c91c7a-f2d2-4059-93f6-1246cc0ae457",
    "tenantId": "test-tenant-id",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "duration": "143.584µs",
    "promptTokens": 10,
    "completionTokens": 20,
    "totalTokens": 30
  }
}
```

### 流式调用

```json
{
  "timestamp": "2025-11-30T00:45:07Z",
  "level": "INFO",
  "message": "流式对话请求处理完成",
  "fields": {
    "sessionId": "c5d765ca-ff23-4091-bbfa-e9926e142cf6",
    "tenantId": "test-tenant-id",
    "modelName": "gpt-4",
    "model": "gpt-4-turbo",
    "duration": "2.5s",
    "ttfb": "500ms",
    "chunkCount": 25,
    "totalContentLen": 1250,
    "promptTokens": 150,
    "completionTokens": 800,
    "totalTokens": 950
  }
}
```

## 监控告警建议

### 1. Token 使用量告警

```yaml
# 单次请求 token 使用量过高
alert: HighTokenUsage
expr: totalTokens > 5000
message: "单次请求 token 使用量过高: {{totalTokens}}"

# 租户每小时 token 使用量过高
alert: TenantHighHourlyUsage
expr: sum(totalTokens) by (tenantId) > 100000
message: "租户 {{tenantId}} 每小时 token 使用量过高"
```

### 2. 性能告警

```yaml
# 请求耗时过长
alert: SlowRequest
expr: duration > 10s
message: "请求耗时过长: {{duration}}"

# 首字节时间过长（流式调用）
alert: SlowTTFB
expr: ttfb > 2s
message: "首字节时间过长: {{ttfb}}"
```

### 3. 成本告警

```yaml
# 每日成本超过预算
alert: DailyCostExceeded
expr: sum(totalTokens) * 0.002 / 1000 > 100
message: "每日成本超过预算: ${{value}}"

# 租户成本超过配额
alert: TenantCostExceeded
expr: sum(totalTokens) by (tenantId) * 0.002 / 1000 > 50
message: "租户 {{tenantId}} 成本超过配额"
```

## 集成建议

### ELK Stack

```json
{
  "index_patterns": ["app-logs-*"],
  "mappings": {
    "properties": {
      "fields.tenantId": { "type": "keyword" },
      "fields.modelName": { "type": "keyword" },
      "fields.model": { "type": "keyword" },
      "fields.promptTokens": { "type": "integer" },
      "fields.completionTokens": { "type": "integer" },
      "fields.totalTokens": { "type": "integer" },
      "fields.duration": { "type": "keyword" }
    }
  }
}
```

### Prometheus

```yaml
# 导出 token 使用量指标
- metric_name: ai_tokens_total
  type: counter
  help: "Total tokens used"
  labels:
    - tenant_id
    - model_name
  value: "{{.fields.totalTokens}}"

# 导出请求耗时指标
- metric_name: ai_request_duration_seconds
  type: histogram
  help: "Request duration in seconds"
  labels:
    - tenant_id
    - model_name
  value: "{{.fields.duration}}"
```

### Grafana Dashboard

```json
{
  "panels": [
    {
      "title": "Token Usage by Tenant",
      "targets": [
        {
          "expr": "sum(ai_tokens_total) by (tenant_id)"
        }
      ]
    },
    {
      "title": "Token Usage by Model",
      "targets": [
        {
          "expr": "sum(ai_tokens_total) by (model_name)"
        }
      ]
    },
    {
      "title": "Average Request Duration",
      "targets": [
        {
          "expr": "avg(ai_request_duration_seconds) by (model_name)"
        }
      ]
    }
  ]
}
```

## 最佳实践

1. **定期分析日志**：每周分析一次 token 使用情况，识别异常模式
2. **设置合理的告警阈值**：根据实际使用情况调整告警阈值
3. **按租户分摊成本**：使用 `tenantId` 字段准确计算每个租户的成本
4. **优化高 token 使用场景**：识别并优化 token 使用量高的场景
5. **监控模型性能**：比较不同模型的 token 使用效率和响应时间

## 相关文档

- [Token 使用统计实现总结](./TOKEN_USAGE_LOGGING_SUMMARY.md)
- [Genkit 多模型支持设计文档](../../.kiro/specs/genkit-multi-model-support/design.md)
- [日志和监控完善任务](../../.kiro/specs/genkit-multi-model-support/tasks.md#task-64-日志和监控完善)
