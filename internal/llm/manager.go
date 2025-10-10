package llm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager LLM管理器
type Manager struct {
	mu               sync.RWMutex
	providers        map[string]LLMProvider
	factory          ProviderFactory
	metrics          MetricsCollector
	limiter          RateLimiter
	breaker          *MultiCircuitBreaker
	monitoring       *MonitoringManager
	costCalculator   CostCalculator
}

// NewManager 创建LLM管理器
func NewManager(factory ProviderFactory) *Manager {
	// 创建组件
	metricsCollector := NewPrometheusMetricsCollector()
	costCalculator := NewDefaultCostCalculator()
	alertManager := NewDefaultAlertManager()
	
	// 创建监控管理器
	monitoring := NewMonitoringManager(metricsCollector, costCalculator, alertManager)
	
	// 创建熔断器
	breaker := NewMultiCircuitBreaker(DefaultCircuitBreakerSettings)
	
	return &Manager{
		providers:      make(map[string]LLMProvider),
		factory:        factory,
		metrics:        metricsCollector,
		limiter:        NewTokenBucketRateLimiter(GetDefaultRateLimitConfig()),
		breaker:        breaker,
		monitoring:     monitoring,
		costCalculator: costCalculator,
	}
}

// NewManagerWithConfig 使用配置创建LLM管理器
func NewManagerWithConfig(factory ProviderFactory, config ManagerConfig) *Manager {
	var metricsCollector MetricsCollector
	if config.EnablePrometheus {
		metricsCollector = NewPrometheusMetricsCollector()
	} else {
		metricsCollector = NewDefaultMetricsCollector()
	}
	
	var rateLimiter RateLimiter
	switch config.RateLimiterType {
	case "token_bucket":
		rateLimiter = NewTokenBucketRateLimiter(config.RateLimitConfig)
	case "sliding_window":
		rateLimiter = NewSlidingWindowRateLimiter(config.SlidingWindowConfig)
	case "adaptive":
		rateLimiter = NewAdaptiveRateLimiter(config.AdaptiveConfig)
	default:
		rateLimiter = NewTokenBucketRateLimiter(GetDefaultRateLimitConfig())
	}
	
	costCalculator := NewDefaultCostCalculator()
	alertManager := NewDefaultAlertManager()
	monitoring := NewMonitoringManager(metricsCollector, costCalculator, alertManager)
	breaker := NewMultiCircuitBreaker(config.CircuitBreakerSettings)
	
	return &Manager{
		providers:      make(map[string]LLMProvider),
		factory:        factory,
		metrics:        metricsCollector,
		limiter:        rateLimiter,
		breaker:        breaker,
		monitoring:     monitoring,
		costCalculator: costCalculator,
	}
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	EnablePrometheus        bool
	RateLimiterType        string
	RateLimitConfig        RateLimitConfig
	SlidingWindowConfig    SlidingWindowConfig
	AdaptiveConfig         AdaptiveConfig
	CircuitBreakerSettings Settings
}

// AddProvider 添加提供商
func (m *Manager) AddProvider(name string, config ProviderConfig) error {
	provider, err := m.factory.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("创建提供商失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = provider

	return nil
}

// RemoveProvider 移除提供商
func (m *Manager) RemoveProvider(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, name)
}

// GetProvider 获取提供商
func (m *Manager) GetProvider(name string) (LLMProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("提供商 %s 不存在", name)
	}

	return provider, nil
}

