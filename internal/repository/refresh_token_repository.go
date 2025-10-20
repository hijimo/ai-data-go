package repository

import (
	"context"
	"errors"
	"time"

	"genkit-ai-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository 刷新令牌数据访问接口
type RefreshTokenRepository interface {
	// Create 创建刷新令牌
	Create(ctx context.Context, token *model.RefreshToken) error

	// GetByTokenHash 根据 token 哈希获取
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// Revoke 撤销令牌
	Revoke(ctx context.Context, tokenID string, replacedBy *string) error

	// RevokeAllByUser 撤销用户的所有令牌
	RevokeAllByUser(ctx context.Context, tenantID, userID string) error

	// DeleteExpired 删除过期的令牌
	DeleteExpired(ctx context.Context) error
}

// refreshTokenRepository 刷新令牌数据访问实现
type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository 创建刷新令牌数据访问实例
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		db: db,
	}
}

// Create 创建刷新令牌
func (r *refreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	if token == nil {
		return errors.New("token cannot be nil")
	}

	// 验证必需字段
	if token.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	if token.TenantID == uuid.Nil {
		return errors.New("tenant_id is required")
	}
	if token.TokenHash == "" {
		return errors.New("token_hash is required")
	}

	// 如果没有设置 ID，生成一个新的
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(token).Error
}

// GetByTokenHash 根据 token 哈希获取
func (r *refreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if tokenHash == "" {
		return nil, errors.New("token_hash cannot be empty")
	}

	var token model.RefreshToken

	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	return &token, nil
}

// Revoke 撤销令牌（实现 token 轮换逻辑）
func (r *refreshTokenRepository) Revoke(ctx context.Context, tokenID string, replacedBy *string) error {
	if tokenID == "" {
		return errors.New("token_id is required")
	}

	// 构建更新数据
	updates := map[string]interface{}{
		"revoked": true,
	}

	// 如果提供了 replaced_by，记录轮换关系
	if replacedBy != nil && *replacedBy != "" {
		updates["replaced_by"] = replacedBy
	}

	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ?", tokenID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("token not found")
	}

	return nil
}

// RevokeAllByUser 撤销用户的所有令牌
func (r *refreshTokenRepository) RevokeAllByUser(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}
	if userID == "" {
		return errors.New("user_id is required")
	}

	// 撤销该用户在指定租户下的所有未撤销的令牌
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND tenant_id = ? AND revoked = ?", userID, tenantID, false).
		Update("revoked", true)

	if result.Error != nil {
		return result.Error
	}

	// 注意：这里不检查 RowsAffected，因为用户可能没有活跃的令牌

	return nil
}

// DeleteExpired 删除过期的令牌
func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()

	// 删除已过期的令牌记录
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&model.RefreshToken{})

	if result.Error != nil {
		return result.Error
	}

	// 可以记录删除的数量用于监控
	// log.Printf("Deleted %d expired refresh tokens", result.RowsAffected)

	return nil
}
