#!/bin/bash

# AI知识管理平台快速环境安装脚本
# 适用于 macOS 系统

set -e

echo "🚀 AI知识管理平台 - 快速环境安装"
echo "=================================="

# 检查 Homebrew
if ! command -v brew &> /dev/null; then
    echo "📦 安装 Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zshrc
    eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# 安装 Go
if ! command -v go &> /dev/null; then
    echo "🐹 安装 Go..."
    brew install go
    echo "export GOPATH=\$HOME/go" >> ~/.zshrc
    echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.zshrc
    source ~/.zshrc
fi

# 安装 Docker Desktop
if ! command -v docker &> /dev/null; then
    echo "🐳 安装 Docker Desktop..."
    brew install --cask docker
    echo "⚠️  请手动启动 Docker Desktop 应用，然后按回车继续..."
    read -p "Docker Desktop 启动完成后按回车继续..."
fi

# 安装开发工具
echo "🛠️  安装开发工具..."
go install github.com/cosmtrek/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/swaggo/swag/cmd/swag@latest

# 创建项目
echo "📁 创建项目..."
mkdir -p ai-knowledge-platform
cd ai-knowledge-platform

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
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
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U aiuser -d aiplatform"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

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
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

volumes:
  postgres_data:
  redis_data:
  minio_data:
EOF

# 启动服务
echo "🚀 启动数据库服务..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "🔍 检查服务状态..."
docker-compose ps

# 创建项目结构
echo "📂 创建项目结构..."
mkdir -p {cmd/server,internal/{api,service,repository,model,config,middleware},pkg/{database,cache,storage,llm,vector},configs,migrations,docs,scripts,test}

# 创建配置文件
cat > configs/config.yaml << 'EOF'
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
EOF

# 创建 go.mod
cat > go.mod << 'EOF'
module ai-knowledge-platform

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/golang-jwt/jwt/v5 v5.0.0
    github.com/golang-migrate/migrate/v4 v4.16.2
    github.com/hibiken/asynq v0.24.1
    github.com/lib/pq v1.10.9
    github.com/minio/minio-go/v7 v7.0.63
    github.com/prometheus/client_golang v1.17.0
    github.com/redis/go-redis/v9 v9.2.1
    github.com/sirupsen/logrus v1.9.3
    github.com/spf13/viper v1.17.0
    github.com/swaggo/gin-swagger v1.6.0
    github.com/swaggo/swag v1.16.2
    go.opentelemetry.io/otel v1.19.0
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)
EOF

# 创建 Makefile
cat > Makefile << 'EOF'
.PHONY: dev build test migrate-up migrate-down docker-up docker-down

dev:
	air

build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

migrate-up:
	migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" down

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

swagger:
	swag init -g cmd/server/main.go -o docs

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

reset-db:
	docker-compose down -v
	docker-compose up -d
	sleep 30
	make migrate-up
EOF

# 创建 .air.toml
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 0
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

# 下载依赖
echo "📦 下载 Go 依赖..."
go mod tidy

echo ""
echo "🎉 安装完成！"
echo "=============="
echo ""
echo "📊 数据库管理库选择："
echo "  • 迁移工具: golang-migrate/migrate"
echo "  • 数据库驱动: lib/pq (PostgreSQL)"
echo "  • ORM框架: GORM"
echo ""
echo "🌐 服务地址："
echo "  • PostgreSQL: localhost:5432 (用户: aiuser, 密码: aipassword, 数据库: aiplatform)"
echo "  • Redis: localhost:6379"
echo "  • MinIO Console: http://localhost:9001 (用户: minioadmin, 密码: minioadmin)"
echo "  • MinIO API: http://localhost:9000"
echo ""
echo "🚀 快速开始："
echo "  1. cd ai-knowledge-platform"
echo "  2. make migrate-create name=init_schema  # 创建初始迁移文件"
echo "  3. make migrate-up                      # 运行数据库迁移"
echo "  4. make dev                             # 启动开发服务器"
echo ""
echo "📚 常用命令："
echo "  • make docker-logs    # 查看服务日志"
echo "  • make docker-down    # 停止服务"
echo "  • make docker-up      # 启动服务"
echo "  • make reset-db       # 重置数据库"
echo ""
echo "📝 下一步："
echo "  1. 查看规范文档: .kiro/specs/ai-knowledge-platform/"
echo "  2. 开始第一个任务: 项目基础架构搭建"
echo ""
EOF