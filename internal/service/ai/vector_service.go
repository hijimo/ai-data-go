// internal/service/ai/vector_service.go
package ai

import (
	"context"
)

// VectorService 向量嵌入服务接口
type VectorService interface {
	// GenerateEmbedding 生成文本向量
	// 参数:
	//   - ctx: 上下文
	//   - text: 要生成向量的文本
	// 返回:
	//   - []float32: 1536维向量
	//   - error: 错误信息
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateBatchEmbeddings 批量生成向量
	// 参数:
	//   - ctx: 上下文
	//   - texts: 要生成向量的文本列表
	// 返回:
	//   - [][]float32: 向量列表，每个向量1536维
	//   - error: 错误信息
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Text  string   // 单个文本
	Texts []string // 批量文本
}

// EmbeddingResult 嵌入结果
type EmbeddingResult struct {
	Embedding  []float32   // 单个向量
	Embeddings [][]float32 // 批量向量
	Model      string      // 使用的模型
	Usage      *Usage      // Token使用情况
}

// Usage Token使用情况
type Usage struct {
	PromptTokens int // 输入Token数
	TotalTokens  int // 总Token数
}
