package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
)

// DegradationService 降级服务接口
// 提供 AI 服务、向量检索和摘要生成的降级策略
type DegradationService interface {
	// DegradeAIService AI 服务降级
	// 当 AI 服务不可用时，尝试从缓存获取相似查询的响应，或返回默认响应
	// 参数:
	//   ctx: 上下文
	//   sessionID: 会话ID
	//   userQuery: 用户查询
	// 返回:
	//   *AIServiceDegradationResult: 降级结果
	//   error: 错误信息
	DegradeAIService(ctx context.Context, sessionID, userQuery string) (*AIServiceDegradationResult, error)

	// DegradeVectorSearch 向量检索降级
	// 当向量服务故障时，使用全文搜索作为降级方案
	// 参数:
	//   ctx: 上下文
	//   sessionID: 会话ID
	//   query: 查询文本
	//   topK: 返回结果数量
	// 返回:
	//   *VectorSearchDegradationResult: 降级结果
	//   error: 错误信息
	DegradeVectorSearch(ctx context.Context, sessionID, query string, topK int) (*VectorSearchDegradationResult, error)

	// DegradeSummaryGeneration 摘要生成降级
	// 当摘要生成失败时，使用简单截断策略
	// 参数:
	//   ctx: 上下文
	//   messages: 消息列表
	//   targetLength: 目标长度
	// 返回:
	//   *SummaryDegradationResult: 降级结果
	//   error: 错误信息
	DegradeSummaryGeneration(ctx context.Context, messages []*model.ConversationMessage, targetLength int) (*SummaryDegradationResult, error)
}

// AIServiceDegradationResult AI 服务降级结果
type AIServiceDegradationResult struct {
	// Response 降级响应内容
	Response string
	// Source 响应来源：cache（缓存）、default（默认响应）
	Source string
	// CacheHit 是否命中缓存
	CacheHit bool
	// DegradationTime 降级处理耗时（毫秒）
	DegradationTime int64
}

// VectorSearchDegradationResult 向量检索降级结果
type VectorSearchDegradationResult struct {
	// Memories 检索到的记忆列表
	Memories []*model.ConversationMemory
	// Source 检索来源：fulltext（全文搜索）、empty（空结果）
	Source string
	// FullTextUsed 是否使用了全文搜索
	FullTextUsed bool
	// DegradationTime 降级处理耗时（毫秒）
	DegradationTime int64
}

// SummaryDegradationResult 摘要降级结果
type SummaryDegradationResult struct {
	// Summary 生成的摘要
	Summary string
	// Method 使用的方法：truncate（截断）、extract（提取）
	Method string
	// OriginalLength 原始长度
	OriginalLength int
	// SummaryLength 摘要长度
	SummaryLength int
	// DegradationTime 降级处理耗时（毫秒）
	DegradationTime int64
}

// degradationServiceImpl 降级服务实现
type degradationServiceImpl struct {
	cache       CacheService
	memoryRepo  repository.GenkitMemoryRepository
	messageRepo repository.GenkitMessageRepository
	log         logger.Logger
}

// NewDegradationService 创建降级服务实例
func NewDegradationService(
	cache CacheService,
	memoryRepo repository.GenkitMemoryRepository,
	messageRepo repository.GenkitMessageRepository,
	log logger.Logger,
) DegradationService {
	return &degradationServiceImpl{
		cache:       cache,
		memoryRepo:  memoryRepo,
		messageRepo: messageRepo,
		log:         log,
	}
}

// DegradeAIService AI 服务降级
func (s *degradationServiceImpl) DegradeAIService(
	ctx context.Context,
	sessionID string,
	userQuery string,
) (*AIServiceDegradationResult, error) {
	startTime := time.Now()

	s.log.WarnContext(ctx, "AI 服务降级触发", logger.Fields{
		"session_id":   sessionID,
		"query_length": len(userQuery),
	})

	// 1. 尝试从缓存获取相似查询的响应
	cachedResponse, cacheHit := s.getCachedResponse(ctx, sessionID, userQuery)
	if cacheHit {
		degradationTime := time.Since(startTime).Milliseconds()
		
		s.log.InfoContext(ctx, "AI 服务降级：使用缓存响应", logger.Fields{
			"session_id":          sessionID,
			"cache_hit":           true,
			"degradation_time_ms": degradationTime,
		})

		return &AIServiceDegradationResult{
			Response:        cachedResponse,
			Source:          "cache",
			CacheHit:        true,
			DegradationTime: degradationTime,
		}, nil
	}

	// 2. 返回默认响应
	defaultResponse := s.getDefaultResponse(userQuery)
	degradationTime := time.Since(startTime).Milliseconds()

	s.log.InfoContext(ctx, "AI 服务降级：使用默认响应", logger.Fields{
		"session_id":          sessionID,
		"cache_hit":           false,
		"degradation_time_ms": degradationTime,
	})

	return &AIServiceDegradationResult{
		Response:        defaultResponse,
		Source:          "default",
		CacheHit:        false,
		DegradationTime: degradationTime,
	}, nil
}

