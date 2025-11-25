package repository

import (
	"context"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelConfigurationRepository 模型配置仓储接口
type ModelConfigurationRepository interface {
	// Create 创建模型配置
	Create(ctx context.Context, config *model.ModelConfiguration) (*model.ModelConfiguration, error)

	// FindByID 根据ID查询模型配置
	FindByID(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error)

	// FindByTenant 根据租户ID查询模型配置列表（支持分页）
	FindByTenant(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error)

	// Update 更新模型配置
	Update(ctx context.Context, id uuid.UUID, config *model.ModelConfiguration) (*model.ModelConfiguration, error)

	// UpdateStatus 更新模型配置状态
	UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error

	// SoftDelete 软删除模型配置
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error

	// FindAvailableByTenant 查询租户下所有可用的模型配置（已启用且未删除）
	FindAvailableByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ModelConfiguration, error)

	// GetByTenantAndModel 根据租户ID和模型名称获取配置
	GetByTenantAndModel(ctx context.Context, tenantID uuid.UUID, modelName string) (*model.ModelConfiguration, error)
}

// modelConfigurationRepository 模型配置仓储实现
type modelConfigurationRepository struct {
	db *gorm.DB
}

// NewModelConfigurationRepository 创建新的模型配置仓储实例
func NewModelConfigurationRepository(db *gorm.DB) ModelConfigurationRepository {
	return &modelConfigurationRepository{
		db: db,
	}
}

// Create 创建模型配置
func (r *modelConfigurationRepository) Create(ctx context.Context, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return nil, errors.NewInternalError(err)
	}
	return config, nil
}

// FindByID 根据ID查询模型配置
func (r *modelConfigurationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error) {
	var config model.ModelConfiguration
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("模型配置不存在")
		}
		return nil, errors.NewInternalError(err)
	}

	return &config, nil
}

// FindByTenant 根据租户ID查询模型配置列表（支持分页）
func (r *modelConfigurationRepository) FindByTenant(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error) {
	var configs []*model.ModelConfiguration
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ModelConfiguration{}).
		Where("is_deleted = ?", false)

	// 如果指定了租户ID，添加租户过滤
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternalError(err)
	}

	// 分页查询
	offset := (pageNo - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").
		Find(&configs).Error; err != nil {
		return nil, 0, errors.NewInternalError(err)
	}

	return configs, total, nil
}

// Update 更新模型配置
func (r *modelConfigurationRepository) Update(ctx context.Context, id uuid.UUID, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	// 先检查记录是否存在
	var existing model.ModelConfiguration
	if err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("模型配置不存在")
		}
		return nil, errors.NewInternalError(err)
	}

	// 更新记录
	if err := r.db.WithContext(ctx).
		Model(&existing).
		Updates(config).Error; err != nil {
		return nil, errors.NewInternalError(err)
	}

	// 重新查询更新后的记录
	return r.FindByID(ctx, id)
}

// UpdateStatus 更新模型配置状态
func (r *modelConfigurationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error {
	result := r.db.WithContext(ctx).
		Model(&model.ModelConfiguration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("is_enabled", enabled)

	if result.Error != nil {
		return errors.NewInternalError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("模型配置不存在")
	}

	return nil
}

// SoftDelete 软删除模型配置
func (r *modelConfigurationRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.ModelConfiguration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_by": deletedBy,
			"deleted_at": now,
		})

	if result.Error != nil {
		return errors.NewInternalError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("模型配置不存在")
	}

	return nil
}

// FindAvailableByTenant 查询租户下所有可用的模型配置（已启用且未删除）
func (r *modelConfigurationRepository) FindAvailableByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ModelConfiguration, error) {
	var configs []*model.ModelConfiguration

	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_enabled = ? AND is_deleted = ?", tenantID, true, false).
		Order("created_at DESC").
		Find(&configs).Error

	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	return configs, nil
}

// GetByTenantAndModel 根据租户ID和模型名称获取配置
func (r *modelConfigurationRepository) GetByTenantAndModel(ctx context.Context, tenantID uuid.UUID, modelName string) (*model.ModelConfiguration, error) {
	var config model.ModelConfiguration
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND is_deleted = ?", tenantID, modelName, false).
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("模型配置不存在")
		}
		return nil, errors.NewInternalError(err)
	}

	return &config, nil
}
