# 摘要管理路由文档

## 概述

本文档描述了摘要管理相关的API路由配置。所有摘要路由都需要JWT认证和租户管理员权限。

## 路由列表

### 1. 生成摘要

**路径**: `POST /api/v1/summaries`

**功能**: 为指定会话生成对话摘要

**权限**: 需要JWT认证 + 租户管理员权限

**请求体**:

```json
{
  "sessionId": "uuid",
  "messageIds": ["uuid1", "uuid2"],
  "startMessageId": "uuid",
  "endMessageId": "uuid",
  "previousSummary": "string",
  "summaryType": "incremental|full",
  "targetLength": 500
}
```

**响应**:

```json
{
  "code": 200,
  "message": "摘要生成成功",
  "data": {
    "summaryId": "uuid",
    "summary": "string",
    "tokenCount": 150,
    "messageCount": 20,
    "startMessageId": "uuid",
    "endMessageId": "uuid",
    "qualityScore": 0.85,
    "compressionRate": 0.75,
    "keyTopics": ["topic1", "topic2"],
    "generationTime": 3500
  }
}
```

**相关需求**: 4.1, 4.2

---

### 2. 获取摘要详情

**路径**: `GET /api/v1/summaries/{id}`

**功能**: 获取指定摘要的详细信息

**权限**: 需要JWT认证 + 租户管理员权限

**路径参数**:

- `id`: 摘要ID (UUID)

**响应**:

```json
{
  "code": 200,
  "message": "获取摘要详情成功",
  "data": {
    "id": "uuid",
    "tenantId": "uuid",
    "sessionId": "uuid",
    "summaryType": "incremental",
    "content": "string",
    "tokenCount": 150,
    "messageCount": 20,
    "startMessageId": "uuid",
    "endMessageId": "uuid",
    "qualityScore": 0.85,
    "compressionRate": 0.75,
    "keyTopics": ["topic1", "topic2"],
    "previousSummaryId": "uuid",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**相关需求**: 4.1

---

### 3. 获取会话摘要列表

**路径**: `GET /api/v1/summaries/session/{sessionId}`

**功能**: 获取指定会话的所有摘要列表

**权限**: 需要JWT认证 + 租户管理员权限

**路径参数**:

- `sessionId`: 会话ID (UUID)

**查询参数**:

- `pageNo`: 页码（可选，默认1）
- `pageSize`: 每页大小（可选，默认10）

**响应**:

```json
{
  "code": 200,
  "message": "获取摘要列表成功",
  "data": {
    "data": [
      {
        "id": "uuid",
        "summaryType": "incremental",
        "content": "string",
        "tokenCount": 150,
        "messageCount": 20,
        "qualityScore": 0.85,
        "compressionRate": 0.75,
        "keyTopics": ["topic1", "topic2"],
        "createdAt": "2024-01-01T00:00:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 5,
    "totalPage": 1
  }
}
```

**相关需求**: 4.1

---

### 4. 检查摘要触发条件

**路径**: `POST /api/v1/summaries/check-trigger`

**功能**: 检查指定会话是否需要生成摘要

**权限**: 需要JWT认证 + 租户管理员权限

**请求体**:

```json
{
  "sessionId": "uuid",
  "checkMode": "auto|force"
}
```

**响应**:

```json
{
  "code": 200,
  "message": "检查完成",
  "data": {
    "shouldTrigger": true,
    "triggerScore": 0.85,
    "urgency": 0.75,
    "estimatedTokenSaving": 2000,
    "recommendedType": "incremental",
    "reasons": [
      "消息数量达到阈值",
      "Token使用率超过80%"
    ],
    "checkTime": 150
  }
}
```

**相关需求**: 4.2, 4.3

---

## 权限说明

### 租户隔离

所有摘要管理接口都实施严格的租户隔离：

- **租户管理员**: 只能访问自己租户下的摘要
- **平台管理员**: 可以访问所有租户的摘要

### 权限验证流程

1. JWT认证中间件验证用户身份
2. RBAC中间件验证用户角色（tenant_admin或system_admin）
3. Handler层从上下文获取用户信息
4. Service层验证资源是否属于当前租户

## 错误响应

### 401 Unauthorized

```json
{
  "code": 401,
  "message": "未授权：请先登录"
}
```

### 403 Forbidden

```json
{
  "code": 403,
  "message": "权限不足：无法访问其他租户的摘要"
}
```

### 404 Not Found

```json
{
  "code": 404,
  "message": "摘要不存在"
}
```

### 400 Bad Request

```json
{
  "code": 400,
  "message": "参数错误：sessionId不能为空"
}
```

### 500 Internal Server Error

```json
{
  "code": 500,
  "message": "服务器内部错误"
}
```

## 使用示例

### 生成摘要

```bash
curl -X POST http://localhost:8080/api/v1/summaries \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "123e4567-e89b-12d3-a456-426614174000",
    "summaryType": "incremental",
    "targetLength": 500
  }'
```

### 获取摘要详情

```bash
curl -X GET http://localhost:8080/api/v1/summaries/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer <jwt_token>"
```

### 获取会话摘要列表

```bash
curl -X GET "http://localhost:8080/api/v1/summaries/session/123e4567-e89b-12d3-a456-426614174000?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer <jwt_token>"
```

### 检查摘要触发条件

```bash
curl -X POST http://localhost:8080/api/v1/summaries/check-trigger \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "123e4567-e89b-12d3-a456-426614174000",
    "checkMode": "auto"
  }'
```

## 注意事项

1. **认证要求**: 所有接口都需要有效的JWT令牌
2. **权限要求**: 需要tenant_admin或system_admin角色
3. **租户隔离**: 租户管理员只能访问自己租户的数据
4. **参数验证**: 所有输入参数都会进行严格验证
5. **错误处理**: 统一的错误响应格式
6. **审计日志**: 所有操作都会记录审计日志

## 相关文档

- [摘要服务实现](../../service/session/summary_service.go)
- [摘要Handler实现](../handler/summary_handler.go)
- [Genkit摘要Flow](../../genkit/flows/summary_flows.go)
- [多租户访问控制规范](../../../.kiro/steering/multi-tenant-access-control.md)
