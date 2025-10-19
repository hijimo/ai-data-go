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
}

// calculateP95 计算 P95 百分位数
func calculateP95(durations []time.Duration) float64 {
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
	
	// 计算 P95 位置
	p95Index := int(float64(len(sorted)) * 0.95)
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	
	return float64(sorted[p95Index].Milliseconds())
}
