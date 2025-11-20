#!/bin/bash

# 模型配置模块集成测试脚本
# 此脚本用于手动测试模型配置API的各项功能

set -e

# 配置
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TENANT1_ADMIN_TOKEN=""
TENANT2_ADMIN_TOKEN=""
SYSTEM_ADMIN_TOKEN=""

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

# 测试计数器
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
run_test() {
    local test_name="$1"
    local expected_code="$2"
    local actual_code="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$actual_code" -eq "$expected_code" ]; then
        print_success "$test_name (HTTP $actual_code)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_error "$test_name (Expected: $expected_code, Got: $actual_code)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

echo "========================================="
echo "模型配置模块集成测试"
echo "========================================="
echo ""

# 检查服务器是否运行
print_info "检查服务器状态..."
if ! curl -s -f "$API_BASE_URL/health" > /dev/null; then
    print_error "服务器未运行，请先启动服务器"
    exit 1
fi
print_success "服务器正在运行"
echo ""

# 注意：需要先手动获取Token
if [ -z "$TENANT1_ADMIN_TOKEN" ] || [ -z "$SYSTEM_ADMIN_TOKEN" ]; then
    print_info "请设置环境变量："
    echo "  export TENANT1_ADMIN_TOKEN='your_tenant1_admin_token'"
    echo "  export TENANT2_ADMIN_TOKEN='your_tenant2_admin_token'"
    echo "  export SYSTEM_ADMIN_TOKEN='your_system_admin_token'"
    echo ""
    print_info "可以通过登录API获取Token"
    exit 1
fi

echo "========================================="
echo "测试1: 租户管理员创建模型配置"
echo "========================================="
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/v1/model-configurations" \
  -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试配置1",
    "model": "gpt-4",
    "modelProvider": "openai",
    "apiKey": "sk-test123456789"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
run_test "租户管理员创建模型配置" 201 "$HTTP_CODE"

if [ "$HTTP_CODE" -eq 201 ]; then
    CONFIG1_ID=$(echo "$BODY" | jq -r '.data.id')
    print_info "创建的配置ID: $CONFIG1_ID"
    
    # 验证API密钥已脱敏
    API_KEY=$(echo "$BODY" | jq -r '.data.apiKey')
    if echo "$API_KEY" | grep -q "****"; then
        print_success "API密钥已正确脱敏"
    else
        print_error "API密钥未脱敏"
    fi
fi
echo ""

echo "========================================="
echo "测试2: 平台管理员创建模型配置（指定租户ID）"
echo "========================================="
# 注意：需要先获取租户ID
print_info "此测试需要租户ID，请手动运行"
echo ""

echo "========================================="
echo "测试3: 租户管理员尝试跨租户创建（应失败）"
echo "========================================="
print_info "此测试需要另一个租户的ID，请手动运行"
echo ""

echo "========================================="
echo "测试4: 获取模型配置详情"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "获取模型配置详情" 200 "$HTTP_CODE"
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试5: 查询模型配置列表"
echo "========================================="
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE_URL/api/v1/model-configurations?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
run_test "查询模型配置列表" 200 "$HTTP_CODE"

if [ "$HTTP_CODE" -eq 200 ]; then
    COUNT=$(echo "$BODY" | jq '.data.data | length')
    print_info "返回配置数量: $COUNT"
fi
echo ""

echo "========================================="
echo "测试6: 更新模型配置"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "name": "更新后的配置名称",
        "model": "gpt-4-turbo"
      }')
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "更新模型配置" 200 "$HTTP_CODE"
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试7: 禁用模型配置"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID/status" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "status": "disabled"
      }')
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "禁用模型配置" 200 "$HTTP_CODE"
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试8: 启用模型配置"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID/status" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "status": "enabled"
      }')
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "启用模型配置" 200 "$HTTP_CODE"
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试9: 查询可用模型列表"
echo "========================================="
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE_URL/api/v1/model-configurations/available" \
  -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
run_test "查询可用模型列表" 200 "$HTTP_CODE"

if [ "$HTTP_CODE" -eq 200 ]; then
    COUNT=$(echo "$BODY" | jq '.data | length')
    print_info "可用配置数量: $COUNT"
fi
echo ""

echo "========================================="
echo "测试10: 验证模型配置"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID/validate" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    run_test "验证模型配置" 200 "$HTTP_CODE"
    
    if [ "$HTTP_CODE" -eq 200 ]; then
        VALID=$(echo "$BODY" | jq -r '.data.valid')
        MESSAGE=$(echo "$BODY" | jq -r '.data.message')
        print_info "验证结果: $VALID - $MESSAGE"
    fi
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试11: 删除模型配置（软删除）"
echo "========================================="
if [ -n "$CONFIG1_ID" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "删除模型配置" 200 "$HTTP_CODE"
    
    # 验证删除后无法访问
    RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE_URL/api/v1/model-configurations/$CONFIG1_ID" \
      -H "Authorization: Bearer $TENANT1_ADMIN_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    run_test "验证删除后无法访问" 403 "$HTTP_CODE"
else
    print_info "跳过：需要先创建配置"
fi
echo ""

echo "========================================="
echo "测试12: 未认证访问（应失败）"
echo "========================================="
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE_URL/api/v1/model-configurations")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
run_test "未认证访问" 401 "$HTTP_CODE"
echo ""

echo "========================================="
echo "测试总结"
echo "========================================="
echo "总测试数: $TOTAL_TESTS"
echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
echo -e "${RED}失败: $FAILED_TESTS${NC}"
echo ""

if [ "$FAILED_TESTS" -eq 0 ]; then
    print_success "所有测试通过！"
    exit 0
else
    print_error "部分测试失败"
    exit 1
fi
