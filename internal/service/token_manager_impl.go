// internal/service/token_manager_impl.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"gorm.io/gorm"
)

type tokenManagerImpl struct {
	db              *gorm.DB
	sessionRepo     repository.SessionRepository
	messageRepo     repository.MessageRepository
	contextRepo     repository.GenkitContextRepository
	tenantRepo      repository.TenantRepository
}

// NewTokenManager 创建Token管理器
func NewTokenManager(
	db *gorm.DB,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	contextRepo repository.GenkitContextRepository,
	tenantRepo repository.TenantRepository,
) TokenManager {
	return &tokenManagerImpl{
		db:          db,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		contextRepo: contextRepo,
		tenantRepo:  tenantRepo,
	}
}

// CalculateContextTokens 计算上下文Token数量
func (tm *tokenManagerImpl) CalculateContextTokens(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
) int {
	totalTokens := 0

	// 计算消息Token
	for _, msg := range messages {
		if msg.Tokens > 0 {
			totalTokens += msg.Tokens
		} else {
			// 如果没有预计算，使用估算
			totalTokens += tm.CalculateTextTokens(msg.Content)
		}
	}

	// 计算记忆Token
	for _, mem := range memories {
		if mem.TokenCount > 0 {
			totalTokens += mem.TokenCount
		} else {
			totalTokens += tm.CalculateTextTokens(mem.Content)
		}
	}

	// 计算摘要Token
	if summary != nil {
		if summary.TokenCount > 0 {
			totalTokens += summary.TokenCount
		} else {
			totalTokens += tm.CalculateTextTokens(summary.Content)
		}
	}

	return totalTokens
}

// CalculateTextTokens 计算文本Token数量
// 使用简单的估算方法：1 token ≈ 4 个字符（英文）或 1.5 个中文字符
func (tm *tokenManagerImpl) CalculateTextTokens(text string) int {
	if text == "" {
		return 0
	}

	// 统计中文字符和其他字符
	chineseCount := 0
	otherCount := 0

	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		} else {
			otherCount++
		}
	}

	// 中文：1.5字符/token，英文：4字符/token
	tokens := int(math.Ceil(float64(chineseCount)/1.5)) + int(math.Ceil(float64(otherCount)/4.0))

	return tokens
}

// CountTokens 计算文本Token数量（别名方法）
func (tm *tokenManagerImpl) CountTokens(text string) int {
	return tm.CalculateTextTokens(text)
}

// GetBudgetStatus 获取预算状态
func (tm *tokenManagerImpl) GetBudgetStatus(
	ctx context.Context,
	req TokenBudgetRequest,
) (*TokenBudgetResult, error) {
	var usedTokens int
	var totalBudget int
	var err error

	switch req.BudgetType {
	case "session":
		usedTokens, totalBudget, err = tm.getSessionBudget(ctx, req.SessionID)
	case "daily":
		usedTokens, totalBudget, err = tm.getDailyBudget(ctx, req.TenantID)
	case "monthly":
		usedTokens, totalBudget, err = tm.getMonthlyBudget(ctx, req.TenantID)
	default:
		return nil, fmt.Errorf("不支持的预算类型: %s", req.BudgetType)
	}

	if err != nil {
		return nil, fmt.Errorf("获取预算状态失败: %w", err)
	}

	// 计算使用率
	usageRate := 0.0
	if totalBudget > 0 {
		usageRate = float64(usedTokens) / float64(totalBudget)
	}

	// 确定状态
	status := "normal"
	if usageRate >= 1.0 {
		status = "exceeded"
	} else if usageRate >= 0.9 {
		status = "critical"
	} else if usageRate >= 0.7 {
		status = "warning"
	}

	// 生成建议
	suggestions := tm.generateBudgetSuggestions(usageRate, req.BudgetType)

	// 预测耗尽时间
	predictedExhaustion := ""
	if usageRate > 0 && usageRate < 1.0 {
		predictedExhaustion = tm.predictExhaustion(ctx, req, usedTokens, totalBudget)
	}

	return &TokenBudgetResult{
		BudgetType:          req.BudgetType,
		TotalBudget:         totalBudget,
		UsedTokens:          usedTokens,
		RemainingTokens:     totalBudget - usedTokens,
		UsageRate:           usageRate,
		Status:              status,
		Suggestions:         suggestions,
		PredictedExhaustion: predictedExhaustion,
	}, nil
}

