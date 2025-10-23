# 租户管理API使用指南

## 问题说明

您遇到的错误 `"RBAC: 未找到 JWT Claims"` 和 `401 Unauthorized` 是**正常的认证流程**，不是代码错误。

这个错误表示：

- ✅ JWT认证中间件正常工作
- ✅ 路由配置正确
- ❌ 请求中没有携带有效的JWT令牌

## 正确的API调用流程

### 1. 先登录获取访问令牌

```bash
# 使用平台管理员账户登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "platform-admin@system.local",
    "password": "你的管理员密码"
  }'
```

**响应示例：**

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "...",
    "expiresIn": 3600,
    "user": {
      "id": "...",
      "email": "platform-admin@system.local",
      "roles": ["system_admin"]
    }
  }
}
```

### 2. 使用访问令牌调用租户管理API

```bash
# 获取租户列表（需要在请求头中携带访问令牌）
curl -X GET "http://localhost:8080/api/v1/tenants?pageNo=1&pageSize=10" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**重要提示：**

- 必须在请求头中添加 `Authorization: Bearer <访问令牌>`
- 访问令牌从登录响应中获取
- 访问令牌有过期时间（默认1小时）

## 获取管理员初始密码

管理员初始密码在服务首次启动时生成，可以从启动日志中找到：

```
{"timestamp":"2025-10-23T12:00:00Z","level":"INFO","message":"系统初始化成功","fields":{
  "平台租户ID": "...",
  "管理员邮箱": "platform-admin@system.local",
  "管理员初始密码": "Admin@123456",
  "重要提示": "请妥善保管管理员初始密码，建议首次登录后立即修改"
}}
```

## 使用测试脚本

我们提供了一个完整的测试脚本 `test_tenant_api.sh`，演示了完整的API调用流程：

```bash
# 设置管理员密码（从启动日志中获取）
export ADMIN_PASSWORD="你的管理员密码"

# 运行测试脚本
./test_tenant_api.sh
```

测试脚本会自动执行以下操作：

1. ✅ 使用平台管理员账户登录
2. ✅ 获取租户列表
3. ✅ 创建新租户
4. ✅ 获取租户详情
5. ✅ 更新租户信息
6. ✅ 使用租户管理员账户登录
7. ✅ 测试租户管理员权限（只能看到自己的租户）
8. ✅ 禁用租户
9. ✅ 删除租户

## API端点列表

### 公开端点（不需要认证）

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/auth/register | 用户注册 |
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/refresh | 刷新令牌 |
| POST | /api/v1/auth/verify-email | 验证邮箱 |

### 租户管理端点（需要认证）

| 方法 | 路径 | 权限要求 | 描述 |
|------|------|----------|------|
| POST | /api/v1/tenants | system_admin | 创建租户 |
| GET | /api/v1/tenants | tenant_admin | 获取租户列表 |
| GET | /api/v1/tenants/{id} | tenant_admin | 获取租户详情 |
| PUT | /api/v1/tenants/{id} | tenant_admin | 更新租户 |
| PATCH | /api/v1/tenants/{id}/status | system_admin | 启用/禁用租户 |
| DELETE | /api/v1/tenants/{id} | system_admin | 删除租户 |

### 用户管理端点（需要认证）

| 方法 | 路径 | 权限要求 | 描述 |
|------|------|----------|------|
| POST | /api/v1/users | tenant_admin | 创建用户 |
| GET | /api/v1/users | tenant_admin | 获取用户列表 |
| GET | /api/v1/users/{id} | tenant_admin | 获取用户详情 |
| PUT | /api/v1/users/{id} | tenant_admin | 更新用户 |
| PATCH | /api/v1/users/{id}/status | tenant_admin | 更新用户状态 |
| DELETE | /api/v1/users/{id} | tenant_admin | 删除用户 |

## 权限说明

### 平台管理员（system_admin）

- ✅ 可以查看、修改、删除所有租户的数据
- ✅ 可以创建新租户
- ✅ 可以管理所有租户下的用户
- ✅ 可以启用/禁用任何租户

### 租户管理员（tenant_admin）

- ✅ 只能查看、修改自己租户的数据
- ❌ 不能访问其他租户的数据
- ✅ 只能管理自己租户下的用户
- ❌ 不能创建新租户
- ❌ 不能启用/禁用租户

## 常见错误

### 401 Unauthorized - 未找到 JWT Claims

**原因：** 请求中没有携带有效的JWT令牌

**解决方案：**

1. 先调用登录接口获取访问令牌
2. 在请求头中添加 `Authorization: Bearer <访问令牌>`

### 403 Forbidden - 权限不足

**原因：** 用户没有执行该操作的权限

**解决方案：**

1. 检查用户角色是否满足接口要求
2. 租户管理员只能访问自己租户的数据
3. 某些操作（如创建租户、禁用租户）只有平台管理员可以执行

### 404 Not Found - 资源不存在

**原因：** 请求的资源不存在或已被删除

**解决方案：**

1. 检查资源ID是否正确
2. 确认资源未被删除（软删除）

## Swagger文档

访问 Swagger UI 查看完整的API文档：

```
http://localhost:8080/swagger/index.html
```

Swagger UI 提供了：

- 📖 完整的API文档
- 🔐 内置的认证测试功能
- 🧪 在线API测试工具

## 下一步

1. ✅ 使用测试脚本验证API功能
2. ✅ 查看Swagger文档了解更多API细节
3. ✅ 根据业务需求调用相应的API接口
4. ✅ 实现前端集成

## 技术支持

如果遇到问题，请检查：

1. 服务是否正常启动
2. 数据库连接是否正常
3. 管理员账户是否已创建
4. JWT令牌是否正确携带
5. 用户权限是否满足要求
