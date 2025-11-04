// Package tracing 提供 OpenTelemetry 配置和初始化
package tracing

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config 追踪配置
type Config struct {
	// Enabled 是否启用追踪
	Enabled bool
	// ServiceName 服务名称
	ServiceName string
	// ServiceVersion 服务版本
	ServiceVersion string
	// Environment 环境（dev, staging, prod）
	Environment string
	// OTLPEndpoint OTLP 收集器端点（支持 Jaeger 0.14+ 的 OTLP 接收器）
	OTLPEndpoint string
	// SamplingRate 采样率（0.0-1.0）
	SamplingRate float64
}

// TracerProvider 追踪提供者
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	config   *Config
}

// NewTracerProvider 创建追踪提供者
func NewTracerProvider(config *Config) (*TracerProvider, error) {
	if !config.Enabled {
		log.Println("追踪功能已禁用")
		return &TracerProvider{
			provider: nil,
			config:   config,
		}, nil
	}

	// 创建 OTLP gRPC 导出器（支持 Jaeger 0.14+ 的 OTLP 接收器）
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 OTLP 导出器失败: %w", err)
	}

	// 创建资源
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("创建资源失败: %w", err)
	}

	// 创建采样器
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(config.SamplingRate),
	)

	// 创建追踪提供者
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局追踪提供者
	otel.SetTracerProvider(provider)

	// 设置全局传播器
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	log.Printf("追踪功能已启用: service=%s, endpoint=%s, sampling=%.2f",
		config.ServiceName, config.OTLPEndpoint, config.SamplingRate)

	return &TracerProvider{
		provider: provider,
		config:   config,
	}, nil
}

// Shutdown 关闭追踪提供者
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}

	log.Println("正在关闭追踪提供者...")
	if err := tp.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("关闭追踪提供者失败: %w", err)
	}

	log.Println("追踪提供者已关闭")
	return nil
}

// ForceFlush 强制刷新所有待处理的 Span
func (tp *TracerProvider) ForceFlush(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}

	return tp.provider.ForceFlush(ctx)
}

// IsEnabled 返回追踪是否启用
func (tp *TracerProvider) IsEnabled() bool {
	return tp.config.Enabled && tp.provider != nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:        false,
		ServiceName:    "genkit-ai-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4317", // Jaeger OTLP gRPC 端口
		SamplingRate:   1.0,               // 开发环境全采样
	}
}

// ProductionConfig 返回生产环境配置
func ProductionConfig(otlpEndpoint string) *Config {
	return &Config{
		Enabled:        true,
		ServiceName:    "genkit-ai-service",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		OTLPEndpoint:   otlpEndpoint,
		SamplingRate:   0.1, // 生产环境 10% 采样
	}
}
