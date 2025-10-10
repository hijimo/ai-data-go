package middleware

import (
	"net/http"
	"strings"

	"ai-knowledge-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中提取令牌
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未授权",
				"message": "缺少认证令牌",
			})
			c.Abort()
			return
		}

		// 验证令牌
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未授权",
				"message": "无效的认证令牌",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("project_id", claims.ProjectID)
		c.Set("roles", claims.Roles)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求认证）
func OptionalAuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token != "" {
			if claims, err := jwtManager.ValidateToken(token); err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("project_id", claims.ProjectID)
				c.Set("roles", claims.Roles)
				c.Set("jwt_claims", claims)
			}
		}
		c.Next()
	}
}

// extractToken 从请求中提取JWT令牌
func extractToken(c *gin.Context) string {
	// 首先尝试从Authorization头中提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 支持 "Bearer <token>" 格式
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// 直接返回令牌
		return authHeader
	}

	// 尝试从查询参数中提取（用于WebSocket等场景）
	token := c.Query("token")
	if token != "" {
		return token
	}

	// 尝试从Cookie中提取
	cookie, err := c.Cookie("access_token")
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

// GetUserID 从上下文中获取用户ID
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetProjectID 从上下文中获取项目ID
func GetProjectID(c *gin.Context) string {
	if projectID, exists := c.Get("project_id"); exists {
		if id, ok := projectID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserRoles 从上下文中获取用户角色
func GetUserRoles(c *gin.Context) []string {
	if roles, exists := c.Get("roles"); exists {
		if roleList, ok := roles.([]string); ok {
			return roleList
		}
	}
	return []string{}
}

// GetJWTClaims 从上下文中获取JWT声明
func GetJWTClaims(c *gin.Context) *auth.JWTClaims {
	if claims, exists := c.Get("jwt_claims"); exists {
		if jwtClaims, ok := claims.(*auth.JWTClaims); ok {
			return jwtClaims
		}
	}
	return nil
}