#!/bin/bash

# 内存使用测试脚本
# 用于测试 Genkit 多模型支持的内存使用情况

set -e

echo "=========================================="
echo "内存使用测试"
echo "=========================================="
echo ""

# 检查环境变量
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "❌ 错误：缺少 GOOGLE_API_KEY 环境变量"
    echo "请设置 GOOGLE_API_KEY 环境变量"
    exit 1
fi

echo "✓ 环境变量检查通过"
echo ""

# 检查数据库连接
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  警告：未设置 DATABASE_URL，使用默认配置"
    export DATABASE_URL="postgres://postgres:postgres@localhost:5432/genkit_test?sslmode=disable"
fi

echo "数据库连接: $DATABASE_URL"
echo ""

# 运行内存使用测试
echo "=========================================="
echo "运行内存使用测试..."
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

# 运行测试，显示详细输出
go test -v -run TestMemoryUsage ./test/e2e/

TEST_EXIT_CODE=$?

echo ""
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✓ 内存使用测试通过"
else
    echo "❌ 内存使用测试失败"
fi
echo "=========================================="

exit $TEST_EXIT_CODE
