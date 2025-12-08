#!/bin/bash

# 测试数据初始化脚本
# 用于在测试环境中创建测试租户和模型配置
# 用法: ./scripts/init_test_data.sh

set -e

echo "========================================="
echo "测试数据初始化"
echo "========================================="
echo ""

# 加载环境变量
if [ -f ".env.test" ]; then
    export $(cat .env.test | grep -v '^#' | xargs)
    echo "✓ 已加载 .env.test 配置"
else
    echo "❌ 错误: .env.test 文件不存在"
    echo "请先运行: ./scripts/setup_test_env.sh"
    exit 1
fi

# 检查服务是否运行
echo ""
echo "检查服务状态..."
if ! curl -s http://localhost:${SERVER_PORT:-8080}/health > /dev/null; then
    echo "❌ 错误: 服务未运行"
    echo "请先启动服务: make run-test"
    exit 1
fi
echo "✓ 服务正在运行"

# API 基础 URL
API_BASE="http://localhost:${SERVER_PORT:-8080}/api/v1"

echo ""
echo "========================================="
echo "步骤 1: 登录平台管理员账户"
echo "========================================="

# 登录获取 Token
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$PLATFORM_ADMIN_EMAIL\",
        \"password\": \"$PLATFORM_ADMIN_PASSWORD\"
    }")

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.accessToken')

if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
    echo "❌ 登录失败"
    echo "响应: $LOGIN_RESPONSE"
    exit 1
fi

echo "✓ 平台管理员登录成功"
echo "Token: ${ADMIN_TOKEN:0:20}..."

echo ""
echo "========================================="
echo "步骤 2: 创建测试租户"
echo "========================================="

# 创建测试租户
TENANT_RESPONSE=$(curl -s -X POST "$API_BASE/tenants" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "name": "测试租户",
        "domain": "test-tenant.local",
        "adminEmail": "tenant-admin@test.local",
        "adminPassword": "Test@Tenant123",
        "adminName": "测试租户管理员"
    }')

TEST_TENANT_ID=$(echo $TENANT_RESPONSE | jq -r '.data.tenant.id')
TENANT_ADMIN_EMAIL=$(echo $TENANT_RESPONSE | jq -r '.data.admin.email')

if [ "$TEST_TENANT_ID" == "null" ] || [ -z "$TEST_TENANT_ID" ]; then
    echo "❌ 创建租户失败"
    echo "响应: $TENANT_RESPONSE"
    exit 1
fi

echo "✓ 测试租户创建成功"
echo "租户ID: $TEST_TENANT_ID"
echo "管理员邮箱: $TENANT_ADMIN_EMAIL"

# 保存租户ID到环境变量文件
sed -i.bak "s/TEST_TENANT_ID=.*/TEST_TENANT_ID=$TEST_TENANT_ID/" .env.test
rm -f .env.test.bak

echo ""
echo "========================================="
echo "步骤 3: 登录测试租户管理员"
echo "========================================="

# 登录租户管理员
TENANT_LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TENANT_ADMIN_EMAIL\",
        \"password\": \"Test@Tenant123\"
    }")

TENANT_ADMIN_TOKEN=$(echo $TENANT_LOGIN_RESPONSE | jq -r '.data.accessToken')

if [ "$TENANT_ADMIN_TOKEN" == "null" ] || [ -z "$TENANT_ADMIN_TOKEN" ]; then
    echo "❌ 租户管理员登录失败"
    echo "响应: $TENANT_LOGIN_RESPONSE"
    exit 1
fi

echo "✓ 租户管理员登录成功"
echo "Token: ${TENANT_ADMIN_TOKEN:0:20}..."

# 保存租户管理员Token
sed -i.bak "s/TEST_TENANT_ADMIN_TOKEN=.*/TEST_TENANT_ADMIN_TOKEN=$TENANT_ADMIN_TOKEN/" .env.test
rm -f .env.test.bak

echo ""
echo "========================================="
echo "步骤 4: 配置模型提供商"
echo "========================================="

