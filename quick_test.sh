#!/bin/bash

# 快速测试脚本 - 帮助您快速测试租户管理API

BASE_URL="http://localhost:8080"
API_BASE="${BASE_URL}/api/v1"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}=========================================="
echo "租户管理API快速测试"
echo -e "==========================================${NC}"
echo ""

# 检查服务是否运行
echo -e "${YELLOW}检查服务状态...${NC}"
if ! curl -s -f "${BASE_URL}/api/v1/health" > /dev/null 2>&1; then
  echo -e "${RED}❌ 服务未运行或无法访问${NC}"
  echo ""
  echo "请确保服务已启动："
  echo "  go run cmd/server/main.go"
  echo ""
  exit 1
fi
echo -e "${GREEN}✓ 服务正常运行${NC}"
echo ""

# 提示用户输入管理员密码
echo -e "${YELLOW}请输入平台管理员密码：${NC}"
echo "（可以从服务启动日志中找到，搜索 '管理员初始密码'）"
echo ""
read -s -p "密码: " ADMIN_PASSWORD
echo ""
echo ""

if [ -z "$ADMIN_PASSWORD" ]; then
  echo -e "${RED}❌ 密码不能为空${NC}"
  exit 1
fi

# 测试登录
echo -e "${YELLOW}测试登录...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"platform-admin@system.local\",
    \"password\": \"${ADMIN_PASSWORD}\"
  }")

# 检查登录是否成功
LOGIN_CODE=$(echo "$LOGIN_RESPONSE" | jq -r '.code // empty')
if [ "$LOGIN_CODE" != "200" ]; then
  echo -e "${RED}❌ 登录失败${NC}"
  echo ""
  echo "响应："
  echo "$LOGIN_RESPONSE" | jq '.'
  echo ""
  echo "可能的原因："
  echo "1. 密码不正确"
  echo "2. 管理员账户尚未创建"
  echo "3. 数据库连接问题"
  echo ""
  echo "解决方案："
  echo "1. 检查服务启动日志，确认管理员账户已创建"
  echo "2. 从日志中获取正确的初始密码"
  echo "3. 确保数据库服务正常运行"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功${NC}"
echo ""

# 提取访问令牌
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.accessToken // empty')
USER_EMAIL=$(echo "$LOGIN_RESPONSE" | jq -r '.data.user.email // empty')
USER_ROLES=$(echo "$LOGIN_RESPONSE" | jq -r '.data.user.roles | join(", ") // empty')

echo "用户信息："
echo "  邮箱: $USER_EMAIL"
echo "  角色: $USER_ROLES"
echo "  令牌: ${ACCESS_TOKEN:0:50}..."
echo ""

# 测试获取租户列表
echo -e "${YELLOW}测试获取租户列表...${NC}"
TENANT_LIST_RESPONSE=$(curl -s -X GET "${API_BASE}/tenants?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

TENANT_LIST_CODE=$(echo "$TENANT_LIST_RESPONSE" | jq -r '.code // empty')
if [ "$TENANT_LIST_CODE" != "200" ]; then
  echo -e "${RED}❌ 获取租户列表失败${NC}"
  echo ""
  echo "响应："
  echo "$TENANT_LIST_RESPONSE" | jq '.'
  exit 1
fi

echo -e "${GREEN}✓ 获取租户列表成功${NC}"
echo ""

# 显示租户列表
TENANT_COUNT=$(echo "$TENANT_LIST_RESPONSE" | jq -r '.data.totalCount // 0')
echo "租户总数: $TENANT_COUNT"
echo ""

if [ "$TENANT_COUNT" -gt 0 ]; then
  echo "租户列表："
  echo "$TENANT_LIST_RESPONSE" | jq -r '.data.data[] | "  - \(.name) (\(.domain)) - 状态: \(.status)"'
  echo ""
fi

# 询问是否创建测试租户
echo -e "${YELLOW}是否创建一个测试租户？ (y/n)${NC}"
read -p "> " CREATE_TENANT

if [ "$CREATE_TENANT" = "y" ] || [ "$CREATE_TENANT" = "Y" ]; then
  echo ""
  echo -e "${YELLOW}创建测试租户...${NC}"
  
  TENANT_NAME="测试租户-$(date +%s)"
  TENANT_DOMAIN="test-$(date +%s)"
  
  CREATE_RESPONSE=$(curl -s -X POST "${API_BASE}/tenants" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${TENANT_NAME}\",
      \"domain\": \"${TENANT_DOMAIN}\",
      \"metadata\": {
        \"description\": \"这是一个测试租户\",
        \"createdBy\": \"快速测试脚本\"
      }
    }")
  
  CREATE_CODE=$(echo "$CREATE_RESPONSE" | jq -r '.code // empty')
  if [ "$CREATE_CODE" != "201" ]; then
    echo -e "${RED}❌ 创建租户失败${NC}"
    echo ""
    echo "响应："
    echo "$CREATE_RESPONSE" | jq '.'
  else
    echo -e "${GREEN}✓ 创建租户成功${NC}"
    echo ""
    
    NEW_TENANT_ID=$(echo "$CREATE_RESPONSE" | jq -r '.data.tenant.id // empty')
    TENANT_ADMIN_EMAIL=$(echo "$CREATE_RESPONSE" | jq -r '.data.admin.email // empty')
    TENANT_ADMIN_PASSWORD=$(echo "$CREATE_RESPONSE" | jq -r '.data.adminPassword // empty')
    
    echo "新租户信息："
    echo "  租户ID: $NEW_TENANT_ID"
    echo "  租户名称: $TENANT_NAME"
    echo "  租户域名: $TENANT_DOMAIN"
    echo ""
    echo "租户管理员信息："
    echo "  邮箱: $TENANT_ADMIN_EMAIL"
    echo "  初始密码: $TENANT_ADMIN_PASSWORD"
    echo ""
    echo -e "${YELLOW}⚠ 请妥善保管租户管理员密码！${NC}"
  fi
fi

echo ""
echo -e "${BLUE}=========================================="
echo "测试完成"
echo -e "==========================================${NC}"
echo ""
echo "下一步："
echo "1. 访问 Swagger UI: ${BASE_URL}/swagger/index.html"
echo "2. 运行完整测试: ./test_tenant_api.sh"
echo "3. 查看使用文档: cat TENANT_API_USAGE.md"
echo ""
