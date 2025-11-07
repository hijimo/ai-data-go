package monitoring

import (
	"sync"
	"time"
)

// Metrics 性能监控指标收集器
type Metrics struct {
	mu sync.RWMutex

	// 认证相关指标
	loginAttempts       int64 // 登录尝试总数
	loginSuccesses      int64 // 登录成功总数
	loginFailures       int64 // 登录失败总数
	tokenRefreshes      int64 // Token 刷新总数
	tokenRefreshFailures int64 // Token 刷新失败总数
	logouts             int64 // 注销总数
	
	// 性能指标
	loginDurations      []time.Duration // 登录响应时间
	refreshDurations    []time.Duration // 刷新响应时间
	
	// 数据库指标
	slowQueries         int64 // 慢查询总数
	dbErrors            int64 // 数据库错误总数
	
	// 安全指标
	invalidTokens       int64 // 无效 Token 总数
	expiredTokens       int64 // 过期 Token 总数
	revokedTokens       int64 // 已撤销 Token 总数
	bruteForceAttempts  int64 // 暴力破解尝试总数
	
	// 租户指标
	activeTenants       int64 // 活跃租户数
	activeUsers         int64 // 活跃用户数
	
	// Flow 执行指标
	flowExecutions      map[string]int64        // Flow 执行次数（按 Flow 名称）
	flowSuccesses       map[string]int64        // Flow 成功次数
	flowErrors          map[string]int64        // Flow 错误次数
	flowDurations       map[string][]time.Duration // Flow 执行时间
	
	// Token 使用量指标
	tokenUsage          map[string]int64        // Token 使用量（按租户ID）
	promptTokens        int64                   // Prompt Token 总数
	completionTokens    int64                   // Completion Token 总数
	totalTokens         int64                   // 总 Token 数
	
	// 缓存指标
	cacheHits           int64                   // 缓存命中次数
	cacheMisses         int64                   // 缓存未命中次数
	cacheHitsByKey      map[string]int64        // 按缓存键的命中次数
	cacheMissesByKey    map[string]int64        // 按缓存键的未命中次数
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	// 认证指标
	LoginAttempts        int64   `json:"login_attempts"`
	LoginSuccesses       int64   `json:"login_successes"`
	LoginFailures        int64   `json:"login_failures"`
	LoginSuccessRate     float64 `json:"login_success_rate"`
	TokenRefreshes       int64   `json:"token_refreshes"`
	TokenRefreshFailures int64   `json:"token_refresh_failures"`
	Logouts              int64   `json:"logouts"`
	
	// 性能指标
	AvgLoginDuration    float64 `json:"avg_login_duration_ms"`
	P95LoginDuration    float64 `json:"p95_login_duration_ms"`
	AvgRefreshDuration  float64 `json:"avg_refresh_duration_ms"`
	P95RefreshDuration  float64 `json:"p95_refresh_duration_ms"`
	
	// 数据库指标
	SlowQueries         int64   `json:"slow_queries"`
	DBErrors            int64   `json:"db_errors"`
	
	// 安全指标
	InvalidTokens       int64   `json:"invalid_tokens"`
	ExpiredTokens       int64   `json:"expired_tokens"`
	RevokedTokens       int64   `json:"revoked_tokens"`
	BruteForceAttempts  int64   `json:"brute_force_attempts"`
	
	// 租户指标
	ActiveTenants       int64   `json:"active_tenants"`
	ActiveUsers         int64   `json:"active_users"`
	
	Timestamp           time.Time `json:"timestamp"`
}

var (
	globalMetrics *Metrics
	once          sync.Once
)

