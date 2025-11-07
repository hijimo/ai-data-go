package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestNewTracer(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)

	// 清理
	err = tracer.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTraceFlow_Success(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	// 测试成功的 Flow
	ctx := context.Background()
	err = tracer.TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestTraceFlow_Error(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	// 测试失败的 Flow
	ctx := context.Background()
	expectedErr := errors.New("test error")
	err = tracer.TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return expectedErr
	})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestTraceFlow_WithContext(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	// 创建带有上下文信息的 context
	ctx := context.Background()
	ctx = context.WithValue(ctx, "session_id", "test-session-123")
	ctx = context.WithValue(ctx, "tenant_id", "test-tenant-456")
	ctx = context.WithValue(ctx, "user_id", "test-user-789")

	err = tracer.TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestStartSpan(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	ctx, span := tracer.StartSpan(ctx, "testSpan")
	assert.NotNil(t, span)

	span.End()
}

func TestTraceOperation(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = TraceOperation(ctx, "testOperation", func(ctx context.Context) error {
		return nil
	}, attribute.String("test.key", "test.value"))

	assert.NoError(t, err)
}

func TestTraceDBQuery(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = TraceDBQuery(ctx, "get_user", "SELECT * FROM users WHERE id = $1", func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestTraceVectorSearch(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = TraceVectorSearch(ctx, "session-123", 5, func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestTraceAIGeneration(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = TraceAIGeneration(ctx, "gemini-2.5-flash", 1000, func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestTraceCacheOperation(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = TraceCacheOperation(ctx, "get", "cache:key:123", func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestNoOpTracer(t *testing.T) {
	tracer := NewNoOpTracer()
	assert.NotNil(t, tracer)

	ctx := context.Background()

	// 测试 TraceFlow
	err := tracer.TraceFlow(ctx, "testFlow", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	// 测试 StartSpan
	ctx, span := tracer.StartSpan(ctx, "testSpan")
	assert.NotNil(t, span)

	// 测试 Shutdown
	err = tracer.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSamplingRate(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
	}{
		{"AlwaysSample", 1.0},
		{"NeverSample", 0.0},
		{"HalfSample", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TracerConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				OTLPEndpoint:   "localhost:4318",
				SamplingRate:   tt.samplingRate,
			}

			tracer, err := NewTracer(config)
			assert.NoError(t, err)
			assert.NotNil(t, tracer)

			err = tracer.Shutdown(context.Background())
			assert.NoError(t, err)
		})
	}
}

func TestNestedSpans(t *testing.T) {
	config := &TracerConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4318",
		SamplingRate:   1.0,
	}

	tracer, err := NewTracer(config)
	assert.NoError(t, err)
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	err = tracer.TraceFlow(ctx, "parentFlow", func(ctx context.Context) error {
		// 创建子 span
		ctx, span1 := tracer.StartSpan(ctx, "childSpan1")
		span1.End()

		// 创建另一个子 span
		ctx, span2 := tracer.StartSpan(ctx, "childSpan2")
		span2.End()

		// 创建嵌套的操作
		return TraceOperation(ctx, "nestedOperation", func(ctx context.Context) error {
			return nil
		})
	})

	assert.NoError(t, err)
}
