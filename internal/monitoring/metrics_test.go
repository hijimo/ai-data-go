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

// TestMetrics_RecordFlowExecution 测试 Flow 执行记录
func TestMetrics_RecordFlowExecution(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录成功执行
	metrics.RecordFlowExecution("contextBuildFlow", "success")
	metrics.RecordFlowExecution("contextBuildFlow", "success")
	
	// 记录失败执行
	metrics.RecordFlowExecution("contextBuildFlow", "error")
	
	flowMetrics := metrics.GetFlowMetrics("contextBuildFlow")
	
	if flowMetrics.Executions != 3 {
		t.Errorf("期望 Flow 执行次数为 3，实际为 %d", flowMetrics.Executions)
	}
	
	if flowMetrics.Successes != 2 {
		t.Errorf("期望 Flow 成功次数为 2，实际为 %d", flowMetrics.Successes)
	}
	
	if flowMetrics.Errors != 1 {
		t.Errorf("期望 Flow 错误次数为 1，实际为 %d", flowMetrics.Errors)
	}
	
	expectedRate := 66.67
	if flowMetrics.SuccessRate < expectedRate-0.1 || flowMetrics.SuccessRate > expectedRate+0.1 {
		t.Errorf("期望 Flow 成功率约为 %.2f%%，实际为 %.2f%%", expectedRate, flowMetrics.SuccessRate)
	}
}

// TestMetrics_RecordFlowDuration 测试 Flow 执行时间记录
func TestMetrics_RecordFlowDuration(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录多个执行时间
	metrics.RecordFlowDuration("chatGenerateFlow", 100*time.Millisecond)
	metrics.RecordFlowDuration("chatGenerateFlow", 200*time.Millisecond)
	metrics.RecordFlowDuration("chatGenerateFlow", 300*time.Millisecond)
	metrics.RecordFlowDuration("chatGenerateFlow", 400*time.Millisecond)
	metrics.RecordFlowDuration("chatGenerateFlow", 500*time.Millisecond)
	
	flowMetrics := metrics.GetFlowMetrics("chatGenerateFlow")
	
	// 平均时间应该是 300ms
	expectedAvg := 300.0
	if flowMetrics.AvgDuration < expectedAvg-1 || flowMetrics.AvgDuration > expectedAvg+1 {
		t.Errorf("期望平均执行时间约为 %.2fms，实际为 %.2fms", expectedAvg, flowMetrics.AvgDuration)
	}
	
	// P95 应该接近 500ms
	if flowMetrics.P95Duration < 450 || flowMetrics.P95Duration > 500 {
		t.Errorf("期望 P95 执行时间在 450-500ms 之间，实际为 %.2fms", flowMetrics.P95Duration)
	}
}

// TestMetrics_RecordTokenUsage 测试 Token 使用量记录
func TestMetrics_RecordTokenUsage(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录不同租户的 Token 使用
	metrics.RecordTokenUsage("tenant-1", 100, 50)
	metrics.RecordTokenUsage("tenant-1", 200, 100)
	metrics.RecordTokenUsage("tenant-2", 150, 75)
	
	tokenMetrics := metrics.GetTokenMetrics()
	
	// 检查全局统计
	if tokenMetrics.PromptTokens != 450 {
		t.Errorf("期望 Prompt Token 总数为 450，实际为 %d", tokenMetrics.PromptTokens)
	}
	
	if tokenMetrics.CompletionTokens != 225 {
		t.Errorf("期望 Completion Token 总数为 225，实际为 %d", tokenMetrics.CompletionTokens)
	}
	
	if tokenMetrics.TotalTokens != 675 {
		t.Errorf("期望总 Token 数为 675，实际为 %d", tokenMetrics.TotalTokens)
	}
	
	// 检查按租户统计
	if tokenMetrics.ByTenant["tenant-1"] != 450 {
		t.Errorf("期望 tenant-1 的 Token 使用量为 450，实际为 %d", tokenMetrics.ByTenant["tenant-1"])
	}
	
	if tokenMetrics.ByTenant["tenant-2"] != 225 {
		t.Errorf("期望 tenant-2 的 Token 使用量为 225，实际为 %d", tokenMetrics.ByTenant["tenant-2"])
	}
}

