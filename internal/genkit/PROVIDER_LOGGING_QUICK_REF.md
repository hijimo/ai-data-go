# 提供商日志记录快速参考

## 概述

Genkit Client 现在包含全面的日志记录功能，记录提供商选择、初始化、API调用和错误处理的完整流程。

## 日志级别

- **DEBUG**：详细的调试信息
- **INFO**：正常的操作信息
- **WARN**：警告信息
- **ERROR**：错误信息

## 关键日志点

### 1. 提供商选择

```go
logger.InfoContext(ctx, "选择模型提供商", logger.Fields{
    "tenantId":      tenantID,
    "modelName":     modelName,
    "provider":      modelConfig.ModelProvider,
    "model":         genkitConfig.Model,
})
```

### 2. 缓存命中

```go
logger.DebugContext(ctx, "从缓存获取 Genkit 实例", logger.Fields{
    "tenantId":  tenantID,
    "modelName": modelName,
    "cacheHit":  true,
})
```

### 3. 初始化成功

```go
logger.InfoContext(ctx, "成功初始化并缓存 Genkit 实例", logger.Fields{
    "tenantId":      tenantID,
    "modelName":     modelName,
    "provider":      modelConfig.ModelProvider,
    "cacheKey":      cacheKey,
})
```

### 4. 生成成功

```go
logger.InfoContext(ctx, "生成内容成功", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "promptTokens":     result.Usage.PromptTokens,
    "completionTokens": result.Usage.CompletionTokens,
    "totalTokens":      result.Usage.TotalTokens,
    "responseLen":      len(result.Text),
})
```

### 5. 流式生成完成

```go
logger.InfoContext(ctx, "流式生成完成", logger.Fields{
    "tenantId":         tenantID,
    "modelName":        modelName,
    "model":            genkitConfig.Model,
    "duration":         duration.String(),
    "ttfb":             ttfb.String(),
    "chunkCount":       chunkCount,
    "totalContentLen":  len(totalContent),
    "promptTokens":     usage.PromptTokens,
    "completionTokens": usage.CompletionTokens,
    "totalTokens":      usage.TotalTokens,
})
```

### 6. 错误日志

```go
logger.ErrorContext(ctx, "获取模型配置失败", logger.Fields{
    "tenantId":  tenantID,
    "modelName": modelName,
    "error":     err.Error(),
})
```

## 日志字段说明

| 字段 | 说明 | 示例 |
|------|------|------|
| `tenantId` | 租户ID | `"e9186d5a-f2b1-475a-ba9d-5dff18e834f7"` |
| `modelName` | 模型名称 | `"gemini-pro"` |
| `provider` | 提供商类型 | `"googlegenai"`, `"azureopenai"`, `"bianlian"` |
| `model` | 实际使用的模型 | `"gemini-1.5-pro"`, `"gpt-4"` |
| `cacheKey` | 缓存键 | `"tenant_model"` |
| `cacheHit` | 是否命中缓存 | `true`, `false` |
| `duration` | **API 调用总耗时** | `"1.234s"` |
| `ttfb` | **首字节时间 (Time To First Byte)** | `"0.456s"` |
| `chunkCount` | 接收的块数量 | `25` |
| `promptTokens` | 提示词Token数 | `10` |
| `completionTokens` | 完成Token数 | `50` |
| `totalTokens` | 总Token数 | `60` |
| `responseLen` | 响应长度 | `200` |
| `totalContentLen` | 总内容长度 | `500` |
| `error` | 错误信息 | `"[404] 模型配置不存在"` |

### 性能指标说明

#### duration (API 调用总耗时)

- **非流式调用**：从调用开始到收到完整响应的总时间
- **流式调用**：从调用开始到接收完所有响应块的总时间
- **失败调用**：从调用开始到失败的时间
- **用途**：评估整体性能、识别慢查询、设置超时时间

#### ttfb (首字节时间)

- **仅流式调用**：从调用开始到收到第一个响应块的时间
- **用途**：评估模型响应速度、优化用户体验
- **正常范围**：通常在 0.3s - 1s 之间
- **告警阈值**：超过 2s 需要关注

## 查看日志

### 1. 查看所有日志

```bash
tail -f logs/app-$(date +%Y-%m-%d).log
```

### 2. 查看特定租户的日志

```bash
grep "tenantId.*e9186d5a" logs/app-$(date +%Y-%m-%d).log
```

### 3. 查看提供商选择日志

```bash
grep "选择模型提供商" logs/app-$(date +%Y-%m-%d).log
```

### 4. 查看错误日志

```bash
grep '"level":"ERROR"' logs/app-$(date +%Y-%m-%d).log
```

### 5. 查看性能日志 (API 调用耗时)

```bash
# 查看所有包含耗时信息的日志
grep -E "(duration|ttfb)" logs/app-$(date +%Y-%m-%d).log | jq .

# 查看非流式调用的耗时
grep "生成内容成功" logs/app-$(date +%Y-%m-%d).log | jq '{timestamp, duration: .fields.duration, model: .fields.modelName}'

# 查看流式调用的耗时和首字节时间
grep "流式生成完成" logs/app-$(date +%Y-%m-%d).log | jq '{timestamp, duration: .fields.duration, ttfb: .fields.ttfb, model: .fields.modelName}'
```

