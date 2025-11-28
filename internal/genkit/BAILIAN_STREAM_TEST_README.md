# 百炼流式调用测试文档

## 概述

本文档描述百炼（阿里云通义千问）流式调用的集成测试套件。测试覆盖流式响应接收、中文处理、错误处理、性能测试等多个方面。

## 测试文件

- **测试代码**: `internal/genkit/bailian_stream_test.go`
- **测试脚本**: `test/test_bailian_stream.sh`

## 环境要求

### 必需的环境变量

```bash
# 百炼 API 密钥（必需）
export BAILIAN_API_KEY="your-bailian-api-key"
```

### 可选的环境变量

```bash
# 百炼 API 端点（默认：https://dashscope.aliyuncs.com/compatible-mode/v1）
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"

# 模型名称（默认：qwen-plus）
export BAILIAN_MODEL="qwen-plus"

# 数据库配置
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
export BAILIAN_API_KEY="your-api-key"

# 运行测试脚本
./test/test_bailian_stream.sh
```

### 直接运行 Go 测试

```bash
# 设置环境变量
export BAILIAN_API_KEY="your-api-key"
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"
export BAILIAN_MODEL="qwen-plus"

# 运行所有流式测试
go test -v -run TestBailianIntegration_Streaming ./internal/genkit/

# 运行特定测试用例
go test -v -run TestBailianIntegration_Streaming/流式响应接收 ./internal/genkit/
```

## 测试用例说明

### 1. 流式响应接收

**测试目标**: 验证基本的流式响应接收功能

**测试内容**:

- 调用 `GenerateStream` 方法
- 接收所有流式响应块
- 验证响应块的完整性
- 验证最后一个块是完成标记
- 验证 Token 统计信息

**预期结果**:

- 成功接收到多个响应块
- 完整文本不为空
- 最后一个块标记为完成
- Token 统计准确（Prompt + Completion = Total）

### 2. 中文流式输出测试

**测试目标**: 验证百炼对中文的流式处理能力

**测试内容**:

- 使用中文提示词
- 接收中文流式响应
- 验证中文内容的完整性

**预期结果**:

- 成功生成中文内容
- 流式块数量合理
- 中文字符正确显示

### 3. 流式响应完整性

**测试目标**: 验证流式响应的完整性

**测试内容**:

- 请求生成特定内容（如列举节日）
- 收集所有流式块
- 验证最终内容的完整性

**预期结果**:

- 响应内容完整
- 包含请求的信息

### 4. 流式中断处理

**测试目标**: 验证流式调用的中断机制

**测试内容**:

- 创建可取消的上下文
- 接收几个块后取消请求
- 验证取消操作正确处理

**预期结果**:

- 成功接收到部分响应
- 取消操作正确执行
- 没有资源泄漏

### 5. 流式参数传递

**测试目标**: 验证自定义参数在流式调用中的传递

**测试内容**:

- 设置自定义 temperature 和 maxTokens
- 调用流式接口
- 验证响应符合参数设置

**预期结果**:

- 参数正确传递
- 响应符合预期

### 6. 验证 SSE 格式转换

**测试目标**: 验证流式响应的 SSE 格式

**测试内容**:

- 接收所有流式块
- 验证每个块的格式
- 验证完成块的结构

**预期结果**:

- 内容块包含文本
- 完成块包含模型信息和使用统计
- 格式符合 SSE 规范

### 7. 错误处理测试

**测试目标**: 验证各种错误场景的处理

**测试内容**:

- 配置不存在
- 租户ID无效
- 模型已禁用

**预期结果**:

- 返回明确的错误信息
- 错误信息包含足够的上下文

### 8. 流式响应性能测试

**测试目标**: 验证流式响应的性能指标

**测试内容**:

- 记录首字节时间（TTFB）
- 记录总耗时
- 统计响应块数量

**预期结果**:

- 首字节时间在 5 秒内
- 性能指标在合理范围内

