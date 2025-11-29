# 性能测试快速参考

## 概述

性能测试用于衡量不同 AI 模型提供商的调用延迟和性能特征。

## 测试内容

### 1. Google AI 延迟测试

- 测量 Google AI (Gemini) 的平均响应时间
- 统计最小、最大、中位数延迟
- 计算标准差

### 2. Azure OpenAI 延迟测试

- 测量 Azure OpenAI (GPT-4) 的平均响应时间
- 统计最小、最大、中位数延迟
- 计算标准差

### 3. 百炼延迟测试

- 测量阿里云百炼 (Qwen) 的平均响应时间
- 统计最小、最大、中位数延迟
- 计算标准差

### 4. 流式调用 TTFB 测试

- 测量流式调用的首字节时间 (Time To First Byte)
- 验证流式响应的启动延迟

## 环境变量

### Google AI

```bash
export GOOGLE_API_KEY="your-google-api-key"
```

### Azure OpenAI

```bash
export AZURE_OPENAI_API_KEY="your-azure-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"  # 可选
```

### 阿里云百炼

```bash
export BAILIAN_API_KEY="your-bailian-api-key"
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"  # 可选
export BAILIAN_MODEL="qwen-plus"  # 可选
```

### 数据库配置

```bash
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="postgres"
export DB_PASSWORD="postgres"
export DB_NAME="genkit_test"
```

## 运行测试

### 使用脚本运行

```bash
./test/test_performance.sh
```

### 使用 go test 运行

```bash
# 运行所有性能测试
go test -v -timeout 10m ./test/e2e -run TestSingleProviderLatency

# 只运行 Google AI 延迟测试
go test -v -timeout 5m ./test/e2e -run TestSingleProviderLatency/Google

# 只运行 Azure OpenAI 延迟测试
go test -v -timeout 5m ./test/e2e -run TestSingleProviderLatency/Azure

# 只运行百炼延迟测试
go test -v -timeout 5m ./test/e2e -run TestSingleProviderLatency/百炼

# 只运行流式 TTFB 测试
go test -v -timeout 5m ./test/e2e -run TestSingleProviderLatency/流式
```

## 测试流程

### 每个提供商的测试流程

1. **环境检查**
   - 验证必需的环境变量是否设置
   - 如果缺少环境变量，跳过该提供商的测试

2. **配置创建**
   - 创建测试租户
   - 创建模型配置
   - 初始化 Genkit Client

3. **预热调用**
   - 执行一次预热调用
   - 排除首次初始化的开销

4. **延迟测量**
   - 执行 5 次调用
   - 记录每次调用的延迟
   - 使用相同的简短提示词

5. **统计计算**
   - 计算平均延迟
   - 计算最小/最大延迟
   - 计算中位数
   - 计算标准差

6. **结果验证**
   - 验证平均延迟 < 5 秒
   - 验证所有调用都成功

### 流式 TTFB 测试流程

1. **配置创建**
   - 创建测试租户和配置

2. **预热调用**
   - 执行一次流式预热调用

3. **TTFB 测量**
   - 执行 5 次流式调用
   - 记录每次调用的首字节时间
   - 消费所有数据块

4. **统计计算**
   - 计算平均 TTFB
   - 计算最小/最大 TTFB
   - 计算中位数和标准差

5. **结果验证**
   - 验证平均 TTFB < 2 秒

## 性能指标

### 延迟统计

- **平均延迟 (Average)**: 所有调用的平均响应时间
- **最小延迟 (Min)**: 最快的响应时间
- **最大延迟 (Max)**: 最慢的响应时间
- **中位数 (Median)**: 排序后中间位置的延迟值
- **标准差 (StdDev)**: 延迟的波动程度

### 性能目标

- **非流式调用**: 平均延迟 < 5 秒
- **流式调用 TTFB**: 平均 < 2 秒

## 测试输出示例

