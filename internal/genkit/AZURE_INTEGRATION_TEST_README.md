# Azure OpenAI 集成测试指南

## 概述

本文档说明如何运行 Azure OpenAI 的集成测试，验证 Azure OpenAI 非流式调用功能。

## 前置条件

### 1. Azure OpenAI 资源

您需要一个已配置的 Azure OpenAI 资源，包括：

- Azure OpenAI Endpoint（例如：`https://your-resource.openai.azure.com`）
- Azure OpenAI API Key
- 已部署的模型（Deployment）名称（例如：`gpt-4`）
- API Version（可选，默认使用 `2024-02-15-preview`）

### 2. 数据库

测试需要连接到 PostgreSQL 数据库：

- 数据库主机（默认：`localhost`）
- 数据库端口（默认：`5432`）
- 数据库用户（默认：`postgres`）
- 数据库密码（默认：`postgres`）
- 数据库名称（默认：`genkit_test`）

## 环境变量配置

### 必需的环境变量

```bash
# Azure OpenAI API 密钥
export AZURE_OPENAI_API_KEY="your-api-key"

# Azure OpenAI Endpoint
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"

# Azure OpenAI Deployment 名称
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"
```

### 可选的环境变量

```bash
# Azure OpenAI API Version（默认：2024-02-15-preview）
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"

# 数据库配置（如果使用非默认值）
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

## 运行测试

### 方法 1：使用 Shell 脚本（推荐）

```bash
# 进入项目根目录
cd /path/to/project

# 运行测试脚本
./test/test_azure_openai.sh
```

### 方法 2：直接使用 Go Test

```bash
# 设置环境变量
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"

# 运行测试
go test -v -run TestAzureOpenAIIntegration_NonStreaming ./internal/genkit/
```

### 方法 3：跳过集成测试

如果您不想运行集成测试（例如在 CI/CD 环境中），可以使用 `-short` 标志：

```bash
go test -short ./internal/genkit/
```

## 测试用例说明

### 1. 基本文本生成

测试 Azure OpenAI 的基本文本生成功能，验证：

- 能够成功调用 Azure OpenAI API
- 返回非空的文本响应
- 模型名称正确

### 2. 参数传递测试

测试自定义参数（temperature、maxTokens）的传递，验证：

- 参数能够正确传递给 Azure OpenAI
- 生成的文本符合预期

### 3. Token 统计测试

测试 Token 使用情况的统计，验证：

- 返回的 Usage 信息不为空
- PromptTokens、CompletionTokens、TotalTokens 都大于 0
- TotalTokens = PromptTokens + CompletionTokens

### 4. 错误处理测试

测试各种错误场景，包括：

- 配置不存在
- 租户ID无效
- 模型已禁用

### 5. 响应格式验证

验证返回的响应格式符合预期，包括：

- Text 字段不为空
- Model 字段正确
- 响应内容符合提示词要求

### 6. 缓存机制测试

测试 Genkit 实例的缓存机制，验证：

- 第一次调用会初始化实例
- 第二次调用会使用缓存的实例

## 测试结果示例

成功的测试输出示例：

```
=== RUN   TestAzureOpenAIIntegration_NonStreaming
=== RUN   TestAzureOpenAIIntegration_NonStreaming/基本文本生成
    azure_integration_test.go:XX: 生成的文本: 我是一个AI助手，可以帮助您回答问题和完成任务。
=== RUN   TestAzureOpenAIIntegration_NonStreaming/参数传递测试
    azure_integration_test.go:XX: 生成的文本: Python、JavaScript、Go
=== RUN   TestAzureOpenAIIntegration_NonStreaming/Token统计测试
    azure_integration_test.go:XX: Token 使用情况: Prompt=10, Completion=5, Total=15
=== RUN   TestAzureOpenAIIntegration_NonStreaming/错误处理_-_配置不存在
=== RUN   TestAzureOpenAIIntegration_NonStreaming/错误处理_-_租户ID无效
=== RUN   TestAzureOpenAIIntegration_NonStreaming/错误处理_-_模型已禁用
=== RUN   TestAzureOpenAIIntegration_NonStreaming/响应格式验证
    azure_integration_test.go:XX: 响应格式验证通过: Text=你好！, Model=gpt-4
=== RUN   TestAzureOpenAIIntegration_NonStreaming/缓存机制测试
    azure_integration_test.go:XX: 第一次调用耗时: 1.234s
    azure_integration_test.go:XX: 第二次调用耗时: 0.987s
--- PASS: TestAzureOpenAIIntegration_NonStreaming (5.67s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/基本文本生成 (1.23s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/参数传递测试 (1.45s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/Token统计测试 (0.89s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/错误处理_-_配置不存在 (0.01s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/错误处理_-_租户ID无效 (0.01s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/错误处理_-_模型已禁用 (0.02s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/响应格式验证 (1.01s)
    --- PASS: TestAzureOpenAIIntegration_NonStreaming/缓存机制测试 (2.05s)
PASS
ok      genkit-ai-service/internal/genkit       5.678s
```

## 故障排查

### 问题 1：缺少环境变量

**错误信息**：

```
跳过 Azure OpenAI 集成测试：缺少必需的环境变量
```

**解决方案**：
确保设置了所有必需的环境变量（AZURE_OPENAI_API_KEY、AZURE_OPENAI_ENDPOINT、AZURE_OPENAI_DEPLOYMENT）。

### 问题 2：数据库连接失败

**错误信息**：

```
failed to connect to database
```

**解决方案**：

1. 确保 PostgreSQL 数据库正在运行
2. 检查数据库连接信息是否正确
3. 确保数据库用户有足够的权限

### 问题 3：Azure OpenAI API 调用失败

**错误信息**：

```
生成内容失败: ...
```

**解决方案**：

1. 检查 API Key 是否正确
2. 检查 Endpoint 和 Deployment 名称是否正确
3. 确保 Azure OpenAI 资源可用且有足够的配额
4. 检查网络连接

### 问题 4：测试超时

**解决方案**：

1. 检查网络连接
2. 增加测试超时时间
3. 检查 Azure OpenAI 服务状态

## 注意事项

1. **成本考虑**：集成测试会实际调用 Azure OpenAI API，会产生费用。建议在开发环境中谨慎运行。

2. **速率限制**：Azure OpenAI 有速率限制，频繁运行测试可能会触发限制。

3. **测试数据**：测试会在数据库中创建临时数据，测试结束后会自动清理。

4. **并发测试**：不建议并发运行多个集成测试实例，可能会导致数据冲突。

5. **CI/CD 集成**：在 CI/CD 环境中，建议使用 `-short` 标志跳过集成测试，或者配置专门的集成测试环境。

## 相关文档

- [Azure OpenAI 集成决策文档](./AZURE_INTEGRATION_DECISION.md)
- [Azure OpenAI 配置验证文档](./AZURE_OPENAI_BASEURL_VERIFICATION.md)
- [Genkit 多模型支持设计文档](../../.kiro/specs/genkit-multi-model-support/design.md)
- [Genkit 多模型支持需求文档](../../.kiro/specs/genkit-multi-model-support/requirements.md)
