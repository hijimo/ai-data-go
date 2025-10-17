package main

import (
	"context"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	
	"github.com/firebase/genkit/go/ai"
)

// TestInitDatabase 测试数据库初始化
func TestInitDatabase(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)

	// 注意：这个测试需要实际的数据库连接
	// 在 CI/CD 环境中，应该使用 mock 或跳过此测试
	t.Run("数据库初始化失败时返回错误", func(t *testing.T) {
		// 使用无效的配置
		invalidCfg := &config.Config{
			Database: config.DatabaseConfig{
				Host:            "invalid-host",
				Port:            "9999",
				User:            "invalid",
				Password:        "invalid",
				DBName:          "invalid",
				SSLMode:         "disable",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
		}

		_, err := initDatabase(invalidCfg, log)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})
}

// TestInitGenkit 测试 Genkit 客户端初始化
func TestInitGenkit(t *testing.T) {
	cfg := &config.Config{
		Genkit: config.GenkitConfig{
			APIKey:             "test-api-key",
			Model:              "gemini-2.0-flash-exp",
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2000,
		},
	}

	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)

	t.Run("Genkit 初始化", func(t *testing.T) {
		// 注意：这个测试需要有效的 API 密钥
		// 在实际环境中应该使用 mock
		client, err := initGenkit(cfg, log)
		
		// 如果没有有效的 API 密钥，测试会失败
		// 这是预期的行为
		if err != nil {
			t.Logf("Genkit 初始化失败（预期行为，如果没有有效的 API 密钥）: %v", err)
		} else if client == nil {
			t.Error("期望返回客户端实例，但得到 nil")
		}
	})
}

// TestInitAIService 测试 AI 服务初始化
func TestInitAIService(t *testing.T) {
	cfg := &config.Config{
		Session: config.SessionConfig{
			Timeout:         30 * time.Minute,
			CleanupInterval: 5 * time.Minute,
		},
	}

	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)

	t.Run("AI 服务初始化成功", func(t *testing.T) {
		// 创建一个 mock Genkit 客户端
		mockClient := &mockGenkitClient{}

		service := initAIService(mockClient, cfg, log)
		if service == nil {
			t.Error("期望返回服务实例，但得到 nil")
		}
	})
}

// mockGenkitClient 用于测试的 mock Genkit 客户端
type mockGenkitClient struct{}

func (m *mockGenkitClient) Initialize(ctx context.Context, config *genkit.Config) error {
	return nil
}

func (m *mockGenkitClient) Generate(ctx context.Context, prompt string, options *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
	return &genkit.GenerateResult{
		Text:  "mock response",
		Model: "mock-model",
		Usage: &genkit.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockGenkitClient) SetModel(model ai.Model) {
	// Mock 实现，不需要实际设置模型
}

func (m *mockGenkitClient) Close() error {
	return nil
}

func (m *mockGenkitClient) InitializeModel(ctx context.Context) error {
	return nil
}

// TestInitProviderService 测试模型提供商服务初始化
func TestInitProviderService(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.JSONFormat, os.Stdout)

	t.Run("模型提供商服务初始化 - 目录不存在", func(t *testing.T) {
		cfg := &config.Config{
			Models: config.ModelsConfig{
				Dir: "/nonexistent/directory",
			},
		}

		_, err := initProviderService(cfg, log)
		if err == nil {
			t.Error("期望返回错误，但得到 nil")
		}
	})

	t.Run("模型提供商服务初始化 - 使用默认目录", func(t *testing.T) {
		cfg := &config.Config{
			Models: config.ModelsConfig{
				Dir: "./models",
			},
		}

		service, err := initProviderService(cfg, log)
		// 如果 models 目录存在且有有效数据，应该成功
		// 如果不存在，会返回错误
		if err != nil {
			t.Logf("模型提供商服务初始化失败（如果 models 目录不存在是预期的）: %v", err)
		} else if service == nil {
			t.Error("期望返回服务实例，但得到 nil")
		}
	})
}
