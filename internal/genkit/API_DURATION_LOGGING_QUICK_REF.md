# API 调用耗时记录快速参考

## 概述

Genkit Client 记录了所有 API 调用的详细耗时信息，包括非流式调用的总耗时和流式调用的总耗时及首字节时间 (TTFB)。

## 记录的指标

### 1. 非流式调用 (Generate)

| 指标 | 字段名 | 说明 | 示例 |
|------|--------|------|------|
| 总耗时 | `duration` | 从调用开始到收到完整响应的时间 | `"1.234s"` |

### 2. 流式调用 (GenerateStream)

| 指标 | 字段名 | 说明 | 示例 |
|------|--------|------|------|
| 首字节时间 | `ttfb` | 从调用开始到收到第一个响应块的时间 | `"0.456s"` |
| 总耗时 | `duration` | 从调用开始到接收完所有响应块的时间 | `"2.345s"` |

### 3. 失败调用

| 指标 | 字段名 | 说明 | 示例 |
|------|--------|------|------|
| 失败耗时 | `duration` | 从调用开始到失败的时间 | `"0.523s"` |

## 日志示例

### 非流式调用成功

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

### 流式调用 - 首字节

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

### 流式调用完成

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

### 调用失败

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

## 查询命令

### 1. 查看所有耗时日志

```bash
# 查看今天的耗时日志
grep -E "(duration|ttfb)" logs/app-$(date +%Y-%m-%d).log | jq .

# 查看非流式调用的耗时
grep "生成内容成功" logs/app-$(date +%Y-%m-%d).log | \
  jq '{timestamp, duration: .fields.duration, model: .fields.modelName}'

# 查看流式调用的耗时和首字节时间
grep "流式生成完成" logs/app-$(date +%Y-%m-%d).log | \
  jq '{timestamp, duration: .fields.duration, ttfb: .fields.ttfb, model: .fields.modelName}'
```

### 2. 计算平均响应时间

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

### 3. 查找慢查询

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

### 4. 按模型统计平均响应时间

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

### 6. 统计慢查询占比

```bash
total=$(grep -E "(生成内容成功|流式生成完成)" logs/app-*.log | wc -l)
slow=$(grep -E "(生成内容成功|流式生成完成)" logs/app-*.log | \
  jq -r 'select(.fields.duration | gsub("s$";"") | tonumber > 2) | .fields.duration' | \
  wc -l)
echo "慢查询占比: $(echo "scale=2; $slow * 100 / $total" | bc)%"
```

## 性能基准

### 正常范围

| 指标 | 正常范围 | 说明 |
|------|----------|------|
| 非流式平均响应时间 | 0.5s - 2s | 取决于模型和提示词长度 |
| 流式平均响应时间 | 1s - 3s | 取决于生成内容长度 |
| 平均首字节时间 | 0.3s - 1s | 模型响应速度 |
| P95 响应时间 | < 5s | 95% 的请求应在此时间内完成 |
| P99 响应时间 | < 10s | 99% 的请求应在此时间内完成 |

### 告警阈值

| 指标 | 告警阈值 | 级别 |
|------|----------|------|
| 平均响应时间 | > 3s | ⚠️ 警告 |
| P95 响应时间 | > 5s | ⚠️ 警告 |
| P99 响应时间 | > 10s | 🚨 严重 |
| 平均首字节时间 | > 1s | ⚠️ 警告 |
| 慢查询占比 | > 5% | ⚠️ 警告 |

## 性能优化建议

### 1. 响应时间过长

**可能原因**：

- 模型负载过高
- 网络延迟
- 提示词过长
- 生成内容过多

**优化方案**：

- 切换到响应更快的模型
- 优化网络配置
- 精简提示词
- 限制生成长度

### 2. 首字节时间过长

**可能原因**：

- 模型冷启动
- 提示词处理慢
- 网络延迟

**优化方案**：

- 使用缓存预热
- 优化提示词结构
- 检查网络连接

### 3. 慢查询过多

**可能原因**：

- 并发请求过多
- 模型配额不足
- 系统资源不足

**优化方案**：

- 实施请求限流
- 增加模型配额
- 扩展系统资源

## 测试脚本

使用提供的测试脚本验证耗时记录功能：

```bash
# 运行测试脚本
./test/test_api_duration_logging.sh

# 查看测试结果
# 脚本会自动分析日志并生成统计报告
```

## 相关文档

- [提供商日志记录快速参考](PROVIDER_LOGGING_QUICK_REF.md)
- [任务完成报告](../../TASK_6.4_API_DURATION_LOGGING_COMPLETION.md)
- [监控指南](../../docs/MONITORING_GUIDE.md)

## 最佳实践

1. **定期监控**：每天检查平均响应时间和慢查询占比
2. **设置告警**：配置自动告警规则，及时发现性能问题
3. **性能分析**：定期分析 P95 和 P99 响应时间，识别性能瓶颈
4. **优化迭代**：根据监控数据持续优化系统性能
5. **容量规划**：根据历史数据预测未来容量需求

## 故障排查

### 问题 1: 响应时间突然增加

**排查步骤**：

1. 检查是否有大量并发请求
2. 检查模型提供商是否有故障
3. 检查网络连接是否正常
4. 检查系统资源使用情况

### 问题 2: 首字节时间异常

**排查步骤**：

1. 检查模型是否冷启动
2. 检查提示词是否过长
3. 检查网络延迟
4. 检查模型提供商状态

### 问题 3: 慢查询占比过高

**排查步骤**：

1. 分析慢查询的共同特征
2. 检查是否特定模型响应慢
3. 检查是否特定租户请求慢
4. 检查系统资源是否充足
