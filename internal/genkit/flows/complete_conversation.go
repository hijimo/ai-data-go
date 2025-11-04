// Package flows 实现完整对话流程编排
package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/google/uuid"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// RegisterCompleteConversationFlow 注册完整对话流程Flow
func RegisterCompleteConversationFlow(
	g *genkit.Genkit,
	contextSvc service.ContextService,
	chatSvc service.ChatService,
	memorySvc service.MemoryService,
	summarySvc service.SummaryService,
	queryClassifySvc service.QueryClassifyService,
) {
	genkit.DefineFlow(
		g,
		"completeConversationFlow",
		func(ctx context.Context, input CompleteConversationInput) (CompleteConversationOutput, error) {
			startTime := time.Now()
			var executedSteps []ExecutedStep
			var warnings []string

			logger.InfoContext(ctx, "开始完整对话流程",
				"session_id", input.SessionID,
				"enable_query_classify", input.EnableQueryClassify,
				"auto_optimize_context", input.AutoOptimizeContext,
				"enable_streaming", input.EnableStreaming,
				"save_memory", input.SaveMemory,
				"auto_generate_summary", input.AutoGenerateSummary,
			)

			// 1. 参数验证
			if err := validateCompleteConversationInput(input); err != nil {
				return CompleteConversationOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 查询分类（可选步骤）
			var queryClassification *QueryClassifyOutput
			if input.EnableQueryClassify {
				step := executeStep(ctx, "查询分类", true, func() error {
					result, err := queryClassifySvc.ClassifyQuery(ctx, service.ClassifyQueryRequest{
						Query:     input.UserMessage,
						SessionID: input.SessionID,
					})
					if err != nil {
						return err
					}
					queryClassification = &QueryClassifyOutput{
						QueryType:           result.QueryType,
						NeedsHistory:        result.NeedsHistory,
						KeyEntities:         result.KeyEntities,
						RecommendedStrategy: result.RecommendedStrategy,
						Confidence:          result.Confidence,
						Reasoning:           result.Reasoning,
					}
					return nil
				})
				executedSteps = append(executedSteps, step)

				// 可选步骤失败时记录警告但继续执行
				if step.Status == "failed" {
					warnings = append(warnings, fmt.Sprintf("查询分类失败: %s", step.Error))
					logger.WarnContext(ctx, "查询分类失败，继续执行主流程", "error", step.Error)
				}
			} else {
				executedSteps = append(executedSteps, ExecutedStep{
					StepName:    "查询分类",
					Status:      "skipped",
					Duration:    0,
					IsOptional:  true,
					Description: "未启用查询分类",
				})
			}

			// 3. 构建上下文（关键步骤）
			var contextResult *ContextBuildOutput
			step := executeStep(ctx, "构建上下文", false, func() error {
				// 根据查询分类结果调整策略
				strategy := input.ContextStrategy
				if queryClassification != nil && queryClassification.RecommendedStrategy != "" {
					strategy = queryClassification.RecommendedStrategy
				}

				result, err := contextSvc.BuildContext(ctx, &service.BuildContextRequest{
					SessionID:       input.SessionID,
					UserQuery:       input.UserMessage,
					MaxTokens:       input.MaxTokens,
					Strategy:        strategy,
					IncludeSummary:  true,
					IncludeLongTerm: true,
					ShortTermWindow: 10,
				})
				if err != nil {
					return err
				}

				contextResult = &ContextBuildOutput{
					SessionID:         result.SessionID,
					Summary:           convertSummaryToContext(result.Summary),
					LongTermMemories:  convertMemoriesToContext(result.LongTermMemories),
					ShortTermMessages: convertMessagesToContext(result.ShortTermMessages),
					TotalTokens:       result.TotalTokens,
					Strategy:          result.Strategy,
					QualityScore:      result.QualityScore,
					BuildTime:         0, // 将在后面计算
				}
				return nil
			})
			executedSteps = append(executedSteps, step)

			// 关键步骤失败时中断流程
			if step.Status == "failed" {
				return CompleteConversationOutput{
					ExecutedSteps: executedSteps,
					TotalTime:     time.Since(startTime).Milliseconds(),
					Warnings:      warnings,
				}, fmt.Errorf("构建上下文失败: %s", step.Error)
			}

			// 4. 优化上下文（可选步骤）
			if input.AutoOptimizeContext && contextResult.TotalTokens > input.MaxTokens*80/100 {
				step := executeStep(ctx, "优化上下文", true, func() error {
					result, err := contextSvc.OptimizeContext(ctx, &service.OptimizeContextRequest{
						Context:         convertContextBuildOutputToResult(contextResult),
						TargetTokens:    input.MaxTokens * 70 / 100, // 目标70%使用率
						Strategy:        "balanced",
						PreserveSummary: true,
					})
					if err != nil {
						return err
					}

					// 更新上下文结果
					contextResult = &ContextBuildOutput{
						SessionID:         result.SessionID,
						Summary:           convertSummaryToContext(result.Summary),
						LongTermMemories:  convertMemoriesToContext(result.LongTermMemories),
						ShortTermMessages: convertMessagesToContext(result.ShortTermMessages),
						TotalTokens:       result.TotalTokens,
						Strategy:          result.Strategy,
						QualityScore:      result.QualityScore,
						BuildTime:         0,
					}
					return nil
				})
				executedSteps = append(executedSteps, step)

				if step.Status == "failed" {
					warnings = append(warnings, fmt.Sprintf("优化上下文失败: %s", step.Error))
					logger.WarnContext(ctx, "优化上下文失败，使用原始上下文", "error", step.Error)
				}
			} else {
				executedSteps = append(executedSteps, ExecutedStep{
					StepName:    "优化上下文",
					Status:      "skipped",
					Duration:    0,
					IsOptional:  true,
					Description: "Token使用率未超过阈值或未启用自动优化",
				})
			}

			// 5. 生成AI响应（关键步骤）
			var chatOutput *ChatGenerateOutput
			if input.EnableStreaming {
				// 流式响应
				step := executeStep(ctx, "生成AI响应（流式）", false, func() error {
					// 注意：实际的流式实现需要特殊处理，这里简化为调用普通生成
					result, err := chatSvc.GenerateResponse(ctx, service.GenerateResponseRequest{
						SessionID:    input.SessionID,
						UserMessage:  input.UserMessage,
						Context:      convertContextBuildOutputToResult(contextResult),
						ModelConfig:  convertModelConfig(input.ModelConfig),
						SystemPrompt: input.SystemPrompt,
						SaveMessage:  true,
					})
					if err != nil {
						return err
					}

					chatOutput = &ChatGenerateOutput{
						MessageID:      result.MessageID,
						Response:       result.Response,
						TokenUsage:     result.TokenUsage,
						FinishReason:   result.FinishReason,
						Model:          result.Model,
						GenerationTime: result.GenerationTime,
						ContextInfo:    result.ContextInfo,
					}
					return nil
				})
				executedSteps = append(executedSteps, step)

				if step.Status == "failed" {
					return CompleteConversationOutput{
						ExecutedSteps: executedSteps,
						TotalTime:     time.Since(startTime).Milliseconds(),
						Warnings:      warnings,
					}, fmt.Errorf("生成AI响应失败: %s", step.Error)
				}
			} else {
				// 普通响应
				step := executeStep(ctx, "生成AI响应", false, func() error {
					result, err := chatSvc.GenerateResponse(ctx, service.GenerateResponseRequest{
						SessionID:    input.SessionID,
						UserMessage:  input.UserMessage,
						Context:      convertContextBuildOutputToResult(contextResult),
						ModelConfig:  convertModelConfig(input.ModelConfig),
						SystemPrompt: input.SystemPrompt,
						SaveMessage:  true,
					})
					if err != nil {
						return err
					}

					chatOutput = &ChatGenerateOutput{
						MessageID:      result.MessageID,
						Response:       result.Response,
						TokenUsage:     result.TokenUsage,
						FinishReason:   result.FinishReason,
						Model:          result.Model,
						GenerationTime: result.GenerationTime,
						ContextInfo:    result.ContextInfo,
					}
					return nil
				})
				executedSteps = append(executedSteps, step)

				if step.Status == "failed" {
					return CompleteConversationOutput{
						ExecutedSteps: executedSteps,
						TotalTime:     time.Since(startTime).Milliseconds(),
						Warnings:      warnings,
					}, fmt.Errorf("生成AI响应失败: %s", step.Error)
				}
			}

			// 6. 存储记忆（可选步骤，异步执行）
			memoryStored := false
			if input.SaveMemory {
				step := executeStep(ctx, "存储记忆", true, func() error {
					// 异步存储记忆
					go func() {
						asyncCtx := context.Background()
						_, err := memorySvc.StoreMemory(asyncCtx, service.StoreMemoryRequest{
							SessionID:  input.SessionID,
							MessageIDs: []string{chatOutput.MessageID},
							MemoryType: "long_term",
							Importance: 0.5, // 默认重要性
						})
						if err != nil {
							logger.ErrorContext(asyncCtx, "异步存储记忆失败",
								"session_id", input.SessionID,
								"message_id", chatOutput.MessageID,
								"error", err,
							)
						} else {
							logger.InfoContext(asyncCtx, "异步存储记忆成功",
								"session_id", input.SessionID,
								"message_id", chatOutput.MessageID,
							)
						}
					}()
					memoryStored = true
					return nil
				})
				executedSteps = append(executedSteps, step)

				if step.Status == "failed" {
					warnings = append(warnings, fmt.Sprintf("存储记忆失败: %s", step.Error))
					logger.WarnContext(ctx, "存储记忆失败", "error", step.Error)
				}
			} else {
				executedSteps = append(executedSteps, ExecutedStep{
					StepName:    "存储记忆",
					Status:      "skipped",
					Duration:    0,
					IsOptional:  true,
					Description: "未启用记忆存储",
				})
			}

			// 7. 检查并生成摘要（可选步骤，异步执行）
			summaryGenerated := false
			if input.AutoGenerateSummary {
				step := executeStep(ctx, "检查摘要触发", true, func() error {
					triggerResult, err := summarySvc.CheckSummaryTrigger(ctx, input.SessionID)
					if err != nil {
						return err
					}

					if triggerResult.ShouldSummarize {
						// 异步生成摘要
						go func() {
							asyncCtx := context.Background()
							_, err := summarySvc.GenerateSummary(asyncCtx, service.GenerateSummaryRequest{
								SessionID:    input.SessionID,
								SummaryType:  triggerResult.RecommendedType,
								TargetLength: 500,
							})
							if err != nil {
								logger.ErrorContext(asyncCtx, "异步生成摘要失败",
									"session_id", input.SessionID,
									"error", err,
								)
							} else {
								logger.InfoContext(asyncCtx, "异步生成摘要成功",
									"session_id", input.SessionID,
									"summary_type", triggerResult.RecommendedType,
								)
							}
						}()
						summaryGenerated = true
					}
					return nil
				})
				executedSteps = append(executedSteps, step)

				if step.Status == "failed" {
					warnings = append(warnings, fmt.Sprintf("检查摘要触发失败: %s", step.Error))
					logger.WarnContext(ctx, "检查摘要触发失败", "error", step.Error)
				}
			} else {
				executedSteps = append(executedSteps, ExecutedStep{
					StepName:    "检查摘要触发",
					Status:      "skipped",
					Duration:    0,
					IsOptional:  true,
					Description: "未启用自动摘要生成",
				})
			}

			// 8. 构建输出
			totalTime := time.Since(startTime).Milliseconds()

			output := CompleteConversationOutput{
				MessageID:           chatOutput.MessageID,
				Response:            chatOutput.Response,
				TokenUsage:          chatOutput.TokenUsage,
				FinishReason:        chatOutput.FinishReason,
				Model:               chatOutput.Model,
				ExecutedSteps:       executedSteps,
				TotalTime:           totalTime,
				ContextInfo:         chatOutput.ContextInfo,
				QueryClassification: queryClassification,
				MemoryStored:        memoryStored,
				SummaryGenerated:    summaryGenerated,
				Warnings:            warnings,
			}

			logger.InfoContext(ctx, "完整对话流程完成",
				"session_id", input.SessionID,
				"message_id", chatOutput.MessageID,
				"total_time", totalTime,
				"executed_steps", len(executedSteps),
				"warnings", len(warnings),
			)

			return output, nil
		},
	)
}

// executeStep 执行单个步骤并记录耗时
func executeStep(ctx context.Context, stepName string, isOptional bool, fn func() error) ExecutedStep {
	startTime := time.Now()

	logger.InfoContext(ctx, "开始执行步骤",
		"step_name", stepName,
		"is_optional", isOptional,
	)

	err := fn()
	duration := time.Since(startTime).Milliseconds()

	step := ExecutedStep{
		StepName:    stepName,
		Duration:    duration,
		IsOptional:  isOptional,
		Description: fmt.Sprintf("执行%s", stepName),
	}

	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		logger.ErrorContext(ctx, "步骤执行失败",
			"step_name", stepName,
			"duration", duration,
			"error", err,
		)
	} else {
		step.Status = "success"
		logger.InfoContext(ctx, "步骤执行成功",
			"step_name", stepName,
			"duration", duration,
		)
	}

	return step
}

