package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultProviderFactory(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	// 测试支持的提供商类型
	supportedTypes := factory.SupportedTypes()
	assert.Contains(t, supportedTypes, ProviderOpenAI)
	assert.Contains(t, supportedTypes, ProviderQianwen)
	assert.Contains(t, supportedTypes, ProviderClaude)
	assert.Contains(t, supportedTypes, ProviderAzure)
}

func TestCreateOpenAIProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	config := &OpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderOpenAI,
			Name:    "test-openai",
			APIKey:  "test-key",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 30 * time.Second,
		},
		Organization: "test-org",
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderOpenAI, provider.GetProviderType())
	assert.Equal(t, "test-openai", provider.GetProviderName())
}

func TestCreateQianwenProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	config := &QianwenConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderQianwen,
			Name:    "test-qianwen",
			APIKey:  "test-key",
			BaseURL: "https://dashscope.aliyuncs.com/api/v1",
			Timeout: 30 * time.Second,
		},
		WorkspaceID: "test-workspace",
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderQianwen, provider.GetProviderType())
	assert.Equal(t, "test-qianwen", provider.GetProviderName())
}

func TestCreateClaudeProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	config := &ClaudeConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderClaude,
			Name:    "test-claude",
			APIKey:  "test-key",
			BaseURL: "https://api.anthropic.com",
			Timeout: 30 * time.Second,
		},
		Version: "2023-06-01",
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderClaude, provider.GetProviderType())
	assert.Equal(t, "test-claude", provider.GetProviderName())
}

func TestCreateAzureOpenAIProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	config := &AzureOpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderAzure,
			Name:    "test-azure",
			APIKey:  "test-key",
			Timeout: 30 * time.Second,
		},
		ResourceName: "test-resource",
		Deployment:   "test-deployment",
		APIVersion:   "2024-02-15-preview",
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderAzure, provider.GetProviderType())
	assert.Equal(t, "test-azure", provider.GetProviderName())
}

func TestCreateProviderWithInvalidConfig(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	// 测试无效的配置类型
	config := &BaseProviderConfig{
		Type:   "invalid-type",
		APIKey: "test-key",
	}
	
	_, err := factory.CreateProvider(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的提供商类型")
}

func TestCreateProviderWithMissingAPIKey(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	config := &OpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type: ProviderOpenAI,
			Name: "test-openai",
			// APIKey 缺失
		},
	}
	
	_, err := factory.CreateProvider(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "配置验证失败")
}

func TestRegisterCustomProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	
	// 注册自定义提供商
	customType := ProviderType("custom")
	factory.RegisterProvider(customType, func(config ProviderConfig) (LLMProvider, error) {
		return NewMockLLMProvider(customType, "Custom Provider"), nil
	})
	
	// 检查是否包含自定义类型
	supportedTypes := factory.SupportedTypes()
	assert.Contains(t, supportedTypes, customType)
	
	// 创建自定义提供商
	config := &BaseProviderConfig{
		Type:   customType,
		APIKey: "test-key",
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, customType, provider.GetProviderType())
}

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		wantNil      bool
	}{
		{"OpenAI", ProviderOpenAI, false},
		{"Qianwen", ProviderQianwen, false},
		{"Claude", ProviderClaude, false},
		{"Azure", ProviderAzure, false},
		{"Invalid", ProviderType("invalid"), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetDefaultConfig(tt.providerType)
			if tt.wantNil {
				assert.Nil(t, config)
			} else {
				assert.NotNil(t, config)
				assert.Equal(t, tt.providerType, config.GetProviderType())
			}
		})
	}
}

func TestAzureOpenAIConfig(t *testing.T) {
	config := &AzureOpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:   ProviderAzure,
			APIKey: "test-key",
		},
		ResourceName: "test-resource",
		Deployment:   "test-deployment",
		APIVersion:   "2024-02-15-preview",
	}
	
	provider := NewAzureOpenAIProvider(config)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderAzure, provider.GetProviderType())
	assert.Equal(t, "Azure OpenAI", provider.GetProviderName())
	
	// 测试自定义名称
	config.Name = "Custom Azure"
	provider = NewAzureOpenAIProvider(config)
	assert.Equal(t, "Custom Azure", provider.GetProviderName())
}

func TestAzureOpenAIConfigDefaults(t *testing.T) {
	config := &AzureOpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:   ProviderAzure,
			APIKey: "test-key",
		},
		ResourceName: "test-resource",
		Deployment:   "test-deployment",
		// APIVersion 未设置
	}
	
	provider := NewAzureOpenAIProvider(config)
	assert.NotNil(t, provider)
	
	// 检查默认值是否设置
	assert.Equal(t, "2024-02-15-preview", config.APIVersion)
	assert.Contains(t, config.BaseURL, "test-resource.openai.azure.com")
}

func TestConfigFromMap(t *testing.T) {
	// 这个测试需要实际实现 mapToStruct 函数后才能正常工作
	t.Skip("需要实现 mapToStruct 函数")
	
	configMap := map[string]interface{}{
		"api_key":  "test-key",
		"base_url": "https://api.openai.com/v1",
		"timeout":  30,
	}
	
	config, err := ConfigFromMap(ProviderOpenAI, configMap)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, ProviderOpenAI, config.GetProviderType())
}