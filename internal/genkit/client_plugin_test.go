package genkit

import (
	"context"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeProvider_GoogleGenAI 测试 Google AI 插件初始化
func TestInitializeProvider_GoogleGenAI(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "googlegenai",
		"apiKey":        "test-google-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model:              "gemini-1.5-pro",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_OpenAI 测试 OpenAI 插件初始化
func TestInitializeProvider_OpenAI(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "openai",
		"apiKey":        "test-openai-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model:              "gpt-4",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_AzureOpenAI 测试 Azure OpenAI 插件初始化
func TestInitializeProvider_AzureOpenAI(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "azureopenai",
		"apiKey":        "test-azure-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置（包含 Azure 特定字段）
	genkitConfig := &GenkitConfig{
		Model:              "gpt-4",
		AzureEndpoint:      "https://test-resource.openai.azure.com",
		AzureDeployment:    "gpt-4-deployment",
		AzureAPIVersion:    "2024-02-15-preview",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_AzureOpenAI_MissingConfig 测试 Azure OpenAI 缺少配置
func TestInitializeProvider_AzureOpenAI_MissingConfig(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	tests := []struct {
		name         string
		genkitConfig *GenkitConfig
		wantErr      string
	}{
		{
			name: "缺少 AzureEndpoint",
			genkitConfig: &GenkitConfig{
				Model:           "gpt-4",
				AzureDeployment: "gpt-4-deployment",
			},
			wantErr: "Azure OpenAI 配置缺少必需字段: azureEndpoint 或 azureDeployment",
		},
		{
			name: "缺少 AzureDeployment",
			genkitConfig: &GenkitConfig{
				Model:         "gpt-4",
				AzureEndpoint: "https://test-resource.openai.azure.com",
			},
			wantErr: "Azure OpenAI 配置缺少必需字段: azureEndpoint 或 azureDeployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelConfig := map[string]interface{}{
				"modelProvider": "azureopenai",
				"apiKey":        "test-azure-api-key",
			}

			g, err := client.initializeProvider(ctx, modelConfig, tt.genkitConfig)

			assert.Error(t, err)
			assert.Nil(t, g)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestInitializeProvider_Bianlian 测试百炼插件初始化
func TestInitializeProvider_Bianlian(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "bianlian",
		"apiKey":        "test-dashscope-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置（使用默认百炼端点）
	genkitConfig := &GenkitConfig{
		Model:              "qwen-plus",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_Bianlian_CustomEndpoint 测试百炼自定义端点
func TestInitializeProvider_Bianlian_CustomEndpoint(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "bianlian",
		"apiKey":        "test-dashscope-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置（使用自定义百炼端点）
	genkitConfig := &GenkitConfig{
		Model:            "qwen-plus",
		BailianEndpoint:  "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		BailianWorkspace: "default",
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_Anthropic 测试 Anthropic 插件初始化
func TestInitializeProvider_Anthropic(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "anthropic",
		"apiKey":        "test-anthropic-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model:              "claude-3-opus-20240229",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_CustomOpenAI 测试自定义 OpenAI 兼容服务
func TestInitializeProvider_CustomOpenAI(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	customBaseURL := "https://custom-openai-service.com/v1"

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "custom_openai",
		"apiKey":        "test-custom-api-key",
		"baseUrl":       &customBaseURL,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model:              "custom-model",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}

// TestInitializeProvider_CustomOpenAI_MissingBaseURL 测试自定义 OpenAI 缺少 BaseURL
func TestInitializeProvider_CustomOpenAI_MissingBaseURL(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置（缺少 baseUrl）
	modelConfig := map[string]interface{}{
		"modelProvider": "custom_openai",
		"apiKey":        "test-custom-api-key",
		"baseUrl":       nil,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model: "custom-model",
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Contains(t, err.Error(), "自定义 OpenAI 提供商必须指定 baseUrl")
}

// TestInitializeProvider_UnsupportedProvider 测试不支持的提供商
func TestInitializeProvider_UnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	// 模拟模型配置（不支持的提供商）
	modelConfig := map[string]interface{}{
		"modelProvider": "unsupported_provider",
		"apiKey":        "test-api-key",
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model: "test-model",
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Contains(t, err.Error(), "不支持的提供商类型: unsupported_provider")
}

// TestInitializeProvider_OpenAI_WithCustomBaseURL 测试 OpenAI 使用自定义 BaseURL
func TestInitializeProvider_OpenAI_WithCustomBaseURL(t *testing.T) {
	ctx := context.Background()
	client := &client{
		instances: make(map[string]*genkit.Genkit),
	}

	customBaseURL := "https://custom-openai-proxy.com/v1"

	// 模拟模型配置
	modelConfig := map[string]interface{}{
		"modelProvider": "openai",
		"apiKey":        "test-openai-api-key",
		"baseUrl":       &customBaseURL,
	}

	// Genkit 配置
	genkitConfig := &GenkitConfig{
		Model:              "gpt-4",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 初始化提供商
	g, err := client.initializeProvider(ctx, modelConfig, genkitConfig)

	// 验证
	require.NoError(t, err)
	assert.NotNil(t, g)
}


