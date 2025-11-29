#!/bin/bash

# 错误场景测试脚本
# 用于测试 Genkit 多模型支持的各种错误场景

set -e

echo "=========================================="
echo "错误场景测试"
echo "=========================================="
echo ""

# 检查环境变量
echo "检查环境变量..."

# 至少需要一个有效的 API 密钥来测试某些场景
if [ -z "$GOOGLE_API_KEY" ] && [ -z "$AZURE_OPENAI_API_KEY" ] && [ -z "$BAILIAN_API_KEY" ]; then
    echo "警告：未设置任何 API 密钥"
    echo "某些测试将被跳过"
    echo ""
fi

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

# 运行错误场景测试
echo "=========================================="
echo "运行错误场景测试..."
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

# 运行测试
go test -v ./test/e2e -run TestErrorScenarios -timeout 10m

echo ""
echo "=========================================="
echo "错误场景测试完成"
echo "=========================================="
echo ""
echo "测试覆盖的错误场景："
echo "  ✓ 配置相关错误（配置不存在、已禁用、已删除、JSON格式错误）"
echo "  ✓ 租户相关错误（租户ID无效、不存在）"
echo "  ✓ API密钥相关错误（密钥为空、无效）"
echo "  ✓ 提供商相关错误（不支持的提供商类型）"
echo "  ✓ 参数相关错误（Temperature、MaxTokens超出范围）"
echo "  ✓ 输入相关错误（空提示词、超长提示词）"
echo "  ✓ 上下文相关错误（上下文取消、超时）"
echo "  ✓ 流式调用错误（配置不存在、租户ID无效、上下文取消）"
echo "  ✓ 并发错误场景（并发调用不存在/禁用的模型）"
echo "  ✓ 边界条件（特殊字符、超长名称）"
echo ""
