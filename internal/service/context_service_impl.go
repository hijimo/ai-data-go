package service

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
)

// contextServiceImpl ContextService 的实现
type contextServiceImpl struct {
	sessionRepo  repository.SessionRepository
	messageRepo  repository.MessageRepository
	memoryRepo   repository.MemoryRepository
	contextRepo  repository.ContextRepository
	summaryRepo  repository.SummaryRepository
	userRepo     repository.UserRepository
	vectorSvc    ai.VectorService
	tokenMgr     TokenManager
}

// NewContextService 创建 ContextService 实例
func NewContextService(
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	memoryRepo repository.MemoryRepository,
	contextRepo repository.ContextRepository,
	summaryRepo repository.SummaryRepository,
	userRepo repository.UserRepository,
	vectorSvc ai.VectorService,
	tokenMgr TokenManager,
) ContextService {
	return &contextServiceImpl{
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		memoryRepo:   memoryRepo,
		contextRepo:  contextRepo,
		summaryRepo:  summaryRepo,
		userRepo:     userRepo,
		vectorSvc:    vectorSvc,
		tokenMgr:     tokenMgr,
	}
}


// BuildContext 构建会话上下文
func (s *contextServiceImpl) BuildContext(
	ctx context.Context,
	req BuildContextRequest,
) (*ContextResult, error) {
	// 1. 权限验证
	if err := s.validateAccess(ctx, req.SessionID); err != nil {
		return nil, err
	}

	// 2. 获取会话信息（用于验证会话存在）
	_, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		logger.ErrorContext(ctx, "获取会话信息失败", logger.Fields{
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeNotFound, "会话不存在", err)
	}

	// 3. 获取短期记忆（最近N条消息）
	shortTermWindow := req.ShortTermWindow
	if shortTermWindow <= 0 {
		shortTermWindow = 10 // 默认10条
	}

	messages, err := s.messageRepo.GetLatestMessages(ctx, req.SessionID, shortTermWindow)
	if err != nil {
		logger.ErrorContext(ctx, "获取短期记忆失败", logger.Fields{
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "获取短期记忆失败", err)
	}

	// 4. 获取长期记忆（如果启用且有查询）
	var memories []*model.ConversationMemory
	if req.IncludeLongTerm && req.UserQuery != "" {
		// 生成查询向量
		embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.UserQuery)
		if err != nil {
			// 记录错误但不中断流程
			logger.WarnContext(ctx, "生成查询向量失败", logger.Fields{
				"session_id": req.SessionID,
				"error":      err.Error(),
			})
		} else {
			// TODO: 执行向量检索（需要集成 Qdrant 客户端）
			// 这里暂时返回空列表
			logger.DebugContext(ctx, "向量检索功能待实现", logger.Fields{
				"session_id":     req.SessionID,
				"embedding_dims": len(embedding),
			})
		}
	}

	// 5. 获取摘要（如果启用）
	var summary *model.ConversationSummary
	if req.IncludeSummary {
		// 获取租户ID
		claims, ok := authservice.GetJWTClaimsFromContext(ctx)
		if ok && claims != nil {
			tenantUUID, err := uuid.Parse(claims.TenantID)
			if err == nil {
				sessionUUID, err := uuid.Parse(req.SessionID)
				if err == nil {
					summary, err = s.summaryRepo.GetLatestBySessionID(ctx, tenantUUID, sessionUUID)
					if err != nil && err != gorm.ErrRecordNotFound {
						logger.WarnContext(ctx, "获取摘要失败", logger.Fields{
							"session_id": req.SessionID,
							"error":      err.Error(),
						})
					}
				}
			}
		}
	}

	// 6. 计算 Token 数量
	totalTokens, err := s.tokenMgr.CalculateContextTokens(ctx, messages, memories, summary)
	if err != nil {
		logger.ErrorContext(ctx, "计算Token数量失败", logger.Fields{
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "计算Token数量失败", err)
	}

	// 7. Token 优化（如果超限）
	if totalTokens > req.MaxTokens {
		logger.InfoContext(ctx, "Token超限，开始优化", logger.Fields{
			"session_id":    req.SessionID,
			"total_tokens":  totalTokens,
			"max_tokens":    req.MaxTokens,
			"message_count": len(messages),
		})

		// 使用 OptimizeContext 进行优化
		optimizeReq := OptimizeContextRequest{
			Context: &ContextResult{
				SessionID:         req.SessionID,
				Summary:           summary,
				LongTermMemories:  memories,
				ShortTermMessages: messages,
				TotalTokens:       totalTokens,
				Strategy:          req.Strategy,
			},
			TargetTokens:    req.MaxTokens,
			Strategy:        req.Strategy,
			PreserveSummary: req.IncludeSummary,
		}

		optimized, err := s.OptimizeContext(ctx, optimizeReq)
		if err != nil {
			logger.ErrorContext(ctx, "优化上下文失败", logger.Fields{
				"session_id": req.SessionID,
				"error":      err.Error(),
			})
			return nil, err
		}

		messages = optimized.ShortTermMessages
		memories = optimized.LongTermMemories
		summary = optimized.Summary
		totalTokens = optimized.TotalTokens
	}

	// 8. 计算质量评分
	qualityScore := s.calculateQualityScore(messages, memories, summary)

	// 9. 构建结果
	result := &ContextResult{
		SessionID:         req.SessionID,
		Summary:           summary,
		LongTermMemories:  memories,
		ShortTermMessages: messages,
		TotalTokens:       totalTokens,
		Strategy:          req.Strategy,
		QualityScore:      qualityScore,
	}

	logger.InfoContext(ctx, "上下文构建成功", logger.Fields{
		"session_id":     req.SessionID,
		"total_tokens":   totalTokens,
		"message_count":  len(messages),
		"memory_count":   len(memories),
		"has_summary":    summary != nil,
		"quality_score":  qualityScore,
	})

	return result, nil
}


// OptimizeContext 优化上下文Token使用
func (s *contextServiceImpl) OptimizeContext(
	ctx context.Context,
	req OptimizeContextRequest,
) (*ContextResult, error) {
	if req.Context == nil {
		return nil, errors.New(errors.CodeBadRequest, "上下文不能为空")
	}

	logger.InfoContext(ctx, "开始优化上下文", logger.Fields{
		"session_id":      req.Context.SessionID,
		"current_tokens":  req.Context.TotalTokens,
		"target_tokens":   req.TargetTokens,
		"strategy":        req.Strategy,
		"preserve_summary": req.PreserveSummary,
	})

	messages := req.Context.ShortTermMessages
	memories := req.Context.LongTermMemories
	summary := req.Context.Summary

	// 根据策略进行优化
	switch req.Strategy {
	case "aggressive":
		// 激进策略：优先保留最新消息，大幅削减长期记忆
		messages, memories = s.optimizeAggressive(ctx, messages, memories, req.TargetTokens)
	case "conservative":
		// 保守策略：尽量保留所有信息，均衡削减
		messages, memories = s.optimizeConservative(ctx, messages, memories, req.TargetTokens)
	default:
		// 平衡策略（默认）
		messages, memories = s.optimizeBalanced(ctx, messages, memories, req.TargetTokens)
	}

	// 如果不保留摘要，清除摘要
	if !req.PreserveSummary {
		summary = nil
	}

	// 重新计算 Token 数量
	totalTokens, err := s.tokenMgr.CalculateContextTokens(ctx, messages, memories, summary)
	if err != nil {
		logger.ErrorContext(ctx, "重新计算Token数量失败", logger.Fields{
			"session_id": req.Context.SessionID,
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "重新计算Token数量失败", err)
	}

	// 重新计算质量评分
	qualityScore := s.calculateQualityScore(messages, memories, summary)

	result := &ContextResult{
		SessionID:         req.Context.SessionID,
		Summary:           summary,
		LongTermMemories:  memories,
		ShortTermMessages: messages,
		TotalTokens:       totalTokens,
		Strategy:          req.Strategy,
		QualityScore:      qualityScore,
	}

	logger.InfoContext(ctx, "上下文优化完成", logger.Fields{
		"session_id":     req.Context.SessionID,
		"original_tokens": req.Context.TotalTokens,
		"optimized_tokens": totalTokens,
		"message_count":  len(messages),
		"memory_count":   len(memories),
		"quality_score":  qualityScore,
	})

	return result, nil
}


// GetContextConfig 获取上下文配置
func (s *contextServiceImpl) GetContextConfig(
	ctx context.Context,
	sessionID string,
) (*model.ConversationContext, error) {
	// 权限验证
	if err := s.validateAccess(ctx, sessionID); err != nil {
		return nil, err
	}

	// 获取配置
	config, err := s.contextRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "上下文配置不存在")
		}
		logger.ErrorContext(ctx, "获取上下文配置失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "获取上下文配置失败", err)
	}

	return config, nil
}

