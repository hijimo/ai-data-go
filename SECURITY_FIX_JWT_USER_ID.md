# 安全修复：从JWT Token获取用户ID

## 问题描述

之前的实现中，多个handler文件使用不安全的方式从HTTP Header中获取用户ID：

```go
userID := r.Header.Get("X-User-ID")
```

这种方式存在严重的安全隐患，因为客户端可以随意伪造Header中的用户ID，导致权限绕过。

## 修复方案

修改所有handler文件，改为从JWT token中安全地获取用户ID：

```go
userID, ok := middleware.GetAuthUserID(ctx)
if !ok || userID == "" {
    h.logger.Warn("缺少用户ID")
    h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少用户认证信息"))
    return
}
```

## 修复的文件

1. **internal/api/handler/session_handler.go**
   - CreateSession
   - ListSessions
   - GetSession
   - UpdateSession
   - DeleteSession
   - SearchSessions
   - PinSession
   - ArchiveSession

2. **internal/api/handler/message_handler.go**
   - SendMessage
   - GetMessage
   - ListMessages
   - AbortMessage

## 工作原理

1. JWT认证中间件（`internal/api/middleware/jwt_auth.go`）在请求处理前验证JWT token
2. 验证成功后，将用户信息（包括用户ID、租户ID、角色等）存入请求上下文
3. Handler通过 `middleware.GetAuthUserID(ctx)` 从上下文中安全地获取用户ID
4. 用户ID来自JWT token的 `Subject` claim，无法被客户端伪造

## 安全优势

- ✅ 用户ID来自经过签名验证的JWT token，无法伪造
- ✅ 符合多租户访问控制规范要求
- ✅ 统一的认证流程，便于维护
- ✅ 自动记录认证失败的审计日志

## 测试建议

1. 测试正常的JWT认证流程
2. 测试缺少JWT token时的错误处理
3. 测试JWT token过期时的错误处理
4. 测试伪造的JWT token被拒绝
5. 测试用户只能访问自己的资源
