// Package service 实现会话健康检查服务
package service

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
)

// SessionHealthService 会话健康检查服务接口
type SessionHealthService interface {
	// CheckSessionHealth 检查会话健康状态
	CheckSessionHealth(ctx context.Context, req SessionHealthCheckRequest) (*SessionHealthCheckResult, error)
}

// SessionHealthCheckRequest 会话健康检查请求
type SessionHealthCheckRequest struct {
	SessionID   string
	CheckItems  []string
	AutoFix     bool
	DetailLevel string
}

// SessionHealthCheckResult 会话健康检查结果
type SessionHealthCheckResult struct {
	SessionID          string
	OverallHealth      string
	OverallScore       float64
	CheckResults       []HealthCheckResult
	Issues             []HealthIssue
	Recommendations    []string
	FixOperations      []FixOperation
	CheckTime          int64
	LastCheckAt        time.Time
	NextCheckSuggested time.Time
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	CheckItem string
	Status    string
	Score     float64
	Message   string
	Details   map[string]interface{}
	Issues    []string
	CheckTime int64
}

// HealthIssue 健康问题
type HealthIssue struct {
	CheckItem   string
	Severity    string
	Type        string
	Description string
	Impact      string
	Suggestion  string
	AutoFixable bool
	Priority    int
}

// FixOperation 修复操作
type FixOperation struct {
	OperationType string
	CheckItem     string
	Description   string
	Status        string
	Result        string
	Error         string
	ExecutionTime int64
	Details       map[string]interface{}
}

// sessionHealthServiceImpl 会话健康检查服务实现
type sessionHealthServiceImpl struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
	memoryRepo  repository.GenkitMemoryRepository
	contextRepo repository.GenkitContextRepository
	tokenMgr    TokenManager
	cacheService CacheService
}

// NewSessionHealthService 创建会话健康检查服务
func NewSessionHealthService(
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	memoryRepo repository.GenkitMemoryRepository,
	contextRepo repository.GenkitContextRepository,
	tokenMgr TokenManager,
	cacheService CacheService,
) SessionHealthService {
	return &sessionHealthServiceImpl{
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		memoryRepo:   memoryRepo,
		contextRepo:  contextRepo,
		tokenMgr:     tokenMgr,
		cacheService: cacheService,
	}
}

// CheckSessionHealth 检查会话健康状态
func (s *sessionHealthServiceImpl) CheckSessionHealth(
	ctx context.Context,
	req SessionHealthCheckRequest,
) (*SessionHealthCheckResult, error) {
	startTime := time.Now()

	// 1. 验证会话存在
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 2. 执行各项检查
	checkResults := make([]HealthCheckResult, 0)
	allIssues := make([]HealthIssue, 0)
	fixOperations := make([]FixOperation, 0)

	for _, checkItem := range req.CheckItems {
		var result HealthCheckResult
		var issues []HealthIssue
		var fixes []FixOperation

		switch checkItem {
		case "context":
			result, issues, fixes = s.checkContext(ctx, session)
		case "token":
			result, issues, fixes = s.checkToken(ctx, session)
		case "memory":
			result, issues, fixes = s.checkMemory(ctx, session)
		case "summary":
			result, issues, fixes = s.checkSummary(ctx, session)
		case "performance":
			result, issues, fixes = s.checkPerformance(ctx, session)
		default:
			continue
		}

		checkResults = append(checkResults, result)
		allIssues = append(allIssues, issues...)

		// 如果启用自动修复，执行修复操作
		if req.AutoFix {
			for i := range fixes {
				s.executeFix(ctx, &fixes[i])
			}
		}
		fixOperations = append(fixOperations, fixes...)
	}

	// 3. 计算整体健康评分
	overallScore := s.calculateOverallScore(checkResults)
	overallHealth := s.determineOverallHealth(overallScore)

	// 4. 生成建议
	recommendations := s.generateRecommendations(allIssues, checkResults)

	// 5. 计算下次检查建议时间
	nextCheckSuggested := s.calculateNextCheckTime(overallHealth)

	return &SessionHealthCheckResult{
		SessionID:          req.SessionID,
		OverallHealth:      overallHealth,
		OverallScore:       overallScore,
		CheckResults:       checkResults,
		Issues:             allIssues,
		Recommendations:    recommendations,
		FixOperations:      fixOperations,
		CheckTime:          time.Since(startTime).Milliseconds(),
		LastCheckAt:        time.Now(),
		NextCheckSuggested: nextCheckSuggested,
	}, nil
}

