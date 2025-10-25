#!/bin/bash

# 租户列表过滤功能测试脚本
# 测试租户名称模糊搜索和状态过滤功能

set -e

# 配置
API_BASE_URL="http://localhost:8080/api/v1"
ADMIN_EMAIL="admin@platform.local"
ADMIN_PASSWORD="Admin@123456"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

print_section() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

# 登录获取 token
login() {
    print_section "1. 平台管理员登录"
    
    LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"${ADMIN_EMAIL}\",
            \"password\": \"${ADMIN_PASSWORD}\"
        }")
    
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.accessToken')
    
    if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
        print_success "登录成功"
        echo "Token: ${TOKEN:0:20}..."
    else
        print_error "登录失败"
        echo "$LOGIN_RESPONSE" | jq '.'
        exit 1
    fi
}

# 测试基本列表查询
test_basic_list() {
    print_section "2. 测试基本列表查询（无过滤条件）"
    
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    TOTAL=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "总租户数: $TOTAL"
}

# 测试租户名称模糊搜索
test_name_filter() {
    print_section "3. 测试租户名称模糊搜索"
    
    # 测试搜索 "平台"
    print_info "搜索关键词: 平台"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&name=平台" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    COUNT=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "匹配结果数: $COUNT"
    
    # 测试搜索 "test"
    echo ""
    print_info "搜索关键词: test"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&name=test" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    COUNT=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "匹配结果数: $COUNT"
}

# 测试状态过滤
test_status_filter() {
    print_section "4. 测试状态过滤"
    
    # 测试查询启用的租户
    print_info "查询启用的租户 (status=true)"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&status=true" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    COUNT=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "启用的租户数: $COUNT"
    
    # 测试查询禁用的租户
    echo ""
    print_info "查询禁用的租户 (status=false)"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&status=false" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    COUNT=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "禁用的租户数: $COUNT"
}

# 测试组合过滤
test_combined_filter() {
    print_section "5. 测试组合过滤（名称 + 状态）"
    
    print_info "搜索关键词: 平台, 状态: 启用"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&name=平台&status=true" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    COUNT=$(echo "$RESPONSE" | jq -r '.data.totalCount')
    print_info "匹配结果数: $COUNT"
}

# 测试无效参数
test_invalid_params() {
    print_section "6. 测试无效参数"
    
    print_info "测试无效的状态参数 (status=invalid)"
    RESPONSE=$(curl -s -X GET "${API_BASE_URL}/tenants?pageNo=1&pageSize=10&status=invalid" \
        -H "Authorization: Bearer ${TOKEN}")
    
    echo "$RESPONSE" | jq '.'
    
    CODE=$(echo "$RESPONSE" | jq -r '.code')
    if [ "$CODE" == "400" ]; then
        print_success "正确返回 400 错误"
    else
        print_error "未正确处理无效参数"
    fi
}

# 测试租户管理员权限
test_tenant_admin() {
    print_section "7. 测试租户管理员权限（如果存在）"
    
    print_info "租户管理员调用列表接口时，过滤参数应被忽略"
    print_info "租户管理员只能看到自己的租户信息"
    print_info "（需要先创建租户管理员账户才能测试此功能）"
}

# 主函数
main() {
    echo "======================================"
    echo "租户列表过滤功能测试"
    echo "======================================"
    echo ""
    
    # 检查 jq 是否安装
    if ! command -v jq &> /dev/null; then
        print_error "需要安装 jq 工具来解析 JSON"
        echo "macOS: brew install jq"
        echo "Ubuntu: sudo apt-get install jq"
        exit 1
    fi
    
    # 执行测试
    login
    test_basic_list
    test_name_filter
    test_status_filter
    test_combined_filter
    test_invalid_params
    test_tenant_admin
    
    print_section "测试完成"
    print_success "所有测试执行完毕"
}

# 运行主函数
main
