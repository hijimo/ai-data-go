#!/bin/bash

# 审计日志查询 API 测试脚本
# 用法: ./scripts/test_audit_api.sh

set -e

# 配置
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TENANT_ID="${TENANT_ID:-}"
ACCESS_TOKEN="${ACCESS_TOKEN:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_section() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

# 检查必需的环境变量
check_env() {
    if [ -z "$ACCESS_TOKEN" ]; then
        print_error "请设置 ACCESS_TOKEN 环境变量"
        echo "示例: export ACCESS_TOKEN='your_access_token_here'"
        exit 1
    fi

    if [ -z "$TENANT_ID" ]; then
        print_warning "未设置 TENANT_ID，将从请求头中获取"
    fi
}

# 执行 API 请求
make_request() {
    local endpoint=$1
    local description=$2
    
    print_info "测试: $description"
    print_info "请求: GET $endpoint"
    
    local headers=(-H "Authorization: Bearer $ACCESS_TOKEN")
    if [ -n "$TENANT_ID" ]; then
        headers+=(-H "X-Tenant-ID: $TENANT_ID")
    fi
    
    local response=$(curl -s -w "\n%{http_code}" "${headers[@]}" "$API_BASE_URL$endpoint")
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        print_info "✓ 成功 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        print_error "✗ 失败 (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    fi
    
    echo ""
}

# 主测试流程
main() {
    print_section "审计日志查询 API 测试"
    
    check_env
    
    print_info "API 基础URL: $API_BASE_URL"
    if [ -n "$TENANT_ID" ]; then
        print_info "租户ID: $TENANT_ID"
    fi
    
    # 测试 1: 查询所有审计日志（第一页）
    print_section "测试 1: 查询所有审计日志（分页）"
    make_request "/api/v1/audit/auth?page=1&pageSize=10" "查询第一页，每页10条"
    
    # 测试 2: 按事件类型过滤
    print_section "测试 2: 按事件类型过滤"
    make_request "/api/v1/audit/auth?event=login&page=1&pageSize=10" "查询登录事件"
    
    # 测试 3: 查询登录失败记录
    print_section "测试 3: 查询登录失败记录"
    make_request "/api/v1/audit/auth?event=failed_login&page=1&pageSize=10" "查询登录失败事件"
    
    # 测试 4: 按时间范围过滤
    print_section "测试 4: 按时间范围过滤"
    local start_time=$(date -u -v-7d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "7 days ago" +"%Y-%m-%dT%H:%M:%SZ")
    local end_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    make_request "/api/v1/audit/auth?startTime=$start_time&endTime=$end_time&page=1&pageSize=10" "查询最近7天的审计日志"
    
    # 测试 5: 查询注销事件
    print_section "测试 5: 查询注销事件"
    make_request "/api/v1/audit/auth?event=logout&page=1&pageSize=10" "查询注销事件"
    
    # 测试 6: 查询 Token 刷新事件
    print_section "测试 6: 查询 Token 刷新事件"
    make_request "/api/v1/audit/auth?event=refresh&page=1&pageSize=10" "查询 Token 刷新事件"
    
    # 测试 7: 大分页测试
    print_section "测试 7: 大分页测试"
    make_request "/api/v1/audit/auth?page=1&pageSize=50" "查询第一页，每页50条"
    
    # 测试 8: 无效参数测试
    print_section "测试 8: 无效参数测试"
    make_request "/api/v1/audit/auth?tenantId=invalid-uuid" "使用无效的租户ID"
    
    print_section "测试完成"
    print_info "所有测试已执行完毕"
}

# 显示帮助信息
show_help() {
    cat << EOF
审计日志查询 API 测试脚本

用法:
    ./scripts/test_audit_api.sh

环境变量:
    ACCESS_TOKEN    (必需) JWT Access Token
    TENANT_ID       (可选) 租户ID
    API_BASE_URL    (可选) API 基础URL，默认: http://localhost:8080

示例:
    # 设置环境变量并运行测试
    export ACCESS_TOKEN="your_access_token_here"
    export TENANT_ID="550e8400-e29b-41d4-a716-446655440000"
    ./scripts/test_audit_api.sh

    # 使用自定义 API URL
    export API_BASE_URL="https://api.example.com"
    export ACCESS_TOKEN="your_access_token_here"
    ./scripts/test_audit_api.sh

注意:
    - 需要管理员权限才能访问审计日志 API
    - 确保已安装 jq 工具以获得更好的 JSON 输出格式
    - 如果未安装 jq，脚本仍可运行，但输出格式可能不够美观

EOF
}

# 检查命令行参数
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    show_help
    exit 0
fi

# 运行主程序
main
