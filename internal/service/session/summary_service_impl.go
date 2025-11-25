package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
)

// summaryServiceImpl 摘要服务实现
type summaryServiceImpl struct {
	summaryRepo  repository.SummaryRepository
	messageRepo  repository.MessageRepository
	contextRepo  repository.ContextRepository
	sessionRepo  repository.SessionRepository
	genkitClient genkit.Client
	tokenMgr     service.TokenManager
	logger       logger.Logger
}

// NewSummaryService 创建摘要服务实例
func NewSummaryService(
	summaryRepo repository.SummaryRepository,
	messageRepo repository.MessageRepository,
	contextRepo repository.ContextRepository,
	sessionRepo repository.SessionRepository,
	genkitClient genkit.Client,
	tokenMgr service.TokenManager,
	log logger.Logger,
) SummaryService {
	return &summaryServiceImpl{
		summaryRepo:  summaryRepo,
		messageRepo:  messageRepo,
		contextRepo:  contextRepo,
		sessionRepo:  sessionRepo,
		genkitClient: genkitClient,
		tokenMgr:     tokenMgr,
		logger:       log,
	}
}

// GenerateSummary 生成摘要
func (s *summaryServiceImpl) GenerateSummary(ctx context.Context, req *GenerateSummaryRequest) (*model.ConversationSummary, error) {
	// 1. 权限验证
	if err := s.validateAccess(ctx, req.TenantID, req.SessionID); err != nil {
		return nil, err
	}

	// 2. 获取消息列表
	messages, err := s.getMessagesForSummary(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取消息列表失败: %w", err)
	}

	if len(messages) == 0 {
		return nil, errors.NewBadRequestError("没有可用于生成摘要的消息")
	}

	s.logger.InfoContext(ctx, "开始生成摘要", logger.Fields{
		"tenant_id":    req.TenantID.String(),
		"session_id":   req.SessionID.String(),
		"message_count": len(messages),
		"summary_type": req.SummaryType,
	})

	// 3. 构建提示词
	prompt := s.buildSummaryPrompt(messages, req.PreviousSummary, req.SummaryType, req.TargetLength)

	// 4. 调用 Genkit AI 生成摘要
	// TODO: TASK-5.2 - 从上下文中获取租户ID，从配置中获取摘要专用模型
	// 临时使用默认值以保持编译通过
	tenantID := req.TenantID.String()
	modelName := "gemini-pro" // 摘要服务使用的默认模型
	
	temperature := 0.3
	result, err := s.genkitClient.Generate(ctx, tenantID, modelName, prompt, &genkit.GenerateOptions{
		Temperature: &temperature, // 使用较低的温度以保证摘要稳定性
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "AI生成摘要失败", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("AI生成摘要失败: %w", err)
	}

	summaryContent := strings.TrimSpace(result.Text)

	// 5. 计算Token数量
	tokenCount, err := s.tokenMgr.CalculateTokens(ctx, summaryContent, result.Model)
	if err != nil {
		s.logger.WarnContext(ctx, "计算摘要Token失败", logger.Fields{
			"error": err.Error(),
		})
		tokenCount = s.tokenMgr.EstimateTokens(summaryContent)
	}

	// 6. 提取关键主题
	keyTopics := s.extractKeyTopics(summaryContent)

	// 7. 计算质量评分
	qualityScore := s.calculateQualityScore(ctx, summaryContent, messages)

	// 8. 计算压缩率
	originalTokens, _ := s.tokenMgr.CalculateMessagesTokens(ctx, messages, result.Model)
	compressionRate := 0.0
	if originalTokens > 0 {
		compressionRate = 1.0 - float64(tokenCount)/float64(originalTokens)
	}

	// 9. 获取消息ID范围
	var startMessageID, endMessageID *uuid.UUID
	if len(messages) > 0 {
		startMessageID = &messages[0].ID
		endMessageID = &messages[len(messages)-1].ID
	}

	// 10. 获取前一个摘要ID
	var previousSummaryID *uuid.UUID
	if req.SummaryType == "incremental" {
		latestSummary, err := s.summaryRepo.GetLatestBySessionID(ctx, req.TenantID, req.SessionID)
		if err == nil && latestSummary != nil {
			previousSummaryID = &latestSummary.ID
		}
	}

	// 11. 创建摘要记录
	summary := &model.ConversationSummary{
		TenantID:          req.TenantID,
		SessionID:         req.SessionID,
		SummaryType:       req.SummaryType,
		Content:           summaryContent,
		TokenCount:        tokenCount,
		MessageCount:      len(messages),
		StartMessageID:    startMessageID,
		EndMessageID:      endMessageID,
		QualityScore:      &qualityScore,
		CompressionRate:   &compressionRate,
		KeyTopics:         keyTopics,
		PreviousSummaryID: previousSummaryID,
	}

	// 12. 保存摘要到数据库
	if err := s.summaryRepo.Create(ctx, summary); err != nil {
		return nil, fmt.Errorf("保存摘要失败: %w", err)
	}

	// 13. 更新会话上下文配置
	if err := s.updateContextAfterSummary(ctx, req.TenantID, req.SessionID, summary.ID); err != nil {
		s.logger.WarnContext(ctx, "更新上下文配置失败", logger.Fields{
			"error": err.Error(),
		})
	}

	s.logger.InfoContext(ctx, "摘要生成成功", logger.Fields{
		"summary_id":       summary.ID.String(),
		"token_count":      tokenCount,
		"quality_score":    qualityScore,
		"compression_rate": compressionRate,
	})

	return summary, nil
}

// CheckSummaryTrigger 检查是否需要生成摘要
func (s *summaryServiceImpl) CheckSummaryTrigger(ctx context.Context, tenantID, sessionID uuid.UUID) (*SummaryTriggerResult, error) {
	// 1. 权限验证
	if err := s.validateAccess(ctx, tenantID, sessionID); err != nil {
		return nil, err
	}

	// 2. 获取会话上下文配置
	contextConfig, err := s.contextRepo.GetBySessionID(ctx, sessionID.String())
	if err != nil {
		return nil, fmt.Errorf("获取上下文配置失败: %w", err)
	}

	// 3. 获取最新摘要
	latestSummary, err := s.summaryRepo.GetLatestBySessionID(ctx, tenantID, sessionID)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("获取最新摘要失败: %w", err)
	}

	// 4. 计算自上次摘要后的新消息数量
	var messagesSinceLastSummary int
	var newMessages []*model.ChatMessage

	if latestSummary != nil && latestSummary.EndMessageID != nil {
		// 获取最后一条摘要消息之后的所有消息
		newMessages, err = s.messageRepo.GetMessagesAfter(ctx, sessionID.String(), latestSummary.EndMessageID.String())
		if err != nil {
			return nil, fmt.Errorf("获取新消息失败: %w", err)
		}
		messagesSinceLastSummary = len(newMessages)
	} else {
		// 没有摘要，获取所有消息
		messagesSinceLastSummary = contextConfig.TotalMessages
		newMessages, err = s.messageRepo.GetLatestMessages(ctx, sessionID.String(), messagesSinceLastSummary)
		if err != nil {
			return nil, fmt.Errorf("获取消息失败: %w", err)
		}
	}

	// 5. 检查触发条件
	result := &SummaryTriggerResult{
		ShouldSummarize:  false,
		MessageCount:     messagesSinceLastSummary,
		RecommendedType:  "incremental",
	}

	// 提取消息ID
	messageIDs := make([]uuid.UUID, len(newMessages))
	for i, msg := range newMessages {
		messageIDs[i] = msg.ID
	}
	result.MessageIDs = messageIDs

	// 条件1：消息数量达到阈值（20条）
	if messagesSinceLastSummary >= 20 {
		result.ShouldSummarize = true
		result.TriggerReason = "新消息数量达到阈值（20条）"
		result.Urgency = 0.7
	}

	// 条件2：Token使用量超过限制的80%
	currentTokens, _ := s.tokenMgr.CalculateMessagesTokens(ctx, newMessages, "")
	tokenUsageRate := float64(currentTokens) / float64(contextConfig.MaxTokens)
	if tokenUsageRate > 0.8 {
		result.ShouldSummarize = true
		result.TriggerReason = "Token使用量超过限制的80%"
		result.Urgency = 1.0 // 最高紧急程度
		result.EstimatedTokenSaving = int(float64(currentTokens) * 0.7) // 估算可节省70%的Token
	}

	// 条件3：距离上次摘要超过24小时且有新消息
	if latestSummary != nil && messagesSinceLastSummary > 0 {
		timeSinceLastSummary := time.Since(latestSummary.CreatedAt)
		if timeSinceLastSummary > 24*time.Hour {
			result.ShouldSummarize = true
			result.TriggerReason = "距离上次摘要超过24小时"
			result.Urgency = 0.5
		}
	}

	// 条件4：如果没有摘要且消息数量较多，建议生成完整摘要
	if latestSummary == nil && messagesSinceLastSummary >= 10 {
		result.ShouldSummarize = true
		result.TriggerReason = "会话尚无摘要且消息数量较多"
		result.RecommendedType = "full"
		result.Urgency = 0.6
	}

	s.logger.InfoContext(ctx, "摘要触发检查完成", logger.Fields{
		"should_summarize":  result.ShouldSummarize,
		"trigger_reason":    result.TriggerReason,
		"message_count":     result.MessageCount,
		"urgency":           result.Urgency,
		"recommended_type":  result.RecommendedType,
	})

	return result, nil
}

