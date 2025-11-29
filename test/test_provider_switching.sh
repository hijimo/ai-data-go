#!/bin/bash

# 提供商切换延迟测试脚本
# 用于测试在同一租户下切换不同提供商时的性能开销

set -e

echo "=========================================="
echo "提供商切换延迟测试"
echo "=========================================="
echo ""

# 检查环境变量
echo "检查环境变量..."

MISSING_VARS=0
AVAILABLE_PROVIDERS=0

if [ -z "$GOOGLE_API_KEY" ]; then
    echo "⚠️  GOOGLE_API_KEY 未设置"
    MISSING_VARS=$((MISSING_VARS + 1))
else
    echo "✓ GOOGLE_API_KEY 已设置"
    AVAILABLE_PROVIDERS=$((AVAILABLE_PROVIDERS + 1))
fi

if [ -z "$AZURE_OPENAI_API_KEY" ] || [ -z "$AZURE_OPENAI_ENDPOINT" ] || [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "⚠️  Azure OpenAI 环境变量未完整设置"
    MISSING_VARS=$((MISSING_VARS + 1))
else
    echo "✓ Azure OpenAI 环境变量已设置"
    AVAILABLE_PROVIDERS=$((AVAILABLE_PROVIDERS + 1))
fi

if [ -z "$BAILIAN_API_KEY" ]; then
    echo "⚠️  BAILIAN_API_KEY 未设置"
    MISSING_VARS=$((MISSING_VARS + 1))
else
    echo "✓ BAILIAN_API_KEY 已设置"
    AVAILABLE_PROVIDERS=$((AVAILABLE_PROVIDERS + 1))
fi

echo ""

if [ $AVAILABLE_PROVIDERS -lt 2 ]; then
    echo "❌ 错误：至少需要配置两个提供商才能测试切换延迟"
    echo "   当前可用提供商数量: $AVAILABLE_PROVIDERS"
    echo ""
    echo "请设置以下环境变量（至少两组）："
    echo ""
    echo "Google AI:"
    echo "  export GOOGLE_API_KEY=your_google_api_key"
    echo ""
    echo "Azure OpenAI:"
    echo "  export AZURE_OPENAI_API_KEY=your_azure_api_key"
    echo "  export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com"
    echo "  export AZURE_OPENAI_DEPLOYMENT=your_deployment_name"
    echo ""
    echo "阿里云百炼:"
    echo "  export BAILIAN_API_KEY=your_bailian_api_key"
    echo "  export BAILIAN_ENDPOINT=https://dashscope.aliyuncs.com/compatible-mode/v1  # 可选"
    echo ""
    exit 1
fi

echo "✓ 环境变量检查通过（可用提供商: $AVAILABLE_PROVIDERS）"
echo ""

# 检查数据库连接
echo "检查数据库连接..."
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  DATABASE_URL 未设置，使用默认值"
    export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/genkit_test?sslmode=disable"
fi
echo "✓ 数据库 URL: $DATABASE_URL"
echo ""

# 运行测试
echo "=========================================="
echo "运行提供商切换延迟测试..."
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

go test -v -run TestProviderSwitchingLatency ./test/e2e/ -timeout 10m

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
