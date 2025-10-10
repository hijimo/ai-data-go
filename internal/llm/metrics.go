package llm

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetricsCollector Prometheus指标收集器
type PrometheusMetricsCollector struct {
	// 请求计数器
	requestsTotal *prometheus.CounterVec
	
	// 请求持续时间直方图
	requestDuration *prometheus.HistogramVec
	
	// Token使用量计数器
	tokensTotal *prometheus.CounterVec
	
	// 成本计数器
	costTotal *prometheus.CounterVec
	
	// 错误计数器
	errorsTotal *prometheus.CounterVec
	
	// 当前活跃请求数
	activeRequests *prometheus.GaugeVec
	
	// 速率限制计数器
	rateLimitHits *prometheus.CounterVec
	
	// 熔断器状态
	circuitBreakerState *prometheus.GaugeVec
}

// NewPrometheusMetricsCollector 创建Prometheus指标收集器
func NewPrometheusMetricsCollector() *PrometheusMetricsCollector {
	return &PrometheusMetricsCollector{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_requests_total",
				Help: "LLM请求总数",
			},
			[]string{"provider", "model", "status"},
		),
		
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "llm_request_duration_seconds",
				Help:    "LLM请求持续时间（秒）",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "model"},
		),
		
		tokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_tokens_total",
				Help: "LLM Token使用总数",
			},
			[]string{"provider", "model", "type"}, // type: prompt, completion, total
		),
		
		costTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_cost_total",
				Help: "LLM调用成本总计",
			},
			[]string{"provider", "model", "currency"},
		),
		
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_errors_total",
				Help: "LLM错误总数",
			},
			[]string{"provider", "model", "error_code"},
		),
		
		activeRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llm_active_requests",
				Help: "当前活跃的LLM请求数",
			},
			[]string{"provider", "model"},
		),
		
		rateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_rate_limit_hits_total",
				Help: "速率限制触发次数",
			},
			[]string{"provider"},
		),
		
		circuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llm_circuit_breaker_state",
				Help: "熔断器状态 (0=closed, 1=open, 2=half-open)",
			},
			[]string{"provider"},
		),
	}
}

// RecordCall 记录调用指标
func (c *PrometheusMetricsCollector) RecordCall(metrics *CallMetrics) {
	provider := string(metrics.ProviderType)
	model := metrics.Model
	
	// 记录请求总数
	status := "success"
	if !metrics.Success {
		status = "error"
	}
	c.requestsTotal.WithLabelValues(provider, model, status).Inc()
	
	// 记录请求持续时间
	c.requestDuration.WithLabelValues(provider, model).Observe(metrics.Duration.Seconds())
	
	// 记录Token使用量
	if metrics.TokenUsage.PromptTokens > 0 {
		c.tokensTotal.WithLabelValues(provider, model, "prompt").Add(float64(metrics.TokenUsage.PromptTokens))
	}
	if metrics.TokenUsage.CompletionTokens > 0 {
		c.tokensTotal.WithLabelValues(provider, model, "completion").Add(float64(metrics.TokenUsage.CompletionTokens))
	}
	if metrics.TokenUsage.TotalTokens > 0 {
		c.tokensTotal.WithLabelValues(provider, model, "total").Add(float64(metrics.TokenUsage.TotalTokens))
	}
	
	// 记录成本
	if metrics.Cost > 0 {
		currency := "USD" // 默认货币
		c.costTotal.WithLabelValues(provider, model, currency).Add(metrics.Cost)
	}
	
	// 记录错误
	if !metrics.Success && metrics.ErrorCode != "" {
		c.errorsTotal.WithLabelValues(provider, model, metrics.ErrorCode).Inc()
	}
}

// RecordActiveRequest 记录活跃请求
func (c *PrometheusMetricsCollector) RecordActiveRequest(provider ProviderType, model string, delta float64) {
	c.activeRequests.WithLabelValues(string(provider), model).Add(delta)
}

// RecordRateLimitHit 记录速率限制触发
func (c *PrometheusMetricsCollector) RecordRateLimitHit(provider ProviderType) {
	c.rateLimitHits.WithLabelValues(string(provider)).Inc()
}

