package service

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/errors"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ContextService 上下文服务接口
type ContextService interface {
	// BuildContext 构建上下文
	BuildContext(ctx context.Context, req *BuildContextRequest) (*ContextResult, error)

	// OptimizeContext 优化上下文
	OptimizeContext(ctx context.Context, req *OptimizeContextRequest) (*ContextResult, error)

	// GetContextConfig 获取上下文配置
	GetContextConfig(ctx context.Context, sessionID string) (*model.ConversationContext, error)

	// UpdateContextConfig 更新上下文配置
	UpdateContextConfig(ctx context.Context, sessionID string, config *model.ConversationContext) error
}

// BuildContextRequest 构建上下文请求
type BuildContextRequest struct {
	SessionID       string
	UserQuery       string
	MaxTokens       int
	Strategy        string
	IncludeSummary  bool
	IncludeLongTerm bool
	ShortTermWindow int
}

// ContextResult 上下文结果
type ContextResult struct {
	SessionID         string
	Summary           *SummaryResult
	LongTermMemories  []*MemoryResult
	ShortTermMessages []*MessageResult
	TotalTokens       int
	Strategy          string
	QualityScore      float64
	QualityLoss       float64  // 质量损失评分（仅用于优化结果）
	Operations        []string // 执行的优化操作列表（仅用于优化结果）
}

// SummaryResult 摘要结果
type SummaryResult struct {
	Content    string
	TokenCount int
	CreatedAt  string
	Coverage   string
}

// MemoryResult 记忆结果
type MemoryResult struct {
	ID         string
	Content    string
	TokenCount int
	Importance float32
	Similarity float32
	CreatedAt  string
}

// MessageResult 消息结果
type MessageResult struct {
	ID         string
	Role       string
	Content    string
	TokenCount int
	CreatedAt  string
}

// OptimizeContextRequest 优化上下文请求
type OptimizeContextRequest struct {
	Context         *ContextResult
	TargetTokens    int
	Strategy        string
	PreserveSummary bool
}

// contextServiceImpl 上下文服务实现
type contextServiceImpl struct {
	sessionRepo  repository.SessionRepository
	messageRepo  repository.MessageRepository
	memoryRepo   repository.GenkitMemoryRepository
	contextRepo  repository.GenkitContextRepository
	vectorSvc    VectorService
	tokenMgr     TokenManager
	cache        CacheService
	log          logger.Logger
}

// NewContextService 创建上下文服务实例
func NewContextService(
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	memoryRepo repository.GenkitMemoryRepository,
	contextRepo repository.GenkitContextRepository,
	vectorSvc VectorService,
	tokenMgr TokenManager,
	cache CacheService,
	log logger.Logger,
) ContextService {
	return &contextServiceImpl{
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		memoryRepo:   memoryRepo,
		contextRepo:  contextRepo,
		vectorSvc:    vectorSvc,
		tokenMgr:     tokenMgr,
		cache:        cache,
		log:          log,
	}
}

