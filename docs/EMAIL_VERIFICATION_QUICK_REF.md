# 邮箱验证功能快速参考

## API端点

### 验证邮箱

```
POST /api/v1/auth/verify-email
Content-Type: application/json

{
  "token": "验证令牌"
}
```

### 重新发送验证邮件

```
POST /api/v1/auth/resend-verification
Content-Type: application/json

{
  "tenantId": "租户ID",
  "userId": "用户ID"
}
```

## 数据库表

### email_verification_tokens

```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    token VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(320) NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

## 代码示例

### 发送验证邮件

```go
err := emailService.SendVerificationEmail(ctx, tenantID, userID, email)
```

### 验证邮箱

```go
err := emailService.VerifyEmail(ctx, token)
```

### 重新发送验证邮件

```go
err := emailService.ResendVerificationEmail(ctx, tenantID, userID)
```

## 配置

### 邮件发送器（开发环境）

```go
emailSender := auth.NewConsoleEmailSender()
```

### 邮件发送器（生产环境）

```go
smtpConfig := auth.SMTPConfig{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "noreply@example.com",
    Password: "password",
    From:     "noreply@example.com",
}
emailSender := auth.NewSMTPEmailSender(smtpConfig)
```

### EmailService初始化

```go
emailService := auth.NewEmailService(
    emailVerificationRepo,
    userRepo,
    emailSender,
    24*time.Hour, // 令牌有效期
)
```

## 测试

### 运行测试

```bash
go test ./internal/service/auth -v -run "Email"
```

### 测试覆盖

- ✅ 发送验证邮件
- ✅ 验证邮箱成功
- ✅ 使用过期令牌验证失败

## 常见错误

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| 验证令牌不存在 | 令牌无效或不存在 | 检查令牌是否正确 |
| 验证令牌已使用 | 令牌已被使用过 | 重新发送验证邮件 |
| 验证令牌已过期 | 令牌超过有效期 | 重新发送验证邮件 |
| 邮箱已验证 | 用户邮箱已验证 | 无需再次验证 |

## 工作流程

```
注册 → 生成令牌 → 发送邮件 → 用户点击链接 → 验证成功
                                    ↓
                              令牌过期/失败
                                    ↓
                              重新发送邮件
```

## 相关文档

- [完整文档](./EMAIL_VERIFICATION.md)
- [使用示例](./EMAIL_VERIFICATION_EXAMPLE.md)
- [实现总结](./EMAIL_VERIFICATION_SUMMARY.md)