// ListProviders 列出所有提供商
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GenerateText 生成文本（带监控和限流）
func (m *Manager) GenerateText(ctx context.Context, providerName string, req *GenerateRequest) (*GenerateResponse, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// 速率限制检查
	if !m.allowRequest(ctx, providerName) {
		return nil, NewLLMError(ErrCodeRateLimitExceeded, "超出速率限制", provider.GetProviderType())
	}

	// 记录开始时间
	startTime := time.Now()
	callID := uuid.New()

	// 记录活跃请求
	if prometheusCollector, ok := m.metrics.(*PrometheusMetricsCollector); ok {
		prometheusCollector.RecordActiveRequest(provider.GetProviderType(), req.Model, 1)
		defer prometheusCollector.RecordActiveRequest(provider.GetProviderType(), req.Model, -1)
	}

	// 使用熔断器执行请求
	var response *GenerateResponse
	err = m.breaker.Call(ctx, providerName, func() error {
		response, err = provider.GenerateText(ctx, req)
		return err
	})

	// 记录指标
	m.recordMetrics(callID, provider, req.Model, startTime, response, err)
	
	// 记录速率限制器结果
	if adaptiveLimiter, ok := m.limiter.(*AdaptiveRateLimiter); ok {
		if err != nil {
			adaptiveLimiter.RecordError(providerName)
		} else {
			adaptiveLimiter.RecordSuccess(providerName)
		}
	}

	return response, err
}

// GenerateStream 流式生成文本（带监控和限流）
func (m *Manager) GenerateStream(ctx context.Context, providerName string, req *GenerateRequest) (<-chan *StreamResponse, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// 速率限制检查
	if !m.allowRequest(ctx, providerName) {
		return nil, NewLLMError(ErrCodeRateLimitExceeded, "超出速率限制", provider.GetProviderType())
	}

	// 记录开始时间
	startTime := time.Now()
	callID := uuid.New()

	// 记录活跃请求
	if prometheusCollector, ok := m.metrics.(*PrometheusMetricsCollector); ok {
		prometheusCollector.RecordActiveRequest(provider.GetProviderType(), req.Model, 1)
		defer prometheusCollector.RecordActiveRequest(provider.GetProviderType(), req.Model, -1)
	}

	// 使用熔断器执行请求
	var responseChan <-chan *StreamResponse
	err = m.breaker.Call(ctx, providerName, func() error {
		responseChan, err = provider.GenerateStream(ctx, req)
		return err
	})

	if err != nil {
		// 记录失败指标
		m.recordMetrics(callID, provider, req.Model, startTime, nil, err)
		return nil, err
	}

	// 包装响应通道以记录指标
	wrappedChan := make(chan *StreamResponse, 10)
	go func() {
		defer close(wrappedChan)
		
		var totalUsage Usage
		var lastResponse *StreamResponse
		
		for response := range responseChan {
			wrappedChan <- response
			lastResponse = response
			
			if response.Usage != nil {
				totalUsage = *response.Usage
			}
		}
		
		// 记录流式请求的指标
		if lastResponse != nil {
			m.recordStreamMetrics(callID, provider, req.Model, startTime, &totalUsage, lastResponse.Error)
		}
	}()

	return wrappedChan, nil
}

// ListModels 列出模型
func (m *Manager) ListModels(ctx context.Context, providerName string) ([]Model, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.ListModels(ctx)
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck(ctx context.Context, providerName string) error {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return err
	}

	return provider.HealthCheck(ctx)
}

// HealthCheckAll 检查所有提供商的健康状态
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	providers := make(map[string]LLMProvider)
	for name, provider := range m.providers {
		providers[name] = provider
	}
	m.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup

	for name, provider := range providers {
		wg.Add(1)
		go func(name string, provider LLMProvider) {
			defer wg.Done()
			results[name] = provider.HealthCheck(ctx)
		}(name, provider)
	}

	wg.Wait()
	return results
}

// GetMetrics 获取指标
func (m *Manager) GetMetrics() interface{} {
	return m.metrics.GetMetrics()
}

// allowRequest 检查是否允许请求
func (m *Manager) allowRequest(ctx context.Context, providerName string) bool {
	if m.limiter == nil {
		return true
	}

	allowed, err := m.limiter.Allow(ctx, providerName)
	if err != nil {
		// 记录错误，但允许请求继续
		return true
	}

	// 记录速率限制触发
	if !allowed {
		if prometheusCollector, ok := m.metrics.(*PrometheusMetricsCollector); ok {
			// 需要从providerName获取provider来获取类型
			if provider, err := m.GetProvider(providerName); err == nil {
				prometheusCollector.RecordRateLimitHit(provider.GetProviderType())
			}
		}
	}

	return allowed
}