// BuildContext 构建上下文
func (s *contextServiceImpl) BuildContext(
	ctx context.Context,
	req *BuildContextRequest,
) (*ContextResult, error) {
	startTime := time.Now()

	// 1. 权限验证
	if err := s.validateAccess(ctx, req.SessionID); err != nil {
		return nil, err
	}

	// 2. 尝试从缓存获取
	cacheKey := fmt.Sprintf("context:%s:%s", req.SessionID, s.cache.HashQuery(req.UserQuery))
	var cached *ContextResult
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && cached != nil {
		s.log.DebugContext(ctx, "从缓存获取上下文", logger.Fields{
			"sessionID": req.SessionID,
			"duration":  time.Since(startTime).Milliseconds(),
		})
		return cached, nil
	}

	// 3. 获取短期记忆（最近的消息）
	messages, err := s.messageRepo.GetLatestMessages(
		ctx,
		req.SessionID,
		req.ShortTermWindow,
	)
	if err != nil {
		return nil, fmt.Errorf("获取短期记忆失败: %w", err)
	}

	// 4. 获取长期记忆（如果启用）
	var memories []*model.ConversationMemory
	if req.IncludeLongTerm && req.UserQuery != "" {
		// 生成查询向量
		embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.UserQuery)
		if err != nil {
			// 记录错误但不中断流程
			s.log.WarnContext(ctx, "生成查询向量失败", logger.Fields{
				"error":     err.Error(),
				"sessionID": req.SessionID,
			})
		} else {
			// 执行向量检索
			// 将 []float32 转换为 pgvector.Vector
			vector := pgvector.NewVector(embedding)
			
			memories, err = s.memoryRepo.SearchByVector(
				ctx,
				req.SessionID,
				vector,
				5,
				0.7,
			)
			if err != nil {
				s.log.WarnContext(ctx, "向量检索失败", logger.Fields{
					"error":     err.Error(),
					"sessionID": req.SessionID,
				})
			}
		}
	}

	// 5. 获取摘要（如果启用）
	var summary *model.ConversationSummary
	if req.IncludeSummary {
		summary, err = s.contextRepo.GetLatestSummary(ctx, req.SessionID)
		if err != nil {
			// 摘要不存在是正常情况，不记录错误
			s.log.DebugContext(ctx, "未找到摘要", logger.Fields{
				"sessionID": req.SessionID,
			})
		}
	}

	// 6. 计算 Token 数量
	totalTokens := s.tokenMgr.CalculateContextTokens(messages, memories, summary)

	// 7. Token 优化（如果超限）
	if totalTokens > req.MaxTokens {
		s.log.InfoContext(ctx, "上下文Token超限，执行优化", logger.Fields{
			"sessionID":   req.SessionID,
			"totalTokens": totalTokens,
			"maxTokens":   req.MaxTokens,
		})

		messages, memories, summary = s.optimizeContextInternal(
			messages,
			memories,
			summary,
			req.MaxTokens,
		)
		totalTokens = s.tokenMgr.CalculateContextTokens(messages, memories, summary)
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

	// 10. 异步缓存结果
	go func() {
		cacheCtx := context.Background()
		if err := s.cache.Set(cacheCtx, cacheKey, result, 5*time.Minute); err != nil {
			s.log.WarnContext(cacheCtx, "缓存上下文失败", logger.Fields{
				"error":     err.Error(),
				"sessionID": req.SessionID,
			})
		}
	}()

	s.log.InfoContext(ctx, "上下文构建完成", logger.Fields{
		"sessionID":      req.SessionID,
		"totalTokens":    totalTokens,
		"qualityScore":   qualityScore,
		"messagesCount":  len(messages),
		"memoriesCount":  len(memories),
		"hasSummary":     summary != nil,
		"duration":       time.Since(startTime).Milliseconds(),
	})

	return result, nil
}

