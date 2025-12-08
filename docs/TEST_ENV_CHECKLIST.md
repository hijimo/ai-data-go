# 测试环境配置检查清单

使用此检查清单确保测试环境配置正确。

## 📋 配置前检查

### 软件环境

- [ ] Go 1.21+ 已安装

  ```bash
  go version
  ```

- [ ] PostgreSQL 14+ 已安装并运行

  ```bash
  psql --version
  pg_isready
  ```

- [ ] Redis 6+ 已安装并运行

  ```bash
  redis-cli --version
  redis-cli ping
  ```

- [ ] 必需工具已安装

  ```bash
  curl --version
  jq --version
  ```

### API 密钥准备

- [ ] Google AI API Key 已获取
  - 获取地址: <https://makersuite.google.com/app/apikey>
  - 已测试可用

- [ ] Azure OpenAI 配置已准备（可选）
  - [ ] API Key
  - [ ] Endpoint URL
  - [ ] Deployment Name
  - [ ] API Version

- [ ] 阿里云百炼配置已准备（可选）
  - [ ] API Key
  - [ ] Endpoint URL
  - [ ] Workspace

## 📝 配置步骤检查

### 1. 配置文件创建

- [ ] 运行配置向导

  ```bash
  make test-env-setup
  ```

- [ ] 或手动创建配置文件

  ```bash
  cp .env.test.example .env.test
  ```

- [ ] `.env.test` 文件已创建

### 2. 数据库配置

- [ ] 数据库连接参数已配置
  - [ ] DB_HOST
  - [ ] DB_PORT
  - [ ] DB_USER
  - [ ] DB_PASSWORD
  - [ ] DB_NAME

- [ ] 测试数据库已创建

  ```bash
  createdb -h localhost -U postgres ai_service_test
  ```

- [ ] 数据库连接测试通过

  ```bash
  psql -h localhost -U postgres -d ai_service_test -c "SELECT 1"
  ```

### 3. Redis 配置

- [ ] Redis 连接参数已配置
  - [ ] REDIS_HOST
  - [ ] REDIS_PORT
  - [ ] REDIS_PASSWORD (如果需要)
  - [ ] REDIS_DB

- [ ] Redis 连接测试通过

  ```bash
  redis-cli -h localhost -p 6379 ping
  ```

### 4. API 密钥配置

- [ ] Google AI 配置
  - [ ] GOOGLE_API_KEY 已设置
  - [ ] GENKIT_API_KEY 已设置
  - [ ] GENKIT_MODEL 已设置

- [ ] Azure OpenAI 配置（如果使用）
  - [ ] AZURE_OPENAI_API_KEY 已设置
  - [ ] AZURE_OPENAI_ENDPOINT 已设置
  - [ ] AZURE_OPENAI_DEPLOYMENT 已设置
  - [ ] AZURE_OPENAI_API_VERSION 已设置

- [ ] 百炼配置（如果使用）
  - [ ] BAILIAN_API_KEY 已设置
  - [ ] BAILIAN_ENDPOINT 已设置
  - [ ] BAILIAN_WORKSPACE 已设置

### 5. 安全配置

- [ ] JWT 密钥已配置
  - [ ] JWT_SECRET 长度 >= 32 字节
  - [ ] JWT_ISSUER 已设置
  - [ ] JWT_AUDIENCE 已设置

- [ ] 加密密钥已配置
  - [ ] ENCRYPTION_SECRET_KEY 长度 = 32 字节

- [ ] 密码策略已配置
  - [ ] BCRYPT_COST 已设置
  - [ ] PASSWORD_MIN_LENGTH 已设置

### 6. 系统管理员配置

- [ ] 平台管理员信息已配置
  - [ ] PLATFORM_ADMIN_EMAIL
  - [ ] PLATFORM_ADMIN_PASSWORD
  - [ ] PLATFORM_ADMIN_NAME

- [ ] 平台租户信息已配置
  - [ ] PLATFORM_TENANT_NAME
  - [ ] PLATFORM_TENANT_DOMAIN

### 7. 日志配置

- [ ] 日志参数已配置
  - [ ] LOG_LEVEL=debug
  - [ ] LOG_FORMAT=json
  - [ ] LOG_ENABLE_FILE=true
  - [ ] LOG_DIR=logs

- [ ] 日志目录已创建

  ```bash
  mkdir -p logs
  ```

## ✅ 验证检查

### 自动验证

- [ ] 运行验证脚本

  ```bash
  make test-env-verify
  ```

