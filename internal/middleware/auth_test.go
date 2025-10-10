package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-knowledge-platform/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	
	// 生成有效令牌
	token, err := jwtManager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 创建测试路由
	r := gin.New()
	r.Use(AuthMiddleware(jwtManager))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		projectID := GetProjectID(c)
		roles := GetUserRoles(c)
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":    userID,
			"project_id": projectID,
			"roles":      roles,
		})
	})
	
	// 创建请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
	assert.Contains(t, w.Body.String(), "project-456")
	assert.Contains(t, w.Body.String(), "admin")
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	
	// 创建测试路由
	r := gin.New()
	r.Use(AuthMiddleware(jwtManager))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// 创建没有令牌的请求
	req := httptest.NewRequest("GET", "/test", nil)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "缺少认证令牌")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	
	// 创建测试路由
	r := gin.New()
	r.Use(AuthMiddleware(jwtManager))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// 创建带有无效令牌的请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "无效的认证令牌")
}

func TestOptionalAuthMiddleware_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	
	// 生成有效令牌
	token, err := jwtManager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 创建测试路由
	r := gin.New()
	r.Use(OptionalAuthMiddleware(jwtManager))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"authenticated": userID != "",
		})
	})
	
	// 创建请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
	assert.Contains(t, w.Body.String(), `"authenticated":true`)
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	
	// 创建测试路由
	r := gin.New()
	r.Use(OptionalAuthMiddleware(jwtManager))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"authenticated": userID != "",
		})
	})
	
	// 创建没有令牌的请求
	req := httptest.NewRequest("GET", "/test", nil)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"authenticated":false`)
}

func TestExtractToken_FromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 测试Bearer格式
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	
	token := extractToken(c)
	assert.Equal(t, "test-token", token)
	
	// 测试直接令牌格式
	c.Request.Header.Set("Authorization", "direct-token")
	token = extractToken(c)
	assert.Equal(t, "direct-token", token)
}

func TestExtractToken_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test?token=query-token", nil)
	
	token := extractToken(c)
	assert.Equal(t, "query-token", token)
}

func TestExtractToken_FromCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "cookie-token",
	})
	
	token := extractToken(c)
	assert.Equal(t, "cookie-token", token)
}

func TestGetUserID_NotExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	
	userID := GetUserID(c)
	assert.Empty(t, userID)
}

func TestGetUserRoles_NotExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	
	roles := GetUserRoles(c)
	assert.Empty(t, roles)
}