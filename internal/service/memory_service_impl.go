// internal/service/memory_service_impl.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
	authservice "genkit-ai-service/internal/service/auth"
	"genkit-ai-service/internal/storage"
	"genkit-ai-service/pkg/errors"
)

// memoryServiceImpl 记忆管理服务实现
type memoryServiceImpl struct {
	memoryRepo   repository.MemoryRepository
	sessionRepo  repository.SessionRepository
	userRepo     repository.UserRepository
	vectorSvc    ai.VectorService
	qdrantClient storage.QdrantClient
	tokenMgr     TokenManager
}

// NewMemoryService 创建记忆管理服务
func NewMemoryService(
	memoryRepo repository.MemoryRepository,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
	vectorSvc ai.VectorService,
	qdrantClient storage.QdrantClient,
	tokenMgr TokenManager,
) MemoryService {
	return &memoryServiceImpl{
		memoryRepo:   memoryRepo,
		sessionRepo:  sessionRepo,
		userRepo:     userRepo,
		vectorSvc:    vectorSvc,
		qdrantClient: qdrantClient,
		tokenMgr:     tokenMgr,
	}
}

// SearchMemories 检索记忆
func (s *memoryServiceImpl) SearchMemories(
	ctx context.Context,
	req *SearchMemoriesRequest,
) ([]*MemorySearchResult, error) {
	// 1. 参数验证
	if req.SessionID == uuid.Nil {
		return nil, errors.New(errors.CodeBadRequest, "会话ID不能为空")
	}
	if req.Query == "" {
		return nil, errors.New(errors.CodeBadRequest, "查询文本不能为空")
	}

	// 设置默认值
	if req.TopK <= 0 {
		req.TopK = 5
	}
	if req.MinSimilarity <= 0 {
		req.MinSimilarity = 0.7
	}

	// 2. 权限验证
	tenantID, err := s.validateSessionAccess(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// 3. 生成查询向量
	queryVector, err := s.vectorSvc.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		logger.ErrorContext(ctx, "生成查询向量失败", logger.Fields{
			"session_id": req.SessionID.String(),
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "生成查询向量失败", err)
	}

	// 4. 构建向量检索请求
	searchReq := &storage.SearchVectorRequest{
		TenantID:    tenantID,
		QueryVector: queryVector,
		TopK:        req.TopK,
		MinScore:    req.MinSimilarity,
	}

	// 添加会话ID过滤（如果不是跨会话检索）
	if !req.IncludeCrossSessions {
		searchReq.SessionID = &req.SessionID
	}

	// 添加记忆类型过滤
	if len(req.MemoryTypes) > 0 {
		// Qdrant 只支持单个值匹配，这里取第一个类型
		// 如果需要多个类型，需要多次查询或修改 Qdrant 客户端
		memoryType := req.MemoryTypes[0]
		searchReq.MemoryType = &memoryType
	}

	// 添加时间范围过滤
	if req.TimeRangeDays > 0 {
		now := time.Now()
		start := now.AddDate(0, 0, -req.TimeRangeDays)
		searchReq.TimeRange = &storage.TimeRange{
			Start: start,
			End:   now,
		}
	}

	// 5. 执行向量检索
	vectorResults, err := s.qdrantClient.SearchVectors(ctx, searchReq)
	if err != nil {
		logger.ErrorContext(ctx, "向量检索失败", logger.Fields{
			"session_id": req.SessionID.String(),
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "向量检索失败", err)
	}

	// 6. 提取记忆ID列表
	memoryIDs := make([]uuid.UUID, 0, len(vectorResults))
	scoreMap := make(map[uuid.UUID]float32)
	for _, result := range vectorResults {
		memoryIDs = append(memoryIDs, result.MemoryID)
		scoreMap[result.MemoryID] = result.Score
	}

	// 7. 从数据库获取记忆元数据
	var memories []*model.ConversationMemory
	if req.IncludeCrossSessions {
		memories, err = s.memoryRepo.SearchByVectorCrossSessions(ctx, tenantID, memoryIDs)
	} else {
		memories, err = s.memoryRepo.SearchByVector(ctx, tenantID, req.SessionID, memoryIDs)
	}
	if err != nil {
		logger.ErrorContext(ctx, "获取记忆元数据失败", logger.Fields{
			"session_id": req.SessionID.String(),
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "获取记忆元数据失败", err)
	}

	// 8. 构建检索结果
	results := make([]*MemorySearchResult, 0, len(memories))
	for _, memory := range memories {
		similarity := scoreMap[memory.ID]
		score := similarity * memory.Importance // 综合评分 = 相似度 × 重要性

		results = append(results, &MemorySearchResult{
			Memory:     memory,
			Similarity: similarity,
			Score:      score,
		})
	}

	// 9. 异步更新访问统计
	go func() {
		bgCtx := context.Background()
		for _, memory := range memories {
			if err := s.memoryRepo.UpdateAccessStats(bgCtx, tenantID, memory.ID); err != nil {
				logger.WarnContext(bgCtx, "更新记忆访问统计失败", logger.Fields{
					"memory_id": memory.ID.String(),
					"error":     err.Error(),
				})
			}
		}
	}()

	logger.InfoContext(ctx, "记忆检索完成", logger.Fields{
		"session_id":     req.SessionID.String(),
		"query":          req.Query,
		"results_count":  len(results),
		"cross_sessions": req.IncludeCrossSessions,
	})

	return results, nil
}

// StoreMemory 存储记忆
func (s *memoryServiceImpl) StoreMemory(
	ctx context.Context,
	req *StoreMemoryRequest,
) (*model.ConversationMemory, error) {
	// 1. 参数验证
	if req.SessionID == uuid.Nil {
		return nil, errors.New(errors.CodeBadRequest, "会话ID不能为空")
	}
	if req.Content == "" {
		return nil, errors.New(errors.CodeBadRequest, "记忆内容不能为空")
	}
	if req.MemoryType == "" {
		return nil, errors.New(errors.CodeBadRequest, "记忆类型不能为空")
	}

	// 设置默认值
	if req.Importance <= 0 {
		req.Importance = 0.5
	}
	if req.Importance > 1.0 {
		req.Importance = 1.0
	}

	// 2. 权限验证
	tenantID, err := s.validateSessionAccess(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// 3. 生成向量嵌入
	embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		logger.ErrorContext(ctx, "生成向量嵌入失败", logger.Fields{
			"session_id": req.SessionID.String(),
			"error":      err.Error(),
		})
		// 重试一次
		embedding, err = s.vectorSvc.GenerateEmbedding(ctx, req.Content)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "生成向量嵌入失败", err)
		}
	}

	// 4. 计算 Token 数量
	tokenCount, err := s.tokenMgr.CalculateTokens(ctx, req.Content, "")
	if err != nil {
		logger.WarnContext(ctx, "计算Token数量失败，使用估算值", logger.Fields{
			"error": err.Error(),
		})
		tokenCount = s.tokenMgr.EstimateTokens(req.Content)
	}

	// 5. 计算过期时间
	var expiresAt *time.Time
	if req.ExpirationDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.ExpirationDays)
		expiresAt = &expiry
	}

	// 6. 创建记忆记录
	now := time.Now()
	memory := &model.ConversationMemory{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SessionID:   req.SessionID,
		MemoryType:  req.MemoryType,
		Content:     req.Content,
		TokenCount:  tokenCount,
		Importance:  req.Importance,
		AccessCount: 0,
		ExpiresAt:   expiresAt,
		IsDeleted:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 处理元数据 - 将 map[string]interface{} 转换为 datatypes.JSON
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			logger.WarnContext(ctx, "序列化元数据失败", logger.Fields{
				"error": err.Error(),
			})
		} else {
			memory.Metadata = metadataJSON
		}
	}

	// 7. 保存记忆元数据到数据库
	if err := s.memoryRepo.Create(ctx, memory); err != nil {
		logger.ErrorContext(ctx, "保存记忆元数据失败", logger.Fields{
			"session_id": req.SessionID.String(),
			"error":      err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "保存记忆元数据失败", err)
	}

	// 8. 保存向量到 Qdrant
	upsertReq := &storage.UpsertVectorRequest{
		TenantID:   tenantID,
		MemoryID:   memory.ID,
		SessionID:  req.SessionID,
		MemoryType: req.MemoryType,
		Vector:     embedding,
		Importance: req.Importance,
		ExpiresAt:  expiresAt,
		Metadata:   req.Metadata,
	}

	if err := s.qdrantClient.UpsertVector(ctx, upsertReq); err != nil {
		logger.ErrorContext(ctx, "保存向量失败", logger.Fields{
			"memory_id": memory.ID.String(),
			"error":     err.Error(),
		})
		// 向量保存失败，删除数据库记录
		_ = s.memoryRepo.HardDelete(ctx, tenantID, memory.ID)
		return nil, errors.Wrap(errors.CodeInternalError, "保存向量失败", err)
	}

	logger.InfoContext(ctx, "记忆存储成功", logger.Fields{
		"memory_id":   memory.ID.String(),
		"session_id":  req.SessionID.String(),
		"memory_type": req.MemoryType,
		"token_count": tokenCount,
		"importance":  req.Importance,
	})

	return memory, nil
}