## 日志分析示例

### 1. 统计各提供商的使用次数

```bash
grep "选择模型提供商" logs/app-*.log | \
  jq -r '.fields.provider' | \
  sort | uniq -c | sort -rn
```

### 2. 计算平均响应时间 (API 调用耗时)

```bash
# 非流式调用平均响应时间
grep "生成内容成功" logs/app-*.log | \
  jq -r '.fields.duration' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "非流式平均响应时间:", sum/count "s"}'

# 流式调用平均响应时间
grep "流式生成完成" logs/app-*.log | \
  jq -r '.fields.duration' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "流式平均响应时间:", sum/count "s"}'

# 流式调用平均首字节时间
grep "流式生成完成" logs/app-*.log | \
  jq -r '.fields.ttfb' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "平均首字节时间:", sum/count "s"}'
```

### 3. 统计Token使用情况

```bash
grep "生成内容成功" logs/app-*.log | \
  jq -r '.fields | "\(.promptTokens) \(.completionTokens) \(.totalTokens)"' | \
  awk '{pt+=$1; ct+=$2; tt+=$3; count++} END {
    print "平均提示词Token:", pt/count;
    print "平均完成Token:", ct/count;
    print "平均总Token:", tt/count
  }'
```

### 4. 查找慢查询 (API 调用耗时超过阈值)

```bash
# 查找响应时间超过 2 秒的请求
grep -E "(生成内容成功|流式生成完成)" logs/app-*.log | \
  jq 'select(.fields.duration | gsub("s$";"") | tonumber > 2) | {
    timestamp: .timestamp,
    tenantId: .fields.tenantId,
    modelName: .fields.modelName,
    duration: .fields.duration,
    type: (if .message == "生成内容成功" then "非流式" else "流式" end)
  }'

# 查找首字节时间超过 1 秒的流式请求
grep "流式生成完成" logs/app-*.log | \
  jq 'select(.fields.ttfb | gsub("s$";"") | tonumber > 1) | {
    timestamp: .timestamp,
    tenantId: .fields.tenantId,
    modelName: .fields.modelName,
    ttfb: .fields.ttfb,
    duration: .fields.duration
  }'
```

### 5. 统计缓存命中率

```bash
total=$(grep "开始获取或初始化" logs/app-*.log | wc -l)
hits=$(grep '"cacheHit":true' logs/app-*.log | wc -l)
echo "缓存命中率: $(echo "scale=2; $hits * 100 / $total" | bc)%"
```

## 监控建议

### 1. 关键指标

- **提供商分布**：各提供商的使用比例
- **响应时间 (API 调用耗时)**：
  - 平均响应时间
  - P95 响应时间（95% 的请求在此时间内完成）
  - P99 响应时间（99% 的请求在此时间内完成）
  - 平均首字节时间 (TTFB)
- **Token使用**：总Token使用量和平均值
- **错误率**：错误请求占比
- **缓存命中率**：缓存命中的比例
- **慢查询率**：超过阈值的请求占比

### 2. 告警规则

- **响应时间告警**：
  - 平均响应时间超过 3 秒
  - P95 响应时间超过 5 秒
  - P99 响应时间超过 10 秒
  - 平均首字节时间超过 1 秒
- **慢查询告警**：慢查询占比超过 5%
- **错误率告警**：错误率超过 5%
- **缓存告警**：缓存命中率低于 80%
- **Token告警**：Token使用量异常增长

### 3. 日志保留

- 建议保留最近30天的日志
- 压缩归档超过7天的日志
- 定期清理超过30天的日志

## 故障排查

### 1. 配置不存在

```json
{
  "level": "ERROR",
  "message": "获取模型配置失败",
  "fields": {
    "error": "[404] 模型配置不存在"
  }
}
```

**解决方案**：检查 model_configurations 表中是否存在对应的配置。

### 2. 模型已禁用

```json
{
  "level": "WARN",
  "message": "模型已禁用",
  "fields": {
    "modelName": "disabled-model"
  }
}
```

**解决方案**：在 model_configurations 表中启用该模型。

### 3. 初始化失败

```json
{
  "level": "ERROR",
  "message": "初始化提供商失败",
  "fields": {
    "provider": "azureopenai",
    "error": "创建 Azure OpenAI 插件失败"
  }
}
```

**解决方案**：检查配置是否完整（如 azureEndpoint、azureDeployment 等）。

### 4. 生成失败

```json
{
  "level": "ERROR",
  "message": "生成内容失败",
  "fields": {
    "error": "API call failed: 429 Too Many Requests"
  }
}
```

**解决方案**：检查API配额和速率限制。

## 最佳实践

1. **使用结构化日志**：所有日志都使用JSON格式，便于解析和分析
2. **包含上下文信息**：每条日志都包含租户ID、模型名称等关键信息
3. **记录性能指标**：记录耗时、Token使用等性能指标
4. **敏感信息脱敏**：API密钥等敏感信息不出现在日志中
5. **合理的日志级别**：根据重要性选择合适的日志级别

## 相关文档

- [任务完成报告](../../TASK_6.4_PROVIDER_LOGGING_COMPLETION.md)
- [日志记录器使用指南](../logger/USAGE_GUIDE.md)
- [监控指南](../../docs/MONITORING_GUIDE.md)
