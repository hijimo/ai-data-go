# Logger 使用指南

## 概述

项目已经实现了完整的结构化日志系统，支持 JSON 和文本格式，支持从上下文自动提取追踪信息。

## 功能特性

- ✅ 结构化日志（JSON/Text 格式）
- ✅ 多级别日志（DEBUG、INFO、WARN、ERROR）
- ✅ 上下文感知（自动提取 traceId、sessionId、requestId、userId）
- ✅ 文件持久化（按天轮转）
- ✅ 调用者信息（DEBUG 级别）
- ✅ 线程安全
- ✅ 预设字段支持

## 基本使用

### 1. 初始化日志系统

在 `main.go` 中初始化：

```go
import "genkit-ai-service/internal/logger"

func main() {
    // 方式1：输出到控制台
    logger.Init("info", "json")
    
    // 方式2：输出到文件（带日志轮转）
    err := logger.InitWithFile("info", "json", "./logs", true)
    if err != nil {
        panic(err)
    }
}
```

### 2. 基本日志记录

```go
import "genkit-ai-service/internal/logger"

// 信息日志
logger.Info("用户登录成功", logger.Fields{
    "user_id": "123",
    "ip": "192.168.1.1",
})

// 警告日志
logger.Warn("缓存未命中", logger.Fields{
    "cache_key": "session:123",
})

// 错误日志
logger.Error("数据库连接失败", logger.Fields{
    "error": err.Error(),
    "retry_count": 3,
})

// 调试日志（包含调用者信息）
logger.Debug("处理请求", logger.Fields{
    "request_id": "req-123",
})
```

### 3. 上下文日志（推荐）

使用上下文日志可以自动提取 traceId、sessionId 等信息：

```go
import (
    "context"
    "genkit-ai-service/internal/logger"
)

func MyService(ctx context.Context) error {
    // 自动从上下文提取 traceId、sessionId、requestId、userId
    logger.InfoContext(ctx, "开始处理请求", logger.Fields{
        "operation": "create_user",
    })
    
    // 执行业务逻辑
    if err := doSomething(); err != nil {
        logger.ErrorContext(ctx, "处理失败", logger.Fields{
            "error": err.Error(),
        })
        return err
    }
    
    logger.InfoContext(ctx, "处理完成", logger.Fields{
        "duration_ms": 150,
    })
    
    return nil
}
```

### 4. 带预设字段的日志记录器

创建带有预设字段的日志记录器，避免重复传递相同字段：

```go
// 创建带预设字段的 logger
serviceLogger := logger.WithFields(logger.Fields{
    "service": "user_service",
    "version": "1.0.0",
})

// 使用预设字段的 logger
serviceLogger.Info("服务启动", logger.Fields{
    "port": 8080,
})
// 输出: {"service":"user_service","version":"1.0.0","port":8080,...}

// 也可以结合上下文使用
serviceLogger.InfoContext(ctx, "处理请求", logger.Fields{
    "user_id": "123",
})
```

## 在不同层级使用

### Handler 层

```go
func (h *UserHandler) HandleCreate(c *gin.Context) {
    ctx := c.Request.Context()
    
    logger.InfoContext(ctx, "收到创建用户请求", logger.Fields{
        "endpoint": "/api/v1/users",
        "method": "POST",
    })
    
    // 业务逻辑
    user, err := h.userService.Create(ctx, req)
    if err != nil {
        logger.ErrorContext(ctx, "创建用户失败", logger.Fields{
            "error": err.Error(),
            "email": req.Email,
        })
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    logger.InfoContext(ctx, "创建用户成功", logger.Fields{
        "user_id": user.ID,
        "email": user.Email,
    })
    
    c.JSON(200, user)
}
```

### Service 层

```go
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
    logger.InfoContext(ctx, "开始创建用户", logger.Fields{
        "email": req.Email,
        "tenant_id": req.TenantID,
    })
    
    // 权限验证
    if err := s.validateAccess(ctx, req.TenantID); err != nil {
        logger.WarnContext(ctx, "权限验证失败", logger.Fields{
            "error": err.Error(),
            "tenant_id": req.TenantID,
        })
        return nil, err
    }
    
    // 创建用户
    user, err := s.repo.Create(ctx, req)
    if err != nil {
        logger.ErrorContext(ctx, "数据库创建用户失败", logger.Fields{
            "error": err.Error(),
            "email": req.Email,
        })
        return nil, err
    }
    
    logger.InfoContext(ctx, "用户创建成功", logger.Fields{
        "user_id": user.ID,
        "email": user.Email,
    })
    
    return user, nil
}
```

