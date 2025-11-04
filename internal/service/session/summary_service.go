package session

import (
	"context"
	"fmt"
	"strings"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
)

// SummaryService 摘要业务逻辑接口
type SummaryService interface {
	// GenerateSummary 生成会话摘要
	GenerateSummary(ctx context.Context, sessionID string) (*model.ChatSummary, error)

	// GetSummary 获取会话摘要
	GetSummary(ctx context.Context, sessionID string) (*model.ChatSummary, error)

	// CheckSummaryTrigger 检查是否需要生成摘要（需求10）
	CheckSummaryTrigger(ctx context.Context, req *CheckSummaryTriggerRequest) (*CheckSummaryTriggerResponse, error)

	// EvaluateSummaryQuality 评估摘要质量（需求11）
	EvaluateSummaryQuality(ctx context.Context, req *EvaluateSummaryQualityRequest) (*EvaluateSummaryQualityResponse, error)
}

// summaryService 摘要业务逻辑实现
type summaryService struct {
	summaryRepo repository.SummaryRepository
	messageRepo repository.MessageRepository
	sessionRepo repository.SessionRepository
	aiService   ai.AIService
	config      *config.Config
	logger      logger.Logger
}

// NewSummaryService 创建摘要服务实例
func NewSummaryService(
	summaryRepo repository.SummaryRepository,
	messageRepo repository.MessageRepository,
	sessionRepo repository.SessionRepository,
	aiService ai.AIService,
	cfg *config.Config,
	log logger.Logger,
) SummaryService {
	return &summaryService{
		summaryRepo: summaryRepo,
		messageRepo: messageRepo,
		sessionRepo: sessionRepo,
		aiService:   aiService,
		config:      cfg,
		logger:      log,
	}
}

