# 登录失败限制功能文档

## 概述

登录失败限制功能用于保护用户账户免受暴力破解攻击。当用户连续多次输入错误密码时，系统会自动锁定账户一段时间。

## 功能特性

### 1. 失败次数记录

- 系统会记录每个用户的登录失败次数
- 每次密码验证失败时，失败次数自动增加
- 失败次数存储在用户表的 `failed_login_attempts` 字段中

### 2. 账户自动锁定

- 当失败次数达到配置的最大值时，账户会被自动锁定
- 锁定时间由配置决定（默认 15 分钟）
- 锁定信息存储在用户表的 `locked_until` 字段中
- 锁定事件会记录到审计日志中

### 3. 锁定期间的行为

- 账户被锁定期间，即使输入正确的密码也无法登录
- 系统会返回明确的错误消息："账户已被锁定，请稍后再试"
- 所有登录尝试都会被记录到审计日志

### 4. 自动解锁

- 锁定时间到期后，账户会自动解锁
- 下次登录时系统会检查锁定时间是否已过期
- 如果已过期，会自动清除锁定状态

### 5. 手动解锁

- 管理员可以通过 API 手动解锁账户
- 解锁操作会重置失败次数并清除锁定时间
- 解锁事件会记录到审计日志

### 6. 成功登录后重置

- 用户成功登录后，失败次数会自动重置为 0
- 锁定状态也会被清除

## 配置参数

在 `.env` 文件或环境变量中配置以下参数：

```bash
# 最大登录失败次数（默认：5）
MAX_LOGIN_ATTEMPTS=5

# 账户锁定时长（默认：15m）
LOGIN_ATTEMPT_WINDOW=15m
```

## API 接口

### 解锁账户

**端点：** `POST /api/v1/auth/unlock-account`

**权限：** 需要管理员权限

**请求体：**

```json
{
  "tenantId": "550e8400-e29b-41d4-a716-446655440000",
  "userId": "550e8400-e29b-41d4-a716-446655440001"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "账户解锁成功",
  "data": {}
}
```

## 数据库字段

### users 表新增字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `failed_login_attempts` | INT | 登录失败次数，默认为 0 |
| `locked_until` | TIMESTAMP | 账户锁定截止时间，NULL 表示未锁定 |

## 审计日志事件

系统会记录以下与登录失败限制相关的事件：

| 事件类型 | 说明 |
|----------|------|
| `failed_login` | 登录失败（包含失败原因和当前失败次数） |
| `account_locked` | 账户被锁定（包含失败次数和锁定时长） |
| `account_unlocked` | 账户被解锁（手动或自动） |

## 使用示例

### 场景 1：正常登录失败

```bash
# 第一次失败
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "email": "user@example.com",
    "password": "wrongpassword"
  }'

# 响应
{
  "code": 401,
  "message": "邮箱或密码错误，还剩 4 次尝试机会"
}
```

### 场景 2：账户被锁定

```bash
# 第 5 次失败后
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "email": "user@example.com",
    "password": "wrongpassword"
  }'

# 响应
{
  "code": 401,
  "message": "登录失败次数过多，账户已被锁定 15 分钟"
}
```

### 场景 3：锁定期间尝试登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-123",
    "email": "user@example.com",
    "password": "correctpassword"
  }'

# 响应
{
  "code": 401,
  "message": "账户已被锁定，请稍后再试"
}
```

### 场景 4：管理员解锁账户

```bash
curl -X POST http://localhost:8080/api/v1/auth/unlock-account \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "tenantId": "tenant-123",
    "userId": "user-456"
  }'

# 响应
{
  "code": 200,
  "message": "账户解锁成功",
  "data": {}
}
```

## 安全建议

1. **合理设置最大失败次数**：建议设置为 3-5 次，既能防止暴力破解，又不会过度影响用户体验

2. **适当的锁定时长**：建议设置为 15-30 分钟，足以阻止自动化攻击，但不会给用户造成太大困扰

3. **监控审计日志**：定期检查 `account_locked` 事件，识别可能的攻击行为

4. **通知用户**：考虑在账户被锁定时发送邮件或短信通知用户

5. **IP 限制**：可以结合 IP 地址限制，对同一 IP 的多次失败尝试进行额外限制

## 实现细节

### 登录流程

1. 检查账户是否被锁定
2. 验证用户状态（是否激活）
3. 验证密码
4. 如果密码错误：
   - 增加失败次数
   - 检查是否达到最大失败次数
   - 如果达到，锁定账户
5. 如果密码正确：
   - 重置失败次数
   - 清除锁定状态
   - 生成 Token

### 自动解锁机制

在 `IsAccountLocked` 方法中实现：

```go
// 如果 locked_until 为 nil 或已过期，则账户未锁定
if user.LockedUntil == nil || time.Now().After(*user.LockedUntil) {
    // 如果锁定时间已过期，自动解锁
    if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
        _ = r.UnlockAccount(ctx, tenantID, userID)
    }
    return false, nil
}
```

## 测试

运行登录失败限制功能的测试：

```bash
go test -v ./internal/service/auth -run TestLoginFailureLimit
```

测试覆盖以下场景：

- 失败次数递增
- 达到最大次数后账户锁定
- 锁定期间无法登录
- 手动解锁账户
- 成功登录后重置失败次数

## 相关文件

- `internal/model/auth.go` - 用户模型（新增字段）
- `internal/repository/user_repository.go` - 用户数据访问（新增方法）
- `internal/service/auth/auth_service.go` - 认证服务（实现逻辑）
- `internal/api/handler/auth_handler.go` - 认证处理器（解锁接口）
- `internal/api/routes/auth_routes.go` - 路由配置
- `internal/service/auth/login_limit_test.go` - 单元测试