// UpdateContextConfig 更新上下文配置
func (s *contextServiceImpl) UpdateContextConfig(
	ctx context.Context,
	sessionID string,
	config *model.ConversationContext,
) error {
	// 权限验证
	if err := s.validateAccess(ctx, sessionID); err != nil {
		return err
	}

	// 验证配置参数
	if config.MaxTokens <= 0 {
		return errors.New(errors.CodeBadRequest, "MaxTokens 必须大于0")
	}

	if config.ShortTermWindow <= 0 {
		return errors.New(errors.CodeBadRequest, "ShortTermWindow 必须大于0")
	}

	// 更新配置
	if err := s.contextRepo.Update(ctx, config); err != nil {
		logger.ErrorContext(ctx, "更新上下文配置失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return errors.Wrap(errors.CodeInternalError, "更新上下文配置失败", err)
	}

	logger.InfoContext(ctx, "上下文配置更新成功", logger.Fields{
		"session_id":        sessionID,
		"max_tokens":        config.MaxTokens,
		"strategy":          config.Strategy,
		"short_term_window": config.ShortTermWindow,
	})

	return nil
}


// validateAccess 验证租户访问权限
func (s *contextServiceImpl) validateAccess(ctx context.Context, sessionID string) error {
	// 获取 JWT 声明
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		logger.WarnContext(ctx, "未找到JWT声明", logger.Fields{
			"session_id": sessionID,
		})
		return errors.New(errors.CodeUnauthorized, "未认证")
	}

	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.CodeNotFound, "会话不存在")
		}
		logger.ErrorContext(ctx, "查询会话失败", logger.Fields{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return errors.Wrap(errors.CodeInternalError, "查询会话失败", err)
	}

	// 平台管理员可以访问所有会话
	if hasRole(claims, model.RoleSystemAdmin) {
		return nil
	}

	// 获取会话所属用户的租户ID
	userUUID := session.UserID
	userIDStr := userUUID.String()
	
	// 使用 GetByIDOnly 因为我们需要获取用户的租户ID来进行验证
	sessionUser, err := s.userRepo.GetByIDOnly(ctx, userIDStr)
	if err != nil {
		logger.ErrorContext(ctx, "获取会话用户信息失败", logger.Fields{
			"session_id": sessionID,
			"user_id":    userIDStr,
			"error":      err.Error(),
		})
		return errors.Wrap(errors.CodeInternalError, "获取用户信息失败", err)
	}

	// 验证租户ID匹配
	if claims.TenantID != sessionUser.TenantID.String() {
		logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话", logger.Fields{
			"user_id":           claims.Subject,
			"user_tenant_id":    claims.TenantID,
			"session_id":        sessionID,
			"session_tenant_id": sessionUser.TenantID.String(),
		})
		return errors.New(errors.CodeForbidden, "权限不足：无法访问其他租户的会话")
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