// OptimizeContext 优化上下文
func (s *contextServiceImpl) OptimizeContext(
	ctx context.Context,
	req *OptimizeContextRequest,
) (*ContextResult, error) {
	if req.Context == nil {
		return nil, errors.NewBadRequestError("上下文不能为空")
	}

	// 记录原始质量评分
	originalQualityScore := req.Context.QualityScore

	// 记录执行的操作
	operations := make([]string, 0)

	// 根据策略执行优化
	var messages []*model.ChatMessage
	var memories []*model.ConversationMemory
	var summary *model.ConversationSummary

	switch req.Strategy {
	case "aggressive":
		// 激进策略：大幅减少长期记忆，保留最少的短期消息
		messages, memories, summary, operations = s.optimizeAggressive(
			req.Context.ShortTermMessages,
			req.Context.LongTermMemories,
			req.Context.Summary,
			req.TargetTokens,
			req.PreserveSummary,
		)

	case "balanced":
		// 平衡策略：均衡减少各部分内容
		messages, memories, summary, operations = s.optimizeBalanced(
			req.Context.ShortTermMessages,
			req.Context.LongTermMemories,
			req.Context.Summary,
			req.TargetTokens,
			req.PreserveSummary,
		)

	case "conservative":
		// 保守策略：优先移除低相关性长期记忆，尽量保留短期消息
		messages, memories, summary, operations = s.optimizeConservative(
			req.Context.ShortTermMessages,
			req.Context.LongTermMemories,
			req.Context.Summary,
			req.TargetTokens,
			req.PreserveSummary,
		)

	default:
		return nil, errors.NewBadRequestError(fmt.Sprintf("不支持的优化策略: %s", req.Strategy))
	}

	// 重新计算Token和质量评分
	totalTokens := s.tokenMgr.CalculateContextTokens(messages, memories, summary)
	qualityScore := s.calculateQualityScore(messages, memories, summary)

	// 计算质量损失
	qualityLoss := originalQualityScore - qualityScore
	if qualityLoss < 0 {
		qualityLoss = 0
	}

	// 如果质量损失超过30%，记录警告
	if qualityLoss > 0.3 {
		s.log.WarnContext(ctx, "上下文优化导致较大质量损失", logger.Fields{
			"sessionID":           req.Context.SessionID,
			"originalQuality":     originalQualityScore,
			"optimizedQuality":    qualityScore,
			"qualityLoss":         qualityLoss,
			"strategy":            req.Strategy,
		})
	}

	result := &ContextResult{
		SessionID:         req.Context.SessionID,
		Summary:           summary,
		LongTermMemories:  memories,
		ShortTermMessages: messages,
		TotalTokens:       totalTokens,
		Strategy:          req.Strategy,
		QualityScore:      qualityScore,
		QualityLoss:       qualityLoss,
		Operations:        operations,
	}

	s.log.InfoContext(ctx, "上下文优化完成", logger.Fields{
		"sessionID":       req.Context.SessionID,
		"originalTokens":  req.Context.TotalTokens,
		"optimizedTokens": totalTokens,
		"targetTokens":    req.TargetTokens,
		"qualityScore":    qualityScore,
		"qualityLoss":     qualityLoss,
		"operations":      len(operations),
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

	config, err := s.contextRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取上下文配置失败: %w", err)
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

	// 确保SessionID匹配
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return errors.NewBadRequestError("无效的会话ID")
	}
	config.SessionID = sessionUUID

	if err := s.contextRepo.Update(ctx, config); err != nil {
		return fmt.Errorf("更新上下文配置失败: %w", err)
	}

	// 清除相关缓存
	cachePattern := fmt.Sprintf("context:%s:*", sessionID)
	if err := s.cache.DeletePattern(ctx, cachePattern); err != nil {
		s.log.WarnContext(ctx, "清除上下文缓存失败", logger.Fields{
			"error":     err.Error(),
			"sessionID": sessionID,
		})
	}

	s.log.InfoContext(ctx, "上下文配置已更新", logger.Fields{
		"sessionID": sessionID,
	})

	return nil
}

// validateAccess 验证访问权限
func (s *contextServiceImpl) validateAccess(ctx context.Context, sessionID string) error {
	// 从上下文获取用户ID
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return errors.NewUnauthorizedError("未认证")
	}

	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.NewNotFoundError("会话不存在")
	}

	// 验证会话所有权
	if session.UserID.String() != userID {
		s.log.WarnContext(ctx, "权限验证失败：尝试访问其他用户的会话", logger.Fields{
			"userID":    userID,
			"sessionID": sessionID,
			"ownerID":   session.UserID.String(),
		})
		return errors.NewForbiddenError("权限不足：无法访问其他用户的会话")
	}

	return nil
}

