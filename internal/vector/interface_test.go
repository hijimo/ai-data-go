package vector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVectorProvider 模拟向量提供商，用于测试
type MockVectorProvider struct {
	indexes map[string]*IndexStats
	vectors map[string]map[string]*Vector
	closed  bool
}

// NewMockVectorProvider 创建模拟向量提供商
func NewMockVectorProvider() *MockVectorProvider {
	return &MockVectorProvider{
		indexes: make(map[string]*IndexStats),
		vectors: make(map[string]map[string]*Vector),
		closed:  false,
	}
}

func (m *MockVectorProvider) CreateIndex(ctx context.Context, req *CreateIndexRequest) error {
	if m.closed {
		return ErrConnectionClosed
	}
	
	if _, exists := m.indexes[req.Name]; exists {
		return ErrIndexAlreadyExists
	}
	
	m.indexes[req.Name] = &IndexStats{
		Name:        req.Name,
		Dimension:   req.Dimension,
		VectorCount: 0,
		IndexSize:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	m.vectors[req.Name] = make(map[string]*Vector)
	return nil
}

func (m *MockVectorProvider) DeleteIndex(ctx context.Context, indexName string) error {
	if m.closed {
		return ErrConnectionClosed
	}
	
	if _, exists := m.indexes[indexName]; !exists {
		return ErrIndexNotFound
	}
	
	delete(m.indexes, indexName)
	delete(m.vectors, indexName)
	return nil
}

func (m *MockVectorProvider) IndexExists(ctx context.Context, indexName string) (bool, error) {
	if m.closed {
		return false, ErrConnectionClosed
	}
	
	_, exists := m.indexes[indexName]
	return exists, nil
}

func (m *MockVectorProvider) InsertVectors(ctx context.Context, indexName string, vectors []Vector) error {
	if m.closed {
		return ErrConnectionClosed
	}
	
	indexVectors, exists := m.vectors[indexName]
	if !exists {
		return ErrIndexNotFound
	}
	
	for _, vector := range vectors {
		indexVectors[vector.ID] = &vector
	}
	
	// 更新统计信息
	if stats, exists := m.indexes[indexName]; exists {
		stats.VectorCount = int64(len(indexVectors))
		stats.UpdatedAt = time.Now()
	}
	
	return nil
}

func (m *MockVectorProvider) BatchInsertVectors(ctx context.Context, indexName string, vectors []Vector, batchSize int) error {
	return m.InsertVectors(ctx, indexName, vectors)
}

func (m *MockVectorProvider) Search(ctx context.Context, indexName string, req *SearchRequest) ([]SearchResult, error) {
	if m.closed {
		return nil, ErrConnectionClosed
	}
	
	indexVectors, exists := m.vectors[indexName]
	if !exists {
		return nil, ErrIndexNotFound
	}
	
	var results []SearchResult
	utils := NewVectorUtils()
	
	for id, vector := range indexVectors {
		// 简单的余弦相似度计算
		score := utils.CosineSimilarity(req.Vector, vector.Values)
		
		results = append(results, SearchResult{
			ID:       id,
			Score:    score,
			Values:   vector.Values,
			Metadata: vector.Metadata,
		})
	}
	
	// 按分数排序
	utils.SortSearchResults(results, false) // 降序排列
	
	// 限制返回数量
	if len(results) > req.TopK {
		results = results[:req.TopK]
	}
	
	return results, nil
}

func (m *MockVectorProvider) DeleteVectors(ctx context.Context, indexName string, ids []string) error {
	if m.closed {
		return ErrConnectionClosed
	}
	
	indexVectors, exists := m.vectors[indexName]
	if !exists {
		return ErrIndexNotFound
	}
	
	for _, id := range ids {
		delete(indexVectors, id)
	}
	
	// 更新统计信息
	if stats, exists := m.indexes[indexName]; exists {
		stats.VectorCount = int64(len(indexVectors))
		stats.UpdatedAt = time.Now()
	}
	
	return nil
}

func (m *MockVectorProvider) UpdateVectors(ctx context.Context, indexName string, vectors []Vector) error {
	return m.InsertVectors(ctx, indexName, vectors) // 简单实现，直接覆盖
}

func (m *MockVectorProvider) GetVector(ctx context.Context, indexName string, id string) (*Vector, error) {
	if m.closed {
		return nil, ErrConnectionClosed
	}
	
	indexVectors, exists := m.vectors[indexName]
	if !exists {
		return nil, ErrIndexNotFound
	}
	
	vector, exists := indexVectors[id]
	if !exists {
		return nil, ErrVectorNotFound
	}
	
	return vector, nil
}

func (m *MockVectorProvider) GetStats(ctx context.Context, indexName string) (*IndexStats, error) {
	if m.closed {
		return nil, ErrConnectionClosed
	}
	
	stats, exists := m.indexes[indexName]
	if !exists {
		return nil, ErrIndexNotFound
	}
	
	return stats, nil
}

func (m *MockVectorProvider) HealthCheck(ctx context.Context) error {
	if m.closed {
		return ErrConnectionClosed
	}
	return nil
}

func (m *MockVectorProvider) Close() error {
	m.closed = true
	return nil
}

// 测试用例
func TestVectorProvider(t *testing.T) {
	ctx := context.Background()
	provider := NewMockVectorProvider()
	defer provider.Close()

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
		
		// 测试重复创建
		err = provider.CreateIndex(ctx, req)
		assert.Equal(t, ErrIndexAlreadyExists, err)
	})

	// 测试索引存在性检查
	t.Run("IndexExists", func(t *testing.T) {
		exists, err := provider.IndexExists(ctx, "test_index")
		require.NoError(t, err)
		assert.True(t, exists)
		
		exists, err = provider.IndexExists(ctx, "nonexistent_index")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	// 测试插入向量
	t.Run("InsertVectors", func(t *testing.T) {
		vectors := []Vector{
			{
				ID:     "vec1",
				Values: []float32{1.0, 2.0, 3.0},
				Metadata: map[string]interface{}{
					"category": "test",
				},
			},
			{
				ID:     "vec2",
				Values: []float32{4.0, 5.0, 6.0},
				Metadata: map[string]interface{}{
					"category": "test",
				},
			},
		}
		
		err := provider.InsertVectors(ctx, "test_index", vectors)
		require.NoError(t, err)
		
		// 验证统计信息
		stats, err := provider.GetStats(ctx, "test_index")
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.VectorCount)
	})

	// 测试向量搜索
	t.Run("Search", func(t *testing.T) {
		searchReq := &SearchRequest{
			Vector: []float32{1.0, 2.0, 3.0},
			TopK:   10,
		}
		
		results, err := provider.Search(ctx, "test_index", searchReq)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		
		// 第一个结果应该是完全匹配的向量
		assert.Equal(t, "vec1", results[0].ID)
		assert.Equal(t, float32(1.0), results[0].Score)
	})

	// 测试获取向量
	t.Run("GetVector", func(t *testing.T) {
		vector, err := provider.GetVector(ctx, "test_index", "vec1")
		require.NoError(t, err)
		assert.Equal(t, "vec1", vector.ID)
		assert.Equal(t, []float32{1.0, 2.0, 3.0}, vector.Values)
	})

	// 测试删除向量
	t.Run("DeleteVectors", func(t *testing.T) {
		err := provider.DeleteVectors(ctx, "test_index", []string{"vec1"})
		require.NoError(t, err)
		
		// 验证向量已删除
		_, err = provider.GetVector(ctx, "test_index", "vec1")
		assert.Equal(t, ErrVectorNotFound, err)
		
		// 验证统计信息更新
		stats, err := provider.GetStats(ctx, "test_index")
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.VectorCount)
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

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		err := provider.HealthCheck(ctx)
		assert.NoError(t, err)
		
		// 关闭连接后健康检查应该失败
		provider.Close()
		err = provider.HealthCheck(ctx)
		assert.Equal(t, ErrConnectionClosed, err)
	})
}