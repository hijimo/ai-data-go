package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// MemorySearchInput 记忆检索Flow的输入参数
type MemorySearchInput struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid"`
	// 查询文本
	Query string `json:"query" validate:"required,max=2000"`
	// 返回结果数量
	TopK int `json:"topK" validate:"min=1,max=20"`
	// 最小相似度（0-1）
	MinSimilarity float32 `json:"minSimilarity" validate:"min=0,max=1"`
	// 时间范围（天数，0表示不限制）
	TimeRangeDays int `json:"timeRangeDays" validate:"min=0,max=365"`
	// 记忆类型过滤
	MemoryTypes []string `json:"memoryTypes" validate:"dive,oneof=short_term long_term summary"`
	// 是否包含跨会话检索
	IncludeCrossSessions bool `json:"includeCrossSessions"`
}

// MemorySearchOutput 记忆检索Flow的输出结果
type MemorySearchOutput struct {
	// 检索到的记忆列表
	Memories []MemoryResult `json:"memories"`
	// 总共找到的数量
	TotalFound int `json:"totalFound"`
	// 返回的数量
	ReturnedCount int `json:"returnedCount"`
	// 检索耗时（毫秒）
	SearchTime int64 `json:"searchTime"`
	// 平均相似度
	AverageSimilarity float32 `json:"averageSimilarity"`
	// 检索策略
	SearchStrategy string `json:"searchStrategy"`
}

// MemoryResult 记忆检索结果
type MemoryResult struct {
	// 记忆ID
	ID string `json:"id"`
	// 会话ID
	SessionID string `json:"sessionId"`
	// 记忆类型
	MemoryType string `json:"memoryType"`
	// 记忆内容
	Content string `json:"content"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 相似度评分（0-1）
	Similarity float32 `json:"similarity"`
	// 重要性评分（0-1）
	Importance float32 `json:"importance"`
	// 综合评分（相似度 × 重要性）
	Score float32 `json:"score"`
	// 访问次数
	AccessCount int `json:"accessCount"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 最后访问时间
	LastAccessAt string `json:"lastAccessAt"`
	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// MemoryStoreInput 记忆存储Flow的输入参数
type MemoryStoreInput struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid"`
	// 消息ID列表（可选）
	MessageIDs []string `json:"messageIds" validate:"dive,uuid"`
	// 记忆类型
	MemoryType string `json:"memoryType" validate:"required,oneof=short_term long_term summary"`
	// 记忆内容
	Content string `json:"content" validate:"required,max=4000"`
	// 重要性评分（0-1）
	Importance float32 `json:"importance" validate:"min=0,max=1"`
	// 过期天数（0表示不过期）
	ExpirationDays int `json:"expirationDays" validate:"min=0,max=365"`
	// 元数据
	Metadata map[string]interface{} `json:"metadata"`
}

// MemoryStoreOutput 记忆存储Flow的输出结果
type MemoryStoreOutput struct {
	// 记忆ID
	MemoryID string `json:"memoryId"`
	// 会话ID
	SessionID string `json:"sessionId"`
	// 记忆类型
	MemoryType string `json:"memoryType"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 重要性评分
	Importance float32 `json:"importance"`
	// 过期时间
	ExpiresAt string `json:"expiresAt,omitempty"`
	// 向量生成状态
	VectorStatus string `json:"vectorStatus"`
	// 存储耗时（毫秒）
	StoreTime int64 `json:"storeTime"`
}

// MemoryCleanupInput 记忆清理Flow的输入参数
type MemoryCleanupInput struct {
	// 会话ID（可选，为空则清理租户所有记忆）
	SessionID string `json:"sessionId" validate:"omitempty,uuid"`
	// 清理策略：expired（过期）、low_quality（低质量）、unused（未使用）、all（全部）
	Strategy string `json:"strategy" validate:"required,oneof=expired low_quality unused all"`
	// 清理模式：soft（软删除）、hard（硬删除）
	Mode string `json:"mode" validate:"required,oneof=soft hard"`
	// 批量处理大小
	BatchSize int `json:"batchSize" validate:"min=10,max=1000"`
	// 是否执行删除（false时仅预览）
	Execute bool `json:"execute"`
}

// MemoryCleanupOutput 记忆清理Flow的输出结果
type MemoryCleanupOutput struct {
	// 清理数量
	CleanedCount int `json:"cleanedCount"`
	// 释放空间（字节）
	FreedSpace int64 `json:"freedSpace"`
	// 清理详情
	Details []CleanupDetailResult `json:"details"`
	// 是否为预览模式
	Preview bool `json:"preview"`
	// 清理耗时（毫秒）
	CleanupTime int64 `json:"cleanupTime"`
}

// CleanupDetailResult 清理详情结果
type CleanupDetailResult struct {
	// 记忆ID
	MemoryID string `json:"memoryId"`
	// 清理原因
	Reason string `json:"reason"`
	// 记忆大小（字节）
	Size int64 `json:"size"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 最后访问时间
	LastAccess string `json:"lastAccess"`
}

