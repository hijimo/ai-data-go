# Task 6.4 - API 调用耗时记录完成报告

## 任务概述

实现任务 6.4 的子任务：记录 API 调用耗时。

## 实现状态

✅ **已完成** - API 调用耗时记录功能已在之前的实现中完成。

## 实现内容

### 1. Generate 方法的耗时记录

在 `internal/genkit/client.go` 的 `Generate` 方法中：

```go
func (c *client) Generate(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (*GenerateResult, error) {
    startTime := time.Now()  // 记录开始时间
    
    // ... 业务逻辑 ...
    
    // 成功时记录耗时
    duration := time.Since(startTime)
    logger.InfoContext(ctx, "生成内容成功", logger.Fields{
        "tenantId":         tenantID,
        "modelName":        modelName,
        "model":            genkitConfig.Model,
        "duration":         duration.String(),  // 记录总耗时
        "promptTokens":     result.Usage.PromptTokens,
        "completionTokens": result.Usage.CompletionTokens,
        "totalTokens":      result.Usage.TotalTokens,
        "responseLen":      len(result.Text),
    })
    
    // 失败时也记录耗时
    duration := time.Since(startTime)
    logger.ErrorContext(ctx, "生成内容失败", logger.Fields{
        "tenantId":  tenantID,
        "modelName": modelName,
        "model":     genkitConfig.Model,
        "duration":  duration.String(),  // 记录失败时的耗时
        "error":     err.Error(),
    })
}
```

### 2. GenerateStream 方法的耗时记录

在 `internal/genkit/client.go` 的 `GenerateStream` 方法中：

```go
func (c *client) GenerateStream(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (<-chan StreamChunk, error) {
    startTime := time.Now()  // 记录开始时间
    
    go func() {
        var firstChunkTime := time.Time{}
        
        // 记录首字节时间 (TTFB - Time To First Byte)
        if chunkCount == 0 {
            firstChunkTime = time.Now()
            ttfb := firstChunkTime.Sub(startTime)
            logger.InfoContext(ctx, "收到首个响应块", logger.Fields{
                "tenantId":  tenantID,
                "modelName": modelName,
                "model":     genkitConfig.Model,
                "ttfb":      ttfb.String(),  // 记录首字节时间
            })
        }
        
        // 完成时记录总耗时和TTFB
        duration := time.Since(startTime)
        var ttfb time.Duration
        if !firstChunkTime.IsZero() {
            ttfb = firstChunkTime.Sub(startTime)
        }
        
        logger.InfoContext(ctx, "流式生成完成", logger.Fields{
            "tenantId":         tenantID,
            "modelName":        modelName,
            "model":            genkitConfig.Model,
            "duration":         duration.String(),  // 记录总耗时
            "ttfb":             ttfb.String(),      // 记录首字节时间
            "chunkCount":       chunkCount,
            "totalContentLen":  len(totalContent),
            "promptTokens":     usage.PromptTokens,
            "completionTokens": usage.CompletionTokens,
            "totalTokens":      usage.TotalTokens,
        })
        
        // 失败时也记录耗时
        duration := time.Since(startTime)
        logger.ErrorContext(ctx, "流式生成失败", logger.Fields{
            "tenantId":   tenantID,
            "modelName":  modelName,
            "model":      genkitConfig.Model,
            "duration":   duration.String(),  // 记录失败时的耗时
            "chunkCount": chunkCount,
            "error":      err.Error(),
        })
    }()
}
```

## 记录的耗时指标

### 1. 非流式调用 (Generate)

| 指标 | 说明 | 示例 |
|------|------|------|
| `duration` | API 调用总耗时 | `"1.234s"` |

记录时机：

- ✅ 调用成功时
- ✅ 调用失败时

### 2. 流式调用 (GenerateStream)

| 指标 | 说明 | 示例 |
|------|------|------|
| `ttfb` | 首字节时间 (Time To First Byte) | `"0.456s"` |
| `duration` | API 调用总耗时 | `"2.345s"` |

记录时机：

- ✅ 收到首个响应块时（记录 TTFB）
- ✅ 流式生成完成时（记录总耗时和 TTFB）
- ✅ 流式生成失败时（记录总耗时）

## 日志示例

### 1. 非流式调用成功

```json
{
  "timestamp": "2025-11-29T14:28:50Z",
  "level": "INFO",
  "message": "生成内容成功",
  "fields": {
    "tenantId": "e9186d5a-f2b1-475a-ba9d-5dff18e834f7",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "duration": "1.234s",
    "promptTokens": 10,
    "completionTokens": 50,
    "totalTokens": 60,
    "responseLen": 200
  }
}
```

### 2. 非流式调用失败

```json
{
  "timestamp": "2025-11-29T14:28:50Z",
  "level": "ERROR",
  "message": "生成内容失败",
  "fields": {
    "tenantId": "e9186d5a-f2b1-475a-ba9d-5dff18e834f7",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "duration": "0.523s",
    "error": "API call failed: 429 Too Many Requests"
  }
}
```

### 3. 流式调用 - 首字节

```json
{
  "timestamp": "2025-11-29T14:28:50.456Z",
  "level": "INFO",
  "message": "收到首个响应块",
  "fields": {
    "tenantId": "e9186d5a-f2b1-475a-ba9d-5dff18e834f7",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "ttfb": "0.456s"
  }
}
```

