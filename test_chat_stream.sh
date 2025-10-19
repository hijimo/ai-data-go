#!/bin/bash

# 流式对话接口测试脚本
# 使用 curl 测试 /api/v1/chat/stream 接口

# 设置颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 服务器地址
BASE_URL="${BASE_URL:-http://localhost:8080}"

echo -e "${BLUE}=== 流式对话接口测试 ===${NC}\n"

# 测试1: 基本流式对话
echo -e "${GREEN}测试1: 基本流式对话${NC}"
echo "请求: 你好，请简单介绍一下你自己"
echo ""

curl -N -X POST "${BASE_URL}/api/v1/chat/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请简单介绍一下你自己"
  }'

echo -e "\n"

# 测试2: 带参数的流式对话
echo -e "${GREEN}测试2: 带参数的流式对话（temperature=0.9）${NC}"
echo "请求: 写一首关于春天的诗"
echo ""

curl -N -X POST "${BASE_URL}/api/v1/chat/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "写一首关于春天的诗",
    "options": {
      "temperature": 0.9,
      "maxTokens": 500
    }
  }'

echo -e "\n"

# 测试3: 继续对话（使用 messageId）
echo -e "${GREEN}测试3: 继续对话（使用 messageId）${NC}"
echo "请求: 继续上一个话题"
echo ""

# 注意：这里需要替换为实际的 messageId
MESSAGE_ID="550e8400-e29b-41d4-a716-446655440000"

curl -N -X POST "${BASE_URL}/api/v1/chat/stream" \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"继续上一个话题\",
    \"messageId\": \"${MESSAGE_ID}\"
  }"

echo -e "\n"

echo -e "${BLUE}=== 测试完成 ===${NC}"