// getSessionBudget 获取会话级别预算
func (tm *tokenManagerImpl) getSessionBudget(ctx context.Context, sessionID string) (int, int, error) {
	// 获取会话上下文配置
	contextConfig, err := tm.contextRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return 0, 0, fmt.Errorf("获取会话配置失败: %w", err)
	}

	// 使用已用Token数
	usedTokens := int(contextConfig.TotalTokensUsed)

	// 会话级别预算：默认为MaxTokens的100倍
	totalBudget := contextConfig.MaxTokens * 100

	return usedTokens, totalBudget, nil
}

// getDailyBudget 获取每日预算
func (tm *tokenManagerImpl) getDailyBudget(ctx context.Context, tenantID string) (int, int, error) {
	// 获取租户配置
	tenant, err := tm.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("获取租户信息失败: %w", err)
	}

	// 从租户元数据中获取每日预算，默认100万tokens
	totalBudget := 1000000
	// Metadata是datatypes.JSON类型（实际是[]byte），需要先解析
	if len(tenant.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(tenant.Metadata, &metadata); err == nil {
			if dailyBudget, ok := metadata["daily_token_budget"].(float64); ok {
				totalBudget = int(dailyBudget)
			}
		}
	}

	// 统计今日使用量
	today := time.Now().Truncate(24 * time.Hour)
	usedTokens, err := tm.getTenantTokenUsage(ctx, tenantID, today, time.Now())
	if err != nil {
		return 0, 0, fmt.Errorf("统计今日使用量失败: %w", err)
	}

	return usedTokens, totalBudget, nil
}

// getMonthlyBudget 获取每月预算
func (tm *tokenManagerImpl) getMonthlyBudget(ctx context.Context, tenantID string) (int, int, error) {
	// 获取租户配置
	tenant, err := tm.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("获取租户信息失败: %w", err)
	}

	// 从租户元数据中获取每月预算，默认3000万tokens
	totalBudget := 30000000
	// Metadata是datatypes.JSON类型（实际是[]byte），需要先解析
	if len(tenant.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(tenant.Metadata, &metadata); err == nil {
			if monthlyBudget, ok := metadata["monthly_token_budget"].(float64); ok {
				totalBudget = int(monthlyBudget)
			}
		}
	}

	// 统计本月使用量
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	usedTokens, err := tm.getTenantTokenUsage(ctx, tenantID, monthStart, now)
	if err != nil {
		return 0, 0, fmt.Errorf("统计本月使用量失败: %w", err)
	}

	return usedTokens, totalBudget, nil
}

// getTenantTokenUsage 获取租户在指定时间范围内的Token使用量
func (tm *tokenManagerImpl) getTenantTokenUsage(
	ctx context.Context,
	tenantID string,
	startTime, endTime time.Time,
) (int, error) {
	var totalTokens int64

	// 从conversation_contexts表统计
	err := tm.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Select("COALESCE(SUM(total_tokens_used), 0)").
		Joins("JOIN conversation_sessions ON conversation_contexts.session_id = conversation_sessions.id").
		Where("conversation_sessions.tenant_id = ?", tenantID).
		Where("conversation_contexts.updated_at BETWEEN ? AND ?", startTime, endTime).
		Scan(&totalTokens).Error

	if err != nil {
		return 0, err
	}

	return int(totalTokens), nil
}

// generateBudgetSuggestions 生成预算建议
func (tm *tokenManagerImpl) generateBudgetSuggestions(usageRate float64, budgetType string) []string {
	suggestions := []string{}

	if usageRate >= 1.0 {
		suggestions = append(suggestions, "Token配额已用尽，请升级配额或等待重置")
		suggestions = append(suggestions, "考虑优化上下文策略以减少Token消耗")
	} else if usageRate >= 0.9 {
		suggestions = append(suggestions, "Token使用率超过90%，建议立即采取优化措施")
		suggestions = append(suggestions, "启用上下文压缩和摘要功能")
	} else if usageRate >= 0.8 {
		suggestions = append(suggestions, "Token使用率超过80%，建议优化上下文配置")
		suggestions = append(suggestions, "减少短期记忆窗口大小")
	} else if usageRate >= 0.7 {
		suggestions = append(suggestions, "Token使用率接近预警线，请关注使用情况")
	}

	return suggestions
}

