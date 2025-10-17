package service

import (
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/storage"
)

// ProviderService 提供商服务接口
type ProviderService interface {
	// GetAllProviders 获取所有提供商（返回列表项格式）
	GetAllProviders() []model.ProviderListItem

	// GetProviderByID 根据ID获取提供商详情
	GetProviderByID(providerID string) (*model.Provider, error)

	// GetProviderModels 获取提供商的所有模型（返回列表项格式）
	GetProviderModels(providerID string) ([]model.ModelListItem, error)

	// GetProviderModel 获取提供商的指定模型
	GetProviderModel(providerID, modelID string) (*model.Model, error)

	// GetModelParameterRules 获取模型的参数规则
	GetModelParameterRules(providerID, modelID string) ([]model.ParameterRule, error)
}

// providerService 提供商服务实现
type providerService struct {
	store storage.Store
}

// NewProviderService 创建新的提供商服务实例
func NewProviderService(store storage.Store) ProviderService {
	return &providerService{
		store: store,
	}
}

// GetAllProviders 获取所有提供商（返回列表项格式）
func (s *providerService) GetAllProviders() []model.ProviderListItem {
	// 从存储层获取所有提供商
	providers := s.store.GetProviders()

	// 转换为列表项格式
	listItems := make([]model.ProviderListItem, 0, len(providers))
	for _, provider := range providers {
		listItems = append(listItems, model.ProviderListItem{
			ID:                 provider.ID,
			Provider:           provider.Provider,
			Label:              provider.Label,
			Background:         provider.Background,
			IconSmall:          provider.IconSmall,
			IconLarge:          provider.IconLarge,
			Help:               provider.Help,
			ConfigurateMethods: provider.ConfigurateMethods,
		})
	}

	return listItems
}

// GetProviderByID 根据ID获取提供商详情
func (s *providerService) GetProviderByID(providerID string) (*model.Provider, error) {
	// 从存储层获取提供商
	provider, err := s.store.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

// GetProviderModels 获取提供商的所有模型（返回列表项格式）
func (s *providerService) GetProviderModels(providerID string) ([]model.ModelListItem, error) {
	// 从存储层获取模型列表
	models, err := s.store.GetModels(providerID)
	if err != nil {
		return nil, err
	}

	// 转换为列表项格式
	listItems := make([]model.ModelListItem, 0, len(models))
	for _, m := range models {
		listItems = append(listItems, model.ModelListItem{
			Model:           m.Model,
			Label:           m.Label,
			ModelType:       m.ModelType,
			Features:        m.Features,
			ModelProperties: m.ModelProperties,
			ParameterRules:  m.ParameterRules,
			Pricing:         m.Pricing,
		})
	}

	return listItems, nil
}

// GetProviderModel 获取提供商的指定模型
func (s *providerService) GetProviderModel(providerID, modelID string) (*model.Model, error) {
	// 从存储层获取模型
	m, err := s.store.GetModel(providerID, modelID)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// GetModelParameterRules 获取模型的参数规则
func (s *providerService) GetModelParameterRules(providerID, modelID string) ([]model.ParameterRule, error) {
	// 从存储层获取模型
	m, err := s.store.GetModel(providerID, modelID)
	if err != nil {
		return nil, err
	}

	// 如果模型没有参数规则，返回空数组
	if m.ParameterRules == nil {
		return []model.ParameterRule{}, nil
	}

	return m.ParameterRules, nil
}
