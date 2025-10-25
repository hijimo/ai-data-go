package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
)

// BenchmarkFullRequestWithTracing 测试完整请求处理的性能（包含 TraceID）
func BenchmarkFullRequestWithTracing(b *testing.B) {
	b.ReportAllocs()
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		// 模拟业务逻辑
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
			"path":    r.URL.Path,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success","traceId":"` + traceID + `"}`))
	}))
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequestWithoutTracing 测试完整请求处理的性能（不包含 TraceID，对比基准）
func BenchmarkFullRequestWithoutTracing(b *testing.B) {
	b.ReportAllocs()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟业务逻辑（不使用 TraceID）
		logger.Info("处理请求", logger.Fields{
			"path": r.URL.Path,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success"}`))
	})
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkFullRequestWithClientTraceID 测试客户端提供 TraceID 的性能
func BenchmarkFullRequestWithClientTraceID(b *testing.B) {
	b.ReportAllocs()
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-Trace-ID", "trace-client-123")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// TestFullRequestLatency 测试完整请求的延迟
func TestFullRequestLatency(t *testing.T) {
	const iterations = 1000
	const maxAvgLatency = 2 * time.Millisecond // 2ms 平均延迟
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	
	var totalDuration time.Duration
	
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		
		start := time.Now()
		handler.ServeHTTP(w, req)
		duration := time.Since(start)
		
		totalDuration += duration
	}
	
	avgLatency := totalDuration / iterations
	
	t.Logf("处理 %d 个请求的总耗时: %v", iterations, totalDuration)
	t.Logf("平均请求延迟: %v", avgLatency)
	
	if avgLatency > maxAvgLatency {
		t.Logf("警告：平均请求延迟 %v 超过目标 %v", avgLatency, maxAvgLatency)
	}
}

// TestConcurrentRequestsPerformance 测试并发请求处理性能
func TestConcurrentRequestsPerformance(t *testing.T) {
	const concurrency = 100
	const requestsPerGoroutine = 100
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	
	start := time.Now()
	wg := sync.WaitGroup{}
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/api/v1/test", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	totalRequests := concurrency * requestsPerGoroutine
	qps := float64(totalRequests) / duration.Seconds()
	
	t.Logf("并发处理 %d 个请求耗时: %v", totalRequests, duration)
	t.Logf("实际 QPS: %.2f", qps)
	t.Logf("平均请求延迟: %v", duration/time.Duration(totalRequests))
}

// TestMemoryOverheadUnderLoad 测试高负载下的内存开销
func TestMemoryOverheadUnderLoad(t *testing.T) {
	const qps = 1000
	const duration = 2 * time.Second
	const maxMemoryOverhead = 200 * 1024 // 200KB (2秒)
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
		})
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"success"}`))
	}))
	
	// 强制 GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// 模拟高负载
	start := time.Now()
	count := 0
	ticker := time.NewTicker(time.Second / time.Duration(qps))
	defer ticker.Stop()
	
	done := make(chan bool)
	go func() {
		time.Sleep(duration)
		done <- true
	}()
	
loop:
	for {
		select {
		case <-ticker.C:
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			count++
		case <-done:
			break loop
		}
	}
	
	actualDuration := time.Since(start)
	
	// 强制 GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	memoryGrowth := memAfter.Alloc - memBefore.Alloc
	actualQPS := float64(count) / actualDuration.Seconds()
	memoryPerSecond := float64(memoryGrowth) / actualDuration.Seconds()
	
	t.Logf("处理请求数: %d", count)
	t.Logf("实际 QPS: %.2f", actualQPS)
	t.Logf("内存增长: %d bytes (%.2f KB)", memoryGrowth, float64(memoryGrowth)/1024)
	t.Logf("每秒内存开销: %.2f KB/s", memoryPerSecond/1024)
	
	if memoryGrowth > uint64(maxMemoryOverhead) {
		t.Logf("警告：内存开销 %d bytes 超过目标 %d bytes", memoryGrowth, maxMemoryOverhead)
	}
}

// TestTraceIDPropagation 测试 TraceID 在整个请求链路中的传播
func TestTraceIDPropagation(t *testing.T) {
	var capturedTraceID string
	
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		capturedTraceID = GetTraceID(ctx)
		
		// 模拟调用服务层
		serviceFunc := func(ctx context.Context) string {
			return GetTraceID(ctx)
		}
		
		serviceTraceID := serviceFunc(ctx)
		
		if capturedTraceID != serviceTraceID {
			t.Errorf("TraceID 传播失败: handler=%s, service=%s", capturedTraceID, serviceTraceID)
		}
		
		w.Header().Set("X-Trace-ID", capturedTraceID)
		w.WriteHeader(http.StatusOK)
	}))
	
	// 测试服务端生成 TraceID
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	responseTraceID := w.Header().Get("X-Trace-ID")
	if responseTraceID != capturedTraceID {
		t.Errorf("响应头 TraceID 不匹配: response=%s, captured=%s", responseTraceID, capturedTraceID)
	}
	
	// 测试客户端提供 TraceID
	clientTraceID := "trace-client-test-123"
	req = httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Trace-ID", clientTraceID)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	responseTraceID = w.Header().Get("X-Trace-ID")
	if responseTraceID != clientTraceID {
		t.Errorf("客户端 TraceID 未正确传播: expected=%s, got=%s", clientTraceID, responseTraceID)
	}
}

// TestTracingOverheadComparison 对比有无 TraceID 的性能差异
func TestTracingOverheadComparison(t *testing.T) {
	const iterations = 1000
	
	// 测试不带 TraceID 的处理器
	handlerWithout := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("处理请求", logger.Fields{
			"path": r.URL.Path,
		})
		w.WriteHeader(http.StatusOK)
	})
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		handlerWithout.ServeHTTP(w, req)
	}
	durationWithout := time.Since(start)
	
	// 测试带 TraceID 的处理器
	handlerWith := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		logger.InfoContext(ctx, "处理请求", logger.Fields{
			"traceId": traceID,
			"path":    r.URL.Path,
		})
		w.WriteHeader(http.StatusOK)
	}))
	
	start = time.Now()
	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		handlerWith.ServeHTTP(w, req)
	}
	durationWith := time.Since(start)
	
	overhead := durationWith - durationWithout
	overheadPercent := float64(overhead) / float64(durationWithout) * 100
	avgOverhead := overhead / iterations
	
	t.Logf("不带 TraceID 总耗时: %v (平均: %v)", durationWithout, durationWithout/iterations)
	t.Logf("带 TraceID 总耗时: %v (平均: %v)", durationWith, durationWith/iterations)
	t.Logf("总额外开销: %v (%.2f%%)", overhead, overheadPercent)
	t.Logf("平均每请求额外开销: %v", avgOverhead)
	
	// 验证额外开销在可接受范围内
	if avgOverhead > 1*time.Millisecond {
		t.Logf("警告：平均每请求额外开销 %v 超过 1ms", avgOverhead)
	}
	
	if overheadPercent > 30 {
		t.Logf("警告：TraceID 带来的性能开销为 %.2f%%，超过 30%%", overheadPercent)
	}
}
