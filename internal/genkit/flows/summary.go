// Package flows 定义摘要相关的 Genkit Flow
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

// SummaryFlowServices 摘要Flow所需的服务依赖
type SummaryFlowServices struct {
	SessionRepo repository.SessionRepository
	MessageRepo repository.MessageRepository
	SummaryRepo repository.SummaryRepository
	ContextRepo repository.GenkitContextRepository
	TokenMgr    service.TokenManager
}

// RegisterSummaryFlows 注册摘要相关的Flow
func RegisterSummaryFlows(g *genkit.Genkit, services *SummaryFlowServices, log logger.Logger) {
	// 注册摘要生成Flow
	genkit.DefineFlow(
		g,
		"summaryGenerateFlow",
		func(ctx context.Context, input SummaryGenerateInput) (SummaryGenerateOutput, error) {
			return executeSummaryGenerateFlow(ctx, g, input, services, log)
		},
	)

	// 注册摘要质量评估Flow
	genkit.DefineFlow(
		g,
		"summaryQualityFlow",
		func(ctx context.Context, input SummaryQualityInput) (SummaryQualityOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if input.Summary == "" && input.SummaryID == "" {
				return SummaryQualityOutput{}, fmt.Errorf("摘要内容或摘要ID必须提供其中之一")
			}

			if len(input.OriginalMessages) == 0 {
				return SummaryQualityOutput{}, fmt.Errorf("原始消息列表不能为空")
			}

			// 2. 如果提供了摘要ID，从数据库加载摘要
			summary := input.Summary
			summaryID := input.SummaryID
			if summaryID != "" && summary == "" {
				summaryRecord, err := services.SummaryRepo.GetByID(ctx, summaryID)
				if err != nil {
					return SummaryQualityOutput{}, fmt.Errorf("获取摘要失败: %w", err)
				}
				summary = summaryRecord.Content
			}

			// 3. 确定评估维度（默认评估所有维度）
			dimensions := input.Dimensions
			if len(dimensions) == 0 {
				dimensions = []string{"completeness", "conciseness", "coherence", "accuracy"}
			}

			// 4. 初始化评估结果
			dimensionScores := make(map[string]float64)
			issues := []QualityIssue{}
			suggestions := []string{}

			// 5. 评估各个维度
			for _, dimension := range dimensions {
				switch dimension {
				case "completeness":
					score, issue, suggestion := evaluateCompleteness(summary, input.OriginalMessages)
					dimensionScores["completeness"] = score
					if issue != nil {
						issues = append(issues, *issue)
					}
					if suggestion != "" {
						suggestions = append(suggestions, suggestion)
					}

				case "conciseness":
					score, issue, suggestion := evaluateConciseness(summary, input.OriginalMessages)
					dimensionScores["conciseness"] = score
					if issue != nil {
						issues = append(issues, *issue)
					}
					if suggestion != "" {
						suggestions = append(suggestions, suggestion)
					}

				case "coherence":
					score, issue, suggestion := evaluateCoherence(summary)
					dimensionScores["coherence"] = score
					if issue != nil {
						issues = append(issues, *issue)
					}
					if suggestion != "" {
						suggestions = append(suggestions, suggestion)
					}

				case "accuracy":
					score, issue, suggestion := evaluateAccuracy(summary, input.OriginalMessages)
					dimensionScores["accuracy"] = score
					if issue != nil {
						issues = append(issues, *issue)
					}
					if suggestion != "" {
						suggestions = append(suggestions, suggestion)
					}
				}
			}

			// 6. 计算总体评分（各维度平均值）
			overallScore := 0.0
			for _, score := range dimensionScores {
				overallScore += score
			}
			overallScore = overallScore / float64(len(dimensionScores))

			// 7. 判断是否通过质量检查（>= 0.7）
			passed := overallScore >= 0.7

			// 8. 计算关键信息覆盖率
			keyInfoCoverage := calculateKeyInfoCoverage(summary, input.OriginalMessages)

			// 9. 计算冗余度评分
			redundancyScore := calculateRedundancyScore(summary)

			// 10. 构建输出
			output := SummaryQualityOutput{
				SummaryID:       summaryID,
				OverallScore:    overallScore,
				DimensionScores: dimensionScores,
				Passed:          passed,
				Issues:          issues,
				Suggestions:     suggestions,
				KeyInfoCoverage: keyInfoCoverage,
				RedundancyScore: redundancyScore,
				EvaluationTime:  time.Since(startTime).Milliseconds(),
			}

			// 11. 记录日志
			if passed {
				logger.InfoContext(ctx, "摘要质量评估：通过",
					"summary_id", summaryID,
					"overall_score", overallScore,
					"key_info_coverage", keyInfoCoverage,
				)
			} else {
				logger.WarnContext(ctx, "摘要质量评估：未通过",
					"summary_id", summaryID,
					"overall_score", overallScore,
					"issues_count", len(issues),
				)
			}

			return output, nil
		},
	)

	// 注册摘要触发检查Flow
	genkit.DefineFlow(
		g,
		"summaryTriggerFlow",
		func(ctx context.Context, input SummaryTriggerInput) (SummaryTriggerOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if input.SessionID == "" {
				return SummaryTriggerOutput{}, fmt.Errorf("会话ID不能为空")
			}

			// 默认检查模式为auto
			if input.CheckMode == "" {
				input.CheckMode = "auto"
			}

			// 2. 权限验证
			if err := validateSessionAccess(ctx, services.SessionRepo, input.SessionID); err != nil {
				return SummaryTriggerOutput{}, err
			}

			// 3. 如果是强制模式，直接返回需要生成摘要
			if input.CheckMode == "force" {
				return SummaryTriggerOutput{
					ShouldSummarize:       true,
					TriggerReason:         "强制触发",
					TriggerConditions:     []string{"force_mode"},
					RecommendedType:       "full",
					Urgency:               1.0,
					TriggerScore:          1.0,
					CheckTime:             time.Since(startTime).Milliseconds(),
				}, nil
			}

			// 4. 获取会话信息
			session, err := services.SessionRepo.GetByID(ctx, input.SessionID)
			if err != nil {
				return SummaryTriggerOutput{}, fmt.Errorf("获取会话信息失败: %w", err)
			}

			// 5. 获取上下文配置
			contextConfig, err := services.ContextRepo.GetBySessionID(ctx, input.SessionID)
			if err != nil {
				logger.WarnContext(ctx, "获取上下文配置失败，使用默认配置", "error", err)
				// 使用默认配置
				contextConfig = &model.ConversationContext{
					MaxTokens: 4000,
				}
			}

			// 6. 获取最新摘要
			latestSummary, err := services.SummaryRepo.GetLatestBySessionID(ctx, input.SessionID)
			if err != nil {
				logger.WarnContext(ctx, "获取最新摘要失败", "error", err)
			}

			// 7. 计算自上次摘要后的消息数
			var messagesSinceLastSummary int
			var timeSinceLastSummary int64
			
			if latestSummary != nil && latestSummary.EndMessageID != nil {
				// 获取摘要之后的消息
				messagesAfter, err := services.MessageRepo.GetMessagesAfter(ctx, input.SessionID, latestSummary.EndMessageID.String())
				if err != nil {
					logger.WarnContext(ctx, "获取摘要后的消息失败", "error", err)
					messagesSinceLastSummary = session.MessageCount
				} else {
					messagesSinceLastSummary = len(messagesAfter)
				}
				
				// 计算时间差（秒）
				timeSinceLastSummary = int64(time.Since(latestSummary.CreatedAt).Seconds())
			} else {
				// 没有摘要，使用总消息数
				messagesSinceLastSummary = session.MessageCount
				timeSinceLastSummary = int64(time.Since(session.CreatedAt).Seconds())
			}

			// 8. 获取最近的消息来计算Token数
			recentMessages, err := services.MessageRepo.GetLatestMessages(ctx, input.SessionID, 20)
			if err != nil {
				logger.WarnContext(ctx, "获取最近消息失败", "error", err)
				recentMessages = []*model.ChatMessage{}
			}

			// 9. 计算当前Token数量
			currentTokenCount := 0
			for _, msg := range recentMessages {
				currentTokenCount += msg.TokenCount
			}

			// 10. 计算Token使用率
			tokenUsageRate := float64(currentTokenCount) / float64(contextConfig.MaxTokens)

			// 11. 检查五种触发条件
			triggerConditions := []string{}
			triggerScore := 0.0

			// 条件1: 消息数量达到阈值（20条）
			if messagesSinceLastSummary >= 20 {
				triggerConditions = append(triggerConditions, "message_count_threshold")
				triggerScore += 0.3
			}

			// 条件2: Token使用率超过80%
			if tokenUsageRate >= 0.8 {
				triggerConditions = append(triggerConditions, "token_usage_high")
				triggerScore += 0.4
			}

			// 条件3: 距离上次摘要超过24小时且有新消息
			if timeSinceLastSummary >= 86400 && messagesSinceLastSummary > 0 {
				triggerConditions = append(triggerConditions, "time_threshold")
				triggerScore += 0.2
			}

			// 条件4: 上下文质量评分低于0.6（简化计算）
			contextQualityScore := calculateSimpleQualityScore(messagesSinceLastSummary, tokenUsageRate)
			if contextQualityScore < 0.6 {
				triggerConditions = append(triggerConditions, "quality_low")
				triggerScore += 0.3
			}

			// 条件5: 消息数量较多（超过10条）且Token使用率较高（超过60%）
			if messagesSinceLastSummary >= 10 && tokenUsageRate >= 0.6 {
				triggerConditions = append(triggerConditions, "combined_threshold")
				triggerScore += 0.2
			}

			// 12. 判断是否应该生成摘要（触发得分 >= 0.5）
			shouldSummarize := triggerScore >= 0.5

			// 13. 计算紧急程度
			urgency := calculateUrgency(tokenUsageRate, contextQualityScore, messagesSinceLastSummary)

			// 14. 推荐摘要类型
			recommendedType := "incremental"
			if latestSummary == nil || messagesSinceLastSummary >= 30 {
				recommendedType = "full"
			}

			// 15. 估算Token节省量
			estimatedTokenSaving := estimateTokenSaving(currentTokenCount, messagesSinceLastSummary)

			// 16. 构建触发原因
			triggerReason := buildTriggerReason(triggerConditions, shouldSummarize)

			// 17. 构建输出
			output := SummaryTriggerOutput{
				ShouldSummarize:          shouldSummarize,
				TriggerReason:            triggerReason,
				TriggerConditions:        triggerConditions,
				MessagesSinceLastSummary: messagesSinceLastSummary,
				CurrentTokenCount:        currentTokenCount,
				MaxTokens:                contextConfig.MaxTokens,
				TokenUsageRate:           tokenUsageRate,
				ContextQualityScore:      contextQualityScore,
				TimeSinceLastSummary:     timeSinceLastSummary,
				EstimatedTokenSaving:     estimatedTokenSaving,
				Urgency:                  urgency,
				RecommendedType:          recommendedType,
				TriggerScore:             triggerScore,
				CheckTime:                time.Since(startTime).Milliseconds(),
			}

			// 18. 记录日志
			if shouldSummarize {
				logger.InfoContext(ctx, "摘要触发检查：需要生成摘要",
					"session_id", input.SessionID,
					"trigger_score", triggerScore,
					"trigger_conditions", triggerConditions,
					"urgency", urgency,
				)
			} else {
				logger.InfoContext(ctx, "摘要触发检查：暂不需要生成摘要",
					"session_id", input.SessionID,
					"trigger_score", triggerScore,
					"messages_since_last", messagesSinceLastSummary,
				)
			}

			return output, nil
		},
	)
}

