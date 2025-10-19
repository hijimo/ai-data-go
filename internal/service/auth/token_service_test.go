package auth

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// mockRefreshTokenRepository 模拟 RefreshTokenRepository
type mockRefreshTokenRepository struct {
	tokens map[string]*model.RefreshToken
}

func newMockRefreshTokenRepository() *mockRefreshTokenRepository {
	return &mockRefreshTokenRepository{
		tokens: make(map[string]*model.RefreshToken),
	}
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	token, ok := m.tokens[tokenHash]
	if !ok {
		return nil, nil
	}
	return token, nil
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, tokenID string, replacedBy *string) error {
	for _, token := range m.tokens {
		if token.ID == tokenID {
			token.Revoked = true
			token.ReplacedBy = replacedBy
			return nil
		}
	}
	return nil
}

func (m *mockRefreshTokenRepository) RevokeAllByUser(ctx context.Context, tenantID, userID string) error {
	for _, token := range m.tokens {
		if token.UserID == userID && token.TenantID == tenantID {
			token.Revoked = true
		}
	}
	return nil
}

func (m *mockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	for hash, token := range m.tokens {
		if token.ExpiresAt.Before(now) {
			delete(m.tokens, hash)
		}
	}
	return nil
}

// TestGenerateAccessToken 测试生成 Access Token
func TestGenerateAccessToken(t *testing.T) {
	cfg := &config.AuthConfig{
		JWTSecret:      "test-secret-key-at-least-32-characters-long",
		JWTIssuer:      "test-issuer",
		JWTAudience:    "test-audience",
		AccessTokenTTL: 60 * time.Minute,
	}

	mockRepo := newMockRefreshTokenRepository()
	service := NewTokenService(cfg, mockRepo)

	// 创建测试用户
	user := &model.User{
		ID:       uuid.New().String(),
		TenantID: uuid.New().String(),
		Email:    "test@example.com",
		Roles:    datatypes.JSON([]byte(`["user"]`)),
	}

	// 生成 token
	token, err := service.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("生成 Access Token 失败: %v", err)
	}

	if token == "" {
		t.Fatal("生成的 token 为空")
	}

	// 验证 token
	claims, err := service.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("验证 Access Token 失败: %v", err)
	}

	if claims.Subject != user.ID {
		t.Errorf("期望用户 ID %s, 得到 %s", user.ID, claims.Subject)
	}

	if claims.TenantID != user.TenantID {
		t.Errorf("期望租户 ID %s, 得到 %s", user.TenantID, claims.TenantID)
	}
}

// TestGenerateRefreshToken 测试生成 Refresh Token
func TestGenerateRefreshToken(t *testing.T) {
	cfg := &config.AuthConfig{
		JWTSecret:       "test-secret-key-at-least-32-characters-long",
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}

	mockRepo := newMockRefreshTokenRepository()
	service := NewTokenService(cfg, mockRepo)

	// 创建测试用户
	user := &model.User{
		ID:       uuid.New().String(),
		TenantID: uuid.New().String(),
		Email:    "test@example.com",
	}

	// 生成 token
	tokenString, refreshToken, err := service.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("生成 Refresh Token 失败: %v", err)
	}

	if tokenString == "" {
		t.Fatal("生成的 token 字符串为空")
	}

	if refreshToken == nil {
		t.Fatal("生成的 RefreshToken 记录为空")
	}

	if refreshToken.UserID != user.ID {
		t.Errorf("期望用户 ID %s, 得到 %s", user.ID, refreshToken.UserID)
	}

	// 验证 token
	validatedToken, err := service.ValidateRefreshToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("验证 Refresh Token 失败: %v", err)
	}

	if validatedToken.UserID != user.ID {
		t.Errorf("期望用户 ID %s, 得到 %s", user.ID, validatedToken.UserID)
	}
}

// TestRevokeRefreshToken 测试撤销 Refresh Token
func TestRevokeRefreshToken(t *testing.T) {
	cfg := &config.AuthConfig{
		JWTSecret:       "test-secret-key-at-least-32-characters-long",
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}

	mockRepo := newMockRefreshTokenRepository()
	service := NewTokenService(cfg, mockRepo)

	// 创建测试用户
	user := &model.User{
		ID:       uuid.New().String(),
		TenantID: uuid.New().String(),
		Email:    "test@example.com",
	}

	// 生成 token
	tokenString, _, err := service.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("生成 Refresh Token 失败: %v", err)
	}

	// 撤销 token
	err = service.RevokeRefreshToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("撤销 Refresh Token 失败: %v", err)
	}

	// 验证已撤销的 token 应该失败
	_, err = service.ValidateRefreshToken(context.Background(), tokenString)
	if err == nil {
		t.Fatal("期望验证已撤销的 token 失败，但成功了")
	}
}

// TestHashToken 测试 token 哈希
func TestHashToken(t *testing.T) {
	cfg := &config.AuthConfig{
		JWTSecret: "test-secret-key-at-least-32-characters-long",
	}

	mockRepo := newMockRefreshTokenRepository()
	service := NewTokenService(cfg, mockRepo)

	token := "test-token-string"
	hash1 := service.HashToken(token)
	hash2 := service.HashToken(token)

	// 相同的 token 应该产生相同的哈希
	if hash1 != hash2 {
		t.Errorf("相同的 token 产生了不同的哈希: %s != %s", hash1, hash2)
	}

	// 不同的 token 应该产生不同的哈希
	differentToken := "different-token"
	hash3 := service.HashToken(differentToken)
	if hash1 == hash3 {
		t.Error("不同的 token 产生了相同的哈希")
	}
}
