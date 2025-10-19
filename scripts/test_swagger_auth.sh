#!/bin/bash

# 认证 API Swagger 文档测试脚本
# 用于验证所有认证相关的 API 是否都在 Swagger 文档中正确定义

set -e

echo "=========================================="
echo "认证 API Swagger 文档验证"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 swagger.json 是否存在
if [ ! -f "docs/swagger.json" ]; then
    echo -e "${RED}❌ 错误: docs/swagger.json 文件不存在${NC}"
    echo "请先运行: make swagger"
    exit 1
fi

echo "✓ 找到 Swagger 文档文件"
echo ""

# 定义需要检查的 API 端点
declare -a AUTH_ENDPOINTS=(
    "/api/v1/auth/register"
    "/api/v1/auth/login"
    "/api/v1/auth/refresh"
    "/api/v1/auth/logout"
    "/api/v1/auth/change-password"
    "/api/v1/auth/me"
)

declare -a TENANT_ENDPOINTS=(
    "/api/v1/tenants"
    "/api/v1/tenants/{id}"
)

declare -a USER_ENDPOINTS=(
    "/api/v1/users"
    "/api/v1/users/{id}"
)

# 定义需要检查的请求模型
declare -a REQUEST_MODELS=(
    "RegisterRequest"
    "LoginRequest"
    "RefreshRequest"
    "LogoutRequest"
    "ChangePasswordRequest"
    "CreateTenantRequest"
    "UpdateTenantRequest"
    "CreateUserRequest"
    "UpdateUserRequest"
)

# 定义需要检查的响应模型
declare -a RESPONSE_MODELS=(
    "User"
    "Tenant"
    "LoginResponse"
    "ResponseData"
    "ResponsePaginationData"
    "ErrorResponse"
)

# 检查函数
check_endpoint() {
    local endpoint=$1
    if grep -q "\"$endpoint\"" docs/swagger.json; then
        echo -e "${GREEN}✓${NC} 端点存在: $endpoint"
        return 0
    else
        echo -e "${RED}✗${NC} 端点缺失: $endpoint"
        return 1
    fi
}

check_model() {
    local model=$1
    # 对于泛型类型，检查是否存在任何包含该名称的定义
    if grep -q "\"internal_api_handler.$model\"" docs/swagger.json || \
       grep -q "\"genkit-ai-service_internal_model.$model\"" docs/swagger.json || \
       grep -q "$model-" docs/swagger.json; then
        echo -e "${GREEN}✓${NC} 模型存在: $model"
        return 0
    else
        echo -e "${RED}✗${NC} 模型缺失: $model"
        return 1
    fi
}

# 检查安全定义
check_security() {
    if grep -q "\"BearerAuth\"" docs/swagger.json; then
        echo -e "${GREEN}✓${NC} BearerAuth 安全定义存在"
        return 0
    else
        echo -e "${RED}✗${NC} BearerAuth 安全定义缺失"
        return 1
    fi
}

# 开始检查
echo "=========================================="
echo "1. 检查认证端点"
echo "=========================================="
failed=0
for endpoint in "${AUTH_ENDPOINTS[@]}"; do
    check_endpoint "$endpoint" || ((failed++))
done
echo ""

echo "=========================================="
echo "2. 检查租户管理端点"
echo "=========================================="
for endpoint in "${TENANT_ENDPOINTS[@]}"; do
    check_endpoint "$endpoint" || ((failed++))
done
echo ""

echo "=========================================="
echo "3. 检查用户管理端点"
echo "=========================================="
for endpoint in "${USER_ENDPOINTS[@]}"; do
    check_endpoint "$endpoint" || ((failed++))
done
echo ""

echo "=========================================="
echo "4. 检查请求模型"
echo "=========================================="
for model in "${REQUEST_MODELS[@]}"; do
    check_model "$model" || ((failed++))
done
echo ""

echo "=========================================="
echo "5. 检查响应模型"
echo "=========================================="
for model in "${RESPONSE_MODELS[@]}"; do
    check_model "$model" || ((failed++))
done
echo ""

echo "=========================================="
echo "6. 检查安全定义"
echo "=========================================="
check_security || ((failed++))
echo ""

# 检查标签
echo "=========================================="
echo "7. 检查 API 标签"
echo "=========================================="
if grep -q "\"name\": \"认证\"" docs/swagger.json; then
    echo -e "${GREEN}✓${NC} 标签存在: 认证"
else
    echo -e "${RED}✗${NC} 标签缺失: 认证"
    ((failed++))
fi

if grep -q "\"name\": \"租户管理\"" docs/swagger.json; then
    echo -e "${GREEN}✓${NC} 标签存在: 租户管理"
else
    echo -e "${RED}✗${NC} 标签缺失: 租户管理"
    ((failed++))
fi

if grep -q "\"name\": \"用户管理\"" docs/swagger.json; then
    echo -e "${GREEN}✓${NC} 标签存在: 用户管理"
else
    echo -e "${RED}✗${NC} 标签缺失: 用户管理"
    ((failed++))
fi
echo ""

# 统计结果
echo "=========================================="
echo "验证结果"
echo "=========================================="
if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✓ 所有检查通过！${NC}"
    echo ""
    echo "Swagger 文档已完整生成，包含："
    echo "  - ${#AUTH_ENDPOINTS[@]} 个认证端点"
    echo "  - ${#TENANT_ENDPOINTS[@]} 个租户管理端点"
    echo "  - ${#USER_ENDPOINTS[@]} 个用户管理端点"
    echo "  - ${#REQUEST_MODELS[@]} 个请求模型"
    echo "  - ${#RESPONSE_MODELS[@]} 个响应模型"
    echo "  - BearerAuth 安全定义"
    echo "  - 3 个 API 标签"
    echo ""
    echo "访问 Swagger UI: http://localhost:8080/swagger/index.html"
    exit 0
else
    echo -e "${RED}✗ 发现 $failed 个问题${NC}"
    echo ""
    echo "请检查并修复上述问题，然后重新生成 Swagger 文档："
    echo "  make swagger"
    exit 1
fi
