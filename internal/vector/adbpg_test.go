package vector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestADBPGProvider(t *testing.T) {
	// 注意：这些测试需要真实的ADB-PG数据库连接
	// 在CI/CD环境中，应该使用测试数据库或模拟
	t.Skip("Skipping ADB-PG integration tests - requires real database")

	ctx := context.Background()
	
	// 创建测试配置
	config := &Config{
		Provider: ProviderADBPG,
		Settings: map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "test_vectors",
			"username": "test_user",
			"password": "test_pass",
			"ssl_mode": "disable",
		},
	}
	
	// 创建提供商
	provider, err := NewADBPGProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Close()
	
	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		err := provider.HealthCheck(ctx)
		assert.NoError(t, err)
	})
	
	// 测试创建索引
	t.Run("CreateIndex", func(t *testing.T) {
		req := &CreateIndexRequest{
			Name:            "test_index",
			Dimension:       128,
			DistanceMeasure: string(DistanceCosine),
			IndexType:       string(IndexHNSW),
		}
		
		err := provider.CreateIndex(ctx, req)
		require.NoError(t, err)
		
		// 验证索引存在
		exists, err := provider.IndexExists(ctx, "test_index")
		require.NoError(t, err)
		assert.True(t, exists)
	})
	
	// 测试插入向量
	t.Run("InsertVectors", func(t *testing.T) {
		vectors := []Vector{
			{
				ID:     "vec1",
				Values: make([]float32, 128), // 零向量
				Metadata: map[string]interface{}{
					"category": "test",
					"type":     "document",
				},
			},
			{
				ID:     "vec2",
				Values: make([]float32, 128), // 零向量
				Metadata: map[string]interface{}{
					"category": "test",
					"type":     "image",
				},
			},
		}
		
		// 设置一些非零值
		vectors[0].Values[0] = 1.0
		vectors[1].Values[1] = 1.0
		
		err := provider.InsertVectors(ctx, "test_index", vectors)
		require.NoError(t, err)
	})
	
	// 测试获取向量
	t.Run("GetVector", func(t *testing.T) {
		vector, err := provider.GetVector(ctx, "test_index", "vec1")
		require.NoError(t, err)
		assert.Equal(t, "vec1", vector.ID)
		assert.Equal(t, float32(1.0), vector.Values[0])
		assert.Equal(t, "test", vector.Metadata["category"])
	})
	
	// 测试向量搜索
	t.Run("Search", func(t *testing.T) {
		queryVector := make([]float32, 128)
		queryVector[0] = 1.0 // 应该与vec1最相似
		
		searchReq := &SearchRequest{
			Vector: queryVector,
			TopK:   10,
		}
		
		results, err := provider.Search(ctx, "test_index", searchReq)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		
		// 第一个结果应该是vec1（最相似）
		assert.Equal(t, "vec1", results[0].ID)
		assert.Greater(t, results[0].Score, results[1].Score)
	})
	
	// 测试带过滤的搜索
	t.Run("SearchWithFilters", func(t *testing.T) {
		queryVector := make([]float32, 128)
		queryVector[0] = 1.0
		
		searchReq := &SearchRequest{
			Vector: queryVector,
			TopK:   10,
			Filters: map[string]interface{}{
				"type": "document",
			},
		}
		
		results, err := provider.Search(ctx, "test_index", searchReq)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "vec1", results[0].ID)
	})
	
	// 测试获取统计信息
	t.Run("GetStats", func(t *testing.T) {
		stats, err := provider.GetStats(ctx, "test_index")
		require.NoError(t, err)
		assert.Equal(t, "test_index", stats.Name)
		assert.Equal(t, 128, stats.Dimension)
		assert.Equal(t, int64(2), stats.VectorCount)
		assert.Greater(t, stats.IndexSize, int64(0))
	})
	
	// 测试删除向量
	t.Run("DeleteVectors", func(t *testing.T) {
		err := provider.DeleteVectors(ctx, "test_index", []string{"vec1"})
		require.NoError(t, err)
		
		// 验证向量已删除
		_, err = provider.GetVector(ctx, "test_index", "vec1")
		assert.Error(t, err)
		assert.True(t, IsVectorError(err))
	})
	
	// 测试删除索引
	t.Run("DeleteIndex", func(t *testing.T) {
		err := provider.DeleteIndex(ctx, "test_index")
		require.NoError(t, err)
		
		// 验证索引已删除
		exists, err := provider.IndexExists(ctx, "test_index")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestADBPGProviderUtils(t *testing.T) {
	provider := &ADBPGProvider{
		utils: NewVectorUtils(),
	}
	
	// 测试向量字符串转换
	t.Run("VectorStringConversion", func(t *testing.T) {
		vector := []float32{1.0, 2.5, -3.14, 0.0}
		
		// 转换为字符串
		vectorStr := provider.vectorToString(vector)
		assert.Equal(t, "[1,2.5,-3.14,0]", vectorStr)
		
		// 转换回向量
		parsedVector, err := provider.stringToVector(vectorStr)
		require.NoError(t, err)
		assert.Equal(t, vector, parsedVector)
		
		// 测试空向量
		emptyStr := provider.vectorToString([]float32{})
		assert.Equal(t, "[]", emptyStr)
		
		emptyVector, err := provider.stringToVector("[]")
		require.NoError(t, err)
		assert.Empty(t, emptyVector)
	})
	
	// 测试表名生成
	t.Run("TableNameGeneration", func(t *testing.T) {
		tableName := provider.getTableName("my_index")
		assert.Equal(t, "vector_index_my_index", tableName)
		
		indexName := provider.getIndexName("my_index")
		assert.Equal(t, "idx_my_index_embedding", indexName)
	})
	
	// 测试索引参数
	t.Run("IndexParameters", func(t *testing.T) {
		req := &CreateIndexRequest{
			IndexType: string(IndexHNSW),
		}
		
		params := provider.getIndexParameters(req)
		assert.Equal(t, 16, params["m"])
		assert.Equal(t, 200, params["ef_construction"])
		
		// 测试自定义参数
		req.Parameters = map[string]interface{}{
			"m": 32,
		}
		
		params = provider.getIndexParameters(req)
		assert.Equal(t, 32, params["m"])
		assert.Equal(t, 200, params["ef_construction"])
	})
}

func TestADBPGConfig(t *testing.T) {
	// 测试配置解析
	t.Run("ConfigParsing", func(t *testing.T) {
		config := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host":           "localhost",
				"port":           5432,
				"database":       "test",
				"username":       "user",
				"password":       "pass",
				"ssl_mode":       "require",
				"max_open_conns": 50,
				"max_idle_conns": 10,
			},
		}
		
		adbpgConfig, err := config.GetADBPGConfig()
		require.NoError(t, err)
		
		assert.Equal(t, "localhost", adbpgConfig.Host)
		assert.Equal(t, 5432, adbpgConfig.Port)
		assert.Equal(t, "test", adbpgConfig.Database)
		assert.Equal(t, "user", adbpgConfig.Username)
		assert.Equal(t, "pass", adbpgConfig.Password)
		assert.Equal(t, "require", adbpgConfig.SSLMode)
		assert.Equal(t, 50, adbpgConfig.MaxOpenConns)
		assert.Equal(t, 10, adbpgConfig.MaxIdleConns)
	})
	
	// 测试默认值
	t.Run("DefaultValues", func(t *testing.T) {
		config := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
		}
		
		adbpgConfig, err := config.GetADBPGConfig()
		require.NoError(t, err)
		
		assert.Equal(t, "require", adbpgConfig.SSLMode)
		assert.Equal(t, 25, adbpgConfig.MaxOpenConns)
		assert.Equal(t, 5, adbpgConfig.MaxIdleConns)
	})
}