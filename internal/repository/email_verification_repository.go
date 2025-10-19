package repository

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/model"

	"gorm.io/gorm"
)

// EmailVerificationRepository 邮箱验证令牌仓库接口
type EmailVerificationRepository interface {
	// Create 创建验证令牌
	Create(ctx context.Context, token *model.EmailVerificationToken) error
	
	// GetByToken 根据令牌获取验证记录
	GetByToken(ctx context.Context, token string) (*model.EmailVerificationToken, error)
	
	// MarkAsUsed 标记令牌为已使用
	MarkAsUsed(ctx context.Context, tokenID string) error
	
	// DeleteExpired 删除过期的验证令牌
	DeleteExpired(ctx context.Context) error
	
	// GetByUserID 获取用户的验证令牌
	GetByUserID(ctx context.Context, tenantID, userID string) ([]*model.EmailVerificationToken, error)
}

// emailVerificationRepository 邮箱验证令牌仓库实现
type emailVerificationRepository struct {
	db *gorm.DB
}

// NewEmailVerificationRepository 创建邮箱验证令牌仓库实例
func NewEmailVerificationRepository(db *gorm.DB) EmailVerificationRepository {
	return &emailVerificationRepository{
		db: db,
	}
}

// Create 创建验证令牌
func (r *emailVerificationRepository) Create(ctx context.Context, token *model.EmailVerificationToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("创建邮箱验证令牌失败: %w", err)
	}
	return nil
}

// GetByToken 根据令牌获取验证记录
func (r *emailVerificationRepository) GetByToken(ctx context.Context, token string) (*model.EmailVerificationToken, error) {
	var verificationToken model.EmailVerificationToken
	if err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&verificationToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("验证令牌不存在")
		}
		return nil, fmt.Errorf("查询验证令牌失败: %w", err)
	}
	return &verificationToken, nil
}

// MarkAsUsed 标记令牌为已使用
func (r *emailVerificationRepository) MarkAsUsed(ctx context.Context, tokenID string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.EmailVerificationToken{}).
		Where("id = ?", tokenID).
		Update("used", true).Error; err != nil {
		return fmt.Errorf("标记验证令牌为已使用失败: %w", err)
	}
	return nil
}

// DeleteExpired 删除过期的验证令牌
func (r *emailVerificationRepository) DeleteExpired(ctx context.Context) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.EmailVerificationToken{}).Error; err != nil {
		return fmt.Errorf("删除过期验证令牌失败: %w", err)
	}
	return nil
}

// GetByUserID 获取用户的验证令牌
func (r *emailVerificationRepository) GetByUserID(ctx context.Context, tenantID, userID string) ([]*model.EmailVerificationToken, error) {
	var tokens []*model.EmailVerificationToken
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("查询用户验证令牌失败: %w", err)
	}
	return tokens, nil
}
