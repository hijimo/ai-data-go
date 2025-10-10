package vector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorService(t *testing.T) {
	ctx := context.Background()
	
	// 创建向量管理器
	vectorFactory := &DefaultProviderFactory{}
	vectorManager := NewManager(vectorFactory)
	defer vectorManager.Close()
	
	// 创建向量化管理器
	embeddingFactory := &MockEmbeddingProviderFactory{}
	embeddingManager := NewEmbeddingManager(embeddingFactory)
	defer embeddingManager.Close()
	
	// 注册向量存储提供商（使用模拟实现）
	vectorConfig := &Config{
		Provider: ProviderADBPG,
		Settings: map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "test",
			"username": "user",
			"password": "pass",
		},
	}
	err := vectorManager.RegisterProvider(ctx, "test_vector", vectorConfig)
	require.NoError(t, err)
	
	// 注册向量化提供商
	embeddingConfig := &EmbeddingConfig{
		Provider:  EmbeddingProviderOpenAI,
		Model:     "test-model",
		APIKey:    "test-key",
		Dimension: 128,
	}
	err = embeddingManager.RegisterProvider("test_embedding", embeddingConfig)
	require.NoError(t, err)
	
	// 创建向量服务
	service := NewVectorService(vectorManager, embeddingManager)

	// 测试索引和存储
	t.Run("IndexAndStore", func(t *testing.T) {
		documents := []Document{
			{
				ID:      "doc1",
				Content: "This is the first document",
				Metadata: map[string]interface{}{
					"category": "test",
					"type":     "document",
				},
			},
			{
				ID:      "doc2",
				Content: "This is the second document with different content",
				Metadata: map[string]interface{}{
					"category": "test",
					"type":     "document",
				},
			},
		}
		
		err := service.IndexAndStore(ctx, "test_vector", "test_embedding", "test_index", documents)
		require.NoError(t, err)
	})

	// 测试相似性搜索
	t.Run("SearchSimilar", func(t *testing.T) {
		results, err := service.SearchSimilar(ctx, "test_vector", "test_embedding", "test_index", 
			"first document", 10, nil)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		
		// 第一个结果应该是更相似的文档
		assert.Equal(t, "doc1", results[0].ID)
		assert.Greater(t, results[0].Score, results[1].Score)
	})

	// 测试带过滤的搜索
	t.Run("SearchWithFilters", func(t *testing.T) {
		filters := map[string]interface{}{
			"type": "document",
		}
		
		results, err := service.SearchSimilar(ctx, "test_vector", "test_embedding", "test_index", 
			"document", 10, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		
		// 验证所有结果都匹配过滤条件
		for _, result := range results {
			assert.Equal(t, "document", result.Metadata["type"])
		}
	})

	// 测试更新文档
	t.Run("UpdateDocument", func(t *testing.T) {
		updatedDoc := Document{
			ID:      "doc1",
			Content: "This is the updated first document with new content",
			Metadata: map[string]interface{}{
				"category": "updated",
				"type":     "document",
			},
		}
		
		err := service.UpdateDocument(ctx, "test_vector", "test_embedding", "test_index", updatedDoc)
		require.NoError(t, err)
		
		// 验证更新后的搜索结果
		results, err := service.SearchSimilar(ctx, "test_vector", "test_embedding", "test_index", 
			"updated document", 10, nil)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)
		
		// 找到更新的文档
		var foundUpdated bool
		for _, result := range results {
			if result.ID == "doc1" {
				assert.Equal(t, "updated", result.Metadata["category"])
				foundUpdated = true
				break
			}
		}
		assert.True(t, foundUpdated)
	})

	// 测试获取索引统计
	t.Run("GetIndexStats", func(t *testing.T) {
		stats, err := service.GetIndexStats(ctx, "test_vector", "test_index")
		require.NoError(t, err)
		assert.Equal(t, "test_index", stats.Name)
		assert.Equal(t, 128, stats.Dimension)
		assert.Equal(t, int64(2), stats.VectorCount)
	})

	// 测试删除文档
	t.Run("DeleteDocument", func(t *testing.T) {
		err := service.DeleteDocument(ctx, "test_vector", "test_index", "doc2")
		require.NoError(t, err)
		
		// 验证文档已删除
		stats, err := service.GetIndexStats(ctx, "test_vector", "test_index")
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.VectorCount)
	})

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		results := service.HealthCheck(ctx)
		assert.Contains(t, results, "vector_test_vector")
		assert.Contains(t, results, "embedding_test_embedding")
		assert.NoError(t, results["vector_test_vector"])
		assert.NoError(t, results["embedding_test_embedding"])
	})
}

