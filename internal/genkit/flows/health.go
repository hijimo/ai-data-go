// Package flows 实现会话健康检查相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/genkit"
)

// RegisterHealthFlows 注册健康检查相关的 Flow
func RegisterHealthFlows(g *genkit.Genkit, healthService HealthService) {
	// 注册会话健康检查 Flow
	genkit.DefineFlow(
		g,
		"sessionHealthCheckFlow",
		sessionHealthCheckFlow(healthService),
	)
}

// HealthService 健康检查服务接口
type HealthService interface {
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
	CheckResults       []CheckResult
	Issues             []Issue
	Recommendations    []string
	FixOperations      []FixOp
	CheckTime          int64
	LastCheckAt        time.Time
	NextCheckSuggested time.Time
}

// CheckResult 检查结果
type CheckResult struct {
	CheckItem string
	Status    string
	Score     float64
	Message   string
	Details   map[string]interface{}
	Issues    []string
	CheckTime int64
}

// Issue 健康问题
type Issue struct {
	CheckItem   string
	Severity    string
	Type        string
	Description string
	Impact      string
	Suggestion  string
	AutoFixable bool
	Priority    int
}

// FixOp 修复操作
type FixOp struct {
	OperationType string
	CheckItem     string
	Description   string
	Status        string
	Result        string
	Error         string
	ExecutionTime int64
	Details       map[string]interface{}
}

// sessionHealthCheckFlow 实现会话健康检查 Flow
func sessionHealthCheckFlow(healthService HealthService) func(context.Context, *SessionHealthCheckInput) (*SessionHealthCheckOutput, error) {
	return func(ctx context.Context, input *SessionHealthCheckInput) (*SessionHealthCheckOutput, error) {
		startTime := time.Now()

		// 1. 设置默认值
		if len(input.CheckItems) == 0 {
			input.CheckItems = []string{"context", "token", "memory", "summary", "performance"}
		}
		if input.DetailLevel == "" {
			input.DetailLevel = "detailed"
		}

		// 2. 构建服务请求
		req := SessionHealthCheckRequest{
			SessionID:   input.SessionID,
			CheckItems:  input.CheckItems,
			AutoFix:     input.AutoFix,
			DetailLevel: input.DetailLevel,
		}

		// 3. 执行健康检查
		result, err := healthService.CheckSessionHealth(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("会话健康检查失败: %w", err)
		}

		// 4. 转换结果
		output := &SessionHealthCheckOutput{
			SessionID:          result.SessionID,
			OverallHealth:      result.OverallHealth,
			OverallScore:       result.OverallScore,
			CheckResults:       convertCheckResults(result.CheckResults),
			Issues:             convertIssues(result.Issues),
			Recommendations:    result.Recommendations,
			FixOperations:      convertFixOperations(result.FixOperations),
			CheckTime:          time.Since(startTime).Milliseconds(),
			LastCheckAt:        result.LastCheckAt.Format(time.RFC3339),
			NextCheckSuggested: result.NextCheckSuggested.Format(time.RFC3339),
		}

		return output, nil
	}
}

// convertCheckResults 转换检查结果
func convertCheckResults(results []CheckResult) []HealthCheckResult {
	converted := make([]HealthCheckResult, len(results))
	for i, r := range results {
		converted[i] = HealthCheckResult{
			CheckItem: r.CheckItem,
			Status:    r.Status,
			Score:     r.Score,
			Message:   r.Message,
			Details:   r.Details,
			Issues:    r.Issues,
			CheckTime: r.CheckTime,
		}
	}
	return converted
}

// convertIssues 转换健康问题
func convertIssues(issues []Issue) []HealthIssue {
	converted := make([]HealthIssue, len(issues))
	for i, issue := range issues {
		converted[i] = HealthIssue{
			CheckItem:   issue.CheckItem,
			Severity:    issue.Severity,
			Type:        issue.Type,
			Description: issue.Description,
			Impact:      issue.Impact,
			Suggestion:  issue.Suggestion,
			AutoFixable: issue.AutoFixable,
			Priority:    issue.Priority,
		}
	}
	return converted
}

// convertFixOperations 转换修复操作
func convertFixOperations(operations []FixOp) []FixOperation {
	converted := make([]FixOperation, len(operations))
	for i, op := range operations {
		converted[i] = FixOperation{
			OperationType: op.OperationType,
			CheckItem:     op.CheckItem,
			Description:   op.Description,
			Status:        op.Status,
			Result:        op.Result,
			Error:         op.Error,
			ExecutionTime: op.ExecutionTime,
			Details:       op.Details,
		}
	}
	return converted
}