// DegradeVectorSearch 向量检索降级
func (s *degradationServiceImpl) DegradeVectorSearch(
	ctx context.Context,
	sessionID string,
	query string,
	topK int,
) (*VectorSearchDegradationResult, error) {
	startTime := time.Now()

	s.log.WarnContext(ctx, "向量检索降级触发", logger.Fields{
		"session_id":   sessionID,
		"query_length": len(query),
		"top_k":        topK,
	})

	// 1. 尝试使用全文搜索作为降级方案
	memories, err := s.fullTextSearch(ctx, sessionID, query, topK)
	if err == nil && len(memories) > 0 {
		degradationTime := time.Since(startTime).Milliseconds()

		s.log.InfoContext(ctx, "向量检索降级：使用全文搜索", logger.Fields{
			"session_id":          sessionID,
			"found_count":         len(memories),
			"degradation_time_ms": degradationTime,
		})

		return &VectorSearchDegradationResult{
			Memories:        memories,
			Source:          "fulltext",
			FullTextUsed:    true,
			DegradationTime: degradationTime,
		}, nil
	}

	// 2. 全文搜索也失败，返回空结果
	degradationTime := time.Since(startTime).Milliseconds()

	s.log.WarnContext(ctx, "向量检索降级：全文搜索失败，返回空结果", logger.Fields{
		"session_id":          sessionID,
		"fulltext_error":      err,
		"degradation_time_ms": degradationTime,
	})

	return &VectorSearchDegradationResult{
		Memories:        []*model.ConversationMemory{},
		Source:          "empty",
		FullTextUsed:    false,
		DegradationTime: degradationTime,
	}, nil
}

// DegradeSummaryGeneration 摘要生成降级
func (s *degradationServiceImpl) DegradeSummaryGeneration(
	ctx context.Context,
	messages []*model.ConversationMessage,
	targetLength int,
) (*SummaryDegradationResult, error) {
	startTime := time.Now()

	s.log.WarnContext(ctx, "摘要生成降级触发", logger.Fields{
		"message_count": len(messages),
		"target_length": targetLength,
	})

	// 1. 计算原始内容长度
	originalContent := s.concatenateMessages(messages)
	originalLength := len(originalContent)

	// 2. 选择降级策略
	var summary string
	var method string

	if originalLength <= targetLength {
		// 内容已经足够短，直接返回
		summary = originalContent
		method = "direct"
	} else if targetLength >= 200 {
		// 使用提取关键句子的方法
		summary = s.extractKeySentences(messages, targetLength)
		method = "extract"
	} else {
		// 使用简单截断策略
		summary = s.truncateContent(originalContent, targetLength)
		method = "truncate"
	}

	degradationTime := time.Since(startTime).Milliseconds()

	s.log.InfoContext(ctx, "摘要生成降级完成", logger.Fields{
		"method":              method,
		"original_length":     originalLength,
		"summary_length":      len(summary),
		"degradation_time_ms": degradationTime,
	})

	return &SummaryDegradationResult{
		Summary:         summary,
		Method:          method,
		OriginalLength:  originalLength,
		SummaryLength:   len(summary),
		DegradationTime: degradationTime,
	}, nil
}

// getCachedResponse 从缓存获取相似查询的响应
func (s *degradationServiceImpl) getCachedResponse(ctx context.Context, sessionID, query string) (string, bool) {
	// 生成缓存键
	queryHash := s.cache.HashQuery(query)
	cacheKey := fmt.Sprintf("ai:response:%s:%s", sessionID, queryHash)

	// 尝试从缓存获取
	var cachedResponse string
	if err := s.cache.Get(ctx, cacheKey, &cachedResponse); err == nil {
		return cachedResponse, true
	}

	// 尝试查找相似查询的缓存
	// 这里简化处理，实际应用中可以使用更复杂的相似度匹配
	similarKey := fmt.Sprintf("ai:response:%s:*", sessionID)
	// 注意：这里需要实现模糊匹配逻辑，暂时返回未命中
	
	return "", false
}

