package auth

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
)

// EmailService 邮箱服务接口
type EmailService interface {
	// SendVerificationEmail 发送验证邮件
	SendVerificationEmail(ctx context.Context, tenantID, userID, email string) error

	// VerifyEmail 验证邮箱
	VerifyEmail(ctx context.Context, token string) error

	// ResendVerificationEmail 重新发送验证邮件
	ResendVerificationEmail(ctx context.Context, tenantID, userID string) error
}

// emailService 邮箱服务实现
type emailService struct {
	verificationRepo repository.EmailVerificationRepository
	userRepo         repository.UserRepository
	emailSender      EmailSender
	tokenTTL         time.Duration
}

// EmailSender 邮件发送器接口
type EmailSender interface {
	// SendEmail 发送邮件
	SendEmail(to, subject, body string) error
}

// NewEmailService 创建邮箱服务实例
func NewEmailService(
	verificationRepo repository.EmailVerificationRepository,
	userRepo repository.UserRepository,
	emailSender EmailSender,
	tokenTTL time.Duration,
) EmailService {
	return &emailService{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		emailSender:      emailSender,
		tokenTTL:         tokenTTL,
	}
}

// SendVerificationEmail 发送验证邮件
func (s *emailService) SendVerificationEmail(ctx context.Context, tenantID, userID, email string) error {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("无效的租户ID: %w", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效的用户ID: %w", err)
	}

	// 生成验证令牌
	token := uuid.New().String()

	// 创建验证令牌记录
	verificationToken := &model.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userUUID,
		TenantID:  tenantUUID,
		Token:     token,
		Email:     email,
		Used:      false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.tokenTTL),
	}

	if err := s.verificationRepo.Create(ctx, verificationToken); err != nil {
		return fmt.Errorf("创建验证令牌失败: %w", err)
	}

	// 构建验证链接
	verificationLink := fmt.Sprintf("https://your-domain.com/verify-email?token=%s", token)

	// 构建邮件内容
	subject := "验证您的邮箱地址"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>邮箱验证</h2>
			<p>请点击下面的链接验证您的邮箱地址：</p>
			<p><a href="%s">验证邮箱</a></p>
			<p>或者复制以下链接到浏览器：</p>
			<p>%s</p>
			<p>此链接将在 %d 小时后过期。</p>
			<p>如果您没有注册账户，请忽略此邮件。</p>
		</body>
		</html>
	`, verificationLink, verificationLink, int(s.tokenTTL.Hours()))

	// 发送邮件
	if err := s.emailSender.SendEmail(email, subject, body); err != nil {
		return fmt.Errorf("发送验证邮件失败: %w", err)
	}

	return nil
}

// VerifyEmail 验证邮箱
func (s *emailService) VerifyEmail(ctx context.Context, token string) error {
	// 查询验证令牌
	verificationToken, err := s.verificationRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("验证令牌无效: %w", err)
	}

	// 检查令牌是否已使用
	if verificationToken.Used {
		return fmt.Errorf("验证令牌已使用")
	}

	// 检查令牌是否过期
	if time.Now().After(verificationToken.ExpiresAt) {
		return fmt.Errorf("验证令牌已过期")
	}

	// 更新用户邮箱验证状态
	user, err := s.userRepo.GetByID(ctx, verificationToken.TenantID.String(), verificationToken.UserID.String())
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	user.EmailVerified = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("更新用户邮箱验证状态失败: %w", err)
	}

	// 标记令牌为已使用
	if err := s.verificationRepo.MarkAsUsed(ctx, verificationToken.ID.String()); err != nil {
		return fmt.Errorf("标记验证令牌为已使用失败: %w", err)
	}

	return nil
}

// ResendVerificationEmail 重新发送验证邮件
func (s *emailService) ResendVerificationEmail(ctx context.Context, tenantID, userID string) error {
	// 查询用户
	user, err := s.userRepo.GetByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查邮箱是否已验证
	if user.EmailVerified {
		return fmt.Errorf("邮箱已验证")
	}

	// 发送验证邮件
	return s.SendVerificationEmail(ctx, tenantID, userID, user.Email)
}
