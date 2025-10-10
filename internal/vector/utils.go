package vector

import (
	"crypto/md5"
	"fmt"
	"math"
	"sort"
)

// DistanceMeasure 距离度量类型
type DistanceMeasure string

const (
	DistanceCosine     DistanceMeasure = "cosine"
	DistanceEuclidean  DistanceMeasure = "euclidean"
	DistanceDotProduct DistanceMeasure = "dot_product"
	DistanceManhattan  DistanceMeasure = "manhattan"
)

// IndexType 索引类型
type IndexType string

const (
	IndexHNSW IndexType = "hnsw" // Hierarchical Navigable Small World
	IndexIVF  IndexType = "ivf"  // Inverted File
	IndexFlat IndexType = "flat" // Brute Force
)

// VectorUtils 向量工具函数
type VectorUtils struct{}

// NewVectorUtils 创建向量工具实例
func NewVectorUtils() *VectorUtils {
	return &VectorUtils{}
}

// NormalizeVector 向量归一化
func (u *VectorUtils) NormalizeVector(vector []float32) []float32 {
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	
	if norm == 0 {
		return vector
	}
	
	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = v / norm
	}
	
	return normalized
}

// CosineSimilarity 计算余弦相似度
func (u *VectorUtils) CosineSimilarity(a, b []float32) float32 {
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
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistance 计算欧几里得距离
func (u *VectorUtils) EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}
	
	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	
	return float32(math.Sqrt(float64(sum)))
}

// DotProduct 计算点积
func (u *VectorUtils) DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	var product float32
	for i := 0; i < len(a); i++ {
		product += a[i] * b[i]
	}
	
	return product
}

// ManhattanDistance 计算曼哈顿距离
func (u *VectorUtils) ManhattanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}
	
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += float32(math.Abs(float64(a[i] - b[i])))
	}
	
	return sum
}

// GenerateVectorID 生成向量ID
func (u *VectorUtils) GenerateVectorID(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// ValidateVector 验证向量
func (u *VectorUtils) ValidateVector(vector []float32, expectedDim int) error {
	if len(vector) == 0 {
		return fmt.Errorf("vector cannot be empty")
	}
	
	if expectedDim > 0 && len(vector) != expectedDim {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", expectedDim, len(vector))
	}
	
	// 检查是否包含无效值
	for i, v := range vector {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return fmt.Errorf("vector contains invalid value at index %d: %f", i, v)
		}
	}
	
	return nil
}

// BatchVectors 批量处理向量
func (u *VectorUtils) BatchVectors(vectors []Vector, batchSize int) [][]Vector {
	if batchSize <= 0 {
		batchSize = 100 // 默认批次大小
	}
	
	var batches [][]Vector
	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}
		batches = append(batches, vectors[i:end])
	}
	
	return batches
}

// SortSearchResults 对搜索结果排序
func (u *VectorUtils) SortSearchResults(results []SearchResult, ascending bool) {
	sort.Slice(results, func(i, j int) bool {
		if ascending {
			return results[i].Score < results[j].Score
		}
		return results[i].Score > results[j].Score
	})
}

// FilterSearchResults 过滤搜索结果
func (u *VectorUtils) FilterSearchResults(results []SearchResult, filters map[string]interface{}) []SearchResult {
	if len(filters) == 0 {
		return results
	}
	
	var filtered []SearchResult
	for _, result := range results {
		if u.matchesFilters(result.Metadata, filters) {
			filtered = append(filtered, result)
		}
	}
	
	return filtered
}

// matchesFilters 检查元数据是否匹配过滤条件
func (u *VectorUtils) matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
	for key, expectedValue := range filters {
		actualValue, exists := metadata[key]
		if !exists {
			return false
		}
		
		// 简单的值匹配，可以根据需要扩展更复杂的过滤逻辑
		if actualValue != expectedValue {
			return false
		}
	}
	
	return true
}

// CalculateIndexParameters 计算索引参数
func (u *VectorUtils) CalculateIndexParameters(vectorCount int64, dimension int) map[string]interface{} {
	params := make(map[string]interface{})
	
	// HNSW参数建议
	if vectorCount < 10000 {
		params["m"] = 16
		params["ef_construction"] = 200
	} else if vectorCount < 100000 {
		params["m"] = 32
		params["ef_construction"] = 400
	} else {
		params["m"] = 64
		params["ef_construction"] = 800
	}
	
	// 根据维度调整参数
	if dimension > 1000 {
		if m, ok := params["m"].(int); ok {
			params["m"] = m / 2 // 高维度时减少连接数
		}
	}
	
	return params
}

// EstimateMemoryUsage 估算内存使用量（字节）
func (u *VectorUtils) EstimateMemoryUsage(vectorCount int64, dimension int, indexType IndexType) int64 {
	vectorSize := int64(dimension * 4) // float32 = 4 bytes
	baseMemory := vectorCount * vectorSize
	
	switch indexType {
	case IndexHNSW:
		// HNSW索引大约需要额外50-100%的内存
		return baseMemory + (baseMemory * 75 / 100)
	case IndexIVF:
		// IVF索引大约需要额外20-30%的内存
		return baseMemory + (baseMemory * 25 / 100)
	case IndexFlat:
		// 平坦索引只需要存储向量本身
		return baseMemory
	default:
		return baseMemory
	}
}