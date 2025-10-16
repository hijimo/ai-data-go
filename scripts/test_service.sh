#!/bin/bash

# 服务测试脚本
# 用于验证 Genkit AI 服务是否正常运行

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 服务地址
BASE_URL="http://localhost:8080"

echo "=========================================="
echo "  Genkit AI 服务测试"
echo "=========================================="
echo ""

# 1. 测试健康检查
echo -e "${YELLOW}[1/3] 测试健康检查接口...${NC}"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "${BASE_URL}/health")
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$HEALTH_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 健康检查通过${NC}"
    echo "响应: $RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
else
    echo -e "${RED}✗ 健康检查失败 (HTTP $HTTP_CODE)${NC}"
    echo "响应: $RESPONSE_BODY"
    exit 1
fi
echo ""

# 2. 测试对话接口
echo -e "${YELLOW}[2/3] 测试对话接口...${NC}"
CHAT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请用一句话介绍你自己",
    "sessionId": "test-session-'$(date +%s)'"
  }')

HTTP_CODE=$(echo "$CHAT_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$CHAT_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ 对话接口测试通过${NC}"
    echo "响应: $RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
else
    echo -e "${RED}✗ 对话接口测试失败 (HTTP $HTTP_CODE)${NC}"
    echo "响应: $RESPONSE_BODY"
    exit 1
fi
echo ""

# 3. 测试参数验证
echo -e "${YELLOW}[3/3] 测试参数验证...${NC}"
VALIDATION_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "",
    "sessionId": "test-session"
  }')

HTTP_CODE=$(echo "$VALIDATION_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$VALIDATION_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "422" ] || [ "$HTTP_CODE" = "400" ]; then
    echo -e "${GREEN}✓ 参数验证测试通过${NC}"
    echo "响应: $RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
else
    echo -e "${YELLOW}⚠ 参数验证返回了意外的状态码 (HTTP $HTTP_CODE)${NC}"
    echo "响应: $RESPONSE_BODY"
fi
echo ""

echo "=========================================="
echo -e "${GREEN}所有测试完成！${NC}"
echo "=========================================="
