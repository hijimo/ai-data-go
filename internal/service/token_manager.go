// internal/service/token_manager.go
package service

import (
	"context"
	"genkit-ai-service/internal/model"
)

// TokenManager Token管理器接口
type TokenManager interface {
	// CalculateContextTokens 计算上下文Token数量
	CalculateContextTokens(
		messages []*model.ChatMessage,
		memories []*model.ConversationMemory,
		summary *model.ConversationSummary,
	) int

	// CalculateTextTokens 计算文本Token数量
	CalculateTextTokens(text string) int

	// CountTokens 计算文本Token数量（别名方法，与CalculateTextTokens相同）
	CountTokens(text string) int

	// GetBudgetStatus 获取预算状态
	GetBudgetStatus(ctx context.Context, req TokenBudgetRequest) (*TokenBudgetResult, error)

	// OptimizeContent 优化内容以减少Token
	OptimizeContent(ctx context.Context, req TokenOptimizeRequest) (*TokenOptimizeResult, error)

	// AnalyzeUsage 分析Token使用情况
	AnalyzeUsage(ctx context.Context, req TokenAnalysisRequest) (*TokenAnalysisResult, error)
}

// TokenBudgetRequest Token预算请求
type TokenBudgetRequest struct {
	SessionID  string
	TenantID   string
	BudgetType string // session, daily, monthly
}

// TokenBudgetResult Token预算结果
type TokenBudgetResult struct {
	BudgetType      string  `json:"budgetType"`
	TotalBudget     int     `json:"totalBudget"`
	UsedTokens      int     `json:"usedTokens"`
	RemainingTokens int     `json:"remainingTokens"`
	UsageRate       float64 `json:"usageRate"`
	Status          string  `json:"status"` // normal, warning, critical, exceeded
	Suggestions     []string `json:"suggestions"`
	PredictedExhaustion string `json:"predictedExhaustion,omitempty"`
}

// TokenOptimizeRequest Token优化请求
type TokenOptimizeRequest struct {
	Content       string
	TargetTokens  int
	Strategy      string  // compress, summarize, truncate, smart
	QualityThreshold float64
}

// TokenOptimizeResult Token优化结果
type TokenOptimizeResult struct {
	OriginalContent string  `json:"originalContent"`
	OptimizedContent string `json:"optimizedContent"`
	OriginalTokens  int     `json:"originalTokens"`
	OptimizedTokens int     `json:"optimizedTokens"`
	TokensSaved     int     `json:"tokensSaved"`
	Strategy        string  `json:"strategy"`
	QualityScore    float64 `json:"qualityScore"`
	Operations      []string `json:"operations"`
}

// TokenAnalysisRequest Token分析请求
type TokenAnalysisRequest struct {
	TenantID      string
	SessionID     string
	TimeRangeDays int
	Dimensions    []string // usage, trend, cost, efficiency
}

// TokenAnalysisResult Token分析结果
type TokenAnalysisResult struct {
	TotalUsage      int                    `json:"totalUsage"`
	AverageDailyUsage int                  `json:"averageDailyUsage"`
	PeakUsage       int                    `json:"peakUsage"`
	Trend           string                 `json:"trend"` // increasing, stable, decreasing
	EstimatedCost   float64                `json:"estimatedCost"`
	EfficiencyScore float64                `json:"efficiencyScore"`
	Suggestions     []OptimizationSuggestion `json:"suggestions"`
	Predictions     TokenPredictions       `json:"predictions"`
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	Priority        string  `json:"priority"` // high, medium, low
	Suggestion      string  `json:"suggestion"`
	EstimatedSaving int     `json:"estimatedSaving"`
}

// TokenPredictions Token预测
type TokenPredictions struct {
	NextDay   int `json:"nextDay"`
	NextWeek  int `json:"nextWeek"`
	NextMonth int `json:"nextMonth"`
}
