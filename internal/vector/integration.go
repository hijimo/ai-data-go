package vector

import (
	"context"
	"fmt"
)

// VectorService 向量服务，集成向量化和向量存储
type VectorService struct {
	vectorManager    *Manager
	embeddingManager *EmbeddingManager
	utils            *VectorUtils
}

// NewVectorService 创建向量服务
func NewVectorService(vectorManager *Manager, embeddingManager *EmbeddingManager) *VectorService {
	return &VectorService{
		vectorManager:    vectorManager,
		embeddingManager: embeddingManager,
		utils:            NewVectorUtils(),
	}
}

// IndexAndStore 向量化文本并存储到向量数据库
func (s *VectorService) IndexAndStore(ctx context.Context, 
	vectorProviderName, embeddingProviderName, indexName string, 
	documents []Document) error {
	
	// 获取向量提供商
	vectorProvider, err := s.vectorManager.GetProvider(vectorProviderName)
	if err != nil {
		return fmt.Errorf("failed to get vector provider: %w", err)
	}
	
	// 获取向量化提供商
	embeddingProvider, err := s.embeddingManager.GetProvider(embeddingProviderName)
	if err != nil {
		return fmt.Errorf("failed to get embedding provider: %w", err)
	}
	
	// 检查索引是否存在，不存在则创建
	exists, err := vectorProvider.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	
	if !exists {
		// 创建索引
		createReq := &CreateIndexRequest{
			Name:            indexName,
			Dimension:       embeddingProvider.GetDimension(),
			DistanceMeasure: string(DistanceCosine),
			IndexType:       string(IndexHNSW),
		}
		
		if err := vectorProvider.CreateIndex(ctx, createReq); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	// 准备文本列表
	var texts []string
	var docIDs []string
	var metadata []map[string]interface{}
	
	for _, doc := range documents {
		texts = append(texts, doc.Content)
		docIDs = append(docIDs, doc.ID)
		metadata = append(metadata, doc.Metadata)
	}
	
	// 批量向量化
	embeddings, err := embeddingProvider.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	
	// 准备向量数据
	var vectors []Vector
	for i, embedding := range embeddings {
		vectors = append(vectors, Vector{
			ID:       docIDs[i],
			Values:   embedding,
			Metadata: metadata[i],
		})
	}
	
	// 存储向量
	if err := vectorProvider.InsertVectors(ctx, indexName, vectors); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}
	
	return nil
}

// SearchSimilar 搜索相似文档
func (s *VectorService) SearchSimilar(ctx context.Context, 
	vectorProviderName, embeddingProviderName, indexName, queryText string, 
	topK int, filters map[string]interface{}) ([]SearchResult, error) {
	
	// 获取向量提供商
	vectorProvider, err := s.vectorManager.GetProvider(vectorProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vector provider: %w", err)
	}
	
	// 获取向量化提供商
	embeddingProvider, err := s.embeddingManager.GetProvider(embeddingProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding provider: %w", err)
	}
	
	// 向量化查询文本
	queryVector, err := embeddingProvider.Embed(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query text: %w", err)
	}
	
	// 执行向量搜索
	searchReq := &SearchRequest{
		Vector:  queryVector,
		TopK:    topK,
		Filters: filters,
	}
	
	results, err := vectorProvider.Search(ctx, indexName, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}
	
	return results, nil
}

// UpdateDocument 更新文档向量
func (s *VectorService) UpdateDocument(ctx context.Context, 
	vectorProviderName, embeddingProviderName, indexName string, 
	doc Document) error {
	
	// 获取向量提供商
	vectorProvider, err := s.vectorManager.GetProvider(vectorProviderName)
	if err != nil {
		return fmt.Errorf("failed to get vector provider: %w", err)
	}
	
	// 获取向量化提供商
	embeddingProvider, err := s.embeddingManager.GetProvider(embeddingProviderName)
	if err != nil {
		return fmt.Errorf("failed to get embedding provider: %w", err)
	}
	
	// 向量化文档内容
	embedding, err := embeddingProvider.Embed(ctx, doc.Content)
	if err != nil {
		return fmt.Errorf("failed to embed document: %w", err)
	}
	
	// 更新向量
	vector := Vector{
		ID:       doc.ID,
		Values:   embedding,
		Metadata: doc.Metadata,
	}
	
	if err := vectorProvider.UpdateVectors(ctx, indexName, []Vector{vector}); err != nil {
		return fmt.Errorf("failed to update vector: %w", err)
	}
	
	return nil
}

// DeleteDocument 删除文档向量
func (s *VectorService) DeleteDocument(ctx context.Context, 
	vectorProviderName, indexName, docID string) error {
	
	// 获取向量提供商
	vectorProvider, err := s.vectorManager.GetProvider(vectorProviderName)
	if err != nil {
		return fmt.Errorf("failed to get vector provider: %w", err)
	}
	
	// 删除向量
	if err := vectorProvider.DeleteVectors(ctx, indexName, []string{docID}); err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}
	
	return nil
}

// GetIndexStats 获取索引统计信息
func (s *VectorService) GetIndexStats(ctx context.Context, 
	vectorProviderName, indexName string) (*IndexStats, error) {
	
	// 获取向量提供商
	vectorProvider, err := s.vectorManager.GetProvider(vectorProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vector provider: %w", err)
	}
	
	// 获取统计信息
	stats, err := vectorProvider.GetStats(ctx, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}
	
	return stats, nil
}

// Document 文档结构
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// BatchIndexAndStore 批量处理大量文档
func (s *VectorService) BatchIndexAndStore(ctx context.Context, 
	vectorProviderName, embeddingProviderName, indexName string, 
	documents []Document, batchSize int) error {
	
	if batchSize <= 0 {
		batchSize = 100 // 默认批次大小
	}
	
	// 分批处理
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}
		
		batch := documents[i:end]
		if err := s.IndexAndStore(ctx, vectorProviderName, embeddingProviderName, indexName, batch); err != nil {
			return fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
		}
	}
	
	return nil
}

// HealthCheck 检查所有服务的健康状态
func (s *VectorService) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	
	// 检查向量存储提供商
	vectorResults := s.vectorManager.HealthCheck(ctx)
	for name, err := range vectorResults {
		results[fmt.Sprintf("vector_%s", name)] = err
	}
	
	// 检查向量化提供商
	embeddingResults := s.embeddingManager.HealthCheck(ctx)
	for name, err := range embeddingResults {
		results[fmt.Sprintf("embedding_%s", name)] = err
	}
	
	return results
}