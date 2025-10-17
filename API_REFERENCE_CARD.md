# API 快速参考卡片

## 🌐 Swagger UI

**访问地址**: <http://localhost:8080/swagger/index.html>

## 📋 所有 API 接口 (8个)

### 🏢 模型提供商 (providers)

```
GET    /api/v1/providers
GET    /api/v1/providers/{providerId}
GET    /api/v1/providers/{providerId}/models
GET    /api/v1/providers/{providerId}/models/{modelId}
GET    /api/v1/providers/{providerId}/models/{modelId}/parameter-rules
```

### 💬 AI 对话 (chat)

```
POST   /api/v1/chat
POST   /api/v1/chat/abort
```

### ❤️ 健康检查 (health)

```
GET    /api/v1/health
```

## 🚀 快速开始

```bash
# 生成文档
make swagger

# 启动服务
make run

# 访问 Swagger UI
open http://localhost:8080/swagger/index.html
```

## 📝 请求示例

### 发送对话

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好",
    "sessionId": "session-123",
    "options": {
      "temperature": 0.7,
      "maxTokens": 2048
    }
  }'
```

### 获取提供商列表

```bash
curl http://localhost:8080/api/v1/providers
```

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

## 📖 文档资源

- [完整使用指南](docs/swagger-guide.md)
- [快速开始](docs/SWAGGER_QUICKSTART_CN.md)
- [更新总结](SWAGGER_UPDATE_SUMMARY.md)

---

**提示**: 使用 Swagger UI 可以直接在浏览器中测试所有接口！
