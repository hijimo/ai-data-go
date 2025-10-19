package auth

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// mockUserRepoForLoginLimit 用于登录限制测试的 mock UserRepository
type mockUserRepoForLoginLimit struct {
	users                map[string]*model.User
	failedAttempts       map[string]int
	lockedUntil          map[string]*time.Time
	getByEmailFunc       func(ctx context.Context, tenantID string, email string) (*model.User, error)
	incrementFailedFunc  func(ctx context.Context, tenantID, userID string) error
	resetFailedFunc      func(ctx context.Context, tenantID, userID string) error
	lockAccountFunc      func(ctx context.Context, tenantID, userID string, lockDuration time.Duration) error
	isAccountLockedFunc  func(ctx context.Context, tenantID, userID string) (bool, error)
}

func (m *mockUserRepoForLoginLimit) GetByEmail(ctx context.Context, tenantID string, email string) (*model.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, tenantID, email)
	}
	key := tenantID + ":" + email
	if user, ok := m.users[key]; ok {
		// 更新失败次数和锁定状态
		if attempts, ok := m.failedAttempts[user.ID]; ok {
			user.FailedLoginAttempts = attempts
		}
		if locked, ok := m.lockedUntil[user.ID]; ok {
			user.LockedUntil = locked
		}
		return user, nil
	}
	return nil, nil
}

func (m *mockUserRepoForLoginLimit) IncrementFailedLoginAttempts(ctx context.Context, tenantID, userID string) error {
	if m.incrementFailedFunc != nil {
		return m.incrementFailedFunc(ctx, tenantID, userID)
	}
	m.failedAttempts[userID]++
	return nil
}

func (m *mockUserRepoForLoginLimit) ResetFailedLoginAttempts(ctx context.Context, tenantID, userID string) error {
	if m.resetFailedFunc != nil {
		return m.resetFailedFunc(ctx, tenantID, userID)
	}
	m.failedAttempts[userID] = 0
	m.lockedUntil[userID] = nil
	return nil
}

func (m *mockUserRepoForLoginLimit) LockAccount(ctx context.Context, tenantID, userID string, lockDuration time.Duration) error {
	if m.lockAccountFunc != nil {
		return m.lockAccountFunc(ctx, tenantID, userID, lockDuration)
	}
	lockTime := time.Now().Add(lockDuration)
	m.lockedUntil[userID] = &lockTime
	return nil
}

func (m *mockUserRepoForLoginLimit) IsAccountLocked(ctx context.Context, tenantID, userID string) (bool, error) {
	if m.isAccountLockedFunc != nil {
		return m.isAccountLockedFunc(ctx, tenantID, userID)
	}
	if locked, ok := m.lockedUntil[userID]; ok && locked != nil {
		return time.Now().Before(*locked), nil
	}
	return false, nil
}

func (m *mockUserRepoForLoginLimit) UnlockAccount(ctx context.Context, tenantID, userID string) error {
	m.failedAttempts[userID] = 0
	m.lockedUntil[userID] = nil
	return nil
}

// 实现其他必需的接口方法（简化实现）
func (m *mockUserRepoForLoginLimit) Create(ctx context.Context, user *model.User) error {
	return nil
}

func (m *mockUserRepoForLoginLimit) GetByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	// 在 users map 中查找用户
	for _, user := range m.users {
		if user.ID == userID && user.TenantID == tenantID {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForLoginLimit) Update(ctx context.Context, user *model.User) error {
	return nil
}

func (m *mockUserRepoForLoginLimit) Delete(ctx context.Context, tenantID, userID string) error {
	return nil
}

func (m *mockUserRepoForLoginLimit) List(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepoForLoginLimit) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	return nil
}

// mockTokenServiceForLoginLimit 用于登录限制测试的 mock TokenService
type mockTokenServiceForLoginLimit struct{}

func (m *mockTokenServiceForLoginLimit) GenerateAccessToken(user *model.User) (string, error) {
	return "mock-access-token", nil
}

func (m *mockTokenServiceForLoginLimit) GenerateRefreshToken(user *model.User) (string, *model.RefreshToken, error) {
	token := &model.RefreshToken{
		ID:       uuid.New().String(),
		UserID:   user.ID,
		TenantID: user.TenantID,
	}
	return "mock-refresh-token", token, nil
}

func (m *mockTokenServiceForLoginLimit) ValidateAccessToken(tokenString string) (*model.JWTClaims, error) {
	return nil, nil
}

func (m *mockTokenServiceForLoginLimit) ValidateRefreshToken(ctx context.Context, tokenString string) (*model.RefreshToken, error) {
	return nil, nil
}

func (m *mockTokenServiceForLoginLimit) RevokeRefreshToken(ctx context.Context, tokenString string) error {
	return nil
}

func (m *mockTokenServiceForLoginLimit) HashToken(token string) string {
	return "hashed-token"
}

// mockAuditRepoForLoginLimit 用于登录限制测试的 mock AuditRepository
type mockAuditRepoForLoginLimit struct{}

func (m *mockAuditRepoForLoginLimit) Create(ctx context.Context, audit *model.AuthAudit) error {
	return nil
}

func (m *mockAuditRepoForLoginLimit) List(ctx context.Context, filter repository.AuditFilter, page, pageSize int) ([]*model.AuthAudit, int64, error) {
	return nil, 0, nil
}

