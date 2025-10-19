package monitoring

import (
	"testing"
	"time"
)

func TestAlertManager_AddRule(t *testing.T) {
	am := NewAlertManager(1 * time.Second)
	
	rule := AlertRule{
		Name:        "测试规则",
		Description: "测试描述",
		Metric:      "test_metric",
		Threshold:   100.0,
		Level:       AlertLevelWarning,
		Condition: func(s MetricsSnapshot) (bool, float64) {
			return false, 0
		},
	}
	
	am.AddRule(rule)
	
	if len(am.rules) != 1 {
		t.Errorf("期望规则数量为 1，实际为 %d", len(am.rules))
	}
}

func TestAlertManager_CheckAlerts(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	am := NewAlertManager(1 * time.Second)
	
	// 添加一个会触发的规则
	rule := AlertRule{
		Name:        "高登录失败率",
		Description: "登录失败率过高",
		Metric:      "login_failures",
		Threshold:   50.0,
		Level:       AlertLevelWarning,
		Condition: func(s MetricsSnapshot) (bool, float64) {
			if s.LoginFailures > 50 {
				return true, float64(s.LoginFailures)
			}
			return false, float64(s.LoginFailures)
		},
	}
	
	am.AddRule(rule)
	
	// 记录足够多的失败登录以触发告警
	for i := 0; i < 60; i++ {
		metrics.RecordLoginAttempt(false, 100*time.Millisecond)
	}
	
	// 手动触发检查
	am.checkAlerts()
	
	alerts := am.GetActiveAlerts()
	
	if len(alerts) == 0 {
		t.Error("期望有告警被触发，但没有")
	}
	
	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Level != AlertLevelWarning {
			t.Errorf("期望告警级别为 warning，实际为 %s", alert.Level)
		}
		if alert.Value <= 50 {
			t.Errorf("期望告警值大于 50，实际为 %.2f", alert.Value)
		}
	}
}

func TestAlertManager_ClearAlerts(t *testing.T) {
	am := NewAlertManager(1 * time.Second)
	
	// 手动添加一些告警
	am.activeAlerts = []Alert{
		{
			Level:     AlertLevelWarning,
			Title:     "测试告警",
			Message:   "测试消息",
			Timestamp: time.Now(),
		},
	}
	
	if len(am.GetActiveAlerts()) != 1 {
		t.Error("期望有 1 个活跃告警")
	}
	
	am.ClearAlerts()
	
	if len(am.GetActiveAlerts()) != 0 {
		t.Error("清空后期望没有活跃告警")
	}
}

func TestGetDefaultAlertRules(t *testing.T) {
	rules := GetDefaultAlertRules()
	
	if len(rules) == 0 {
		t.Error("期望有默认告警规则")
	}
	
	// 检查是否包含关键规则
	hasLoginFailureRule := false
	hasSlowQueryRule := false
	
	for _, rule := range rules {
		if rule.Metric == "login_failures" {
			hasLoginFailureRule = true
		}
		if rule.Metric == "slow_queries" {
			hasSlowQueryRule = true
		}
	}
	
	if !hasLoginFailureRule {
		t.Error("期望包含登录失败告警规则")
	}
	
	if !hasSlowQueryRule {
		t.Error("期望包含慢查询告警规则")
	}
}

// testLogger 实现 logger 接口用于测试
type testLogger struct {
	warnCalled  bool
	errorCalled bool
}

func (tl *testLogger) Warn(format string, args ...interface{}) {
	tl.warnCalled = true
}

func (tl *testLogger) Error(format string, args ...interface{}) {
	tl.errorCalled = true
}

func TestLogAlertHandler(t *testing.T) {
	// 创建一个模拟 logger
	logger := &testLogger{}
	
	handler := &LogAlertHandler{
		logger: logger,
	}
	
	// 测试 Warning 级别告警
	alert := Alert{
		Level:     AlertLevelWarning,
		Title:     "测试告警",
		Message:   "测试消息",
		Value:     100,
		Threshold: 50,
		Timestamp: time.Now(),
	}
	
	err := handler.HandleAlert(alert)
	if err != nil {
		t.Errorf("处理告警失败: %v", err)
	}
	
	if !logger.warnCalled {
		t.Error("期望调用 Warn 方法")
	}
	
	// 测试 Critical 级别告警
	logger.warnCalled = false
	logger.errorCalled = false
	
	alert.Level = AlertLevelCritical
	err = handler.HandleAlert(alert)
	if err != nil {
		t.Errorf("处理告警失败: %v", err)
	}
	
	if !logger.errorCalled {
		t.Error("期望调用 Error 方法")
	}
}

func TestAlertRule_Condition(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录一些数据
	for i := 0; i < 100; i++ {
		metrics.RecordLoginAttempt(true, 100*time.Millisecond)
	}
	for i := 0; i < 60; i++ {
		metrics.RecordLoginAttempt(false, 100*time.Millisecond)
	}
	
	snapshot := metrics.GetSnapshot()
	
	// 测试登录失败率规则
	rule := AlertRule{
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
	}
	
	triggered, value := rule.Condition(snapshot)
	
	if !triggered {
		t.Error("期望规则被触发")
	}
	
	if value >= 70.0 {
		t.Errorf("期望登录成功率小于 70%%，实际为 %.2f%%", value)
	}
}
