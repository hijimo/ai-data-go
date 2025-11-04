// internal/genkit/flows/token.go
package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"genkit-ai-service/internal/service"
)

// TokenBudgetInput Token预算Flow输入
type TokenBudgetInput struct {
	SessionID  string `json:"sessionId" validate:"omitempty,uuid"`
	TenantID   string `json:"tenantId" validate:"required,uuid"`
	BudgetType string `json:"budgetType" validate:"required,oneof=session daily monthly"`
}

// TokenBudgetOutput Token预算Flow输出
type TokenBudgetOutput struct {
	BudgetType          string   `json:"budgetType"`
	TotalBudget         int      `json:"totalBudget"`
	UsedTokens          int      `json:"usedTokens"`
	RemainingTokens     int      `json:"remainingTokens"`
	UsageRate           float64  `json:"usageRate"`
	Status              string   `json:"status"`
	Suggestions         []string `json:"suggestions"`
	PredictedExhaustion string   `json:"predictedExhaustion,omitempty"`
	CheckTime           string   `json:"checkTime"`
}

// TokenOptimizeInput Token优化Flow输入
type TokenOptimizeInput struct {
	Content          string  `json:"content" validate:"required,max=10000"`
	TargetTokens     int     `json:"targetTokens" validate:"required,min=10,max=8000"`
	Strategy         string  `json:"strategy" validate:"required,oneof=compress summarize truncate smart"`
	QualityThreshold float64 `json:"qualityThreshold" validate:"min=0,max=1"`
}

// TokenOptimizeOutput Token优化Flow输出
type TokenOptimizeOutput struct {
	OriginalContent  string   `json:"originalContent"`
	OptimizedContent string   `json:"optimizedContent"`
	OriginalTokens   int      `json:"originalTokens"`
	OptimizedTokens  int      `json:"optimizedTokens"`
	TokensSaved      int      `json:"tokensSaved"`
	SavingRate       float64  `json:"savingRate"`
	Strategy         string   `json:"strategy"`
	QualityScore     float64  `json:"qualityScore"`
	Operations       []string `json:"operations"`
	OptimizeTime     int64    `json:"optimizeTime"`
}

// TokenAnalysisInput Token分析Flow输入
type TokenAnalysisInput struct {
	TenantID      string   `json:"tenantId" validate:"required,uuid"`
	SessionID     string   `json:"sessionId" validate:"omitempty,uuid"`
	TimeRangeDays int      `json:"timeRangeDays" validate:"required,min=1,max=365"`
	Dimensions    []string `json:"dimensions" validate:"dive,oneof=usage trend cost efficiency"`
}

// TokenAnalysisOutput Token分析Flow输出
type TokenAnalysisOutput struct {
	TotalUsage        int                              `json:"totalUsage"`
	AverageDailyUsage int                              `json:"averageDailyUsage"`
	PeakUsage         int                              `json:"peakUsage"`
	Trend             string                           `json:"trend"`
	EstimatedCost     float64                          `json:"estimatedCost"`
	EfficiencyScore   float64                          `json:"efficiencyScore"`
	Suggestions       []service.OptimizationSuggestion `json:"suggestions"`
	Predictions       service.TokenPredictions         `json:"predictions"`
	AnalysisTime      string                           `json:"analysisTime"`
}

