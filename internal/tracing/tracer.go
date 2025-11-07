// Package tracing 提供分布式追踪功能
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer 追踪器接口
type Tracer interface {
	// TraceFlow 追踪 Flow 执行
	TraceFlow(ctx context.Context, flowName string, fn func(context.Context) error) error

	// StartSpan 开始一个新的 span
	StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)

	// Shutdown 关闭追踪器
	Shutdown(ctx context.Context) error
}

// tracerImpl 追踪器实现
type tracerImpl struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// TracerConfig 追踪器配置
type TracerConfig struct {
	// ServiceName 服务名称
	ServiceName string
	// ServiceVersion 服务版本
	ServiceVersion string
	// Environment 环境（dev, staging, prod）
	Environment string
	// OTLPEndpoint OTLP 端点地址（支持 Jaeger、Tempo 等）
	OTLPEndpoint string
	// SamplingRate 采样率（0.0-1.0）
	SamplingRate float64
}

// NewTracer 创建新的追踪器
func NewTracer(config *TracerConfig) (Tracer, error) {
	// 创建 OTLP HTTP 导出器（兼容 Jaeger v1.35+）
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // 开发环境使用，生产环境应移除
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
	var sampler sdktrace.Sampler
	if config.SamplingRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRate)
	}

	// 创建追踪提供者
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局追踪提供者
	otel.SetTracerProvider(provider)

	// 创建追踪器
	tracer := provider.Tracer("genkit-flows")

	return &tracerImpl{
		provider: provider,
		tracer:   tracer,
	}, nil
}

// TraceFlow 追踪 Flow 执行
func (t *tracerImpl) TraceFlow(
	ctx context.Context,
	flowName string,
	fn func(context.Context) error,
) error {
	// 开始 span
	ctx, span := t.tracer.Start(ctx, flowName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	// 添加基本属性
	span.SetAttributes(
		attribute.String("flow.name", flowName),
		attribute.String("component", "genkit-flow"),
	)

	// 从上下文提取信息并添加到 span
	t.addContextAttributes(ctx, span)

	// 执行 Flow
	err := fn(ctx)

	// 记录执行结果
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.Bool("flow.success", false),
			attribute.String("flow.error", err.Error()),
		)
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(
			attribute.Bool("flow.success", true),
		)
	}

	return err
}

// StartSpan 开始一个新的 span
func (t *tracerImpl) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, spanName, opts...)

	// 添加上下文属性
	t.addContextAttributes(ctx, span)

	return ctx, span
}

// Shutdown 关闭追踪器
func (t *tracerImpl) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// addContextAttributes 从上下文中提取信息并添加到 span
func (t *tracerImpl) addContextAttributes(ctx context.Context, span trace.Span) {
	// 提取会话 ID
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			span.SetAttributes(attribute.String("session.id", sid))
		}
	}

	// 提取租户 ID
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		if tid, ok := tenantID.(string); ok {
			span.SetAttributes(attribute.String("tenant.id", tid))
		}
	}

	// 提取用户 ID
	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			span.SetAttributes(attribute.String("user.id", uid))
		}
	}

	// 提取请求 ID
	if requestID := ctx.Value("request_id"); requestID != nil {
		if rid, ok := requestID.(string); ok {
			span.SetAttributes(attribute.String("request.id", rid))
		}
	}

	// 提取追踪 ID
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if tid, ok := traceID.(string); ok {
			span.SetAttributes(attribute.String("trace.id", tid))
		}
	}
}

// TraceOperation 追踪一个操作（辅助函数）
func TraceOperation(
	ctx context.Context,
	operationName string,
	fn func(context.Context) error,
	attrs ...attribute.KeyValue,
) error {
	tracer := otel.Tracer("genkit-flows")
	ctx, span := tracer.Start(ctx, operationName)
	defer span.End()

	// 添加自定义属性
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}

	// 执行操作
	err := fn(ctx)

	// 记录结果
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceDBQuery 追踪数据库查询（辅助函数）
func TraceDBQuery(
	ctx context.Context,
	queryName string,
	query string,
	fn func(context.Context) error,
) error {
	return TraceOperation(ctx, fmt.Sprintf("db.%s", queryName), fn,
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
	)
}

// TraceVectorSearch 追踪向量检索（辅助函数）
func TraceVectorSearch(
	ctx context.Context,
	sessionID string,
	topK int,
	fn func(context.Context) error,
) error {
	return TraceOperation(ctx, "vector.search", fn,
		attribute.String("vector.session_id", sessionID),
		attribute.Int("vector.top_k", topK),
		attribute.String("vector.db", "qdrant"),
	)
}

// TraceAIGeneration 追踪 AI 生成（辅助函数）
func TraceAIGeneration(
	ctx context.Context,
	model string,
	promptTokens int,
	fn func(context.Context) error,
) error {
	return TraceOperation(ctx, "ai.generation", fn,
		attribute.String("ai.model", model),
		attribute.Int("ai.prompt_tokens", promptTokens),
		attribute.String("ai.provider", "genkit"),
	)
}

// TraceCacheOperation 追踪缓存操作（辅助函数）
func TraceCacheOperation(
	ctx context.Context,
	operation string,
	key string,
	fn func(context.Context) error,
) error {
	return TraceOperation(ctx, fmt.Sprintf("cache.%s", operation), fn,
		attribute.String("cache.key", key),
		attribute.String("cache.system", "redis"),
	)
}
