package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/response"
)

// TestEndToEndTracing 测试端到端的追踪功能
// 验证：请求头 → Context → 日志 → 响应
func TestEndToEndTracing(t *testing.T) {
	tests := []struct {
		name           string
		clientTraceID  string
		expectTraceID  string
		shouldGenerate bool
	}{
		{
			name:           "客户端提供TraceID",
			clientTraceID:  "trace-client-123",
			expectTraceID:  "trace-client-123",
			shouldGenerate: false,
		},
		{
			name:           "客户端未提供TraceID",
			clientTraceID:  "",
			expectTraceID:  "", // 将由系统生成
			shouldGenerate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 创建测试处理器，返回包含TraceID的响应
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				
				// 从Context提取TraceID
				traceID := GetTraceID(ctx)
				
				// 使用带Context的响应函数
				resp := response.SuccessWithContext(ctx, &map[string]string{
					"message": "test",
				})
				
				// 验证响应中的TraceID与Context中的一致
				if resp.TraceID != traceID {
					t.Errorf("响应中的TraceID (%s) 与Context中的不一致 (%s)", resp.TraceID, traceID)
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			})

			// 2. 应用Logger中间件
			mux := Logger(handler)

			// 3. 创建测试请求
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.clientTraceID != "" {
				req.Header.Set("X-Trace-ID", tt.clientTraceID)
			}

			// 4. 执行请求
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			// 5. 验证响应头
			responseTraceID := rec.Header().Get("X-Trace-ID")
			if responseTraceID == "" {
				t.Error("响应头中缺少 X-Trace-ID")
			}

			if tt.shouldGenerate {
				// 验证生成的TraceID格式
				if !strings.HasPrefix(responseTraceID, "trace-") {
					t.Errorf("生成的TraceID格式不正确: %s", responseTraceID)
				}
			} else {
				// 验证使用了客户端提供的TraceID
				if responseTraceID != tt.expectTraceID {
					t.Errorf("期望TraceID %s, 得到 %s", tt.expectTraceID, responseTraceID)
				}
			}

			// 6. 验证响应体
			var respBody model.ResponseData[map[string]string]
			if err := json.NewDecoder(rec.Body).Decode(&respBody); err != nil {
				t.Fatalf("解析响应体失败: %v", err)
			}

			if respBody.TraceID == "" {
				t.Error("响应体中缺少 traceId 字段")
			}

			if respBody.TraceID != responseTraceID {
				t.Errorf("响应体中的traceId (%s) 与响应头不一致 (%s)", 
					respBody.TraceID, responseTraceID)
			}
		})
	}
}

// TestConcurrentTraceIDGeneration 测试并发生成TraceID的唯一性
func TestConcurrentTraceIDGeneration(t *testing.T) {
	const (
		goroutines = 100
		iterations = 100
	)

	traceIDs := make(map[string]bool)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	// 并发生成TraceID
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				traceID := GenerateTraceID()
				
				mu.Lock()
				if traceIDs[traceID] {
					t.Errorf("发现重复的TraceID: %s", traceID)
				}
				traceIDs[traceID] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 验证生成的TraceID数量
	expectedCount := goroutines * iterations
	if len(traceIDs) != expectedCount {
		t.Errorf("期望生成 %d 个唯一TraceID, 实际生成 %d 个", 
			expectedCount, len(traceIDs))
	}

	t.Logf("成功生成 %d 个唯一的TraceID", len(traceIDs))
}

// TestConcurrentRequestsWithTracing 测试并发请求的TraceID隔离
func TestConcurrentRequestsWithTracing(t *testing.T) {
	const numRequests = 50

	// 创建测试处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		// 模拟一些处理时间
		time.Sleep(10 * time.Millisecond)
		
		resp := response.SuccessWithContext(ctx, &map[string]string{
			"traceId": traceID,
		})
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	// 并发发送请求
	wg := sync.WaitGroup{}
	results := make([]string, numRequests)
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			
			mux.ServeHTTP(rec, req)
			
			// 提取响应中的TraceID
			var respBody model.ResponseData[map[string]string]
			if err := json.NewDecoder(rec.Body).Decode(&respBody); err != nil {
				t.Errorf("解析响应失败: %v", err)
				return
			}
			
			results[index] = respBody.TraceID
		}(i)
	}

	wg.Wait()

	// 验证所有TraceID都是唯一的
	traceIDMap := make(map[string]int)
	for i, traceID := range results {
		if traceID == "" {
			t.Errorf("请求 %d 的TraceID为空", i)
			continue
		}
		traceIDMap[traceID]++
	}

	// 检查是否有重复
	for traceID, count := range traceIDMap {
		if count > 1 {
			t.Errorf("TraceID %s 出现了 %d 次（应该只出现1次）", traceID, count)
		}
	}

	t.Logf("成功处理 %d 个并发请求，所有TraceID都是唯一的", numRequests)
}

