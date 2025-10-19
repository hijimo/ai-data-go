package monitoring

import (
	"testing"
	"time"
)

func TestMetrics_RecordLoginAttempt(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录成功登录
	metrics.RecordLoginAttempt(true, 100*time.Millisecond)
	metrics.RecordLoginAttempt(true, 150*time.Millisecond)
	
	// 记录失败登录
	metrics.RecordLoginAttempt(false, 50*time.Millisecond)
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.LoginAttempts != 3 {
		t.Errorf("期望登录尝试次数为 3，实际为 %d", snapshot.LoginAttempts)
	}
	
	if snapshot.LoginSuccesses != 2 {
		t.Errorf("期望登录成功次数为 2，实际为 %d", snapshot.LoginSuccesses)
	}
	
	if snapshot.LoginFailures != 1 {
		t.Errorf("期望登录失败次数为 1，实际为 %d", snapshot.LoginFailures)
	}
	
	expectedRate := 66.67
	if snapshot.LoginSuccessRate < expectedRate-0.1 || snapshot.LoginSuccessRate > expectedRate+0.1 {
		t.Errorf("期望登录成功率约为 %.2f%%，实际为 %.2f%%", expectedRate, snapshot.LoginSuccessRate)
	}
}

func TestMetrics_RecordTokenRefresh(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录成功刷新
	metrics.RecordTokenRefresh(true, 30*time.Millisecond)
	metrics.RecordTokenRefresh(true, 40*time.Millisecond)
	
	// 记录失败刷新
	metrics.RecordTokenRefresh(false, 20*time.Millisecond)
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.TokenRefreshes != 3 {
		t.Errorf("期望 Token 刷新次数为 3，实际为 %d", snapshot.TokenRefreshes)
	}
	
	if snapshot.TokenRefreshFailures != 1 {
		t.Errorf("期望 Token 刷新失败次数为 1，实际为 %d", snapshot.TokenRefreshFailures)
	}
}

func TestMetrics_RecordSecurityEvents(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	metrics.RecordInvalidToken()
	metrics.RecordInvalidToken()
	metrics.RecordExpiredToken()
	metrics.RecordRevokedToken()
	metrics.RecordBruteForceAttempt()
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.InvalidTokens != 2 {
		t.Errorf("期望无效 Token 次数为 2，实际为 %d", snapshot.InvalidTokens)
	}
	
	if snapshot.ExpiredTokens != 1 {
		t.Errorf("期望过期 Token 次数为 1，实际为 %d", snapshot.ExpiredTokens)
	}
	
	if snapshot.RevokedTokens != 1 {
		t.Errorf("期望已撤销 Token 次数为 1，实际为 %d", snapshot.RevokedTokens)
	}
	
	if snapshot.BruteForceAttempts != 1 {
		t.Errorf("期望暴力破解尝试次数为 1，实际为 %d", snapshot.BruteForceAttempts)
	}
}

func TestMetrics_UpdateTenantAndUserCounts(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	metrics.UpdateActiveTenants(10)
	metrics.UpdateActiveUsers(150)
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.ActiveTenants != 10 {
		t.Errorf("期望活跃租户数为 10，实际为 %d", snapshot.ActiveTenants)
	}
	
	if snapshot.ActiveUsers != 150 {
		t.Errorf("期望活跃用户数为 150，实际为 %d", snapshot.ActiveUsers)
	}
}

func TestMetrics_CalculateP95(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}
	
	p95 := calculateP95(durations)
	
	// P95 应该接近 95ms
	if p95 < 90 || p95 > 100 {
		t.Errorf("期望 P95 在 90-100ms 之间，实际为 %.2fms", p95)
	}
}

func TestMetrics_Reset(t *testing.T) {
	metrics := GetMetrics()
	
	// 记录一些数据
	metrics.RecordLoginAttempt(true, 100*time.Millisecond)
	metrics.RecordTokenRefresh(true, 50*time.Millisecond)
	metrics.RecordSlowQuery()
	
	// 重置
	metrics.Reset()
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.LoginAttempts != 0 {
		t.Errorf("重置后期望登录尝试次数为 0，实际为 %d", snapshot.LoginAttempts)
	}
	
	if snapshot.TokenRefreshes != 0 {
		t.Errorf("重置后期望 Token 刷新次数为 0，实际为 %d", snapshot.TokenRefreshes)
	}
	
	if snapshot.SlowQueries != 0 {
		t.Errorf("重置后期望慢查询次数为 0，实际为 %d", snapshot.SlowQueries)
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 并发记录
	done := make(chan bool)
	
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metrics.RecordLoginAttempt(true, 100*time.Millisecond)
			}
			done <- true
		}()
	}
	
	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
	
	snapshot := metrics.GetSnapshot()
	
	if snapshot.LoginAttempts != 1000 {
		t.Errorf("期望登录尝试次数为 1000，实际为 %d", snapshot.LoginAttempts)
	}
}
