package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
)

// MemorySearchInput 记忆搜索输入
type MemorySearchInput struct {
	SessionID            string   `json:"sessionId"`            // 会话ID
	Query                string   `json:"query"`                // 查询文本
	TopK                 int      `json:"topK"`                 // 返回结果数量
	MinSimilarity        float32  `json:"minSimilarity"`        // 最小相似度阈值
	TimeRangeDays        int      `json:"timeRangeDays"`        // 时间范围（天）
	MemoryTypes          []string `json:"memoryTypes"`          // 记忆类型过滤
	IncludeCrossSessions bool     `json:"includeCrossSessions"` // 是否包含跨会话检索
}

// MemorySearchOutput 记忆搜索输出
type MemorySearchOutput struct {
	Memories          []MemoryResult `json:"memories"`          // 记忆结果列表
	TotalFound        int            `json:"totalFound"`        // 找到的总数
	ReturnedCount     int            `json:"returnedCount"`     // 返回的数量
	SearchTime        int64          `json:"searchTime"`        // 搜索耗时（毫秒）
	AverageSimilarity float32        `json:"averageSimilarity"` // 平均相似度
	SearchStrategy    string         `json:"searchStrategy"`    // 搜索策略
}

// MemoryResult 记忆结果
type MemoryResult struct {
	ID           string                 `json:"id"`           // 记忆ID
	SessionID    string                 `json:"sessionId"`    // 会话ID
	MemoryType   string                 `json:"memoryType"`   // 记忆类型
	Content      string                 `json:"content"`      // 内容
	TokenCount   int                    `json:"tokenCount"`   // Token数量
	Similarity   float32                `json:"similarity"`   // 相似度
	Importance   float32                `json:"importance"`   // 重要性
	Score        float32                `json:"score"`        // 综合得分
	AccessCount  int                    `json:"accessCount"`  // 访问次数
	CreatedAt    string                 `json:"createdAt"`    // 创建时间
	LastAccessAt string                 `json:"lastAccessAt"` // 最后访问时间
	Metadata     map[string]interface{} `json:"metadata"`     // 元数据
}

