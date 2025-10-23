package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims 自定义 JWT Claims 结构
// @Description JWT Token 的 Claims 结构，包含用户身份和权限信息
// 包含标准 Claims 和自定义字段，用于 Access Token
type JWTClaims struct {
	jwt.RegisteredClaims
	// 租户 ID，标识用户所属的租户
	TenantID string `json:"tid" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 用户显示名称，用于在创建数据时记录创建者名称
	DisplayName string `json:"displayName" example:"张三"`
	// 用户角色列表，用于权限验证
	// 可选值：system_admin（平台管理员，可管理所有租户）, tenant_admin（租户管理员，可管理本租户用户）, user（普通用户）
	Roles []string `json:"roles" example:"[\"user\"]"`
	// 权限范围列表，用于细粒度权限控制
	Scopes []string `json:"scopes" example:"[\"read:users\",\"write:users\"]"`
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
