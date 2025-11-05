package service

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
)

// ProtectedAIService AI 服务的熔断器保护包装
// 为 AI 服务调用提供熔断保护
type ProtectedAIService struct {
	aiService      AIService
	circuitBreaker *CircuitBreaker
	degradation    DegradationService
	log            logger.Logger
}

// NewProtectedAIService 创建受保护的 AI 服务
func NewProtectedAIService(
	aiService AIService,
	cbManager *CircuitBreakerManager,
	degradation DegradationService,
	log logger.Logger,
) *ProtectedAIService {
	// 为 AI 服务创建专用的熔断器
	config := CircuitBreakerConfig{
		MaxFailures:              5,
		Timeout:                  30 * time.Second,
		HalfOpenMaxRequests:      3,
		HalfOpenSuccessThreshold: 2,
		ResetTimeout:             60 * time.Second,
	}

	return &ProtectedAIService{
		aiService:      aiService,
		circuitBreaker: cbManager.GetOrCreate("ai-service", config),
		degradation:    degradation,
		log:            log,
	}
}

// Generate 生成 AI 响应（带熔断保护）
func (s *ProtectedAIService) Generate(ctx context.Context, sessionID, userQuery string) (string, error) {
	var response string
	var err error

	// 使用熔断器执行 AI 服务调用
	executeErr := s.circuitBreaker.Execute(ctx, func() error {
		response, err = s.aiService.Generate(ctx, sessionID, userQuery)
		return err
	})

	// 如果熔断器打开或服务调用失败，执行降级策略
	if executeErr != nil {
		s.log.WarnContext(ctx, "AI 服务调用失败，执行降级策略", logger.Fields{
			"session_id": sessionID,
			"error":      executeErr.Error(),
			"cb_state":   s.circuitBreaker.GetState().String(),
		})

		// 执行降级
		degradationResult, degradationErr := s.degradation.DegradeAIService(ctx, sessionID, userQuery)
		if degradationErr != nil {
			return "", fmt.Errorf("AI 服务和降级策略均失败: %w", degradationErr)
		}

		return degradationResult.Response, nil
	}

	return response, nil
}

// ProtectedVectorService 向量服务的熔断器保护包装
// 为向量检索提供熔断保护
type ProtectedVectorService struct {
	vectorService  VectorService
	circuitBreaker *CircuitBreaker
	degradation    DegradationService
	log            logger.Logger
}

// NewProtectedVectorService 创建受保护的向量服务
func NewProtectedVectorService(
	vectorService VectorService,
	cbManager *CircuitBreakerManager,
	degradation DegradationService,
	log logger.Logger,
) *ProtectedVectorService {
	// 为向量服务创建专用的熔断器
	config := CircuitBreakerConfig{
		MaxFailures:              3,
		Timeout:                  20 * time.Second,
		HalfOpenMaxRequests:      2,
		HalfOpenSuccessThreshold: 2,
		ResetTimeout:             45 * time.Second,
	}

	return &ProtectedVectorService{
		vectorService:  vectorService,
		circuitBreaker: cbManager.GetOrCreate("vector-service", config),
		degradation:    degradation,
		log:            log,
	}
}

// SearchMemories 搜索记忆（带熔断保护）
func (s *ProtectedVectorService) SearchMemories(
	ctx context.Context,
	sessionID string,
	query string,
	topK int,
) ([]*MemorySearchResult, error) {
	var results []*MemorySearchResult
	var err error

	// 使用熔断器执行向量检索
	executeErr := s.circuitBreaker.Execute(ctx, func() error {
		results, err = s.vectorService.SearchMemories(ctx, sessionID, query, topK)
		return err
	})

	// 如果熔断器打开或服务调用失败，执行降级策略
	if executeErr != nil {
		s.log.WarnContext(ctx, "向量检索失败，执行降级策略", logger.Fields{
			"session_id": sessionID,
			"error":      executeErr.Error(),
			"cb_state":   s.circuitBreaker.GetState().String(),
		})

		// 执行降级
		degradationResult, degradationErr := s.degradation.DegradeVectorSearch(ctx, sessionID, query, topK)
		if degradationErr != nil {
			// 降级也失败，返回空结果
			s.log.ErrorContext(ctx, "向量检索降级失败", logger.Fields{
				"session_id": sessionID,
				"error":      degradationErr.Error(),
			})
			return []*MemorySearchResult{}, nil
		}

		// 转换降级结果
		degradedResults := make([]*MemorySearchResult, len(degradationResult.Memories))
		for i, memory := range degradationResult.Memories {
			degradedResults[i] = &MemorySearchResult{
				Memory:     memory,
				Similarity: 0.5, // 降级结果使用默认相似度
				Score:      0.5,
			}
		}

		return degradedResults, nil
	}

	return results, nil
}

// ProtectedSummaryService 摘要服务的熔断器保护包装
// 为摘要生成提供熔断保护
type ProtectedSummaryService struct {
	summaryService SummaryService
	circuitBreaker *CircuitBreaker
	degradation    DegradationService
	log            logger.Logger
}

