// Package tracing 提供追踪辅助函数
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan 开始一个新的 Span
// 返回新的 context 和 span，调用者需要在完成后调用 span.End()
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(TracerName)
	return tracer.Start(ctx, name, opts...)
}

// TraceDBQuery 追踪数据库查询
func TraceDBQuery(ctx context.Context, query string, args ...interface{}) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "db.query")
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", query),
		attribute.String("component", "database"),
	)
	return ctx, span
}

// TraceVectorSearch 追踪向量检索
func TraceVectorSearch(ctx context.Context, sessionID string, topK int) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "vector.search")
	span.SetAttributes(
		attribute.String("session.id", sessionID),
		attribute.Int("vector.topk", topK),
		attribute.String("component", "vector-service"),
	)
	return ctx, span
}

// TraceAIGeneration 追踪 AI 生成
func TraceAIGeneration(ctx context.Context, model string, promptTokens int) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "ai.generation")
	span.SetAttributes(
		attribute.String("ai.model", model),
		attribute.Int("ai.prompt_tokens", promptTokens),
		attribute.String("component", "ai-service"),
	)
	return ctx, span
}

// TraceCacheOperation 追踪缓存操作
func TraceCacheOperation(ctx context.Context, operation string, key string) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, fmt.Sprintf("cache.%s", operation))
	span.SetAttributes(
		attribute.String("cache.operation", operation),
		attribute.String("cache.key", key),
		attribute.String("component", "cache"),
	)
	return ctx, span
}

// SetSpanSuccess 设置 Span 为成功状态
func SetSpanSuccess(span trace.Span) {
	if span.IsRecording() {
		span.SetStatus(codes.Ok, "success")
	}
}

// SetSpanError 设置 Span 为错误状态
func SetSpanError(span trace.Span, err error) {
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("error", true))
	}
}

// AddTokenUsage 添加 Token 使用统计到 Span
func AddTokenUsage(ctx context.Context, promptTokens, completionTokens, totalTokens int) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("tokens.prompt", promptTokens),
			attribute.Int("tokens.completion", completionTokens),
			attribute.Int("tokens.total", totalTokens),
		)
	}
}

// AddContextMetrics 添加上下文指标到 Span
func AddContextMetrics(ctx context.Context, totalTokens int, qualityScore float64, strategy string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("context.tokens", totalTokens),
			attribute.Float64("context.quality_score", qualityScore),
			attribute.String("context.strategy", strategy),
		)
	}
}

// AddMemoryMetrics 添加记忆指标到 Span
func AddMemoryMetrics(ctx context.Context, memoryCount int, avgSimilarity float64) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("memory.count", memoryCount),
			attribute.Float64("memory.avg_similarity", avgSimilarity),
		)
	}
}

// AddSummaryMetrics 添加摘要指标到 Span
func AddSummaryMetrics(ctx context.Context, messageCount int, compressionRate float64, qualityScore float64) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("summary.message_count", messageCount),
			attribute.Float64("summary.compression_rate", compressionRate),
			attribute.Float64("summary.quality_score", qualityScore),
		)
	}
}

// GetTraceID 获取当前 Span 的 TraceID
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 获取当前 Span 的 SpanID
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsTracing 检查当前是否在追踪中
func IsTracing(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().IsValid() && span.IsRecording()
}
