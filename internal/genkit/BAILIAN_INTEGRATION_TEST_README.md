# 百炼集成测试指南

## 概述

本文档说明如何运行阿里云百炼的集成测试。这些测试验证百炼插件与 Genkit 框架的集成是否正常工作。

## 前置条件

### 1. 阿里云百炼账号和 API 密钥

- 注册阿里云账号：<https://www.aliyun.com/>
- 开通百炼服务：<https://dashscope.aliyun.com/>
- 获取 API 密钥（API-KEY）

### 2. 数据库环境

测试需要 PostgreSQL 数据库。确保数据库已启动并可访问。

### 3. Go 环境

- Go 1.21 或更高版本
- 已安装项目依赖：`go mod download`

## 环境变量配置

### 必需的环境变量

```bash
# 百炼 API 密钥（必需）
export BAILIAN_API_KEY="your-api-key-here"
```

### 可选的环境变量

```bash
# 百炼 API 端点（可选，默认：北京地域）
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"

# 模型名称（可选，默认：qwen-plus）
export BAILIAN_MODEL="qwen-plus"

# 数据库配置（可选）
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

### 百炼端点说明

百炼提供多个地域的端点：

- **北京地域**（默认）：`https://dashscope.aliyuncs.com/compatible-mode/v1`
- **新加坡地域**：`https://dashscope-intl.aliyuncs.com/compatible-mode/v1`
- **金融云**：`https://dashscope-finance.aliyuncs.com/compatible-mode/v1`

### 支持的模型

百炼支持多个模型，常用的包括：

- `qwen-plus`：通义千问增强版（推荐）
- `qwen-max`：通义千问旗舰版
- `qwen-turbo`：通义千问快速版
- `qwen-long`：长文本模型

## 运行测试

### 方法 1：使用测试脚本（推荐）

```bash
# 设置 API 密钥
export BAILIAN_API_KEY="your-api-key-here"

# 运行测试脚本
./test/test_bailian.sh
```

### 方法 2：直接运行 Go 测试

```bash
# 设置环境变量
export BAILIAN_API_KEY="your-api-key-here"
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"
export BAILIAN_MODEL="qwen-plus"

# 运行测试
go test -v -run TestBailianIntegration_NonStreaming ./internal/genkit/
```

### 方法 3：跳过集成测试

如果不想运行集成测试（例如在 CI 环境中），可以使用 `-short` 标志：

```bash
go test -short ./internal/genkit/
```

## 测试用例说明

### 1. 基本文本生成

测试百炼能否正常生成文本响应。

**测试内容**：

- 发送简单的中文提示词
- 验证返回的文本不为空
- 验证模型名称正确

### 2. 中文处理能力测试

测试百炼对中文的理解和生成能力。

**测试内容**：

- 发送中文问题
- 验证返回的是中文内容
- 验证包含关键词

### 3. 参数传递测试

测试自定义参数（temperature、maxTokens）是否正确传递。

**测试内容**：

- 使用自定义的 temperature 和 maxTokens
- 验证生成结果符合预期

### 4. Token 统计测试

测试 Token 使用情况的统计是否准确。

**测试内容**：

- 验证 PromptTokens > 0
- 验证 CompletionTokens > 0
- 验证 TotalTokens = PromptTokens + CompletionTokens

### 5. 错误处理测试

测试各种错误场景的处理。

**测试内容**：

- 配置不存在
- 租户ID无效
- 模型已禁用

### 6. 响应格式验证

验证响应格式是否符合预期。

**测试内容**：

- 验证 Text 字段不为空
- 验证 Model 字段正确
- 验证内容符合提示词要求

### 7. 缓存机制测试

测试 Genkit 实例的缓存机制。

**测试内容**：

- 第一次调用（初始化实例）
- 第二次调用（使用缓存）
- 记录两次调用的耗时

### 8. 复杂中文对话测试

测试更复杂的中文理解和生成能力。

**测试内容**：

- 发送包含多个要求的提示词
- 验证生成的内容符合要求
- 验证文本质量

## 测试结果示例

成功的测试输出示例：

