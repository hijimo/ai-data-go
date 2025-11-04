// Package middleware 提供错误处理中间件
package middleware

import (
	"context"
	stdErrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// ErrorHandler 错误处理中间件
// 捕获并统一处理所有错误，返回标准格式的错误响应
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 执行后续处理器
		c.Next()

		// 检查是否有错误
		if len(c.Errors) == 0 {
			return
		}

		// 获取最后一个错误
		err := c.Errors.Last().Err

		// 处理错误并返回响应
		handleError(c, err)
	}
}

// handleError 处理错误并返回适当的响应
func handleError(c *gin.Context, err error) {
	ctx := c.Request.Context()

	// 1. 检查是否是 AppError
	var appErr *errors.AppError
	if stdErrors.As(err, &appErr) {
		handleAppError(c, ctx, appErr)
		return
	}

	// 2. 检查是否是 GORM 错误
	if stdErrors.Is(err, gorm.ErrRecordNotFound) {
		appErr = errors.NewNotFoundError("资源不存在")
		handleAppError(c, ctx, appErr)
		return
	}

	// 3. 检查是否是上下文取消错误
	if stdErrors.Is(err, context.Canceled) {
		appErr = errors.NewContextCancelledError()
		handleAppError(c, ctx, appErr)
		return
	}

	// 4. 检查是否是上下文超时错误
	if stdErrors.Is(err, context.DeadlineExceeded) {
		appErr = errors.NewServiceUnavailableError("请求超时")
		handleAppError(c, ctx, appErr)
		return
	}

	// 5. 未知错误，返回内部错误
	appErr = errors.NewInternalError(err)
	handleAppError(c, ctx, appErr)
}

// handleAppError 处理 AppError 并返回响应
func handleAppError(c *gin.Context, ctx context.Context, appErr *errors.AppError) {
	// 记录错误日志
	logError(ctx, appErr)

	// 确定 HTTP 状态码
	httpStatus := getHTTPStatus(appErr.Code)

	// 返回错误响应
	response.Error(c, httpStatus, appErr.Code, appErr.Message)
}

// logError 记录错误日志
func logError(ctx context.Context, appErr *errors.AppError) {
	// 根据错误码确定日志级别
	if appErr.Code >= 500 {
		// 服务器错误，记录 ERROR 级别
		logger.ErrorContext(ctx, appErr.Message, logger.Fields{
			"error_code": appErr.Code,
			"error":      appErr.Err,
		})
	} else if appErr.Code >= 400 {
		// 客户端错误，记录 WARN 级别
		logger.WarnContext(ctx, appErr.Message, logger.Fields{
			"error_code": appErr.Code,
		})
	}
}

// getHTTPStatus 根据错误码获取 HTTP 状态码
func getHTTPStatus(code int) int {
	switch {
	case code >= 600 && code < 700:
		// Genkit 会话管理错误，映射到相应的 HTTP 状态码
		return mapGenkitErrorToHTTP(code)
	case code >= 500 && code < 600:
		// 服务器错误
		if code == errors.CodeServiceUnavailable {
			return http.StatusServiceUnavailable
		}
		return http.StatusInternalServerError
	case code == errors.CodeNotFound:
		return http.StatusNotFound
	case code == errors.CodeForbidden:
		return http.StatusForbidden
	case code == errors.CodeUnauthorized:
		return http.StatusUnauthorized
	case code == errors.CodeValidationError:
		return http.StatusUnprocessableEntity
	case code >= 400 && code < 500:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// mapGenkitErrorToHTTP 将 Genkit 错误码映射到 HTTP 状态码
func mapGenkitErrorToHTTP(code int) int {
	switch {
	case code >= 600 && code < 610:
		// 上下文管理错误
		if code == errors.CodeTokenExceeded { // Token 超限
			return http.StatusRequestEntityTooLarge
		}
		return http.StatusInternalServerError
	case code >= 610 && code < 620:
		// 记忆管理错误
		if code == errors.CodeMemoryNotFound { // 记忆不存在
			return http.StatusNotFound
		}
		return http.StatusInternalServerError
	case code >= 620 && code < 630:
		// 查询分类错误
		return http.StatusInternalServerError
	case code >= 630 && code < 640:
		// Token 管理错误
		if code == errors.CodeTokenBudgetExceeded { // Token 预算超限
			return http.StatusPaymentRequired
		}
		return http.StatusInternalServerError
	case code >= 640 && code < 650:
		// 对话生成错误
		if code == errors.CodeModelConfigInvalid { // 模型配置无效
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	case code >= 650 && code < 660:
		// 批量处理错误
		if code == errors.CodeBatchPartialFailure { // 批量部分失败
			return http.StatusMultiStatus
		}
		return http.StatusInternalServerError
	case code >= 660 && code < 670:
		// 健康检查错误
		return http.StatusInternalServerError
	case code >= 670 && code < 680:
		// 降级和熔断错误
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// AbortWithError 中止请求并记录错误
// 这是一个辅助函数，用于在处理器中快速返回错误
func AbortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}

// AbortWithAppError 中止请求并记录 AppError
func AbortWithAppError(c *gin.Context, appErr *errors.AppError) {
	c.Error(appErr)
	c.Abort()
}
