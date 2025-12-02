# Genkit 多模型提供商使用指南

## 概述

本系统支持通过 Google Genkit 框架统一调用多个 AI 模型提供商，包括：

- **Google AI (Gemini)** - Google 的 Gemini 系列模型
- **OpenAI** - GPT 系列模型
- **Azure OpenAI** - 微软 Azure 托管的 OpenAI 模型
- **阿里云百炼** - 阿里云的通义千问系列模型
- **Anthropic** - Claude 系列模型
- **自定义 OpenAI 兼容服务** - 任何兼容 OpenAI API 的服务

所有模型配置通过数据库 `model_configurations` 表进行管理，支持多租户隔离。

## 核心特性

### 1. 多租户支持

每个租户可以配置多个模型，不同租户的配置完全隔离：

- 租户管理员只能管理自己租户的模型配置
- 平台管理员可以管理所有租户的模型配置
- 每个租户可以使用不同的 API 密钥和配置

### 2. 动态配置

- 配置存储在数据库中，无需重启服务即可生效
- 支持启用/禁用模型
- 支持配置更新和删除
- 自动缓存机制提高性能

### 3. 统一接口

- 所有提供商通过相同的 API 接口调用
- 统一的请求和响应格式
- 统一的错误处理
- 统一的日志和监控

### 4. 流式支持

- 所有提供商都支持流式响应
- 实时返回生成内容
- SSE (Server-Sent Events) 格式
- 自动处理连接中断

## 快速开始

### 1. 配置模型

#### 创建模型配置

**API 端点**: `POST /api/v1/model-configurations`

**权限要求**:

- 平台管理员：可以为任何租户创建配置
- 租户管理员：只能为自己的租户创建配置

**请求示例**:

```json
{
  "name": "GPT-4 配置",
  "model": "gpt-4",
  "modelProvider": "openai",
  "apiKey": "sk-your-api-key-here"
}
```

**响应示例**:

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "name": "GPT-4 配置",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-yo****here",
    "isEnabled": true,
    "createdAt": "2025-12-01T10:00:00Z"
  }
}
```

### 2. 使用模型

#### 发送消息

**API 端点**: `POST /api/v1/chat/sessions/{sessionId}/messages`

**请求示例**:

```json
{
  "message": "你好，请介绍一下自己",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.7,
    "maxTokens": 2000
  }
}
```

**说明**:

- `modelName`: 指定要使用的模型名称（对应配置中的 `name` 字段）
- 如果不指定 `modelName`，将使用会话的默认模型
- 系统会根据当前租户 ID 和模型名称自动查询配置

#### 流式响应

**API 端点**: `POST /api/v1/chat/sessions/{sessionId}/messages/stream`

**请求示例**:

```json
{
  "message": "写一首关于春天的诗",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.8
  }
}
```

**响应格式** (SSE):

```
data: {"event":"message","data":{"delta":"春"}}

data: {"event":"message","data":{"delta":"天"}}

data: {"event":"message","data":{"delta":"来"}}

