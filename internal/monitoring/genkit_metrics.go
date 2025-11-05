package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Genkit Flow 执行指标
var (
	// Flow 执行次数
	flowExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_flow_executions_total",
			Help: "Total number of flow executions",
		},
		[]string{"flow_name", "status", "tenant_id"},
	)

	// Flow 执行时间
	flowDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_flow_duration_seconds",
			Help:    "Flow execution duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
		[]string{"flow_name", "tenant_id"},
	)

	// Flow 错误次数
	flowErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_flow_errors_total",
			Help: "Total number of flow errors",
		},
		[]string{"flow_name", "error_type", "tenant_id"},
	)

	// Token 使用量
	tokenUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_token_usage_total",
			Help: "Total token usage",
		},
		[]string{"tenant_id", "token_type", "flow_name"},
	)

	// Token 使用量（按会话）
	sessionTokenUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "genkit_session_token_usage",
			Help: "Current token usage per session",
		},
		[]string{"session_id", "tenant_id"},
	)

	// 缓存命中率
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_type", "tenant_id"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_type", "tenant_id"},
	)

	// 上下文构建指标
	contextBuildSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_context_build_tokens",
			Help:    "Number of tokens in built context",
			Buckets: []float64{100, 500, 1000, 2000, 4000, 8000, 16000},
		},
		[]string{"session_id", "tenant_id"},
	)

	contextQualityScore = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_context_quality_score",
			Help:    "Context quality score (0-1)",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"session_id", "tenant_id"},
	)

	// 向量检索指标
	vectorSearchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_vector_search_duration_seconds",
			Help:    "Vector search duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0},
		},
		[]string{"tenant_id"},
	)

	vectorSearchResults = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_vector_search_results",
			Help:    "Number of results returned by vector search",
			Buckets: []float64{0, 1, 5, 10, 20, 50},
		},
		[]string{"tenant_id"},
	)

	// 摘要生成指标
	summaryGenerations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_summary_generations_total",
			Help: "Total number of summary generations",
		},
		[]string{"summary_type", "tenant_id"},
	)

	summaryQuality = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_summary_quality_score",
			Help:    "Summary quality score (0-1)",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"summary_type", "tenant_id"},
	)

	summaryCompressionRate = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_summary_compression_rate",
			Help:    "Summary compression rate (0-1)",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"summary_type", "tenant_id"},
	)

	// AI 服务调用指标
	aiServiceCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_ai_service_calls_total",
			Help: "Total number of AI service calls",
		},
		[]string{"provider", "model", "status", "tenant_id"},
	)

	aiServiceDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genkit_ai_service_duration_seconds",
			Help:    "AI service call duration in seconds",
			Buckets: []float64{0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"provider", "model", "tenant_id"},
	)

	// 会话健康度指标
	sessionHealthScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "genkit_session_health_score",
			Help: "Session health score (0-1)",
		},
		[]string{"session_id", "tenant_id"},
	)

	// 活跃会话数
	activeSessions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "genkit_active_sessions",
			Help: "Number of active sessions",
		},
		[]string{"tenant_id"},
	)

	// 记忆存储指标
	memoryStores = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_memory_stores_total",
			Help: "Total number of memory stores",
		},
		[]string{"memory_type", "tenant_id"},
	)

	memoryCleanups = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genkit_memory_cleanups_total",
			Help: "Total number of memory cleanups",
		},
		[]string{"strategy", "mode", "tenant_id"},
	)

	// 系统资源指标
	databaseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "genkit_database_connections",
			Help: "Number of active database connections",
		},
	)

	redisConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "genkit_redis_connections",
			Help: "Number of active Redis connections",
		},
	)
)

// GenkitMetrics Genkit 指标收集器
type GenkitMetrics struct{}

// NewGenkitMetrics 创建 Genkit 指标收集器
func NewGenkitMetrics() *GenkitMetrics {
	return &GenkitMetrics{}
}

// RecordFlowExecution 记录 Flow 执行
func (m *GenkitMetrics) RecordFlowExecution(flowName, status, tenantID string) {
	flowExecutions.WithLabelValues(flowName, status, tenantID).Inc()
}

// RecordFlowDuration 记录 Flow 执行时间
func (m *GenkitMetrics) RecordFlowDuration(flowName, tenantID string, duration time.Duration) {
	flowDuration.WithLabelValues(flowName, tenantID).Observe(duration.Seconds())
}

