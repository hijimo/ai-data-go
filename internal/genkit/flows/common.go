// Package flows 提供共享的辅助函数
package flows

import (
	"genkit-ai-service/internal/model"
)

// hasRole 检查用户是否具有指定角色
func hasRole(claims *model.JWTClaims, role string) bool {
	if claims == nil {
		return false
	}
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}
