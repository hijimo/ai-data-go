package middleware

import (
	"context"
	"errors"
	"sync"
	"time"

	"genkit-ai-service/internal/logger"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	// StateClosed 关闭状态：正常执行请求
	StateClosed CircuitState = iota
	// StateOpen 打开状态：拒绝所有请求
	StateOpen
	// StateHalfOpen 半开状态：允许部分请求通过以测试服务是否恢复
	StateHalfOpen
)

// String 返回状态的字符串表示
func (s CircuitState) String() string {
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
	// SuccessThreshold 半开状态下连续成功次数达到此阈值后关闭熔断器
	SuccessThreshold int
	// OnStateChange 状态变化回调函数
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig 返回默认配置
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:         5,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 3,
		SuccessThreshold:    2,
		OnStateChange:       nil,
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name   string
	config *CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	halfOpenRequests  int
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		name:   name,
		config: config,
		state:  StateClosed,
	}
}

// Execute 执行带熔断保护的操作
// 参数:
//   - ctx: 上下文
//   - fn: 要执行的函数
// 返回:
//   - 函数执行结果
//   - 错误信息
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	// 检查是否可以执行
	if !cb.canExecute() {
		logger.WarnContext(ctx, "熔断器拒绝请求", logger.Fields{
			"circuit_breaker": cb.name,
			"state":           cb.state.String(),
		})
		return nil, ErrCircuitBreakerOpen
	}

	// 执行函数
	result, err := fn(ctx)

	// 记录执行结果
	cb.recordResult(ctx, err)

	return result, err
}

// canExecute 检查是否可以执行请求
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// 关闭状态：允许所有请求
		return true

	case StateOpen:
		// 检查是否超时，如果超时则进入半开状态
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 0
			return true
		}
		// 未超时，拒绝请求
		return false

	case StateHalfOpen:
		// 半开状态：限制请求数量
		if cb.halfOpenRequests < cb.config.HalfOpenMaxRequests {
			cb.halfOpenRequests++
			return true
		}
		return false

	default:
		return false
	}
}

// recordResult 记录执行结果
func (cb *CircuitBreaker) recordResult(ctx context.Context, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// 执行失败
		cb.onFailure(ctx)
	} else {
		// 执行成功
		cb.onSuccess(ctx)
	}
}

// onFailure 处理失败情况
func (cb *CircuitBreaker) onFailure(ctx context.Context) {
	cb.failureCount++
	cb.successCount = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// 关闭状态：检查是否达到失败阈值
		if cb.failureCount >= cb.config.MaxFailures {
			logger.WarnContext(ctx, "熔断器打开", logger.Fields{
				"circuit_breaker": cb.name,
				"failure_count":   cb.failureCount,
				"max_failures":    cb.config.MaxFailures,
			})
			cb.setState(StateOpen)
		}

	case StateHalfOpen:
		// 半开状态：任何失败都会重新打开熔断器
		logger.WarnContext(ctx, "熔断器重新打开", logger.Fields{
			"circuit_breaker": cb.name,
			"reason":          "半开状态下请求失败",
		})
		cb.setState(StateOpen)
		cb.halfOpenRequests = 0
	}
}

// onSuccess 处理成功情况
func (cb *CircuitBreaker) onSuccess(ctx context.Context) {
	switch cb.state {
	case StateClosed:
		// 关闭状态：重置失败计数
		cb.failureCount = 0

	case StateHalfOpen:
		// 半开状态：增加成功计数
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			// 连续成功达到阈值，关闭熔断器
			logger.InfoContext(ctx, "熔断器关闭", logger.Fields{
				"circuit_breaker":    cb.name,
				"success_count":      cb.successCount,
				"success_threshold":  cb.config.SuccessThreshold,
			})
			cb.setState(StateClosed)
			cb.failureCount = 0
			cb.successCount = 0
			cb.halfOpenRequests = 0
		}
	}
}

// setState 设置熔断器状态
func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	// 调用状态变化回调
	if cb.config.OnStateChange != nil && oldState != newState {
		cb.config.OnStateChange(oldState, newState)
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats 获取统计信息
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:             cb.name,
		State:            cb.state.String(),
		FailureCount:     cb.failureCount,
		SuccessCount:     cb.successCount,
		LastFailureTime:  cb.lastFailureTime,
		HalfOpenRequests: cb.halfOpenRequests,
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenRequests = 0
	cb.lastFailureTime = time.Time{}

	if cb.config.OnStateChange != nil && oldState != StateClosed {
		cb.config.OnStateChange(oldState, StateClosed)
	}
}

// CircuitBreakerStats 熔断器统计信息
type CircuitBreakerStats struct {
	Name             string    `json:"name"`
	State            string    `json:"state"`
	FailureCount     int       `json:"failureCount"`
	SuccessCount     int       `json:"successCount"`
	LastFailureTime  time.Time `json:"lastFailureTime"`
	HalfOpenRequests int       `json:"halfOpenRequests"`
}

// ErrCircuitBreakerOpen 熔断器打开错误
var ErrCircuitBreakerOpen = errors.New("熔断器已打开，请求被拒绝")

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate 获取或创建熔断器
func (m *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.breakers[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(name, config)
	m.breakers[name] = cb
	return cb
}

// Get 获取熔断器
func (m *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cb, exists := m.breakers[name]
	return cb, exists
}

// GetAllStats 获取所有熔断器的统计信息
func (m *CircuitBreakerManager) GetAllStats() []CircuitBreakerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]CircuitBreakerStats, 0, len(m.breakers))
	for _, cb := range m.breakers {
		stats = append(stats, cb.GetStats())
	}
	return stats
}

// ResetAll 重置所有熔断器
func (m *CircuitBreakerManager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cb := range m.breakers {
		cb.Reset()
	}
}
