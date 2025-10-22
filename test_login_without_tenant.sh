#!/bin/bash

# 测试不带租户ID的登录接口

BASE_URL="http://localhost:8080"

echo "=== 测试登录接口（不需要租户ID） ==="
echo ""

# 测试登录
echo "1. 测试登录（只需邮箱和密码）"
echo "请求："
cat <<EOF
{
  "email": "admin@system.local",
  "password": "Admin@123456"
}
EOF

echo ""
echo "响应："
curl -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "Admin@123456"
  }' | jq '.'

echo ""
echo ""

# 测试错误的密码
echo "2. 测试错误密码"
echo "请求："
cat <<EOF
{
  "email": "admin@system.local",
  "password": "wrongpassword"
}
EOF

echo ""
echo "响应："
curl -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "wrongpassword"
  }' | jq '.'

echo ""
echo ""

# 测试不存在的邮箱
echo "3. 测试不存在的邮箱"
echo "请求："
cat <<EOF
{
  "email": "notexist@example.com",
  "password": "password123"
}
EOF

echo ""
echo "响应："
curl -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "notexist@example.com",
    "password": "password123"
  }' | jq '.'

echo ""
echo "=== 测试完成 ==="