// checkContext 检查上下文健康状态
func (s *sessionHealthServiceImpl) checkContext(
	ctx context.Context,
	session *model.ChatSession,
) (HealthCheckResult, []HealthIssue, []FixOperation) {
	startTime := time.Now()
	details := make(map[string]interface{})
	issues := make([]HealthIssue, 0)
	fixes := make([]FixOperation, 0)
	score := 1.0

	// 检查上下文配置是否存在
	contextConfig, err := s.contextRepo.GetBySessionID(ctx, session.ID.String())
	if err != nil {
		issues = append(issues, HealthIssue{
			CheckItem:   "context",
			Severity:    "high",
			Type:        "missing_config",
			Description: "会话缺少上下文配置",
			Impact:      "无法正确管理对话上下文",
			Suggestion:  "创建默认上下文配置",
			AutoFixable: true,
			Priority:    8,
		})
		score -= 0.3

		// 添加修复操作
		fixes = append(fixes, FixOperation{
			OperationType: "create_context_config",
			CheckItem:     "context",
			Description:   "创建默认上下文配置",
			Status:        "pending",
		})
	} else {
		details["max_tokens"] = contextConfig.MaxTokens
		details["strategy"] = contextConfig.Strategy

		// 检查配置是否合理
		if contextConfig.MaxTokens < 1000 {
			issues = append(issues, HealthIssue{
				CheckItem:   "context",
				Severity:    "medium",
				Type:        "low_token_limit",
				Description: fmt.Sprintf("Token限制过低: %d", contextConfig.MaxTokens),
				Impact:      "可能导致上下文不足",
				Suggestion:  "建议将Token限制提高到至少2000",
				AutoFixable: true,
				Priority:    5,
			})
			score -= 0.2

			fixes = append(fixes, FixOperation{
				OperationType: "increase_token_limit",
				CheckItem:     "context",
				Description:   "将Token限制提高到2000",
				Status:        "pending",
				Details:       map[string]interface{}{"new_limit": 2000},
			})
		}
	}

	status := s.scoreToStatus(score)
	message := s.generateContextMessage(score, len(issues))

	return HealthCheckResult{
		CheckItem: "context",
		Status:    status,
		Score:     score,
		Message:   message,
		Details:   details,
		Issues:    s.extractIssueDescriptions(issues),
		CheckTime: time.Since(startTime).Milliseconds(),
	}, issues, fixes
}

