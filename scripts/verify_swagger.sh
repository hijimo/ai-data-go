#!/bin/bash

# Swagger 文档验证脚本

echo "🔍 验证 Swagger 文档..."
echo ""

# 检查文件是否存在
if [ ! -f "docs/swagger.json" ]; then
    echo "❌ 错误: docs/swagger.json 不存在"
    echo "请先运行: make swagger"
    exit 1
fi

if [ ! -f "docs/swagger.yaml" ]; then
    echo "❌ 错误: docs/swagger.yaml 不存在"
    echo "请先运行: make swagger"
    exit 1
fi

echo "✅ Swagger 文件存在"
echo ""

# 检查关键 API 端点
echo "📋 检查关键 API 端点..."
echo ""

endpoints=(
    "/api/v1/auth/register"
    "/api/v1/auth/login"
    "/api/v1/auth/refresh"
    "/api/v1/audit/auth"
    "/api/v1/users"
    "/api/v1/tenants"
    "/chat/sessions"
)

for endpoint in "${endpoints[@]}"; do
    if grep -q "\"$endpoint\"" docs/swagger.json; then
        echo "  ✅ $endpoint"
    else
        echo "  ❌ $endpoint (未找到)"
    fi
done

echo ""

# 检查响应类型定义
echo "📦 检查响应类型定义..."
echo ""

response_types=(
    "AuthAuditListResponse"
    "UserDataResponse"
    "TenantDataResponse"
    "LoginDataResponse"
    "SessionDataResponse"
    "AnyDataResponse"
    "UserListResponse"
    "TenantListResponse"
    "SessionListResponse"
)

for type in "${response_types[@]}"; do
    if grep -q "\"genkit-ai-service_internal_model.$type\"" docs/swagger.json; then
        echo "  ✅ $type"
    else
        echo "  ❌ $type (未找到)"
    fi
done

echo ""

# 统计信息
echo "📊 统计信息..."
echo ""

total_paths=$(grep -c '"/' docs/swagger.json || echo "0")
total_definitions=$(grep -c '"genkit-ai-service_internal_model\.' docs/swagger.json || echo "0")

echo "  API 路径数量: $total_paths"
echo "  类型定义数量: $total_definitions"
echo ""

# 文件大小
echo "📁 文件大小..."
echo ""
ls -lh docs/swagger.json docs/swagger.yaml docs/docs.go | awk '{print "  " $9 ": " $5}'
echo ""

echo "✅ Swagger 文档验证完成！"
echo ""
echo "💡 提示："
echo "  - 启动服务器: make run"
echo "  - 访问 Swagger UI: http://localhost:8080/swagger/index.html"
