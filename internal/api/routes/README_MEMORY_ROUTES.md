# 记忆管理路由文档

## 概述

记忆管理路由提供了对话记忆的检索、存储、清理和查询功能。所有路由都需要 JWT 认证和租户管理员权限。

## 路由列表

### 1. 检索记忆

**路径**: `POST /api/v1/memories/search`

**权限**: 租户管理员（tenant_admin）

**功能**: 基于向量相似度检索相关的历史对话记忆

**请求体**:

```json
{
  "sessionId": "uuid",
  "query": "用户查询文本",
  "topK": 5,
  "minSimilarity": 0.7,
  "timeRangeDays": 30,
  "memoryTypes": ["fact", "preference", "context"],
  "includeCrossSessions": false
}
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [
      {
        "id": "uuid",
        "sessionId": "uuid",
        "memoryType": "fact",
        "content": "记忆内容",
        "importance": 0.8,
        "similarity": 0.85,
        "score": 0.68,
        "accessCount": 5,
        "lastAccessedAt": "2024-01-01T00:00:00Z",
        "createdAt": "2024-01-01T00:00:00Z",
        "metadata": {}
      }
    ],
    "totalCount": 10,
    "query": "用户查询文本",
    "searchTime": 150
  }
}
```

### 2. 存储记忆

**路径**: `POST /api/v1/memories`

**权限**: 租户管理员（tenant_admin）

**功能**: 将对话消息转换为长期记忆并存储

**请求体**:

```json
{
  "sessionId": "uuid",
  "messageIds": ["uuid1", "uuid2"],
  "memoryType": "fact",
  "content": "记忆内容",
  "importance": 0.8,
  "expirationDays": 365,
  "metadata": {
    "source": "conversation",
    "tags": ["important"]
  }
}
```

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "uuid",
    "sessionId": "uuid",
    "memoryType": "fact",
    "content": "记忆内容",
    "importance": 0.8,
    "expiresAt": "2025-01-01T00:00:00Z",
    "createdAt": "2024-01-01T00:00:00Z",
    "metadata": {}
  }
}
```

### 3. 清理记忆

**路径**: `POST /api/v1/memories/cleanup`

**权限**: 租户管理员（tenant_admin）

**功能**: 按策略清理过期或低质量的记忆

**请求体**:

```json
{
  "sessionId": "uuid",
  "strategy": "expired",
  "mode": "soft",
  "batchSize": 100,
  "execute": true
}
```

**参数说明**:

- `strategy`: 清理策略
  - `expired`: 清理已过期的记忆
  - `low_quality`: 清理低质量记忆（重要性低且访问次数少）
  - `unused`: 清理长期未访问的记忆
  - `all`: 清理所有符合条件的记忆
- `mode`: 清理模式
  - `soft`: 软删除（标记为已删除）
  - `hard`: 硬删除（物理删除）
- `execute`: 是否执行清理（false 时仅预览）

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "cleanedCount": 50,
    "freedSpace": 1024000,
    "details": [
      {
        "memoryId": "uuid",
        "reason": "已过期",
        "size": 2048,
        "createdAt": "2023-01-01T00:00:00Z",
        "lastAccess": "2023-06-01T00:00:00Z"
      }
    ],
    "preview": false,
    "strategy": "expired",
    "mode": "soft"
  }
}
```

### 4. 获取记忆详情

**路径**: `GET /api/v1/memories/{id}`

**权限**: 租户管理员（tenant_admin）

**功能**: 获取指定记忆的详细信息

**路径参数**:

- `id`: 记忆ID（UUID）

**响应**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "uuid",
    "sessionId": "uuid",
    "memoryType": "fact",
    "content": "记忆内容",
    "importance": 0.8,
    "accessCount": 10,
    "lastAccessedAt": "2024-01-01T00:00:00Z",
    "expiresAt": "2025-01-01T00:00:00Z",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z",
    "metadata": {}
  }
}
```

## 记忆类型

系统支持以下记忆类型：

- `fact`: 事实性信息（如用户提到的具体数据、日期等）
- `preference`: 用户偏好（如喜好、习惯等）
- `context`: 上下文信息（如当前讨论的主题）
- `event`: 事件记录（如重要的对话节点）
- `summary`: 摘要信息（如对话总结）

## 权限控制

所有记忆管理路由都实施严格的多租户隔离：

1. **JWT 认证**: 所有请求必须携带有效的 JWT token
2. **角色验证**: 只有租户管理员（tenant_admin）可以访问
3. **租户隔离**:
   - 租户管理员只能访问自己租户的记忆
   - 平台管理员可以访问所有租户的记忆
4. **会话验证**: 验证会话是否属于当前租户

## 错误响应

### 401 Unauthorized

```json
{
  "code": 401,
  "message": "缺少用户认证信息"
}
```

### 403 Forbidden

```json
{
  "code": 403,
  "message": "权限不足：无法访问其他租户的记忆"
}
```

### 400 Bad Request

```json
{
  "code": 400,
  "message": "无效的请求参数"
}
```

### 422 Validation Error

```json
{
  "code": 422,
  "message": "参数验证失败",
  "data": {
    "errors": [
      {
        "field": "sessionId",
        "message": "sessionId 必须是有效的 UUID"
      }
    ]
  }
}
```

### 500 Internal Server Error

```json
{
  "code": 500,
  "message": "内部服务器错误"
}
```

## 使用示例

### 检索记忆示例

```bash
curl -X POST http://localhost:8080/api/v1/memories/search \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "123e4567-e89b-12d3-a456-426614174000",
    "query": "用户的偏好是什么",
    "topK": 5,
    "minSimilarity": 0.7,
    "memoryTypes": ["preference"]
  }'
```

### 存储记忆示例

```bash
curl -X POST http://localhost:8080/api/v1/memories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "123e4567-e89b-12d3-a456-426614174000",
    "memoryType": "preference",
    "content": "用户喜欢简洁的回答",
    "importance": 0.8,
    "expirationDays": 365
  }'
```

### 清理记忆示例

```bash
curl -X POST http://localhost:8080/api/v1/memories/cleanup \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "expired",
    "mode": "soft",
    "execute": true
  }'
```

### 获取记忆详情示例

```bash
curl -X GET http://localhost:8080/api/v1/memories/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 性能考虑

1. **向量检索**: 使用 Qdrant 向量数据库进行高效的相似度搜索
2. **缓存**: 常用记忆会被缓存以提高响应速度
3. **批量处理**: 清理操作支持批量处理以优化性能
4. **异步更新**: 访问统计异步更新，不影响主流程

## 安全注意事项

1. **租户隔离**: 所有查询都强制包含租户ID过滤
2. **参数验证**: 严格验证所有输入参数
3. **审计日志**: 记录所有权限验证失败的尝试
4. **敏感信息**: 记忆内容可能包含敏感信息，需要适当的访问控制

## 相关文档

- [上下文管理路由](./README_CONTEXT_ROUTES.md)
- [摘要管理路由](./README_SUMMARY_ROUTES.md)
- [记忆管理 Flow](../../genkit/flows/MEMORY_FLOWS_README.md)
- [多租户访问控制规范](../../../.kiro/steering/multi-tenant-access-control.md)
