package middleware

import (
	"net/http"

	"ai-knowledge-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// PermissionMiddleware 权限检查中间件
func PermissionMiddleware(roleManager *auth.RoleManager, requiredPermissions ...auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		userRoles := GetUserRoles(c)
		if len(userRoles) == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "用户没有分配角色",
			})
			c.Abort()
			return
		}

		// 检查权限
		if !roleManager.CheckUserPermissions(userRoles, requiredPermissions) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "用户没有执行此操作的权限",
				"required_permissions": requiredPermissions,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AnyPermissionMiddleware 任意权限检查中间件（用户拥有任意一个权限即可）
func AnyPermissionMiddleware(roleManager *auth.RoleManager, permissions ...auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		userRoles := GetUserRoles(c)
		if len(userRoles) == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "用户没有分配角色",
			})
			c.Abort()
			return
		}

		// 检查是否拥有任意一个权限
		if !roleManager.CheckUserAnyPermission(userRoles, permissions) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "用户没有执行此操作的权限",
				"required_permissions": permissions,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SystemAdminMiddleware 系统管理员权限中间件
func SystemAdminMiddleware(roleManager *auth.RoleManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := GetUserRoles(c)
		if !roleManager.IsSystemAdmin(userRoles) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "需要系统管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ProjectOwnerMiddleware 项目所有者权限中间件
func ProjectOwnerMiddleware(roleManager *auth.RoleManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := GetUserRoles(c)
		
		// 系统管理员或项目所有者都可以访问
		if !roleManager.IsSystemAdmin(userRoles) && !roleManager.IsProjectOwner(userRoles) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "需要项目所有者或系统管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ProjectIsolationMiddleware 项目隔离中间件
func ProjectIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从JWT中获取项目ID
		jwtProjectID := GetProjectID(c)
		
		// 从URL参数中获取项目ID（如果存在）
		urlProjectID := c.Param("project_id")
		if urlProjectID == "" {
			urlProjectID = c.Query("project_id")
		}
		
		// 如果URL中指定了项目ID，检查是否与JWT中的项目ID匹配
		if urlProjectID != "" && jwtProjectID != "" && urlProjectID != jwtProjectID {
			// 检查用户是否为系统管理员（系统管理员可以跨项目访问）
			userRoles := GetUserRoles(c)
			isSystemAdmin := false
			for _, role := range userRoles {
				if role == auth.RoleSystemAdmin.Name {
					isSystemAdmin = true
					break
				}
			}
			
			if !isSystemAdmin {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "权限不足",
					"message": "无法访问其他项目的资源",
				})
				c.Abort()
				return
			}
		}
		
		// 将有效的项目ID设置到上下文中
		if urlProjectID != "" {
			c.Set("current_project_id", urlProjectID)
		} else if jwtProjectID != "" {
			c.Set("current_project_id", jwtProjectID)
		}
		
		c.Next()
	}
}

// RequireProjectMiddleware 要求项目ID中间件
func RequireProjectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := GetProjectID(c)
		if projectID == "" {
			// 尝试从URL参数获取
			projectID = c.Param("project_id")
			if projectID == "" {
				projectID = c.Query("project_id")
			}
		}
		
		if projectID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "参数错误",
				"message": "缺少项目ID",
			})
			c.Abort()
			return
		}
		
		c.Set("current_project_id", projectID)
		c.Next()
	}
}

// GetCurrentProjectID 从上下文中获取当前项目ID
func GetCurrentProjectID(c *gin.Context) string {
	if projectID, exists := c.Get("current_project_id"); exists {
		if id, ok := projectID.(string); ok {
			return id
		}
	}
	return GetProjectID(c)
}

// HasPermission 检查当前用户是否拥有指定权限
func HasPermission(c *gin.Context, roleManager *auth.RoleManager, permission auth.Permission) bool {
	userRoles := GetUserRoles(c)
	return roleManager.CheckUserPermissions(userRoles, []auth.Permission{permission})
}

// HasAnyPermission 检查当前用户是否拥有任意一个权限
func HasAnyPermission(c *gin.Context, roleManager *auth.RoleManager, permissions ...auth.Permission) bool {
	userRoles := GetUserRoles(c)
	return roleManager.CheckUserAnyPermission(userRoles, permissions)
}

// GetUserPermissions 获取当前用户的所有权限
func GetUserPermissions(c *gin.Context, roleManager *auth.RoleManager) []auth.Permission {
	userRoles := GetUserRoles(c)
	return roleManager.GetUserPermissions(userRoles)
}