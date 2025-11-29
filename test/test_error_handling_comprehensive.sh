#!/bin/bash

# 全面错误处理测试脚本
# 测试所有错误场景，确保系统能够正确处理各种异常情况

set -e

echo "========================================="
echo "全面错误处理测试"
echo "========================================="
echo ""

# 检查环境变量
if [ -z "$DATABASE_URL" ]; then
    echo "❌ 错误: DATABASE_URL 环境变量未设置"
    echo "请设置数据库连接字符串，例如:"
    echo "export DATABASE_URL='host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable'"
    exit 1
fi

echo "✓ 数据库连接: $DATABASE_URL"
echo ""

# 检查 Google API Key（可选）
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "⚠️  警告: GOOGLE_API_KEY 未设置"
    echo "某些测试将被跳过"
    echo ""
else
    echo "✓ Google API Key: ${GOOGLE_API_KEY:0:10}..."
    echo ""
fi

# 运行测试
echo "========================================="
echo "运行全面错误处理测试..."
echo "========================================="
echo ""

cd "$(dirname "$0")/.."

go test -v \
    -timeout 10m \
    -run TestComprehensiveErrorHandling \
    ./test/e2e/

TEST_EXIT_CODE=$?

echo ""
echo "========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ 全面错误处理测试通过"
else
    echo "❌ 全面错误处理测试失败"
fi
echo "========================================="

exit $TEST_EXIT_CODE
