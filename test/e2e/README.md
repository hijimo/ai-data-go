# 端到端测试 (E2E Tests)

## 概述

端到端测试模拟真实用户场景，从配置创建到模型调用的完整流程，确保整个系统的各个组件能够正确协同工作。

## 测试文件

### Azure OpenAI 端到端测试

- **文件**: `azure_e2e_test.go`
- **测试函数**: `TestAzureOpenAI_E2E_Complete`
- **测试脚本**: `../test_azure_e2e.sh`
- **文档**: `../../internal/genkit/AZURE_E2E_TEST_QUICK_REF.md`

### 百炼端到端测试

- **文件**: `bailian_e2e_test.go`
- **测试函数**: `TestBailian_E2E_Complete`
- **测试脚本**: `../test_bailian_e2e.sh`
- **文档**: `../../internal/genkit/BAILIAN_E2E_TEST_QUICK_REF.md`

### 提供商切换测试

- **文件**: `provider_switching_test.go`
- **测试函数**: `TestProviderSwitching`
- **测试脚本**: `../test_provider_switching.sh`
- **文档**: `../../internal/genkit/PROVIDER_SWITCHING_TEST_QUICK_REF.md`

### 默认提供商测试

- **文件**: `default_provider_test.go`
- **测试函数**: `TestDefaultProvider`
- **测试脚本**: `../test_default_provider.sh`
- **文档**: `../../internal/genkit/DEFAULT_PROVIDER_TEST_QUICK_REF.md`

### 错误场景测试

- **文件**: `error_scenarios_test.go`
- **测试函数**: `TestErrorScenarios`
- **测试脚本**: `../test_error_scenarios.sh`
- **文档**: `./ERROR_SCENARIOS_QUICK_REF.md`

## 环境配置

### Azure OpenAI 配置

```bash
# 必需的环境变量
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"

# 可选：API Version（默认: 2024-02-15-preview）
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"
```

### 百炼配置

```bash
# 必需的环境变量
export BAILIAN_API_KEY="your-api-key"

# 可选：Endpoint 和 Model（使用默认值）
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"
export BAILIAN_MODEL="qwen-plus"
```

### Google AI 配置（默认提供商）

```bash
# 必需的环境变量
export GOOGLE_API_KEY="your-google-api-key"
```

### 数据库配置

```bash
# 可选：数据库配置（使用默认值）
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

## 运行测试

### 方法 1: 使用测试脚本（推荐）

```bash
# Azure OpenAI 端到端测试
cd test
./test_azure_e2e.sh

# 百炼端到端测试
cd test
./test_bailian_e2e.sh

# 提供商切换测试
cd test
./test_provider_switching.sh

# 默认提供商测试
cd test
./test_default_provider.sh

# 错误场景测试
cd test
./test_error_scenarios.sh
```

### 方法 2: 使用 go test 命令

```bash
# 运行所有端到端测试
go test -v -timeout 10m ./test/e2e

# 运行特定的端到端测试
go test -v -timeout 5m ./test/e2e -run TestAzureOpenAI_E2E_Complete
go test -v -timeout 5m ./test/e2e -run TestBailian_E2E_Complete
go test -v -timeout 10m ./test/e2e -run TestProviderSwitching
go test -v -timeout 5m ./test/e2e -run TestDefaultProvider
go test -v -timeout 10m ./test/e2e -run TestErrorScenarios

