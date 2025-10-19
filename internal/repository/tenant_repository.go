package repository

import (
	"context"
	"errors"

	"genkit-ai-service/internal/model"

	"gorm.io/gorm"
)

// TenantRepository 租户数据访问接口
type TenantRepository interface {
	// Create 创建租户
	Create(ctx context.Context, tenant *model.Tenant) error

	// GetByID 根据 ID 获取租户
	GetByID(ctx context.Context, id string) (*model.Tenant, error)

	// GetByDomain 根据域名获取租户
	GetByDomain(ctx context.Context, domain string) (*model.Tenant, error)

	// Update 更新租户
	Update(ctx context.Context, tenant *model.Tenant) error

	// Delete 软删除租户
	Delete(ctx context.Context, id string) error

	// List 列出租户（支持分页）
	List(ctx context.Context, page, pageSize int) ([]*model.Tenant, int64, error)
}

// tenantRepository 租户数据访问实现
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository 创建租户数据访问实例
func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{
		db: db,
	}
}

// Create 创建租户
func (r *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	if tenant == nil {
		return errors.New("tenant cannot be nil")
	}

	return r.db.WithContext(ctx).Create(tenant).Error
}

// GetByID 根据 ID 获取租户
func (r *tenantRepository) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	var tenant model.Tenant

	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&tenant).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}

	return &tenant, nil
}

// GetByDomain 根据域名获取租户
func (r *tenantRepository) GetByDomain(ctx context.Context, domain string) (*model.Tenant, error) {
	if domain == "" {
		return nil, errors.New("domain cannot be empty")
	}

	var tenant model.Tenant

	err := r.db.WithContext(ctx).
		Where("domain = ? AND is_deleted = ?", domain, false).
		First(&tenant).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}

	return &tenant, nil
}

// Update 更新租户
func (r *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	if tenant == nil {
		return errors.New("tenant cannot be nil")
	}

	// 确保只更新未删除的租户
	result := r.db.WithContext(ctx).
		Model(&model.Tenant{}).
		Where("id = ? AND is_deleted = ?", tenant.ID, false).
		Updates(tenant)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("tenant not found or already deleted")
	}

	return nil
}

// Delete 软删除租户
func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&model.Tenant{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("is_deleted", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("tenant not found or already deleted")
	}

	return nil
}

// List 列出租户（支持分页）
func (r *tenantRepository) List(ctx context.Context, page, pageSize int) ([]*model.Tenant, int64, error) {
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

	var tenants []*model.Tenant
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.db.WithContext(ctx).
		Model(&model.Tenant{}).
		Where("is_deleted = ?", false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	if err := r.db.WithContext(ctx).
		Where("is_deleted = ?", false).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}
