package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracer 设置测试用的追踪器
func setupTestTracer() *tracetest.SpanRecorder {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	return sr
}

func TestTraceFlow_Success(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 执行追踪
	err := TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return nil
	})

	// 验证
	assert.NoError(t, err)
	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "testFlow", span.Name())
	assert.Contains(t, span.Attributes(), attribute.String("flow.name", "testFlow"))
	assert.Contains(t, span.Attributes(), attribute.String("component", "genkit-flow"))
}

func TestTraceFlow_WithError(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()
	testErr := errors.New("测试错误")

	// 执行追踪
	err := TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return testErr
	})

	// 验证
	assert.Error(t, err)
	assert.Equal(t, testErr, err)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "testFlow", span.Name())
	assert.Contains(t, span.Attributes(), attribute.Bool("error", true))
}

func TestTraceFlow_WithContextValues(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()
	ctx = context.WithValue(ctx, "session_id", "test-session-123")
	ctx = context.WithValue(ctx, "user_id", "test-user-456")
	ctx = context.WithValue(ctx, "tenant_id", "test-tenant-789")

	// 执行追踪
	err := TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return nil
	})

	// 验证
	assert.NoError(t, err)
	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Contains(t, span.Attributes(), attribute.String("session.id", "test-session-123"))
	assert.Contains(t, span.Attributes(), attribute.String("user.id", "test-user-456"))
	assert.Contains(t, span.Attributes(), attribute.String("tenant.id", "test-tenant-789"))
}

func TestTraceService_Success(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 执行追踪
	err := TraceService(ctx, "ContextService", "BuildContext", func(ctx context.Context) error {
		return nil
	})

	// 验证
	assert.NoError(t, err)
	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "ContextService.BuildContext", span.Name())
	assert.Contains(t, span.Attributes(), attribute.String("service.name", "ContextService"))
	assert.Contains(t, span.Attributes(), attribute.String("method.name", "BuildContext"))
	assert.Contains(t, span.Attributes(), attribute.String("component", "service"))
}

func TestTraceRepository_Success(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 执行追踪
	err := TraceRepository(ctx, "MemoryRepository", "SearchByVector", func(ctx context.Context) error {
		return nil
	})

	// 验证
	assert.NoError(t, err)
	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "MemoryRepository.SearchByVector", span.Name())
	assert.Contains(t, span.Attributes(), attribute.String("db.repository", "MemoryRepository"))
	assert.Contains(t, span.Attributes(), attribute.String("db.operation", "SearchByVector"))
	assert.Contains(t, span.Attributes(), attribute.String("component", "repository"))
}

func TestTraceExternalCall_Success(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 执行追踪
	err := TraceExternalCall(ctx, "OpenAI", "ChatCompletion", func(ctx context.Context) error {
		return nil
	})

	// 验证
	assert.NoError(t, err)
	spans := sr.Ended()
	assert.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "external.OpenAI.ChatCompletion", span.Name())
	assert.Contains(t, span.Attributes(), attribute.String("external.service", "OpenAI"))
	assert.Contains(t, span.Attributes(), attribute.String("external.operation", "ChatCompletion"))
	assert.Contains(t, span.Attributes(), attribute.String("component", "external"))
}

func TestAddSpanAttributes(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	defer span.End()

	// 添加属性
	AddSpanAttributes(ctx,
		attribute.String("custom.key", "custom.value"),
		attribute.Int("custom.count", 42),
	)

	// 结束 Span
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Contains(t, spans[0].Attributes(), attribute.String("custom.key", "custom.value"))
	assert.Contains(t, spans[0].Attributes(), attribute.Int("custom.count", 42))
}

func TestAddSpanEvent(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	defer span.End()

	// 添加事件
	AddSpanEvent(ctx, "test.event",
		attribute.String("event.detail", "测试事件"),
	)

	// 结束 Span
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	events := spans[0].Events()
	assert.Len(t, events, 1)
	assert.Equal(t, "test.event", events[0].Name)
}

func TestRecordError(t *testing.T) {
	sr := setupTestTracer()
	ctx := context.Background()

	// 创建 Span
	ctx, span := StartSpan(ctx, "testSpan")
	defer span.End()

	// 记录错误
	testErr := errors.New("测试错误")
	RecordError(ctx, testErr)

	// 结束 Span
	span.End()

	// 验证
	spans := sr.Ended()
	assert.Len(t, spans, 1)
	events := spans[0].Events()
	assert.Greater(t, len(events), 0)
}
