package bailian

import (
	"context"
	"testing"

	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBailianPlugin 测试创建百炼插件
func TestNewBailianPlugin(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
		checkFunc   func(*testing.T, *BailianPlugin)
	}{
		{
			name: "成功创建插件 - 使用默认地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, plugin *BailianPlugin) {
				assert.Equal(t, "test-api-key", plugin.APIKey)
				assert.Equal(t, "qwen-plus", plugin.Model)
				assert.Equal(t, DefaultEndpoints["beijing"], plugin.Endpoint)
				assert.NotNil(t, plugin.oaiPlugin)
			},
		},
		{
			name: "成功创建插件 - 指定北京地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-max",
				Region: "beijing",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, plugin *BailianPlugin) {
				assert.Equal(t, "beijing", plugin.Region)
				assert.Equal(t, DefaultEndpoints["beijing"], plugin.Endpoint)
			},
		},
		{
			name: "成功创建插件 - 指定新加坡地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-turbo",
				Region: "singapore",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, plugin *BailianPlugin) {
				assert.Equal(t, "singapore", plugin.Region)
				assert.Equal(t, DefaultEndpoints["singapore"], plugin.Endpoint)
			},
		},
		{
			name: "成功创建插件 - 指定金融云地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
				Region: "finance",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, plugin *BailianPlugin) {
				assert.Equal(t, "finance", plugin.Region)
				assert.Equal(t, DefaultEndpoints["finance"], plugin.Endpoint)
			},
		},
		{
			name: "成功创建插件 - 自定义 Endpoint",
			config: &Config{
				APIKey:   "test-api-key",
				Model:    "qwen-plus",
				Endpoint: "https://custom-endpoint.example.com/v1",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, plugin *BailianPlugin) {
				assert.Equal(t, "https://custom-endpoint.example.com/v1", plugin.Endpoint)
			},
		},
		{
			name:        "配置为空",
			config:      nil,
			wantErr:     true,
			errContains: "配置不能为空",
		},
		{
			name: "API 密钥为空",
			config: &Config{
				Model: "qwen-plus",
			},
			wantErr:     true,
			errContains: "API 密钥不能为空",
		},
		{
			name: "模型名称为空",
			config: &Config{
				APIKey: "test-api-key",
			},
			wantErr:     true,
			errContains: "模型名称不能为空",
		},
		{
			name: "不支持的地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
				Region: "invalid-region",
			},
			wantErr:     true,
			errContains: "不支持的地域",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := NewBailianPlugin(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, plugin)
			} else {
				require.NoError(t, err)
				require.NotNil(t, plugin)
				if tt.checkFunc != nil {
					tt.checkFunc(t, plugin)
				}
			}
		})
	}
}