```
=== RUN   TestBailianIntegration_NonStreaming
=== RUN   TestBailianIntegration_NonStreaming/基本文本生成
    bailian_integration_test.go:XX: 生成的文本: 你好！我是通义千问，由阿里云开发的AI助手。
=== RUN   TestBailianIntegration_NonStreaming/中文处理能力测试
    bailian_integration_test.go:XX: 中文生成结果: 人工智能是计算机科学的一个分支...
=== RUN   TestBailianIntegration_NonStreaming/参数传递测试
    bailian_integration_test.go:XX: 生成的文本: 春节、中秋节、端午节
=== RUN   TestBailianIntegration_NonStreaming/Token_统计测试
    bailian_integration_test.go:XX: Token 使用情况: Prompt=10, Completion=5, Total=15
=== RUN   TestBailianIntegration_NonStreaming/错误处理_-_配置不存在
=== RUN   TestBailianIntegration_NonStreaming/错误处理_-_租户ID无效
=== RUN   TestBailianIntegration_NonStreaming/错误处理_-_模型已禁用
=== RUN   TestBailianIntegration_NonStreaming/响应格式验证
    bailian_integration_test.go:XX: 响应格式验证通过: Text=你好！, Model=qwen-plus
=== RUN   TestBailianIntegration_NonStreaming/缓存机制测试
    bailian_integration_test.go:XX: 第一次调用耗时: 1.234s
    bailian_integration_test.go:XX: 第二次调用耗时: 0.987s
=== RUN   TestBailianIntegration_NonStreaming/复杂中文对话测试
    bailian_integration_test.go:XX: 复杂对话结果: 春天来了，花儿绽放...
--- PASS: TestBailianIntegration_NonStreaming (5.67s)
    --- PASS: TestBailianIntegration_NonStreaming/基本文本生成 (1.23s)
    --- PASS: TestBailianIntegration_NonStreaming/中文处理能力测试 (0.98s)
    --- PASS: TestBailianIntegration_NonStreaming/参数传递测试 (1.05s)
    --- PASS: TestBailianIntegration_NonStreaming/Token_统计测试 (0.87s)
    --- PASS: TestBailianIntegration_NonStreaming/错误处理_-_配置不存在 (0.01s)
    --- PASS: TestBailianIntegration_NonStreaming/错误处理_-_租户ID无效 (0.01s)
    --- PASS: TestBailianIntegration_NonStreaming/错误处理_-_模型已禁用 (0.02s)
    --- PASS: TestBailianIntegration_NonStreaming/响应格式验证 (0.95s)
    --- PASS: TestBailianIntegration_NonStreaming/缓存机制测试 (2.10s)
    --- PASS: TestBailianIntegration_NonStreaming/复杂中文对话测试 (1.15s)
PASS
ok      genkit-ai-service/internal/genkit       5.678s
```

## 常见问题

### 1. 测试失败：缺少 API 密钥

**错误信息**：

```
跳过百炼集成测试：缺少必需的环境变量 BAILIAN_API_KEY
```

**解决方法**：
设置 `BAILIAN_API_KEY` 环境变量：

```bash
export BAILIAN_API_KEY="your-api-key-here"
```

### 2. 测试失败：数据库连接错误

**错误信息**：

```
failed to connect to database
```

**解决方法**：

- 确保 PostgreSQL 数据库已启动
- 检查数据库连接参数是否正确
- 确保数据库用户有足够的权限

### 3. 测试失败：API 调用错误

**错误信息**：

```
API call failed: invalid API key
```

**解决方法**：

- 检查 API 密钥是否正确
- 确认百炼服务已开通
- 检查账户余额是否充足

### 4. 测试超时

**解决方法**：

- 检查网络连接
- 尝试使用不同的地域端点
- 增加测试超时时间

## 性能基准

基于 `qwen-plus` 模型的性能参考：

- **首次调用延迟**：1-2 秒（包含实例初始化）
- **后续调用延迟**：0.8-1.5 秒（使用缓存实例）
- **Token 生成速度**：约 20-30 tokens/秒
- **并发支持**：支持多个并发请求

## 下一步

完成非流式调用测试后，可以继续进行：

1. **流式调用测试**：`TASK-4.5: 测试百炼流式调用`
2. **API 层集成**：`TASK-5.1: 扩展 ChatOptions 支持模型名称`
3. **端到端测试**：`TASK-6.1: 端到端测试`

## 参考资料

- [阿里云百炼文档](https://help.aliyun.com/zh/model-studio/)
- [百炼 API 参考](https://help.aliyun.com/zh/model-studio/developer-reference/api-reference)
- [通义千问模型介绍](https://help.aliyun.com/zh/model-studio/getting-started/models)
- [Genkit 文档](https://firebase.google.com/docs/genkit)

## 联系支持

如果遇到问题，可以：

1. 查看项目 README
2. 查看百炼官方文档
3. 提交 Issue 到项目仓库