// optimizeAggressive 激进优化策略
// 优先保留最新消息，大幅削减长期记忆
func (s *contextServiceImpl) optimizeAggressive(
	ctx context.Context,
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	targetTokens int,
) ([]*model.ChatMessage, []*model.ConversationMemory) {
	// 首先清空长期记忆
	memories = []*model.ConversationMemory{}

	// 计算当前Token数
	currentTokens, _ := s.tokenMgr.CalculateContextTokens(ctx, messages, memories, nil)

	// 如果还是超限，从最旧的消息开始删除
	for currentTokens > targetTokens && len(messages) > 1 {
		messages = messages[1:] // 删除最旧的消息
		currentTokens, _ = s.tokenMgr.CalculateContextTokens(ctx, messages, memories, nil)
	}

	logger.DebugContext(ctx, "激进优化完成", logger.Fields{
		"message_count": len(messages),
		"memory_count":  len(memories),
		"tokens":        currentTokens,
	})

	return messages, memories
}

// optimizeBalanced 平衡优化策略
// 均衡削减消息和记忆
func (s *contextServiceImpl) optimizeBalanced(
	ctx context.Context,
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	targetTokens int,
) ([]*model.ChatMessage, []*model.ConversationMemory) {
	// 先削减一半的长期记忆
	if len(memories) > 2 {
		memories = memories[:len(memories)/2]
	}

	// 计算当前Token数
	currentTokens, _ := s.tokenMgr.CalculateContextTokens(ctx, messages, memories, nil)

	// 如果还是超限，交替删除消息和记忆
	deleteMessage := true
	for currentTokens > targetTokens && (len(messages) > 1 || len(memories) > 0) {
		if deleteMessage && len(messages) > 1 {
			messages = messages[1:] // 删除最旧的消息
		} else if len(memories) > 0 {
			memories = memories[1:] // 删除最旧的记忆
		}

		deleteMessage = !deleteMessage
		currentTokens, _ = s.tokenMgr.CalculateContextTokens(ctx, messages, memories, nil)
	}

	logger.DebugContext(ctx, "平衡优化完成", logger.Fields{
		"message_count": len(messages),
		"memory_count":  len(memories),
		"tokens":        currentTokens,
	})

	return messages, memories
}