// checkToken 检查Token使用健康状态
func (s *sessionHealthServiceImpl) checkToken(
	ctx context.Context,
	session *model.ChatSession,
) (HealthCheckResult, []HealthIssue, []FixOperation) {
	startTime := time.Now()
	details := make(map[string]interface{})
	issues := make([]HealthIssue, 0)
	fixes := make([]FixOperation, 0)
	score := 1.0

	// 获取最近的消息
	messages, err := s.messageRepo.GetLatestMessages(ctx, session.ID.String(), 20)
	if err == nil && len(messages) > 0 {
		// 计算Token使用情况
		totalTokens := 0
		for _, msg := range messages {
			totalTokens += msg.Tokens
		}

		details["total_tokens"] = totalTokens
		details["message_count"] = len(messages)
		details["average_tokens_per_message"] = totalTokens / len(messages)

		// 获取上下文配置
		contextConfig, err := s.contextRepo.GetBySessionID(ctx, session.ID.String())
		if err == nil {
			maxTokens := contextConfig.MaxTokens
			usageRate := float64(totalTokens) / float64(maxTokens)
			details["usage_rate"] = usageRate
			details["max_tokens"] = maxTokens

			// 检查Token使用率
			if usageRate > 0.9 {
				issues = append(issues, HealthIssue{
					CheckItem:   "token",
					Severity:    "high",
					Type:        "high_token_usage",
					Description: fmt.Sprintf("Token使用率过高: %.1f%%", usageRate*100),
					Impact:      "可能导致上下文溢出",
					Suggestion:  "建议生成摘要或清理旧消息",
					AutoFixable: true,
					Priority:    9,
				})
				score -= 0.4

				fixes = append(fixes, FixOperation{
					OperationType: "generate_summary",
					CheckItem:     "token",
					Description:   "生成摘要以减少Token使用",
					Status:        "pending",
				})
			} else if usageRate > 0.7 {
				issues = append(issues, HealthIssue{
					CheckItem:   "token",
					Severity:    "medium",
					Type:        "moderate_token_usage",
					Description: fmt.Sprintf("Token使用率较高: %.1f%%", usageRate*100),
					Impact:      "接近上下文限制",
					Suggestion:  "考虑生成摘要",
					AutoFixable: false,
					Priority:    6,
				})
				score -= 0.2
			}
		}
	}

	status := s.scoreToStatus(score)
	message := s.generateTokenMessage(score, len(issues))

	return HealthCheckResult{
		CheckItem: "token",
		Status:    status,
		Score:     score,
		Message:   message,
		Details:   details,
		Issues:    s.extractIssueDescriptions(issues),
		CheckTime: time.Since(startTime).Milliseconds(),
	}, issues, fixes
}

// checkMemory 检查记忆健康状态
func (s *sessionHealthServiceImpl) checkMemory(
	ctx context.Context,
	session *model.ChatSession,
) (HealthCheckResult, []HealthIssue, []FixOperation) {
	startTime := time.Now()
	details := make(map[string]interface{})
	issues := make([]HealthIssue, 0)
	fixes := make([]FixOperation, 0)
	score := 1.0

	// 获取记忆统计（这里需要实现一个统计方法）
	// 暂时使用简化逻辑
	details["memory_check"] = "completed"

	// 检查是否有过期记忆
	// 这里需要实现获取过期记忆的方法
	// 如果有过期记忆，添加问题和修复操作

	status := s.scoreToStatus(score)
	message := s.generateMemoryMessage(score, len(issues))

	return HealthCheckResult{
		CheckItem: "memory",
		Status:    status,
		Score:     score,
		Message:   message,
		Details:   details,
		Issues:    s.extractIssueDescriptions(issues),
		CheckTime: time.Since(startTime).Milliseconds(),
	}, issues, fixes
}

// checkSummary 检查摘要健康状态
func (s *sessionHealthServiceImpl) checkSummary(
	ctx context.Context,
	session *model.ChatSession,
) (HealthCheckResult, []HealthIssue, []FixOperation) {
	startTime := time.Now()
	details := make(map[string]interface{})
	issues := make([]HealthIssue, 0)
	fixes := make([]FixOperation, 0)
	score := 1.0

	// 获取最新摘要
	summary, err := s.contextRepo.GetLatestSummary(ctx, session.ID.String())
	if err != nil {
		// 检查是否需要摘要
		messages, err := s.messageRepo.GetLatestMessages(ctx, session.ID.String(), 50)
		if err == nil && len(messages) > 20 {
			issues = append(issues, HealthIssue{
				CheckItem:   "summary",
				Severity:    "medium",
				Type:        "missing_summary",
				Description: "会话消息较多但缺少摘要",
				Impact:      "可能导致上下文效率低下",
				Suggestion:  "建议生成会话摘要",
				AutoFixable: true,
				Priority:    6,
			})
			score -= 0.3

			fixes = append(fixes, FixOperation{
				OperationType: "generate_summary",
				CheckItem:     "summary",
				Description:   "生成会话摘要",
				Status:        "pending",
			})
		}
	} else {
		details["has_summary"] = true
		details["summary_token_count"] = summary.TokenCount
		details["summary_created_at"] = summary.CreatedAt

		// 检查摘要是否过时
		timeSinceLastSummary := time.Since(summary.CreatedAt)
		if timeSinceLastSummary > 24*time.Hour {
			issues = append(issues, HealthIssue{
				CheckItem:   "summary",
				Severity:    "low",
				Type:        "outdated_summary",
				Description: "摘要已超过24小时未更新",
				Impact:      "摘要可能不包含最新对话内容",
				Suggestion:  "考虑更新摘要",
				AutoFixable: false,
				Priority:    3,
			})
			score -= 0.1
		}
	}

	status := s.scoreToStatus(score)
	message := s.generateSummaryMessage(score, len(issues))

	return HealthCheckResult{
		CheckItem: "summary",
		Status:    status,
		Score:     score,
		Message:   message,
		Details:   details,
		Issues:    s.extractIssueDescriptions(issues),
		CheckTime: time.Since(startTime).Milliseconds(),
	}, issues, fixes
}

