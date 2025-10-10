#!/bin/bash

# 环境测试脚本
# 测试所有服务是否正常运行

set -e

echo "🧪 测试开发环境连接..."
echo "======================"

# 测试 PostgreSQL 连接
echo "🐘 测试 PostgreSQL 连接..."
if docker exec -it $(docker-compose ps -q postgres) psql -U aiuser -d aiplatform -c "SELECT version();" > /dev/null 2>&1; then
    echo "✅ PostgreSQL 连接成功"
else
    echo "❌ PostgreSQL 连接失败"
    exit 1
fi

# 测试 Redis 连接
echo "🔴 测试 Redis 连接..."
if docker exec -it $(docker-compose ps -q redis) redis-cli ping | grep -q "PONG"; then
    echo "✅ Redis 连接成功"
else
    echo "❌ Redis 连接失败"
    exit 1
fi

# 测试 MinIO 连接
echo "📦 测试 MinIO 连接..."
if curl -s http://localhost:9000/minio/health/live > /dev/null; then
    echo "✅ MinIO 连接成功"
else
    echo "❌ MinIO 连接失败"
    exit 1
fi

# 测试 Go 环境
echo "🐹 测试 Go 环境..."
if go version > /dev/null 2>&1; then
    echo "✅ Go 环境正常: $(go version)"
else
    echo "❌ Go 环境异常"
    exit 1
fi

# 测试开发工具
echo "🛠️  测试开发工具..."
if air -v > /dev/null 2>&1; then
    echo "✅ Air 工具正常"
else
    echo "❌ Air 工具异常"
fi

if migrate -version > /dev/null 2>&1; then
    echo "✅ Migrate 工具正常"
else
    echo "❌ Migrate 工具异常"
fi

if swag -v > /dev/null 2>&1; then
    echo "✅ Swag 工具正常"
else
    echo "❌ Swag 工具异常"
fi

echo ""
echo "🎉 环境测试完成！所有服务运行正常。"
echo ""
echo "📊 服务状态："
docker-compose ps

echo ""
echo "💡 提示："
echo "  • 如果某个服务异常，请运行: make docker-logs"
echo "  • 重置环境请运行: make reset-db"
echo "  • 停止服务请运行: make docker-down"
EOF