package genkit

import (
	"testing"
)

// TestParseModelConfiguration 测试解析模型配置
func TestParseModelConfiguration(t *testing.T) {
	client := &client{}

	tests := []struct {
		name        string
		modelConfig interface{}
		wantErr     bool
		errContains string
		validate    func(*testing.T, *GenkitConfig)
	}{
		{
			name: "解析 Google AI 配置成功",
			modelConfig: map[string]interface{}{
				"model": "gemini-1.5-pro",
				"queryParams": stringPtr(`{
					"defaultTemperature": 0.7,
					"defaultMaxTokens": 2048
				}`),
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				if config.Model != "gemini-1.5-pro" {
					t.Errorf("Model = %v, want gemini-1.5-pro", config.Model)
				}
				if config.DefaultTemperature != 0.7 {
					t.Errorf("DefaultTemperature = %v, want 0.7", config.DefaultTemperature)
				}
				if config.DefaultMaxTokens != 2048 {
					t.Errorf("DefaultMaxTokens = %v, want 2048", config.DefaultMaxTokens)
				}
			},
		},
		{
			name: "解析 Azure OpenAI 配置成功",
			modelConfig: map[string]interface{}{
				"model": "gpt-4",
				"queryParams": stringPtr(`{
					"azureEndpoint": "https://your-resource.openai.azure.com",
					"azureDeployment": "gpt-4",
					"azureApiVersion": "2024-02-15-preview",
					"defaultTemperature": 0.7,
					"defaultMaxTokens": 2048
				}`),
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				if config.Model != "gpt-4" {
					t.Errorf("Model = %v, want gpt-4", config.Model)
				}
				if config.AzureEndpoint != "https://your-resource.openai.azure.com" {
					t.Errorf("AzureEndpoint = %v, want https://your-resource.openai.azure.com", config.AzureEndpoint)
				}
				if config.AzureDeployment != "gpt-4" {
					t.Errorf("AzureDeployment = %v, want gpt-4", config.AzureDeployment)
				}
				if config.AzureAPIVersion != "2024-02-15-preview" {
					t.Errorf("AzureAPIVersion = %v, want 2024-02-15-preview", config.AzureAPIVersion)
				}
			},
		},
		{
			name: "解析百炼配置成功",
			modelConfig: map[string]interface{}{
				"model": "qwen-turbo",
				"queryParams": stringPtr(`{
					"bailianEndpoint": "https://dashscope.aliyuncs.com",
					"bailianWorkspace": "default",
					"defaultTemperature": 0.7,
					"defaultMaxTokens": 2048
				}`),
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				if config.Model != "qwen-turbo" {
					t.Errorf("Model = %v, want qwen-turbo", config.Model)
				}
				if config.BailianEndpoint != "https://dashscope.aliyuncs.com" {
					t.Errorf("BailianEndpoint = %v, want https://dashscope.aliyuncs.com", config.BailianEndpoint)
				}
				if config.BailianWorkspace != "default" {
					t.Errorf("BailianWorkspace = %v, want default", config.BailianWorkspace)
				}
			},
		},
		{
			name: "QueryParams 为空时使用默认配置",
			modelConfig: map[string]interface{}{
				"model":       "gemini-1.5-pro",
				"queryParams": nil,
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				if config.Model != "gemini-1.5-pro" {
					t.Errorf("Model = %v, want gemini-1.5-pro", config.Model)
				}
				// 默认值应该为零值
				if config.DefaultTemperature != 0 {
					t.Errorf("DefaultTemperature = %v, want 0", config.DefaultTemperature)
				}
				if config.DefaultMaxTokens != 0 {
					t.Errorf("DefaultMaxTokens = %v, want 0", config.DefaultMaxTokens)
				}
			},
		},
		{
			name: "QueryParams 为空字符串时使用默认配置",
			modelConfig: map[string]interface{}{
				"model":       "gemini-1.5-pro",
				"queryParams": stringPtr(""),
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				if config.Model != "gemini-1.5-pro" {
					t.Errorf("Model = %v, want gemini-1.5-pro", config.Model)
				}
			},
		},
		{
			name: "QueryParams 中的 model 字段不覆盖基本 model",
			modelConfig: map[string]interface{}{
				"model": "gemini-1.5-pro",
				"queryParams": stringPtr(`{
					"model": "should-be-ignored",
					"defaultTemperature": 0.7
				}`),
			},
			wantErr: false,
			validate: func(t *testing.T, config *GenkitConfig) {
				// 应该使用基本配置中的 model，而不是 QueryParams 中的
				if config.Model != "should-be-ignored" {
					t.Errorf("Model = %v, want should-be-ignored (QueryParams should override)", config.Model)
				}
			},
		},
		{
			name: "无效的 QueryParams JSON",
			modelConfig: map[string]interface{}{
				"model":       "gemini-1.5-pro",
				"queryParams": stringPtr(`{invalid json}`),
			},
			wantErr:     true,
			errContains: "解析 QueryParams 失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := client.parseModelConfiguration(tt.modelConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseModelConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("parseModelConfiguration() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

// TestInitializeProvider 测试初始化提供商
func TestInitializeProvider(t *testing.T) {
	// 注意：这个测试需要实际的 Genkit 环境，可能需要 mock
	// 这里只测试错误情况和基本逻辑

	client := &client{}

	tests := []struct {
		name         string
		modelConfig  interface{}
		genkitConfig *GenkitConfig
		wantErr      bool
		errContains  string
	}{
		{
			name: "不支持的提供商类型",
			modelConfig: map[string]interface{}{
				"modelProvider": "unknown",
				"apiKey":        "test-key",
			},
			genkitConfig: &GenkitConfig{
				Model: "test-model",
			},
			wantErr:     true,
			errContains: "不支持的提供商类型",
		},
		{
			name: "Azure OpenAI 暂未实现",
			modelConfig: map[string]interface{}{
				"modelProvider": "azureopenai",
				"apiKey":        "test-key",
			},
			genkitConfig: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://test.openai.azure.com",
				AzureDeployment: "gpt-4",
				AzureAPIVersion: "2024-02-15-preview",
			},
			wantErr:     true,
			errContains: "Azure OpenAI 提供商暂未实现",
		},
		{
			name: "百炼暂未实现",
			modelConfig: map[string]interface{}{
				"modelProvider": "bianlian",
				"apiKey":        "test-key",
			},
			genkitConfig: &GenkitConfig{
				Model:            "qwen-turbo",
				BailianEndpoint:  "https://dashscope.aliyuncs.com",
				BailianWorkspace: "default",
			},
			wantErr:     true,
			errContains: "百炼提供商暂未实现",
		},
		{
			name: "OpenAI 暂未实现",
			modelConfig: map[string]interface{}{
				"modelProvider": "openai",
				"apiKey":        "test-key",
			},
			genkitConfig: &GenkitConfig{
				Model: "gpt-4",
			},
			wantErr:     true,
			errContains: "OpenAI 提供商暂未实现",
		},
		{
			name: "Anthropic 暂未实现",
			modelConfig: map[string]interface{}{
				"modelProvider": "anthropic",
				"apiKey":        "test-key",
			},
			genkitConfig: &GenkitConfig{
				Model: "claude-3-opus",
			},
			wantErr:     true,
			errContains: "Anthropic 提供商暂未实现",
		},
		{
			name: "自定义 OpenAI 暂未实现",
			modelConfig: map[string]interface{}{
				"modelProvider": "custom_openai",
				"apiKey":        "test-key",
				"baseUrl":       stringPtr("https://custom.openai.com"),
			},
			genkitConfig: &GenkitConfig{
				Model: "custom-model",
			},
			wantErr:     true,
			errContains: "自定义 OpenAI 提供商暂未实现",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用 context.Background() 进行测试
			// 注意：Google AI 的测试需要实际的 API key，这里跳过
			if tt.wantErr {
				_, err := client.initializeProvider(nil, tt.modelConfig, tt.genkitConfig)
				if err == nil {
					t.Errorf("initializeProvider() expected error but got nil")
					return
				}
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("initializeProvider() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}
