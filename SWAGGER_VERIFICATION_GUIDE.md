# Swagger 文档验证指南

## 访问 Swagger UI

启动服务后，可以通过以下 URL 访问 Swagger 文档：

### Swagger UI

```
http://localhost:8080/swagger/index.html
```

### Swagger JSON

```
http://localhost:8080/swagger/doc.json
```

### Swagger YAML

```
http://localhost:8080/swagger/doc.yaml
```

## 验证 modelName 字段

在 Swagger UI 中，可以验证以下接口的 `modelName` 字段：

### 1. POST /api/v1/chat

发送对话消息接口，请求体中的 `options.modelName` 字段：

```json
{
  "message": "你好",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.7,
    "maxTokens": 2048
  }
}
```

### 2. POST /api/v1/chat/sessions/{id}/messages

向会话发送消息接口，请求体中的 `options.modelName` 字段：

```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "message": "你好",
  "options": {
    "modelName": "gpt-4",
    "temperature": 0.7
  }
}
```

## 字段说明验证

在 Swagger UI 的 Models 部分，找到 `ChatOptions` 模型，应该能看到：

- **modelName** (string, optional)
  - 描述：指定要使用的AI模型名称，如 "gpt-4"、"gemini-pro"、"qwen-turbo" 等
  - 说明：系统会根据当前租户ID和模型名称从 model_configurations 表中查询配置
  - 默认行为：如果不指定，将使用会话的默认模型
  - 验证规则：最小长度1，最大长度128
  - 示例：gpt-4

## 启动服务验证

```bash
# 启动服务
make run

# 或者
go run cmd/server/main.go
```

启动后，在日志中应该能看到：

```
Swagger UI 已启用
  swagger_ui: http://localhost:8080/swagger/index.html
  swagger_json: http://localhost:8080/swagger/doc.json
  swagger_yaml: http://localhost:8080/swagger/doc.yaml
```

## 相关文档

- [Swagger 使用指南](docs/swagger-guide.md)
- [多模型支持配置指南](docs/MULTI_PROVIDER_GUIDE.md)
- [迁移指南](docs/MIGRATION_GUIDE.md)
