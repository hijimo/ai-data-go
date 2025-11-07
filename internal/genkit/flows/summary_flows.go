package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/internal/service/session"
)

// SummaryGenerateInput 摘要生成Flow的输入参数
type SummaryGenerateInput struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid"`
	// 消息ID列表（可选）
	MessageIDs []string `json:"messageIds,omitempty" validate:"dive,uuid"`
	// 起始消息ID（可选）
	StartMessageID string `json:"startMessageId,omitempty" validate:"omitempty,uuid"`
	// 结束消息ID（可选）
	EndMessageID string `json:"endMessageId,omitempty" validate:"omitempty,uuid"`
	// 前一个摘要内容（用于增量摘要）
	PreviousSummary string `json:"previousSummary,omitempty" validate:"max=2000"`
	// 摘要类型：incremental（增量）、full（完整）
	SummaryType string `json:"summaryType" validate:"required,oneof=incremental full"`
	// 目标长度（Token数量）
	TargetLength int `json:"targetLength" validate:"min=50,max=1000"`
}

// SummaryGenerateOutput 摘要生成Flow的输出结果
type SummaryGenerateOutput struct {
	// 摘要ID
	SummaryID string `json:"summaryId"`
	// 摘要内容
	Summary string `json:"summary"`
	// Token数量
	TokenCount int `json:"tokenCount"`
	// 消息数量
	MessageCount int `json:"messageCount"`
	// 起始消息ID
	StartMessageID string `json:"startMessageId"`
	// 结束消息ID
	EndMessageID string `json:"endMessageId"`
	// 质量评分（0-1）
	QualityScore float64 `json:"qualityScore"`
	// 压缩率（节省的Token比例）
	CompressionRate float64 `json:"compressionRate"`
	// 关键主题列表
	KeyTopics []string `json:"keyTopics"`
	// 生成耗时（毫秒）
	GenerationTime int64 `json:"generationTime"`
}

// SummaryTriggerCheckInput 摘要触发检查Flow的输入参数
type SummaryTriggerCheckInput struct {
	// 会话ID
	SessionID string `json:"sessionId" validate:"required,uuid"`
	// 检查模式：auto（自动）、force（强制）
	CheckMode string `json:"checkMode" validate:"oneof=auto force"`
}

// SummaryTriggerCheckOutput 摘要触发检查Flow的输出结果
type SummaryTriggerCheckOutput struct {
	// 是否应该生成摘要
	ShouldSummarize bool `json:"shouldSummarize"`
	// 触发原因
	TriggerReason string `json:"triggerReason"`
	// 建议包含的消息ID列表
	MessageIDs []string `json:"messageIds"`
	// 消息数量
	MessageCount int `json:"messageCount"`
	// 估算的Token节省量
	EstimatedTokenSaving int `json:"estimatedTokenSaving"`
	// 紧急程度（0-1）
	Urgency float64 `json:"urgency"`
	// 推荐的摘要类型
	RecommendedType string `json:"recommendedType"`
	// 检查耗时（毫秒）
	CheckTime int64 `json:"checkTime"`
}

// RegisterSummaryFlows 注册摘要相关的Flow
// 参数：
//   - g: Genkit实例
//   - summarySvc: 摘要服务
func RegisterSummaryFlows(g *genkit.Genkit, summarySvc session.SummaryService) {
	// 注册摘要生成Flow
	genkit.DefineFlow(
		g,
		"summaryGenerateFlow",
		summaryGenerateFlow(summarySvc),
	)

	// 注册摘要触发检查Flow
	genkit.DefineFlow(
		g,
		"summaryTriggerCheckFlow",
		summaryTriggerCheckFlow(summarySvc),
	)

	logger.Info("摘要管理Flow注册完成", logger.Fields{
		"flows": []string{"summaryGenerateFlow", "summaryTriggerCheckFlow"},
	})
}

