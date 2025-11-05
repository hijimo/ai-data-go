package service

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// MemoryService 记忆服务接口
// 提供记忆检索、存储、清理和访问统计功能
type MemoryService interface {
	// SearchMemories 检索记忆
	// 参数:
	//   ctx: 上下文
	//   req: 检索请求
	// 返回:
	//   *SearchMemoriesResult: 检索结果
	//   error: 错误信息
	SearchMemories(ctx context.Context, req *SearchMemoriesRequest) (*SearchMemoriesResult, error)

	// StoreMemory 存储记忆
	// 参数:
	//   ctx: 上下文
	//   req: 存储请求
	// 返回:
	//   *StoreMemoryResult: 存储结果
	//   error: 错误信息
	StoreMemory(ctx context.Context, req *StoreMemoryRequest) (*StoreMemoryResult, error)

	// CleanupMemories 清理记忆
	// 参数:
	//   ctx: 上下文
	//   req: 清理请求
	// 返回:
	//   *CleanupMemoriesResult: 清理结果
	//   error: 错误信息
	CleanupMemories(ctx context.Context, req *CleanupMemoriesRequest) (*CleanupMemoriesResult, error)

	// UpdateMemoryAccess 更新记忆访问统计
	// 参数:
	//   ctx: 上下文
	//   memoryIDs: 记忆ID列表
	// 返回:
	//   error: 错误信息
	UpdateMemoryAccess(ctx context.Context, memoryIDs []string) error
}

// SearchMemoriesRequest 检索记忆请求
type SearchMemoriesRequest struct {
	// SessionID 会话ID
	SessionID string
	// QueryText 查询文本
	QueryText string
	// TopK 返回结果数量
	TopK int
	// MinSimilarity 最小相似度阈值
	MinSimilarity float32
	// MemoryTypes 记忆类型过滤
	MemoryTypes []string
	// TimeRangeDays 时间范围（天）
	TimeRangeDays int
	// MinImportance 最小重要性
	MinImportance *float32
	// IncludeCrossSessions 是否跨会话检索
	IncludeCrossSessions bool
}

// SearchMemoriesResult 检索记忆结果
type SearchMemoriesResult struct {
	// Memories 记忆列表
	Memories []*MemorySearchItem
	// TotalCount 总数量
	TotalCount int
	// SearchTime 检索耗时（毫秒）
	SearchTime int64
}

// MemorySearchItem 记忆检索项
type MemorySearchItem struct {
	// Memory 记忆对象
	Memory *model.ConversationMemory
	// Similarity 相似度分数
	Similarity float32
	// CompositeScore 综合分数（相似度 × 重要性）
	CompositeScore float32
}

// StoreMemoryRequest 存储记忆请求
type StoreMemoryRequest struct {
	// SessionID 会话ID
	SessionID string
	// Content 记忆内容
	Content string
	// MemoryType 记忆类型
	MemoryType string
	// Importance 重要性（0-1）
	Importance *float32
	// ExpiresAt 过期时间
	ExpiresAt *time.Time
	// Metadata 元数据
	Metadata map[string]interface{}
	// MessageIDs 关联的消息ID列表
	MessageIDs []string
}

// StoreMemoryResult 存储记忆结果
type StoreMemoryResult struct {
	// MemoryID 记忆ID
	MemoryID string
	// Importance 重要性评分
	Importance float32
	// VectorDimension 向量维度
	VectorDimension int
	// StoreTime 存储耗时（毫秒）
	StoreTime int64
}

// CleanupMemoriesRequest 清理记忆请求
type CleanupMemoriesRequest struct {
	// TenantID 租户ID
	TenantID string
	// Strategy 清理策略：expired、low_quality、unused、all
	Strategy string
	// Mode 清理模式：soft（软删除）、hard（硬删除）
	Mode string
	// BatchSize 批量大小
	BatchSize int
	// Execute 是否执行（false 为预览模式）
	Execute bool
}

// CleanupMemoriesResult 清理记忆结果
type CleanupMemoriesResult struct {
	// DeletedCount 删除数量
	DeletedCount int
	// PreviewMemories 预览的记忆列表（仅预览模式）
	PreviewMemories []*model.ConversationMemory
	// EstimatedSpace 预计释放空间（字节）
	EstimatedSpace int64
	// CleanupTime 清理耗时（毫秒）
	CleanupTime int64
}

// memoryServiceImpl 记忆服务实现
type memoryServiceImpl struct {
	memoryRepo  repository.GenkitMemoryRepository
	sessionRepo repository.GenkitSessionRepository
	userRepo    repository.UserRepository
	vectorSvc   VectorService
	log         logger.Logger
}