// EvaluateSummaryQuality 评估摘要质量
func (s *summaryServiceImpl) EvaluateSummaryQuality(ctx context.Context, req *EvaluateSummaryRequest) (*SummaryQualityResult, error) {
	result := &SummaryQualityResult{
		DimensionScores: make(map[string]float64),
		Issues:          []QualityIssue{},
		Suggestions:     []string{},
	}

	// 默认评估所有维度
	dimensions := req.Dimensions
	if len(dimensions) == 0 {
		dimensions = []string{"completeness", "conciseness", "coherence", "accuracy"}
	}

	// 评估各个维度
	for _, dimension := range dimensions {
		score := s.evaluateDimension(ctx, dimension, req.Summary, req.OriginalMessages)
		result.DimensionScores[dimension] = score

		// 记录低分维度的问题
		if score < 0.7 {
			issue := QualityIssue{
				Dimension:   dimension,
				Score:       score,
				Description: s.getDimensionIssueDescription(dimension, score),
			}

			if score < 0.5 {
				issue.Severity = "high"
			} else if score < 0.7 {
				issue.Severity = "medium"
			} else {
				issue.Severity = "low"
			}

			result.Issues = append(result.Issues, issue)
		}
	}

	// 计算总体评分（各维度平均值）
	totalScore := 0.0
	for _, score := range result.DimensionScores {
		totalScore += score
	}
	result.OverallScore = totalScore / float64(len(result.DimensionScores))

	// 判断是否通过质量检查（总体评分 >= 0.7）
	result.Passed = result.OverallScore >= 0.7

	// 计算关键信息覆盖率
	result.KeyInfoCoverage = s.calculateKeyInfoCoverage(req.Summary, req.OriginalMessages)

	// 生成改进建议
	result.Suggestions = s.generateImprovementSuggestions(result)

	s.logger.InfoContext(ctx, "摘要质量评估完成", logger.Fields{
		"overall_score":      result.OverallScore,
		"passed":             result.Passed,
		"key_info_coverage":  result.KeyInfoCoverage,
		"issues_count":       len(result.Issues),
	})

	return result, nil
}