// TestTraceIDPropagationThroughMiddlewareChain 测试TraceID在中间件链中的传播
func TestTraceIDPropagationThroughMiddlewareChain(t *testing.T) {
	var capturedTraceIDs []string
	mu := sync.Mutex{}

	// 创建多个中间件，每个都会捕获TraceID
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetTraceID(r.Context())
			mu.Lock()
			capturedTraceIDs = append(capturedTraceIDs, fmt.Sprintf("mw1:%s", traceID))
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetTraceID(r.Context())
			mu.Lock()
			capturedTraceIDs = append(capturedTraceIDs, fmt.Sprintf("mw2:%s", traceID))
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		mu.Lock()
		capturedTraceIDs = append(capturedTraceIDs, fmt.Sprintf("handler:%s", traceID))
		mu.Unlock()
		
		resp := response.SuccessWithContext(r.Context(), &map[string]string{
			"message": "ok",
		})
		json.NewEncoder(w).Encode(resp)
	})

	// 构建中间件链：Logger -> middleware1 -> middleware2 -> handler
	mux := Logger(middleware1(middleware2(handler)))

	// 发送请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "trace-chain-test")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// 验证所有中间件和处理器都捕获到了相同的TraceID
	expectedTraceID := "trace-chain-test"
	for _, captured := range capturedTraceIDs {
		if !strings.Contains(captured, expectedTraceID) {
			t.Errorf("捕获的TraceID不正确: %s, 期望包含 %s", captured, expectedTraceID)
		}
	}

	if len(capturedTraceIDs) != 3 {
		t.Errorf("期望捕获3个TraceID, 实际捕获 %d 个", len(capturedTraceIDs))
	}

	t.Logf("TraceID在中间件链中正确传播: %v", capturedTraceIDs)
}

// TestTraceIDInLogOutput 测试日志输出中包含TraceID
func TestTraceIDInLogOutput(t *testing.T) {
	// 创建一个缓冲区来捕获日志输出
	var logBuffer strings.Builder
	log := logger.New(logger.InfoLevel, logger.JSONFormat, &logBuffer)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// 记录一些日志
		log.InfoContext(ctx, "处理请求", logger.Fields{
			"path": r.URL.Path,
		})
		
		resp := response.SuccessWithContext(ctx, &map[string]string{
			"message": "ok",
		})
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	// 发送请求
	testTraceID := "trace-log-test-123"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", testTraceID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// 验证日志输出包含TraceID
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, testTraceID) {
		t.Errorf("日志输出中未找到TraceID: %s\n日志内容:\n%s", testTraceID, logOutput)
	}

	// 验证日志是JSON格式且包含traceId字段
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("日志不是有效的JSON: %v\n行内容: %s", err, line)
			continue
		}
		
		if traceID, ok := logEntry["traceId"]; ok {
			if traceID != testTraceID {
				t.Errorf("日志中的traceId (%v) 与期望值不一致 (%s)", traceID, testTraceID)
			}
		}
	}

	t.Logf("日志输出正确包含TraceID")
}

// TestErrorResponseWithTraceID 测试错误响应中包含TraceID
func TestErrorResponseWithTraceID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// 模拟错误情况
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		
		errorResp := response.ErrorWithContext[any](ctx, http.StatusBadRequest, "请求参数错误")
		json.NewEncoder(w).Encode(errorResp)
	})

	mux := Logger(handler)

	// 发送请求
	testTraceID := "trace-error-test"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", testTraceID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// 验证响应头
	if rec.Header().Get("X-Trace-ID") != testTraceID {
		t.Errorf("响应头中的TraceID不正确")
	}

	// 验证响应体
	var errorResp model.ResponseData[any]
	if err := json.NewDecoder(rec.Body).Decode(&errorResp); err != nil {
		t.Fatalf("解析错误响应失败: %v", err)
	}

	if errorResp.TraceID != testTraceID {
		t.Errorf("错误响应中的TraceID (%s) 与期望值不一致 (%s)", 
			errorResp.TraceID, testTraceID)
	}

	t.Logf("错误响应正确包含TraceID")
}