// CleanupMemories 清理记忆
func (s *memoryServiceImpl) CleanupMemories(
	ctx context.Context,
	req *CleanupMemoriesRequest,
) (*CleanupResult, error) {
	// 1. 参数验证
	if req.Strategy == "" {
		return nil, errors.New(errors.CodeBadRequest, "清理策略不能为空")
	}
	if req.Mode == "" {
		req.Mode = "soft" // 默认软删除
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 100
	}

	// 2. 权限验证
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, errors.New(errors.CodeUnauthorized, "未认证")
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, errors.New(errors.CodeBadRequest, "无效的租户ID")
	}

	// 3. 如果指定了会话ID，验证会话访问权限
	if req.SessionID != uuid.Nil {
		_, err := s.validateSessionAccess(ctx, req.SessionID)
		if err != nil {
			return nil, err
		}
	}

	// 4. 验证清理策略
	var deleteStrategy repository.DeleteStrategy
	switch req.Strategy {
	case "expired":
		deleteStrategy = repository.DeleteStrategyExpired
	case "low_quality":
		deleteStrategy = repository.DeleteStrategyLowQuality
	case "unused":
		deleteStrategy = repository.DeleteStrategyUnused
	case "all":
		deleteStrategy = repository.DeleteStrategyAll
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("无效的清理策略: %s", req.Strategy))
	}

	// 5. 验证清理模式
	var deleteMode repository.DeleteMode
	switch req.Mode {
	case "soft":
		deleteMode = repository.DeleteModeSoft
	case "hard":
		deleteMode = repository.DeleteModeHard
	default:
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("无效的清理模式: %s", req.Mode))
	}

	// 6. 如果是预览模式，获取待清理的记忆列表
	if !req.Execute {
		// TODO: 实现预览功能
		// 这里需要在 repository 中添加一个查询方法来获取符合条件的记忆列表
		return &CleanupResult{
			CleanedCount: 0,
			FreedSpace:   0,
			Details:      []CleanupDetail{},
			Preview:      true,
		}, nil
	}

	// 7. 执行清理
	cleanedCount, err := s.memoryRepo.DeleteByStrategy(ctx, tenantID, deleteStrategy, deleteMode)
	if err != nil {
		logger.ErrorContext(ctx, "清理记忆失败", logger.Fields{
			"tenant_id": tenantID.String(),
			"strategy":  req.Strategy,
			"mode":      req.Mode,
			"error":     err.Error(),
		})
		return nil, errors.Wrap(errors.CodeInternalError, "清理记忆失败", err)
	}

	// 8. 如果是硬删除，同时删除 Qdrant 中的向量
	if deleteMode == repository.DeleteModeHard {
		// 构建过滤条件
		filter := make(map[string]interface{})
		if req.SessionID != uuid.Nil {
			filter["session_id"] = req.SessionID.String()
		}

		// 根据策略添加过滤条件
		switch deleteStrategy {
		case repository.DeleteStrategyExpired:
			// Qdrant 会自动处理过期的向量
		case repository.DeleteStrategyLowQuality:
			// 需要在 Qdrant 中添加 importance 过滤
			// 这里简化处理，实际应该在 Qdrant 客户端中实现
		case repository.DeleteStrategyUnused:
			// 需要在 Qdrant 中添加 last_access_at 过滤
			// 这里简化处理，实际应该在 Qdrant 客户端中实现
		}

		if err := s.qdrantClient.DeleteByFilter(ctx, tenantID, filter); err != nil {
			logger.WarnContext(ctx, "删除 Qdrant 向量失败", logger.Fields{
				"tenant_id": tenantID.String(),
				"error":     err.Error(),
			})
			// 不中断流程，仅记录警告
		}
	}

	logger.InfoContext(ctx, "记忆清理完成", logger.Fields{
		"tenant_id":     tenantID.String(),
		"strategy":      req.Strategy,
		"mode":          req.Mode,
		"cleaned_count": cleanedCount,
	})

	return &CleanupResult{
		CleanedCount: int(cleanedCount),
		FreedSpace:   0, // TODO: 计算释放的空间
		Details:      []CleanupDetail{},
		Preview:      false,
	}, nil
}

