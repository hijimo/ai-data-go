package middleware

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestCircuitBreakerClosed 测试熔断器关闭状态
func TestCircuitBreakerClosed(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 初始状态应该是关闭
	if cb.GetState() != StateClosed {
		t.Errorf("初始状态应该是 Closed，实际是 %s", cb.GetState().String())
	}

	// 成功的请求应该保持关闭状态
	_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("执行成功的请求不应该返回错误: %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("成功请求后状态应该保持 Closed，实际是 %s", cb.GetState().String())
	}
}

// TestCircuitBreakerOpen 测试熔断器打开状态
func TestCircuitBreakerOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 执行3次失败的请求
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})
	}

	// 熔断器应该打开
	if cb.GetState() != StateOpen {
		t.Errorf("3次失败后状态应该是 Open，实际是 %s", cb.GetState().String())
	}

	// 熔断器打开时，请求应该被拒绝
	_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "should not execute", nil
	})

	if err != ErrCircuitBreakerOpen {
		t.Errorf("熔断器打开时应该返回 ErrCircuitBreakerOpen，实际返回: %v", err)
	}
}

// TestCircuitBreakerHalfOpen 测试熔断器半开状态
func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond, // 短超时以便快速测试
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 执行2次失败的请求，打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("2次失败后状态应该是 Open，实际是 %s", cb.GetState().String())
	}

	// 等待超时，进入半开状态
	time.Sleep(150 * time.Millisecond)

	// 下一个请求应该触发半开状态
	_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("半开状态下的请求不应该返回错误: %v", err)
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("超时后第一个请求应该进入 HalfOpen 状态，实际是 %s", cb.GetState().String())
	}
}

// TestCircuitBreakerRecovery 测试熔断器恢复
func TestCircuitBreakerRecovery(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 3,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})
	}

	// 等待进入半开状态
	time.Sleep(150 * time.Millisecond)

	// 执行2次成功的请求，应该关闭熔断器
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("半开状态下的成功请求不应该返回错误: %v", err)
		}
	}

	// 熔断器应该关闭
	if cb.GetState() != StateClosed {
		t.Errorf("2次成功后状态应该是 Closed，实际是 %s", cb.GetState().String())
	}
}

// TestCircuitBreakerHalfOpenFailure 测试半开状态下的失败
func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})
	}

	// 等待进入半开状态
	time.Sleep(150 * time.Millisecond)

	// 半开状态下执行一次失败的请求
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	})

	// 熔断器应该重新打开
	if cb.GetState() != StateOpen {
		t.Errorf("半开状态下失败后应该重新打开，实际是 %s", cb.GetState().String())
	}
}

// TestCircuitBreakerStats 测试统计信息
func TestCircuitBreakerStats(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-stats", config)
	ctx := context.Background()

	// 执行一些请求
	testErr := errors.New("test error")
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	})

	// 获取统计信息
	stats := cb.GetStats()

	if stats.Name != "test-stats" {
		t.Errorf("统计信息中的名称不正确: %s", stats.Name)
	}

	if stats.FailureCount != 1 {
		t.Errorf("失败计数应该是 1，实际是 %d", stats.FailureCount)
	}
}

// TestCircuitBreakerReset 测试重置功能
func TestCircuitBreakerReset(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		Timeout:             1 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// 打开熔断器
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("2次失败后状态应该是 Open")
	}

	// 重置熔断器
	cb.Reset()

	// 状态应该恢复到关闭
	if cb.GetState() != StateClosed {
		t.Errorf("重置后状态应该是 Closed，实际是 %s", cb.GetState().String())
	}

	// 失败计数应该清零
	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("重置后失败计数应该是 0，实际是 %d", stats.FailureCount)
	}
}

// TestCircuitBreakerManager 测试熔断器管理器
func TestCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager()

	// 创建熔断器
	config := DefaultCircuitBreakerConfig()
	cb1 := manager.GetOrCreate("service1", config)
	cb2 := manager.GetOrCreate("service2", config)

	if cb1 == nil || cb2 == nil {
		t.Error("创建熔断器失败")
	}

	// 获取相同名称的熔断器应该返回同一个实例
	cb1Again := manager.GetOrCreate("service1", config)
	if cb1 != cb1Again {
		t.Error("相同名称应该返回同一个熔断器实例")
	}

	// 获取所有统计信息
	stats := manager.GetAllStats()
	if len(stats) != 2 {
		t.Errorf("应该有2个熔断器，实际有 %d 个", len(stats))
	}
}
