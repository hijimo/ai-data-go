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

	// ShouldGenerateSummary 判断是否需要生成摘要
	ShouldGenerateSummary(ctx context.Context, sessionID string) (bool, error)
}

// summaryService 摘要业务逻辑实现
type summaryService struct {
	summaryRepo  repository.SummaryRepository
	messageRepo  repository.MessageRepository
	sessionRepo  repository.SessionRepository
	aiService    ai.AIService
	config       *config.Config
	logger       logger.Logger
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
		messages, err = s.messageRepo.GetMessagesAfter(ctx, sessionID, latestSummary.LastMessageID)
		if err != nil {
			s.logger.Error("获取摘要后的消息失败", map[string]interface{}{
				"sessionId":     sessionID,
				"lastMessageId": latestSummary.LastMessageID,
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
		SessionID:     sessionID,
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
	messagesAfterSummary, err := s.messageRepo.GetMessagesAfter(ctx, sessionID, latestSummary.LastMessageID)
	if err != nil {
		s.logger.Error("获取摘要后的消息失败", map[string]interface{}{
			"sessionId":     sessionID,
			"lastMessageId": latestSummary.LastMessageID,
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
