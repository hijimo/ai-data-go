package repository

import (
	"context"
	"errors"
	"time"

	"genkit-ai-service/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *model.User) error

	// GetByID 根据 ID 获取用户
	GetByID(ctx context.Context, tenantID, userID string) (*model.User, error)

	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx context.Context, tenantID string, email string) (*model.User, error)

	// Update 更新用户
	Update(ctx context.Context, user *model.User) error

	// Delete 软删除用户
	Delete(ctx context.Context, tenantID, userID string) error

	// List 列出租户下的用户（支持分页）
	List(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error)

	// UpdateLastLogin 更新最后登录时间
	UpdateLastLogin(ctx context.Context, tenantID, userID string) error

	// IncrementFailedLoginAttempts 增加登录失败次数
	IncrementFailedLoginAttempts(ctx context.Context, tenantID, userID string) error

	// ResetFailedLoginAttempts 重置登录失败次数
	ResetFailedLoginAttempts(ctx context.Context, tenantID, userID string) error

	// LockAccount 锁定账户
	LockAccount(ctx context.Context, tenantID, userID string, lockDuration time.Duration) error

	// UnlockAccount 解锁账户
	UnlockAccount(ctx context.Context, tenantID, userID string) error

	// IsAccountLocked 检查账户是否被锁定
	IsAccountLocked(ctx context.Context, tenantID, userID string) (bool, error)
}

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// 验证租户 ID
	if user.TenantID == uuid.Nil {
		return errors.New("tenant_id is required")
	}

	// 如果没有设置 ID，生成一个新的
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根据 ID 获取用户
func (r *userRepository) GetByID(ctx context.Context, tenantID, userID string) (*model.User, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	var user model.User

	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(ctx context.Context, tenantID string, email string) (*model.User, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	var user model.User

	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ? AND is_deleted = ?", tenantID, email, false).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// 验证租户 ID
	if user.TenantID == uuid.Nil {
		return errors.New("tenant_id is required")
	}

	// 确保只更新指定租户下未删除的用户
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", user.ID, user.TenantID, false).
		Updates(user)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// Delete 软删除用户
func (r *userRepository) Delete(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// List 列出租户下的用户（支持分页）
func (r *userRepository) List(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error) {
	if tenantID == "" {
		return nil, 0, errors.New("tenant_id is required")
	}

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var users []*model.User
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数（确保包含租户隔离）
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("tenant_id = ? AND is_deleted = ?", tenantID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据（确保包含租户隔离）
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_deleted = ?", tenantID, false).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *userRepository) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		Update("last_login_at", now)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// IncrementFailedLoginAttempts 增加登录失败次数
func (r *userRepository) IncrementFailedLoginAttempts(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		UpdateColumn("failed_login_attempts", gorm.Expr("failed_login_attempts + ?", 1))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// ResetFailedLoginAttempts 重置登录失败次数
func (r *userRepository) ResetFailedLoginAttempts(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		Updates(map[string]interface{}{
			"failed_login_attempts": 0,
			"locked_until":          nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// LockAccount 锁定账户
func (r *userRepository) LockAccount(ctx context.Context, tenantID, userID string, lockDuration time.Duration) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	lockUntil := time.Now().Add(lockDuration)

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		Update("locked_until", lockUntil)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// UnlockAccount 解锁账户
func (r *userRepository) UnlockAccount(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		Updates(map[string]interface{}{
			"failed_login_attempts": 0,
			"locked_until":          nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

// IsAccountLocked 检查账户是否被锁定
func (r *userRepository) IsAccountLocked(ctx context.Context, tenantID, userID string) (bool, error) {
	if tenantID == "" {
		return false, errors.New("tenant_id is required")
	}

	var user model.User

	err := r.db.WithContext(ctx).
		Select("locked_until").
		Where("id = ? AND tenant_id = ? AND is_deleted = ?", userID, tenantID, false).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("user not found")
		}
		return false, err
	}

	// 如果 locked_until 为 nil 或已过期，则账户未锁定
	if user.LockedUntil == nil || time.Now().After(*user.LockedUntil) {
		// 如果锁定时间已过期，自动解锁
		if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
			_ = r.UnlockAccount(ctx, tenantID, userID)
		}
		return false, nil
	}

	return true, nil
}