// summaryGenerateFlow 创建摘要生成Flow
func summaryGenerateFlow(summarySvc session.SummaryService) func(context.Context, SummaryGenerateInput) (SummaryGenerateOutput, error) {
	return func(ctx context.Context, input SummaryGenerateInput) (SummaryGenerateOutput, error) {
		startTime := time.Now()

		// 1. 参数验证
		if err := validateSummaryGenerateInput(input); err != nil {
			logger.ErrorContext(ctx, "摘要生成参数验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryGenerateOutput{}, fmt.Errorf("参数验证失败: %w", err)
		}

		// 2. 获取JWT声明进行权限验证
		claims, ok := auth.GetJWTClaimsFromContext(ctx)
		if !ok || claims == nil {
			logger.WarnContext(ctx, "未认证的摘要生成请求", logger.Fields{
				"session_id": input.SessionID,
			})
			return SummaryGenerateOutput{}, fmt.Errorf("未认证")
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			logger.ErrorContext(ctx, "无效的租户ID", logger.Fields{
				"error":     err.Error(),
				"tenant_id": claims.TenantID,
			})
			return SummaryGenerateOutput{}, fmt.Errorf("无效的租户ID: %w", err)
		}

		sessionID, err := uuid.Parse(input.SessionID)
		if err != nil {
			logger.ErrorContext(ctx, "无效的会话ID", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryGenerateOutput{}, fmt.Errorf("无效的会话ID: %w", err)
		}

		// 3. 构建服务请求
		req := &session.GenerateSummaryRequest{
			TenantID:        tenantID,
			SessionID:       sessionID,
			PreviousSummary: input.PreviousSummary,
			SummaryType:     input.SummaryType,
			TargetLength:    input.TargetLength,
		}

		// 处理消息ID列表
		if len(input.MessageIDs) > 0 {
			req.MessageIDs = make([]uuid.UUID, len(input.MessageIDs))
			for i, idStr := range input.MessageIDs {
				id, err := uuid.Parse(idStr)
				if err != nil {
					logger.ErrorContext(ctx, "无效的消息ID", logger.Fields{
						"error":      err.Error(),
						"message_id": idStr,
					})
					return SummaryGenerateOutput{}, fmt.Errorf("无效的消息ID: %w", err)
				}
				req.MessageIDs[i] = id
			}
		}

		// 处理起始和结束消息ID
		if input.StartMessageID != "" {
			startMsgID, err := uuid.Parse(input.StartMessageID)
			if err != nil {
				logger.ErrorContext(ctx, "无效的起始消息ID", logger.Fields{
					"error":            err.Error(),
					"start_message_id": input.StartMessageID,
				})
				return SummaryGenerateOutput{}, fmt.Errorf("无效的起始消息ID: %w", err)
			}
			req.StartMessageID = &startMsgID
		}

		if input.EndMessageID != "" {
			endMsgID, err := uuid.Parse(input.EndMessageID)
			if err != nil {
				logger.ErrorContext(ctx, "无效的结束消息ID", logger.Fields{
					"error":          err.Error(),
					"end_message_id": input.EndMessageID,
				})
				return SummaryGenerateOutput{}, fmt.Errorf("无效的结束消息ID: %w", err)
			}
			req.EndMessageID = &endMsgID
		}

		// 4. 调用服务层生成摘要
		logger.InfoContext(ctx, "开始生成摘要", logger.Fields{
			"session_id":    input.SessionID,
			"summary_type":  input.SummaryType,
			"target_length": input.TargetLength,
		})

		summary, err := summarySvc.GenerateSummary(ctx, req)
		if err != nil {
			logger.ErrorContext(ctx, "生成摘要失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryGenerateOutput{}, fmt.Errorf("生成摘要失败: %w", err)
		}

		// 5. 构建输出结果
		output := SummaryGenerateOutput{
			SummaryID:       summary.ID.String(),
			Summary:         summary.Content,
			TokenCount:      summary.TokenCount,
			MessageCount:    summary.MessageCount,
			GenerationTime:  time.Since(startTime).Milliseconds(),
		}

		// 设置起始和结束消息ID
		if summary.StartMessageID != nil {
			output.StartMessageID = summary.StartMessageID.String()
		}
		if summary.EndMessageID != nil {
			output.EndMessageID = summary.EndMessageID.String()
		}

		// 设置质量评分和压缩率
		if summary.QualityScore != nil {
			output.QualityScore = *summary.QualityScore
		}
		if summary.CompressionRate != nil {
			output.CompressionRate = *summary.CompressionRate
		}

		// 设置关键主题
		if len(summary.KeyTopics) > 0 {
			output.KeyTopics = summary.KeyTopics
		}

		logger.InfoContext(ctx, "摘要生成成功", logger.Fields{
			"session_id":         input.SessionID,
			"summary_id":         output.SummaryID,
			"token_count":        output.TokenCount,
			"message_count":      output.MessageCount,
			"quality_score":      output.QualityScore,
			"compression_rate":   output.CompressionRate,
			"generation_time_ms": output.GenerationTime,
		})

		return output, nil
	}
}

// summaryTriggerCheckFlow 创建摘要触发检查Flow
func summaryTriggerCheckFlow(summarySvc session.SummaryService) func(context.Context, SummaryTriggerCheckInput) (SummaryTriggerCheckOutput, error) {
	return func(ctx context.Context, input SummaryTriggerCheckInput) (SummaryTriggerCheckOutput, error) {
		startTime := time.Now()

		// 1. 参数验证
		if err := validateSummaryTriggerCheckInput(input); err != nil {
			logger.ErrorContext(ctx, "摘要触发检查参数验证失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryTriggerCheckOutput{}, fmt.Errorf("参数验证失败: %w", err)
		}

		// 2. 获取JWT声明进行权限验证
		claims, ok := auth.GetJWTClaimsFromContext(ctx)
		if !ok || claims == nil {
			logger.WarnContext(ctx, "未认证的摘要触发检查请求", logger.Fields{
				"session_id": input.SessionID,
			})
			return SummaryTriggerCheckOutput{}, fmt.Errorf("未认证")
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			logger.ErrorContext(ctx, "无效的租户ID", logger.Fields{
				"error":     err.Error(),
				"tenant_id": claims.TenantID,
			})
			return SummaryTriggerCheckOutput{}, fmt.Errorf("无效的租户ID: %w", err)
		}

		sessionID, err := uuid.Parse(input.SessionID)
		if err != nil {
			logger.ErrorContext(ctx, "无效的会话ID", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryTriggerCheckOutput{}, fmt.Errorf("无效的会话ID: %w", err)
		}

		// 3. 调用服务层检查触发条件
		logger.InfoContext(ctx, "开始检查摘要触发条件", logger.Fields{
			"session_id": input.SessionID,
			"check_mode": input.CheckMode,
		})

		result, err := summarySvc.CheckSummaryTrigger(ctx, tenantID, sessionID)
		if err != nil {
			logger.ErrorContext(ctx, "检查摘要触发条件失败", logger.Fields{
				"error":      err.Error(),
				"session_id": input.SessionID,
			})
			return SummaryTriggerCheckOutput{}, fmt.Errorf("检查摘要触发条件失败: %w", err)
		}

		// 4. 如果是强制模式，覆盖结果
		if input.CheckMode == "force" {
			result.ShouldSummarize = true
			result.TriggerReason = "强制触发"
			result.Urgency = 1.0
		}

		// 5. 构建输出结果
		output := SummaryTriggerCheckOutput{
			ShouldSummarize:      result.ShouldSummarize,
			TriggerReason:        result.TriggerReason,
			MessageCount:         result.MessageCount,
			EstimatedTokenSaving: result.EstimatedTokenSaving,
			Urgency:              result.Urgency,
			RecommendedType:      result.RecommendedType,
			CheckTime:            time.Since(startTime).Milliseconds(),
		}

		// 转换消息ID列表
		if len(result.MessageIDs) > 0 {
			output.MessageIDs = make([]string, len(result.MessageIDs))
			for i, id := range result.MessageIDs {
				output.MessageIDs[i] = id.String()
			}
		}

		logger.InfoContext(ctx, "摘要触发检查完成", logger.Fields{
			"session_id":       input.SessionID,
			"should_summarize": output.ShouldSummarize,
			"trigger_reason":   output.TriggerReason,
			"message_count":    output.MessageCount,
			"urgency":          output.Urgency,
			"recommended_type": output.RecommendedType,
			"check_time_ms":    output.CheckTime,
		})

		return output, nil
	}
}

// validateSummaryGenerateInput 验证摘要生成输入参数
func validateSummaryGenerateInput(input SummaryGenerateInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("无效的会话ID格式")
	}

	if input.SummaryType != "incremental" && input.SummaryType != "full" {
		return fmt.Errorf("无效的摘要类型，必须是 incremental 或 full")
	}

	if input.TargetLength < 50 || input.TargetLength > 1000 {
		return fmt.Errorf("目标长度必须在50-1000之间")
	}

	// 验证消息ID列表
	for _, idStr := range input.MessageIDs {
		if _, err := uuid.Parse(idStr); err != nil {
			return fmt.Errorf("无效的消息ID格式: %s", idStr)
		}
	}

	// 验证起始消息ID
	if input.StartMessageID != "" {
		if _, err := uuid.Parse(input.StartMessageID); err != nil {
			return fmt.Errorf("无效的起始消息ID格式")
		}
	}

	// 验证结束消息ID
	if input.EndMessageID != "" {
		if _, err := uuid.Parse(input.EndMessageID); err != nil {
			return fmt.Errorf("无效的结束消息ID格式")
		}
	}

	return nil
}

// validateSummaryTriggerCheckInput 验证摘要触发检查输入参数
func validateSummaryTriggerCheckInput(input SummaryTriggerCheckInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("无效的会话ID格式")
	}

	if input.CheckMode != "auto" && input.CheckMode != "force" {
		return fmt.Errorf("无效的检查模式，必须是 auto 或 force")
	}

	return nil
}