// GenerateSummary 生成会话摘要
func (s *summaryService) GenerateSummary(ctx context.Context, sessionID string) (*model.ChatSummary, error) {
	s.logger.Info("开始生成会话摘要", map[string]interface{}{
		"sessionId": sessionID,
	})

	// 1. 验证会话是否存在
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		s.logger.Error("查询会话失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}
	if session == nil {
		s.logger.Warn("会话不存在", map[string]interface{}{
			"sessionId": sessionID,
		})
		return nil, fmt.Errorf("会话不存在")
	}

	// 2. 获取最新的摘要（如果存在）
	latestSummary, err := s.summaryRepo.GetLatestBySessionID(ctx, sessionID)
	if err != nil {
		s.logger.Error("查询最新摘要失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("查询最新摘要失败: %w", err)
	}

	// 3. 获取需要摘要的消息列表
	var messages []*model.ChatMessage
	if latestSummary != nil {
		// 如果已有摘要，只获取摘要之后的消息
		messages, err = s.messageRepo.GetMessagesAfter(ctx, sessionID, latestSummary.LastMessageID.String())
		if err != nil {
			s.logger.Error("获取摘要后的消息失败", map[string]interface{}{
				"sessionId":     sessionID,
				"lastMessageId": latestSummary.LastMessageID.String(),
				"error":         err.Error(),
			})
			return nil, fmt.Errorf("获取消息失败: %w", err)
		}
	} else {
		// 如果没有摘要，获取所有消息
		messages, _, err = s.messageRepo.GetBySessionID(ctx, sessionID, 1, 10000)
		if err != nil {
			s.logger.Error("获取会话消息失败", map[string]interface{}{
				"sessionId": sessionID,
				"error":     err.Error(),
			})
			return nil, fmt.Errorf("获取消息失败: %w", err)
		}
	}

	// 4. 检查是否有足够的消息生成摘要
	if len(messages) == 0 {
		s.logger.Warn("没有新消息需要生成摘要", map[string]interface{}{
			"sessionId": sessionID,
		})
		return latestSummary, nil
	}

	// 5. 构建摘要提示词
	summaryPrompt := s.buildSummaryPrompt(messages, latestSummary)

	// 6. 调用AI服务生成摘要
	temperature := 0.3 // 使用较低的温度以获得更稳定的摘要
	maxTokens := 1000
	chatReq := &model.ChatRequest{
		Message: summaryPrompt,
		Options: &model.ChatOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		},
	}

	chatResp, err := s.aiService.Chat(ctx, chatReq)
	if err != nil {
		s.logger.Error("AI生成摘要失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("AI生成摘要失败: %w", err)
	}

	// 7. 创建摘要记录
	lastMessageID := messages[len(messages)-1].ID
	summary := &model.ChatSummary{
		SessionID:     session.ID,
		Summary:       chatResp.Message,
		LastMessageID: lastMessageID,
		TokenCount:    chatResp.Usage.TotalTokens,
	}

	if err := s.summaryRepo.Create(ctx, summary); err != nil {
		s.logger.Error("保存摘要失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("保存摘要失败: %w", err)
	}

	s.logger.Info("会话摘要生成成功", map[string]interface{}{
		"sessionId":     sessionID,
		"summaryId":     summary.ID,
		"lastMessageId": lastMessageID,
		"tokenCount":    summary.TokenCount,
	})

	return summary, nil
}

// GetSummary 获取会话摘要
func (s *summaryService) GetSummary(ctx context.Context, sessionID string) (*model.ChatSummary, error) {
	s.logger.Debug("获取会话摘要", map[string]interface{}{
		"sessionId": sessionID,
	})

	summary, err := s.summaryRepo.GetLatestBySessionID(ctx, sessionID)
	if err != nil {
		s.logger.Error("查询会话摘要失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("查询会话摘要失败: %w", err)
	}

	return summary, nil
}

// ShouldGenerateSummary 判断是否需要生成摘要
func (s *summaryService) ShouldGenerateSummary(ctx context.Context, sessionID string) (bool, error) {
	s.logger.Debug("检查是否需要生成摘要", map[string]interface{}{
		"sessionId": sessionID,
	})

	// 1. 获取会话的消息总数
	messageCount, err := s.messageRepo.CountBySessionID(ctx, sessionID)
	if err != nil {
		s.logger.Error("统计消息数量失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return false, fmt.Errorf("统计消息数量失败: %w", err)
	}

	// 2. 检查消息数量是否达到阈值
	threshold := s.config.Session.SummaryThreshold
	if messageCount < threshold {
		s.logger.Debug("消息数量未达到摘要阈值", map[string]interface{}{
			"sessionId":    sessionID,
			"messageCount": messageCount,
			"threshold":    threshold,
		})
		return false, nil
	}

	// 3. 获取最新的摘要
	latestSummary, err := s.summaryRepo.GetLatestBySessionID(ctx, sessionID)
	if err != nil {
		s.logger.Error("查询最新摘要失败", map[string]interface{}{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return false, fmt.Errorf("查询最新摘要失败: %w", err)
	}

	// 4. 如果没有摘要，需要生成
	if latestSummary == nil {
		s.logger.Info("会话无摘要且消息数达到阈值，需要生成摘要", map[string]interface{}{
			"sessionId":    sessionID,
			"messageCount": messageCount,
			"threshold":    threshold,
		})
		return true, nil
	}

	// 5. 计算摘要后的新消息数量
	messagesAfterSummary, err := s.messageRepo.GetMessagesAfter(ctx, sessionID, latestSummary.LastMessageID.String())
	if err != nil {
		s.logger.Error("获取摘要后的消息失败", map[string]interface{}{
			"sessionId":     sessionID,
			"lastMessageId": latestSummary.LastMessageID.String(),
			"error":         err.Error(),
		})
		return false, fmt.Errorf("获取摘要后的消息失败: %w", err)
	}

	newMessageCount := len(messagesAfterSummary)
	shouldGenerate := newMessageCount >= threshold

	s.logger.Debug("检查摘要生成条件", map[string]interface{}{
		"sessionId":       sessionID,
		"newMessageCount": newMessageCount,
		"threshold":       threshold,
		"shouldGenerate":  shouldGenerate,
	})

	return shouldGenerate, nil
}

// CheckSummaryTriggerRequest 检查摘要触发请求
type CheckSummaryTriggerRequest struct {
	SessionID string
	CheckMode string // 'auto', 'force'
}

// CheckSummaryTriggerResponse 检查摘要触发响应
type CheckSummaryTriggerResponse struct {
	ShouldTrigger      bool
	TriggerScore       float64
	Urgency            float64
	EstimatedSavings   int
	RecommendedType    string
	TriggerReasons     []string
	MessagesSinceLastSummary int
	CurrentTokenUsage  int
	MaxTokenLimit      int
}

// EvaluateSummaryQualityRequest 评估摘要质量请求
type EvaluateSummaryQualityRequest struct {
	SummaryContent   string
	OriginalMessages []*model.ChatMessage
}

// EvaluateSummaryQualityResponse 评估摘要质量响应
type EvaluateSummaryQualityResponse struct {
	OverallScore      float64
	Completeness      float64
	Conciseness       float64
	Coherence         float64
	Accuracy          float64
	Passed            bool
	Issues            []string
	Suggestions       []string
	KeyInfoCoverage   float64
}

// CheckSummaryTrigger 检查是否需要生成摘要（需求10）
func (s *summaryService) CheckSummaryTrigger(ctx context.Context, req *CheckSummaryTriggerRequest) (*CheckSummaryTriggerResponse, error) {
	s.logger.InfoContext(ctx, "检查摘要触发条件", "session_id", req.SessionID, "check_mode", req.CheckMode)

	// 强制模式：无条件触发
	if req.CheckMode == "force" {
		return &CheckSummaryTriggerResponse{
			ShouldTrigger:    true,
			TriggerScore:     1.0,
			Urgency:          1.0,
			RecommendedType:  "full",
			TriggerReasons:   []string{"强制触发模式"},
		}, nil
	}

	// 1. 获取会话信息
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话信息失败: %w", err)
	}

	// 2. 获取最新摘要
	latestSummary, err := s.summaryRepo.GetLatestBySessionID(ctx, req.SessionID)
	if err != nil {
		s.logger.WarnContext(ctx, "获取最新摘要失败", "error", err)
	}

	// 3. 获取消息统计
	var messagesSinceLastSummary int
	if latestSummary != nil {
		messages, err := s.messageRepo.GetMessagesAfter(ctx, req.SessionID, latestSummary.LastMessageID.String())
		if err != nil {
			return nil, fmt.Errorf("获取摘要后的消息失败: %w", err)
		}
		messagesSinceLastSummary = len(messages)
	} else {
		totalCount, err := s.messageRepo.CountBySessionID(ctx, req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("统计消息数量失败: %w", err)
		}
		messagesSinceLastSummary = totalCount
	}

	// 4. 检查五种触发条件
	triggerReasons := []string{}
	var triggerScore float64 = 0.0

	// 条件1: 消息数量达到阈值（20条）
	messageThreshold := 20
	if messagesSinceLastSummary >= messageThreshold {
		triggerReasons = append(triggerReasons, fmt.Sprintf("新增消息达到%d条", messagesSinceLastSummary))
		triggerScore += 0.3
	}

	// 条件2: Token使用率超过80%
	maxTokenLimit := 4000 // 默认值
	currentTokenUsage := session.TotalTokens
	tokenUsageRate := float64(currentTokenUsage) / float64(maxTokenLimit)
	if tokenUsageRate > 0.8 {
		triggerReasons = append(triggerReasons, fmt.Sprintf("Token使用率达到%.1f%%", tokenUsageRate*100))
		triggerScore += 0.4
	}

	// 条件3: 距离上次摘要超过24小时
	if latestSummary != nil {
		hoursSinceLastSummary := float64(session.UpdatedAt.Sub(latestSummary.CreatedAt).Hours())
		if hoursSinceLastSummary > 24 && messagesSinceLastSummary > 0 {
			triggerReasons = append(triggerReasons, fmt.Sprintf("距离上次摘要已%.1f小时", hoursSinceLastSummary))
			triggerScore += 0.2
		}
	}

	// 条件4: 上下文质量评分低于0.6（简化评估）
	qualityScore := s.estimateContextQuality(messagesSinceLastSummary, currentTokenUsage, maxTokenLimit)
	if qualityScore < 0.6 {
		triggerReasons = append(triggerReasons, fmt.Sprintf("上下文质量评分较低(%.2f)", qualityScore))
		triggerScore += 0.1
	}

	// 5. 计算紧急程度
	urgency := 0.0
	if tokenUsageRate > 0.9 {
		urgency = 1.0
	} else if tokenUsageRate > 0.8 {
		urgency = 0.7
	} else if messagesSinceLastSummary > 30 {
		urgency = 0.6
	} else if messagesSinceLastSummary > 20 {
		urgency = 0.4
	}

	// 6. 估算Token节省量
	estimatedSavings := messagesSinceLastSummary * 50 // 假设每条消息平均50 tokens

	// 7. 推荐摘要类型
	recommendedType := "incremental"
	if latestSummary == nil || messagesSinceLastSummary > 50 {
		recommendedType = "full"
	}

	// 8. 判断是否应该触发
	shouldTrigger := triggerScore >= 0.5 || urgency >= 0.7

	response := &CheckSummaryTriggerResponse{
		ShouldTrigger:            shouldTrigger,
		TriggerScore:             triggerScore,
		Urgency:                  urgency,
		EstimatedSavings:         estimatedSavings,
		RecommendedType:          recommendedType,
		TriggerReasons:           triggerReasons,
		MessagesSinceLastSummary: messagesSinceLastSummary,
		CurrentTokenUsage:        currentTokenUsage,
		MaxTokenLimit:            maxTokenLimit,
	}

	s.logger.InfoContext(ctx, "摘要触发检查完成",
		"session_id", req.SessionID,
		"should_trigger", shouldTrigger,
		"trigger_score", triggerScore,
		"urgency", urgency,
	)

	return response, nil
}

// EvaluateSummaryQuality 评估摘要质量（需求11）
func (s *summaryService) EvaluateSummaryQuality(ctx context.Context, req *EvaluateSummaryQualityRequest) (*EvaluateSummaryQualityResponse, error) {
	s.logger.InfoContext(ctx, "开始评估摘要质量", "summary_length", len(req.SummaryContent), "message_count", len(req.OriginalMessages))

	// 1. 评估完整性（Completeness）
	completeness := s.evaluateCompleteness(req.SummaryContent, req.OriginalMessages)

	// 2. 评估简洁性（Conciseness）
	conciseness := s.evaluateConciseness(req.SummaryContent, req.OriginalMessages)

	// 3. 评估连贯性（Coherence）
	coherence := s.evaluateCoherence(req.SummaryContent)

	// 4. 评估准确性（Accuracy）
	accuracy := s.evaluateAccuracy(req.SummaryContent, req.OriginalMessages)

	// 5. 计算总体评分
	overallScore := (completeness + conciseness + coherence + accuracy) / 4.0

	// 6. 判断是否通过质量检查
	passed := overallScore >= 0.7

	// 7. 识别质量问题
	issues := []string{}
	if completeness < 0.7 {
		issues = append(issues, "摘要不够完整，遗漏了重要信息")
	}
	if conciseness < 0.7 {
		issues = append(issues, "摘要过于冗长，需要更加简洁")
	}
	if coherence < 0.7 {
		issues = append(issues, "摘要逻辑不够连贯")
	}
	if accuracy < 0.7 {
		issues = append(issues, "摘要存在不准确的信息")
	}

	// 8. 提供改进建议
	suggestions := []string{}
	if completeness < 0.8 {
		suggestions = append(suggestions, "增加关键信息的覆盖")
	}
	if conciseness < 0.8 {
		suggestions = append(suggestions, "删除冗余和重复的内容")
	}
	if coherence < 0.8 {
		suggestions = append(suggestions, "改善段落之间的逻辑连接")
	}
	if accuracy < 0.8 {
		suggestions = append(suggestions, "核对事实信息的准确性")
	}

	// 9. 计算关键信息覆盖率
	keyInfoCoverage := s.calculateKeyInfoCoverage(req.SummaryContent, req.OriginalMessages)

	response := &EvaluateSummaryQualityResponse{
		OverallScore:    overallScore,
		Completeness:    completeness,
		Conciseness:     conciseness,
		Coherence:       coherence,
		Accuracy:        accuracy,
		Passed:          passed,
		Issues:          issues,
		Suggestions:     suggestions,
		KeyInfoCoverage: keyInfoCoverage,
	}

	s.logger.InfoContext(ctx, "摘要质量评估完成",
		"overall_score", overallScore,
		"passed", passed,
		"issues_count", len(issues),
	)

	return response, nil
}

// estimateContextQuality 估算上下文质量
func (s *summaryService) estimateContextQuality(messageCount, currentTokens, maxTokens int) float64 {
	// 基于消息数量和Token使用率的简单评估
	messageScore := 1.0
	if messageCount > 30 {
		messageScore = 0.5
	} else if messageCount > 20 {
		messageScore = 0.7
	}

	tokenScore := 1.0 - float64(currentTokens)/float64(maxTokens)
	if tokenScore < 0 {
		tokenScore = 0
	}

	return (messageScore + tokenScore) / 2.0
}

// evaluateCompleteness 评估完整性
func (s *summaryService) evaluateCompleteness(summary string, messages []*model.ChatMessage) float64 {
	// 简化实现：基于摘要长度和消息数量的比例
	summaryLength := len(summary)
	totalMessageLength := 0
	for _, msg := range messages {
		totalMessageLength += len(msg.Content)
	}

	if totalMessageLength == 0 {
		return 0.5
	}

	// 期望摘要长度约为原文的10-20%
	ratio := float64(summaryLength) / float64(totalMessageLength)
	if ratio >= 0.1 && ratio <= 0.3 {
		return 0.9
	} else if ratio >= 0.05 && ratio <= 0.4 {
		return 0.7
	}
	return 0.5
}

// evaluateConciseness 评估简洁性
func (s *summaryService) evaluateConciseness(summary string, messages []*model.ChatMessage) float64 {
	// 简化实现：基于摘要长度
	summaryLength := len(summary)

	// 理想摘要长度：200-500字
	if summaryLength >= 200 && summaryLength <= 500 {
		return 0.9
	} else if summaryLength >= 100 && summaryLength <= 800 {
		return 0.7
	} else if summaryLength < 100 {
		return 0.5 // 太短
	}
	return 0.6 // 太长
}

// evaluateCoherence 评估连贯性
func (s *summaryService) evaluateCoherence(summary string) float64 {
	// 简化实现：检查基本的连贯性指标
	score := 0.7 // 基础分

	// 检查是否有段落结构
	if strings.Contains(summary, "\n") {
		score += 0.1
	}

	// 检查是否有连接词
	connectors := []string{"因此", "所以", "然后", "接着", "最后", "首先", "其次", "另外", "此外"}
	for _, connector := range connectors {
		if strings.Contains(summary, connector) {
			score += 0.05
			break
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// evaluateAccuracy 评估准确性
func (s *summaryService) evaluateAccuracy(summary string, messages []*model.ChatMessage) float64 {
	// 简化实现：检查摘要中的关键词是否出现在原始消息中
	score := 0.8 // 基础分

	// 提取摘要中的关键词（简化：取长度>2的词）
	summaryWords := strings.Fields(summary)
	matchCount := 0
	totalWords := 0

	for _, word := range summaryWords {
		if len(word) > 2 {
			totalWords++
			// 检查是否在原始消息中出现
			for _, msg := range messages {
				if strings.Contains(msg.Content, word) {
					matchCount++
					break
				}
			}
		}
	}

	if totalWords > 0 {
		matchRate := float64(matchCount) / float64(totalWords)
		score = matchRate
	}

	return score
}

// calculateKeyInfoCoverage 计算关键信息覆盖率
func (s *summaryService) calculateKeyInfoCoverage(summary string, messages []*model.ChatMessage) float64 {
	// 简化实现：基于关键词匹配
	keywords := s.extractKeywords(messages)
	if len(keywords) == 0 {
		return 0.5
	}

	coveredCount := 0
	for _, keyword := range keywords {
		if strings.Contains(summary, keyword) {
			coveredCount++
		}
	}

	return float64(coveredCount) / float64(len(keywords))
}

// extractKeywords 提取关键词
func (s *summaryService) extractKeywords(messages []*model.ChatMessage) []string {
	// 简化实现：提取长度>3的高频词
	wordCount := make(map[string]int)

	for _, msg := range messages {
		words := strings.Fields(msg.Content)
		for _, word := range words {
			if len(word) > 3 {
				wordCount[word]++
			}
		}
	}

	// 选择出现次数>1的词作为关键词
	keywords := []string{}
	for word, count := range wordCount {
		if count > 1 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// buildSummaryPrompt 构建摘要提示词
func (s *summaryService) buildSummaryPrompt(messages []*model.ChatMessage, previousSummary *model.ChatSummary) string {
	var builder strings.Builder

	// 添加任务说明
	builder.WriteString("请为以下对话生成一个简洁的摘要，保留关键信息和上下文。\n\n")

	// 如果有之前的摘要，先包含它
	if previousSummary != nil {
		builder.WriteString("之前的对话摘要：\n")
		builder.WriteString(previousSummary.Summary)
		builder.WriteString("\n\n新的对话内容：\n")
	} else {
		builder.WriteString("对话内容：\n")
	}

	// 添加消息内容
	for _, msg := range messages {
		builder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	builder.WriteString("\n请生成摘要（200字以内）：")

	return builder.String()
}
