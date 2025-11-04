package session

import "time"

// CheckSummaryTriggerRequest 检查摘要触发请求
type CheckSummaryTriggerRequest struct {
	SessionID string `json:"sessionId"`
	CheckMode string `json:"checkMode"` // auto, force
}

// CheckSummaryTriggerResponse 检查摘要触发响应
type CheckSummaryTriggerResponse struct {
	ShouldSummarize          bool     `json:"shouldSummarize"`
	TriggerReason            string   `json:"triggerReason"`
	TriggerConditions        []string `json:"triggerConditions"`
	MessagesSinceLastSummary int      `json:"messagesSinceLastSummary"`
	CurrentTokenCount        int      `json:"currentTokenCount"`
	MaxTokens                int      `json:"maxTokens"`
	TokenUsageRate           float64  `json:"tokenUsageRate"`
	ContextQualityScore      float64  `json:"contextQualityScore"`
	TimeSinceLastSummary     int64    `json:"timeSinceLastSummary"`
	EstimatedTokenSaving     int      `json:"estimatedTokenSaving"`
	Urgency                  float64  `json:"urgency"`
	RecommendedType          string   `json:"recommendedType"`
	TriggerScore             float64  `json:"triggerScore"`
	CheckTime                int64    `json:"checkTime"`
}

// EvaluateSummaryQualityRequest 评估摘要质量请求
type EvaluateSummaryQualityRequest struct {
	Summary          string   `json:"summary"`
	SummaryID        string   `json:"summaryId"`
	OriginalMessages []string `json:"originalMessages"`
	Dimensions       []string `json:"dimensions"` // completeness, conciseness, coherence, accuracy
}

// EvaluateSummaryQualityResponse 评估摘要质量响应
type EvaluateSummaryQualityResponse struct {
	SummaryID       string             `json:"summaryId"`
	OverallScore    float64            `json:"overallScore"`
	DimensionScores map[string]float64 `json:"dimensionScores"`
	Passed          bool               `json:"passed"`
	Issues          []QualityIssue     `json:"issues"`
	Suggestions     []string           `json:"suggestions"`
	KeyInfoCoverage float64            `json:"keyInfoCoverage"`
	RedundancyScore float64            `json:"redundancyScore"`
	EvaluationTime  int64              `json:"evaluationTime"`
}

// QualityIssue 质量问题
type QualityIssue struct {
	Dimension   string  `json:"dimension"`
	Severity    string  `json:"severity"` // low, medium, high
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Impact      string  `json:"impact"`
}
