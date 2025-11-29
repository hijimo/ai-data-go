#!/bin/bash

# 默认提供商测试脚本
# 用于测试系统的默认提供商逻辑（Google AI Gemini）

set -e

echo "=========================================="
echo "默认提供商测试"
echo "=========================================="
echo ""

# 检查环境变量
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "❌ 错误: 缺少 GOOGLE_API_KEY 环境变量"
    echo ""
    echo "请设置环境变量："
    echo "  export GOOGLE_API_KEY='your-google-api-key'"
    echo ""
    exit 1
fi

echo "✓ 环境变量检查通过"
echo ""

# 运行测试
echo "开始运行默认提供商测试..."
echo ""

cd "$(dirname "$0")/.."

go test -v \
    -timeout 5m \
    -run TestDefaultProvider \
    ./test/e2e/

TEST_EXIT_CODE=$?

echo ""
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✓ 默认提供商测试通过"
else
    echo "❌ 默认提供商测试失败"
fi
echo "=========================================="

exit $TEST_EXIT_CODE
