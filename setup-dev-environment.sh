#!/bin/bash

# AI知识管理平台开发环境安装脚本
# 支持 macOS 系统

set -e

echo "🚀 开始安装AI知识管理平台开发环境..."

# 检查操作系统
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ 此脚本仅支持 macOS 系统"
    exit 1
fi

# 检查并安装 Homebrew
if ! command -v brew &> /dev/null; then
    echo "📦 安装 Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
    echo "✅ Homebrew 已安装"
fi

# 更新 Homebrew
echo "🔄 更新 Homebrew..."
brew update

# 安装 Go
if ! command -v go &> /dev/null; then
    echo "🐹 安装 Go..."
    brew install go
    
    # 设置 Go 环境变量
    echo "export GOPATH=\$HOME/go" >> ~/.zshrc
    echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.zshrc
    source ~/.zshrc
else
    echo "✅ Go 已安装，版本: $(go version)"
fi

# 安装 PostgreSQL
if ! command -v psql &> /dev/null; then
    echo "🐘 安装 PostgreSQL..."
    brew install postgresql@15
    
    # 启动 PostgreSQL 服务
    brew services start postgresql@15
    
    # 创建数据库用户和数据库
    echo "📊 创建数据库..."
    createdb aiplatform
    psql aiplatform -c "CREATE USER aiuser WITH PASSWORD 'aipassword';"
    psql aiplatform -c "GRANT ALL PRIVILEGES ON DATABASE aiplatform TO aiuser;"
    psql aiplatform -c "ALTER USER aiuser CREATEDB;"
else
    echo "✅ PostgreSQL 已安装"
fi

# 安装 Redis
if ! command -v redis-server &> /dev/null; then
    echo "🔴 安装 Redis..."
    brew install redis
    
    # 启动 Redis 服务
    brew services start redis
else
    echo "✅ Redis 已安装"
fi

# 安装 MinIO (本地对象存储)
if ! command -v minio &> /dev/null; then
    echo "📦 安装 MinIO..."
    brew install minio/stable/minio
    
    # 创建 MinIO 数据目录
    mkdir -p ~/minio-data
    
    echo "💡 MinIO 安装完成，启动命令："
    echo "minio server ~/minio-data --console-address :9001"
else
    echo "✅ MinIO 已安装"
fi

# 安装 Docker (可选，用于容器化部署)
if ! command -v docker &> /dev/null; then
    echo "🐳 安装 Docker Desktop..."
    brew install --cask docker
    echo "⚠️  请手动启动 Docker Desktop 应用"
else
    echo "✅ Docker 已安装"
fi

# 安装开发工具
echo "🛠️  安装开发工具..."

# 安装 Air (Go 热重载工具)
if ! command -v air &> /dev/null; then
    go install github.com/cosmtrek/air@latest
fi

# 安装 golang-migrate
if ! command -v migrate &> /dev/null; then
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

# 安装 swag (Swagger 文档生成)
if ! command -v swag &> /dev/null; then
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# 创建项目目录结构
echo "📁 创建项目目录结构..."
mkdir -p ai-knowledge-platform/{cmd/server,internal/{api,service,repository,model,config,middleware},pkg/{database,cache,storage,llm,vector},configs,migrations,docs,scripts,test}

# 创建基础配置文件
cat > ai-knowledge-platform/configs/config.yaml << EOF
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

# 创建 docker-compose.yml 用于本地开发
cat > ai-knowledge-platform/docker-compose.yml << EOF
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
EOF

# 创建 go.mod 文件
cat > ai-knowledge-platform/go.mod << EOF
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
cat > ai-knowledge-platform/Makefile << EOF
.PHONY: dev build test migrate-up migrate-down docker-up docker-down

# 开发环境运行
dev:
	air

# 构建应用
build:
	go build -o bin/server cmd/server/main.go

# 运行测试
test:
	go test -v ./...

# 数据库迁移 - 向上
migrate-up:
	migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" up

# 数据库迁移 - 向下
migrate-down:
	migrate -path migrations -database "postgres://aiuser:aipassword@localhost:5432/aiplatform?sslmode=disable" down

# 创建新的迁移文件
migrate-create:
	migrate create -ext sql -dir migrations -seq \$(name)

# 生成 Swagger 文档
swagger:
	swag init -g cmd/server/main.go -o docs

# 启动 Docker 服务
docker-up:
	docker-compose up -d

# 停止 Docker 服务
docker-down:
	docker-compose down

# 查看 Docker 日志
docker-logs:
	docker-compose logs -f

# 重置数据库
reset-db:
	docker-compose down -v
	docker-compose up -d postgres
	sleep 5
	make migrate-up
EOF

# 创建 .air.toml 配置文件（热重载）
cat > ai-knowledge-platform/.air.toml << EOF
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

echo ""
echo "🎉 开发环境安装完成！"
echo ""
echo "📋 安装的组件："
echo "  ✅ Go $(go version | cut -d' ' -f3)"
echo "  ✅ PostgreSQL"
echo "  ✅ Redis"
echo "  ✅ MinIO"
echo "  ✅ Docker"
echo "  ✅ 开发工具 (air, migrate, swag)"
echo ""
echo "🚀 快速开始："
echo "  1. cd ai-knowledge-platform"
echo "  2. make docker-up          # 启动数据库服务"
echo "  3. make migrate-up         # 运行数据库迁移"
echo "  4. make dev                # 启动开发服务器"
echo ""
echo "🌐 服务地址："
echo "  • API服务: http://localhost:8080"
echo "  • PostgreSQL: localhost:5432"
echo "  • Redis: localhost:6379"
echo "  • MinIO Console: http://localhost:9001"
echo "  • MinIO API: http://localhost:9000"
echo ""
echo "📚 有用的命令："
echo "  • make docker-logs         # 查看服务日志"
echo "  • make test               # 运行测试"
echo "  • make swagger            # 生成API文档"
echo "  • make migrate-create name=create_users_table  # 创建迁移文件"
echo ""
echo "⚠️  注意事项："
echo "  • 请确保 Docker Desktop 已启动"
echo "  • 首次运行需要下载 Docker 镜像，可能需要一些时间"
echo "  • MinIO 默认用户名/密码: minioadmin/minioadmin"
EOF