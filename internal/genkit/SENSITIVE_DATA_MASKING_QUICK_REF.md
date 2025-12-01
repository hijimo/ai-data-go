# 敏感信息脱敏 - 快速参考

## 核心函数

### MaskAPIKey - API 密钥脱敏

```go
import "genkit-ai-service/internal/genkit"

// 脱敏 API 密钥
masked := genkit.MaskAPIKey("sk-1234567890abcdef")
// 返回: "sk-1****cdef"
```

**规则**：

- 长度 ≤ 8：返回 `****`
- 长度 > 8：保留前4位和后4位，中间用 `****` 替换

### MaskSensitiveConfig - 配置对象脱敏

```go
import "genkit-ai-service/internal/genkit"

config := map[string]interface{}{
    "apiKey": "sk-1234567890abcdef",
    "model":  "gpt-4",
}

// 脱敏配置对象
masked := genkit.MaskSensitiveConfig(config)
// 返回: {"apiKey": "sk-1****cdef", "model": "gpt-4"}
```

**脱敏字段**：`apiKey`、`APIKey`、`api_key`

## 使用场景

### ✅ 正确用法

```go
// 在日志中记录配置
logger.InfoContext(ctx, "配置信息", logger.Fields{
    "config": genkit.MaskSensitiveConfig(config),
})

// 在错误消息中使用脱敏密钥
return fmt.Errorf("API 密钥验证失败: %s", genkit.MaskAPIKey(apiKey))

// 调试日志
logger.DebugContext(ctx, "初始化提供商", logger.Fields{
    "provider": providerType,
    "apiKey":   genkit.MaskAPIKey(apiKey),
})
```

### ❌ 错误用法

```go
// ❌ 直接记录 API 密钥
logger.InfoContext(ctx, "配置", logger.Fields{
    "apiKey": apiKey,  // 泄露敏感信息！
})

// ❌ 在错误消息中包含原始密钥
return fmt.Errorf("验证失败: %s", apiKey)

// ❌ 记录完整的配置对象
logger.DebugContext(ctx, "配置", logger.Fields{
    "config": config,  // 可能包含 API 密钥！
})
```

## 安全检查清单

在添加日志时检查：

- [ ] 不记录原始 API 密钥
- [ ] 不记录完整配置对象（除非已脱敏）
- [ ] 错误消息中不包含敏感信息
- [ ] 调试日志中的敏感信息已脱敏

## 测试

```bash
# 运行脱敏功能测试
go test -v -run TestMask ./internal/genkit/
```

## 相关文档

- [详细实现文档](./SENSITIVE_DATA_MASKING.md)
- [任务完成报告](./TASK-6.4-SENSITIVE-DATA-MASKING-COMPLETION.md)
