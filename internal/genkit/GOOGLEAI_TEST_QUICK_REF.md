# Google AI (Gemini) 集成测试快速参考

## 概述

本文档提供 Google AI (Gemini) 集成测试的快速参考指南。

## 测试文件

- **测试文件**: `internal/genkit/googleai_integration_test.go`
- **测试脚本**: `test/test_googleai.sh`

## 环境变量配置

### 必需的环境变量

```bash
# Google AI API 密钥（必需）
export GOOGLE_API_KEY="your_google_api_key"

# Google AI 模型名称（可选，默认: gemini-2.0-flash-exp）
export GOOGLE_MODEL="gemini-2.0-flash-exp"
```

### 数据库配置（可选）

```bash
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

## 运行测试

### 方式 1: 使用测试脚本（推荐）

```bash
# 设置环境变量
export GOOGLE_API_KEY="your_google_api_key"

# 运行测试脚本
./test/test_googleai.sh
```

### 方式 2: 直接运行 Go 测试

```bash
# 运行所有 Google AI 测试
go test -v -run TestGoogleAIIntegration ./internal/genkit/

# 只运行非流式测试
go test -v -run TestGoogleAIIntegration_NonStreaming ./internal/genkit/

# 只运行流式测试
go test -v -run TestGoogleAIIntegration_Streaming ./internal/genkit/
```

### 方式 3: 运行特定的测试用例

```bash
# 运行基本文本生成测试
go test -v -run TestGoogleAIIntegration_NonStreaming/基本文本生成 ./internal/genkit/

# 运行流式生成测试
go test -v -run TestGoogleAIIntegration_Streaming/基本流式生成 ./internal/genkit/

# 运行中文处理测试
go test -v -run TestGoogleAIIntegration_NonStreaming/中文处理能力测试 ./internal/genkit/
```

## 测试覆盖范围

### 非流式调用测试 (TestGoogleAIIntegration_NonStreaming)

1. **基本文本生成** - 测试基本的文本生成功能
2. **参数传递测试** - 测试 temperature 和 maxTokens 参数
3. **Token 统计测试** - 验证 Token 使用统计
4. **中文处理能力测试** - 测试中文理解和生成
5. **错误处理 - 配置不存在** - 测试配置不存在的错误处理
6. **错误处理 - 租户ID无效** - 测试无效租户ID的错误处理
7. **错误处理 - 模型已禁用** - 测试模型禁用的错误处理
8. **响应格式验证** - 验证响应格式的正确性
9. **缓存机制测试** - 测试实例缓存机制
10. **多轮对话测试** - 测试多轮对话能力
11. **长文本生成测试** - 测试生成较长文本
12. **特殊字符处理测试** - 测试特殊字符的处理
13. **并发调用测试** - 测试并发调用的安全性

### 流式调用测试 (TestGoogleAIIntegration_Streaming)

1. **基本流式生成** - 测试基本的流式生成功能
2. **流式响应完整性测试** - 验证流式响应的完整性
3. **流式中断处理测试** - 测试流式请求的中断处理
4. **流式参数传递测试** - 测试流式调用的参数传递
5. **流式中文输出测试** - 测试中文流式输出
6. **流式错误处理 - 配置不存在** - 测试配置不存在的错误处理
7. **流式错误处理 - 租户ID无效** - 测试无效租户ID的错误处理
8. **流式错误处理 - 模型已禁用** - 测试模型禁用的错误处理
9. **流式性能测试 - TTFB** - 测试首字节时间
10. **流式并发调用测试** - 测试并发流式调用

## 测试数据

测试会自动创建以下测试数据：

- **测试租户**: 随机生成的 UUID
- **测试模型配置**:
  - 非流式测试: `google-gemini-test`
  - 流式测试: `google-gemini-stream-test`

测试完成后会自动清理测试数据。

## 预期结果

### 成功的测试输出示例

```
=== RUN   TestGoogleAIIntegration_NonStreaming
=== RUN   TestGoogleAIIntegration_NonStreaming/基本文本生成
    googleai_integration_test.go:XX: 生成的文本: 我是 Gemini，一个由 Google 开发的大型语言模型。
=== RUN   TestGoogleAIIntegration_NonStreaming/Token_统计测试
    googleai_integration_test.go:XX: Token 使用情况: Prompt=10, Completion=15, Total=25
--- PASS: TestGoogleAIIntegration_NonStreaming (XX.XXs)
    --- PASS: TestGoogleAIIntegration_NonStreaming/基本文本生成 (X.XXs)
    --- PASS: TestGoogleAIIntegration_NonStreaming/Token_统计测试 (X.XXs)
```

## 常见问题

### 1. 测试失败：缺少 GOOGLE_API_KEY

**错误信息**:

```
跳过 Google AI 集成测试：缺少必需的环境变量 GOOGLE_API_KEY
```

**解决方案**:

```bash
export GOOGLE_API_KEY="your_google_api_key"
```

### 2. 测试失败：数据库连接错误

**错误信息**:

```
failed to connect to database
```

**解决方案**:

```bash
# 检查数据库是否运行
# 检查数据库连接配置
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

### 3. 测试失败：API 调用超时

**可能原因**:

- 网络连接问题
- API 密钥无效
- Google AI 服务不可用

**解决方案**:

- 检查网络连接
- 验证 API 密钥是否有效
- 检查 Google AI 服务状态

### 4. 测试失败：模型不存在

**错误信息**:

```
model not found
```

**解决方案**:

```bash
# 使用有效的模型名称
export GOOGLE_MODEL="gemini-2.0-flash-exp"
# 或
export GOOGLE_MODEL="gemini-1.5-pro"
```

## 性能基准

### 非流式调用

- **平均响应时间**: 1-3 秒
- **Token 使用**: 根据提示词和响应长度而定

### 流式调用

- **首字节时间 (TTFB)**: < 2 秒
- **总响应时间**: 根据生成内容长度而定

## 注意事项

1. **API 配额**: 注意 Google AI API 的配额限制
2. **测试数据**: 测试会创建临时数据，测试完成后自动清理
3. **并发测试**: 并发测试可能会消耗较多 API 配额
4. **网络依赖**: 测试需要网络连接到 Google AI 服务

## 相关文档

- [Google AI (Gemini) 官方文档](https://ai.google.dev/docs)
- [Genkit 文档](https://firebase.google.com/docs/genkit)
- [任务列表](../../.kiro/specs/genkit-multi-model-support/tasks.md)
- [设计文档](../../.kiro/specs/genkit-multi-model-support/design.md)

## 更新日志

- **2025-11-28**: 创建 Google AI 集成测试套件
  - 添加非流式调用测试（13个测试用例）
  - 添加流式调用测试（10个测试用例）
  - 创建测试脚本和快速参考文档