// getDefaultResponse 获取默认响应
func (s *degradationServiceImpl) getDefaultResponse(query string) string {
	// 根据查询类型返回不同的默认响应
	queryLower := strings.ToLower(query)

	// 检查是否是问候语
	greetings := []string{"你好", "hello", "hi", "嗨"}
	for _, greeting := range greetings {
		if strings.Contains(queryLower, greeting) {
			return "您好！很抱歉，AI 服务暂时不可用。我们正在努力恢复服务，请稍后重试。"
		}
	}

	// 检查是否是帮助请求
	helpKeywords := []string{"帮助", "help", "怎么", "如何"}
	for _, keyword := range helpKeywords {
		if strings.Contains(queryLower, keyword) {
			return "很抱歉，AI 服务暂时不可用，无法为您提供详细帮助。请稍后重试，或联系技术支持。"
		}
	}

	// 默认响应
	return "抱歉，AI 服务暂时不可用。我们正在努力恢复服务，请稍后重试。如果问题持续存在，请联系技术支持。"
}

// fullTextSearch 全文搜索
func (s *degradationServiceImpl) fullTextSearch(
	ctx context.Context,
	sessionID string,
	query string,
	topK int,
) ([]*model.ConversationMemory, error) {
	// 使用数据库的全文搜索功能
	// 这里使用 LIKE 查询作为简单实现
	// 实际应用中应该使用 PostgreSQL 的全文搜索功能（tsvector, tsquery）
	
	memories, err := s.memoryRepo.SearchByContent(ctx, sessionID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("全文搜索失败: %w", err)
	}

	return memories, nil
}

// concatenateMessages 连接消息内容
func (s *degradationServiceImpl) concatenateMessages(messages []*model.ConversationMessage) string {
	var builder strings.Builder
	
	for i, msg := range messages {
		if i > 0 {
			builder.WriteString("\n")
		}
		
		// 添加角色标识
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		} else if msg.Role == "system" {
			role = "系统"
		}
		
		builder.WriteString(fmt.Sprintf("[%s]: %s", role, msg.Content))
	}
	
	return builder.String()
}

// extractKeySentences 提取关键句子
func (s *degradationServiceImpl) extractKeySentences(messages []*model.ConversationMessage, targetLength int) string {
	var builder strings.Builder
	currentLength := 0

	// 优先提取用户的问题和助手的回答
	for _, msg := range messages {
		if currentLength >= targetLength {
			break
		}

		// 跳过系统消息
		if msg.Role == "system" {
			continue
		}

		// 提取句子
		sentences := s.splitIntoSentences(msg.Content)
		for _, sentence := range sentences {
			if currentLength+len(sentence) > targetLength {
				break
			}

			if builder.Len() > 0 {
				builder.WriteString(" ")
			}
			builder.WriteString(sentence)
			currentLength += len(sentence)
		}
	}

	summary := builder.String()
	if summary == "" {
		// 如果没有提取到任何内容，使用第一条消息
		if len(messages) > 0 {
			summary = s.truncateContent(messages[0].Content, targetLength)
		}
	}

	return summary
}

// splitIntoSentences 将文本分割成句子
func (s *degradationServiceImpl) splitIntoSentences(text string) []string {
	// 简单的句子分割实现
	// 实际应用中应该使用更复杂的 NLP 工具
	
	// 按句号、问号、感叹号分割
	separators := []string{"。", "？", "！", ".", "?", "!"}
	
	sentences := []string{text}
	for _, sep := range separators {
		var newSentences []string
		for _, sentence := range sentences {
			parts := strings.Split(sentence, sep)
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					if i < len(parts)-1 {
						part += sep
					}
					newSentences = append(newSentences, part)
				}
			}
		}
		sentences = newSentences
	}
	
	return sentences
}

// truncateContent 截断内容
func (s *degradationServiceImpl) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// 截断到最大长度
	truncated := content[:maxLength]

	// 尝试在最后一个完整的词或句子处截断
	lastSpace := strings.LastIndex(truncated, " ")
	lastPeriod := strings.LastIndexAny(truncated, "。.？?！!")

	cutPoint := maxLength
	if lastPeriod > maxLength/2 {
		// 如果有句子结束符，在那里截断
		cutPoint = lastPeriod + 1
	} else if lastSpace > maxLength/2 {
		// 否则在最后一个空格处截断
		cutPoint = lastSpace
	}

	truncated = content[:cutPoint]
	
	// 添加省略号
	if cutPoint < len(content) {
		truncated += "..."
	}

	return truncated
}
