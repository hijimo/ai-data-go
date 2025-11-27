#!/bin/bash

# Azure OpenAI 流式调用测试脚本
# 用于测试 Azure OpenAI 的流式调用功能

set -e

echo "=========================================="
echo "Azure OpenAI 流式调用集成测试"
echo "=========================================="
echo ""

# 检查环境变量
if [ -z "$AZURE_OPENAI_API_KEY" ]; then
    echo "错误: 未设置 AZURE_OPENAI_API_KEY 环境变量"
    echo "请设置以下环境变量："
    echo "  export AZURE_OPENAI_API_KEY=your-api-key"
    echo "  export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com"
    echo "  export AZURE_OPENAI_DEPLOYMENT=your-deployment-name"
    echo "  export AZURE_OPENAI_API_VERSION=2024-02-15-preview  # 可选"
    exit 1
fi

if [ -z "$AZURE_OPENAI_ENDPOINT" ]; then
    echo "错误: 未设置 AZURE_OPENAI_ENDPOINT 环境变量"
    exit 1
fi

if [ -z "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "错误: 未设置 AZURE_OPENAI_DEPLOYMENT 环境变量"
    exit 1
fi

# 设置默认的 API Version
if [ -z "$AZURE_OPENAI_API_VERSION" ]; then
    export AZURE_OPENAI_API_VERSION="2024-02-15-preview"
    echo "使用默认 API Version: $AZURE_OPENAI_API_VERSION"
fi

echo "配置信息："
echo "  Endpoint: $AZURE_OPENAI_ENDPOINT"
echo "  Deployment: $AZURE_OPENAI_DEPLOYMENT"
echo "  API Version: $AZURE_OPENAI_API_VERSION"
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

echo "数据库配置："
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# 进入项目根目录
cd "$(dirname "$0")/.."

echo "运行 Azure OpenAI 流式调用测试..."
echo ""

# 运行测试
go test -v \
    -timeout 5m \
    -run TestAzureOpenAIIntegration_Streaming \
    ./internal/genkit/

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
