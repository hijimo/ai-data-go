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
