package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"genkit-ai-service/internal/monitoring"
)

func TestFlowMonitor_MonitorFlow_Success(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	monitor := NewFlowMonitor(metrics)

	// 创建测试上下文
	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 创建测试 Flow
	testFlow := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// 包装 Flow
	monitoredFlow := monitor.MonitorFlow("testFlow", testFlow)

	// 执行 Flow
	err := monitoredFlow(ctx)

	// 验证
	assert.NoError(t, err)
}

func TestFlowMonitor_MonitorFlow_Error(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	monitor := NewFlowMonitor(metrics)

	// 创建测试上下文
	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 创建会失败的测试 Flow
	testFlow := func(ctx context.Context) error {
		return errors.New("test error")
	}

	// 包装 Flow
	monitoredFlow := monitor.MonitorFlow("testFlow", testFlow)

	// 执行 Flow
	err := monitoredFlow(ctx)

	// 验证
	assert.Error(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestFlowMonitor_MonitorFlowWithMetrics(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	monitor := NewFlowMonitor(metrics)

	// 创建测试上下文
	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 创建测试 Flow
	testFlow := func(ctx context.Context) (interface{}, error) {
		return map[string]interface{}{
			"session_id":    "session-123",
			"total_tokens":  1000,
			"quality_score": 0.85,
		}, nil
	}

	// 创建指标记录器
	metricsRecorded := false
	metricsRecorder := func(ctx context.Context, result interface{}, err error) {
		metricsRecorded = true
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	// 包装 Flow
	monitoredFlow := monitor.MonitorFlowWithMetrics("testFlow", testFlow, metricsRecorder)

	// 执行 Flow
	result, err := monitoredFlow(ctx)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, metricsRecorded)
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "timeout error",
			err:      errors.New("operation timeout"),
			expected: "timeout",
		},
		{
			name:     "deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: "timeout",
		},
		{
			name:     "permission error",
			err:      errors.New("permission denied"),
			expected: "permission",
		},
		{
			name:     "forbidden error",
			err:      errors.New("forbidden access"),
			expected: "permission",
		},
		{
			name:     "validation error",
			err:      errors.New("validation failed"),
			expected: "validation",
		},
		{
			name:     "not found error",
			err:      errors.New("resource not found"),
			expected: "not_found",
		},
		{
			name:     "ai service error",
			err:      errors.New("ai service unavailable"),
			expected: "ai_service",
		},
		{
			name:     "database error",
			err:      errors.New("database connection failed"),
			expected: "database",
		},
		{
			name:     "cache error",
			err:      errors.New("cache miss"),
			expected: "cache",
		},
		{
			name:     "vector service error",
			err:      errors.New("vector generation failed"),
			expected: "vector_service",
		},
		{
			name:     "quota exceeded",
			err:      errors.New("quota limit exceeded"),
			expected: "quota_exceeded",
		},
		{
			name:     "unknown error",
			err:      errors.New("something went wrong"),
			expected: "unknown",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorType(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "with tenant ID",
			ctx:      context.WithValue(context.Background(), "tenant_id", "tenant-123"),
			expected: "tenant-123",
		},
		{
			name:     "without tenant ID",
			ctx:      context.Background(),
			expected: "unknown",
		},
		{
			name:     "with invalid tenant ID type",
			ctx:      context.WithValue(context.Background(), "tenant_id", 123),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTenantIDFromContext(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuildMetricsRecorder(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	recorder := NewContextBuildMetricsRecorder(metrics)

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 测试成功情况
	result := map[string]interface{}{
		"session_id":    "session-123",
		"total_tokens":  1000,
		"quality_score": 0.85,
	}

	recorder.RecordMetrics(ctx, result, nil)

	// 测试错误情况（不应记录指标）
	recorder.RecordMetrics(ctx, result, errors.New("test error"))

	// 测试无效结果类型
	recorder.RecordMetrics(ctx, "invalid", nil)
}

func TestTokenUsageMetricsRecorder(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	recorder := NewTokenUsageMetricsRecorder(metrics, "chatGenerateFlow")

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 测试成功情况
	result := map[string]interface{}{
		"token_usage": map[string]interface{}{
			"prompt_tokens":     1000,
			"completion_tokens": 500,
			"total_tokens":      1500,
		},
	}

	recorder.RecordMetrics(ctx, result, nil)

	// 测试错误情况
	recorder.RecordMetrics(ctx, result, errors.New("test error"))

	// 测试无效结果类型
	recorder.RecordMetrics(ctx, "invalid", nil)
}

func TestVectorSearchMetricsRecorder(t *testing.T) {
	metrics := monitoring.NewGenkitMetrics()
	recorder := NewVectorSearchMetricsRecorder(metrics)

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	// 模拟一些延迟
	time.Sleep(10 * time.Millisecond)

	// 测试成功情况
	result := map[string]interface{}{
		"returned_count": 5,
	}

	recorder.RecordMetrics(ctx, result, nil)

	// 测试错误情况
	recorder.RecordMetrics(ctx, result, errors.New("test error"))
}

func TestMonitoringContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")
	ctx = context.WithValue(ctx, "session_id", "session-123")

	// 创建监控上下文
	mc := NewMonitoringContext(ctx, "testFlow")

	// 验证初始化
	assert.Equal(t, "testFlow", mc.FlowName)
	assert.Equal(t, "tenant-123", mc.TenantID)
	assert.Equal(t, "session-123", mc.SessionID)
	assert.NotNil(t, mc.Metadata)

	// 添加元数据
	mc.AddMetadata("key1", "value1")
	mc.AddMetadata("key2", 123)

	assert.Equal(t, "value1", mc.Metadata["key1"])
	assert.Equal(t, 123, mc.Metadata["key2"])

	// 测试获取执行时间
	time.Sleep(10 * time.Millisecond)
	duration := mc.GetDuration()
	assert.Greater(t, duration.Milliseconds(), int64(0))

	// 测试记录完成日志
	mc.LogCompletion(ctx, nil)
	mc.LogCompletion(ctx, errors.New("test error"))
}

func TestGetStringValue(t *testing.T) {
	m := map[string]interface{}{
		"string_key": "value",
		"int_key":    123,
	}

	assert.Equal(t, "value", getStringValue(m, "string_key"))
	assert.Equal(t, "", getStringValue(m, "int_key"))
	assert.Equal(t, "", getStringValue(m, "missing_key"))
}

func TestGetIntValue(t *testing.T) {
	m := map[string]interface{}{
		"int_key":     123,
		"int64_key":   int64(456),
		"float64_key": float64(789),
		"string_key":  "not_a_number",
	}

	assert.Equal(t, 123, getIntValue(m, "int_key"))
	assert.Equal(t, 456, getIntValue(m, "int64_key"))
	assert.Equal(t, 789, getIntValue(m, "float64_key"))
	assert.Equal(t, 0, getIntValue(m, "string_key"))
	assert.Equal(t, 0, getIntValue(m, "missing_key"))
}

func TestGetFloatValue(t *testing.T) {
	m := map[string]interface{}{
		"float64_key": float64(1.23),
		"float32_key": float32(4.56),
		"int_key":     789,
		"int64_key":   int64(101112),
		"string_key":  "not_a_number",
	}

	assert.Equal(t, 1.23, getFloatValue(m, "float64_key"))
	assert.InDelta(t, 4.56, getFloatValue(m, "float32_key"), 0.01)
	assert.Equal(t, float64(789), getFloatValue(m, "int_key"))
	assert.Equal(t, float64(101112), getFloatValue(m, "int64_key"))
	assert.Equal(t, 0.0, getFloatValue(m, "string_key"))
	assert.Equal(t, 0.0, getFloatValue(m, "missing_key"))
}

func BenchmarkFlowMonitor_MonitorFlow(b *testing.B) {
	metrics := monitoring.NewGenkitMetrics()
	monitor := NewFlowMonitor(metrics)

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")

	testFlow := func(ctx context.Context) error {
		return nil
	}

	monitoredFlow := monitor.MonitorFlow("testFlow", testFlow)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitoredFlow(ctx)
	}
}

func BenchmarkGetErrorType(b *testing.B) {
	err := errors.New("test timeout error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getErrorType(err)
	}
}