// GetMetrics 获取全局 Metrics 实例（单例模式）
func GetMetrics() *Metrics {
	once.Do(func() {
		globalMetrics = &Metrics{
			loginDurations:   make([]time.Duration, 0, 1000),
			refreshDurations: make([]time.Duration, 0, 1000),
			flowExecutions:   make(map[string]int64),
			flowSuccesses:    make(map[string]int64),
			flowErrors:       make(map[string]int64),
			flowDurations:    make(map[string][]time.Duration),
			tokenUsage:       make(map[string]int64),
			cacheHitsByKey:   make(map[string]int64),
			cacheMissesByKey: make(map[string]int64),
		}
	})
	return globalMetrics
}

// RecordLoginAttempt 记录登录尝试
func (m *Metrics) RecordLoginAttempt(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.loginAttempts++
	if success {
		m.loginSuccesses++
	} else {
		m.loginFailures++
	}
	
	// 记录响应时间（保留最近1000条）
	if len(m.loginDurations) >= 1000 {
		m.loginDurations = m.loginDurations[1:]
	}
	m.loginDurations = append(m.loginDurations, duration)
}

// RecordTokenRefresh 记录 Token 刷新
func (m *Metrics) RecordTokenRefresh(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.tokenRefreshes++
	if !success {
		m.tokenRefreshFailures++
	}
	
	// 记录响应时间（保留最近1000条）
	if len(m.refreshDurations) >= 1000 {
		m.refreshDurations = m.refreshDurations[1:]
	}
	m.refreshDurations = append(m.refreshDurations, duration)
}

// RecordLogout 记录注销
func (m *Metrics) RecordLogout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logouts++
}

// RecordSlowQuery 记录慢查询
func (m *Metrics) RecordSlowQuery() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slowQueries++
}

// RecordDBError 记录数据库错误
func (m *Metrics) RecordDBError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbErrors++
}

// RecordInvalidToken 记录无效 Token
func (m *Metrics) RecordInvalidToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidTokens++
}

// RecordExpiredToken 记录过期 Token
func (m *Metrics) RecordExpiredToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expiredTokens++
}

// RecordRevokedToken 记录已撤销 Token
func (m *Metrics) RecordRevokedToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedTokens++
}

// RecordBruteForceAttempt 记录暴力破解尝试
func (m *Metrics) RecordBruteForceAttempt() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bruteForceAttempts++
}

// UpdateActiveTenants 更新活跃租户数
func (m *Metrics) UpdateActiveTenants(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeTenants = count
}

// UpdateActiveUsers 更新活跃用户数
func (m *Metrics) UpdateActiveUsers(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeUsers = count
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	snapshot := MetricsSnapshot{
		LoginAttempts:        m.loginAttempts,
		LoginSuccesses:       m.loginSuccesses,
		LoginFailures:        m.loginFailures,
		TokenRefreshes:       m.tokenRefreshes,
		TokenRefreshFailures: m.tokenRefreshFailures,
		Logouts:              m.logouts,
		SlowQueries:          m.slowQueries,
		DBErrors:             m.dbErrors,
		InvalidTokens:        m.invalidTokens,
		ExpiredTokens:        m.expiredTokens,
		RevokedTokens:        m.revokedTokens,
		BruteForceAttempts:   m.bruteForceAttempts,
		ActiveTenants:        m.activeTenants,
		ActiveUsers:          m.activeUsers,
		Timestamp:            time.Now(),
	}
	
	// 计算登录成功率
	if m.loginAttempts > 0 {
		snapshot.LoginSuccessRate = float64(m.loginSuccesses) / float64(m.loginAttempts) * 100
	}
	
	// 计算平均登录时间
	if len(m.loginDurations) > 0 {
		var total time.Duration
		for _, d := range m.loginDurations {
			total += d
		}
		snapshot.AvgLoginDuration = float64(total.Milliseconds()) / float64(len(m.loginDurations))
		snapshot.P95LoginDuration = calculateP95(m.loginDurations)
	}
	
	// 计算平均刷新时间
	if len(m.refreshDurations) > 0 {
		var total time.Duration
		for _, d := range m.refreshDurations {
			total += d
		}
		snapshot.AvgRefreshDuration = float64(total.Milliseconds()) / float64(len(m.refreshDurations))
		snapshot.P95RefreshDuration = calculateP95(m.refreshDurations)
	}
	
	return snapshot
}

