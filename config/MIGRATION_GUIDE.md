# 配置管理迁移指南

## 概述

本指南帮助现有用户从纯环境变量配置迁移到新的YAML配置系统。

## 向后兼容性

**重要**: 新的配置系统完全向后兼容，现有的环境变量配置方式仍然有效。

- ✅ 现有代码无需修改
- ✅ 环境变量配置继续工作
- ✅ 可以逐步迁移到YAML配置

## 迁移策略

### 策略1: 保持现状（推荐用于稳定系统）

如果当前系统运行稳定，可以继续使用环境变量配置：

```go
// 现有代码保持不变
cfg, err := config.Load()
```

### 策略2: 混合模式（推荐用于过渡期）

使用YAML配置文件 + 环境变量覆盖敏感信息：

```yaml
# config/prod.yaml
database:
  host: "db.example.com"
  port: 5432
  database: "mydb"
  user: "dbuser"
  password: "${DB_PASSWORD}"  # 从环境变量读取
  ssl_mode: require

genkit:
  api_key: "${GENKIT_API_KEY}"  # 从环境变量读取
  model: "gemini-2.5-flash"
```

```bash
# 只需设置敏感信息
export DB_PASSWORD=your-password
export GENKIT_API_KEY=your-api-key
```

### 策略3: 完全迁移（推荐用于新部署）

将所有配置迁移到YAML文件：

1. 创建配置文件
2. 将环境变量值迁移到YAML
3. 更新部署脚本

## 迁移步骤

### 步骤1: 创建配置文件

```bash
# 复制模板
cp config/dev.yaml config/myapp.yaml

# 编辑配置文件
vim config/myapp.yaml
```

### 步骤2: 迁移环境变量

将现有的环境变量值填入YAML文件：

**之前（环境变量）**:

```bash
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=mydb
export REDIS_HOST=localhost
```

**之后（YAML配置）**:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 5432
  database: "mydb"

redis:
  host: "localhost"
  port: 6379
```

### 步骤3: 保留敏感信息在环境变量

```yaml
# 使用环境变量替换
database:
  password: "${DB_PASSWORD}"

genkit:
  api_key: "${GENKIT_API_KEY}"

auth:
  jwt_secret: "${JWT_SECRET}"
```

### 步骤4: 更新代码（可选）

如果想使用新的加载器：

**之前**:

```go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

**之后**:

```go
// 方式1: 自动加载（推荐）
loader := config.NewConfigLoader()
cfg, err := loader.Load()
if err != nil {
    log.Fatal(err)
}

// 方式2: 快速加载
cfg := config.MustLoad()

// 方式3: 指定配置文件
cfg, err := config.LoadFromYAML("config/myapp.yaml")
```

### 步骤5: 测试

```bash
# 测试配置加载
export APP_ENV=development
./server

# 检查配置摘要输出
```

## 环境变量映射

### 服务器配置

| 环境变量 | YAML路径 |
|---------|---------|
| SERVER_PORT | server.port |
| SERVER_HOST | server.host |

### 数据库配置

| 环境变量 | YAML路径 |
|---------|---------|
| DATABASE_URL | database.url |
| DB_HOST | database.host |
| DB_PORT | database.port |
| DB_NAME | database.database |
| DB_USER | database.user |
| DB_PASSWORD | database.password |
| DB_SSLMODE | database.ssl_mode |
| DB_MAX_OPEN_CONNS | database.max_connections |
| DB_MAX_IDLE_CONNS | database.max_idle_conns |
| DB_CONN_MAX_LIFETIME | database.conn_max_lifetime |
| DB_LOG_LEVEL | database.log_level |

### Redis配置

| 环境变量 | YAML路径 |
|---------|---------|
| REDIS_HOST | redis.host |
| REDIS_PORT | redis.port |
| REDIS_PASSWORD | redis.password |
| REDIS_DB | redis.database |
| REDIS_ENABLED | redis.enabled |

### Genkit配置

