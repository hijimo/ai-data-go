# Azure OpenAI 流式调用测试指南

## 概述

本文档介绍如何运行 Azure OpenAI 流式调用的集成测试。

## 前置条件

### 1. Azure OpenAI 资源

您需要一个已配置的 Azure OpenAI 资源，包括：

- Azure OpenAI API 密钥
- Azure OpenAI 端点 URL
- 已部署的模型（如 GPT-4）

### 2. 数据库

测试需要连接到 PostgreSQL 数据库：

- PostgreSQL 13+ （支持 `gen_random_uuid()`）
- 测试数据库（默认：`genkit_test`）

### 3. Go 环境

- Go 1.21+
- 已安装项目依赖

## 配置环境变量

### Azure OpenAI 配置（必需）

```bash
# Azure OpenAI API 密钥
export AZURE_OPENAI_API_KEY=your-api-key

# Azure OpenAI 端点
export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com

# 部署名称
export AZURE_OPENAI_DEPLOYMENT=your-deployment-name

# API 版本（可选，默认：2024-02-15-preview）
export AZURE_OPENAI_API_VERSION=2024-02-15-preview
```

### 数据库配置（可选）

```bash
# 数据库主机（默认：localhost）
export DB_HOST=localhost

# 数据库端口（默认：5432）
export DB_PORT=5432

# 数据库用户（默认：postgres）
export DB_USER=postgres

# 数据库密码（默认：postgres）
export DB_PASSWORD=postgres

# 数据库名称（默认：genkit_test）
export DB_NAME=genkit_test
```

## 运行测试

### 方式 1：使用测试脚本（推荐）

```bash
# 进入项目根目录
cd /path/to/ai-data-go

# 运行流式调用测试脚本
./test/test_azure_stream.sh
```

测试脚本会：

1. 检查必需的环境变量
2. 显示配置信息
3. 运行所有流式调用测试
4. 显示测试结果

### 方式 2：直接运行 Go 测试

```bash
# 运行所有流式调用测试
go test -v -timeout 5m -run TestAzureOpenAIIntegration_Streaming ./internal/genkit/

# 运行特定的测试用例
go test -v -timeout 5m -run TestAzureOpenAIIntegration_Streaming/流式响应接收 ./internal/genkit/
```

### 方式 3：跳过集成测试

如果没有配置 Azure OpenAI 或不想运行集成测试：

```bash
# 使用 -short 标志跳过集成测试
go test -short ./internal/genkit/
```

## 测试用例说明

### 1. 流式响应接收

测试基本的流式响应接收功能，验证：

- 能够接收多个流式块
- 完整文本的拼接正确
- Token 使用统计准确

### 2. 流式响应完整性

测试流式响应的完整性，验证：

- 响应内容符合预期
- 所有流式块都被正确接收

### 3. 流式中断处理

测试上下文取消机制，验证：

- 流式响应可以被正确中断
- 中断后资源被正确清理

### 4. 流式参数传递

测试自定义参数的传递，验证：

- temperature 参数生效
- maxTokens 参数生效

### 5. 流式响应格式验证

验证流式响应的格式，确保：

- 每个块的格式正确
- 完成块包含模型信息和使用统计

### 6. 错误处理

测试各种错误场景：

- 配置不存在
- 租户ID无效
- 模型已禁用

### 7. 流式响应性能

测试性能指标：

- 首字节时间（TTFB）< 5秒
- 记录总耗时和块数量

### 8. 并发流式调用

测试并发场景：

- 多个并发流式调用
- 验证稳定性和线程安全

## 测试输出示例

