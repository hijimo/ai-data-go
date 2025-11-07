// Package tracing 提供分布式追踪功能
package tracing

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/trace"
)

// InitTracer 从环境变量初始化追踪器
func InitTracer() (Tracer, error) {
	config := &TracerConfig{
		ServiceName:    getEnv("TRACING_SERVICE_NAME", "genkit-ai-service"),
		ServiceVersion: getEnv("TRACING_SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("TRACING_ENVIRONMENT", "development"),
		OTLPEndpoint:   getEnv("OTLP_ENDPOINT", "localhost:4318"),
		SamplingRate:   getEnvAsFloat("TRACING_SAMPLING_RATE", 1.0),
	}

	// 检查是否启用追踪
	if !getEnvAsBool("TRACING_ENABLED", true) {
		log.Println("追踪功能已禁用")
		return NewNoOpTracer(), nil
	}

	tracer, err := NewTracer(config)
	if err != nil {
		return nil, fmt.Errorf("初始化追踪器失败: %w", err)
	}

	log.Printf("追踪器已初始化: service=%s, environment=%s, endpoint=%s, sampling=%.2f",
		config.ServiceName,
		config.Environment,
		config.OTLPEndpoint,
		config.SamplingRate,
	)

	return tracer, nil
}

// InitTracerWithConfig 使用自定义配置初始化追踪器
func InitTracerWithConfig(config *TracerConfig) (Tracer, error) {
	return NewTracer(config)
}

// NoOpTracer 空操作追踪器（用于禁用追踪时）
type NoOpTracer struct{}

// NewNoOpTracer 创建空操作追踪器
func NewNoOpTracer() Tracer {
	return &NoOpTracer{}
}

// TraceFlow 空操作实现
func (t *NoOpTracer) TraceFlow(ctx context.Context, flowName string, fn func(context.Context) error) error {
	return fn(ctx)
}

// StartSpan 空操作实现
func (t *NoOpTracer) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// 使用 OpenTelemetry 的 noop span
	return ctx, trace.SpanFromContext(ctx)
}

// Shutdown 空操作实现
func (t *NoOpTracer) Shutdown(ctx context.Context) error {
	return nil
}



// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
