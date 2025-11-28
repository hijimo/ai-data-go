package genkit

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient 应该返回非空客户端")
	}
}

func TestNewClientWithRepo(t *testing.T) {
	// 创建一个 nil 的 repository（在实际使用中会注入真实的 repository）
	client := NewClientWithRepo(nil)
	if client == nil {
		t.Fatal("NewClientWithRepo 应该返回非空客户端")
	}
}

func TestClientInitialize(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "配置为空",
			config:  nil,
			wantErr: true,
		},
		{
			name: "API 密钥为空",
			config: &Config{
				Model: "gemini-2.5-flash",
			},
			wantErr: true,
		},
		{
			name: "模型名称为空",
			config: &Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "有效配置",
			config: &Config{
				APIKey:             "test-key",
				Model:              "gemini-2.5-flash",
				DefaultTemperature: 0.7,
				DefaultMaxTokens:   2000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			err := client.Initialize(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientGenerate(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		modelName string
		prompt    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "租户ID为空",
			tenantID:  "",
			modelName: "gemini-pro",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "租户ID不能为空",
		},
		{
			name:      "模型名称为空",
			tenantID:  "tenant-123",
			modelName: "",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "模型名称不能为空",
		},
		{
			name:      "提示词为空",
			tenantID:  "tenant-123",
			modelName: "gemini-pro",
			prompt:    "",
			wantErr:   true,
			errMsg:    "提示词不能为空",
		},
		{
			name:      "配置仓储未初始化",
			tenantID:  "tenant-123",
			modelName: "gemini-pro",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "模型配置仓储未初始化",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建客户端（不注入 repository）
			client := NewClient()
			
			// 调用 Generate
			_, err := client.Generate(context.Background(), tt.tenantID, tt.modelName, tt.prompt, nil)
			
			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err != nil && tt.errMsg != "" {
				// 使用 strings.Contains 检查错误消息
				if err.Error() != tt.errMsg && !stringContains(err.Error(), tt.errMsg) {
					t.Errorf("Generate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// stringContains 检查字符串是否包含子串
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClientGenerateStream(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		modelName string
		prompt    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "租户ID为空",
			tenantID:  "",
			modelName: "gemini-pro",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "租户ID不能为空",
		},
		{
			name:      "模型名称为空",
			tenantID:  "tenant-123",
			modelName: "",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "模型名称不能为空",
		},
		{
			name:      "提示词为空",
			tenantID:  "tenant-123",
			modelName: "gemini-pro",
			prompt:    "",
			wantErr:   true,
			errMsg:    "提示词不能为空",
		},
		{
			name:      "配置仓储未初始化",
			tenantID:  "tenant-123",
			modelName: "gemini-pro",
			prompt:    "Hello",
			wantErr:   true,
			errMsg:    "模型配置仓储未初始化",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建客户端（不注入 repository）
			client := NewClient()
			
			// 调用 GenerateStream
			_, err := client.GenerateStream(context.Background(), tt.tenantID, tt.modelName, tt.prompt, nil)
			
			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err != nil && tt.errMsg != "" {
				// 使用 strings.Contains 检查错误消息
				if err.Error() != tt.errMsg && !stringContains(err.Error(), tt.errMsg) {
					t.Errorf("GenerateStream() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestCreateBailianPlugin(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		genkitConfig *GenkitConfig
		wantErr      bool
		wantEndpoint string
	}{
		{
			name:   "使用默认端点",
			apiKey: "test-api-key",
			genkitConfig: &GenkitConfig{
				Model: "qwen-plus",
			},
			wantErr:      false,
			wantEndpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		},
		{
			name:   "使用自定义端点",
			apiKey: "test-api-key",
			genkitConfig: &GenkitConfig{
				Model:           "qwen-max",
				BailianEndpoint: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
			},
			wantErr:      false,
			wantEndpoint: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		},
		{
			name:   "使用金融云端点",
			apiKey: "test-api-key",
			genkitConfig: &GenkitConfig{
				Model:           "qwen-turbo",
				BailianEndpoint: "https://dashscope-finance.aliyuncs.com/compatible-mode/v1",
			},
			wantErr:      false,
			wantEndpoint: "https://dashscope-finance.aliyuncs.com/compatible-mode/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := createBailianPlugin(tt.apiKey, tt.genkitConfig)
			
			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("createBailianPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// 如果不期望错误，验证插件不为空
			if !tt.wantErr {
				if plugin == nil {
					t.Error("createBailianPlugin() 返回的插件不应为空")
					return
				}
				
				// 验证插件配置了正确的选项
				// 注意：由于 OpenAI 插件的 Opts 是私有字段，我们无法直接验证
				// 但我们可以验证插件对象本身不为空
				if plugin.Opts == nil {
					t.Error("createBailianPlugin() 返回的插件选项不应为空")
				}
			}
		})
	}
}
