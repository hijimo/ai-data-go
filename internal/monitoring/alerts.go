package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert 告警信息
type Alert struct {
	Level       AlertLevel `json:"level"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	Metric      string     `json:"metric"`
	Value       float64    `json:"value"`
	Threshold   float64    `json:"threshold"`
	Timestamp   time.Time  `json:"timestamp"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name        string
	Description string
	Metric      string
	Threshold   float64
	Level       AlertLevel
	Condition   func(snapshot MetricsSnapshot) (bool, float64) // 返回是否触发和当前值
}

// AlertManager 告警管理器
type AlertManager struct {
	mu            sync.RWMutex
	rules         []AlertRule
	activeAlerts  []Alert
	alertHandlers []AlertHandler
	checkInterval time.Duration
	stopChan      chan struct{}
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	HandleAlert(alert Alert) error
}

// LogAlertHandler 日志告警处理器
type LogAlertHandler struct {
	logger interface {
		Warn(format string, args ...interface{})
		Error(format string, args ...interface{})
	}
}

// HandleAlert 处理告警
func (h *LogAlertHandler) HandleAlert(alert Alert) error {
	msg := fmt.Sprintf("[%s] %s: %s (value: %.2f, threshold: %.2f)",
		alert.Level, alert.Title, alert.Message, alert.Value, alert.Threshold)
	
	switch alert.Level {
	case AlertLevelCritical:
		h.logger.Error(msg)
	case AlertLevelWarning:
		h.logger.Warn(msg)
	default:
		h.logger.Warn(msg)
	}
	
	return nil
}

// NewAlertManager 创建告警管理器
func NewAlertManager(checkInterval time.Duration) *AlertManager {
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute
	}
	
	return &AlertManager{
		rules:         make([]AlertRule, 0),
		activeAlerts:  make([]Alert, 0),
		alertHandlers: make([]AlertHandler, 0),
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// AddHandler 添加告警处理器
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.alertHandlers = append(am.alertHandlers, handler)
}

// Start 启动告警检查
func (am *AlertManager) Start() {
	go am.checkLoop()
}

// Stop 停止告警检查
func (am *AlertManager) Stop() {
	close(am.stopChan)
}

// checkLoop 告警检查循环
func (am *AlertManager) checkLoop() {
	ticker := time.NewTicker(am.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			am.checkAlerts()
		case <-am.stopChan:
			return
		}
	}
}

// checkAlerts 检查告警
func (am *AlertManager) checkAlerts() {
	snapshot := GetMetrics().GetSnapshot()
	
	am.mu.Lock()
	defer am.mu.Unlock()
	
	for _, rule := range am.rules {
		triggered, value := rule.Condition(snapshot)
		if triggered {
			alert := Alert{
				Level:     rule.Level,
				Title:     rule.Name,
				Message:   rule.Description,
				Metric:    rule.Metric,
				Value:     value,
				Threshold: rule.Threshold,
				Timestamp: time.Now(),
			}
			
			// 添加到活跃告警
			am.activeAlerts = append(am.activeAlerts, alert)
			
			// 触发告警处理器
			for _, handler := range am.alertHandlers {
				go handler.HandleAlert(alert)
			}
		}
	}
	
	// 清理旧告警（保留最近100条）
	if len(am.activeAlerts) > 100 {
		am.activeAlerts = am.activeAlerts[len(am.activeAlerts)-100:]
	}
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	// 返回副本
	alerts := make([]Alert, len(am.activeAlerts))
	copy(alerts, am.activeAlerts)
	return alerts
}

// ClearAlerts 清空告警
func (am *AlertManager) ClearAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.activeAlerts = make([]Alert, 0)
}

// GetDefaultAlertRules 获取默认告警规则
func GetDefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:        "高登录失败率",
			Description: "登录失败率超过30%",
			Metric:      "login_success_rate",
			Threshold:   70.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.LoginAttempts > 10 && s.LoginSuccessRate < 70.0 {
					return true, s.LoginSuccessRate
				}
				return false, s.LoginSuccessRate
			},
		},
		{
			Name:        "大量登录失败",
			Description: "登录失败次数过多，可能存在暴力破解",
			Metric:      "login_failures",
			Threshold:   50.0,
			Level:       AlertLevelCritical,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.LoginFailures > 50 {
					return true, float64(s.LoginFailures)
				}
				return false, float64(s.LoginFailures)
			},
		},
		{
			Name:        "Token 刷新失败率高",
			Description: "Token 刷新失败率超过10%",
			Metric:      "token_refresh_failures",
			Threshold:   10.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.TokenRefreshes > 0 {
					failureRate := float64(s.TokenRefreshFailures) / float64(s.TokenRefreshes) * 100
					if failureRate > 10.0 {
						return true, failureRate
					}
					return false, failureRate
				}
				return false, 0
			},
		},
		{
			Name:        "慢查询过多",
			Description: "慢查询数量超过阈值",
			Metric:      "slow_queries",
			Threshold:   20.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.SlowQueries > 20 {
					return true, float64(s.SlowQueries)
				}
				return false, float64(s.SlowQueries)
			},
		},
		{
			Name:        "数据库错误频繁",
			Description: "数据库错误次数过多",
			Metric:      "db_errors",
			Threshold:   10.0,
			Level:       AlertLevelCritical,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.DBErrors > 10 {
					return true, float64(s.DBErrors)
				}
				return false, float64(s.DBErrors)
			},
		},
		{
			Name:        "无效 Token 过多",
			Description: "无效 Token 数量异常，可能存在攻击",
			Metric:      "invalid_tokens",
			Threshold:   30.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.InvalidTokens > 30 {
					return true, float64(s.InvalidTokens)
				}
				return false, float64(s.InvalidTokens)
			},
		},
		{
			Name:        "暴力破解攻击",
			Description: "检测到大量暴力破解尝试",
			Metric:      "brute_force_attempts",
			Threshold:   20.0,
			Level:       AlertLevelCritical,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.BruteForceAttempts > 20 {
					return true, float64(s.BruteForceAttempts)
				}
				return false, float64(s.BruteForceAttempts)
			},
		},
		{
			Name:        "登录响应时间过长",
			Description: "P95 登录响应时间超过1秒",
			Metric:      "p95_login_duration",
			Threshold:   1000.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.P95LoginDuration > 1000.0 {
					return true, s.P95LoginDuration
				}
				return false, s.P95LoginDuration
			},
		},
		{
			Name:        "Token 刷新响应时间过长",
			Description: "P95 Token 刷新响应时间超过500ms",
			Metric:      "p95_refresh_duration",
			Threshold:   500.0,
			Level:       AlertLevelWarning,
			Condition: func(s MetricsSnapshot) (bool, float64) {
				if s.P95RefreshDuration > 500.0 {
					return true, s.P95RefreshDuration
				}
				return false, s.P95RefreshDuration
			},
		},
	}
}
