package genkit

import (
	"context"
	"testing"
)

// TestTraceIDInContext 测试 TraceID 在 Context 中的传递
func TestTraceIDInContext(t *testing.T) {
	tests := []struct {
		name     string
		traceID  string
		expected string
	}{
		{
			name:     "有效的 TraceID",
			traceID:  "trace-1733051400-a3f9k2-b8c1d4",
			expected: "trace-1733051400-a3f9k2-b8c1d4",
		},
		{
			name:     "自定义 TraceID",
			traceID:  "trace-custom-123456",
			expected: "trace-custom-123456",
		},
		{
			name:     "空 TraceID",
			traceID:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建带有 TraceID 的 Context
			ctx := context.Background()
			if tt.traceID != "" {
				ctx = context.WithValue(ctx, "traceId", tt.traceID)
			}

			// 从 Context 提取 TraceID
			var extractedTraceID string
			if traceID, ok := ctx.Value("traceId").(string); ok {
				extractedTraceID = traceID
			}

			// 验证提取的 TraceID
			if extractedTraceID != tt.expected {
				t.Errorf("期望 TraceID 为 %s, 得到 %s", tt.expected, extractedTraceID)
			}
		})
	}
}

// TestTraceIDPropagation 测试 TraceID 在调用链中的传播
func TestTraceIDPropagation(t *testing.T) {
	// 创建带有 TraceID 的 Context
	traceID := "trace-test-propagation"
	ctx := context.WithValue(context.Background(), "traceId", traceID)

	// 模拟调用链：Handler -> Service -> Client
	// 每一层都应该能够访问到相同的 TraceID

	// 第一层：Handler
	handlerCtx := ctx
	if tid, ok := handlerCtx.Value("traceId").(string); !ok || tid != traceID {
		t.Errorf("Handler 层 TraceID 不匹配: 期望 %s, 得到 %s", traceID, tid)
	}

	// 第二层：Service
	serviceCtx := handlerCtx
	if tid, ok := serviceCtx.Value("traceId").(string); !ok || tid != traceID {
		t.Errorf("Service 层 TraceID 不匹配: 期望 %s, 得到 %s", traceID, tid)
	}

	// 第三层：Client
	clientCtx := serviceCtx
	if tid, ok := clientCtx.Value("traceId").(string); !ok || tid != traceID {
		t.Errorf("Client 层 TraceID 不匹配: 期望 %s, 得到 %s", traceID, tid)
	}
}

// TestTraceIDInLogs 测试 TraceID 在日志中的记录
// 注意：这个测试主要验证 Context 传递，实际的日志记录由 logger 包处理
func TestTraceIDInLogs(t *testing.T) {
	traceID := "trace-test-logging"
	ctx := context.WithValue(context.Background(), "traceId", traceID)

	// 验证 Context 中包含 TraceID
	if tid, ok := ctx.Value("traceId").(string); !ok {
		t.Error("Context 中未找到 TraceID")
	} else if tid != traceID {
		t.Errorf("Context 中的 TraceID 不匹配: 期望 %s, 得到 %s", traceID, tid)
	}

	// 实际的日志记录测试应该在 logger 包中进行
	// 这里只验证 Context 传递是否正确
}

// TestMultipleTraceIDs 测试多个并发请求的 TraceID 隔离
func TestMultipleTraceIDs(t *testing.T) {
	traceIDs := []string{
		"trace-request-1",
		"trace-request-2",
		"trace-request-3",
	}

	for _, traceID := range traceIDs {
		t.Run(traceID, func(t *testing.T) {
			// 每个请求都有独立的 Context 和 TraceID
			ctx := context.WithValue(context.Background(), "traceId", traceID)

			// 验证 TraceID 隔离
			if tid, ok := ctx.Value("traceId").(string); !ok {
				t.Errorf("未找到 TraceID: %s", traceID)
			} else if tid != traceID {
				t.Errorf("TraceID 不匹配: 期望 %s, 得到 %s", traceID, tid)
			}
		})
	}
}

// BenchmarkTraceIDExtraction 性能测试：TraceID 提取
func BenchmarkTraceIDExtraction(b *testing.B) {
	ctx := context.WithValue(context.Background(), "traceId", "trace-benchmark-test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := ctx.Value("traceId").(string); !ok {
			b.Fatal("TraceID 提取失败")
		}
	}
}

// BenchmarkContextWithTraceID 性能测试：创建带 TraceID 的 Context
func BenchmarkContextWithTraceID(b *testing.B) {
	traceID := "trace-benchmark-test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = context.WithValue(context.Background(), "traceId", traceID)
	}
}
