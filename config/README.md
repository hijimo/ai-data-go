# 配置管理说明

## 概述

本系统支持两种配置方式：

1. **YAML配置文件**：适合结构化配置，支持环境变量替换
2. **环境变量**：适合容器化部署和敏感信息配置

配置优先级：**环境变量 > YAML配置文件 > 默认值**

## 配置文件

### 文件结构

```
config/
├── dev.yaml      # 开发环境配置
├── prod.yaml     # 生产环境配置
├── test.yaml     # 测试环境配置
└── README.md     # 本文档
```

### 环境选择

系统通过以下环境变量确定当前环境：

- `APP_ENV` 或 `GO_ENV`：环境名称（development, production, test）
- `CONFIG_FILE`：直接指定配置文件路径（优先级最高）

示例：

```bash
# 使用开发环境配置
export APP_ENV=development
./server

# 使用生产环境配置
export APP_ENV=production
./server

# 直接指定配置文件
export CONFIG_FILE=/path/to/custom.yaml
./server
```

## 环境变量替换

YAML配置文件支持环境变量替换，格式如下：

### 基本格式

```yaml
# 使用环境变量（必须存在）
api_key: "${GENKIT_API_KEY}"

# 使用环境变量，提供默认值
port: "${SERVER_PORT:8080}"
host: "${SERVER_HOST:0.0.0.0}"
```

### 替换规则

- `${VAR_NAME}`：使用环境变量VAR_NAME的值，如果不存在则为空字符串
- `${VAR_NAME:default}`：使用环境变量VAR_NAME的值，如果不存在则使用default

## 配置项说明

### 服务器配置 (server)

```yaml
server:
  port: "8080"           # 服务器端口
  host: "0.0.0.0"        # 监听地址
  mode: debug            # 运行模式: debug, release
```

### Genkit配置 (genkit)

```yaml
genkit:
  provider: google                # AI提供商: google, openai
  api_key: "${GENKIT_API_KEY}"   # API密钥（必需）
  model: "gemini-2.5-flash"      # 默认模型
  default_temperature: 0.7        # 默认温度参数 (0-2)
  default_max_tokens: 2000        # 默认最大token数
  log_level: info                 # 日志级别
  timeout: "30s"                  # 请求超时时间
```

### 数据库配置 (database)

```yaml
database:
  # 方式1: 使用连接URL（推荐生产环境）
  url: "${DATABASE_URL}"
  
  # 方式2: 使用独立配置项
  host: "localhost"
  port: 5432
  database: "genkit_ai_service"
  user: "postgres"
  password: "${DB_PASSWORD}"
  ssl_mode: disable              # disable, require, verify-full
  
  # 连接池配置
  max_connections: 50
  max_idle_conns: 10
  conn_max_lifetime: "10m"
  
  # 日志级别
  log_level: error               # silent, error, warn, info
```

### Redis配置 (redis)

```yaml
redis:
  host: "localhost"
  port: 6379
  password: "${REDIS_PASSWORD}"
  database: 0
  enabled: true
```

### 日志配置 (log)

```yaml
log:
  level: info                    # debug, info, warn, error
  format: json                   # json, text
  enable_file: true              # 是否启用文件日志
  log_dir: "/var/log/genkit"    # 日志目录
  enable_console: true           # 是否输出到控制台
```

### 会话配置 (session)

```yaml
session:
  timeout: "30m"                 # 会话超时时间
  cleanup_interval: "5m"         # 清理间隔
  summary_threshold: 50          # 摘要生成阈值（消息数）
  default_page_size: 20          # 默认分页大小
  max_page_size: 100             # 最大分页大小
  max_title_length: 255          # 标题最大长度
```

### 认证配置 (auth)

```yaml
auth:
  jwt_secret: "${JWT_SECRET}"              # JWT签名密钥（必需，至少32字符）
  jwt_issuer: "genkit-ai-service"          # JWT签发者
  jwt_audience: "genkit-api"               # JWT受众
  access_token_ttl: "60m"                  # Access Token有效期
  refresh_token_ttl: "720h"                # Refresh Token有效期
  bcrypt_cost: 12                          # Bcrypt加密强度 (4-31)
  max_login_attempts: 5                    # 最大登录尝试次数
  login_attempt_window: "15m"              # 登录尝试时间窗口
  password_min_length: 12                  # 密码最小长度
  enable_refresh_rotation: true            # 是否启用Token轮换
  tenant_identify_strategy: header         # 租户识别策略
  token_cleanup_interval: "1h"             # Token清理间隔
  enable_token_blacklist: true             # 是否启用黑名单
```

### 系统初始化配置 (bootstrap)

```yaml
bootstrap:
  admin_email: "${PLATFORM_ADMIN_EMAIL}"           # 管理员邮箱（必需）
  admin_password: "${PLATFORM_ADMIN_PASSWORD}"     # 管理员密码（留空自动生成）
  admin_display_name: "Platform Admin"             # 管理员显示名称
  tenant_name: "Platform"                          # 平台租户名称
  tenant_domain: "${PLATFORM_TENANT_DOMAIN}"       # 平台租户域名
```

### 监控配置 (monitoring)

```yaml
monitoring:
  prometheus_port: 9090                    # Prometheus端口
  jaeger_endpoint: "${JAEGER_ENDPOINT}"    # Jaeger端点
  enable_tracing: true                     # 是否启用追踪
  enable_metrics: true                     # 是否启用指标
  metrics_path: "/metrics"                 # 指标路径
  tracing_sampling: 0.1                    # 追踪采样率 (0-1)
```

