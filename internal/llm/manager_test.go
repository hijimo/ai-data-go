package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	factory := NewDefaultProviderFactory()
	manager := NewManager(factory)
	
	assert.NotNil(t, manager)
	assert.Empty(t, manager.ListProviders())
}

func TestManagerAddProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	manager := NewManager(factory)
	
	config := &OpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderOpenAI,
			Name:    "test-openai",
			APIKey:  "test-key",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 30 * time.Second,
		},
	}
	
	err := manager.AddProvider("openai-1", config)
	require.NoError(t, err)
	
	providers := manager.ListProviders()
	assert.Len(t, providers, 1)
	assert.Contains(t, providers, "openai-1")
}

func TestManagerGetProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	manager := NewManager(factory)
	
	config := &OpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderOpenAI,
			Name:    "test-openai",
			APIKey:  "test-key",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 30 * time.Second,
		},
	}
	
	err := manager.AddProvider("openai-1", config)
	require.NoError(t, err)
	
	provider, err := manager.GetProvider("openai-1")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderOpenAI, provider.GetProviderType())
	
	// 测试获取不存在的提供商
	_, err = manager.GetProvider("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestManagerRemoveProvider(t *testing.T) {
	factory := NewDefaultProviderFactory()
	manager := NewManager(factory)
	
	config := &OpenAIConfig{
		BaseProviderConfig: BaseProviderConfig{
			Type:    ProviderOpenAI,
			Name:    "test-openai",
			APIKey:  "test-key",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 30 * time.Second,
		},
	}
	
	err := manager.AddProvider("openai-1", config)
	require.NoError(t, err)
	
	assert.Len(t, manager.ListProviders(), 1)
	
	manager.RemoveProvider("openai-1")
	assert.Empty(t, manager.ListProviders())
}

func TestManagerWithMockProvider(t *testing.T) {
	// 创建自定义工厂用于测试
	factory := &MockProviderFactory{}
	manager := NewManager(factory)
	
	config := &BaseProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key",
	}
	
	err := manager.AddProvider("mock-provider", config)
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// 测试生成文本
	req := &GenerateRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	
	resp, err := manager.GenerateText(ctx, "mock-provider", req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Mock response", resp.Choices[0].Message.Content)
	
	// 测试流式生成
	streamCh, err := manager.GenerateStream(ctx, "mock-provider", req)
	require.NoError(t, err)
	
	responses := make([]*StreamResponse, 0)
	for resp := range streamCh {
		responses = append(responses, resp)
	}
	
	assert.NotEmpty(t, responses)
	
	// 测试列出模型
	models, err := manager.ListModels(ctx, "mock-provider")
	require.NoError(t, err)
	assert.NotEmpty(t, models)
	
	// 测试健康检查
	err = manager.HealthCheck(ctx, "mock-provider")
	assert.NoError(t, err)
}

func TestManagerHealthCheckAll(t *testing.T) {
	factory := &MockProviderFactory{}
	manager := NewManager(factory)
	
	// 添加多个提供商
	config1 := &BaseProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key-1",
	}
	config2 := &BaseProviderConfig{
		Type:   ProviderQianwen,
		APIKey: "test-key-2",
	}
	
	err := manager.AddProvider("provider-1", config1)
	require.NoError(t, err)
	err = manager.AddProvider("provider-2", config2)
	require.NoError(t, err)
	
	ctx := context.Background()
	results := manager.HealthCheckAll(ctx)
	
	assert.Len(t, results, 2)
	assert.Contains(t, results, "provider-1")
	assert.Contains(t, results, "provider-2")
	assert.NoError(t, results["provider-1"])
	assert.NoError(t, results["provider-2"])
}

