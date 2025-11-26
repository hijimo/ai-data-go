# Azure OpenAI 集成测试快速开始

## 最简单的运行方式

```bash
# 1. 设置环境变量
export AZURE_OPENAI_API_KEY="your-api-key-here"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="gpt-4"

# 2. 运行测试
./test/test_azure_openai.sh
```

## 使用配置文件

```bash
# 1. 复制并编辑配置文件
cp test/.env.azure.example test/.env.azure
vim test/.env.azure

# 2. 加载配置
source test/.env.azure

# 3. 运行测试
./test/test_azure_openai.sh
```

## 跳过集成测试

```bash
# 在 CI/CD 或不想运行集成测试时
go test -short ./internal/genkit/
```

## 获取 Azure OpenAI 配置

### API Key

1. 登录 [Azure Portal](https://portal.azure.com)
2. 找到您的 Azure OpenAI 资源
3. 在左侧菜单中选择"密钥和终结点"
4. 复制"密钥 1"或"密钥 2"

### Endpoint

在同一页面的"终结点"字段中复制，格式类似：

```
https://your-resource-name.openai.azure.com
```

### Deployment

1. 在 Azure OpenAI Studio 中查看您的部署
2. 或在 Azure Portal 的"模型部署"页面查看
3. 使用您创建的部署名称（例如：gpt-4, gpt-35-turbo）

## 常见问题

### Q: 测试失败，提示"缺少环境变量"

A: 确保设置了所有必需的环境变量：

- AZURE_OPENAI_API_KEY
- AZURE_OPENAI_ENDPOINT
- AZURE_OPENAI_DEPLOYMENT

### Q: 测试失败，提示"数据库连接失败"

A: 确保 PostgreSQL 正在运行，并且连接信息正确。

### Q: 测试失败，提示"API 调用失败"

A: 检查：

1. API Key 是否正确
2. Endpoint 和 Deployment 是否匹配
3. Azure OpenAI 资源是否可用
4. 是否有足够的配额

### Q: 如何在 CI/CD 中运行？

A: 使用 `-short` 标志跳过集成测试：

```bash
go test -short ./internal/genkit/
```

## 更多信息

详细文档请参考：

- [Azure OpenAI 集成测试指南](../internal/genkit/AZURE_INTEGRATION_TEST_README.md)
- [TASK-3.3 实现总结](../internal/genkit/TASK-3.3-IMPLEMENTATION-SUMMARY.md)