### 缓存配置 (cache)

```yaml
cache:
  namespace: "genkit:prod"                 # 缓存命名空间
  default_ttl: "5m"                        # 默认TTL
  enable_warmup: true                      # 是否启用预热
  warmup_interval: "30m"                   # 预热间隔
  
  # 各类缓存的TTL
  context_ttl: "5m"                        # 上下文缓存
  vector_search_ttl: "30m"                 # 向量检索缓存
  summary_ttl: "1h"                        # 摘要缓存
  session_list_ttl: "10m"                  # 会话列表缓存
  token_usage_ttl: "5m"                    # Token使用统计缓存
  
  # 自定义缓存TTL
  custom_ttl:
    user_profile: "15m"
    tenant_config: "30m"
```

### 向量服务配置 (vector)

```yaml
vector:
  provider: google                         # 向量服务提供商
  embedding_model: "text-embedding-004"    # 嵌入模型
  dimension: 768                           # 向量维度
  batch_size: 20                           # 批处理大小
  timeout: "60s"                           # 超时时间
```

## 使用示例

### 开发环境

```bash
# 1. 复制环境变量模板
cp .env.example .env

# 2. 编辑.env文件，设置必需的环境变量
# GENKIT_API_KEY=your-api-key
# JWT_SECRET=your-jwt-secret-min-32-chars

# 3. 启动服务（自动使用dev.yaml）
export APP_ENV=development
./server
```

### 生产环境

```bash
# 1. 设置环境变量
export APP_ENV=production
export GENKIT_API_KEY=your-api-key
export JWT_SECRET=your-jwt-secret
export DATABASE_URL=postgres://user:pass@host:5432/dbname
export REDIS_HOST=redis.example.com
export REDIS_PASSWORD=your-redis-password

# 2. 启动服务
./server
```

### Docker部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
COPY config ./config

# 设置环境
ENV APP_ENV=production

CMD ["./server"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    environment:
      - APP_ENV=production
      - GENKIT_API_KEY=${GENKIT_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - DATABASE_URL=postgres://postgres:postgres@db:5432/genkit
      - REDIS_HOST=redis
    ports:
      - "8080:8080"
    depends_on:
      - db
      - redis
  
  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=genkit
      - POSTGRES_PASSWORD=postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### Kubernetes部署

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: genkit-config
data:
  APP_ENV: "production"
  SERVER_PORT: "8080"
  DB_HOST: "postgres-service"
  REDIS_HOST: "redis-service"

---
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: genkit-secrets
type: Opaque
stringData:
  GENKIT_API_KEY: "your-api-key"
  JWT_SECRET: "your-jwt-secret-min-32-chars"
  DB_PASSWORD: "your-db-password"
  REDIS_PASSWORD: "your-redis-password"

---
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: genkit-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: genkit-service
  template:
    metadata:
      labels:
        app: genkit-service
    spec:
      containers:
      - name: genkit-service
        image: genkit-service:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: genkit-config
        - secretRef:
            name: genkit-secrets
```

## 配置验证

系统启动时会自动验证配置的有效性，包括：

- 必需字段检查
- 数据类型验证
- 取值范围验证
- 格式验证（如邮箱、端口号）
- 依赖关系验证

如果配置无效，系统会输出详细的错误信息并拒绝启动。

## 最佳实践

### 1. 敏感信息管理

- ✅ 使用环境变量存储敏感信息（API密钥、密码）
- ✅ 不要将敏感信息提交到版本控制系统
- ✅ 使用密钥管理服务（如AWS Secrets Manager、HashiCorp Vault）
- ❌ 不要在YAML文件中硬编码敏感信息

### 2. 环境隔离

- ✅ 为每个环境使用独立的配置文件
- ✅ 开发环境使用宽松的配置（如低bcrypt cost）
- ✅ 生产环境使用严格的配置（如高bcrypt cost、SSL）
- ✅ 测试环境使用独立的数据库和Redis

### 3. 配置管理

- ✅ 使用版本控制管理配置文件模板
- ✅ 使用配置管理工具（如Ansible、Terraform）
- ✅ 定期审查和更新配置
- ✅ 记录配置变更历史

### 4. 安全建议

- ✅ JWT密钥至少32字符，使用强随机字符串
- ✅ 生产环境使用SSL/TLS连接数据库
- ✅ 定期轮换密钥和密码
- ✅ 限制配置文件的访问权限（chmod 600）
- ✅ 使用防火墙限制服务访问

## 故障排查

### 配置文件未找到

```
错误: 读取配置文件失败: open config/prod.yaml: no such file or directory
```

解决方案：

1. 检查配置文件是否存在
2. 检查APP_ENV环境变量是否正确
3. 使用CONFIG_FILE直接指定配置文件路径

### 环境变量未设置

```
错误: 配置验证失败: Genkit API密钥不能为空
```

解决方案：

1. 检查环境变量是否已设置：`echo $GENKIT_API_KEY`
2. 在YAML中提供默认值：`api_key: "${GENKIT_API_KEY:default-value}"`
3. 使用.env文件管理环境变量

### 配置验证失败

```
错误: 配置验证失败: JWT 签名密钥长度必须至少为 32 个字符
```

解决方案：

1. 检查配置项是否符合要求
2. 查看错误信息中的具体要求
3. 参考本文档的配置项说明

## 参考资料

- [YAML语法](https://yaml.org/)
- [Go time.Duration格式](https://pkg.go.dev/time#ParseDuration)
- [PostgreSQL连接字符串](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Redis配置](https://redis.io/docs/management/config/)
