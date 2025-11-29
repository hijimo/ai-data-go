# 默认提供商测试文档

## 概述

本文档描述了默认提供商逻辑的测试实现。默认提供商测试验证了当系统未指定特定模型时，能够正确使用默认的 Google AI (Gemini) 模型。

## 测试目标

验证以下默认提供商逻辑：

1. **基本功能**：使用默认模型名称进行非流式调用
2. **流式调用**：使用默认模型进行流式调用
3. **参数传递**：默认模型支持自定义参数（temperature、maxTokens等）
4. **并发调用**：默认模型支持并发请求
5. **错误处理**：正确处理默认模型被禁用或不存在的情况
6. **性能测试**：测量默认模型的响应时间
7. **缓存机制**：验证默认模型实例被正确缓存

## 测试文件

- **测试代码**：`test/e2e/default_provider_test.go`
- **测试脚本**：`test/test_default_provider.sh`
- **文档**：`internal/genkit/DEFAULT_PROVIDER_TEST_README.md`（本文件）

## 前置条件

### 环境变量

测试需要以下环境变量：

```bash
export GOOGLE_API_KEY='your-google-api-key'
```

### 数据库

测试会自动创建和清理测试数据库，无需手动配置。

## 运行测试

### 使用测试脚本（推荐）

```bash
# 设置环境变量
export GOOGLE_API_KEY='your-google-api-key'

# 运行测试
./test/test_default_provider.sh
```

### 直接使用 go test

```bash
# 设置环境变量
export GOOGLE_API_KEY='your-google-api-key'

# 运行测试
go test -v -timeout 5m -run TestDefaultProvider ./test/e2e/
```

### 跳过测试

如果要跳过端到端测试（例如在 CI 环境中），可以使用 `-short` 标志：

```bash
go test -short ./test/e2e/
```

## 测试场景

### 阶段 1: 设置测试环境

- 连接测试数据库
- 验证数据库连接成功

### 阶段 2: 创建租户和默认模型配置

- 创建测试租户
- 创建默认的 Google AI (Gemini) 模型配置
- 模型名称：`gemini-pro`
- 模型：`gemini-1.5-pro`
- 提供商：`googlegenai`

### 阶段 3: 初始化 Genkit Client

- 创建 ModelConfigurationRepository
- 初始化 Genkit Client

### 阶段 4: 测试使用默认提供商

#### 测试用例 4.1: 使用默认模型名称

- 使用默认模型名称 `gemini-pro` 进行调用
- 验证响应成功
- 验证返回的模型名称为 `gemini-1.5-pro`

#### 测试用例 4.2: 流式调用使用默认模型

- 使用默认模型进行流式调用
- 验证接收到多个数据块
- 验证最终模型名称正确

### 阶段 5: 测试默认模型的参数传递

#### 测试用例 5.1: 默认模型使用自定义参数

- 设置自定义 temperature 和 maxTokens
- 验证参数正确传递
- 验证响应成功

### 阶段 6: 测试默认模型的并发调用

#### 测试用例 6.1: 并发使用默认模型

- 发起 5 个并发请求
- 验证所有请求都成功
- 验证所有响应的模型名称一致

### 阶段 7: 测试默认模型的错误处理

#### 测试用例 7.1: 禁用默认模型后的错误处理

- 禁用默认模型配置
- 尝试使用默认模型
- 验证返回"模型已禁用"错误
- 恢复默认模型配置

#### 测试用例 7.2: 使用不存在的模型名称

- 使用不存在的模型名称
- 验证返回"模型配置"相关错误

### 阶段 8: 测试默认模型的性能

#### 测试用例 8.1: 测量默认模型的响应时间

- 预热：发起一次调用
- 测量：发起 3 次调用并记录耗时
- 计算平均响应时间
- 输出性能统计

### 阶段 9: 测试默认模型的缓存机制

#### 测试用例 9.1: 验证默认模型实例被缓存

- 第一次调用：初始化实例
- 第二次调用：使用缓存的实例
- 比较两次调用的耗时
- 验证缓存机制正常工作

## 测试结果示例