### Repository 层

```go
func (r *UserRepository) Create(ctx context.Context, user *User) error {
    logger.DebugContext(ctx, "执行数据库插入", logger.Fields{
        "table": "users",
        "email": user.Email,
    })
    
    result := r.db.WithContext(ctx).Create(user)
    if result.Error != nil {
        logger.ErrorContext(ctx, "数据库插入失败", logger.Fields{
            "error": result.Error.Error(),
            "table": "users",
        })
        return result.Error
    }
    
    logger.DebugContext(ctx, "数据库插入成功", logger.Fields{
        "user_id": user.ID,
        "rows_affected": result.RowsAffected,
    })
    
    return nil
}
```

### Flow 层

```go
func contextBuildFlow(contextSvc service.ContextService) func(context.Context, ContextBuildInput) (ContextBuildOutput, error) {
    return func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
        logger.InfoContext(ctx, "开始执行上下文构建Flow", logger.Fields{
            "session_id": input.SessionID,
            "max_tokens": input.MaxTokens,
            "strategy": input.Strategy,
        })
        
        result, err := contextSvc.BuildContext(ctx, convertInput(input))
        if err != nil {
            logger.ErrorContext(ctx, "上下文构建失败", logger.Fields{
                "error": err.Error(),
                "session_id": input.SessionID,
            })
            return ContextBuildOutput{}, err
        }
        
        logger.InfoContext(ctx, "上下文构建成功", logger.Fields{
            "session_id": input.SessionID,
            "total_tokens": result.TotalTokens,
            "quality_score": result.QualityScore,
        })
        
        return convertOutput(result), nil
    }
}
```

## 审计日志

记录权限验证失败等安全相关事件：

```go
func (s *UserService) validateAccess(ctx context.Context, tenantID string) error {
    claims := middleware.GetJWTClaims(ctx)
    
    // 租户管理员只能访问自己租户的资源
    if hasRole(claims, model.RoleTenantAdmin) && claims.TenantID != tenantID {
        logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的数据", logger.Fields{
            "event": "permission_denied",
            "reason": "cross_tenant_access",
            "user_id": claims.Subject,
            "user_tenant_id": claims.TenantID,
            "target_tenant_id": tenantID,
            "user_role": model.RoleTenantAdmin,
        })
        return errors.NewForbiddenError("权限不足：无法访问其他租户的数据")
    }
    
    return nil
}
```

## 日志格式

### JSON 格式（默认）

```json
{
  "timestamp": "2025-11-07T13:45:45Z",
  "level": "INFO",
  "message": "用户登录成功",
  "fields": {
    "traceId": "abc123",
    "sessionId": "sess-456",
    "user_id": "123",
    "ip": "192.168.1.1"
  }
}
```

### Text 格式

```
2025-11-07T13:45:45Z [INFO] 用户登录成功 traceId=abc123 sessionId=sess-456 user_id=123 ip=192.168.1.1
```

## 日志级别

### DEBUG

- 详细的调试信息
- 包含调用者信息（文件名和行号）
- 仅在开发环境使用

```go
logger.DebugContext(ctx, "查询参数", logger.Fields{
    "sql": "SELECT * FROM users WHERE id = ?",
    "params": []interface{}{123},
})
```

### INFO

- 一般信息
- 记录重要的业务操作
- 生产环境默认级别

```go
logger.InfoContext(ctx, "用户创建成功", logger.Fields{
    "user_id": "123",
})
```

### WARN

- 警告信息
- 可能的问题，但不影响系统运行
- 需要关注但不需要立即处理

```go
logger.WarnContext(ctx, "缓存未命中", logger.Fields{
    "cache_key": "session:123",
})
```

### ERROR

- 错误信息
- 影响功能的错误
- 需要立即关注和处理

```go
logger.ErrorContext(ctx, "数据库连接失败", logger.Fields{
    "error": err.Error(),
    "retry_count": 3,
})
```

## 上下文字段

Logger 会自动从上下文中提取以下字段：

- **traceId**: 请求追踪ID（由 middleware 设置）
- **sessionId**: 会话ID
- **requestId**: 请求ID
- **userId**: 用户ID

### 设置上下文字段

在 middleware 或 handler 中设置：

