package genkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMaskAPIKey 测试 API 密钥脱敏功能
func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "空字符串",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "短密钥（8位以下）",
			apiKey:   "short",
			expected: "****",
		},
		{
			name:     "8位密钥",
			apiKey:   "12345678",
			expected: "****",
		},
		{
			name:     "标准长度密钥",
			apiKey:   "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "长密钥",
			apiKey:   "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "sk-p****wxyz",
		},
		{
			name:     "Azure密钥",
			apiKey:   "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
			expected: "a1b2****o5p6",
		},
		{
			name:     "百炼密钥",
			apiKey:   "sk-1234567890abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "sk-1****7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.apiKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMaskSensitiveConfig 测试配置脱敏功能
func TestMaskSensitiveConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   interface{}
		expected map[string]interface{}
	}{
		{
			name: "包含 apiKey 的配置",
			config: map[string]interface{}{
				"apiKey": "sk-1234567890abcdef",
				"model":  "gpt-4",
			},
			expected: map[string]interface{}{
				"apiKey": "sk-1****cdef",
				"model":  "gpt-4",
			},
		},
		{
			name: "包含 APIKey 的配置",
			config: map[string]interface{}{
				"APIKey": "sk-1234567890abcdef",
				"model":  "gemini-pro",
			},
			expected: map[string]interface{}{
				"APIKey": "sk-1****cdef",
				"model":  "gemini-pro",
			},
		},
		{
			name: "包含 api_key 的配置",
			config: map[string]interface{}{
				"api_key": "sk-1234567890abcdef",
				"model":   "qwen-plus",
			},
			expected: map[string]interface{}{
				"api_key": "sk-1****cdef",
				"model":   "qwen-plus",
			},
		},
		{
			name: "复杂配置",
			config: map[string]interface{}{
				"apiKey":          "sk-1234567890abcdef",
				"model":           "gpt-4",
				"azureEndpoint":   "https://example.openai.azure.com",
				"azureDeployment": "gpt-4",
			},
			expected: map[string]interface{}{
				"apiKey":          "sk-1****cdef",
				"model":           "gpt-4",
				"azureEndpoint":   "https://example.openai.azure.com",
				"azureDeployment": "gpt-4",
			},
		},
		{
			name: "不包含敏感信息的配置",
			config: map[string]interface{}{
				"model":           "gemini-pro",
				"temperature":     0.7,
				"maxTokens":       2048,
			},
			expected: map[string]interface{}{
				"model":       "gemini-pro",
				"temperature": 0.7,
				"maxTokens":   float64(2048), // JSON 反序列化会将数字转为 float64
			},
		},
		{
			name: "结构体配置",
			config: Config{
				APIKey: "sk-1234567890abcdef",
				Model:  "gpt-4",
			},
			expected: map[string]interface{}{
				"APIKey": "sk-1****cdef",
				"Model":  "gpt-4",
				"DefaultTemperature": float64(0),
				"DefaultMaxTokens":   float64(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveConfig(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMaskSensitiveConfig_EdgeCases 测试边界情况
func TestMaskSensitiveConfig_EdgeCases(t *testing.T) {
	t.Run("空配置", func(t *testing.T) {
		result := MaskSensitiveConfig(map[string]interface{}{})
		assert.Empty(t, result)
	})

	t.Run("nil 配置", func(t *testing.T) {
		result := MaskSensitiveConfig(nil)
		assert.Empty(t, result)
	})

	t.Run("非 JSON 可序列化的配置", func(t *testing.T) {
		// 包含 channel 的结构体无法序列化为 JSON
		type InvalidConfig struct {
			Ch chan int
		}
		result := MaskSensitiveConfig(InvalidConfig{Ch: make(chan int)})
		assert.Empty(t, result)
	})
}