// TestMetrics_RecordCacheHitAndMiss 测试缓存命中和未命中记录
func TestMetrics_RecordCacheHitAndMiss(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录缓存命中
	metrics.RecordCacheHit("context:session-1")
	metrics.RecordCacheHit("context:session-1")
	metrics.RecordCacheHit("summary:session-2")
	
	// 记录缓存未命中
	metrics.RecordCacheMiss("context:session-3")
	
	cacheMetrics := metrics.GetCacheMetrics()
	
	if cacheMetrics.Hits != 3 {
		t.Errorf("期望缓存命中次数为 3，实际为 %d", cacheMetrics.Hits)
	}
	
	if cacheMetrics.Misses != 1 {
		t.Errorf("期望缓存未命中次数为 1，实际为 %d", cacheMetrics.Misses)
	}
	
	// 命中率应该是 75%
	expectedRate := 75.0
	if cacheMetrics.HitRate < expectedRate-0.1 || cacheMetrics.HitRate > expectedRate+0.1 {
		t.Errorf("期望缓存命中率约为 %.2f%%，实际为 %.2f%%", expectedRate, cacheMetrics.HitRate)
	}
}

// TestMetrics_GlobalFunctions 测试全局便捷函数
func TestMetrics_GlobalFunctions(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 使用全局函数记录指标
	RecordFlowExecution("testFlow", "success")
	RecordFlowDuration("testFlow", 100*time.Millisecond)
	RecordTokenUsage("tenant-test", 50, 25)
	RecordCacheHit("test-key")
	RecordCacheMiss("test-key-2")
	
	// 验证指标已记录
	flowMetrics := metrics.GetFlowMetrics("testFlow")
	if flowMetrics.Executions != 1 {
		t.Errorf("期望 Flow 执行次数为 1，实际为 %d", flowMetrics.Executions)
	}
	
	tokenMetrics := metrics.GetTokenMetrics()
	if tokenMetrics.TotalTokens != 75 {
		t.Errorf("期望总 Token 数为 75，实际为 %d", tokenMetrics.TotalTokens)
	}
	
	cacheMetrics := metrics.GetCacheMetrics()
	if cacheMetrics.Hits != 1 || cacheMetrics.Misses != 1 {
		t.Errorf("期望缓存命中 1 次，未命中 1 次，实际命中 %d 次，未命中 %d 次", 
			cacheMetrics.Hits, cacheMetrics.Misses)
	}
}

// TestMetrics_MultipleFlows 测试多个 Flow 的指标记录
func TestMetrics_MultipleFlows(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录多个不同的 Flow
	flows := []string{"contextBuildFlow", "chatGenerateFlow", "memorySearchFlow"}
	
	for _, flow := range flows {
		metrics.RecordFlowExecution(flow, "success")
		metrics.RecordFlowDuration(flow, 100*time.Millisecond)
	}
	
	// 验证每个 Flow 的指标
	for _, flow := range flows {
		flowMetrics := metrics.GetFlowMetrics(flow)
		if flowMetrics.Executions != 1 {
			t.Errorf("Flow %s: 期望执行次数为 1，实际为 %d", flow, flowMetrics.Executions)
		}
		if flowMetrics.Successes != 1 {
			t.Errorf("Flow %s: 期望成功次数为 1，实际为 %d", flow, flowMetrics.Successes)
		}
	}
}

// TestMetrics_P99Calculation 测试 P99 百分位数计算
func TestMetrics_P99Calculation(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	// 记录 100 个不同的执行时间
	for i := 1; i <= 100; i++ {
		metrics.RecordFlowDuration("testFlow", time.Duration(i)*time.Millisecond)
	}
	
	flowMetrics := metrics.GetFlowMetrics("testFlow")
	
	// P99 应该接近 99ms
	if flowMetrics.P99Duration < 98 || flowMetrics.P99Duration > 100 {
		t.Errorf("期望 P99 在 98-100ms 之间，实际为 %.2fms", flowMetrics.P99Duration)
	}
}

// TestMetrics_EmptyFlowMetrics 测试获取不存在的 Flow 指标
func TestMetrics_EmptyFlowMetrics(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	flowMetrics := metrics.GetFlowMetrics("nonExistentFlow")
	
	if flowMetrics.Executions != 0 {
		t.Errorf("期望不存在的 Flow 执行次数为 0，实际为 %d", flowMetrics.Executions)
	}
	
	if flowMetrics.SuccessRate != 0 {
		t.Errorf("期望不存在的 Flow 成功率为 0，实际为 %.2f", flowMetrics.SuccessRate)
	}
}

// TestMetrics_ZeroCacheHitRate 测试零缓存命中率
func TestMetrics_ZeroCacheHitRate(t *testing.T) {
	metrics := GetMetrics()
	metrics.Reset()
	
	cacheMetrics := metrics.GetCacheMetrics()
	
	if cacheMetrics.HitRate != 0 {
		t.Errorf("期望空缓存的命中率为 0，实际为 %.2f", cacheMetrics.HitRate)
	}
}
