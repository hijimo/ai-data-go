package llm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMetricsCollector(t *testing.T) {
	collector := NewPrometheusMetricsCollector()
	assert.NotNil(t, collector)

	// 测试记录调用指标
	metrics := &CallMetrics{
		ID:           uuid.New(),
		ProviderType: ProviderOpenAI,
		Model:        "gpt-3.5-turbo",
		StartTime:    time.Now().Add(-time.Second),
		EndTime:      time.Now(),
		Duration:     time.Second,
		TokenUsage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Success: true,
		Cost:    0.001,
	}

	// 记录指标不应该出错
	assert.NotPanics(t, func() {
		collector.RecordCall(metrics)
	})

	// 测试记录活跃请求
	assert.NotPanics(t, func() {
		collector.RecordActiveRequest(ProviderOpenAI, "gpt-3.5-turbo", 1)
		collector.RecordActiveRequest(ProviderOpenAI, "gpt-3.5-turbo", -1)
	})

	// 测试记录速率限制
	assert.NotPanics(t, func() {
		collector.RecordRateLimitHit(ProviderOpenAI)
	})

	// 测试记录熔断器状态
	assert.NotPanics(t, func() {
		collector.RecordCircuitBreakerState(ProviderOpenAI, CircuitBreakerClosed)
		collector.RecordCircuitBreakerState(ProviderOpenAI, CircuitBreakerOpen)
		collector.RecordCircuitBreakerState(ProviderOpenAI, CircuitBreakerHalfOpen)
	})
}

func TestDefaultCostCalculator(t *testing.T) {
	calculator := NewDefaultCostCalculator()
	assert.NotNil(t, calculator)

	// 测试计算OpenAI GPT-3.5成本
	usage := Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	cost := calculator.CalculateCost(ProviderOpenAI, "gpt-3.5-turbo", usage)
	expectedCost := (1000.0/1000.0)*0.0015 + (500.0/1000.0)*0.002 // 输入 + 输出成本
	assert.Equal(t, expectedCost, cost)

	// 测试计算千问成本
	cost = calculator.CalculateCost(ProviderQianwen, "qwen-turbo", usage)
	expectedCost = (1000.0/1000.0)*0.0008 + (500.0/1000.0)*0.002
	assert.Equal(t, expectedCost, cost)

	// 测试未知模型
	cost = calculator.CalculateCost(ProviderOpenAI, "unknown-model", usage)
	assert.Equal(t, 0.0, cost)
}

func TestCostCalculatorUpdatePricing(t *testing.T) {
	calculator := NewDefaultCostCalculator()

	// 更新价格
	newPricing := ModelPricing{
		InputPrice:  0.01,
		OutputPrice: 0.02,
		Currency:    "USD",
	}
	calculator.UpdatePricing(ProviderOpenAI, "custom-model", newPricing)

	// 获取价格
	pricing, exists := calculator.GetPricing(ProviderOpenAI, "custom-model")
	assert.True(t, exists)
	assert.Equal(t, newPricing, pricing)

	// 计算成本
	usage := Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}
	cost := calculator.CalculateCost(ProviderOpenAI, "custom-model", usage)
	expectedCost := (1000.0/1000.0)*0.01 + (500.0/1000.0)*0.02
	assert.Equal(t, expectedCost, cost)
}

func TestMonitoringManager(t *testing.T) {
	metricsCollector := NewDefaultMetricsCollector()
	costCalculator := NewDefaultCostCalculator()
	alertManager := NewDefaultAlertManager()

	manager := NewMonitoringManager(metricsCollector, costCalculator, alertManager)
	assert.NotNil(t, manager)

	// 测试记录调用
	metrics := &CallMetrics{
		ID:           uuid.New(),
		ProviderType: ProviderOpenAI,
		Model:        "gpt-3.5-turbo",
		StartTime:    time.Now().Add(-time.Second),
		EndTime:      time.Now(),
		Duration:     time.Second,
		TokenUsage: Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		},
		Success: true,
	}

	assert.NotPanics(t, func() {
		manager.RecordCall(metrics)
	})

	// 验证成本已计算
	assert.Greater(t, metrics.Cost, 0.0)
}

func TestDefaultAlertManager(t *testing.T) {
	alertManager := NewDefaultAlertManager()
	assert.NotNil(t, alertManager)

	// 测试各种告警检查方法不会panic
	assert.NotPanics(t, func() {
		alertManager.CheckErrorRate(ProviderOpenAI, "gpt-3.5-turbo")
	})

	assert.NotPanics(t, func() {
		alertManager.CheckResponseTime(ProviderOpenAI, "gpt-3.5-turbo", 30*time.Second)
	})

	assert.NotPanics(t, func() {
		alertManager.CheckCost(ProviderOpenAI, "gpt-3.5-turbo", 1.5)
	})

	assert.NotPanics(t, func() {
		alertManager.CheckRateLimit(ProviderOpenAI)
	})
}

func TestCircuitBreakerState(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected float64
	}{
		{CircuitBreakerClosed, 0},
		{CircuitBreakerOpen, 1},
		{CircuitBreakerHalfOpen, 2},
	}

	collector := NewPrometheusMetricsCollector()

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordCircuitBreakerState(ProviderOpenAI, tt.state)
			})
		})
	}
}

func TestCallMetricsWithError(t *testing.T) {
	collector := NewDefaultMetricsCollector()

	// 测试记录失败的调用
	metrics := &CallMetrics{
		ID:           uuid.New(),
		ProviderType: ProviderOpenAI,
		Model:        "gpt-3.5-turbo",
		StartTime:    time.Now().Add(-time.Second),
		EndTime:      time.Now(),
		Duration:     time.Second,
		TokenUsage: Usage{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
		Success:      false,
		ErrorCode:    "RATE_LIMIT_EXCEEDED",
		ErrorMessage: "Rate limit exceeded",
	}

	collector.RecordCall(metrics)

	// 获取统计信息
	stats := collector.GetMetrics()
	assert.NotNil(t, stats)

	statsMap, ok := stats.(map[string]interface{})
	assert.True(t, ok)

	// 验证统计信息
	assert.Equal(t, 1, statsMap["total_calls"])
	assert.Equal(t, 0, statsMap["success_calls"])
	assert.Equal(t, 0.0, statsMap["success_rate"])
}