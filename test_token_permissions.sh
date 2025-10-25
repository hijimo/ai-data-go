#!/bin/bash

# 测试 Token 权限脚本
# 使用提供的 system_admin token 测试各种接口权限

set -e

# 配置
BASE_URL="http://localhost:8080"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJnZW5raXQtYWktc2VydmljZSIsInN1YiI6ImU0YzU3YzQ0LWQyOGQtNDIyZi1hNzU4LWEwZjVmMWQ4NDkyNyIsImF1ZCI6WyJnZW5raXQtYXBpIl0sImV4cCI6MTc2MTM3ODQyNCwiaWF0IjoxNzYxMzc0ODI0LCJqdGkiOiI4YmI1NGU0MC1mNTc1LTQ0NzItOWNhZC04NjczMDBmNzUxYTUiLCJ0aWQiOiJmOThjNzA1YS0zYmI0LTQ1NTEtOTU2OC03YTYwN2ZiMjUxOTUiLCJkaXNwbGF5TmFtZSI6IlBsYXRmb3JtIEFkbWluIiwicm9sZXMiOlsic3lzdGVtX2FkbWluIl0sInNjb3BlcyI6W119.AIyTNyKQNpC7ru5SiM0TkC-dlOQ_4LZ96n9k0wirHek"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印分隔线
print_separator() {
    echo -e "${BLUE}========================================${NC}"
}

# 打印测试标题
print_test() {
    echo -e "\n${YELLOW}测试: $1${NC}"
}

# 打印成功信息
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 打印失败信息
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# 执行 API 请求
api_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            "$BASE_URL$endpoint"
    else
        curl -s -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint"
    fi
}

# 开始测试
print_separator
echo -e "${BLUE}Token 权限测试${NC}"
print_separator

# 1. 测试获取当前用户信息
print_test "1. 获取当前用户信息 (GET /api/v1/auth/me)"
response=$(api_request "GET" "/api/v1/auth/me")
echo "$response" | jq '.'
if echo "$response" | jq -e '.code == 200' > /dev/null; then
    print_success "成功获取用户信息"
    echo "$response" | jq -r '.data | "用户ID: \(.id)\n显示名称: \(.displayName)\n角色: \(.roles | join(", "))\n租户ID: \(.tenantId)"'
else
    print_error "获取用户信息失败"
fi

# 2. 测试获取租户列表
print_test "2. 获取租户列表 (GET /api/v1/tenants)"
response=$(api_request "GET" "/api/v1/tenants?pageNo=1&pageSize=10")
echo "$response" | jq '.'
if echo "$response" | jq -e '.code == 200' > /dev/null; then
    print_success "成功获取租户列表"
    tenant_count=$(echo "$response" | jq -r '.data.totalCount')
    echo "租户总数: $tenant_count"
else
    print_error "获取租户列表失败"
fi

# 3. 测试获取用户列表
print_test "3. 获取用户列表 (GET /api/v1/users)"
response=$(api_request "GET" "/api/v1/users?pageNo=1&pageSize=10")
echo "$response" | jq '.'
if echo "$response" | jq -e '.code == 200' > /dev/null; then
    print_success "成功获取用户列表"
    user_count=$(echo "$response" | jq -r '.data.totalCount')
    echo "用户总数: $user_count"
else
    print_error "获取用户列表失败"
fi

# 4. 测试创建租户（平台管理员权限）
print_test "4. 创建租户 (POST /api/v1/tenants)"
tenant_data='{
  "name": "测试租户_'$(date +%s)'",
  "domain": "test'$(date +%s)'.example.com",
  "adminEmail": "admin'$(date +%s)'@test.com",
  "adminName": "测试管理员",
  "metadata": {
    "description": "这是一个测试租户"
  }
}'
response=$(api_request "POST" "/api/v1/tenants" "$tenant_data")
echo "$response" | jq '.'
if echo "$response" | jq -e '.code == 200' > /dev/null; then
    print_success "成功创建租户"
    new_tenant_id=$(echo "$response" | jq -r '.data.tenant.id')
    new_admin_id=$(echo "$response" | jq -r '.data.admin.id')
    new_admin_password=$(echo "$response" | jq -r '.data.initialPassword')
    echo "新租户ID: $new_tenant_id"
    echo "新管理员ID: $new_admin_id"
    echo "初始密码: $new_admin_password"
    
    # 保存新租户ID用于后续测试
    NEW_TENANT_ID=$new_tenant_id
    NEW_ADMIN_EMAIL=$(echo "$response" | jq -r '.data.admin.email')
else
    print_error "创建租户失败"
fi

