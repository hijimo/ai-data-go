package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims 自定义 JWT Claims 结构
// 包含标准 Claims 和自定义字段，用于 Access Token
type JWTClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tid"`    // 租户 ID
	Roles    []string `json:"roles"`  // 用户角色列表
	Scopes   []string `json:"scopes"` // 权限范围列表
}

// GetExpirationTime 实现 jwt.Claims 接口
func (c JWTClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.ExpiresAt, nil
}

// GetIssuedAt 实现 jwt.Claims 接口
func (c JWTClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.IssuedAt, nil
}

// GetNotBefore 实现 jwt.Claims 接口
func (c JWTClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.NotBefore, nil
}

// GetIssuer 实现 jwt.Claims 接口
func (c JWTClaims) GetIssuer() (string, error) {
	return c.Issuer, nil
}

// GetSubject 实现 jwt.Claims 接口
func (c JWTClaims) GetSubject() (string, error) {
	return c.Subject, nil
}

// GetAudience 实现 jwt.Claims 接口
func (c JWTClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.Audience, nil
}