// calculateQualityScore 计算质量评分
func (s *contextServiceImpl) calculateQualityScore(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
) float64 {
	var score float64 = 0.0
	var weights float64 = 0.0

	// 短期消息评分（权重：0.4）
	if len(messages) > 0 {
		messageScore := float64(len(messages)) / 10.0 // 假设10条消息为满分
		if messageScore > 1.0 {
			messageScore = 1.0
		}
		score += messageScore * 0.4
		weights += 0.4
	}

	// 长期记忆评分（权重：0.3）
	if len(memories) > 0 {
		// 基于记忆数量和平均重要性
		memoryScore := float64(len(memories)) / 5.0 // 假设5条记忆为满分
		if memoryScore > 1.0 {
			memoryScore = 1.0
		}

		// 计算平均重要性
		var totalImportance float32 = 0.0
		for _, mem := range memories {
			totalImportance += mem.Importance
		}
		avgImportance := float64(totalImportance) / float64(len(memories))

		// 综合评分
		memoryScore = (memoryScore + avgImportance) / 2.0
		score += memoryScore * 0.3
		weights += 0.3
	}

	// 摘要评分（权重：0.3）
	if summary != nil {
		summaryScore := 1.0
		if summary.QualityScore != nil {
			summaryScore = *summary.QualityScore
		}
		score += summaryScore * 0.3
		weights += 0.3
	}

	// 归一化评分
	if weights > 0 {
		score = score / weights
	}

	// 确保评分在 0-1 范围内
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// optimizeContextInternal 内部优化上下文方法
func (s *contextServiceImpl) optimizeContextInternal(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
	targetTokens int,
) ([]*model.ChatMessage, []*model.ConversationMemory, *model.ConversationSummary) {
	currentTokens := s.tokenMgr.CalculateContextTokens(messages, memories, summary)

	// 如果已经在目标范围内，直接返回
	if currentTokens <= targetTokens {
		return messages, memories, summary
	}

	// 策略1: 减少长期记忆数量
	if len(memories) > 0 {
		// 保留重要性最高的记忆
		targetMemoryCount := len(memories) * targetTokens / currentTokens
		if targetMemoryCount < 1 {
			targetMemoryCount = 1
		}

		// 按重要性排序并截取
		sortedMemories := make([]*model.ConversationMemory, len(memories))
		copy(sortedMemories, memories)

		// 简单排序：按重要性降序
		for i := 0; i < len(sortedMemories)-1; i++ {
			for j := i + 1; j < len(sortedMemories); j++ {
				if sortedMemories[i].Importance < sortedMemories[j].Importance {
					sortedMemories[i], sortedMemories[j] = sortedMemories[j], sortedMemories[i]
				}
			}
		}

		memories = sortedMemories[:targetMemoryCount]
		currentTokens = s.tokenMgr.CalculateContextTokens(messages, memories, summary)
	}

	// 策略2: 如果还是超限，减少短期消息数量
	if currentTokens > targetTokens && len(messages) > 2 {
		// 至少保留最后2条消息（用户问题和AI回答）
		targetMessageCount := len(messages) * targetTokens / currentTokens
		if targetMessageCount < 2 {
			targetMessageCount = 2
		}

		// 保留最新的消息
		messages = messages[len(messages)-targetMessageCount:]
	}

	return messages, memories, summary
}

// optimizeAggressive 激进优化策略
func (s *contextServiceImpl) optimizeAggressive(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
	targetTokens int,
	preserveSummary bool,
) ([]*model.ChatMessage, []*model.ConversationMemory, *model.ConversationSummary, []string) {
	operations := []string{}

	// 激进策略：大幅减少长期记忆
	if len(memories) > 2 {
		memories = memories[:2]
		operations = append(operations, "大幅减少长期记忆至2条")
	}

	// 减少短期消息
	if len(messages) > 5 {
		messages = messages[len(messages)-5:]
		operations = append(operations, "减少短期消息至最近5条")
	}

	// 如果不保留摘要，移除摘要
	if !preserveSummary {
		summary = nil
		operations = append(operations, "移除摘要")
	}

	return messages, memories, summary, operations
}

// optimizeBalanced 平衡优化策略
func (s *contextServiceImpl) optimizeBalanced(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
	targetTokens int,
	preserveSummary bool,
) ([]*model.ChatMessage, []*model.ConversationMemory, *model.ConversationSummary, []string) {
	operations := []string{}

	// 平衡策略：均衡减少
	if len(memories) > 5 {
		memories = memories[:5]
		operations = append(operations, "减少长期记忆至5条")
	}

	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
		operations = append(operations, "减少短期消息至最近10条")
	}

	return messages, memories, summary, operations
}

// optimizeConservative 保守优化策略
func (s *contextServiceImpl) optimizeConservative(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
	targetTokens int,
	preserveSummary bool,
) ([]*model.ChatMessage, []*model.ConversationMemory, *model.ConversationSummary, []string) {
	operations := []string{}

	// 保守策略：优先移除低相关性长期记忆
	if len(memories) > 8 {
		// 按相似度排序，保留高相关性的记忆
		sortedMemories := make([]*model.ConversationMemory, len(memories))
		copy(sortedMemories, memories)

		// 简单排序：按重要性降序
		for i := 0; i < len(sortedMemories)-1; i++ {
			for j := i + 1; j < len(sortedMemories); j++ {
				if sortedMemories[i].Importance < sortedMemories[j].Importance {
					sortedMemories[i], sortedMemories[j] = sortedMemories[j], sortedMemories[i]
				}
			}
		}

		memories = sortedMemories[:8]
		operations = append(operations, "移除低相关性长期记忆，保留8条")
	}

	// 尽量保留短期消息
	if len(messages) > 15 {
		messages = messages[len(messages)-15:]
		operations = append(operations, "减少短期消息至最近15条")
	}

	return messages, memories, summary, operations
}