// TestBailianPlugin_Validate 测试插件验证
func TestBailianPlugin_Validate(t *testing.T) {
	tests := []struct {
		name        string
		plugin      *BailianPlugin
		wantErr     bool
		errContains string
	}{
		{
			name: "验证成功",
			plugin: func() *BailianPlugin {
				plugin, _ := NewBailianPlugin(&Config{
					APIKey: "test-api-key",
					Model:  "qwen-plus",
				})
				return plugin
			}(),
			wantErr: false,
		},
		{
			name: "API 密钥为空",
			plugin: func() *BailianPlugin {
				p := &BailianPlugin{
					Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
					Model:    "qwen-plus",
				}
				// 创建一个有效的 oaiPlugin 实例
				p.oaiPlugin = &oai.OpenAI{
					APIKey: "dummy",
				}
				// 但是清空 APIKey 字段
				p.APIKey = ""
				return p
			}(),
			wantErr:     true,
			errContains: "API 密钥不能为空",
		},
		{
			name: "Endpoint 为空",
			plugin: func() *BailianPlugin {
				p := &BailianPlugin{
					APIKey: "test-api-key",
					Model:  "qwen-plus",
				}
				// 创建一个有效的 oaiPlugin 实例
				p.oaiPlugin = &oai.OpenAI{
					APIKey: "dummy",
				}
				// 但是清空 Endpoint 字段
				p.Endpoint = ""
				return p
			}(),
			wantErr:     true,
			errContains: "API 端点不能为空",
		},
		{
			name: "模型名称为空",
			plugin: func() *BailianPlugin {
				p := &BailianPlugin{
					APIKey:   "test-api-key",
					Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
				}
				// 创建一个有效的 oaiPlugin 实例
				p.oaiPlugin = &oai.OpenAI{
					APIKey: "dummy",
				}
				// 但是清空 Model 字段
				p.Model = ""
				return p
			}(),
			wantErr:     true,
			errContains: "模型名称不能为空",
		},
		{
			name: "OpenAI 插件未初始化",
			plugin: &BailianPlugin{
				APIKey:   "test-api-key",
				Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
				Model:    "qwen-plus",
			},
			wantErr:     true,
			errContains: "OpenAI 插件未初始化",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBailianPlugin_GetModel 测试获取模型名称
func TestBailianPlugin_GetModel(t *testing.T) {
	plugin, err := NewBailianPlugin(&Config{
		APIKey: "test-api-key",
		Model:  "qwen-max",
	})
	require.NoError(t, err)

	assert.Equal(t, "qwen-max", plugin.GetModel())
}

// TestBailianPlugin_GetEndpoint 测试获取 API 端点
func TestBailianPlugin_GetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "默认地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
			},
			expected: DefaultEndpoints["beijing"],
		},
		{
			name: "新加坡地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
				Region: "singapore",
			},
			expected: DefaultEndpoints["singapore"],
		},
		{
			name: "自定义 Endpoint",
			config: &Config{
				APIKey:   "test-api-key",
				Model:    "qwen-plus",
				Endpoint: "https://custom.example.com/v1",
			},
			expected: "https://custom.example.com/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := NewBailianPlugin(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, plugin.GetEndpoint())
		})
	}
}

// TestBailianPlugin_GetRegion 测试获取地域
func TestBailianPlugin_GetRegion(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "默认地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
			},
			expected: "beijing",
		},
		{
			name: "指定地域",
			config: &Config{
				APIKey: "test-api-key",
				Model:  "qwen-plus",
				Region: "singapore",
			},
			expected: "singapore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := NewBailianPlugin(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, plugin.GetRegion())
		})
	}
}

// TestBailianPlugin_Init 测试插件初始化
func TestBailianPlugin_Init(t *testing.T) {
	t.Run("初始化成功", func(t *testing.T) {
		plugin, err := NewBailianPlugin(&Config{
			APIKey: "test-api-key",
			Model:  "qwen-plus",
		})
		require.NoError(t, err)

		// 注意：实际的 Init 需要 Genkit 实例，这里只测试插件是否正确创建
		// 完整的集成测试将在 bailian_integration_test.go 中进行
		assert.NotNil(t, plugin.oaiPlugin)
	})

	t.Run("OpenAI 插件未初始化", func(t *testing.T) {
		plugin := &BailianPlugin{
			APIKey:   "test-api-key",
			Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Model:    "qwen-plus",
		}

		ctx := context.Background()
		actions := plugin.Init(ctx)
		// 如果 OpenAI 插件未初始化，应该返回空的 Action 列表
		assert.Empty(t, actions)
	})
}

// TestDefaultEndpoints 测试默认端点配置
func TestDefaultEndpoints(t *testing.T) {
	// 验证所有默认端点都已配置
	expectedRegions := []string{"beijing", "singapore", "finance"}
	
	for _, region := range expectedRegions {
		endpoint, ok := DefaultEndpoints[region]
		assert.True(t, ok, "地域 %s 应该有默认端点", region)
		assert.NotEmpty(t, endpoint, "地域 %s 的端点不应为空", region)
		assert.Contains(t, endpoint, "dashscope", "端点应包含 dashscope")
		assert.Contains(t, endpoint, "compatible-mode/v1", "端点应包含 compatible-mode/v1")
	}
}
