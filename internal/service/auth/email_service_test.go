package auth

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockEmailSender 模拟邮件发送器
type mockEmailSender struct {
	sentEmails []struct {
		to      string
		subject string
		body    string
	}
}

func (m *mockEmailSender) SendEmail(to, subject, body string) error {
	m.sentEmails = append(m.sentEmails, struct {
		to      string
		subject string
		body    string
	}{to, subject, body})
	return nil
}

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.EmailVerificationToken{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// TestSendVerificationEmail 测试发送验证邮件
func TestSendVerificationEmail(t *testing.T) {
	// 1. 设置测试环境
	db := setupTestDB(t)
	verificationRepo := repository.NewEmailVerificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	mockSender := &mockEmailSender{}
	
	emailService := NewEmailService(
		verificationRepo,
		userRepo,
		mockSender,
		24*time.Hour,
	)

	// 2. 创建测试租户和用户
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	email := "test@example.com"

	tenant := &model.Tenant{
		ID:     tenantID,
		Name:   "测试租户",
		Status: true,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("创建测试租户失败: %v", err)
	}

	user := &model.User{
		ID:            userID,
		TenantID:      tenantID,
		Email:         email,
		EmailVerified: false,
		PasswordHash:  "hash",
		DisplayName:   "测试用户",
		IsActive:      true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 3. 发送验证邮件
	ctx := context.Background()
	err := emailService.SendVerificationEmail(ctx, tenantID, userID, email)
	if err != nil {
		t.Fatalf("发送验证邮件失败: %v", err)
	}

	// 4. 验证邮件已发送
	if len(mockSender.sentEmails) != 1 {
		t.Fatalf("期望发送1封邮件，实际发送了%d封", len(mockSender.sentEmails))
	}

	sentEmail := mockSender.sentEmails[0]
	if sentEmail.to != email {
		t.Errorf("期望收件人为%s，实际为%s", email, sentEmail.to)
	}

	// 5. 验证数据库中创建了验证令牌
	var tokens []*model.EmailVerificationToken
	if err := db.Where("user_id = ?", userID).Find(&tokens).Error; err != nil {
		t.Fatalf("查询验证令牌失败: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("期望创建1个验证令牌，实际创建了%d个", len(tokens))
	}

	token := tokens[0]
	if token.Email != email {
		t.Errorf("期望邮箱为%s，实际为%s", email, token.Email)
	}
	if token.Used {
		t.Error("验证令牌不应该被标记为已使用")
	}
}

// TestVerifyEmail 测试验证邮箱
func TestVerifyEmail(t *testing.T) {
	// 1. 设置测试环境
	db := setupTestDB(t)
	verificationRepo := repository.NewEmailVerificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	mockSender := &mockEmailSender{}
	
	emailService := NewEmailService(
		verificationRepo,
		userRepo,
		mockSender,
		24*time.Hour,
	)

	// 2. 创建测试租户和用户
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	email := "test@example.com"

	tenant := &model.Tenant{
		ID:     tenantID,
		Name:   "测试租户",
		Status: true,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("创建测试租户失败: %v", err)
	}

	user := &model.User{
		ID:            userID,
		TenantID:      tenantID,
		Email:         email,
		EmailVerified: false,
		PasswordHash:  "hash",
		DisplayName:   "测试用户",
		IsActive:      true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 3. 创建验证令牌
	token := uuid.New().String()
	verificationToken := &model.EmailVerificationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		Token:     token,
		Email:     email,
		Used:      false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(verificationToken).Error; err != nil {
		t.Fatalf("创建验证令牌失败: %v", err)
	}

	// 4. 验证邮箱
	ctx := context.Background()
	err := emailService.VerifyEmail(ctx, token)
	if err != nil {
		t.Fatalf("验证邮箱失败: %v", err)
	}

	// 5. 验证用户邮箱状态已更新
	var updatedUser model.User
	if err := db.Where("id = ?", userID).First(&updatedUser).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	if !updatedUser.EmailVerified {
		t.Error("用户邮箱应该被标记为已验证")
	}

	// 6. 验证令牌已被标记为已使用
	var updatedToken model.EmailVerificationToken
	if err := db.Where("id = ?", verificationToken.ID).First(&updatedToken).Error; err != nil {
		t.Fatalf("查询验证令牌失败: %v", err)
	}

	if !updatedToken.Used {
		t.Error("验证令牌应该被标记为已使用")
	}
}

// TestVerifyEmailWithExpiredToken 测试使用过期令牌验证邮箱
func TestVerifyEmailWithExpiredToken(t *testing.T) {
	// 1. 设置测试环境
	db := setupTestDB(t)
	verificationRepo := repository.NewEmailVerificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	mockSender := &mockEmailSender{}
	
	emailService := NewEmailService(
		verificationRepo,
		userRepo,
		mockSender,
		24*time.Hour,
	)

	// 2. 创建测试租户和用户
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	email := "test@example.com"

	tenant := &model.Tenant{
		ID:     tenantID,
		Name:   "测试租户",
		Status: true,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("创建测试租户失败: %v", err)
	}

	user := &model.User{
		ID:            userID,
		TenantID:      tenantID,
		Email:         email,
		EmailVerified: false,
		PasswordHash:  "hash",
		DisplayName:   "测试用户",
		IsActive:      true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 3. 创建过期的验证令牌
	token := uuid.New().String()
	verificationToken := &model.EmailVerificationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		Token:     token,
		Email:     email,
		Used:      false,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // 已过期
	}
	if err := db.Create(verificationToken).Error; err != nil {
		t.Fatalf("创建验证令牌失败: %v", err)
	}

	// 4. 尝试验证邮箱（应该失败）
	ctx := context.Background()
	err := emailService.VerifyEmail(ctx, token)
	if err == nil {
		t.Fatal("使用过期令牌验证邮箱应该失败")
	}

	// 5. 验证用户邮箱状态未更新
	var updatedUser model.User
	if err := db.Where("id = ?", userID).First(&updatedUser).Error; err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}

	if updatedUser.EmailVerified {
		t.Error("用户邮箱不应该被标记为已验证")
	}
}
