#!/bin/bash

# 数据库初始化脚本
# 用于开发环境的快速数据库设置

set -e

echo "🚀 开始初始化AI知识管理平台数据库..."

# 检查环境变量
if [ -f .env ]; then
    echo "📋 加载环境变量..."
    source .env
else
    echo "⚠️  未找到.env文件，请先复制.env.example并配置数据库连接"
    exit 1
fi

# 检查数据库连接
echo "🔍 检查数据库连接..."
if ! pg_isready -h ${DB_HOST:-localhost} -p ${DB_PORT:-5432} -U ${DB_USER:-postgres} > /dev/null 2>&1; then
    echo "❌ 无法连接到数据库，请确保PostgreSQL服务正在运行"
    echo "   主机: ${DB_HOST:-localhost}"
    echo "   端口: ${DB_PORT:-5432}"
    echo "   用户: ${DB_USER:-postgres}"
    exit 1
fi

echo "✅ 数据库连接正常"

# 创建数据库（如果不存在）
echo "📦 创建数据库 ${DB_NAME:-ai_knowledge_platform}..."
createdb -h ${DB_HOST:-localhost} -p ${DB_PORT:-5432} -U ${DB_USER:-postgres} ${DB_NAME:-ai_knowledge_platform} 2>/dev/null || echo "数据库已存在，跳过创建"

# 构建迁移工具
echo "🔨 构建迁移工具..."
make build-migrate

# 运行数据库迁移
echo "📊 运行数据库迁移..."
make migrate-up

# 初始化种子数据
echo "🌱 初始化种子数据..."
make db-seed

echo "🎉 数据库初始化完成！"
echo ""
echo "📋 数据库信息:"
echo "   主机: ${DB_HOST:-localhost}"
echo "   端口: ${DB_PORT:-5432}"
echo "   数据库: ${DB_NAME:-ai_knowledge_platform}"
echo "   用户: ${DB_USER:-postgres}"
echo ""
echo "🔧 可用的数据库管理命令:"
echo "   make migrate-up      - 运行迁移"
echo "   make migrate-down    - 回滚迁移"
echo "   make migrate-version - 查看版本"
echo "   make db-seed         - 重新初始化种子数据"
echo "   make db-clean        - 清理种子数据"
echo ""
echo "✨ 现在可以启动应用程序: make run"