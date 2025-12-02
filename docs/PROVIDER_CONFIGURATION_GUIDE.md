# 提供商配置详细指南

## 概述

本文档提供各个 AI 模型提供商的详细配置指南，包括配置步骤、参数说明、最佳实践和常见问题。

## 目录

- [Google AI (Gemini)](#google-ai-gemini)
- [OpenAI](#openai)
- [Azure OpenAI](#azure-openai)
- [阿里云百炼](#阿里云百炼)
- [Anthropic (Claude)](#anthropic-claude)
- [自定义 OpenAI 兼容服务](#自定义-openai-兼容服务)

---

## Google AI (Gemini)

### 概述

Google AI 提供 Gemini 系列模型，支持文本生成、多模态理解等功能。

### 获取 API 密钥

1. 访问 [Google AI Studio](https://makersuite.google.com/app/apikey)
2. 使用 Google 账号登录
3. 点击"Create API Key"按钮
4. 选择或创建一个 Google Cloud 项目
5. 复制生成的 API 密钥并妥善保存

**注意事项**：

- API 密钥与 Google Cloud 项目关联
- 需要启用 Generative Language API
- 免费配额有限，超出需要付费

### 基础配置

```json
{
  "name": "Gemini Pro",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "apiKey": "AIzaSyXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
}
```

### 支持的模型

| 模型名称 | 说明 | 上下文长度 | 适用场景 |
|---------|------|-----------|---------|
| gemini-1.5-pro | 最新 Pro 版本 | 1M tokens | 复杂任务、长文本处理 |
| gemini-1.5-flash | 快速版本 | 1M tokens | 快速响应、高并发场景 |
| gemini-pro | 标准版本 | 32K tokens | 通用任务 |
| gemini-pro-vision | 视觉版本 | 16K tokens | 图像理解、多模态任务 |

### 高级配置

```json
{
  "name": "Gemini Pro 高级配置",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "apiKey": "AIzaSyXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048,
    "topP": 0.95,
    "topK": 40
  }
}
```

### 配置参数说明

| 参数 | 类型 | 范围 | 默认值 | 说明 |
|------|------|------|--------|------|
| defaultTemperature | number | 0-2 | 0.7 | 控制输出随机性，越高越随机 |
| defaultMaxTokens | number | 1-1048576 | 2048 | 最大输出 token 数 |
| topP | number | 0-1 | 0.95 | 核采样参数 |
| topK | number | 1-100 | 40 | Top-K 采样参数 |

### 最佳实践

#### 1. 模型选择

- **复杂推理任务**：使用 `gemini-1.5-pro`
- **快速响应场景**：使用 `gemini-1.5-flash`
- **图像理解任务**：使用 `gemini-pro-vision`
- **成本敏感场景**：使用 `gemini-pro`

#### 2. 参数调优

**创意写作**：

```json
{
  "defaultTemperature": 0.9,
  "topP": 0.95
}
```

**事实性回答**：

```json
{
  "defaultTemperature": 0.2,
  "topP": 0.8
}
```

**代码生成**：

```json
{
  "defaultTemperature": 0.0,
  "topP": 0.9
}
```

#### 3. 成本优化

- 使用 `gemini-1.5-flash` 替代 `gemini-1.5-pro` 可节省约 50% 成本
- 合理设置 `maxTokens` 避免浪费
- 使用缓存机制减少重复调用
- 监控 token 使用量

### 常见问题

**Q: API 密钥无效怎么办？**

A: 检查以下几点：

1. 确认密钥复制完整，没有多余空格
2. 确认 Google Cloud 项目已启用 Generative Language API
3. 确认账号有足够的配额
4. 尝试重新生成密钥

**Q: 如何处理速率限制？**

A:

1. 实现指数退避重试机制
2. 使用队列控制并发请求数
3. 升级到付费计划获取更高配额
4. 分散请求到多个 API 密钥

**Q: 支持哪些语言？**

A: Gemini 支持 100+ 种语言，包括中文、英文、日文等主流语言。

---

## OpenAI

### 概述

OpenAI 提供 GPT 系列模型，是目前最流行的大语言模型之一，支持文本生成、对话、代码生成等多种任务。

### 获取 API 密钥

1. 访问 [OpenAI Platform](https://platform.openai.com/api-keys)
2. 使用 OpenAI 账号登录（如无账号需先注册）
3. 点击"Create new secret key"按钮
4. 为密钥命名（可选）
5. 复制生成的 API 密钥并妥善保存

**重要提示**：

- API 密钥只在创建时显示一次，请立即保存
- 建议为不同应用创建不同的密钥
- 定期轮换密钥以提高安全性
- 可以设置密钥的使用限额

### 基础配置

```json
{
  "name": "GPT-4",
  "model": "gpt-4",
  "modelProvider": "openai",
  "apiKey": "sk-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
}
```

### 支持的模型

| 模型名称 | 说明 | 上下文长度 | 适用场景 | 相对成本 |
|---------|------|-----------|---------|---------|
| gpt-4 | GPT-4 标准版 | 8K tokens | 复杂推理、专业任务 | 高 |
| gpt-4-turbo | GPT-4 Turbo | 128K tokens | 长文本处理、复杂任务 | 中高 |
| gpt-4-turbo-preview | GPT-4 Turbo 预览版 | 128K tokens | 最新功能测试 | 中高 |
| gpt-3.5-turbo | GPT-3.5 Turbo | 16K tokens | 通用任务、高并发 | 低 |
| gpt-3.5-turbo-16k | GPT-3.5 Turbo 16K | 16K tokens | 中等长度文本 | 低 |

### 高级配置

```json
{
  "name": "GPT-4 Turbo 高级配置",
  "model": "gpt-4-turbo-preview",
  "modelProvider": "openai",
  "apiKey": "sk-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 4096,
    "topP": 1.0,
    "frequencyPenalty": 0.0,
    "presencePenalty": 0.0
  }
}
```

### 配置参数说明

| 参数 | 类型 | 范围 | 默认值 | 说明 |
|------|------|------|--------|------|
| defaultTemperature | number | 0-2 | 0.7 | 控制输出随机性 |
| defaultMaxTokens | number | 1-128000 | 4096 | 最大输出 token 数 |
| topP | number | 0-1 | 1.0 | 核采样参数 |
| frequencyPenalty | number | -2.0-2.0 | 0.0 | 频率惩罚，降低重复 |
| presencePenalty | number | -2.0-2.0 | 0.0 | 存在惩罚，鼓励新话题 |

### 最佳实践

#### 1. 模型选择策略

**高质量要求**：

- 使用 `gpt-4` 或 `gpt-4-turbo`
- 适合：专业写作、复杂推理、代码审查

**平衡性能和成本**：

- 使用 `gpt-3.5-turbo`
- 适合：客服对话、内容摘要、简单问答

**长文本处理**：

- 使用 `gpt-4-turbo`（128K 上下文）
- 适合：文档分析、长对话、代码库理解

#### 2. 参数调优建议

**创意写作**：

```json
{
  "defaultTemperature": 0.8,
  "topP": 0.95,
  "presencePenalty": 0.6
}
```

**技术文档**：

```json
{
  "defaultTemperature": 0.3,
  "topP": 0.9,
  "frequencyPenalty": 0.3
}
```

**代码生成**：

```json
{
  "defaultTemperature": 0.2,
  "topP": 0.95,
  "frequencyPenalty": 0.0
}
```

#### 3. 成本优化

- **使用 GPT-3.5 替代 GPT-4**：成本降低约 90%
- **合理设置 maxTokens**：避免不必要的长输出
- **实现缓存机制**：相同请求复用结果
- **批量处理**：合并多个小请求
- **监控使用量**：设置预算告警

### 常见问题

**Q: 如何处理速率限制？**

A: OpenAI 有以下限制：

- RPM (Requests Per Minute)：每分钟请求数
- TPM (Tokens Per Minute)：每分钟 token 数

解决方案：

1. 实现指数退避重试
2. 使用队列控制并发
3. 升级到更高级别的计划
4. 分散请求到多个 API 密钥

**Q: API 密钥泄露怎么办？**

A:

1. 立即在 OpenAI 控制台撤销该密钥
2. 创建新的 API 密钥
3. 更新系统配置
4. 检查是否有异常使用
5. 考虑启用使用限额

**Q: 如何选择合适的模型？**

A: 决策树：

- 需要最高质量？→ GPT-4
- 需要处理长文本？→ GPT-4 Turbo
- 成本敏感？→ GPT-3.5 Turbo
- 需要最新功能？→ GPT-4 Turbo Preview

**Q: 支持哪些语言？**

A: OpenAI 模型支持多种语言，包括但不限于：

- 英语（最佳性能）
- 中文、日文、韩文
- 西班牙语、法语、德语
- 其他主流语言

---

## Azure OpenAI

### 概述

Azure OpenAI 是微软 Azure 云平台托管的 OpenAI 服务，提供企业级的安全性、合规性和 SLA 保障。

### 获取配置信息

#### 步骤 1：创建 Azure OpenAI 资源

1. 登录 [Azure Portal](https://portal.azure.com)
2. 搜索"Azure OpenAI"
3. 点击"创建"按钮
4. 填写资源信息：
   - 订阅
   - 资源组
   - 区域（建议选择离用户近的区域）
   - 名称
   - 定价层
5. 点击"审阅 + 创建"

#### 步骤 2：部署模型

1. 进入创建的 Azure OpenAI 资源
2. 点击"模型部署"
3. 点击"创建新部署"
4. 选择模型（如 gpt-4）
5. 输入部署名称（记住这个名称，配置时需要）
6. 选择模型版本
7. 点击"创建"

#### 步骤 3：获取密钥和端点

1. 在资源页面，点击"密钥和终结点"
2. 复制以下信息：
   - **终结点**（Endpoint）：形如 `https://your-resource.openai.azure.com`
   - **密钥 1** 或 **密钥 2**（Key）
3. 记录部署名称（在"模型部署"页面查看）

### 基础配置

```json
{
  "name": "Azure GPT-4",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key-here",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

### 必需的配置字段

| 字段 | 说明 | 示例 |
|------|------|------|
| azureEndpoint | Azure OpenAI 资源端点 | `https://your-resource.openai.azure.com` |
| azureDeployment | 部署名称 | `gpt-4` |
| azureApiVersion | API 版本 | `2024-02-15-preview` |

### 支持的 API 版本

| API 版本 | 状态 | 说明 | 推荐使用 |
|---------|------|------|---------|
| 2024-02-15-preview | 预览 | 最新功能 | ✅ 推荐 |
| 2023-12-01-preview | 预览 | 稳定预览版 | ✅ 推荐 |
| 2023-05-15 | 稳定 | 生产环境 | ✅ 推荐 |
| 2023-03-15-preview | 预览 | 旧版本 | ❌ 不推荐 |

### 支持的模型

Azure OpenAI 支持的模型取决于你的订阅和区域。常见模型包括：

| 模型系列 | 可用模型 | 说明 |
|---------|---------|------|
| GPT-4 | gpt-4, gpt-4-32k | 最强大的模型 |
| GPT-4 Turbo | gpt-4-turbo, gpt-4-1106-preview | 长上下文版本 |
| GPT-3.5 | gpt-35-turbo, gpt-35-turbo-16k | 高性价比 |

**注意**：Azure 中的模型名称可能与 OpenAI 不同（如 `gpt-35-turbo` vs `gpt-3.5-turbo`）。

### 高级配置

```json
{
  "name": "Azure GPT-4 完整配置",
  "model": "gpt-4",
  "modelProvider": "azureopenai",
  "apiKey": "your-azure-api-key-here",
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4-deployment",
    "azureApiVersion": "2024-02-15-preview",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 8192
  }
}
```

### 最佳实践

#### 1. 区域选择

**考虑因素**：

- **延迟**：选择离用户最近的区域
- **可用性**：不是所有区域都支持所有模型
- **合规性**：某些行业需要数据驻留在特定区域

**推荐区域**：

- 美国东部（East US）：模型最全
- 西欧（West Europe）：欧洲用户
- 日本东部（Japan East）：亚洲用户

#### 2. 部署策略

**单部署**：

- 简单场景
- 成本敏感
- 流量可预测

**多部署**：

- 高可用性要求
- 负载均衡
- A/B 测试

#### 3. 安全配置

**网络安全**：

```json
{
  "queryParams": {
    "azureEndpoint": "https://your-resource.openai.azure.com",
    "azureDeployment": "gpt-4",
    "azureApiVersion": "2024-02-15-preview"
  }
}
```

**密钥管理**：

- 使用 Azure Key Vault 存储密钥
- 定期轮换密钥
- 使用托管标识（Managed Identity）

#### 4. 成本优化

- **使用预留容量**：长期使用可节省成本
- **监控使用量**：设置预算告警
- **选择合适的模型**：不是所有任务都需要 GPT-4
- **优化提示词**：减少不必要的 token 消耗

### 常见问题

**Q: Azure OpenAI 和 OpenAI 有什么区别？**

A: 主要区别：

- **部署位置**：Azure 在微软数据中心
- **合规性**：Azure 提供企业级合规认证
- **SLA**：Azure 提供 99.9% 可用性保证
- **定价**：定价模式略有不同
- **功能**：Azure 可能稍晚于 OpenAI 推出新功能

**Q: 如何处理配额限制？**

A:

1. 在 Azure Portal 中查看当前配额
2. 提交配额增加请求
3. 考虑使用多个部署分散负载
4. 实现请求队列和重试机制

**Q: 端点 URL 格式是什么？**

A: 标准格式：

```
https://{resource-name}.openai.azure.com
```

完整 API 调用 URL：

```
https://{resource-name}.openai.azure.com/openai/deployments/{deployment-name}/chat/completions?api-version={api-version}
```

**Q: 如何验证配置是否正确？**

A: 使用 curl 测试：

```bash
curl https://your-resource.openai.azure.com/openai/deployments/gpt-4/chat/completions?api-version=2024-02-15-preview \
  -H "Content-Type: application/json" \
  -H "api-key: YOUR_API_KEY" \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## 阿里云百炼

### 概述

阿里云百炼（DashScope）是阿里云提供的大模型服务平台，提供通义千问系列模型，特别适合中文场景。

### 获取 API 密钥

#### 步骤 1：开通服务

1. 访问 [阿里云百炼控制台](https://bailian.console.aliyun.com/)
2. 使用阿里云账号登录（如无账号需先注册）
3. 开通百炼服务
4. 同意服务协议

#### 步骤 2：创建 API Key

1. 在控制台左侧菜单，点击"API-KEY 管理"
2. 点击"创建新的 API-KEY"
3. 输入 API Key 名称（可选）
4. 点击"确定"
5. 复制生成的 API Key 并妥善保存

**重要提示**：

- API Key 只在创建时显示一次
- 建议为不同应用创建不同的 Key
- 可以设置 Key 的有效期
- 支持设置 IP 白名单

### 基础配置

```json
{
  "name": "通义千问 Turbo",
  "model": "qwen-turbo",
  "modelProvider": "bianlian",
  "apiKey": "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

### 支持的模型

| 模型名称 | 说明 | 上下文长度 | 适用场景 | 相对成本 |
|---------|------|-----------|---------|---------|
| qwen-max | 通义千问 Max | 8K tokens | 复杂推理、专业任务 | 高 |
| qwen-plus | 通义千问 Plus | 32K tokens | 通用任务、长文本 | 中 |
| qwen-turbo | 通义千问 Turbo | 8K tokens | 快速响应、高并发 | 低 |
| qwen-max-longcontext | 长上下文版本 | 30K tokens | 长文档处理 | 高 |

### 高级配置

```json
{
  "name": "通义千问 Plus 高级配置",
  "model": "qwen-plus",
  "modelProvider": "bianlian",
  "apiKey": "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "queryParams": {
    "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "bailianWorkspace": "default",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

### 配置参数说明

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| bailianEndpoint | string | 否 | 北京地域端点 | API 端点 URL |
| bailianWorkspace | string | 否 | default | 工作空间名称 |
| defaultTemperature | number | 否 | 0.7 | 控制输出随机性 |
| defaultMaxTokens | number | 否 | 2048 | 最大输出 token 数 |

### 地域端点

| 地域 | 端点 URL | 说明 |
|------|---------|------|
| 北京 | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 默认，推荐 |
| 上海 | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | 国际版 |

### 最佳实践

#### 1. 模型选择策略

**高质量中文任务**：

- 使用 `qwen-max`
- 适合：专业写作、复杂推理、知识问答

**平衡性能和成本**：

- 使用 `qwen-plus`
- 适合：通用对话、内容生成、文档处理

**高并发场景**：

- 使用 `qwen-turbo`
- 适合：客服机器人、实时问答、简单任务

**长文本处理**：

- 使用 `qwen-max-longcontext`
- 适合：长文档分析、书籍摘要、法律文件

#### 2. 参数调优建议

**中文创意写作**：

```json
{
  "defaultTemperature": 0.85,
  "defaultMaxTokens": 2000
}
```

**事实性问答**：

```json
{
  "defaultTemperature": 0.1,
  "defaultMaxTokens": 1000
}
```

**代码生成**：

```json
{
  "defaultTemperature": 0.2,
  "defaultMaxTokens": 2048
}
```

#### 3. 成本优化

- **使用 qwen-turbo**：成本最低，适合大部分场景
- **合理设置 maxTokens**：避免不必要的长输出
- **批量处理**：合并相似请求
- **缓存结果**：相同问题复用答案
- **监控使用量**：定期查看消费情况

### 常见问题

**Q: 百炼和 OpenAI 的 API 兼容吗？**

A: 是的，百炼提供 OpenAI 兼容模式：

- 端点：`/compatible-mode/v1`
- 支持 Chat Completions API
- 支持流式响应
- 请求和响应格式与 OpenAI 一致

**Q: 如何处理速率限制？**

A: 百炼的限制包括：

- QPM (Queries Per Minute)：每分钟查询数
- TPM (Tokens Per Minute)：每分钟 token 数

解决方案：

1. 实现请求队列
2. 使用指数退避重试
3. 升级到更高级别的套餐
4. 联系客服申请提额

**Q: 支持哪些功能？**

A: 百炼支持：

- ✅ 文本生成
- ✅ 对话补全
- ✅ 流式响应
- ✅ 函数调用（Function Calling）
- ✅ 多轮对话
- ❌ 图像理解（部分模型支持）
- ❌ 语音功能

**Q: 如何优化中文效果？**

A: 建议：

1. 使用中文提示词
2. 提供清晰的上下文
3. 使用 qwen-max 获得最佳效果
4. 调整 temperature 参数
5. 提供示例（Few-shot Learning）

**Q: 数据安全如何保障？**

A:

- 数据不用于模型训练
- 支持私有化部署
- 符合国内数据安全法规
- 提供数据加密传输
- 支持 VPC 专线接入

---

## Anthropic (Claude)

### 概述

Anthropic 的 Claude 系列模型以安全性和可靠性著称，特别擅长长文本理解和复杂推理任务。

### 获取 API 密钥

1. 访问 [Anthropic Console](https://console.anthropic.com/)
2. 使用邮箱注册或登录
3. 完成账号验证
4. 点击"API Keys"
5. 点击"Create Key"
6. 输入密钥名称
7. 复制生成的 API 密钥

**注意事项**：

- 需要信用卡验证
- 新账号有免费额度
- 密钥只显示一次

### 基础配置

```json
{
  "name": "Claude 3 Opus",
  "model": "claude-3-opus-20240229",
  "modelProvider": "anthropic",
  "apiKey": "sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

### 支持的模型

| 模型名称 | 说明 | 上下文长度 | 适用场景 | 相对成本 |
|---------|------|-----------|---------|---------|
| claude-3-opus-20240229 | Claude 3 Opus | 200K tokens | 最复杂任务 | 最高 |
| claude-3-sonnet-20240229 | Claude 3 Sonnet | 200K tokens | 平衡性能 | 中 |
| claude-3-haiku-20240307 | Claude 3 Haiku | 200K tokens | 快速响应 | 低 |
| claude-2.1 | Claude 2.1 | 200K tokens | 上一代模型 | 中 |
| claude-2.0 | Claude 2.0 | 100K tokens | 旧版本 | 中 |

### 高级配置

```json
{
  "name": "Claude 3 Sonnet 高级配置",
  "model": "claude-3-sonnet-20240229",
  "modelProvider": "anthropic",
  "apiKey": "sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "queryParams": {
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 4096,
    "topP": 0.9,
    "topK": 40
  }
}
```

### 配置参数说明

| 参数 | 类型 | 范围 | 默认值 | 说明 |
|------|------|------|--------|------|
| defaultTemperature | number | 0-1 | 0.7 | 控制输出随机性 |
| defaultMaxTokens | number | 1-200000 | 4096 | 最大输出 token 数 |
| topP | number | 0-1 | 0.9 | 核采样参数 |
| topK | number | 1-500 | 40 | Top-K 采样参数 |

### 最佳实践

#### 1. 模型选择策略

**最高质量要求**：

- 使用 `claude-3-opus`
- 适合：研究分析、专业写作、复杂推理

**平衡场景**：

- 使用 `claude-3-sonnet`
- 适合：通用对话、内容生成、代码辅助

**高性能要求**：

- 使用 `claude-3-haiku`
- 适合：客服机器人、实时问答、简单任务

#### 2. 参数调优建议

**分析性任务**：

```json
{
  "defaultTemperature": 0.3,
  "defaultMaxTokens": 8192
}
```

**创意写作**：

```json
{
  "defaultTemperature": 0.8,
  "defaultMaxTokens": 4096
}
```

**代码生成**：

```json
{
  "defaultTemperature": 0.2,
  "defaultMaxTokens": 4096
}
```

#### 3. 长文本处理

Claude 3 支持 200K token 上下文，适合：

- 整本书籍分析
- 大型代码库理解
- 长篇文档摘要
- 复杂对话历史

**最佳实践**：

```json
{
  "model": "claude-3-opus-20240229",
  "queryParams": {
    "defaultMaxTokens": 16384,
    "defaultTemperature": 0.5
  }
}
```

### 常见问题

**Q: Claude 和 GPT 有什么区别？**

A: 主要区别：

- **上下文长度**：Claude 3 支持 200K tokens
- **安全性**：Claude 更注重安全和可靠性
- **长文本理解**：Claude 在长文本任务上表现更好
- **API 格式**：略有不同，但系统已做适配

**Q: 如何处理速率限制？**

A: Claude 的限制：

- RPM：每分钟请求数
- TPM：每分钟 token 数
- 不同模型有不同限制

解决方案：

1. 实现请求队列
2. 使用指数退避
3. 升级到更高级别
4. 联系支持团队

**Q: 支持哪些语言？**

A: Claude 支持多种语言：

- 英语（最佳）
- 中文、日文、韩文
- 西班牙语、法语、德语
- 其他主流语言

---

## 自定义 OpenAI 兼容服务

### 概述

支持任何兼容 OpenAI API 规范的自定义服务，包括本地部署的模型服务。

### 适用场景

- **本地部署模型**：Ollama、vLLM、LocalAI
- **私有化服务**：企业内部部署的模型服务
- **第三方服务**：其他提供 OpenAI 兼容 API 的服务
- **测试环境**：开发和测试用的模拟服务

### 基础配置

```json
{
  "name": "本地 Llama 模型",
  "model": "llama-2-70b",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:8000/v1",
  "apiKey": "not-needed"
}
```

### 配置要求

| 字段 | 必需 | 说明 |
|------|------|------|
| baseUrl | 是 | API 端点 URL，必须包含 `/v1` |
| apiKey | 是 | API 密钥（某些服务可能不需要） |
| model | 是 | 模型标识符 |

### 常见服务配置

#### 1. Ollama

**安装 Ollama**：

```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# 启动服务
ollama serve

# 下载模型
ollama pull llama2
```

**配置**：

```json
{
  "name": "Ollama Llama2",
  "model": "llama2",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:11434/v1",
  "apiKey": "ollama"
}
```

#### 2. vLLM

**启动 vLLM 服务**：

```bash
python -m vllm.entrypoints.openai.api_server \
  --model meta-llama/Llama-2-70b-chat-hf \
  --port 8000
```

**配置**：

```json
{
  "name": "vLLM Llama2",
  "model": "meta-llama/Llama-2-70b-chat-hf",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:8000/v1",
  "apiKey": "token-abc123"
}
```

#### 3. LocalAI

**启动 LocalAI**：

```bash
docker run -p 8080:8080 \
  -v $PWD/models:/models \
  localai/localai:latest
```

**配置**：

```json
{
  "name": "LocalAI GPT4All",
  "model": "gpt4all-j",
  "modelProvider": "custom_openai",
  "baseUrl": "http://localhost:8080/v1",
  "apiKey": "not-needed"
}
```

### 验证配置

使用 curl 测试连接：

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "your-model-name",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

### 最佳实践

#### 1. 性能优化

- 使用 GPU 加速
- 调整批处理大小
- 启用模型量化
- 配置合适的并发数

#### 2. 安全配置

- 使用 HTTPS（生产环境）
- 配置防火墙规则
- 实施访问控制
- 定期更新服务

#### 3. 监控和日志

- 监控服务健康状态
- 记录请求和响应
- 追踪性能指标
- 设置告警规则

### 常见问题

**Q: 如何确认服务兼容 OpenAI API？**

A: 检查以下端点：

- `/v1/models` - 列出可用模型
- `/v1/chat/completions` - 聊天补全
- `/v1/completions` - 文本补全

**Q: baseUrl 应该包含什么？**

A: 标准格式：

```
http(s)://host:port/v1
```

示例：

- `http://localhost:8000/v1`
- `https://api.example.com/v1`

**Q: 如何处理认证？**

A: 大多数服务支持：

- Bearer Token：`Authorization: Bearer YOUR_TOKEN`
- API Key：`api-key: YOUR_KEY`
- 无认证：某些本地服务

**Q: 性能如何优化？**

A:

1. 使用 GPU 加速
2. 启用模型量化（如 4-bit、8-bit）
3. 调整 batch size
4. 使用 vLLM 等优化框架
5. 配置合适的并发限制

---

## 总结

### 提供商对比

| 提供商 | 优势 | 劣势 | 适用场景 |
|--------|------|------|---------|
| Google AI | 多模态、长上下文 | 中文效果一般 | 图像理解、长文本 |
| OpenAI | 生态完善、效果好 | 成本较高 | 通用任务、高质量要求 |
| Azure OpenAI | 企业级、合规性 | 功能更新慢 | 企业应用、合规要求 |
| 百炼 | 中文效果好、成本低 | 生态较小 | 中文场景、成本敏感 |
| Anthropic | 长上下文、安全性 | 价格较高 | 长文本、复杂推理 |
| 自定义 | 灵活、私有化 | 需要维护 | 特殊需求、数据安全 |

### 选择建议

**通用场景**：OpenAI GPT-3.5 Turbo 或百炼 qwen-turbo

**高质量要求**：OpenAI GPT-4 或 Anthropic Claude 3 Opus

**中文场景**：百炼 qwen-max 或 qwen-plus

**长文本处理**：Anthropic Claude 3 或 Google Gemini 1.5 Pro

**企业应用**：Azure OpenAI

**成本敏感**：百炼 qwen-turbo 或 OpenAI GPT-3.5 Turbo

**数据安全**：自定义本地部署

### 下一步

- 查看 [多提供商使用指南](./MULTI_PROVIDER_GUIDE.md)
- 查看 [配置指南](./CONFIGURATION_GUIDE.md)
- 查看 [故障排查指南](./TROUBLESHOOTING.md)
