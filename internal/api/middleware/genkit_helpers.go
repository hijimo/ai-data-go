package middleware

import (
	"context"

	"genkit-ai-service/internal/model"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
)

// GetSessionID 从上下文中获取会话 ID
func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value("session_id").(string)
	return sessionID, ok
}

// MustGetSessionID 从上下文中获取会话 ID，如果不存在则 panic
func MustGetSessionID(ctx context.Context) string {
	sessionID, ok := GetSessionID(ctx)
	if !ok {
		panic("会话ID未在上下文中设置")
	}
	return sessionID
}

// GetMemoryID 从上下文中获取记忆 ID
func GetMemoryID(ctx context.Context) (string, bool) {
	memoryID, ok := ctx.Value("memory_id").(string)
	return memoryID, ok
}

// MustGetMemoryID 从上下文中获取记忆 ID，如果不存在则 panic
func MustGetMemoryID(ctx context.Context) string {
	memoryID, ok := GetMemoryID(ctx)
	if !ok {
		panic("记忆ID未在上下文中设置")
	}
	return memoryID
}

// HasSystemAdminRole 检查用户是否具有平台管理员角色
func HasSystemAdminRole(ctx context.Context) bool {
	return HasRole(ctx, model.RoleSystemAdmin)
}

// HasTenantAdminRole 检查用户是否具有租户管理员角色
func HasTenantAdminRole(ctx context.Context) bool {
	return HasRole(ctx, model.RoleTenantAdmin)
}

// HasAdminRole 检查用户是否具有任意管理员角色（平台或租户）
func HasAdminRole(ctx context.Context) bool {
	return HasAnyRole(ctx, model.RoleSystemAdmin, model.RoleTenantAdmin)
}

// GetUserContext 从上下文中获取完整的用户上下文信息
func GetUserContext(ctx context.Context) *UserContext {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return nil
	}

	userID, _ := GetAuthUserID(ctx)
	tenantID, _ := GetTenantID(ctx)
	roles, _ := GetAuthUserRoles(ctx)
	scopes, _ := GetAuthUserScopes(ctx)

	return &UserContext{
		UserID:   userID,
		TenantID: tenantID,
		Roles:    roles,
		Scopes:   scopes,
		Claims:   claims,
	}
}

// UserContext 用户上下文信息
type UserContext struct {
	UserID   string
	TenantID string
	Roles    []string
	Scopes   []string
	Claims   *model.JWTClaims
}

// IsSystemAdmin 检查是否为平台管理员
func (uc *UserContext) IsSystemAdmin() bool {
	if uc == nil {
		return false
	}
	for _, role := range uc.Roles {
		if role == model.RoleSystemAdmin {
			return true
		}
	}
	return false
}

// IsTenantAdmin 检查是否为租户管理员
func (uc *UserContext) IsTenantAdmin() bool {
	if uc == nil {
		return false
	}
	for _, role := range uc.Roles {
		if role == model.RoleTenantAdmin {
			return true
		}
	}
	return false
}

// IsAdmin 检查是否为任意管理员
func (uc *UserContext) IsAdmin() bool {
	return uc.IsSystemAdmin() || uc.IsTenantAdmin()
}

// CanAccessTenant 检查是否可以访问指定租户
func (uc *UserContext) CanAccessTenant(targetTenantID string) bool {
	if uc == nil {
		return false
	}
	// 平台管理员可以访问所有租户
	if uc.IsSystemAdmin() {
		return true
	}
	// 其他用户只能访问自己的租户
	return uc.TenantID == targetTenantID
}

// ValidateSessionAccess 验证是否可以访问指定会话
// 这是一个辅助函数，可以在服务层使用
func ValidateSessionAccess(ctx context.Context, sessionTenantID string) error {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return errors.NewUnauthorizedError("未认证")
	}

	// 平台管理员可以访问所有会话
	if HasSystemAdminRole(ctx) {
		return nil
	}

	// 验证租户 ID 匹配
	if claims.TenantID != sessionTenantID {
		return errors.NewForbiddenError("权限不足：无法访问其他租户的会话")
	}

	return nil
}

// ValidateMemoryAccess 验证是否可以访问指定记忆
func ValidateMemoryAccess(ctx context.Context, memoryTenantID string) error {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return errors.NewUnauthorizedError("未认证")
	}

	// 平台管理员可以访问所有记忆
	if HasSystemAdminRole(ctx) {
		return nil
	}

	// 验证租户 ID 匹配
	if claims.TenantID != memoryTenantID {
		return errors.NewForbiddenError("权限不足：无法访问其他租户的记忆")
	}

	return nil
}

// GetJWTClaimsFromContext 从上下文中获取 JWT Claims（别名函数）
// 这个函数提供了更清晰的命名
func GetJWTClaimsFromContext(ctx context.Context) (*model.JWTClaims, bool) {
	return GetJWTClaims(ctx)
}

// MustGetJWTClaims 从上下文中获取 JWT Claims，如果不存在则 panic
func MustGetJWTClaims(ctx context.Context) *model.JWTClaims {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		panic("JWT Claims 未在上下文中设置")
	}
	return claims
}

// SetUserContext 将用户上下文信息注入到 Context
// 这个函数可以在测试中使用
func SetUserContext(ctx context.Context, userID, tenantID string, roles []string) context.Context {
	ctx = context.WithValue(ctx, UserIDContextKey, userID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserRolesKey, roles)

	// 创建 JWT Claims
	claims := &model.JWTClaims{
		Subject:  userID,
		TenantID: tenantID,
		Roles:    roles,
	}
	ctx = context.WithValue(ctx, authservice.JWTClaimsContextKey, claims)

	return ctx
}
