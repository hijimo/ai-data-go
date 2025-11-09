// internal/storage/qdrant_client.go
package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// QdrantClient Qdrant 向量数据库客户端接口
type QdrantClient interface {
	// InitializeCollection 初始化单个共享 Collection（仅在启动时调用一次）
	// Collection 名称：conversation_memories
	// 向量维度：1536（text-embedding-ada-002）
	// 距离度量：Cosine
	InitializeCollection(ctx context.Context) error

	// UpsertVector 插入或更新向量
	// Payload 包含 tenant_id 字段（设置 is_tenant=true 索引）
	// Payload 包含 session_id 字段
	// Payload 包含 memory_type 字段
	// Payload 包含其他元数据
	UpsertVector(ctx context.Context, req *UpsertVectorRequest) error

	// BatchUpsertVectors 批量插入或更新向量（优化性能）
	BatchUpsertVectors(ctx context.Context, reqs []*UpsertVectorRequest) error

	// SearchVectors 向量检索
	// 支持按 tenant_id 过滤（必须）
	// 支持按 session_id 过滤（可选）
	// 支持按 memory_type 过滤（可选）
	SearchVectors(ctx context.Context, req *SearchVectorRequest) ([]*VectorSearchResult, error)

	// DeleteVector 删除向量
	DeleteVector(ctx context.Context, tenantID, memoryID uuid.UUID) error

	// DeleteByFilter 按条件批量删除
	DeleteByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error

	// UpdatePayload 更新 payload（验证租户权限）
	UpdatePayload(ctx context.Context, tenantID, memoryID uuid.UUID, payload map[string]interface{}) error

	// OptimizeCollection 优化 Collection（compact、reindex）
	OptimizeCollection(ctx context.Context) error

	// GetCollectionInfo 获取 Collection 信息（用于监控）
	GetCollectionInfo(ctx context.Context) (*CollectionInfo, error)

	// UpdateCollectionConfig 更新 Collection 配置（分片、副本等）
	UpdateCollectionConfig(ctx context.Context, config *CollectionConfig) error

	// Close 关闭客户端连接
	Close() error
}

// UpsertVectorRequest 插入或更新向量的请求
type UpsertVectorRequest struct {
	TenantID   uuid.UUID              // 租户ID（必须）
	MemoryID   uuid.UUID              // 记忆ID（必须）
	SessionID  uuid.UUID              // 会话ID（必须）
	MemoryType string                 // 记忆类型：short_term, long_term, summary
	Vector     []float32              // 向量数据（1536维）
	Importance float32                // 重要性评分（0-1）
	ExpiresAt  *time.Time             // 过期时间（可选）
	Metadata   map[string]interface{} // 其他元数据（可选）
}

// SearchVectorRequest 向量检索请求
type SearchVectorRequest struct {
	TenantID    uuid.UUID  // 租户ID（必须，用于租户隔离）
	SessionID   *uuid.UUID // 会话ID（可选，用于会话内搜索）
	QueryVector []float32  // 查询向量（1536维）
	TopK        int        // 返回结果数量
	MinScore    float32    // 最小相似度分数（0-1）
	MemoryType  *string    // 记忆类型过滤（可选）
	TimeRange   *TimeRange // 时间范围过滤（可选）
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time // 开始时间
	End   time.Time // 结束时间
}

// VectorSearchResult 向量检索结果
type VectorSearchResult struct {
	MemoryID   uuid.UUID              // 记忆ID
	Score      float32                // 相似度分数（0-1）
	Payload    map[string]interface{} // Payload数据
	TenantID   uuid.UUID              // 租户ID
	SessionID  uuid.UUID              // 会话ID
	MemoryType string                 // 记忆类型
	Importance float32                // 重要性评分
	CreatedAt  time.Time              // 创建时间
	ExpiresAt  *time.Time             // 过期时间
}

// QdrantConfig Qdrant 配置
type QdrantConfig struct {
	// 方式1：使用完整的 Endpoint URL（推荐用于 Qdrant Cloud）
	Endpoint string // 完整的 Qdrant 端点 URL，例如：https://xxx.cloud.qdrant.io
	
	// 方式2：使用 Host + Port（用于自托管）
	Host   string // Qdrant 服务器地址
	Port   int    // Qdrant 服务器端口
	UseTLS bool   // 是否使用 TLS（仅用于方式2）
	
	// 通用配置
	APIKey    string // API Key / Access Token（必需）
	ClusterID string // 集群ID（可选，用于日志记录）
	
	// 性能优化配置
	ShardNumber       int  // 分片数量（默认：4，根据租户数量调整）
	ReplicationFactor int  // 副本数量（默认：2，高可用配置）
	HnswM             int  // HNSW 参数 m（默认：16，控制图的连接度）
	HnswEfConstruction int // HNSW 参数 ef_construction（默认：100，构建时的搜索深度）
	EnableOptimization bool // 是否启用定期优化（默认：true）
	OptimizationInterval int // 优化间隔（小时，默认：24）
}

// CollectionConfig Collection 配置
type CollectionConfig struct {
	ShardNumber       int // 分片数量
	ReplicationFactor int // 副本数量
	HnswM             int // HNSW 参数 m
	HnswEfConstruction int // HNSW 参数 ef_construction
}

// CollectionInfo Collection 信息
type CollectionInfo struct {
	Status            string // Collection 状态
	VectorsCount      int64  // 向量数量
	IndexedVectorsCount int64 // 已索引向量数量
	PointsCount       int64  // 点数量
	SegmentsCount     int    // 段数量
	Config            *CollectionConfig // 配置信息
}

// Collection 名称常量
const (
	CollectionName = "conversation_memories" // 共享 Collection 名称
	VectorDim      = 1536                    // 向量维度（text-embedding-ada-002）
	
	// 默认配置值
	DefaultShardNumber       = 4   // 默认分片数量
	DefaultReplicationFactor = 2   // 默认副本数量
	DefaultHnswM             = 16  // 默认 HNSW m 参数
	DefaultHnswEfConstruction = 100 // 默认 HNSW ef_construction 参数
	DefaultOptimizationInterval = 24 // 默认优化间隔（小时）
)
