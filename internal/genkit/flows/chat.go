// Package flows 实现对话相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
)

// ChatFlowServices 对话 Flow 所需的服务
type ChatFlowServices struct {
	ContextService service.ContextService
	MessageRepo    repository.MessageRepository
	SessionRepo    repository.SessionRepository
	VectorService  service.VectorService
	Logger         logger.Logger
}

// RegisterChatFlows 注册对话相关的 Flow
func RegisterChatFlows(g *genkit.Genkit, services *ChatFlowServices) {
	// 注册 chatGenerateFlow
	genkit.DefineFlow(
		g,
		"chatGenerateFlow",
		func(ctx context.Context, input ChatGenerateInput) (ChatGenerateOutput, error) {
			return executeChatGenerateFlow(ctx, g, input, services)
		},
	)

	// 注册 multiTurnChatFlow
	genkit.DefineFlow(
		g,
		"multiTurnChatFlow",
		func(ctx context.Context, input MultiTurnChatInput) (MultiTurnChatOutput, error) {
			return executeMultiTurnChatFlow(ctx, g, input, services)
		},
	)

	// 注册 chatRetryFlow
	genkit.DefineFlow(
		g,
		"chatRetryFlow",
		func(ctx context.Context, input ChatRetryInput) (ChatRetryOutput, error) {
			return executeChatRetryFlow(ctx, g, input, services)
		},
	)
}