// GetCircuitBreakerStates 获取所有熔断器状态
func (m *Manager) GetCircuitBreakerStates() map[string]string {
	if m.breaker == nil {
		return make(map[string]string)
	}
	return m.breaker.GetAllStates()
}

// GetCircuitBreakerCounts 获取所有熔断器计数
func (m *Manager) GetCircuitBreakerCounts() map[string]Counts {
	if m.breaker == nil {
		return make(map[string]Counts)
	}
	return m.breaker.GetAllCounts()
}

// UpdateModelPricing 更新模型价格
func (m *Manager) UpdateModelPricing(providerType ProviderType, model string, pricing ModelPricing) {
	if m.costCalculator != nil {
		if calculator, ok := m.costCalculator.(*DefaultCostCalculator); ok {
			calculator.UpdatePricing(providerType, model, pricing)
		}
	}
}

// GetModelPricing 获取模型价格
func (m *Manager) GetModelPricing(providerType ProviderType, model string) (ModelPricing, bool) {
	if m.costCalculator != nil {
		if calculator, ok := m.costCalculator.(*DefaultCostCalculator); ok {
			return calculator.GetPricing(providerType, model)
		}
	}
	return ModelPricing{}, false
}

// recordMetrics 记录指标
func (m *Manager) recordMetrics(callID uuid.UUID, provider LLMProvider, model string, startTime time.Time, response *GenerateResponse, err error) {
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	metrics := &CallMetrics{
		ID:           callID,
		ProviderType: provider.GetProviderType(),
		Model:        model,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     duration,
		Success:      err == nil,
	}

	if response != nil {
		metrics.TokenUsage = response.Usage
	}

	if err != nil {
		var llmErr *LLMError
		if errors.As(err, &llmErr) {
			metrics.ErrorCode = llmErr.Code
			metrics.ErrorMessage = llmErr.Message
		} else {
			metrics.ErrorCode = "UNKNOWN_ERROR"
			metrics.ErrorMessage = err.Error()
		}
	}

	// 使用监控管理器记录指标（包含成本计算和告警检查）
	if m.monitoring != nil {
		m.monitoring.RecordCall(metrics)
	} else if m.metrics != nil {
		m.metrics.RecordCall(metrics)
	}
}

// recordStreamMetrics 记录流式请求指标
func (m *Manager) recordStreamMetrics(callID uuid.UUID, provider LLMProvider, model string, startTime time.Time, usage *Usage, streamErr *StreamError) {
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	metrics := &CallMetrics{
		ID:           callID,
		ProviderType: provider.GetProviderType(),
		Model:        model,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     duration,
		Success:      streamErr == nil,
	}

	if usage != nil {
		metrics.TokenUsage = *usage
	}

	if streamErr != nil {
		metrics.ErrorCode = streamErr.Code
		metrics.ErrorMessage = streamErr.Message
	}

	// 使用监控管理器记录指标
	if m.monitoring != nil {
		m.monitoring.RecordCall(metrics)
	} else if m.metrics != nil {
		m.metrics.RecordCall(metrics)
	}
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	RecordCall(metrics *CallMetrics)
	GetMetrics() interface{}
}

// DefaultMetricsCollector 默认指标收集器
type DefaultMetricsCollector struct {
	mu      sync.RWMutex
	calls   []*CallMetrics
	maxSize int
}

// NewDefaultMetricsCollector 创建默认指标收集器
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		calls:   make([]*CallMetrics, 0),
		maxSize: 1000, // 最多保存1000条记录
	}
}

// RecordCall 记录调用
func (c *DefaultMetricsCollector) RecordCall(metrics *CallMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls = append(c.calls, metrics)

	// 保持最大大小限制
	if len(c.calls) > c.maxSize {
		c.calls = c.calls[len(c.calls)-c.maxSize:]
	}
}