```go
// 在 middleware 中设置 traceId
ctx = context.WithValue(ctx, "traceId", generateTraceID())

// 设置其他字段
ctx = context.WithValue(ctx, logger.SessionIDKey, sessionID)
ctx = context.WithValue(ctx, logger.RequestIDKey, requestID)
ctx = context.WithValue(ctx, logger.UserIDKey, userID)
```

## 文件日志

### 启用文件日志

```go
// 启用文件日志，同时输出到控制台
err := logger.InitWithFile("info", "json", "./logs", true)

// 仅输出到文件
err := logger.InitWithFile("info", "json", "./logs", false)
```

### 日志文件命名

日志文件按天轮转，命名格式：`app-YYYY-MM-DD.log`

```
logs/
├── app-2025-11-07.log
├── app-2025-11-08.log
└── app-2025-11-09.log
```

### 日志轮转

- 自动按天轮转
- 每天 00:00 创建新文件
- 旧文件自动保留

## 性能考虑

### 1. 避免在循环中记录大量日志

```go
// ❌ 不推荐
for _, item := range items {
    logger.Debug("处理项目", logger.Fields{"item": item})
}

// ✅ 推荐
logger.Debug("开始批量处理", logger.Fields{"count": len(items)})
for _, item := range items {
    // 处理逻辑
}
logger.Debug("批量处理完成", logger.Fields{"count": len(items)})
```

### 2. 使用合适的日志级别

```go
// 生产环境设置为 INFO
logger.Init("info", "json")

// 开发环境可以设置为 DEBUG
logger.Init("debug", "json")
```

### 3. 避免记录敏感信息

```go
// ❌ 不要记录密码、Token 等敏感信息
logger.Info("用户登录", logger.Fields{
    "password": user.Password, // 危险！
    "token": token,            // 危险！
})

// ✅ 只记录必要的非敏感信息
logger.Info("用户登录", logger.Fields{
    "user_id": user.ID,
    "email": user.Email,
})
```

## 测试

### 创建测试用 Logger

```go
import "genkit-ai-service/internal/logger"

func TestMyFunction(t *testing.T) {
    // 创建不输出的 logger
    testLogger := logger.NewTestLogger()
    
    // 或者创建输出到缓冲区的 logger
    var buf bytes.Buffer
    testLogger := logger.New(logger.DebugLevel, logger.JSONFormat, &buf)
    
    // 测试逻辑
    // ...
    
    // 验证日志输出
    output := buf.String()
    if !strings.Contains(output, "expected message") {
        t.Error("日志输出不符合预期")
    }
}
```

## 最佳实践

### 1. 使用上下文日志

始终使用 `*Context` 方法，以便自动提取追踪信息：

```go
// ✅ 推荐
logger.InfoContext(ctx, "操作成功", logger.Fields{"key": "value"})

// ❌ 不推荐（除非确实不需要上下文信息）
logger.Info("操作成功", logger.Fields{"key": "value"})
```

### 2. 记录关键操作

记录所有重要的业务操作：

- 用户认证和授权
- 数据创建、更新、删除
- 外部服务调用
- 错误和异常

### 3. 使用结构化字段

使用 `logger.Fields` 而不是在消息中拼接：

```go
// ✅ 推荐
logger.InfoContext(ctx, "用户创建成功", logger.Fields{
    "user_id": user.ID,
    "email": user.Email,
})

// ❌ 不推荐
logger.InfoContext(ctx, fmt.Sprintf("用户创建成功: %s (%s)", user.ID, user.Email))
```

### 4. 记录错误上下文

记录错误时，提供足够的上下文信息：

```go
logger.ErrorContext(ctx, "创建用户失败", logger.Fields{
    "error": err.Error(),
    "email": req.Email,
    "tenant_id": req.TenantID,
    "retry_count": retryCount,
})
```

### 5. 使用一致的字段名

在整个项目中使用一致的字段名：

- `user_id` 而不是 `userId` 或 `userID`
- `tenant_id` 而不是 `tenantId`
- `session_id` 而不是 `sessionId`
- `error` 用于错误消息
- `duration_ms` 用于时长（毫秒）

## 总结

项目的 logger 实现已经非常完善，提供了：

- ✅ 结构化日志
- ✅ 上下文感知
- ✅ 文件持久化
- ✅ 日志轮转
- ✅ 多级别支持
- ✅ 线程安全
- ✅ 高性能

直接使用即可，无需额外实现。在所有服务层、Handler 层和 Flow 层中都已经正确使用了 logger。
