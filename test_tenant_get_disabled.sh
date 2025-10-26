#!/bin/bash

# 测试获取禁用租户详情的接口

BASE_URL="http://localhost:8080/api/v1"

echo "=========================================="
echo "测试获取禁用租户详情"
echo "=========================================="
echo ""

# 1. 平台管理员登录
echo "1. 平台管理员登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "Admin123456!"
  }')

echo "登录响应: $LOGIN_RESPONSE"
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  exit 1
fi

echo "✅ 登录成功"
echo "Token: ${TOKEN:0:50}..."
echo ""

# 2. 创建测试租户
echo "2. 创建测试租户..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/tenants" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenantName": "测试禁用租户",
    "tenantDomain": "test-disabled-tenant.local",
    "adminEmail": "admin@test-disabled-tenant.local"
  }')

echo "创建响应: $CREATE_RESPONSE"
TENANT_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ -z "$TENANT_ID" ]; then
  echo "❌ 创建租户失败"
  exit 1
fi

echo "✅ 租户创建成功"
echo "租户ID: $TENANT_ID"
echo ""

# 3. 获取租户详情（启用状态）
echo "3. 获取租户详情（启用状态）..."
GET_RESPONSE=$(curl -s -X GET "$BASE_URL/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "获取响应: $GET_RESPONSE"
STATUS=$(echo $GET_RESPONSE | grep -o '"status":[^,}]*' | cut -d':' -f2)

if [ "$STATUS" != "true" ]; then
  echo "❌ 租户状态不正确"
  exit 1
fi

echo "✅ 成功获取启用状态的租户详情"
echo ""

# 4. 禁用租户
echo "4. 禁用租户..."
DISABLE_RESPONSE=$(curl -s -X PATCH "$BASE_URL/tenants/$TENANT_ID/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "status": false
  }')

echo "禁用响应: $DISABLE_RESPONSE"
echo "✅ 租户已禁用"
echo ""

# 5. 获取租户详情（禁用状态）
echo "5. 获取租户详情（禁用状态）..."
GET_DISABLED_RESPONSE=$(curl -s -X GET "$BASE_URL/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "获取响应: $GET_DISABLED_RESPONSE"

# 检查是否成功获取
CODE=$(echo $GET_DISABLED_RESPONSE | grep -o '"code":[^,}]*' | cut -d':' -f2)
STATUS=$(echo $GET_DISABLED_RESPONSE | grep -o '"status":[^,}]*' | cut -d':' -f2)

if [ "$CODE" = "200" ] && [ "$STATUS" = "false" ]; then
  echo "✅ 成功获取禁用状态的租户详情"
  echo "   - 返回码: $CODE"
  echo "   - 租户状态: $STATUS"
else
  echo "❌ 获取禁用租户详情失败"
  echo "   - 返回码: $CODE"
  echo "   - 租户状态: $STATUS"
  exit 1
fi
echo ""

# 6. 清理：删除测试租户
echo "6. 清理测试数据..."
DELETE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "删除响应: $DELETE_RESPONSE"
echo "✅ 测试数据已清理"
echo ""

echo "=========================================="
echo "✅ 所有测试通过！"
echo "=========================================="
