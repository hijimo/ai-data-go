// internal/service/ai/vector_service_impl.go
package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// vectorServiceImpl 向量嵌入服务实现
type vectorServiceImpl struct {
	genkit    *genkit.Genkit
	embedder  ai.Embedder
	config    *VectorServiceConfig
	retryConf *RetryConfig
}

// VectorServiceConfig 向量服务配置
type VectorServiceConfig struct {
	// 嵌入模型名称（例如：text-embedding-004）
	EmbedderModel string
	// 向量维度（默认：1536）
	VectorDim int
	// 批量处理大小（默认：10）
	BatchSize int
	// Google AI API Key
	APIKey string
}

// RetryConfig 重试配置
type RetryConfig struct {
	// 最大重试次数
	MaxRetries int
	// 初始重试延迟
	InitialDelay time.Duration
	// 最大重试延迟
	MaxDelay time.Duration
	// 退避倍数
	Multiplier float64
}

// NewVectorService 创建新的向量嵌入服务
func NewVectorService(config *VectorServiceConfig) (VectorService, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 设置默认值
	if config.EmbedderModel == "" {
		config.EmbedderModel = "text-embedding-004"
	}
	if config.VectorDim == 0 {
		config.VectorDim = 1536
	}
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API密钥不能为空")
	}

	// 初始化 Genkit
	ctx := context.Background()
	g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: config.APIKey,
		}),
	)

	// 创建Google AI嵌入器
	// 使用完整的模型名称，例如 "text-embedding-004"
	embedder := googlegenai.GoogleAIEmbedder(g, config.EmbedderModel)

	// 默认重试配置
	retryConf := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	return &vectorServiceImpl{
		genkit:    g,
		embedder:  embedder,
		config:    config,
		retryConf: retryConf,
	}, nil
}

// GenerateEmbedding 生成文本向量
func (s *vectorServiceImpl) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("文本不能为空")
	}

	// 使用重试机制
	var embedding []float32
	err := s.retryWithBackoff(ctx, func() error {
		// 构建嵌入请求
		req := &ai.EmbedRequest{
			Input: []*ai.Document{
				ai.DocumentFromText(text, nil),
			},
		}

		// 调用embedder
		resp, err := s.embedder.Embed(ctx, req)
		if err != nil {
			return fmt.Errorf("生成向量失败: %w", err)
		}

		// 检查响应
		if len(resp.Embeddings) == 0 {
			return fmt.Errorf("未返回向量数据")
		}

		// 提取向量
		embedding = resp.Embeddings[0].Embedding

		// 验证向量维度
		if len(embedding) != s.config.VectorDim {
			return fmt.Errorf("向量维度不匹配: 期望 %d, 实际 %d", s.config.VectorDim, len(embedding))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return embedding, nil
}

// GenerateBatchEmbeddings 批量生成向量
func (s *vectorServiceImpl) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("文本列表不能为空")
	}

	// 验证文本内容
	for i, text := range texts {
		if text == "" {
			return nil, fmt.Errorf("文本[%d]不能为空", i)
		}
	}

	// 分批处理
	var allEmbeddings [][]float32
	batchSize := s.config.BatchSize

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]

		// 使用重试机制处理每个批次
		var batchEmbeddings [][]float32
		err := s.retryWithBackoff(ctx, func() error {
			// 构建文档列表
			documents := make([]*ai.Document, len(batch))
			for j, text := range batch {
				documents[j] = ai.DocumentFromText(text, nil)
			}

			// 构建嵌入请求
			req := &ai.EmbedRequest{
				Input: documents,
			}

			// 调用embedder
			resp, err := s.embedder.Embed(ctx, req)
			if err != nil {
				return fmt.Errorf("批量生成向量失败: %w", err)
			}

			// 检查响应
			if len(resp.Embeddings) != len(batch) {
				return fmt.Errorf("返回的向量数量不匹配: 期望 %d, 实际 %d", len(batch), len(resp.Embeddings))
			}

			// 提取向量
			batchEmbeddings = make([][]float32, len(resp.Embeddings))
			for j, emb := range resp.Embeddings {
				// 验证向量维度
				if len(emb.Embedding) != s.config.VectorDim {
					return fmt.Errorf("向量[%d]维度不匹配: 期望 %d, 实际 %d", j, s.config.VectorDim, len(emb.Embedding))
				}
				batchEmbeddings[j] = emb.Embedding
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("处理批次[%d-%d]失败: %w", i, end-1, err)
		}

		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
	}

	return allEmbeddings, nil
}

// retryWithBackoff 使用指数退避重试
func (s *vectorServiceImpl) retryWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	delay := s.retryConf.InitialDelay

	for attempt := 0; attempt <= s.retryConf.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，直接返回错误
		if attempt == s.retryConf.MaxRetries {
			break
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// 计算下一次延迟（指数退避）
		delay = time.Duration(float64(delay) * s.retryConf.Multiplier)
		if delay > s.retryConf.MaxDelay {
			delay = s.retryConf.MaxDelay
		}
	}

	return fmt.Errorf("重试%d次后仍然失败: %w", s.retryConf.MaxRetries, lastErr)
}
