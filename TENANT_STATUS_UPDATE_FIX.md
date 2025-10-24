# 租户状态更新问题修复

## 问题描述

在更新租户信息时，当传入 `status: false` 时，接口提示更新成功，但实际上租户的状态并未改变，仍然保持为启用状态。

## 问题原因

问题出在 `internal/repository/tenant_repository.go` 的 `Update` 方法中。

原代码使用了 GORM 的 `Updates(tenant)` 方法：

```go
result := r.db.WithContext(ctx).
    Model(&model.Tenant{}).
    Where("id = ? AND is_deleted = ?", tenant.ID, false).
    Updates(tenant)
```

**GORM 的 `Updates()` 方法有一个特性**：它会忽略零值字段（zero value）。对于 `bool` 类型，`false` 被视为零值，因此当 `status` 字段为 `false` 时，GORM 会跳过这个字段的更新。

## 解决方案

修改 `Update` 方法，使用 `map[string]interface{}` 来构建更新字段，这样可以确保零值字段也能被正确更新：

```go
// Update 更新租户
func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
 if tenant == nil {
  return errors.New("tenant cannot be nil")
 }

 // 构建更新字段映射，以支持零值更新（如 status=false）
 updates := map[string]interface{}{
  "name":       tenant.Name,
  "domain":     tenant.Domain,
  "metadata":   tenant.Metadata,
  "status":     tenant.Status, // 使用 map 可以更新零值
  "updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
 }

 // 确保只更新未删除的租户
 result := r.db.WithContext(ctx).
  Model(&model.Tenant{}).
  Where("id = ? AND is_deleted = ?", tenant.ID, false).
  Updates(updates)

 if result.Error != nil {
  return result.Error
 }

 if result.RowsAffected == 0 {
  return errors.New("tenant not found or already deleted")
 }

 return nil
}
```

## 修改的文件

- `internal/repository/tenant_repository.go` - 修复租户状态更新问题
- `internal/repository/user_repository.go` - 修复用户 bool 字段更新问题（预防性修复）

## 测试方法

运行测试脚本验证修复：

```bash
./test_tenant_status_update.sh
```

测试脚本会：

1. 创建一个测试租户（默认 status=true）
2. 使用 PUT 接口将 status 更新为 false
3. 查询租户验证 status 是否为 false
4. 使用 PATCH 接口将 status 更新为 true
5. 查询租户验证 status 是否为 true
6. 清理测试数据

## 相关知识

### GORM 零值更新问题

GORM 在使用 `Updates()` 方法时，会忽略以下零值：

- `bool`: `false`
- `int`: `0`
- `string`: `""`
- `float`: `0.0`
- 指针: `nil`

### 解决零值更新的方法

1. **使用 map[string]interface{}**（推荐）

   ```go
   db.Model(&user).Updates(map[string]interface{}{
       "active": false,
       "age": 0,
   })
   ```

2. **使用 Select 指定字段**

   ```go
   db.Model(&user).Select("active", "age").Updates(user)
   ```

3. **使用 UpdateColumn**（跳过钩子和时间戳更新）

   ```go
   db.Model(&user).UpdateColumn("active", false)
   ```

## 影响范围

### 租户相关接口

此修复影响所有通过 `TenantRepository.Update()` 方法更新租户的操作，包括：

- PUT `/api/v1/tenants/{id}` - 更新租户信息
- PATCH `/api/v1/tenants/{id}/status` - 更新租户状态

修复后，这些接口都能正确处理 `status: false` 的更新请求。

### 用户相关接口

同时修复了 `UserRepository.Update()` 方法，影响以下 bool 字段的更新：

- `email_verified` - 邮箱验证状态
- `is_active` - 用户激活状态
- `is_admin` - 管理员标识

相关接口：

- PUT `/api/v1/users/{id}` - 更新用户信息
- PATCH `/api/v1/users/{id}/status` - 更新用户状态

## 注意事项

- 修复后需要重启服务才能生效
- 建议在生产环境部署前进行充分测试
- 已检查所有 Repository，目前只有 Tenant 和 User 两个模型使用了 `Updates()` 方法
- 其他模型如需添加类似功能，请注意避免零值更新问题
