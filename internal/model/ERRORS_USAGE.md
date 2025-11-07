# 统一错误处理使用指南

## 概述

本文档介绍如何使用 `internal/model/errors.go` 中定义的统一错误处理系统。

## 错误码设计

错误码采用 5 位数字格式：`{模块代码}{错误类型}{序号}`

### 模块代码

- `10xxx`: 通用错误
- `30xxx`: 会话管理错误
- `40xxx`: 上下文管理错误
- `50xxx`: 记忆管理错误
- `60xxx`: AI 服务错误

### 错误类型

- `x01xx`: 客户端错误（4xx HTTP 状态码）
- `x02xx`: 服务器错误（5xx HTTP 状态码）
- `x03xx`: 业务逻辑错误

## 核心结构

### AppError

```go
type AppError struct {
    Code       int    // 错误码
    Message    string // 错误消息
    Details    string // 错误详情（可选）
    HTTPStatus int    // HTTP 状态码（自动映射）
    Err        error  // 原始错误（可选）
}
```

## 使用方法

### 1. 创建新错误

#### 通用错误

```go
// 请求参数错误
err := model.NewBadRequestError("缺少必填字段 'name'")

// 未授权
err := model.NewUnauthorizedError("Token 已过期")

// 禁止访问
err := model.NewForbiddenError("无权访问该资源")

// 资源不存在
err := model.NewNotFoundError("用户不存在")

// 参数验证失败
err := model.NewValidationError("邮箱格式不正确")
```

#### 会话管理错误

```go
// 会话不存在
err := model.NewSessionNotFoundError("session123")

// 会话已过期
err := model.NewSessionExpiredError("session123")

// 会话访问拒绝
err := model.NewSessionAccessDeniedError()

// 会话创建失败
err := model.NewSessionCreateFailedError(dbErr)
```

#### 上下文管理错误

```go
// 上下文构建失败
err := model.NewContextBuildFailedError(buildErr)

// Token 超出限制
err := model.NewTokenExceededError(5000, 4096)

// 上下文不存在
err := model.NewContextNotFoundError("context456")

// 上下文无效
err := model.NewContextInvalidError("上下文格式错误")
```

#### 记忆管理错误

```go
// 记忆不存在
err := model.NewMemoryNotFoundError("memory789")

// 向量生成失败
err := model.NewVectorGenerationFailedError(vectorErr)

// 记忆存储失败
err := model.NewMemoryStoreFailedError(storeErr)

// 记忆检索失败
err := model.NewMemoryRetrieveFailedError(retrieveErr)
```

#### AI 服务错误

```go
// AI 服务超时
err := model.NewAIServiceTimeoutError(timeoutErr)

// AI 服务错误
err := model.NewAIServiceError(apiErr)

// 配额超出限制
err := model.NewQuotaExceededError("每日请求次数已达上限")

// 模型不可用
err := model.NewModelNotAvailableError("gpt-4")

// 提供商不存在
err := model.NewProviderNotFoundError("openai")

// 模型不存在
err := model.NewModelNotFoundError("gpt-4")

// 流式响应失败
err := model.NewStreamingFailedError(streamErr)
```

### 2. 包装现有错误

```go
// 包装数据库错误
dbErr := db.Query(...)
if dbErr != nil {
    return model.WrapError(
        model.ErrCodeInternalError,
        model.MsgInternalError,
        dbErr,
    )
}

// 包装错误并添加详情
apiErr := callExternalAPI(...)
if apiErr != nil {
    return model.WrapErrorWithDetails(
        model.ErrCodeAIServiceError,
        model.MsgAIServiceError,
        "调用 OpenAI API 失败",
        apiErr,
    )
}
```

### 3. 在服务层使用

```go
// internal/service/session_service.go
package service

import (
    "context"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/internal/logger"
)

func (s *sessionService) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
    // 查询会话
    session, err := s.repo.FindByID(ctx, sessionID)
    if err != nil {
        // 记录错误日志
        logger.ErrorContext(ctx, "查询会话失败", logger.Fields{
            "sessionId": sessionID,
            "error": err.Error(),
        })
        
        // 返回统一错误
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, model.NewSessionNotFoundError(sessionID)
        }
        return nil, model.NewInternalError(err)
    }
    
    // 验证权限
    claims := middleware.GetJWTClaims(ctx)
    if session.UserID != claims.Subject {
        logger.WarnContext(ctx, "会话访问被拒绝", logger.Fields{
            "sessionId": sessionID,
            "userId": claims.Subject,
            "sessionUserId": session.UserID,
        })
        return nil, model.NewSessionAccessDeniedError()
    }
    
    return session, nil
}
```

### 4. 在 Handler 层处理错误

```go
// internal/handler/session_handler.go
package handler

import (
    "net/http"
    "genkit-ai-service/internal/model"
    "genkit-ai-service/pkg/response"
)

func (h *sessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    
    session, err := h.service.GetSession(r.Context(), sessionID)
    if err != nil {
        // 处理 AppError
        if appErr, ok := err.(*model.AppError); ok {
            response.Error(w, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
            return
        }
        
        // 处理未知错误
        response.Error(w, http.StatusInternalServerError, 
            model.ErrCodeInternalError, 
            model.MsgInternalError, 
            "")
        return
    }
    
    response.Success(w, session)
}
```

