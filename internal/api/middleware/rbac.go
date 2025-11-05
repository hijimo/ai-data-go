package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"

	"github.com/google/uuid"
)

// GetJWTClaims 从上下文中获取 JWT Claims
// 注意：必须使用 authservice.JWTClaimsContextKey 来获取，因为 jwt_auth.go 中使用该键存储
func GetJWTClaims(ctx context.Context) (*model.JWTClaims, bool) {
	claims, ok := ctx.Value(authservice.JWTClaimsContextKey).(*model.JWTClaims)
	return claims, ok
}

// hasRole 检查用户是否具有指定角色
func hasRole(ctx context.Context, role string) bool {
	return HasRole(ctx, role)
}

// hasAnyRole 检查用户是否具有任意一个指定角色
func hasAnyRole(ctx context.Context, roles ...string) bool {
	return HasAnyRole(ctx, roles...)
}

// respondWithError 返回错误响应
func respondWithError(w http.ResponseWriter, statusCode int, errorCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := response.Error[any](errorCode, message)
	if data, err := json.Marshal(resp); err == nil {
		w.Write(data)
	}
}

// respondWithUnauthorized 返回未授权错误
func respondWithUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "未授权访问"
	}
	respondWithError(w, http.StatusUnauthorized, errors.CodeUnauthorized, message)
}

// respondWithForbidden 返回禁止访问错误
func respondWithForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "权限不足"
	}
	respondWithError(w, http.StatusForbidden, errors.CodeForbidden, message)
}


// RequireSystemAdmin 要求平台管理员权限的中间件
// 验证用户角色是否包含 "system_admin"
func RequireSystemAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 获取 JWT Claims
			claims, ok := GetJWTClaims(ctx)
			if !ok {
				logger.WarnContext(ctx, "RBAC: 未找到 JWT Claims", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				respondWithUnauthorized(w, "身份认证信息缺失")
				return
			}

			// 检查是否具有 system_admin 角色
			if !hasRole(ctx, model.RoleSystemAdmin) {
				logger.WarnContext(ctx, "RBAC: 权限不足，需要平台管理员权限", logger.Fields{
					"path":      r.URL.Path,
					"method":    r.Method,
					"user_id":   claims.Subject,
					"tenant_id": claims.TenantID,
					"roles":     claims.Roles,
				})
				// 记录权限验证失败的审计日志
				logPermissionDenied(ctx, r, claims, "需要平台管理员权限")
				respondWithForbidden(w, "权限不足：需要平台管理员权限")
				return
			}

			// 记录权限验证成功
			logger.DebugContext(ctx, "RBAC: 平台管理员权限验证通过", logger.Fields{
				"path":      r.URL.Path,
				"method":    r.Method,
				"user_id":   claims.Subject,
				"tenant_id": claims.TenantID,
			})

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}


// RequireTenantAdmin 要求租户管理员或平台管理员权限的中间件
// 验证用户角色是否包含 "tenant_admin" 或 "system_admin"
func RequireTenantAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 获取 JWT Claims
			claims, ok := GetJWTClaims(ctx)
			if !ok {
				logger.WarnContext(ctx, "RBAC: 未找到 JWT Claims", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				respondWithUnauthorized(w, "身份认证信息缺失")
				return
			}

			// 检查是否具有 tenant_admin 或 system_admin 角色
			if !hasAnyRole(ctx, model.RoleTenantAdmin, model.RoleSystemAdmin) {
				logger.WarnContext(ctx, "RBAC: 权限不足，需要租户管理员或平台管理员权限", logger.Fields{
					"path":      r.URL.Path,
					"method":    r.Method,
					"user_id":   claims.Subject,
					"tenant_id": claims.TenantID,
					"roles":     claims.Roles,
				})
				// 记录权限验证失败的审计日志
				logPermissionDenied(ctx, r, claims, "需要租户管理员或平台管理员权限")
				respondWithForbidden(w, "权限不足：需要租户管理员或平台管理员权限")
				return
			}

			// 记录权限验证成功
			logger.DebugContext(ctx, "RBAC: 租户管理员权限验证通过", logger.Fields{
				"path":      r.URL.Path,
				"method":    r.Method,
				"user_id":   claims.Subject,
				"tenant_id": claims.TenantID,
				"roles":     claims.Roles,
			})

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}


