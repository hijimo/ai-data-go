# Azure OpenAI 端到端测试快速参考

## 概述

Azure OpenAI 端到端测试模拟真实用户场景，从配置创建到模型调用的完整流程。

## 测试文件

- **测试代码**: `test/e2e/azure_e2e_test.go`
- **测试脚本**: `test/test_azure_e2e.sh`

## 环境变量配置

### 必需的环境变量

```bash
# Azure OpenAI 配置
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"

# 可选：API Version（默认: 2024-02-15-preview）
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"
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

### 使用测试脚本（推荐）

```bash
# 设置环境变量
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"

# 运行测试
./test/test_azure_e2e.sh
```

### 使用 go test 命令

```bash
# 运行完整的端到端测试
go test -v -timeout 5m ./test/e2e -run TestAzureOpenAI_E2E_Complete

# 跳过端到端测试（快速测试）
go test -v -short ./test/e2e
```

## 测试阶段

端到端测试包含以下 9 个阶段：

### 阶段 1: 设置测试环境

- 创建数据库连接
- 验证数据库可用性

### 阶段 2: 创建租户和模型配置

- 创建测试租户
- 创建 Azure OpenAI 模型配置
- 验证配置保存成功

### 阶段 3: 初始化 Genkit Client

- 创建 ModelConfigurationRepository
- 初始化 Genkit Client

### 阶段 4: 测试非流式调用

- **简单问答**: 测试基本的文本生成
- **带参数的调用**: 测试自定义 temperature 和 maxTokens
- **Token 统计**: 验证 Token 使用统计

### 阶段 5: 测试流式调用

- **流式响应**: 测试流式数据接收
- **流式中断**: 测试上下文取消机制

### 阶段 6: 测试缓存机制

- **实例缓存**: 验证 Genkit 实例缓存工作正常

### 阶段 7: 测试错误处理

- **配置不存在**: 测试查询不存在的模型配置
- **租户ID无效**: 测试无效的租户ID
- **模型已禁用**: 测试禁用模型的错误处理

### 阶段 8: 测试并发场景

- **并发调用**: 测试多个并发请求的处理

### 阶段 9: 测试多轮对话

- **多轮对话**: 测试连续的多轮对话能力

## 测试覆盖

### 功能覆盖

- ✅ 模型配置创建和查询
- ✅ 非流式文本生成
- ✅ 流式文本生成
- ✅ 参数传递（temperature, maxTokens）
- ✅ Token 使用统计
- ✅ 实例缓存机制
- ✅ 错误处理（配置不存在、租户无效、模型禁用）
- ✅ 并发调用
- ✅ 多轮对话
- ✅ 流式中断处理

### 场景覆盖

- ✅ 租户创建模型配置
- ✅ 用户发送简单问题
- ✅ 用户发送带参数的请求
- ✅ 用户使用流式接口
- ✅ 用户取消流式请求
- ✅ 多个用户并发请求
- ✅ 用户进行多轮对话
- ✅ 系统处理各种错误情况

## 预期结果

### 成功标准

- 所有 9 个测试阶段都应该通过
- 非流式调用应该返回有效的文本响应
- 流式调用应该返回多个数据块
- Token 统计应该准确
- 错误处理应该返回明确的错误信息
- 并发调用应该全部成功
- 多轮对话应该保持上下文

### 性能指标

- 非流式调用响应时间: < 10 秒
- 流式调用首字节时间: < 5 秒
- 并发调用成功率: 100%

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

### 调试技巧

1. **查看详细日志**

   ```bash
   go test -v -timeout 5m ./test/e2e -run TestAzureOpenAI_E2E_Complete
   ```

2. **运行特定子测试**

   ```bash
   go test -v ./test/e2e -run TestAzureOpenAI_E2E_Complete/简单问答
   ```

3. **启用调试输出**

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
   - 使用测试专用的 Azure OpenAI 资源

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

- [Azure OpenAI 集成测试](./AZURE_INTEGRATION_TEST_README.md)
- [Azure OpenAI 流式测试](./AZURE_STREAM_TEST_README.md)
- [Azure OpenAI 配置指南](../../docs/AZURE_OPENAI_CONFIGURATION_GUIDE.md)

## 更新日志

### 2024-11-28

- ✅ 创建 Azure OpenAI 端到端测试
- ✅ 实现 9 个测试阶段
- ✅ 添加测试脚本和文档
