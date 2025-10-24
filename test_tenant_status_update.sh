#!/bin/bash

# 测试租户状态更新功能
# 验证 status=false 是否能正确更新

set -e

# 配置
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@platform.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123456}"

echo "=========================================="
echo "测试租户状态更新功能"
echo "=========================================="
echo ""

# 1. 平台管理员登录
echo "1. 平台管理员登录..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${ADMIN_EMAIL}\",
    \"password\": \"${ADMIN_PASSWORD}\"
  }")

echo "登录响应: ${LOGIN_RESPONSE}"

ACCESS_TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)

if [ -z "${ACCESS_TOKEN}" ]; then
  echo "❌ 登录失败，无法获取访问令牌"
  exit 1
fi

echo "✅ 登录成功"
echo ""

# 2. 创建测试租户
echo "2. 创建测试租户..."
TENANT_NAME="测试租户-$(date +%s)"
TENANT_DOMAIN="test-$(date +%s).example.com"

CREATE_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\"
  }")

echo "创建响应: ${CREATE_RESPONSE}"

TENANT_ID=$(echo "${CREATE_RESPONSE}" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -z "${TENANT_ID}" ]; then
  echo "❌ 创建租户失败"
  exit 1
fi

echo "✅ 租户创建成功，ID: ${TENANT_ID}"
echo ""

# 3. 查询租户初始状态
echo "3. 查询租户初始状态..."
GET_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants/${TENANT_ID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "查询响应: ${GET_RESPONSE}"

INITIAL_STATUS=$(echo "${GET_RESPONSE}" | grep -o '"status":[^,}]*' | cut -d':' -f2)
echo "初始状态: ${INITIAL_STATUS}"
echo ""

# 4. 使用 PUT 接口更新租户状态为 false
echo "4. 使用 PUT 接口更新租户状态为 false..."
UPDATE_RESPONSE=$(curl -s -X PUT "${API_BASE_URL}/tenants/${TENANT_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"status\": false
  }")

echo "更新响应: ${UPDATE_RESPONSE}"
echo ""

# 5. 再次查询租户状态，验证是否更新成功
echo "5. 验证状态是否更新为 false..."
VERIFY_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants/${TENANT_ID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "验证响应: ${VERIFY_RESPONSE}"

UPDATED_STATUS=$(echo "${VERIFY_RESPONSE}" | grep -o '"status":[^,}]*' | cut -d':' -f2)
echo "更新后状态: ${UPDATED_STATUS}"

if [ "${UPDATED_STATUS}" = "false" ]; then
  echo "✅ 状态更新成功！status 已变为 false"
else
  echo "❌ 状态更新失败！status 仍为 ${UPDATED_STATUS}"
  exit 1
fi
echo ""

# 6. 使用 PATCH 接口更新租户状态为 true
echo "6. 使用 PATCH 接口更新租户状态为 true..."
PATCH_RESPONSE=$(curl -s -X PATCH "${API_BASE_URL}/tenants/${TENANT_ID}/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"status\": true
  }")

echo "PATCH 响应: ${PATCH_RESPONSE}"
echo ""

# 7. 再次验证状态
echo "7. 验证状态是否更新为 true..."
FINAL_RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants/${TENANT_ID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "最终响应: ${FINAL_RESPONSE}"

FINAL_STATUS=$(echo "${FINAL_RESPONSE}" | grep -o '"status":[^,}]*' | cut -d':' -f2)
echo "最终状态: ${FINAL_STATUS}"

if [ "${FINAL_STATUS}" = "true" ]; then
  echo "✅ 状态恢复成功！status 已变为 true"
else
  echo "❌ 状态恢复失败！status 仍为 ${FINAL_STATUS}"
  exit 1
fi
echo ""

# 8. 清理：删除测试租户
echo "8. 清理测试数据..."
DELETE_RESPONSE=$(curl -s -X DELETE "${API_BASE_URL}/tenants/${TENANT_ID}" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "删除响应: ${DELETE_RESPONSE}"
echo "✅ 测试租户已删除"
echo ""

echo "=========================================="
echo "✅ 所有测试通过！"
echo "=========================================="
