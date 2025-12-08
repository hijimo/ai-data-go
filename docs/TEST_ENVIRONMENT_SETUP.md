# 测试环境配置指南

本文档说明如何配置和部署多提供商支持的测试环境。

## 目录

- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [详细配置步骤](#详细配置步骤)
- [验证测试环境](#验证测试环境)
- [运行测试](#运行测试)
- [故障排查](#故障排查)

## 前置要求

### 必需软件

- Go 1.21 或更高版本
- PostgreSQL 14 或更高版本
- Redis 6 或更高版本
- curl 和 jq 工具

### API 密钥

至少需要配置以下一个提供商的 API 密钥：

1. **Google AI (Gemini)** - 推荐用于基础测试
   - 获取地址: <https://makersuite.google.com/app/apikey>

2. **Azure OpenAI** - 可选
   - 需要 Azure 订阅和 OpenAI 资源

3. **阿里云百炼** - 可选
   - 需要阿里云账号和百炼服务

## 快速开始

### 1. 配置测试环境

运行配置向导：

```bash
make test-env-setup
```

配置向导会引导你完成以下步骤：

- 数据库配置
- Redis 配置
- Google AI 配置
- Azure OpenAI 配置（可选）
- 阿里云百炼配置（可选）

### 2. 验证配置

```bash
make test-env-verify
```

验证脚本会检查：

- ✓ 基础环境（Go、PostgreSQL、Redis 等）
- ✓ 数据库连接
- ✓ API 密钥配置
- ✓ 安全配置（JWT 密钥、加密密钥）
- ✓ 服务状态

### 3. 启动服务

```bash
make run-test
```

服务将使用 `.env.test` 配置启动。

### 4. 初始化测试数据

在另一个终端窗口运行：

```bash
make test-env-init
```

这将：

- 创建测试租户
- 创建租户管理员账户
- 配置所有可用的模型提供商

### 5. 运行测试

```bash
# 运行端到端测试
make test-e2e

# 运行性能测试
make test-performance
```

## 详细配置步骤

### 步骤 1: 准备配置文件

如果不使用配置向导，可以手动创建配置文件：

```bash
cp .env.test.example .env.test
```

### 步骤 2: 配置数据库

编辑 `.env.test` 文件，配置数据库连接：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ai_service_test
DB_USER=postgres
DB_PASSWORD=your_password
```

创建测试数据库：

```bash
createdb -h localhost -U postgres ai_service_test
```

### 步骤 3: 配置 Redis

```bash
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password
REDIS_DB=1
```

### 步骤 4: 配置 API 密钥

#### Google AI (必需)

```bash
GOOGLE_API_KEY=your_google_api_key
GENKIT_API_KEY=your_google_api_key
GENKIT_MODEL=gemini-2.0-flash-exp
```

#### Azure OpenAI (可选)

```bash
AZURE_OPENAI_API_KEY=your_azure_api_key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=gpt-4
AZURE_OPENAI_API_VERSION=2024-02-15-preview
```

#### 阿里云百炼 (可选)

```bash
BAILIAN_API_KEY=your_bailian_api_key
BAILIAN_ENDPOINT=https://dashscope.aliyuncs.com
BAILIAN_WORKSPACE=default
```

### 步骤 5: 配置安全密钥

```bash
# JWT 签名密钥（至少 32 字节）
JWT_SECRET=your-test-jwt-secret-key-32-bytes!!

# 加密密钥（必须是 32 字节）
ENCRYPTION_SECRET_KEY=your-test-encryption-key-32!!
```

### 步骤 6: 配置系统管理员

```bash
PLATFORM_ADMIN_EMAIL=admin@test.local
PLATFORM_ADMIN_PASSWORD=Test@Admin123
PLATFORM_ADMIN_NAME=Test Admin
```

## 验证测试环境

### 自动验证

```bash
make test-env-verify
```

### 手动验证

#### 1. 检查数据库连接

```bash
psql -h localhost -U postgres -d ai_service_test -c "SELECT 1"
```

#### 2. 检查 Redis 连接

```bash
redis-cli -h localhost -p 6379 ping
```

#### 3. 检查服务健康

```bash
curl http://localhost:8080/health
```

#### 4. 检查 API 端点

```bash
curl http://localhost:8080/api/v1/health
```

#### 5. 检查 Swagger 文档

访问: <http://localhost:8080/swagger/index.html>

## 运行测试

### 端到端测试

测试所有提供商的完整功能：

```bash
make test-e2e
```

或者单独测试特定提供商：

```bash
# Google AI
go test -v ./test/e2e -run TestGoogleAI

# Azure OpenAI
go test -v ./test/e2e -run TestAzureOpenAI

# 百炼
go test -v ./test/e2e -run TestBailian
```

### 性能测试

```bash
make test-performance
```

性能测试包括：

- 单提供商延迟测试
- 提供商切换延迟测试
- 并发调用性能测试
- 内存使用测试

### 集成测试

```bash
# 运行所有集成测试
go test -v ./internal/genkit/...

# 运行特定测试
go test -v ./internal/genkit -run TestAzureIntegration
```

## 测试数据

### 测试租户信息

初始化后会创建一个测试租户：

```
租户ID: 自动生成
租户名称: 测试租户
租户域名: test-tenant.local
管理员邮箱: tenant-admin@test.local
管理员密码: Test@Tenant123
```

### 模型配置

测试租户会自动配置所有可用的模型提供商：

1. **Google AI (Gemini)**
   - 模型名称: gemini-pro
   - 模型: gemini-2.0-flash-exp

2. **Azure OpenAI** (如果配置了 API Key)
   - 模型名称: gpt-4
   - 部署: 根据配置

3. **阿里云百炼** (如果配置了 API Key)
   - 模型名称: qwen-turbo
   - 模型: qwen-turbo

## 故障排查

### 问题 1: 数据库连接失败

**症状**: `connection refused` 或 `authentication failed`

**解决方案**:

1. 检查 PostgreSQL 是否运行: `pg_isready`
2. 检查连接参数是否正确
3. 检查 `pg_hba.conf` 配置
4. 确认数据库已创建

### 问题 2: Redis 连接失败

**症状**: `connection refused` 或 `NOAUTH`

**解决方案**:

1. 检查 Redis 是否运行: `redis-cli ping`
2. 检查密码配置
3. 检查 Redis 配置文件中的 `bind` 和 `protected-mode`

### 问题 3: API 密钥无效

**症状**: `401 Unauthorized` 或 `invalid API key`

**解决方案**:

1. 确认 API 密钥正确
2. 检查 API 密钥是否有效期
3. 确认 API 密钥有足够的权限
4. 检查环境变量是否正确加载

### 问题 4: 服务启动失败

**症状**: 服务无法启动或立即退出

**解决方案**:

1. 检查日志文件: `logs/app-*.log`
2. 确认所有必需的环境变量已配置
3. 检查端口是否被占用: `lsof -i :8080`
4. 验证配置文件语法

### 问题 5: 测试失败

**症状**: 测试用例失败

**解决方案**:

1. 确认服务正在运行
2. 检查测试数据是否已初始化
3. 查看测试日志获取详细错误信息
4. 确认所有提供商的 API 密钥都有效

## 环境清理

### 清理测试数据

```bash
# 删除测试数据库
dropdb -h localhost -U postgres ai_service_test

# 清理 Redis 数据
redis-cli -h localhost -p 6379 -n 1 FLUSHDB
```

### 重置测试环境

```bash
# 停止服务
pkill -f "bin/server"

# 删除配置文件
rm .env.test

# 重新配置
make test-env-setup
```

## 最佳实践

### 1. 使用独立的测试数据库

不要在生产数据库上运行测试，始终使用独立的测试数据库。

### 2. 定期更新 API 密钥

定期轮换测试环境的 API 密钥，确保安全性。

### 3. 监控资源使用

测试时监控数据库和 Redis 的资源使用情况。

### 4. 保存测试日志

保存测试日志用于问题排查和性能分析。

### 5. 自动化测试

将测试集成到 CI/CD 流程中，确保每次代码变更都经过测试。

## 相关文档

- [多提供商配置指南](MULTI_PROVIDER_GUIDE.md)
- [迁移指南](MIGRATION_GUIDE.md)
- [故障排查指南](TROUBLESHOOTING.md)
- [API 文档](http://localhost:8080/swagger/index.html)

## 支持

如有问题，请：

1. 查看故障排查部分
2. 检查日志文件
3. 查阅相关文档
4. 联系技术支持团队
