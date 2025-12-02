# Genkit 多模型故障排查指南

## 概述

本文档提供 Genkit 多模型支持系统的常见问题和解决方案。按照问题类型分类，帮助快速定位和解决问题。

## 目录

- [配置相关问题](#配置相关问题)
- [API 调用问题](#api-调用问题)
- [提供商特定问题](#提供商特定问题)
- [性能问题](#性能问题)
- [权限问题](#权限问题)
- [日志和监控](#日志和监控)

## 配置相关问题

### 问题 1: 配置不存在

**错误信息**:

```
获取模型配置失败: record not found
```

**可能原因**:

1. 租户 ID 或模型名称不正确
2. 配置已被删除
3. 配置未创建

**解决方案**:

1. **检查配置是否存在**:

```bash
curl -X GET "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

2. **验证租户 ID**:

```bash
# 检查当前用户的租户 ID
curl -X GET "http://localhost:8080/api/v1/users/me" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

3. **创建配置**:

```bash
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpt-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-..."
  }'
```

### 问题 2: 配置验证失败

**错误信息**:

```
配置验证失败: Azure OpenAI 配置缺少必需字段: azureEndpoint
```

**可能原因**:

- 缺少提供商特定的必需字段
- 字段格式不正确

**解决方案**:

1. **检查 Azure OpenAI 配置**:

```json
{
  "name": "Azure GPT-4",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "your-key",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

2. **检查百炼配置**:

```json
{
  "name": "通义千问",
  "model": "qwen-turbo",
  "modelProvider": "bianlian",
  "apiKey": "sk-...",
  "queryParams": {
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "bailianWorkspace": "default"
  }
}
```

3. **检查自定义 OpenAI 配置**:

```json
{
  "name": "自定义模型",
  "model": "custom-model",
  "modelProvider": "custom_openai",
  "baseUrl": "https://your-api.com/v1",  // 必需
  "apiKey": "your-key"
}
```

### 问题 3: 模型已禁用

**错误信息**:

```
模型已禁用: gpt-4
```

**解决方案**:

1. **检查模型状态**:

```bash
curl -X GET "http://localhost:8080/api/v1/model-configurations/{id}" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

2. **启用模型**:

```bash
curl -X PATCH "http://localhost:8080/api/v1/model-configurations/{id}/status" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "enabled"}'
```

## API 调用问题

### 问题 4: API 密钥无效

**错误信息**:

```
生成内容失败: invalid API key
```

**可能原因**:

1. API 密钥错误或过期
2. API 密钥权限不足
3. API 密钥被撤销

**解决方案**:

1. **验证 API 密钥**:
   - Google AI: 访问 [Google AI Studio](https://makersuite.google.com/app/apikey)
   - OpenAI: 访问 [OpenAI Platform](https://platform.openai.com/api-keys)
   - Azure: 访问 [Azure Portal](https://portal.azure.com)
   - 百炼: 访问 [百炼控制台](https://bailian.console.aliyun.com/)

2. **更新 API 密钥**:

```bash
curl -X PUT "http://localhost:8080/api/v1/model-configurations/{id}" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"apiKey": "new-api-key"}'
```

3. **检查密钥权限**:
   - 确认密钥有调用相应模型的权限
   - 检查密钥的使用限制和配额

### 问题 5: 速率限制

**错误信息**:

```
生成内容失败: rate limit exceeded
```

**可能原因**:

- 超过 API 调用频率限制
- 超过 token 使用配额

**解决方案**:

1. **实施重试机制**:

```go
// 示例：指数退避重试
func retryWithBackoff(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        if isRateLimitError(err) {
            waitTime := time.Duration(math.Pow(2, float64(i))) * time.Second
            time.Sleep(waitTime)
            continue
        }
        
        return err
    }
    return fmt.Errorf("max retries exceeded")
}
```

2. **检查配额使用情况**:
   - OpenAI: [Usage Dashboard](https://platform.openai.com/usage)
   - Azure: Azure Portal 监控页面
   - 百炼: 百炼控制台用量统计

3. **优化调用频率**:
   - 实施请求队列
   - 使用缓存减少重复调用
   - 批量处理请求

### 问题 6: 超时

**错误信息**:

```
生成内容失败: context deadline exceeded
```

**可能原因**:

1. 网络延迟过高
2. 模型响应时间过长
3. 超时设置过短

**解决方案**:

1. **检查网络连接**:

```bash
# 测试到提供商的网络连接
curl -I https://api.openai.com
curl -I https://dashscope.aliyuncs.com
```

2. **调整超时设置**:

```go
// 在调用时设置更长的超时
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

result, err := client.Generate(ctx, tenantID, modelName, prompt, options)
```

3. **使用流式响应**:
   - 流式响应可以更快返回首字节
   - 提升用户体验
   - 避免长时间等待

## 提供商特定问题

### Google AI (Gemini)

#### 问题 7: 模型不可用

**错误信息**:

```
model not found: gemini-1.5-pro
```

**解决方案**:

1. **检查模型名称**:
   - 确认使用正确的模型标识
   - 参考 [Google AI 文档](https://ai.google.dev/models)

2. **检查 API 密钥权限**:
   - 确认密钥有访问该模型的权限
   - 某些模型可能需要申请访问

3. **使用可用的模型**:

```json
{
  "model": "gemini-pro"  // 使用稳定版本
}
```

### OpenAI

#### 问题 8: 模型访问被拒绝

**错误信息**:

```
You do not have access to model gpt-4
```

**解决方案**:

1. **检查账户权限**:
   - 访问 [OpenAI Platform](https://platform.openai.com/account/limits)
   - 确认账户有 GPT-4 访问权限

2. **使用可用的模型**:

```json
{
  "model": "gpt-3.5-turbo"  // 使用有权限的模型
}
```

3. **申请访问权限**:
   - 联系 OpenAI 支持
   - 升级账户等级

### Azure OpenAI

#### 问题 9: 端点连接失败

**错误信息**:

```
failed to connect to Azure OpenAI endpoint
```

**可能原因**:

1. Endpoint URL 不正确
2. 网络无法访问 Azure
3. 防火墙阻止连接

**解决方案**:

1. **验证 Endpoint URL**:

```bash
# 测试连接
curl -I https://your-resource.openai.azure.com
```

2. **检查 URL 格式**:

```json
{
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    // 不要包含 /openai/deployments/... 路径
  }
}
```

3. **检查网络配置**:
   - 确认服务器可以访问 Azure
   - 检查防火墙规则
   - 验证 DNS 解析

#### 问题 10: 部署不存在

**错误信息**:

```
deployment not found: gpt-4
```

**解决方案**:

1. **检查部署名称**:
   - 登录 Azure Portal
   - 查看 Azure OpenAI 资源的部署列表
   - 使用正确的部署名称

2. **创建部署**:
   - 在 Azure Portal 中创建新部署
   - 选择合适的模型和区域

3. **更新配置**:

```json
{
  "queryParams": {
    "azureDeployment": "your-actual-deployment-name"
  }
}
```

### 阿里云百炼

#### 问题 11: 地域端点错误

**错误信息**:

```
failed to connect to Bailian endpoint
```

**解决方案**:

1. **使用正确的地域端点**:

```json
{
  "queryParams": {
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1"
  }
}
```

2. **测试端点连接**:

```bash
curl -I https://dashscope.aliyuncs.com/compatible-mode/v1
```

3. **检查网络访问**:
   - 确认服务器可以访问阿里云
   - 检查是否需要配置代理

#### 问题 12: 模型名称错误

**错误信息**:

```
model not found: qwen-turbo
```

**解决方案**:

1. **使用正确的模型名称**:

```json
{
  "model": "qwen-turbo"  // 确保拼写正确
}
```

2. **查看可用模型**:
   - 访问 [百炼文档](https://help.aliyun.com/zh/model-studio/)
   - 确认模型名称和可用性

## 性能问题

### 问题 13: 响应延迟高

**症状**:

- API 调用耗时过长
- 用户体验差

**诊断步骤**:

1. **检查日志中的性能指标**:

```json
{
  "message": "生成内容成功",
  "durationMs": 5000,  // 总耗时
  "ttfbMs": 2000,      // 首字节时间
  "promptTokens": 100,
  "completionTokens": 500
}
```

2. **分析延迟来源**:
   - 网络延迟
   - 模型处理时间
   - 系统处理时间

**解决方案**:

1. **使用流式响应**:

```javascript
// 前端使用 EventSource 接收流式响应
const eventSource = new EventSource('/api/v1/chat/sessions/{id}/messages/stream');
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // 实时显示内容
};
```

2. **优化提示词**:
   - 减少不必要的上下文
   - 使用更简洁的提示词
   - 限制输出长度

3. **选择合适的模型**:
   - 简单任务使用快速模型
   - 复杂任务使用强大模型

### 问题 14: 缓存未生效

**症状**:

- 每次调用都重新初始化
- 性能没有提升

**诊断步骤**:

1. **检查缓存大小**:

```go
// 在代码中添加日志
cacheSize := client.GetCacheSize()
logger.Info("当前缓存大小", "size", cacheSize)
```

2. **检查缓存键**:

```go
// 缓存键格式: {tenantID}_{modelName}
cacheKey := fmt.Sprintf("%s_%s", tenantID, modelName)
```

**解决方案**:

1. **确保使用相同的参数**:
   - 租户 ID 必须一致
   - 模型名称必须一致

2. **避免频繁清除缓存**:
   - 只在配置更新时清除
   - 不要在每次调用后清除

3. **监控缓存命中率**:

```go
// 添加缓存命中率监控
if cacheHit {
    metrics.IncrementCacheHit()
} else {
    metrics.IncrementCacheMiss()
}
```

## 权限问题

### 问题 15: 权限不足

**错误信息**:

```
权限不足：无法访问其他租户的模型配置
```

**可能原因**:

- 租户管理员尝试访问其他租户的配置
- 普通用户尝试管理配置

**解决方案**:

1. **检查用户角色**:

```bash
curl -X GET "http://localhost:8080/api/v1/users/me" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

2. **使用正确的权限**:
   - 平台管理员：可以管理所有租户的配置
   - 租户管理员：只能管理自己租户的配置
   - 普通用户：只能使用配置，不能管理

3. **联系管理员**:
   - 如需访问其他租户的配置，联系平台管理员
   - 如需管理权限，联系租户管理员

### 问题 16: Token 过期

**错误信息**:

```
401 Unauthorized: token expired
```

**解决方案**:

1. **刷新 Token**:

```bash
curl -X POST "http://localhost:8080/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "YOUR_REFRESH_TOKEN"}'
```

2. **重新登录**:

```bash
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }'
```

## 日志和监控

### 查看日志

#### 1. 应用日志

**位置**: `logs/app-{date}.log`

**查看最近的错误**:

```bash
tail -f logs/app-$(date +%Y-%m-%d).log | grep ERROR
```

**查看特定租户的日志**:

```bash
grep "tenantId.*738dbb1f" logs/app-$(date +%Y-%m-%d).log
```

**查看特定模型的日志**:

```bash
grep "modelName.*gpt-4" logs/app-$(date +%Y-%m-%d).log
```

#### 2. 性能日志

**查看慢请求**:

```bash
grep "durationMs" logs/app-$(date +%Y-%m-%d).log | \
  awk -F'durationMs":' '{print $2}' | \
  awk -F',' '{print $1}' | \
  sort -n | tail -10
```

**查看 Token 使用统计**:

```bash
grep "totalTokens" logs/app-$(date +%Y-%m-%d).log | \
  awk -F'totalTokens":' '{print $2}' | \
  awk -F',' '{print $1}' | \
  awk '{sum+=$1; count++} END {print "平均:", sum/count, "总计:", sum}'
```

### 监控指标

#### 关键指标

1. **调用成功率**:

```
成功调用数 / 总调用数 * 100%
```

2. **平均延迟**:

```
总延迟 / 调用次数
```

3. **Token 使用量**:

```
每日/每月的 token 消耗统计
```

4. **错误率**:

```
错误调用数 / 总调用数 * 100%
```

#### 设置告警

建议为以下情况设置告警：

- 错误率 > 5%
- 平均延迟 > 5 秒
- Token 使用量接近配额
- API 密钥即将过期

## 调试技巧

### 1. 启用详细日志

在开发环境中启用 DEBUG 级别日志：

```bash
export LOG_LEVEL=debug
```

### 2. 使用 TraceID 追踪请求

每个请求都有唯一的 TraceID，用于追踪整个调用链：

```bash
# 查找特定请求的所有日志
grep "traceId.*abc123" logs/app-$(date +%Y-%m-%d).log
```

### 3. 测试配置

使用 curl 测试配置是否正确：

```bash
# 测试非流式调用
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{id}/messages" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello",
    "options": {
      "modelName": "gpt-4"
    }
  }'

# 测试流式调用
curl -N -X POST "http://localhost:8080/api/v1/chat/sessions/{id}/messages/stream" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello",
    "options": {
      "modelName": "gpt-4"
    }
  }'
```

### 4. 检查数据库

直接查询数据库检查配置：

```sql
-- 查看所有配置
SELECT id, tenant_id, name, model, model_provider, is_enabled, is_deleted
FROM model_configurations
WHERE is_deleted = false;

-- 查看特定租户的配置
SELECT *
FROM model_configurations
WHERE tenant_id = '738dbb1f-83e6-4bf5-935c-f0498236440d'
  AND is_deleted = false;

-- 查看禁用的配置
SELECT name, model, model_provider
FROM model_configurations
WHERE is_enabled = false
  AND is_deleted = false;
```

## 获取帮助

### 1. 查看文档

- [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md)
- [配置指南](./CONFIGURATION_GUIDE.md)
- [迁移指南](./MIGRATION_GUIDE.md)

### 2. 查看日志

- 应用日志：`logs/app-{date}.log`
- 错误日志：过滤 ERROR 级别
- 性能日志：查看 durationMs 字段

### 3. 联系支持

如果问题仍未解决：

1. 收集以下信息：
   - 错误信息
   - 相关日志
   - 配置信息（脱敏后）
   - 复现步骤

2. 提交问题：
   - 内部支持渠道
   - GitHub Issues（如果是开源项目）

## 常见错误代码

| 错误代码 | 说明 | 解决方案 |
|---------|------|---------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 401 | 未授权 | 检查 Token 是否有效 |
| 403 | 权限不足 | 检查用户角色和权限 |
| 404 | 资源不存在 | 检查配置是否存在 |
| 429 | 速率限制 | 实施重试机制 |
| 500 | 服务器错误 | 查看服务器日志 |
| 502 | 网关错误 | 检查提供商服务状态 |
| 504 | 网关超时 | 增加超时时间或使用流式响应 |

## 常见问题解答 (FAQ)

### 配置相关

**Q1: 如何为新租户配置模型？**

A: 平台管理员可以通过以下步骤为新租户配置模型：

```bash
# 1. 创建租户（如果还没有）
curl -X POST "http://localhost:8080/api/v1/tenants" \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "新租户",
    "domain": "newtenant.com"
  }'

# 2. 为租户创建模型配置
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "新租户的UUID",
    "name": "gpt-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-..."
  }'
```

**Q2: 租户管理员可以配置多个模型吗？**

A: 可以。租户管理员可以为自己的租户配置多个模型，每个模型使用不同的名称标识。

```bash
# 配置第一个模型
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer TENANT_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpt-4",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-..."
  }'

# 配置第二个模型
curl -X POST "http://localhost:8080/api/v1/model-configurations" \
  -H "Authorization: Bearer TENANT_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "claude-3",
    "model": "claude-3-opus-20240229",
    "modelProvider": "anthropic",
    "apiKey": "sk-ant-..."
  }'
```

**Q3: 如何更换 API 密钥？**

A: 可以通过更新配置接口更换 API 密钥：

```bash
curl -X PUT "http://localhost:8080/api/v1/model-configurations/{id}" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "apiKey": "新的API密钥"
  }'
```

更新后，系统会自动清除缓存，下次调用时使用新密钥。

**Q4: 配置更新后多久生效？**

A: 立即生效。配置更新后会自动清除缓存，下次 API 调用时会使用新配置初始化。无需重启服务。

**Q5: 如何临时禁用某个模型？**

A: 使用状态更新接口：

```bash
curl -X PATCH "http://localhost:8080/api/v1/model-configurations/{id}/status" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "disabled"}'
```

禁用后，该模型将无法被使用，但配置仍然保留。需要时可以重新启用。

### 使用相关

**Q6: 如何在 API 调用中指定使用哪个模型？**

A: 在发送消息时，通过 `options.modelName` 参数指定：

```bash
curl -X POST "http://localhost:8080/api/v1/chat/sessions/{id}/messages" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好",
    "options": {
      "modelName": "gpt-4",
      "temperature": 0.7
    }
  }'
```

**Q7: 如果不指定模型名称会怎样？**

A: 系统会使用会话的默认模型。如果会话没有设置默认模型，会返回错误。

**Q8: 可以在同一个会话中切换不同的模型吗？**

A: 可以。每次发送消息时都可以指定不同的模型。但要注意：

- 不同模型的上下文是独立的
- 切换模型可能影响对话连贯性
- 建议在新会话中使用新模型

**Q9: 流式响应和非流式响应有什么区别？**

A:

- **非流式响应**：等待模型生成完整内容后一次性返回，适合短文本
- **流式响应**：实时返回生成的内容片段，适合长文本，用户体验更好

```bash
# 非流式
POST /api/v1/chat/sessions/{id}/messages

# 流式
POST /api/v1/chat/sessions/{id}/messages/stream
```

**Q10: 如何控制生成内容的长度？**

A: 通过 `maxTokens` 参数控制：

```json
{
  "message": "写一篇文章",
  "options": {
    "modelName": "gpt-4",
    "maxTokens": 1000  // 限制最多生成1000个token
  }
}
```

### 提供商相关

**Q11: Azure OpenAI 和 OpenAI 有什么区别？**

A:

- **OpenAI**：直接使用 OpenAI 的公共 API，需要 OpenAI 账户
- **Azure OpenAI**：使用微软 Azure 托管的 OpenAI 服务，需要 Azure 账户
- Azure OpenAI 提供企业级 SLA、数据隐私保护和区域部署

**Q12: 百炼支持哪些模型？**

A: 百炼支持通义千问系列模型：

- `qwen-turbo` - 快速版本，适合简单任务
- `qwen-plus` - 平衡版本，性价比高
- `qwen-max` - 最强版本，适合复杂任务
- `qwen-max-longcontext` - 长上下文版本

**Q13: 可以使用本地部署的模型吗？**

A: 可以。使用 `custom_openai` 提供商类型，配置本地服务的 URL：

```json
{
  "name": "本地模型",
  "model": "llama-2-70b",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:8000/v1",
  "apiKey": "not-needed"
}
```

前提是本地服务必须兼容 OpenAI API 规范。

**Q14: 不同提供商的 API 密钥格式有什么要求？**

A:

- **Google AI**: 以 `AIzaSy` 开头
- **OpenAI**: 以 `sk-` 开头
- **Azure OpenAI**: 32 位十六进制字符串
- **百炼**: 以 `sk-` 开头
- **Anthropic**: 以 `sk-ant-` 开头

**Q15: 如何选择合适的模型？**

A: 根据任务类型选择：

| 任务类型 | 推荐模型 | 原因 |
|---------|---------|------|
| 简单问答 | GPT-3.5 Turbo, Qwen Turbo | 快速、成本低 |
| 复杂推理 | GPT-4, Claude 3 Opus | 能力强、准确度高 |
| 代码生成 | GPT-4, Claude 3 | 代码理解能力强 |
| 中文任务 | Qwen Plus/Max | 中文理解能力强 |
| 长文本 | Claude 3, Qwen Max Longcontext | 支持长上下文 |

### 性能相关

**Q16: 为什么第一次调用比较慢？**

A: 第一次调用需要初始化 Genkit 实例和建立连接，后续调用会使用缓存的实例，速度会快很多。

**Q17: 如何提高响应速度？**

A:

1. 使用流式响应，提升用户体验
2. 选择快速模型（如 GPT-3.5 Turbo）
3. 优化提示词，减少不必要的上下文
4. 限制 maxTokens，避免生成过长内容
5. 使用缓存机制

**Q18: Token 使用量如何计算？**

A: Token 使用量 = 输入 Token + 输出 Token

- 输入 Token：提示词和上下文的 token 数
- 输出 Token：模型生成内容的 token 数
- 中文：约 1.5-2 个字符 = 1 token
- 英文：约 4 个字符 = 1 token

**Q19: 如何监控 Token 使用量？**

A: 查看日志中的 token 统计：

```bash
# 查看今天的 token 使用
grep "totalTokens" logs/app-$(date +%Y-%m-%d).log | \
  awk -F'totalTokens":' '{print $2}' | \
  awk -F',' '{print $1}' | \
  awk '{sum+=$1} END {print "今日总计:", sum, "tokens"}'
```

**Q20: 如何优化成本？**

A:

1. 根据任务选择合适的模型（不要总用最贵的）
2. 合理设置 maxTokens，避免浪费
3. 优化提示词，减少输入 token
4. 使用缓存减少重复调用
5. 定期审查和优化使用模式

### 安全相关

**Q21: API 密钥会被记录在日志中吗？**

A: 不会。系统会自动脱敏 API 密钥，日志中只显示前4位和后4位，中间用 `****` 替代。

**Q22: 如何保护 API 密钥安全？**

A:

1. ✅ 定期轮换密钥
2. ✅ 为不同环境使用不同密钥
3. ✅ 限制密钥权限
4. ✅ 监控密钥使用情况
5. ❌ 不要在代码中硬编码
6. ❌ 不要在公共仓库中提交
7. ❌ 不要在日志中记录完整密钥

**Q23: 如果 API 密钥泄露了怎么办？**

A: 立即采取以下措施：

1. 在提供商控制台撤销泄露的密钥
2. 生成新的 API 密钥
3. 在系统中更新为新密钥
4. 检查是否有异常使用
5. 审查访问日志
6. 加强密钥管理流程

**Q24: 不同租户可以使用相同的 API 密钥吗？**

A: 技术上可以，但强烈不推荐。建议每个租户使用独立的 API 密钥，原因：

- 便于成本追踪和分摊
- 便于权限管理
- 降低安全风险
- 便于问题排查

**Q25: 如何限制用户的 API 调用频率？**

A: 系统层面可以实施：

1. 速率限制中间件
2. 租户级别的配额管理
3. 用户级别的调用限制
4. 基于 Token 的使用限制

### 故障处理

**Q26: 遇到 "rate limit exceeded" 错误怎么办？**

A:

1. 等待一段时间后重试
2. 实施指数退避重试策略
3. 检查配额使用情况
4. 考虑升级账户等级
5. 分散请求到不同时间段

**Q27: 模型响应质量不好怎么办？**

A:

1. 调整 temperature 参数（降低可提高确定性）
2. 优化提示词，提供更清晰的指令
3. 增加示例（few-shot learning）
4. 尝试不同的模型
5. 增加上下文信息

**Q28: 如何处理超时问题？**

A:

1. 使用流式响应
2. 增加超时时间
3. 减少输入长度
4. 降低 maxTokens
5. 检查网络连接

**Q29: 配置更新后还是使用旧配置怎么办？**

A: 可能是缓存问题，尝试：

1. 等待几秒钟（缓存会自动更新）
2. 重启应用（强制清除所有缓存）
3. 检查配置是否真的更新成功
4. 查看日志确认使用的配置

**Q30: 如何回滚到之前的配置？**

A: 系统支持配置历史记录：

1. 查看配置变更历史
2. 找到之前的配置版本
3. 创建新配置或更新现有配置
4. 测试确认配置正确

## 完整配置示例

### 示例 1: 多模型混合配置

适用场景：企业需要使用多个提供商的模型，根据不同任务选择最合适的模型。

```json
{
  "configurations": [
    {
      "name": "gpt-4-general",
      "model": "gpt-4",
      "modelProvider": "openai",
      "apiKey": "sk-openai-key-here",
      "queryParams": {
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
      },
      "description": "通用任务使用"
    },
    {
      "name": "gpt-35-fast",
      "model": "gpt-3.5-turbo",
      "modelProvider": "openai",
      "apiKey": "sk-openai-key-here",
      "queryParams": {
        "defaultTemperature": 0.5,
        "defaultMaxTokens": 1024
      },
      "description": "简单快速任务"
    },
    {
      "name": "claude-3-opus",
      "model": "claude-3-opus-20240229",
      "modelProvider": "anthropic",
      "apiKey": "sk-ant-anthropic-key-here",
      "queryParams": {
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 4096
      },
      "description": "复杂推理任务"
    },
    {
      "name": "qwen-chinese",
      "model": "qwen-max",
      "modelProvider": "bianlian",
      "apiKey": "sk-bailian-key-here",
      "queryParams": {
        "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
      },
      "description": "中文任务专用"
    }
  ]
}
```

### 示例 2: Azure OpenAI 企业配置

适用场景：企业使用 Azure OpenAI 服务，需要配置多个部署。

```json
{
  "configurations": [
    {
      "name": "azure-gpt4-prod",
      "model": "gpt-4",
      "modelProvider": "azureopenai",
      "apiKey": "azure-api-key-here",
      "queryParams": {
        "azureEndpoint": "https://your-prod-resource.openai.azure.com",
        "azureDeployment": "gpt-4-prod",
        "azureApiVersion": "2024-02-15-preview",
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 8192
      },
      "description": "生产环境 GPT-4"
    },
    {
      "name": "azure-gpt35-dev",
      "model": "gpt-3.5-turbo",
      "modelProvider": "azureopenai",
      "apiKey": "azure-api-key-here",
      "queryParams": {
        "azureEndpoint": "https://your-dev-resource.openai.azure.com",
        "azureDeployment": "gpt-35-turbo-dev",
        "azureApiVersion": "2024-02-15-preview",
        "defaultTemperature": 0.5,
        "defaultMaxTokens": 4096
      },
      "description": "开发环境 GPT-3.5"
    }
  ]
}
```

### 示例 3: 本地模型配置

适用场景：使用本地部署的开源模型（如 Ollama、vLLM）。

```json
{
  "configurations": [
    {
      "name": "local-llama2",
      "model": "llama2-70b",
      "modelProvider": "custom_openai",
      "baseUrl": "http://localhost:11434/v1",
      "apiKey": "not-needed",
      "queryParams": {
        "defaultTemperature": 0.7,
        "defaultMaxTokens": 2048
      },
      "description": "本地 Llama 2 模型"
    },
    {
      "name": "local-mistral",
      "model": "mistral-7b",
      "modelProvider": "custom_openai",
      "baseUrl": "http://localhost:8000/v1",
      "apiKey": "not-needed",
      "queryParams": {
        "defaultTemperature": 0.8,
        "defaultMaxTokens": 4096
      },
      "description": "本地 Mistral 模型"
    }
  ]
}
```

### 示例 4: 多租户配置

适用场景：SaaS 平台，不同租户使用不同的模型配置。

```json
{
  "tenant_a": {
    "configurations": [
      {
        "name": "gpt-4",
        "model": "gpt-4",
        "modelProvider": "openai",
        "apiKey": "sk-tenant-a-key",
        "queryParams": {
          "defaultTemperature": 0.7,
          "defaultMaxTokens": 2048
        }
      }
    ]
  },
  "tenant_b": {
    "configurations": [
      {
        "name": "claude-3",
        "model": "claude-3-sonnet-20240229",
        "modelProvider": "anthropic",
        "apiKey": "sk-ant-tenant-b-key",
        "queryParams": {
          "defaultTemperature": 0.7,
          "defaultMaxTokens": 4096
        }
      }
    ]
  },
  "tenant_c": {
    "configurations": [
      {
        "name": "qwen-max",
        "model": "qwen-max",
        "modelProvider": "bianlian",
        "apiKey": "sk-tenant-c-key",
        "queryParams": {
          "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
          "defaultTemperature": 0.7,
          "defaultMaxTokens": 2048
        }
      }
    ]
  }
}
```

### 示例 5: 开发/测试/生产环境配置

适用场景：不同环境使用不同的配置和密钥。

```json
{
  "development": {
    "configurations": [
      {
        "name": "dev-gpt35",
        "model": "gpt-3.5-turbo",
        "modelProvider": "openai",
        "apiKey": "sk-dev-key",
        "queryParams": {
          "defaultTemperature": 0.5,
          "defaultMaxTokens": 1024
        }
      }
    ]
  },
  "testing": {
    "configurations": [
      {
        "name": "test-gpt4",
        "model": "gpt-4",
        "modelProvider": "openai",
        "apiKey": "sk-test-key",
        "queryParams": {
          "defaultTemperature": 0.7,
          "defaultMaxTokens": 2048
        }
      }
    ]
  },
  "production": {
    "configurations": [
      {
        "name": "prod-azure-gpt4",
        "model": "gpt-4",
        "modelProvider": "azureopenai",
        "apiKey": "prod-azure-key",
        "queryParams": {
          "azureEndpoint": "https://prod-resource.openai.azure.com",
          "azureDeployment": "gpt-4-prod",
          "azureApiVersion": "2024-02-15-preview",
          "defaultTemperature": 0.7,
          "defaultMaxTokens": 8192
        }
      }
    ]
  }
}
```

### 示例 6: 高级参数配置

适用场景：需要精细控制模型行为的场景。

```json
{
  "configurations": [
    {
      "name": "creative-writing",
      "model": "gpt-4",
      "modelProvider": "openai",
      "apiKey": "sk-key-here",
      "queryParams": {
        "defaultTemperature": 0.9,
        "defaultMaxTokens": 4096,
        "topP": 0.95,
        "frequencyPenalty": 0.5,
        "presencePenalty": 0.5
      },
      "description": "创意写作，高随机性"
    },
    {
      "name": "factual-qa",
      "model": "gpt-4",
      "modelProvider": "openai",
      "apiKey": "sk-key-here",
      "queryParams": {
        "defaultTemperature": 0.1,
        "defaultMaxTokens": 1024,
        "topP": 0.9,
        "frequencyPenalty": 0.0,
        "presencePenalty": 0.0
      },
      "description": "事实问答，低随机性"
    },
    {
      "name": "code-generation",
      "model": "gpt-4",
      "modelProvider": "openai",
      "apiKey": "sk-key-here",
      "queryParams": {
        "defaultTemperature": 0.2,
        "defaultMaxTokens": 2048,
        "topP": 0.95,
        "frequencyPenalty": 0.0,
        "presencePenalty": 0.0
      },
      "description": "代码生成，中低随机性"
    }
  ]
}
```

## 预防措施

### 1. 定期维护

- ✅ 定期检查 API 密钥有效性
- ✅ 监控配额使用情况
- ✅ 更新过期的配置
- ✅ 清理无用的配置

### 2. 监控告警

- ✅ 设置错误率告警
- ✅ 设置延迟告警
- ✅ 设置配额告警
- ✅ 定期查看监控面板

### 3. 备份恢复

- ✅ 定期备份配置
- ✅ 测试恢复流程
- ✅ 记录配置变更
- ✅ 保留历史版本

### 4. 文档更新

- ✅ 记录常见问题
- ✅ 更新解决方案
- ✅ 分享最佳实践
- ✅ 培训团队成员
