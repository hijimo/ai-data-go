# 测试环境配置说明

本文档说明 TASK 7.4 中新增的测试环境配置功能。

## 📦 新增内容

### 配置文件

- `.env.test.example` - 测试环境配置模板

### 自动化脚本

- `scripts/setup_test_env.sh` - 测试环境配置向导
- `scripts/init_test_data.sh` - 测试数据初始化脚本
- `scripts/verify_test_env.sh` - 测试环境验证脚本

### 文档

- `docs/TEST_ENVIRONMENT_SETUP.md` - 完整配置指南
- `docs/TEST_ENV_CHECKLIST.md` - 配置检查清单
- `docs/TEST_ENV_QUICK_REF.md` - 快速参考

### Makefile 命令

```bash
make test-env-setup      # 配置测试环境
make test-env-verify     # 验证测试环境配置
make test-env-init       # 初始化测试数据
make run-test            # 使用测试环境配置运行服务
make test-e2e            # 运行端到端测试
make test-performance    # 运行性能测试
```

## 🚀 快速开始

### 方式 1: 使用配置向导（推荐）

```bash
# 1. 运行配置向导
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

### 方式 2: 手动配置

```bash
# 1. 复制配置模板
cp .env.test.example .env.test

# 2. 编辑配置文件
vim .env.test

# 3. 验证配置
make test-env-verify

# 4. 启动服务
make run-test

# 5. 初始化测试数据
make test-env-init
```

## 📋 配置项说明

### 必需配置

1. **数据库配置**
   - DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD

2. **Redis 配置**
   - REDIS_HOST, REDIS_PORT, REDIS_PASSWORD (可选)

3. **Google AI 配置**
   - GOOGLE_API_KEY, GENKIT_API_KEY

4. **安全配置**
   - JWT_SECRET (至少 32 字节)
   - ENCRYPTION_SECRET_KEY (必须 32 字节)

### 可选配置

1. **Azure OpenAI**
   - AZURE_OPENAI_API_KEY
   - AZURE_OPENAI_ENDPOINT
   - AZURE_OPENAI_DEPLOYMENT

2. **阿里云百炼**
   - BAILIAN_API_KEY
   - BAILIAN_ENDPOINT

## 🔍 验证测试环境

### 自动验证

```bash
make test-env-verify
```

验证脚本会检查：

- ✓ 基础环境（Go、PostgreSQL、Redis 等）
- ✓ 数据库连接
- ✓ API 密钥配置
- ✓ 安全配置
- ✓ 服务状态

### 手动验证

```bash
# 数据库连接
psql -h localhost -U postgres -d ai_service_test -c "SELECT 1"

# Redis 连接
redis-cli ping

# 服务健康检查
curl http://localhost:8080/health

# API 健康检查
curl http://localhost:8080/api/v1/health
```

## 🧪 运行测试

### 端到端测试

```bash
# 运行所有端到端测试
make test-e2e

# 运行特定提供商测试
go test -v ./test/e2e -run TestGoogleAI
go test -v ./test/e2e -run TestAzureOpenAI
go test -v ./test/e2e -run TestBailian
```

### 性能测试

```bash
make test-performance
```

### 集成测试

```bash
go test -v ./internal/genkit/...
```

## 📊 测试数据

### 默认账户

初始化后会创建以下账户：

```
平台管理员:
  邮箱: admin@test.local
  密码: Test@Admin123

测试租户管理员:
  邮箱: tenant-admin@test.local
  密码: Test@Tenant123
```

### 模型配置

测试租户会自动配置所有可用的模型提供商：

- Google AI (gemini-pro)
- Azure OpenAI (gpt-4) - 如果配置了 API Key
- 百炼 (qwen-turbo) - 如果配置了 API Key

## 🐛 故障排查

### 问题 1: 数据库连接失败

```bash
# 检查 PostgreSQL 状态
pg_isready

# 创建测试数据库
createdb -h localhost -U postgres ai_service_test
```

### 问题 2: Redis 连接失败

```bash
# 检查 Redis 状态
redis-cli ping

# 启动 Redis
redis-server
```

### 问题 3: API 密钥无效

```bash
# 检查环境变量
echo $GOOGLE_API_KEY

# 重新加载配置
export $(cat .env.test | grep -v '^#' | xargs)
```

### 问题 4: 服务启动失败

```bash
# 检查端口占用
lsof -i :8080

# 查看日志
tail -f logs/app-*.log

# 验证配置
make test-env-verify
```

## 📚 详细文档

- [完整配置指南](docs/TEST_ENVIRONMENT_SETUP.md) - 详细的配置步骤和说明
- [配置检查清单](docs/TEST_ENV_CHECKLIST.md) - 完整的配置检查清单
- [快速参考](docs/TEST_ENV_QUICK_REF.md) - 常用命令和配置快速参考

## 🎯 下一步

完成测试环境配置后：

1. 配置所有 API 密钥
2. 部署应用
3. 验证所有提供商可用
4. 进行冒烟测试
5. 记录部署日志

## 💡 最佳实践

1. **使用独立的测试数据库**
   - 不要在生产数据库上运行测试

2. **定期更新 API 密钥**
   - 定期轮换测试环境的 API 密钥

3. **监控资源使用**
   - 测试时监控数据库和 Redis 的资源使用

4. **保存测试日志**
   - 保存测试日志用于问题排查

5. **自动化测试**
   - 将测试集成到 CI/CD 流程中

## 🔗 相关链接

- [多提供商配置指南](docs/MULTI_PROVIDER_GUIDE.md)
- [迁移指南](docs/MIGRATION_GUIDE.md)
- [故障排查指南](docs/TROUBLESHOOTING.md)
- [API 文档](http://localhost:8080/swagger/index.html)

## 📝 反馈

如有问题或建议，请：

1. 查看故障排查部分
2. 检查日志文件
3. 查阅相关文档
4. 联系技术支持团队
