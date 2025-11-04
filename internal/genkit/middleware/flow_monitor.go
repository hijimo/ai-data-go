package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/monitoring"
)

// FlowMonitor Flow 监控中间件
type FlowMonitor struct {
	metrics *monitoring.GenkitMetrics
}

// NewFlowMonitor 创建 Flow 监控中间件
func NewFlowMonitor(metrics *monitoring.GenkitMetrics) *FlowMonitor {
	return &FlowMonitor{
		metrics: metrics,
	}
}

// MonitorFlow 监控 Flow 执行
// 返回一个包装函数，用于监控 Flow 的执行
func (m *FlowMonitor) MonitorFlow(
	flowName string,
	fn func(context.Context) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		startTime := time.Now()

		// 从上下文获取租户 ID
		tenantID := getTenantIDFromContext(ctx)

		// 记录 Flow 开始执行
		logger.InfoContext(ctx, "Flow 开始执行", logger.Fields{
			"flow":      flowName,
			"tenant_id": tenantID,
		})

		// 执行 Flow
		err := fn(ctx)

		// 计算执行时间
		duration := time.Since(startTime)

		// 记录执行时间
		m.metrics.RecordFlowDuration(flowName, tenantID, duration)

		// 记录执行结果
		status := "success"
		if err != nil {
			status = "error"
			errorType := getErrorType(err)
			m.metrics.RecordFlowError(flowName, errorType, tenantID)

			logger.ErrorContext(ctx, "Flow 执行失败", logger.Fields{
				"flow":        flowName,
				"tenant_id":   tenantID,
				"duration_ms": duration.Milliseconds(),
				"error":       err.Error(),
				"error_type":  errorType,
			})
		} else {
			logger.InfoContext(ctx, "Flow 执行成功", logger.Fields{
				"flow":        flowName,
				"tenant_id":   tenantID,
				"duration_ms": duration.Milliseconds(),
			})
		}

		m.metrics.RecordFlowExecution(flowName, status, tenantID)

		return err
	}
}

// MonitorFlowWithMetrics 监控 Flow 执行并记录额外指标
// 这个版本允许在 Flow 执行后记录额外的业务指标
func (m *FlowMonitor) MonitorFlowWithMetrics(
	flowName string,
	fn func(context.Context) (interface{}, error),
	metricsRecorder func(context.Context, interface{}, error),
) func(context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
		startTime := time.Now()

		// 从上下文获取租户 ID
		tenantID := getTenantIDFromContext(ctx)

		// 记录 Flow 开始执行
		logger.InfoContext(ctx, "Flow 开始执行", logger.Fields{
			"flow":      flowName,
			"tenant_id": tenantID,
		})

		// 执行 Flow
		result, err := fn(ctx)

		// 计算执行时间
		duration := time.Since(startTime)

		// 记录执行时间
		m.metrics.RecordFlowDuration(flowName, tenantID, duration)

		// 记录执行结果
		status := "success"
		if err != nil {
			status = "error"
			errorType := getErrorType(err)
			m.metrics.RecordFlowError(flowName, errorType, tenantID)

			logger.ErrorContext(ctx, "Flow 执行失败", logger.Fields{
				"flow":        flowName,
				"tenant_id":   tenantID,
				"duration_ms": duration.Milliseconds(),
				"error":       err.Error(),
				"error_type":  errorType,
			})
		} else {
			logger.InfoContext(ctx, "Flow 执行成功", logger.Fields{
				"flow":        flowName,
				"tenant_id":   tenantID,
				"duration_ms": duration.Milliseconds(),
			})
		}

		m.metrics.RecordFlowExecution(flowName, status, tenantID)

		// 记录额外的业务指标
		if metricsRecorder != nil {
			metricsRecorder(ctx, result, err)
		}

		return result, err
	}
}

// getTenantIDFromContext 从上下文获取租户 ID
func getTenantIDFromContext(ctx context.Context) string {
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		if id, ok := tenantID.(string); ok {
			return id
		}
	}
	return "unknown"
}

