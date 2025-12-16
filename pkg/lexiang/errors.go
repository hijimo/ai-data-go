// Package lexiang 提供腾讯乐享知识库 API 的 Go 客户端封装
package lexiang

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// HTTP 状态码常量
const (
	// StatusBadRequest 请求参数错误
	StatusBadRequest = 400
	// StatusUnauthorized 未授权（token 无效或过期）
	StatusUnauthorized = 401
	// StatusForbidden 权限不足
	StatusForbidden = 403
	// StatusNotFound 资源不存在
	StatusNotFound = 404
	// StatusRateLimit 请求频率超限
	StatusRateLimit = 429
	// StatusServerError 服务器内部错误
	StatusServerError = 500
)

// LexiangError 乐享 API 错误
type LexiangError struct {
	// StatusCode HTTP 状态码
	StatusCode int `json:"status_code"`
	// Code 错误代码
	Code string `json:"code"`
	// Message 错误信息
	Message string `json:"message"`
	// RawBody 原始响应体
	RawBody string `json:"raw_body,omitempty"`
}

// Error 实现 error 接口
func (e *LexiangError) Error() string {
	return fmt.Sprintf("lexiang api error: status=%d, code=%s, message=%s",
		e.StatusCode, e.Code, e.Message)
}

// apiErrorResponse 乐享 API 错误响应格式
type apiErrorResponse struct {
	Errors []struct {
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

// IsBadRequestError 判断是否为请求参数错误（400）
func IsBadRequestError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode == StatusBadRequest
	}
	return false
}

// IsUnauthorizedError 判断是否为未授权错误（401）
func IsUnauthorizedError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode == StatusUnauthorized
	}
	return false
}

// IsForbiddenError 判断是否为权限不足错误（403）
func IsForbiddenError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode == StatusForbidden
	}
	return false
}

// IsNotFoundError 判断是否为资源不存在错误（404）
func IsNotFoundError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode == StatusNotFound
	}
	return false
}

// IsRateLimitError 判断是否为请求频率超限错误（429）
func IsRateLimitError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode == StatusRateLimit
	}
	return false
}

// IsServerError 判断是否为服务器内部错误（500+）
func IsServerError(err error) bool {
	var lexErr *LexiangError
	if errors.As(err, &lexErr) {
		return lexErr.StatusCode >= StatusServerError
	}
	return false
}

// handleAPIError 解析 HTTP 响应错误并返回 LexiangError
// 该函数会读取响应体并尝试解析乐享 API 的错误格式
func handleAPIError(resp *http.Response) error {
	if resp == nil {
		return &LexiangError{
			StatusCode: 0,
			Code:       "unknown",
			Message:    "响应为空",
		}
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &LexiangError{
			StatusCode: resp.StatusCode,
			Code:       "read_error",
			Message:    fmt.Sprintf("读取响应体失败: %v", err),
		}
	}

	rawBody := string(body)

	// 尝试解析乐享 API 错误格式
	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && len(apiErr.Errors) > 0 {
		// 使用第一个错误信息
		firstErr := apiErr.Errors[0]
		message := firstErr.Detail
		if message == "" {
			message = firstErr.Title
		}
		return &LexiangError{
			StatusCode: resp.StatusCode,
			Code:       firstErr.Code,
			Message:    message,
			RawBody:    rawBody,
		}
	}

	// 无法解析时，根据状态码生成默认错误信息
	return &LexiangError{
		StatusCode: resp.StatusCode,
		Code:       getDefaultErrorCode(resp.StatusCode),
		Message:    getDefaultErrorMessage(resp.StatusCode),
		RawBody:    rawBody,
	}
}

// getDefaultErrorCode 根据状态码返回默认错误代码
func getDefaultErrorCode(statusCode int) string {
	switch statusCode {
	case StatusBadRequest:
		return "bad_request"
	case StatusUnauthorized:
		return "unauthorized"
	case StatusForbidden:
		return "forbidden"
	case StatusNotFound:
		return "not_found"
	case StatusRateLimit:
		return "rate_limit"
	default:
		if statusCode >= StatusServerError {
			return "server_error"
		}
		return "unknown"
	}
}

// getDefaultErrorMessage 根据状态码返回默认错误信息
func getDefaultErrorMessage(statusCode int) string {
	switch statusCode {
	case StatusBadRequest:
		return "请求参数错误"
	case StatusUnauthorized:
		return "未授权，token 无效或已过期"
	case StatusForbidden:
		return "权限不足"
	case StatusNotFound:
		return "资源不存在"
	case StatusRateLimit:
		return "请求频率超限，请稍后重试"
	default:
		if statusCode >= StatusServerError {
			return "服务器内部错误"
		}
		return fmt.Sprintf("未知错误，状态码: %d", statusCode)
	}
}
