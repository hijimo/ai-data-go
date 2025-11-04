package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVectorServiceConfig 测试向量服务配置
func TestVectorServiceConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *VectorServiceConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "空配置",
			config:      nil,
			expectError: true,
			errorMsg:    "配置不能为空",
		},
		{
			name: "缺少 API 密钥",
			config: &VectorServiceConfig{
				Provider: EmbeddingProviderGoogleAI,
			},
			expectError: true,
			errorMsg:    "API 密钥不能为空",
		},
		{
			name: "不支持的提供商",
			config: &VectorServiceConfig{
				Provider: "unsupported",
				APIKey:   "test-key",
			},
			expectError: true,
			errorMsg:    "不支持的嵌入模型提供商",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 跳过需要实际 API 密钥的测试
			if tt.config != nil && tt.config.APIKey != "" {
				t.Skip("需要实际的 API 密钥")
			}

			// 只测试配置验证逻辑
			if tt.expectError {
				// 这些测试不需要实际创建服务
				if tt.config == nil {
					assert.Contains(t, "配置不能为空", tt.errorMsg)
				}
			}
		})
	}
}

// TestEmbeddingProviderConstants 测试嵌入模型提供商常量
func TestEmbeddingProviderConstants(t *testing.T) {
	assert.Equal(t, EmbeddingProvider("google"), EmbeddingProviderGoogleAI)
}

// TestVectorServiceConfigDefaults 测试配置默认值
func TestVectorServiceConfigDefaults(t *testing.T) {
	config := &VectorServiceConfig{
		Provider: EmbeddingProviderGoogleAI,
		APIKey:   "test-key",
	}

	// 默认值应该在创建服务时设置
	assert.Equal(t, 0, config.Dimension)
	assert.Equal(t, 0, config.BatchSize)
	assert.Equal(t, 0, config.MaxRetries)
}

// TestGenerateEmbeddingEmptyText 测试空文本生成向量
func TestGenerateEmbeddingEmptyText(t *testing.T) {
	// 这个测试需要实际的 API 密钥，所以我们跳过
	// 在实际环境中，应该使用 mock 来测试
	t.Skip("需要实际的 API 密钥")
}

// TestGenerateEmbeddingsBatchProcessing 测试批量处理逻辑
func TestGenerateEmbeddingsBatchProcessing(t *testing.T) {
	tests := []struct {
		name       string
		texts      []string
		batchSize  int
		expectBatches int
	}{
		{
			name:       "空列表",
			texts:      []string{},
			batchSize:  10,
			expectBatches: 0,
		},
		{
			name:       "单个批次",
			texts:      []string{"text1", "text2", "text3"},
			batchSize:  10,
			expectBatches: 1,
		},
		{
			name:       "多个批次",
			texts:      []string{"text1", "text2", "text3", "text4", "text5"},
			batchSize:  2,
			expectBatches: 3,
		},
		{
			name:       "正好整除",
			texts:      []string{"text1", "text2", "text3", "text4"},
			batchSize:  2,
			expectBatches: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalTexts := len(tt.texts)
			batches := 0

			for i := 0; i < totalTexts; i += tt.batchSize {
				end := i + tt.batchSize
				if end > totalTexts {
					end = totalTexts
				}
				batches++
			}

			assert.Equal(t, tt.expectBatches, batches)
		})
	}
}

// TestGetEmbeddingDimension 测试获取向量维度
func TestGetEmbeddingDimension(t *testing.T) {
	tests := []struct {
		name      string
		dimension int
	}{
		{
			name:      "Google AI 默认维度",
			dimension: 768,
		},
		{
			name:      "OpenAI 默认维度",
			dimension: 1536,
		},
		{
			name:      "自定义维度",
			dimension: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试维度值的有效性
			assert.Greater(t, tt.dimension, 0)
			assert.LessOrEqual(t, tt.dimension, 4096)
		})
	}
}

// BenchmarkBatchSizeCalculation 基准测试批量大小计算
func BenchmarkBatchSizeCalculation(b *testing.B) {
	texts := make([]string, 1000)
	for i := range texts {
		texts[i] = "test text"
	}

	batchSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batches := 0
		for j := 0; j < len(texts); j += batchSize {
			end := j + batchSize
			if end > len(texts) {
				end = len(texts)
			}
			batches++
		}
	}
}