### 5. 错误解包

```go
import "errors"

// 检查是否是特定类型的错误
err := someOperation()
if appErr, ok := err.(*model.AppError); ok {
    fmt.Printf("错误码: %d\n", appErr.Code)
    fmt.Printf("HTTP 状态码: %d\n", appErr.HTTPStatus)
}

// 使用 errors.Unwrap 获取原始错误
originalErr := errors.Unwrap(err)

// 使用 errors.Is 检查错误链
if errors.Is(err, gorm.ErrRecordNotFound) {
    // 处理记录不存在的情况
}
```

## HTTP 状态码映射

错误码会自动映射到相应的 HTTP 状态码：

| 错误码范围 | HTTP 状态码 | 说明 |
|-----------|------------|------|
| 10000 | 200 | 成功 |
| 10101 | 400 | 请求参数错误 |
| 10102 | 401 | 未授权 |
| 10103 | 403 | 禁止访问 |
| 10104 | 404 | 资源不存在 |
| 10201 | 500 | 内部错误 |
| 10301 | 400 | 参数验证失败 |
| 30104 | 404 | 会话不存在 |
| 30301 | 410 | 会话已过期 |
| 30302 | 403 | 会话访问拒绝 |
| 40104 | 404 | 上下文不存在 |
| 40302 | 400 | Token 超出限制 |
| 50104 | 404 | 记忆不存在 |
| 60104 | 404 | 提供商不存在 |
| 60105 | 404 | 模型不存在 |
| 60201 | 504 | AI 服务超时 |
| 60301 | 429 | 配额超出限制 |
| 60302 | 503 | 模型不可用 |

## 最佳实践

### 1. 始终使用统一错误

❌ 不推荐：

```go
return nil, errors.New("会话不存在")
```

✅ 推荐：

```go
return nil, model.NewSessionNotFoundError(sessionID)
```

### 2. 记录错误日志

```go
if err != nil {
    logger.ErrorContext(ctx, "操作失败", logger.Fields{
        "operation": "createSession",
        "error": err.Error(),
    })
    return model.NewSessionCreateFailedError(err)
}
```

### 3. 提供有用的错误详情

❌ 不推荐：

```go
return model.NewBadRequestError("")
```

✅ 推荐：

```go
return model.NewBadRequestError("缺少必填字段 'name'")
```

### 4. 包装底层错误

```go
// 保留原始错误信息，便于调试
dbErr := db.Create(&session).Error
if dbErr != nil {
    return model.NewSessionCreateFailedError(dbErr)
}
```

### 5. 在 Handler 层统一处理错误

```go
// 创建一个错误处理中间件或辅助函数
func handleError(w http.ResponseWriter, err error) {
    if appErr, ok := err.(*model.AppError); ok {
        response.Error(w, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
        return
    }
    response.Error(w, http.StatusInternalServerError, 
        model.ErrCodeInternalError, 
        model.MsgInternalError, 
        "")
}
```

## 测试

```go
func TestSessionService_GetSession(t *testing.T) {
    // 测试会话不存在的情况
    err := service.GetSession(ctx, "nonexistent")
    
    // 验证错误类型
    appErr, ok := err.(*model.AppError)
    assert.True(t, ok)
    assert.Equal(t, model.ErrCodeSessionNotFound, appErr.Code)
    assert.Equal(t, http.StatusNotFound, appErr.HTTPStatus)
}
```

## 扩展错误码

如果需要添加新的错误码：

1. 在 `internal/model/errors.go` 中定义错误码常量
2. 定义对应的错误消息常量
3. 在 `getHTTPStatus` 函数中添加 HTTP 状态码映射
4. 创建错误构造函数
5. 添加单元测试

示例：

```go
// 1. 定义错误码
const (
    ErrCodeCustomError = 70101 // 自定义错误
)

// 2. 定义错误消息
const (
    MsgCustomError = "自定义错误"
)

// 3. 在 getHTTPStatus 中添加映射
case code >= 70000 && code < 80000:
    return http.StatusBadRequest

// 4. 创建构造函数
func NewCustomError(details string) *AppError {
    return NewAppError(ErrCodeCustomError, MsgCustomError, details)
}

// 5. 添加测试
func TestNewCustomError(t *testing.T) {
    err := NewCustomError("测试详情")
    assert.Equal(t, ErrCodeCustomError, err.Code)
    assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
}
```

## 总结

统一错误处理系统提供了：

- ✅ 标准化的错误码和消息
- ✅ 自动的 HTTP 状态码映射
- ✅ 错误链支持（Unwrap）
- ✅ 详细的错误信息
- ✅ 类型安全的错误处理
- ✅ 便于日志记录和监控

遵循本指南，可以确保整个系统的错误处理保持一致性和可维护性。