```
========== 阶段 1: 设置测试环境 ==========
✓ 数据库连接成功

========== 阶段 2: 创建租户和默认模型配置 ==========
创建测试租户: 12345678-1234-1234-1234-123456789012
✓ 默认模型配置创建成功: gemini-pro

========== 阶段 3: 初始化 Genkit Client ==========
✓ Genkit Client 初始化成功

========== 阶段 4: 测试使用默认提供商 ==========
=== RUN   TestDefaultProvider/使用默认模型名称
✓ 默认模型响应: 我是 Gemini，一个由 Google 开发的大型语言模型。
✓ 使用的模型: gemini-1.5-pro
--- PASS: TestDefaultProvider/使用默认模型名称 (2.34s)

=== RUN   TestDefaultProvider/流式调用使用默认模型
✓ 默认模型流式响应成功，接收 15 个数据块
✓ 使用的模型: gemini-1.5-pro
--- PASS: TestDefaultProvider/流式调用使用默认模型 (3.12s)

========== 阶段 5: 测试默认模型的参数传递 ==========
=== RUN   TestDefaultProvider/默认模型使用自定义参数
✓ 默认模型使用自定义参数成功
  Temperature: 0.8, MaxTokens: 500
--- PASS: TestDefaultProvider/默认模型使用自定义参数 (2.56s)

========== 阶段 6: 测试默认模型的并发调用 ==========
=== RUN   TestDefaultProvider/并发使用默认模型
✓ 并发使用默认模型成功: 5/5
--- PASS: TestDefaultProvider/并发使用默认模型 (3.45s)

========== 阶段 7: 测试默认模型的错误处理 ==========
=== RUN   TestDefaultProvider/禁用默认模型后的错误处理
✓ 禁用的默认模型错误处理正常: 获取模型实例失败: 模型已禁用: gemini-pro
--- PASS: TestDefaultProvider/禁用默认模型后的错误处理 (0.12s)

=== RUN   TestDefaultProvider/使用不存在的模型名称
✓ 不存在的模型错误处理正常: 获取模型实例失败: 获取模型配置失败: record not found
--- PASS: TestDefaultProvider/使用不存在的模型名称 (0.08s)

========== 阶段 8: 测试默认模型的性能 ==========
=== RUN   TestDefaultProvider/测量默认模型的响应时间
  第 1 次调用耗时: 2.345s
  第 2 次调用耗时: 2.123s
  第 3 次调用耗时: 2.234s
✓ 平均响应时间: 2.234s
  总调用次数: 3
--- PASS: TestDefaultProvider/测量默认模型的响应时间 (8.12s)

========== 阶段 9: 测试默认模型的缓存机制 ==========
=== RUN   TestDefaultProvider/验证默认模型实例被缓存
  第一次调用耗时: 2.456s
  第二次调用耗时: 2.123s
✓ 默认模型实例缓存机制正常工作
--- PASS: TestDefaultProvider/验证默认模型实例被缓存 (4.58s)

========== 默认提供商测试完成 ==========
✓ 所有测试阶段通过
✓ 验证了默认模型: gemini-pro

PASS
ok      genkit-ai-service/test/e2e      28.456s
```

## 验证的需求

本测试验证了以下需求：

### 需求 5: 保持向后兼容

- ✅ 当使用现有的 API 接口时，系统继续正常工作
- ✅ 当未配置新的提供商时，系统默认使用 Google AI
- ✅ 当调用现有的流式接口时，响应格式保持不变

### FR-2: Genkit 客户端扩展

- ✅ 客户端接口保持不变，确保向后兼容
- ✅ 客户端能够根据配置动态选择插件

## 技术实现

### 默认模型配置

默认模型配置存储在 `model_configurations` 表中：

```sql
INSERT INTO model_configurations (
    tenant_id,
    name,
    model,
    model_provider,
    api_key,
    query_params,
    is_enabled
) VALUES (
    'tenant-uuid',
    'gemini-pro',
    'gemini-1.5-pro',
    'googlegenai',
    'your-api-key',
    '{"model": "gemini-1.5-pro", "defaultTemperature": 0.7, "defaultMaxTokens": 2048}',
    true
);
```

### 默认模型选择逻辑

在 `internal/service/ai/genkit_service.go` 中：

```go
// 从请求中获取模型名称
// 如果请求中指定了模型名称，使用它；否则使用默认模型
modelName := "gemini-pro" // 默认模型
if req.Options != nil && req.Options.ModelName != nil && *req.Options.ModelName != "" {
    modelName = *req.Options.ModelName
}
```

### 实例缓存机制

在 `internal/genkit/client.go` 中：

```go
// 生成缓存键
cacheKey := fmt.Sprintf("%s_%s", tenantID, modelName)

// 尝试从缓存获取实例
c.mu.RLock()
g, exists := c.instances[cacheKey]
c.mu.RUnlock()

if exists {
    // 使用缓存的实例
    return g, genkitConfig, nil
}

// 初始化新实例并缓存
// ...
c.instances[cacheKey] = g
```

## 故障排查

### 问题 1: 测试失败 - 缺少环境变量

**错误信息**：

```
❌ 错误: 缺少 GOOGLE_API_KEY 环境变量
```

**解决方案**：

```bash
export GOOGLE_API_KEY='your-google-api-key'
```

### 问题 2: 测试失败 - 数据库连接错误

**错误信息**：

```
数据库连接应该成功: dial tcp: connection refused
```

**解决方案**：

- 确保 PostgreSQL 服务正在运行
- 检查数据库连接配置
- 验证数据库用户权限

### 问题 3: 测试失败 - API 调用超时

**错误信息**：

```
context deadline exceeded
```

**解决方案**：

- 检查网络连接
- 验证 API 密钥是否有效
- 增加测试超时时间：`go test -timeout 10m`

### 问题 4: 测试失败 - 模型配置不存在

**错误信息**：

```
获取模型实例失败: 获取模型配置失败: record not found
```

**解决方案**：

- 确保测试正确创建了模型配置
- 检查租户ID是否正确
- 验证模型名称是否匹配

## 相关文档

- [需求文档](../../.kiro/specs/genkit-multi-model-support/requirements.md)
- [设计文档](../../.kiro/specs/genkit-multi-model-support/design.md)
- [任务列表](../../.kiro/specs/genkit-multi-model-support/tasks.md)
- [提供商切换测试](./PROVIDER_SWITCHING_TEST_QUICK_REF.md)
- [Azure E2E 测试](./AZURE_E2E_TEST_QUICK_REF.md)
- [百炼 E2E 测试](./BAILIAN_E2E_TEST_QUICK_REF.md)

## 后续工作

- [ ] 添加更多默认模型的边界测试
- [ ] 测试默认模型的配置更新场景
- [ ] 测试默认模型的故障转移逻辑
- [ ] 添加默认模型的性能基准测试
- [ ] 测试默认模型在高并发场景下的表现

## 总结

默认提供商测试全面验证了系统的向后兼容性和默认模型逻辑。测试覆盖了基本功能、流式调用、参数传递、并发调用、错误处理、性能和缓存机制等多个方面，确保默认提供商（Google AI Gemini）能够正常工作。
