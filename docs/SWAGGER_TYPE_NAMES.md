# Swagger 类型名称优化

## 问题描述

默认情况下，Swaggo 生成的 Swagger 文档中的类型名称会包含完整的包路径前缀，例如：

- `genkit-ai-service_internal_model.UserListResponse`
- `genkit-ai-service_internal_model.TenantListResponse`
- `genkit-ai-service_internal_model.ErrorResponse`

这样的命名虽然可以避免不同包之间的类型名称冲突，但在 Swagger UI 中显示时会显得冗长且不够友好。

## 解决方案

我们通过以下两个步骤来优化类型名称：

### 1. 在模型定义中添加 @name 注解

在 `internal/model/` 目录下的所有结构体定义上添加 `// @name` 注解，例如：

```go
// User 用户模型
// @Description 用户信息，包含用户的基本信息、角色和状态
// @name User
type User struct {
    // ...
}

// UserListResponse 用户列表响应（用于 Swagger）
// @name UserListResponse
type UserListResponse = ResponsePaginationData[[]User]
```

### 2. 使用后处理脚本清理类型名称

创建了 `scripts/fix_swagger_names.sh` 脚本，在 Swagger 文档生成后自动移除类型名称中的包路径前缀。

脚本会处理以下文件：

- `docs/swagger.json`
- `docs/swagger.yaml`
- `docs/docs.go`

### 3. 更新 Makefile

在 `Makefile` 的 `swagger` 目标中添加了后处理步骤：

```makefile
swagger:
 @echo "生成 Swagger 文档..."
 @if command -v swag >/dev/null 2>&1; then \
  swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal; \
 elif [ -f ~/go/bin/swag ]; then \
  ~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal; \
 else \
  echo "❌ 错误: swag 命令未找到"; \
  echo "请运行: make swagger-install"; \
  exit 1; \
 fi
 @./scripts/fix_swagger_names.sh
 @echo "✅ Swagger 文档生成完成"
```

## 效果对比

### 优化前

```json
{
  "$ref": "#/definitions/genkit-ai-service_internal_model.TenantListResponse"
}
```

### 优化后

```json
{
  "$ref": "#/definitions/TenantListResponse"
}
```

## 使用方法

每次修改 API 接口或模型定义后，运行以下命令重新生成 Swagger 文档：

```bash
make swagger
```

脚本会自动执行类型名称的清理工作，无需手动干预。

## 注意事项

1. **@name 注解的位置**：`@name` 注解必须紧跟在类型定义的注释之后，在 `type` 关键字之前。

2. **类型别名的限制**：虽然我们为类型别名（type alias）添加了 `@name` 注解，但 Swaggo 对类型别名的支持有限，因此仍需要后处理脚本来完成最终的清理。

3. **重新生成文档**：每次运行 `make swagger` 时，都会重新生成文档并自动清理类型名称，因此不要手动编辑生成的文档文件。

4. **跨包类型冲突**：如果项目中有多个包定义了同名的类型，移除包路径前缀后可能会导致类型名称冲突。在这种情况下，需要手动为这些类型指定不同的 `@name` 注解。

## 已优化的类型

以下类型已经完成了名称优化：

### 响应类型

- `ErrorResponse`
- `SuccessResponse`
- `EmptyData`

### 认证相关

- `User`
- `Tenant`
- `LoginResponse`
- `AuthAuditItem`

### 列表响应

- `UserListResponse`
- `TenantListResponse`
- `AuthAuditListResponse`
- `SessionListResponse`
- `MessageDetailListResponse`

### 数据响应

- `UserDataResponse`
- `TenantDataResponse`
- `LoginDataResponse`
- `SessionDataResponse`
- `MessageResponseData`
- `MessageDetailDataResponse`
- `ChatResponseData`

### AI 相关

- `Provider`
- `Model`
- `ParameterRule`
- `ChatResponse`
- `Usage`
- `Message`
- `MessagePreview`
- `MessageResponse`
- `MessageDetailResponse`
- `SessionResponse`

### Handler 请求类型

- `RegisterRequest`
- `LoginRequest`
- `RefreshRequest`
- `LogoutRequest`
- `ChangePasswordRequest`
- `UnlockAccountRequest`
- `VerifyEmailRequest`
- `ResendVerificationRequest`
- `CreateTenantRequest`
- `UpdateTenantRequest`
- `CreateTenantWithAdminRequest`
- `UpdateTenantStatusRequest`
- `CreateUserRequest`
- `UpdateUserRequest`
- `UpdateUserStatusRequest`
- `AuditQueryRequest`

### Handler 响应类型

- `CreateTenantWithAdminResponse`
- `CreateTenantWithAdminDataResponse`
- `HealthStatusResponse`

### 其他

- `ProviderListDataResponse`
- `ProviderDataResponse`
- `ModelListDataResponse`
- `ModelDataResponse`
- `ParameterRuleListDataResponse`
- `MetricsDataResponse`
- `AlertListDataResponse`
- `HealthDataResponse`
- `AnyDataResponse`

## 维护指南

当添加新的模型或响应类型时，请遵循以下步骤：

1. 在结构体定义上添加 `// @name TypeName` 注解
2. 运行 `make swagger` 重新生成文档
3. 验证生成的文档中类型名称是否正确（无包路径前缀）

如果发现某些类型名称仍然包含前缀，可能需要：

- 检查 `@name` 注解是否正确添加
- 检查 `scripts/fix_swagger_names.sh` 脚本是否需要更新
- 确认该类型是否在 Swagger 文档中被引用

### Handler 包中的类型

对于在 `internal/api/handler` 包中定义的请求和响应类型，虽然添加了 `@name` 注解，但 swag 工具对这些注解的支持有限。因此，我们通过后处理脚本 `scripts/fix_swagger_names.sh` 来移除以下前缀：

- `internal_api_handler.` - Handler 包中的类型
- `genkit-ai-service_internal_model.` - Model 包中的类型
- `genkit-ai-service_internal_service_health.` - Health 服务包中的类型

脚本会自动在 `make swagger` 命令执行后运行，无需手动干预。
