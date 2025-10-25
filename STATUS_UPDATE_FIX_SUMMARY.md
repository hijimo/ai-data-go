# 租户和用户状态更新接口修复总结

## 问题描述

更新租户状态接口（`PATCH /api/v1/tenants/{id}/status`）和用户状态更新接口（`PATCH /api/v1/users/{id}/status`）在处理 `false` 值时失败，返回参数验证错误。

### 错误信息

```json
{
  "code": 422,
  "message": "参数验证失败",
  "data": {
    "errors": [
      {
        "field": "status",
        "message": "status 是必填字段"
      }
    ]
  }
}
```

## 根本原因

在 Go 语言中，布尔类型的零值是 `false`。当使用 `validate:"required"` 标签验证布尔类型字段时，验证器会将 `false` 值视为零值，从而认为该字段未提供，导致验证失败。

### 问题代码

```go
// 租户状态更新请求（修复前）
type UpdateTenantStatusRequest struct {
    Status bool `json:"status" validate:"required" example:"true"`
}

// 用户状态更新请求（修复前）
type UpdateUserStatusRequest struct {
    IsActive bool `json:"isActive" validate:"required" example:"true"`
}
```

## 解决方案

将布尔类型字段改为指针类型（`*bool`），这样可以区分以下三种情况：

- `nil`：字段未提供（验证失败）
- `false`：明确设置为 false（验证通过）
- `true`：明确设置为 true（验证通过）

### 修复后的代码

```go
// 租户状态更新请求（修复后）
type UpdateTenantStatusRequest struct {
    Status *bool `json:"status" validate:"required" example:"true"`
}

// 用户状态更新请求（修复后）
type UpdateUserStatusRequest struct {
    IsActive *bool `json:"isActive" validate:"required" example:"true"`
}
```

## 修改的文件

1. **internal/api/handler/tenant_handler.go**
   - 修改 `UpdateTenantStatusRequest.Status` 字段类型为 `*bool`
   - 更新 `HandleUpdateStatus` 方法中的字段引用

2. **internal/api/handler/user_handler.go**
   - 修改 `UpdateUserStatusRequest.IsActive` 字段类型为 `*bool`
   - 更新 `HandleUpdateStatus` 方法中的字段引用

3. **测试脚本修复**
   - **test_token_permissions.sh**：将 `"status": "inactive"` 和 `"status": "active"` 改为 `"status": false` 和 `"status": true`
   - **test_tenant_api.sh**：将 `"status": "inactive"` 改为 `"status": false`

## 测试验证

### 租户状态更新测试

```bash
# 禁用租户
curl -X PATCH "http://localhost:8080/api/v1/tenants/{id}/status" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"status": false}'

# 启用租户
curl -X PATCH "http://localhost:8080/api/v1/tenants/{id}/status" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"status": true}'
```

### 用户状态更新测试

```bash
# 禁用用户
curl -X PATCH "http://localhost:8080/api/v1/users/{id}/status" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"isActive": false}'

# 启用用户
curl -X PATCH "http://localhost:8080/api/v1/users/{id}/status" \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"isActive": true}'
```

### 测试结果

✅ 所有测试通过：

- 租户禁用（status: false）- 成功
- 租户启用（status: true）- 成功
- 用户禁用（isActive: false）- 成功
- 用户启用（isActive: true）- 成功

## 影响范围

- 修复了租户状态更新接口无法设置 `status: false` 的问题
- 修复了用户状态更新接口无法设置 `isActive: false` 的问题
- 更新了相关测试脚本，使用正确的布尔值而不是字符串

## 注意事项

1. **API 兼容性**：此修复不影响 API 的外部接口，客户端仍然发送相同的 JSON 格式
2. **验证逻辑**：使用指针类型后，`validate:"required"` 标签会正确验证字段是否存在
3. **代码访问**：在代码中访问这些字段时需要解引用（使用 `*req.Status` 或 `*req.IsActive`）

## 相关问题

这是 Go 语言中使用验证器处理布尔类型字段的常见问题。类似的问题可能出现在其他使用布尔类型且标记为 `required` 的字段中。

## 建议

对于所有需要明确区分"未提供"和"false"的布尔类型字段，建议使用指针类型（`*bool`）而不是普通布尔类型（`bool`）。
