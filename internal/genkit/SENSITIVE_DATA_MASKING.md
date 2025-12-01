# 敏感信息脱敏实现文档

## 概述

本文档描述了 Genkit 模块中敏感信息脱敏的实现，确保 API 密钥等敏感数据不会在日志中泄露。

## 实现的功能

### 1. API 密钥脱敏

提供 `MaskAPIKey` 函数用于脱敏 API 密钥：

```go
func MaskAPIKey(apiKey string) string
```

**脱敏规则**：

- 空字符串：返回空字符串
- 长度 ≤ 8：返回 `****`
- 长度 > 8：保留前4位和后4位，中间用 `****` 替换

**示例**：

```go
MaskAPIKey("sk-1234567890abcdef")  // 返回: "sk-1****cdef"
MaskAPIKey("short")                 // 返回: "****"
MaskAPIKey("")                      // 返回: ""
```

### 2. 配置对象脱敏

提供 `MaskSensitiveConfig` 函数用于脱敏配置对象中的敏感字段：

```go
func MaskSensitiveConfig(config interface{}) map[string]interface{}
```

**脱敏字段**：

- `apiKey`
- `APIKey`
- `api_key`

**使用示例**：

```go
config := map[string]interface{}{
    "apiKey": "sk-1234567890abcdef",
    "model":  "gpt-4",
}

masked := MaskSensitiveConfig(config)
// 返回: {"apiKey": "sk-1****cdef", "model": "gpt-4"}
```

## 当前实现状态

### ✅ 已实现的保护

1. **Genkit Client 层**
   - 所有日志记录都不包含原始 API 密钥
   - 配置初始化时不记录敏感信息
   - 错误日志中不包含 API 密钥

2. **模型配置服务层**
   - API 密钥在存储前已加密
   - 日志中不记录原始 API 密钥
   - 审计日志中不包含敏感信息

3. **Handler 层**
   - 请求日志不记录 API 密钥
   - 响应中的 API 密钥字段已标记为 `json:"-"`，不会返回给客户端

### 🔍 代码审查结果

经过全面审查，以下位置已确认不会泄露敏感信息：

1. **internal/genkit/client.go**
   - ✅ 所有 `logger.InfoContext`、`logger.DebugContext`、`logger.ErrorContext` 调用
   - ✅ 不记录配置对象本身
   - ✅ 只记录非敏感的元数据（租户ID、模型名称、提供商类型等）

2. **internal/service/model_configuration_service.go**
   - ✅ API 密钥在存储前已通过 `encryptionService.EncryptAPIKey` 加密
   - ✅ 日志中只记录配置ID、租户ID等非敏感信息
   - ✅ 审计日志不包含 API 密钥

3. **internal/api/handler/model_configuration_handler.go**
   - ✅ 请求日志不记录完整的请求体
   - ✅ 只记录租户ID、配置ID等元数据
   - ✅ 响应中的 API 密钥字段已标记为 `json:"-"`

## 使用指南

### 在日志中使用脱敏函数

如果需要在日志中记录配置信息（用于调试），应使用脱敏函数：

```go
// ❌ 错误：直接记录配置对象
logger.InfoContext(ctx, "配置信息", logger.Fields{
    "config": config,  // 可能包含 API 密钥
})

// ✅ 正确：使用脱敏函数
logger.InfoContext(ctx, "配置信息", logger.Fields{
    "config": MaskSensitiveConfig(config),
})
```

### 在错误消息中避免敏感信息

```go
// ❌ 错误：错误消息中包含 API 密钥
return fmt.Errorf("API 密钥验证失败: %s", apiKey)

// ✅ 正确：使用脱敏后的密钥
return fmt.Errorf("API 密钥验证失败: %s", MaskAPIKey(apiKey))
```

## 测试覆盖

已实现完整的单元测试：

1. **TestMaskAPIKey**
   - 测试空字符串
   - 测试短密钥（≤8位）
   - 测试标准长度密钥
   - 测试各种提供商的密钥格式

2. **TestMaskSensitiveConfig**
   - 测试不同的敏感字段名称（apiKey、APIKey、api_key）
   - 测试复杂配置对象
   - 测试结构体配置
   - 测试边界情况（空配置、nil、非JSON可序列化）

## 安全检查清单

在添加新的日志记录时，请检查：

- [ ] 日志中不包含原始 API 密钥
- [ ] 日志中不包含完整的配置对象（除非已脱敏）
- [ ] 错误消息中不包含敏感信息
- [ ] 调试日志中的敏感信息已脱敏
- [ ] 审计日志中不包含密码、令牌等敏感数据

## 相关文件

- `internal/genkit/config.go` - 脱敏函数实现
- `internal/genkit/config_mask_test.go` - 脱敏函数测试
- `internal/genkit/client.go` - Genkit 客户端实现
- `internal/service/model_configuration_service.go` - 模型配置服务
- `internal/api/handler/model_configuration_handler.go` - 模型配置 Handler

## 最佳实践

1. **永远不要在日志中记录原始 API 密钥**
2. **使用脱敏函数处理所有可能包含敏感信息的数据**
3. **在数据库中存储加密后的 API 密钥**
4. **API 响应中不返回 API 密钥（使用 `json:"-"` 标签）**
5. **定期审查日志输出，确保没有敏感信息泄露**

## 更新日志

- 2024-12-01: 初始实现，添加 `MaskAPIKey` 和 `MaskSensitiveConfig` 函数
- 2024-12-01: 完成代码审查，确认所有日志记录都不包含敏感信息
- 2024-12-01: 添加完整的单元测试覆盖
