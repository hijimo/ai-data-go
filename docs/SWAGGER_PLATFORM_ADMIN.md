# 平台管理员租户系统 Swagger 文档

## 概述

本文档说明如何访问和使用平台管理员租户系统的 Swagger API 文档。

## 访问 Swagger UI

启动服务后，可以通过以下 URL 访问 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

## 新增 API 接口

### 1. 平台管理 API

平台管理 API 用于管理所有租户的生命周期，需要平台管理员权限（`system_admin` 角色）。

#### 1.1 创建租户（带管理员）

- **端点**: `POST /api/v1/platform/tenants`
- **权限**: `system_admin`
- **功能**: 创建业务租户并自动生成租户管理员账户
- **响应**: 返回租户信息、管理员信息和初始密码

#### 1.2 获取租户列表

- **端点**: `GET /api/v1/platform/tenants`
- **权限**: `system_admin`
- **功能**: 获取所有租户列表，支持分页和类型过滤
- **参数**:
  - `pageNo`: 页码（默认 1）
  - `pageSize`: 每页大小（默认 10，最大 100）
  - `type`: 租户类型过滤（可选，`system` 或 `tenant`）

#### 1.3 更新租户状态

- **端点**: `PATCH /api/v1/platform/tenants/{id}/status`
- **权限**: `system_admin`
- **功能**: 启用或禁用租户
- **说明**: 禁用的租户下所有用户将无法登录

#### 1.4 删除租户

- **端点**: `DELETE /api/v1/platform/tenants/{id}`
- **权限**: `system_admin`
- **功能**: 软删除业务租户
- **限制**: 不允许删除平台租户（`type=system`）

### 2. 租户用户管理 API

租户用户管理 API 用于管理租户内的用户，需要租户管理员或平台管理员权限。

#### 2.1 在租户下创建用户

- **端点**: `POST /api/v1/tenants/{tenantId}/users`
- **权限**: `tenant_admin` 或 `system_admin`
- **功能**: 在指定租户下创建新用户

#### 2.2 获取租户用户列表

- **端点**: `GET /api/v1/tenants/{tenantId}/users`
- **权限**: `tenant_admin` 或 `system_admin`
- **功能**: 获取指定租户下的用户列表，支持分页

#### 2.3 更新用户状态

- **端点**: `PATCH /api/v1/tenants/{tenantId}/users/{userId}/status`
- **权限**: `tenant_admin` 或 `system_admin`
- **功能**: 启用或禁用用户
- **说明**: 禁用用户时会自动撤销其所有有效 Token

#### 2.4 删除用户

- **端点**: `DELETE /api/v1/tenants/{tenantId}/users/{userId}`
- **权限**: `tenant_admin` 或 `system_admin`
- **功能**: 软删除用户

## 数据模型更新

### Tenant 模型

新增字段：

- `type`: 租户类型
  - `system`: 平台租户（系统级租户，只能有一个）
  - `tenant`: 业务租户（普通租户）

### User 模型

增强字段说明：

- `roles`: 用户角色列表（支持多角色）
  - `system_admin`: 平台管理员（可管理所有租户）
  - `tenant_admin`: 租户管理员（可管理本租户用户）
  - `user`: 普通用户（基本业务权限）

### JWTClaims 模型

增强字段说明：

- `tid`: 租户 ID
- `roles`: 用户角色列表（用于权限验证）
- `scopes`: 权限范围列表（用于细粒度权限控制）

## 认证说明

### 使用 Bearer Token 认证

1. 首先通过登录接口获取 Access Token：

   ```
   POST /api/v1/auth/login
   ```

2. 在 Swagger UI 中点击右上角的 "Authorize" 按钮

3. 输入 Token（格式：`Bearer {your_access_token}`）

4. 点击 "Authorize" 完成认证

### 角色权限说明

- **平台管理员** (`system_admin`):
  - 可以访问所有平台管理 API
  - 可以跨租户访问数据
  - 可以管理所有租户和用户

- **租户管理员** (`tenant_admin`):
  - 可以管理本租户内的用户
  - 只能访问本租户的数据
  - 不能访问平台管理 API

- **普通用户** (`user`):
  - 只能访问基本业务功能
  - 不能管理其他用户

## 测试流程

### 1. 平台管理员测试流程

1. 使用平台管理员账户登录（默认：`admin@system.local`）
2. 创建业务租户（会自动生成租户管理员）
3. 查看租户列表
4. 更新租户状态（启用/禁用）
5. 删除租户

### 2. 租户管理员测试流程

1. 使用租户管理员账户登录
2. 在本租户下创建用户
3. 查看本租户用户列表
4. 更新用户状态（启用/禁用）
5. 删除用户

### 3. 权限验证测试

1. 使用租户管理员尝试访问平台管理 API（应返回 403）
2. 使用租户 A 的管理员尝试访问租户 B 的数据（应返回 403）
3. 使用平台管理员访问所有租户的数据（应成功）

## 常见问题

### Q: 如何获取平台管理员的初始密码？

A: 平台管理员的初始密码在系统首次启动时会记录在日志中。如果使用环境变量 `PLATFORM_ADMIN_PASSWORD` 设置了密码，则使用该密码。

### Q: 创建租户时返回的管理员密码在哪里？

A: 创建租户的 API 响应中包含 `adminPassword` 字段，这是租户管理员的初始密码。建议立即通过安全渠道传递给租户管理员。

### Q: 如何区分平台租户和业务租户？

A: 通过 `type` 字段区分：

- `type=system`: 平台租户
- `type=tenant`: 业务租户

### Q: 禁用租户后会发生什么？

A: 禁用租户后，该租户下的所有用户将无法登录，已登录的用户在 Token 过期后也无法刷新 Token。

### Q: 可以删除平台租户吗？

A: 不可以。平台租户是系统级租户，不允许删除。尝试删除会返回错误。

## 相关文档

- [平台管理员租户系统需求文档](../.kiro/specs/platform-admin-tenant/requirements.md)
- [平台管理员租户系统设计文档](../.kiro/specs/platform-admin-tenant/design.md)
- [平台管理员租户系统实施任务](../.kiro/specs/platform-admin-tenant/tasks.md)
- [RBAC 权限中间件使用指南](../internal/api/middleware/RBAC_USAGE.md)

## 更新日志

### 2025-01-20

- 添加平台管理 API 的 Swagger 注释
- 添加租户用户管理 API 的 Swagger 注释
- 更新 Tenant 和 User 模型的文档
- 更新 JWTClaims 模型的文档
- 生成完整的 Swagger 文档