// predictExhaustion 预测配额耗尽时间
func (tm *tokenManagerImpl) predictExhaustion(
	ctx context.Context,
	req TokenBudgetRequest,
	usedTokens, totalBudget int,
) string {
	// 获取历史使用趋势
	var avgDailyUsage int
	var err error

	switch req.BudgetType {
	case "session":
		// 会话级别不预测
		return ""
	case "daily":
		// 每日预算在当天结束时重置
		remaining := time.Until(time.Now().Truncate(24*time.Hour).Add(24 * time.Hour))
		return fmt.Sprintf("将在%s后重置", formatDuration(remaining))
	case "monthly":
		// 基于本月平均每日使用量预测
		avgDailyUsage, err = tm.getAverageDailyUsage(ctx, req.TenantID, 7)
		if err != nil || avgDailyUsage == 0 {
			return ""
		}
	}

	// 计算剩余天数
	remainingTokens := totalBudget - usedTokens
	if avgDailyUsage > 0 {
		daysRemaining := float64(remainingTokens) / float64(avgDailyUsage)
		if daysRemaining > 0 {
			exhaustionDate := time.Now().Add(time.Duration(daysRemaining*24) * time.Hour)
			return exhaustionDate.Format("2006-01-02")
		}
	}

	return ""
}

// getAverageDailyUsage 获取平均每日使用量
func (tm *tokenManagerImpl) getAverageDailyUsage(ctx context.Context, tenantID string, days int) (int, error) {
	startTime := time.Now().AddDate(0, 0, -days)
	totalUsage, err := tm.getTenantTokenUsage(ctx, tenantID, startTime, time.Now())
	if err != nil {
		return 0, err
	}

	return totalUsage / days, nil
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// OptimizeContent 优化内容以减少Token
func (tm *tokenManagerImpl) OptimizeContent(
	ctx context.Context,
	req TokenOptimizeRequest,
) (*TokenOptimizeResult, error) {
	// 计算原始Token数
	originalTokens := tm.CalculateTextTokens(req.Content)

	// 如果已经低于目标，直接返回
	if originalTokens <= req.TargetTokens {
		return &TokenOptimizeResult{
			OriginalContent:  req.Content,
			OptimizedContent: req.Content,
			OriginalTokens:   originalTokens,
			OptimizedTokens:  originalTokens,
			TokensSaved:      0,
			Strategy:         req.Strategy,
			QualityScore:     1.0,
			Operations:       []string{"无需优化"},
		}, nil
	}

	var optimizedContent string
	var operations []string
	var qualityScore float64

	switch req.Strategy {
	case "compress":
		optimizedContent, operations = tm.compressContent(req.Content, req.TargetTokens)
		qualityScore = 0.85
	case "summarize":
		optimizedContent, operations = tm.summarizeContent(req.Content, req.TargetTokens)
		qualityScore = 0.75
	case "truncate":
		optimizedContent, operations = tm.truncateContent(req.Content, req.TargetTokens)
		qualityScore = 0.60
	case "smart":
		optimizedContent, operations, qualityScore = tm.smartOptimize(req.Content, req.TargetTokens, req.QualityThreshold)
	default:
		return nil, fmt.Errorf("不支持的优化策略: %s", req.Strategy)
	}

	// 计算优化后的Token数
	optimizedTokens := tm.CalculateTextTokens(optimizedContent)
	tokensSaved := originalTokens - optimizedTokens

	// 如果质量评分低于阈值，返回错误
	if qualityScore < req.QualityThreshold {
		return nil, fmt.Errorf("优化后质量评分%.2f低于阈值%.2f", qualityScore, req.QualityThreshold)
	}

	return &TokenOptimizeResult{
		OriginalContent:  req.Content,
		OptimizedContent: optimizedContent,
		OriginalTokens:   originalTokens,
		OptimizedTokens:  optimizedTokens,
		TokensSaved:      tokensSaved,
		Strategy:         req.Strategy,
		QualityScore:     qualityScore,
		Operations:       operations,
	}, nil
}

// compressContent 压缩内容
func (tm *tokenManagerImpl) compressContent(content string, targetTokens int) (string, []string) {
	operations := []string{}
	result := content

	// 1. 移除多余空白
	result = strings.Join(strings.Fields(result), " ")
	operations = append(operations, "移除多余空白")

	// 2. 移除重复句子
	sentences := strings.Split(result, "。")
	uniqueSentences := make(map[string]bool)
	var compressed []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" && !uniqueSentences[s] {
			uniqueSentences[s] = true
			compressed = append(compressed, s)
		}
	}
	result = strings.Join(compressed, "。")
	if len(compressed) < len(sentences) {
		operations = append(operations, fmt.Sprintf("移除%d个重复句子", len(sentences)-len(compressed)))
	}

	// 3. 简化表达
	result = strings.ReplaceAll(result, "非常", "很")
	result = strings.ReplaceAll(result, "特别", "很")
	operations = append(operations, "简化表达")

	return result, operations
}

