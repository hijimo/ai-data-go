# 配置管理快速入门

## 5分钟快速开始

### 1. 准备环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑.env文件，设置必需的变量
# 必需: GENKIT_API_KEY, JWT_SECRET
```

### 2. 选择配置方式

#### 方式A: 使用YAML配置（推荐）

```bash
# 设置环境
export APP_ENV=development

# 启动服务（自动加载 config/dev.yaml）
./server
```

#### 方式B: 仅使用环境变量

```bash
# 设置所有必需的环境变量
export GENKIT_API_KEY=your-api-key
export JWT_SECRET=your-jwt-secret-min-32-chars
export DATABASE_URL=postgres://user:pass@localhost:5432/dbname

# 启动服务
./server
```

### 3. 验证配置

启动时会显示配置摘要：

```
========================================
配置加载成功
========================================
环境: development
服务器: 0.0.0.0:8080
数据库: localhost:5432/genkit_ai_service
Redis: localhost:6379 (DB:0)
Genkit模型: gemini-2.5-flash
日志级别: debug
日志格式: json
========================================
```

## 常用配置

### 开发环境

```bash
export APP_ENV=development
export GENKIT_API_KEY=your-api-key
export JWT_SECRET=dev-secret-key-change-in-production-min-32-chars
```

### 生产环境

```bash
export APP_ENV=production
export GENKIT_API_KEY=your-api-key
export JWT_SECRET=your-production-jwt-secret
export DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
export REDIS_HOST=redis.example.com
export REDIS_PASSWORD=your-redis-password
```

### Docker

```bash
docker run -e APP_ENV=production \
  -e GENKIT_API_KEY=your-api-key \
  -e JWT_SECRET=your-jwt-secret \
  -e DATABASE_URL=postgres://... \
  your-image
```

## 环境变量替换

在YAML文件中使用环境变量：

```yaml
# 必须存在
api_key: "${GENKIT_API_KEY}"

# 带默认值
port: "${SERVER_PORT:8080}"
host: "${SERVER_HOST:0.0.0.0}"
```

## 故障排查

### 问题1: 配置文件未找到

```
错误: 读取配置文件失败: open config/prod.yaml: no such file or directory
```

**解决**: 检查APP_ENV环境变量或使用CONFIG_FILE指定路径

### 问题2: 环境变量未设置

```
错误: 配置验证失败: Genkit API密钥不能为空
```

**解决**: 设置GENKIT_API_KEY环境变量

### 问题3: JWT密钥太短

```
错误: JWT 签名密钥长度必须至少为 32 个字符
```

**解决**: 使用至少32字符的JWT_SECRET

## 更多信息

- 完整文档: [config/README.md](README.md)
- 示例程序: [examples/config_example.go](../examples/config_example.go)
- 配置文件: [config/dev.yaml](dev.yaml), [config/prod.yaml](prod.yaml)
