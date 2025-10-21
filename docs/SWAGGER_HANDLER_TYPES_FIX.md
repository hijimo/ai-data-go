# Swagger Handler 类型名称优化

## 问题

在 Swagger 文档中，定义在 `internal/api/handler` 包中的请求和响应类型会自动添加 `internal_api_handler.` 前缀，例如：

- `internal_api_handler.ResendVerificationRequest`
- `internal_api_handler.LoginRequest`
- `internal_api_handler.CreateTenantWithAdminRequest`

这样的命名在 Swagger UI 中显示时不够简洁友好。

## 解决方案

### 1. 添加 @name 注解

为所有 handler 包中的请求和响应类型添加 `@name` 注解：

```go
// ResendVerificationRequest 重新发送验证邮件请求（用于 Swagger）
// @name ResendVerificationRequest
type ResendVerificationRequest struct {
    TenantID string `json:"tenantId" validate:"required"`
    UserID   string `json:"userId" validate:"required"`
}
```

### 2. 更新后处理脚本

更新 `scripts/fix_swagger_names.sh` 脚本，添加对 `internal_api_handler.` 前缀的处理：

```bash
PREFIX2="internal_api_handler."

# 在 swagger.json、swagger.yaml 和 docs.go 中移除该前缀
sed -i.bak "s/${PREFIX2}//g" docs/swagger.json
```

### 3. 重新生成文档

运行以下命令重新生成 Swagger 文档：

```bash
make swagger
```

## 效果对比

### 优化前

```json
{
  "$ref": "#/definitions/internal_api_handler.ResendVerificationRequest"
}
```

### 优化后

```json
{
  "$ref": "#/definitions/ResendVerificationRequest"
}
```

## 已优化的类型

### 认证相关请求

- `RegisterRequest` - 用户注册
- `LoginRequest` - 用户登录
- `RefreshRequest` - 刷新令牌
- `LogoutRequest` - 用户注销
- `ChangePasswordRequest` - 修改密码
- `UnlockAccountRequest` - 解锁账户
- `VerifyEmailRequest` - 验证邮箱
- `ResendVerificationRequest` - 重发验证邮件

### 租户管理请求

- `CreateTenantRequest` - 创建租户
- `UpdateTenantRequest` - 更新租户
- `CreateTenantWithAdminRequest` - 创建租户（带管理员）
- `UpdateTenantStatusRequest` - 更新租户状态

### 用户管理请求

- `CreateUserRequest` - 创建用户
- `UpdateUserRequest` - 更新用户
- `UpdateUserStatusRequest` - 更新用户状态

### 审计日志请求

- `AuditQueryRequest` - 审计日志查询

### 响应类型

- `CreateTenantWithAdminResponse` - 创建租户响应
- `CreateTenantWithAdminDataResponse` - 创建租户数据响应
- `HealthStatusResponse` - 健康状态响应

## 注意事项

1. **自动处理**：每次运行 `make swagger` 时，脚本会自动移除类型名称前缀，无需手动干预。

2. **新增类型**：当在 handler 包中添加新的请求或响应类型时，记得添加 `@name` 注解，然后运行 `make swagger`。

3. **@name 注解位置**：注解必须紧跟在类型注释之后，在 `type` 关键字之前。

4. **脚本维护**：如果项目中新增了其他包，且这些包的类型也需要移除前缀，可以在 `scripts/fix_swagger_names.sh` 中添加相应的处理逻辑。
