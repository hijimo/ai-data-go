package storage

import (
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"
	"sync"
)

// Store 存储接口定义
type Store interface {
	// SetModels 设置提供商的模型列表
	SetModels(providerID string, models []model.Model)

	// GetModels 获取提供商的所有模型
	GetModels(providerID string) ([]model.Model, error)

	// GetModel 获取指定模型
	GetModel(providerID, modelID string) (*model.Model, error)

	// GetModelsCount 获取模型总数
	GetModelsCount() int
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	mu     sync.RWMutex
	models map[string][]model.Model // key: provider_id, value: models list
}

// NewMemoryStore 创建新的内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		models: make(map[string][]model.Model),
	}
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

	// 获取模型列表
	models, exists := s.models[providerID]
	if !exists {
		// 提供商不存在或没有模型，返回空列表
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
