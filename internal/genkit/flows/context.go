// Package flows 实现上下文相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/go-playground/validator/v10"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
)

var validate = validator.New()

// RegisterContextFlows 注册上下文相关的 Flow
func RegisterContextFlows(g *genkit.Genkit, contextSvc service.ContextService) {
	// contextBuildFlow - 构建上下文
	genkit.DefineFlow(
		g,
		"contextBuildFlow",
		func(ctx context.Context, input ContextBuildInput) (ContextBuildOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateContextInput(input); err != nil {
				// 参数验证失败，记录错误
				return ContextBuildOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 权限验证（服务层也会验证，这里可以做额外的 Flow 层验证）
			if err := validateTenantAccess(ctx, input.SessionID, contextSvc); err != nil {
				// 权限验证失败，记录警告
				return ContextBuildOutput{}, fmt.Errorf("权限验证失败: %w", err)
			}

			// 3. 调用服务层构建上下文（服务层会处理缓存）
			result, err := contextSvc.BuildContext(ctx, &service.BuildContextRequest{
				SessionID:       input.SessionID,
				UserQuery:       input.UserQuery,
				MaxTokens:       input.MaxTokens,
				Strategy:        input.Strategy,
				IncludeSummary:  input.IncludeSummary,
				IncludeLongTerm: input.IncludeLongTerm,
				ShortTermWindow: input.ShortTermWindow,
			})
			if err != nil {
				return ContextBuildOutput{}, fmt.Errorf("构建上下文失败: %w", err)
			}

			// 4. 转换为输出格式
			output := convertToContextBuildOutput(result, time.Since(startTime).Milliseconds())

			return output, nil
		},
	)

	// contextOptimizeFlow - 优化上下文
	genkit.DefineFlow(
		g,
		"contextOptimizeFlow",
		func(ctx context.Context, input ContextOptimizeInput) (ContextOptimizeOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateOptimizeInput(input); err != nil {
				return ContextOptimizeOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 转换输入上下文为服务层格式
			contextResult := convertFromContextBuildOutput(input.Context)

			// 3. 调用服务层优化上下文
			result, err := contextSvc.OptimizeContext(ctx, &service.OptimizeContextRequest{
				Context:         contextResult,
				TargetTokens:    input.TargetTokens,
				Strategy:        input.Strategy,
				PreserveSummary: input.PreserveSummary,
			})
			if err != nil {
				return ContextOptimizeOutput{}, fmt.Errorf("优化上下文失败: %w", err)
			}

			// 4. 转换为输出格式
			output := convertToContextOptimizeOutput(result, time.Since(startTime).Milliseconds())

			return output, nil
		},
	)
}

// validateContextInput 验证输入参数
func validateContextInput(input ContextBuildInput) error {
	if err := validate.Struct(input); err != nil {
		return err
	}

	// 额外的业务验证
	if input.MaxTokens < 100 || input.MaxTokens > 32000 {
		return fmt.Errorf("maxTokens 必须在 100 到 32000 之间")
	}

	if input.ShortTermWindow < 1 || input.ShortTermWindow > 50 {
		return fmt.Errorf("shortTermWindow 必须在 1 到 50 之间")
	}

	if input.Strategy != "auto" && input.Strategy != "short" && input.Strategy != "full" {
		return fmt.Errorf("strategy 必须是 auto、short 或 full")
	}

	return nil
}

// validateTenantAccess 验证租户访问权限
func validateTenantAccess(ctx context.Context, sessionID string, contextSvc service.ContextService) error {
	// 获取 JWT 声明
	claims, ok := middleware.GetJWTClaims(ctx)
	if !ok || claims == nil {
		return fmt.Errorf("未认证：无法获取用户信息")
	}

	// 调用服务层验证权限
	// 服务层的 BuildContext 方法会进行完整的权限验证
	// 这里只做基本的认证检查
	return nil
}

// convertToContextBuildOutput 转换服务层结果为输出格式
func convertToContextBuildOutput(result *service.ContextResult, buildTime int64) ContextBuildOutput {
	output := ContextBuildOutput{
		SessionID:         result.SessionID,
		TotalTokens:       result.TotalTokens,
		Strategy:          result.Strategy,
		QualityScore:      result.QualityScore,
		BuildTime:         buildTime,
		ShortTermMessages: make([]MessageContext, 0),
		LongTermMemories:  make([]MemoryContext, 0),
	}

	// 转换摘要
	if result.Summary != nil {
		output.Summary = &SummaryContext{
			Content:    result.Summary.Content,
			TokenCount: result.Summary.TokenCount,
			CreatedAt:  result.Summary.CreatedAt.Format(time.RFC3339),
			Coverage:   fmt.Sprintf("%d 条消息", result.Summary.MessageCount),
		}
	}

	// 转换长期记忆
	for _, memory := range result.LongTermMemories {
		output.LongTermMemories = append(output.LongTermMemories, MemoryContext{
			ID:         memory.ID.String(),
			Content:    memory.Content,
			TokenCount: memory.TokenCount,
			Importance: memory.Importance,
			Similarity: 0.0, // 需要从检索结果中获取
			CreatedAt:  memory.CreatedAt.Format(time.RFC3339),
		})
	}

	// 转换短期消息
	for _, message := range result.ShortTermMessages {
		output.ShortTermMessages = append(output.ShortTermMessages, MessageContext{
			ID:         message.ID.String(),
			Role:       message.Role,
			Content:    message.Content,
			TokenCount: 0, // 需要计算
			CreatedAt:  message.CreatedAt.Format(time.RFC3339),
		})
	}

	return output
}

// validateOptimizeInput 验证优化输入参数
func validateOptimizeInput(input ContextOptimizeInput) error {
	if err := validate.Struct(input); err != nil {
		return err
	}

	// 验证上下文不为空
	if input.Context == nil {
		return fmt.Errorf("context 不能为空")
	}

	// 验证目标 Token 数
	if input.TargetTokens < 100 || input.TargetTokens > 32000 {
		return fmt.Errorf("targetTokens 必须在 100 到 32000 之间")
	}

	// 验证策略
	if input.Strategy != "aggressive" && input.Strategy != "balanced" && input.Strategy != "conservative" {
		return fmt.Errorf("strategy 必须是 aggressive、balanced 或 conservative")
	}

	// 验证目标 Token 数小于当前 Token 数
	if input.TargetTokens >= input.Context.TotalTokens {
		return fmt.Errorf("targetTokens (%d) 必须小于当前 Token 数 (%d)", input.TargetTokens, input.Context.TotalTokens)
	}

	return nil
}

// convertFromContextBuildOutput 将 ContextBuildOutput 转换为服务层的 ContextResult
func convertFromContextBuildOutput(output *ContextBuildOutput) *service.ContextResult {
	if output == nil {
		return nil
	}

	result := &service.ContextResult{
		SessionID:         output.SessionID,
		TotalTokens:       output.TotalTokens,
		Strategy:          output.Strategy,
		QualityScore:      output.QualityScore,
		ShortTermMessages: make([]*model.ChatMessage, 0),
		LongTermMemories:  make([]*model.ConversationMemory, 0),
	}

	// 转换摘要
	if output.Summary != nil {
		createdAt, _ := time.Parse(time.RFC3339, output.Summary.CreatedAt)
		result.Summary = &model.ConversationSummary{
			Content:    output.Summary.Content,
			TokenCount: output.Summary.TokenCount,
			CreatedAt:  createdAt,
		}
	}

	// 转换长期记忆
	for _, memory := range output.LongTermMemories {
		createdAt, _ := time.Parse(time.RFC3339, memory.CreatedAt)
		result.LongTermMemories = append(result.LongTermMemories, &model.ConversationMemory{
			Content:    memory.Content,
			TokenCount: memory.TokenCount,
			Importance: memory.Importance,
			CreatedAt:  createdAt,
		})
	}

	// 转换短期消息
	for _, message := range output.ShortTermMessages {
		createdAt, _ := time.Parse(time.RFC3339, message.CreatedAt)
		result.ShortTermMessages = append(result.ShortTermMessages, &model.ChatMessage{
			Role:      message.Role,
			Content:   message.Content,
			CreatedAt: createdAt,
		})
	}

	return result
}

// convertToContextOptimizeOutput 转换服务层结果为优化输出格式
func convertToContextOptimizeOutput(result *service.ContextResult, optimizationTime int64) ContextOptimizeOutput {
	output := ContextOptimizeOutput{
		SessionID:        result.SessionID,
		TotalTokens:      result.TotalTokens,
		Strategy:         result.Strategy,
		QualityScore:     result.QualityScore,
		QualityLoss:      result.QualityLoss,
		OptimizationTime: optimizationTime,
		Operations:       result.Operations,
		ShortTermMessages: make([]MessageContext, 0),
		LongTermMemories:  make([]MemoryContext, 0),
	}

	// 转换摘要
	if result.Summary != nil {
		output.Summary = &SummaryContext{
			Content:    result.Summary.Content,
			TokenCount: result.Summary.TokenCount,
			CreatedAt:  result.Summary.CreatedAt.Format(time.RFC3339),
			Coverage:   fmt.Sprintf("%d 条消息", result.Summary.MessageCount),
		}
	}

	// 转换长期记忆
	for _, memory := range result.LongTermMemories {
		output.LongTermMemories = append(output.LongTermMemories, MemoryContext{
			ID:         memory.ID.String(),
			Content:    memory.Content,
			TokenCount: memory.TokenCount,
			Importance: memory.Importance,
			Similarity: 0.0,
			CreatedAt:  memory.CreatedAt.Format(time.RFC3339),
		})
	}

	// 转换短期消息
	for _, message := range result.ShortTermMessages {
		output.ShortTermMessages = append(output.ShortTermMessages, MessageContext{
			ID:         message.ID.String(),
			Role:       message.Role,
			Content:    message.Content,
			TokenCount: 0,
			CreatedAt:  message.CreatedAt.Format(time.RFC3339),
		})
	}

	return output
}