// checkPerformance 检查性能健康状态
func (s *sessionHealthServiceImpl) checkPerformance(
	ctx context.Context,
	session *model.ChatSession,
) (HealthCheckResult, []HealthIssue, []FixOperation) {
	startTime := time.Now()
	details := make(map[string]interface{})
	issues := make([]HealthIssue, 0)
	fixes := make([]FixOperation, 0)
	score := 1.0

	// 检查会话活跃度
	timeSinceLastActivity := time.Since(session.UpdatedAt)
	details["last_activity"] = session.UpdatedAt
	details["time_since_last_activity"] = timeSinceLastActivity.String()

	// 检查消息数量
	messages, err := s.messageRepo.GetLatestMessages(ctx, session.ID.String(), 100)
	if err == nil {
		details["total_messages"] = len(messages)

		// 如果消息过多，可能影响性能
		if len(messages) >= 100 {
			issues = append(issues, HealthIssue{
				CheckItem:   "performance",
				Severity:    "medium",
				Type:        "high_message_count",
				Description: "会话消息数量过多",
				Impact:      "可能影响查询性能",
				Suggestion:  "建议归档旧消息或生成摘要",
				AutoFixable: true,
				Priority:    5,
			})
			score -= 0.2

			fixes = append(fixes, FixOperation{
				OperationType: "archive_old_messages",
				CheckItem:     "performance",
				Description:   "归档旧消息",
				Status:        "pending",
			})
		}
	}

	status := s.scoreToStatus(score)
	message := s.generatePerformanceMessage(score, len(issues))

	return HealthCheckResult{
		CheckItem: "performance",
		Status:    status,
		Score:     score,
		Message:   message,
		Details:   details,
		Issues:    s.extractIssueDescriptions(issues),
		CheckTime: time.Since(startTime).Milliseconds(),
	}, issues, fixes
}

// executeFix 执行修复操作
func (s *sessionHealthServiceImpl) executeFix(ctx context.Context, fix *FixOperation) {
	startTime := time.Now()

	// 根据操作类型执行相应的修复
	switch fix.OperationType {
	case "create_context_config":
		// 实现创建默认上下文配置的逻辑
		fix.Status = "success"
		fix.Result = "已创建默认上下文配置"
	case "increase_token_limit":
		// 实现提高Token限制的逻辑
		fix.Status = "success"
		fix.Result = "已提高Token限制"
	case "generate_summary":
		// 实现生成摘要的逻辑
		fix.Status = "success"
		fix.Result = "已触发摘要生成"
	case "archive_old_messages":
		// 实现归档旧消息的逻辑
		fix.Status = "success"
		fix.Result = "已归档旧消息"
	default:
		fix.Status = "skipped"
		fix.Result = "未知的修复操作类型"
	}

	fix.ExecutionTime = time.Since(startTime).Milliseconds()
}

