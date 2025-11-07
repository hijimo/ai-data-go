package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/go-playground/validator/v10"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/monitoring"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/pkg/errors"
)

// validate 参数验证器
var validate = validator.New()

// RegisterContextFlows 注册上下文相关的Flow
// 参数：
//   - g: Genkit实例
//   - contextSvc: 上下文服务
func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
	// 注册上下文构建Flow（已应用监控中间件）
	// 注意：监控逻辑已经集成在contextBuildFlow内部
	genkit.DefineFlow(
		g,
		"contextBuildFlow",
		contextBuildFlow(contextSvc),
	)
}

// contextBuildFlow 创建上下文构建Flow
// 该Flow负责构建智能对话上下文，包括短期记忆、长期记忆和摘要
func contextBuildFlow(contextSvc service.ContextService) func(context.Context, ContextBuildInput) (ContextBuildOutput, error) {
	return func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
		startTime := time.Now()

		// 记录Flow开始执行
		logger.InfoContext(ctx, "开始执行上下文构建Flow", logger.Fields{
			"session_id": input.SessionID,
			"user_query": input.UserQuery,
			"max_tokens": input.MaxTokens,
			"strategy":   input.Strategy,
		})

		// 1. 参数验证
		if err := validateContextInput(input); err != nil {
			logger.ErrorContext(ctx, "参数验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return ContextBuildOutput{}, err
		}

		// 2. 权限验证
		if err := validateTenantAccess(ctx, input.SessionID); err != nil {
			logger.WarnContext(ctx, "权限验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return ContextBuildOutput{}, err
		}

		// 3. 调用服务层构建上下文
		result, err := contextSvc.BuildContext(ctx, service.BuildContextRequest{
			SessionID:       input.SessionID,
			UserQuery:       input.UserQuery,
			MaxTokens:       input.MaxTokens,
			Strategy:        input.Strategy,
			IncludeSummary:  input.IncludeSummary,
			IncludeLongTerm: input.IncludeLongTerm,
			ShortTermWindow: input.ShortTermWindow,
		})
		if err != nil {
			logger.ErrorContext(ctx, "构建上下文失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			// 记录监控指标
			recordFlowMetrics(ctx, "contextBuildFlow", "error", time.Since(startTime))
			return ContextBuildOutput{}, fmt.Errorf("构建上下文失败: %w", err)
		}

		// 4. 转换为输出格式
		output := ContextBuildOutput{
			SessionID:         result.SessionID,
			Summary:           convertSummary(result.Summary),
			LongTermMemories:  convertMemories(result.LongTermMemories),
			ShortTermMessages: convertMessages(result.ShortTermMessages),
			TotalTokens:       result.TotalTokens,
			Strategy:          result.Strategy,
			QualityScore:      result.QualityScore,
			BuildTime:         time.Since(startTime).Milliseconds(),
		}

		// 5. 记录成功日志和监控指标
		logger.InfoContext(ctx, "上下文构建成功", logger.Fields{
			"session_id":        output.SessionID,
			"total_tokens":      output.TotalTokens,
			"quality_score":     output.QualityScore,
			"build_time_ms":     output.BuildTime,
			"short_term_count":  len(output.ShortTermMessages),
			"long_term_count":   len(output.LongTermMemories),
			"has_summary":       output.Summary != nil,
		})

		recordFlowMetrics(ctx, "contextBuildFlow", "success", time.Since(startTime))

		return output, nil
	}
}

// validateContextInput 验证输入参数
func validateContextInput(input ContextBuildInput) error {
	// 使用validator进行结构体验证
	if err := validate.Struct(input); err != nil {
		return errors.NewBadRequestError(fmt.Sprintf("参数验证失败: %v", err))
	}

	// 额外的业务逻辑验证
	if input.SessionID == "" {
		return errors.NewBadRequestError("会话ID不能为空")
	}

	if input.UserQuery == "" {
		return errors.NewBadRequestError("用户查询不能为空")
	}

	if input.MaxTokens < 100 || input.MaxTokens > 32000 {
		return errors.NewBadRequestError("MaxTokens必须在100到32000之间")
	}

	if input.ShortTermWindow < 1 || input.ShortTermWindow > 50 {
		return errors.NewBadRequestError("ShortTermWindow必须在1到50之间")
	}

	// 验证策略值
	validStrategies := map[string]bool{
		"auto":  true,
		"short": true,
		"full":  true,
	}
	if !validStrategies[input.Strategy] {
		return errors.NewBadRequestError("Strategy必须是auto、short或full之一")
	}

	return nil
}

// validateTenantAccess 验证租户访问权限
// 确保用户只能访问自己租户的会话
func validateTenantAccess(ctx context.Context, sessionID string) error {
	// 获取JWT声明
	claims, ok := middleware.GetJWTClaims(ctx)
	if !ok || claims == nil {
		return errors.NewUnauthorizedError("未认证")
	}

	// 注意：这里简化了权限验证逻辑
	// 实际的租户验证应该在服务层进行，因为需要查询会话所属的租户
	// 这里只做基础的认证检查
	if claims.Subject == "" {
		return errors.NewUnauthorizedError("无效的用户身份")
	}

	// 详细的租户隔离验证将在ContextService.BuildContext中进行
	return nil
}

// convertSummary 转换摘要为输出格式
func convertSummary(summary *model.ConversationSummary) *SummaryContext {
	if summary == nil {
		return nil
	}

	return &SummaryContext{
		Content:    summary.Content,
		TokenCount: summary.TokenCount,
		CreatedAt:  summary.CreatedAt.Format(time.RFC3339),
		Coverage:   fmt.Sprintf("%d条消息", summary.MessageCount),
	}
}

// convertMemories 转换记忆列表为输出格式
func convertMemories(memories []*model.ConversationMemory) []MemoryContext {
	if len(memories) == 0 {
		return []MemoryContext{}
	}

	result := make([]MemoryContext, 0, len(memories))
	for _, mem := range memories {
		result = append(result, MemoryContext{
			ID:         mem.ID.String(),
			Content:    mem.Content,
			TokenCount: mem.TokenCount,
			Importance: mem.Importance,
			Similarity: 0.0, // 相似度需要从向量检索结果中获取
			CreatedAt:  mem.CreatedAt.Format(time.RFC3339),
		})
	}

	return result
}

// convertMessages 转换消息列表为输出格式
func convertMessages(messages []*model.ChatMessage) []MessageContext {
	if len(messages) == 0 {
		return []MessageContext{}
	}

	result := make([]MessageContext, 0, len(messages))
	for _, msg := range messages {
		result = append(result, MessageContext{
			ID:         msg.ID.String(),
			Role:       msg.Role,
			Content:    msg.Content,
			TokenCount: msg.Tokens,
			CreatedAt:  msg.CreatedAt.Format(time.RFC3339),
		})
	}

	return result
}

// recordFlowMetrics 记录Flow执行的监控指标
func recordFlowMetrics(ctx context.Context, flowName string, status string, duration time.Duration) {
	// 记录Flow执行次数
	monitoring.RecordFlowExecution(flowName, status)

	// 记录Flow执行时间
	monitoring.RecordFlowDuration(flowName, duration)

	// 从上下文中提取租户ID并记录
	if claims, ok := middleware.GetJWTClaims(ctx); ok && claims != nil {
		// 可以在这里记录租户级别的指标
		logger.InfoContext(ctx, "Flow执行指标已记录", logger.Fields{
			"flow_name":   flowName,
			"status":      status,
			"duration_ms": duration.Milliseconds(),
			"tenant_id":   claims.TenantID,
		})
	}
}