// RegisterMemoryFlows 注册记忆相关的Flow
func RegisterMemoryFlows(
	g *genkit.Genkit,
	memoryRepo repository.GenkitMemoryRepository,
	messageRepo repository.MessageRepository,
	vectorSvc service.VectorService,
	tokenMgr service.TokenManager,
	log logger.Logger,
) {
	// 注册 memorySearchFlow
	genkit.DefineFlow(
		g,
		"memorySearchFlow",
		func(ctx context.Context, input MemorySearchInput) (MemorySearchOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateMemorySearchInput(input); err != nil {
				log.ErrorContext(ctx, "参数验证失败", logger.Fields{
					"error": err.Error(),
					"input": input,
				})
				return MemorySearchOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 权限验证
			claims, ok := middleware.GetJWTClaims(ctx)
			if !ok || claims == nil {
				log.WarnContext(ctx, "未认证的请求", logger.Fields{
					"session_id": input.SessionID,
				})
				return MemorySearchOutput{}, fmt.Errorf("未认证")
			}

			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				log.ErrorContext(ctx, "无效的租户ID", logger.Fields{
					"tenant_id": claims.TenantID,
					"error":     err.Error(),
				})
				return MemorySearchOutput{}, fmt.Errorf("无效的租户ID")
			}

			// 3. 生成查询向量
			log.InfoContext(ctx, "开始生成查询向量", logger.Fields{
				"query":      input.Query,
				"session_id": input.SessionID,
			})

			embedding, err := vectorSvc.GenerateEmbedding(ctx, input.Query)
			if err != nil {
				log.ErrorContext(ctx, "生成查询向量失败", logger.Fields{
					"error": err.Error(),
					"query": input.Query,
				})
				return MemorySearchOutput{}, fmt.Errorf("生成查询向量失败: %w", err)
			}

			// 转换为 pgvector.Vector
			embeddingVector := pgvector.NewVector(embedding)

			// 4. 执行向量检索
			var memories []*model.ConversationMemory
			searchStrategy := "single_session"

			if input.IncludeCrossSessions {
				// 跨会话检索（同租户内）
				searchStrategy = "cross_sessions"
				log.InfoContext(ctx, "执行跨会话向量检索", logger.Fields{
					"tenant_id":      tenantID.String(),
					"top_k":          input.TopK,
					"min_similarity": input.MinSimilarity,
				})

				memories, err = memoryRepo.SearchByVectorCrossSessions(
					ctx,
					tenantID.String(),
					embeddingVector,
					input.TopK,
					input.MinSimilarity,
				)
			} else {
				// 单会话检索
				log.InfoContext(ctx, "执行单会话向量检索", logger.Fields{
					"session_id":     input.SessionID,
					"top_k":          input.TopK,
					"min_similarity": input.MinSimilarity,
				})

				// 如果有额外的过滤条件，使用带过滤的搜索
				if len(input.MemoryTypes) > 0 || input.TimeRangeDays > 0 {
					filters := &repository.MemorySearchFilters{
						TopK:          input.TopK,
						MinSimilarity: input.MinSimilarity,
						MemoryTypes:   input.MemoryTypes,
						TimeRangeDays: input.TimeRangeDays,
					}

					memories, err = memoryRepo.SearchByVectorWithFilters(
						ctx,
						input.SessionID,
						embeddingVector,
						filters,
					)
				} else {
					memories, err = memoryRepo.SearchByVector(
						ctx,
						input.SessionID,
						embeddingVector,
						input.TopK,
						input.MinSimilarity,
					)
				}
			}

			if err != nil {
				log.ErrorContext(ctx, "向量检索失败", logger.Fields{
					"error":    err.Error(),
					"strategy": searchStrategy,
				})
				return MemorySearchOutput{}, fmt.Errorf("向量检索失败: %w", err)
			}

			// 5. 计算相似度和综合得分
			results := make([]MemoryResult, 0, len(memories))
			var totalSimilarity float32

			for _, memory := range memories {
				// 计算余弦相似度
				similarity := calculateCosineSimilarity(embedding, memory.Embedding.Slice())

				// 计算综合得分（相似度 × 重要性）
				score := similarity * memory.Importance

				// 格式化时间
				lastAccessAt := ""
				if memory.LastAccessAt != nil {
					lastAccessAt = memory.LastAccessAt.Format(time.RFC3339)
				}

				result := MemoryResult{
					ID:           memory.ID.String(),
					SessionID:    memory.SessionID.String(),
					MemoryType:   memory.MemoryType,
					Content:      memory.Content,
					TokenCount:   memory.TokenCount,
					Similarity:   similarity,
					Importance:   memory.Importance,
					Score:        score,
					AccessCount:  memory.AccessCount,
					CreatedAt:    memory.CreatedAt.Format(time.RFC3339),
					LastAccessAt: lastAccessAt,
					Metadata:     memory.Metadata,
				}

				results = append(results, result)
				totalSimilarity += similarity
			}

			// 6. 按综合得分排序（已经在数据库层排序，这里只是确保）
			// 数据库查询已经按照相似度排序，综合得分也会保持类似的顺序

			// 7. 异步更新访问统计
			if len(memories) > 0 {
				memoryIDs := make([]string, len(memories))
				for i, memory := range memories {
					memoryIDs[i] = memory.ID.String()
				}

				// 在后台异步更新
				go func() {
					bgCtx := context.Background()
					if err := memoryRepo.BatchUpdateAccessStats(bgCtx, memoryIDs); err != nil {
						log.WarnContext(bgCtx, "异步更新访问统计失败", logger.Fields{
							"error":        err.Error(),
							"memory_count": len(memoryIDs),
						})
					} else {
						log.InfoContext(bgCtx, "成功更新访问统计", logger.Fields{
							"memory_count": len(memoryIDs),
						})
					}
				}()
			}

			// 8. 计算平均相似度
			avgSimilarity := float32(0)
			if len(results) > 0 {
				avgSimilarity = totalSimilarity / float32(len(results))
			}

			// 9. 构建输出
			searchTime := time.Since(startTime).Milliseconds()

			output := MemorySearchOutput{
				Memories:          results,
				TotalFound:        len(results),
				ReturnedCount:     len(results),
				SearchTime:        searchTime,
				AverageSimilarity: avgSimilarity,
				SearchStrategy:    searchStrategy,
			}

			log.InfoContext(ctx, "记忆搜索完成", logger.Fields{
				"total_found":    output.TotalFound,
				"search_time_ms": searchTime,
				"avg_similarity": avgSimilarity,
				"strategy":       searchStrategy,
			})

			return output, nil
		},
	)

	// 注册 memoryStoreFlow
	genkit.DefineFlow(
		g,
		"memoryStoreFlow",
		func(ctx context.Context, input MemoryStoreInput) (MemoryStoreOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateMemoryStoreInput(input); err != nil {
				log.ErrorContext(ctx, "参数验证失败", logger.Fields{
					"error": err.Error(),
					"input": input,
				})
				return MemoryStoreOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 权限验证
			claims, ok := middleware.GetJWTClaims(ctx)
			if !ok || claims == nil {
				log.WarnContext(ctx, "未认证的请求", logger.Fields{
					"session_id": input.SessionID,
				})
				return MemoryStoreOutput{}, fmt.Errorf("未认证")
			}

			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				log.ErrorContext(ctx, "无效的租户ID", logger.Fields{
					"tenant_id": claims.TenantID,
					"error":     err.Error(),
				})
				return MemoryStoreOutput{}, fmt.Errorf("无效的租户ID")
			}

			sessionID, err := uuid.Parse(input.SessionID)
			if err != nil {
				log.ErrorContext(ctx, "无效的会话ID", logger.Fields{
					"session_id": input.SessionID,
					"error":      err.Error(),
				})
				return MemoryStoreOutput{}, fmt.Errorf("无效的会话ID")
			}

			// 3. 准备内容
			var content string
			var tokenCount int

			if input.Content != "" {
				// 使用提供的内容
				content = input.Content
				log.InfoContext(ctx, "使用提供的内容", logger.Fields{
					"content_length": len(content),
				})
			} else if len(input.MessageIDs) > 0 {
				// 从消息中提取内容
				log.InfoContext(ctx, "从消息中提取内容", logger.Fields{
					"message_count": len(input.MessageIDs),
				})

				var messages []*model.ChatMessage
				for _, msgID := range input.MessageIDs {
					msg, err := messageRepo.GetByID(ctx, msgID)
					if err != nil {
						log.WarnContext(ctx, "获取消息失败", logger.Fields{
							"message_id": msgID,
							"error":      err.Error(),
						})
						continue
					}
					messages = append(messages, msg)
				}

				if len(messages) == 0 {
					return MemoryStoreOutput{}, fmt.Errorf("未找到有效的消息")
				}

				// 组合消息内容
				for i, msg := range messages {
					if i > 0 {
						content += "\n"
					}
					content += fmt.Sprintf("[%s]: %s", msg.Role, msg.Content)
				}

				log.InfoContext(ctx, "成功提取消息内容", logger.Fields{
					"message_count":  len(messages),
					"content_length": len(content),
				})
			} else {
				return MemoryStoreOutput{}, fmt.Errorf("必须提供内容或消息ID")
			}

			// 4. 计算Token数量
			if tokenMgr != nil {
				tokenCount = tokenMgr.CountTokens(content)
			} else {
				// 简单估算：1个token约等于4个字符
				tokenCount = len(content) / 4
			}

			log.InfoContext(ctx, "计算Token数量", logger.Fields{
				"token_count": tokenCount,
			})

			// 5. 生成向量
			log.InfoContext(ctx, "开始生成向量", logger.Fields{
				"content_length": len(content),
			})

			embedding, err := vectorSvc.GenerateEmbedding(ctx, content)
			if err != nil {
				log.ErrorContext(ctx, "生成向量失败", logger.Fields{
					"error": err.Error(),
				})
				return MemoryStoreOutput{}, fmt.Errorf("生成向量失败: %w", err)
			}

			// 验证向量维度
			expectedDim := vectorSvc.GetEmbeddingDimension()
			if len(embedding) != expectedDim {
				log.ErrorContext(ctx, "向量维度不匹配", logger.Fields{
					"expected": expectedDim,
					"actual":   len(embedding),
				})
				return MemoryStoreOutput{}, fmt.Errorf("向量维度不匹配: 期望 %d, 实际 %d", expectedDim, len(embedding))
			}

			log.InfoContext(ctx, "成功生成向量", logger.Fields{
				"dimension": len(embedding),
			})

			// 6. 提取关键词和实体
			keywords, entities := extractKeywordsAndEntities(content)

			log.InfoContext(ctx, "提取关键词和实体", logger.Fields{
				"keyword_count": len(keywords),
				"entity_count":  len(entities),
			})

			// 7. 评估重要性
			importance := input.Importance
			if importance == nil {
				// 自动评估重要性
				calculatedImportance := evaluateImportance(content, keywords, entities, tokenCount)
				importance = &calculatedImportance

				log.InfoContext(ctx, "自动评估重要性", logger.Fields{
					"importance": *importance,
				})
			}

			// 8. 计算过期时间
			var expiresAt *time.Time
			if input.ExpirationDays > 0 {
				expiration := time.Now().AddDate(0, 0, input.ExpirationDays)
				expiresAt = &expiration

				log.InfoContext(ctx, "设置过期时间", logger.Fields{
					"expires_at": expiration.Format(time.RFC3339),
					"days":       input.ExpirationDays,
				})
			}

			// 9. 准备元数据
			metadata := input.Metadata
			if metadata == nil {
				metadata = make(map[string]interface{})
			}

			// 添加提取的信息到元数据
			if len(keywords) > 0 {
				metadata["keywords"] = keywords
			}
			if len(entities) > 0 {
				metadata["entities"] = entities
			}
			if len(input.MessageIDs) > 0 {
				metadata["source_message_ids"] = input.MessageIDs
			}
			metadata["token_count"] = tokenCount

			// 10. 创建记忆对象
			memory := &model.ConversationMemory{
				TenantID:     tenantID,
				SessionID:    sessionID,
				MemoryType:   input.MemoryType,
				Content:      content,
				Embedding:    pgvector.NewVector(embedding),
				TokenCount:   tokenCount,
				Importance:   *importance,
				AccessCount:  0,
				LastAccessAt: nil,
				Metadata:     metadata,
				ExpiresAt:    expiresAt,
				IsDeleted:    false,
			}

			// 11. 保存到数据库
			log.InfoContext(ctx, "保存记忆到数据库", logger.Fields{
				"session_id":  input.SessionID,
				"memory_type": input.MemoryType,
				"importance":  *importance,
			})

			if err := memoryRepo.Create(ctx, memory); err != nil {
				log.ErrorContext(ctx, "保存记忆失败", logger.Fields{
					"error": err.Error(),
				})
				return MemoryStoreOutput{}, fmt.Errorf("保存记忆失败: %w", err)
			}

			// 12. 构建输出
			storageTime := time.Since(startTime).Milliseconds()

			expiresAtStr := ""
			if expiresAt != nil {
				expiresAtStr = expiresAt.Format(time.RFC3339)
			}

			output := MemoryStoreOutput{
				MemoryID:        memory.ID.String(),
				SessionID:       input.SessionID,
				MemoryType:      input.MemoryType,
				Content:         content,
				TokenCount:      tokenCount,
				Importance:      *importance,
				KeyEntities:     entities,
				Keywords:        keywords,
				ExpiresAt:       expiresAtStr,
				Metadata:        metadata,
				VectorGenerated: true,
				StorageTime:     storageTime,
			}

			log.InfoContext(ctx, "记忆存储完成", logger.Fields{
				"memory_id":     output.MemoryID,
				"token_count":   tokenCount,
				"importance":    *importance,
				"storage_time_ms": storageTime,
			})

			return output, nil
		},
	)

	// 注册 memoryCleanupFlow
	genkit.DefineFlow(
		g,
		"memoryCleanupFlow",
		func(ctx context.Context, input MemoryCleanupInput) (MemoryCleanupOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateMemoryCleanupInput(input); err != nil {
				log.ErrorContext(ctx, "参数验证失败", logger.Fields{
					"error": err.Error(),
					"input": input,
				})
				return MemoryCleanupOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 权限验证
			claims, ok := middleware.GetJWTClaims(ctx)
			if !ok || claims == nil {
				log.WarnContext(ctx, "未认证的请求", logger.Fields{
					"session_id": input.SessionID,
				})
				return MemoryCleanupOutput{}, fmt.Errorf("未认证")
			}

			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				log.ErrorContext(ctx, "无效的租户ID", logger.Fields{
					"tenant_id": claims.TenantID,
					"error":     err.Error(),
				})
				return MemoryCleanupOutput{}, fmt.Errorf("无效的租户ID")
			}

			// 3. 构建清理过滤条件
			filters := &repository.MemoryCleanupFilters{
				TenantID:  tenantID.String(),
				SessionID: input.SessionID,
				Strategy:  input.Strategy,
				BatchSize: input.BatchSize,
			}

			log.InfoContext(ctx, "开始记忆清理", logger.Fields{
				"tenant_id":  tenantID.String(),
				"session_id": input.SessionID,
				"strategy":   input.Strategy,
				"mode":       input.Mode,
				"execute":    input.Execute,
				"batch_size": input.BatchSize,
			})

			// 4. 查询待清理的记忆
			var memoriesToClean []*model.ConversationMemory

			switch input.Strategy {
			case "expired":
				// 清理已过期的记忆
				memoriesToClean, err = memoryRepo.GetExpiredMemories(ctx, filters)
			case "low_quality":
				// 清理低质量记忆（重要性低于0.3且访问次数少于2）
				memoriesToClean, err = memoryRepo.GetLowQualityMemories(ctx, filters)
			case "unused":
				// 清理90天未访问的记忆
				memoriesToClean, err = memoryRepo.GetUnusedMemories(ctx, filters)
			case "all":
				// 清理所有记忆（谨慎使用）
				memoriesToClean, err = memoryRepo.GetAllMemoriesForCleanup(ctx, filters)
			default:
				return MemoryCleanupOutput{}, fmt.Errorf("不支持的清理策略: %s", input.Strategy)
			}

			if err != nil {
				log.ErrorContext(ctx, "查询待清理记忆失败", logger.Fields{
					"error":    err.Error(),
					"strategy": input.Strategy,
				})
				return MemoryCleanupOutput{}, fmt.Errorf("查询待清理记忆失败: %w", err)
			}

			log.InfoContext(ctx, "查询到待清理记忆", logger.Fields{
				"count":    len(memoriesToClean),
				"strategy": input.Strategy,
			})

			// 5. 构建清理详情
			details := make([]CleanupDetail, 0, len(memoriesToClean))
			var totalFreedSpace int64
			var totalFreedTokens int

			for _, memory := range memoriesToClean {
				// 估算大小（内容长度 + 向量大小）
				contentSize := int64(len(memory.Content))
				vectorSize := int64(len(memory.Embedding.Slice()) * 4) // float32 = 4字节
				totalSize := contentSize + vectorSize

				// 确定清理原因
				reason := getCleanupReason(memory, input.Strategy)

				// 格式化最后访问时间
				lastAccess := ""
				if memory.LastAccessAt != nil {
					lastAccess = memory.LastAccessAt.Format(time.RFC3339)
				}

				detail := CleanupDetail{
					MemoryID:   memory.ID.String(),
					SessionID:  memory.SessionID.String(),
					MemoryType: memory.MemoryType,
					Reason:     reason,
					Size:       totalSize,
					TokenCount: memory.TokenCount,
					Importance: memory.Importance,
					CreatedAt:  memory.CreatedAt.Format(time.RFC3339),
					LastAccess: lastAccess,
				}

				details = append(details, detail)
				totalFreedSpace += totalSize
				totalFreedTokens += memory.TokenCount
			}

			// 6. 执行清理（如果不是预览模式）
			cleanedCount := 0
			if input.Execute {
				log.InfoContext(ctx, "开始执行清理", logger.Fields{
					"count": len(memoriesToClean),
					"mode":  input.Mode,
				})

				// 提取记忆ID列表
				memoryIDs := make([]string, len(memoriesToClean))
				for i, memory := range memoriesToClean {
					memoryIDs[i] = memory.ID.String()
				}

				// 根据模式执行删除
				if input.Mode == "soft" {
					// 软删除：标记为已删除
					cleanedCount, err = memoryRepo.SoftDeleteBatch(ctx, memoryIDs)
				} else {
					// 硬删除：物理删除
					cleanedCount, err = memoryRepo.HardDeleteBatch(ctx, memoryIDs)
				}

				if err != nil {
					log.ErrorContext(ctx, "执行清理失败", logger.Fields{
						"error": err.Error(),
						"mode":  input.Mode,
						"count": len(memoryIDs),
					})
					return MemoryCleanupOutput{}, fmt.Errorf("执行清理失败: %w", err)
				}

				log.InfoContext(ctx, "清理执行完成", logger.Fields{
					"cleaned_count": cleanedCount,
					"mode":          input.Mode,
				})
			} else {
				log.InfoContext(ctx, "预览模式，不执行实际删除", logger.Fields{
					"preview_count": len(memoriesToClean),
				})
			}

			// 7. 构建输出
			cleanupTime := time.Since(startTime).Milliseconds()

			output := MemoryCleanupOutput{
				SessionID:      input.SessionID,
				Strategy:       input.Strategy,
				Mode:           input.Mode,
				CleanedCount:   cleanedCount,
				FreedSpace:     totalFreedSpace,
				FreedTokens:    totalFreedTokens,
				Details:        details,
				PreviewMode:    !input.Execute,
				CleanupTime:    cleanupTime,
				TotalProcessed: len(memoriesToClean),
			}

			log.InfoContext(ctx, "记忆清理完成", logger.Fields{
				"cleaned_count":    cleanedCount,
				"freed_space":      totalFreedSpace,
				"freed_tokens":     totalFreedTokens,
				"preview_mode":     !input.Execute,
				"cleanup_time_ms":  cleanupTime,
				"total_processed":  len(memoriesToClean),
			})

			return output, nil
		},
	)
}