// RecordCircuitBreakerState 记录熔断器状态
func (c *PrometheusMetricsCollector) RecordCircuitBreakerState(provider ProviderType, state CircuitBreakerState) {
	var stateValue float64
	switch state {
	case CircuitBreakerClosed:
		stateValue = 0
	case CircuitBreakerOpen:
		stateValue = 1
	case CircuitBreakerHalfOpen:
		stateValue = 2
	}
	c.circuitBreakerState.WithLabelValues(string(provider)).Set(stateValue)
}

// GetMetrics 获取指标（实现MetricsCollector接口）
func (c *PrometheusMetricsCollector) GetMetrics() interface{} {
	// Prometheus指标通过HTTP端点暴露，这里返回空
	return nil
}

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CostCalculator 成本计算器接口
type CostCalculator interface {
	CalculateCost(providerType ProviderType, model string, usage Usage) float64
}

// DefaultCostCalculator 默认成本计算器
type DefaultCostCalculator struct {
	mu     sync.RWMutex
	prices map[string]ModelPricing
}

// NewDefaultCostCalculator 创建默认成本计算器
func NewDefaultCostCalculator() *DefaultCostCalculator {
	calculator := &DefaultCostCalculator{
		prices: make(map[string]ModelPricing),
	}
	
	// 初始化一些常见模型的价格
	calculator.initDefaultPrices()
	
	return calculator
}

// initDefaultPrices 初始化默认价格
func (c *DefaultCostCalculator) initDefaultPrices() {
	// OpenAI价格 (USD per 1K tokens)
	c.prices["openai:gpt-4"] = ModelPricing{
		InputPrice:  0.03,
		OutputPrice: 0.06,
		Currency:    "USD",
	}
	c.prices["openai:gpt-4-turbo"] = ModelPricing{
		InputPrice:  0.01,
		OutputPrice: 0.03,
		Currency:    "USD",
	}
	c.prices["openai:gpt-4o"] = ModelPricing{
		InputPrice:  0.005,
		OutputPrice: 0.015,
		Currency:    "USD",
	}
	c.prices["openai:gpt-4o-mini"] = ModelPricing{
		InputPrice:  0.00015,
		OutputPrice: 0.0006,
		Currency:    "USD",
	}
	c.prices["openai:gpt-3.5-turbo"] = ModelPricing{
		InputPrice:  0.0015,
		OutputPrice: 0.002,
		Currency:    "USD",
	}
	
	// 千问价格 (CNY per 1K tokens)
	c.prices["qianwen:qwen-turbo"] = ModelPricing{
		InputPrice:  0.0008,
		OutputPrice: 0.002,
		Currency:    "CNY",
	}
	c.prices["qianwen:qwen-plus"] = ModelPricing{
		InputPrice:  0.004,
		OutputPrice: 0.012,
		Currency:    "CNY",
	}
	c.prices["qianwen:qwen-max"] = ModelPricing{
		InputPrice:  0.02,
		OutputPrice: 0.06,
		Currency:    "CNY",
	}
	
	// Claude价格 (USD per 1K tokens)
	c.prices["claude:claude-3-5-sonnet-20241022"] = ModelPricing{
		InputPrice:  0.003,
		OutputPrice: 0.015,
		Currency:    "USD",
	}
	c.prices["claude:claude-3-opus-20240229"] = ModelPricing{
		InputPrice:  0.015,
		OutputPrice: 0.075,
		Currency:    "USD",
	}
	c.prices["claude:claude-3-haiku-20240307"] = ModelPricing{
		InputPrice:  0.00025,
		OutputPrice: 0.00125,
		Currency:    "USD",
	}
}

// CalculateCost 计算成本
func (c *DefaultCostCalculator) CalculateCost(providerType ProviderType, model string, usage Usage) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key := string(providerType) + ":" + model
	pricing, exists := c.prices[key]
	if !exists {
		return 0 // 未知模型，返回0成本
	}
	
	// 计算输入和输出成本
	inputCost := float64(usage.PromptTokens) / 1000.0 * pricing.InputPrice
	outputCost := float64(usage.CompletionTokens) / 1000.0 * pricing.OutputPrice
	
	return inputCost + outputCost
}

