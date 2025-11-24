package genkit

import (
	"testing"
)

// TestParseGenkitConfig 测试解析 GenkitConfig
func TestParseGenkitConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *GenkitConfig)
	}{
		{
			name: "解析 Google AI 配置成功",
			configJSON: `{
				"model": "gemini-1.5-pro",
				"defaultTemperature": 0.7,
				"defaultMaxTokens": 2048
			}`,
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
			configJSON: `{
				"model": "gpt-4",
				"azureEndpoint": "https://your-resource.openai.azure.com",
				"azureDeployment": "gpt-4",
				"azureApiVersion": "2024-02-15-preview",
				"defaultTemperature": 0.7,
				"defaultMaxTokens": 2048
			}`,
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
			configJSON: `{
				"model": "qwen-turbo",
				"bailianEndpoint": "https://dashscope.aliyuncs.com",
				"bailianWorkspace": "default",
				"defaultTemperature": 0.7,
				"defaultMaxTokens": 2048
			}`,
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
			name:        "空配置JSON",
			configJSON:  "",
			wantErr:     true,
			errContains: "配置JSON不能为空",
		},
		{
			name:        "无效的JSON格式",
			configJSON:  `{invalid json}`,
			wantErr:     true,
			errContains: "解析配置JSON失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseGenkitConfig(tt.configJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGenkitConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseGenkitConfig() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

// TestGenkitConfig_Validate 测试配置验证
func TestGenkitConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		config       *GenkitConfig
		providerType string
		wantErr      bool
		errContains  string
	}{
		{
			name: "Google AI 配置验证成功",
			config: &GenkitConfig{
				Model:              "gemini-1.5-pro",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			providerType: "googlegenai",
			wantErr:      false,
		},
		{
			name: "Azure OpenAI 配置验证成功",
			config: &GenkitConfig{
				Model:              "gpt-4",
				AzureEndpoint:      "https://your-resource.openai.azure.com",
				AzureDeployment:    "gpt-4",
				AzureAPIVersion:    "2024-02-15-preview",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			providerType: "azureopenai",
			wantErr:      false,
		},
		{
			name: "百炼配置验证成功",
			config: &GenkitConfig{
				Model:              "qwen-turbo",
				BailianEndpoint:    "https://dashscope.aliyuncs.com",
				BailianWorkspace:   "default",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			providerType: "bianlian",
			wantErr:      false,
		},
		{
			name: "模型名称为空",
			config: &GenkitConfig{
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			providerType: "googlegenai",
			wantErr:      true,
			errContains:  "模型名称不能为空",
		},
		{
			name: "Azure OpenAI 缺少 endpoint",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureDeployment: "gpt-4",
				AzureAPIVersion: "2024-02-15-preview",
			},
			providerType: "azureopenai",
			wantErr:      true,
			errContains:  "azureEndpoint",
		},
		{
			name: "Azure OpenAI 缺少 deployment",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://your-resource.openai.azure.com",
				AzureAPIVersion: "2024-02-15-preview",
			},
			providerType: "azureopenai",
			wantErr:      true,
			errContains:  "azureDeployment",
		},
		{
			name: "Azure OpenAI 缺少 API version",
			config: &GenkitConfig{
				Model:           "gpt-4",
				AzureEndpoint:   "https://your-resource.openai.azure.com",
				AzureDeployment: "gpt-4",
			},
			providerType: "azureopenai",
			wantErr:      true,
			errContains:  "azureApiVersion",
		},
		{
			name: "百炼缺少 endpoint",
			config: &GenkitConfig{
				Model:            "qwen-turbo",
				BailianWorkspace: "default",
			},
			providerType: "bianlian",
			wantErr:      true,
			errContains:  "bailianEndpoint",
		},
		{
			name: "百炼缺少 workspace",
			config: &GenkitConfig{
				Model:           "qwen-turbo",
				BailianEndpoint: "https://dashscope.aliyuncs.com",
			},
			providerType: "bianlian",
			wantErr:      true,
			errContains:  "bailianWorkspace",
		},
		{
			name: "不支持的提供商类型",
			config: &GenkitConfig{
				Model: "test-model",
			},
			providerType: "unknown",
			wantErr:      true,
			errContains:  "不支持的提供商类型",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.providerType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenkitConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("GenkitConfig.Validate() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestGenkitConfig_ToJSON 测试配置序列化
func TestGenkitConfig_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		config  *GenkitConfig
		wantErr bool
	}{
		{
			name: "序列化 Google AI 配置",
			config: &GenkitConfig{
				Model:              "gemini-1.5-pro",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			wantErr: false,
		},
		{
			name: "序列化 Azure OpenAI 配置",
			config: &GenkitConfig{
				Model:              "gpt-4",
				AzureEndpoint:      "https://your-resource.openai.azure.com",
				AzureDeployment:    "gpt-4",
				AzureAPIVersion:    "2024-02-15-preview",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			wantErr: false,
		},
		{
			name: "序列化百炼配置",
			config: &GenkitConfig{
				Model:              "qwen-turbo",
				BailianEndpoint:    "https://dashscope.aliyuncs.com",
				BailianWorkspace:   "default",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2048,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := tt.config.ToJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenkitConfig.ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// 验证可以重新解析
				parsed, err := ParseGenkitConfig(jsonStr)
				if err != nil {
					t.Errorf("Failed to parse serialized config: %v", err)
					return
				}
				// 验证关键字段
				if parsed.Model != tt.config.Model {
					t.Errorf("Model mismatch after serialization: got %v, want %v", parsed.Model, tt.config.Model)
				}
			}
		})
	}
}

// TestGenkitConfig_RoundTrip 测试配置的往返转换
func TestGenkitConfig_RoundTrip(t *testing.T) {
	original := &GenkitConfig{
		Model:              "gpt-4",
		AzureEndpoint:      "https://your-resource.openai.azure.com",
		AzureDeployment:    "gpt-4",
		AzureAPIVersion:    "2024-02-15-preview",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
	}

	// 序列化
	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// 反序列化
	parsed, err := ParseGenkitConfig(jsonStr)
	if err != nil {
		t.Fatalf("ParseGenkitConfig() failed: %v", err)
	}

	// 验证所有字段
	if parsed.Model != original.Model {
		t.Errorf("Model = %v, want %v", parsed.Model, original.Model)
	}
	if parsed.AzureEndpoint != original.AzureEndpoint {
		t.Errorf("AzureEndpoint = %v, want %v", parsed.AzureEndpoint, original.AzureEndpoint)
	}
	if parsed.AzureDeployment != original.AzureDeployment {
		t.Errorf("AzureDeployment = %v, want %v", parsed.AzureDeployment, original.AzureDeployment)
	}
	if parsed.AzureAPIVersion != original.AzureAPIVersion {
		t.Errorf("AzureAPIVersion = %v, want %v", parsed.AzureAPIVersion, original.AzureAPIVersion)
	}
	if parsed.DefaultTemperature != original.DefaultTemperature {
		t.Errorf("DefaultTemperature = %v, want %v", parsed.DefaultTemperature, original.DefaultTemperature)
	}
	if parsed.DefaultMaxTokens != original.DefaultMaxTokens {
		t.Errorf("DefaultMaxTokens = %v, want %v", parsed.DefaultMaxTokens, original.DefaultMaxTokens)
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
