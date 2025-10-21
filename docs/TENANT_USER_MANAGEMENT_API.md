# 租户用户管理 API 文档

## 概述

本文档描述了租户用户管理 API 的使用方法。这些 API 允许租户管理员和平台管理员管理特定租户下的用户。

## 权限要求

所有租户用户管理 API 都需要以下权限之一：

- **租户管理员** (`tenant_admin`)：只能管理自己所属租户的用户
- **平台管理员** (`system_admin`)：可以管理所有租户的用户

## API 端点

### 1. 在租户下创建用户

在指定租户下创建新用户。

**端点:** `POST /api/v1/tenants/{tenantId}/users`

**权限:** 需要租户管理员或平台管理员权限

**路径参数:**

- `tenantId` (string, required): 租户ID

**请求体:**

```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "displayName": "张三",
  "phone": "+8613800138000",
  "roles": ["user"],
  "isAdmin": false,
  "meta": {
    "department": "技术部"
  }
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "创建用户成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenantId": "660e8400-e29b-41d4-a716-446655440001",
    "email": "user@example.com",
    "displayName": "张三",
    "phone": "+8613800138000",
    "isActive": true,
    "isAdmin": false,
    "roles": ["user"],
    "createdAt": "2025-01-20T10:00:00Z"
  }
}
```

### 2. 获取租户用户列表

获取指定租户下的用户列表，支持分页。

**端点:** `GET /api/v1/tenants/{tenantId}/users`

**权限:** 需要租户管理员或平台管理员权限

**路径参数:**

- `tenantId` (string, required): 租户ID

**查询参数:**

- `pageNo` (int, optional): 页码，默认为 1
- `pageSize` (int, optional): 每页大小，默认为 20，最大为 100

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "tenantId": "660e8400-e29b-41d4-a716-446655440001",
        "email": "admin@example.com",
        "displayName": "管理员",
        "isActive": true,
        "isAdmin": true,
        "roles": ["tenant_admin"]
      },
      {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "tenantId": "660e8400-e29b-41d4-a716-446655440001",
        "email": "user@example.com",
        "displayName": "张三",
        "isActive": true,
        "isAdmin": false,
        "roles": ["user"]
      }
    ],
    "pageNo": 1,
    "pageSize": 20,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

### 3. 更新用户状态

启用或禁用指定租户下的用户。禁用用户时，系统会自动撤销该用户的所有有效 Token。

**端点:** `PATCH /api/v1/tenants/{tenantId}/users/{userId}/status`

**权限:** 需要租户管理员或平台管理员权限

**路径参数:**

- `tenantId` (string, required): 租户ID
- `userId` (string, required): 用户ID

**请求体:**

```json
{
  "isActive": false
}
```

**响应示例:**

```json
{
  "code": 200,
  "message": "更新用户状态成功",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "tenantId": "660e8400-e29b-41d4-a716-446655440001",
    "email": "user@example.com",
    "displayName": "张三",
    "isActive": false,
    "isAdmin": false,
    "roles": ["user"]
  }
}
```

### 4. 删除用户

软删除指定租户下的用户。

**端点:** `DELETE /api/v1/tenants/{tenantId}/users/{userId}`

**权限:** 需要租户管理员或平台管理员权限

**路径参数:**

- `tenantId` (string, required): 租户ID
- `userId` (string, required): 用户ID

**响应示例:**

```json
{
  "code": 200,
  "message": "删除用户成功",
  "data": {}
}
```

## 错误响应

所有 API 在出错时都会返回统一的错误响应格式：

```json
{
  "code": 400,
  "message": "错误描述信息"
}
```

### 常见错误码

- `400` - 请求参数错误
- `401` - 未认证（需要登录）
- `403` - 权限不足
- `404` - 资源不存在
- `422` - 参数验证失败
- `500` - 服务器内部错误

## 使用示例

### 示例 1: 租户管理员创建用户

```bash
# 1. 租户管理员登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@acme.com",
    "password": "AdminPassword123!"
  }'

# 响应中获取 accessToken

# 2. 在自己的租户下创建用户
curl -X POST http://localhost:8080/api/v1/tenants/660e8400-e29b-41d4-a716-446655440001/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {accessToken}" \
  -d '{
    "email": "newuser@acme.com",
    "password": "UserPassword123!",
    "displayName": "新用户",
    "roles": ["user"]
  }'
```

### 示例 2: 平台管理员管理任意租户的用户

```bash
# 1. 平台管理员登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@system.local",
    "password": "PlatformAdminPassword123!"
  }'

# 2. 查看任意租户的用户列表
curl -X GET "http://localhost:8080/api/v1/tenants/660e8400-e29b-41d4-a716-446655440001/users?pageNo=1&pageSize=20" \
  -H "Authorization: Bearer {accessToken}"

# 3. 禁用任意租户的用户
curl -X PATCH http://localhost:8080/api/v1/tenants/660e8400-e29b-41d4-a716-446655440001/users/770e8400-e29b-41d4-a716-446655440002/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {accessToken}" \
  -d '{
    "isActive": false
  }'
```

### 示例 3: 租户管理员尝试访问其他租户（会失败）

```bash
# 租户 A 的管理员尝试访问租户 B 的用户
curl -X GET "http://localhost:8080/api/v1/tenants/other-tenant-id/users" \
  -H "Authorization: Bearer {tenantAAdminToken}"

# 响应：
# {
#   "code": 403,
#   "message": "权限不足：无法查看该租户的用户列表"
# }
```

## 权限验证逻辑

### 租户管理员

租户管理员只能管理自己所属租户的用户：

1. 系统验证用户的角色包含 `tenant_admin`
2. 系统验证用户的 `tenant_id` 与请求路径中的 `tenantId` 匹配
3. 如果两个条件都满足，允许访问；否则返回 403 错误

### 平台管理员

平台管理员可以管理所有租户的用户：

1. 系统验证用户的角色包含 `system_admin`
2. 如果包含，允许访问任意租户的数据

## 注意事项

1. **租户隔离**: 租户管理员只能管理自己租户下的用户，无法访问其他租户的数据
2. **Token 撤销**: 禁用用户时，系统会自动撤销该用户的所有有效 Refresh Token
3. **软删除**: 删除用户使用软删除机制，数据不会真正从数据库中删除
4. **密码强度**: 创建用户时，密码必须满足强度要求（至少 8 个字符）
5. **邮箱唯一性**: 邮箱在租户内必须唯一，但不同租户可以有相同的邮箱

## 相关文档

- [平台管理 API 文档](./PLATFORM_ADMIN_API.md)
- [认证 API 文档](./AUTH_QUICK_REFERENCE.md)
- [RBAC 权限控制](../internal/api/middleware/RBAC_USAGE.md)
