#!/bin/bash

# Swagger 演示脚本
# 展示如何使用 Swagger 文档和 API

echo "=========================================="
echo "🚀 Swagger API 文档演示"
echo "=========================================="
echo ""

# 检查服务器是否运行
echo "📡 检查服务器状态..."
if curl -s http://localhost:8080/api/v1/providers > /dev/null 2>&1; then
    echo "✅ 服务器正在运行"
else
    echo "❌ 服务器未运行"
    echo ""
    echo "请先启动服务器："
    echo "  make run"
    echo ""
    echo "或者："
    echo "  ./bin/server"
    exit 1
fi

echo ""
echo "=========================================="
echo "📚 Swagger UI 访问地址"
echo "=========================================="
echo ""
echo "🌐 Swagger UI: http://localhost:8080/swagger/index.html"
echo "📄 OpenAPI JSON: http://localhost:8080/swagger/doc.json"
echo ""

echo "=========================================="
echo "🧪 API 接口演示"
echo "=========================================="
echo ""

# 演示 1: 获取所有提供商
echo "1️⃣  获取所有提供商列表"
echo "   GET /api/v1/providers"
echo ""
curl -s http://localhost:8080/api/v1/providers | jq '.data[] | {id, provider, label}' 2>/dev/null || curl -s http://localhost:8080/api/v1/providers
echo ""
echo ""

# 演示 2: 获取 Gemini 提供商详情
echo "2️⃣  获取 Gemini 提供商详情"
echo "   GET /api/v1/providers/gemini"
echo ""
curl -s http://localhost:8080/api/v1/providers/gemini | jq '.data | {id, provider, label, background}' 2>/dev/null || curl -s http://localhost:8080/api/v1/providers/gemini
echo ""
echo ""

# 演示 3: 获取 Gemini 的模型列表
echo "3️⃣  获取 Gemini 的模型列表"
echo "   GET /api/v1/providers/gemini/models"
echo ""
curl -s http://localhost:8080/api/v1/providers/gemini/models | jq '.data[] | {model, model_type, label}' 2>/dev/null | head -20 || curl -s http://localhost:8080/api/v1/providers/gemini/models | head -20
echo ""
echo ""

# 演示 4: 获取特定模型详情
echo "4️⃣  获取 Gemini 1.5 Flash 模型详情"
echo "   GET /api/v1/providers/gemini/models/gemini-1.5-flash"
echo ""
curl -s http://localhost:8080/api/v1/providers/gemini/models/gemini-1.5-flash | jq '.data | {model, model_type, features, model_properties}' 2>/dev/null || curl -s http://localhost:8080/api/v1/providers/gemini/models/gemini-1.5-flash
echo ""
echo ""

# 演示 5: 获取模型参数规则
echo "5️⃣  获取模型参数规则"
echo "   GET /api/v1/providers/gemini/models/gemini-1.5-flash/parameter-rules"
echo ""
curl -s http://localhost:8080/api/v1/providers/gemini/models/gemini-1.5-flash/parameter-rules | jq '.data[] | {name, type, required, default}' 2>/dev/null || curl -s http://localhost:8080/api/v1/providers/gemini/models/gemini-1.5-flash/parameter-rules
echo ""
echo ""

echo "=========================================="
echo "💡 使用提示"
echo "=========================================="
echo ""
echo "1. 在浏览器中打开 Swagger UI 查看完整文档"
echo "   http://localhost:8080/swagger/index.html"
echo ""
echo "2. 使用 Swagger UI 的 'Try it out' 功能测试 API"
echo ""
echo "3. 查看数据模型定义（在页面底部的 Schemas 部分）"
echo ""
echo "4. 使用 curl 或其他工具调用 API："
echo "   curl http://localhost:8080/api/v1/providers"
echo ""
echo "=========================================="
echo "✅ 演示完成！"
echo "=========================================="
