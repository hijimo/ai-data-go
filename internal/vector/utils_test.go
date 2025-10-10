package vector

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectorUtils(t *testing.T) {
	utils := NewVectorUtils()

	// 测试向量归一化
	t.Run("NormalizeVector", func(t *testing.T) {
		vector := []float32{3.0, 4.0}
		normalized := utils.NormalizeVector(vector)
		
		// 验证归一化后的向量长度为1
		var norm float32
		for _, v := range normalized {
			norm += v * v
		}
		assert.InDelta(t, 1.0, math.Sqrt(float64(norm)), 0.0001)
		
		// 测试零向量
		zeroVector := []float32{0.0, 0.0}
		normalizedZero := utils.NormalizeVector(zeroVector)
		assert.Equal(t, zeroVector, normalizedZero)
	})

	// 测试余弦相似度
	t.Run("CosineSimilarity", func(t *testing.T) {
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0}
		similarity := utils.CosineSimilarity(a, b)
		assert.InDelta(t, 1.0, similarity, 0.0001)
		
		a = []float32{1.0, 0.0}
		b = []float32{0.0, 1.0}
		similarity = utils.CosineSimilarity(a, b)
		assert.InDelta(t, 0.0, similarity, 0.0001)
		
		// 测试不同长度的向量
		a = []float32{1.0, 0.0}
		b = []float32{1.0}
		similarity = utils.CosineSimilarity(a, b)
		assert.Equal(t, float32(0.0), similarity)
	})

	// 测试欧几里得距离
	t.Run("EuclideanDistance", func(t *testing.T) {
		a := []float32{0.0, 0.0}
		b := []float32{3.0, 4.0}
		distance := utils.EuclideanDistance(a, b)
		assert.InDelta(t, 5.0, distance, 0.0001)
		
		// 测试相同向量
		distance = utils.EuclideanDistance(a, a)
		assert.Equal(t, float32(0.0), distance)
		
		// 测试不同长度的向量
		c := []float32{1.0}
		distance = utils.EuclideanDistance(a, c)
		assert.True(t, math.IsInf(float64(distance), 1))
	})

	// 测试点积
	t.Run("DotProduct", func(t *testing.T) {
		a := []float32{1.0, 2.0, 3.0}
		b := []float32{4.0, 5.0, 6.0}
		product := utils.DotProduct(a, b)
		assert.Equal(t, float32(32.0), product) // 1*4 + 2*5 + 3*6 = 32
		
		// 测试不同长度的向量
		c := []float32{1.0, 2.0}
		product = utils.DotProduct(a, c)
		assert.Equal(t, float32(0.0), product)
	})

	// 测试曼哈顿距离
	t.Run("ManhattanDistance", func(t *testing.T) {
		a := []float32{1.0, 2.0}
		b := []float32{4.0, 6.0}
		distance := utils.ManhattanDistance(a, b)
		assert.Equal(t, float32(7.0), distance) // |1-4| + |2-6| = 3 + 4 = 7
		
		// 测试相同向量
		distance = utils.ManhattanDistance(a, a)
		assert.Equal(t, float32(0.0), distance)
	})

	// 测试生成向量ID
	t.Run("GenerateVectorID", func(t *testing.T) {
		content1 := "test content"
		content2 := "test content"
		content3 := "different content"
		
		id1 := utils.GenerateVectorID(content1)
		id2 := utils.GenerateVectorID(content2)
		id3 := utils.GenerateVectorID(content3)
		
		// 相同内容应该生成相同ID
		assert.Equal(t, id1, id2)
		// 不同内容应该生成不同ID
		assert.NotEqual(t, id1, id3)
		// ID应该是32位十六进制字符串
		assert.Len(t, id1, 32)
	})

	// 测试向量验证
	t.Run("ValidateVector", func(t *testing.T) {
		// 有效向量
		vector := []float32{1.0, 2.0, 3.0}
		err := utils.ValidateVector(vector, 3)
		assert.NoError(t, err)
		
		// 空向量
		emptyVector := []float32{}
		err = utils.ValidateVector(emptyVector, 0)
		assert.Error(t, err)
		
		// 维度不匹配
		err = utils.ValidateVector(vector, 5)
		assert.Error(t, err)
		
		// 包含NaN
		nanVector := []float32{1.0, float32(math.NaN()), 3.0}
		err = utils.ValidateVector(nanVector, 3)
		assert.Error(t, err)
		
		// 包含无穷大
		infVector := []float32{1.0, float32(math.Inf(1)), 3.0}
		err = utils.ValidateVector(infVector, 3)
		assert.Error(t, err)
	})

	// 测试批量处理向量
	t.Run("BatchVectors", func(t *testing.T) {
		vectors := []Vector{
			{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"},
		}
		
		batches := utils.BatchVectors(vectors, 2)
		assert.Len(t, batches, 3)
		assert.Len(t, batches[0], 2)
		assert.Len(t, batches[1], 2)
		assert.Len(t, batches[2], 1)
		
		// 测试默认批次大小
		batches = utils.BatchVectors(vectors, 0)
		assert.Len(t, batches, 1) // 默认批次大小100，所以只有一个批次
	})

	// 测试搜索结果排序
	t.Run("SortSearchResults", func(t *testing.T) {
		results := []SearchResult{
			{ID: "1", Score: 0.8},
			{ID: "2", Score: 0.9},
			{ID: "3", Score: 0.7},
		}
		
		// 降序排序
		utils.SortSearchResults(results, false)
		assert.Equal(t, "2", results[0].ID) // 最高分
		assert.Equal(t, "3", results[2].ID) // 最低分
		
		// 升序排序
		utils.SortSearchResults(results, true)
		assert.Equal(t, "3", results[0].ID) // 最低分
		assert.Equal(t, "2", results[2].ID) // 最高分
	})

	// 测试搜索结果过滤
	t.Run("FilterSearchResults", func(t *testing.T) {
		results := []SearchResult{
			{
				ID:    "1",
				Score: 0.8,
				Metadata: map[string]interface{}{
					"category": "A",
					"type":     "document",
				},
			},
			{
				ID:    "2",
				Score: 0.9,
				Metadata: map[string]interface{}{
					"category": "B",
					"type":     "document",
				},
			},
			{
				ID:    "3",
				Score: 0.7,
				Metadata: map[string]interface{}{
					"category": "A",
					"type":     "image",
				},
			},
		}
		
		// 按类别过滤
		filters := map[string]interface{}{
			"category": "A",
		}
		filtered := utils.FilterSearchResults(results, filters)
		assert.Len(t, filtered, 2)
		
		// 按多个条件过滤
		filters = map[string]interface{}{
			"category": "A",
			"type":     "document",
		}
		filtered = utils.FilterSearchResults(results, filters)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "1", filtered[0].ID)
		
		// 无过滤条件
		filtered = utils.FilterSearchResults(results, nil)
		assert.Len(t, filtered, 3)
	})

	// 测试计算索引参数
	t.Run("CalculateIndexParameters", func(t *testing.T) {
		// 小数据集
		params := utils.CalculateIndexParameters(5000, 128)
		assert.Equal(t, 16, params["m"])
		assert.Equal(t, 200, params["ef_construction"])
		
		// 中等数据集
		params = utils.CalculateIndexParameters(50000, 128)
		assert.Equal(t, 32, params["m"])
		assert.Equal(t, 400, params["ef_construction"])
		
		// 大数据集
		params = utils.CalculateIndexParameters(500000, 128)
		assert.Equal(t, 64, params["m"])
		assert.Equal(t, 800, params["ef_construction"])
		
		// 高维度数据
		params = utils.CalculateIndexParameters(50000, 1536)
		assert.Equal(t, 16, params["m"]) // m值减半
	})

	// 测试估算内存使用量
	t.Run("EstimateMemoryUsage", func(t *testing.T) {
		vectorCount := int64(10000)
		dimension := 128
		
		// HNSW索引
		memoryHNSW := utils.EstimateMemoryUsage(vectorCount, dimension, IndexHNSW)
		baseMemory := vectorCount * int64(dimension*4)
		expectedHNSW := baseMemory + (baseMemory * 75 / 100)
		assert.Equal(t, expectedHNSW, memoryHNSW)
		
		// IVF索引
		memoryIVF := utils.EstimateMemoryUsage(vectorCount, dimension, IndexIVF)
		expectedIVF := baseMemory + (baseMemory * 25 / 100)
		assert.Equal(t, expectedIVF, memoryIVF)
		
		// 平坦索引
		memoryFlat := utils.EstimateMemoryUsage(vectorCount, dimension, IndexFlat)
		assert.Equal(t, baseMemory, memoryFlat)
	})
}