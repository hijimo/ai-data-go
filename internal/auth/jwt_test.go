package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	
	userID := "user-123"
	projectID := "project-456"
	roles := []string{"admin", "user"}
	
	token, err := manager.GenerateToken(userID, projectID, roles)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	
	userID := "user-123"
	projectID := "project-456"
	roles := []string{"admin", "user"}
	
	// 生成令牌
	token, err := manager.GenerateToken(userID, projectID, roles)
	assert.NoError(t, err)
	
	// 验证令牌
	claims, err := manager.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, projectID, claims.ProjectID)
	assert.Equal(t, roles, claims.Roles)
}

func TestJWTManager_ValidateToken_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	
	// 测试无效令牌
	_, err := manager.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager("secret1", time.Hour)
	manager2 := NewJWTManager("secret2", time.Hour)
	
	// 用manager1生成令牌
	token, err := manager1.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 用manager2验证令牌（不同的密钥）
	_, err = manager2.ValidateToken(token)
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_ExpiredToken(t *testing.T) {
	// 创建一个很短过期时间的管理器
	manager := NewJWTManager("test-secret", time.Millisecond)
	
	token, err := manager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 等待令牌过期
	time.Sleep(10 * time.Millisecond)
	
	// 验证过期令牌
	_, err = manager.ValidateToken(token)
	assert.Error(t, err)
}

func TestJWTManager_RefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	
	userID := "user-123"
	projectID := "project-456"
	roles := []string{"admin"}
	
	// 生成原始令牌
	originalToken, err := manager.GenerateToken(userID, projectID, roles)
	assert.NoError(t, err)
	
	// 创建一个即将过期的令牌（用于测试刷新逻辑）
	shortManager := NewJWTManager("test-secret", 20*time.Minute) // 20分钟，在30分钟刷新窗口内
	shortToken, err := shortManager.GenerateToken(userID, projectID, roles)
	assert.NoError(t, err)
	
	// 刷新令牌
	newToken, err := manager.RefreshToken(shortToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, originalToken, newToken)
	
	// 验证新令牌
	claims, err := manager.ValidateToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, projectID, claims.ProjectID)
	assert.Equal(t, roles, claims.Roles)
}

func TestJWTManager_RefreshToken_TooEarly(t *testing.T) {
	manager := NewJWTManager("test-secret", 2*time.Hour) // 2小时，超出30分钟刷新窗口
	
	token, err := manager.GenerateToken("user-123", "project-456", []string{"admin"})
	assert.NoError(t, err)
	
	// 尝试刷新还未到刷新时间的令牌
	_, err = manager.RefreshToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "尚未到刷新时间")
}