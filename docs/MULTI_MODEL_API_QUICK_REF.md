# 多模型支持 API 快速参考

## 概述

系统现在支持多个 AI 模型提供商，包括：

- **Google AI** (Gemini)
- **Azure OpenAI** (GPT-4, GPT-3.5-turbo 等)
- **阿里云百炼** (Qwen 系列)

用户可以通过 API 请求中的 `modelName` 参数动态选择要使用的模型。

## 核心概念

### 模型配置

每个租户可以在 `model_configurations` 表中配置多个 AI 模型。系统会根据：

- **租户ID** (从 JWT token 中获取)
- **模型名称** (从 API 请求中获取)

来查询对应的模型配置并使用该模型。

### 向后兼容

- `modelName` 参数是**可选的**
- 如果不指定 `modelName`，系统将使用会话的默认模型
- 现有的 API 调用无需修改即可继续工作

## API 使用示例

### 1. 发送消息（使用默认模型）

```bash
POST /api/v1/chat/sessions/{sessionId}/messages
Content-Type: application/json
Authorization: Bearer {token}

{
  "message": "你好，请介绍一下你自己"
}
```

### 2. 发送消息（指定使用 GPT-4）

```bash
POST /api/v1/chat/sessions/{sessionId}/messages
Content-Type: application/json
Authorization: Bearer {token}

{
  "message": "你好，请介绍一下你自己",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.8,
    "maxTokens": 1000
  }
}
```

### 3. 发送消息（指定使用阿里云百炼）

```bash
POST /api/v1/chat/sessions/{sessionId}/messages
Content-Type: application/json
Authorization: Bearer {token}

{
  "message": "你好，请介绍一下你自己",
  "options": {
    "modelName": "qwen-turbo",
    "temperature": 0.7
  }
}
```

### 4. 创建会话（使用 Azure OpenAI）

```bash
POST /api/v1/chat/sessions
Content-Type: application/json
Authorization: Bearer {token}

{
  "title": "Azure GPT-4 会话",
  "modelName": "gpt-4",
  "systemPrompt": "你是一个专业的编程助手",
  "temperature": 0.7
}
```

### 5. 更新会话模型

```bash
PUT /api/v1/chat/sessions/{sessionId}
Content-Type: application/json
Authorization: Bearer {token}

{
  "modelName": "gpt-4-turbo",
  "temperature": 0.8
}
```

### 6. 流式对话（指定模型）

```bash
POST /api/v1/chat/sessions/{sessionId}/messages/stream
Content-Type: application/json
Authorization: Bearer {token}

{
  "message": "请写一个 Python 快速排序的实现",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.5,
    "maxTokens": 2000
  }
}
```

## ChatOptions 参数说明

### modelName (可选)

- **类型**: string
- **说明**: 指定要使用的 AI 模型名称
- **示例**: `"gpt-4"`, `"gemini-pro"`, `"qwen-turbo"`
- **验证**: 长度 1-128 字符
- **默认**: 使用会话的默认模型

系统会根据当前租户ID和模型名称从 `model_configurations` 表中查询配置。

### temperature (可选)

- **类型**: number
- **说明**: 控制生成文本的随机性
- **范围**: 0.0 - 2.0
- **示例**: `0.7`
- **默认**: 使用模型配置中的默认值

值越高，输出越随机；值越低，输出越确定。

### maxTokens (可选)

- **类型**: integer
- **说明**: 生成内容的最大 token 数量
- **示例**: `2048`
- **默认**: 使用模型配置中的默认值

实际生成的 token 数可能少于此值。

### topP (可选)

- **类型**: number
- **说明**: 核采样参数，控制生成文本的多样性
- **范围**: 0.0 - 1.0
- **示例**: `0.9`
- **默认**: 使用模型配置中的默认值

值越小，输出越集中；值越大，输出越多样。

### topK (可选)

- **类型**: integer
- **说明**: 限制每步采样时考虑的 token 数量
- **示例**: `40`
- **默认**: 使用模型配置中的默认值

值越小，输出越集中。

## 响应格式

### 非流式响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "sessionId": "550e8400-e29b-41d4-a716-446655440000",
    "message": "你好！我是一个 AI 助手...",
    "model": "gpt-4",
    "usage": {
      "promptTokens": 10,
      "completionTokens": 50,
      "totalTokens": 60
    }
  }
}
```

### 流式响应 (SSE)

```
data: {"event":"message","data":{"content":"你好"}}

