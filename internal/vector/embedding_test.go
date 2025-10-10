package vector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbeddingProvider 模拟向量化提供商
type MockEmbeddingProvider struct {
	dimension int
	model     string
	closed    bool
}

// NewMockEmbeddingProvider 创建模拟向量化提供商
func NewMockEmbeddingProvider(dimension int, model string) *MockEmbeddingProvider {
	return &MockEmbeddingProvider{
		dimension: dimension,
		model:     model,
		closed:    false,
	}
}

func (m *MockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.closed {
		return nil, ErrConnectionClosed
	}
	
	// 生成模拟向量（基于文本长度）
	vector := make([]float32, m.dimension)
	for i := range vector {
		vector[i] = float32(len(text)) / float32(m.dimension+i+1)
	}
	
	return vector, nil
}

func (m *MockEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if m.closed {
		return nil, ErrConnectionClosed
	}
	
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

func (m *MockEmbeddingProvider) GetDimension() int {
	return m.dimension
}

func (m *MockEmbeddingProvider) GetModelName() string {
	return m.model
}

func (m *MockEmbeddingProvider) HealthCheck(ctx context.Context) error {
	if m.closed {
		return ErrConnectionClosed
	}
	return nil
}

func (m *MockEmbeddingProvider) Close() error {
	m.closed = true
	return nil
}

// MockEmbeddingProviderFactory 模拟向量化提供商工厂
type MockEmbeddingProviderFactory struct{}

func (f *MockEmbeddingProviderFactory) CreateProvider(config *EmbeddingConfig) (EmbeddingProvider, error) {
	return NewMockEmbeddingProvider(config.Dimension, config.Model), nil
}

func (f *MockEmbeddingProviderFactory) SupportedProviders() []EmbeddingProviderType {
	return []EmbeddingProviderType{EmbeddingProviderOpenAI}
}

func TestEmbeddingConfig(t *testing.T) {
	// 测试配置验证
	t.Run("ValidateConfig", func(t *testing.T) {
		config := &EmbeddingConfig{
			Provider:  EmbeddingProviderOpenAI,
			Model:     "text-embedding-ada-002",
			APIKey:    "test-key",
			Dimension: 1536,
		}
		
		err := config.Validate()
		assert.NoError(t, err)
		
		// 检查默认值
		assert.Equal(t, 10, config.BatchSize)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 3, config.RetryCount)
	})
	
	// 测试无效配置
	t.Run("InvalidConfig", func(t *testing.T) {
		// 缺少提供商
		config := &EmbeddingConfig{}
		err := config.Validate()
		assert.Error(t, err)
		
		// 缺少模型
		config = &EmbeddingConfig{
			Provider: EmbeddingProviderOpenAI,
		}
		err = config.Validate()
		assert.Error(t, err)
		
		// 缺少API密钥
		config = &EmbeddingConfig{
			Provider: EmbeddingProviderOpenAI,
			Model:    "test-model",
		}
		err = config.Validate()
		assert.Error(t, err)
		
		// 无效维度
		config = &EmbeddingConfig{
			Provider:  EmbeddingProviderOpenAI,
			Model:     "test-model",
			APIKey:    "test-key",
			Dimension: 0,
		}
		err = config.Validate()
		assert.Error(t, err)
	})
	
	// 测试默认配置
	t.Run("DefaultConfigs", func(t *testing.T) {
		defaults := GetDefaultConfigs()
		
		assert.Contains(t, defaults, EmbeddingProviderOpenAI)
		assert.Contains(t, defaults, EmbeddingProviderQianwen)
		
		openAIConfig := defaults[EmbeddingProviderOpenAI]
		assert.Equal(t, "text-embedding-ada-002", openAIConfig.Model)
		assert.Equal(t, 1536, openAIConfig.Dimension)
	})
}

