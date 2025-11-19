package repository

import (
	"context"
	"errors"

	"genkit-ai-service/internal/model"

	"gorm.io/gorm"
)

// ProviderRepository 模型配置数据访问接口
type ProviderRepository interface {
	// Create 创建模型配置
	Create(ctx context.Context, config *model.ModelConfiguration) error

	// FindByID 根据ID查询模型配置
	FindByID(ctx context.Context, id string) (*model.ModelConfiguration, error)

	// FindByTenant 根据租户查询模型配置列表（支持分页）
	FindByTenant(ctx context.Context, tenantID *string, page, pageSize int) ([]*model.ModelConfiguration, int64, error)

	// Update 更新模型配置
	Update(ctx context.Context, id string, config *model.ModelConfiguration) error

	// UpdateStatus 更新模型配置启用/禁用状态
	UpdateStatus(ctx context.Context, id string, enabled bool) error

	// SoftDelete 逻辑删除模型配置
	SoftDelete(ctx context.Context, id string, deletedBy string) error

	// FindAvailableByTenant 查询租户下所有可用的模型配置（已启用且未删除）
	FindAvailableByTenant(ctx context.Context, tenantID string) ([]*model.ModelConfiguration, error)
}

// providerRepository 模型配置数据访问实现
type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository 创建模型配置数据访问实例
func NewProviderRepository(db *gorm.DB) ProviderRepository {
	return &providerRepository{
		db: db,
	}
}

// Create 创建模型配置
func (r *providerRepository) Create(ctx context.Context, config *model.ModelConfiguration) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// 验证租户ID
	if config.TenantID == "" {
		return errors.New("tenant_id is required")
	}

	// 验证必填字段
	if config.Name == "" {
		return errors.New("name is required")
	}
	if config.Model == "" {
		return errors.New("model is required")
	}
	if config.ModelProvider == "" {
		return errors.New("model_provider is required")
	}
	if config.APIKey == "" {
		return errors.New("api_key is required")
	}

	// 验证模型提供商
	if !model.IsValidModelProvider(config.ModelProvider) {
		return errors.New("invalid model provider")
	}

	return r.db.WithContext(ctx).Create(config).Error
}

// FindByID 根据ID查询模型配置
func (r *providerRepository) FindByID(ctx context.Context, id string) (*model.ModelConfiguration, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}

	var config model.ModelConfiguration

	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("model configuration not found")
		}
		return nil, err
	}

	return &config, nil
}

// FindByTenant 根据租户查询模型配置列表（支持分页）
func (r *providerRepository) FindByTenant(ctx context.Context, tenantID *string, page, pageSize int) ([]*model.ModelConfiguration, int64, error) {
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

	var configs []*model.ModelConfiguration
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建基础查询
	query := r.db.WithContext(ctx).Model(&model.ModelConfiguration{}).
		Where("is_deleted = ?", false)

	// 如果指定了租户ID，添加租户过滤
	if tenantID != nil && *tenantID != "" {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	if err := query.
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// Update 更新模型配置
func (r *providerRepository) Update(ctx context.Context, id string, config *model.ModelConfiguration) error {
	if id == "" {
		return errors.New("id is required")
	}
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// 构建更新字段映射
	updates := map[string]interface{}{
		"name":        config.Name,
		"model":       config.Model,
		"base_url":    config.BaseURL,
		"api_key":     config.APIKey,
		"query_params": config.QueryParams,
		"updated_by":  config.UpdatedBy,
		"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
	}

	// 确保只更新未删除的配置
	result := r.db.WithContext(ctx).
		Model(&model.ModelConfiguration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("model configuration not found or already deleted")
	}

	return nil
}

// UpdateStatus 更新模型配置启用/禁用状态
func (r *providerRepository) UpdateStatus(ctx context.Context, id string, enabled bool) error {
	if id == "" {
		return errors.New("id is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ModelConfiguration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_enabled": enabled,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("model configuration not found or already deleted")
	}

	return nil
}

// SoftDelete 逻辑删除模型配置
func (r *providerRepository) SoftDelete(ctx context.Context, id string, deletedBy string) error {
	if id == "" {
		return errors.New("id is required")
	}
	if deletedBy == "" {
		return errors.New("deleted_by is required")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ModelConfiguration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("model configuration not found or already deleted")
	}

	return nil
}

// FindAvailableByTenant 查询租户下所有可用的模型配置（已启用且未删除）
func (r *providerRepository) FindAvailableByTenant(ctx context.Context, tenantID string) ([]*model.ModelConfiguration, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	var configs []*model.ModelConfiguration

	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_enabled = ? AND is_deleted = ?", tenantID, true, false).
		Order("created_at DESC").
		Find(&configs).Error

	if err != nil {
		return nil, err
	}

	return configs, nil
}
