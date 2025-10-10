package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenBucketRateLimiter 基于令牌桶的速率限制器
type TokenBucketRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	config   RateLimitConfig
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	// 每秒请求数限制
	RequestsPerSecond float64
	// 突发请求数限制
	BurstSize int
	// 清理间隔
	CleanupInterval time.Duration
}

// NewTokenBucketRateLimiter 创建令牌桶速率限制器
func NewTokenBucketRateLimiter(config RateLimitConfig) *TokenBucketRateLimiter {
	limiter := &TokenBucketRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
	
	// 启动清理goroutine
	go limiter.cleanup()
	
	return limiter
}

// Allow 检查是否允许请求
func (r *TokenBucketRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	limiter := r.getLimiter(key)
	return limiter.Allow(), nil
}

// Wait 等待直到可以发送请求
func (r *TokenBucketRateLimiter) Wait(ctx context.Context, key string) error {
	limiter := r.getLimiter(key)
	return limiter.Wait(ctx)
}

// getLimiter 获取或创建限制器
func (r *TokenBucketRateLimiter) getLimiter(key string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()
	
	if exists {
		return limiter
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 双重检查
	if limiter, exists := r.limiters[key]; exists {
		return limiter
	}
	
	// 创建新的限制器
	limiter = rate.NewLimiter(rate.Limit(r.config.RequestsPerSecond), r.config.BurstSize)
	r.limiters[key] = limiter
	
	return limiter
}

// cleanup 清理不活跃的限制器
func (r *TokenBucketRateLimiter) cleanup() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		// 简单的清理策略：清理所有限制器
		// 在实际应用中，可以基于最后使用时间进行清理
		if len(r.limiters) > 1000 { // 防止内存泄漏
			r.limiters = make(map[string]*rate.Limiter)
		}
		r.mu.Unlock()
	}
}

// SlidingWindowRateLimiter 滑动窗口速率限制器
type SlidingWindowRateLimiter struct {
	mu      sync.RWMutex
	windows map[string]*SlidingWindow
	config  SlidingWindowConfig
}

// SlidingWindowConfig 滑动窗口配置
type SlidingWindowConfig struct {
	// 窗口大小
	WindowSize time.Duration
	// 窗口内最大请求数
	MaxRequests int
	// 子窗口数量
	SubWindows int
	// 清理间隔
	CleanupInterval time.Duration
}

// SlidingWindow 滑动窗口
type SlidingWindow struct {
	mu          sync.Mutex
	subWindows  []int
	windowStart time.Time
	windowSize  time.Duration
	maxRequests int
	subWindows  int
}

// NewSlidingWindowRateLimiter 创建滑动窗口速率限制器
func NewSlidingWindowRateLimiter(config SlidingWindowConfig) *SlidingWindowRateLimiter {
	limiter := &SlidingWindowRateLimiter{
		windows: make(map[string]*SlidingWindow),
		config:  config,
	}
	
	// 启动清理goroutine
	go limiter.cleanup()
	
	return limiter
}

// Allow 检查是否允许请求
func (r *SlidingWindowRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	window := r.getWindow(key)
	return window.Allow(), nil
}

// Wait 等待直到可以发送请求
func (r *SlidingWindowRateLimiter) Wait(ctx context.Context, key string) error {
	window := r.getWindow(key)
	
	for {
		if window.Allow() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// 继续尝试
		}
	}
}