// GetSummary 获取摘要详情
func (s *summaryServiceImpl) GetSummary(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error) {
	summary, err := s.summaryRepo.GetByID(ctx, tenantID, summaryID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, errors.NewNotFoundError("摘要不存在")
		}
		return nil, fmt.Errorf("获取摘要失败: %w", err)
	}

	// 权限验证
	if err := s.validateAccess(ctx, tenantID, summary.SessionID); err != nil {
		return nil, err
	}

	return summary, nil
}

// ListSummaries 获取会话摘要列表
func (s *summaryServiceImpl) ListSummaries(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error) {
	// 权限验证
	if err := s.validateAccess(ctx, tenantID, sessionID); err != nil {
		return nil, err
	}

	summaries, err := s.summaryRepo.ListBySessionID(ctx, tenantID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("获取摘要列表失败: %w", err)
	}

	return summaries, nil
}

// validateAccess 验证租户访问权限
func (s *summaryServiceImpl) validateAccess(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	// 获取 JWT 声明
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		return errors.NewUnauthorizedError("未认证")
	}

	// 平台管理员可以访问所有会话
	if hasRole(claims, model.RoleSystemAdmin) {
		return nil
	}

	// 验证租户ID匹配
	claimsTenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return errors.NewUnauthorizedError("无效的租户ID")
	}

	if claimsTenantID != tenantID {
		s.logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话",
			logger.Fields{
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_tenant_id": tenantID.String(),
				"session_id":       sessionID.String(),
			},
		)
		return errors.NewForbiddenError("权限不足：无法访问其他租户的会话")
	}

	return nil
}

