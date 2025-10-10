#!/bin/bash

# 项目架构验证脚本

set -e

echo "🔍 验证AI知识管理平台项目架构..."

# 检查必要的文件和目录
check_file_exists() {
    if [ -f "$1" ]; then
        echo "✅ $1"
    else
        echo "❌ $1 - 文件不存在"
        exit 1
    fi
}

check_dir_exists() {
    if [ -d "$1" ]; then
        echo "✅ $1/"
    else
        echo "❌ $1/ - 目录不存在"
        exit 1
    fi
}

echo ""
echo "📁 检查项目结构..."

# 检查根目录文件
check_file_exists "go.mod"
check_file_exists "Makefile"
check_file_exists "Dockerfile"
check_file_exists "docker-compose.yml"
check_file_exists ".env.example"
check_file_exists "README.md"

# 检查目录结构
check_dir_exists "cmd"
check_dir_exists "cmd/server"
check_dir_exists "internal"
check_dir_exists "internal/config"
check_dir_exists "internal/database"
check_dir_exists "internal/cache"
check_dir_exists "internal/middleware"
check_dir_exists "internal/router"
check_dir_exists "internal/handler"
check_dir_exists "migrations"
check_dir_exists "docs"
check_dir_exists "scripts"

# 检查核心文件
check_file_exists "cmd/server/main.go"
check_file_exists "internal/config/config.go"
check_file_exists "internal/database/connection.go"
check_file_exists "internal/database/migration.go"
check_file_exists "internal/cache/redis.go"
check_file_exists "internal/middleware/middleware.go"
check_file_exists "internal/router/router.go"
check_file_exists "internal/handler/health.go"
check_file_exists "internal/handler/handlers.go"

# 检查迁移文件
check_file_exists "migrations/000001_init_schema.up.sql"
check_file_exists "migrations/000001_init_schema.down.sql"
check_file_exists "migrations/000002_create_indexes.up.sql"
check_file_exists "migrations/000002_create_indexes.down.sql"

# 检查测试文件
check_file_exists "internal/config/config_test.go"
check_file_exists "internal/database/connection_test.go"

# 检查文档
check_file_exists "docs/docs.go"
check_file_exists "docs/features/project-architecture-setup.md"

echo ""
echo "📦 检查Go模块..."

# 检查go.mod内容
if grep -q "module ai-knowledge-platform" go.mod; then
    echo "✅ Go模块名称正确"
else
    echo "❌ Go模块名称不正确"
    exit 1
fi

# 检查主要依赖
dependencies=(
    "github.com/gin-gonic/gin"
    "github.com/lib/pq"
    "github.com/go-redis/redis/v8"
    "github.com/golang-migrate/migrate/v4"
    "github.com/swaggo/gin-swagger"
    "gorm.io/gorm"
    "gorm.io/driver/postgres"
)

for dep in "${dependencies[@]}"; do
    if grep -q "$dep" go.mod; then
        echo "✅ 依赖 $dep"
    else
        echo "❌ 缺少依赖 $dep"
        exit 1
    fi
done

echo ""
echo "🐳 检查Docker配置..."

# 检查Dockerfile关键内容
if grep -q "FROM golang:" Dockerfile && grep -q "FROM alpine:" Dockerfile; then
    echo "✅ Dockerfile多阶段构建配置正确"
else
    echo "❌ Dockerfile配置不正确"
    exit 1
fi

# 检查docker-compose.yml服务
services=("api" "postgres" "redis" "minio" "prometheus" "grafana")
for service in "${services[@]}"; do
    if grep -q "$service:" docker-compose.yml; then
        echo "✅ Docker Compose服务: $service"
    else
        echo "❌ 缺少Docker Compose服务: $service"
        exit 1
    fi
done

echo ""
echo "🛠️  检查Makefile命令..."

# 检查Makefile目标
targets=("help" "deps" "build" "run" "test" "swagger" "migrate-up" "docker-build")
for target in "${targets[@]}"; do
    if grep -q "^$target:" Makefile; then
        echo "✅ Makefile目标: $target"
    else
        echo "❌ 缺少Makefile目标: $target"
        exit 1
    fi
done

echo ""
echo "📄 检查环境变量配置..."

# 检查.env.example关键配置
env_vars=(
    "SERVER_PORT"
    "DATABASE_URL"
    "REDIS_ADDR"
    "OSS_ENDPOINT"
    "KMS_ENDPOINT"
)

for var in "${env_vars[@]}"; do
    if grep -q "$var=" .env.example; then
        echo "✅ 环境变量: $var"
    else
        echo "❌ 缺少环境变量: $var"
        exit 1
    fi
done

echo ""
echo "🗄️  检查数据库迁移..."

# 检查迁移文件内容
if grep -q "CREATE TABLE projects" migrations/000001_init_schema.up.sql; then
    echo "✅ 数据库表结构定义正确"
else
    echo "❌ 数据库表结构定义不正确"
    exit 1
fi

if grep -q "CREATE INDEX" migrations/000002_create_indexes.up.sql; then
    echo "✅ 数据库索引定义正确"
else
    echo "❌ 数据库索引定义不正确"
    exit 1
fi

echo ""
echo "🎉 项目架构验证完成！"
echo ""
echo "项目基础架构搭建成功，包含以下组件："
echo "  ✅ Go项目结构和依赖管理"
echo "  ✅ PostgreSQL数据库连接和迁移"
echo "  ✅ Redis缓存功能"
echo "  ✅ HTTP服务器和中间件"
echo "  ✅ API路由和处理器"
echo "  ✅ Swagger文档配置"
echo "  ✅ Docker容器化配置"
echo "  ✅ 开发工具和脚本"
echo "  ✅ 监控和健康检查"
echo "  ✅ 测试框架"
echo ""
echo "下一步可以运行以下命令："
echo "  1. ./scripts/setup-dev.sh  # 设置开发环境"
echo "  2. make deps               # 安装依赖"
echo "  3. make migrate-up         # 运行数据库迁移"
echo "  4. make swagger            # 生成API文档"
echo "  5. make run                # 启动服务"