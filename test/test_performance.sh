#!/bin/bash

# 性能测试脚本
# 用于测试单提供商调用延迟

set -e

echo "=========================================="
echo "性能测试 - 单提供商调用延迟"
echo "=========================================="
echo ""

# 检查环境变量
echo "检查环境变量..."

if [ -z "$GOOGLE_API_KEY" ]; then
    echo "⚠️  警告: 未设置 GOOGLE_API_KEY，将跳过 Google AI 测试"
fi

if [ -z "$AZURE_OPENAI_API_KEY" ] || [ -z "$AZURE_OPENAI_ENDPOINT" ] || [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "⚠️  警告: 未设置 Azure OpenAI 环境变量，将跳过 Azure OpenAI 测试"
fi

if [ -z "$BAILIAN_API_KEY" ]; then
    echo "⚠️  警告: 未设置 BAILIAN_API_KEY，将跳过百炼测试"
fi

echo ""
echo "开始运行性能测试..."
echo ""

# 运行性能测试
cd "$(dirname "$0")/.."
go test -v -timeout 10m ./test/e2e -run TestSingleProviderLatency

echo ""
echo "=========================================="
echo "性能测试完成"
echo "=========================================="
