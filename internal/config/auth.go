package config

import (
	"os"
	"time"
)

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	JWTExpiration       time.Duration `json:"jwt_expiration"`
	RefreshTokenSecret  string        `json:"refresh_token_secret"`
	RefreshTokenExpiration time.Duration `json:"refresh_token_expiration"`
}

// GetAuthConfig 获取认证配置
func GetAuthConfig() *AuthConfig {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret-key-change-in-production"
	}
	
	refreshSecret := os.Getenv("REFRESH_TOKEN_SECRET")
	if refreshSecret == "" {
		refreshSecret = "default-refresh-secret-key-change-in-production"
	}
	
	return &AuthConfig{
		JWTSecret:              jwtSecret,
		JWTExpiration:          24 * time.Hour, // 24小时
		RefreshTokenSecret:     refreshSecret,
		RefreshTokenExpiration: 7 * 24 * time.Hour, // 7天
	}
}