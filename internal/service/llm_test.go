package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"your-project/internal/llm"
	"your-project/internal/model"
)

// MockLLMRepository LLM仓库模拟
type MockLLMRepository struct {
	mock.Mock
}

func (m *MockLLMRepository) CreateProvider(ctx context.Context, provider *model.LLMProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockLLMRepository) GetProvider(ctx context.Context, id uuid.UUID) (*model.LLMProvider, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.LLMProvider), args.Error(1)
}

func (m *MockLLMRepository) GetProviderByName(ctx context.Context, name string) (*model.LLMProvider, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*model.LLMProvider), args.Error(1)
}

func (m *MockLLMRepository) ListProviders(ctx context.Context, isActive *bool) ([]*model.LLMProvider, error) {
	args := m.Called(ctx, isActive)
	return args.Get(0).([]*model.LLMProvider), args.Error(1)
}

func (m *MockLLMRepository) UpdateProvider(ctx context.Context, provider *model.LLMProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockLLMRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLLMRepository) CreateModel(ctx context.Context, model *model.LLMModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockLLMRepository) GetModel(ctx context.Context, id uuid.UUID) (*model.LLMModel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.LLMModel), args.Error(1)
}

func (m *MockLLMRepository) GetModelByName(ctx context.Context, providerID uuid.UUID, modelName string) (*model.LLMModel, error) {
	args := m.Called(ctx, providerID, modelName)
	return args.Get(0).(*model.LLMModel), args.Error(1)
}

func (m *MockLLMRepository) ListModels(ctx context.Context, providerID *uuid.UUID, modelType *string, isActive *bool) ([]*model.LLMModel, error) {
	args := m.Called(ctx, providerID, modelType, isActive)
	return args.Get(0).([]*model.LLMModel), args.Error(1)
}

func (m *MockLLMRepository) UpdateModel(ctx context.Context, model *model.LLMModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockLLMRepository) DeleteModel(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLLMRepository) CreateModelsForProvider(ctx context.Context, providerID uuid.UUID, models []*model.LLMModel) error {
	args := m.Called(ctx, providerID, models)
	return args.Error(0)
}

func (m *MockLLMRepository) DeleteModelsByProvider(ctx context.Context, providerID uuid.UUID) error {
	args := m.Called(ctx, providerID)
	return args.Error(0)
}

func (m *MockLLMRepository) GetProviderStats(ctx context.Context, providerID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, providerID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLLMRepository) CheckProviderExists(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, name, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLLMRepository) CheckModelExists(ctx context.Context, providerID uuid.UUID, modelName string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, providerID, modelName, excludeID)
	return args.Bool(0), args.Error(1)
}

