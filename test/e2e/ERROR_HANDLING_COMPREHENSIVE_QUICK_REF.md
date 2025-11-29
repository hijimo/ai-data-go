# 全面错误处理测试 - 快速参考

## 概述

全面的错误处理测试套件，验证系统在各种异常情况下的错误处理能力和恢复能力。

## 测试覆盖

### 1. 网络相关错误

- ✅ 网络超时
- ✅ 无效的API端点
- ✅ 连接失败

### 2. Azure OpenAI 特定错误

- ✅ 缺少 endpoint 配置
- ✅ 缺少 deployment 配置
- ✅ 无效的 endpoint URL
- ✅ 无效的 API 版本

### 3. 百炼特定错误

- ✅ 无效的 endpoint
- ✅ 无效的 workspace
- ✅ API 密钥错误

### 4. 自定义 OpenAI 提供商错误

- ✅ 缺少 baseUrl 配置
- ✅ 无效的 baseUrl
- ✅ 认证失败

### 5. 速率限制错误

- ✅ 并发请求触发速率限制
- ✅ 速率限制后的重试
- ✅ 429 错误处理

### 6. 内容过滤错误

- ✅ 敏感内容被拒绝
- ✅ 内容安全策略
- ✅ 过滤后的错误信息

### 7. 并发安全性

- ✅ 并发初始化同一模型
- ✅ 并发调用不同模型
- ✅ 缓存一致性
- ✅ 线程安全

### 8. 资源清理

- ✅ 客户端关闭后的调用
- ✅ 资源泄漏检测
- ✅ 缓存清理

### 9. 错误恢复

- ✅ 错误后的正常调用
- ✅ 流式错误后的恢复
- ✅ 多次错误后的恢复
- ✅ 状态一致性

### 10. 配置错误（已在 error_scenarios_test.go 中覆盖）

- ✅ 配置不存在
- ✅ 模型已禁用
- ✅ 模型已删除
- ✅ 配置 JSON 格式错误

### 11. 租户相关错误（已在 error_scenarios_test.go 中覆盖）

- ✅ 租户ID无效
- ✅ 租户ID不存在
- ✅ 跨租户访问

### 12. API密钥错误（已在 error_scenarios_test.go 中覆盖）

- ✅ API密钥为空
- ✅ API密钥无效
- ✅ API密钥过期

## 快速开始

### 运行所有错误处理测试

```bash
# 运行全面错误处理测试
./test/test_error_handling_comprehensive.sh

# 运行基础错误场景测试
./test/test_error_scenarios.sh
```

### 运行特定测试

```bash
# 只运行网络错误测试
go test -v -run TestComprehensiveErrorHandling/网络超时 ./test/e2e/

# 只运行Azure错误测试
go test -v -run TestComprehensiveErrorHandling/Azure ./test/e2e/

# 只运行并发安全测试
go test -v -run TestComprehensiveErrorHandling/并发 ./test/e2e/
```

## 环境变量

```bash
# 必需
export DATABASE_URL='host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable'

# 可选（用于实际API调用测试）
export GOOGLE_API_KEY='your-google-api-key'
export AZURE_OPENAI_KEY='your-azure-api-key'
export AZURE_OPENAI_ENDPOINT='https://your-resource.openai.azure.com'
export BAILIAN_API_KEY='your-bailian-api-key'
```

## 测试结果示例