| 环境变量 | YAML路径 |
|---------|---------|
| GENKIT_API_KEY | genkit.api_key |
| GENKIT_MODEL | genkit.model |
| GENKIT_DEFAULT_TEMPERATURE | genkit.default_temperature |
| GENKIT_DEFAULT_MAX_TOKENS | genkit.default_max_tokens |

### 认证配置

| 环境变量 | YAML路径 |
|---------|---------|
| JWT_SECRET | auth.jwt_secret |
| JWT_ISSUER | auth.jwt_issuer |
| JWT_AUDIENCE | auth.jwt_audience |
| ACCESS_TOKEN_TTL | auth.access_token_ttl |
| REFRESH_TOKEN_TTL | auth.refresh_token_ttl |
| BCRYPT_COST | auth.bcrypt_cost |
| MAX_LOGIN_ATTEMPTS | auth.max_login_attempts |
| LOGIN_ATTEMPT_WINDOW | auth.login_attempt_window |
| PASSWORD_MIN_LENGTH | auth.password_min_length |

## 常见问题

### Q1: 迁移后原有的环境变量还能用吗？

**A**: 可以。配置优先级是：环境变量 > YAML配置 > 默认值。环境变量会覆盖YAML配置。

### Q2: 必须迁移到YAML配置吗？

**A**: 不必须。纯环境变量配置仍然完全支持。YAML配置是可选的增强功能。

### Q3: 如何在Docker中使用YAML配置？

**A**: 将配置文件复制到镜像中，敏感信息通过环境变量注入：

```dockerfile
COPY config/prod.yaml /app/config/
ENV APP_ENV=production
ENV GENKIT_API_KEY=${GENKIT_API_KEY}
ENV JWT_SECRET=${JWT_SECRET}
```

### Q4: 如何在Kubernetes中使用？

**A**: 使用ConfigMap存储配置文件，Secret存储敏感信息：

```yaml
# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  prod.yaml: |
    server:
      port: "8080"
    # ...

# Secret
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
stringData:
  GENKIT_API_KEY: "your-api-key"
  JWT_SECRET: "your-jwt-secret"
```

### Q5: 配置文件变更后需要重启吗？

**A**: 是的，当前版本需要重启服务。未来版本可能支持热加载。

### Q6: 如何验证配置是否正确？

**A**: 启动服务时会自动验证配置并显示摘要。也可以使用：

```go
if err := config.ValidateConfig(cfg); err != nil {
    log.Fatal(err)
}
```

## 迁移检查清单

- [ ] 创建配置文件（dev.yaml, prod.yaml）
- [ ] 迁移非敏感配置到YAML
- [ ] 保留敏感信息在环境变量
- [ ] 更新部署脚本
- [ ] 测试开发环境
- [ ] 测试生产环境
- [ ] 更新文档
- [ ] 培训团队成员

## 回滚计划

如果迁移后遇到问题，可以快速回滚：

1. 恢复原有的环境变量设置
2. 删除或重命名配置文件
3. 重启服务

系统会自动回退到环境变量配置模式。

## 获取帮助

- 查看完整文档: [config/README.md](README.md)
- 查看快速入门: [config/QUICK_START.md](QUICK_START.md)
- 查看示例程序: [examples/config_example.go](../examples/config_example.go)

## 最佳实践

1. **开发环境**: 使用YAML配置 + .env文件
2. **生产环境**: 使用YAML配置 + 环境变量注入敏感信息
3. **测试环境**: 使用独立的test.yaml配置
4. **版本控制**: 提交YAML配置文件，不提交.env文件
5. **密钥管理**: 使用密钥管理服务（AWS Secrets Manager、HashiCorp Vault）

## 总结

新的配置系统提供了更灵活的配置方式，同时保持了完全的向后兼容性。您可以：

- 继续使用现有的环境变量配置
- 逐步迁移到YAML配置
- 使用混合模式（YAML + 环境变量）

选择最适合您团队和项目的方式！