// UpdateMemoryAccess 更新记忆访问统计
func (s *memoryServiceImpl) UpdateMemoryAccess(
	ctx context.Context,
	tenantID, memoryID uuid.UUID,
) error {
	// 1. 参数验证
	if tenantID == uuid.Nil {
		return errors.New(errors.CodeBadRequest, "租户ID不能为空")
	}
	if memoryID == uuid.Nil {
		return errors.New(errors.CodeBadRequest, "记忆ID不能为空")
	}

	// 2. 权限验证
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		return errors.New(errors.CodeUnauthorized, "未认证")
	}

	// 平台管理员可以访问所有租户的记忆
	if !hasRoleMemory(claims, model.RoleSystemAdmin) {
		// 租户管理员只能访问自己租户的记忆
		if claims.TenantID != tenantID.String() {
			logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的记忆", logger.Fields{
				"user_id":          claims.Subject,
				"user_tenant_id":   claims.TenantID,
				"target_tenant_id": tenantID.String(),
			})
			return errors.New(errors.CodeForbidden, "权限不足：无法访问其他租户的记忆")
		}
	}

	// 3. 更新访问统计
	if err := s.memoryRepo.UpdateAccessStats(ctx, tenantID, memoryID); err != nil {
		logger.ErrorContext(ctx, "更新记忆访问统计失败", logger.Fields{
			"memory_id": memoryID.String(),
			"error":     err.Error(),
		})
		return errors.Wrap(errors.CodeInternalError, "更新记忆访问统计失败", err)
	}

	return nil
}

