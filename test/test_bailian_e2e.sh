#!/bin/bash

# 百炼端到端测试脚本
# 用途：运行百炼的完整端到端测试

set -e

echo "=========================================="
echo "百炼端到端测试"
echo "=========================================="
echo ""

# 检查必需的环境变量
if [ -z "$BAILIAN_API_KEY" ]; then
    echo "❌ 错误: 缺少环境变量 BAILIAN_API_KEY"
    echo ""
    echo "请设置以下环境变量："
    echo "  export BAILIAN_API_KEY='your-api-key'"
    echo ""
    echo "可选环境变量："
    echo "  export BAILIAN_ENDPOINT='https://dashscope.aliyuncs.com/compatible-mode/v1'"
    echo "  export BAILIAN_MODEL='qwen-plus'"
    echo ""
    exit 1
fi

# 显示配置信息
echo "配置信息："
echo "  BAILIAN_API_KEY: ${BAILIAN_API_KEY:0:10}..."
echo "  BAILIAN_ENDPOINT: ${BAILIAN_ENDPOINT:-https://dashscope.aliyuncs.com/compatible-mode/v1}"
echo "  BAILIAN_MODEL: ${BAILIAN_MODEL:-qwen-plus}"
echo ""

# 显示数据库配置
echo "数据库配置："
echo "  DB_HOST: ${DB_HOST:-localhost}"
echo "  DB_PORT: ${DB_PORT:-5432}"
echo "  DB_USER: ${DB_USER:-postgres}"
echo "  DB_NAME: ${DB_NAME:-genkit_test}"
echo ""

# 运行测试
echo "开始运行端到端测试..."
echo ""

cd "$(dirname "$0")/.."

go test -v -timeout 5m ./test/e2e -run TestBailian_E2E_Complete

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
