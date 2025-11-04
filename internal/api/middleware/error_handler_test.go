package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	pkgErrors "genkit-ai-service/pkg/errors"
)

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupHandler   func(*gin.Context)
		expectedStatus int
		expectedCode   int
	}{
		{
			name: "处理 AppError - BadRequest",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewBadRequestError("无效的参数"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   pkgErrors.CodeBadRequest,
		},
		{
			name: "处理 AppError - Unauthorized",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewUnauthorizedError(""))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   pkgErrors.CodeUnauthorized,
		},
		{
			name: "处理 AppError - Forbidden",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewForbiddenError(""))
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   pkgErrors.CodeForbidden,
		},
		{
			name: "处理 AppError - NotFound",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewNotFoundError(""))
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeNotFound,
		},
		{
			name: "处理 AppError - InternalError",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewInternalError(errors.New("内部错误")))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   pkgErrors.CodeInternalError,
		},
		{
			name: "处理 GORM RecordNotFound",
			setupHandler: func(c *gin.Context) {
				c.Error(gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeNotFound,
		},
		{
			name: "处理 Context Canceled",
			setupHandler: func(c *gin.Context) {
				c.Error(context.Canceled)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   pkgErrors.CodeContextCancelled,
		},
		{
			name: "处理 Context DeadlineExceeded",
			setupHandler: func(c *gin.Context) {
				c.Error(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   pkgErrors.CodeServiceUnavailable,
		},
		{
			name: "处理未知错误",
			setupHandler: func(c *gin.Context) {
				c.Error(errors.New("未知错误"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   pkgErrors.CodeInternalError,
		},
		{
			name: "处理 Token 超限错误",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewTokenExceededError(5000, 4000))
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedCode:   pkgErrors.CodeTokenExceeded,
		},
		{
			name: "处理记忆不存在错误",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewMemoryNotFoundError("mem-123"))
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   pkgErrors.CodeMemoryNotFound,
		},
		{
			name: "处理熔断器打开错误",
			setupHandler: func(c *gin.Context) {
				c.Error(pkgErrors.NewCircuitBreakerOpenError("AI服务"))
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   pkgErrors.CodeCircuitBreakerOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.Use(ErrorHandler())
			router.GET("/test", func(c *gin.Context) {
				tt.setupHandler(c)
			})

			// 创建测试请求
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		expectedHTTP int
	}{
		{"请求参数错误", 400, http.StatusBadRequest},
		{"未授权", 401, http.StatusUnauthorized},
		{"禁止访问", 403, http.StatusForbidden},
		{"资源不存在", 404, http.StatusNotFound},
		{"参数验证失败", 422, http.StatusUnprocessableEntity},
		{"内部错误", 500, http.StatusInternalServerError},
		{"服务不可用", 503, http.StatusServiceUnavailable},
		{"上下文构建失败", 600, http.StatusInternalServerError},
		{"Token 超限", 602, http.StatusRequestEntityTooLarge},
		{"记忆不存在", 610, http.StatusNotFound},
		{"Token 预算超限", 630, http.StatusPaymentRequired},
		{"模型配置无效", 643, http.StatusBadRequest},
		{"批量部分失败", 651, http.StatusMultiStatus},
		{"熔断器已打开", 671, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHTTPStatus(tt.code)
			assert.Equal(t, tt.expectedHTTP, result)
		})
	}
}

func TestAbortWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		err := pkgErrors.NewBadRequestError("测试错误")
		AbortWithError(c, err)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 验证错误被记录
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, len(w.Body.String()) > 0)
}

func TestAbortWithAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		appErr := pkgErrors.NewForbiddenError("无权访问")
		AbortWithAppError(c, appErr)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 验证错误被记录
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, len(w.Body.String()) > 0)
}

func TestErrorHandlerNoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
