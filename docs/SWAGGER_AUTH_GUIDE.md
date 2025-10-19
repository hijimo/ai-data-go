# 认证 API Swagger 文档使用指南

## 概述

本文档说明如何使用 Swagger UI 查看和测试多租户用户管理与 JWT 身份认证系统的 API。

## 访问 Swagger UI

启动服务后，访问以下 URL 查看 Swagger 文档：

```
http://localhost:8080/swagger/index.html
```

## API 分类

### 1. 认证相关接口（标签：认证）

这些接口用于用户注册、登录、Token 管理等操作：

- **POST /api/v1/auth/register** - 用户注册
- **POST /api/v1/auth/login** - 用户登录
- **POST /api/v1/auth/refresh** - 刷新访问令牌
- **POST /api/v1/auth/logout** - 用户注销
- **POST /api/v1/auth/change-password** - 修改密码（需要认证）
- **GET /api/v1/auth/me** - 获取当前用户信息（需要认证）

### 2. 租户管理接口（标签：租户管理）

这些接口用于管理租户，需要管理员权限：

- **POST /api/v1/tenants** - 创建租户
- **GET /api/v1/tenants** - 获取租户列表
- **GET /api/v1/tenants/{id}** - 获取租户详情
- **PUT /api/v1/tenants/{id}** - 更新租户
- **DELETE /api/v1/tenants/{id}** - 删除租户

### 3. 用户管理接口（标签：用户管理）

这些接口用于管理租户下的用户，需要租户管理员权限：

- **POST /api/v1/users** - 创建用户
- **GET /api/v1/users** - 获取用户列表
- **GET /api/v1/users/{id}** - 获取用户详情
- **PUT /api/v1/users/{id}** - 更新用户
- **DELETE /api/v1/users/{id}** - 删除用户

## 使用 Bearer Token 认证

### 步骤 1：获取 Access Token

1. 首先使用 **POST /api/v1/auth/login** 接口登录
2. 在请求体中提供：

   ```json
   {
     "email": "user@example.com",
     "password": "password123",
     "tenantId": "550e8400-e29b-41d4-a716-446655440000"
   }
   ```

3. 登录成功后，响应中会包含 `accessToken` 和 `refreshToken`

### 步骤 2：在 Swagger UI 中配置认证

1. 点击 Swagger UI 右上角的 **Authorize** 按钮（锁形图标）
2. 在弹出的对话框中，输入：

   ```
   Bearer <your_access_token>
   ```

   注意：必须包含 "Bearer " 前缀，后面跟上实际的 token
3. 点击 **Authorize** 按钮
4. 点击 **Close** 关闭对话框

### 步骤 3：测试需要认证的接口

配置认证后，所有标记为 🔒 的接口都可以正常调用了。

## 租户识别

系统支持多种租户识别方式，在 Swagger UI 中测试时，推荐使用请求头方式：

### 方式 1：使用 X-Tenant-ID 请求头（推荐）

在 Swagger UI 的请求参数中添加：

- Header Name: `X-Tenant-ID`
- Value: 租户的 UUID

### 方式 2：在登录时指定租户

在登录请求中包含 `tenantId` 字段，系统会将租户信息编码到 JWT token 中。

## 常见响应格式

### 成功响应（普通数据）

```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 实际数据
  }
}
```

### 成功响应（分页数据）

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

### 错误响应

```json
{
  "code": 400,
  "message": "错误描述"
}
```

## 常见错误码

- **400** - 请求参数错误
- **401** - 未认证或 Token 无效
- **403** - 权限不足
- **404** - 资源不存在
- **422** - 参数验证失败
- **500** - 服务器内部错误

## Token 生命周期管理

### Access Token

- **生命周期**：60 分钟
- **用途**：API 访问授权
- **过期处理**：使用 Refresh Token 获取新的 Access Token

### Refresh Token

- **生命周期**：30 天（可配置）
- **用途**：获取新的 Access Token
- **安全机制**：一次性使用，使用后自动轮换

### Token 刷新流程

1. 当 Access Token 过期时，调用 **POST /api/v1/auth/refresh**
2. 在请求体中提供 Refresh Token：

   ```json
   {
     "refreshToken": "your_refresh_token"
   }
   ```

3. 成功后会返回新的 Access Token 和 Refresh Token
4. 使用新的 Access Token 更新 Swagger UI 的认证配置

## 测试流程示例

### 完整的认证测试流程

1. **注册用户**
   - 调用 POST /api/v1/auth/register
   - 提供租户 ID、邮箱、密码等信息

2. **用户登录**
   - 调用 POST /api/v1/auth/login
   - 获取 Access Token 和 Refresh Token

3. **配置认证**
   - 在 Swagger UI 中点击 Authorize
   - 输入 "Bearer {access_token}"

4. **测试受保护的接口**
   - 调用 GET /api/v1/auth/me 获取当前用户信息
   - 调用其他需要认证的接口

5. **刷新 Token**
   - 当 Access Token 过期时
   - 调用 POST /api/v1/auth/refresh

6. **用户注销**
   - 调用 POST /api/v1/auth/logout
   - 提供 Refresh Token

## 权限说明

### 普通用户权限

- 可以访问自己的信息
- 可以修改自己的密码
- 可以在自己的租户内进行操作

### 租户管理员权限

- 拥有普通用户的所有权限
- 可以管理租户内的用户（创建、更新、删除）
- 可以查看租户内的所有数据

### 系统管理员权限

- 拥有租户管理员的所有权限
- 可以管理所有租户（创建、更新、删除）
- 可以跨租户操作

## 注意事项

1. **安全性**
   - 不要在生产环境中暴露 Swagger UI
   - 不要在日志或截图中泄露 Token
   - 定期更换密码和 Token

2. **租户隔离**
   - 所有操作都必须在租户上下文中进行
   - 确保请求中包含正确的租户标识符
   - 不能跨租户访问数据

3. **Token 管理**
   - Access Token 应存储在内存中，不要存储在 localStorage
   - Refresh Token 应安全存储
   - Token 过期后及时刷新

4. **错误处理**
   - 注意查看响应中的错误信息
   - 401 错误表示需要重新登录
   - 403 错误表示权限不足

## 更新 Swagger 文档

如果修改了 API 接口或注释，需要重新生成 Swagger 文档：

```bash
# 生成 Swagger 文档
make swagger

# 或者手动执行
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

## 相关文档

- [需求文档](.kiro/specs/multi-tenant-auth/requirements.md)
- [设计文档](.kiro/specs/multi-tenant-auth/design.md)
- [实施计划](.kiro/specs/multi-tenant-auth/tasks.md)
- [API 参考卡片](../API_REFERENCE_CARD.md)

## 技术支持

如有问题，请联系：

- Email: <support@example.com>
- 项目仓库：提交 Issue
