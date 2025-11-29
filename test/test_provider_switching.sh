#!/bin/bash

# 提供商切换端到端测试脚本
# 用于测试在同一租户下切换不同的模型提供商

set -e

echo "=========================================="
echo "提供商切换端到端测试"
echo "=========================================="
echo ""

# 检查环境变量
echo "检查环境变量..."

# Azure OpenAI 配置
if [ -n "$AZURE_OPENAI_API_KEY" ] && [ -n "$AZURE_OPENAI_ENDPOINT" ] && [ -n "$AZURE_OPENAI_DEPLOYMENT" ]; then
    echo "✓ Azure OpenAI 配置已设置"
    HAS_AZURE=true
else
    echo "⚠ Azure OpenAI 配置未完整设置"
    HAS_AZURE=false
fi

# 百炼配置
if [ -n "$BAILIAN_API_KEY" ]; then
    echo "✓ 百炼配置已设置"
    HAS_BAILIAN=true
else
    echo "⚠ 百炼配置未设置"
    HAS_BAILIAN=false
fi

# 检查是否至少有两个提供商配置
if [ "$HAS_AZURE" = false ] && [ "$HAS_BAILIAN" = false ]; then
    echo ""
    echo "❌ 错误：至少需要两个提供商的配置才能测试切换"
    echo ""
    echo "请设置以下环境变量："
    echo ""
    echo "Azure OpenAI:"
    echo "  export AZURE_OPENAI_API_KEY=\"your-api-key\""
    echo "  export AZURE_OPENAI_ENDPOINT=\"https://your-resource.openai.azure.com\""
    echo "  export AZURE_OPENAI_DEPLOYMENT=\"your-deployment-name\""
    echo "  export AZURE_OPENAI_API_VERSION=\"2024-02-15-preview\"  # 可选"
    echo ""
    echo "百炼:"
    echo "  export BAILIAN_API_KEY=\"your-api-key\""
    echo "  export BAILIAN_ENDPOINT=\"https://dashscope.aliyuncs.com/compatible-mode/v1\"  # 可选"
    echo "  export BAILIAN_MODEL=\"qwen-plus\"  # 可选"
    echo ""
    exit 1
fi

if [ "$HAS_AZURE" = false ] || [ "$HAS_BAILIAN" = false ]; then
    echo ""
    echo "⚠ 警告：只配置了一个提供商，无法完整测试切换功能"
    echo "建议配置至少两个提供商以获得完整的测试覆盖"
    echo ""
fi

# 数据库配置（可选）
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
echo "数据库配置:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# 运行测试
echo "=========================================="
echo "开始运行提供商切换测试..."
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

go test -v -timeout 10m ./test/e2e -run TestProviderSwitching

TEST_EXIT_CODE=$?

echo ""
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✓ 提供商切换测试通过"
else
    echo "✗ 提供商切换测试失败"
fi
echo "=========================================="

exit $TEST_EXIT_CODE