// validateMemorySearchInput 验证记忆搜索输入参数
func validateMemorySearchInput(input MemorySearchInput) error {
	// 验证会话ID
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("无效的会话ID格式: %w", err)
	}

	// 验证查询文本
	if input.Query == "" {
		return fmt.Errorf("查询文本不能为空")
	}

	if len(input.Query) > 2000 {
		return fmt.Errorf("查询文本长度不能超过2000字符")
	}

	// 验证TopK
	if input.TopK <= 0 {
		return fmt.Errorf("TopK必须大于0")
	}

	if input.TopK > 20 {
		return fmt.Errorf("TopK不能超过20")
	}

	// 验证最小相似度
	if input.MinSimilarity < 0 || input.MinSimilarity > 1 {
		return fmt.Errorf("最小相似度必须在0-1之间")
	}

	// 验证时间范围
	if input.TimeRangeDays < 0 || input.TimeRangeDays > 365 {
		return fmt.Errorf("时间范围必须在0-365天之间")
	}

	// 验证记忆类型
	validMemoryTypes := map[string]bool{
		model.MemoryTypeShortTerm: true,
		model.MemoryTypeLongTerm:  true,
		model.MemoryTypeSummary:   true,
	}

	for _, memoryType := range input.MemoryTypes {
		if !validMemoryTypes[memoryType] {
			return fmt.Errorf("无效的记忆类型: %s", memoryType)
		}
	}

	return nil
}