// UpdatePricing 更新模型价格
func (c *DefaultCostCalculator) UpdatePricing(providerType ProviderType, model string, pricing ModelPricing) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := string(providerType) + ":" + model
	c.prices[key] = pricing
}

// GetPricing 获取模型价格
func (c *DefaultCostCalculator) GetPricing(providerType ProviderType, model string) (ModelPricing, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key := string(providerType) + ":" + model
	pricing, exists := c.prices[key]
	return pricing, exists
}

// MonitoringManager 监控管理器
type MonitoringManager struct {
	metricsCollector MetricsCollector
	costCalculator   CostCalculator
	alertManager     AlertManager
}

// NewMonitoringManager 创建监控管理器
func NewMonitoringManager(metricsCollector MetricsCollector, costCalculator CostCalculator, alertManager AlertManager) *MonitoringManager {
	return &MonitoringManager{
		metricsCollector: metricsCollector,
		costCalculator:   costCalculator,
		alertManager:     alertManager,
	}
}

// RecordCall 记录调用（增强版）
func (m *MonitoringManager) RecordCall(metrics *CallMetrics) {
	// 计算成本
	if m.costCalculator != nil {
		metrics.Cost = m.costCalculator.CalculateCost(metrics.ProviderType, metrics.Model, metrics.TokenUsage)
	}
	
	// 记录指标
	if m.metricsCollector != nil {
		m.metricsCollector.RecordCall(metrics)
	}
	
	// 检查告警条件
	if m.alertManager != nil {
		m.checkAlerts(metrics)
	}
}

// checkAlerts 检查告警条件
func (m *MonitoringManager) checkAlerts(metrics *CallMetrics) {
	// 检查错误率告警
	if !metrics.Success {
		m.alertManager.CheckErrorRate(metrics.ProviderType, metrics.Model)
	}
	
	// 检查响应时间告警
	if metrics.Duration > 30*time.Second {
		m.alertManager.CheckResponseTime(metrics.ProviderType, metrics.Model, metrics.Duration)
	}
	
	// 检查成本告警
	if metrics.Cost > 1.0 { // 单次调用成本超过1美元
		m.alertManager.CheckCost(metrics.ProviderType, metrics.Model, metrics.Cost)
	}
}

// AlertManager 告警管理器接口
type AlertManager interface {
	CheckErrorRate(provider ProviderType, model string)
	CheckResponseTime(provider ProviderType, model string, duration time.Duration)
	CheckCost(provider ProviderType, model string, cost float64)
	CheckRateLimit(provider ProviderType)
}

// DefaultAlertManager 默认告警管理器
type DefaultAlertManager struct {
	// 这里可以集成实际的告警系统，如PagerDuty、钉钉等
}

// NewDefaultAlertManager 创建默认告警管理器
func NewDefaultAlertManager() *DefaultAlertManager {
	return &DefaultAlertManager{}
}

// CheckErrorRate 检查错误率
func (a *DefaultAlertManager) CheckErrorRate(provider ProviderType, model string) {
	// TODO: 实现错误率检查逻辑
	// 可以查询最近一段时间的错误率，如果超过阈值则发送告警
}

// CheckResponseTime 检查响应时间
func (a *DefaultAlertManager) CheckResponseTime(provider ProviderType, model string, duration time.Duration) {
	// TODO: 实现响应时间检查逻辑
	// 可以检查响应时间是否超过阈值
}

// CheckCost 检查成本
func (a *DefaultAlertManager) CheckCost(provider ProviderType, model string, cost float64) {
	// TODO: 实现成本检查逻辑
	// 可以检查单次调用成本或累计成本是否超过阈值
}

// CheckRateLimit 检查速率限制
func (a *DefaultAlertManager) CheckRateLimit(provider ProviderType) {
	// TODO: 实现速率限制检查逻辑
	// 可以检查速率限制触发频率
}