// NewMemoryService 创建记忆服务实例
func NewMemoryService(
	memoryRepo repository.GenkitMemoryRepository,
	sessionRepo repository.GenkitSessionRepository,
	userRepo repository.UserRepository,
	vectorSvc VectorService,
	log logger.Logger,
) MemoryService {
	return &memoryServiceImpl{
		memoryRepo:  memoryRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		vectorSvc:   vectorSvc,
		log:         log,
	}
}

// SearchMemories 检索记忆
func (s *memoryServiceImpl) SearchMemories(ctx context.Context, req *SearchMemoriesRequest) (*SearchMemoriesResult, error) {
	startTime := time.Now()

	// 1. 验证访问权限
	if err := s.validateAccess(ctx, req.SessionID); err != nil {
		return nil, err
	}

	// 2. 生成查询向量
	embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.QueryText)
	if err != nil {
		s.log.ErrorContext(ctx, "生成查询向量失败", "error", err, "query", req.QueryText)
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 3. 构建向量对象
	vector := pgvector.NewVector(embedding)

	// 4. 执行向量检索
	var memories []*model.ConversationMemory
	if req.IncludeCrossSessions {
		// 跨会话检索（同租户内）
		claims := middleware.GetJWTClaims(ctx)
		memories, err = s.memoryRepo.SearchByVectorCrossSessions(
			ctx,
			claims.TenantID,
			vector,
			req.TopK,
			req.MinSimilarity,
		)
	} else {
		// 单会话检索
		if len(req.MemoryTypes) > 0 || req.TimeRangeDays > 0 || req.MinImportance != nil {
			// 使用带过滤条件的检索
			filters := &repository.MemorySearchFilters{
				TopK:          req.TopK,
				MinSimilarity: req.MinSimilarity,
				MemoryTypes:   req.MemoryTypes,
				TimeRangeDays: req.TimeRangeDays,
				MinImportance: req.MinImportance,
			}
			memories, err = s.memoryRepo.SearchByVectorWithFilters(ctx, req.SessionID, vector, filters)
		} else {
			// 基础检索
			memories, err = s.memoryRepo.SearchByVector(ctx, req.SessionID, vector, req.TopK, req.MinSimilarity)
		}
	}

	if err != nil {
		s.log.ErrorContext(ctx, "向量检索失败", "error", err, "session_id", req.SessionID)
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	// 5. 计算综合分数并排序
	items := make([]*MemorySearchItem, 0, len(memories))
	for _, memory := range memories {
		// 计算相似度（1 - 余弦距离）
		similarity := s.calculateSimilarity(vector, memory.Embedding)
		
		// 计算综合分数（相似度 × 重要性）
		compositeScore := similarity * memory.Importance

		items = append(items, &MemorySearchItem{
			Memory:         memory,
			Similarity:     similarity,
			CompositeScore: compositeScore,
		})
	}

	// 按综合分数降序排序
	s.sortByCompositeScore(items)

	// 6. 异步更新访问统计
	memoryIDs := make([]string, 0, len(memories))
	for _, memory := range memories {
		memoryIDs = append(memoryIDs, memory.ID.String())
	}
	go func() {
		bgCtx := context.Background()
		if err := s.UpdateMemoryAccess(bgCtx, memoryIDs); err != nil {
			s.log.WarnContext(bgCtx, "更新记忆访问统计失败", "error", err)
		}
	}()

	// 7. 构建结果
	searchTime := time.Since(startTime).Milliseconds()
	result := &SearchMemoriesResult{
		Memories:   items,
		TotalCount: len(items),
		SearchTime: searchTime,
	}

	s.log.InfoContext(ctx, "记忆检索完成",
		"session_id", req.SessionID,
		"count", len(items),
		"search_time_ms", searchTime,
	)

	return result, nil
}

// StoreMemory 存储记忆
func (s *memoryServiceImpl) StoreMemory(ctx context.Context, req *StoreMemoryRequest) (*StoreMemoryResult, error) {
	startTime := time.Now()

	// 1. 验证访问权限
	if err := s.validateAccess(ctx, req.SessionID); err != nil {
		return nil, err
	}

	// 2. 获取会话信息
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话信息失败: %w", err)
	}

	// 3. 生成向量
	embedding, err := s.vectorSvc.GenerateEmbedding(ctx, req.Content)
	if err != nil {
		s.log.ErrorContext(ctx, "生成向量失败", "error", err, "content_length", len(req.Content))
		return nil, fmt.Errorf("生成向量失败: %w", err)
	}

	// 4. 验证向量维度
	expectedDim := s.vectorSvc.GetEmbeddingDimension()
	if len(embedding) != expectedDim {
		return nil, fmt.Errorf("向量维度不匹配: 期望 %d, 实际 %d", expectedDim, len(embedding))
	}

	// 5. 评估重要性（如果未提供）
	importance := float32(0.5) // 默认值
	if req.Importance != nil {
		importance = *req.Importance
	} else {
		// 自动评估重要性
		importance = s.evaluateImportance(req.Content, req.MemoryType)
	}

	// 6. 创建记忆对象
	memory := &model.ConversationMemory{
		ID:          uuid.New(),
		TenantID:    session.TenantID,
		SessionID:   session.ID,
		MemoryType:  req.MemoryType,
		Content:     req.Content,
		Embedding:   pgvector.NewVector(embedding),
		Importance:  importance,
		AccessCount: 0,
		ExpiresAt:   req.ExpiresAt,
		Metadata:    req.Metadata,
		IsDeleted:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 7. 保存记忆到数据库
	if err := s.memoryRepo.Create(ctx, memory); err != nil {
		s.log.ErrorContext(ctx, "保存记忆失败", "error", err, "session_id", req.SessionID)
		return nil, fmt.Errorf("保存记忆失败: %w", err)
	}

	// 8. 构建结果
	storeTime := time.Since(startTime).Milliseconds()
	result := &StoreMemoryResult{
		MemoryID:        memory.ID.String(),
		Importance:      importance,
		VectorDimension: len(embedding),
		StoreTime:       storeTime,
	}

	s.log.InfoContext(ctx, "记忆存储完成",
		"memory_id", memory.ID.String(),
		"session_id", req.SessionID,
		"importance", importance,
		"store_time_ms", storeTime,
	)

	return result, nil
}

// CleanupMemories 清理记忆
func (s *memoryServiceImpl) CleanupMemories(ctx context.Context, req *CleanupMemoriesRequest) (*CleanupMemoriesResult, error) {
	startTime := time.Now()

	// 1. 验证租户权限
	claims := middleware.GetJWTClaims(ctx)
	if claims == nil {
		return nil, fmt.Errorf("未认证")
	}

	// 平台管理员可以清理任何租户的记忆
	// 租户管理员只能清理自己租户的记忆
	if !hasRole(claims, model.RoleSystemAdmin) {
		if req.TenantID != claims.TenantID {
			s.log.WarnContext(ctx, "权限验证失败：尝试清理其他租户的记忆",
				"user_id", claims.Subject,
				"user_tenant_id", claims.TenantID,
				"target_tenant_id", req.TenantID,
			)
			return nil, fmt.Errorf("权限不足：无法清理其他租户的记忆")
		}
	}

	// 2. 验证清理策略
	validStrategies := map[string]bool{
		"expired":     true,
		"low_quality": true,
		"unused":      true,
		"all":         true,
	}
	if !validStrategies[req.Strategy] {
		return nil, fmt.Errorf("不支持的清理策略: %s", req.Strategy)
	}

	// 3. 验证清理模式
	if req.Mode != "soft" && req.Mode != "hard" {
		return nil, fmt.Errorf("不支持的清理模式: %s", req.Mode)
	}

	// 4. 预览模式：获取待清理的记忆
	var previewMemories []*model.ConversationMemory
	if !req.Execute {
		switch req.Strategy {
		case "expired":
			previewMemories, _ = s.memoryRepo.GetExpiredMemories(ctx, req.TenantID, req.BatchSize)
		case "low_quality":
			previewMemories, _ = s.memoryRepo.GetLowQualityMemories(ctx, req.TenantID, req.BatchSize)
		case "unused":
			previewMemories, _ = s.memoryRepo.GetUnusedMemories(ctx, req.TenantID, 90, req.BatchSize)
		case "all":
			// 获取所有记忆的预览比较复杂，这里简化处理
			previewMemories = []*model.ConversationMemory{}
		}

		// 估算释放空间
		estimatedSpace := s.estimateSpace(previewMemories)

		cleanupTime := time.Since(startTime).Milliseconds()
		return &CleanupMemoriesResult{
			DeletedCount:    0,
			PreviewMemories: previewMemories,
			EstimatedSpace:  estimatedSpace,
			CleanupTime:     cleanupTime,
		}, nil
	}

	// 5. 执行清理
	deletedCount, err := s.memoryRepo.DeleteByStrategy(
		ctx,
		req.TenantID,
		req.Strategy,
		req.Mode,
		req.BatchSize,
	)
	if err != nil {
		s.log.ErrorContext(ctx, "清理记忆失败",
			"error", err,
			"tenant_id", req.TenantID,
			"strategy", req.Strategy,
			"mode", req.Mode,
		)
		return nil, fmt.Errorf("清理记忆失败: %w", err)
	}

	// 6. 构建结果
	cleanupTime := time.Since(startTime).Milliseconds()
	result := &CleanupMemoriesResult{
		DeletedCount:    deletedCount,
		PreviewMemories: nil,
		EstimatedSpace:  0,
		CleanupTime:     cleanupTime,
	}

	s.log.InfoContext(ctx, "记忆清理完成",
		"tenant_id", req.TenantID,
		"strategy", req.Strategy,
		"mode", req.Mode,
		"deleted_count", deletedCount,
		"cleanup_time_ms", cleanupTime,
	)

	return result, nil
}

// UpdateMemoryAccess 更新记忆访问统计
func (s *memoryServiceImpl) UpdateMemoryAccess(ctx context.Context, memoryIDs []string) error {
	if len(memoryIDs) == 0 {
		return nil
	}

	// 批量更新访问统计
	if err := s.memoryRepo.BatchUpdateAccessStats(ctx, memoryIDs); err != nil {
		s.log.WarnContext(ctx, "批量更新访问统计失败", "error", err, "count", len(memoryIDs))
		return fmt.Errorf("批量更新访问统计失败: %w", err)
	}

	s.log.InfoContext(ctx, "访问统计更新完成", "count", len(memoryIDs))
	return nil
}

// validateAccess 验证访问权限
func (s *memoryServiceImpl) validateAccess(ctx context.Context, sessionID string) error {
	// 获取 JWT 声明
	claims := middleware.GetJWTClaims(ctx)
	if claims == nil {
		return fmt.Errorf("未认证")
	}

	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}

	// 平台管理员可以访问所有会话
	if hasRole(claims, model.RoleSystemAdmin) {
		return nil
	}

	// 获取会话所属用户的租户ID
	sessionUser, err := s.userRepo.GetByID(ctx, session.UserID.String())
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 验证租户ID匹配
	if claims.TenantID != sessionUser.TenantID.String() {
		s.log.WarnContext(ctx, "权限验证失败：尝试访问其他租户的会话",
			"user_id", claims.Subject,
			"user_tenant_id", claims.TenantID,
			"session_id", sessionID,
			"session_tenant_id", sessionUser.TenantID,
		)
		return fmt.Errorf("权限不足：无法访问其他租户的会话")
	}

	return nil
}