// RecordFlowExecution 记录 Flow 执行
// flowName: Flow 名称
// status: 执行状态（"success" 或 "error"）
func (m *Metrics) RecordFlowExecution(flowName string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.flowExecutions[flowName]++
	if status == "success" {
		m.flowSuccesses[flowName]++
	} else {
		m.flowErrors[flowName]++
	}
}

// RecordFlowDuration 记录 Flow 执行时间
// flowName: Flow 名称
// duration: 执行时长
func (m *Metrics) RecordFlowDuration(flowName string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.flowDurations[flowName]; !exists {
		m.flowDurations[flowName] = make([]time.Duration, 0, 1000)
	}
	
	// 保留最近1000条
	if len(m.flowDurations[flowName]) >= 1000 {
		m.flowDurations[flowName] = m.flowDurations[flowName][1:]
	}
	m.flowDurations[flowName] = append(m.flowDurations[flowName], duration)
}

// RecordTokenUsage 记录 Token 使用量
// tenantID: 租户ID
// promptTokens: Prompt Token 数量
// completionTokens: Completion Token 数量
func (m *Metrics) RecordTokenUsage(tenantID string, promptTokens, completionTokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	totalTokens := promptTokens + completionTokens
	
	// 按租户记录
	m.tokenUsage[tenantID] += int64(totalTokens)
	
	// 全局统计
	m.promptTokens += int64(promptTokens)
	m.completionTokens += int64(completionTokens)
	m.totalTokens += int64(totalTokens)
}

// RecordCacheHit 记录缓存命中
// cacheKey: 缓存键（可选，用于详细统计）
func (m *Metrics) RecordCacheHit(cacheKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cacheHits++
	if cacheKey != "" {
		m.cacheHitsByKey[cacheKey]++
	}
}

// RecordCacheMiss 记录缓存未命中
// cacheKey: 缓存键（可选，用于详细统计）
func (m *Metrics) RecordCacheMiss(cacheKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cacheMisses++
	if cacheKey != "" {
		m.cacheMissesByKey[cacheKey]++
	}
}

// GetFlowMetrics 获取指定 Flow 的指标
func (m *Metrics) GetFlowMetrics(flowName string) FlowMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := FlowMetrics{
		FlowName:   flowName,
		Executions: m.flowExecutions[flowName],
		Successes:  m.flowSuccesses[flowName],
		Errors:     m.flowErrors[flowName],
	}
	
	// 计算成功率
	if metrics.Executions > 0 {
		metrics.SuccessRate = float64(metrics.Successes) / float64(metrics.Executions) * 100
	}
	
	// 计算执行时间统计
	if durations, exists := m.flowDurations[flowName]; exists && len(durations) > 0 {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		metrics.AvgDuration = float64(total.Milliseconds()) / float64(len(durations))
		metrics.P95Duration = calculateP95(durations)
		metrics.P99Duration = calculateP99(durations)
	}
	
	return metrics
}

// GetTokenMetrics 获取 Token 使用指标
func (m *Metrics) GetTokenMetrics() TokenMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return TokenMetrics{
		PromptTokens:     m.promptTokens,
		CompletionTokens: m.completionTokens,
		TotalTokens:      m.totalTokens,
		ByTenant:         copyInt64Map(m.tokenUsage),
	}
}

// GetCacheMetrics 获取缓存指标
func (m *Metrics) GetCacheMetrics() CacheMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := CacheMetrics{
		Hits:   m.cacheHits,
		Misses: m.cacheMisses,
	}
	
	// 计算命中率
	total := m.cacheHits + m.cacheMisses
	if total > 0 {
		metrics.HitRate = float64(m.cacheHits) / float64(total) * 100
	}
	
	return metrics
}