// getWindow 获取或创建滑动窗口
func (r *SlidingWindowRateLimiter) getWindow(key string) *SlidingWindow {
	r.mu.RLock()
	window, exists := r.windows[key]
	r.mu.RUnlock()
	
	if exists {
		return window
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 双重检查
	if window, exists := r.windows[key]; exists {
		return window
	}
	
	// 创建新的滑动窗口
	window = &SlidingWindow{
		subWindows:  make([]int, r.config.SubWindows),
		windowStart: time.Now(),
		windowSize:  r.config.WindowSize,
		maxRequests: r.config.MaxRequests,
		subWindows:  r.config.SubWindows,
	}
	r.windows[key] = window
	
	return window
}

// Allow 检查滑动窗口是否允许请求
func (w *SlidingWindow) Allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	now := time.Now()
	
	// 计算当前子窗口索引
	elapsed := now.Sub(w.windowStart)
	if elapsed >= w.windowSize {
		// 重置窗口
		w.windowStart = now
		w.subWindows = make([]int, w.subWindows)
		elapsed = 0
	}
	
	subWindowDuration := w.windowSize / time.Duration(w.subWindows)
	currentSubWindow := int(elapsed / subWindowDuration)
	if currentSubWindow >= w.subWindows {
		currentSubWindow = w.subWindows - 1
	}
	
	// 计算当前总请求数
	totalRequests := 0
	for _, count := range w.subWindows {
		totalRequests += count
	}
	
	// 检查是否超过限制
	if totalRequests >= w.maxRequests {
		return false
	}
	
	// 增加当前子窗口的计数
	w.subWindows[currentSubWindow]++
	
	return true
}

// cleanup 清理不活跃的窗口
func (r *SlidingWindowRateLimiter) cleanup() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		
		// 清理过期的窗口
		for key, window := range r.windows {
			window.mu.Lock()
			if now.Sub(window.windowStart) > r.config.WindowSize*2 {
				delete(r.windows, key)
			}
			window.mu.Unlock()
		}
		
		r.mu.Unlock()
	}
}

// AdaptiveRateLimiter 自适应速率限制器
type AdaptiveRateLimiter struct {
	mu            sync.RWMutex
	limiters      map[string]*AdaptiveLimiter
	config        AdaptiveConfig
	errorTracker  *ErrorTracker
}

// AdaptiveConfig 自适应配置
type AdaptiveConfig struct {
	// 基础速率
	BaseRate float64
	// 最小速率
	MinRate float64
	// 最大速率
	MaxRate float64
	// 调整因子
	AdjustmentFactor float64
	// 错误率阈值
	ErrorRateThreshold float64
	// 调整间隔
	AdjustmentInterval time.Duration
}

// AdaptiveLimiter 自适应限制器
type AdaptiveLimiter struct {
	mu           sync.Mutex
	limiter      *rate.Limiter
	currentRate  float64
	config       AdaptiveConfig
	lastAdjusted time.Time
}

// ErrorTracker 错误跟踪器
type ErrorTracker struct {
	mu     sync.RWMutex
	errors map[string]*ErrorWindow
}

// ErrorWindow 错误窗口
type ErrorWindow struct {
	mu          sync.Mutex
	totalCount  int
	errorCount  int
	windowStart time.Time
	windowSize  time.Duration
}

// NewAdaptiveRateLimiter 创建自适应速率限制器
func NewAdaptiveRateLimiter(config AdaptiveConfig) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		limiters:     make(map[string]*AdaptiveLimiter),
		config:       config,
		errorTracker: NewErrorTracker(),
	}
}

// NewErrorTracker 创建错误跟踪器
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errors: make(map[string]*ErrorWindow),
	}
}

// Allow 检查是否允许请求
func (r *AdaptiveRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	limiter := r.getLimiter(key)
	return limiter.Allow(), nil
}

// Wait 等待直到可以发送请求
func (r *AdaptiveRateLimiter) Wait(ctx context.Context, key string) error {
	limiter := r.getLimiter(key)
	return limiter.Wait(ctx)
}

// RecordSuccess 记录成功请求
func (r *AdaptiveRateLimiter) RecordSuccess(key string) {
	r.errorTracker.RecordSuccess(key)
	r.adjustRate(key)
}

// RecordError 记录错误请求
func (r *AdaptiveRateLimiter) RecordError(key string) {
	r.errorTracker.RecordError(key)
	r.adjustRate(key)
}