// NewProtectedSummaryService 创建受保护的摘要服务
func NewProtectedSummaryService(
	summaryService SummaryService,
	cbManager *CircuitBreakerManager,
	degradation DegradationService,
	log logger.Logger,
) *ProtectedSummaryService {
	// 为摘要服务创建专用的熔断器
	config := CircuitBreakerConfig{
		MaxFailures:              4,
		Timeout:                  25 * time.Second,
		HalfOpenMaxRequests:      2,
		HalfOpenSuccessThreshold: 2,
		ResetTimeout:             50 * time.Second,
	}

	return &ProtectedSummaryService{
		summaryService: summaryService,
		circuitBreaker: cbManager.GetOrCreate("summary-service", config),
		degradation:    degradation,
		log:            log,
	}
}

// GenerateSummary 生成摘要（带熔断保护）
func (s *ProtectedSummaryService) GenerateSummary(
	ctx context.Context,
	req GenerateSummaryRequest,
) (*GenerateSummaryResult, error) {
	var result *GenerateSummaryResult
	var err error

	// 使用熔断器执行摘要生成
	executeErr := s.circuitBreaker.Execute(ctx, func() error {
		result, err = s.summaryService.GenerateSummary(ctx, req)
		return err
	})

	// 如果熔断器打开或服务调用失败，执行降级策略
	if executeErr != nil {
		s.log.WarnContext(ctx, "摘要生成失败，执行降级策略", logger.Fields{
			"session_id": req.SessionID,
			"error":      executeErr.Error(),
			"cb_state":   s.circuitBreaker.GetState().String(),
		})

		// 执行降级
		degradationResult, degradationErr := s.degradation.DegradeSummaryGeneration(
			ctx,
			req.Messages,
			req.TargetLength,
		)
		if degradationErr != nil {
			return nil, fmt.Errorf("摘要生成和降级策略均失败: %w", degradationErr)
		}

		// 转换降级结果
		return &GenerateSummaryResult{
			Summary:         degradationResult.Summary,
			TokenCount:      degradationResult.SummaryLength,
			MessageCount:    len(req.Messages),
			QualityScore:    0.5, // 降级结果使用默认质量分数
			CompressionRate: float64(degradationResult.SummaryLength) / float64(degradationResult.OriginalLength),
			Method:          degradationResult.Method,
		}, nil
	}

	return result, nil
}

// CircuitBreakerMiddleware 熔断器中间件
// 可以用于包装任何服务调用
type CircuitBreakerMiddleware struct {
	manager *CircuitBreakerManager
	log     logger.Logger
}

// NewCircuitBreakerMiddleware 创建熔断器中间件
func NewCircuitBreakerMiddleware(manager *CircuitBreakerManager, log logger.Logger) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		manager: manager,
		log:     log,
	}
}

// Wrap 包装函数调用
func (m *CircuitBreakerMiddleware) Wrap(
	ctx context.Context,
	serviceName string,
	config CircuitBreakerConfig,
	fn func() error,
) error {
	breaker := m.manager.GetOrCreate(serviceName, config)
	return breaker.Execute(ctx, fn)
}

// WrapWithFallback 包装函数调用并提供降级函数
func (m *CircuitBreakerMiddleware) WrapWithFallback(
	ctx context.Context,
	serviceName string,
	config CircuitBreakerConfig,
	fn func() error,
	fallback func() error,
) error {
	breaker := m.manager.GetOrCreate(serviceName, config)
	
	err := breaker.Execute(ctx, fn)
	if err != nil {
		m.log.WarnContext(ctx, "服务调用失败，执行降级", logger.Fields{
			"service":  serviceName,
			"error":    err.Error(),
			"cb_state": breaker.GetState().String(),
		})

		// 执行降级函数
		if fallback != nil {
			return fallback()
		}
	}

	return err
}

// 示例接口定义（用于演示）

// AIService AI 服务接口
type AIService interface {
	Generate(ctx context.Context, sessionID, userQuery string) (string, error)
}

// VectorService 向量服务接口
type VectorService interface {
	SearchMemories(ctx context.Context, sessionID, query string, topK int) ([]*MemorySearchResult, error)
}

// SummaryService 摘要服务接口
type SummaryService interface {
	GenerateSummary(ctx context.Context, req GenerateSummaryRequest) (*GenerateSummaryResult, error)
}

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	SessionID    string
	Messages     []*ConversationMessage
	TargetLength int
}

// GenerateSummaryResult 生成摘要结果
type GenerateSummaryResult struct {
	Summary         string
	TokenCount      int
	MessageCount    int
	QualityScore    float64
	CompressionRate float64
	Method          string
}

// ConversationMessage 对话消息（简化版）
type ConversationMessage struct {
	Role    string
	Content string
}

// MemorySearchResult 记忆搜索结果（简化版）
type MemorySearchResult struct {
	Memory     interface{}
	Similarity float32
	Score      float32
}