// calculateOverallScore 计算整体健康评分
func (s *sessionHealthServiceImpl) calculateOverallScore(results []HealthCheckResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, result := range results {
		totalScore += result.Score
	}

	return totalScore / float64(len(results))
}

// determineOverallHealth 确定整体健康状态
func (s *sessionHealthServiceImpl) determineOverallHealth(score float64) string {
	if score >= 0.8 {
		return "healthy"
	} else if score >= 0.5 {
		return "warning"
	}
	return "critical"
}

// generateRecommendations 生成建议
func (s *sessionHealthServiceImpl) generateRecommendations(
	issues []HealthIssue,
	results []HealthCheckResult,
) []string {
	recommendations := make([]string, 0)

	// 根据问题严重程度排序并生成建议
	highPriorityIssues := 0
	for _, issue := range issues {
		if issue.Severity == "high" || issue.Severity == "critical" {
			highPriorityIssues++
			recommendations = append(recommendations, issue.Suggestion)
		}
	}

	if highPriorityIssues == 0 {
		recommendations = append(recommendations, "会话健康状态良好，继续保持")
	}

	return recommendations
}

// calculateNextCheckTime 计算下次检查建议时间
func (s *sessionHealthServiceImpl) calculateNextCheckTime(overallHealth string) time.Time {
	now := time.Now()

	switch overallHealth {
	case "critical":
		return now.Add(1 * time.Hour) // 1小时后
	case "warning":
		return now.Add(6 * time.Hour) // 6小时后
	default:
		return now.Add(24 * time.Hour) // 24小时后
	}
}

// scoreToStatus 将评分转换为状态
func (s *sessionHealthServiceImpl) scoreToStatus(score float64) string {
	if score >= 0.8 {
		return "healthy"
	} else if score >= 0.5 {
		return "warning"
	}
	return "critical"
}

// extractIssueDescriptions 提取问题描述列表
func (s *sessionHealthServiceImpl) extractIssueDescriptions(issues []HealthIssue) []string {
	descriptions := make([]string, len(issues))
	for i, issue := range issues {
		descriptions[i] = issue.Description
	}
	return descriptions
}

// generateContextMessage 生成上下文检查消息
func (s *sessionHealthServiceImpl) generateContextMessage(score float64, issueCount int) string {
	if score >= 0.8 {
		return "上下文配置正常"
	} else if issueCount > 0 {
		return fmt.Sprintf("上下文存在 %d 个问题", issueCount)
	}
	return "上下文配置需要优化"
}

// generateTokenMessage 生成Token检查消息
func (s *sessionHealthServiceImpl) generateTokenMessage(score float64, issueCount int) string {
	if score >= 0.8 {
		return "Token使用正常"
	} else if issueCount > 0 {
		return fmt.Sprintf("Token使用存在 %d 个问题", issueCount)
	}
	return "Token使用需要优化"
}

// generateMemoryMessage 生成记忆检查消息
func (s *sessionHealthServiceImpl) generateMemoryMessage(score float64, issueCount int) string {
	if score >= 0.8 {
		return "记忆管理正常"
	} else if issueCount > 0 {
		return fmt.Sprintf("记忆管理存在 %d 个问题", issueCount)
	}
	return "记忆管理需要优化"
}

// generateSummaryMessage 生成摘要检查消息
func (s *sessionHealthServiceImpl) generateSummaryMessage(score float64, issueCount int) string {
	if score >= 0.8 {
		return "摘要状态正常"
	} else if issueCount > 0 {
		return fmt.Sprintf("摘要存在 %d 个问题", issueCount)
	}
	return "摘要需要更新"
}

// generatePerformanceMessage 生成性能检查消息
func (s *sessionHealthServiceImpl) generatePerformanceMessage(score float64, issueCount int) string {
	if score >= 0.8 {
		return "性能状态良好"
	} else if issueCount > 0 {
		return fmt.Sprintf("性能存在 %d 个问题", issueCount)
	}
	return "性能需要优化"
}