// TestPaginationResponseWithTraceID 测试分页响应中包含TraceID
func TestPaginationResponseWithTraceID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// 模拟分页数据
		data := []string{"item1", "item2", "item3"}
		resp := response.PaginationWithContext(ctx, data, 1, 10, 3)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	// 发送请求
	testTraceID := "trace-pagination-test"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", testTraceID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// 验证响应体
	var paginationResp model.ResponsePaginationData[[]string]
	if err := json.NewDecoder(rec.Body).Decode(&paginationResp); err != nil {
		t.Fatalf("解析分页响应失败: %v", err)
	}

	if paginationResp.TraceID != testTraceID {
		t.Errorf("分页响应中的TraceID (%s) 与期望值不一致 (%s)", 
			paginationResp.TraceID, testTraceID)
	}

	t.Logf("分页响应正确包含TraceID")
}

// BenchmarkTraceIDGeneration 性能基准测试：TraceID生成
func BenchmarkTraceIDGeneration(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = GenerateTraceID()
	}
}

// BenchmarkRequestWithTracing 性能基准测试：带追踪的请求处理
func BenchmarkRequestWithTracing(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := response.SuccessWithContext(ctx, &map[string]string{
			"message": "ok",
		})
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

// BenchmarkRequestWithClientTraceID 性能基准测试：客户端提供TraceID的请求
func BenchmarkRequestWithClientTraceID(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := response.SuccessWithContext(ctx, &map[string]string{
			"message": "ok",
		})
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Trace-ID", "trace-benchmark-test")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

// BenchmarkContextOperations 性能基准测试：Context操作
func BenchmarkContextOperations(b *testing.B) {
	ctx := context.Background()
	traceID := "trace-benchmark-test"

	b.Run("SetTraceID", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = SetTraceID(ctx, traceID)
		}
	})

	b.Run("GetTraceID", func(b *testing.B) {
		ctx = SetTraceID(ctx, traceID)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = GetTraceID(ctx)
		}
	})
}

// TestTraceIDFormat 测试TraceID格式
func TestTraceIDFormat(t *testing.T) {
	traceID := GenerateTraceID()

	// 验证前缀
	if !strings.HasPrefix(traceID, "trace-") {
		t.Errorf("TraceID应该以'trace-'开头, 得到: %s", traceID)
	}

	// 验证格式：trace-{timestamp}-{random}
	parts := strings.Split(traceID, "-")
	if len(parts) != 3 {
		t.Errorf("TraceID格式不正确，期望3个部分，得到 %d 个: %s", len(parts), traceID)
	}

	// 验证长度（大约25-32字符）
	// 格式：trace-{timestamp}-{nanoHex}{random}
	// 例如：trace-1761373314-f63885726ef2 (29字符)
	if len(traceID) < 25 || len(traceID) > 35 {
		t.Errorf("TraceID长度不在预期范围内: %d 字符", len(traceID))
	}

	t.Logf("生成的TraceID格式正确: %s", traceID)
}

// TestMultipleRequestsWithSameTraceID 测试多个请求使用相同的TraceID
func TestMultipleRequestsWithSameTraceID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := response.SuccessWithContext(ctx, &map[string]string{
			"path": r.URL.Path,
		})
		json.NewEncoder(w).Encode(resp)
	})

	mux := Logger(handler)

	// 使用相同的TraceID发送多个请求
	sharedTraceID := "trace-shared-123"
	paths := []string{"/api/v1/users", "/api/v1/sessions", "/api/v1/messages"}

	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("X-Trace-ID", sharedTraceID)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		// 验证响应中的TraceID
		var respBody model.ResponseData[map[string]string]
		if err := json.NewDecoder(rec.Body).Decode(&respBody); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if respBody.TraceID != sharedTraceID {
			t.Errorf("路径 %s 的TraceID (%s) 与期望值不一致 (%s)", 
				path, respBody.TraceID, sharedTraceID)
		}
	}

	t.Logf("多个请求成功使用相同的TraceID: %s", sharedTraceID)
}

// TestTraceIDWithStreamingResponse 测试流式响应中的TraceID
func TestTraceIDWithStreamingResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := GetTraceID(ctx)
		
		// 设置流式响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		// 发送多个事件
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter不支持Flusher接口")
		}
		
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: {\"message\": \"event %d\", \"traceId\": \"%s\"}\n\n", i, traceID)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	})

	mux := Logger(handler)

	// 发送请求
	testTraceID := "trace-stream-test"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", testTraceID)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// 验证响应头
	if rec.Header().Get("X-Trace-ID") != testTraceID {
		t.Errorf("响应头中的TraceID不正确")
	}

	// 验证响应体包含TraceID
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, testTraceID) {
		t.Errorf("流式响应中未找到TraceID: %s", testTraceID)
	}

	t.Logf("流式响应正确包含TraceID")
}