// Reset 重置所有指标（用于测试或定期重置）
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.loginAttempts = 0
	m.loginSuccesses = 0
	m.loginFailures = 0
	m.tokenRefreshes = 0
	m.tokenRefreshFailures = 0
	m.logouts = 0
	m.slowQueries = 0
	m.dbErrors = 0
	m.invalidTokens = 0
	m.expiredTokens = 0
	m.revokedTokens = 0
	m.bruteForceAttempts = 0
	m.loginDurations = make([]time.Duration, 0, 1000)
	m.refreshDurations = make([]time.Duration, 0, 1000)
	m.flowExecutions = make(map[string]int64)
	m.flowSuccesses = make(map[string]int64)
	m.flowErrors = make(map[string]int64)
	m.flowDurations = make(map[string][]time.Duration)
	m.tokenUsage = make(map[string]int64)
	m.promptTokens = 0
	m.completionTokens = 0
	m.totalTokens = 0
	m.cacheHits = 0
	m.cacheMisses = 0
	m.cacheHitsByKey = make(map[string]int64)
	m.cacheMissesByKey = make(map[string]int64)
}

// FlowMetrics Flow 指标
type FlowMetrics struct {
	FlowName    string  `json:"flowName"`
	Executions  int64   `json:"executions"`
	Successes   int64   `json:"successes"`
	Errors      int64   `json:"errors"`
	SuccessRate float64 `json:"successRate"`
	AvgDuration float64 `json:"avgDuration"`
	P95Duration float64 `json:"p95Duration"`
	P99Duration float64 `json:"p99Duration"`
}

// TokenMetrics Token 使用指标
type TokenMetrics struct {
	PromptTokens     int64            `json:"promptTokens"`
	CompletionTokens int64            `json:"completionTokens"`
	TotalTokens      int64            `json:"totalTokens"`
	ByTenant         map[string]int64 `json:"byTenant"`
}

// CacheMetrics 缓存指标
type CacheMetrics struct {
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hitRate"`
}

// calculateP95 计算 P95 百分位数
func calculateP95(durations []time.Duration) float64 {
	return calculatePercentile(durations, 0.95)
}

// calculateP99 计算 P99 百分位数
func calculateP99(durations []time.Duration) float64 {
	return calculatePercentile(durations, 0.99)
}

// calculatePercentile 计算指定百分位数
func calculatePercentile(durations []time.Duration, percentile float64) float64 {
	if len(durations) == 0 {
		return 0
	}
	
	// 复制并排序
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	// 简单冒泡排序（因为数据量不大）
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// 计算百分位位置
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return float64(sorted[index].Milliseconds())
}

// copyInt64Map 复制 int64 map（用于线程安全）
func copyInt64Map(src map[string]int64) map[string]int64 {
	dst := make(map[string]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// 全局便捷函数

// RecordFlowExecution 全局函数：记录 Flow 执行
func RecordFlowExecution(flowName string, status string) {
	GetMetrics().RecordFlowExecution(flowName, status)
}

// RecordFlowDuration 全局函数：记录 Flow 执行时间
func RecordFlowDuration(flowName string, duration time.Duration) {
	GetMetrics().RecordFlowDuration(flowName, duration)
}

// RecordTokenUsage 全局函数：记录 Token 使用量
func RecordTokenUsage(tenantID string, promptTokens, completionTokens int) {
	GetMetrics().RecordTokenUsage(tenantID, promptTokens, completionTokens)
}

// RecordCacheHit 全局函数：记录缓存命中
func RecordCacheHit(cacheKey string) {
	GetMetrics().RecordCacheHit(cacheKey)
}

// RecordCacheMiss 全局函数：记录缓存未命中
func RecordCacheMiss(cacheKey string) {
	GetMetrics().RecordCacheMiss(cacheKey)
}
