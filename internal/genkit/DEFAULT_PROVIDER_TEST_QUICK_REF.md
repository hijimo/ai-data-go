# 默认提供商测试快速参考

## 快速开始

```bash
# 1. 设置环境变量
export GOOGLE_API_KEY='your-google-api-key'

# 2. 运行测试
./test/test_default_provider.sh
```

## 测试内容

| 测试场景 | 描述 | 验证点 |
|---------|------|--------|
| 使用默认模型名称 | 使用 `gemini-pro` 进行非流式调用 | 响应成功，模型名称正确 |
| 流式调用使用默认模型 | 使用默认模型进行流式调用 | 接收多个数据块，模型名称正确 |
| 默认模型使用自定义参数 | 传递 temperature 和 maxTokens | 参数正确传递，响应成功 |
| 并发使用默认模型 | 5 个并发请求 | 所有请求成功，模型名称一致 |
| 禁用默认模型后的错误处理 | 禁用模型后尝试调用 | 返回"模型已禁用"错误 |
| 使用不存在的模型名称 | 使用不存在的模型 | 返回"模型配置"错误 |
| 测量默认模型的响应时间 | 3 次调用测量性能 | 记录平均响应时间 |
| 验证默认模型实例被缓存 | 两次调用比较耗时 | 缓存机制正常工作 |

## 默认模型配置

```json
{
  "name": "gemini-pro",
  "model": "gemini-1.5-pro",
  "modelProvider": "googlegenai",
  "queryParams": {
    "model": "gemini-1.5-pro",
    "defaultTemperature": 0.7,
    "defaultMaxTokens": 2048
  }
}
```

## 验证的需求

- ✅ **需求 5**: 保持向后兼容
  - 使用现有 API 接口正常工作
  - 未配置新提供商时默认使用 Google AI
  - 流式接口响应格式保持不变

- ✅ **FR-2**: Genkit 客户端扩展
  - 客户端接口保持不变
  - 根据配置动态选择插件

## 常见问题

### Q: 如何跳过测试？

```bash
go test -short ./test/e2e/
```

### Q: 如何增加超时时间？

```bash
go test -timeout 10m -run TestDefaultProvider ./test/e2e/
```

### Q: 如何查看详细日志？

```bash
go test -v -run TestDefaultProvider ./test/e2e/
```

## 相关文件

- 测试代码：`test/e2e/default_provider_test.go`
- 测试脚本：`test/test_default_provider.sh`
- 详细文档：`internal/genkit/DEFAULT_PROVIDER_TEST_README.md`
