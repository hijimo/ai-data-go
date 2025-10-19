#!/bin/bash

# 性能监控测试脚本
# 用于测试监控 API 端点

set -e

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
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

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        print_error "curl 未安装"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        print_info "jq 未安装，输出将不会格式化"
    fi
}

# 获取管理员 Token（如果未提供）
get_admin_token() {
    if [ -z "$ADMIN_TOKEN" ]; then
        print_info "未提供管理员 Token，尝试登录获取..."
        
        # 这里需要根据实际情况修改登录信息
        RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d '{
                "email": "admin@example.com",
                "password": "admin123456"
            }')
        
        ADMIN_TOKEN=$(echo "$RESPONSE" | jq -r '.data.access_token')
        
        if [ "$ADMIN_TOKEN" = "null" ] || [ -z "$ADMIN_TOKEN" ]; then
            print_error "无法获取管理员 Token"
            print_info "请设置 ADMIN_TOKEN 环境变量或确保管理员账户存在"
            exit 1
        fi
        
        print_success "成功获取管理员 Token"
    fi
}

# 测试健康检查端点
test_health_check() {
    print_info "测试健康检查端点..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/monitoring/health")
    
    if command -v jq &> /dev/null; then
        echo "$RESPONSE" | jq '.'
    else
        echo "$RESPONSE"
    fi
    
    STATUS=$(echo "$RESPONSE" | jq -r '.data.status')
    
    if [ "$STATUS" = "healthy" ] || [ "$STATUS" = "degraded" ]; then
        print_success "健康检查通过，状态: $STATUS"
    else
        print_error "健康检查失败，状态: $STATUS"
    fi
    
    echo ""
}

# 测试获取性能指标
test_get_metrics() {
    print_info "测试获取性能指标..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/monitoring/metrics" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    
    if command -v jq &> /dev/null; then
        echo "$RESPONSE" | jq '.'
    else
        echo "$RESPONSE"
    fi
    
    CODE=$(echo "$RESPONSE" | jq -r '.code')
    
    if [ "$CODE" = "200" ]; then
        print_success "成功获取性能指标"
        
        # 显示关键指标
        LOGIN_ATTEMPTS=$(echo "$RESPONSE" | jq -r '.data.login_attempts')
        LOGIN_SUCCESS_RATE=$(echo "$RESPONSE" | jq -r '.data.login_success_rate')
        SLOW_QUERIES=$(echo "$RESPONSE" | jq -r '.data.slow_queries')
        
        print_info "登录尝试次数: $LOGIN_ATTEMPTS"
        print_info "登录成功率: $LOGIN_SUCCESS_RATE%"
        print_info "慢查询次数: $SLOW_QUERIES"
    else
        print_error "获取性能指标失败"
    fi
    
    echo ""
}

# 测试获取告警
test_get_alerts() {
    print_info "测试获取告警..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/monitoring/alerts" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    
    if command -v jq &> /dev/null; then
        echo "$RESPONSE" | jq '.'
    else
        echo "$RESPONSE"
    fi
    
    CODE=$(echo "$RESPONSE" | jq -r '.code')
    
    if [ "$CODE" = "200" ]; then
        print_success "成功获取告警列表"
        
        ALERT_COUNT=$(echo "$RESPONSE" | jq -r '.data | length')
        print_info "当前活跃告警数: $ALERT_COUNT"
        
        if [ "$ALERT_COUNT" -gt 0 ]; then
            print_info "告警详情:"
            echo "$RESPONSE" | jq -r '.data[] | "  - [\(.level)] \(.title): \(.message)"'
        fi
    else
        print_error "获取告警失败"
    fi
    
    echo ""
}

# 测试清空告警
test_clear_alerts() {
    print_info "测试清空告警..."
    
    RESPONSE=$(curl -s -X DELETE "$BASE_URL/api/v1/monitoring/alerts" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    
    CODE=$(echo "$RESPONSE" | jq -r '.code')
    
    if [ "$CODE" = "200" ]; then
        print_success "成功清空告警"
    else
        print_error "清空告警失败"
    fi
    
    echo ""
}

# 测试重置指标
test_reset_metrics() {
    print_info "测试重置指标..."
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/monitoring/metrics/reset" \
        -H "Authorization: Bearer $ADMIN_TOKEN")
    
    CODE=$(echo "$RESPONSE" | jq -r '.code')
    
    if [ "$CODE" = "200" ]; then
        print_success "成功重置指标"
    else
        print_error "重置指标失败"
    fi
    
    echo ""
}

# 模拟一些登录请求以生成指标
simulate_login_requests() {
    print_info "模拟登录请求以生成指标..."
    
    # 模拟成功登录
    for i in {1..5}; do
        curl -s -X POST "$BASE_URL/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d '{
                "email": "user'$i'@example.com",
                "password": "password123"
            }' > /dev/null
    done
    
    # 模拟失败登录
    for i in {1..3}; do
        curl -s -X POST "$BASE_URL/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d '{
                "email": "invalid@example.com",
                "password": "wrongpassword"
            }' > /dev/null
    done
    
    print_success "完成模拟登录请求"
    echo ""
}

# 主函数
main() {
    echo "========================================="
    echo "  性能监控测试脚本"
    echo "========================================="
    echo ""
    
    check_dependencies
    get_admin_token
    
    # 运行测试
    test_health_check
    
    # 可选：模拟一些请求
    read -p "是否模拟登录请求以生成指标？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        simulate_login_requests
    fi
    
    test_get_metrics
    test_get_alerts
    
    # 询问是否清空告警
    if [ "$(curl -s -X GET "$BASE_URL/api/v1/monitoring/alerts" -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.data | length')" -gt 0 ]; then
        read -p "是否清空告警？(y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            test_clear_alerts
        fi
    fi
    
    # 询问是否重置指标
    read -p "是否重置指标？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        test_reset_metrics
        test_get_metrics
    fi
    
    echo "========================================="
    echo "  测试完成"
    echo "========================================="
}

# 运行主函数
main