// validateSessionAccess 验证会话访问权限
func validateSessionAccess(ctx context.Context, sessionRepo repository.SessionRepository, sessionID string) error {
	// 获取JWT声明
	claims := middleware.GetJWTClaims(ctx)
	if claims == nil {
		return fmt.Errorf("未认证")
	}

	// 查询会话
	session, err := sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}

	// 平台管理员可以访问所有会话
	if hasRole(claims, model.RoleSystemAdmin) {
		return nil
	}

	// 验证会话所属用户的租户ID
	// 注意：这里假设session.UserID是会话创建者的ID
	// 实际实现中需要查询用户信息获取租户ID
	// 为了简化，这里假设JWT中的TenantID已经正确设置
	
	logger.InfoContext(ctx, "会话访问权限验证通过",
		"session_id", sessionID,
		"user_id", claims.Subject,
		"tenant_id", claims.TenantID,
	)

	return nil
}

// hasRole 检查用户是否具有指定角色
func hasRole(claims *middleware.JWTClaims, role string) bool {
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

// calculateSimpleQualityScore 计算简单的质量评分
func calculateSimpleQualityScore(messageCount int, tokenUsageRate float64) float64 {
	// 基础评分从1.0开始
	score := 1.0

	// 消息数量越多，质量越低
	if messageCount > 20 {
		score -= 0.3
	} else if messageCount > 10 {
		score -= 0.2
	}

	// Token使用率越高，质量越低
	if tokenUsageRate > 0.8 {
		score -= 0.3
	} else if tokenUsageRate > 0.6 {
		score -= 0.2
	}

	// 确保评分在0-1范围内
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// calculateUrgency 计算紧急程度
func calculateUrgency(tokenUsageRate, qualityScore float64, messageCount int) float64 {
	urgency := 0.0

	// Token使用率的影响（权重40%）
	urgency += tokenUsageRate * 0.4

	// 质量评分的影响（权重30%，质量越低越紧急）
	urgency += (1.0 - qualityScore) * 0.3

	// 消息数量的影响（权重30%）
	messageUrgency := float64(messageCount) / 30.0 // 30条消息为满分
	if messageUrgency > 1.0 {
		messageUrgency = 1.0
	}
	urgency += messageUrgency * 0.3

	// 确保紧急程度在0-1范围内
	if urgency < 0 {
		urgency = 0
	}
	if urgency > 1 {
		urgency = 1
	}

	return urgency
}

// estimateTokenSaving 估算Token节省量
func estimateTokenSaving(currentTokenCount, messageCount int) int {
	if messageCount == 0 {
		return 0
	}

	// 假设摘要可以将Token数量压缩到原来的30%
	// 节省量 = 当前Token数 * 70%
	saving := int(float64(currentTokenCount) * 0.7)

	return saving
}

// buildTriggerReason 构建触发原因描述
func buildTriggerReason(conditions []string, shouldSummarize bool) string {
	if !shouldSummarize {
		return "未满足摘要触发条件"
	}

	if len(conditions) == 0 {
		return "满足摘要触发条件"
	}

	reasonMap := map[string]string{
		"message_count_threshold": "消息数量达到阈值",
		"token_usage_high":        "Token使用率过高",
		"time_threshold":          "距离上次摘要时间过长",
		"quality_low":             "上下文质量评分过低",
		"combined_threshold":      "消息数量和Token使用率均较高",
		"force_mode":              "强制触发模式",
	}

	reasons := []string{}
	for _, condition := range conditions {
		if reason, ok := reasonMap[condition]; ok {
			reasons = append(reasons, reason)
		}
	}

	if len(reasons) == 0 {
		return "满足摘要触发条件"
	}

	// 拼接原因
	result := "触发条件："
	for i, reason := range reasons {
		if i > 0 {
			result += "、"
		}
		result += reason
	}

	return result
}

// ========== 摘要质量评估辅助函数 ==========

// evaluateCompleteness 评估完整性维度
// 检查摘要是否涵盖了原始消息中的关键信息
func evaluateCompleteness(summary string, originalMessages []string) (float64, *QualityIssue, string) {
	// 1. 提取原始消息中的关键词
	originalKeywords := extractKeywords(originalMessages)
	
	// 2. 提取摘要中的关键词
	summaryKeywords := extractKeywordsFromText(summary)
	
	// 3. 计算关键词覆盖率
	coveredCount := 0
	for _, keyword := range originalKeywords {
		for _, summaryKeyword := range summaryKeywords {
			if keyword == summaryKeyword {
				coveredCount++
				break
			}
		}
	}
	
	coverageRate := 0.0
	if len(originalKeywords) > 0 {
		coverageRate = float64(coveredCount) / float64(len(originalKeywords))
	}
	
	// 4. 基于覆盖率计算评分
	score := coverageRate
	
	// 5. 识别问题
	var issue *QualityIssue
	suggestion := ""
	
	if score < 0.7 {
		severity := "medium"
		if score < 0.5 {
			severity = "high"
		}
		
		issue = &QualityIssue{
			Dimension:   "completeness",
			Severity:    severity,
			Description: fmt.Sprintf("摘要的关键信息覆盖率较低（%.1f%%），可能遗漏了重要内容", coverageRate*100),
			Score:       score,
			Impact:      "摘要可能无法完整反映原始对话的核心内容",
		}
		
		suggestion = "建议重新生成摘要，确保包含所有关键主题和重要信息点"
	}
	
	return score, issue, suggestion
}

// evaluateConciseness 评估简洁性维度
// 检查摘要是否简洁，避免冗余和啰嗦
func evaluateConciseness(summary string, originalMessages []string) (float64, *QualityIssue, string) {
	// 1. 计算原始消息总长度
	originalLength := 0
	for _, msg := range originalMessages {
		originalLength += len(msg)
	}
	
	// 2. 计算摘要长度
	summaryLength := len(summary)
	
	// 3. 计算压缩率
	compressionRate := 0.0
	if originalLength > 0 {
		compressionRate = 1.0 - float64(summaryLength)/float64(originalLength)
	}
	
	// 4. 基于压缩率计算评分
	// 理想压缩率：50%-80%（即摘要长度为原文的20%-50%）
	score := 1.0
	if compressionRate < 0.5 {
		// 压缩率太低，摘要太长
		score = compressionRate / 0.5
	} else if compressionRate > 0.8 {
		// 压缩率太高，摘要可能太短
		score = (1.0 - compressionRate) / 0.2
	}
	
	// 5. 识别问题
	var issue *QualityIssue
	suggestion := ""
	
	if score < 0.7 {
		severity := "medium"
		if score < 0.5 {
			severity = "high"
		}
		
		description := ""
		impact := ""
		if compressionRate < 0.5 {
			description = fmt.Sprintf("摘要过长（压缩率仅%.1f%%），可能包含过多细节", compressionRate*100)
			impact = "摘要失去了简洁性，可能影响阅读效率"
			suggestion = "建议精简摘要内容，去除次要细节，只保留核心信息"
		} else {
			description = fmt.Sprintf("摘要过短（压缩率%.1f%%），可能过于简略", compressionRate*100)
			impact = "摘要可能遗漏重要信息"
			suggestion = "建议适当扩充摘要内容，确保关键信息的完整性"
		}
		
		issue = &QualityIssue{
			Dimension:   "conciseness",
			Severity:    severity,
			Description: description,
			Score:       score,
			Impact:      impact,
		}
	}
	
	return score, issue, suggestion
}

// evaluateCoherence 评估连贯性维度
// 检查摘要的逻辑结构和语言流畅性
func evaluateCoherence(summary string) (float64, *QualityIssue, string) {
	// 1. 检查摘要长度（太短或太长都可能影响连贯性）
	summaryLength := len(summary)
	lengthScore := 1.0
	if summaryLength < 50 {
		lengthScore = float64(summaryLength) / 50.0
	} else if summaryLength > 2000 {
		lengthScore = 2000.0 / float64(summaryLength)
	}
	
	// 2. 检查句子数量（至少应该有2-3个句子）
	sentenceCount := countSentences(summary)
	sentenceScore := 1.0
	if sentenceCount < 2 {
		sentenceScore = 0.5
	} else if sentenceCount > 20 {
		sentenceScore = 0.8
	}
	
	// 3. 检查连接词使用（表示逻辑关系）
	connectorScore := checkConnectors(summary)
	
	// 4. 综合评分
	score := (lengthScore + sentenceScore + connectorScore) / 3.0
	
	// 5. 识别问题
	var issue *QualityIssue
	suggestion := ""
	
	if score < 0.7 {
		severity := "medium"
		if score < 0.5 {
			severity = "high"
		}
		
		description := "摘要的连贯性不足"
		impact := "摘要可能难以理解或逻辑不清晰"
		
		if sentenceCount < 2 {
			description = "摘要过于简短，缺乏必要的结构"
			suggestion = "建议使用多个句子组织内容，使摘要更加连贯"
		} else if sentenceCount > 20 {
			description = "摘要句子过多，结构可能过于复杂"
			suggestion = "建议合并相关句子，简化摘要结构"
		} else {
			description = "摘要缺乏逻辑连接词，句子之间关系不清晰"
			suggestion = "建议使用适当的连接词（如：首先、其次、然后、最后等）增强逻辑性"
		}
		
		issue = &QualityIssue{
			Dimension:   "coherence",
			Severity:    severity,
			Description: description,
			Score:       score,
			Impact:      impact,
		}
	}
	
	return score, issue, suggestion
}

// evaluateAccuracy 评估准确性维度
// 检查摘要是否准确反映原始内容，没有引入错误信息
func evaluateAccuracy(summary string, originalMessages []string) (float64, *QualityIssue, string) {
	// 1. 提取摘要中的关键实体和概念
	summaryEntities := extractEntities(summary)
	
	// 2. 检查这些实体是否在原始消息中出现
	validEntityCount := 0
	for _, entity := range summaryEntities {
		found := false
		for _, msg := range originalMessages {
			if containsEntity(msg, entity) {
				found = true
				break
			}
		}
		if found {
			validEntityCount++
		}
	}
	
	// 3. 计算准确率
	accuracyRate := 1.0
	if len(summaryEntities) > 0 {
		accuracyRate = float64(validEntityCount) / float64(len(summaryEntities))
	}
	
	// 4. 检查是否有明显的矛盾或错误
	// 简化实现：检查摘要长度是否合理
	score := accuracyRate
	
	// 5. 识别问题
	var issue *QualityIssue
	suggestion := ""
	
	if score < 0.9 {
		severity := "low"
		if score < 0.7 {
			severity = "high"
		} else if score < 0.8 {
			severity = "medium"
		}
		
		issue = &QualityIssue{
			Dimension:   "accuracy",
			Severity:    severity,
			Description: fmt.Sprintf("摘要中可能包含不准确的信息（准确率%.1f%%）", accuracyRate*100),
			Score:       score,
			Impact:      "摘要可能误导用户或包含错误信息",
		}
		
		suggestion = "建议仔细核对摘要内容，确保所有信息都来自原始对话"
	}
	
	return score, issue, suggestion
}

// calculateKeyInfoCoverage 计算关键信息覆盖率
func calculateKeyInfoCoverage(summary string, originalMessages []string) float64 {
	// 提取原始消息中的关键信息点
	keyPoints := extractKeyPoints(originalMessages)
	
	// 检查摘要中覆盖了多少关键信息点
	coveredCount := 0
	for _, point := range keyPoints {
		if containsKeyPoint(summary, point) {
			coveredCount++
		}
	}
	
	if len(keyPoints) == 0 {
		return 1.0
	}
	
	return float64(coveredCount) / float64(len(keyPoints))
}

// calculateRedundancyScore 计算冗余度评分
func calculateRedundancyScore(summary string) float64 {
	// 简化实现：检查重复词汇的比例
	words := extractWords(summary)
	uniqueWords := make(map[string]bool)
	
	for _, word := range words {
		uniqueWords[word] = true
	}
	
	if len(words) == 0 {
		return 0.0
	}
	
	// 冗余度 = 1 - (唯一词汇数 / 总词汇数)
	redundancy := 1.0 - float64(len(uniqueWords))/float64(len(words))
	
	// 转换为评分（冗余度越低越好）
	score := 1.0 - redundancy
	
	return score
}

// ========== 文本分析辅助函数 ==========

// extractKeywords 从消息列表中提取关键词
func extractKeywords(messages []string) []string {
	keywords := make(map[string]int)
	
	for _, msg := range messages {
		words := extractWords(msg)
		for _, word := range words {
			// 过滤停用词和短词
			if len(word) > 2 && !isStopWord(word) {
				keywords[word]++
			}
		}
	}
	
	// 选择出现频率较高的词作为关键词
	result := []string{}
	for word, count := range keywords {
		if count >= 2 || len(keywords) < 10 {
			result = append(result, word)
		}
	}
	
	return result
}

// extractKeywordsFromText 从单个文本中提取关键词
func extractKeywordsFromText(text string) []string {
	words := extractWords(text)
	keywords := []string{}
	
	for _, word := range words {
		if len(word) > 2 && !isStopWord(word) {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}

// extractWords 提取文本中的词汇
func extractWords(text string) []string {
	// 简化实现：按空格和标点分割
	words := []string{}
	currentWord := ""
	
	for _, char := range text {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
		   (char >= '0' && char <= '9') || (char >= 0x4e00 && char <= 0x9fa5) {
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

// isStopWord 判断是否为停用词
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"有": true, "和": true, "就": true, "不": true, "人": true,
		"都": true, "一": true, "一个": true, "上": true, "也": true,
		"很": true, "到": true, "说": true, "要": true, "去": true,
		"你": true, "会": true, "着": true, "没有": true, "看": true,
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
	}
	
	return stopWords[word]
}

// countSentences 统计句子数量
func countSentences(text string) int {
	count := 0
	for _, char := range text {
		if char == '。' || char == '！' || char == '？' || 
		   char == '.' || char == '!' || char == '?' {
			count++
		}
	}
	
	// 如果没有标点，至少算一个句子
	if count == 0 && len(text) > 0 {
		count = 1
	}
	
	return count
}

// checkConnectors 检查连接词使用情况
func checkConnectors(text string) float64 {
	connectors := []string{
		"首先", "其次", "然后", "接着", "最后", "总之",
		"因此", "所以", "但是", "然而", "而且", "并且",
		"另外", "此外", "同时", "例如", "比如",
		"first", "second", "then", "next", "finally",
		"therefore", "however", "moreover", "furthermore",
	}
	
	connectorCount := 0
	for _, connector := range connectors {
		if containsSubstring(text, connector) {
			connectorCount++
		}
	}
	
	// 根据连接词数量评分
	score := 0.5 // 基础分
	if connectorCount >= 2 {
		score = 1.0
	} else if connectorCount == 1 {
		score = 0.8
	}
	
	return score
}

// extractEntities 提取实体（简化实现）
func extractEntities(text string) []string {
	// 简化实现：提取较长的词组作为实体
	words := extractWords(text)
	entities := []string{}
	
	for _, word := range words {
		if len(word) >= 3 && !isStopWord(word) {
			entities = append(entities, word)
		}
	}
	
	return entities
}

// containsEntity 检查文本是否包含实体
func containsEntity(text, entity string) bool {
	return containsSubstring(text, entity)
}

// extractKeyPoints 提取关键信息点
func extractKeyPoints(messages []string) []string {
	// 简化实现：将每条消息视为一个关键点
	keyPoints := []string{}
	
	for _, msg := range messages {
		if len(msg) > 10 { // 过滤太短的消息
			keyPoints = append(keyPoints, msg)
		}
	}
	
	return keyPoints
}

// containsKeyPoint 检查摘要是否包含关键点
func containsKeyPoint(summary, keyPoint string) bool {
	// 简化实现：检查关键点中的主要词汇是否在摘要中
	keyWords := extractKeywordsFromText(keyPoint)
	
	matchCount := 0
	for _, word := range keyWords {
		if containsSubstring(summary, word) {
			matchCount++
		}
	}
	
	// 如果至少一半的关键词匹配，认为包含该关键点
	return len(keyWords) > 0 && float64(matchCount)/float64(len(keyWords)) >= 0.5
}

// containsSubstring 检查文本是否包含子串（不区分大小写）
func containsSubstring(text, substr string) bool {
	// 简化实现：直接字符串包含检查
	return len(text) > 0 && len(substr) > 0 && 
	       (findSubstring(text, substr) || findSubstring(toLower(text), toLower(substr)))
}

// findSubstring 查找子串
func findSubstring(text, substr string) bool {
	if len(substr) > len(text) {
		return false
	}
	
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	
	return false
}

// toLower 转换为小写（简化实现，仅处理英文）
func toLower(text string) string {
	result := ""
	for _, char := range text {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

// executeSummaryGenerateFlow 执行摘要生成Flow
func executeSummaryGenerateFlow(
	ctx context.Context,
	g *genkit.Genkit,
	input SummaryGenerateInput,
	services *SummaryFlowServices,
	log logger.Logger,
) (SummaryGenerateOutput, error) {
	startTime := time.Now()

	// 1. 参数验证
	if err := validateSummaryGenerateInput(input); err != nil {
		log.ErrorContext(ctx, "参数验证失败", logger.Fields{
			"error": err.Error(),
		})
		return SummaryGenerateOutput{}, fmt.Errorf("参数验证失败: %w", err)
	}

	// 2. 权限验证
	if err := validateSessionAccess(ctx, services.SessionRepo, input.SessionID); err != nil {
		return SummaryGenerateOutput{}, err
	}

	// 3. 获取会话信息
	session, err := services.SessionRepo.GetByID(ctx, input.SessionID)
	if err != nil {
		return SummaryGenerateOutput{}, fmt.Errorf("获取会话信息失败: %w", err)
	}

	log.InfoContext(ctx, "开始生成摘要", logger.Fields{
		"session_id":    input.SessionID,
		"summary_type":  input.SummaryType,
		"target_length": input.TargetLength,
	})

	// 4. 获取需要摘要的消息
	var messages []*model.ChatMessage
	var startMessageID, endMessageID string

	if len(input.MessageIDs) > 0 {
		// 使用指定的消息ID列表
		for _, msgID := range input.MessageIDs {
			msg, err := services.MessageRepo.GetByID(ctx, msgID)
			if err != nil {
				log.WarnContext(ctx, "获取消息失败", logger.Fields{
					"message_id": msgID,
					"error":      err.Error(),
				})
				continue
			}
			messages = append(messages, msg)
		}
	} else if input.StartMessageID != "" && input.EndMessageID != "" {
		// 使用消息范围
		messages, err = services.MessageRepo.GetMessageRange(ctx, input.SessionID, input.StartMessageID, input.EndMessageID)
		if err != nil {
			return SummaryGenerateOutput{}, fmt.Errorf("获取消息范围失败: %w", err)
		}
		startMessageID = input.StartMessageID
		endMessageID = input.EndMessageID
	} else {
		// 获取最新的摘要
		latestSummary, err := services.SummaryRepo.GetLatestBySessionID(ctx, input.SessionID)
		if err != nil {
			log.WarnContext(ctx, "获取最新摘要失败", logger.Fields{
				"error": err.Error(),
			})
		}

		// 获取自上次摘要后的所有消息
		if latestSummary != nil && latestSummary.EndMessageID != nil {
			messages, err = services.MessageRepo.GetMessagesAfter(ctx, input.SessionID, latestSummary.EndMessageID.String())
			if err != nil {
				return SummaryGenerateOutput{}, fmt.Errorf("获取摘要后的消息失败: %w", err)
			}
			startMessageID = latestSummary.EndMessageID.String()
		} else {
			// 没有摘要，获取所有消息
			messages, err = services.MessageRepo.GetBySessionID(ctx, input.SessionID)
			if err != nil {
				return SummaryGenerateOutput{}, fmt.Errorf("获取会话消息失败: %w", err)
			}
		}

		// 设置结束消息ID为最后一条消息
		if len(messages) > 0 {
			if startMessageID == "" {
				startMessageID = messages[0].ID.String()
			}
			endMessageID = messages[len(messages)-1].ID.String()
		}
	}

	// 5. 验证消息数量
	if len(messages) < 5 {
		return SummaryGenerateOutput{}, fmt.Errorf("消息数量不足（至少需要5条），当前: %d", len(messages))
	}

	log.InfoContext(ctx, "获取到待摘要的消息", logger.Fields{
		"message_count":    len(messages),
		"start_message_id": startMessageID,
		"end_message_id":   endMessageID,
	})

	// 6. 计算原始Token数量
	originalTokenCount := 0
	for _, msg := range messages {
		originalTokenCount += msg.TokenCount
	}

	// 7. 构建摘要提示词
	prompt := buildSummaryPrompt(input, messages, services.TokenMgr)

	log.InfoContext(ctx, "构建摘要提示词完成", logger.Fields{
		"prompt_length":        len(prompt),
		"original_token_count": originalTokenCount,
	})

	// 8. 调用AI生成摘要（使用低温度参数保证稳定性）
	var summaryText string
	maxRetries := 2
	retryCount := 0

	for retryCount <= maxRetries {
		response, err := g.Generate(ctx, ai.WithPrompt(prompt), ai.WithTemperature(0.3))
		if err == nil {
			summaryText = response.Text()
			break
		}

		retryCount++
		if retryCount <= maxRetries {
			log.WarnContext(ctx, "AI生成摘要失败，准备重试", logger.Fields{
				"retry_count": retryCount,
				"error":       err.Error(),
			})
			time.Sleep(time.Duration(retryCount) * time.Second)
		}
	}

	if summaryText == "" {
		return SummaryGenerateOutput{}, fmt.Errorf("AI生成摘要失败，已达最大重试次数")
	}

	log.InfoContext(ctx, "AI生成摘要成功", logger.Fields{
		"summary_length": len(summaryText),
	})

	// 9. 计算摘要Token数量
	summaryTokenCount := 0
	if services.TokenMgr != nil {
		summaryTokenCount = services.TokenMgr.CountTokens(summaryText)
	} else {
		// 简单估算：1个token约等于4个字符
		summaryTokenCount = len(summaryText) / 4
	}

	// 10. 提取关键主题
	keyTopics := extractKeyTopics(summaryText)

	log.InfoContext(ctx, "提取关键主题完成", logger.Fields{
		"topic_count": len(keyTopics),
	})

	// 11. 计算质量评分
	qualityScore := evaluateSummaryQuality(summaryText, messages)

	// 12. 如果质量评分过低，重新生成一次
	if qualityScore < 0.7 && retryCount == 0 {
		log.WarnContext(ctx, "摘要质量评分过低，尝试重新生成", logger.Fields{
			"quality_score": qualityScore,
		})

		// 调整提示词，要求更详细
		enhancedPrompt := prompt + "\n\n请确保摘要包含所有关键信息点，保持完整性和准确性。"
		response, err := g.Generate(ctx, ai.WithPrompt(enhancedPrompt), ai.WithTemperature(0.3))
		if err == nil {
			newSummaryText := response.Text()
			newQualityScore := evaluateSummaryQuality(newSummaryText, messages)

			if newQualityScore > qualityScore {
				summaryText = newSummaryText
				qualityScore = newQualityScore
				summaryTokenCount = services.TokenMgr.CountTokens(summaryText)
				keyTopics = extractKeyTopics(summaryText)

				log.InfoContext(ctx, "重新生成摘要成功，质量提升", logger.Fields{
					"old_quality_score": qualityScore,
					"new_quality_score": newQualityScore,
				})
			}
		}
	}

	// 13. 计算压缩率
	compressionRate := 0.0
	if originalTokenCount > 0 {
		compressionRate = float64(originalTokenCount-summaryTokenCount) / float64(originalTokenCount)
	}

	// 14. 如果压缩率过低，记录警告
	if compressionRate < 0.5 {
		log.WarnContext(ctx, "摘要压缩率过低", logger.Fields{
			"compression_rate":     compressionRate,
			"original_token_count": originalTokenCount,
			"summary_token_count":  summaryTokenCount,
		})
	}

	// 15. 保存摘要到数据库
	sessionUUID, _ := uuid.Parse(input.SessionID)
	startMsgUUID, _ := uuid.Parse(startMessageID)
	endMsgUUID, _ := uuid.Parse(endMessageID)

	summary := &model.ChatSummary{
		SessionID:      sessionUUID,
		Content:        summaryText,
		TokenCount:     summaryTokenCount,
		MessageCount:   len(messages),
		StartMessageID: &startMsgUUID,
		EndMessageID:   &endMsgUUID,
		QualityScore:   qualityScore,
		KeyTopics:      keyTopics,
		SummaryType:    input.SummaryType,
	}

	if err := services.SummaryRepo.Create(ctx, summary); err != nil {
		log.ErrorContext(ctx, "保存摘要失败", logger.Fields{
			"error": err.Error(),
		})
		return SummaryGenerateOutput{}, fmt.Errorf("保存摘要失败: %w", err)
	}

	log.InfoContext(ctx, "摘要保存成功", logger.Fields{
		"summary_id": summary.ID.String(),
	})

	// 16. 更新会话配置（更新最后摘要时间）
	contextConfig, err := services.ContextRepo.GetBySessionID(ctx, input.SessionID)
	if err == nil && contextConfig != nil {
		now := time.Now()
		contextConfig.LastSummaryAt = &now
		if err := services.ContextRepo.Update(ctx, contextConfig); err != nil {
			log.WarnContext(ctx, "更新上下文配置失败", logger.Fields{
				"error": err.Error(),
			})
		}
	}

	// 17. 构建输出
	generationTime := time.Since(startTime).Milliseconds()

	output := SummaryGenerateOutput{
		SummaryID:       summary.ID.String(),
		Summary:         summaryText,
		TokenCount:      summaryTokenCount,
		MessageCount:    len(messages),
		StartMessageID:  startMessageID,
		EndMessageID:    endMessageID,
		QualityScore:    qualityScore,
		CompressionRate: compressionRate,
		KeyTopics:       keyTopics,
		GenerationTime:  generationTime,
	}

	log.InfoContext(ctx, "摘要生成完成", logger.Fields{
		"summary_id":       output.SummaryID,
		"token_count":      summaryTokenCount,
		"quality_score":    qualityScore,
		"compression_rate": compressionRate,
		"generation_time":  generationTime,
	})

	return output, nil
}

// validateSummaryGenerateInput 验证摘要生成输入参数
func validateSummaryGenerateInput(input SummaryGenerateInput) error {
	// 验证会话ID
	if input.SessionID == "" {
		return fmt.Errorf("会话ID不能为空")
	}

	if _, err := uuid.Parse(input.SessionID); err != nil {
		return fmt.Errorf("无效的会话ID格式: %w", err)
	}

	// 验证摘要类型
	if input.SummaryType == "" {
		input.SummaryType = "incremental" // 默认为增量摘要
	}

	validTypes := map[string]bool{
		"incremental": true,
		"full":        true,
	}

	if !validTypes[input.SummaryType] {
		return fmt.Errorf("无效的摘要类型: %s", input.SummaryType)
	}

	// 验证目标长度
	if input.TargetLength < 0 || input.TargetLength > 2000 {
		return fmt.Errorf("目标长度必须在0-2000之间")
	}

	// 验证消息ID格式
	for _, msgID := range input.MessageIDs {
		if _, err := uuid.Parse(msgID); err != nil {
			return fmt.Errorf("无效的消息ID格式: %s", msgID)
		}
	}

	// 验证起始和结束消息ID
	if input.StartMessageID != "" {
		if _, err := uuid.Parse(input.StartMessageID); err != nil {
			return fmt.Errorf("无效的起始消息ID格式: %w", err)
		}
	}

	if input.EndMessageID != "" {
		if _, err := uuid.Parse(input.EndMessageID); err != nil {
			return fmt.Errorf("无效的结束消息ID格式: %w", err)
		}
	}

	return nil
}

// buildSummaryPrompt 构建摘要提示词
func buildSummaryPrompt(input SummaryGenerateInput, messages []*model.ChatMessage, tokenMgr service.TokenManager) string {
	var builder strings.Builder

	// 1. 系统指令
	builder.WriteString("你是一个专业的对话摘要助手。请根据以下对话内容生成一个简洁、准确、完整的摘要。\n\n")

	// 2. 摘要要求
	builder.WriteString("摘要要求：\n")
	builder.WriteString("- 保留所有关键信息和重要观点\n")
	builder.WriteString("- 使用清晰、连贯的语言\n")
	builder.WriteString("- 避免冗余和重复\n")
	builder.WriteString("- 保持客观中立的语气\n")

	// 3. 目标长度
	if input.TargetLength > 0 {
		builder.WriteString(fmt.Sprintf("- 控制摘要长度在 %d 个Token左右\n", input.TargetLength))
	} else {
		builder.WriteString("- 控制摘要长度在 200-500 个Token之间\n")
	}

	builder.WriteString("\n")

	// 4. 如果是增量摘要且有之前的摘要，包含之前的摘要
	if input.SummaryType == "incremental" && input.PreviousSummary != "" {
		builder.WriteString("之前的摘要：\n")
		builder.WriteString(input.PreviousSummary)
		builder.WriteString("\n\n")
		builder.WriteString("请基于之前的摘要，补充以下新对话的内容：\n\n")
	} else {
		builder.WriteString("对话内容：\n\n")
	}

	// 5. 添加对话消息
	for i, msg := range messages {
		roleLabel := "用户"
		if msg.Role == "assistant" {
			roleLabel = "助手"
		} else if msg.Role == "system" {
			roleLabel = "系统"
		}

		builder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, roleLabel, msg.Content))
	}

	builder.WriteString("\n")

	// 6. 输出格式要求
	builder.WriteString("请直接输出摘要内容，不需要额外的说明或格式标记。")

	return builder.String()
}

// extractKeyTopics 提取关键主题
func extractKeyTopics(summary string) []string {
	// 简化实现：提取摘要中的关键词作为主题
	keywords := extractKeywordsFromText(summary)

	// 去重并限制数量
	topicMap := make(map[string]bool)
	topics := []string{}

	for _, keyword := range keywords {
		if len(keyword) > 2 && !topicMap[keyword] {
			topicMap[keyword] = true
			topics = append(topics, keyword)

			// 最多返回10个主题
			if len(topics) >= 10 {
				break
			}
		}
	}

	return topics
}

// evaluateSummaryQuality 评估摘要质量
func evaluateSummaryQuality(summary string, messages []*model.ChatMessage) float64 {
	// 简化的质量评估实现
	score := 1.0

	// 1. 检查摘要长度（不能太短或太长）
	summaryLength := len(summary)
	if summaryLength < 50 {
		score -= 0.3
	} else if summaryLength > 2000 {
		score -= 0.2
	}

	// 2. 检查摘要是否包含关键信息
	// 提取原始消息的关键词
	originalKeywords := []string{}
	for _, msg := range messages {
		keywords := extractKeywordsFromText(msg.Content)
		originalKeywords = append(originalKeywords, keywords...)
	}

	// 计算关键词覆盖率
	if len(originalKeywords) > 0 {
		summaryKeywords := extractKeywordsFromText(summary)
		coveredCount := 0

		for _, origKeyword := range originalKeywords {
			for _, summKeyword := range summaryKeywords {
				if origKeyword == summKeyword {
					coveredCount++
					break
				}
			}
		}

		coverageRate := float64(coveredCount) / float64(len(originalKeywords))
		if coverageRate < 0.5 {
			score -= 0.3
		} else if coverageRate < 0.7 {
			score -= 0.1
		}
	}

	// 3. 检查句子数量（至少应该有2-3个句子）
	sentenceCount := countSentences(summary)
	if sentenceCount < 2 {
		score -= 0.2
	}

	// 确保评分在0-1范围内
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}
