# Azure OpenAI 流式调用测试 - 快速参考

## 快速开始

### 1. 设置环境变量

```bash
export AZURE_OPENAI_API_KEY=your-api-key
export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
export AZURE_OPENAI_DEPLOYMENT=your-deployment-name
```

### 2. 运行测试

```bash
# 使用测试脚本
./test/test_azure_stream.sh

# 或直接运行
go test -v -timeout 5m -run TestAzureOpenAIIntegration_Streaming ./internal/genkit/
```

## 测试用例列表

| 测试用例 | 描述 | 验证内容 |
|---------|------|---------|
| 流式响应接收 | 基本流式功能 | 接收块、拼接文本、Token统计 |
| 流式响应完整性 | 响应完整性 | 内容正确、无遗漏 |
| 流式中断处理 | 上下文取消 | 正确中断、资源清理 |
| 流式参数传递 | 自定义参数 | temperature、maxTokens |
| 流式响应格式验证 | 格式正确性 | 块格式、完成标记 |
| 错误处理 | 异常场景 | 配置错误、ID无效、模型禁用 |
| 流式响应性能 | 性能指标 | TTFB < 5秒 |
| 并发流式调用 | 并发稳定性 | 3个并发调用 |

## 环境变量

### 必需

- `AZURE_OPENAI_API_KEY` - API 密钥
- `AZURE_OPENAI_ENDPOINT` - 端点 URL
- `AZURE_OPENAI_DEPLOYMENT` - 部署名称

### 可选

- `AZURE_OPENAI_API_VERSION` - API 版本（默认：2024-02-15-preview）
- `DB_HOST` - 数据库主机（默认：localhost）
- `DB_PORT` - 数据库端口（默认：5432）
- `DB_USER` - 数据库用户（默认：postgres）
- `DB_PASSWORD` - 数据库密码（默认：postgres）
- `DB_NAME` - 数据库名称（默认：genkit_test）

## 常用命令

```bash
# 运行所有流式测试
go test -v -run TestAzureOpenAIIntegration_Streaming ./internal/genkit/

# 运行特定测试
go test -v -run TestAzureOpenAIIntegration_Streaming/流式响应接收 ./internal/genkit/

# 跳过集成测试
go test -short ./internal/genkit/

# 增加超时时间
go test -v -timeout 10m -run TestAzureOpenAIIntegration_Streaming ./internal/genkit/

# 显示详细日志
go test -v -run TestAzureOpenAIIntegration_Streaming ./internal/genkit/ 2>&1 | tee test.log
```

## 性能基准

| 指标 | 预期值 |
|------|--------|
| 首字节时间 | < 2秒 |
| 总耗时 | < 10秒 |
| 流式块数量 | 5-20个 |

## 故障排查

| 问题 | 解决方案 |
|------|---------|
| 测试被跳过 | 检查环境变量是否设置 |
| 数据库连接失败 | 确保 PostgreSQL 运行，数据库存在 |
| API 调用失败 | 验证 API 密钥、端点、部署名称 |
| 测试超时 | 检查网络连接，增加超时时间 |
| TTFB 过长 | 检查网络延迟，选择更近的区域 |

## 相关文件

- `internal/genkit/azure_stream_test.go` - 测试代码
- `test/test_azure_stream.sh` - 测试脚本
- `internal/genkit/AZURE_STREAM_TEST_README.md` - 详细文档
- `internal/genkit/TASK-3.4-IMPLEMENTATION-SUMMARY.md` - 实现总结

## 测试覆盖

- ✅ 流式响应接收和处理
- ✅ 流式响应完整性验证
- ✅ 流式中断和资源清理
- ✅ 参数传递和格式验证
- ✅ 错误处理和边界情况
- ✅ 性能测试（TTFB、总耗时）
- ✅ 并发调用和线程安全
- ✅ Token 统计和验证

## 下一步

完成 TASK-3.4 后，可以继续：

- TASK-4.1: 调研百炼 API 和集成方案
- TASK-4.2: 实现百炼自定义插件
- 或其他后续任务
