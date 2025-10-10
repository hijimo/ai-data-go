package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"your-project/internal/model"
)

// LLMRepository LLM仓库接口
type LLMRepository interface {
	// 提供商管理
	CreateProvider(ctx context.Context, provider *model.LLMProvider) error
	GetProvider(ctx context.Context, id uuid.UUID) (*model.LLMProvider, error)
	GetProviderByName(ctx context.Context, name string) (*model.LLMProvider, error)
	ListProviders(ctx context.Context, isActive *bool) ([]*model.LLMProvider, error)
	UpdateProvider(ctx context.Context, provider *model.LLMProvider) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error
	
	// 模型管理
	CreateModel(ctx context.Context, model *model.LLMModel) error
	GetModel(ctx context.Context, id uuid.UUID) (*model.LLMModel, error)
	GetModelByName(ctx context.Context, providerID uuid.UUID, modelName string) (*model.LLMModel, error)
	ListModels(ctx context.Context, providerID *uuid.UUID, modelType *string, isActive *bool) ([]*model.LLMModel, error)
	UpdateModel(ctx context.Context, model *model.LLMModel) error
	DeleteModel(ctx context.Context, id uuid.UUID) error
	
	// 批量操作
	CreateModelsForProvider(ctx context.Context, providerID uuid.UUID, models []*model.LLMModel) error
	DeleteModelsByProvider(ctx context.Context, providerID uuid.UUID) error
}

// llmRepository LLM仓库实现
type llmRepository struct {
	db *gorm.DB
}

// NewLLMRepository 创建LLM仓库
func NewLLMRepository(db *gorm.DB) LLMRepository {
	return &llmRepository{
		db: db,
	}
}

// CreateProvider 创建提供商
func (r *llmRepository) CreateProvider(ctx context.Context, provider *model.LLMProvider) error {
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	return r.db.WithContext(ctx).Create(provider).Error
}

// GetProvider 获取提供商
func (r *llmRepository) GetProvider(ctx context.Context, id uuid.UUID) (*model.LLMProvider, error) {
	var provider model.LLMProvider
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		Preload("Models", "is_deleted = ?", false).
		First(&provider).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, model.ErrProviderNotFound
		}
		return nil, err
	}
	
	return &provider, nil
}

// GetProviderByName 根据名称获取提供商
func (r *llmRepository) GetProviderByName(ctx context.Context, name string) (*model.LLMProvider, error) {
	var provider model.LLMProvider
	err := r.db.WithContext(ctx).
		Where("name = ? AND is_deleted = ?", name, false).
		Preload("Models", "is_deleted = ?", false).
		First(&provider).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, model.ErrProviderNotFound
		}
		return nil, err
	}
	
	return &provider, nil
}

// ListProviders 列出提供商
func (r *llmRepository) ListProviders(ctx context.Context, isActive *bool) ([]*model.LLMProvider, error) {
	var providers []*model.LLMProvider
	
	query := r.db.WithContext(ctx).Where("is_deleted = ?", false)
	
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	
	err := query.
		Preload("Models", "is_deleted = ?", false).
		Order("created_at DESC").
		Find(&providers).Error
	
	return providers, err
}

// UpdateProvider 更新提供商
func (r *llmRepository) UpdateProvider(ctx context.Context, provider *model.LLMProvider) error {
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	return r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", provider.ID, false).
		Updates(provider).Error
}

// DeleteProvider 删除提供商
func (r *llmRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 软删除提供商
		now := gorm.Expr("NOW()")
		err := tx.Model(&model.LLMProvider{}).
			Where("id = ? AND is_deleted = ?", id, false).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"deleted_at": now,
				"updated_at": now,
			}).Error
		if err != nil {
			return err
		}
		
		// 软删除关联的模型
		err = tx.Model(&model.LLMModel{}).
			Where("provider_id = ? AND is_deleted = ?", id, false).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"deleted_at": now,
			}).Error
		
		return err
	})
}

// CreateModel 创建模型
func (r *llmRepository) CreateModel(ctx context.Context, model *model.LLMModel) error {
	if err := model.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	return r.db.WithContext(ctx).Create(model).Error
}

