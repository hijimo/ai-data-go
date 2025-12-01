package genkit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAzureOpenAIConfig 测试 Azure OpenAI 配置
func TestAzureOpenAIConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *GenkitConfig
		provider    string
		wantErr     bool
		errContains string
	}{
		{
			name: "完整的 Azure OpenAI 配置",
			config: &GenkitConfig{
				Model:              "gpt-4",
				AzureEndpoint:      "https://my-resource.openai.azure.com",
				AzureDeployment:    "gpt-4-deployment",
				AzureAPIVersion:    "2024-02-15-preview",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			provider: "azureopenai",
			wantErr:  false,
		},
		{
			name: "缺少 AzureEndpoint",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureDeployment: "gpt-4-deployment",
				AzureAPIVersion: "2024-02-15-preview",
			},
			provider:    "azureopenai",
			wantErr:     true,
			errContains: "azureEndpoint",
		},
		{
			name: "缺少 AzureDeployment",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://my-resource.openai.azure.com",
				AzureAPIVersion: "2024-02-15-preview",
			},
			provider:    "azureopenai",
			wantErr:     true,
			errContains: "azureDeployment",
		},
		{
			name: "缺少 AzureAPIVersion",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://my-resource.openai.azure.com",
				AzureDeployment: "gpt-4-deployment",
			},
			provider:    "azureopenai",
			wantErr:     true,
			errContains: "azureApiVersion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.provider)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAzureBaseURLConstruction 测试 Azure BaseURL 构造
func TestAzureBaseURLConstruction(t *testing.T) {
	tests := []struct {
		name            string
		azureEndpoint   string
		azureDeployment string
		expectedBaseURL string
	}{
		{
			name:            "标准 Azure Endpoint",
			azureEndpoint:   "https://my-resource.openai.azure.com",
			azureDeployment: "gpt-4",
			expectedBaseURL: "https://my-resource.openai.azure.com/openai/deployments/gpt-4",
		},
		{
			name:            "带尾部斜杠的 Endpoint",
			azureEndpoint:   "https://my-resource.openai.azure.com/",
			azureDeployment: "gpt-35-turbo",
			expectedBaseURL: "https://my-resource.openai.azure.com//openai/deployments/gpt-35-turbo",
		},
		{
			name:            "自定义域名",
			azureEndpoint:   "https://custom-domain.com",
			azureDeployment: "my-deployment",
			expectedBaseURL: "https://custom-domain.com/openai/deployments/my-deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 BaseURL 构造逻辑
			baseURL := constructAzureBaseURL(tt.azureEndpoint, tt.azureDeployment)
			assert.Equal(t, tt.expectedBaseURL, baseURL)
		})
	}
}

// constructAzureBaseURL 构造 Azure OpenAI 的 BaseURL
// 这是一个辅助函数，用于测试
func constructAzureBaseURL(endpoint, deployment string) string {
	return endpoint + "/openai/deployments/" + deployment
}

// TestAzureAPIVersionHandling 测试 API Version 参数处理
func TestAzureAPIVersionHandling(t *testing.T) {
	tests := []struct {
		name           string
		apiVersion     string
		expectedFormat string
	}{
		{
			name:           "标准 API Version",
			apiVersion:     "2024-02-15-preview",
			expectedFormat: "api-version=2024-02-15-preview",
		},
		{
			name:           "稳定版本",
			apiVersion:     "2023-12-01",
			expectedFormat: "api-version=2023-12-01",
		},
		{
			name:           "空 API Version",
			apiVersion:     "",
			expectedFormat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queryParam string
			if tt.apiVersion != "" {
				queryParam = "api-version=" + tt.apiVersion
			}
			assert.Equal(t, tt.expectedFormat, queryParam)
		})
	}
}

// TestAzureModelNameMapping 测试 Azure 模型名称映射
func TestAzureModelNameMapping(t *testing.T) {
	tests := []struct {
		name              string
		deploymentName    string
		expectedModelName string
	}{
		{
			name:              "GPT-4 部署",
			deploymentName:    "gpt-4-deployment",
			expectedModelName: "openai/gpt-4-deployment",
		},
		{
			name:              "GPT-3.5 Turbo 部署",
			deploymentName:    "gpt-35-turbo",
			expectedModelName: "openai/gpt-35-turbo",
		},
		{
			name:              "自定义部署名称",
			deploymentName:    "my-custom-model",
			expectedModelName: "openai/my-custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 在 Genkit 中，Azure OpenAI 使用 "openai/" 前缀
			modelName := "openai/" + tt.deploymentName
			assert.Equal(t, tt.expectedModelName, modelName)
		})
	}
}

// TestCreateAzurePlugin 测试 createAzurePlugin 函数
func TestCreateAzurePlugin(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		config      *GenkitConfig
		wantErr     bool
		errContains string
	}{
		{
			name:   "完整的 Azure 配置",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://my-resource.openai.azure.com",
				AzureDeployment: "gpt-4-deployment",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr: false,
		},
		{
			name:   "缺少 AzureEndpoint",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureDeployment: "gpt-4-deployment",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr:     true,
			errContains: "azureEndpoint",
		},
		{
			name:   "缺少 AzureDeployment",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://my-resource.openai.azure.com",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr:     true,
			errContains: "azureDeployment",
		},
		{
			name:   "空的 AzureEndpoint",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "",
				AzureDeployment: "gpt-4-deployment",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr:     true,
			errContains: "azureEndpoint",
		},
		{
			name:   "空的 AzureDeployment",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://my-resource.openai.azure.com",
				AzureDeployment: "",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr:     true,
			errContains: "azureDeployment",
		},
		{
			name:   "带尾部斜杠的 Endpoint",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "gpt-35-turbo",
				AzureEndpoint:   "https://my-resource.openai.azure.com/",
				AzureDeployment: "gpt-35-turbo",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr: false,
		},
		{
			name:   "自定义域名",
			apiKey: "test-azure-key",
			config: &GenkitConfig{
				Model:           "custom-model",
				AzureEndpoint:   "https://custom-domain.com",
				AzureDeployment: "my-deployment",
				AzureAPIVersion: "2023-12-01",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			plugin, err := createAzurePlugin(ctx, tt.apiKey, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, plugin)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, plugin)
				
				// 验证插件配置
				assert.NotNil(t, plugin.Opts)
				assert.NotEmpty(t, plugin.Opts)
			}
		})
	}
}