// validateCompleteConversationInput 验证输入参数
func validateCompleteConversationInput(input CompleteConversationInput) error {
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("会话ID格式无效: %w", err)
	}

	if input.UserMessage == "" {
		return fmt.Errorf("用户消息不能为空")
	}

	if len(input.UserMessage) > 4000 {
		return fmt.Errorf("用户消息长度不能超过4000字符")
	}

	if input.MaxTokens < 100 || input.MaxTokens > 32000 {
		return fmt.Errorf("MaxTokens必须在100-32000之间")
	}

	if input.ContextStrategy != "" && input.ContextStrategy != "auto" &&
		input.ContextStrategy != "short" && input.ContextStrategy != "full" {
		return fmt.Errorf("无效的上下文策略: %s", input.ContextStrategy)
	}

	return nil
}

// 辅助转换函数

func convertSummaryToContext(summary *service.SummaryResult) *SummaryContext {
	if summary == nil {
		return nil
	}
	return &SummaryContext{
		Content:    summary.Content,
		TokenCount: summary.TokenCount,
		CreatedAt:  summary.CreatedAt,
		Coverage:   summary.Coverage,
	}
}

func convertMemoriesToContext(memories []*service.MemoryResult) []MemoryContext {
	if memories == nil {
		return nil
	}
	result := make([]MemoryContext, len(memories))
	for i, m := range memories {
		result[i] = MemoryContext{
			ID:         m.ID,
			Content:    m.Content,
			TokenCount: m.TokenCount,
			Importance: m.Importance,
			Similarity: m.Similarity,
			CreatedAt:  m.CreatedAt,
		}
	}
	return result
}