// GetModel 获取模型
func (r *llmRepository) GetModel(ctx context.Context, id uuid.UUID) (*model.LLMModel, error) {
	var model model.LLMModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		Preload("Provider", "is_deleted = ?", false).
		First(&model).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, model.ErrModelNotFound
		}
		return nil, err
	}
	
	return &model, nil
}

// GetModelByName 根据名称获取模型
func (r *llmRepository) GetModelByName(ctx context.Context, providerID uuid.UUID, modelName string) (*model.LLMModel, error) {
	var model model.LLMModel
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND model_name = ? AND is_deleted = ?", providerID, modelName, false).
		Preload("Provider", "is_deleted = ?", false).
		First(&model).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, model.ErrModelNotFound
		}
		return nil, err
	}
	
	return &model, nil
}

// ListModels 列出模型
func (r *llmRepository) ListModels(ctx context.Context, providerID *uuid.UUID, modelType *string, isActive *bool) ([]*model.LLMModel, error) {
	var models []*model.LLMModel
	
	query := r.db.WithContext(ctx).Where("is_deleted = ?", false)
	
	if providerID != nil {
		query = query.Where("provider_id = ?", *providerID)
	}
	
	if modelType != nil {
		query = query.Where("model_type = ?", *modelType)
	}
	
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	
	err := query.
		Preload("Provider", "is_deleted = ?", false).
		Order("created_at DESC").
		Find(&models).Error
	
	return models, err
}

// UpdateModel 更新模型
func (r *llmRepository) UpdateModel(ctx context.Context, model *model.LLMModel) error {
	if err := model.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	
	return r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", model.ID, false).
		Updates(model).Error
}

// DeleteModel 删除模型
func (r *llmRepository) DeleteModel(ctx context.Context, id uuid.UUID) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
		}).Error
}

// CreateModelsForProvider 为提供商创建多个模型
func (r *llmRepository) CreateModelsForProvider(ctx context.Context, providerID uuid.UUID, models []*model.LLMModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, model := range models {
			model.ProviderID = providerID
			if err := model.ValidateConfig(); err != nil {
				return fmt.Errorf("模型 %s 配置验证失败: %w", model.ModelName, err)
			}
			
			if err := tx.Create(model).Error; err != nil {
				return fmt.Errorf("创建模型 %s 失败: %w", model.ModelName, err)
			}
		}
		return nil
	})
}

// DeleteModelsByProvider 删除提供商的所有模型
func (r *llmRepository) DeleteModelsByProvider(ctx context.Context, providerID uuid.UUID) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Where("provider_id = ? AND is_deleted = ?", providerID, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
		}).Error
}

// GetProviderStats 获取提供商统计信息
func (r *llmRepository) GetProviderStats(ctx context.Context, providerID uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 统计模型数量
	var totalModels, activeModels int64
	
	err := r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Where("provider_id = ? AND is_deleted = ?", providerID, false).
		Count(&totalModels).Error
	if err != nil {
		return nil, err
	}
	
	err = r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Where("provider_id = ? AND is_deleted = ? AND is_active = ?", providerID, false, true).
		Count(&activeModels).Error
	if err != nil {
		return nil, err
	}
	
	stats["total_models"] = totalModels
	stats["active_models"] = activeModels
	
	// 按模型类型统计
	var modelTypeStats []struct {
		ModelType string `json:"model_type"`
		Count     int64  `json:"count"`
	}
	
	err = r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Select("model_type, COUNT(*) as count").
		Where("provider_id = ? AND is_deleted = ?", providerID, false).
		Group("model_type").
		Scan(&modelTypeStats).Error
	if err != nil {
		return nil, err
	}
	
	stats["model_type_stats"] = modelTypeStats
	
	return stats, nil
}

// CheckProviderExists 检查提供商是否存在
func (r *llmRepository) CheckProviderExists(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&model.LLMProvider{}).
		Where("name = ? AND is_deleted = ?", name, false)
	
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	
	err := query.Count(&count).Error
	return count > 0, err
}

// CheckModelExists 检查模型是否存在
func (r *llmRepository) CheckModelExists(ctx context.Context, providerID uuid.UUID, modelName string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&model.LLMModel{}).
		Where("provider_id = ? AND model_name = ? AND is_deleted = ?", providerID, modelName, false)
	
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	
	err := query.Count(&count).Error
	return count > 0, err
}