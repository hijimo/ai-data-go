#!/bin/bash

# 用户列表搜索功能测试脚本

BASE_URL="http://localhost:8080/api/v1"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印分隔线
print_separator() {
    echo -e "${YELLOW}========================================${NC}"
}

# 打印测试标题
print_test() {
    echo -e "\n${YELLOW}测试: $1${NC}"
    print_separator
}

# 打印成功信息
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 打印错误信息
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# 1. 平台管理员登录
print_test "1. 平台管理员登录"
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.com",
    "password": "Admin123456"
  }')

ADMIN_TOKEN=$(echo $ADMIN_LOGIN_RESPONSE | jq -r '.data.accessToken')

if [ "$ADMIN_TOKEN" != "null" ] && [ -n "$ADMIN_TOKEN" ]; then
    print_success "平台管理员登录成功"
    echo "Token: ${ADMIN_TOKEN:0:50}..."
else
    print_error "平台管理员登录失败"
    echo "响应: $ADMIN_LOGIN_RESPONSE"
    exit 1
fi

# 2. 获取租户列表（用于后续测试）
print_test "2. 获取租户列表"
TENANT_LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/tenants?pageSize=1" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

TENANT_ID=$(echo $TENANT_LIST_RESPONSE | jq -r '.data.data[0].id')

if [ "$TENANT_ID" != "null" ] && [ -n "$TENANT_ID" ]; then
    print_success "获取租户ID成功: $TENANT_ID"
else
    print_error "获取租户ID失败"
    echo "响应: $TENANT_LIST_RESPONSE"
fi

# 3. 创建测试用户（用于搜索测试）
print_test "3. 创建测试用户"

# 创建用户1：张三
USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"$TENANT_ID\",
    \"email\": \"zhangsan@test.com\",
    \"password\": \"Test123456\",
    \"displayName\": \"张三\",
    \"phone\": \"13800138001\"
  }")

USER1_ID=$(echo $USER1_RESPONSE | jq -r '.data.id')
if [ "$USER1_ID" != "null" ] && [ -n "$USER1_ID" ]; then
    print_success "创建用户1成功: 张三 (zhangsan@test.com, 13800138001)"
else
    print_error "创建用户1失败"
    echo "响应: $USER1_RESPONSE"
fi

# 创建用户2：李四
USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"$TENANT_ID\",
    \"email\": \"lisi@test.com\",
    \"password\": \"Test123456\",
    \"displayName\": \"李四\",
    \"phone\": \"13900139002\"
  }")

USER2_ID=$(echo $USER2_RESPONSE | jq -r '.data.id')
if [ "$USER2_ID" != "null" ] && [ -n "$USER2_ID" ]; then
    print_success "创建用户2成功: 李四 (lisi@test.com, 13900139002)"
else
    print_error "创建用户2失败"
    echo "响应: $USER2_RESPONSE"
fi

# 创建用户3：王五
USER3_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenantId\": \"$TENANT_ID\",
    \"email\": \"wangwu@test.com\",
    \"password\": \"Test123456\",
    \"displayName\": \"王五\",
    \"phone\": \"13700137003\"
  }")

USER3_ID=$(echo $USER3_RESPONSE | jq -r '.data.id')
if [ "$USER3_ID" != "null" ] && [ -n "$USER3_ID" ]; then
    print_success "创建用户3成功: 王五 (wangwu@test.com, 13700137003)"
else
    print_error "创建用户3失败"
    echo "响应: $USER3_RESPONSE"
fi

# 4. 测试搜索功能 - 按 displayName 搜索
print_test "4. 按 displayName 搜索 - 搜索'张三'"
SEARCH_NAME_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=张三" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_NAME_COUNT=$(echo $SEARCH_NAME_RESPONSE | jq -r '.data.data | length')
SEARCH_NAME_FIRST=$(echo $SEARCH_NAME_RESPONSE | jq -r '.data.data[0].displayName')

echo "响应: $SEARCH_NAME_RESPONSE" | jq '.'
if [ "$SEARCH_NAME_COUNT" -ge "1" ] && [ "$SEARCH_NAME_FIRST" = "张三" ]; then
    print_success "按 displayName 搜索成功，找到 $SEARCH_NAME_COUNT 个结果"
else
    print_error "按 displayName 搜索失败"
fi

# 5. 测试搜索功能 - 按 phone 搜索
print_test "5. 按 phone 搜索 - 搜索'13900139002'"
SEARCH_PHONE_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=13900139002" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_PHONE_COUNT=$(echo $SEARCH_PHONE_RESPONSE | jq -r '.data.data | length')
SEARCH_PHONE_FIRST=$(echo $SEARCH_PHONE_RESPONSE | jq -r '.data.data[0].phone')

echo "响应: $SEARCH_PHONE_RESPONSE" | jq '.'
if [ "$SEARCH_PHONE_COUNT" -ge "1" ] && [ "$SEARCH_PHONE_FIRST" = "13900139002" ]; then
    print_success "按 phone 搜索成功，找到 $SEARCH_PHONE_COUNT 个结果"