// getErrorType 获取错误类型
func getErrorType(err error) string {
	if err == nil {
		return ""
	}

	// 根据错误类型返回分类
	errStr := err.Error()

	// 超时错误
	if contains(errStr, "timeout") || contains(errStr, "deadline exceeded") {
		return "timeout"
	}

	// 权限错误
	if contains(errStr, "forbidden") || contains(errStr, "unauthorized") || contains(errStr, "permission denied") {
		return "permission"
	}

	// 验证错误
	if contains(errStr, "validation") || contains(errStr, "invalid") {
		return "validation"
	}

	// 资源不存在
	if contains(errStr, "not found") {
		return "not_found"
	}

	// AI 服务错误
	if contains(errStr, "ai service") || contains(errStr, "genkit") {
		return "ai_service"
	}

	// 数据库错误
	if contains(errStr, "database") || contains(errStr, "sql") {
		return "database"
	}

	// 缓存错误
	if contains(errStr, "cache") || contains(errStr, "redis") {
		return "cache"
	}

	// 向量服务错误
	if contains(errStr, "vector") || contains(errStr, "embedding") {
		return "vector_service"
	}

	// 配额超限
	if contains(errStr, "quota") || contains(errStr, "limit exceeded") {
		return "quota_exceeded"
	}

	// 其他错误
	return "unknown"
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// FlowMetricsRecorder Flow 指标记录器接口
type FlowMetricsRecorder interface {
	RecordMetrics(ctx context.Context, result interface{}, err error)
}

// ContextBuildMetricsRecorder 上下文构建指标记录器
type ContextBuildMetricsRecorder struct {
	metrics *monitoring.GenkitMetrics
}

// NewContextBuildMetricsRecorder 创建上下文构建指标记录器
func NewContextBuildMetricsRecorder(metrics *monitoring.GenkitMetrics) *ContextBuildMetricsRecorder {
	return &ContextBuildMetricsRecorder{metrics: metrics}
}

// RecordMetrics 记录上下文构建指标
func (r *ContextBuildMetricsRecorder) RecordMetrics(ctx context.Context, result interface{}, err error) {
	if err != nil {
		return
	}

	// 类型断言获取上下文构建结果
	// 这里需要根据实际的结果类型进行调整
	if contextResult, ok := result.(map[string]interface{}); ok {
		sessionID := getStringValue(contextResult, "session_id")
		tenantID := getTenantIDFromContext(ctx)
		tokens := getIntValue(contextResult, "total_tokens")
		qualityScore := getFloatValue(contextResult, "quality_score")

		r.metrics.RecordContextBuild(sessionID, tenantID, tokens, qualityScore)
	}
}

// TokenUsageMetricsRecorder Token 使用指标记录器
type TokenUsageMetricsRecorder struct {
	metrics  *monitoring.GenkitMetrics
	flowName string
}

// NewTokenUsageMetricsRecorder 创建 Token 使用指标记录器
func NewTokenUsageMetricsRecorder(metrics *monitoring.GenkitMetrics, flowName string) *TokenUsageMetricsRecorder {
	return &TokenUsageMetricsRecorder{
		metrics:  metrics,
		flowName: flowName,
	}
}

// RecordMetrics 记录 Token 使用指标
func (r *TokenUsageMetricsRecorder) RecordMetrics(ctx context.Context, result interface{}, err error) {
	if err != nil {
		return
	}

	tenantID := getTenantIDFromContext(ctx)

	// 类型断言获取 Token 使用信息
	if tokenResult, ok := result.(map[string]interface{}); ok {
		if tokenUsage, ok := tokenResult["token_usage"].(map[string]interface{}); ok {
			promptTokens := getIntValue(tokenUsage, "prompt_tokens")
			completionTokens := getIntValue(tokenUsage, "completion_tokens")
			totalTokens := getIntValue(tokenUsage, "total_tokens")

			r.metrics.RecordTokenUsage(tenantID, "prompt", r.flowName, promptTokens)
			r.metrics.RecordTokenUsage(tenantID, "completion", r.flowName, completionTokens)
			r.metrics.RecordTokenUsage(tenantID, "total", r.flowName, totalTokens)
		}
	}
}

// VectorSearchMetricsRecorder 向量检索指标记录器
type VectorSearchMetricsRecorder struct {
	metrics   *monitoring.GenkitMetrics
	startTime time.Time
}

// NewVectorSearchMetricsRecorder 创建向量检索指标记录器
func NewVectorSearchMetricsRecorder(metrics *monitoring.GenkitMetrics) *VectorSearchMetricsRecorder {
	return &VectorSearchMetricsRecorder{
		metrics:   metrics,
		startTime: time.Now(),
	}
}

// RecordMetrics 记录向量检索指标
func (r *VectorSearchMetricsRecorder) RecordMetrics(ctx context.Context, result interface{}, err error) {
	if err != nil {
		return
	}

	tenantID := getTenantIDFromContext(ctx)
	duration := time.Since(r.startTime)

	// 类型断言获取检索结果
	if searchResult, ok := result.(map[string]interface{}); ok {
		resultCount := getIntValue(searchResult, "returned_count")
		r.metrics.RecordVectorSearch(tenantID, duration, resultCount)
	}
}

// 辅助函数
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}

func getFloatValue(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0.0
}

// MonitoringContext 监控上下文
type MonitoringContext struct {
	FlowName  string
	StartTime time.Time
	TenantID  string
	SessionID string
	Metadata  map[string]interface{}
}

// NewMonitoringContext 创建监控上下文
func NewMonitoringContext(ctx context.Context, flowName string) *MonitoringContext {
	return &MonitoringContext{
		FlowName:  flowName,
		StartTime: time.Now(),
		TenantID:  getTenantIDFromContext(ctx),
		SessionID: getSessionIDFromContext(ctx),
		Metadata:  make(map[string]interface{}),
	}
}

// getSessionIDFromContext 从上下文获取会话 ID
func getSessionIDFromContext(ctx context.Context) string {
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			return id
		}
	}
	return ""
}

// AddMetadata 添加元数据
func (mc *MonitoringContext) AddMetadata(key string, value interface{}) {
	mc.Metadata[key] = value
}

// GetDuration 获取执行时间
func (mc *MonitoringContext) GetDuration() time.Duration {
	return time.Since(mc.StartTime)
}

// LogCompletion 记录完成日志
func (mc *MonitoringContext) LogCompletion(ctx context.Context, err error) {
	duration := mc.GetDuration()

	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("%s 执行失败", mc.FlowName), logger.Fields{
			"flow":        mc.FlowName,
			"tenant_id":   mc.TenantID,
			"session_id":  mc.SessionID,
			"duration_ms": duration.Milliseconds(),
			"error":       err.Error(),
			"metadata":    mc.Metadata,
		})
	} else {
		logger.InfoContext(ctx, fmt.Sprintf("%s 执行成功", mc.FlowName), logger.Fields{
			"flow":        mc.FlowName,
			"tenant_id":   mc.TenantID,
			"session_id":  mc.SessionID,
			"duration_ms": duration.Milliseconds(),
			"metadata":    mc.Metadata,
		})
	}
}
