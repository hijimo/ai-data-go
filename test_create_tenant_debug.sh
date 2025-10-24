#!/bin/bash

# 测试创建租户接口 - 调试版本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# API 基础 URL
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"

echo -e "${YELLOW}=== 创建租户接口调试测试 ===${NC}"
echo ""

# 1. 登录获取 token
echo "1. 登录获取访问令牌..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@platform.local",
    "password": "Admin@123456"
  }')

echo "登录响应:"
echo "$LOGIN_RESPONSE" | jq '.'
echo ""

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.accessToken // empty')

if [ -z "$ACCESS_TOKEN" ]; then
  echo -e "${RED}❌ 登录失败，无法获取访问令牌${NC}"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功${NC}"
echo ""

# 2. 测试不同的 metadata 格式

# 测试 1: 不包含 metadata 字段
echo "2. 测试 1: 不包含 metadata 字段..."
TENANT_NAME="test-no-metadata-$(date +%s)"
TENANT_DOMAIN="test-no-metadata-$(date +%s).example.com"

CREATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\"
  }")

echo "响应:"
echo "$CREATE_RESPONSE" | jq '.'
echo ""

# 测试 2: metadata 为空对象
echo "3. 测试 2: metadata 为空对象..."
TENANT_NAME="test-empty-metadata-$(date +%s)"
TENANT_DOMAIN="test-empty-metadata-$(date +%s).example.com"

CREATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\",
    \"metadata\": {}
  }")

echo "响应:"
echo "$CREATE_RESPONSE" | jq '.'
echo ""

# 测试 3: metadata 包含数据
echo "4. 测试 3: metadata 包含数据..."
TENANT_NAME="test-with-metadata-$(date +%s)"
TENANT_DOMAIN="test-with-metadata-$(date +%s).example.com"

CREATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\",
    \"metadata\": {
      \"description\": \"测试租户\",
      \"createdBy\": \"调试脚本\"
    }
  }")

echo "响应:"
echo "$CREATE_RESPONSE" | jq '.'
echo ""

# 测试 4: metadata 为 null
echo "5. 测试 4: metadata 为 null..."
TENANT_NAME="test-null-metadata-$(date +%s)"
TENANT_DOMAIN="test-null-metadata-$(date +%s).example.com"

CREATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\",
    \"metadata\": null
  }")

echo "响应:"
echo "$CREATE_RESPONSE" | jq '.'
echo ""

echo -e "${GREEN}=== 测试完成 ===${NC}"
