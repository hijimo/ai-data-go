# 百炼测试快速参考

## 快速开始

### 1. 设置环境变量

```bash
# 必需：百炼 API 密钥
export BAILIAN_API_KEY="sk-xxxxxxxxxxxxx"

# 可选：自定义配置
export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"
export BAILIAN_MODEL="qwen-plus"
```

### 2. 运行测试

```bash
# 使用测试脚本（推荐）
./test/test_bailian.sh

# 或直接运行 Go 测试
go test -v -run TestBailianIntegration_NonStreaming ./internal/genkit/
```

## 支持的模型

| 模型名称 | 说明 | 适用场景 |
|---------|------|---------|
| `qwen-plus` | 通义千问增强版 | 推荐，平衡性能和成本 |
| `qwen-max` | 通义千问旗舰版 | 复杂任务，最高质量 |
| `qwen-turbo` | 通义千问快速版 | 简单任务，快速响应 |
| `qwen-long` | 长文本模型 | 长文本处理 |

## 地域端点

| 地域 | 端点 URL |
|-----|---------|
| 北京（默认） | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| 新加坡 | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` |
| 金融云 | `https://dashscope-finance.aliyuncs.com/compatible-mode/v1` |

## 测试用例

1. ✅ 基本文本生成
2. ✅ 中文处理能力
3. ✅ 参数传递
4. ✅ Token 统计
5. ✅ 错误处理（配置不存在）
6. ✅ 错误处理（租户ID无效）
7. ✅ 错误处理（模型已禁用）
8. ✅ 响应格式验证
9. ✅ 缓存机制
10. ✅ 复杂中文对话

## 常见问题

### Q: 如何获取 API 密钥？

A: 访问 <https://dashscope.aliyun.com/> 注册并获取 API-KEY

### Q: 测试失败怎么办？

A: 检查以下几点：

- API 密钥是否正确
- 网络连接是否正常
- 数据库是否启动
- 账户余额是否充足

### Q: 如何跳过集成测试？

A: 使用 `-short` 标志：

```bash
go test -short ./internal/genkit/
```

## 相关文档

- 详细测试指南：`internal/genkit/BAILIAN_INTEGRATION_TEST_README.md`
- 实现总结：`internal/genkit/TASK-4.4-IMPLEMENTATION-SUMMARY.md`
- 百炼插件文档：`internal/genkit/BAILIAN_INTEGRATION_GUIDE.md`
