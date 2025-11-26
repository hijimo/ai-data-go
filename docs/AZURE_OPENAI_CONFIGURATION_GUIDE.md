# Azure OpenAI 配置指南

## 概述

本指南介绍如何在系统中配置和使用 Azure OpenAI 服务。

## 前置条件

1. 拥有 Azure 账户
2. 已创建 Azure OpenAI 资源
3. 已部署至少一个模型（如 GPT-4）
4. 获取了 API Key 和 Endpoint

## 配置步骤

### 1. 获取 Azure OpenAI 信息

登录 Azure Portal，找到你的 OpenAI 资源，获取以下信息：

- **Endpoint**：资源的 endpoint URL
  - 格式：`https://{resource-name}.openai.azure.com`
  - 示例：`https://my-openai-resource.openai.azure.com`

- **API Key**：在"Keys and Endpoint"页面获取
  - 示例：`1234567890abcdef1234567890abcdef`

- **Deployment Name**：你部署的模型名称
  - 示例：`gpt-4-deployment`、`gpt-35-turbo`

- **API Version**：使用的 API 版本
  - 推荐：`2024-02-15-preview`
  - 稳定版：`2023-12-01`

### 2. 在数据库中创建模型配置

使用 Model Configuration API 创建配置：

```bash
curl -X POST http://localhost:8080/api/v1/model-configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "tenantId": "your-tenant-id",
    "modelName": "azure-gpt4",
    "modelProvider": "azureopenai",
    "apiKey": "your-azure-api-key",
    "queryParams": "{\"model\":\"gpt-4\",\"azureEndpoint\":\"https://my-resource.openai.azure.com\",\"azureDeployment\":\"gpt-4-deployment\",\"azureApiVersion\":\"2024-02-15-preview\",\"defaultTemperature\":0.7,\"defaultMaxTokens\":2048}"
  }'
```

### 3. 配置字段说明

#### 必需字段

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `tenantId` | UUID | 租户ID | `738dbb1f-83e6-4bf5-935c-f0498236440d` |
| `modelName` | String | 模型名称（自定义） | `azure-gpt4` |
| `modelProvider` | String | 提供商类型 | `azureopenai` |
| `apiKey` | String | Azure API Key | `1234567890abcdef...` |

#### queryParams 字段（JSON 格式）

| 字段 | 类型 | 必需 | 说明 | 示例 |
|------|------|------|------|------|
| `model` | String | 是 | 模型标识 | `gpt-4` |
| `azureEndpoint` | String | 是 | Azure Endpoint | `https://my-resource.openai.azure.com` |
| `azureDeployment` | String | 是 | 部署名称 | `gpt-4-deployment` |
| `azureApiVersion` | String | 是 | API 版本 | `2024-02-15-preview` |
| `defaultTemperature` | Number | 否 | 默认温度值 | `0.7` |
| `defaultMaxTokens` | Number | 否 | 默认最大 token 数 | `2048` |

### 4. 配置示例

#### GPT-4 配置

```json
{
  "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
  "modelName": "azure-gpt4",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key",
  "queryParams": "{\"model\":\"gpt-4\",\"azureEndpoint\":\"https://my-resource.openai.azure.com\",\"azureDeployment\":\"gpt-4-deployment\",\"azureApiVersion\":\"2024-02-15-preview\",\"defaultTemperature\":0.7,\"defaultMaxTokens\":2048}"
}
```

#### GPT-3.5 Turbo 配置

```json
{
  "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
  "modelName": "azure-gpt35-turbo",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key",
  "queryParams": "{\"model\":\"gpt-3.5-turbo\",\"azureEndpoint\":\"https://my-resource.openai.azure.com\",\"azureDeployment\":\"gpt-35-turbo\",\"azureApiVersion\":\"2024-02-15-preview\",\"defaultTemperature\":0.8,\"defaultMaxTokens\":1024}"
}
```

## 使用方法

### 1. 在聊天接口中使用

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session-id}/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "message": "你好，请介绍一下自己",
    "options": {
      "modelName": "azure-gpt4"
    }
  }'
```

### 2. 流式调用

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session-id}/messages/stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": "请写一首关于春天的诗",
    "options": {
      "modelName": "azure-gpt4"
    }
  }'
```

## 技术实现细节

### BaseURL 构造

系统会自动构造 Azure OpenAI 的 BaseURL：

```
{azureEndpoint}/openai/deployments/{azureDeployment}?api-version={azureApiVersion}
```

示例：

```
https://my-resource.openai.azure.com/openai/deployments/gpt-4-deployment?api-version=2024-02-15-preview
```

### 模型名称映射