// summarizeContent 摘要内容
func (tm *tokenManagerImpl) summarizeContent(content string, targetTokens int) (string, []string) {
	operations := []string{}

	// 简单的摘要策略：保留前N个句子
	sentences := strings.Split(content, "。")
	targetSentences := len(sentences) * targetTokens / tm.CalculateTextTokens(content)
	if targetSentences < 1 {
		targetSentences = 1
	}
	if targetSentences > len(sentences) {
		targetSentences = len(sentences)
	}

	result := strings.Join(sentences[:targetSentences], "。")
	operations = append(operations, fmt.Sprintf("保留前%d个句子", targetSentences))

	return result, operations
}

// truncateContent 截断内容
func (tm *tokenManagerImpl) truncateContent(content string, targetTokens int) (string, []string) {
	operations := []string{}

	// 计算目标字符数（粗略估算）
	targetChars := targetTokens * 3 // 平均每token约3个字符

	if len(content) <= targetChars {
		return content, operations
	}

	// 截断到目标长度
	result := content[:targetChars]

	// 尝试在句子边界截断
	lastPeriod := strings.LastIndex(result, "。")
	if lastPeriod > targetChars/2 {
		result = result[:lastPeriod+len("。")]
	}

	operations = append(operations, fmt.Sprintf("截断到%d字符", len(result)))

	return result, operations
}

// smartOptimize 智能优化
func (tm *tokenManagerImpl) smartOptimize(content string, targetTokens int, qualityThreshold float64) (string, []string, float64) {
	operations := []string{}
	result := content
	currentTokens := tm.CalculateTextTokens(result)

	// 策略1：先尝试压缩
	if currentTokens > targetTokens {
		compressed, ops := tm.compressContent(result, targetTokens)
		result = compressed
		operations = append(operations, ops...)
		currentTokens = tm.CalculateTextTokens(result)
	}

	// 策略2：如果还不够，尝试摘要
	if currentTokens > targetTokens {
		summarized, ops := tm.summarizeContent(result, targetTokens)
		result = summarized
		operations = append(operations, ops...)
		currentTokens = tm.CalculateTextTokens(result)
	}

	// 策略3：最后才截断
	if currentTokens > targetTokens {
		truncated, ops := tm.truncateContent(result, targetTokens)
		result = truncated
		operations = append(operations, ops...)
	}

	// 计算质量评分
	finalTokens := tm.CalculateTextTokens(result)
	originalTokens := tm.CalculateTextTokens(content)
	compressionRate := float64(finalTokens) / float64(originalTokens)

	// 质量评分：保留率越高，质量越好
	qualityScore := 0.5 + (compressionRate * 0.5)

	return result, operations, qualityScore
}