// hasRole 检查用户是否具有指定角色
func hasRole(claims *model.JWTClaims, role string) bool {
	if claims == nil || claims.Roles == nil {
		return false
	}
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// getMessagesForSummary 获取用于生成摘要的消息列表
func (s *summaryServiceImpl) getMessagesForSummary(ctx context.Context, req *GenerateSummaryRequest) ([]*model.ChatMessage, error) {
	var messages []*model.ChatMessage
	var err error

	// 如果提供了消息ID列表，直接获取这些消息
	if len(req.MessageIDs) > 0 {
		messages = make([]*model.ChatMessage, 0, len(req.MessageIDs))
		for _, msgID := range req.MessageIDs {
			msg, err := s.messageRepo.GetByID(ctx, msgID.String())
			if err != nil {
				s.logger.WarnContext(ctx, "获取消息失败", logger.Fields{
					"message_id": msgID.String(),
					"error":      err.Error(),
				})
				continue
			}
			messages = append(messages, msg)
		}
	} else if req.StartMessageID != nil && req.EndMessageID != nil {
		// 如果提供了起始和结束消息ID，获取范围内的消息
		messages, err = s.messageRepo.GetMessagesAfter(ctx, req.SessionID.String(), req.StartMessageID.String())
		if err != nil {
			return nil, err
		}
		// 过滤到结束消息ID
		endIndex := -1
		for i, msg := range messages {
			if msg.ID == *req.EndMessageID {
				endIndex = i
				break
			}
		}
		if endIndex >= 0 {
			messages = messages[:endIndex+1]
		}
	} else {
		// 否则获取最新的消息
		limit := 20 // 默认获取最近20条消息
		messages, err = s.messageRepo.GetLatestMessages(ctx, req.SessionID.String(), limit)
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// buildSummaryPrompt 构建摘要生成的提示词
func (s *summaryServiceImpl) buildSummaryPrompt(messages []*model.ChatMessage, previousSummary, summaryType string, targetLength int) string {
	var prompt strings.Builder

	// 系统提示
	prompt.WriteString("你是一个专业的对话摘要助手。请根据以下对话内容生成一个简洁、准确的摘要。\n\n")

	// 摘要要求
	prompt.WriteString("摘要要求：\n")
	prompt.WriteString("1. 保留关键信息和重要观点\n")
	prompt.WriteString("2. 使用清晰、连贯的语言\n")
	prompt.WriteString("3. 避免冗余和重复\n")
	if targetLength > 0 {
		prompt.WriteString(fmt.Sprintf("4. 控制长度在约%d个Token以内\n", targetLength))
	}
	prompt.WriteString("\n")

	// 如果是增量摘要，包含前一个摘要
	if summaryType == "incremental" && previousSummary != "" {
		prompt.WriteString("前一个摘要：\n")
		prompt.WriteString(previousSummary)
		prompt.WriteString("\n\n")
		prompt.WriteString("请基于前一个摘要，总结以下新增的对话内容：\n\n")
	} else {
		prompt.WriteString("对话内容：\n\n")
	}

	// 添加对话消息
	for i, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		} else if msg.Role == "system" {
			role = "系统"
		}
		prompt.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content))
	}

	prompt.WriteString("\n请生成摘要：")

	return prompt.String()
}

// extractKeyTopics 从摘要中提取关键主题
func (s *summaryServiceImpl) extractKeyTopics(summary string) []string {
	// 简单的关键词提取算法
	// 在实际应用中，可以使用更复杂的NLP技术
	topics := []string{}

	// 分割成句子
	sentences := strings.Split(summary, "。")
	
	// 提取每个句子的主要内容（简化版）
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// 提取句子中的关键短语（这里使用简单的规则）
		words := strings.Fields(sentence)
		if len(words) > 3 {
			// 取前几个词作为主题
			topic := strings.Join(words[:min(5, len(words))], " ")
			topics = append(topics, topic)
		}
	}

	// 限制主题数量
	if len(topics) > 5 {
		topics = topics[:5]
	}

	return topics
}