### 4. 流式调用完成

```json
{
  "timestamp": "2025-11-29T14:28:51Z",
  "level": "INFO",
  "message": "流式生成完成",
  "fields": {
    "tenantId": "e9186d5a-f2b1-475a-ba9d-5dff18e834f7",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "duration": "2.345s",
    "ttfb": "0.456s",
    "chunkCount": 25,
    "totalContentLen": 500,
    "promptTokens": 10,
    "completionTokens": 100,
    "totalTokens": 110
  }
}
```

### 5. 流式调用失败

```json
{
  "timestamp": "2025-11-29T14:28:51Z",
  "level": "ERROR",
  "message": "流式生成失败",
  "fields": {
    "tenantId": "e9186d5a-f2b1-475a-ba9d-5dff18e834f7",
    "modelName": "gemini-pro",
    "model": "gemini-1.5-pro",
    "duration": "1.123s",
    "chunkCount": 5,
    "error": "context canceled"
  }
}
```

## 性能分析

### 1. 查看平均响应时间

```bash
# 非流式调用
grep "生成内容成功" logs/app-*.log | \
  jq -r '.fields.duration' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "平均响应时间:", sum/count "s"}'

# 流式调用
grep "流式生成完成" logs/app-*.log | \
  jq -r '.fields.duration' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "平均响应时间:", sum/count "s"}'
```

### 2. 查看平均首字节时间

```bash
grep "流式生成完成" logs/app-*.log | \
  jq -r '.fields.ttfb' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print "平均首字节时间:", sum/count "s"}'
```

### 3. 查找慢查询 (超过2秒)

```bash
grep -E "(生成内容成功|流式生成完成)" logs/app-*.log | \
  jq 'select(.fields.duration | gsub("s$";"") | tonumber > 2) | {
    timestamp: .timestamp,
    tenantId: .fields.tenantId,
    modelName: .fields.modelName,
    duration: .fields.duration
  }'
```

### 4. 按提供商统计平均响应时间

```bash
grep "生成内容成功" logs/app-*.log | \
  jq -r '[.fields.modelName, .fields.duration] | @tsv' | \
  awk '{
    gsub(/s$/, "", $2);
    sum[$1] += $2;
    count[$1]++;
  } END {
    for (model in sum) {
      printf "%s: %.3fs\n", model, sum[model]/count[model];
    }
  }' | sort -t: -k2 -n
```

### 5. 计算 P95 和 P99 响应时间

```bash
grep "生成内容成功" logs/app-*.log | \
  jq -r '.fields.duration' | \
  sed 's/s$//' | \
  sort -n | \
  awk '{
    values[NR] = $1;
  } END {
    p95_idx = int(NR * 0.95);
    p99_idx = int(NR * 0.99);
    print "P95:", values[p95_idx] "s";
    print "P99:", values[p99_idx] "s";
  }'
```

## 监控建议

### 1. 关键指标

- **平均响应时间**：所有调用的平均耗时
- **P95 响应时间**：95% 的请求在此时间内完成
- **P99 响应时间**：99% 的请求在此时间内完成
- **平均首字节时间**：流式调用的平均 TTFB
- **慢查询数量**：超过阈值的请求数量

### 2. 告警规则

- ⚠️ 平均响应时间超过 3 秒
- ⚠️ P95 响应时间超过 5 秒
- ⚠️ P99 响应时间超过 10 秒
- ⚠️ 平均首字节时间超过 1 秒
- ⚠️ 慢查询占比超过 5%

### 3. 性能优化建议

根据耗时数据进行优化：

1. **缓存优化**：如果缓存命中率低，考虑增加缓存容量
2. **提供商选择**：如果某个提供商响应慢，考虑切换到其他提供商
3. **并发控制**：如果并发请求导致响应变慢，考虑限流
4. **超时设置**：根据 P99 响应时间设置合理的超时时间

## 优势

1. ✅ **完整的性能监控**：记录了所有 API 调用的耗时
2. ✅ **细粒度指标**：区分了总耗时和首字节时间
3. ✅ **失败也记录**：即使调用失败也记录耗时，便于分析失败原因
4. ✅ **结构化日志**：使用 JSON 格式，便于解析和分析
5. ✅ **上下文信息**：包含租户ID、模型名称等关键信息

## 相关文档

- [提供商日志记录快速参考](internal/genkit/PROVIDER_LOGGING_QUICK_REF.md)
- [任务 6.4 完成报告](TASK_6.4_PROVIDER_LOGGING_COMPLETION.md)
- [监控指南](docs/MONITORING_GUIDE.md)

## 文件变更

- ✅ `internal/genkit/client.go` - 已包含耗时记录功能
- ✅ `TASK_6.4_API_DURATION_LOGGING_COMPLETION.md` - 本文档

## 总结

API 调用耗时记录功能已在之前的实现中完成。所有 API 调用（包括成功和失败）都记录了详细的耗时信息：

- **非流式调用**：记录总耗时
- **流式调用**：记录总耗时和首字节时间 (TTFB)

这些指标为性能监控、问题排查和系统优化提供了重要的数据支持。
