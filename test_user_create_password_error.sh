#!/bin/bash

# 测试用户创建接口的密码验证错误返回

BASE_URL="http://localhost:8080/api/v1"

echo "=========================================="
echo "测试用户创建接口 - 密码验证错误"
echo "=========================================="

# 1. 先登录获取 token
echo -e "\n1. 登录获取 token..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "Admin123!@#"
  }')

echo "登录响应: $LOGIN_RESPONSE"

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，无法获取 token"
  exit 1
fi

echo "✅ 登录成功，获取到 token"

# 2. 测试创建用户 - 密码强度不足（只有小写字母）
echo -e "\n2. 测试创建用户 - 密码强度不足（只有小写字母）..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenantId": "f98c705a-3bb4-4551-9568-7a607fb25195",
    "email": "test_weak_password@example.com",
    "password": "weakpassword",
    "displayName": "测试用户"
  }')

echo "响应: $RESPONSE"

# 检查是否返回了正确的错误信息
if echo "$RESPONSE" | grep -q "密码"; then
  echo "✅ 成功返回密码相关错误信息"
else
  echo "❌ 未返回密码相关错误信息"
fi

# 3. 测试创建用户 - 密码强度不足（缺少特殊字符）
echo -e "\n3. 测试创建用户 - 密码强度不足（缺少特殊字符）..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenantId": "f98c705a-3bb4-4551-9568-7a607fb25195",
    "email": "test_no_special@example.com",
    "password": "Password123",
    "displayName": "测试用户"
  }')

echo "响应: $RESPONSE"

if echo "$RESPONSE" | grep -q "密码"; then
  echo "✅ 成功返回密码相关错误信息"
else
  echo "❌ 未返回密码相关错误信息"
fi

# 4. 测试创建用户 - 密码太短
echo -e "\n4. 测试创建用户 - 密码太短..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenantId": "f98c705a-3bb4-4551-9568-7a607fb25195",
    "email": "test_short@example.com",
    "password": "Abc1!",
    "displayName": "测试用户"
  }')

echo "响应: $RESPONSE"

if echo "$RESPONSE" | grep -q "密码"; then
  echo "✅ 成功返回密码相关错误信息"
else
  echo "❌ 未返回密码相关错误信息"
fi

# 5. 测试创建用户 - 正确的密码
echo -e "\n5. 测试创建用户 - 正确的密码..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenantId": "f98c705a-3bb4-4551-9568-7a607fb25195",
    "email": "test_valid_password@example.com",
    "password": "ValidPass123!@#",
    "displayName": "测试用户"
  }')

echo "响应: $RESPONSE"

if echo "$RESPONSE" | grep -q '"code":201'; then
  echo "✅ 使用正确密码创建用户成功"
else
  echo "❌ 使用正确密码创建用户失败"
fi

echo -e "\n=========================================="
echo "测试完成"
echo "=========================================="
