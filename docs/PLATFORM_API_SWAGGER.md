# 平台管理 API Swagger 文档

## 概述

本文档说明平台管理 API 的 Swagger 注释已完成添加，所有接口都包含详细的文档说明。

## 已完成的 API 文档

### 1. 创建租户（带管理员）

**端点:** `POST /api/v1/platform/tenants`

**权限要求:** system_admin

**功能说明:**

- 创建类型为 "tenant" 的业务租户
- 自动生成租户管理员账户（角色为 tenant_admin）
- 生成16位随机强密码（包含大小写字母、数字和特殊字符）
- 返回租户信息和管理员初始密码

**请求参数:**

- `tenantName` (必填): 租户名称（1-255字符）
- `tenantDomain` (必填): 租户域名（最多255字符）
- `tenantMetadata` (可选): 租户元数据（JSON对象）
- `adminEmail` (可选): 管理员邮箱（默认为 admin@{tenantDomain}）
- `adminDisplayName` (可选): 管理员显示名称（最多255字符）

**响应示例:**

```json
{
  "code": 201,
  "message": "租户创建成功",
  "data": {
    "tenant": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "示例公司",
      "domain": "example.com",
      "type": "tenant",
      "status": true
    },
    "adminUser": {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "email": "admin@example.com",
      "displayName": "管理员",
      "roles": ["tenant_admin"],
      "isAdmin": true
    },
    "adminPassword": "Xy9#mK2$pL5@qR8!"
  }
}
```

### 2. 获取租户列表

**端点:** `GET /api/v1/platform/tenants`

**权限要求:** system_admin

**功能说明:**

- 支持分页查询（默认每页10条，最多100条）
- 支持按租户类型过滤（system: 平台租户, tenant: 业务租户）
- 返回租户的完整信息，包括名称、域名、类型、状态等

**查询参数:**

- `pageNo` (可选): 页码（从1开始，默认1）
- `pageSize` (可选): 每页大小（1-100，默认10）
- `type` (可选): 租户类型过滤（system 或 tenant）

**响应示例:**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Platform",
        "domain": "system.local",
        "type": "system",
        "status": true
      },
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "示例公司",
        "domain": "example.com",
        "type": "tenant",
        "status": true
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 2,
    "totalPage": 1
  }
}
```

### 3. 启用/禁用租户

**端点:** `PATCH /api/v1/platform/tenants/{id}/status`

**权限要求:** system_admin

**功能说明:**

- 启用租户：设置 status = true，该租户下的用户可以正常登录和访问系统
- 禁用租户：设置 status = false，该租户下的所有用户将无法登录和访问系统

**影响范围:**

- 禁用租户后，该租户下所有用户的登录请求将被拒绝
- 禁用租户后，该租户下所有用户的 API 访问请求将被拒绝
- 启用租户后，该租户下用户恢复正常访问

**路径参数:**

- `id` (必填): 租户ID（UUID格式）

**请求参数:**

- `status` (必填): 租户状态（true: 启用, false: 禁用）

**响应示例:**

```json
{
  "code": 200,
  "message": "租户状态更新成功",
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "示例公司",
    "domain": "example.com",
    "type": "tenant",
    "status": false
  }
}
```

### 4. 删除租户

**端点:** `DELETE /api/v1/platform/tenants/{id}`

**权限要求:** system_admin

**功能说明:**

- 执行软删除操作，设置 is_deleted = true
- 不会物理删除数据库记录，保留数据用于审计和恢复
- 删除后的租户不会出现在租户列表中

**限制条件:**

- 不允许删除平台租户（type = "system"）
- 删除租户时会级联处理相关数据（根据数据库外键约束）

**注意事项:**

- 删除操作不可逆（除非通过数据库直接恢复）
- 建议在删除前先禁用租户，观察一段时间后再删除
- 删除租户会影响该租户下的所有用户和数据

**路径参数:**

- `id` (必填): 租户ID（UUID格式）

**响应示例:**

```json
{
  "code": 200,
  "message": "删除租户成功",
  "data": {}
}
```

## 访问 Swagger UI

启动服务器后，可以通过以下 URL 访问 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

## 生成 Swagger 文档

如果修改了 API 注释，需要重新生成 Swagger 文档：

```bash
make swagger
```

或者使用完整的开发命令（生成文档并启动服务器）：

```bash
make dev
```

## 注意事项

1. **权限要求**: 所有平台管理 API 都需要 `system_admin` 角色权限
2. **认证方式**: 使用 Bearer Token 认证（在请求头中添加 `Authorization: Bearer <token>`）
3. **响应格式**: 所有 API 都遵循统一的响应格式（ResponseData 或 ResponsePaginationData）
4. **错误处理**: 详细的错误码和错误消息说明

## 相关文档

- [平台管理员租户系统需求文档](../.kiro/specs/platform-admin-tenant/requirements.md)
- [平台管理员租户系统设计文档](../.kiro/specs/platform-admin-tenant/design.md)
- [租户用户管理 API 文档](./TENANT_USER_MANAGEMENT_API.md)
