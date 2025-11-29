# 提供商切换测试快速参考

## 概述

提供商切换测试验证系统能够在同一租户下使用不同的模型提供商，并正确切换。这是多模型支持的核心功能之一。

## 测试文件

- **测试文件**: `test/e2e/provider_switching_test.go`
- **测试函数**: `TestProviderSwitching`
- **测试脚本**: `test/test_provider_switching.sh`

## 环境配置

### 最低要求

至少需要配置**两个**提供商才能测试切换功能。

### Azure OpenAI 配置

```bash
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="your-deployment-name"
export AZURE_OPENAI_API_VERSION="2024-02-15-preview"  # 可选
```

### 百炼配置

```bash
export BAILIAN_API_KEY="your-api-key"
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"  # 可选
export BAILIAN_MODEL="qwen-plus"  # 可选
```

### 数据库配置

```bash
export DB_HOST="localhost"        # 可选，默认 localhost
export DB_PORT="5432"             # 可选，默认 5432
export DB_USER="postgres"         # 可选，默认 postgres
export DB_PASSWORD="postgres"     # 可选，默认 postgres
export DB_NAME="genkit_test"      # 可选，默认 genkit_test
```

## 快速开始

### 方法 1: 使用测试脚本（推荐）

```bash
cd test
./test_provider_switching.sh
```

### 方法 2: 使用 go test 命令

```bash
# 运行完整测试
go test -v -timeout 10m ./test/e2e -run TestProviderSwitching

# 运行特定子测试
go test -v ./test/e2e -run TestProviderSwitching/顺序切换提供商
go test -v ./test/e2e -run TestProviderSwitching/并发使用不同提供商
```

## 测试阶段

### 10 个测试阶段

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

## 测试覆盖

### 功能覆盖

- ✅ 顺序切换提供商
- ✅ 快速切换提供商（测试缓存）
- ✅ 流式调用切换提供商
- ✅ 并发使用不同提供商
- ✅ 切换性能测量
- ✅ 错误处理（不存在的提供商、禁用的提供商）
- ✅ 不同提供商使用不同参数
- ✅ 响应一致性验证

### 场景覆盖

- ✅ 同一租户使用多个提供商
- ✅ 在不同提供商之间快速切换
- ✅ 并发请求使用不同提供商
- ✅ 禁用一个提供商后切换到另一个
- ✅ 相同问题在不同提供商上的响应

## 子测试说明

### 1. 顺序切换提供商

测试按顺序使用每个提供商，验证基本切换功能。

```bash
go test -v ./test/e2e -run TestProviderSwitching/顺序切换提供商
```

### 2. 快速切换提供商

测试在不同提供商之间快速切换，验证缓存机制。

```bash
go test -v ./test/e2e -run TestProviderSwitching/快速切换提供商
```

### 3. 流式调用切换提供商

测试流式调用时切换提供商。

```bash
go test -v ./test/e2e -run TestProviderSwitching/流式调用切换提供商
```

### 4. 并发使用不同提供商

测试多个并发请求使用不同的提供商。

```bash
go test -v ./test/e2e -run TestProviderSwitching/并发使用不同提供商
```

### 5. 测量切换延迟

测量提供商切换的性能开销。

```bash
go test -v ./test/e2e -run TestProviderSwitching/测量切换延迟
```

### 6. 切换到不存在的提供商

测试错误处理：尝试使用不存在的提供商。

```bash
go test -v ./test/e2e -run TestProviderSwitching/切换到不存在的提供商
```

### 7. 禁用一个提供商后切换

测试禁用一个提供商后切换到另一个提供商。

```bash
go test -v ./test/e2e -run TestProviderSwitching/禁用一个提供商后切换
```

### 8. 不同提供商使用不同参数

测试不同提供商使用不同的生成参数。

```bash
go test -v ./test/e2e -run TestProviderSwitching/不同提供商使用不同参数
```

### 9. 相同问题不同提供商的响应

测试相同问题在不同提供商上的响应一致性。

```bash
go test -v ./test/e2e -run TestProviderSwitching/相同问题不同提供商的响应
```

## 预期结果

### 成功标准

