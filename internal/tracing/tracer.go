// Package tracing 提供 OpenTelemetry 分布式追踪功能
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName 追踪器名称
	TracerName = "genkit-flows"
)

// TraceFlow 追踪 Flow 执行
// 为 Flow 创建一个 Span，记录执行过程和结果
func TraceFlow(
	ctx context.Context,
	flowName string,
	fn func(context.Context) error,
) error {
	tracer := otel.Tracer(TracerName)
	ctx, span := tracer.Start(ctx, flowName)
	defer span.End()

	// 添加基本属性
	span.SetAttributes(
		attribute.String("flow.name", flowName),
		attribute.String("component", "genkit-flow"),
	)

	// 从上下文提取信息并添加到 Span
	addContextAttributes(ctx, span)

	// 执行 Flow
	err := fn(ctx)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TraceService 追踪服务层方法执行
// 为服务方法创建一个 Span，记录执行过程和结果
func TraceService(
	ctx context.Context,
	serviceName string,
	methodName string,
	fn func(context.Context) error,
) error {
	tracer := otel.Tracer(TracerName)
	spanName := fmt.Sprintf("%s.%s", serviceName, methodName)
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	// 添加属性
	span.SetAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("method.name", methodName),
		attribute.String("component", "service"),
	)

	// 从上下文提取信息
	addContextAttributes(ctx, span)

	// 执行方法
	err := fn(ctx)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TraceRepository 追踪数据库操作
// 为数据库操作创建一个 Span，记录查询和结果
func TraceRepository(
	ctx context.Context,
	repoName string,
	operation string,
	fn func(context.Context) error,
) error {
	tracer := otel.Tracer(TracerName)
	spanName := fmt.Sprintf("%s.%s", repoName, operation)
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	// 添加属性
	span.SetAttributes(
		attribute.String("db.repository", repoName),
		attribute.String("db.operation", operation),
		attribute.String("component", "repository"),
	)

	// 从上下文提取信息
	addContextAttributes(ctx, span)

	// 执行操作
	err := fn(ctx)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TraceExternalCall 追踪外部服务调用
// 为外部服务调用创建一个 Span，记录请求和响应
func TraceExternalCall(
	ctx context.Context,
	serviceName string,
	operation string,
	fn func(context.Context) error,
) error {
	tracer := otel.Tracer(TracerName)
	spanName := fmt.Sprintf("external.%s.%s", serviceName, operation)
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	// 添加属性
	span.SetAttributes(
		attribute.String("external.service", serviceName),
		attribute.String("external.operation", operation),
		attribute.String("component", "external"),
	)

	// 从上下文提取信息
	addContextAttributes(ctx, span)

	// 执行调用
	err := fn(ctx)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// AddSpanAttributes 向当前 Span 添加属性
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent 向当前 Span 添加事件
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError 记录错误到当前 Span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// addContextAttributes 从上下文提取信息并添加到 Span
func addContextAttributes(ctx context.Context, span trace.Span) {
	// 提取会话 ID
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			span.SetAttributes(attribute.String("session.id", sid))
		}
	}

	// 提取用户 ID
	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			span.SetAttributes(attribute.String("user.id", uid))
		}
	}

	// 提取租户 ID
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		if tid, ok := tenantID.(string); ok {
			span.SetAttributes(attribute.String("tenant.id", tid))
		}
	}

	// 提取 TraceID（用于日志关联）
	if traceID := ctx.Value("traceId"); traceID != nil {
		if tid, ok := traceID.(string); ok {
			span.SetAttributes(attribute.String("trace.id", tid))
		}
	}
}
