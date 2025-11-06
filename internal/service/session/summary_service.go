package session

import (
	"context"

	"github.com/google/uuid"

	"genkit-ai-service/internal/model"
)

// SummaryService 摘要服务接口
// 负责生成、评估和管理会话摘要
type SummaryService interface {
	// GenerateSummary 生成摘要
	// 参数：
	//   - ctx: 上下文
	//   - req: 生成摘要请求
	// 返回：
	//   - *model.ConversationSummary: 生成的摘要
	//   - error: 错误信息
	GenerateSummary(ctx context.Context, req *GenerateSummaryRequest) (*model.ConversationSummary, error)

	// CheckSummaryTrigger 检查是否需要生成摘要
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - sessionID: 会话ID
	// 返回：
	//   - *SummaryTriggerResult: 触发检查结果
	//   - error: 错误信息
	CheckSummaryTrigger(ctx context.Context, tenantID, sessionID uuid.UUID) (*SummaryTriggerResult, error)

	// EvaluateSummaryQuality 评估摘要质量
	// 参数：
	//   - ctx: 上下文
	//   - req: 评估请求
	// 返回：
	//   - *SummaryQualityResult: 质量评估结果
	//   - error: 错误信息
	EvaluateSummaryQuality(ctx context.Context, req *EvaluateSummaryRequest) (*SummaryQualityResult, error)

	// GetSummary 获取摘要详情
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - summaryID: 摘要ID
	// 返回：
	//   - *model.ConversationSummary: 摘要详情
	//   - error: 错误信息
	GetSummary(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error)

	// ListSummaries 获取会话摘要列表
	// 参数：
	//   - ctx: 上下文
	//   - tenantID: 租户ID
	//   - sessionID: 会话ID
	//   - limit: 限制数量（0表示不限制）
	// 返回：
	//   - []*model.ConversationSummary: 摘要列表
	//   - error: 错误信息
	ListSummaries(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error)
}

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	// 租户ID
	TenantID uuid.UUID
	// 会话ID
	SessionID uuid.UUID
	// 消息ID列表（可选，如果不提供则使用StartMessageID和EndMessageID范围）
	MessageIDs []uuid.UUID
	// 起始消息ID（可选）
	StartMessageID *uuid.UUID
	// 结束消息ID（可选）
	EndMessageID *uuid.UUID
	// 前一个摘要内容（用于增量摘要）
	PreviousSummary string
	// 摘要类型 (incremental, full)
	SummaryType string
	// 目标长度（Token数量）
	TargetLength int
}

// SummaryTriggerResult 摘要触发检查结果
type SummaryTriggerResult struct {
	// 是否应该生成摘要
	ShouldSummarize bool
	// 触发原因
	TriggerReason string
	// 建议包含的消息ID列表
	MessageIDs []uuid.UUID
	// 消息数量
	MessageCount int
	// 估算的Token节省量
	EstimatedTokenSaving int
	// 紧急程度 (0-1)
	Urgency float64
	// 推荐的摘要类型
	RecommendedType string
}

// EvaluateSummaryRequest 评估摘要请求
type EvaluateSummaryRequest struct {
	// 摘要内容
	Summary string
	// 原始消息列表
	OriginalMessages []*model.ChatMessage
	// 评估维度（可选，如果不提供则评估所有维度）
	Dimensions []string
}

// SummaryQualityResult 摘要质量评估结果
type SummaryQualityResult struct {
	// 总体评分 (0-1)
	OverallScore float64
	// 各维度评分
	DimensionScores map[string]float64
	// 是否通过质量检查
	Passed bool
	// 质量问题列表
	Issues []QualityIssue
	// 改进建议
	Suggestions []string
	// 关键信息覆盖率 (0-1)
	KeyInfoCoverage float64
}

// QualityIssue 质量问题
type QualityIssue struct {
	// 维度名称
	Dimension string
	// 严重程度 (low, medium, high)
	Severity string
	// 问题描述
	Description string
	// 评分
	Score float64
}
