package handler

import (
	"net/http"
	"time"

	"ai-knowledge-platform/internal/auth"
	"ai-knowledge-platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtManager *auth.JWTManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
	}
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	ProjectID string `json:"project_id,omitempty"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	User         UserInfo  `json:"user"`
}

// UserInfo 用户信息结构
type UserInfo struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email,omitempty"`
	Roles     []string `json:"roles"`
	ProjectID string   `json:"project_id,omitempty"`
}

// RefreshTokenRequest 刷新令牌请求结构
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} LoginResponse "登录成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "认证失败"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}

	// TODO: 这里应该验证用户名和密码
	// 目前为演示目的，使用硬编码的用户信息
	if req.Username != "admin" || req.Password != "password" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "认证失败",
			"message": "用户名或密码错误",
		})
		return
	}

	// 模拟用户信息（实际应该从数据库查询）
	userID := "user-123"
	roles := []string{"admin"}
	
	// 生成访问令牌
	accessToken, err := h.jwtManager.GenerateToken(userID, req.ProjectID, roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "令牌生成失败",
			"message": err.Error(),
		})
		return
	}

	// 生成刷新令牌（有效期更长）
	refreshJWTManager := auth.NewJWTManager("refresh-secret-key", 7*24*time.Hour) // 7天
	refreshToken, err := refreshJWTManager.GenerateToken(userID, req.ProjectID, roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "刷新令牌生成失败",
			"message": err.Error(),
		})
		return
	}

	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64((24 * time.Hour).Seconds()), // 24小时
		User: UserInfo{
			ID:        userID,
			Username:  req.Username,
			Email:     "admin@example.com",
			Roles:     roles,
			ProjectID: req.ProjectID,
		},
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} LoginResponse "刷新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "刷新令牌无效"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": err.Error(),
		})
		return
	}

	// 验证刷新令牌
	refreshJWTManager := auth.NewJWTManager("refresh-secret-key", 7*24*time.Hour)
	claims, err := refreshJWTManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "刷新令牌无效",
			"message": err.Error(),
		})
		return
	}

	// 生成新的访问令牌
	accessToken, err := h.jwtManager.GenerateToken(claims.UserID, claims.ProjectID, claims.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "令牌生成失败",
			"message": err.Error(),
		})
		return
	}

	// 生成新的刷新令牌
	newRefreshToken, err := refreshJWTManager.GenerateToken(claims.UserID, claims.ProjectID, claims.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "刷新令牌生成失败",
			"message": err.Error(),
		})
		return
	}

	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64((24 * time.Hour).Seconds()),
		User: UserInfo{
			ID:        claims.UserID,
			Username:  "admin", // TODO: 从数据库获取
			Roles:     claims.Roles,
			ProjectID: claims.ProjectID,
		},
	}

	c.JSON(http.StatusOK, response)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出（客户端需要删除本地令牌）
// @Tags 认证
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "登出成功"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT是无状态的，服务端无法直接使令牌失效
	// 实际应用中可以维护一个黑名单或使用Redis存储已登出的令牌
	userID := middleware.GetUserID(c)
	
	// TODO: 将令牌加入黑名单
	// 这里可以将令牌ID存储到Redis中，在验证时检查黑名单
	
	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
		"user_id": userID,
	})
}

// GetProfile 获取用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} UserInfo "用户信息"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "未授权",
			"message": "无效的认证信息",
		})
		return
	}

	// TODO: 从数据库获取完整的用户信息
	userInfo := UserInfo{
		ID:        claims.UserID,
		Username:  "admin", // TODO: 从数据库获取
		Email:     "admin@example.com",
		Roles:     claims.Roles,
		ProjectID: claims.ProjectID,
	}

	c.JSON(http.StatusOK, userInfo)
}

// ValidateToken 验证令牌
// @Summary 验证访问令牌
// @Description 验证访问令牌的有效性
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "令牌有效"
// @Failure 401 {object} map[string]interface{} "令牌无效"
// @Router /api/v1/auth/validate [get]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "令牌无效",
			"message": "无效的认证令牌",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"user_id":    claims.UserID,
		"project_id": claims.ProjectID,
		"roles":      claims.Roles,
		"expires_at": claims.ExpiresAt.Time,
	})
}