func TestBatchIndexAndStore(t *testing.T) {
	ctx := context.Background()
	
	// 创建向量管理器
	vectorFactory := &DefaultProviderFactory{}
	vectorManager := NewManager(vectorFactory)
	defer vectorManager.Close()
	
	// 创建向量化管理器
	embeddingFactory := &MockEmbeddingProviderFactory{}
	embeddingManager := NewEmbeddingManager(embeddingFactory)
	defer embeddingManager.Close()
	
	// 注册提供商
	vectorConfig := &Config{
		Provider: ProviderADBPG,
		Settings: map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "test",
			"username": "user",
			"password": "pass",
		},
	}
	err := vectorManager.RegisterProvider(ctx, "test_vector", vectorConfig)
	require.NoError(t, err)
	
	embeddingConfig := &EmbeddingConfig{
		Provider:  EmbeddingProviderOpenAI,
		Model:     "test-model",
		APIKey:    "test-key",
		Dimension: 128,
	}
	err = embeddingManager.RegisterProvider("test_embedding", embeddingConfig)
	require.NoError(t, err)
	
	// 创建向量服务
	service := NewVectorService(vectorManager, embeddingManager)

	// 测试批量处理
	t.Run("BatchIndexAndStore", func(t *testing.T) {
		// 创建大量文档
		var documents []Document
		for i := 0; i < 250; i++ {
			documents = append(documents, Document{
				ID:      fmt.Sprintf("doc_%d", i),
				Content: fmt.Sprintf("This is document number %d with unique content", i),
				Metadata: map[string]interface{}{
					"batch":  i / 100, // 批次号
					"number": i,
				},
			})
		}
		
		// 批量处理，每批100个文档
		err := service.BatchIndexAndStore(ctx, "test_vector", "test_embedding", "batch_index", documents, 100)
		require.NoError(t, err)
		
		// 验证所有文档都已索引
		stats, err := service.GetIndexStats(ctx, "test_vector", "batch_index")
		require.NoError(t, err)
		assert.Equal(t, int64(250), stats.VectorCount)
		
		// 测试搜索
		results, err := service.SearchSimilar(ctx, "test_vector", "test_embedding", "batch_index", 
			"document number 100", 5, nil)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)
	})
}

func TestVectorServiceErrors(t *testing.T) {
	ctx := context.Background()
	
	// 创建空的管理器
	vectorManager := NewManager(&DefaultProviderFactory{})
	embeddingManager := NewEmbeddingManager(&MockEmbeddingProviderFactory{})
	service := NewVectorService(vectorManager, embeddingManager)

	// 测试不存在的提供商
	t.Run("NonexistentProviders", func(t *testing.T) {
		documents := []Document{
			{ID: "doc1", Content: "test", Metadata: map[string]interface{}{}},
		}
		
		// 向量提供商不存在
		err := service.IndexAndStore(ctx, "nonexistent_vector", "nonexistent_embedding", "test_index", documents)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get vector provider")
		
		// 向量化提供商不存在
		vectorConfig := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host": "localhost", "port": 5432, "database": "test", "username": "user", "password": "pass",
			},
		}
		vectorManager.RegisterProvider(ctx, "test_vector", vectorConfig)
		
		err = service.IndexAndStore(ctx, "test_vector", "nonexistent_embedding", "test_index", documents)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get embedding provider")
	})
}