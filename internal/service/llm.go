package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"your-project/internal/llm"
	"your-project/internal/model"
	"your-project/internal/repository"
)

// LLMService LLM服务接口
type LLMService interface {
	// 提供商管理
	CreateProvider(ctx context.Context, req *CreateProviderRequest) (*model.LLMProvider, error)
	GetProvider(ctx context.Context, id uuid.UUID) (*model.LLMProvider, error)
	GetProviderByName(ctx context.Context, name string) (*model.LLMProvider, error)
	ListProviders(ctx context.Context, isActive *bool) ([]*model.LLMProvider, error)
	UpdateProvider(ctx context.Context, id uuid.UUID, req *UpdateProviderRequest) (*model.LLMProvider, error)
	DeleteProvider(ctx context.Context, id uuid.UUID) error
	TestProviderConnection(ctx context.Context, id uuid.UUID) error
	
	// 模型管理
	CreateModel(ctx context.Context, req *CreateModelRequest) (*model.LLMModel, error)
	GetModel(ctx context.Context, id uuid.UUID) (*model.LLMModel, error)
	ListModels(ctx context.Context, providerID *uuid.UUID, modelType *string, isActive *bool) ([]*model.LLMModel, error)
	UpdateModel(ctx context.Context, id uuid.UUID, req *UpdateModelRequest) (*model.LLMModel, error)
	DeleteModel(ctx context.Context, id uuid.UUID) error
	SyncProviderModels(ctx context.Context, providerID uuid.UUID) error
	
	// 统计信息
	GetProviderStats(ctx context.Context, providerID uuid.UUID) (map[string]interface{}, error)
}

// llmService LLM服务实现
type llmService struct {
	repo    repository.LLMRepository
	manager *llm.Manager
	factory llm.ProviderFactory
}

// NewLLMService 创建LLM服务
func NewLLMService(repo repository.LLMRepository, manager *llm.Manager, factory llm.ProviderFactory) LLMService {
	return &llmService{
		repo:    repo,
		manager: manager,
		factory: factory,
	}
}

// CreateProviderRequest 创建提供商请求
type CreateProviderRequest struct {
	Name         string            `json:"name" validate:"required,min=1,max=100"`
	ProviderType string            `json:"provider_type" validate:"required,oneof=openai azure qianwen claude baichuan chatglm"`
	Config       map[string]interface{} `json:"config" validate:"required"`
	IsActive     *bool             `json:"is_active,omitempty"`
}