// calculateQualityScore 计算摘要质量评分
func (s *summaryServiceImpl) calculateQualityScore(ctx context.Context, summary string, messages []*model.ChatMessage) float64 {
	// 简化的质量评分算法
	score := 1.0

	// 1. 长度检查（摘要不应该太短或太长）
	summaryTokens := s.tokenMgr.EstimateTokens(summary)
	originalTokens := 0
	for _, msg := range messages {
		originalTokens += s.tokenMgr.EstimateTokens(msg.Content)
	}

	compressionRate := 1.0 - float64(summaryTokens)/float64(originalTokens)
	if compressionRate < 0.3 {
		// 压缩率太低（摘要太长）
		score -= 0.2
	} else if compressionRate > 0.9 {
		// 压缩率太高（摘要太短）
		score -= 0.3
	}

	// 2. 内容完整性检查（简化版：检查是否包含关键词）
	keywordCount := 0
	for _, msg := range messages {
		// 提取消息中的关键词（简化：长度>2的词）
		words := strings.Fields(msg.Content)
		for _, word := range words {
			if len(word) > 2 && strings.Contains(summary, word) {
				keywordCount++
			}
		}
	}

	coverageRate := float64(keywordCount) / float64(len(messages)*5) // 假设每条消息平均5个关键词
	if coverageRate < 0.3 {
		score -= 0.2
	}

	// 3. 连贯性检查（简化版：检查句子数量）
	sentences := strings.Split(summary, "。")
	if len(sentences) < 2 {
		score -= 0.1
	}

	// 确保评分在0-1范围内
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// updateContextAfterSummary 更新会话上下文配置
func (s *summaryServiceImpl) updateContextAfterSummary(ctx context.Context, tenantID, sessionID, summaryID uuid.UUID) error {
	contextConfig, err := s.contextRepo.GetBySessionID(ctx, sessionID.String())
	if err != nil {
		return err
	}

	now := time.Now()
	contextConfig.LastSummaryID = &summaryID
	contextConfig.LastSummaryAt = &now

	return s.contextRepo.Update(ctx, contextConfig)
}

// evaluateDimension 评估特定维度的质量
func (s *summaryServiceImpl) evaluateDimension(ctx context.Context, dimension string, summary string, messages []*model.ChatMessage) float64 {
	switch dimension {
	case "completeness":
		// 完整性：检查摘要是否覆盖了主要信息
		return s.evaluateCompleteness(summary, messages)
	case "conciseness":
		// 简洁性：检查摘要是否简洁
		return s.evaluateConciseness(summary, messages)
	case "coherence":
		// 连贯性：检查摘要是否连贯
		return s.evaluateCoherence(summary)
	case "accuracy":
		// 准确性：检查摘要是否准确
		return s.evaluateAccuracy(summary, messages)
	default:
		return 0.5
	}
}

// evaluateCompleteness 评估完整性
func (s *summaryServiceImpl) evaluateCompleteness(summary string, messages []*model.ChatMessage) float64 {
	// 简化算法：检查关键词覆盖率
	totalKeywords := 0
	coveredKeywords := 0

	for _, msg := range messages {
		words := strings.Fields(msg.Content)
		for _, word := range words {
			if len(word) > 2 {
				totalKeywords++
				if strings.Contains(summary, word) {
					coveredKeywords++
				}
			}
		}
	}

	if totalKeywords == 0 {
		return 0.5
	}

	coverageRate := float64(coveredKeywords) / float64(totalKeywords)
	
	// 完整性评分：覆盖率在30%-70%之间为最佳
	if coverageRate < 0.3 {
		return coverageRate / 0.3 * 0.7
	} else if coverageRate > 0.7 {
		return 1.0 - (coverageRate-0.7)/0.3*0.3
	}
	return 1.0
}

// evaluateConciseness 评估简洁性
func (s *summaryServiceImpl) evaluateConciseness(summary string, messages []*model.ChatMessage) float64 {
	summaryTokens := s.tokenMgr.EstimateTokens(summary)
	originalTokens := 0
	for _, msg := range messages {
		originalTokens += s.tokenMgr.EstimateTokens(msg.Content)
	}

	if originalTokens == 0 {
		return 0.5
	}

	compressionRate := 1.0 - float64(summaryTokens)/float64(originalTokens)
	
	// 简洁性评分：压缩率在50%-80%之间为最佳
	if compressionRate < 0.5 {
		return compressionRate / 0.5 * 0.7
	} else if compressionRate > 0.8 {
		return 1.0 - (compressionRate-0.8)/0.2*0.3
	}
	return 1.0
}

// evaluateCoherence 评估连贯性
func (s *summaryServiceImpl) evaluateCoherence(summary string) float64 {
	// 简化算法：检查句子数量和长度分布
	sentences := strings.Split(summary, "。")
	validSentences := 0
	totalLength := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 5 {
			validSentences++
			totalLength += len(sentence)
		}
	}

	if validSentences == 0 {
		return 0.3
	}

	avgLength := totalLength / validSentences

	// 连贯性评分：句子数量适中（2-10句），平均长度适中（20-100字符）
	score := 1.0
	if validSentences < 2 {
		score -= 0.3
	} else if validSentences > 10 {
		score -= 0.2
	}

	if avgLength < 20 {
		score -= 0.2
	} else if avgLength > 100 {
		score -= 0.1
	}

	if score < 0 {
		score = 0
	}
	return score
}

