package service

import (
	"context"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
)

// conversationContextServiceImpl ConversationContextService 的实现
// 用于构建对话历史，实现"记忆"功能
type conversationContextServiceImpl struct {
	messageRepo repository.MessageRepository
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
}

// NewConversationContextService 创建 ConversationContextService 实例
func NewConversationContextService(
	messageRepo repository.MessageRepository,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) ai.ConversationContextService {
	return &conversationContextServiceImpl{
		messageRepo: messageRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// BuildConversationHistory 构建对话历史
// 根据会话ID获取历史消息，用于多轮对话
func (s *conversationContextServiceImpl) BuildConversationHistory(
	ctx context.Context,
	sessionID string,
	maxMessages int,
) ([]*model.ChatHistoryMessage, error) {
	// 设置默认值
	if maxMessages <= 0 {
		maxMessages = 20 // 默认获取最近20条消息
	}

	// 验证会话访问权限
	if err := s.validateSessionAccess(ctx, sessionID); err != nil {
		return nil, err
	}

	// 从数据库获取历史消息
	messages, err := s.messageRepo.GetLatestMessages(ctx, sessionID, maxMessages)
	if err != nil {
		logger.ErrorContext(ctx, "获取历史消息失败", logger.Fields{
			"sessionId":   sessionID,
			"maxMessages": maxMessages,
			"error":       err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "获取历史消息失败", err)
	}

	// 转换为 ChatHistoryMessage 格式
	// 注意：需要过滤空内容的 assistant 消息，因为 Azure AI 要求 assistant 消息必须包含文本内容或工具调用
	history := make([]*model.ChatHistoryMessage, 0, len(messages))
	var skippedEmptyAssistant int
	for _, msg := range messages {
		// 跳过空内容的 assistant 消息
		// Azure AI 要求 assistant 消息必须包含文本内容或工具调用
		// 空的 assistant 消息可能是流式生成中断或失败导致的
		if msg.Role == "assistant" && msg.Content == "" {
			skippedEmptyAssistant++
			continue
		}
		history = append(history, &model.ChatHistoryMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	logger.DebugContext(ctx, "成功构建对话历史", logger.Fields{
		"sessionId":             sessionID,
		"messageCount":          len(history),
		"skippedEmptyAssistant": skippedEmptyAssistant,
	})

	return history, nil
}

// validateSessionAccess 验证会话访问权限
func (s *conversationContextServiceImpl) validateSessionAccess(ctx context.Context, sessionID string) error {
	// 获取 JWT 声明
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		logger.WarnContext(ctx, "未找到JWT声明", logger.Fields{
			"sessionId": sessionID,
		})
		return errors.New(errors.CodeUnauthorized, "未认证")
	}

	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		logger.ErrorContext(ctx, "查询会话失败", logger.Fields{
			"sessionId": sessionID,
			"error":     err.Error(),
		})
		return errors.Wrap(errors.CodeNotFound, "会话不存在", err)
	}

	// 平台管理员可以访问所有会话
	if hasSystemAdminRole(claims) {
		return nil
	}

	// 获取会话所属用户的租户ID
	userIDStr := session.UserID.String()
	sessionUser, err := s.userRepo.GetByIDOnly(ctx, userIDStr)
	if err != nil {
		logger.ErrorContext(ctx, "获取会话用户信息失败", logger.Fields{
			"sessionId": sessionID,
			"userId":    userIDStr,
			"error":     err.Error(),
		})
		return errors.Wrap(errors.CodeInternalError, "获取用户信息失败", err)
	}

	// 验证租户ID匹配
	if claims.TenantID != sessionUser.TenantID.String() {
		logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话", logger.Fields{
			"userId":          claims.Subject,
			"userTenantId":    claims.TenantID,
			"sessionId":       sessionID,
			"sessionTenantId": sessionUser.TenantID.String(),
		})
		return errors.New(errors.CodeForbidden, "权限不足：无法访问其他租户的会话")
	}

	return nil
}

// hasSystemAdminRole 检查用户是否具有系统管理员角色
func hasSystemAdminRole(claims *model.JWTClaims) bool {
	if claims == nil || claims.Roles == nil {
		return false
	}

	for _, r := range claims.Roles {
		if r == model.RoleSystemAdmin {
			return true
		}
	}

	return false
}