// RegisterMemoryFlows 注册记忆管理相关的Flow
func RegisterMemoryFlows(g *genkit.Genkit, memorySvc service.MemoryService) {
	// 注册记忆检索Flow
	genkit.DefineFlow(
		g,
		"memorySearchFlow",
		memorySearchFlow(memorySvc),
	)

	// 注册记忆存储Flow
	genkit.DefineFlow(
		g,
		"memoryStoreFlow",
		memoryStoreFlow(memorySvc),
	)

	// 注册记忆清理Flow
	genkit.DefineFlow(
		g,
		"memoryCleanupFlow",
		memoryCleanupFlow(memorySvc),
	)
}

// memorySearchFlow 创建记忆检索Flow
func memorySearchFlow(memorySvc service.MemoryService) func(context.Context, MemorySearchInput) (MemorySearchOutput, error) {
	return func(ctx context.Context, input MemorySearchInput) (MemorySearchOutput, error) {
		startTime := time.Now()

		// 1. 参数验证
		if err := validateMemorySearchInput(input); err != nil {
			logger.ErrorContext(ctx, "记忆检索参数验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return MemorySearchOutput{}, fmt.Errorf("参数验证失败: %w", err)
		}

		// 2. 解析会话ID
		sessionID, err := uuid.Parse(input.SessionID)
		if err != nil {
			return MemorySearchOutput{}, fmt.Errorf("无效的会话ID: %w", err)
		}

		// 3. 构建检索请求
		req := &service.SearchMemoriesRequest{
			SessionID:            sessionID,
			Query:                input.Query,
			TopK:                 input.TopK,
			MinSimilarity:        input.MinSimilarity,
			TimeRangeDays:        input.TimeRangeDays,
			MemoryTypes:          input.MemoryTypes,
			IncludeCrossSessions: input.IncludeCrossSessions,
		}

		// 4. 调用服务层检索记忆
		results, err := memorySvc.SearchMemories(ctx, req)
		if err != nil {
			logger.ErrorContext(ctx, "记忆检索失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
				"query":      input.Query,
			})
			return MemorySearchOutput{}, fmt.Errorf("记忆检索失败: %w", err)
		}

		// 5. 转换为输出格式
		memories := make([]MemoryResult, 0, len(results))
		var totalSimilarity float32
		for _, result := range results {
			// 转换Metadata
			var metadata map[string]interface{}
			if result.Memory.Metadata != nil && len(result.Memory.Metadata) > 0 {
				_ = json.Unmarshal(result.Memory.Metadata, &metadata)
			}

			memories = append(memories, MemoryResult{
				ID:           result.Memory.ID.String(),
				SessionID:    result.Memory.SessionID.String(),
				MemoryType:   result.Memory.MemoryType,
				Content:      result.Memory.Content,
				TokenCount:   result.Memory.TokenCount,
				Similarity:   result.Similarity,
				Importance:   result.Memory.Importance,
				Score:        result.Score,
				AccessCount:  result.Memory.AccessCount,
				CreatedAt:    result.Memory.CreatedAt.Format(time.RFC3339),
				LastAccessAt: formatTimePtr(result.Memory.LastAccessAt),
				Metadata:     metadata,
			})
			totalSimilarity += result.Similarity
		}

		// 6. 计算平均相似度
		avgSimilarity := float32(0)
		if len(memories) > 0 {
			avgSimilarity = totalSimilarity / float32(len(memories))
		}

		// 7. 确定检索策略
		strategy := "session"
		if input.IncludeCrossSessions {
			strategy = "cross_session"
		}

		// 8. 构建输出结果
		output := MemorySearchOutput{
			Memories:          memories,
			TotalFound:        len(memories),
			ReturnedCount:     len(memories),
			SearchTime:        time.Since(startTime).Milliseconds(),
			AverageSimilarity: avgSimilarity,
			SearchStrategy:    strategy,
		}

		logger.InfoContext(ctx, "记忆检索完成", logger.Fields{
			"session_id":  input.SessionID,
			"query":       input.Query,
			"found_count": len(memories),
			"duration_ms": output.SearchTime,
		})

		return output, nil
	}
}

