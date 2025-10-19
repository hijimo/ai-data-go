package auth

import (
	"context"
	"genkit-ai-service/internal/model"
)

// TokenService Token 管理服务接口
// 负责 JWT Access Token 和 Refresh Token 的生成、验证和撤销
type TokenService interface {
	// GenerateAccessToken 生成访问令牌（JWT）
	// 参数：
	//   - user: 用户信息，用于生成 token claims
	// 返回：
	//   - string: JWT token 字符串
	//   - error: 生成失败时返回错误
	GenerateAccessToken(user *model.User) (string, error)

	// GenerateRefreshToken 生成刷新令牌
	// 参数：
	//   - user: 用户信息
	// 返回：
	//   - string: Refresh Token 字符串（UUID）
	//   - *model.RefreshToken: 数据库中的 RefreshToken 记录
	//   - error: 生成失败时返回错误
	GenerateRefreshToken(user *model.User) (string, *model.RefreshToken, error)

	// ValidateAccessToken 验证访问令牌
	// 参数：
	//   - tokenString: JWT token 字符串
	// 返回：
	//   - *model.JWTClaims: 解析后的 Claims
	//   - error: 验证失败时返回错误（如过期、签名无效等）
	ValidateAccessToken(tokenString string) (*model.JWTClaims, error)

	// ValidateRefreshToken 验证刷新令牌
	// 参数：
	//   - ctx: 上下文
	//   - tokenString: Refresh Token 字符串
	// 返回：
	//   - *model.RefreshToken: 数据库中的 RefreshToken 记录
	//   - error: 验证失败时返回错误（如不存在、已撤销、已过期等）
	ValidateRefreshToken(ctx context.Context, tokenString string) (*model.RefreshToken, error)

	// RevokeRefreshToken 撤销刷新令牌
	// 参数：
	//   - ctx: 上下文
	//   - tokenString: Refresh Token 字符串
	// 返回：
	//   - error: 撤销失败时返回错误
	RevokeRefreshToken(ctx context.Context, tokenString string) error

	// HashToken 计算 token 的哈希值
	// 使用 SHA256 算法对 token 进行哈希，用于安全存储
	// 参数：
	//   - token: 原始 token 字符串
	// 返回：
	//   - string: 哈希后的字符串（十六进制格式）
	HashToken(token string) string
}
