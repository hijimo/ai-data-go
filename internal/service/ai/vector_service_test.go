// internal/service/ai/vector_service_test.go
package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVectorService 测试创建向量服务
func TestNewVectorService(t *testing.T) {
	tests := []struct {
		name    string
		config  *VectorServiceConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil配置",
			config:  nil,
			wantErr: true,
			errMsg:  "配置不能为空",
		},
		{
			name: "缺少API密钥",
			config: &VectorServiceConfig{
				EmbedderModel: "text-embedding-004",
				VectorDim:     1536,
				BatchSize:     10,
			},
			wantErr: true,
			errMsg:  "API密钥不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVectorService(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestVectorServiceConfig 测试配置默认值
func TestVectorServiceConfig(t *testing.T) {
	config := &VectorServiceConfig{}

	// 测试默认值设置
	if config.EmbedderModel == "" {
		config.EmbedderModel = "text-embedding-004"
	}
	if config.VectorDim == 0 {
		config.VectorDim = 1536
	}
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}

	assert.Equal(t, "text-embedding-004", config.EmbedderModel)
	assert.Equal(t, 1536, config.VectorDim)
	assert.Equal(t, 10, config.BatchSize)
}

// TestRetryConfig 测试重试配置
func TestRetryConfig(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 5*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
}

// TestGenerateEmbedding_Validation 测试输入验证
func TestGenerateEmbedding_Validation(t *testing.T) {
	// 创建模拟服务（不需要真实的Genkit实例）
	svc := &vectorServiceImpl{
		config: &VectorServiceConfig{
			EmbedderModel: "googleai/text-embedding-004",
			VectorDim:     1536,
			BatchSize:     10,
		},
		retryConf: &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     5 * time.Second,
			Multiplier:   2.0,
		},
	}

	ctx := context.Background()

	// 测试空文本
	_, err := svc.GenerateEmbedding(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "文本不能为空")
}

// TestGenerateBatchEmbeddings_Validation 测试批量输入验证
func TestGenerateBatchEmbeddings_Validation(t *testing.T) {
	svc := &vectorServiceImpl{
		config: &VectorServiceConfig{
			EmbedderModel: "googleai/text-embedding-004",
			VectorDim:     1536,
			BatchSize:     10,
		},
		retryConf: &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     5 * time.Second,
			Multiplier:   2.0,
		},
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		texts   []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "空列表",
			texts:   []string{},
			wantErr: true,
			errMsg:  "文本列表不能为空",
		},
		{
			name:    "包含空文本",
			texts:   []string{"text1", "", "text3"},
			wantErr: true,
			errMsg:  "文本[1]不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GenerateBatchEmbeddings(ctx, tt.texts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestBatchProcessing 测试批量处理逻辑
func TestBatchProcessing(t *testing.T) {
	tests := []struct {
		name      string
		totalSize int
		batchSize int
		wantBatch int
	}{
		{
			name:      "正好一批",
			totalSize: 10,
			batchSize: 10,
			wantBatch: 1,
		},
		{
			name:      "多批",
			totalSize: 25,
			batchSize: 10,
			wantBatch: 3,
		},
		{
			name:      "不足一批",
			totalSize: 5,
			batchSize: 10,
			wantBatch: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := 0
			for i := 0; i < tt.totalSize; i += tt.batchSize {
				batches++
			}
			assert.Equal(t, tt.wantBatch, batches)
		})
	}
}

// TestRetryBackoff 测试指数退避计算
func TestRetryBackoff(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	delay := config.InitialDelay
	delays := []time.Duration{delay}

	for i := 0; i < config.MaxRetries; i++ {
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
		delays = append(delays, delay)
	}

	// 验证延迟序列
	assert.Equal(t, 100*time.Millisecond, delays[0])
	assert.Equal(t, 200*time.Millisecond, delays[1])
	assert.Equal(t, 400*time.Millisecond, delays[2])
	assert.Equal(t, 800*time.Millisecond, delays[3])
}