func TestDefaultMetricsCollector(t *testing.T) {
	collector := NewDefaultMetricsCollector()
	
	// 记录一些调用
	metrics1 := &CallMetrics{
		ID:           generateTestUUID(),
		ProviderType: ProviderOpenAI,
		Model:        "gpt-3.5-turbo",
		StartTime:    time.Now().Add(-time.Second),
		EndTime:      time.Now(),
		Duration:     time.Second,
		TokenUsage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Success: true,
	}
	
	metrics2 := &CallMetrics{
		ID:           generateTestUUID(),
		ProviderType: ProviderQianwen,
		Model:        "qwen-turbo",
		StartTime:    time.Now().Add(-2 * time.Second),
		EndTime:      time.Now().Add(-time.Second),
		Duration:     time.Second,
		TokenUsage: Usage{
			PromptTokens:     15,
			CompletionTokens: 25,
			TotalTokens:      40,
		},
		Success: false,
		ErrorCode: "API_ERROR",
	}
	
	collector.RecordCall(metrics1)
	collector.RecordCall(metrics2)
	
	// 获取统计信息
	stats := collector.GetMetrics()
	assert.NotNil(t, stats)
	
	statsMap, ok := stats.(map[string]interface{})
	require.True(t, ok)
	
	assert.Equal(t, 2, statsMap["total_calls"])
	assert.Equal(t, 1, statsMap["success_calls"])
	assert.Equal(t, 0.5, statsMap["success_rate"])
	
	// 检查提供商统计
	providerStats, ok := statsMap["provider_stats"].(map[ProviderType]map[string]interface{})
	require.True(t, ok)
	
	assert.Contains(t, providerStats, ProviderOpenAI)
	assert.Contains(t, providerStats, ProviderQianwen)
	
	openaiStats := providerStats[ProviderOpenAI]
	assert.Equal(t, 1, openaiStats["total_calls"])
	assert.Equal(t, 1, openaiStats["success_calls"])
	
	qianwenStats := providerStats[ProviderQianwen]
	assert.Equal(t, 1, qianwenStats["total_calls"])
	assert.Equal(t, 0, qianwenStats["success_calls"])
}

func TestDefaultRateLimiter(t *testing.T) {
	limiter := NewDefaultRateLimiter()
	
	ctx := context.Background()
	
	// 测试允许请求
	allowed, err := limiter.Allow(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, allowed)
	
	// 测试等待
	err = limiter.Wait(ctx, "test-key")
	assert.NoError(t, err)
}

func TestDefaultCircuitBreaker(t *testing.T) {
	breaker := NewDefaultCircuitBreaker()
	
	ctx := context.Background()
	
	// 测试调用
	called := false
	err := breaker.Call(ctx, func() error {
		called = true
		return nil
	})
	
	assert.NoError(t, err)
	assert.True(t, called)
	
	// 测试状态
	state := breaker.State()
	assert.Equal(t, "closed", state)
}

// Mock工厂用于测试
type MockProviderFactory struct{}

func (f *MockProviderFactory) CreateProvider(config ProviderConfig) (LLMProvider, error) {
	return NewMockLLMProvider(config.GetProviderType(), "Mock Provider"), nil
}

func (f *MockProviderFactory) SupportedTypes() []ProviderType {
	return []ProviderType{ProviderOpenAI, ProviderQianwen, ProviderClaude}
}

func TestManagerGetMetrics(t *testing.T) {
	factory := &MockProviderFactory{}
	manager := NewManager(factory)
	
	// 获取初始指标
	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
	
	// 添加提供商并进行一些调用
	config := &BaseProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key",
	}
	
	err := manager.AddProvider("test-provider", config)
	require.NoError(t, err)
	
	ctx := context.Background()
	req := &GenerateRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	
	_, err = manager.GenerateText(ctx, "test-provider", req)
	require.NoError(t, err)
	
	// 再次获取指标，应该有变化
	metricsAfter := manager.GetMetrics()
	assert.NotNil(t, metricsAfter)
	
	statsMap, ok := metricsAfter.(map[string]interface{})
	require.True(t, ok)
	
	totalCalls, ok := statsMap["total_calls"].(int)
	require.True(t, ok)
	assert.Greater(t, totalCalls, 0)
}