// UpdateProviderRequest 更新提供商请求
type UpdateProviderRequest struct {
	Name     *string               `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Config   map[string]interface{} `json:"config,omitempty"`
	IsActive *bool                 `json:"is_active,omitempty"`
}

// CreateModelRequest 创建模型请求
type CreateModelRequest struct {
	ProviderID  uuid.UUID         `json:"provider_id" validate:"required"`
	ModelName   string            `json:"model_name" validate:"required,min=1,max=100"`
	DisplayName string            `json:"display_name" validate:"required,min=1,max=200"`
	ModelType   string            `json:"model_type" validate:"required,oneof=chat completion embedding image audio"`
	Config      map[string]interface{} `json:"config,omitempty"`
	IsActive    *bool             `json:"is_active,omitempty"`
}

// UpdateModelRequest 更新模型请求
type UpdateModelRequest struct {
	DisplayName *string               `json:"display_name,omitempty" validate:"omitempty,min=1,max=200"`
	Config      map[string]interface{} `json:"config,omitempty"`
	IsActive    *bool                 `json:"is_active,omitempty"`
}

// CreateProvider 创建提供商
func (s *llmService) CreateProvider(ctx context.Context, req *CreateProviderRequest) (*model.LLMProvider, error) {
	// 检查名称是否已存在
	exists, err := s.repo.CheckProviderExists(ctx, req.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("检查提供商名称失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("提供商名称 '%s' 已存在", req.Name)
	}
	
	// 验证提供商类型
	if !model.IsValidProviderType(req.ProviderType) {
		return nil, fmt.Errorf("无效的提供商类型: %s", req.ProviderType)
	}
	
	// 创建提供商模型
	provider := &model.LLMProvider{
		Name:         req.Name,
		ProviderType: req.ProviderType,
		Config:       req.Config,
		IsActive:     true,
	}
	
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}
	
	// 保存到数据库
	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("创建提供商失败: %w", err)
	}
	
	// 如果提供商是激活状态，尝试添加到管理器
	if provider.IsActive {
		if err := s.addProviderToManager(provider); err != nil {
			// 记录警告，但不影响创建流程
			// TODO: 添加日志记录
		}
	}
	
	return provider, nil
}

// GetProvider 获取提供商
func (s *llmService) GetProvider(ctx context.Context, id uuid.UUID) (*model.LLMProvider, error) {
	return s.repo.GetProvider(ctx, id)
}

// GetProviderByName 根据名称获取提供商
func (s *llmService) GetProviderByName(ctx context.Context, name string) (*model.LLMProvider, error) {
	return s.repo.GetProviderByName(ctx, name)
}

// ListProviders 列出提供商
func (s *llmService) ListProviders(ctx context.Context, isActive *bool) ([]*model.LLMProvider, error) {
	return s.repo.ListProviders(ctx, isActive)
}

// UpdateProvider 更新提供商
func (s *llmService) UpdateProvider(ctx context.Context, id uuid.UUID, req *UpdateProviderRequest) (*model.LLMProvider, error) {
	// 获取现有提供商
	provider, err := s.repo.GetProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// 检查名称是否已存在（排除当前提供商）
	if req.Name != nil && *req.Name != provider.Name {
		exists, err := s.repo.CheckProviderExists(ctx, *req.Name, &id)
		if err != nil {
			return nil, fmt.Errorf("检查提供商名称失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("提供商名称 '%s' 已存在", *req.Name)
		}
		provider.Name = *req.Name
	}
	
	// 更新配置
	if req.Config != nil {
		provider.Config = req.Config
	}
	
	// 更新激活状态
	oldIsActive := provider.IsActive
	if req.IsActive != nil {
		provider.IsActive = *req.IsActive
	}
	
	// 保存到数据库
	if err := s.repo.UpdateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("更新提供商失败: %w", err)
	}
	
	// 更新管理器中的提供商
	if oldIsActive != provider.IsActive {
		if provider.IsActive {
			s.addProviderToManager(provider)
		} else {
			s.manager.RemoveProvider(provider.Name)
		}
	} else if provider.IsActive {
		// 重新添加以更新配置
		s.manager.RemoveProvider(provider.Name)
		s.addProviderToManager(provider)
	}
	
	return provider, nil
}

// DeleteProvider 删除提供商
func (s *llmService) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	// 获取提供商信息
	provider, err := s.repo.GetProvider(ctx, id)
	if err != nil {
		return err
	}
	
	// 从管理器中移除
	s.manager.RemoveProvider(provider.Name)
	
	// 从数据库中软删除
	return s.repo.DeleteProvider(ctx, id)
}

// TestProviderConnection 测试提供商连接
func (s *llmService) TestProviderConnection(ctx context.Context, id uuid.UUID) error {
	provider, err := s.repo.GetProvider(ctx, id)
	if err != nil {
		return err
	}
	
	// 创建临时提供商实例进行测试
	config, err := s.convertToLLMConfig(provider)
	if err != nil {
		return fmt.Errorf("转换配置失败: %w", err)
	}
	
	llmProvider, err := s.factory.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("创建提供商实例失败: %w", err)
	}
	
	// 执行健康检查
	return llmProvider.HealthCheck(ctx)
}

// CreateModel 创建模型
func (s *llmService) CreateModel(ctx context.Context, req *CreateModelRequest) (*model.LLMModel, error) {
	// 检查提供商是否存在
	_, err := s.repo.GetProvider(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("提供商不存在: %w", err)
	}
	
	// 检查模型名称是否已存在
	exists, err := s.repo.CheckModelExists(ctx, req.ProviderID, req.ModelName, nil)
	if err != nil {
		return nil, fmt.Errorf("检查模型名称失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("模型名称 '%s' 在该提供商下已存在", req.ModelName)
	}
	
	// 验证模型类型
	if !model.IsValidModelType(req.ModelType) {
		return nil, fmt.Errorf("无效的模型类型: %s", req.ModelType)
	}
	
	// 创建模型
	llmModel := &model.LLMModel{
		ProviderID:  req.ProviderID,
		ModelName:   req.ModelName,
		DisplayName: req.DisplayName,
		ModelType:   req.ModelType,
		Config:      req.Config,
		IsActive:    true,
	}
	
	if req.IsActive != nil {
		llmModel.IsActive = *req.IsActive
	}
	
	if req.Config == nil {
		llmModel.Config = make(map[string]interface{})
	}
	
	// 保存到数据库
	if err := s.repo.CreateModel(ctx, llmModel); err != nil {
		return nil, fmt.Errorf("创建模型失败: %w", err)
	}
	
	return llmModel, nil
}

// GetModel 获取模型
func (s *llmService) GetModel(ctx context.Context, id uuid.UUID) (*model.LLMModel, error) {
	return s.repo.GetModel(ctx, id)
}

// ListModels 列出模型
func (s *llmService) ListModels(ctx context.Context, providerID *uuid.UUID, modelType *string, isActive *bool) ([]*model.LLMModel, error) {
	return s.repo.ListModels(ctx, providerID, modelType, isActive)
}

// UpdateModel 更新模型
func (s *llmService) UpdateModel(ctx context.Context, id uuid.UUID, req *UpdateModelRequest) (*model.LLMModel, error) {
	// 获取现有模型
	llmModel, err := s.repo.GetModel(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// 更新字段
	if req.DisplayName != nil {
		llmModel.DisplayName = *req.DisplayName
	}
	
	if req.Config != nil {
		llmModel.Config = req.Config
	}
	
	if req.IsActive != nil {
		llmModel.IsActive = *req.IsActive
	}
	
	// 保存到数据库
	if err := s.repo.UpdateModel(ctx, llmModel); err != nil {
		return nil, fmt.Errorf("更新模型失败: %w", err)
	}
	
	return llmModel, nil
}

// DeleteModel 删除模型
func (s *llmService) DeleteModel(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteModel(ctx, id)
}

// SyncProviderModels 同步提供商模型
func (s *llmService) SyncProviderModels(ctx context.Context, providerID uuid.UUID) error {
	// 获取提供商
	provider, err := s.repo.GetProvider(ctx, providerID)
	if err != nil {
		return err
	}
	
	// 创建提供商实例
	config, err := s.convertToLLMConfig(provider)
	if err != nil {
		return fmt.Errorf("转换配置失败: %w", err)
	}
	
	llmProvider, err := s.factory.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("创建提供商实例失败: %w", err)
	}
	
	// 获取远程模型列表
	remoteModels, err := llmProvider.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("获取远程模型列表失败: %w", err)
	}
	
	// 转换为数据库模型
	var dbModels []*model.LLMModel
	for _, remoteModel := range remoteModels {
		dbModel := &model.LLMModel{
			ProviderID:  providerID,
			ModelName:   remoteModel.ID,
			DisplayName: remoteModel.DisplayName,
			ModelType:   string(remoteModel.ModelType),
			Config: map[string]interface{}{
				"capabilities": remoteModel.Capabilities,
				"limits":       remoteModel.Limits,
				"pricing":      remoteModel.Pricing,
				"description":  remoteModel.Description,
			},
			IsActive: true,
		}
		dbModels = append(dbModels, dbModel)
	}
	
	// 批量创建或更新模型
	for _, dbModel := range dbModels {
		// 检查模型是否已存在
		existingModel, err := s.repo.GetModelByName(ctx, providerID, dbModel.ModelName)
		if err != nil && err != model.ErrModelNotFound {
			return fmt.Errorf("检查模型存在性失败: %w", err)
		}
		
		if existingModel != nil {
			// 更新现有模型
			existingModel.DisplayName = dbModel.DisplayName
			existingModel.Config = dbModel.Config
			if err := s.repo.UpdateModel(ctx, existingModel); err != nil {
				return fmt.Errorf("更新模型 %s 失败: %w", dbModel.ModelName, err)
			}
		} else {
			// 创建新模型
			if err := s.repo.CreateModel(ctx, dbModel); err != nil {
				return fmt.Errorf("创建模型 %s 失败: %w", dbModel.ModelName, err)
			}
		}
	}
	
	return nil
}

// GetProviderStats 获取提供商统计信息
func (s *llmService) GetProviderStats(ctx context.Context, providerID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetProviderStats(ctx, providerID)
}

// addProviderToManager 将提供商添加到管理器
func (s *llmService) addProviderToManager(provider *model.LLMProvider) error {
	config, err := s.convertToLLMConfig(provider)
	if err != nil {
		return fmt.Errorf("转换配置失败: %w", err)
	}
	
	return s.manager.AddProvider(provider.Name, config)
}

// convertToLLMConfig 将数据库模型转换为LLM配置
func (s *llmService) convertToLLMConfig(provider *model.LLMProvider) (llm.ProviderConfig, error) {
	providerType := llm.ProviderType(provider.ProviderType)
	
	switch providerType {
	case llm.ProviderOpenAI:
		config := &llm.OpenAIConfig{
			BaseProviderConfig: llm.BaseProviderConfig{
				Type:   providerType,
				Name:   provider.Name,
				APIKey: getStringFromConfig(provider.Config, "api_key"),
			},
		}
		
		if baseURL, ok := provider.Config["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if org, ok := provider.Config["organization"].(string); ok {
			config.Organization = org
		}
		
		return config, nil
		
	case llm.ProviderQianwen:
		config := &llm.QianwenConfig{
			BaseProviderConfig: llm.BaseProviderConfig{
				Type:   providerType,
				Name:   provider.Name,
				APIKey: getStringFromConfig(provider.Config, "api_key"),
			},
		}
		
		if baseURL, ok := provider.Config["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if workspaceID, ok := provider.Config["workspace_id"].(string); ok {
			config.WorkspaceID = workspaceID
		}
		
		return config, nil
		
	case llm.ProviderClaude:
		config := &llm.ClaudeConfig{
			BaseProviderConfig: llm.BaseProviderConfig{
				Type:   providerType,
				Name:   provider.Name,
				APIKey: getStringFromConfig(provider.Config, "api_key"),
			},
		}
		
		if baseURL, ok := provider.Config["base_url"].(string); ok {
			config.BaseURL = baseURL
		}
		if version, ok := provider.Config["version"].(string); ok {
			config.Version = version
		}
		
		return config, nil
		
	case llm.ProviderAzure:
		config := &llm.AzureOpenAIConfig{
			BaseProviderConfig: llm.BaseProviderConfig{
				Type:   providerType,
				Name:   provider.Name,
				APIKey: getStringFromConfig(provider.Config, "api_key"),
			},
			ResourceName: getStringFromConfig(provider.Config, "resource_name"),
			Deployment:   getStringFromConfig(provider.Config, "deployment"),
		}
		
		if apiVersion, ok := provider.Config["api_version"].(string); ok {
			config.APIVersion = apiVersion
		}
		
		return config, nil
		
	default:
		return nil, fmt.Errorf("不支持的提供商类型: %s", providerType)
	}
}

// getStringFromConfig 从配置中获取字符串值
func getStringFromConfig(config map[string]interface{}, key string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return ""
}