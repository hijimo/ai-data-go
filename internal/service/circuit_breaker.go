package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	// StateClosed 关闭状态：正常执行请求
	StateClosed CircuitBreakerState = iota
	// StateOpen 打开状态：拒绝所有请求
	StateOpen
	// StateHalfOpen 半开状态：允许部分请求通过以测试服务是否恢复
	StateHalfOpen
)

// String 返回状态的字符串表示
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// MaxFailures 触发熔断的最大失败次数
	MaxFailures int
	// Timeout 熔断器打开后的超时时间，超时后进入半开状态
	Timeout time.Duration
	// HalfOpenMaxRequests 半开状态下允许的最大请求数
	HalfOpenMaxRequests int
	// HalfOpenSuccessThreshold 半开状态下关闭熔断器所需的连续成功次数
	HalfOpenSuccessThreshold int
	// ResetTimeout 关闭状态下重置失败计数的超时时间
	ResetTimeout time.Duration
}

// DefaultCircuitBreakerConfig 返回默认的熔断器配置
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:              5,
		Timeout:                  30 * time.Second,
		HalfOpenMaxRequests:      3,
		HalfOpenSuccessThreshold: 2,
		ResetTimeout:             60 * time.Second,
	}
}

// CircuitBreaker 熔断器
// 实现三种状态管理：Closed（关闭）、Open（打开）、HalfOpen（半开）
// 用于保护外部服务调用，防止级联故障
type CircuitBreaker struct {
	mu                   sync.RWMutex
	state                CircuitBreakerState
	failureCount         int
	successCount         int
	lastFailureTime      time.Time
	lastSuccessTime      time.Time
	lastStateChangeTime  time.Time
	halfOpenRequestCount int
	config               CircuitBreakerConfig
	name                 string
	log                  logger.Logger
}

// NewCircuitBreaker 创建新的熔断器实例
// 参数:
//   name: 熔断器名称，用于日志和监控
//   config: 熔断器配置
//   log: 日志记录器
// 返回:
//   *CircuitBreaker: 熔断器实例
func NewCircuitBreaker(name string, config CircuitBreakerConfig, log logger.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		state:               StateClosed,
		config:              config,
		name:                name,
		lastStateChangeTime: time.Now(),
		log:                 log,
	}
}

// Execute 执行受熔断器保护的函数
// 参数:
//   ctx: 上下文
//   fn: 要执行的函数
// 返回:
//   error: 执行错误或熔断器错误
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// 检查是否可以执行
	if err := cb.beforeRequest(ctx); err != nil {
		return err
	}

	// 执行函数
	err := fn()

	// 记录执行结果
	cb.afterRequest(ctx, err)

	return err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest(ctx context.Context) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// 关闭状态：检查是否需要重置失败计数
		if time.Since(cb.lastFailureTime) > cb.config.ResetTimeout && cb.failureCount > 0 {
			cb.log.InfoContext(ctx, "熔断器重置失败计数", logger.Fields{
				"circuit_breaker": cb.name,
				"failure_count":   cb.failureCount,
			})
			cb.failureCount = 0
		}
		return nil

	case StateOpen:
		// 打开状态：检查是否可以进入半开状态
		if time.Since(cb.lastStateChangeTime) > cb.config.Timeout {
			cb.setState(ctx, StateHalfOpen)
			cb.halfOpenRequestCount = 0
			cb.successCount = 0
			return nil
		}
		// 仍在打开状态，拒绝请求
		return fmt.Errorf("熔断器已打开: %s", cb.name)

	case StateHalfOpen:
		// 半开状态：检查是否已达到最大请求数
		if cb.halfOpenRequestCount >= cb.config.HalfOpenMaxRequests {
			return fmt.Errorf("熔断器半开状态请求数已达上限: %s", cb.name)
		}
		cb.halfOpenRequestCount++
		return nil

	default:
		return errors.New("未知的熔断器状态")
	}
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(ctx context.Context, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// 请求失败
		cb.onFailure(ctx)
	} else {
		// 请求成功
		cb.onSuccess(ctx)
	}
}

