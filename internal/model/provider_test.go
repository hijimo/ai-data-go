package model

import (
	"testing"
)

// TestIsValidModelProvider 测试模型提供商验证函数
func TestIsValidModelProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{
			name:     "有效的提供商 - OpenAI",
			provider: ModelProviderOpenAI,
			want:     true,
		},
		{
			name:     "有效的提供商 - Anthropic",
			provider: ModelProviderAnthropic,
			want:     true,
		},
		{
			name:     "有效的提供商 - GoogleGenAI",
			provider: ModelProviderGoogleGenAI,
			want:     true,
		},
		{
			name:     "有效的提供商 - AzureOpenAI",
			provider: ModelProviderAzureOpenAI,
			want:     true,
		},
		{
			name:     "有效的提供商 - Bianlian",
			provider: ModelProviderBianlian,
			want:     true,
		},
		{
			name:     "有效的提供商 - CustomOpenAI",
			provider: ModelProviderCustomOpenAI,
			want:     true,
		},
		{
			name:     "无效的提供商",
			provider: "invalid_provider",
			want:     false,
		},
		{
			name:     "空字符串",
			provider: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidModelProvider(tt.provider); got != tt.want {
				t.Errorf("IsValidModelProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestModelConfigurationTableName 测试表名
func TestModelConfigurationTableName(t *testing.T) {
	config := ModelConfiguration{}
	expected := "model_configurations"
	if got := config.TableName(); got != expected {
		t.Errorf("TableName() = %v, want %v", got, expected)
	}
}

// TestValidModelProviders 测试有效提供商列表
func TestValidModelProviders(t *testing.T) {
	expected := 6
	if len(ValidModelProviders) != expected {
		t.Errorf("ValidModelProviders 长度 = %v, want %v", len(ValidModelProviders), expected)
	}

	// 验证所有提供商都是唯一的
	seen := make(map[string]bool)
	for _, provider := range ValidModelProviders {
		if seen[provider] {
			t.Errorf("发现重复的提供商: %s", provider)
		}
		seen[provider] = true
	}
}

// TestModelConfigurationStructure 测试 ModelConfiguration 结构体字段
func TestModelConfigurationStructure(t *testing.T) {
	// 创建一个测试实例
	baseURL := "https://api.example.com"
	queryParams := `{"param1": "value1"}`
	updatedBy := "user-uuid-2"
	updatedAt := "2024-01-01T00:00:00Z"
	deletedBy := "user-uuid-3"
	deletedAt := "2024-01-02T00:00:00Z"

	config := ModelConfiguration{
		ID:            "test-uuid",
		TenantID:      "tenant-uuid",
		Name:          "Test Configuration",
		Model:         "gpt-4",
		ModelProvider: ModelProviderOpenAI,
		BaseURL:       &baseURL,
		APIKey:        "encrypted-key",
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     "user-uuid-1",
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedBy:     &updatedBy,
		UpdatedAt:     &updatedAt,
		DeletedBy:     &deletedBy,
		DeletedAt:     &deletedAt,
	}

	// 验证字段值
	if config.ID != "test-uuid" {
		t.Errorf("ID = %v, want %v", config.ID, "test-uuid")
	}
	if config.TenantID != "tenant-uuid" {
		t.Errorf("TenantID = %v, want %v", config.TenantID, "tenant-uuid")
	}
	if config.Name != "Test Configuration" {
		t.Errorf("Name = %v, want %v", config.Name, "Test Configuration")
	}
	if config.Model != "gpt-4" {
		t.Errorf("Model = %v, want %v", config.Model, "gpt-4")
	}
	if config.ModelProvider != ModelProviderOpenAI {
		t.Errorf("ModelProvider = %v, want %v", config.ModelProvider, ModelProviderOpenAI)
	}
	if config.BaseURL == nil || *config.BaseURL != baseURL {
		t.Errorf("BaseURL = %v, want %v", config.BaseURL, baseURL)
	}
	if config.APIKey != "encrypted-key" {
		t.Errorf("APIKey = %v, want %v", config.APIKey, "encrypted-key")
	}
	if config.QueryParams == nil || *config.QueryParams != queryParams {
		t.Errorf("QueryParams = %v, want %v", config.QueryParams, queryParams)
	}
	if !config.IsEnabled {
		t.Errorf("IsEnabled = %v, want %v", config.IsEnabled, true)
	}
	if config.IsDeleted {
		t.Errorf("IsDeleted = %v, want %v", config.IsDeleted, false)
	}
}