data: {"event":"done","data":{"usage":{"promptTokens":10,"completionTokens":50,"totalTokens":60}}}
```

## 提供商配置指南

### Google AI (Gemini)

**提供商标识**: `googlegenai`

**配置示例**:

```json
{
  "name": "Gemini Pro",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "apiKey": "your-google-api-key"
}
```

**支持的模型**:

- `gemini-1.5-pro` - 最新的 Pro 版本
- `gemini-1.5-flash` - 快速版本
- `gemini-pro` - 标准版本

**获取 API 密钥**:

1. 访问 [Google AI Studio](https://makersuite.google.com/app/apikey)
2. 创建新的 API 密钥
3. 复制密钥并保存

### OpenAI

**提供商标识**: `openai`

**配置示例**:

```json
{
  "name": "GPT-4",
  "model": "gpt-4",
  "modelProvider": "openai",
  "apiKey": "sk-your-openai-api-key"
}
```

**支持的模型**:

- `gpt-4` - GPT-4 标准版
- `gpt-4-turbo` - GPT-4 Turbo 版本
- `gpt-3.5-turbo` - GPT-3.5 Turbo 版本

**获取 API 密钥**:

1. 访问 [OpenAI Platform](https://platform.openai.com/api-keys)
2. 创建新的 API 密钥
3. 复制密钥并保存

### Azure OpenAI

**提供商标识**: `azureopenai`

**配置示例**:

```json
{
  "name": "Azure GPT-4",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

**必需的配置字段**:

- `azureEndpoint`: Azure OpenAI 资源的端点 URL
- `azureDeployment`: 部署名称
- `azureApiVersion`: API 版本

**获取配置信息**:

1. 登录 [Azure Portal](https://portal.azure.com)
2. 找到你的 Azure OpenAI 资源
3. 在"密钥和终结点"页面获取：
   - 端点 URL (`azureEndpoint`)
   - API 密钥 (`apiKey`)
4. 在"模型部署"页面获取部署名称 (`azureDeployment`)

### 阿里云百炼

**提供商标识**: `bianlian`

**配置示例**:

```json
{
  "name": "通义千问",
  "model": "qwen-turbo",
  "modelProvider": "bianlian",
  "apiKey": "sk-your-dashscope-api-key",
  "queryParams": {
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "bailianWorkspace": "default"
  }
}
```

**支持的模型**:

- `qwen-turbo` - 通义千问 Turbo 版本
- `qwen-plus` - 通义千问 Plus 版本
- `qwen-max` - 通义千问 Max 版本

**配置说明**:

- `bailianEndpoint`: 百炼 API 端点（可选，默认使用北京地域）
- `bailianWorkspace`: 工作空间名称（可选，默认为 "default"）

**获取 API 密钥**:

1. 访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)
2. 创建 API Key
3. 复制密钥并保存

### Anthropic (Claude)

**提供商标识**: `anthropic`

**配置示例**:

```json
{
  "name": "Claude 3 Opus",
  "model": "claude-3-opus-20240229",
  "modelProvider": "anthropic",
  "apiKey": "sk-ant-your-api-key"
}
```

**支持的模型**:

- `claude-3-opus-20240229` - Claude 3 Opus
- `claude-3-sonnet-20240229` - Claude 3 Sonnet
- `claude-3-haiku-20240307` - Claude 3 Haiku

**获取 API 密钥**:

1. 访问 [Anthropic Console](https://console.anthropic.com/)
2. 创建新的 API 密钥
3. 复制密钥并保存

### 自定义 OpenAI 兼容服务

**提供商标识**: `custom_openai`

**配置示例**:

```json
{
  "name": "自定义模型",
  "model": "custom-model-name",
  "modelProvider": "custom_openai",
  "baseUrl": "https://your-custom-api.com/v1",
  "apiKey": "your-custom-api-key"
}
```

**说明**:

- 适用于任何兼容 OpenAI API 规范的服务
- 必须提供 `baseUrl` 字段
- 支持本地部署的模型服务

## 高级配置

### QueryParams 配置

`queryParams` 字段用于存储提供商特定的配置参数，格式为 JSON 字符串。

**示例**:

```json
{
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048,
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

**支持的通用参数**:

- `defaultTemperature`: 默认温度值 (0-2)
- `defaultMaxTokens`: 默认最大 token 数

**Azure OpenAI 特定参数**:

- `azureEndpoint`: Azure 端点 URL
- `azureDeployment`: 部署名称
- `azureApiVersion`: API 版本

**百炼特定参数**:

- `bailianEndpoint`: 百炼端点 URL
- `bailianWorkspace`: 工作空间名称

### 启用/禁用模型

**API 端点**: `PATCH /api/v1/model-configurations/{id}/status`

**请求示例**:

```json
{
  "status": "disabled"
}
```

**说明**:

- `enabled`: 启用模型
- `disabled`: 禁用模型
- 禁用后，该模型将无法被使用

### 更新配置

**API 端点**: `PUT /api/v1/model-configurations/{id}`

**请求示例**:

```json
{
  "name": "GPT-4 Turbo 配置",
  "model": "gpt-4-turbo",
  "apiKey": "sk-new-api-key-here"
}
```

**说明**:

- 所有字段都是可选的
- 只更新提供的字段
- 更新后会自动清除缓存

### 删除配置

**API 端点**: `DELETE /api/v1/model-configurations/{id}`

**说明**:

- 执行软删除，数据不会真正删除
- 删除后该配置将无法使用
- 只有平台管理员可以删除配置

## 最佳实践

### 1. API 密钥管理

- ✅ 定期轮换 API 密钥
- ✅ 为不同环境使用不同的密钥
- ✅ 不要在代码中硬编码密钥
- ✅ 使用环境变量或密钥管理服务
- ❌ 不要在日志中记录完整的 API 密钥

### 2. 模型选择

- ✅ 根据任务复杂度选择合适的模型
- ✅ 简单任务使用快速模型（如 gpt-3.5-turbo）
- ✅ 复杂任务使用强大模型（如 gpt-4）
- ✅ 考虑成本和性能的平衡

### 3. 参数调优

- ✅ 根据场景调整 temperature
  - 创意任务：0.7-1.0
  - 事实性任务：0.0-0.3
- ✅ 合理设置 maxTokens 避免浪费
- ✅ 使用流式响应提升用户体验

### 4. 错误处理

- ✅ 实现重试机制
- ✅ 处理速率限制
- ✅ 提供友好的错误提示
- ✅ 记录详细的错误日志

### 5. 性能优化

- ✅ 利用系统的缓存机制
- ✅ 避免频繁切换模型
- ✅ 批量处理请求
- ✅ 监控 API 调用延迟

## 监控和日志

### 日志记录

系统会自动记录以下信息：

- 模型选择和初始化
- API 调用开始和结束
- Token 使用统计
- 错误和异常
- 性能指标（延迟、TTFB 等）

### 日志示例

```json
{
  "level": "info",
  "time": "2025-12-01T10:00:00Z",
  "message": "生成内容成功",
  "traceId": "abc123",
  "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
  "modelName": "gpt-4",
  "model": "gpt-4",
  "provider": "openai",
  "durationMs": 1500,
  "promptTokens": 10,
  "completionTokens": 50,
  "totalTokens": 60
}
```

### 监控指标

建议监控以下指标：

- **调用成功率**: 成功调用 / 总调用数
- **平均延迟**: API 调用的平均响应时间
- **Token 使用量**: 每日/每月的 token 消耗
- **错误率**: 错误调用 / 总调用数
- **TTFB**: 流式响应的首字节时间

## 常见问题

### Q: 如何切换模型？

A: 在发送消息时，通过 `options.modelName` 参数指定要使用的模型名称。系统会自动查询对应的配置并使用该模型。

### Q: 配置更新后需要重启服务吗？

A: 不需要。配置存储在数据库中，更新后会自动清除缓存，下次调用时会使用新配置。

### Q: 如何查看可用的模型？

A: 调用 `GET /api/v1/model-configurations` 接口查看当前租户的所有模型配置。

### Q: 不同租户可以使用相同的 API 密钥吗？

A: 可以，但不推荐。建议每个租户使用独立的 API 密钥，便于管理和成本追踪。

### Q: 如何处理 API 密钥泄露？

A: 立即在提供商控制台撤销泄露的密钥，然后在系统中更新为新的密钥。

### Q: 支持自定义模型参数吗？

A: 支持。可以在 `queryParams` 中配置默认参数，也可以在每次调用时通过 `options` 参数覆盖。

### Q: 如何优化成本？

A:

1. 根据任务选择合适的模型
2. 合理设置 maxTokens
3. 使用缓存减少重复调用
4. 监控 token 使用量
5. 定期审查和优化提示词

## 下一步

- 查看 [配置指南](./CONFIGURATION_GUIDE.md) 了解详细的配置说明
- 查看 [故障排查指南](./TROUBLESHOOTING.md) 解决常见问题
- 查看 [迁移指南](./MIGRATION_GUIDE.md) 了解如何从单提供商迁移

## 相关资源

- [Google Genkit 文档](https://firebase.google.com/docs/genkit)
- [OpenAI API 文档](https://platform.openai.com/docs)
- [Azure OpenAI 文档](https://learn.microsoft.com/azure/ai-services/openai/)
- [阿里云百炼文档](https://help.aliyun.com/zh/model-studio/)
- [Anthropic API 文档](https://docs.anthropic.com/)
