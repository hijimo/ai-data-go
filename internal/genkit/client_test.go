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

func TestBuildGenerateConfig(t *testing.T) {
	c := &client{
		config: &Config{
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2000,
		},
	}

	// 测试使用自定义选项
	temp := 0.9
	maxTokens := 1000
	topP := 0.95
	topK := 40

	options := &GenerateOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
		TopK:        &topK,
	}

	config := c.buildGenerateConfig(options)

	if config.Temperature != temp {
		t.Errorf("Temperature = %v, want %v", config.Temperature, temp)
	}
	if config.MaxOutputTokens != maxTokens {
		t.Errorf("MaxOutputTokens = %v, want %v", config.MaxOutputTokens, maxTokens)
	}
	if config.TopP != topP {
		t.Errorf("TopP = %v, want %v", config.TopP, topP)
	}
	if config.TopK != topK {
		t.Errorf("TopK = %v, want %v", config.TopK, topK)
	}
}

func TestBuildDefaultConfig(t *testing.T) {
	c := &client{
		config: &Config{
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2000,
		},
	}

	config := c.buildDefaultConfig()

	if config.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", config.Temperature, 0.7)
	}
	if config.MaxOutputTokens != 2000 {
		t.Errorf("MaxOutputTokens = %v, want %v", config.MaxOutputTokens, 2000)
	}
}