```
=== RUN   TestAzureOpenAIIntegration_Streaming
=== RUN   TestAzureOpenAIIntegration_Streaming/流式响应接收
    azure_stream_test.go:95: 接收到流式块: 你好
    azure_stream_test.go:95: 接收到流式块: ！我是
    azure_stream_test.go:95: 接收到流式块: GPT-4
    azure_stream_test.go:97: 流式响应完成
    azure_stream_test.go:107: 完整响应: 你好！我是GPT-4
    azure_stream_test.go:108: 总共接收到 4 个块
    azure_stream_test.go:118: Token 使用情况: Prompt=15, Completion=8, Total=23
=== RUN   TestAzureOpenAIIntegration_Streaming/流式响应完整性
    azure_stream_test.go:145: 流式响应: Python, JavaScript, Java
=== RUN   TestAzureOpenAIIntegration_Streaming/流式中断处理
    azure_stream_test.go:171: 接收到第 1 个块
    azure_stream_test.go:171: 接收到第 2 个块
    azure_stream_test.go:171: 接收到第 3 个块
    azure_stream_test.go:176: 取消流式响应
    azure_stream_test.go:173: 流式响应被正确取消
--- PASS: TestAzureOpenAIIntegration_Streaming (15.23s)
    --- PASS: TestAzureOpenAIIntegration_Streaming/流式响应接收 (3.45s)
    --- PASS: TestAzureOpenAIIntegration_Streaming/流式响应完整性 (2.87s)
    --- PASS: TestAzureOpenAIIntegration_Streaming/流式中断处理 (1.23s)
PASS
ok      genkit-ai-service/internal/genkit       15.234s
```

## 常见问题

### Q1: 测试被跳过

**问题**: 测试输出显示 "跳过 Azure OpenAI 集成测试：缺少必需的环境变量"

**解决方案**: 确保设置了以下环境变量：

- `AZURE_OPENAI_API_KEY`
- `AZURE_OPENAI_ENDPOINT`
- `AZURE_OPENAI_DEPLOYMENT`

### Q2: 数据库连接失败

**问题**: 测试失败，提示数据库连接错误

**解决方案**:

1. 确保 PostgreSQL 服务正在运行
2. 检查数据库配置环境变量
3. 确保测试数据库存在：`createdb genkit_test`

### Q3: API 调用失败

**问题**: 测试失败，提示 API 调用错误

**解决方案**:

1. 验证 API 密钥是否正确
2. 检查端点 URL 格式是否正确
3. 确认部署名称与 Azure 门户中的一致
4. 检查 API 版本是否支持

### Q4: 测试超时

**问题**: 测试运行时间过长或超时

**解决方案**:

1. 检查网络连接
2. 增加超时时间：`go test -timeout 10m ...`
3. 检查 Azure OpenAI 服务状态

### Q5: 首字节时间过长

**问题**: 首字节时间超过 5 秒

**解决方案**:

1. 检查网络延迟
2. 选择更近的 Azure 区域
3. 检查 Azure OpenAI 服务负载

## 性能基准

基于正常网络条件的性能基准：

| 指标 | 预期值 | 说明 |
|------|--------|------|
| 首字节时间（TTFB） | < 2秒 | 从发起请求到接收第一个块 |
| 总耗时 | < 10秒 | 完整响应的总时间 |
| 流式块数量 | 5-20个 | 取决于响应长度 |
| Token 使用 | 10-100 | 取决于提示词和响应 |

## 相关文档

- [Azure OpenAI 非流式测试](./AZURE_INTEGRATION_TEST_README.md)
- [TASK-3.4 实现总结](./TASK-3.4-IMPLEMENTATION-SUMMARY.md)
- [TASK-3.3 实现总结](./TASK-3.3-IMPLEMENTATION-SUMMARY.md)
- [Azure OpenAI 集成决策](./AZURE_INTEGRATION_DECISION.md)

## 技术支持

如果遇到问题，请：

1. 查看测试日志输出
2. 检查环境变量配置
3. 参考相关文档
4. 联系开发团队

## 更新日志

- 2025-11-26: 创建流式调用测试套件
- 2025-11-26: 添加 8 个测试用例
- 2025-11-26: 创建测试脚本和文档
