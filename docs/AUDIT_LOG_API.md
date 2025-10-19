# 审计日志查询 API 使用指南

## 概述

审计日志查询 API 允许管理员查询系统中的认证相关审计日志，支持多条件过滤和分页功能。

## API 端点

```
GET /api/v1/audit/auth
```

## 权限要求

- 需要管理员权限（`admin` 角色）
- 需要有效的 JWT Access Token

## 请求参数

所有参数都是可选的，通过 URL 查询参数传递：

| 参数名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `tenantId` | string | 租户ID（UUID格式） | `550e8400-e29b-41d4-a716-446655440000` |
| `userId` | string | 用户ID（UUID格式） | `550e8400-e29b-41d4-a716-446655440001` |
| `event` | string | 事件类型 | `login`, `logout`, `refresh`, `revoke`, `failed_login` |
| `startTime` | string | 开始时间（RFC3339格式） | `2024-01-01T00:00:00Z` |
| `endTime` | string | 结束时间（RFC3339格式） | `2024-12-31T23:59:59Z` |
| `page` | int | 页码（从1开始） | `1` |
| `pageSize` | int | 每页大小（最大100） | `10` |

## 事件类型说明

- `login` - 用户登录成功
- `logout` - 用户注销
- `refresh` - Token 刷新
- `revoke` - Token 撤销
- `failed_login` - 登录失败

## 响应格式

成功响应（HTTP 200）：

```json
{
  "code": 200,
  "message": "查询审计日志成功",
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "tenantId": "550e8400-e29b-41d4-a716-446655440001",
        "userId": "550e8400-e29b-41d4-a716-446655440002",
        "event": "login",
        "ip": "192.168.1.1",
        "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
        "meta": {
          "email": "user@example.com"
        },
        "createdAt": "2024-01-15T10:30:00Z"
      }
    ],
    "pageNo": 1,
    "pageSize": 10,
    "totalCount": 100,
    "totalPage": 10
  }
}
```

错误响应：

```json
{
  "code": 400,
  "message": "无效的租户ID",
  "data": null
}
```

## 使用示例

### 1. 查询所有审计日志（分页）

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?page=1&pageSize=20" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

### 2. 查询特定租户的审计日志

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?tenantId=550e8400-e29b-41d4-a716-446655440000&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

### 3. 查询特定用户的登录记录

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?userId=550e8400-e29b-41d4-a716-446655440001&event=login&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

### 4. 查询时间范围内的审计日志

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?startTime=2024-01-01T00:00:00Z&endTime=2024-01-31T23:59:59Z&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

### 5. 查询登录失败记录

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?event=failed_login&page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

### 6. 组合多个过滤条件

```bash
curl -X GET "http://localhost:8080/api/v1/audit/auth?tenantId=550e8400-e29b-41d4-a716-446655440000&event=login&startTime=2024-01-01T00:00:00Z&endTime=2024-01-31T23:59:59Z&page=1&pageSize=20" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "X-Tenant-ID: YOUR_TENANT_ID"
```

## 错误码说明

| HTTP状态码 | 错误码 | 说明 |
|-----------|--------|------|
| 400 | 400 | 请求参数错误（如无效的UUID格式、时间格式等） |
| 401 | 401 | 未授权（缺少或无效的 Access Token） |
| 403 | 403 | 权限不足（非管理员用户） |
| 500 | 500 | 服务器内部错误 |

## 注意事项

1. **权限要求**：只有具有 `admin` 角色的用户才能访问此 API
2. **时间格式**：时间参数必须使用 RFC3339 格式（如 `2024-01-01T00:00:00Z`）
3. **UUID 格式**：租户ID和用户ID必须是有效的 UUID 格式
4. **分页限制**：每页最大返回 100 条记录
5. **性能考虑**：建议使用时间范围过滤来限制查询结果，避免查询过大的数据集

## 安全建议

1. 始终使用 HTTPS 传输敏感数据
2. 定期审查审计日志，检测异常活动
3. 设置合理的日志保留策略
4. 限制审计日志的访问权限

## 相关文档

- [认证系统快速参考](./AUTH_QUICK_REFERENCE.md)
- [认证系统设置指南](./AUTH_SETUP.md)
- [Swagger API 文档](http://localhost:8080/swagger/index.html)
