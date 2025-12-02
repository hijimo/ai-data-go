# Genkit 多模型配置指南

## 概述

本文档详细说明如何配置和管理 Genkit 多模型支持系统。配置通过数据库 `model_configurations` 表进行管理，支持多租户隔离和动态更新。

## 数据库表结构

### model_configurations 表

```sql
CREATE TABLE model_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    model_provider VARCHAR(50) NOT NULL,
    base_url VARCHAR(500),
    api_key TEXT NOT NULL,
    query_params JSONB,
    is_enabled BOOLEAN DEFAULT true NOT NULL,
    is_deleted BOOLEAN DEFAULT false NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    updated_at TIMESTAMP,
    deleted_by UUID,
    deleted_at TIMESTAMP,
    
    CONSTRAINT uk_tenant_name UNIQUE (tenant_id, name, is_deleted),
    INDEX idx_tenant_provider (tenant_id, model_provider),
    INDEX idx_deleted (is_deleted),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);
```

### 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | UUID | 是 | 主键，自动生成 |
| tenant_id | UUID | 是 | 租户ID，外键关联 tenants 表 |
| name | VARCHAR(255) | 是 | 配置名称，租户内唯一 |
| model | VARCHAR(255) | 是 | 模型标识（如 gpt-4、gemini-pro） |
| model_provider | VARCHAR(50) | 是 | 提供商类型（见下文） |
| base_url | VARCHAR(500) | 否 | 自定义 API 端点 |
| api_key | TEXT | 是 | API 密钥（加密存储） |
| query_params | JSONB | 否 | 提供商特定配置（JSON 格式） |
| is_enabled | BOOLEAN | 是 | 是否启用，默认 true |
| is_deleted | BOOLEAN | 是 | 软删除标记，默认 false |
| created_by | UUID | 是 | 创建者用户ID |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_by | UUID | 否 | 更新者用户ID |
| updated_at | TIMESTAMP | 否 | 更新时间 |
| deleted_by | UUID | 否 | 删除者用户ID |
| deleted_at | TIMESTAMP | 否 | 删除时间 |

## 提供商类型

### 支持的提供商

| 提供商标识 | 说明 | 插件类型 |
|-----------|------|---------|
| `googlegenai` | Google AI (Gemini) | Google AI 官方插件 |
| `openai` | OpenAI | OpenAI 官方插件 |
| `azureopenai` | Azure OpenAI | OpenAI 插件 + Azure BaseURL |
| `bianlian` | 阿里云百炼 | OpenAI 插件 + 百炼兼容模式 |
| `anthropic` | Anthropic (Claude) | Anthropic 官方插件 |
| `custom_openai` | 自定义 OpenAI 兼容服务 | OpenAI 插件 + 自定义 BaseURL |

## 配置示例

### 1. Google AI (Gemini)

**基础配置**:

```json
{
  "name": "Gemini Pro",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "apiKey": "AIzaSy..."
}
```

**完整配置**:

