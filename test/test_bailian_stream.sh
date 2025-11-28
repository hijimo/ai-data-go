#!/bin/bash

# 百炼流式调用集成测试脚本
# 用于测试阿里云百炼的流式调用功能

set -e

echo "========================================="
echo "百炼流式调用集成测试"
echo "========================================="
echo ""

# 检查必需的环境变量
if [ -z "$BAILIAN_API_KEY" ]; then
    echo "错误: 缺少环境变量 BAILIAN_API_KEY"
    echo "请设置百炼 API 密钥："
    echo "  export BAILIAN_API_KEY='your-api-key'"
    exit 1
fi

# 设置默认值
if [ -z "$BAILIAN_ENDPOINT" ]; then
    export BAILIAN_ENDPOINT="https://dashscope.aliyuncs.com/compatible-mode/v1"
    echo "使用默认端点: $BAILIAN_ENDPOINT"
fi

if [ -z "$BAILIAN_MODEL" ]; then
    export BAILIAN_MODEL="qwen-plus"
    echo "使用默认模型: $BAILIAN_MODEL"
fi

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

echo ""
echo "配置信息："
echo "  API 端点: $BAILIAN_ENDPOINT"
echo "  模型名称: $BAILIAN_MODEL"
echo "  数据库: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# 运行测试
echo "开始运行百炼流式调用集成测试..."
echo ""

cd "$(dirname "$0")/.."

go test -v -run TestBailianIntegration_Streaming ./internal/genkit/

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
