package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/storage"
)

// degradationServiceImpl 降级服务实现
type degradationServiceImpl struct {
	cache       storage.CacheService
	messageRepo repository.MessageRepository
	memoryRepo  repository.MemoryRepository
	logger      logger.Logger
}

// NewDegradationService 创建降级服务实例
func NewDegradationService(
	cache storage.CacheService,
	messageRepo repository.MessageRepository,
	memoryRepo repository.MemoryRepository,
	log logger.Logger,
) DegradationService {
	return &degradationServiceImpl{
		cache:       cache,
		messageRepo: messageRepo,
		memoryRepo:  memoryRepo,
		logger:      log,
	}
}

// DegradeAIService AI服务降级
func (s *degradationServiceImpl) DegradeAIService(
	ctx context.Context,
	sessionID string,
	userQuery string,
) (string, error) {
	s.logger.WarnContext(ctx, "AI服务降级触发", logger.Fields{
		"session_id": sessionID,
		"query":      userQuery,
	})

	// 1. 尝试从缓存获取相似查询的响应
	cachedResponse, err := s.getCachedResponse(ctx, sessionID, userQuery)
	if err == nil && cachedResponse != "" {
		s.logger.InfoContext(ctx, "从缓存获取到降级响应", logger.Fields{
			"session_id": sessionID,
			"source":     "cache",
		})
		return cachedResponse, nil
	}

	// 2. 返回默认响应
	defaultResponse := "抱歉，AI服务暂时不可用，请稍后重试。我们正在努力恢复服务。"
	
	s.logger.InfoContext(ctx, "使用默认降级响应", logger.Fields{
		"session_id": sessionID,
		"source":     "default",
	})

	return defaultResponse, nil
}