// calculateCosineSimilarity 计算余弦相似度
func calculateCosineSimilarity(vec1, vec2 []float32) float32 {
	if len(vec1) != len(vec2) {
		return 0
	}

	var dotProduct, norm1, norm2 float32

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	// 余弦相似度 = 点积 / (向量1的模 * 向量2的模)
	similarity := dotProduct / (float32(sqrt(float64(norm1))) * float32(sqrt(float64(norm2))))

	// 确保结果在 [0, 1] 范围内
	if similarity < 0 {
		similarity = 0
	}
	if similarity > 1 {
		similarity = 1
	}

	return similarity
}

// sqrt 计算平方根
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// 使用牛顿迭代法计算平方根
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// validateMemoryStoreInput 验证记忆存储输入参数
func validateMemoryStoreInput(input MemoryStoreInput) error {
	// 验证会话ID
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("无效的会话ID格式: %w", err)
	}

	// 验证记忆类型
	validMemoryTypes := map[string]bool{
		model.MemoryTypeShortTerm: true,
		model.MemoryTypeLongTerm:  true,
		model.MemoryTypeSummary:   true,
	}

	if !validMemoryTypes[input.MemoryType] {
		return fmt.Errorf("无效的记忆类型: %s", input.MemoryType)
	}

	// 验证内容或消息ID
	if input.Content == "" && len(input.MessageIDs) == 0 {
		return fmt.Errorf("必须提供内容或消息ID")
	}

	// 验证内容长度
	if len(input.Content) > 10000 {
		return fmt.Errorf("内容长度不能超过10000字符")
	}

	// 验证消息ID格式
	for _, msgID := range input.MessageIDs {
		if _, err := uuid.Parse(msgID); err != nil {
			return fmt.Errorf("无效的消息ID格式: %s", msgID)
		}
	}

	// 验证重要性
	if input.Importance != nil {
		if *input.Importance < 0 || *input.Importance > 1 {
			return fmt.Errorf("重要性必须在0-1之间")
		}
	}

	// 验证过期天数
	if input.ExpirationDays < 0 || input.ExpirationDays > 365 {
		return fmt.Errorf("过期天数必须在0-365之间")
	}

	return nil
}

