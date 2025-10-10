package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-knowledge-platform/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/login", authHandler.Login)
	
	// 创建登录请求
	loginReq := LoginRequest{
		Username:  "admin",
		Password:  "password",
		ProjectID: "project-123",
	}
	
	reqBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, "Bearer", response.TokenType)
	assert.Equal(t, "admin", response.User.Username)
	assert.Equal(t, "project-123", response.User.ProjectID)
	assert.Contains(t, response.User.Roles, "admin")
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/login", authHandler.Login)
	
	// 创建错误的登录请求
	loginReq := LoginRequest{
		Username: "admin",
		Password: "wrong-password",
	}
	
	reqBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "用户名或密码错误")
}

func TestAuthHandler_Login_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/login", authHandler.Login)
	
	// 创建无效的请求（缺少必填字段）
	loginReq := map[string]string{
		"username": "admin",
		// 缺少password字段
	}
	
	reqBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "请求参数错误")
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 生成刷新令牌
	refreshJWTManager := auth.NewJWTManager("refresh-secret-key", 7*24*time.Hour)
	refreshToken, err := refreshJWTManager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/refresh", authHandler.RefreshToken)
	
	// 创建刷新请求
	refreshReq := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}
	
	reqBody, _ := json.Marshal(refreshReq)
	req := httptest.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, "Bearer", response.TokenType)
}

func TestAuthHandler_RefreshToken_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/refresh", authHandler.RefreshToken)
	
	// 创建无效的刷新请求
	refreshReq := RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}
	
	reqBody, _ := json.Marshal(refreshReq)
	req := httptest.NewRequest("POST", "/refresh", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "刷新令牌无效")
}

func TestAuthHandler_GetProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 生成有效令牌
	token, err := jwtManager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 创建测试路由
	r := gin.New()
	r.GET("/profile", func(c *gin.Context) {
		// 模拟认证中间件设置的上下文
		claims, _ := jwtManager.ValidateToken(token)
		c.Set("jwt_claims", claims)
		authHandler.GetProfile(c)
	})
	
	// 创建请求
	req := httptest.NewRequest("GET", "/profile", nil)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var userInfo UserInfo
	err = json.Unmarshal(w.Body.Bytes(), &userInfo)
	assert.NoError(t, err)
	
	assert.Equal(t, "user-123", userInfo.ID)
	assert.Equal(t, "project-456", userInfo.ProjectID)
	assert.Contains(t, userInfo.Roles, "admin")
}

func TestAuthHandler_ValidateToken_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 生成有效令牌
	token, err := jwtManager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 创建测试路由
	r := gin.New()
	r.GET("/validate", func(c *gin.Context) {
		// 模拟认证中间件设置的上下文
		claims, _ := jwtManager.ValidateToken(token)
		c.Set("jwt_claims", claims)
		authHandler.ValidateToken(c)
	})
	
	// 创建请求
	req := httptest.NewRequest("GET", "/validate", nil)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, true, response["valid"])
	assert.Equal(t, "user-123", response["user_id"])
	assert.Equal(t, "project-456", response["project_id"])
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	authHandler := NewAuthHandler(jwtManager)
	
	// 创建测试路由
	r := gin.New()
	r.POST("/logout", func(c *gin.Context) {
		// 模拟认证中间件设置的上下文
		c.Set("user_id", "user-123")
		authHandler.Logout(c)
	})
	
	// 创建请求
	req := httptest.NewRequest("POST", "/logout", nil)
	
	// 执行请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "登出成功")
	assert.Contains(t, w.Body.String(), "user-123")
}