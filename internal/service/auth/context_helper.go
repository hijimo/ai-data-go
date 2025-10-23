package auth

import (
	"context"

	"genkit-ai-service/internal/model"
)

// AuthContextKey 认证上下文键类型
type AuthContextKey string

const (
	// JWTClaimsContextKey JWT Claims 上下文键
	JWTClaimsContextKey AuthContextKey = "jwt_claims"
)

// GetJWTClaimsFromContext 从上下文中获取 JWT Claims
// 这个函数避免了对 middleware 包的循环依赖
func GetJWTClaimsFromContext(ctx context.Context) (*model.JWTClaims, bool) {
	claims, ok := ctx.Value(JWTClaimsContextKey).(*model.JWTClaims)
	return claims, ok
}

// hasSystemAdminRole 检查用户是否具有平台管理员角色
func hasSystemAdminRole(claims *model.JWTClaims) bool {
	if claims == nil {
		return false
	}
	for _, role := range claims.Roles {
		if role == model.RoleSystemAdmin {
			return true
		}
	}
	return false
}
