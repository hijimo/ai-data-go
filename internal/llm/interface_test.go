package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderTypes(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		expected     string
	}{
		{"OpenAI", ProviderOpenAI, "openai"},
		{"Azure", ProviderAzure, "azure"},
		{"Qianwen", ProviderQianwen, "qianwen"},
		{"Claude", ProviderClaude, "claude"},
		{"Baichuan", ProviderBaichuan, "baichuan"},
		{"ChatGLM", ProviderChatGLM, "chatglm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.providerType))
		})
	}
}

func TestModelTypes(t *testing.T) {
	tests := []struct {
		name      string
		modelType ModelType
		expected  string
	}{
		{"Chat", ModelTypeChat, "chat"},
		{"Completion", ModelTypeCompletion, "completion"},
		{"Embedding", ModelTypeEmbedding, "embedding"},
		{"Image", ModelTypeImage, "image"},
		{"Audio", ModelTypeAudio, "audio"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.modelType))
		})
	}
}

func TestBaseProviderConfig(t *testing.T) {
	config := &BaseProviderConfig{
		Type:    ProviderOpenAI,
		Name:    "test-openai",
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
		Timeout: 30 * time.Second,
	}

	assert.Equal(t, ProviderOpenAI, config.GetProviderType())
	assert.Equal(t, "test-key", config.GetAPIKey())
	assert.Equal(t, "https://api.openai.com/v1", config.GetBaseURL())
	assert.NoError(t, config.Validate())
}

func TestBaseProviderConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *BaseProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &BaseProviderConfig{
				Type:   ProviderOpenAI,
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing provider type",
			config: &BaseProviderConfig{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &BaseProviderConfig{
				Type: ProviderOpenAI,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     *GenerateRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &GenerateRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: &GenerateRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty messages",
			req: &GenerateRequest{
				Model:    "gpt-3.5-turbo",
				Messages: []Message{},
			},
			wantErr: true,
		},
		{
			name: "invalid message role",
			req: &GenerateRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "invalid", Content: "Hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty message content",
			req: &GenerateRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: ""},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里需要实际的验证逻辑，暂时跳过
			// 在实际实现中，应该使用validator库进行验证
			t.Skip("需要实现验证逻辑")
		})
	}
}

func TestCallMetrics(t *testing.T) {
	metrics := &CallMetrics{
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

	assert.Equal(t, ProviderOpenAI, metrics.ProviderType)
	assert.Equal(t, "gpt-3.5-turbo", metrics.Model)
	assert.Equal(t, time.Second, metrics.Duration)
	assert.Equal(t, 30, metrics.TokenUsage.TotalTokens)
	assert.True(t, metrics.Success)
}

func TestModelLimits(t *testing.T) {
	limits := ModelLimits{
		MaxTokens:       4096,
		MaxInputTokens:  4096,
		MaxOutputTokens: 4096,
		ContextWindow:   4096,
	}

	assert.Equal(t, 4096, limits.MaxTokens)
	assert.Equal(t, 4096, limits.ContextWindow)
}

func TestModelPricing(t *testing.T) {
	pricing := &ModelPricing{
		InputPrice:  0.001,
		OutputPrice: 0.002,
		Currency:    "USD",
	}

	assert.Equal(t, 0.001, pricing.InputPrice)
	assert.Equal(t, 0.002, pricing.OutputPrice)
	assert.Equal(t, "USD", pricing.Currency)
}

// Mock实现用于测试
type MockLLMProvider struct {
	providerType ProviderType
	providerName string
	models       []Model
	healthError  error
}

func NewMockLLMProvider(providerType ProviderType, providerName string) *MockLLMProvider {
	return &MockLLMProvider{
		providerType: providerType,
		providerName: providerName,
		models: []Model{
			{
				ID:          "mock-model-1",
				DisplayName: "Mock Model 1",
				ModelType:   ModelTypeChat,
			},
		},
	}
}

func (m *MockLLMProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{
		ID:      "mock-response",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Mock response",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (m *MockLLMProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error) {
	ch := make(chan *StreamResponse, 2)
	go func() {
		defer close(ch)
		ch <- &StreamResponse{
			ID:      "mock-stream",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: MessageDelta{
						Content: "Mock",
					},
				},
			},
		}
		ch <- &StreamResponse{
			Done: true,
		}
	}()
	return ch, nil
}

func (m *MockLLMProvider) ListModels(ctx context.Context) ([]Model, error) {
	return m.models, nil
}

func (m *MockLLMProvider) HealthCheck(ctx context.Context) error {
	return m.healthError
}

func (m *MockLLMProvider) GetProviderType() ProviderType {
	return m.providerType
}

func (m *MockLLMProvider) GetProviderName() string {
	return m.providerName
}

func TestMockLLMProvider(t *testing.T) {
	provider := NewMockLLMProvider(ProviderOpenAI, "Mock OpenAI")
	
	assert.Equal(t, ProviderOpenAI, provider.GetProviderType())
	assert.Equal(t, "Mock OpenAI", provider.GetProviderName())

	ctx := context.Background()
	
	// 测试生成文本
	req := &GenerateRequest{
		Model: "mock-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	
	resp, err := provider.GenerateText(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "mock-response", resp.ID)
	assert.Equal(t, "Mock response", resp.Choices[0].Message.Content)

	// 测试流式生成
	streamCh, err := provider.GenerateStream(ctx, req)
	require.NoError(t, err)
	
	responses := make([]*StreamResponse, 0)
	for resp := range streamCh {
		responses = append(responses, resp)
	}
	
	assert.Len(t, responses, 2)
	assert.Equal(t, "Mock", responses[0].Choices[0].Delta.Content)
	assert.True(t, responses[1].Done)

	// 测试列出模型
	models, err := provider.ListModels(ctx)
	require.NoError(t, err)
	assert.Len(t, models, 1)
	assert.Equal(t, "mock-model-1", models[0].ID)

	// 测试健康检查
	err = provider.HealthCheck(ctx)
	assert.NoError(t, err)
}

// 辅助函数
func generateTestUUID() uuid.UUID {
	return uuid.New()
}