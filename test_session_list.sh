#!/bin/bash

# 测试会话列表接口

BASE_URL="http://localhost:8080/api/v1"

echo "=== 测试会话列表接口 ==="
echo ""

# 测试1: 缺少用户ID（应返回401）
echo "测试1: 缺少用户ID"
curl -X GET "${BASE_URL}/chat/sessions?pageNo=1&pageSize=10" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s | jq .
echo ""

# 测试2: 无效的用户ID格式（应返回400）
echo "测试2: 无效的用户ID格式"
curl -X GET "${BASE_URL}/chat/sessions?pageNo=1&pageSize=10" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: invalid-user-id" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s | jq .
echo ""

# 测试3: 使用有效的UUID（需要先创建一个用户）
echo "测试3: 使用有效的UUID"
# 生成一个有效的UUID
VALID_UUID="550e8400-e29b-41d4-a716-446655440000"
curl -X GET "${BASE_URL}/chat/sessions?pageNo=1&pageSize=10" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: ${VALID_UUID}" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s | jq .
echo ""

echo "=== 测试完成 ==="
