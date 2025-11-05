package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestStartSpan(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 开始 Span
	ctx, span := StartSpan(ctx, "testSpan")
	assert.NotNil(t, span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "testSpan", spans[0].Name())
}

func TestTraceDBQuery(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 追踪数据库查询
	ctx, span := TraceDBQuery(ctx, "SELECT * FROM users WHERE id = $1", "user-123")
	assert.NotNil(t, span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "db.query", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), attribute.String("db.system", "postgresql"))
	assert.Contains(t, spans[0].Attributes(), attribute.String("component", "database"))
}

func TestTraceVectorSearch(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 追踪向量检索
	ctx, span := TraceVectorSearch(ctx, "session-123", 10)
	assert.NotNil(t, span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "vector.search", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), attribute.String("session.id", "session-123"))
	assert.Contains(t, spans[0].Attributes(), attribute.Int("vector.topk", 10))
	assert.Contains(t, spans[0].Attributes(), attribute.String("component", "vector-service"))
}

func TestTraceAIGeneration(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 追踪 AI 生成
	ctx, span := TraceAIGeneration(ctx, "gpt-4", 1500)
	assert.NotNil(t, span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "ai.generation", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), attribute.String("ai.model", "gpt-4"))
	assert.Contains(t, spans[0].Attributes(), attribute.Int("ai.prompt_tokens", 1500))
	assert.Contains(t, spans[0].Attributes(), attribute.String("component", "ai-service"))
}

func TestTraceCacheOperation(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 追踪缓存操作
	ctx, span := TraceCacheOperation(ctx, "get", "context:session-123")
	assert.NotNil(t, span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "cache.get", spans[0].Name())
	assert.Contains(t, spans[0].Attributes(), attribute.String("cache.operation", "get"))
	assert.Contains(t, spans[0].Attributes(), attribute.String("cache.key", "context:session-123"))
	assert.Contains(t, spans[0].Attributes(), attribute.String("component", "cache"))
}

func TestSetSpanSuccess(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	SetSpanSuccess(span)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "Ok", spans[0].Status().Code.String())
}

func TestAddTokenUsage(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	AddTokenUsage(ctx, 1000, 500, 1500)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.Int("tokens.prompt", 1000))
	assert.Contains(t, spans[0].Attributes(), attribute.Int("tokens.completion", 500))
	assert.Contains(t, spans[0].Attributes(), attribute.Int("tokens.total", 1500))
}

func TestAddContextMetrics(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	AddContextMetrics(ctx, 2000, 0.85, "auto")
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.Int("context.tokens", 2000))
	assert.Contains(t, spans[0].Attributes(), attribute.Float64("context.quality_score", 0.85))
	assert.Contains(t, spans[0].Attributes(), attribute.String("context.strategy", "auto"))
}

func TestAddMemoryMetrics(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	AddMemoryMetrics(ctx, 5, 0.78)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.Int("memory.count", 5))
	assert.Contains(t, spans[0].Attributes(), attribute.Float64("memory.avg_similarity", 0.78))
}

func TestAddSummaryMetrics(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	AddSummaryMetrics(ctx, 20, 0.65, 0.82)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.Int("summary.message_count", 20))
	assert.Contains(t, spans[0].Attributes(), attribute.Float64("summary.compression_rate", 0.65))
	assert.Contains(t, spans[0].Attributes(), attribute.Float64("summary.quality_score", 0.82))
}

func TestGetTraceID(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 没有 Span 的情况
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID)

	// 有 Span 的情况
	ctx, span := StartSpan(ctx, "testSpan")
	traceID = GetTraceID(ctx)
	assert.NotEmpty(t, traceID)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
}

func TestGetSpanID(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 没有 Span 的情况
	spanID := GetSpanID(ctx)
	assert.Empty(t, spanID)

	// 有 Span 的情况
	ctx, span := StartSpan(ctx, "testSpan")
	spanID = GetSpanID(ctx)
	assert.NotEmpty(t, spanID)
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
}

func TestIsTracing(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 没有 Span 的情况
	assert.False(t, IsTracing(ctx))

	// 有 Span 的情况
	ctx, span := StartSpan(ctx, "testSpan")
	assert.True(t, IsTracing(ctx))
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
}

func TestSetSpanError(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	
	// 设置错误
	testErr := errors.New("测试错误")
	SetSpanError(span, testErr)
	
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
}