// memoryStoreFlow 创建记忆存储Flow
func memoryStoreFlow(memorySvc service.MemoryService) func(context.Context, MemoryStoreInput) (MemoryStoreOutput, error) {
	return func(ctx context.Context, input MemoryStoreInput) (MemoryStoreOutput, error) {
		startTime := time.Now()

		// 1. 参数验证
		if err := validateMemoryStoreInput(input); err != nil {
			logger.ErrorContext(ctx, "记忆存储参数验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return MemoryStoreOutput{}, fmt.Errorf("参数验证失败: %w", err)
		}

		// 2. 解析会话ID
		sessionID, err := uuid.Parse(input.SessionID)
		if err != nil {
			return MemoryStoreOutput{}, fmt.Errorf("无效的会话ID: %w", err)
		}

		// 3. 解析消息ID列表
		messageIDs := make([]uuid.UUID, 0, len(input.MessageIDs))
		for _, idStr := range input.MessageIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return MemoryStoreOutput{}, fmt.Errorf("无效的消息ID %s: %w", idStr, err)
			}
			messageIDs = append(messageIDs, id)
		}

		// 4. 构建存储请求
		req := &service.StoreMemoryRequest{
			SessionID:      sessionID,
			MessageIDs:     messageIDs,
			MemoryType:     input.MemoryType,
			Content:        input.Content,
			Importance:     input.Importance,
			ExpirationDays: input.ExpirationDays,
			Metadata:       input.Metadata,
		}

		// 5. 调用服务层存储记忆
		memory, err := memorySvc.StoreMemory(ctx, req)
		if err != nil {
			logger.ErrorContext(ctx, "记忆存储失败", logger.Fields{
				"error":       err.Error(),
				"session_id":  input.SessionID,
				"memory_type": input.MemoryType,
			})
			return MemoryStoreOutput{}, fmt.Errorf("记忆存储失败: %w", err)
		}

		// 6. 构建输出结果
		output := MemoryStoreOutput{
			MemoryID:     memory.ID.String(),
			SessionID:    memory.SessionID.String(),
			MemoryType:   memory.MemoryType,
			TokenCount:   memory.TokenCount,
			Importance:   memory.Importance,
			ExpiresAt:    formatTimePtr(memory.ExpiresAt),
			VectorStatus: "generated", // 向量已生成
			StoreTime:    time.Since(startTime).Milliseconds(),
		}

		logger.InfoContext(ctx, "记忆存储完成", logger.Fields{
			"memory_id":   output.MemoryID,
			"session_id":  input.SessionID,
			"memory_type": input.MemoryType,
			"token_count": output.TokenCount,
			"duration_ms": output.StoreTime,
		})

		return output, nil
	}
}

