#!/bin/bash

# Azure OpenAI 集成测试脚本
# 用于测试 Azure OpenAI 非流式调用功能

set -e

echo "========================================="
echo "Azure OpenAI 集成测试"
echo "========================================="

# 检查必需的环境变量
if [ -z "$AZURE_OPENAI_API_KEY" ]; then
    echo "错误: 缺少环境变量 AZURE_OPENAI_API_KEY"
    echo "请设置 Azure OpenAI API 密钥："
    echo "  export AZURE_OPENAI_API_KEY='your-api-key'"
    exit 1
fi

if [ -z "$AZURE_OPENAI_ENDPOINT" ]; then
    echo "错误: 缺少环境变量 AZURE_OPENAI_ENDPOINT"
    echo "请设置 Azure OpenAI Endpoint："
    echo "  export AZURE_OPENAI_ENDPOINT='https://your-resource.openai.azure.com'"
    exit 1
fi

if [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "错误: 缺少环境变量 AZURE_OPENAI_DEPLOYMENT"
    echo "请设置 Azure OpenAI Deployment 名称："
    echo "  export AZURE_OPENAI_DEPLOYMENT='your-deployment-name'"
    exit 1
fi

# 设置默认的 API Version（如果未设置）
if [ -z "$AZURE_OPENAI_API_VERSION" ]; then
    export AZURE_OPENAI_API_VERSION="2024-02-15-preview"
    echo "使用默认 API Version: $AZURE_OPENAI_API_VERSION"
fi

# 设置数据库连接信息（如果未设置，使用默认值）
export DB_HOST=${DB_HOST:-localhost}
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-postgres}
export DB_PASSWORD=${DB_PASSWORD:-postgres}
export DB_NAME=${DB_NAME:-genkit_test}

echo ""
echo "配置信息："
echo "  Azure Endpoint: $AZURE_OPENAI_ENDPOINT"
echo "  Azure Deployment: $AZURE_OPENAI_DEPLOYMENT"
echo "  Azure API Version: $AZURE_OPENAI_API_VERSION"
echo "  Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# 进入项目根目录
cd "$(dirname "$0")/.."

# 运行集成测试
echo "运行 Azure OpenAI 集成测试..."
echo ""

go test -v -run TestAzureOpenAIIntegration_NonStreaming ./internal/genkit/

# 检查测试结果
if [ $? -eq 0 ]; then
    echo ""
    echo "========================================="
    echo "✅ Azure OpenAI 集成测试通过"
    echo "========================================="
else
    echo ""
    echo "========================================="
    echo "❌ Azure OpenAI 集成测试失败"
    echo "========================================="
    exit 1
fi