// extractKeywordsAndEntities 提取关键词和命名实体
// 这是一个简化的实现，实际应用中可以使用NLP库
func extractKeywordsAndEntities(content string) ([]string, []string) {
	// 简化实现：提取长度大于3的单词作为关键词
	keywords := make([]string, 0)
	entities := make([]string, 0)

	// 分词（简单按空格分割）
	words := splitWords(content)

	// 统计词频
	wordFreq := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 {
			wordFreq[word]++
		}
	}

	// 选择频率最高的词作为关键词
	type wordCount struct {
		word  string
		count int
	}

	var wordCounts []wordCount
	for word, count := range wordFreq {
		wordCounts = append(wordCounts, wordCount{word, count})
	}

	// 简单排序（冒泡排序）
	for i := 0; i < len(wordCounts); i++ {
		for j := i + 1; j < len(wordCounts); j++ {
			if wordCounts[j].count > wordCounts[i].count {
				wordCounts[i], wordCounts[j] = wordCounts[j], wordCounts[i]
			}
		}
	}

	// 取前5个作为关键词
	maxKeywords := 5
	if len(wordCounts) < maxKeywords {
		maxKeywords = len(wordCounts)
	}

	for i := 0; i < maxKeywords; i++ {
		keywords = append(keywords, wordCounts[i].word)
	}

	// 简化实现：提取首字母大写的词作为实体
	for _, word := range words {
		if len(word) > 0 && isUpperCase(word[0]) && len(word) > 2 {
			// 避免重复
			found := false
			for _, entity := range entities {
				if entity == word {
					found = true
					break
				}
			}
			if !found && len(entities) < 10 {
				entities = append(entities, word)
			}
		}
	}

	return keywords, entities
}

