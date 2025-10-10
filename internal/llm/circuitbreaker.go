package llm

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreakerError 熔断器错误
var (
	ErrCircuitBreakerOpen     = errors.New("熔断器已打开")
	ErrCircuitBreakerTimeout  = errors.New("熔断器超时")
	ErrTooManyRequests        = errors.New("请求过多")
)

// State 熔断器状态
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// String 状态字符串表示
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Counts 计数器
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// OnRequest 请求时调用
func (c *Counts) OnRequest() {
	c.Requests++
}

// OnSuccess 成功时调用
func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// OnFailure 失败时调用
func (c *Counts) OnFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// Clear 清空计数器
func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// Settings 熔断器设置
type Settings struct {
	Name        string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
	ReadyToTrip func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          st.Name,
		onStateChange: st.OnStateChange,
	}

	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval <= 0 {
		cb.interval = time.Duration(0)
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout <= 0 {
		cb.timeout = 60 * time.Second
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Name 获取熔断器名称
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts 获取当前计数
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// Execute 执行函数
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// Call 实现CircuitBreaker接口
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	err = fn()
	cb.afterRequest(generation, err == nil)
	return err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrCircuitBreakerOpen
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.OnRequest()
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.OnSuccess()

	if state == StateHalfOpen {
		cb.setState(StateClosed, now)
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.OnFailure()

	switch state {
	case StateClosed:
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// currentState 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration 转到新一代
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.Clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// TwoStepCircuitBreaker 两步熔断器
type TwoStepCircuitBreaker struct {
	cb *CircuitBreaker
}

// NewTwoStepCircuitBreaker 创建两步熔断器
func NewTwoStepCircuitBreaker(st Settings) *TwoStepCircuitBreaker {
	return &TwoStepCircuitBreaker{
		cb: NewCircuitBreaker(st),
	}
}

// Name 获取名称
func (tscb *TwoStepCircuitBreaker) Name() string {
	return tscb.cb.Name()
}

// State 获取状态
func (tscb *TwoStepCircuitBreaker) State() State {
	return tscb.cb.State()
}

// Allow 检查是否允许请求
func (tscb *TwoStepCircuitBreaker) Allow() (done func(success bool), err error) {
	generation, err := tscb.cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	return func(success bool) {
		tscb.cb.afterRequest(generation, success)
	}, nil
}

// MultiCircuitBreaker 多熔断器管理器
type MultiCircuitBreaker struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	settings Settings
}

// NewMultiCircuitBreaker 创建多熔断器管理器
func NewMultiCircuitBreaker(settings Settings) *MultiCircuitBreaker {
	return &MultiCircuitBreaker{
		breakers: make(map[string]*CircuitBreaker),
		settings: settings,
	}
}

// Call 调用指定键的熔断器
func (mcb *MultiCircuitBreaker) Call(ctx context.Context, key string, fn func() error) error {
	breaker := mcb.getBreaker(key)
	return breaker.Call(ctx, fn)
}

// State 获取指定键的熔断器状态
func (mcb *MultiCircuitBreaker) State(key string) string {
	breaker := mcb.getBreaker(key)
	return breaker.State().String()
}

// getBreaker 获取或创建熔断器
func (mcb *MultiCircuitBreaker) getBreaker(key string) *CircuitBreaker {
	mcb.mu.RLock()
	breaker, exists := mcb.breakers[key]
	mcb.mu.RUnlock()

	if exists {
		return breaker
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	// 双重检查
	if breaker, exists := mcb.breakers[key]; exists {
		return breaker
	}

	// 创建新的熔断器
	settings := mcb.settings
	settings.Name = key
	breaker = NewCircuitBreaker(settings)
	mcb.breakers[key] = breaker

	return breaker
}

// GetAllStates 获取所有熔断器状态
func (mcb *MultiCircuitBreaker) GetAllStates() map[string]string {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	states := make(map[string]string)
	for key, breaker := range mcb.breakers {
		states[key] = breaker.State().String()
	}
	return states
}

// GetAllCounts 获取所有熔断器计数
func (mcb *MultiCircuitBreaker) GetAllCounts() map[string]Counts {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	counts := make(map[string]Counts)
	for key, breaker := range mcb.breakers {
		counts[key] = breaker.Counts()
	}
	return counts
}

// 预定义的熔断器配置
var (
	// DefaultCircuitBreakerSettings 默认熔断器设置
	DefaultCircuitBreakerSettings = Settings{
		Name:        "default",
		MaxRequests: 1,
		Interval:    time.Duration(0),
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}

	// AggressiveCircuitBreakerSettings 激进的熔断器设置
	AggressiveCircuitBreakerSettings = Settings{
		Name:        "aggressive",
		MaxRequests: 1,
		Interval:    time.Duration(0),
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}

	// ConservativeCircuitBreakerSettings 保守的熔断器设置
	ConservativeCircuitBreakerSettings = Settings{
		Name:        "conservative",
		MaxRequests: 3,
		Interval:    2 * time.Minute,
		Timeout:     2 * time.Minute,
		ReadyToTrip: func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.8
		},
	}
)

// GetCircuitBreakerSettings 根据提供商类型获取熔断器设置
func GetCircuitBreakerSettings(providerType ProviderType) Settings {
	settings := DefaultCircuitBreakerSettings
	settings.Name = string(providerType)
	
	// 根据不同提供商调整设置
	switch providerType {
	case ProviderOpenAI:
		// OpenAI相对稳定，使用默认设置
		settings.Timeout = 60 * time.Second
		settings.ReadyToTrip = func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.7
		}
	case ProviderQianwen:
		// 千问可能有更多限制，使用稍微激进的设置
		settings.Timeout = 45 * time.Second
		settings.ReadyToTrip = func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		}
	case ProviderClaude:
		// Claude相对稳定，使用保守设置
		settings.Timeout = 90 * time.Second
		settings.ReadyToTrip = func(counts Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.8
		}
	default:
		// 其他提供商使用默认设置
	}
	
	return settings
}