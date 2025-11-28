# 百炼流式调用测试快速参考

## 快速开始

```bash
# 1. 设置环境变量
export BAILIAN_API_KEY="your-api-key"

# 2. 运行测试
./test/test_bailian_stream.sh
```

## 环境变量

| 变量名 | 必需 | 默认值 | 说明 |
|--------|------|--------|------|
| `BAILIAN_API_KEY` | ✅ | - | 百炼 API 密钥 |
| `BAILIAN_ENDPOINT` | ❌ | `https://dashscope.aliyuncs.com/compatible-mode/v1` | API 端点 |
| `BAILIAN_MODEL` | ❌ | `qwen-plus` | 模型名称 |

## 测试命令

```bash
# 运行所有流式测试
go test -v -run TestBailianIntegration_Streaming ./internal/genkit/

# 运行特定测试
go test -v -run TestBailianIntegration_Streaming/流式响应接收 ./internal/genkit/
go test -v -run TestBailianIntegration_Streaming/中文流式输出 ./internal/genkit/
go test -v -run TestBailianIntegration_Streaming/流式中断处理 ./internal/genkit/
```

## 测试用例列表

| 测试用例 | 说明 |
|---------|------|
| 流式响应接收 | 基本流式响应功能 |
| 中文流式输出测试 | 中文处理能力 |
| 流式响应完整性 | 响应完整性验证 |
| 流式中断处理 | 取消机制测试 |
| 流式参数传递 | 自定义参数测试 |
| 验证SSE格式转换 | SSE 格式验证 |
| 错误处理 - 配置不存在 | 错误场景测试 |
| 错误处理 - 租户ID无效 | 错误场景测试 |
| 错误处理 - 模型已禁用 | 错误场景测试 |
| 流式响应性能测试 | 性能指标测试 |
| 并发流式调用 | 并发安全测试 |
| 长文本流式生成 | 长文本处理测试 |

## 性能指标

- **首字节时间**: < 5 秒
- **短文本块数**: 5-15 个
- **长文本块数**: > 20 个
- **并发数**: 3 个同时请求

## 常见错误

### API 密钥未设置

```
跳过百炼集成测试：缺少必需的环境变量 BAILIAN_API_KEY
```

**解决**: `export BAILIAN_API_KEY="your-api-key"`

### 数据库连接失败

```
Error connecting to database
```

**解决**: 检查数据库配置和服务状态

### 网络超时

```
context deadline exceeded
```

**解决**: 检查网络连接和防火墙设置

## 测试输出示例

```
=== RUN   TestBailianIntegration_Streaming
=== RUN   TestBailianIntegration_Streaming/流式响应接收
    bailian_stream_test.go:XX: 接收到流式块: 我是
    bailian_stream_test.go:XX: 接收到流式块: 通义千问
    bailian_stream_test.go:XX: 接收到流式块: ，一个
    bailian_stream_test.go:XX: 接收到流式块: 大型语言模型
    bailian_stream_test.go:XX: 流式响应完成
    bailian_stream_test.go:XX: 完整响应: 我是通义千问，一个大型语言模型
    bailian_stream_test.go:XX: 总共接收到 5 个块
    bailian_stream_test.go:XX: Token 使用情况: Prompt=10, Completion=15, Total=25
--- PASS: TestBailianIntegration_Streaming/流式响应接收 (2.34s)
```

## 相关文档

- 📖 [详细测试文档](./BAILIAN_STREAM_TEST_README.md)
- 📖 [百炼集成指南](./BAILIAN_INTEGRATION_GUIDE.md)
- 📖 [百炼非流式测试](./BAILIAN_INTEGRATION_TEST_README.md)

## 快速调试

```bash
# 启用详细日志
export LOG_LEVEL=debug

# 运行单个测试并查看详细输出
go test -v -run TestBailianIntegration_Streaming/流式响应接收 ./internal/genkit/ 2>&1 | tee test.log

# 检查测试覆盖率
go test -cover -run TestBailianIntegration_Streaming ./internal/genkit/
```

## 支持的模型

- `qwen-turbo` - 通义千问 Turbo
- `qwen-plus` - 通义千问 Plus（推荐）
- `qwen-max` - 通义千问 Max
- `qwen-max-longcontext` - 通义千问 Max 长文本

## 注意事项

1. ⚠️ 集成测试会调用真实的百炼 API，可能产生费用
2. ⚠️ 测试需要稳定的网络连接
3. ⚠️ 某些测试可能需要较长时间完成
4. ⚠️ 并发测试可能受到 API 速率限制影响

## 最佳实践

1. 在 CI/CD 中使用测试环境的 API 密钥
2. 定期运行测试以确保集成稳定
3. 监控测试性能指标的变化
4. 保持测试数据的清理