```
========== Google AI 延迟测试 ==========
预热调用...
开始测量 5 次调用的延迟...
  第 1 次调用延迟: 1.234s
  第 2 次调用延迟: 1.156s
  第 3 次调用延迟: 1.289s
  第 4 次调用延迟: 1.198s
  第 5 次调用延迟: 1.245s

✓ Google AI 延迟统计:
  平均延迟: 1.224s
  最小延迟: 1.156s
  最大延迟: 1.289s
  中位数: 1.234s
  标准差: 45.678ms
```

## 故障排查

### 测试超时

- 增加超时时间：`-timeout 15m`
- 检查网络连接
- 验证 API 密钥是否有效

### 延迟过高

- 检查网络延迟
- 验证 API 配额是否充足
- 检查提供商服务状态

### 测试失败

- 查看详细错误信息
- 验证环境变量配置
- 检查数据库连接

## 注意事项

1. **API 配额**
   - 性能测试会消耗 API 配额
   - 每个提供商测试约 6 次调用（1 次预热 + 5 次测量）

2. **网络影响**
   - 测试结果受网络延迟影响
   - 建议在稳定的网络环境下运行

3. **并发限制**
   - 测试是串行执行的
   - 不会触发并发限制

4. **数据库清理**
   - 测试结束后自动清理测试数据
   - 不会影响生产数据

## 相关文件

- `test/e2e/performance_test.go` - 性能测试实现
- `test/test_performance.sh` - 测试运行脚本
- `.kiro/specs/genkit-multi-model-support/tasks.md` - 任务列表

## 提供商切换延迟测试

### 测试内容

- 测量在同一租户下切换不同提供商时的延迟
- 对比切换延迟和基准延迟
- 测试快速连续切换的性能

### 运行测试

```bash
# 使用脚本运行
./test/test_provider_switching.sh

# 使用 go test 运行
go test -v -timeout 10m ./test/e2e -run TestProviderSwitchingLatency
```

### 环境要求

- 至少需要配置两个提供商
- 建议配置所有三个提供商以获得完整的测试覆盖

### 测试流程

1. **环境检查**
   - 验证至少有两个提供商可用
   - 如果少于两个提供商，跳过测试

2. **配置创建**
   - 为同一租户创建多个提供商配置
   - 每个提供商使用不同的模型

3. **预热所有提供商**
   - 对每个提供商执行一次预热调用
   - 确保所有实例都已初始化

4. **测量切换延迟**
   - 执行 10 次提供商切换
   - 记录每次切换的延迟

5. **基准测试**
   - 使用同一提供商执行 10 次连续调用
   - 作为性能基准

6. **对比分析**
   - 计算切换开销
   - 验证开销是否在可接受范围内（< 50ms）

7. **快速连续切换测试**
   - 执行 20 次快速连续切换
   - 验证性能稳定性

### 性能目标

- **切换开销**: < 50ms（相对于基准延迟）
- **快速切换**: 性能不应显著下降

### 测试输出示例

```
========== 提供商切换延迟测试 ==========
✓ 共创建 3 个提供商配置
✓ 所有提供商预热完成

开始测量 10 次提供商切换的延迟...
  第 1 次切换 (googlegenai -> azureopenai): 1.234s
  第 2 次切换 (azureopenai -> bianlian): 1.156s
  ...

✓ 提供商切换延迟统计:
  平均延迟: 1.224s
  最小延迟: 1.156s
  最大延迟: 1.289s

✓ 同一提供商连续调用延迟统计（基准）:
  平均延迟: 1.200s
  最小延迟: 1.150s
  最大延迟: 1.250s

✓ 提供商切换开销分析:
  切换平均延迟: 1.224s
  基准平均延迟: 1.200s
  额外开销: 24ms (2.00%)

✓ 提供商切换开销 (24ms) 在可接受范围内（< 50ms）
```

## 下一步

完成性能测试后，可以继续：

- TASK-6.3: 错误处理完善
- TASK-6.4: 日志和监控完善
