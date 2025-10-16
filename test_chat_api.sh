#!/bin/bash

# 测试聊天 API 接口

echo "测试聊天 API..."
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下你自己",
    "sessionId": "test-session-001"
  }'

echo -e "\n\n测试完成"
