# API Handler 安全指南

## 概述

本文档描述了 API Handler 层必须遵循的安全措施和最佳实践，以确保系统的安全性和可靠性。

## 1. 输入验证

### 1.1 参数类型验证

所有 Handler 必须使用 `validator` 包进行参数验证：

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Username string `json:"username" validate:"required,min=3,max=50"`
    Age      int    `json:"age" validate:"omitempty,min=0,max=150"`
}

// 在 Handler 中验证
if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
    h.logger.Warn("参数验证失败", logger.Fields{"errors": validationErrors})
    h.writeValidationErrorResponse(w, r, validationErrors)
    return
}
```

### 1.2 长度限制

**字符串字段**：

- 用户输入的文本内容：`max=2000`
- 用户查询：`max=2000`
- 邮箱地址：`max=255`
- 用户名：`max=50`
- 摘要内容：`max=10000`

**数组字段**：

- 消息ID列表：`max=100`
- 标签列表：`max=20`

**数值字段**：

- Token数量：`min=100,max=32000`
- 页码：`min=1`
- 每页大小：`min=1,max=100`

### 1.3 格式验证

使用 validator 标签进行格式验证：

```go
type Request struct {
    Email     string `validate:"required,email"`
    URL       string `validate:"omitempty,url"`
    UUID      string `validate:"required,uuid"`
    IPAddress string `validate:"omitempty,ip"`
    Enum      string `validate:"oneof=option1 option2 option3"`
}
```

### 1.4 UUID 验证

所有 UUID 参数必须进行格式验证：

```go
// 从请求中获取 UUID
sessionID := req.SessionID

// 验证 UUID 格式
sessionUUID, err := uuid.Parse(sessionID)
if err != nil {
    h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": sessionID})
    h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
    return
}
```

## 2. SQL 注入防护

### 2.1 使用参数化查询

**正确示例**：

```go
// ✅ 使用 GORM 的参数化查询
db.Where("tenant_id = ? AND is_deleted = ?", tenantID, false).Find(&users)

// ✅ 使用命名参数
db.Where("email = @email AND tenant_id = @tenantId", 
    sql.Named("email", email),
    sql.Named("tenantId", tenantID)).First(&user)
```

**错误示例**：

```go
// ❌ 字符串拼接（容易受到 SQL 注入攻击）
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
db.Raw(query).Scan(&users)
```

### 2.2 GORM 安全实践

- 始终使用 GORM 的查询构建器
- 避免使用 `db.Raw()` 除非绝对必要
- 如果必须使用原始 SQL，使用参数化查询

## 3. XSS 防护

### 3.1 输出转义

在返回用户输入的内容时，确保进行适当的转义：

```go
import "html"

// 转义 HTML 特殊字符
safeContent := html.EscapeString(userInput)
```

### 3.2 Content-Type 设置

所有 JSON 响应必须设置正确的 Content-Type：

```go
w.Header().Set("Content-Type", "application/json")
```

### 3.3 避免直接渲染 HTML

- 不要直接返回 HTML 内容
- 使用 JSON 格式返回数据
- 前端负责渲染和转义

## 4. 速率限制

### 4.1 基于 IP 的速率限制

用于防止单个 IP 地址的滥用：

```go
// 在路由中应用 IP 速率限制
router.Use(rateLimiter.RateLimitByIP())
```

**默认配置**：

- 容量：20 个请求
- 补充率：每秒 10 个请求

### 4.2 基于租户的速率限制

用于防止单个租户的滥用：

```go
// 在路由中应用租户速率限制
router.Use(rateLimiter.RateLimitByTenant())
```

**默认配置**：

- 容量：100 个请求
- 补充率：每秒 50 个请求

### 4.3 组合速率限制

同时应用 IP 和租户限制：

```go
// 同时应用两种限制
router.Use(rateLimiter.RateLimit())
```

### 4.4 自定义速率限制

针对特定端点的自定义限制：

```go
config := &middleware.RateLimiterConfig{
    IPCapacity:        10,  // 更严格的限制
    IPRefillRate:      5,
    TenantCapacity:    50,
    TenantRefillRate:  25,
    EnableIPLimit:     true,
    EnableTenantLimit: true,
}

customLimiter := middleware.NewRateLimiterMiddleware(config, log)
router.Use(customLimiter.RateLimit())
```

## 5. 认证和授权

### 5.1 JWT 验证

所有需要认证的端点必须应用 JWT 中间件：

```go
router.Use(middleware.JWTAuth())
```

### 5.2 获取用户信息

从上下文中安全地获取用户信息：

```go
// 获取用户ID
userID, ok := middleware.GetAuthUserID(ctx)
if !ok || userID == "" {
    h.logger.Warn("缺少用户ID")
    h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少用户认证信息"))
    return
}

