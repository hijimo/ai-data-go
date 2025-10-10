#!/bin/bash

# 开发环境设置脚本

set -e

echo "🚀 开始设置AI知识管理平台开发环境..."

# 检查是否安装了Homebrew
if ! command -v brew &> /dev/null; then
    echo "❌ 未找到Homebrew，请先安装Homebrew:"
    echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    exit 1
fi

echo "✅ 找到Homebrew"

# 安装Go
if ! command -v go &> /dev/null; then
    echo "📦 正在安装Go..."
    brew install go
else
    echo "✅ Go已安装: $(go version)"
fi

# 安装PostgreSQL
if ! command -v psql &> /dev/null; then
    echo "📦 正在安装PostgreSQL..."
    brew install postgresql@15
    brew services start postgresql@15
else
    echo "✅ PostgreSQL已安装"
fi

# 安装Redis
if ! command -v redis-server &> /dev/null; then
    echo "📦 正在安装Redis..."
    brew install redis
    brew services start redis
else
    echo "✅ Redis已安装"
fi

# 安装migrate工具
if ! command -v migrate &> /dev/null; then
    echo "📦 正在安装migrate工具..."
    brew install golang-migrate
else
    echo "✅ migrate工具已安装"
fi

# 安装swag工具
if ! command -v swag &> /dev/null; then
    echo "📦 正在安装swag工具..."
    go install github.com/swaggo/swag/cmd/swag@latest
else
    echo "✅ swag工具已安装"
fi

# 安装golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "📦 正在安装golangci-lint..."
    brew install golangci-lint
else
    echo "✅ golangci-lint已安装"
fi

# 创建数据库
echo "🗄️  设置数据库..."
createdb ai_knowledge_platform 2>/dev/null || echo "数据库可能已存在"

# 复制环境变量文件
if [ ! -f .env ]; then
    echo "📝 创建环境变量文件..."
    cp .env.example .env
    echo "请编辑 .env 文件配置您的环境变量"
fi

# 初始化Go模块
if [ ! -f go.mod ]; then
    echo "📦 初始化Go模块..."
    go mod init ai-knowledge-platform
fi

# 下载依赖
echo "📦 下载Go依赖..."
go mod tidy

echo ""
echo "🎉 开发环境设置完成！"
echo ""
echo "下一步："
echo "1. 编辑 .env 文件配置环境变量"
echo "2. 运行数据库迁移: make migrate-up"
echo "3. 生成API文档: make swagger"
echo "4. 启动服务: make run"
echo ""
echo "访问 http://localhost:8080/swagger/index.html 查看API文档"