```json
{
  "name": "Gemini Pro 高级配置",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "apiKey": "AIzaSy...",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

**支持的模型**:

- `gemini-1.5-pro` - 最新 Pro 版本
- `gemini-1.5-flash` - 快速版本
- `gemini-pro` - 标准版本
- `gemini-pro-vision` - 支持视觉的版本

### 2. OpenAI

**基础配置**:

```json
{
  "name": "GPT-4",
  "model": "gpt-4",
  "modelProvider": "openai",
  "apiKey": "sk-..."
}
```

**完整配置**:

```json
{
  "name": "GPT-4 Turbo",
  "model": "gpt-4-turbo-preview",
  "modelProvider": "openai",
  "apiKey": "sk-...",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 4096
  }
}
```

**支持的模型**:

- `gpt-4` - GPT-4 标准版
- `gpt-4-turbo` - GPT-4 Turbo
- `gpt-4-turbo-preview` - GPT-4 Turbo 预览版
- `gpt-3.5-turbo` - GPT-3.5 Turbo
- `gpt-3.5-turbo-16k` - GPT-3.5 Turbo 16K 上下文

### 3. Azure OpenAI

**基础配置**:

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

**完整配置**:

```json
{
  "name": "Azure GPT-4 完整配置",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 8192
  }
}
```

**必需的 queryParams 字段**:

- `azureEndpoint`: Azure OpenAI 资源端点
- `azureDeployment`: 部署名称
- `azureApiVersion`: API 版本

**支持的 API 版本**:

- `2024-02-15-preview` - 最新预览版
- `2023-12-01-preview` - 稳定预览版
- `2023-05-15` - 稳定版

### 4. 阿里云百炼

**基础配置**:

```json
{
  "name": "通义千问 Turbo",
  "model": "qwen-turbo",
  "modelProvider": "bianlian",
  "apiKey": "sk-..."
}
```

**完整配置**:

```json
{
  "name": "通义千问 Max",
  "model": "qwen-max",
  "modelProvider": "bianlian",
  "apiKey": "sk-...",
  "queryParams": {
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "bailianWorkspace": "default",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

**可选的 queryParams 字段**:

- `bailianEndpoint`: 百炼 API 端点（默认：北京地域）
- `bailianWorkspace`: 工作空间名称（默认：default）

**支持的模型**:

- `qwen-turbo` - 通义千问 Turbo
- `qwen-plus` - 通义千问 Plus
- `qwen-max` - 通义千问 Max
- `qwen-max-longcontext` - 长上下文版本

**地域端点**:

- 北京：`https://dashscope.aliyuncs.com/compatible-mode/v1`
- 上海：`https://dashscope-intl.aliyuncs.com/compatible-mode/v1`

### 5. Anthropic (Claude)

**基础配置**:

```json
{
  "name": "Claude 3 Opus",
  "model": "claude-3-opus-20240229",
  "modelProvider": "anthropic",
  "apiKey": "sk-ant-..."
}
```

**完整配置**:

```json
{
  "name": "Claude 3 Sonnet",
  "model": "claude-3-sonnet-20240229",
  "modelProvider": "anthropic",
  "apiKey": "sk-ant-...",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 4096
  }
}
```

**支持的模型**:

- `claude-3-opus-20240229` - Claude 3 Opus（最强）
- `claude-3-sonnet-20240229` - Claude 3 Sonnet（平衡）
- `claude-3-haiku-20240307` - Claude 3 Haiku（快速）
- `claude-2.1` - Claude 2.1
- `claude-2.0` - Claude 2.0

### 6. 自定义 OpenAI 兼容服务

**基础配置**:

```json
{
  "name": "本地模型",
  "model": "llama-2-70b",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:8000/v1",
  "apiKey": "not-needed"
}
```

**完整配置**:

```json
{
  "name": "自定义 LLM 服务",
  "model": "custom-model",
  "modelProvider": "custom_openai",
  "baseUrl": "https://your-api.com/v1",
  "apiKey": "your-api-key",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

**说明**:

- 必须提供 `baseUrl` 字段
- 服务必须兼容 OpenAI API 规范
- 适用于本地部署的模型（如 Ollama、vLLM 等）

## QueryParams 配置详解

### 通用参数

所有提供商都支持以下通用参数：

```json
{
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

| 参数 | 类型 | 范围 | 说明 |
|------|------|------|------|
| defaultTemperature | number | 0-2 | 默认温度值，控制输出随机性 |
| defaultMaxTokens | number | >0 | 默认最大 token 数 |

### Azure OpenAI 特定参数

```json
{
  "azureEndpoint": "https://your-resource.openai.azure.com",
  "azureDeployment": "gpt-4",
  "azureApiVersion": "2024-02-15-preview"
}
```

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| azureEndpoint | string | 是 | Azure OpenAI 资源端点 |
| azureDeployment | string | 是 | 部署名称 |
| azureApiVersion | string | 是 | API 版本 |

### 百炼特定参数

```json
{
  "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "bailianWorkspace": "default"
}
```

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| bailianEndpoint | string | 否 | 百炼 API 端点 |
| bailianWorkspace | string | 否 | 工作空间名称 |

## API 接口

### 创建配置

**端点**: `POST /api/v1/model-configurations`

**权限**:

- 平台管理员：可以为任何租户创建
- 租户管理员：只能为自己的租户创建

**请求体**:

```json
{
  "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",  // 可选，租户管理员自动使用当前租户
  "name": "GPT-4 配置",
  "model": "gpt-4",
  "modelProvider": "openai",
  "baseUrl": null,  // 可选
  "apiKey": "sk-...",
  "queryParams": {  // 可选
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

**响应**:

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
    "apiKey": "sk-yo****here",  // 脱敏后的密钥
    "isEnabled": true,
    "createdBy": "user-uuid",
    "createdAt": "2025-12-01T10:00:00Z"
  }
}
```

### 查询配置列表

**端点**: `GET /api/v1/model-configurations`

**查询参数**:

- `pageNo`: 页码（默认 1）
- `pageSize`: 每页大小（默认 10）
- `modelProvider`: 按提供商过滤（可选）
- `isEnabled`: 按启用状态过滤（可选）

**响应**:

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "GPT-4 配置",
        "model": "gpt-4",
        "modelProvider": "openai",
        "isEnabled": true
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 1,
    "totalPage": 1
  }
}
```

### 查询单个配置

**端点**: `GET /api/v1/model-configurations/{id}`

**响应**:

```json
{
  "code": 200,
  "message": "查询成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenantId": "738dbb1f-83e6-4bf5-935c-f0498236440d",
    "name": "GPT-4 配置",
    "model": "gpt-4",
    "modelProvider": "openai",
    "baseUrl": null,
    "apiKey": "sk-yo****here",
    "queryParams": {
      "defaultTemperature": 0.7,
      "defaultMaxTokens": 2048
    },
    "isEnabled": true,
    "createdBy": "user-uuid",
    "createdAt": "2025-12-01T10:00:00Z"
  }
}
```

### 更新配置

**端点**: `PUT /api/v1/model-configurations/{id}`

**请求体**:

```json
{
  "name": "GPT-4 Turbo 配置",  // 可选
  "model": "gpt-4-turbo",  // 可选
  "apiKey": "sk-new-key",  // 可选
  "queryParams": {  // 可选
    "defaultTemperature": 0.8
  }
}
```

**说明**:

- 所有字段都是可选的
- 只更新提供的字段
- 更新后会自动清除缓存

### 更新状态

**端点**: `PATCH /api/v1/model-configurations/{id}/status`

**请求体**:

```json
{
  "status": "disabled"  // enabled 或 disabled
}
```

### 删除配置

**端点**: `DELETE /api/v1/model-configurations/{id}`

**说明**:

- 执行软删除
- 只有平台管理员可以删除

## 配置验证

### 验证规则

系统会自动验证配置的有效性：

1. **通用验证**:
   - `model` 字段不能为空
   - `apiKey` 字段不能为空
   - `modelProvider` 必须是支持的类型

2. **Azure OpenAI 验证**:
   - 必须提供 `azureEndpoint`
   - 必须提供 `azureDeployment`
   - 必须提供 `azureApiVersion`

3. **百炼验证**:
   - 如果提供 `bailianEndpoint`，必须是有效的 URL
   - 如果提供 `bailianWorkspace`，不能为空

4. **自定义 OpenAI 验证**:
   - 必须提供 `baseUrl`
   - `baseUrl` 必须是有效的 URL

### 验证示例

**有效配置**:

```json
{
  "name": "Azure GPT-4",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "valid-key",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

**无效配置**（缺少必需字段）:

```json
{
  "name": "Azure GPT-4",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "valid-key",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com"
    // 缺少 azureDeployment 和 azureApiVersion
  }
}
```

**错误响应**:

```json
{
  "code": 400,
  "message": "配置验证失败: Azure OpenAI 配置缺少必需字段: azureDeployment"
}
```

## 安全最佳实践

### 1. API 密钥管理

- ✅ 使用强密钥
- ✅ 定期轮换密钥
- ✅ 为不同环境使用不同密钥
- ✅ 限制密钥权限
- ❌ 不要在代码中硬编码密钥
- ❌ 不要在日志中记录完整密钥

### 2. 访问控制

- ✅ 遵循最小权限原则
- ✅ 租户管理员只能管理自己的配置
- ✅ 定期审查权限
- ✅ 记录所有配置变更

### 3. 数据保护

- ✅ API 密钥加密存储
- ✅ 传输使用 HTTPS
- ✅ 定期备份配置
- ✅ 实施审计日志

## 性能优化

### 1. 缓存机制

系统会自动缓存 Genkit 实例：

- 缓存键：`{tenantId}_{modelName}`
- 首次调用时初始化并缓存
- 后续调用直接使用缓存
- 配置更新时自动清除缓存

### 2. 并发控制

- 使用读写锁保护缓存
- 支持并发读取
- 双重检查锁定避免重复初始化

### 3. 懒加载

- 只在首次使用时初始化提供商
- 避免启动时加载所有提供商
- 减少内存占用

## 故障排查

### 常见问题

1. **配置不存在**
   - 检查租户 ID 和模型名称是否正确
   - 确认配置未被删除
   - 确认配置已启用

2. **API 密钥无效**
   - 检查密钥是否正确
   - 确认密钥未过期
   - 检查密钥权限

3. **Azure OpenAI 连接失败**
   - 检查 endpoint URL 是否正确
   - 确认 deployment 名称正确
   - 验证 API 版本支持

4. **百炼调用失败**
   - 检查 endpoint 是否可访问
   - 确认 API 密钥有效
   - 验证模型名称正确

详细的故障排查步骤请参考 [故障排查指南](./TROUBLESHOOTING.md)。

## 下一步

- 查看 [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md) 了解如何使用
- 查看 [故障排查指南](./TROUBLESHOOTING.md) 解决问题
- 查看 [迁移指南](./MIGRATION_GUIDE.md) 了解迁移步骤