在 Genkit 内部，Azure OpenAI 使用 `openai/` 前缀：

- 配置中的 `model`: `gpt-4`
- Genkit 中的模型名: `openai/gpt-4`

### 认证方式

使用 `api-key` header 进行认证，与标准 OpenAI 的 `Authorization: Bearer` 不同。

## 常见问题

### Q1: 如何获取 Azure OpenAI 的 Endpoint？

**A**: 登录 Azure Portal → 找到你的 OpenAI 资源 → 在"Keys and Endpoint"页面查看。

### Q2: Deployment Name 和 Model Name 有什么区别？

**A**:

- **Deployment Name**: Azure 中部署的实例名称，可以自定义
- **Model Name**: 底层的模型类型（如 gpt-4, gpt-35-turbo）

在配置中：

- `azureDeployment` 使用 Deployment Name
- `model` 使用 Model Name

### Q3: 支持哪些 API 版本？

**A**: 推荐使用以下版本：

- 最新预览版：`2024-02-15-preview`
- 稳定版：`2023-12-01`

查看完整列表：[Azure OpenAI API Versions](https://learn.microsoft.com/azure/ai-services/openai/reference)

### Q4: 如何测试配置是否正确？

**A**: 使用以下步骤测试：

1. 创建配置后，查询配置确认保存成功
2. 发送一个简单的测试消息
3. 检查响应和日志

```bash
# 1. 查询配置
curl -X GET http://localhost:8080/api/v1/model-configurations/azure-gpt4 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 2. 发送测试消息
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session-id}/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "message": "测试",
    "options": {
      "modelName": "azure-gpt4"
    }
  }'
```

### Q5: 遇到 401 Unauthorized 错误怎么办？

**A**: 检查以下几点：

1. API Key 是否正确
2. API Key 是否已过期
3. Azure 资源是否已启用
4. 是否有足够的配额

### Q6: 遇到 404 Not Found 错误怎么办？

**A**: 检查以下几点：

1. Endpoint URL 是否正确
2. Deployment Name 是否正确
3. API Version 是否正确
4. 部署是否已完成

### Q7: 如何切换到不同的 Azure 区域？

**A**: 只需更新 `azureEndpoint` 字段为新区域的 endpoint：

```json
{
  "azureEndpoint": "https://new-region-resource.openai.azure.com"
}
```

### Q8: 可以为同一个租户配置多个 Azure 模型吗？

**A**: 可以！只需使用不同的 `modelName`：

```json
// GPT-4 配置
{
  "modelName": "azure-gpt4",
  "azureDeployment": "gpt-4-deployment"
}

// GPT-3.5 配置
{
  "modelName": "azure-gpt35",
  "azureDeployment": "gpt-35-turbo"
}
```

## 最佳实践

### 1. 命名规范

建议使用清晰的命名规范：

- `azure-gpt4`: Azure 的 GPT-4
- `azure-gpt35-turbo`: Azure 的 GPT-3.5 Turbo
- `azure-gpt4-32k`: Azure 的 GPT-4 32K 版本

### 2. API 版本选择

- **生产环境**：使用稳定版本（如 `2023-12-01`）
- **测试环境**：可以使用预览版本（如 `2024-02-15-preview`）

### 3. 配额管理

- 监控 API 使用量
- 设置合理的 `defaultMaxTokens`
- 考虑使用多个部署分散负载

### 4. 安全性

- 定期轮换 API Key
- 使用环境变量存储敏感信息
- 限制 API Key 的访问权限

### 5. 性能优化

- 使用流式调用提升用户体验
- 合理设置 `temperature` 和 `maxTokens`
- 利用系统的缓存机制

## 故障排查

### 日志查看

查看应用日志以获取详细错误信息：

```bash
tail -f logs/app-$(date +%Y-%m-%d).log | grep -i azure
```

### 常见错误代码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 401 | 认证失败 | 检查 API Key |
| 404 | 资源不存在 | 检查 Endpoint 和 Deployment |
| 429 | 超过配额 | 等待或增加配额 |
| 500 | 服务器错误 | 检查 Azure 服务状态 |

### 调试模式

启用调试日志：

```bash
export LOG_LEVEL=debug
./server
```

## 参考资料

- [Azure OpenAI Service 文档](https://learn.microsoft.com/azure/ai-services/openai/)
- [Azure OpenAI REST API 参考](https://learn.microsoft.com/azure/ai-services/openai/reference)
- [Genkit 文档](https://genkit.dev/)
- [OpenAI Go SDK](https://github.com/openai/openai-go)

## 支持

如有问题，请联系技术支持或查看项目文档。