// AnalyzeUsage 分析Token使用情况
func (tm *tokenManagerImpl) AnalyzeUsage(
	ctx context.Context,
	req TokenAnalysisRequest,
) (*TokenAnalysisResult, error) {
	// 计算时间范围
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -req.TimeRangeDays)

	// 获取使用量数据
	var totalUsage int
	var err error

	if req.SessionID != "" {
		// 会话级别分析
		totalUsage, err = tm.getSessionTokenUsage(ctx, req.SessionID, startTime, endTime)
	} else {
		// 租户级别分析
		totalUsage, err = tm.getTenantTokenUsage(ctx, req.TenantID, startTime, endTime)
	}

	if err != nil {
		return nil, fmt.Errorf("获取使用量失败: %w", err)
	}

	// 计算平均每日使用量
	avgDailyUsage := 0
	if req.TimeRangeDays > 0 {
		avgDailyUsage = totalUsage / req.TimeRangeDays
	}

	// 获取峰值使用量
	peakUsage, err := tm.getPeakUsage(ctx, req.TenantID, req.SessionID, startTime, endTime)
	if err != nil {
		peakUsage = avgDailyUsage // 使用平均值作为后备
	}

	// 分析趋势
	trend := tm.analyzeTrend(ctx, req.TenantID, req.SessionID, req.TimeRangeDays)

	// 估算成本（假设每1000 tokens = $0.002）
	estimatedCost := float64(totalUsage) / 1000.0 * 0.002

	// 计算效率评分
	efficiencyScore := tm.calculateEfficiencyScore(totalUsage, avgDailyUsage, peakUsage)

	// 生成优化建议
	suggestions := tm.generateOptimizationSuggestions(totalUsage, avgDailyUsage, trend, efficiencyScore)

	// 预测未来使用量
	predictions := tm.predictFutureUsage(avgDailyUsage, trend)

	return &TokenAnalysisResult{
		TotalUsage:        totalUsage,
		AverageDailyUsage: avgDailyUsage,
		PeakUsage:         peakUsage,
		Trend:             trend,
		EstimatedCost:     estimatedCost,
		EfficiencyScore:   efficiencyScore,
		Suggestions:       suggestions,
		Predictions:       predictions,
	}, nil
}

// getSessionTokenUsage 获取会话Token使用量
func (tm *tokenManagerImpl) getSessionTokenUsage(
	ctx context.Context,
	sessionID string,
	startTime, endTime time.Time,
) (int, error) {
	var totalTokens int64

	err := tm.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Select("COALESCE(total_tokens_used, 0)").
		Where("session_id = ?", sessionID).
		Where("updated_at BETWEEN ? AND ?", startTime, endTime).
		Scan(&totalTokens).Error

	if err != nil {
		return 0, err
	}

	return int(totalTokens), nil
}

// getPeakUsage 获取峰值使用量
func (tm *tokenManagerImpl) getPeakUsage(
	ctx context.Context,
	tenantID, sessionID string,
	startTime, endTime time.Time,
) (int, error) {
	// 按天统计，找出最大值
	type DailyUsage struct {
		Date  time.Time
		Usage int64
	}

	var dailyUsages []DailyUsage
	query := tm.db.WithContext(ctx).
		Model(&model.ConversationContext{}).
		Select("DATE(updated_at) as date, SUM(total_tokens_used) as usage").
		Where("updated_at BETWEEN ? AND ?", startTime, endTime).
		Group("DATE(updated_at)")

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	} else {
		query = query.
			Joins("JOIN conversation_sessions ON conversation_contexts.session_id = conversation_sessions.id").
			Where("conversation_sessions.tenant_id = ?", tenantID)
	}

	err := query.Scan(&dailyUsages).Error
	if err != nil {
		return 0, err
	}

	// 找出最大值
	maxUsage := int64(0)
	for _, du := range dailyUsages {
		if du.Usage > maxUsage {
			maxUsage = du.Usage
		}
	}

	return int(maxUsage), nil
}

// analyzeTrend 分析使用趋势
func (tm *tokenManagerImpl) analyzeTrend(
	ctx context.Context,
	tenantID, sessionID string,
	days int,
) string {
	// 将时间段分为两半，比较前后使用量
	midPoint := days / 2
	if midPoint < 1 {
		return "stable"
	}

	now := time.Now()
	midTime := now.AddDate(0, 0, -midPoint)
	startTime := now.AddDate(0, 0, -days)

	var firstHalfUsage, secondHalfUsage int
	var err error

	if sessionID != "" {
		firstHalfUsage, _ = tm.getSessionTokenUsage(ctx, sessionID, startTime, midTime)
		secondHalfUsage, _ = tm.getSessionTokenUsage(ctx, sessionID, midTime, now)
	} else {
		firstHalfUsage, _ = tm.getTenantTokenUsage(ctx, tenantID, startTime, midTime)
		secondHalfUsage, _ = tm.getTenantTokenUsage(ctx, tenantID, midTime, now)
	}

	if err != nil {
		return "stable"
	}

	// 计算变化率
	if firstHalfUsage == 0 {
		if secondHalfUsage > 0 {
			return "increasing"
		}
		return "stable"
	}

	changeRate := float64(secondHalfUsage-firstHalfUsage) / float64(firstHalfUsage)

	if changeRate > 0.2 {
		return "increasing"
	} else if changeRate < -0.2 {
		return "decreasing"
	}
	return "stable"
}