// mockRefreshTokenRepoForLoginLimit 用于登录限制测试的 mock RefreshTokenRepository
type mockRefreshTokenRepoForLoginLimit struct{}

func (m *mockRefreshTokenRepoForLoginLimit) Create(ctx context.Context, token *model.RefreshToken) error {
	return nil
}

func (m *mockRefreshTokenRepoForLoginLimit) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	return nil, nil
}

func (m *mockRefreshTokenRepoForLoginLimit) Revoke(ctx context.Context, tokenID string, replacedBy *string) error {
	return nil
}

func (m *mockRefreshTokenRepoForLoginLimit) RevokeAllByUser(ctx context.Context, tenantID, userID string) error {
	return nil
}

func (m *mockRefreshTokenRepoForLoginLimit) DeleteExpired(ctx context.Context) error {
	return nil
}

// TestLoginFailureLimit 测试登录失败限制功能
func TestLoginFailureLimit(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := crypto.HashPassword(password)

	user := &model.User{
		ID:                  userID,
		TenantID:            tenantID,
		Email:               email,
		PasswordHash:        passwordHash,
		IsActive:            true,
		FailedLoginAttempts: 0,
		Roles:               datatypes.JSON([]byte(`["user"]`)),
	}

	// 创建 mock repositories
	userRepo := &mockUserRepoForLoginLimit{
		users:          make(map[string]*model.User),
		failedAttempts: make(map[string]int),
		lockedUntil:    make(map[string]*time.Time),
	}
	userRepo.users[tenantID+":"+email] = user

	tokenService := &mockTokenServiceForLoginLimit{}
	auditRepo := &mockAuditRepoForLoginLimit{}
	refreshTokenRepo := &mockRefreshTokenRepoForLoginLimit{}

	// 创建 AuthService，设置最大失败次数为 3
	authService := NewAuthService(
		userRepo,
		tokenService,
		auditRepo,
		refreshTokenRepo,
		60*time.Minute,
		3, // maxLoginAttempts
		15*time.Minute, // lockDuration
	)

	ctx := context.Background()

	// 测试场景1：连续失败登录，但未达到最大次数
	t.Run("Failed login attempts increment", func(t *testing.T) {
		req := LoginRequest{
			TenantID: tenantID,
			Email:    email,
			Password: "wrongpassword",
		}

		// 第一次失败
		_, err := authService.Login(ctx, req)
		if err == nil {
			t.Error("Expected login to fail with wrong password")
		}
		if userRepo.failedAttempts[userID] != 1 {
			t.Errorf("Expected 1 failed attempt, got %d", userRepo.failedAttempts[userID])
		}

		// 第二次失败
		_, err = authService.Login(ctx, req)
		if err == nil {
			t.Error("Expected login to fail with wrong password")
		}
		if userRepo.failedAttempts[userID] != 2 {
			t.Errorf("Expected 2 failed attempts, got %d", userRepo.failedAttempts[userID])
		}
	})

	// 测试场景2：达到最大失败次数，账户被锁定
	t.Run("Account locked after max attempts", func(t *testing.T) {
		req := LoginRequest{
			TenantID: tenantID,
			Email:    email,
			Password: "wrongpassword",
		}

		// 第三次失败，应该触发锁定
		_, err := authService.Login(ctx, req)
		if err == nil {
			t.Error("Expected login to fail and account to be locked")
		}

		// 验证账户已被锁定
		isLocked, _ := userRepo.IsAccountLocked(ctx, tenantID, userID)
		if !isLocked {
			t.Error("Expected account to be locked")
		}
	})

	// 测试场景3：账户锁定后尝试登录
	t.Run("Login blocked when account is locked", func(t *testing.T) {
		req := LoginRequest{
			TenantID: tenantID,
			Email:    email,
			Password: password, // 使用正确的密码
		}

		_, err := authService.Login(ctx, req)
		if err == nil {
			t.Error("Expected login to fail when account is locked")
		}
		if err.Error() != "账户已被锁定，请稍后再试" {
			t.Errorf("Expected locked account error, got: %v", err)
		}
	})

	// 测试场景4：解锁账户
	t.Run("Unlock account", func(t *testing.T) {
		err := authService.UnlockAccount(ctx, tenantID, userID)
		if err != nil {
			t.Errorf("Failed to unlock account: %v", err)
		}

		// 验证账户已解锁
		isLocked, _ := userRepo.IsAccountLocked(ctx, tenantID, userID)
		if isLocked {
			t.Error("Expected account to be unlocked")
		}

		// 验证失败次数已重置
		if userRepo.failedAttempts[userID] != 0 {
			t.Errorf("Expected failed attempts to be reset, got %d", userRepo.failedAttempts[userID])
		}
	})

	// 测试场景5：成功登录后重置失败次数
	t.Run("Successful login resets failed attempts", func(t *testing.T) {
		// 先增加一些失败次数
		userRepo.failedAttempts[userID] = 2

		req := LoginRequest{
			TenantID: tenantID,
			Email:    email,
			Password: password, // 正确的密码
		}

		_, err := authService.Login(ctx, req)
		if err != nil {
			t.Errorf("Expected successful login, got error: %v", err)
		}

		// 验证失败次数已重置
		if userRepo.failedAttempts[userID] != 0 {
			t.Errorf("Expected failed attempts to be reset after successful login, got %d", userRepo.failedAttempts[userID])
		}
	})
}