// executeChatGenerateFlow 执行对话生成 Flow
func executeChatGenerateFlow(
	ctx context.Context,
	g *genkit.Genkit,
	input ChatGenerateInput,
	services *ChatFlowServices,
) (ChatGenerateOutput, error) {
	startTime := time.Now()

	// 1. 参数验证
	if err := validateChatGenerateInput(input); err != nil {
		return ChatGenerateOutput{}, fmt.Errorf("参数验证失败: %w", err)
	}

	// 2. 权限验证
	if err := validateSessionAccess(ctx, input.SessionID, services); err != nil {
		return ChatGenerateOutput{}, fmt.Errorf("权限验证失败: %w", err)
	}

	// 3. 构建上下文（如果未提供）
	var contextResult *ContextBuildOutput
	var err error

	if input.Context == nil {
		services.Logger.InfoContext(ctx, "未提供上下文，自动构建", logger.Fields{
			"sessionId": input.SessionID,
		})

		// 调用 ContextService 构建上下文
		contextReq := &service.BuildContextRequest{
			SessionID:       input.SessionID,
			UserQuery:       input.UserMessage,
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  true,
			IncludeLongTerm: true,
			ShortTermWindow: 10,
		}

		contextSvcResult, err := services.ContextService.BuildContext(ctx, contextReq)
		if err != nil {
			services.Logger.ErrorContext(ctx, "构建上下文失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
			return ChatGenerateOutput{}, fmt.Errorf("构建上下文失败: %w", err)
		}

		// 转换为 ContextBuildOutput
		contextResult = &ContextBuildOutput{
			SessionID:         contextSvcResult.SessionID,
			ShortTermMessages: []MessageContext{}, // TODO: 转换消息
			TotalTokens:       contextSvcResult.TotalTokens,
			Strategy:          contextSvcResult.Strategy,
			QualityScore:      contextSvcResult.QualityScore,
			BuildTime:         0,
		}
	} else {
		contextResult = input.Context
	}

	// 4. 构建提示词
	prompt := buildPrompt(input, contextResult)

	services.Logger.InfoContext(ctx, "开始生成 AI 响应", logger.Fields{
		"sessionId":     input.SessionID,
		"contextTokens": contextResult.TotalTokens,
		"promptLength":  len(prompt),
	})

	// 5. 调用 Genkit Generate API
	var response *ai.ModelResponse
	retryCount := 0
	maxRetries := 3

	for retryCount < maxRetries {
		response, err = genkit.Generate(ctx, g, ai.WithPrompt(prompt))
		if err == nil {
			break
		}

		retryCount++
		if retryCount < maxRetries {
			services.Logger.WarnContext(ctx, "AI 生成失败，准备重试", logger.Fields{
				"sessionId":  input.SessionID,
				"retryCount": retryCount,
				"error":      err.Error(),
			})
			time.Sleep(time.Duration(retryCount) * time.Second)
		}
	}

	if err != nil {
		services.Logger.ErrorContext(ctx, "AI 生成失败，已达最大重试次数", logger.Fields{
			"sessionId":  input.SessionID,
			"retryCount": retryCount,
			"error":      err.Error(),
		})
		return ChatGenerateOutput{}, fmt.Errorf("AI 生成失败: %w", err)
	}

	// 6. 提取响应内容
	responseText := response.Text()
	finishReason := "stop"
	if response.FinishReason != "" {
		finishReason = string(response.FinishReason)
	}

	// 7. 记录 Token 使用情况
	tokenUsage := TokenUsage{
		PromptTokens:     int(response.Usage.InputTokens),
		CompletionTokens: int(response.Usage.OutputTokens),
		TotalTokens:      int(response.Usage.TotalTokens),
	}

	services.Logger.InfoContext(ctx, "AI 响应生成成功", logger.Fields{
		"sessionId":        input.SessionID,
		"responseLength":   len(responseText),
		"promptTokens":     tokenUsage.PromptTokens,
		"completionTokens": tokenUsage.CompletionTokens,
		"totalTokens":      tokenUsage.TotalTokens,
	})

	// 8. 保存消息（如果需要）
	messageID := uuid.New().String()
	if input.SaveMessage {
		go func() {
			saveCtx := context.Background()
			if err := saveMessages(saveCtx, input, responseText, messageID, services); err != nil {
				services.Logger.ErrorContext(saveCtx, "保存消息失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 9. 异步生成向量
	if input.SaveMessage {
		go func() {
			vectorCtx := context.Background()
			if err := generateVectorsAsync(vectorCtx, input, responseText, messageID, services); err != nil {
				services.Logger.ErrorContext(vectorCtx, "生成向量失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 10. 构建输出
	generationTime := time.Since(startTime).Milliseconds()

	output := ChatGenerateOutput{
		MessageID:      messageID,
		Response:       responseText,
		TokenUsage:     tokenUsage,
		FinishReason:   finishReason,
		Model:          getModelName(input.ModelConfig),
		GenerationTime: generationTime,
		ContextInfo: ContextInfo{
			ContextTokens: contextResult.TotalTokens,
			Strategy:      contextResult.Strategy,
			QualityScore:  contextResult.QualityScore,
		},
	}

	return output, nil
}

// validateChatGenerateInput 验证输入参数
func validateChatGenerateInput(input ChatGenerateInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("sessionId 不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("sessionId 格式无效: %w", err)
	}

	if input.UserMessage == "" {
		return fmt.Errorf("userMessage 不能为空")
	}

	if len(input.UserMessage) > 4000 {
		return fmt.Errorf("userMessage 长度不能超过 4000 字符")
	}

	if input.SystemPrompt != "" && len(input.SystemPrompt) > 1000 {
		return fmt.Errorf("systemPrompt 长度不能超过 1000 字符")
	}

	return nil
}

// validateSessionAccess 验证会话访问权限
func validateSessionAccess(ctx context.Context, sessionID string, services *ChatFlowServices) error {
	// 获取 JWT 声明
	claims, ok := middleware.GetJWTClaims(ctx)
	if !ok || claims == nil {
		return fmt.Errorf("未认证")
	}

	// 查询会话
	session, err := services.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}

	// 平台管理员可以访问所有会话
	if hasRole(claims, model.RoleSystemAdmin) {
		return nil
	}

	// 验证用户ID匹配（暂时使用用户ID验证，后续可以改为租户ID）
	if claims.Subject != session.UserID.String() {
		services.Logger.WarnContext(ctx, "权限验证失败：尝试访问其他用户的会话", logger.Fields{
			"userId":        claims.Subject,
			"sessionId":     sessionID,
			"sessionUserId": session.UserID.String(),
		})
		return fmt.Errorf("权限不足：无法访问其他用户的会话")
	}

	return nil
}

// buildPrompt 构建提示词
func buildPrompt(input ChatGenerateInput, context *ContextBuildOutput) string {
	var builder strings.Builder

	// 1. 系统提示词
	if input.SystemPrompt != "" {
		builder.WriteString("系统指令：\n")
		builder.WriteString(input.SystemPrompt)
		builder.WriteString("\n\n")
	}

	// 2. 摘要上下文
	if context.Summary != nil {
		builder.WriteString("对话摘要：\n")
		builder.WriteString(context.Summary.Content)
		builder.WriteString("\n\n")
	}

	// 3. 长期记忆
	if len(context.LongTermMemories) > 0 {
		builder.WriteString("相关历史记忆：\n")
		for i, memory := range context.LongTermMemories {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, memory.Content))
		}
		builder.WriteString("\n")
	}

	// 4. 短期消息历史
	if len(context.ShortTermMessages) > 0 {
		builder.WriteString("最近对话：\n")
		for _, msg := range context.ShortTermMessages {
			roleLabel := "用户"
			if msg.Role == "assistant" {
				roleLabel = "助手"
			}
			builder.WriteString(fmt.Sprintf("%s: %s\n", roleLabel, msg.Content))
		}
		builder.WriteString("\n")
	}

	// 5. 当前用户消息
	builder.WriteString("用户: ")
	builder.WriteString(input.UserMessage)

	return builder.String()
}

// saveMessages 保存用户消息和 AI 响应
func saveMessages(
	ctx context.Context,
	input ChatGenerateInput,
	response string,
	messageID string,
	services *ChatFlowServices,
) error {
	sessionUUID, err := uuid.Parse(input.SessionID)
	if err != nil {
		return fmt.Errorf("解析 sessionID 失败: %w", err)
	}

	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		return fmt.Errorf("解析 messageID 失败: %w", err)
	}

	// 获取会话信息
	session, err := services.SessionRepo.GetByID(ctx, input.SessionID)
	if err != nil {
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	// 获取当前消息序列号
	currentSequence := session.MessageCount

	// 保存用户消息
	userMessage := &model.ChatMessage{
		ID:        uuid.New(),
		SessionID: sessionUUID,
		Role:      "user",
		Content:   input.UserMessage,
		Sequence:  currentSequence + 1,
		CreatedAt: time.Now(),
	}

	if err := services.MessageRepo.Create(ctx, userMessage); err != nil {
		return fmt.Errorf("保存用户消息失败: %w", err)
	}

	// 保存 AI 响应
	assistantMessage := &model.ChatMessage{
		ID:        messageUUID,
		SessionID: sessionUUID,
		Role:      "assistant",
		Content:   response,
		Sequence:  currentSequence + 2,
		CreatedAt: time.Now(),
	}

	if err := services.MessageRepo.Create(ctx, assistantMessage); err != nil {
		return fmt.Errorf("保存 AI 响应失败: %w", err)
	}

	services.Logger.InfoContext(ctx, "消息保存成功", logger.Fields{
		"sessionId":       input.SessionID,
		"userMessageId":   userMessage.ID.String(),
		"assistMessageId": messageID,
	})

	return nil
}

// generateVectorsAsync 异步生成消息向量
func generateVectorsAsync(
	ctx context.Context,
	input ChatGenerateInput,
	response string,
	messageID string,
	services *ChatFlowServices,
) error {
	// 为用户消息生成向量
	userEmbedding, err := services.VectorService.GenerateEmbedding(ctx, input.UserMessage)
	if err != nil {
		services.Logger.WarnContext(ctx, "生成用户消息向量失败", logger.Fields{
			"sessionId": input.SessionID,
			"error":     err.Error(),
		})
	}

	// 为 AI 响应生成向量
	assistantEmbedding, err := services.VectorService.GenerateEmbedding(ctx, response)
	if err != nil {
		services.Logger.WarnContext(ctx, "生成 AI 响应向量失败", logger.Fields{
			"sessionId": input.SessionID,
			"messageId": messageID,
			"error":     err.Error(),
		})
	}

	services.Logger.InfoContext(ctx, "向量生成完成", logger.Fields{
		"sessionId":           input.SessionID,
		"userEmbeddingSize":   len(userEmbedding),
		"assistEmbeddingSize": len(assistantEmbedding),
	})

	// TODO: 将向量保存到 conversation_memories 表
	// 这部分将在 memoryStoreFlow 实现时完成

	return nil
}

// getModelName 获取模型名称
func getModelName(config *ModelConfig) string {
	if config != nil && config.ModelName != "" {
		return config.ModelName
	}
	return "gemini-1.5-flash" // 默认模型
}

// executeMultiTurnChatFlow 执行多轮对话管理 Flow
func executeMultiTurnChatFlow(
	ctx context.Context,
	g *genkit.Genkit,
	input MultiTurnChatInput,
	services *ChatFlowServices,
) (MultiTurnChatOutput, error) {
	startTime := time.Now()

	// 1. 参数验证
	if err := validateMultiTurnChatInput(input); err != nil {
		return MultiTurnChatOutput{}, fmt.Errorf("参数验证失败: %w", err)
	}

	// 2. 权限验证
	if err := validateSessionAccess(ctx, input.SessionID, services); err != nil {
		return MultiTurnChatOutput{}, fmt.Errorf("权限验证失败: %w", err)
	}

	// 3. 获取会话信息
	session, err := services.SessionRepo.GetByID(ctx, input.SessionID)
	if err != nil {
		return MultiTurnChatOutput{}, fmt.Errorf("获取会话信息失败: %w", err)
	}

	services.Logger.InfoContext(ctx, "开始多轮对话管理", logger.Fields{
		"sessionId":    input.SessionID,
		"messageCount": session.MessageCount,
		"resetContext": input.ResetContext,
	})

	// 4. 处理上下文重置（如果需要）
	if input.ResetContext {
		if err := resetSessionContext(ctx, input.SessionID, services); err != nil {
			services.Logger.WarnContext(ctx, "重置上下文失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
		} else {
			services.Logger.InfoContext(ctx, "上下文重置成功", logger.Fields{
				"sessionId": input.SessionID,
			})
		}
	}

	// 5. 检查会话状态
	sessionState, healthScore := evaluateSessionHealth(ctx, session, services)

	services.Logger.InfoContext(ctx, "会话健康度评估完成", logger.Fields{
		"sessionId":    input.SessionID,
		"sessionState": sessionState,
		"healthScore":  healthScore,
	})

	// 6. 生成建议
	suggestions := generateSuggestions(ctx, session, sessionState, healthScore, services)

	// 7. 计算 Token 使用率
	tokenUsageRate := calculateTokenUsageRate(ctx, session, services)

	// 8. 调用 chatGenerateFlow 生成响应
	chatInput := ChatGenerateInput{
		SessionID:   input.SessionID,
		UserMessage: input.UserMessage,
		SaveMessage: true,
	}

	chatOutput, err := executeChatGenerateFlow(ctx, g, chatInput, services)
	if err != nil {
		return MultiTurnChatOutput{}, fmt.Errorf("生成对话响应失败: %w", err)
	}

	// 9. 获取上下文信息
	contextInfo := buildMultiTurnContextInfo(ctx, session, services)

	// 10. 构建输出
	output := MultiTurnChatOutput{
		SessionID:      input.SessionID,
		TurnNumber:     session.MessageCount + 2, // 用户消息 + AI 响应
		SessionState:   sessionState,
		HealthScore:    healthScore,
		TokenUsageRate: tokenUsageRate,
		Suggestions:    suggestions,
		ContextInfo:    contextInfo,
		Response:       chatOutput.Response,
		MessageID:      chatOutput.MessageID,
	}

	executionTime := time.Since(startTime).Milliseconds()
	services.Logger.InfoContext(ctx, "多轮对话管理完成", logger.Fields{
		"sessionId":     input.SessionID,
		"turnNumber":    output.TurnNumber,
		"sessionState":  output.SessionState,
		"healthScore":   output.HealthScore,
		"executionTime": executionTime,
	})

	return output, nil
}

// validateMultiTurnChatInput 验证多轮对话输入参数
func validateMultiTurnChatInput(input MultiTurnChatInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("sessionId 不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("sessionId 格式无效: %w", err)
	}

	if input.UserMessage == "" {
		return fmt.Errorf("userMessage 不能为空")
	}

	if len(input.UserMessage) > 4000 {
		return fmt.Errorf("userMessage 长度不能超过 4000 字符")
	}

	return nil
}

// resetSessionContext 重置会话上下文
func resetSessionContext(ctx context.Context, sessionID string, services *ChatFlowServices) error {
	// TODO: 实现上下文重置逻辑
	// 1. 清理短期记忆（保留最近几条消息）
	// 2. 保留摘要
	// 3. 重置 Token 计数器

	services.Logger.InfoContext(ctx, "执行上下文重置", logger.Fields{
		"sessionId": sessionID,
	})

	return nil
}

// evaluateSessionHealth 评估会话健康度
func evaluateSessionHealth(
	ctx context.Context,
	session *model.ChatSession,
	services *ChatFlowServices,
) (string, float64) {
	// 初始化健康评分
	healthScore := 1.0
	sessionState := "healthy"

	// 1. 检查消息数量
	messageCount := session.MessageCount
	if messageCount > 20 {
		healthScore -= 0.2
		sessionState = "needs_summary"
		services.Logger.InfoContext(ctx, "会话消息数量较多，建议生成摘要", logger.Fields{
			"sessionId":    session.ID.String(),
			"messageCount": messageCount,
		})
	}

	// 2. 检查 Token 使用情况
	// TODO: 从 conversation_contexts 表获取实际的 Token 使用情况
	// 这里使用估算值
	estimatedTokens := messageCount * 100 // 假设每条消息平均 100 tokens
	maxTokens := 4000
	tokenUsageRate := float64(estimatedTokens) / float64(maxTokens)

	if tokenUsageRate > 0.8 {
		healthScore -= 0.3
		sessionState = "token_warning"
		services.Logger.WarnContext(ctx, "Token 使用率过高", logger.Fields{
			"sessionId":      session.ID.String(),
			"tokenUsageRate": tokenUsageRate,
		})
	}

	// 3. 检查上下文质量
	// TODO: 从 conversation_contexts 表获取实际的质量评分
	// 这里使用默认值
	contextQuality := 0.8
	if contextQuality < 0.6 {
		healthScore -= 0.2
		sessionState = "needs_cleanup"
		services.Logger.WarnContext(ctx, "上下文质量较低", logger.Fields{
			"sessionId":      session.ID.String(),
			"contextQuality": contextQuality,
		})
	}

	// 4. 检查会话活跃度
	if session.MessageCount > 0 {
		sessionState = "active"
	}

	// 确保健康评分在 0-1 范围内
	if healthScore < 0 {
		healthScore = 0
	}

	return sessionState, healthScore
}

// generateSuggestions 生成建议操作列表
func generateSuggestions(
	ctx context.Context,
	session *model.ChatSession,
	sessionState string,
	healthScore float64,
	services *ChatFlowServices,
) []string {
	suggestions := make([]string, 0)

	// 根据会话状态生成建议
	switch sessionState {
	case "needs_summary":
		suggestions = append(suggestions, "建议生成对话摘要以优化上下文")
		suggestions = append(suggestions, "对话轮次已超过 20 次，摘要可以减少 Token 消耗")

	case "token_warning":
		suggestions = append(suggestions, "Token 使用率超过 80%，建议优化上下文")
		suggestions = append(suggestions, "可以考虑生成摘要或清理低质量记忆")

	case "needs_cleanup":
		suggestions = append(suggestions, "上下文质量较低，建议重置上下文")
		suggestions = append(suggestions, "可以保留摘要，清理短期消息")

	case "healthy":
		if session.MessageCount > 10 {
			suggestions = append(suggestions, "会话运行良好，可以继续对话")
		}
	}

	// 根据健康评分生成额外建议
	if healthScore < 0.5 {
		suggestions = append(suggestions, "会话健康度较低，建议采取优化措施")
	}

	// 如果没有建议，添加默认建议
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "会话状态良好，无需特殊操作")
	}

	return suggestions
}

// calculateTokenUsageRate 计算 Token 使用率
func calculateTokenUsageRate(
	ctx context.Context,
	session *model.ChatSession,
	services *ChatFlowServices,
) float64 {
	// TODO: 从 conversation_contexts 表获取实际的 Token 使用情况
	// 这里使用估算值
	estimatedTokens := session.MessageCount * 100 // 假设每条消息平均 100 tokens
	maxTokens := 4000

	usageRate := float64(estimatedTokens) / float64(maxTokens)

	// 确保使用率在 0-1 范围内
	if usageRate > 1.0 {
		usageRate = 1.0
	}

	return usageRate
}

// buildMultiTurnContextInfo 构建多轮对话上下文信息
func buildMultiTurnContextInfo(
	ctx context.Context,
	session *model.ChatSession,
	services *ChatFlowServices,
) MultiTurnContextInfo {
	// TODO: 从 conversation_contexts 表获取实际的上下文信息
	// 这里使用估算值

	estimatedTokens := session.MessageCount * 100
	maxTokens := 4000

	contextInfo := MultiTurnContextInfo{
		TotalMessages:            session.MessageCount,
		TotalTokens:              estimatedTokens,
		MaxTokens:                maxTokens,
		QualityScore:             0.8, // 默认质量评分
		MessagesSinceLastSummary: session.MessageCount,
	}

	// TODO: 获取最后一次摘要时间
	// if lastSummaryAt != nil {
	//     contextInfo.LastSummaryAt = lastSummaryAt.Format(time.RFC3339)
	// }

	return contextInfo
}

// executeChatRetryFlow 执行对话重试 Flow
func executeChatRetryFlow(
	ctx context.Context,
	g *genkit.Genkit,
	input ChatRetryInput,
	services *ChatFlowServices,
) (ChatRetryOutput, error) {
	startTime := time.Now()

	// 1. 参数验证
	if err := validateChatRetryInput(input); err != nil {
		return ChatRetryOutput{}, fmt.Errorf("参数验证失败: %w", err)
	}

	// 2. 权限验证
	if err := validateSessionAccess(ctx, input.SessionID, services); err != nil {
		return ChatRetryOutput{}, fmt.Errorf("权限验证失败: %w", err)
	}

	services.Logger.InfoContext(ctx, "开始对话重试流程", logger.Fields{
		"sessionId":     input.SessionID,
		"retryStrategy": input.RetryStrategy,
		"maxRetries":    input.MaxRetries,
	})

	// 3. 构建上下文（如果未提供）
	var contextResult *ContextBuildOutput
	var err error

	if input.Context == nil {
		services.Logger.InfoContext(ctx, "未提供上下文，自动构建", logger.Fields{
			"sessionId": input.SessionID,
		})

		contextReq := &service.BuildContextRequest{
			SessionID:       input.SessionID,
			UserQuery:       input.UserMessage,
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  true,
			IncludeLongTerm: true,
			ShortTermWindow: 10,
		}

		contextSvcResult, err := services.ContextService.BuildContext(ctx, contextReq)
		if err != nil {
			services.Logger.ErrorContext(ctx, "构建上下文失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
			return ChatRetryOutput{}, fmt.Errorf("构建上下文失败: %w", err)
		}

		contextResult = &ContextBuildOutput{
			SessionID:         contextSvcResult.SessionID,
			ShortTermMessages: []MessageContext{},
			TotalTokens:       contextSvcResult.TotalTokens,
			Strategy:          contextSvcResult.Strategy,
			QualityScore:      contextSvcResult.QualityScore,
			BuildTime:         0,
		}
	} else {
		contextResult = input.Context
	}

	// 4. 初始化重试信息
	retryInfo := RetryInfo{
		Strategy:       input.RetryStrategy,
		TotalAttempts:  0,
		SuccessAttempt: 0,
		FailedAttempts: make([]RetryAttempt, 0),
		TotalRetryTime: 0,
	}

	// 5. 构建提示词
	prompt := buildPrompt(ChatGenerateInput{
		SessionID:    input.SessionID,
		UserMessage:  input.UserMessage,
		SystemPrompt: input.SystemPrompt,
	}, contextResult)

	// 6. 根据策略执行重试
	var response *ai.ModelResponse
	var fallbackUsed bool
	var fallbackReason string

	switch input.RetryStrategy {
	case "simple":
		response, err = executeSimpleRetry(ctx, g, prompt, input.MaxRetries, &retryInfo, services)
	case "exponential":
		response, err = executeExponentialRetry(ctx, g, prompt, input.MaxRetries, &retryInfo, services)
	case "adaptive":
		response, err = executeAdaptiveRetry(ctx, g, prompt, input, contextResult, &retryInfo, services)
	default:
		return ChatRetryOutput{}, fmt.Errorf("不支持的重试策略: %s", input.RetryStrategy)
	}

	// 7. 如果所有重试失败，执行回退操作
	if err != nil {
		services.Logger.WarnContext(ctx, "所有重试失败，执行回退操作", logger.Fields{
			"sessionId":     input.SessionID,
			"totalAttempts": retryInfo.TotalAttempts,
			"error":         err.Error(),
		})

		response, fallbackReason, err = executeFallback(ctx, g, input, contextResult, services)
		if err != nil {
			services.Logger.ErrorContext(ctx, "回退操作失败", logger.Fields{
				"sessionId": input.SessionID,
				"error":     err.Error(),
			})
			return ChatRetryOutput{}, fmt.Errorf("对话生成失败且回退操作失败: %w", err)
		}
		fallbackUsed = true
	}

	// 8. 提取响应内容
	responseText := response.Text()
	finishReason := "stop"
	if response.FinishReason != "" {
		finishReason = string(response.FinishReason)
	}

	// 9. 记录 Token 使用情况
	tokenUsage := TokenUsage{
		PromptTokens:     int(response.Usage.InputTokens),
		CompletionTokens: int(response.Usage.OutputTokens),
		TotalTokens:      int(response.Usage.TotalTokens),
	}

	services.Logger.InfoContext(ctx, "对话重试流程完成", logger.Fields{
		"sessionId":        input.SessionID,
		"totalAttempts":    retryInfo.TotalAttempts,
		"successAttempt":   retryInfo.SuccessAttempt,
		"fallbackUsed":     fallbackUsed,
		"responseLength":   len(responseText),
		"promptTokens":     tokenUsage.PromptTokens,
		"completionTokens": tokenUsage.CompletionTokens,
	})

	// 10. 保存消息（如果需要）
	messageID := uuid.New().String()
	if input.SaveMessage {
		go func() {
			saveCtx := context.Background()
			if err := saveMessages(saveCtx, ChatGenerateInput{
				SessionID:   input.SessionID,
				UserMessage: input.UserMessage,
			}, responseText, messageID, services); err != nil {
				services.Logger.ErrorContext(saveCtx, "保存消息失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 11. 异步生成向量
	if input.SaveMessage {
		go func() {
			vectorCtx := context.Background()
			if err := generateVectorsAsync(vectorCtx, ChatGenerateInput{
				SessionID:   input.SessionID,
				UserMessage: input.UserMessage,
			}, responseText, messageID, services); err != nil {
				services.Logger.ErrorContext(vectorCtx, "生成向量失败", logger.Fields{
					"sessionId": input.SessionID,
					"messageId": messageID,
					"error":     err.Error(),
				})
			}
		}()
	}

	// 12. 计算总执行时间
	retryInfo.TotalRetryTime = time.Since(startTime).Milliseconds()

	// 13. 构建输出
	output := ChatRetryOutput{
		MessageID:      messageID,
		Response:       responseText,
		TokenUsage:     tokenUsage,
		FinishReason:   finishReason,
		Model:          getModelName(input.ModelConfig),
		GenerationTime: retryInfo.TotalRetryTime,
		ContextInfo: ContextInfo{
			ContextTokens: contextResult.TotalTokens,
			Strategy:      contextResult.Strategy,
			QualityScore:  contextResult.QualityScore,
		},
		RetryInfo:      retryInfo,
		FallbackUsed:   fallbackUsed,
		FallbackReason: fallbackReason,
	}

	return output, nil
}

// validateChatRetryInput 验证重试输入参数
func validateChatRetryInput(input ChatRetryInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("sessionId 不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("sessionId 格式无效: %w", err)
	}

	if input.UserMessage == "" {
		return fmt.Errorf("userMessage 不能为空")
	}

	if len(input.UserMessage) > 4000 {
		return fmt.Errorf("userMessage 长度不能超过 4000 字符")
	}

	if input.SystemPrompt != "" && len(input.SystemPrompt) > 1000 {
		return fmt.Errorf("systemPrompt 长度不能超过 1000 字符")
	}

	if input.RetryStrategy == "" {
		return fmt.Errorf("retryStrategy 不能为空")
	}

	validStrategies := map[string]bool{
		"simple":      true,
		"exponential": true,
		"adaptive":    true,
	}

	if !validStrategies[input.RetryStrategy] {
		return fmt.Errorf("不支持的重试策略: %s", input.RetryStrategy)
	}

	if input.MaxRetries < 1 || input.MaxRetries > 10 {
		return fmt.Errorf("maxRetries 必须在 1-10 之间")
	}

	return nil
}

// executeSimpleRetry 执行简单重试策略（固定间隔）
func executeSimpleRetry(
	ctx context.Context,
	g *genkit.Genkit,
	prompt string,
	maxRetries int,
	retryInfo *RetryInfo,
	services *ChatFlowServices,
) (*ai.ModelResponse, error) {
	var response *ai.ModelResponse
	var err error

	// 固定重试间隔：1秒
	retryInterval := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		retryInfo.TotalAttempts = attempt

		services.Logger.InfoContext(ctx, "执行简单重试", logger.Fields{
			"attempt":       attempt,
			"maxRetries":    maxRetries,
			"retryInterval": retryInterval.Seconds(),
		})

		response, err = genkit.Generate(ctx, g, ai.WithPrompt(prompt))
		if err == nil {
			retryInfo.SuccessAttempt = attempt
			services.Logger.InfoContext(ctx, "重试成功", logger.Fields{
				"attempt": attempt,
			})
			return response, nil
		}

		// 记录失败尝试
		retryInfo.FailedAttempts = append(retryInfo.FailedAttempts, RetryAttempt{
			AttemptNumber: attempt,
			Error:         err.Error(),
			WaitTime:      retryInterval.Milliseconds(),
			Timestamp:     time.Now().Format(time.RFC3339),
		})

		services.Logger.WarnContext(ctx, "重试失败", logger.Fields{
			"attempt": attempt,
			"error":   err.Error(),
		})

		// 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("简单重试失败，已尝试 %d 次: %w", maxRetries, err)
}

// executeExponentialRetry 执行指数退避重试策略
func executeExponentialRetry(
	ctx context.Context,
	g *genkit.Genkit,
	prompt string,
	maxRetries int,
	retryInfo *RetryInfo,
	services *ChatFlowServices,
) (*ai.ModelResponse, error) {
	var response *ai.ModelResponse
	var err error

	// 基础重试间隔：1秒
	baseInterval := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		retryInfo.TotalAttempts = attempt

		// 计算指数退避间隔：1s, 2s, 4s, 8s, 16s
		retryInterval := baseInterval * time.Duration(1<<uint(attempt-1))
		// 限制最大间隔为 30 秒
		if retryInterval > 30*time.Second {
			retryInterval = 30 * time.Second
		}

		services.Logger.InfoContext(ctx, "执行指数退避重试", logger.Fields{
			"attempt":       attempt,
			"maxRetries":    maxRetries,
			"retryInterval": retryInterval.Seconds(),
		})

		response, err = genkit.Generate(ctx, g, ai.WithPrompt(prompt))
		if err == nil {
			retryInfo.SuccessAttempt = attempt
			services.Logger.InfoContext(ctx, "重试成功", logger.Fields{
				"attempt": attempt,
			})
			return response, nil
		}

		// 记录失败尝试
		retryInfo.FailedAttempts = append(retryInfo.FailedAttempts, RetryAttempt{
			AttemptNumber: attempt,
			Error:         err.Error(),
			WaitTime:      retryInterval.Milliseconds(),
			Timestamp:     time.Now().Format(time.RFC3339),
		})

		services.Logger.WarnContext(ctx, "重试失败", logger.Fields{
			"attempt": attempt,
			"error":   err.Error(),
		})

		// 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("指数退避重试失败，已尝试 %d 次: %w", maxRetries, err)
}

// executeAdaptiveRetry 执行自适应重试策略（根据失败原因调整）
func executeAdaptiveRetry(
	ctx context.Context,
	g *genkit.Genkit,
	prompt string,
	input ChatRetryInput,
	contextResult *ContextBuildOutput,
	retryInfo *RetryInfo,
	services *ChatFlowServices,
) (*ai.ModelResponse, error) {
	var response *ai.ModelResponse
	var err error

	// 基础重试间隔：1秒
	baseInterval := 1 * time.Second

	for attempt := 1; attempt <= input.MaxRetries; attempt++ {
		retryInfo.TotalAttempts = attempt

		services.Logger.InfoContext(ctx, "执行自适应重试", logger.Fields{
			"attempt":    attempt,
			"maxRetries": input.MaxRetries,
		})

		response, err = genkit.Generate(ctx, g, ai.WithPrompt(prompt))
		if err == nil {
			retryInfo.SuccessAttempt = attempt
			services.Logger.InfoContext(ctx, "重试成功", logger.Fields{
				"attempt": attempt,
			})
			return response, nil
		}

		// 分析失败原因并调整策略
		errorType := analyzeError(err)
		retryInterval := baseInterval

		services.Logger.WarnContext(ctx, "重试失败，分析错误类型", logger.Fields{
			"attempt":   attempt,
			"error":     err.Error(),
			"errorType": errorType,
		})

		// 根据错误类型调整重试策略
		switch errorType {
		case "rate_limit":
			// 速率限制：使用指数退避
			retryInterval = baseInterval * time.Duration(1<<uint(attempt-1))
			if retryInterval > 30*time.Second {
				retryInterval = 30 * time.Second
			}
			services.Logger.InfoContext(ctx, "检测到速率限制，使用指数退避", logger.Fields{
				"retryInterval": retryInterval.Seconds(),
			})

		case "timeout":
			// 超时：减少上下文大小
			if attempt > 1 && contextResult.TotalTokens > 1000 {
				services.Logger.InfoContext(ctx, "检测到超时，尝试减少上下文", logger.Fields{
					"originalTokens": contextResult.TotalTokens,
				})
				// 减少上下文（保留 70%）
				targetTokens := int(float64(contextResult.TotalTokens) * 0.7)
				prompt = optimizePromptForRetry(prompt, targetTokens)
			}
			retryInterval = baseInterval * 2

		case "context_length":
			// 上下文长度超限：大幅减少上下文
			services.Logger.InfoContext(ctx, "检测到上下文长度超限，大幅减少上下文", logger.Fields{
				"originalTokens": contextResult.TotalTokens,
			})
			// 减少上下文（保留 50%）
			targetTokens := int(float64(contextResult.TotalTokens) * 0.5)
			prompt = optimizePromptForRetry(prompt, targetTokens)
			retryInterval = baseInterval

		case "server_error":
			// 服务器错误：使用指数退避
			retryInterval = baseInterval * time.Duration(1<<uint(attempt-1))
			if retryInterval > 30*time.Second {
				retryInterval = 30 * time.Second
			}

		default:
			// 未知错误：使用固定间隔
			retryInterval = baseInterval * 2
		}

		// 记录失败尝试
		retryInfo.FailedAttempts = append(retryInfo.FailedAttempts, RetryAttempt{
			AttemptNumber: attempt,
			Error:         fmt.Sprintf("%s: %s", errorType, err.Error()),
			WaitTime:      retryInterval.Milliseconds(),
			Timestamp:     time.Now().Format(time.RFC3339),
		})

		// 如果不是最后一次尝试，等待后重试
		if attempt < input.MaxRetries {
			time.Sleep(retryInterval)
		}
	}

	return nil, fmt.Errorf("自适应重试失败，已尝试 %d 次: %w", input.MaxRetries, err)
}

// analyzeError 分析错误类型
func analyzeError(err error) string {
	if err == nil {
		return "unknown"
	}

	errorMsg := strings.ToLower(err.Error())

	// 检查常见错误类型
	if strings.Contains(errorMsg, "rate limit") || strings.Contains(errorMsg, "too many requests") {
		return "rate_limit"
	}

	if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		return "timeout"
	}

	if strings.Contains(errorMsg, "context length") || strings.Contains(errorMsg, "token limit") {
		return "context_length"
	}

	if strings.Contains(errorMsg, "server error") || strings.Contains(errorMsg, "internal error") {
		return "server_error"
	}

	if strings.Contains(errorMsg, "invalid") || strings.Contains(errorMsg, "bad request") {
		return "invalid_request"
	}

	return "unknown"
}

// optimizePromptForRetry 优化提示词以减少 Token 数量
func optimizePromptForRetry(prompt string, targetTokens int) string {
	// 简单的优化策略：截断提示词
	// 估算：1 token ≈ 4 字符（中文）或 4 字符（英文）
	targetChars := targetTokens * 4

	if len(prompt) <= targetChars {
		return prompt
	}

	// 保留前面的部分（包含系统提示词和重要上下文）
	// 保留后面的部分（包含用户消息）
	// 中间部分可以截断

	// 简单实现：保留前 60% 和后 40%
	frontChars := int(float64(targetChars) * 0.6)
	backChars := targetChars - frontChars

	if len(prompt) <= frontChars+backChars {
		return prompt
	}

	front := prompt[:frontChars]
	back := prompt[len(prompt)-backChars:]

	return front + "\n...[部分内容已省略]...\n" + back
}

// executeFallback 执行回退操作
func executeFallback(
	ctx context.Context,
	g *genkit.Genkit,
	input ChatRetryInput,
	contextResult *ContextBuildOutput,
	services *ChatFlowServices,
) (*ai.ModelResponse, string, error) {
	services.Logger.InfoContext(ctx, "开始执行回退操作", logger.Fields{
		"sessionId": input.SessionID,
	})

	// 回退策略 1: 减少上下文
	if contextResult.TotalTokens > 1000 {
		services.Logger.InfoContext(ctx, "回退策略 1: 减少上下文", logger.Fields{
			"originalTokens": contextResult.TotalTokens,
		})

		// 构建简化的提示词（只保留用户消息）
		simplePrompt := fmt.Sprintf("用户: %s", input.UserMessage)

		response, err := genkit.Generate(ctx, g, ai.WithPrompt(simplePrompt))
		if err == nil {
			return response, "减少上下文", nil
		}

		services.Logger.WarnContext(ctx, "回退策略 1 失败", logger.Fields{
			"error": err.Error(),
		})
	}

	// 回退策略 2: 使用备用模型（如果配置了）
	// TODO: 实现备用模型切换逻辑
	// if input.ModelConfig != nil && input.ModelConfig.FallbackModel != "" {
	//     services.Logger.InfoContext(ctx, "回退策略 2: 使用备用模型", logger.Fields{
	//         "fallbackModel": input.ModelConfig.FallbackModel,
	//     })
	//     // 尝试使用备用模型
	// }

	// 回退策略 3: 返回预设响应
	services.Logger.InfoContext(ctx, "回退策略 3: 返回预设响应", logger.Fields{
		"sessionId": input.SessionID,
	})

	// 创建一个模拟的响应
	fallbackResponse := &ai.ModelResponse{
		Message: &ai.Message{
			Content: []*ai.Part{
				ai.NewTextPart("抱歉，服务暂时不可用。我们正在努力解决问题，请稍后重试。"),
			},
		},
		Usage: &ai.GenerationUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
		FinishReason: ai.FinishReasonStop,
	}

	return fallbackResponse, "返回预设响应", nil
}
