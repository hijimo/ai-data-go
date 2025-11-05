package service

import (
	"context"
	"fmt"

	"genkit-ai-service/internal/logger"
	"github.com/google/uuid"
)

// VectorService 向量服务接口
// 提供文本向量化、存储和检索能力，支持多租户隔离
type VectorService interface {
	// GenerateEmbedding 生成单个文本的向量
	// 参数:
	//   ctx: 上下文
	//   text: 要向量化的文本
	// 返回:
	//   []float32: 生成的向量
	//   error: 错误信息
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateEmbeddings 批量生成文本向量
	// 参数:
	//   ctx: 上下文
	//   texts: 要向量化的文本列表
	// 返回:
	//   [][]float32: 生成的向量列表
	//   error: 错误信息
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// StoreVector 存储向量到 Qdrant
	// 参数:
	//   ctx: 上下文
	//   req: 存储向量请求
	// 返回:
	//   error: 错误信息
	StoreVector(ctx context.Context, req *StoreVectorRequest) error

	// StoreVectors 批量存储向量到 Qdrant
	// 参数:
	//   ctx: 上下文
	//   reqs: 存储向量请求列表
	// 返回:
	//   error: 错误信息
	StoreVectors(ctx context.Context, reqs []*StoreVectorRequest) error

	// SearchVectors 向量相似度搜索（支持多租户隔离）
	// 参数:
	//   ctx: 上下文
	//   req: 搜索请求
	// 返回:
	//   []*VectorSearchResult: 搜索结果列表
	//   error: 错误信息
	SearchVectors(ctx context.Context, req *SearchVectorRequest) ([]*VectorSearchResult, error)

	// DeleteVector 删除向量
	// 参数:
	//   ctx: 上下文
	//   tenantID: 租户ID
	//   pointID: 点ID
	// 返回:
	//   error: 错误信息
	DeleteVector(ctx context.Context, tenantID uuid.UUID, pointID string) error

	// DeleteVectorsByFilter 根据过滤条件删除向量
	// 参数:
	//   ctx: 上下文
	//   tenantID: 租户ID
	//   filter: 过滤条件
	// 返回:
	//   error: 错误信息
	DeleteVectorsByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error

	// GetEmbeddingDimension 获取向量维度
	// 返回:
	//   int: 向量维度
	GetEmbeddingDimension() int

	// EnsureCollection 确保集合存在
	// 参数:
	//   ctx: 上下文
	// 返回:
	//   error: 错误信息
	EnsureCollection(ctx context.Context) error
}

// StoreVectorRequest 存储向量请求
type StoreVectorRequest struct {
	// PointID 点ID（唯一标识）
	PointID string
	// TenantID 租户ID（用于多租户隔离）
	TenantID uuid.UUID
	// SessionID 会话ID
	SessionID uuid.UUID
	// Content 文本内容
	Content string
	// Vector 向量数据
	Vector []float32
	// Metadata 元数据
	Metadata map[string]interface{}
}

// SearchVectorRequest 搜索向量请求
type SearchVectorRequest struct {
	// TenantID 租户ID（用于多租户隔离）
	TenantID uuid.UUID
	// SessionID 会话ID（可选，用于限制在特定会话内搜索）
	SessionID *uuid.UUID
	// QueryVector 查询向量
	QueryVector []float32
	// QueryText 查询文本（如果提供，将自动生成向量）
	QueryText string
	// Limit 返回结果数量
	Limit int
	// ScoreThreshold 相似度阈值（0-1）
	ScoreThreshold float32
	// Filter 额外的过滤条件
	Filter map[string]interface{}
}

// VectorSearchResult 向量搜索结果
type VectorSearchResult struct {
	// PointID 点ID
	PointID string
	// Score 相似度分数
	Score float32
	// Content 文本内容
	Content string
	// Metadata 元数据
	Metadata map[string]interface{}
}

// EmbeddingProvider 嵌入模型提供商类型
type EmbeddingProvider string

const (
	// EmbeddingProviderGoogleAI Google AI 嵌入模型
	EmbeddingProviderGoogleAI EmbeddingProvider = "google"
)

// VectorServiceConfig 向量服务配置
type VectorServiceConfig struct {
	// Provider 嵌入模型提供商
	Provider EmbeddingProvider
	// APIKey API 密钥（用于嵌入模型）
	APIKey string
	// Model 模型名称
	Model string
	// Dimension 向量维度
	Dimension int
	// BatchSize 批量处理大小
	BatchSize int
	// MaxRetries 最大重试次数
	MaxRetries int
	// QdrantEndpoint Qdrant 端点
	QdrantEndpoint string
	// QdrantAPIKey Qdrant API 密钥
	QdrantAPIKey string
	// CollectionName 集合名称
	CollectionName string
}


// NewVectorService 创建向量服务
// 根据配置的提供商类型创建相应的向量服务实例
// 参数:
//   config: 向量服务配置
//   log: 日志记录器
// 返回:
//   VectorService: 向量服务实例
//   error: 错误信息
func NewVectorService(config *VectorServiceConfig, log logger.Logger) (VectorService, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	switch config.Provider {
	case EmbeddingProviderGoogleAI:
		return NewGoogleAIVectorService(config, log)
	default:
		return nil, fmt.Errorf("不支持的嵌入模型提供商: %s (当前仅支持 'google')", config.Provider)
	}
}