# 5. 测试获取特定租户详情
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "5. 获取租户详情 (GET /api/v1/tenants/$NEW_TENANT_ID)"
    response=$(api_request "GET" "/api/v1/tenants/$NEW_TENANT_ID")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功获取租户详情"
    else
        print_error "获取租户详情失败"
    fi
fi

# 6. 测试更新租户
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "6. 更新租户 (PUT /api/v1/tenants/$NEW_TENANT_ID)"
    update_data='{
      "name": "更新后的测试租户",
      "metadata": {
        "description": "这是一个更新后的测试租户",
        "updated": true
      }
    }'
    response=$(api_request "PUT" "/api/v1/tenants/$NEW_TENANT_ID" "$update_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功更新租户"
    else
        print_error "更新租户失败"
    fi
fi

# 7. 测试禁用租户
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "7. 禁用租户 (PATCH /api/v1/tenants/$NEW_TENANT_ID/status)"
    status_data='{"status": false}'
    response=$(api_request "PATCH" "/api/v1/tenants/$NEW_TENANT_ID/status" "$status_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功禁用租户"
    else
        print_error "禁用租户失败"
    fi
fi

# 8. 测试启用租户
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "8. 启用租户 (PATCH /api/v1/tenants/$NEW_TENANT_ID/status)"
    status_data='{"status": true}'
    response=$(api_request "PATCH" "/api/v1/tenants/$NEW_TENANT_ID/status" "$status_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功启用租户"
    else
        print_error "启用租户失败"
    fi
fi

# 9. 测试在新租户下创建用户
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "9. 在新租户下创建用户 (POST /api/v1/users)"
    user_data='{
      "tenantId": "'$NEW_TENANT_ID'",
      "email": "user'$(date +%s)'@test.com",
      "displayName": "测试用户",
      "password": "TestUser123!",
      "roles": ["user"]
    }'
    response=$(api_request "POST" "/api/v1/users" "$user_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功创建用户"
        NEW_USER_ID=$(echo "$response" | jq -r '.data.id')
        echo "新用户ID: $NEW_USER_ID"
    else
        print_error "创建用户失败"
    fi
fi

# 10. 测试获取特定用户详情
if [ ! -z "$NEW_USER_ID" ]; then
    print_test "10. 获取用户详情 (GET /api/v1/users/$NEW_USER_ID)"
    response=$(api_request "GET" "/api/v1/users/$NEW_USER_ID")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功获取用户详情"
    else
        print_error "获取用户详情失败"
    fi
fi

# 11. 测试更新用户
if [ ! -z "$NEW_USER_ID" ]; then
    print_test "11. 更新用户 (PUT /api/v1/users/$NEW_USER_ID)"
    update_user_data='{
      "displayName": "更新后的测试用户",
      "metadata": {
        "updated": true
      }
    }'
    response=$(api_request "PUT" "/api/v1/users/$NEW_USER_ID" "$update_user_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功更新用户"
    else
        print_error "更新用户失败"
    fi
fi

# 12. 测试禁用用户
if [ ! -z "$NEW_USER_ID" ]; then
    print_test "12. 禁用用户 (PATCH /api/v1/users/$NEW_USER_ID/status)"
    user_status_data='{"status": false}'
    response=$(api_request "PATCH" "/api/v1/users/$NEW_USER_ID/status" "$user_status_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功禁用用户"
    else
        print_error "禁用用户失败"
    fi
fi

# 13. 测试启用用户
if [ ! -z "$NEW_USER_ID" ]; then
    print_test "13. 启用用户 (PATCH /api/v1/users/$NEW_USER_ID/status)"
    user_status_data='{"status": true}'
    response=$(api_request "PATCH" "/api/v1/users/$NEW_USER_ID/status" "$user_status_data")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功启用用户"
    else
        print_error "启用用户失败"
    fi
fi

# 14. 测试删除用户
if [ ! -z "$NEW_USER_ID" ]; then
    print_test "14. 删除用户 (DELETE /api/v1/users/$NEW_USER_ID)"
    response=$(api_request "DELETE" "/api/v1/users/$NEW_USER_ID")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功删除用户"
    else
        print_error "删除用户失败"
    fi
fi

# 15. 测试删除租户
if [ ! -z "$NEW_TENANT_ID" ]; then
    print_test "15. 删除租户 (DELETE /api/v1/tenants/$NEW_TENANT_ID)"
    response=$(api_request "DELETE" "/api/v1/tenants/$NEW_TENANT_ID")
    echo "$response" | jq '.'
    if echo "$response" | jq -e '.code == 200' > /dev/null; then
        print_success "成功删除租户"
    else
        print_error "删除租户失败"
    fi
fi

# 测试总结
print_separator
echo -e "${BLUE}测试完成！${NC}"
print_separator