// ========== 私有辅助方法 ==========

// validateSessionAccess 验证会话访问权限
func (s *memoryServiceImpl) validateSessionAccess(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	// 获取 JWT 声明
	claims, ok := authservice.GetJWTClaimsFromContext(ctx)
	if !ok || claims == nil {
		return uuid.Nil, errors.New(errors.CodeUnauthorized, "未认证")
	}

	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID.String())
	if err != nil {
		return uuid.Nil, errors.New(errors.CodeNotFound, "会话不存在")
	}

	// 获取会话所属用户的租户ID
	userUUID := session.UserID
	userIDStr := userUUID.String()
	
	// 使用 GetByIDOnly 获取用户信息
	sessionUser, err := s.userRepo.GetByIDOnly(ctx, userIDStr)
	if err != nil {
		logger.ErrorContext(ctx, "获取会话用户信息失败", logger.Fields{
			"session_id": sessionID.String(),
			"user_id":    userIDStr,
			"error":      err.Error(),
		})
		return uuid.Nil, errors.Wrap(errors.CodeInternalError, "获取用户信息失败", err)
	}

	// 平台管理员可以访问所有会话
	if hasRoleMemory(claims, model.RoleSystemAdmin) {
		return sessionUser.TenantID, nil
	}

	// 验证租户ID匹配
	if claims.TenantID != sessionUser.TenantID.String() {
		logger.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话", logger.Fields{
			"user_id":           claims.Subject,
			"user_tenant_id":    claims.TenantID,
			"session_id":        sessionID.String(),
			"session_tenant_id": sessionUser.TenantID.String(),
		})
		return uuid.Nil, errors.New(errors.CodeForbidden, "权限不足：无法访问其他租户的会话")
	}

	return sessionUser.TenantID, nil
}

// hasRoleMemory 检查用户是否具有指定角色
func hasRoleMemory(claims *model.JWTClaims, role string) bool {
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