// evaluateAccuracy 评估准确性
func (s *summaryServiceImpl) evaluateAccuracy(summary string, messages []*model.ChatMessage) float64 {
	// 简化算法：检查摘要中的内容是否都来自原始消息
	summaryWords := strings.Fields(summary)
	accurateWords := 0

	for _, word := range summaryWords {
		if len(word) <= 2 {
			continue
		}

		found := false
		for _, msg := range messages {
			if strings.Contains(msg.Content, word) {
				found = true
				break
			}
		}

		if found {
			accurateWords++
		}
	}

	if len(summaryWords) == 0 {
		return 0.5
	}

	accuracyRate := float64(accurateWords) / float64(len(summaryWords))
	return accuracyRate
}

// calculateKeyInfoCoverage 计算关键信息覆盖率
func (s *summaryServiceImpl) calculateKeyInfoCoverage(summary string, messages []*model.ChatMessage) float64 {
	// 简化算法：检查重要消息的关键词是否被覆盖
	totalImportantWords := 0
	coveredImportantWords := 0

	for _, msg := range messages {
		// 假设较长的消息更重要
		if len(msg.Content) > 50 {
			words := strings.Fields(msg.Content)
			for _, word := range words {
				if len(word) > 3 {
					totalImportantWords++
					if strings.Contains(summary, word) {
						coveredImportantWords++
					}
				}
			}
		}
	}

	if totalImportantWords == 0 {
		return 0.5
	}

	return float64(coveredImportantWords) / float64(totalImportantWords)
}

// getDimensionIssueDescription 获取维度问题描述
func (s *summaryServiceImpl) getDimensionIssueDescription(dimension string, score float64) string {
	switch dimension {
	case "completeness":
		if score < 0.5 {
			return "摘要缺少重要信息，建议增加关键内容"
		}
		return "摘要信息不够完整，建议补充更多细节"
	case "conciseness":
		if score < 0.5 {
			return "摘要过于冗长或过于简短，建议调整长度"
		}
		return "摘要长度不够理想，建议优化表达"
	case "coherence":
		if score < 0.5 {
			return "摘要缺乏连贯性，建议重新组织内容"
		}
		return "摘要连贯性有待提高，建议优化句子结构"
	case "accuracy":
		if score < 0.5 {
			return "摘要包含不准确的信息，建议核对原文"
		}
		return "摘要准确性有待提高，建议仔细检查"
	default:
		return "质量评分较低，建议改进"
	}
}

// generateImprovementSuggestions 生成改进建议
func (s *summaryServiceImpl) generateImprovementSuggestions(result *SummaryQualityResult) []string {
	suggestions := []string{}

	// 根据问题生成建议
	for _, issue := range result.Issues {
		switch issue.Dimension {
		case "completeness":
			if issue.Severity == "high" {
				suggestions = append(suggestions, "建议重新生成摘要，确保包含所有关键信息")
			} else {
				suggestions = append(suggestions, "建议补充遗漏的重要细节")
			}
		case "conciseness":
			if issue.Score < 0.5 {
				suggestions = append(suggestions, "建议调整摘要长度，使其更加简洁或详细")
			}
		case "coherence":
			suggestions = append(suggestions, "建议优化句子结构，提高摘要的连贯性")
		case "accuracy":
			suggestions = append(suggestions, "建议仔细核对摘要内容，确保准确性")
		}
	}

	// 根据关键信息覆盖率生成建议
	if result.KeyInfoCoverage < 0.5 {
		suggestions = append(suggestions, "关键信息覆盖率较低，建议增加重要内容的描述")
	}

	// 如果没有具体建议，提供通用建议
	if len(suggestions) == 0 && !result.Passed {
		suggestions = append(suggestions, "建议重新生成摘要以提高质量")
	}

	return suggestions
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
