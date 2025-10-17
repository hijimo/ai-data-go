package storage

import (
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
	"sync"
)

// Store 存储接口定义
type Store interface {
	// SetProviders 设置提供商列表
	SetProviders(providers []model.Provider)

	// GetProviders 获取所有提供商
	GetProviders() []model.Provider

	// GetProvider 根据ID获取提供商
	GetProvider(providerID string) (*model.Provider, error)

	// SetModels 设置提供商的模型列表
	SetModels(providerID string, models []model.Model)

	// GetModels 获取提供商的所有模型
	GetModels(providerID string) ([]model.Model, error)

	// GetModel 获取指定模型
	GetModel(providerID, modelID string) (*model.Model, error)

	// GetProvidersCount 获取提供商数量
	GetProvidersCount() int

	// GetModelsCount 获取模型总数
	GetModelsCount() int
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu        sync.RWMutex
	providers map[string]*model.Provider // key: provider_id
	models    map[string][]model.Model   // key: provider_id, value: models list
}

// NewMemoryStore 创建新的内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		providers: make(map[string]*model.Provider),
		models:    make(map[string][]model.Model),
	}
}

// SetProviders 设置提供商列表
func (s *MemoryStore) SetProviders(providers []model.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清空现有数据
	s.providers = make(map[string]*model.Provider)

	// 存储新数据
	for i := range providers {
		s.providers[providers[i].ID] = &providers[i]
	}
}

// GetProviders 获取所有提供商
func (s *MemoryStore) GetProviders() []model.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]model.Provider, 0, len(s.providers))
	for _, provider := range s.providers {
		providers = append(providers, *provider)
	}

	return providers
}

// GetProvider 根据ID获取提供商
func (s *MemoryStore) GetProvider(providerID string) (*model.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, exists := s.providers[providerID]
	if !exists {
		return nil, errors.NewProviderNotFoundError(providerID)
	}

	// 返回副本以避免外部修改
	providerCopy := *provider
	return &providerCopy, nil
}

// SetModels 设置提供商的模型列表
func (s *MemoryStore) SetModels(providerID string, models []model.Model) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 存储模型列表（创建副本）
	modelsCopy := make([]model.Model, len(models))
	copy(modelsCopy, models)
	s.models[providerID] = modelsCopy
}

// GetModels 获取提供商的所有模型
func (s *MemoryStore) GetModels(providerID string) ([]model.Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查提供商是否存在
	if _, exists := s.providers[providerID]; !exists {
		return nil, errors.NewProviderNotFoundError(providerID)
	}

	// 获取模型列表
	models, exists := s.models[providerID]
	if !exists {
		// 提供商存在但没有模型，返回空列表
		return []model.Model{}, nil
	}

	// 返回副本以避免外部修改
	modelsCopy := make([]model.Model, len(models))
	copy(modelsCopy, models)
	return modelsCopy, nil
}

// GetModel 获取指定模型
func (s *MemoryStore) GetModel(providerID, modelID string) (*model.Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查提供商是否存在
	if _, exists := s.providers[providerID]; !exists {
		return nil, errors.NewProviderNotFoundError(providerID)
	}

	// 获取模型列表
	models, exists := s.models[providerID]
	if !exists {
		return nil, errors.NewModelNotFoundError(modelID)
	}

	// 查找指定模型
	for i := range models {
		if models[i].Model == modelID {
			// 返回副本以避免外部修改
			modelCopy := models[i]
			return &modelCopy, nil
		}
	}

	return nil, errors.NewModelNotFoundError(modelID)
}

// GetProvidersCount 获取提供商数量
func (s *MemoryStore) GetProvidersCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.providers)
}

// GetModelsCount 获取模型总数
func (s *MemoryStore) GetModelsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, models := range s.models {
		count += len(models)
	}

	return count
}
