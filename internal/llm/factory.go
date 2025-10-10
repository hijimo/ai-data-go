package llm

import (
	"fmt"
	"sync"
)

// DefaultProviderFactory 默认提供商工厂
type DefaultProviderFactory struct {
	mu        sync.RWMutex
	providers map[ProviderType]func(ProviderConfig) (LLMProvider, error)
}

// NewDefaultProviderFactory 创建默认提供商工厂
func NewDefaultProviderFactory() *DefaultProviderFactory {
	factory := &DefaultProviderFactory{
		providers: make(map[ProviderType]func(ProviderConfig) (LLMProvider, error)),
	}

	// 注册内置提供商
	factory.RegisterProvider(ProviderOpenAI, func(config ProviderConfig) (LLMProvider, error) {
		openaiConfig, ok := config.(*OpenAIConfig)
		if !ok {
			// 尝试从基础配置转换
			baseConfig, ok := config.(*BaseProviderConfig)
			if !ok {
				return nil, fmt.Errorf("无效的OpenAI配置类型")
			}
			openaiConfig = &OpenAIConfig{
				BaseProviderConfig: *baseConfig,
			}
		}
		return NewOpenAIProvider(openaiConfig), nil
	})

	factory.RegisterProvider(ProviderQianwen, func(config ProviderConfig) (LLMProvider, error) {
		qianwenConfig, ok := config.(*QianwenConfig)
		if !ok {
			// 尝试从基础配置转换
			baseConfig, ok := config.(*BaseProviderConfig)
			if !ok {
				return nil, fmt.Errorf("无效的千问配置类型")
			}
			qianwenConfig = &QianwenConfig{
				BaseProviderConfig: *baseConfig,
			}
		}
		return NewQianwenProvider(qianwenConfig), nil
	})

	factory.RegisterProvider(ProviderClaude, func(config ProviderConfig) (LLMProvider, error) {
		claudeConfig, ok := config.(*ClaudeConfig)
		if !ok {
			// 尝试从基础配置转换
			baseConfig, ok := config.(*BaseProviderConfig)
			if !ok {
				return nil, fmt.Errorf("无效的Claude配置类型")
			}
			claudeConfig = &ClaudeConfig{
				BaseProviderConfig: *baseConfig,
			}
		}
		return NewClaudeProvider(claudeConfig), nil
	})

	// 注册Azure OpenAI提供商
	factory.RegisterProvider(ProviderAzure, func(config ProviderConfig) (LLMProvider, error) {
		azureConfig, ok := config.(*AzureOpenAIConfig)
		if !ok {
			return nil, fmt.Errorf("无效的Azure OpenAI配置类型")
		}
		return NewAzureOpenAIProvider(azureConfig), nil
	})

	return factory
}

// RegisterProvider 注册提供商
func (f *DefaultProviderFactory) RegisterProvider(providerType ProviderType, creator func(ProviderConfig) (LLMProvider, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[providerType] = creator
}

// CreateProvider 创建提供商实例
func (f *DefaultProviderFactory) CreateProvider(config ProviderConfig) (LLMProvider, error) {
	f.mu.RLock()
	creator, exists := f.providers[config.GetProviderType()]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("不支持的提供商类型: %s", config.GetProviderType())
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return creator(config)
}

// SupportedTypes 获取支持的提供商类型
func (f *DefaultProviderFactory) SupportedTypes() []ProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]ProviderType, 0, len(f.providers))
	for providerType := range f.providers {
		types = append(types, providerType)
	}
	return types
}

// AzureOpenAIConfig Azure OpenAI配置
type AzureOpenAIConfig struct {
	BaseProviderConfig
	ResourceName string `json:"resource_name" validate:"required"`
	Deployment   string `json:"deployment" validate:"required"`
	APIVersion   string `json:"api_version"`
}

// AzureOpenAIProvider Azure OpenAI提供商实现
type AzureOpenAIProvider struct {
	*OpenAIProvider
	config *AzureOpenAIConfig
}

// NewAzureOpenAIProvider 创建Azure OpenAI提供商
func NewAzureOpenAIProvider(config *AzureOpenAIConfig) *AzureOpenAIProvider {
	// 构建Azure OpenAI的URL
	if config.BaseURL == "" {
		config.BaseURL = fmt.Sprintf("https://%s.openai.azure.com", config.ResourceName)
	}
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-15-preview"
	}

	// 创建OpenAI配置
	openaiConfig := &OpenAIConfig{
		BaseProviderConfig: config.BaseProviderConfig,
	}

	provider := &AzureOpenAIProvider{
		OpenAIProvider: NewOpenAIProvider(openaiConfig),
		config:         config,
	}

	return provider
}

// GetProviderType 获取提供商类型
func (p *AzureOpenAIProvider) GetProviderType() ProviderType {
	return ProviderAzure
}

// GetProviderName 获取提供商名称
func (p *AzureOpenAIProvider) GetProviderName() string {
	if p.config.Name != "" {
		return p.config.Name
	}
	return "Azure OpenAI"
}

// ConfigFromMap 从map创建配置
func ConfigFromMap(providerType ProviderType, configMap map[string]interface{}) (ProviderConfig, error) {
	switch providerType {
	case ProviderOpenAI:
		config := &OpenAIConfig{}
		if err := mapToStruct(configMap, config); err != nil {
			return nil, err
		}
		config.Type = providerType
		return config, nil

	case ProviderQianwen:
		config := &QianwenConfig{}
		if err := mapToStruct(configMap, config); err != nil {
			return nil, err
		}
		config.Type = providerType
		return config, nil

	case ProviderClaude:
		config := &ClaudeConfig{}
		if err := mapToStruct(configMap, config); err != nil {
			return nil, err
		}
		config.Type = providerType
		return config, nil

	case ProviderAzure:
		config := &AzureOpenAIConfig{}
		if err := mapToStruct(configMap, config); err != nil {
			return nil, err
		}
		config.Type = providerType
		return config, nil

	default:
		return nil, fmt.Errorf("不支持的提供商类型: %s", providerType)
	}
}

// mapToStruct 将map转换为结构体
func mapToStruct(m map[string]interface{}, result interface{}) error {
	// 这里可以使用更复杂的映射逻辑，比如使用反射或者第三方库
	// 为了简化，这里只是一个基础实现
	
	// 可以使用 mapstructure 库来实现更完善的映射
	// 这里先返回nil，实际实现中需要完善这个函数
	return nil
}

// GetDefaultConfig 获取提供商的默认配置
func GetDefaultConfig(providerType ProviderType) ProviderConfig {
	switch providerType {
	case ProviderOpenAI:
		return &OpenAIConfig{
			BaseProviderConfig: BaseProviderConfig{
				Type:    ProviderOpenAI,
				BaseURL: "https://api.openai.com/v1",
			},
		}
	case ProviderQianwen:
		return &QianwenConfig{
			BaseProviderConfig: BaseProviderConfig{
				Type:    ProviderQianwen,
				BaseURL: "https://dashscope.aliyuncs.com/api/v1",
			},
		}
	case ProviderClaude:
		return &ClaudeConfig{
			BaseProviderConfig: BaseProviderConfig{
				Type:    ProviderClaude,
				BaseURL: "https://api.anthropic.com",
			},
			Version: "2023-06-01",
		}
	case ProviderAzure:
		return &AzureOpenAIConfig{
			BaseProviderConfig: BaseProviderConfig{
				Type: ProviderAzure,
			},
			APIVersion: "2024-02-15-preview",
		}
	default:
		return nil
	}
}