// onFailure 处理失败情况
func (cb *CircuitBreaker) onFailure(ctx context.Context) {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	cb.log.WarnContext(ctx, "熔断器记录失败", logger.Fields{
		"circuit_breaker": cb.name,
		"state":           cb.state.String(),
		"failure_count":   cb.failureCount,
		"max_failures":    cb.config.MaxFailures,
	})

	switch cb.state {
	case StateClosed:
		// 关闭状态：检查是否达到失败阈值
		if cb.failureCount >= cb.config.MaxFailures {
			cb.setState(ctx, StateOpen)
		}

	case StateHalfOpen:
		// 半开状态：任何失败都会重新打开熔断器
		cb.setState(ctx, StateOpen)
		cb.halfOpenRequestCount = 0
		cb.successCount = 0
	}
}

// onSuccess 处理成功情况
func (cb *CircuitBreaker) onSuccess(ctx context.Context) {
	cb.lastSuccessTime = time.Now()

	switch cb.state {
	case StateClosed:
		// 关闭状态：重置失败计数
		if cb.failureCount > 0 {
			cb.log.InfoContext(ctx, "熔断器成功执行，重置失败计数", logger.Fields{
				"circuit_breaker":     cb.name,
				"previous_failures":   cb.failureCount,
			})
			cb.failureCount = 0
		}

	case StateHalfOpen:
		// 半开状态：累计成功次数
		cb.successCount++
		
		cb.log.InfoContext(ctx, "熔断器半开状态成功执行", logger.Fields{
			"circuit_breaker":   cb.name,
			"success_count":     cb.successCount,
			"success_threshold": cb.config.HalfOpenSuccessThreshold,
		})

		// 检查是否达到成功阈值
		if cb.successCount >= cb.config.HalfOpenSuccessThreshold {
			cb.setState(ctx, StateClosed)
			cb.failureCount = 0
			cb.successCount = 0
			cb.halfOpenRequestCount = 0
		}
	}
}

// setState 设置熔断器状态
func (cb *CircuitBreaker) setState(ctx context.Context, newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChangeTime = time.Now()

	cb.log.InfoContext(ctx, "熔断器状态变更", logger.Fields{
		"circuit_breaker": cb.name,
		"old_state":       oldState.String(),
		"new_state":       newState.String(),
		"failure_count":   cb.failureCount,
		"success_count":   cb.successCount,
	})
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats 获取熔断器统计信息
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:                cb.name,
		State:               cb.state.String(),
		FailureCount:        cb.failureCount,
		SuccessCount:        cb.successCount,
		LastFailureTime:     cb.lastFailureTime,
		LastSuccessTime:     cb.lastSuccessTime,
		LastStateChangeTime: cb.lastStateChangeTime,
	}
}

// Reset 重置熔断器到关闭状态
func (cb *CircuitBreaker) Reset(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.log.InfoContext(ctx, "手动重置熔断器", logger.Fields{
		"circuit_breaker": cb.name,
		"old_state":       cb.state.String(),
	})

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenRequestCount = 0
	cb.lastStateChangeTime = time.Now()
}

// CircuitBreakerStats 熔断器统计信息
type CircuitBreakerStats struct {
	Name                string
	State               string
	FailureCount        int
	SuccessCount        int
	LastFailureTime     time.Time
	LastSuccessTime     time.Time
	LastStateChangeTime time.Time
}

// CircuitBreakerManager 熔断器管理器
// 管理多个熔断器实例
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	log      logger.Logger
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager(log logger.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		log:      log,
	}
}

// GetOrCreate 获取或创建熔断器
func (m *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	breaker := NewCircuitBreaker(name, config, m.log)
	m.breakers[name] = breaker

	m.log.InfoContext(context.Background(), "创建新的熔断器", logger.Fields{
		"circuit_breaker": name,
		"max_failures":    config.MaxFailures,
		"timeout":         config.Timeout.String(),
	})

	return breaker
}

// Get 获取熔断器
func (m *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breaker, exists := m.breakers[name]
	return breaker, exists
}

// GetAllStats 获取所有熔断器的统计信息
func (m *CircuitBreakerManager) GetAllStats() []CircuitBreakerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]CircuitBreakerStats, 0, len(m.breakers))
	for _, breaker := range m.breakers {
		stats = append(stats, breaker.GetStats())
	}

	return stats
}

// ResetAll 重置所有熔断器
func (m *CircuitBreakerManager) ResetAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset(ctx)
	}

	m.log.InfoContext(ctx, "重置所有熔断器", logger.Fields{
		"breaker_count": len(m.breakers),
	})
}
