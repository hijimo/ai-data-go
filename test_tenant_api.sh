#!/bin/bash

# 租户管理API测试脚本
# 演示完整的认证和租户管理流程

BASE_URL="http://localhost:8080"
API_BASE="${BASE_URL}/api/v1"

echo "=========================================="
echo "租户管理API测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 登录获取访问令牌
echo -e "${YELLOW}步骤 1: 使用平台管理员账户登录${NC}"
echo "POST ${API_BASE}/auth/login"
echo ""

# 使用系统初始化时创建的平台管理员账户
# 默认邮箱: platform-admin@system.local
# 默认密码: 从启动日志中获取，或使用环境变量
ADMIN_EMAIL="${ADMIN_EMAIL:-platform-admin@system.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin@123456}"

LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${ADMIN_EMAIL}\",
    \"password\": \"${ADMIN_PASSWORD}\"
  }")

echo "响应:"
echo "$LOGIN_RESPONSE" | jq '.'
echo ""

# 提取访问令牌
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.accessToken // empty')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
  echo -e "${RED}❌ 登录失败，无法获取访问令牌${NC}"
  echo ""
  echo "可能的原因："
  echo "1. 管理员账户尚未创建（请检查服务启动日志）"
  echo "2. 密码不正确（请从启动日志中获取初始密码）"
  echo "3. 服务未正常启动"
  echo ""
  echo "解决方案："
  echo "1. 检查服务启动日志，找到管理员初始密码"
  echo "2. 使用环境变量设置密码: export ADMIN_PASSWORD='your-password'"
  echo "3. 重新运行此脚本"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功，已获取访问令牌${NC}"
echo "访问令牌: ${ACCESS_TOKEN:0:50}..."
echo ""

# 2. 获取租户列表
echo -e "${YELLOW}步骤 2: 获取租户列表${NC}"
echo "GET ${API_BASE}/tenants?pageNo=1&pageSize=10"
echo ""

TENANT_LIST_RESPONSE=$(curl -s -X GET "${API_BASE}/tenants?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

echo "响应:"
echo "$TENANT_LIST_RESPONSE" | jq '.'
echo ""

# 检查响应状态
TENANT_LIST_CODE=$(echo "$TENANT_LIST_RESPONSE" | jq -r '.code // empty')
if [ "$TENANT_LIST_CODE" = "200" ]; then
  echo -e "${GREEN}✓ 获取租户列表成功${NC}"
else
  echo -e "${RED}❌ 获取租户列表失败${NC}"
fi
echo ""

# 3. 创建新租户
echo -e "${YELLOW}步骤 3: 创建新租户${NC}"
echo "POST ${API_BASE}/tenants"
echo ""

TENANT_NAME="测试租户-$(date +%s)"
TENANT_DOMAIN="test-tenant-$(date +%s)"

CREATE_TENANT_RESPONSE=$(curl -s -X POST "${API_BASE}/tenants" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"${TENANT_NAME}\",
    \"domain\": \"${TENANT_DOMAIN}\",
    \"metadata\": {
      \"description\": \"这是一个测试租户\",
      \"createdBy\": \"API测试脚本\"
    }
  }")

echo "响应:"
echo "$CREATE_TENANT_RESPONSE" | jq '.'
echo ""

# 提取新创建的租户ID
NEW_TENANT_ID=$(echo "$CREATE_TENANT_RESPONSE" | jq -r '.data.tenant.id // empty')
TENANT_ADMIN_EMAIL=$(echo "$CREATE_TENANT_RESPONSE" | jq -r '.data.admin.email // empty')
TENANT_ADMIN_PASSWORD=$(echo "$CREATE_TENANT_RESPONSE" | jq -r '.data.adminPassword // empty')

if [ -n "$NEW_TENANT_ID" ] && [ "$NEW_TENANT_ID" != "null" ]; then
  echo -e "${GREEN}✓ 创建租户成功${NC}"
  echo "租户ID: $NEW_TENANT_ID"
  echo "租户管理员邮箱: $TENANT_ADMIN_EMAIL"
  echo "租户管理员初始密码: $TENANT_ADMIN_PASSWORD"
else
  echo -e "${RED}❌ 创建租户失败${NC}"
  NEW_TENANT_ID=""
fi
echo ""

