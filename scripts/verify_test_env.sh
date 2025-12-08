#!/bin/bash

# 测试环境验证脚本
# 用于验证测试环境配置是否正确
# 用法: ./scripts/verify_test_env.sh

set -e

echo "========================================="
echo "测试环境验证"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 验证结果统计
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# 检查函数
check() {
    local name=$1
    local command=$2
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    echo -n "检查 $name... "
    
    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 通过${NC}"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        echo -e "${RED}✗ 失败${NC}"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# 检查环境变量
check_env() {
    local name=$1
    local var=$2
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    echo -n "检查环境变量 $name... "
    
    if [ -n "${!var}" ] && [ "${!var}" != "your_"* ]; then
        echo -e "${GREEN}✓ 已配置${NC}"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        echo -e "${YELLOW}⚠ 未配置${NC}"
        return 1
    fi
}

# 加载环境变量
if [ -f ".env.test" ]; then
    export $(cat .env.test | grep -v '^#' | xargs)
    echo -e "${GREEN}✓ 已加载 .env.test 配置${NC}"
else
    echo -e "${RED}❌ 错误: .env.test 文件不存在${NC}"
    echo "请先运行: ./scripts/setup_test_env.sh"
    exit 1
fi

echo ""
echo "========================================="
echo "1. 基础环境检查"
echo "========================================="

# 检查必需的命令
check "Go 环境" "command -v go"
check "PostgreSQL 客户端" "command -v psql"
check "Redis 客户端" "command -v redis-cli"
check "curl 工具" "command -v curl"
check "jq 工具" "command -v jq"

echo ""
echo "========================================="
echo "2. 数据库连接检查"
echo "========================================="

# 检查 PostgreSQL 连接
check "PostgreSQL 连接" "PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c 'SELECT 1'"

# 检查测试数据库是否存在
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo -e "检查测试数据库... ${GREEN}✓ 存在${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "检查测试数据库... ${YELLOW}⚠ 不存在（将自动创建）${NC}"
fi
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

# 检查 Redis 连接
if [ "$REDIS_ENABLED" == "true" ]; then
    if [ -n "$REDIS_PASSWORD" ]; then
        check "Redis 连接" "redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD ping"
    else
        check "Redis 连接" "redis-cli -h $REDIS_HOST -p $REDIS_PORT ping"
    fi
fi

echo ""
echo "========================================="
echo "3. API 密钥配置检查"
echo "========================================="

# 检查 Google AI 配置
check_env "Google AI API Key" "GOOGLE_API_KEY"

# 检查 Azure OpenAI 配置
if check_env "Azure OpenAI API Key" "AZURE_OPENAI_API_KEY"; then
    check_env "Azure OpenAI Endpoint" "AZURE_OPENAI_ENDPOINT"
    check_env "Azure OpenAI Deployment" "AZURE_OPENAI_DEPLOYMENT"
fi

# 检查百炼配置
if check_env "百炼 API Key" "BAILIAN_API_KEY"; then
    check_env "百炼 Endpoint" "BAILIAN_ENDPOINT"
fi

echo ""
echo "========================================="
echo "4. 安全配置检查"
echo "========================================="

# 检查 JWT 密钥
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
echo -n "检查 JWT 密钥长度... "
if [ ${#JWT_SECRET} -ge 32 ]; then
    echo -e "${GREEN}✓ 符合要求 (${#JWT_SECRET} 字节)${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "${RED}✗ 太短 (${#JWT_SECRET} 字节，建议至少 32 字节)${NC}"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# 检查加密密钥
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
echo -n "检查加密密钥长度... "
if [ ${#ENCRYPTION_SECRET_KEY} -eq 32 ]; then
    echo -e "${GREEN}✓ 符合要求 (32 字节)${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "${RED}✗ 不符合 (${#ENCRYPTION_SECRET_KEY} 字节，必须是 32 字节)${NC}"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

echo ""
echo "========================================="
echo "5. 服务状态检查"
echo "========================================="

# 检查服务是否运行
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
echo -n "检查服务状态... "
if curl -s http://localhost:${SERVER_PORT:-8080}/health > /dev/null; then
    echo -e "${GREEN}✓ 运行中${NC}"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
    
    # 检查 API 端点
    check "API 健康检查" "curl -s http://localhost:${SERVER_PORT:-8080}/api/v1/health"
    check "Swagger 文档" "curl -s http://localhost:${SERVER_PORT:-8080}/swagger/index.html"
else
    echo -e "${YELLOW}⚠ 未运行${NC}"
    echo "  提示: 运行 'make run-test' 启动服务"
fi

echo ""
echo "========================================="
echo "验证结果汇总"
echo "========================================="
echo ""
echo "总检查项: $TOTAL_CHECKS"
echo -e "${GREEN}通过: $PASSED_CHECKS${NC}"
echo -e "${RED}失败: $FAILED_CHECKS${NC}"
echo ""

# 计算通过率
PASS_RATE=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))

if [ $PASS_RATE -eq 100 ]; then
    echo -e "${GREEN}✓ 所有检查通过！测试环境配置正确。${NC}"
    exit 0
elif [ $PASS_RATE -ge 80 ]; then
    echo -e "${YELLOW}⚠ 大部分检查通过 ($PASS_RATE%)，但仍有一些问题需要解决。${NC}"
    exit 0
else
    echo -e "${RED}✗ 检查通过率较低 ($PASS_RATE%)，请检查配置。${NC}"
    exit 1
fi
