# 租户ID上下文获取 - 快速参考

## 概述

AI Service 现在能够从请求上下文中获取当前用户的租户ID，用于多租户隔离和模型配置查询。

## 使用方法

### 在 Service 层获取租户ID

```go
import (
    authservice "genkit-ai-service/internal/service/auth"
)

func (s *yourService) YourMethod(ctx context.Context, req *Request) (*Response, error) {
    // 从上下文中获取 JWT Claims
    claims, ok := authservice.GetJWTClaimsFromContext(ctx)
    if !ok || claims == nil {
        return nil, errors.NewUnauthorizedError("身份认证信息缺失")
    }

    // 获取租户ID
    tenantID := claims.TenantID
    if tenantID == "" {
        return nil, errors.NewUnauthorizedError("租户信息缺失")
    }

    // 使用租户ID
    // ...
}
```

## JWT Claims 结构

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    TenantID    string   `json:"tid"`           // 租户ID
    DisplayName string   `json:"displayName"`   // 用户显示名称
    Roles       []string `json:"roles"`         // 用户角色
    Scopes      []string `json:"scopes"`        // 权限范围
}
```

## 上下文传递流程

```
HTTP Request
    ↓
JWT 中间件 (jwt_auth.go)
    ↓ 解析 JWT token
    ↓ 验证 token
    ↓ 提取 claims
    ↓ 存储到上下文
    ↓
Handler 层
    ↓
Service 层
    ↓ GetJWTClaimsFromContext()
    ↓ 提取 TenantID
    ↓
使用 TenantID
```

## 错误处理

### 缺少 JWT Claims

```go
claims, ok := authservice.GetJWTClaimsFromContext(ctx)
if !ok || claims == nil {
    // 返回 401 Unauthorized
    return nil, errors.NewUnauthorizedError("身份认证信息缺失")
}
```

### 缺少租户ID

```go
tenantID := claims.TenantID
if tenantID == "" {
    // 返回 401 Unauthorized
    return nil, errors.NewUnauthorizedError("租户信息缺失")
}
```

## 日志记录

建议记录租户ID以便追踪：

```go
logger.DebugContext(ctx, "从上下文获取租户ID", logger.Fields{
    "tenantId": tenantID,
    "userId":   claims.Subject,
})
```

## 测试

### 创建测试上下文

```go
import (
    authservice "genkit-ai-service/internal/service/auth"
)

func createTestContext() context.Context {
    claims := &model.JWTClaims{
        TenantID:    "test-tenant-id",
        DisplayName: "测试用户",
        Roles:       []string{"user"},
    }
    claims.Subject = "test-user-id"
    
    ctx := context.Background()
    ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)
    return ctx
}
```

### 测试用例示例

```go
func TestYourMethod_Success(t *testing.T) {
    service := NewYourService(...)
    
    req := &Request{
        // ...
    }
    
    ctx := createTestContext()
    resp, err := service.YourMethod(ctx, req)
    
    if err != nil {
        t.Fatalf("方法调用失败: %v", err)
    }
    
    // 验证响应
    // ...
}

func TestYourMethod_MissingTenantID(t *testing.T) {
    service := NewYourService(...)
    
    // 创建没有租户ID的 claims
    claims := &model.JWTClaims{
        DisplayName: "测试用户",
        Roles:       []string{"user"},
    }
    claims.Subject = "test-user-id"
    
    ctx := context.Background()
    ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)
    
    req := &Request{
        // ...
    }
    
    _, err := service.YourMethod(ctx, req)
    if err == nil {
        t.Fatal("期望返回错误")
    }
}
```

## 安全注意事项

1. **始终验证 JWT Claims 存在**: 不要假设上下文中一定有 claims
2. **验证租户ID非空**: 确保租户ID有效
3. **记录审计日志**: 记录租户ID用于安全审计
4. **不要信任客户端传入的租户ID**: 始终从 JWT Claims 中获取

## 相关文件

- `internal/service/auth/context_helper.go`: JWT Claims 提取函数
- `internal/model/jwt.go`: JWT Claims 数据结构
- `internal/api/middleware/jwt_auth.go`: JWT 认证中间件
- `internal/service/ai/genkit_service.go`: 使用示例

## 常见问题

### Q: 为什么不直接从中间件的上下文键获取租户ID？

A: 使用 `GetJWTClaimsFromContext()` 函数可以：

- 避免循环依赖
- 提供类型安全的访问
- 统一错误处理
- 便于测试

### Q: 如何在测试中模拟不同的租户？

A: 创建不同的 JWT Claims 并设置不同的 TenantID：

```go
func createTestContextWithTenant(tenantID string) context.Context {
    claims := &model.JWTClaims{
        TenantID:    tenantID,
        DisplayName: "测试用户",
        Roles:       []string{"user"},
    }
    claims.Subject = "test-user-id"
    
    ctx := context.Background()
    ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)
    return ctx
}
```

### Q: 租户ID的格式是什么？

A: 租户ID是 UUID 格式的字符串，例如：`"738dbb1f-83e6-4bf5-935c-f0498236440d"`

## 更新日期

2025-11-28