// calculateEfficiencyScore 计算效率评分
func (tm *tokenManagerImpl) calculateEfficiencyScore(totalUsage, avgDailyUsage, peakUsage int) float64 {
	// 效率评分基于使用的稳定性和合理性
	// 峰值与平均值的比率越接近1，效率越高
	if avgDailyUsage == 0 {
		return 0.5
	}

	ratio := float64(peakUsage) / float64(avgDailyUsage)

	// 理想比率在1.5-2.0之间
	if ratio >= 1.5 && ratio <= 2.0 {
		return 0.9
	} else if ratio < 1.5 {
		return 0.7 + (ratio / 1.5 * 0.2)
	} else {
		// 峰值过高，效率较低
		return math.Max(0.3, 0.9-(ratio-2.0)*0.1)
	}
}

// generateOptimizationSuggestions 生成优化建议
func (tm *tokenManagerImpl) generateOptimizationSuggestions(
	totalUsage, avgDailyUsage int,
	trend string,
	efficiencyScore float64,
) []OptimizationSuggestion {
	suggestions := []OptimizationSuggestion{}

	// 基于趋势的建议
	if trend == "increasing" {
		suggestions = append(suggestions, OptimizationSuggestion{
			Priority:        "high",
			Suggestion:      "Token使用量持续增长，建议启用上下文优化和摘要功能",
			EstimatedSaving: avgDailyUsage / 4, // 预计节省25%
		})
	}

	// 基于效率评分的建议
	if efficiencyScore < 0.6 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Priority:        "high",
			Suggestion:      "Token使用效率较低，建议优化上下文策略和减少冗余内容",
			EstimatedSaving: avgDailyUsage / 3, // 预计节省33%
		})
	}

	// 通用优化建议
	if totalUsage > 100000 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Priority:        "medium",
			Suggestion:      "启用缓存机制以减少重复查询的Token消耗",
			EstimatedSaving: avgDailyUsage / 10, // 预计节省10%
		})

		suggestions = append(suggestions, OptimizationSuggestion{
			Priority:        "medium",
			Suggestion:      "定期清理低质量记忆以优化存储和检索效率",
			EstimatedSaving: avgDailyUsage / 20, // 预计节省5%
		})
	}

	// 如果没有建议，添加一个默认建议
	if len(suggestions) == 0 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Priority:        "low",
			Suggestion:      "当前Token使用情况良好，继续保持",
			EstimatedSaving: 0,
		})
	}

	return suggestions
}

// predictFutureUsage 预测未来使用量
func (tm *tokenManagerImpl) predictFutureUsage(avgDailyUsage int, trend string) TokenPredictions {
	// 基础预测值
	nextDay := avgDailyUsage
	nextWeek := avgDailyUsage * 7
	nextMonth := avgDailyUsage * 30

	// 根据趋势调整
	switch trend {
	case "increasing":
		// 假设每天增长5%
		nextDay = int(float64(avgDailyUsage) * 1.05)
		nextWeek = int(float64(avgDailyUsage) * 7 * 1.05)
		nextMonth = int(float64(avgDailyUsage) * 30 * 1.10)
	case "decreasing":
		// 假设每天减少3%
		nextDay = int(float64(avgDailyUsage) * 0.97)
		nextWeek = int(float64(avgDailyUsage) * 7 * 0.97)
		nextMonth = int(float64(avgDailyUsage) * 30 * 0.90)
	}

	return TokenPredictions{
		NextDay:   nextDay,
		NextWeek:  nextWeek,
		NextMonth: nextMonth,
	}
}
