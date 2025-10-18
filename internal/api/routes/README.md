# API 路由文档

本目录包含所有 API 路由的注册逻辑。

## 路由组织结构

### 1. 模型提供商路由 (provider_routes.go)

提供模型提供商和模型信息的查询接口。

| 方法 | 路径 | 描述 | Handler |
|------|------|------|---------|
| GET | /api/v1/providers | 获取所有提供商列表 | GetProviders |
| GET | /api/v1/providers/{providerId} | 获取提供商详情 | GetProviderByID |
| GET | /api/v1/providers/{providerId}/models | 获取提供商的所有模型 | GetProviderModels |
| GET | /api/v1/providers/{providerId}/models/{modelId} | 获取指定模型详情 | GetProviderModel |
| GET | /api/v1/providers/{providerId}/models/{modelId}/parameter-rules | 获取模型参数规则 | GetModelParameterRules |

### 2. 会话管理路由 (session_routes.go)

提供会话和消息管理的完整功能。

#### 会话管理接口

| 方法 | 路径 | 描述 | Handler |
|------|------|------|---------|
| POST | /api/v1/chat/sessions | 创建新会话 | CreateSession |
| GET | /api/v1/chat/sessions | 获取会话列表（支持分页和过滤） | ListSessions |
| GET | /api/v1/chat/sessions/search | 搜索会话 | SearchSessions |
| GET | /api/v1/chat/sessions/{id} | 获取会话详情 | GetSession |
| PATCH | /api/v1/chat/sessions/{id} | 更新会话 | UpdateSession |
| DELETE | /api/v1/chat/sessions/{id} | 删除会话（软删除） | DeleteSession |
| POST | /api/v1/chat/sessions/{id}/pin | 置顶/取消置顶会话 | PinSession |
| POST | /api/v1/chat/sessions/{id}/archive | 归档/取消归档会话 | ArchiveSession |

#### 消息管理接口

| 方法 | 路径 | 描述 | Handler |
|------|------|------|---------|
| POST | /api/v1/chat/sessions/{id}/messages | 在会话中发送消息 | SendMessage |
| GET | /api/v1/chat/sessions/{id}/messages | 获取会话的消息历史（支持分页） | GetMessages |
| GET | /api/v1/chat/messages/{id} | 获取单条消息详情 | GetMessageByID |
| POST | /api/v1/chat/messages/{id}/abort | 中止消息生成 | AbortMessage |

### 3. AI 对话路由 (在 main.go 中直接注册)

提供基础的 AI 对话功能（遗留接口）。

| 方法 | 路径 | 描述 | Handler |
|------|------|------|---------|
| POST | /api/v1/chat | 发送对话消息 | HandleChat |
| POST | /api/v1/chat/abort | 中止对话生成 | HandleAbort |

### 4. 健康检查路由 (在 main.go 中直接注册)

提供服务健康状态检查。

| 方法 | 路径 | 描述 | Handler |
|------|------|------|---------|
| GET | /api/v1/health | 健康检查 | Handle |

### 5. Swagger 文档路由 (在 main.go 中直接注册)

提供 API 文档界面。

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /swagger/* | Swagger UI 文档界面 |

## 路由注册顺序

路由注册顺序很重要，特别是对于可能产生冲突的路径模式：

1. **更具体的路径优先**：例如 `/api/v1/chat/sessions/search` 必须在 `/api/v1/chat/sessions/{id}` 之前注册
2. **带参数的路径后注册**：例如 `/api/v1/chat/sessions/{id}` 应该在所有固定路径之后注册
3. **操作路径最后注册**：例如 `/api/v1/chat/sessions/{id}/pin` 在基础 CRUD 路由之后注册

## 中间件应用

所有路由都会经过以下中间件（按顺序）：

1. **Recovery**: 捕获 panic 并返回 500 错误
2. **Logger**: 记录请求日志
3. **CORS**: 处理跨域请求

## 认证和授权

当前实现使用临时的用户ID提取方式（从 `X-User-ID` 请求头）。

**TODO**: 实现完整的认证中间件：

- JWT Token 验证
- 用户身份提取
- 权限验证

## 使用示例

### 创建会话

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "title": "我的第一个会话",
    "modelName": "gemini-1.5-pro",
    "systemPrompt": "你是一个有帮助的AI助手",
    "temperature": 0.7
  }'
```

### 发送消息

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions/{session-id}/messages \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "message": "你好，请介绍一下自己"
  }'
```

### 获取会话列表

```bash
curl -X GET "http://localhost:8080/api/v1/chat/sessions?pageNo=1&pageSize=20" \
  -H "X-User-ID: user-123"
```

### 搜索会话

```bash
curl -X GET "http://localhost:8080/api/v1/chat/sessions/search?keyword=AI&pageNo=1&pageSize=20" \
  -H "X-User-ID: user-123"
```

## 错误处理

所有接口都遵循统一的错误响应格式：

```json
{
  "code": 40401,
  "message": "会话不存在",
  "data": null
}
```

常见错误码：

- `400`: 请求参数错误
- `401`: 未认证
- `403`: 无权访问
- `404`: 资源不存在
- `422`: 参数验证失败
- `500`: 服务器内部错误

## 响应格式

### 普通响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 业务数据
  }
}
```

### 分页响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      // 数据列表
    ],
    "pageNo": 1,
    "pageSize": 20,
    "totalCount": 100,
    "totalPage": 5
  }
}
```

## 注意事项

1. **路径参数提取**：使用 Go 1.22+ 的新路由模式，路径参数通过 `{id}` 语法定义
2. **HTTP 方法限制**：每个路由都明确指定了允许的 HTTP 方法（如 `GET`, `POST`, `PATCH`, `DELETE`）
3. **内容类型**：所有接口都使用 `application/json` 格式
4. **用户隔离**：会话管理接口都会验证用户权限，确保用户只能访问自己的数据
5. **软删除**：删除操作使用软删除，数据不会被物理删除

## 扩展指南

### 添加新路由

1. 在相应的 `*_routes.go` 文件中添加路由注册
2. 确保路由注册顺序正确
3. 更新本文档的路由表
4. 添加 Swagger 注释到 Handler 方法
5. 运行 `swag init` 更新 Swagger 文档

### 添加新的路由组

1. 创建新的 `*_routes.go` 文件
2. 实现 `Register*Routes` 函数
3. 在 `main.go` 中调用注册函数
4. 更新本文档添加新的路由组说明