// RegisterTokenFlows 注册Token管理相关Flow
func RegisterTokenFlows(g *genkit.Genkit, tokenMgr service.TokenManager) {
	// Token预算管理Flow
	genkit.DefineFlow(
		g,
		"tokenBudgetFlow",
		func(ctx context.Context, input TokenBudgetInput) (TokenBudgetOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateTokenBudgetInput(input); err != nil {
				return TokenBudgetOutput{}, err
			}

			// 2. 调用服务层获取预算状态
			result, err := tokenMgr.GetBudgetStatus(ctx, service.TokenBudgetRequest{
				SessionID:  input.SessionID,
				TenantID:   input.TenantID,
				BudgetType: input.BudgetType,
			})
			if err != nil {
				return TokenBudgetOutput{}, fmt.Errorf("获取Token预算状态失败: %w", err)
			}

			// 3. 构建输出
			output := TokenBudgetOutput{
				BudgetType:          result.BudgetType,
				TotalBudget:         result.TotalBudget,
				UsedTokens:          result.UsedTokens,
				RemainingTokens:     result.RemainingTokens,
				UsageRate:           result.UsageRate,
				Status:              result.Status,
				Suggestions:         result.Suggestions,
				PredictedExhaustion: result.PredictedExhaustion,
				CheckTime:           time.Now().Format(time.RFC3339),
			}

			// 记录执行时间
			_ = time.Since(startTime)

			return output, nil
		},
	)

	// Token优化Flow
	genkit.DefineFlow(
		g,
		"tokenOptimizeFlow",
		func(ctx context.Context, input TokenOptimizeInput) (TokenOptimizeOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateTokenOptimizeInput(input); err != nil {
				return TokenOptimizeOutput{}, err
			}

			// 2. 设置默认质量阈值
			qualityThreshold := input.QualityThreshold
			if qualityThreshold == 0 {
				qualityThreshold = 0.7
			}

			// 3. 调用服务层优化内容
			result, err := tokenMgr.OptimizeContent(ctx, service.TokenOptimizeRequest{
				Content:          input.Content,
				TargetTokens:     input.TargetTokens,
				Strategy:         input.Strategy,
				QualityThreshold: qualityThreshold,
			})
			if err != nil {
				return TokenOptimizeOutput{}, fmt.Errorf("Token优化失败: %w", err)
			}

			// 4. 计算节省率
			savingRate := 0.0
			if result.OriginalTokens > 0 {
				savingRate = float64(result.TokensSaved) / float64(result.OriginalTokens)
			}

			// 5. 构建输出
			output := TokenOptimizeOutput{
				OriginalContent:  result.OriginalContent,
				OptimizedContent: result.OptimizedContent,
				OriginalTokens:   result.OriginalTokens,
				OptimizedTokens:  result.OptimizedTokens,
				TokensSaved:      result.TokensSaved,
				SavingRate:       savingRate,
				Strategy:         result.Strategy,
				QualityScore:     result.QualityScore,
				Operations:       result.Operations,
				OptimizeTime:     time.Since(startTime).Milliseconds(),
			}

			return output, nil
		},
	)

	// Token使用分析Flow
	genkit.DefineFlow(
		g,
		"tokenAnalysisFlow",
		func(ctx context.Context, input TokenAnalysisInput) (TokenAnalysisOutput, error) {
			startTime := time.Now()

			// 1. 参数验证
			if err := validateTokenAnalysisInput(input); err != nil {
				return TokenAnalysisOutput{}, err
			}

			// 2. 设置默认分析维度
			dimensions := input.Dimensions
			if len(dimensions) == 0 {
				dimensions = []string{"usage", "trend", "cost", "efficiency"}
			}

			// 3. 调用服务层分析使用情况
			result, err := tokenMgr.AnalyzeUsage(ctx, service.TokenAnalysisRequest{
				TenantID:      input.TenantID,
				SessionID:     input.SessionID,
				TimeRangeDays: input.TimeRangeDays,
				Dimensions:    dimensions,
			})
			if err != nil {
				return TokenAnalysisOutput{}, fmt.Errorf("Token使用分析失败: %w", err)
			}

			// 4. 构建输出
			output := TokenAnalysisOutput{
				TotalUsage:        result.TotalUsage,
				AverageDailyUsage: result.AverageDailyUsage,
				PeakUsage:         result.PeakUsage,
				Trend:             result.Trend,
				EstimatedCost:     result.EstimatedCost,
				EfficiencyScore:   result.EfficiencyScore,
				Suggestions:       result.Suggestions,
				Predictions:       result.Predictions,
				AnalysisTime:      time.Now().Format(time.RFC3339),
			}

			// 记录执行时间
			_ = time.Since(startTime)

			return output, nil
		},
	)
}

// validateTokenBudgetInput 验证Token预算输入
func validateTokenBudgetInput(input TokenBudgetInput) error {
	if input.TenantID == "" {
		return fmt.Errorf("租户ID不能为空")
	}

	if input.BudgetType == "" {
		return fmt.Errorf("预算类型不能为空")
	}

	validTypes := map[string]bool{"session": true, "daily": true, "monthly": true}
	if !validTypes[input.BudgetType] {
		return fmt.Errorf("无效的预算类型: %s", input.BudgetType)
	}

	if input.BudgetType == "session" && input.SessionID == "" {
		return fmt.Errorf("会话级别预算需要提供会话ID")
	}

	return nil
}

// validateTokenOptimizeInput 验证Token优化输入
func validateTokenOptimizeInput(input TokenOptimizeInput) error {
	if input.Content == "" {
		return fmt.Errorf("内容不能为空")
	}

	if len(input.Content) > 10000 {
		return fmt.Errorf("内容长度不能超过10000字符")
	}

	if input.TargetTokens < 10 || input.TargetTokens > 8000 {
		return fmt.Errorf("目标Token数必须在10-8000之间")
	}

	if input.Strategy == "" {
		return fmt.Errorf("优化策略不能为空")
	}

	validStrategies := map[string]bool{
		"compress":  true,
		"summarize": true,
		"truncate":  true,
		"smart":     true,
	}
	if !validStrategies[input.Strategy] {
		return fmt.Errorf("无效的优化策略: %s", input.Strategy)
	}

	if input.QualityThreshold < 0 || input.QualityThreshold > 1 {
		return fmt.Errorf("质量阈值必须在0-1之间")
	}

	return nil
}

// validateTokenAnalysisInput 验证Token分析输入
func validateTokenAnalysisInput(input TokenAnalysisInput) error {
	if input.TenantID == "" {
		return fmt.Errorf("租户ID不能为空")
	}

	if input.TimeRangeDays < 1 || input.TimeRangeDays > 365 {
		return fmt.Errorf("时间范围必须在1-365天之间")
	}

	// 验证分析维度
	validDimensions := map[string]bool{
		"usage":      true,
		"trend":      true,
		"cost":       true,
		"efficiency": true,
	}

	for _, dim := range input.Dimensions {
		if !validDimensions[dim] {
			return fmt.Errorf("无效的分析维度: %s", dim)
		}
	}

	return nil
}