func TestLLMService_CreateProvider(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	req := &CreateProviderRequest{
		Name:         "test-openai",
		ProviderType: "openai",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	}

	// 模拟检查名称不存在
	mockRepo.On("CheckProviderExists", ctx, req.Name, (*uuid.UUID)(nil)).Return(false, nil)
	
	// 模拟创建成功
	mockRepo.On("CreateProvider", ctx, mock.AnythingOfType("*model.LLMProvider")).Return(nil)

	provider, err := service.CreateProvider(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, req.Name, provider.Name)
	assert.Equal(t, req.ProviderType, provider.ProviderType)
	assert.True(t, provider.IsActive)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_CreateProvider_DuplicateName(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	req := &CreateProviderRequest{
		Name:         "existing-provider",
		ProviderType: "openai",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	}

	// 模拟名称已存在
	mockRepo.On("CheckProviderExists", ctx, req.Name, (*uuid.UUID)(nil)).Return(true, nil)

	_, err := service.CreateProvider(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")

	mockRepo.AssertExpectations(t)
}

func TestLLMService_GetProvider(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	expectedProvider := &model.LLMProvider{
		ID:           providerID,
		Name:         "test-provider",
		ProviderType: "openai",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	mockRepo.On("GetProvider", ctx, providerID).Return(expectedProvider, nil)

	provider, err := service.GetProvider(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, expectedProvider, provider)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_ListProviders(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	isActive := true
	expectedProviders := []*model.LLMProvider{
		{
			ID:           uuid.New(),
			Name:         "provider-1",
			ProviderType: "openai",
			IsActive:     true,
		},
		{
			ID:           uuid.New(),
			Name:         "provider-2",
			ProviderType: "qianwen",
			IsActive:     true,
		},
	}

	mockRepo.On("ListProviders", ctx, &isActive).Return(expectedProviders, nil)

	providers, err := service.ListProviders(ctx, &isActive)
	require.NoError(t, err)
	assert.Equal(t, expectedProviders, providers)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_UpdateProvider(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	existingProvider := &model.LLMProvider{
		ID:           providerID,
		Name:         "old-name",
		ProviderType: "openai",
		Config: map[string]interface{}{
			"api_key": "old-key",
		},
		IsActive: true,
	}

	newName := "new-name"
	newConfig := map[string]interface{}{
		"api_key": "new-key",
	}
	req := &UpdateProviderRequest{
		Name:   &newName,
		Config: newConfig,
	}

	// 模拟获取现有提供商
	mockRepo.On("GetProvider", ctx, providerID).Return(existingProvider, nil)
	
	// 模拟检查新名称不存在
	mockRepo.On("CheckProviderExists", ctx, newName, &providerID).Return(false, nil)
	
	// 模拟更新成功
	mockRepo.On("UpdateProvider", ctx, mock.AnythingOfType("*model.LLMProvider")).Return(nil)

	provider, err := service.UpdateProvider(ctx, providerID, req)
	require.NoError(t, err)
	assert.Equal(t, newName, provider.Name)
	assert.Equal(t, newConfig, provider.Config)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_DeleteProvider(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	provider := &model.LLMProvider{
		ID:   providerID,
		Name: "test-provider",
	}

	mockRepo.On("GetProvider", ctx, providerID).Return(provider, nil)
	mockRepo.On("DeleteProvider", ctx, providerID).Return(nil)

	err := service.DeleteProvider(ctx, providerID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_CreateModel(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	provider := &model.LLMProvider{
		ID:   providerID,
		Name: "test-provider",
	}

	req := &CreateModelRequest{
		ProviderID:  providerID,
		ModelName:   "gpt-3.5-turbo",
		DisplayName: "GPT-3.5 Turbo",
		ModelType:   "chat",
	}

	// 模拟提供商存在
	mockRepo.On("GetProvider", ctx, providerID).Return(provider, nil)
	
	// 模拟模型名称不存在
	mockRepo.On("CheckModelExists", ctx, providerID, req.ModelName, (*uuid.UUID)(nil)).Return(false, nil)
	
	// 模拟创建成功
	mockRepo.On("CreateModel", ctx, mock.AnythingOfType("*model.LLMModel")).Return(nil)

	model, err := service.CreateModel(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, req.ModelName, model.ModelName)
	assert.Equal(t, req.DisplayName, model.DisplayName)
	assert.Equal(t, req.ModelType, model.ModelType)
	assert.True(t, model.IsActive)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_GetModel(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	modelID := uuid.New()
	expectedModel := &model.LLMModel{
		ID:          modelID,
		ModelName:   "gpt-3.5-turbo",
		DisplayName: "GPT-3.5 Turbo",
		ModelType:   "chat",
		IsActive:    true,
	}

	mockRepo.On("GetModel", ctx, modelID).Return(expectedModel, nil)

	model, err := service.GetModel(ctx, modelID)
	require.NoError(t, err)
	assert.Equal(t, expectedModel, model)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_ListModels(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	modelType := "chat"
	isActive := true

	expectedModels := []*model.LLMModel{
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			ModelName:   "gpt-3.5-turbo",
			DisplayName: "GPT-3.5 Turbo",
			ModelType:   "chat",
			IsActive:    true,
		},
		{
			ID:          uuid.New(),
			ProviderID:  providerID,
			ModelName:   "gpt-4",
			DisplayName: "GPT-4",
			ModelType:   "chat",
			IsActive:    true,
		},
	}

	mockRepo.On("ListModels", ctx, &providerID, &modelType, &isActive).Return(expectedModels, nil)

	models, err := service.ListModels(ctx, &providerID, &modelType, &isActive)
	require.NoError(t, err)
	assert.Equal(t, expectedModels, models)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_UpdateModel(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	modelID := uuid.New()
	existingModel := &model.LLMModel{
		ID:          modelID,
		ModelName:   "gpt-3.5-turbo",
		DisplayName: "Old Display Name",
		ModelType:   "chat",
		IsActive:    true,
	}

	newDisplayName := "New Display Name"
	newConfig := map[string]interface{}{
		"temperature": 0.7,
	}
	req := &UpdateModelRequest{
		DisplayName: &newDisplayName,
		Config:      newConfig,
	}

	mockRepo.On("GetModel", ctx, modelID).Return(existingModel, nil)
	mockRepo.On("UpdateModel", ctx, mock.AnythingOfType("*model.LLMModel")).Return(nil)

	model, err := service.UpdateModel(ctx, modelID, req)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, model.DisplayName)
	assert.Equal(t, newConfig, model.Config)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_DeleteModel(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	modelID := uuid.New()

	mockRepo.On("DeleteModel", ctx, modelID).Return(nil)

	err := service.DeleteModel(ctx, modelID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestLLMService_GetProviderStats(t *testing.T) {
	mockRepo := new(MockLLMRepository)
	manager := llm.NewManager(llm.NewDefaultProviderFactory())
	factory := llm.NewDefaultProviderFactory()
	service := NewLLMService(mockRepo, manager, factory)

	ctx := context.Background()
	providerID := uuid.New()
	expectedStats := map[string]interface{}{
		"total_models":  5,
		"active_models": 3,
		"model_type_stats": []map[string]interface{}{
			{"model_type": "chat", "count": 3},
			{"model_type": "embedding", "count": 2},
		},
	}

	mockRepo.On("GetProviderStats", ctx, providerID).Return(expectedStats, nil)

	stats, err := service.GetProviderStats(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, expectedStats, stats)

	mockRepo.AssertExpectations(t)
}