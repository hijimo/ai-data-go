// +build integration

// internal/service/ai/vector_service_integration_test.go
package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVectorService_Integration 集成测试
// 需要设置环境变量 GOOGLE_AI_API_KEY
func TestVectorService_Integration(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过集成测试：未设置 GOOGLE_AI_API_KEY 环境变量")
	}

	// 创建向量服务
	config := &VectorServiceConfig{
		EmbedderModel: "text-embedding-004",
		VectorDim:     1536,
		BatchSize:     10,
		APIKey:        apiKey,
	}

	svc, err := NewVectorService(config)
	require.NoError(t, err)
	require.NotNil(t, svc)

	t.Run("生成单个文本向量", func(t *testing.T) {
		text := "这是一段测试文本，用于生成向量嵌入。"

		embedding, err := svc.GenerateEmbedding(ctx, text)
		require.NoError(t, err)
		require.NotNil(t, embedding)

		// 验证向量维度
		assert.Equal(t, 1536, len(embedding))

		// 验证向量值在合理范围内
		for i, val := range embedding {
			assert.True(t, val >= -1.0 && val <= 1.0,
				"向量值[%d]超出范围: %f", i, val)
		}
	})

	t.Run("生成批量文本向量", func(t *testing.T) {
		texts := []string{
			"人工智能是计算机科学的一个分支。",
			"机器学习是人工智能的核心技术之一。",
			"深度学习使用神经网络进行模式识别。",
		}

		embeddings, err := svc.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		require.NotNil(t, embeddings)

		// 验证返回的向量数量
		assert.Equal(t, len(texts), len(embeddings))

		// 验证每个向量的维度
		for i, embedding := range embeddings {
			assert.Equal(t, 1536, len(embedding),
				"向量[%d]维度不正确", i)
		}
	})

	t.Run("语义相似度测试", func(t *testing.T) {
		// 生成相似文本的向量
		text1 := "机器学习是人工智能的一个重要分支"
		text2 := "人工智能包含机器学习等多个领域"
		text3 := "今天天气很好，适合出去散步"

		embedding1, err := svc.GenerateEmbedding(ctx, text1)
		require.NoError(t, err)

		embedding2, err := svc.GenerateEmbedding(ctx, text2)
		require.NoError(t, err)

		embedding3, err := svc.GenerateEmbedding(ctx, text3)
		require.NoError(t, err)

		// 计算余弦相似度
		similarity12 := cosineSimilarity(embedding1, embedding2)
		similarity13 := cosineSimilarity(embedding1, embedding3)

		// 相似文本的相似度应该更高
		assert.Greater(t, similarity12, similarity13,
			"相似文本的相似度应该更高")

		t.Logf("相似文本相似度: %.4f", similarity12)
		t.Logf("不相似文本相似度: %.4f", similarity13)
	})

	t.Run("大批量处理测试", func(t *testing.T) {
		// 生成25个文本（超过默认批量大小10）
		texts := make([]string, 25)
		for i := 0; i < 25; i++ {
			texts[i] = "这是测试文本编号 " + string(rune(i))
		}

		start := time.Now()
		embeddings, err := svc.GenerateBatchEmbeddings(ctx, texts)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, 25, len(embeddings))

		t.Logf("批量生成25个向量耗时: %v", duration)
	})

	t.Run("中文文本测试", func(t *testing.T) {
		texts := []string{
			"向量数据库用于存储和检索高维向量。",
			"Qdrant是一个高性能的向量搜索引擎。",
			"语义搜索可以理解查询的真实意图。",
		}

		embeddings, err := svc.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		assert.Equal(t, len(texts), len(embeddings))
	})

	t.Run("长文本测试", func(t *testing.T) {
		// 生成一个较长的文本（约1000字符）
		longText := ""
		for i := 0; i < 50; i++ {
			longText += "这是一段较长的测试文本，用于验证向量服务对长文本的处理能力。"
		}

		embedding, err := svc.GenerateEmbedding(ctx, longText)
		require.NoError(t, err)
		assert.Equal(t, 1536, len(embedding))

		t.Logf("长文本长度: %d 字符", len(longText))
	})
}

// TestVectorService_ErrorHandling 错误处理集成测试
func TestVectorService_ErrorHandling(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过集成测试：未设置 GOOGLE_AI_API_KEY 环境变量")
	}

	config := &VectorServiceConfig{
		APIKey: apiKey,
	}

	svc, err := NewVectorService(config)
	require.NoError(t, err)

	t.Run("空文本错误", func(t *testing.T) {
		_, err := svc.GenerateEmbedding(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文本不能为空")
	})

	t.Run("空列表错误", func(t *testing.T) {
		_, err := svc.GenerateBatchEmbeddings(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文本列表不能为空")
	})

	t.Run("包含空文本的列表", func(t *testing.T) {
		texts := []string{"text1", "", "text3"}
		_, err := svc.GenerateBatchEmbeddings(ctx, texts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不能为空")
	})

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		_, err := svc.GenerateEmbedding(ctx, "test text")
		assert.Error(t, err)
	})

	t.Run("上下文超时", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // 确保超时

		_, err := svc.GenerateEmbedding(ctx, "test text")
		assert.Error(t, err)
	})
}

// TestVectorService_Performance 性能测试
func TestVectorService_Performance(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		t.Skip("跳过集成测试：未设置 GOOGLE_AI_API_KEY 环境变量")
	}

	config := &VectorServiceConfig{
		APIKey: apiKey,
	}

	svc, err := NewVectorService(config)
	require.NoError(t, err)

	t.Run("单个文本性能", func(t *testing.T) {
		text := "这是一段测试文本"
		iterations := 5

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := svc.GenerateEmbedding(ctx, text)
			require.NoError(t, err)
		}
		duration := time.Since(start)

		avgDuration := duration / time.Duration(iterations)
		t.Logf("平均单次生成耗时: %v", avgDuration)
	})

	t.Run("批量处理性能对比", func(t *testing.T) {
		texts := make([]string, 10)
		for i := 0; i < 10; i++ {
			texts[i] = "测试文本 " + string(rune(i))
		}

		// 逐个生成
		start1 := time.Now()
		for _, text := range texts {
			_, err := svc.GenerateEmbedding(ctx, text)
			require.NoError(t, err)
		}
		duration1 := time.Since(start1)

		// 批量生成
		start2 := time.Now()
		_, err := svc.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		duration2 := time.Since(start2)

		t.Logf("逐个生成耗时: %v", duration1)
		t.Logf("批量生成耗时: %v", duration2)
		t.Logf("性能提升: %.2fx", float64(duration1)/float64(duration2))

		// 批量处理应该更快
		assert.Less(t, duration2, duration1)
	})
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt 计算平方根
func sqrt(x float32) float32 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
