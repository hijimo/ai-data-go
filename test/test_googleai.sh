#!/bin/bash

# Google AI (Gemini) 集成测试脚本
# 用于测试 Google AI 的非流式和流式调用功能

set -e

echo "========================================="
echo "Google AI (Gemini) 集成测试"
echo "========================================="
echo ""

# 检查必需的环境变量
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "错误: 缺少必需的环境变量 GOOGLE_API_KEY"
    echo "请设置环境变量: export GOOGLE_API_KEY=your_api_key"
    exit 1
fi

# 设置默认模型（如果未设置）
if [ -z "$GOOGLE_MODEL" ]; then
    export GOOGLE_MODEL="gemini-2.0-flash-exp"
    echo "使用默认模型: $GOOGLE_MODEL"
fi

echo "配置信息:"
echo "  模型: $GOOGLE_MODEL"
echo "  API Key: ${GOOGLE_API_KEY:0:10}..."
echo ""

# 设置数据库连接信息（如果未设置）
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
echo "  主机: $DB_HOST"
echo "  端口: $DB_PORT"
echo "  用户: $DB_USER"
echo "  数据库: $DB_NAME"
echo ""

# 运行非流式调用测试
echo "========================================="
echo "1. 运行非流式调用测试"
echo "========================================="
echo ""

go test -v -run TestGoogleAIIntegration_NonStreaming ./internal/genkit/

echo ""
echo "========================================="
echo "2. 运行流式调用测试"
echo "========================================="
echo ""

go test -v -run TestGoogleAIIntegration_Streaming ./internal/genkit/

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