// RecordFlowError 记录 Flow 错误
func (m *GenkitMetrics) RecordFlowError(flowName, errorType, tenantID string) {
	flowErrors.WithLabelValues(flowName, errorType, tenantID).Inc()
}

// RecordTokenUsage 记录 Token 使用量
func (m *GenkitMetrics) RecordTokenUsage(tenantID, tokenType, flowName string, count int) {
	tokenUsage.WithLabelValues(tenantID, tokenType, flowName).Add(float64(count))
}

// UpdateSessionTokenUsage 更新会话 Token 使用量
func (m *GenkitMetrics) UpdateSessionTokenUsage(sessionID, tenantID string, tokens int) {
	sessionTokenUsage.WithLabelValues(sessionID, tenantID).Set(float64(tokens))
}

// RecordCacheHit 记录缓存命中
func (m *GenkitMetrics) RecordCacheHit(cacheType, tenantID string) {
	cacheHits.WithLabelValues(cacheType, tenantID).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (m *GenkitMetrics) RecordCacheMiss(cacheType, tenantID string) {
	cacheMisses.WithLabelValues(cacheType, tenantID).Inc()
}

// RecordContextBuild 记录上下文构建
func (m *GenkitMetrics) RecordContextBuild(sessionID, tenantID string, tokens int, qualityScore float64) {
	contextBuildSize.WithLabelValues(sessionID, tenantID).Observe(float64(tokens))
	contextQualityScore.WithLabelValues(sessionID, tenantID).Observe(qualityScore)
}

// RecordVectorSearch 记录向量检索
func (m *GenkitMetrics) RecordVectorSearch(tenantID string, duration time.Duration, resultCount int) {
	vectorSearchDuration.WithLabelValues(tenantID).Observe(duration.Seconds())
	vectorSearchResults.WithLabelValues(tenantID).Observe(float64(resultCount))
}

// RecordSummaryGeneration 记录摘要生成
func (m *GenkitMetrics) RecordSummaryGeneration(summaryType, tenantID string, qualityScore, compressionRate float64) {
	summaryGenerations.WithLabelValues(summaryType, tenantID).Inc()
	summaryQuality.WithLabelValues(summaryType, tenantID).Observe(qualityScore)
	summaryCompressionRate.WithLabelValues(summaryType, tenantID).Observe(compressionRate)
}

// RecordAIServiceCall 记录 AI 服务调用
func (m *GenkitMetrics) RecordAIServiceCall(provider, model, status, tenantID string, duration time.Duration) {
	aiServiceCalls.WithLabelValues(provider, model, status, tenantID).Inc()
	aiServiceDuration.WithLabelValues(provider, model, tenantID).Observe(duration.Seconds())
}

// UpdateSessionHealth 更新会话健康度
func (m *GenkitMetrics) UpdateSessionHealth(sessionID, tenantID string, healthScore float64) {
	sessionHealthScore.WithLabelValues(sessionID, tenantID).Set(healthScore)
}

// UpdateActiveSessions 更新活跃会话数
func (m *GenkitMetrics) UpdateActiveSessions(tenantID string, count int) {
	activeSessions.WithLabelValues(tenantID).Set(float64(count))
}

// RecordMemoryStore 记录记忆存储
func (m *GenkitMetrics) RecordMemoryStore(memoryType, tenantID string) {
	memoryStores.WithLabelValues(memoryType, tenantID).Inc()
}

// RecordMemoryCleanup 记录记忆清理
func (m *GenkitMetrics) RecordMemoryCleanup(strategy, mode, tenantID string, count int) {
	memoryCleanups.WithLabelValues(strategy, mode, tenantID).Add(float64(count))
}

// UpdateDatabaseConnections 更新数据库连接数
func (m *GenkitMetrics) UpdateDatabaseConnections(count int) {
	databaseConnections.Set(float64(count))
}

// UpdateRedisConnections 更新 Redis 连接数
func (m *GenkitMetrics) UpdateRedisConnections(count int) {
	redisConnections.Set(float64(count))
}

// GetCacheHitRate 计算缓存命中率
func (m *GenkitMetrics) GetCacheHitRate(cacheType, tenantID string) float64 {
	// 这里需要从 Prometheus 查询实际的指标值
	// 在实际实现中，可以使用 Prometheus 的 API 查询
	// 这里返回一个占位值
	return 0.0
}
