# 开发环境安装指南

## 概述

本指南将帮助您在 macOS 系统上安装 AI 知识管理平台的完整开发环境。

## 数据库管理库选择

我们选择了以下数据库管理库：

### 1. 数据库迁移

- **库名**: `golang-migrate/migrate`
- **用途**: 数据库版本控制和迁移管理
- **特点**: 支持多种数据库，命令行工具简单易用

### 2. 数据库驱动

- **库名**: `lib/pq`
- **用途**: PostgreSQL 数据库驱动
- **特点**: 纯 Go 实现，性能优秀

### 3. ORM 框架

- **库名**: `GORM`
- **用途**: 对象关系映射，简化数据库操作
- **特点**: 功能完整，支持自动迁移、关联、钩子等

## 自动安装（推荐）

### 1. 运行安装脚本

```bash
# 下载并运行安装脚本
curl -fsSL https://raw.githubusercontent.com/your-repo/ai-knowledge-platform/main/setup-dev-environment.sh | bash

# 或者如果您已经下载了脚本
chmod +x setup-dev-environment.sh
./setup-dev-environment.sh
```

### 2. 验证安装

```bash
# 检查 Go 版本
go version

# 检查 PostgreSQL
psql --version

# 检查 Redis
redis-cli ping

# 检查 MinIO
minio --version

# 检查开发工具
air -v
migrate -version
swag -v
```

## 手动安装

如果您希望手动安装，请按照以下步骤：

### 1. 安装 Homebrew

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. 安装 Go

```bash
brew install go

# 设置环境变量
echo "export GOPATH=\$HOME/go" >> ~/.zshrc
echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.zshrc
source ~/.zshrc
```

### 3. 安装 PostgreSQL

```bash
brew install postgresql@15
brew services start postgresql@15

# 创建数据库和用户
createdb aiplatform
psql aiplatform -c "CREATE USER aiuser WITH PASSWORD 'aipassword';"
psql aiplatform -c "GRANT ALL PRIVILEGES ON DATABASE aiplatform TO aiuser;"
psql aiplatform -c "ALTER USER aiuser CREATEDB;"
```

### 4. 安装 Redis

```bash
brew install redis
brew services start redis
```

### 5. 安装 MinIO

```bash
brew install minio/stable/minio
mkdir -p ~/minio-data
```

### 6. 安装开发工具

```bash
# 热重载工具
go install github.com/cosmtrek/air@latest

# 数据库迁移工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Swagger 文档生成工具
go install github.com/swaggo/swag/cmd/swag@latest
```

## 项目初始化

### 1. 创建项目目录

```bash
mkdir ai-knowledge-platform
cd ai-knowledge-platform
```

### 2. 初始化 Go 模块

```bash
go mod init ai-knowledge-platform
```

### 3. 创建目录结构

```bash
mkdir -p {cmd/server,internal/{api,service,repository,model,config,middleware},pkg/{database,cache,storage,llm,vector},configs,migrations,docs,scripts,test}
```

### 4. 创建配置文件

创建 `configs/config.yaml`:

```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 5432
  user: aiuser
  password: aipassword
  dbname: aiplatform
  sslmode: disable

redis:
  addr: localhost:6379
  password: ""
  db: 0

minio:
  endpoint: localhost:9000
  access_key: minioadmin
  secret_key: minioadmin
  bucket: ai-platform
  use_ssl: false

logging:
  level: info
  format: json
```

### 5. 创建 Docker Compose 文件

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: aiplatform
      POSTGRES_USER: aiuser
      POSTGRES_PASSWORD: aipassword
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

## 启动开发环境

### 1. 启动数据库服务

```bash
# 使用 Docker Compose 启动服务
docker-compose up -d

# 或者使用 Homebrew 服务
brew services start postgresql@15
brew services start redis
minio server ~/minio-data --console-address :9001 &
```

### 2. 运行数据库迁移

```bash
# 创建迁移文件
migrate create -ext sql -dir migrations -seq init_schema

# 运行迁移
migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" up
```

### 3. 启动开发服务器

```bash
# 使用热重载启动
air

# 或者直接运行
go run cmd/server/main.go
```

## 常用命令

### Makefile 命令

```bash
# 开发环境运行
make dev

# 构建应用
make build

# 运行测试
make test

# 数据库迁移
make migrate-up
make migrate-down

# 生成 Swagger 文档
make swagger

# Docker 服务管理
make docker-up
make docker-down
make docker-logs
```

### 数据库操作

```bash
# 连接数据库
psql -h localhost -U aiuser -d aiplatform

# 创建新的迁移文件
migrate create -ext sql -dir migrations -seq create_users_table

# 查看迁移状态
migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" version
```

### Redis 操作

```bash
# 连接 Redis
redis-cli

# 查看所有键
redis-cli keys "*"

# 清空数据库
redis-cli flushdb
```

## 服务地址

安装完成后，您可以通过以下地址访问各个服务：

- **API 服务**: <http://localhost:8080>
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **MinIO Console**: <http://localhost:9001> (用户名/密码: minioadmin/minioadmin)
- **MinIO API**: <http://localhost:9000>

## 故障排除

### 常见问题

1. **端口被占用**

   ```bash
   # 查看端口占用
   lsof -i :8080
   
   # 杀死进程
   kill -9 <PID>
   ```

2. **数据库连接失败**

   ```bash
   # 检查 PostgreSQL 状态
   brew services list | grep postgresql
   
   # 重启 PostgreSQL
   brew services restart postgresql@15
   ```

3. **Go 模块下载失败**

   ```bash
   # 设置 Go 代理
   go env -w GOPROXY=https://goproxy.cn,direct
   go env -w GOSUMDB=sum.golang.google.cn
   ```

4. **Docker 服务启动失败**

   ```bash
   # 检查 Docker 状态
   docker ps
   
   # 查看日志
   docker-compose logs
   
   # 重新构建
   docker-compose down -v
   docker-compose up -d
   ```

### 环境重置

如果需要完全重置开发环境：

```bash
# 停止所有服务
make docker-down
brew services stop postgresql@15
brew services stop redis

# 清理数据
docker-compose down -v
rm -rf ~/minio-data

# 重新启动
make docker-up
make migrate-up
```

## 下一步

环境安装完成后，您可以：

1. 查看项目规范文档：`.kiro/specs/ai-knowledge-platform/`
2. 开始实施第一个任务：项目基础架构搭建
3. 运行测试确保环境正常工作

如果遇到任何问题，请参考故障排除部分或查看项目文档。
