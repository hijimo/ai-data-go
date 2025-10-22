# 登录接口优化总结

## 变更概述

已成功将登录接口从"需要租户ID+邮箱+密码"简化为"只需邮箱+密码"，因为邮箱已经是全局唯一的。

## 主要变更

### 1. 数据库层面

**文件**: `internal/model/auth.go`

- 将邮箱字段的索引从 `uniqueIndex:idx_tenant_email`（租户内唯一）改为 `uniqueIndex`（全局唯一）

**新增迁移**:

- `internal/database/migrations/email_unique_migration.go` - 邮箱全局唯一性迁移
- `scripts/migrate_email_unique.go` - 迁移执行脚本

### 2. Repository 层面

**文件**: `internal/repository/user_repository.go`

新增方法：

```go
// GetByEmailOnly 仅根据邮箱获取用户（不需要租户ID）
GetByEmailOnly(ctx context.Context, email string) (*model.User, error)
```

### 3. Service 层面

**文件**: `internal/service/auth/auth_service.go`

**LoginRequest 结构变更**:

```go
// 变更前
type LoginRequest struct {
    TenantID string `json:"tenantId"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

// 变更后
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

**Login 方法变更**:

- 使用 `GetByEmailOnly` 替代 `GetByEmail`
- 从用户记录中获取租户ID，无需客户端提供

**Register 方法变更**:

- 使用 `GetByEmailOnly` 检查邮箱全局唯一性

### 4. Handler 层面

**文件**: `internal/api/handler/auth_handler.go`

**LoginRequest 结构变更**:

- 移除 `TenantID` 字段
- 移除从上下文获取租户ID的逻辑
- 简化日志记录

### 5. 测试更新

**文件**: `internal/service/auth/login_limit_test.go`

- 更新所有测试用例，移除 `TenantID` 参数
- 添加 `GetByEmailOnly` mock 实现
- 添加 `mockTenantRepoForLoginLimit`

## API 变更

### 登录接口

**端点**: `POST /auth/login`

**变更前**:

```json
{
  "tenantId": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "password": "password123"
}
```

**变更后**:

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应**: 保持不变，仍然包含完整的用户信息（包括租户ID）

## 执行步骤

### 1. 执行数据库迁移

```bash
go run scripts/migrate_email_unique.go
```

### 2. 测试登录功能

```bash
./test_login_without_tenant.sh
```

### 3. 更新 Swagger 文档

```bash
swag init -g cmd/server/main.go -o docs
```

## 优势

1. **用户体验提升**: 用户不需要知道或记住租户ID
2. **简化接口**: 减少必填参数，降低使用门槛
3. **符合常规**: 大多数应用都使用邮箱作为唯一登录标识
4. **安全性**: 邮箱全局唯一可以防止账户混淆

## 注意事项

1. **邮箱唯一性**: 迁移后，同一个邮箱不能在多个租户中注册
2. **现有数据**: 如果现有数据中有重复邮箱，需要先处理
3. **注册流程**: 注册时仍需提供租户ID
4. **向后兼容**: 旧的 `GetByEmail` 方法仍然保留

## 相关文件

### 新增文件

- `internal/database/migrations/email_unique_migration.go`
- `scripts/migrate_email_unique.go`
- `docs/EMAIL_UNIQUE_MIGRATION.md`
- `test_login_without_tenant.sh`
- `LOGIN_WITHOUT_TENANT_SUMMARY.md`

### 修改文件

- `internal/model/auth.go`
- `internal/repository/user_repository.go`
- `internal/service/auth/auth_service.go`
- `internal/api/handler/auth_handler.go`
- `internal/service/auth/login_limit_test.go`

## 测试清单

- [x] 数据库迁移脚本
- [x] Repository 层新增方法
- [x] Service 层登录逻辑
- [x] Handler 层请求处理
- [x] 单元测试更新
- [ ] 集成测试（需要运行服务器）
- [ ] Swagger 文档更新

## 后续工作

1. 运行服务器并执行集成测试
2. 更新 Swagger 文档
3. 更新 API 文档和用户手册
4. 通知前端团队更新登录表单