// getLimiter 获取或创建自适应限制器
func (r *AdaptiveRateLimiter) getLimiter(key string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()
	
	if exists {
		return limiter.limiter
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 双重检查
	if limiter, exists := r.limiters[key]; exists {
		return limiter.limiter
	}
	
	// 创建新的自适应限制器
	limiter = &AdaptiveLimiter{
		limiter:      rate.NewLimiter(rate.Limit(r.config.BaseRate), 1),
		currentRate:  r.config.BaseRate,
		config:       r.config,
		lastAdjusted: time.Now(),
	}
	r.limiters[key] = limiter
	
	return limiter.limiter
}

// adjustRate 调整速率
func (r *AdaptiveRateLimiter) adjustRate(key string) {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()
	
	if !exists {
		return
	}
	
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	
	now := time.Now()
	if now.Sub(limiter.lastAdjusted) < r.config.AdjustmentInterval {
		return
	}
	
	// 获取错误率
	errorRate := r.errorTracker.GetErrorRate(key)
	
	// 根据错误率调整速率
	if errorRate > r.config.ErrorRateThreshold {
		// 降低速率
		newRate := limiter.currentRate * (1 - r.config.AdjustmentFactor)
		if newRate < r.config.MinRate {
			newRate = r.config.MinRate
		}
		limiter.currentRate = newRate
		limiter.limiter.SetLimit(rate.Limit(newRate))
	} else if errorRate < r.config.ErrorRateThreshold/2 {
		// 提高速率
		newRate := limiter.currentRate * (1 + r.config.AdjustmentFactor)
		if newRate > r.config.MaxRate {
			newRate = r.config.MaxRate
		}
		limiter.currentRate = newRate
		limiter.limiter.SetLimit(rate.Limit(newRate))
	}
	
	limiter.lastAdjusted = now
}

// RecordSuccess 记录成功请求
func (e *ErrorTracker) RecordSuccess(key string) {
	window := e.getWindow(key)
	window.mu.Lock()
	defer window.mu.Unlock()
	
	window.totalCount++
}

// RecordError 记录错误请求
func (e *ErrorTracker) RecordError(key string) {
	window := e.getWindow(key)
	window.mu.Lock()
	defer window.mu.Unlock()
	
	window.totalCount++
	window.errorCount++
}

// GetErrorRate 获取错误率
func (e *ErrorTracker) GetErrorRate(key string) float64 {
	window := e.getWindow(key)
	window.mu.Lock()
	defer window.mu.Unlock()
	
	if window.totalCount == 0 {
		return 0
	}
	
	return float64(window.errorCount) / float64(window.totalCount)
}

// getWindow 获取或创建错误窗口
func (e *ErrorTracker) getWindow(key string) *ErrorWindow {
	e.mu.RLock()
	window, exists := e.errors[key]
	e.mu.RUnlock()
	
	if exists {
		// 检查窗口是否需要重置
		if time.Since(window.windowStart) > window.windowSize {
			window.mu.Lock()
			window.totalCount = 0
			window.errorCount = 0
			window.windowStart = time.Now()
			window.mu.Unlock()
		}
		return window
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// 双重检查
	if window, exists := e.errors[key]; exists {
		return window
	}
	
	// 创建新的错误窗口
	window = &ErrorWindow{
		totalCount:  0,
		errorCount:  0,
		windowStart: time.Now(),
		windowSize:  5 * time.Minute, // 5分钟窗口
	}
	e.errors[key] = window
	
	return window
}

// GetDefaultRateLimitConfig 获取默认速率限制配置
func GetDefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 10.0,  // 每秒10个请求
		BurstSize:         20,    // 突发20个请求
		CleanupInterval:   5 * time.Minute,
	}
}

// GetDefaultSlidingWindowConfig 获取默认滑动窗口配置
func GetDefaultSlidingWindowConfig() SlidingWindowConfig {
	return SlidingWindowConfig{
		WindowSize:      time.Minute,     // 1分钟窗口
		MaxRequests:     100,             // 窗口内最大100个请求
		SubWindows:      6,               // 6个子窗口（每10秒一个）
		CleanupInterval: 5 * time.Minute,
	}
}

// GetDefaultAdaptiveConfig 获取默认自适应配置
func GetDefaultAdaptiveConfig() AdaptiveConfig {
	return AdaptiveConfig{
		BaseRate:           10.0,  // 基础每秒10个请求
		MinRate:            1.0,   // 最小每秒1个请求
		MaxRate:            50.0,  // 最大每秒50个请求
		AdjustmentFactor:   0.1,   // 10%调整因子
		ErrorRateThreshold: 0.1,   // 10%错误率阈值
		AdjustmentInterval: 30 * time.Second,
	}
}