// DegradeVectorSearch 向量检索降级
func (s *degradationServiceImpl) DegradeVectorSearch(
	ctx context.Context,
	sessionID string,
	query string,
) ([]*model.ConversationMemory, error) {
	s.logger.WarnContext(ctx, "向量检索降级触发", logger.Fields{
		"session_id": sessionID,
		"query":      query,
	})

	// 1. 尝试使用全文搜索作为降级方案
	memories, err := s.fullTextSearch(ctx, sessionID, query)
	if err == nil && len(memories) > 0 {
		s.logger.InfoContext(ctx, "使用全文搜索作为降级方案", logger.Fields{
			"session_id":    sessionID,
			"memory_count":  len(memories),
			"search_method": "full_text",
		})
		return memories, nil
	}

	// 2. 如果全文搜索也失败，记录错误并返回空结果
	if err != nil {
		s.logger.ErrorContext(ctx, "全文搜索降级失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
	}

	s.logger.InfoContext(ctx, "返回空记忆列表", logger.Fields{
		"session_id": sessionID,
		"source":     "empty_fallback",
	})

	// 返回空结果，不影响主流程
	return []*model.ConversationMemory{}, nil
}

// DegradeSummaryGeneration 摘要生成降级
func (s *degradationServiceImpl) DegradeSummaryGeneration(
	ctx context.Context,
	messages []*model.ChatMessage,
	targetLength int,
) (string, error) {
	s.logger.WarnContext(ctx, "摘要生成降级触发", logger.Fields{
		"message_count": len(messages),
		"target_length": targetLength,
	})

	if len(messages) == 0 {
		return "", fmt.Errorf("消息列表为空，无法生成摘要")
	}

	// 使用简单截断策略生成摘要
	summary := s.simpleTruncateSummary(messages, targetLength)

	s.logger.InfoContext(ctx, "使用简单截断策略生成摘要", logger.Fields{
		"message_count":  len(messages),
		"summary_length": len(summary),
		"method":         "simple_truncate",
	})

	return summary, nil
}

// getCachedResponse 从缓存获取相似查询的响应
func (s *degradationServiceImpl) getCachedResponse(
	ctx context.Context,
	sessionID string,
	userQuery string,
) (string, error) {
	// 生成查询的哈希值作为缓存键的一部分
	queryHash := s.hashQuery(userQuery)
	cacheKey := fmt.Sprintf("ai_response:%s:%s", sessionID, queryHash)

	var response string
	err := s.cache.Get(ctx, cacheKey, &response)
	if err != nil {
		return "", err
	}

	return response, nil
}

// fullTextSearch 使用全文搜索检索记忆
func (s *degradationServiceImpl) fullTextSearch(
	ctx context.Context,
	sessionID string,
	query string,
) ([]*model.ConversationMemory, error) {
	// 提取查询关键词
	keywords := s.extractKeywords(query)
	if len(keywords) == 0 {
		return []*model.ConversationMemory{}, nil
	}

	// 使用关键词进行全文搜索
	// 注意：这里假设 MemoryRepository 有一个全文搜索方法
	// 如果没有，可以通过 LIKE 查询实现简单的全文搜索
	memories, err := s.searchByKeywords(ctx, sessionID, keywords)
	if err != nil {
		return nil, err
	}

	return memories, nil
}

// searchByKeywords 通过关键词搜索记忆
func (s *degradationServiceImpl) searchByKeywords(
	ctx context.Context,
	sessionID string,
	keywords []string,
) ([]*model.ConversationMemory, error) {
	// 这里实现简单的关键词匹配逻辑
	// 实际实现中，可以调用 repository 的全文搜索方法
	// 或者使用数据库的全文搜索功能

	// 由于当前 MemoryRepository 没有全文搜索方法
	// 这里返回空结果，实际项目中需要实现相应的 repository 方法
	s.logger.DebugContext(ctx, "关键词搜索", logger.Fields{
		"session_id": sessionID,
		"keywords":   keywords,
	})

	// TODO: 实现实际的关键词搜索逻辑
	// 可以通过以下方式实现：
	// 1. 在 MemoryRepository 中添加 SearchByKeywords 方法
	// 2. 使用 PostgreSQL 的全文搜索功能（tsvector, tsquery）
	// 3. 或者使用 LIKE 查询进行简单匹配

	return []*model.ConversationMemory{}, nil
}

// simpleTruncateSummary 使用简单截断策略生成摘要
func (s *degradationServiceImpl) simpleTruncateSummary(
	messages []*model.ChatMessage,
	targetLength int,
) string {
	var builder strings.Builder
	builder.WriteString("对话摘要：\n\n")

	currentLength := builder.Len()

	// 遍历消息，按顺序添加到摘要中
	for _, msg := range messages {
		// 构建消息文本
		var msgText string
		if msg.Role == "user" {
			msgText = fmt.Sprintf("用户：%s\n", msg.Content)
		} else {
			msgText = fmt.Sprintf("助手：%s\n", msg.Content)
		}

		// 检查是否超过目标长度
		if currentLength+len(msgText) > targetLength {
			// 如果添加这条消息会超过目标长度，进行截断
			remainingLength := targetLength - currentLength
			if remainingLength > 20 { // 至少保留20个字符
				truncated := msgText
				if len(msgText) > remainingLength {
					truncated = msgText[:remainingLength-3] + "..."
				}
				builder.WriteString(truncated)
			}
			break
		}

		builder.WriteString(msgText)
		currentLength += len(msgText)
	}

	// 添加摘要说明
	builder.WriteString("\n[注：此摘要由简化算法生成，可能不完整]")

	return builder.String()
}

// extractKeywords 从查询中提取关键词
func (s *degradationServiceImpl) extractKeywords(query string) []string {
	// 简单的关键词提取：分词并过滤停用词
	words := strings.Fields(query)
	
	// 定义常见停用词
	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"有": true, "和": true, "就": true, "不": true, "人": true,
		"都": true, "一": true, "一个": true, "上": true, "也": true,
		"很": true, "到": true, "说": true, "要": true, "去": true,
		"你": true, "会": true, "着": true, "没有": true, "看": true,
		"好": true, "自己": true, "这": true, "那": true, "什么": true,
		"怎么": true, "为什么": true, "吗": true, "呢": true, "啊": true,
	}

	var keywords []string
	for _, word := range words {
		// 过滤停用词和短词
		if len(word) > 1 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// hashQuery 生成查询的哈希值
func (s *degradationServiceImpl) hashQuery(query string) string {
	hash := md5.Sum([]byte(query))
	return hex.EncodeToString(hash[:])
}

// CacheAIResponse 缓存AI响应（供其他服务调用）
// 这个方法可以在正常的AI服务调用成功后，缓存响应以供降级使用
func (s *degradationServiceImpl) CacheAIResponse(
	ctx context.Context,
	sessionID string,
	userQuery string,
	response string,
	ttl time.Duration,
) error {
	queryHash := s.hashQuery(userQuery)
	cacheKey := fmt.Sprintf("ai_response:%s:%s", sessionID, queryHash)

	err := s.cache.Set(ctx, cacheKey, response, ttl)
	if err != nil {
		s.logger.ErrorContext(ctx, "缓存AI响应失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return err
	}

	s.logger.DebugContext(ctx, "AI响应已缓存", logger.Fields{
		"session_id": sessionID,
		"cache_key":  cacheKey,
		"ttl":        ttl.String(),
	})

	return nil
}
