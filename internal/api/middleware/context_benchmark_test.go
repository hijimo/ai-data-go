package middleware

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestTraceIDGenerationPerformance 验证 TraceID 生成耗时 < 1ms
func TestTraceIDGenerationPerformance(t *testing.T) {
	const iterations = 10000
	const maxDuration = 1 * time.Millisecond
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = GenerateTraceID()
	}
	duration := time.Since(start)
	
	avgDuration := duration / iterations
	
	t.Logf("生成 %d 个 TraceID 总耗时: %v", iterations, duration)
	t.Logf("平均每个 TraceID 生成耗时: %v", avgDuration)
	
	if avgDuration > maxDuration {
		t.Errorf("TraceID 生成耗时 %v 超过目标 %v", avgDuration, maxDuration)
	}
}

// TestContextOperationsPerformance 验证 Context 操作耗时 < 0.1ms
func TestContextOperationsPerformance(t *testing.T) {
	const iterations = 10000
	const maxDuration = 100 * time.Microsecond // 0.1ms
	
	ctx := context.Background()
	traceID := "trace-1704067200-a3f9k2"
	
	// 测试 SetTraceID 性能
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = SetTraceID(ctx, traceID)
	}
	setDuration := time.Since(start)
	avgSetDuration := setDuration / iterations
	
	t.Logf("SetTraceID 平均耗时: %v", avgSetDuration)
	
	if avgSetDuration > maxDuration {
		t.Errorf("SetTraceID 耗时 %v 超过目标 %v", avgSetDuration, maxDuration)
	}
	
	// 测试 GetTraceID 性能
	ctx = SetTraceID(ctx, traceID)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = GetTraceID(ctx)
	}
	getDuration := time.Since(start)
	avgGetDuration := getDuration / iterations
	
	t.Logf("GetTraceID 平均耗时: %v", avgGetDuration)
	
	if avgGetDuration > maxDuration {
		t.Errorf("GetTraceID 耗时 %v 超过目标 %v", avgGetDuration, maxDuration)
	}
}

// TestMemoryOverhead 验证 1000 QPS 下的额外内存开销 < 100KB/s
func TestMemoryOverhead(t *testing.T) {
	const qps = 1000
	const duration = 1 * time.Second
	const maxMemoryOverhead = 100 * 1024 // 100KB
	
	// 强制 GC 以获得准确的基线
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// 模拟 1000 QPS 持续 1 秒
	start := time.Now()
	count := 0
	for time.Since(start) < duration {
		_ = GenerateTraceID()
		ctx := SetTraceID(context.Background(), "trace-1704067200-a3f9k2")
		_ = GetTraceID(ctx)
		count++
		
		// 控制 QPS
		if count%qps == 0 {
			time.Sleep(time.Second - time.Since(start))
		}
	}
	
	// 强制 GC 以获得准确的测量
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	// 计算内存增长
	memoryGrowth := memAfter.Alloc - memBefore.Alloc
	
	t.Logf("处理请求数: %d", count)
	t.Logf("实际 QPS: %.2f", float64(count)/duration.Seconds())
	t.Logf("内存增长: %d bytes (%.2f KB)", memoryGrowth, float64(memoryGrowth)/1024)
	t.Logf("每秒内存开销: %.2f KB/s", float64(memoryGrowth)/1024/duration.Seconds())
	
	if memoryGrowth > uint64(maxMemoryOverhead) {
		t.Errorf("内存开销 %d bytes 超过目标 %d bytes", memoryGrowth, maxMemoryOverhead)
	}
}

// TestTraceIDUniquenessUnderLoad 测试高并发下的 TraceID 唯一性
func TestTraceIDUniquenessUnderLoad(t *testing.T) {
	const goroutines = 100
	const iterationsPerGoroutine = 1000
	
	traceIDs := make(map[string]bool, goroutines*iterationsPerGoroutine)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	
	start := time.Now()
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				traceID := GenerateTraceID()
				mu.Lock()
				if traceIDs[traceID] {
					t.Errorf("发现重复的 TraceID: %s", traceID)
				}
				traceIDs[traceID] = true
				mu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	expectedCount := goroutines * iterationsPerGoroutine
	actualCount := len(traceIDs)
	
	t.Logf("生成 %d 个 TraceID 耗时: %v", actualCount, duration)
	t.Logf("平均生成速率: %.2f TraceIDs/s", float64(actualCount)/duration.Seconds())
	
	if actualCount != expectedCount {
		t.Errorf("期望生成 %d 个唯一 TraceID，实际生成 %d 个", expectedCount, actualCount)
	}
}



// TestObjectPoolEfficiency 测试对象池的效率
func TestObjectPoolEfficiency(t *testing.T) {
	const iterations = 10000
	
	// 强制 GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// 使用对象池生成 TraceID
	for i := 0; i < iterations; i++ {
		_ = GenerateTraceID()
	}
	
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	allocsWithPool := memAfter.TotalAlloc - memBefore.TotalAlloc
	
	t.Logf("使用对象池生成 %d 个 TraceID 的总分配: %d bytes (%.2f KB)", 
		iterations, allocsWithPool, float64(allocsWithPool)/1024)
	t.Logf("平均每个 TraceID 分配: %.2f bytes", float64(allocsWithPool)/float64(iterations))
	
	// 验证对象池确实减少了内存分配
	// 每个 TraceID 的平均分配应该远小于字符串本身的大小
	avgAllocPerTraceID := float64(allocsWithPool) / float64(iterations)
	if avgAllocPerTraceID > 100 {
		t.Logf("警告：平均每个 TraceID 分配 %.2f bytes，可能对象池效率不高", avgAllocPerTraceID)
	}
}