// 获取租户ID
tenantID, ok := middleware.GetTenantID(ctx)
if !ok || tenantID == "" {
    h.logger.Warn("缺少租户ID")
    h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少租户信息"))
    return
}

// 验证 UUID 格式
if _, err := uuid.Parse(userID); err != nil {
    h.logger.Warn("用户ID格式无效", logger.Fields{"userId": userID})
    h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID格式无效"))
    return
}
```

### 5.3 租户隔离验证

在服务层验证租户访问权限（参考 `multi-tenant-access-control.md`）：

```go
// 服务层验证
func (s *Service) GetResource(ctx context.Context, resourceID string) (*Resource, error) {
    resource, err := s.repo.FindByID(ctx, resourceID)
    if err != nil {
        return nil, err
    }
    
    // 获取当前用户的租户ID
    claims := middleware.GetJWTClaims(ctx)
    
    // 平台管理员可以访问所有资源
    if hasRole(claims, model.RoleSystemAdmin) {
        return resource, nil
    }
    
    // 租户管理员只能访问自己租户的资源
    if resource.TenantID != claims.TenantID {
        return nil, errors.NewForbiddenError("权限不足：无法访问其他租户的资源")
    }
    
    return resource, nil
}
```

## 6. 错误处理

### 6.1 不泄露敏感信息

错误消息不应包含：

- 数据库结构信息
- 内部路径
- 堆栈跟踪（生产环境）
- 其他租户的数据

**正确示例**：

```go
// ✅ 通用错误消息
if err != nil {
    h.logger.ErrorContext(ctx, "操作失败", logger.Fields{"error": err})
    h.writeErrorResponse(w, r, errors.NewInternalError(err))
    return
}
```

**错误示例**：

```go
// ❌ 泄露内部信息
if err != nil {
    h.writeErrorResponse(w, r, errors.New(500, fmt.Sprintf("数据库错误: %v", err)))
    return
}
```

### 6.2 统一错误响应

使用统一的错误响应格式：

```go
func (h *Handler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
    ctx := r.Context()
    resp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)
    
    statusCode := http.StatusInternalServerError
    switch appErr.Code {
    case errors.CodeBadRequest:
        statusCode = http.StatusBadRequest
    case errors.CodeUnauthorized:
        statusCode = http.StatusUnauthorized
    case errors.CodeForbidden:
        statusCode = http.StatusForbidden
    case errors.CodeNotFound:
        statusCode = http.StatusNotFound
    case errors.CodeTooManyRequests:
        statusCode = http.StatusTooManyRequests
    }
    
    h.writeJSONResponse(w, statusCode, resp)
}
```

## 7. 日志记录

### 7.1 记录安全事件

记录所有安全相关的事件：

```go
// 记录认证失败
h.logger.WarnContext(ctx, "认证失败", logger.Fields{
    "ip":     c.ClientIP(),
    "path":   c.Request.URL.Path,
    "reason": "无效的令牌",
})

// 记录权限验证失败
h.logger.WarnContext(ctx, "权限验证失败", logger.Fields{
    "userId":   userID,
    "tenantId": tenantID,
    "resource": resourceID,
    "action":   "read",
})

// 记录速率限制触发
h.logger.WarnContext(ctx, "速率限制触发", logger.Fields{
    "ip":     clientIP,
    "path":   path,
    "method": method,
})
```

### 7.2 不记录敏感信息

日志中不应包含：

- 密码
- JWT 令牌
- API 密钥
- 信用卡号
- 个人身份信息（PII）

## 8. 安全头设置

### 8.1 必需的安全头

```go
// 在中间件中设置安全头
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Next()
    }
}
```

## 9. 检查清单

在实现新的 Handler 时，请确保：

- [ ] 使用 validator 进行参数验证
- [ ] 设置适当的字段长度限制
- [ ] 验证所有 UUID 格式
- [ ] 使用参数化查询（GORM）
- [ ] 设置正确的 Content-Type
- [ ] 应用适当的速率限制
- [ ] 验证 JWT 令牌
- [ ] 从上下文获取用户和租户信息
- [ ] 在服务层验证租户隔离
- [ ] 使用统一的错误响应格式
- [ ] 记录安全事件
- [ ] 不在日志中记录敏感信息
- [ ] 不在错误消息中泄露内部信息

## 10. 参考资料

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- 项目内部文档：`multi-tenant-access-control.md`
