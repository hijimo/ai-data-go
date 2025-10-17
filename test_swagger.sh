#!/bin/bash

# 测试 Swagger UI 集成

echo "=========================================="
echo "测试 Swagger UI 集成"
echo "=========================================="
echo ""

# 启动服务器（后台运行）
echo "1. 启动服务器..."
./bin/server &
SERVER_PID=$!

# 等待服务器启动
echo "2. 等待服务器启动..."
sleep 3

# 测试 Swagger JSON 端点
echo ""
echo "3. 测试 Swagger JSON 端点..."
echo "GET http://localhost:8080/swagger/doc.json"
curl -s http://localhost:8080/swagger/doc.json | head -20
echo ""

# 测试 Swagger UI 页面
echo ""
echo "4. 测试 Swagger UI 页面..."
echo "GET http://localhost:8080/swagger/index.html"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/swagger/index.html)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Swagger UI 页面访问成功 (HTTP $HTTP_CODE)"
else
    echo "❌ Swagger UI 页面访问失败 (HTTP $HTTP_CODE)"
fi

# 测试 API 端点
echo ""
echo "5. 测试 API 端点..."
echo "GET http://localhost:8080/api/v1/providers"
curl -s http://localhost:8080/api/v1/providers | jq '.' | head -30
echo ""

# 停止服务器
echo ""
echo "6. 停止服务器..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo ""
echo "=========================================="
echo "✅ 测试完成！"
echo ""
echo "访问 Swagger UI: http://localhost:8080/swagger/index.html"
echo "=========================================="