### 9. 并发流式调用

**测试目标**: 验证并发流式调用的稳定性

**测试内容**:

- 同时发起 3 个流式调用
- 验证所有调用都成功完成
- 验证没有资源竞争

**预期结果**:

- 所有并发调用成功
- 响应内容正确
- 没有死锁或资源泄漏

### 10. 长文本流式生成

**测试目标**: 验证长文本的流式生成能力

**测试内容**:

- 请求生成较长的文本
- 统计流式块数量
- 验证文本长度

**预期结果**:

- 生成多个流式块
- 文本长度符合预期
- 内容完整

## 测试数据

测试使用以下配置：

```json
{
  "model": "qwen-plus",
  "bailianEndpoint": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "defaultTemperature": 0.7,
  "defaultMaxTokens": 2048
}
```

## 性能指标

### 首字节时间（TTFB）

- **目标**: < 5 秒
- **说明**: 从发起请求到接收第一个响应块的时间

### 流式块数量

- **短文本**: 通常 5-15 个块
- **长文本**: 通常 > 20 个块

### 并发性能

- **并发数**: 3 个同时请求
- **预期**: 所有请求都能成功完成

## 常见问题

### 1. 测试跳过

**问题**: 测试被跳过，显示 "跳过集成测试"

**原因**:

- 运行了 `go test -short`
- 缺少必需的环境变量

**解决方案**:

```bash
# 不使用 -short 标志
go test -v -run TestBailianIntegration_Streaming ./internal/genkit/

# 确保设置了环境变量
export BAILIAN_API_KEY="your-api-key"
```

### 2. API 密钥错误

**问题**: 测试失败，提示 API 密钥无效

**解决方案**:

- 检查 API 密钥是否正确
- 确认 API 密钥有足够的权限
- 检查 API 密钥是否过期

### 3. 网络超时

**问题**: 测试超时或连接失败

**解决方案**:

- 检查网络连接
- 确认百炼服务可访问
- 检查防火墙设置

### 4. 数据库连接失败

**问题**: 测试失败，提示数据库连接错误

**解决方案**:

- 确认数据库服务正在运行
- 检查数据库连接参数
- 确认数据库用户有足够的权限

## 测试覆盖率

本测试套件覆盖以下方面：

- ✅ 基本流式响应接收
- ✅ 中文流式处理
- ✅ 流式响应完整性
- ✅ 流式中断处理
- ✅ 参数传递
- ✅ SSE 格式验证
- ✅ 错误处理
- ✅ 性能测试
- ✅ 并发测试
- ✅ 长文本生成

## 相关文档

- [百炼集成指南](./BAILIAN_INTEGRATION_GUIDE.md)
- [百炼非流式测试](./BAILIAN_INTEGRATION_TEST_README.md)
- [百炼快速参考](./BAILIAN_TEST_QUICK_REF.md)
- [Azure 流式测试](./AZURE_STREAM_TEST_README.md)

## 维护说明

### 添加新测试用例

1. 在 `bailian_stream_test.go` 中添加新的 `t.Run()` 块
2. 遵循现有的测试模式
3. 添加适当的断言和日志
4. 更新本文档

### 更新测试配置

1. 修改环境变量默认值
2. 更新测试脚本
3. 更新文档中的配置示例

### 性能基准更新

如果百炼服务性能有显著变化，需要更新性能指标：

1. 运行多次测试收集数据
2. 计算平均值和标准差
3. 更新文档中的性能指标
4. 调整测试断言的阈值

## 总结

本测试套件提供了全面的百炼流式调用测试，确保：

1. **功能正确性**: 流式响应正确接收和处理
2. **中文支持**: 中文内容正确处理
3. **错误处理**: 各种错误场景正确处理
4. **性能指标**: 满足性能要求
5. **并发安全**: 支持并发调用
6. **格式规范**: 符合 SSE 格式规范

通过这些测试，我们可以确保百炼流式调用功能的稳定性和可靠性。