- [ ] 所有检查项通过

### 手动验证

- [ ] 编译项目成功

  ```bash
  make build
  ```

- [ ] 服务启动成功

  ```bash
  make run-test
  ```

- [ ] 健康检查通过

  ```bash
  curl http://localhost:8080/health
  ```

- [ ] API 健康检查通过

  ```bash
  curl http://localhost:8080/api/v1/health
  ```

- [ ] Swagger 文档可访问
  - 访问: <http://localhost:8080/swagger/index.html>

## 🔧 测试数据初始化

- [ ] 服务正在运行

- [ ] 运行初始化脚本

  ```bash
  make test-env-init
  ```

- [ ] 测试租户创建成功
  - [ ] 租户ID已记录
  - [ ] 管理员账户已创建

- [ ] 模型配置创建成功
  - [ ] Google AI 配置已创建
  - [ ] Azure OpenAI 配置已创建（如果配置了）
  - [ ] 百炼配置已创建（如果配置了）

- [ ] 测试租户管理员登录成功
  - [ ] Token 已获取并保存

## 🧪 测试执行检查

### 端到端测试

- [ ] 运行端到端测试

  ```bash
  make test-e2e
  ```

- [ ] Google AI 测试通过

- [ ] Azure OpenAI 测试通过（如果配置了）

- [ ] 百炼测试通过（如果配置了）

- [ ] 提供商切换测试通过

### 性能测试

- [ ] 运行性能测试

  ```bash
  make test-performance
  ```

- [ ] 单提供商延迟测试通过

- [ ] 提供商切换延迟测试通过

- [ ] 并发调用测试通过

- [ ] 内存使用测试通过

### 集成测试

- [ ] 运行集成测试

  ```bash
  go test -v ./internal/genkit/...
  ```

- [ ] 所有集成测试通过

## 📊 性能基准检查

### 响应时间

- [ ] Google AI 响应时间 < 2s

- [ ] Azure OpenAI 响应时间 < 3s

- [ ] 百炼响应时间 < 2s

### 提供商切换

- [ ] 首次切换延迟 < 100ms

- [ ] 后续切换延迟 < 10ms

### 并发性能

- [ ] 支持 100+ 并发请求

- [ ] 无明显性能降级

## 🔍 故障排查检查

如果任何检查项失败，参考以下步骤：

### 数据库问题

- [ ] 检查 PostgreSQL 服务状态
- [ ] 检查连接参数
- [ ] 检查数据库权限
- [ ] 查看数据库日志

### Redis 问题

- [ ] 检查 Redis 服务状态
- [ ] 检查连接参数
- [ ] 检查密码配置
- [ ] 查看 Redis 日志

### API 密钥问题

- [ ] 验证 API 密钥格式
- [ ] 检查 API 密钥有效期
- [ ] 确认 API 密钥权限
- [ ] 测试 API 密钥可用性

### 服务启动问题

- [ ] 检查端口占用
- [ ] 查看应用日志
- [ ] 验证配置文件语法
- [ ] 检查环境变量加载

### 测试失败问题

- [ ] 确认服务运行中
- [ ] 检查测试数据
- [ ] 查看测试日志
- [ ] 验证 API 密钥

## 📚 文档检查

- [ ] 已阅读测试环境配置指南
  - `docs/TEST_ENVIRONMENT_SETUP.md`

- [ ] 已阅读多提供商配置指南
  - `docs/MULTI_PROVIDER_GUIDE.md`

- [ ] 已阅读迁移指南
  - `docs/MIGRATION_GUIDE.md`

- [ ] 已阅读故障排查指南
  - `docs/TROUBLESHOOTING.md`

## ✨ 最终确认

- [ ] 所有配置项已完成

- [ ] 所有验证检查通过

- [ ] 测试数据已初始化

- [ ] 所有测试执行成功

- [ ] 性能基准达标

- [ ] 相关文档已阅读

## 📝 备注

记录配置过程中的问题和解决方案：

```
日期: ___________
配置人: ___________

遇到的问题:
1. 
2. 
3. 

解决方案:
1. 
2. 
3. 

特殊配置:
1. 
2. 
3. 
```

## 🎯 下一步

配置完成后：

1. [ ] 将配置信息记录到团队文档
2. [ ] 通知团队成员测试环境已就绪
3. [ ] 安排定期的环境维护
4. [ ] 设置监控和告警
5. [ ] 准备生产环境部署计划
