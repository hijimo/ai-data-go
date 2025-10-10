package vector

import (
	"context"
	"time"
)

// Vector 表示一个向量及其元数据
type Vector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchResult 表示向量检索结果
type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Values   []float32              `json:"values,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// IndexStats 表示向量索引统计信息
type IndexStats struct {
	Name        string    `json:"name"`
	Dimension   int       `json:"dimension"`
	VectorCount int64     `json:"vector_count"`
	IndexSize   int64     `json:"index_size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateIndexRequest 创建索引请求
type CreateIndexRequest struct {
	Name           string            `json:"name"`
	Dimension      int               `json:"dimension"`
	DistanceMeasure string           `json:"distance_measure"` // cosine, euclidean, dot_product
	IndexType      string            `json:"index_type"`       // hnsw, ivf
	Parameters     map[string]interface{} `json:"parameters"`
}

// SearchRequest 向量检索请求
type SearchRequest struct {
	Vector  []float32              `json:"vector"`
	TopK    int                    `json:"top_k"`
	Filters map[string]interface{} `json:"filters"`
}

// VectorProvider 定义向量存储的统一接口
type VectorProvider interface {
	// 创建向量索引
	CreateIndex(ctx context.Context, req *CreateIndexRequest) error
	
	// 删除向量索引
	DeleteIndex(ctx context.Context, indexName string) error
	
	// 检查索引是否存在
	IndexExists(ctx context.Context, indexName string) (bool, error)
	
	// 插入向量
	InsertVectors(ctx context.Context, indexName string, vectors []Vector) error
	
	// 批量插入向量
	BatchInsertVectors(ctx context.Context, indexName string, vectors []Vector, batchSize int) error
	
	// 相似度检索
	Search(ctx context.Context, indexName string, req *SearchRequest) ([]SearchResult, error)
	
	// 删除向量
	DeleteVectors(ctx context.Context, indexName string, ids []string) error
	
	// 更新向量
	UpdateVectors(ctx context.Context, indexName string, vectors []Vector) error
	
	// 获取向量
	GetVector(ctx context.Context, indexName string, id string) (*Vector, error)
	
	// 获取统计信息
	GetStats(ctx context.Context, indexName string) (*IndexStats, error)
	
	// 健康检查
	HealthCheck(ctx context.Context) error
	
	// 关闭连接
	Close() error
}