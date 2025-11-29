#!/bin/bash

# Azure OpenAI 端到端测试脚本
# 用法: ./test_azure_e2e.sh

set -e

echo "========================================="
echo "Azure OpenAI 端到端测试"
echo "========================================="
echo ""

# 检查必需的环境变量
if [ -z "$AZURE_OPENAI_API_KEY" ]; then
    echo "❌ 错误: 缺少环境变量 AZURE_OPENAI_API_KEY"
    echo "请设置 Azure OpenAI API Key:"
    echo "  export AZURE_OPENAI_API_KEY='your-api-key'"
    exit 1
fi

if [ -z "$AZURE_OPENAI_ENDPOINT" ]; then
    echo "❌ 错误: 缺少环境变量 AZURE_OPENAI_ENDPOINT"
    echo "请设置 Azure OpenAI Endpoint:"
    echo "  export AZURE_OPENAI_ENDPOINT='https://your-resource.openai.azure.com'"
    exit 1
fi

if [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "❌ 错误: 缺少环境变量 AZURE_OPENAI_DEPLOYMENT"
    echo "请设置 Azure OpenAI Deployment:"
    echo "  export AZURE_OPENAI_DEPLOYMENT='your-deployment-name'"
    exit 1
fi

# 设置默认的 API Version
if [ -z "$AZURE_OPENAI_API_VERSION" ]; then
    export AZURE_OPENAI_API_VERSION="2024-02-15-preview"
    echo "使用默认 API Version: $AZURE_OPENAI_API_VERSION"
fi

# 显示配置信息（隐藏敏感信息）
echo "配置信息:"
echo "  Endpoint: $AZURE_OPENAI_ENDPOINT"
echo "  Deployment: $AZURE_OPENAI_DEPLOYMENT"
echo "  API Version: $AZURE_OPENAI_API_VERSION"
echo "  API Key: ${AZURE_OPENAI_API_KEY:0:8}****"
echo ""

# 检查数据库配置
if [ -z "$DB_HOST" ]; then
    export DB_HOST="localhost"
fi

if [ -z "$DB_PORT" ]; then
    export DB_PORT="5432"
fi

if [ -z "$DB_USER" ]; then
    export DB_USER="postgres"
fi

if [ -z "$DB_PASSWORD" ]; then
    export DB_PASSWORD="postgres"
fi

if [ -z "$DB_NAME" ]; then
    export DB_NAME="genkit_test"
fi

echo "数据库配置:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# 运行端到端测试
echo "开始运行端到端测试..."
echo ""

cd "$(dirname "$0")/.."

go test -v -timeout 5m ./test/e2e -run TestAzureOpenAI_E2E_Complete

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