// optimizeConservative 保守优化策略
// 尽量保留所有信息，均衡削减
func (s *contextServiceImpl) optimizeConservative(
	ctx context.Context,
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	targetTokens int,
) ([]*model.ChatMessage, []*model.ConversationMemory) {
	// 计算当前Token数
	currentTokens, _ := s.tokenMgr.CalculateContextTokens(ctx, messages, memories, nil)

	// 计算需要削减的比例
	if currentTokens <= targetTokens {
		return messages, memories
	}

	reductionRatio := float64(targetTokens) / float64(currentTokens)

	// 按比例削减消息和记忆
	targetMessageCount := int(float64(len(messages)) * reductionRatio)
	if targetMessageCount < 1 {
		targetMessageCount = 1
	}

	targetMemoryCount := int(float64(len(memories)) * reductionRatio)

	// 保留最新的消息
	if len(messages) > targetMessageCount {
		messages = messages[len(messages)-targetMessageCount:]
	}

	// 保留最新的记忆
	if len(memories) > targetMemoryCount {
		memories = memories[len(memories)-targetMemoryCount:]
	}

	logger.DebugContext(ctx, "保守优化完成", logger.Fields{
		"message_count": len(messages),
		"memory_count":  len(memories),
		"reduction_ratio": reductionRatio,
	})

	return messages, memories
}


// calculateQualityScore 计算上下文质量评分
// 评分范围：0-1，越高表示质量越好
func (s *contextServiceImpl) calculateQualityScore(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
) float64 {
	var score float64 = 0.0

	// 1. 消息数量评分（权重：0.4）
	// 有足够的消息历史（5-20条）得分最高
	messageCount := len(messages)
	var messageScore float64
	if messageCount >= 5 && messageCount <= 20 {
		messageScore = 1.0
	} else if messageCount > 20 {
		messageScore = 0.8
	} else if messageCount >= 3 {
		messageScore = 0.6
	} else if messageCount >= 1 {
		messageScore = 0.4
	} else {
		messageScore = 0.0
	}
	score += messageScore * 0.4

	// 2. 长期记忆评分（权重：0.3）
	// 有相关的长期记忆可以提供更好的上下文
	memoryCount := len(memories)
	var memoryScore float64
	if memoryCount >= 3 && memoryCount <= 10 {
		memoryScore = 1.0
	} else if memoryCount > 10 {
		memoryScore = 0.8
	} else if memoryCount >= 1 {
		memoryScore = 0.5
	} else {
		memoryScore = 0.0
	}
	score += memoryScore * 0.3

	// 3. 摘要评分（权重：0.3）
	// 有高质量的摘要可以提供更好的上下文压缩
	var summaryScore float64
	if summary != nil {
		if summary.QualityScore != nil {
			summaryScore = *summary.QualityScore
		} else {
			summaryScore = 0.7 // 默认评分
		}
	} else {
		summaryScore = 0.0
	}
	score += summaryScore * 0.3

	return score
}
