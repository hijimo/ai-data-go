package middleware

import (
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// RBACConfig RBAC 授权中间件配置
type RBACConfig struct {
	// RequiredRoles 必需的角色列表（满足任意一个即可）
	RequiredRoles []string
	// RequireAll 是否需要满足所有角色（默认为 false，满足任意一个即可）
	RequireAll bool
}

// RBACAuthorizer RBAC 授权中间件
// 验证用户是否具有所需的角色权限
func RBACAuthorizer(config RBACConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文提取用户角色
			roles, ok := GetAuthUserRoles(r.Context())
			if !ok || len(roles) == 0 {
				logger.WarnContext(r.Context(), "用户角色未在上下文中设置", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				resp := response.Error[any](errors.CodeForbidden, "权限不足")

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			// 检查是否具有所需角色
			hasPermission := false

			if config.RequireAll {
				// 需要满足所有角色
				hasPermission = HasAllRoles(r.Context(), config.RequiredRoles...)
			} else {
				// 满足任意一个角色即可
				hasPermission = HasAnyRole(r.Context(), config.RequiredRoles...)
			}

			if !hasPermission {
				// 获取用户 ID 用于日志记录
				userID, _ := GetAuthUserID(r.Context())
				tenantID, _ := GetTenantID(r.Context())

				logger.WarnContext(r.Context(), "用户权限不足", logger.Fields{
					"user_id":        userID,
					"tenant_id":      tenantID,
					"user_roles":     roles,
					"required_roles": config.RequiredRoles,
					"require_all":    config.RequireAll,
					"path":           r.URL.Path,
					"method":         r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				resp := response.Error[any](errors.CodeForbidden, "权限不足，需要以下角色之一："+joinRoles(config.RequiredRoles))

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			// 记录授权成功
			userID, _ := GetAuthUserID(r.Context())
			logger.DebugContext(r.Context(), "RBAC 授权成功", logger.Fields{
				"user_id":        userID,
				"user_roles":     roles,
				"required_roles": config.RequiredRoles,
				"path":           r.URL.Path,
			})

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole 创建需要特定角色的中间件（满足任意一个即可）
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return RBACAuthorizer(RBACConfig{
		RequiredRoles: roles,
		RequireAll:    false,
	})
}

// RequireAllRoles 创建需要所有指定角色的中间件
func RequireAllRoles(roles ...string) func(http.Handler) http.Handler {
	return RBACAuthorizer(RBACConfig{
		RequiredRoles: roles,
		RequireAll:    true,
	})
}

// RequireAdmin 创建需要管理员角色的中间件
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole("admin")
}

// RequireTenantAdmin 创建需要租户管理员角色的中间件
func RequireTenantAdmin() func(http.Handler) http.Handler {
	return RequireRole("admin", "tenant_admin")
}

// joinRoles 将角色列表连接成字符串
func joinRoles(roles []string) string {
	if len(roles) == 0 {
		return ""
	}

	result := roles[0]
	for i := 1; i < len(roles); i++ {
		result += ", " + roles[i]
	}

	return result
}