func TestEmbeddingManager(t *testing.T) {
	factory := &MockEmbeddingProviderFactory{}
	manager := NewEmbeddingManager(factory)
	defer manager.Close()

	// 测试注册提供商
	t.Run("RegisterProvider", func(t *testing.T) {
		config := &EmbeddingConfig{
			Provider:  EmbeddingProviderOpenAI,
			Model:     "test-model",
			APIKey:    "test-key",
			Dimension: 128,
		}
		
		err := manager.RegisterProvider("test_provider", config)
		require.NoError(t, err)
		
		// 验证提供商已注册
		providers := manager.ListProviders()
		assert.Contains(t, providers, "test_provider")
	})

	// 测试获取提供商
	t.Run("GetProvider", func(t *testing.T) {
		provider, err := manager.GetProvider("test_provider")
		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, 128, provider.GetDimension())
		assert.Equal(t, "test-model", provider.GetModelName())
		
		// 测试获取不存在的提供商
		_, err = manager.GetProvider("nonexistent")
		assert.Error(t, err)
	})

	// 测试获取提供商配置
	t.Run("GetProviderConfig", func(t *testing.T) {
		config, err := manager.GetProviderConfig("test_provider")
		require.NoError(t, err)
		assert.Equal(t, EmbeddingProviderOpenAI, config.Provider)
		assert.Equal(t, "test-model", config.Model)
	})

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		results := manager.HealthCheck(ctx)
		assert.Contains(t, results, "test_provider")
		assert.NoError(t, results["test_provider"])
	})

	// 测试向量化
	t.Run("EmbedWithProvider", func(t *testing.T) {
		ctx := context.Background()
		
		embedding, err := manager.EmbedWithProvider(ctx, "test_provider", "test text")
		require.NoError(t, err)
		assert.Len(t, embedding, 128)
		
		// 测试批量向量化
		embeddings, err := manager.EmbedBatchWithProvider(ctx, "test_provider", []string{"text1", "text2"})
		require.NoError(t, err)
		assert.Len(t, embeddings, 2)
		assert.Len(t, embeddings[0], 128)
		assert.Len(t, embeddings[1], 128)
	})

	// 测试移除提供商
	t.Run("RemoveProvider", func(t *testing.T) {
		err := manager.RemoveProvider("test_provider")
		require.NoError(t, err)
		
		// 验证提供商已移除
		providers := manager.ListProviders()
		assert.NotContains(t, providers, "test_provider")
		
		// 测试移除不存在的提供商
		err = manager.RemoveProvider("nonexistent")
		assert.Error(t, err)
	})
}

func TestMockEmbeddingProvider(t *testing.T) {
	ctx := context.Background()
	provider := NewMockEmbeddingProvider(128, "test-model")
	defer provider.Close()

	// 测试单个文本向量化
	t.Run("Embed", func(t *testing.T) {
		embedding, err := provider.Embed(ctx, "test text")
		require.NoError(t, err)
		assert.Len(t, embedding, 128)
		
		// 验证向量值基于文本长度
		expectedBase := float32(len("test text")) / float32(128+1)
		assert.InDelta(t, expectedBase, embedding[0], 0.001)
	})

	// 测试批量向量化
	t.Run("EmbedBatch", func(t *testing.T) {
		texts := []string{"short", "medium length text", "very long text with many words"}
		
		embeddings, err := provider.EmbedBatch(ctx, texts)
		require.NoError(t, err)
		assert.Len(t, embeddings, 3)
		
		// 验证每个向量的维度
		for _, embedding := range embeddings {
			assert.Len(t, embedding, 128)
		}
		
		// 验证不同长度文本产生不同向量
		assert.NotEqual(t, embeddings[0][0], embeddings[1][0])
		assert.NotEqual(t, embeddings[1][0], embeddings[2][0])
	})

	// 测试获取信息
	t.Run("GetInfo", func(t *testing.T) {
		assert.Equal(t, 128, provider.GetDimension())
		assert.Equal(t, "test-model", provider.GetModelName())
	})

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		err := provider.HealthCheck(ctx)
		assert.NoError(t, err)
		
		// 关闭后健康检查应该失败
		provider.Close()
		err = provider.HealthCheck(ctx)
		assert.Equal(t, ErrConnectionClosed, err)
	})
}

func TestAsyncEmbeddingProcessor(t *testing.T) {
	factory := &MockEmbeddingProviderFactory{}
	manager := NewEmbeddingManager(factory)
	defer manager.Close()
	
	// 注册提供商
	config := &EmbeddingConfig{
		Provider:  EmbeddingProviderOpenAI,
		Model:     "test-model",
		APIKey:    "test-key",
		Dimension: 128,
	}
	err := manager.RegisterProvider("default", config)
	require.NoError(t, err)
	
	// 创建异步处理器
	processor := NewAsyncEmbeddingProcessor(manager, 2)
	processor.Start()
	defer processor.Stop()

	// 测试提交任务
	t.Run("SubmitTask", func(t *testing.T) {
		texts := []string{"text1", "text2", "text3"}
		
		task, err := processor.SubmitTask("default", texts)
		require.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Equal(t, texts, task.Texts)
	})

	// 测试获取任务状态
	t.Run("GetTask", func(t *testing.T) {
		texts := []string{"test text"}
		
		task, err := processor.SubmitTask("default", texts)
		require.NoError(t, err)
		
		// 等待任务处理
		time.Sleep(100 * time.Millisecond)
		
		retrievedTask, err := processor.GetTask(task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, retrievedTask.ID)
		
		// 任务应该已完成或正在处理
		assert.True(t, retrievedTask.Status == TaskStatusCompleted || retrievedTask.Status == TaskStatusProcessing)
	})
}

func TestTaskStatus(t *testing.T) {
	// 测试任务状态字符串表示
	t.Run("TaskStatusString", func(t *testing.T) {
		assert.Equal(t, "pending", TaskStatusPending.String())
		assert.Equal(t, "processing", TaskStatusProcessing.String())
		assert.Equal(t, "completed", TaskStatusCompleted.String())
		assert.Equal(t, "failed", TaskStatusFailed.String())
		assert.Equal(t, "cancelled", TaskStatusCancelled.String())
		assert.Equal(t, "unknown", TaskStatus(999).String())
	})
}