# 4. 获取租户详情
if [ -n "$NEW_TENANT_ID" ]; then
  echo -e "${YELLOW}步骤 4: 获取租户详情${NC}"
  echo "GET ${API_BASE}/tenants/${NEW_TENANT_ID}"
  echo ""

  TENANT_DETAIL_RESPONSE=$(curl -s -X GET "${API_BASE}/tenants/${NEW_TENANT_ID}" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}")

  echo "响应:"
  echo "$TENANT_DETAIL_RESPONSE" | jq '.'
  echo ""

  TENANT_DETAIL_CODE=$(echo "$TENANT_DETAIL_RESPONSE" | jq -r '.code // empty')
  if [ "$TENANT_DETAIL_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 获取租户详情成功${NC}"
  else
    echo -e "${RED}❌ 获取租户详情失败${NC}"
  fi
  echo ""
fi

# 5. 更新租户信息
if [ -n "$NEW_TENANT_ID" ]; then
  echo -e "${YELLOW}步骤 5: 更新租户信息${NC}"
  echo "PUT ${API_BASE}/tenants/${NEW_TENANT_ID}"
  echo ""

  UPDATE_TENANT_RESPONSE=$(curl -s -X PUT "${API_BASE}/tenants/${NEW_TENANT_ID}" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${TENANT_NAME} (已更新)\",
      \"metadata\": {
        \"description\": \"这是一个已更新的测试租户\",
        \"updatedBy\": \"API测试脚本\"
      }
    }")

  echo "响应:"
  echo "$UPDATE_TENANT_RESPONSE" | jq '.'
  echo ""

  UPDATE_TENANT_CODE=$(echo "$UPDATE_TENANT_RESPONSE" | jq -r '.code // empty')
  if [ "$UPDATE_TENANT_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 更新租户信息成功${NC}"
  else
    echo -e "${RED}❌ 更新租户信息失败${NC}"
  fi
  echo ""
fi

# 6. 测试租户管理员登录
if [ -n "$TENANT_ADMIN_EMAIL" ] && [ -n "$TENANT_ADMIN_PASSWORD" ]; then
  echo -e "${YELLOW}步骤 6: 使用租户管理员账户登录${NC}"
  echo "POST ${API_BASE}/auth/login"
  echo ""

  TENANT_ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE}/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${TENANT_ADMIN_EMAIL}\",
      \"password\": \"${TENANT_ADMIN_PASSWORD}\"
    }")

  echo "响应:"
  echo "$TENANT_ADMIN_LOGIN_RESPONSE" | jq '.'
  echo ""

  TENANT_ADMIN_TOKEN=$(echo "$TENANT_ADMIN_LOGIN_RESPONSE" | jq -r '.data.accessToken // empty')

  if [ -n "$TENANT_ADMIN_TOKEN" ] && [ "$TENANT_ADMIN_TOKEN" != "null" ]; then
    echo -e "${GREEN}✓ 租户管理员登录成功${NC}"
    echo ""

    # 7. 租户管理员尝试获取租户列表（应该只能看到自己的租户）
    echo -e "${YELLOW}步骤 7: 租户管理员获取租户列表（权限测试）${NC}"
    echo "GET ${API_BASE}/tenants?pageNo=1&pageSize=10"
    echo ""

    TENANT_ADMIN_LIST_RESPONSE=$(curl -s -X GET "${API_BASE}/tenants?pageNo=1&pageSize=10" \
      -H "Authorization: Bearer ${TENANT_ADMIN_TOKEN}")

    echo "响应:"
    echo "$TENANT_ADMIN_LIST_RESPONSE" | jq '.'
    echo ""

    TENANT_COUNT=$(echo "$TENANT_ADMIN_LIST_RESPONSE" | jq -r '.data.data | length // 0')
    if [ "$TENANT_COUNT" = "1" ]; then
      echo -e "${GREEN}✓ 权限验证成功：租户管理员只能看到自己的租户${NC}"
    else
      echo -e "${YELLOW}⚠ 租户管理员看到了 $TENANT_COUNT 个租户${NC}"
    fi
  else
    echo -e "${RED}❌ 租户管理员登录失败${NC}"
  fi
  echo ""
fi

# 8. 禁用租户（仅平台管理员）
if [ -n "$NEW_TENANT_ID" ]; then
  echo -e "${YELLOW}步骤 8: 禁用租户（平台管理员操作）${NC}"
  echo "PATCH ${API_BASE}/tenants/${NEW_TENANT_ID}/status"
  echo ""

  DISABLE_TENANT_RESPONSE=$(curl -s -X PATCH "${API_BASE}/tenants/${NEW_TENANT_ID}/status" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"status\": false
    }")

  echo "响应:"
  echo "$DISABLE_TENANT_RESPONSE" | jq '.'
  echo ""

  DISABLE_TENANT_CODE=$(echo "$DISABLE_TENANT_RESPONSE" | jq -r '.code // empty')
  if [ "$DISABLE_TENANT_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 禁用租户成功${NC}"
  else
    echo -e "${RED}❌ 禁用租户失败${NC}"
  fi
  echo ""
fi

# 9. 删除租户（仅平台管理员）
if [ -n "$NEW_TENANT_ID" ]; then
  echo -e "${YELLOW}步骤 9: 删除租户（平台管理员操作）${NC}"
  echo "DELETE ${API_BASE}/tenants/${NEW_TENANT_ID}"
  echo ""

  DELETE_TENANT_RESPONSE=$(curl -s -X DELETE "${API_BASE}/tenants/${NEW_TENANT_ID}" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}")

  echo "响应:"
  echo "$DELETE_TENANT_RESPONSE" | jq '.'
  echo ""

  DELETE_TENANT_CODE=$(echo "$DELETE_TENANT_RESPONSE" | jq -r '.code // empty')
  if [ "$DELETE_TENANT_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 删除租户成功${NC}"
  else
    echo -e "${RED}❌ 删除租户失败${NC}"
  fi
  echo ""
fi

echo "=========================================="
echo "测试完成"
echo "=========================================="