// memoryCleanupFlow 创建记忆清理Flow
func memoryCleanupFlow(memorySvc service.MemoryService) func(context.Context, MemoryCleanupInput) (MemoryCleanupOutput, error) {
	return func(ctx context.Context, input MemoryCleanupInput) (MemoryCleanupOutput, error) {
		startTime := time.Now()

		// 1. 参数验证
		if err := validateMemoryCleanupInput(input); err != nil {
			logger.ErrorContext(ctx, "记忆清理参数验证失败", logger.Fields{
				"error":    err.Error(),
				"strategy": input.Strategy,
			})
			return MemoryCleanupOutput{}, fmt.Errorf("参数验证失败: %w", err)
		}

		// 2. 解析会话ID（如果提供）
		var sessionID uuid.UUID
		if input.SessionID != "" {
			var err error
			sessionID, err = uuid.Parse(input.SessionID)
			if err != nil {
				return MemoryCleanupOutput{}, fmt.Errorf("无效的会话ID: %w", err)
			}
		}

		// 3. 构建清理请求
		req := &service.CleanupMemoriesRequest{
			SessionID: sessionID,
			Strategy:  input.Strategy,
			Mode:      input.Mode,
			BatchSize: input.BatchSize,
			Execute:   input.Execute,
		}

		// 4. 调用服务层清理记忆
		result, err := memorySvc.CleanupMemories(ctx, req)
		if err != nil {
			logger.ErrorContext(ctx, "记忆清理失败", logger.Fields{
				"error":    err.Error(),
				"strategy": input.Strategy,
				"mode":     input.Mode,
			})
			return MemoryCleanupOutput{}, fmt.Errorf("记忆清理失败: %w", err)
		}

		// 5. 转换清理详情
		details := make([]CleanupDetailResult, 0, len(result.Details))
		for _, detail := range result.Details {
			details = append(details, CleanupDetailResult{
				MemoryID:   detail.MemoryID.String(),
				Reason:     detail.Reason,
				Size:       detail.Size,
				CreatedAt:  detail.CreatedAt.Format(time.RFC3339),
				LastAccess: detail.LastAccess.Format(time.RFC3339),
			})
		}

		// 6. 构建输出结果
		output := MemoryCleanupOutput{
			CleanedCount: result.CleanedCount,
			FreedSpace:   result.FreedSpace,
			Details:      details,
			Preview:      result.Preview,
			CleanupTime:  time.Since(startTime).Milliseconds(),
		}

		logger.InfoContext(ctx, "记忆清理完成", logger.Fields{
			"strategy":      input.Strategy,
			"mode":          input.Mode,
			"cleaned_count": output.CleanedCount,
			"freed_space":   output.FreedSpace,
			"preview":       output.Preview,
			"duration_ms":   output.CleanupTime,
		})

		return output, nil
	}
}

// validateMemorySearchInput 验证记忆检索输入参数
func validateMemorySearchInput(input MemorySearchInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	if input.Query == "" {
		return fmt.Errorf("查询文本不能为空")
	}
	if input.TopK <= 0 {
		return fmt.Errorf("TopK必须大于0")
	}
	if input.MinSimilarity < 0 || input.MinSimilarity > 1 {
		return fmt.Errorf("最小相似度必须在0-1之间")
	}
	return nil
}

// validateMemoryStoreInput 验证记忆存储输入参数
func validateMemoryStoreInput(input MemoryStoreInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	if input.MemoryType == "" {
		return fmt.Errorf("记忆类型不能为空")
	}
	if input.Content == "" {
		return fmt.Errorf("记忆内容不能为空")
	}
	if input.Importance < 0 || input.Importance > 1 {
		return fmt.Errorf("重要性评分必须在0-1之间")
	}
	return nil
}

// validateMemoryCleanupInput 验证记忆清理输入参数
func validateMemoryCleanupInput(input MemoryCleanupInput) error {
	if input.Strategy == "" {
		return fmt.Errorf("清理策略不能为空")
	}
	if input.Mode == "" {
		return fmt.Errorf("清理模式不能为空")
	}
	if input.BatchSize <= 0 {
		return fmt.Errorf("批量处理大小必须大于0")
	}
	return nil
}

// formatTimePtr 格式化时间指针
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