// GetMetrics 获取指标
func (c *DefaultMetricsCollector) GetMetrics() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 计算统计信息
	stats := make(map[string]interface{})
	
	totalCalls := len(c.calls)
	successCalls := 0
	var totalDuration time.Duration
	var totalTokens int

	providerStats := make(map[ProviderType]map[string]interface{})
	modelStats := make(map[string]map[string]interface{})

	for _, call := range c.calls {
		if call.Success {
			successCalls++
		}
		totalDuration += call.Duration
		totalTokens += call.TokenUsage.TotalTokens

		// 提供商统计
		if _, exists := providerStats[call.ProviderType]; !exists {
			providerStats[call.ProviderType] = map[string]interface{}{
				"total_calls":    0,
				"success_calls":  0,
				"total_duration": time.Duration(0),
				"total_tokens":   0,
			}
		}
		providerStats[call.ProviderType]["total_calls"] = providerStats[call.ProviderType]["total_calls"].(int) + 1
		if call.Success {
			providerStats[call.ProviderType]["success_calls"] = providerStats[call.ProviderType]["success_calls"].(int) + 1
		}
		providerStats[call.ProviderType]["total_duration"] = providerStats[call.ProviderType]["total_duration"].(time.Duration) + call.Duration
		providerStats[call.ProviderType]["total_tokens"] = providerStats[call.ProviderType]["total_tokens"].(int) + call.TokenUsage.TotalTokens

		// 模型统计
		if _, exists := modelStats[call.Model]; !exists {
			modelStats[call.Model] = map[string]interface{}{
				"total_calls":    0,
				"success_calls":  0,
				"total_duration": time.Duration(0),
				"total_tokens":   0,
			}
		}
		modelStats[call.Model]["total_calls"] = modelStats[call.Model]["total_calls"].(int) + 1
		if call.Success {
			modelStats[call.Model]["success_calls"] = modelStats[call.Model]["success_calls"].(int) + 1
		}
		modelStats[call.Model]["total_duration"] = modelStats[call.Model]["total_duration"].(time.Duration) + call.Duration
		modelStats[call.Model]["total_tokens"] = modelStats[call.Model]["total_tokens"].(int) + call.TokenUsage.TotalTokens
	}

	stats["total_calls"] = totalCalls
	stats["success_calls"] = successCalls
	stats["success_rate"] = float64(successCalls) / float64(totalCalls)
	if totalCalls > 0 {
		stats["avg_duration"] = totalDuration / time.Duration(totalCalls)
		stats["avg_tokens"] = totalTokens / totalCalls
	}
	stats["provider_stats"] = providerStats
	stats["model_stats"] = modelStats

	return stats
}

// DefaultRateLimiter 默认速率限制器
type DefaultRateLimiter struct {
	// 简单实现，实际应该使用更复杂的算法
}

// NewDefaultRateLimiter 创建默认速率限制器
func NewDefaultRateLimiter() *DefaultRateLimiter {
	return &DefaultRateLimiter{}
}

// Allow 检查是否允许请求
func (l *DefaultRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// 简单实现，总是允许
	return true, nil
}

// Wait 等待直到可以发送请求
func (l *DefaultRateLimiter) Wait(ctx context.Context, key string) error {
	// 简单实现，不等待
	return nil
}

// DefaultCircuitBreaker 默认熔断器
type DefaultCircuitBreaker struct {
	// 简单实现，实际应该使用更复杂的熔断逻辑
}

// NewDefaultCircuitBreaker 创建默认熔断器
func NewDefaultCircuitBreaker() *DefaultCircuitBreaker {
	return &DefaultCircuitBreaker{}
}

// Call 执行调用
func (b *DefaultCircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// 简单实现，直接调用
	return fn()
}

// State 获取熔断器状态
func (b *DefaultCircuitBreaker) State() string {
	return "closed"
}

// GetDefaultManagerConfig 获取默认管理器配置
func GetDefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		EnablePrometheus:        true,
		RateLimiterType:        "token_bucket",
		RateLimitConfig:        GetDefaultRateLimitConfig(),
		SlidingWindowConfig:    GetDefaultSlidingWindowConfig(),
		AdaptiveConfig:         GetDefaultAdaptiveConfig(),
		CircuitBreakerSettings: DefaultCircuitBreakerSettings,
	}
}