func convertMessagesToContext(messages []*service.MessageResult) []MessageContext {
	if messages == nil {
		return nil
	}
	result := make([]MessageContext, len(messages))
	for i, m := range messages {
		result[i] = MessageContext{
			ID:         m.ID,
			Role:       m.Role,
			Content:    m.Content,
			TokenCount: m.TokenCount,
			CreatedAt:  m.CreatedAt,
		}
	}
	return result
}

func convertContextBuildOutputToResult(output *ContextBuildOutput) *service.ContextResult {
	if output == nil {
		return nil
	}

	var summary *service.SummaryResult
	if output.Summary != nil {
		summary = &service.SummaryResult{
			Content:    output.Summary.Content,
			TokenCount: output.Summary.TokenCount,
			CreatedAt:  output.Summary.CreatedAt,
			Coverage:   output.Summary.Coverage,
		}
	}

	var memories []*service.MemoryResult
	if output.LongTermMemories != nil {
		memories = make([]*service.MemoryResult, len(output.LongTermMemories))
		for i, m := range output.LongTermMemories {
			memories[i] = &service.MemoryResult{
				ID:         m.ID,
				Content:    m.Content,
				TokenCount: m.TokenCount,
				Importance: m.Importance,
				Similarity: m.Similarity,
				CreatedAt:  m.CreatedAt,
			}
		}
	}

	var messages []*service.MessageResult
	if output.ShortTermMessages != nil {
		messages = make([]*service.MessageResult, len(output.ShortTermMessages))
		for i, m := range output.ShortTermMessages {
			messages[i] = &service.MessageResult{
				ID:         m.ID,
				Role:       m.Role,
				Content:    m.Content,
				TokenCount: m.TokenCount,
				CreatedAt:  m.CreatedAt,
			}
		}
	}

	return &service.ContextResult{
		SessionID:         output.SessionID,
		Summary:           summary,
		LongTermMemories:  memories,
		ShortTermMessages: messages,
		TotalTokens:       output.TotalTokens,
		Strategy:          output.Strategy,
		QualityScore:      output.QualityScore,
	}
}

func convertModelConfig(config *ModelConfig) *service.ModelConfig {
	if config == nil {
		return nil
	}
	return &service.ModelConfig{
		ModelName:        config.ModelName,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		MaxTokens:        config.MaxTokens,
		StopSequences:    config.StopSequences,
		FrequencyPenalty: config.FrequencyPenalty,
		PresencePenalty:  config.PresencePenalty,
	}
}
