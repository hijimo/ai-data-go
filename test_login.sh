#!/bin/bash

# 测试登录接口

# 从 .env 读取配置
source .env

# API 地址
API_URL="http://localhost:${SERVER_PORT:-8080}"

echo "测试平台管理员登录"
echo "===================="
echo "邮箱: $PLATFORM_ADMIN_EMAIL"
echo "密码: $PLATFORM_ADMIN_PASSWORD"
echo ""

# 首先获取平台租户ID
echo "1. 获取平台租户信息..."
TENANT_RESPONSE=$(curl -s "${API_URL}/api/v1/tenants?type=system")
echo "租户响应: $TENANT_RESPONSE"
echo ""

# 提取租户ID（假设返回的是JSON格式）
TENANT_ID=$(echo $TENANT_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "平台租户ID: $TENANT_ID"
echo ""

# 测试登录（不带租户ID）
echo "2. 测试登录（不带租户ID）..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${PLATFORM_ADMIN_EMAIL}\",
    \"password\": \"${PLATFORM_ADMIN_PASSWORD}\"
  }")
echo "响应: $LOGIN_RESPONSE"
echo ""

# 测试登录（带租户ID）
if [ ! -z "$TENANT_ID" ]; then
  echo "3. 测试登录（带租户ID）..."
  LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"tenantId\": \"${TENANT_ID}\",
      \"email\": \"${PLATFORM_ADMIN_EMAIL}\",
      \"password\": \"${PLATFORM_ADMIN_PASSWORD}\"
    }")
  echo "响应: $LOGIN_RESPONSE"
  echo ""
fi

# 测试一些常见的错误情况
echo "4. 测试常见错误情况..."

echo "4.1 密码带空格:"
curl -s -X POST "${API_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${PLATFORM_ADMIN_EMAIL}\",
    \"password\": \"${PLATFORM_ADMIN_PASSWORD} \"
  }"
echo ""
echo ""

echo "4.2 邮箱大小写:"
curl -s -X POST "${API_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"Admin@System.Local\",
    \"password\": \"${PLATFORM_ADMIN_PASSWORD}\"
  }"
echo ""
echo ""

echo "4.3 密码大小写:"
curl -s -X POST "${API_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${PLATFORM_ADMIN_EMAIL}\",
    \"password\": \"admin123456\"
  }"
echo ""
