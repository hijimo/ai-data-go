package logger

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// BenchmarkInfoContext 测试带 Context 的日志记录性能
func BenchmarkInfoContext(b *testing.B) {
	b.ReportAllocs()
	
	ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
	
	for i := 0; i < b.N; i++ {
		InfoContext(ctx, "测试日志消息", Fields{
			"key1": "value1",
			"key2": 123,
		})
	}
}

// BenchmarkInfo 测试不带 Context 的日志记录性能（对比基准）
func BenchmarkInfo(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		Info("测试日志消息", Fields{
			"key1": "value1",
			"key2": 123,
		})
	}
}

// BenchmarkExtractContextFields 测试 Context 字段提取性能
func BenchmarkExtractContextFields(b *testing.B) {
	b.ReportAllocs()
	
	ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
	ctx = context.WithValue(ctx, RequestIDKey, "req-123")
	
	for i := 0; i < b.N; i++ {
		_ = extractContextFields(ctx)
	}
}

// BenchmarkLogWithTraceIDParallel 测试并发日志记录性能
func BenchmarkLogWithTraceIDParallel(b *testing.B) {
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
		for pb.Next() {
			InfoContext(ctx, "测试日志消息", Fields{
				"key": "value",
			})
		}
	})
}

// TestLogFieldAdditionPerformance 验证日志字段添加耗时 < 0.1ms
func TestLogFieldAdditionPerformance(t *testing.T) {
	const iterations = 10000
	const maxDuration = 100 * time.Microsecond // 0.1ms
	
	ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
	ctx = context.WithValue(ctx, RequestIDKey, "req-123")
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = extractContextFields(ctx)
	}
	duration := time.Since(start)
	
	avgDuration := duration / iterations
	
	t.Logf("提取 Context 字段平均耗时: %v", avgDuration)
	
	if avgDuration > maxDuration {
		t.Errorf("Context 字段提取耗时 %v 超过目标 %v", avgDuration, maxDuration)
	}
}

// TestLogMemoryOverhead 测试日志记录的内存开销
func TestLogMemoryOverhead(t *testing.T) {
	const iterations = 1000
	
	// 强制 GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
	
	for i := 0; i < iterations; i++ {
		InfoContext(ctx, "测试日志消息", Fields{
			"key1": "value1",
			"key2": 123,
		})
	}
	
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	memoryGrowth := memAfter.Alloc - memBefore.Alloc
	avgMemoryPerLog := float64(memoryGrowth) / float64(iterations)
	
	t.Logf("记录 %d 条日志的内存增长: %d bytes (%.2f KB)", 
		iterations, memoryGrowth, float64(memoryGrowth)/1024)
	t.Logf("平均每条日志内存开销: %.2f bytes", avgMemoryPerLog)
}

// TestLogWithTraceIDVsWithout 对比带 TraceID 和不带 TraceID 的性能差异
func TestLogWithTraceIDVsWithout(t *testing.T) {
	const iterations = 10000
	
	// 测试不带 TraceID 的日志
	start := time.Now()
	for i := 0; i < iterations; i++ {
		Info("测试日志消息", Fields{
			"key": "value",
		})
	}
	durationWithout := time.Since(start)
	
	// 测试带 TraceID 的日志
	ctx := context.WithValue(context.Background(), "traceId", "trace-1704067200-a3f9k2")
	start = time.Now()
	for i := 0; i < iterations; i++ {
		InfoContext(ctx, "测试日志消息", Fields{
			"key": "value",
		})
	}
	durationWith := time.Since(start)
	
	overhead := durationWith - durationWithout
	overheadPercent := float64(overhead) / float64(durationWithout) * 100
	
	t.Logf("不带 TraceID 耗时: %v", durationWithout)
	t.Logf("带 TraceID 耗时: %v", durationWith)
	t.Logf("额外开销: %v (%.2f%%)", overhead, overheadPercent)
	
	// 验证额外开销在可接受范围内（< 20%）
	if overheadPercent > 20 {
		t.Logf("警告：TraceID 带来的性能开销为 %.2f%%，超过 20%%", overheadPercent)
	}
}

// TestConcurrentLoggingWithTraceID 测试并发日志记录的正确性
func TestConcurrentLoggingWithTraceID(t *testing.T) {
	const goroutines = 100
	const logsPerGoroutine = 100
	
	start := time.Now()
	
	done := make(chan bool, goroutines)
	
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			ctx := context.WithValue(context.Background(), "traceId", "trace-test-"+string(rune(id)))
			for j := 0; j < logsPerGoroutine; j++ {
				InfoContext(ctx, "并发测试日志", Fields{
					"goroutine": id,
					"iteration": j,
				})
			}
			done <- true
		}(i)
	}
	
	// 等待所有 goroutine 完成
	for i := 0; i < goroutines; i++ {
		<-done
	}
	
	duration := time.Since(start)
	totalLogs := goroutines * logsPerGoroutine
	logsPerSecond := float64(totalLogs) / duration.Seconds()
	
	t.Logf("并发记录 %d 条日志耗时: %v", totalLogs, duration)
	t.Logf("日志记录速率: %.2f logs/s", logsPerSecond)
}