# 跳过端到端测试（快速测试）
go test -v -short ./test/e2e
```

### 方法 3: 运行特定的子测试

```bash
# 运行特定阶段的测试
go test -v ./test/e2e -run TestAzureOpenAI_E2E_Complete/简单问答
go test -v ./test/e2e -run TestAzureOpenAI_E2E_Complete/流式响应
go test -v ./test/e2e -run TestAzureOpenAI_E2E_Complete/并发调用
```

## 测试阶段

### Azure OpenAI 端到端测试包含 9 个阶段

1. **设置测试环境** - 创建数据库连接
2. **创建租户和模型配置** - 创建测试数据
3. **初始化 Genkit Client** - 初始化客户端
4. **测试非流式调用** - 测试基本文本生成
5. **测试流式调用** - 测试流式响应
6. **测试缓存机制** - 验证实例缓存
7. **测试错误处理** - 测试各种错误场景
8. **测试并发场景** - 测试并发请求
9. **测试多轮对话** - 测试连续对话

### 百炼端到端测试包含 10 个阶段

1. **设置测试环境** - 创建数据库连接
2. **创建租户和模型配置** - 创建测试数据
3. **初始化 Genkit Client** - 初始化客户端
4. **测试非流式调用** - 测试基本文本生成和中文处理
5. **测试流式调用** - 测试流式响应和中文流式输出
6. **测试缓存机制** - 验证实例缓存
7. **测试错误处理** - 测试各种错误场景
8. **测试并发场景** - 测试并发请求
9. **测试多轮对话** - 测试连续对话
10. **测试复杂中文场景** - 测试古诗词、成语等中文特色场景

### 默认提供商测试包含 9 个阶段

1. **设置测试环境** - 创建数据库连接
2. **创建租户和默认模型配置** - 创建 Google AI 默认配置
3. **初始化 Genkit Client** - 初始化客户端
4. **测试使用默认提供商** - 测试非流式和流式调用
5. **测试默认模型的参数传递** - 测试自定义参数
6. **测试默认模型的并发调用** - 测试并发请求
7. **测试默认模型的错误处理** - 测试各种错误场景
8. **测试默认模型的性能** - 测量响应时间
9. **测试默认模型的缓存机制** - 验证实例缓存

### 提供商切换测试包含 10 个阶段

1. **设置测试环境** - 创建数据库连接
2. **创建租户和多个模型配置** - 为同一租户创建多个提供商配置
3. **初始化 Genkit Client** - 初始化客户端
4. **测试基本的提供商切换** - 顺序切换和快速切换
5. **测试流式调用的提供商切换** - 流式调用时切换提供商
6. **测试并发切换提供商** - 并发使用不同提供商
7. **测试提供商切换的性能** - 测量切换延迟
8. **测试提供商切换的错误处理** - 测试各种错误场景
9. **测试不同提供商的参数传递** - 不同提供商使用不同参数
10. **测试提供商切换的一致性** - 验证响应一致性

### 错误场景测试包含 13 个阶段

1. **设置测试环境** - 创建数据库连接
2. **创建测试数据** - 创建有效和无效的模型配置
3. **初始化 Genkit Client** - 初始化客户端
4. **测试配置相关错误** - 配置不存在、已禁用、已删除、JSON格式错误
5. **测试租户相关错误** - 租户ID无效、不存在
6. **测试API密钥相关错误** - 密钥为空、无效
7. **测试提供商相关错误** - 不支持的提供商类型
8. **测试参数相关错误** - Temperature、MaxTokens超出范围
9. **测试输入相关错误** - 空提示词、超长提示词
10. **测试上下文相关错误** - 上下文取消、超时
11. **测试流式调用错误** - 配置不存在、租户ID无效、上下文取消
12. **测试并发错误场景** - 并发调用不存在/禁用的模型
13. **测试边界条件** - 特殊字符、超长名称

## 测试覆盖

### 功能覆盖

- ✅ 模型配置创建和查询
- ✅ 非流式文本生成
- ✅ 流式文本生成
- ✅ 参数传递
- ✅ Token 使用统计
- ✅ 实例缓存机制
- ✅ 错误处理
- ✅ 并发调用
- ✅ 多轮对话
- ✅ 流式中断处理
- ✅ 提供商切换
- ✅ 多提供商并发使用
- ✅ 默认提供商逻辑
- ✅ 向后兼容性

### 场景覆盖

- ✅ 租户创建模型配置
- ✅ 用户发送各种类型的请求
- ✅ 系统处理正常和异常情况
- ✅ 多用户并发访问
- ✅ 长时间运行的对话
- ✅ 同一租户使用多个提供商
- ✅ 在不同提供商之间切换
- ✅ 禁用一个提供商后切换到另一个
- ✅ 使用默认提供商（未指定模型时）
- ✅ 默认提供商的缓存和性能
- ✅ 各种错误场景的处理
- ✅ 配置、租户、API密钥相关错误
- ✅ 参数、输入、上下文相关错误
- ✅ 流式调用和并发场景的错误处理
- ✅ 边界条件测试

## 预期结果

### 成功标准

- 所有测试阶段都应该通过
- 非流式调用返回有效的文本响应
- 流式调用返回多个数据块
- Token 统计准确
- 错误处理返回明确的错误信息
- 并发调用全部成功
- 多轮对话保持上下文

### 性能指标

- 非流式调用响应时间: < 10 秒
- 流式调用首字节时间: < 5 秒
- 并发调用成功率: 100%
- 测试总耗时: < 5 分钟

## 故障排查

### 常见问题

#### 1. 环境变量未设置

```
错误: 缺少环境变量 AZURE_OPENAI_API_KEY
解决: 设置所有必需的环境变量
```

#### 2. 数据库连接失败

```
错误: failed to connect to database
解决: 检查数据库配置和服务状态
```

#### 3. API 调用失败

```
错误: Azure OpenAI API call failed
解决: 
- 检查 API Key 是否有效
- 检查 Endpoint 和 Deployment 是否正确
- 检查网络连接
```

#### 4. 测试超时

```
错误: test timeout
解决: 增加超时时间 -timeout 10m
```

## 调试技巧

### 1. 查看详细日志

```bash
go test -v -timeout 5m ./test/e2e -run TestAzureOpenAI_E2E_Complete
```

### 2. 运行特定子测试

```bash
go test -v ./test/e2e -run TestAzureOpenAI_E2E_Complete/简单问答
```

### 3. 启用调试输出

```bash
export GENKIT_DEBUG=true
./test/test_azure_e2e.sh
```

## 测试数据清理

测试完成后会自动清理测试数据：

- 删除所有名称包含 "e2e" 的模型配置
- 不会影响生产数据

## 最佳实践

1. **使用独立的测试环境**
   - 使用专门的测试数据库
   - 使用测试专用的 AI 服务资源

2. **定期运行测试**
   - 在每次代码变更后运行
   - 在部署前运行完整测试

3. **监控测试结果**
   - 记录测试执行时间
   - 跟踪失败率和错误类型

4. **保护敏感信息**
   - 不要在代码中硬编码 API Key
   - 使用环境变量管理配置
   - 不要提交包含敏感信息的日志

## 相关文档

### Azure OpenAI

- [Azure OpenAI 端到端测试快速参考](../../internal/genkit/AZURE_E2E_TEST_QUICK_REF.md)
- [Azure OpenAI 集成测试](../../internal/genkit/AZURE_INTEGRATION_TEST_README.md)
- [Azure OpenAI 流式测试](../../internal/genkit/AZURE_STREAM_TEST_README.md)

### 百炼

- [百炼端到端测试快速参考](../../internal/genkit/BAILIAN_E2E_TEST_QUICK_REF.md)
- [百炼集成测试](../../internal/genkit/BAILIAN_INTEGRATION_TEST_README.md)
- [百炼流式测试](../../internal/genkit/BAILIAN_STREAM_TEST_README.md)

### 提供商切换

- [提供商切换测试快速参考](../../internal/genkit/PROVIDER_SWITCHING_TEST_QUICK_REF.md)

### 默认提供商

- [默认提供商测试快速参考](../../internal/genkit/DEFAULT_PROVIDER_TEST_QUICK_REF.md)
- [默认提供商测试详细文档](../../internal/genkit/DEFAULT_PROVIDER_TEST_README.md)

### 错误场景

- [错误场景测试快速参考](./ERROR_SCENARIOS_QUICK_REF.md)

## 贡献指南

### 添加新的端到端测试

1. 在 `test/e2e/` 目录下创建新的测试文件
2. 遵循现有的测试结构和命名规范
3. 包含完整的测试阶段和子测试
4. 创建对应的测试脚本
5. 编写快速参考文档

### 测试文件命名规范

- 端到端测试文件: `{provider}_e2e_test.go`
- 测试脚本: `test_{provider}_e2e.sh`
- 快速参考文档: `{PROVIDER}_E2E_TEST_QUICK_REF.md`

## 更新日志

### 2024-11-28

- ✅ 创建 Azure OpenAI 端到端测试
- ✅ 实现 9 个测试阶段
- ✅ 添加测试脚本和文档
- ✅ 创建百炼端到端测试
- ✅ 实现 10 个测试阶段（包含中文特色场景）
- ✅ 添加百炼测试脚本和文档
- ✅ 创建提供商切换测试
- ✅ 实现 10 个测试阶段（包含切换、并发、性能测试）
- ✅ 添加提供商切换测试脚本和文档
- ✅ 创建默认提供商测试
- ✅ 实现 9 个测试阶段（包含向后兼容性验证）
- ✅ 添加默认提供商测试脚本和文档

### 2024-11-29

- ✅ 创建错误场景测试
- ✅ 实现 13 个测试阶段（覆盖所有错误类型）
- ✅ 添加错误场景测试脚本和文档
- ✅ 测试配置、租户、API密钥相关错误
- ✅ 测试参数、输入、上下文相关错误
- ✅ 测试流式调用和并发场景的错误处理
- ✅ 测试边界条件