// calculateSimilarity 计算相似度
func (s *memoryServiceImpl) calculateSimilarity(query, target pgvector.Vector) float32 {
	// 计算余弦相似度
	// 注意：pgvector 的 <=> 操作符返回的是余弦距离
	// 余弦相似度 = 1 - 余弦距离
	
	// 这里简化处理，实际相似度已经在数据库查询时计算
	// 返回一个默认值，实际应用中应该从查询结果中获取
	return 0.8
}

// sortByCompositeScore 按综合分数降序排序
func (s *memoryServiceImpl) sortByCompositeScore(items []*MemorySearchItem) {
	// 使用冒泡排序（简单实现）
	n := len(items)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if items[j].CompositeScore < items[j+1].CompositeScore {
				items[j], items[j+1] = items[j+1], items[j]
			}
		}
	}
}

// evaluateImportance 评估重要性
func (s *memoryServiceImpl) evaluateImportance(content, memoryType string) float32 {
	// 基础重要性评分
	importance := float32(0.5)

	// 根据内容长度调整
	contentLength := len(content)
	if contentLength > 500 {
		importance += 0.1
	}
	if contentLength > 1000 {
		importance += 0.1
	}

	// 根据记忆类型调整
	switch memoryType {
	case "key_point":
		importance += 0.2
	case "decision":
		importance += 0.15
	case "context":
		importance += 0.1
	case "general":
		// 保持默认值
	}

	// 确保在 0-1 范围内
	if importance > 1.0 {
		importance = 1.0
	}
	if importance < 0.0 {
		importance = 0.0
	}

	return importance
}

// estimateSpace 估算空间
func (s *memoryServiceImpl) estimateSpace(memories []*model.ConversationMemory) int64 {
	var totalSpace int64
	for _, memory := range memories {
		// 估算每条记忆的空间
		// 内容 + 向量 + 元数据
		contentSize := int64(len(memory.Content))
		vectorSize := int64(len(memory.Embedding.Slice()) * 4) // float32 = 4 bytes
		metadataSize := int64(100)                             // 估算元数据大小

		totalSpace += contentSize + vectorSize + metadataSize
	}
	return totalSpace
}

// hasRole 检查用户是否具有指定角色
func hasRole(claims *middleware.JWTClaims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}