else
    print_error "按 phone 搜索失败"
fi

# 6. 测试搜索功能 - 按 email 搜索
print_test "6. 按 email 搜索 - 搜索'wangwu@test.com'"
SEARCH_EMAIL_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=wangwu@test.com" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_EMAIL_COUNT=$(echo $SEARCH_EMAIL_RESPONSE | jq -r '.data.data | length')
SEARCH_EMAIL_FIRST=$(echo $SEARCH_EMAIL_RESPONSE | jq -r '.data.data[0].email')

echo "响应: $SEARCH_EMAIL_RESPONSE" | jq '.'
if [ "$SEARCH_EMAIL_COUNT" -ge "1" ] && [ "$SEARCH_EMAIL_FIRST" = "wangwu@test.com" ]; then
    print_success "按 email 搜索成功，找到 $SEARCH_EMAIL_COUNT 个结果"
else
    print_error "按 email 搜索失败"
fi

# 7. 测试搜索功能 - 模糊搜索
print_test "7. 模糊搜索 - 搜索'test'"
SEARCH_FUZZY_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=test" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_FUZZY_COUNT=$(echo $SEARCH_FUZZY_RESPONSE | jq -r '.data.data | length')

echo "响应: $SEARCH_FUZZY_RESPONSE" | jq '.'
if [ "$SEARCH_FUZZY_COUNT" -ge "3" ]; then
    print_success "模糊搜索成功，找到 $SEARCH_FUZZY_COUNT 个结果（应包含所有测试用户）"
else
    print_error "模糊搜索失败，只找到 $SEARCH_FUZZY_COUNT 个结果"
fi

# 8. 测试搜索功能 - 部分手机号搜索
print_test "8. 部分手机号搜索 - 搜索'138'"
SEARCH_PARTIAL_PHONE_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=138" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_PARTIAL_PHONE_COUNT=$(echo $SEARCH_PARTIAL_PHONE_RESPONSE | jq -r '.data.data | length')

echo "响应: $SEARCH_PARTIAL_PHONE_RESPONSE" | jq '.'
if [ "$SEARCH_PARTIAL_PHONE_COUNT" -ge "1" ]; then
    print_success "部分手机号搜索成功，找到 $SEARCH_PARTIAL_PHONE_COUNT 个结果"
else
    print_error "部分手机号搜索失败"
fi

# 9. 测试搜索功能 - 结合租户ID和搜索
print_test "9. 结合租户ID和搜索 - tenantId=$TENANT_ID, search=李四"
SEARCH_WITH_TENANT_RESPONSE=$(curl -s -X GET "$BASE_URL/users?tenantId=$TENANT_ID&search=李四" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_WITH_TENANT_COUNT=$(echo $SEARCH_WITH_TENANT_RESPONSE | jq -r '.data.data | length')
SEARCH_WITH_TENANT_FIRST=$(echo $SEARCH_WITH_TENANT_RESPONSE | jq -r '.data.data[0].displayName')

echo "响应: $SEARCH_WITH_TENANT_RESPONSE" | jq '.'
if [ "$SEARCH_WITH_TENANT_COUNT" -ge "1" ] && [ "$SEARCH_WITH_TENANT_FIRST" = "李四" ]; then
    print_success "结合租户ID和搜索成功，找到 $SEARCH_WITH_TENANT_COUNT 个结果"
else
    print_error "结合租户ID和搜索失败"
fi

# 10. 测试搜索功能 - 空搜索（应返回所有用户）
print_test "10. 空搜索 - 应返回所有用户"
SEARCH_EMPTY_RESPONSE=$(curl -s -X GET "$BASE_URL/users?search=" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

SEARCH_EMPTY_COUNT=$(echo $SEARCH_EMPTY_RESPONSE | jq -r '.data.data | length')

echo "响应: $SEARCH_EMPTY_RESPONSE" | jq '.'
if [ "$SEARCH_EMPTY_COUNT" -ge "3" ]; then
    print_success "空搜索成功，返回 $SEARCH_EMPTY_COUNT 个结果"
else
    print_error "空搜索失败，只返回 $SEARCH_EMPTY_COUNT 个结果"
fi

# 11. 清理测试数据
print_test "11. 清理测试数据"

if [ "$USER1_ID" != "null" ] && [ -n "$USER1_ID" ]; then
    curl -s -X DELETE "$BASE_URL/users/$USER1_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
    print_success "删除用户1成功"
fi

if [ "$USER2_ID" != "null" ] && [ -n "$USER2_ID" ]; then
    curl -s -X DELETE "$BASE_URL/users/$USER2_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
    print_success "删除用户2成功"
fi

if [ "$USER3_ID" != "null" ] && [ -n "$USER3_ID" ]; then
    curl -s -X DELETE "$BASE_URL/users/$USER3_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
    print_success "删除用户3成功"
fi

print_separator
echo -e "\n${GREEN}所有测试完成！${NC}\n"
