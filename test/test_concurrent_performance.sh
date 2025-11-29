#!/bin/bash

# 并发性能测试脚本
# 用于测试 Genkit 多模型支持的并发调用性能

set -e

echo "=========================================="
echo "并发性能测试"
echo "=========================================="
echo ""

# 检查必需的环境变量
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "❌ 错误：缺少 GOOGLE_API_KEY 环境变量"
    echo "请设置: export GOOGLE_API_KEY=your_api_key"
    exit 1
fi

echo "✓ 环境变量检查通过"
echo ""

# 设置测试数据库连接
export DB_HOST=${DB_HOST:-localhost}
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-postgres}
export DB_PASSWORD=${DB_PASSWORD:-postgres}
export DB_NAME=${DB_NAME:-genkit_ai_service_test}

echo "数据库配置:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo ""

# 运行并发性能测试
echo "=========================================="
echo "运行并发性能测试..."
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

go test -v -timeout 30m \
    ./test/e2e \
    -run TestConcurrentCallsPerformance \
    -count=1

echo ""
echo "=========================================="
echo "并发性能测试完成"
echo "=========================================="