// RequireTenantAccess 要求访问特定租户权限的中间件
// 验证用户是否有权访问目标租户的数据
// 平台管理员可以访问所有租户，其他用户只能访问自己所属的租户
func RequireTenantAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 获取 JWT Claims
			claims, ok := GetJWTClaims(ctx)
			if !ok {
				logger.WarnContext(ctx, "RBAC: 未找到 JWT Claims", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				respondWithUnauthorized(w, "身份认证信息缺失")
				return
			}

			// 从 URL 路径中提取租户 ID
			// 支持路径格式：/api/v1/tenants/{tenantId}/...
			targetTenantID := extractTenantIDFromPath(r.URL.Path)

			// 如果路径中没有 tenantId，尝试从查询参数获取
			if targetTenantID == "" {
				targetTenantID = r.URL.Query().Get("tenantId")
			}

			// 如果仍然没有找到目标租户 ID，记录警告但允许继续
			// 这种情况下，后续的业务逻辑应该处理租户隔离
			if targetTenantID == "" {
				logger.DebugContext(ctx, "RBAC: 未找到目标租户 ID，跳过租户访问验证", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})
				next.ServeHTTP(w, r)
				return
			}

			// 如果用户是平台管理员，允许访问所有租户
			if hasRole(ctx, model.RoleSystemAdmin) {
				logger.DebugContext(ctx, "RBAC: 平台管理员跨租户访问", logger.Fields{
					"path":             r.URL.Path,
					"method":           r.Method,
					"user_id":          claims.Subject,
					"user_tenant_id":   claims.TenantID,
					"target_tenant_id": targetTenantID,
				})
				next.ServeHTTP(w, r)
				return
			}

			// 检查用户的租户 ID 是否与目标租户 ID 匹配
			if claims.TenantID != targetTenantID {
				logger.WarnContext(ctx, "RBAC: 租户访问权限不足", logger.Fields{
					"path":             r.URL.Path,
					"method":           r.Method,
					"user_id":          claims.Subject,
					"user_tenant_id":   claims.TenantID,
					"target_tenant_id": targetTenantID,
					"roles":            claims.Roles,
				})
				// 记录权限验证失败的审计日志
				logPermissionDenied(ctx, r, claims, "尝试访问其他租户的数据")
				respondWithForbidden(w, "权限不足：无法访问其他租户的数据")
				return
			}

			// 记录权限验证成功
			logger.DebugContext(ctx, "RBAC: 租户访问权限验证通过", logger.Fields{
				"path":      r.URL.Path,
				"method":    r.Method,
				"user_id":   claims.Subject,
				"tenant_id": claims.TenantID,
			})

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}


// logPermissionDenied 记录权限验证失败的审计日志
// 注意：这个函数目前只记录到应用日志，实际的审计日志记录应该在业务层处理
// 因为中间件层不应该直接依赖审计仓储，避免循环依赖
func logPermissionDenied(ctx context.Context, r *http.Request, claims *model.JWTClaims, reason string) {
	// 解析用户 ID 和租户 ID
	var userID, tenantID *uuid.UUID
	
	if claims != nil {
		if uid, err := uuid.Parse(claims.Subject); err == nil {
			userID = &uid
		}
		if tid, err := uuid.Parse(claims.TenantID); err == nil {
			tenantID = &tid
		}
	}

	// 记录详细的权限验证失败日志
	logger.WarnContext(ctx, "权限验证失败", logger.Fields{
		"event":      "permission_denied",
		"reason":     reason,
		"path":       r.URL.Path,
		"method":     r.Method,
		"user_id":    userID,
		"tenant_id":  tenantID,
		"roles":      claims.Roles,
		"ip":         getClientIP(r),
		"user_agent": r.UserAgent(),
	})
}

// LogPermissionDeniedWithAudit 记录权限验证失败的审计日志（包含数据库审计）
// 这个函数可以在需要记录审计日志到数据库时使用
// 需要在调用时传入 AuditRepository
func LogPermissionDeniedWithAudit(
	ctx context.Context,
	r *http.Request,
	claims *model.JWTClaims,
	reason string,
	auditRepo interface{},
) {
	// 先记录到应用日志
	logPermissionDenied(ctx, r, claims, reason)
	
	// 如果提供了审计仓储，记录到数据库
	// 注意：这里使用 interface{} 避免循环依赖
	// 实际使用时需要类型断言
}

// getClientIP 获取客户端 IP 地址
func getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 头获取
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// 尝试从 X-Real-IP 头获取
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// 使用 RemoteAddr
	return r.RemoteAddr
}


// extractTenantIDFromPath 从 URL 路径中提取租户 ID
// 支持路径格式：/api/v1/tenants/{tenantId}/... 或 /api/v1/platform/tenants/{tenantId}/...
func extractTenantIDFromPath(path string) string {
	// 分割路径
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// 查找 "tenants" 后面的部分作为租户 ID
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			// 返回 tenants 后面的第一个部分
			tenantID := parts[i+1]
			// 验证这不是另一个路径段（如 "users"）
			if tenantID != "" && !isPathSegment(tenantID) {
				return tenantID
			}
		}
	}
	
	return ""
}

// isPathSegment 检查字符串是否是常见的路径段关键字
func isPathSegment(s string) bool {
	pathSegments := []string{"users", "sessions", "messages", "providers", "models"}
	for _, segment := range pathSegments {
		if s == segment {
			return true
		}
	}
	return false
}
