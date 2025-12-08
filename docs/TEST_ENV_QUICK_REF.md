# 测试环境配置快速参考

## 🚀 快速开始（5 分钟）

```bash
# 1. 配置测试环境
make test-env-setup

# 2. 验证配置
make test-env-verify

# 3. 启动服务
make run-test

# 4. 初始化测试数据（新终端）
make test-env-init

# 5. 运行测试
make test-e2e
```

## 📋 常用命令

### 配置管理

```bash
# 配置测试环境（交互式向导）
make test-env-setup

# 手动创建配置文件
cp .env.test.example .env.test
vim .env.test

# 验证配置
make test-env-verify
```

### 服务管理

```bash
# 启动测试服务
make run-test

# 编译项目
make build

# 停止服务
pkill -f "bin/server"
```

### 测试数据

```bash
# 初始化测试数据
make test-env-init

# 查看测试租户信息
grep TEST_TENANT .env.test
```

### 运行测试

```bash
# 端到端测试
make test-e2e

# 性能测试
make test-performance

# 集成测试
go test -v ./internal/genkit/...
```

## 🔧 必需配置

### 最小配置（仅 Google AI）

```bash
# 数据库
DB_HOST=localhost
DB_NAME=ai_service_test
DB_USER=postgres
DB_PASSWORD=your_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Google AI
GOOGLE_API_KEY=your_google_api_key
GENKIT_API_KEY=your_google_api_key

# 安全
JWT_SECRET=your-32-byte-secret-key!!
ENCRYPTION_SECRET_KEY=your-32-byte-encryption-key!!
```

### 完整配置（所有提供商）

在最小配置基础上添加：

```bash
# Azure OpenAI
AZURE_OPENAI_API_KEY=your_azure_key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=gpt-4

# 百炼
BAILIAN_API_KEY=your_bailian_key
BAILIAN_ENDPOINT=https://dashscope.aliyuncs.com
```

## 🔍 验证检查

### 快速验证

```bash
# 运行验证脚本
make test-env-verify

# 检查服务健康
curl http://localhost:8080/health

# 检查 API
curl http://localhost:8080/api/v1/health
```

### 手动验证

```bash
# 数据库连接
psql -h localhost -U postgres -d ai_service_test -c "SELECT 1"

# Redis 连接
redis-cli ping

# 查看日志
tail -f logs/app-*.log
```

## 🐛 常见问题

### 数据库连接失败

```bash
# 检查 PostgreSQL 状态
pg_isready

# 创建测试数据库
createdb -h localhost -U postgres ai_service_test
```

### Redis 连接失败

```bash
# 检查 Redis 状态
redis-cli ping

# 启动 Redis
redis-server
```

### API 密钥无效

```bash
# 检查环境变量
echo $GOOGLE_API_KEY

# 重新加载配置
export $(cat .env.test | grep -v '^#' | xargs)
```

### 服务启动失败

```bash
# 检查端口占用
lsof -i :8080

# 查看日志
tail -f logs/app-*.log

# 检查配置
make test-env-verify
```

## 📊 测试数据

### 默认账户

```
平台管理员:
  邮箱: admin@test.local
  密码: Test@Admin123

测试租户管理员:
  邮箱: tenant-admin@test.local
  密码: Test@Tenant123
```

### 模型配置

```
Google AI:
  模型名称: gemini-pro
  模型: gemini-2.0-flash-exp

Azure OpenAI:
  模型名称: gpt-4
  部署: gpt-4

百炼:
  模型名称: qwen-turbo
  模型: qwen-turbo
```

## 🔗 相关链接

- [完整配置指南](TEST_ENVIRONMENT_SETUP.md)
- [配置检查清单](TEST_ENV_CHECKLIST.md)
- [多提供商指南](MULTI_PROVIDER_GUIDE.md)
- [故障排查](TROUBLESHOOTING.md)

## 💡 提示

- 使用 `make test-env-verify` 定期检查配置
- 测试前确保服务正在运行
- 查看日志文件排查问题
- 使用独立的测试数据库
- 定期更新 API 密钥
