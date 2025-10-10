package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims JWT声明结构
type JWTClaims struct {
	UserID    string   `json:"user_id"`
	ProjectID string   `json:"project_id,omitempty"`
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// GenerateToken 生成JWT令牌
func (manager *JWTManager) GenerateToken(userID string, projectID string, roles []string) (string, error) {
	claims := &JWTClaims{
		UserID:    userID,
		ProjectID: projectID,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ai-knowledge-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(manager.secretKey)
}

// ValidateToken 验证JWT令牌
func (manager *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("意外的签名方法")
		}
		return manager.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// RefreshToken 刷新JWT令牌
func (manager *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := manager.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查令牌是否即将过期（在过期前30分钟内可以刷新）
	if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
		return "", errors.New("令牌尚未到刷新时间")
	}

	// 生成新令牌
	return manager.GenerateToken(claims.UserID, claims.ProjectID, claims.Roles)
}