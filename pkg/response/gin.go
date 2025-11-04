package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
)

// Success 返回成功响应（Gin版本）
func Success[T any](c *gin.Context, message string, data T) {
	resp := SuccessWithMessageContext(c.Request.Context(), message, &data)
	c.JSON(http.StatusOK, resp)
}

// Error 返回错误响应（Gin版本）
func Error(c *gin.Context, statusCode int, code int, message string) {
	resp := model.ResponseData[any]{
		Code:    code,
		Message: message,
		TraceID: getTraceID(c.Request.Context()),
	}
	
	c.JSON(statusCode, resp)
}

// ErrorWithAppError 使用AppError返回错误响应（Gin版本）
func ErrorWithAppError(c *gin.Context, appErr *errors.AppError) {
	resp := FromAppErrorContext[any](c.Request.Context(), appErr)
	
	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, resp)
}

// PaginationSuccess 返回分页成功响应（Gin版本）
func PaginationSuccess[T any](c *gin.Context, message string, data T, pageNo, pageSize, totalCount int) {
	resp := PaginationWithMessageContext(c.Request.Context(), message, data, pageNo, pageSize, totalCount)
	c.JSON(http.StatusOK, resp)
}

// PaginationError 返回分页错误响应（Gin版本）
func PaginationError[T any](c *gin.Context, statusCode int, message string) {
	resp := PaginationErrorContext[T](c.Request.Context(), statusCode, message)
	c.JSON(statusCode, resp)
}