data: {"event":"message","data":{"content":"！"}}

data: {"event":"message","data":{"content":"我是"}}

data: {"event":"done","data":{"usage":{"promptTokens":10,"completionTokens":50,"totalTokens":60}}}
```

## 错误处理

### 模型配置不存在

```json
{
  "code": 404,
  "message": "模型配置不存在: gpt-4",
  "data": null
}
```

**解决方案**: 确保在 `model_configurations` 表中为当前租户配置了该模型。

### 模型已禁用

```json
{
  "code": 400,
  "message": "模型已禁用: gpt-4",
  "data": null
}
```

**解决方案**: 联系管理员启用该模型配置。

### 无效的模型名称

```json
{
  "code": 400,
  "message": "验证失败: modelName 长度必须在 1-128 之间",
  "data": null
}
```

**解决方案**: 检查 `modelName` 参数的值是否符合要求。

### 权限不足

```json
{
  "code": 403,
  "message": "权限不足：无法访问其他租户的模型配置",
  "data": null
}
```

**解决方案**: 确保使用正确的租户凭证。

## 最佳实践

### 1. 模型选择

- **通用对话**: 使用 `gemini-pro` 或 `gpt-3.5-turbo`（成本低，速度快）
- **复杂任务**: 使用 `gpt-4` 或 `gpt-4-turbo`（质量高，推理能力强）
- **中文场景**: 使用 `qwen-turbo` 或 `qwen-plus`（中文理解能力强）

### 2. 参数调优

- **创意写作**: `temperature: 0.8-1.0`
- **代码生成**: `temperature: 0.3-0.5`
- **事实问答**: `temperature: 0.0-0.3`

### 3. Token 管理

- 设置合理的 `maxTokens` 值，避免不必要的成本
- 监控 `usage` 字段，了解 token 消耗情况
- 对于长文本生成，考虑使用流式接口

### 4. 错误处理

- 始终检查响应的 `code` 字段
- 对于 404 错误，提示用户配置模型
- 对于 500 错误，实施重试机制

### 5. 性能优化

- 使用流式接口提升用户体验
- 缓存常用的会话配置
- 合理设置超时时间

## 配置模型

### 管理员配置模型

管理员可以通过以下 API 为租户配置模型：

```bash
POST /api/v1/model-configurations
Content-Type: application/json
Authorization: Bearer {admin_token}

{
  "tenantId": "550e8400-e29b-41d4-a716-446655440000",
  "modelName": "gpt-4",
  "providerType": "azure",
  "apiKey": "your-api-key",
  "configuration": {
    "model": "gpt-4",
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

### 查询可用模型

```bash
GET /api/v1/model-configurations
Authorization: Bearer {token}
```

响应：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "...",
        "modelName": "gpt-4",
        "providerType": "azure",
        "isEnabled": true,
        "createdAt": "2024-01-01T00:00:00Z"
      },
      {
        "id": "...",
        "modelName": "qwen-turbo",
        "providerType": "bailian",
        "isEnabled": true,
        "createdAt": "2024-01-01T00:00:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 20,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

## Swagger 文档

访问 Swagger UI 查看完整的 API 文档：

```
http://localhost:8080/swagger/index.html
```

在 Swagger UI 中可以：

- 查看所有 API 接口的详细文档
- 查看请求和响应的数据结构
- 在线测试 API 接口
- 查看参数的验证规则和示例

## 相关文档

- [需求文档](../.kiro/specs/genkit-multi-model-support/requirements.md)
- [设计文档](../.kiro/specs/genkit-multi-model-support/design.md)
- [Swagger 更新总结](../internal/model/SWAGGER_UPDATE_SUMMARY.md)
- [模型配置管理 API](./MODEL_CONFIGURATION_API.md)

## 技术支持

如有问题，请联系：

- 技术支持邮箱: <support@example.com>
- 开发团队: <dev@example.com>

## 更新日志

### 2024-11-28

- ✅ 添加 `modelName` 参数支持
- ✅ 更新 Swagger 文档
- ✅ 支持 Google AI、Azure OpenAI、阿里云百炼
- ✅ 保持向后兼容性