```
========== 阶段 1: 设置测试环境 ==========
✓ 数据库连接成功

========== 阶段 2: 创建测试数据 ==========
✓ 有效模型配置创建成功: valid-model

========== 阶段 3: 初始化 Genkit Client ==========
✓ Genkit Client 初始化成功

========== 阶段 4: 测试网络相关错误 ==========
✓ 网络超时错误: context deadline exceeded
✓ 无效API端点错误: connection refused

========== 阶段 5: 测试Azure特定错误 ==========
✓ Azure缺少endpoint错误: 配置验证失败: Azure OpenAI 配置缺少必需字段: azureEndpoint
✓ Azure缺少deployment错误: 配置验证失败: Azure OpenAI 配置缺少必需字段: azureDeployment

========== 阶段 6: 测试百炼特定错误 ==========
✓ 百炼无效endpoint错误: connection refused

========== 阶段 7: 测试自定义OpenAI提供商错误 ==========
✓ 自定义OpenAI缺少baseUrl错误: 自定义 OpenAI 提供商必须指定 baseUrl

========== 阶段 8: 测试速率限制错误 ==========
✓ 速率限制测试完成: 成功=8, 速率限制=2, 总计=10

========== 阶段 9: 测试内容过滤错误 ==========
✓ 敏感内容过滤测试完成

========== 阶段 10: 测试并发安全性 ==========
✓ 并发初始化测试完成: 成功=10, 失败=0

========== 阶段 11: 测试资源清理 ==========
  注意：关闭后仍然可以调用（会重新初始化）

========== 阶段 12: 测试错误恢复 ==========
✓ 错误后的正常调用成功

========== 阶段 13: 测试流式错误恢复 ==========
✓ 流式错误后的正常调用成功

========== 全面错误处理测试完成 ==========
✓ 所有错误处理场景测试通过
✓ 验证了系统的错误处理能力和恢复能力
```

## 错误类型分类

### 1. 配置错误

- 配置不存在
- 配置格式错误
- 必需字段缺失
- 配置验证失败

### 2. 认证错误

- API密钥无效
- API密钥过期
- 认证失败
- 权限不足

### 3. 网络错误

- 连接超时
- 连接失败
- DNS解析失败
- 网络不可达

### 4. API错误

- 速率限制（429）
- 服务不可用（503）
- 内部错误（500）
- 请求格式错误（400）

### 5. 业务错误

- 模型已禁用
- 租户不存在
- 内容被过滤
- 参数超出范围

## 错误处理最佳实践

### 1. 错误信息应该清晰明确

```go
// ❌ 不好的错误信息
return fmt.Errorf("error")

// ✅ 好的错误信息
return fmt.Errorf("获取模型配置失败: 租户ID=%s, 模型名称=%s: %w", tenantID, modelName, err)
```

### 2. 错误应该包含上下文

```go
// ❌ 丢失上下文
return err

// ✅ 保留上下文
return fmt.Errorf("初始化提供商失败: %w", err)
```

### 3. 错误应该可以恢复

```go
// ✅ 错误后系统仍然可用
if err != nil {
    logger.ErrorContext(ctx, "调用失败", "error", err)
    // 清理状态，准备下次调用
    return err
}
```

### 4. 敏感信息不应该出现在错误中

```go
// ❌ 泄露敏感信息
return fmt.Errorf("API密钥 %s 无效", apiKey)

// ✅ 脱敏处理
return fmt.Errorf("API密钥无效: %s", maskAPIKey(apiKey))
```

## 故障排查

### 测试失败：数据库连接错误

```bash
# 检查数据库是否运行
pg_isready -h localhost -p 5432

# 检查环境变量
echo $DATABASE_URL

# 测试数据库连接
psql $DATABASE_URL -c "SELECT 1"
```

### 测试失败：API密钥错误

```bash
# 检查环境变量
echo $GOOGLE_API_KEY

# 验证API密钥格式
# Google API Key 应该以 "AIza" 开头
```

### 测试超时

```bash
# 增加超时时间
go test -v -timeout 20m -run TestComprehensiveErrorHandling ./test/e2e/
```

## 性能指标

- **测试数量**: 13个主要测试场景
- **预计运行时间**: 5-10分钟（取决于网络和API响应）
- **并发测试**: 支持
- **资源使用**: 低（主要是网络I/O）

## 相关文档

- [错误场景测试](./error_scenarios_test.go)
- [错误场景快速参考](./ERROR_SCENARIOS_QUICK_REF.md)
- [性能测试](./performance_test.go)
- [端到端测试指南](./README.md)

## 注意事项

1. **网络依赖**: 某些测试需要实际的网络连接
2. **API配额**: 频繁测试可能消耗API配额
3. **测试隔离**: 每个测试使用独立的租户和配置
4. **清理**: 测试完成后自动清理测试数据
5. **幂等性**: 测试可以重复运行

## 更新日志

### 2024-11-29

- ✅ 创建全面错误处理测试套件
- ✅ 添加网络错误测试
- ✅ 添加Azure特定错误测试
- ✅ 添加百炼特定错误测试
- ✅ 添加并发安全性测试
- ✅ 添加错误恢复测试
- ✅ 创建测试脚本和文档
