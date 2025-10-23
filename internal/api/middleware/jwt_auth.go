package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"genkit-ai-service/internal/logger"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// AuthContextKey 认证上下文键类型
type AuthContextKey string

const (
	// UserIDContextKey 用户ID上下文键
	UserIDContextKey AuthContextKey = "user_id"

	// UserRolesKey 用户角色上下文键
	UserRolesKey AuthContextKey = "user_roles"

	// UserScopesKey 用户权限范围上下文键
	UserScopesKey AuthContextKey = "user_scopes"

	// JWTClaimsKey JWT Claims 上下文键
	// 注意：这个键必须与 internal/service/auth/context_helper.go 中的 JWTClaimsContextKey 保持一致
	JWTClaimsKey AuthContextKey = "jwt_claims"
)

// JWTAuth JWT 认证中间件
// 从 Authorization 头提取和验证 JWT token，并将用户信息注入上下文
func JWTAuth(tokenService authservice.TokenService, blacklistService authservice.TokenBlacklistService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从 Authorization 头提取 token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.WarnContext(r.Context(), "缺少 Authorization 请求头", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				resp := response.Error[any](errors.CodeUnauthorized, "缺少身份认证信息")

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			// 验证 Bearer 格式
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.WarnContext(r.Context(), "Authorization 头格式无效", logger.Fields{
					"path":   r.URL.Path,
					"method": r.Method,
					"header": authHeader,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				resp := response.Error[any](errors.CodeUnauthorized, "身份认证格式无效，期望格式：Bearer <token>")

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			tokenString := parts[1]

			// 检查 token 是否在黑名单中
			if blacklistService != nil {
				isBlacklisted, err := blacklistService.IsBlacklisted(r.Context(), tokenString)
				if err != nil {
					logger.ErrorContext(r.Context(), "检查 token 黑名单状态失败", logger.Fields{
						"path":  r.URL.Path,
						"error": err.Error(),
					})
					// 继续处理，不因黑名单检查失败而阻止请求
				} else if isBlacklisted {
					logger.WarnContext(r.Context(), "token 已被撤销", logger.Fields{
						"path": r.URL.Path,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)

					resp := response.Error[any](errors.CodeUnauthorized, "身份认证已被撤销，请重新登录")

					if data, err := json.Marshal(resp); err == nil {
						w.Write(data)
					}
					return
				}
			}

			// 验证 token
			claims, err := tokenService.ValidateAccessToken(tokenString)
			if err != nil {
				logger.WarnContext(r.Context(), "JWT token 验证失败", logger.Fields{
					"path":  r.URL.Path,
					"error": err.Error(),
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				// 根据错误类型返回不同的消息
				message := "身份认证失败"
				if strings.Contains(err.Error(), "expired") {
					message = "身份认证已过期，请重新登录"
				} else if strings.Contains(err.Error(), "invalid") {
					message = "身份认证无效"
				}

				resp := response.Error[any](errors.CodeUnauthorized, message)

				if data, jsonErr := json.Marshal(resp); jsonErr == nil {
					w.Write(data)
				}
				return
			}

			// 验证必需的 claims
			if claims.Subject == "" {
				logger.WarnContext(r.Context(), "JWT token 缺少 subject claim", logger.Fields{
					"path": r.URL.Path,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				resp := response.Error[any](errors.CodeUnauthorized, "身份认证信息不完整")

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			if claims.TenantID == "" {
				logger.WarnContext(r.Context(), "JWT token 缺少 tenant_id claim", logger.Fields{
					"path": r.URL.Path,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				resp := response.Error[any](errors.CodeUnauthorized, "身份认证信息不完整")

				if data, err := json.Marshal(resp); err == nil {
					w.Write(data)
				}
				return
			}

			// 获取用户 ID 和租户 ID
			userID := claims.Subject
			tenantID := claims.TenantID

			// 将用户信息注入上下文
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDContextKey, userID)
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, UserRolesKey, claims.Roles)
			ctx = context.WithValue(ctx, UserScopesKey, claims.Scopes)
			// 使用 authservice 包中定义的键来存储 JWT Claims，以便服务层可以访问
			ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)

			r = r.WithContext(ctx)

			// 记录认证成功
			logger.DebugContext(ctx, "JWT 认证成功", logger.Fields{
				"user_id":   userID,
				"tenant_id": tenantID,
				"roles":     claims.Roles,
				"path":      r.URL.Path,
			})

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthUserID 从上下文中获取认证用户 ID
func GetAuthUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}

// MustGetAuthUserID 从上下文中获取认证用户 ID，如果不存在则 panic
// 注意：仅在确保已经过 JWTAuth 中间件处理后使用
func MustGetAuthUserID(ctx context.Context) string {
	userID, ok := GetAuthUserID(ctx)
	if !ok {
		panic("认证用户ID未在上下文中设置")
	}
	return userID
}

// GetAuthUserRoles 从上下文中获取用户角色
func GetAuthUserRoles(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(UserRolesKey).([]string)
	return roles, ok
}

// GetAuthUserScopes 从上下文中获取用户权限范围
func GetAuthUserScopes(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(UserScopesKey).([]string)
	return scopes, ok
}

// HasRole 检查用户是否具有指定角色
func HasRole(ctx context.Context, role string) bool {
	roles, ok := GetAuthUserRoles(ctx)
	if !ok {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasAnyRole 检查用户是否具有任意一个指定角色
func HasAnyRole(ctx context.Context, requiredRoles ...string) bool {
	roles, ok := GetAuthUserRoles(ctx)
	if !ok {
		return false
	}

	for _, required := range requiredRoles {
		for _, r := range roles {
			if r == required {
				return true
			}
		}
	}

	return false
}

// HasAllRoles 检查用户是否具有所有指定角色
func HasAllRoles(ctx context.Context, requiredRoles ...string) bool {
	roles, ok := GetAuthUserRoles(ctx)
	if !ok {
		return false
	}

	roleMap := make(map[string]bool)
	for _, r := range roles {
		roleMap[r] = true
	}

	for _, required := range requiredRoles {
		if !roleMap[required] {
			return false
		}
	}

	return true
}
