#!/bin/bash

# 测试流式聊天修复

echo "=== 测试流式聊天修复 ==="
echo ""

# 配置
API_BASE="http://localhost:8080/api"
SESSION_ID="test-session-$(date +%s)"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}步骤 1: 创建测试会话${NC}"
CREATE_RESPONSE=$(curl -s -X POST "${API_BASE}/chat/sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "流式测试会话"
  }')

SESSION_ID=$(echo $CREATE_RESPONSE | jq -r '.data.id')
echo "会话ID: $SESSION_ID"
echo ""

if [ "$SESSION_ID" = "null" ] || [ -z "$SESSION_ID" ]; then
  echo -e "${RED}创建会话失败${NC}"
  echo $CREATE_RESPONSE | jq '.'
  exit 1
fi

echo -e "${GREEN}✓ 会话创建成功${NC}"
echo ""

echo -e "${YELLOW}步骤 2: 发送流式消息${NC}"
echo "发送消息: '介绍一下你自己'"
echo ""

# 使用 curl 发送流式请求并保存响应
STREAM_OUTPUT=$(mktemp)
curl -s -N -X POST "${API_BASE}/chat/sessions/${SESSION_ID}/messages/stream" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "介绍一下你自己"
  }' > "$STREAM_OUTPUT"

echo "流式响应内容:"
cat "$STREAM_OUTPUT"
echo ""
echo ""

# 检查是否有错误
if grep -q '"event":"error"' "$STREAM_OUTPUT"; then
  echo -e "${RED}✗ 流式响应包含错误${NC}"
  echo "错误详情:"
  grep '"event":"error"' "$STREAM_OUTPUT" | jq '.'
  rm "$STREAM_OUTPUT"
  exit 1
fi

# 检查是否有重复主键错误
if grep -q "duplicate key value violates unique constraint" "$STREAM_OUTPUT"; then
  echo -e "${RED}✗ 检测到主键冲突错误（问题未修复）${NC}"
  rm "$STREAM_OUTPUT"
  exit 1
fi

# 检查是否有完成事件
if grep -q '"event":"done"' "$STREAM_OUTPUT"; then
  echo -e "${GREEN}✓ 流式响应正常完成${NC}"
else
  echo -e "${YELLOW}⚠ 未检测到完成事件${NC}"
fi

# 统计事件类型
echo ""
echo "事件统计:"
echo "- user_message 事件: $(grep -c '"event":"user_message"' "$STREAM_OUTPUT" || echo 0)"
echo "- content 事件: $(grep -c '"event":"content"' "$STREAM_OUTPUT" || echo 0)"
echo "- done 事件: $(grep -c '"event":"done"' "$STREAM_OUTPUT" || echo 0)"
echo "- error 事件: $(grep -c '"event":"error"' "$STREAM_OUTPUT" || echo 0)"

rm "$STREAM_OUTPUT"
echo ""

echo -e "${YELLOW}步骤 3: 验证消息已保存${NC}"
MESSAGES_RESPONSE=$(curl -s -X GET "${API_BASE}/chat/sessions/${SESSION_ID}/messages?pageNo=1&pageSize=10")

MESSAGE_COUNT=$(echo $MESSAGES_RESPONSE | jq -r '.data.totalCount')
echo "消息总数: $MESSAGE_COUNT"

if [ "$MESSAGE_COUNT" = "2" ]; then
  echo -e "${GREEN}✓ 消息保存正确（用户消息 + AI消息）${NC}"
else
  echo -e "${RED}✗ 消息数量不正确，期望 2，实际 $MESSAGE_COUNT${NC}"
  echo $MESSAGES_RESPONSE | jq '.'
  exit 1
fi

echo ""
echo -e "${GREEN}=== 所有测试通过 ===${NC}"