// splitWords 分词（简单实现）
func splitWords(text string) []string {
	words := make([]string, 0)
	currentWord := ""

	for _, char := range text {
		if isLetterOrDigit(char) {
			currentWord += string(char)
		} else {
			if len(currentWord) > 0 {
				words = append(words, currentWord)
				currentWord = ""
			}
		}
	}

	if len(currentWord) > 0 {
		words = append(words, currentWord)
	}

	return words
}

// isLetterOrDigit 判断是否为字母或数字
func isLetterOrDigit(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		(char >= 0x4e00 && char <= 0x9fff) // 中文字符范围
}

// isUpperCase 判断是否为大写字母
func isUpperCase(char byte) bool {
	return char >= 'A' && char <= 'Z'
}

// evaluateImportance 评估内容重要性
// 基于多个因素计算重要性分数（0-1）
func evaluateImportance(content string, keywords []string, entities []string, tokenCount int) float32 {
	var score float32 = 0.5 // 基础分数

	// 因素1：内容长度（更长的内容可能更重要）
	if tokenCount > 100 {
		score += 0.1
	}
	if tokenCount > 200 {
		score += 0.1
	}

	// 因素2：关键词数量
	if len(keywords) > 3 {
		score += 0.1
	}

	// 因素3：实体数量（包含更多实体可能更重要）
	if len(entities) > 2 {
		score += 0.1
	}
	if len(entities) > 5 {
		score += 0.1
	}

	// 因素4：检查是否包含重要标记词
	importantMarkers := []string{
		"重要", "关键", "必须", "注意", "警告",
		"important", "critical", "must", "warning", "note",
	}

	for _, marker := range importantMarkers {
		if contains(content, marker) {
			score += 0.1
			break
		}
	}

	// 确保分数在0-1范围内
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(text, substr string) bool {
	// 简单实现：转换为小写后比较
	textLower := toLowerCase(text)
	substrLower := toLowerCase(substr)

	return containsSubstring(textLower, substrLower)
}

// toLowerCase 转换为小写
func toLowerCase(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

// containsSubstring 检查是否包含子串
func containsSubstring(text, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(text) < len(substr) {
		return false
	}

	for i := 0; i <= len(text)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if text[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// validateMemoryCleanupInput 验证记忆清理输入参数
func validateMemoryCleanupInput(input MemoryCleanupInput) error {
	// 验证会话ID（如果提供）
	if input.SessionID != "" {
		if _, err := uuid.Parse(input.SessionID); err != nil {
			return fmt.Errorf("无效的会话ID格式: %w", err)
		}
	}

	// 验证清理策略
	validStrategies := map[string]bool{
		"expired":     true,
		"low_quality": true,
		"unused":      true,
		"all":         true,
	}

	if !validStrategies[input.Strategy] {
		return fmt.Errorf("无效的清理策略: %s", input.Strategy)
	}

	// 验证清理模式
	validModes := map[string]bool{
		"soft": true,
		"hard": true,
	}

	if !validModes[input.Mode] {
		return fmt.Errorf("无效的清理模式: %s", input.Mode)
	}

	// 验证批量大小
	if input.BatchSize <= 0 {
		return fmt.Errorf("批量大小必须大于0")
	}

	if input.BatchSize > 1000 {
		return fmt.Errorf("批量大小不能超过1000")
	}

	return nil
}

// getCleanupReason 根据策略和记忆属性确定清理原因
func getCleanupReason(memory *model.ConversationMemory, strategy string) string {
	switch strategy {
	case "expired":
		if memory.ExpiresAt != nil && memory.ExpiresAt.Before(time.Now()) {
			return fmt.Sprintf("已过期（过期时间：%s）", memory.ExpiresAt.Format("2006-01-02"))
		}
		return "已过期"

	case "low_quality":
		return fmt.Sprintf("低质量（重要性：%.2f，访问次数：%d）", memory.Importance, memory.AccessCount)

	case "unused":
		if memory.LastAccessAt != nil {
			daysSinceAccess := int(time.Since(*memory.LastAccessAt).Hours() / 24)
			return fmt.Sprintf("长期未使用（%d天未访问）", daysSinceAccess)
		}
		return "从未访问"

	case "all":
		return "批量清理"

	default:
		return "未知原因"
	}
}