# 配置 Google AI (如果有 API Key)
if [ -n "$GOOGLE_API_KEY" ] && [ "$GOOGLE_API_KEY" != "your_google_api_key_here" ]; then
    echo ""
    echo "配置 Google AI (Gemini)..."
    
    GOOGLE_CONFIG=$(curl -s -X POST "$API_BASE/model-configurations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TENANT_ADMIN_TOKEN" \
        -d "{
            \"modelName\": \"gemini-pro\",
            \"providerType\": \"google\",
            \"apiKey\": \"$GOOGLE_API_KEY\",
            \"configuration\": {
                \"model\": \"gemini-2.0-flash-exp\",
                \"defaultTemperature\": 0.7,
                \"defaultMaxTokens\": 2048
            }
        }")
    
    if echo $GOOGLE_CONFIG | jq -e '.data.id' > /dev/null; then
        echo "✓ Google AI 配置成功"
    else
        echo "⚠️  Google AI 配置失败: $GOOGLE_CONFIG"
    fi
fi

# 配置 Azure OpenAI (如果有 API Key)
if [ -n "$AZURE_OPENAI_API_KEY" ] && [ "$AZURE_OPENAI_API_KEY" != "your_azure_api_key_here" ]; then
    echo ""
    echo "配置 Azure OpenAI..."
    
    AZURE_CONFIG=$(curl -s -X POST "$API_BASE/model-configurations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TENANT_ADMIN_TOKEN" \
        -d "{
            \"modelName\": \"gpt-4\",
            \"providerType\": \"azure\",
            \"apiKey\": \"$AZURE_OPENAI_API_KEY\",
            \"configuration\": {
                \"model\": \"gpt-4\",
                \"azureEndpoint\": \"$AZURE_OPENAI_ENDPOINT\",
                \"azureDeployment\": \"$AZURE_OPENAI_DEPLOYMENT\",
                \"azureApiVersion\": \"$AZURE_OPENAI_API_VERSION\",
                \"defaultTemperature\": 0.7,
                \"defaultMaxTokens\": 2048
            }
        }")
    
    if echo $AZURE_CONFIG | jq -e '.data.id' > /dev/null; then
        echo "✓ Azure OpenAI 配置成功"
    else
        echo "⚠️  Azure OpenAI 配置失败: $AZURE_CONFIG"
    fi
fi

# 配置百炼 (如果有 API Key)
if [ -n "$BAILIAN_API_KEY" ] && [ "$BAILIAN_API_KEY" != "your_bailian_api_key_here" ]; then
    echo ""
    echo "配置阿里云百炼..."
    
    BAILIAN_CONFIG=$(curl -s -X POST "$API_BASE/model-configurations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TENANT_ADMIN_TOKEN" \
        -d "{
            \"modelName\": \"qwen-turbo\",
            \"providerType\": \"bailian\",
            \"apiKey\": \"$BAILIAN_API_KEY\",
            \"configuration\": {
                \"model\": \"qwen-turbo\",
                \"bailianEndpoint\": \"$BAILIAN_ENDPOINT\",
                \"bailianWorkspace\": \"default\",
                \"defaultTemperature\": 0.7,
                \"defaultMaxTokens\": 2048
            }
        }")
    
    if echo $BAILIAN_CONFIG | jq -e '.data.id' > /dev/null; then
        echo "✓ 百炼配置成功"
    else
        echo "⚠️  百炼配置失败: $BAILIAN_CONFIG"
    fi
fi

echo ""
echo "========================================="
echo "✓ 测试数据初始化完成！"
echo "========================================="
echo ""
echo "测试环境信息："
echo "  租户ID: $TEST_TENANT_ID"
echo "  租户管理员邮箱: $TENANT_ADMIN_EMAIL"
echo "  租户管理员密码: Test@Tenant123"
echo ""
echo "已配置的模型提供商："
[ -n "$GOOGLE_API_KEY" ] && [ "$GOOGLE_API_KEY" != "your_google_api_key_here" ] && echo "  ✓ Google AI (Gemini)"
[ -n "$AZURE_OPENAI_API_KEY" ] && [ "$AZURE_OPENAI_API_KEY" != "your_azure_api_key_here" ] && echo "  ✓ Azure OpenAI"
[ -n "$BAILIAN_API_KEY" ] && [ "$BAILIAN_API_KEY" != "your_bailian_api_key_here" ] && echo "  ✓ 阿里云百炼"
echo ""
echo "下一步："
echo "1. 运行端到端测试: make test-e2e"
echo "2. 运行性能测试: make test-performance"
echo "3. 查看 API 文档: http://localhost:${SERVER_PORT:-8080}/swagger/index.html"
echo ""