- ✅ 所有提供商都能正常工作
- ✅ 切换提供商不会导致错误
- ✅ 缓存机制正常工作
- ✅ 并发切换不会出现竞态条件
- ✅ 错误处理返回明确的错误信息
- ✅ 所有提供商都返回有效响应

### 性能指标

- 切换延迟: < 100ms（使用缓存时）
- 并发调用成功率: 100%
- 测试总耗时: < 10 分钟

## 测试输出示例

```
========== 阶段 1: 设置测试环境 ==========
✓ 数据库连接成功

========== 阶段 2: 创建租户和多个模型配置 ==========
创建测试租户: 12345678-1234-1234-1234-123456789abc
✓ Azure OpenAI 配置创建成功: azure-gpt-4-switch
✓ 百炼配置创建成功: bailian-qwen-switch

========== 阶段 3: 初始化 Genkit Client ==========
✓ Genkit Client 初始化成功

========== 阶段 4: 测试基本的提供商切换 ==========
=== RUN   TestProviderSwitching/顺序切换提供商
  切换到提供商 1: azure-gpt-4-switch
  ✓ 提供商 azure-gpt-4-switch 响应: 我是一个AI助手...
  切换到提供商 2: bailian-qwen-switch
  ✓ 提供商 bailian-qwen-switch 响应: 我是通义千问...
✓ 顺序切换提供商成功

========== 阶段 6: 测试并发切换提供商 ==========
=== RUN   TestProviderSwitching/并发使用不同提供商
✓ 并发使用不同提供商成功: 4/4

========== 提供商切换测试完成 ==========
✓ 所有测试阶段通过
✓ 测试了 2 个提供商的切换
```

## 故障排查

### 常见问题

#### 1. 只配置了一个提供商

```
错误: 至少需要两个提供商的配置才能测试切换
解决: 配置至少两个提供商（Azure OpenAI 和百炼）
```

#### 2. 提供商切换失败

```
错误: 切换到提供商 X 失败
解决: 
- 检查提供商配置是否正确
- 检查 API Key 是否有效
- 查看详细错误日志
```

#### 3. 并发测试超时

```
错误: 并发调用超时
解决: 
- 增加超时时间
- 检查网络连接
- 减少并发数量
```

#### 4. 缓存问题

```
错误: 切换后仍使用旧提供商
解决: 
- 检查缓存键是否正确（tenantID_modelName）
- 清理测试数据后重试
```

## 调试技巧

### 1. 查看详细日志

```bash
go test -v -timeout 10m ./test/e2e -run TestProviderSwitching
```

### 2. 运行特定子测试

```bash
go test -v ./test/e2e -run TestProviderSwitching/顺序切换提供商
```

### 3. 启用调试输出

```bash
export GENKIT_DEBUG=true
./test/test_provider_switching.sh
```

### 4. 检查数据库状态

```sql
-- 查看测试租户的模型配置
SELECT id, tenant_id, name, model_provider, is_enabled 
FROM model_configurations 
WHERE name LIKE '%switch%';
```

## 测试数据清理

测试完成后会自动清理测试数据：

- 删除所有名称包含 "switch" 的模型配置
- 不会影响生产数据

## 最佳实践

1. **配置多个提供商**
   - 至少配置两个提供商以获得完整测试覆盖
   - 使用测试专用的 API Key

2. **定期运行测试**
   - 在每次修改提供商切换逻辑后运行
   - 在部署前运行完整测试

3. **监控性能**
   - 关注切换延迟
   - 跟踪并发调用成功率

4. **保护敏感信息**
   - 不要在代码中硬编码 API Key
   - 使用环境变量管理配置

## 相关文档

- [Azure OpenAI 端到端测试](AZURE_E2E_TEST_QUICK_REF.md)
- [百炼端到端测试](BAILIAN_E2E_TEST_QUICK_REF.md)
- [端到端测试 README](../../test/e2e/README.md)
- [多模型支持设计文档](../../.kiro/specs/genkit-multi-model-support/design.md)

## 更新日志

### 2024-11-28

- ✅ 创建提供商切换测试
- ✅ 实现 10 个测试阶段
- ✅ 添加测试脚本
- ✅ 创建快速参考文档
- ✅ 测试覆盖：顺序切换、快速切换、流式切换、并发切换、性能测试、错误处理
