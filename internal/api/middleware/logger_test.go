package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		handler        http.HandlerFunc
		expectedStatus int
	}{
		{
			name:   "记录 GET 请求",
			method: http.MethodGet,
			path:   "/api/v1/test",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "记录 POST 请求",
			method: http.MethodPost,
			path:   "/api/v1/chat",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("Created"))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "记录错误响应",
			method: http.MethodGet,
			path:   "/api/v1/notfound",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Not Found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			// 应用中间件
			handler := Logger(tt.handler)

			// 执行请求
			handler.ServeHTTP(rec, req)

			// 验证状态码
			if rec.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, rec.Code)
			}

			// 验证请求ID头
			requestID := rec.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("期望设置 X-Request-ID 响应头")
			}

			// 验证 TraceID 头
			traceID := rec.Header().Get("X-Trace-ID")
			if traceID == "" {
				t.Error("期望设置 X-Trace-ID 响应头")
			}

			// 注意：由于测试中使用的是原始请求对象，
			// 而中间件创建了新的请求对象，所以这里无法直接验证上下文
			// 实际使用中，处理器会收到带有请求ID和TraceID的上下文
		})
	}
}

func TestLoggerWithClientTraceID(t *testing.T) {
	t.Run("使用客户端提供的 TraceID", func(t *testing.T) {
		clientTraceID := "trace-client-123456"
		var capturedTraceID string

		// 创建测试请求，带有客户端 TraceID
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.Header.Set("X-Trace-ID", clientTraceID)
		rec := httptest.NewRecorder()

		// 创建处理器，捕获上下文中的 TraceID
		handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTraceID = GetTraceID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		// 执行请求
		handler.ServeHTTP(rec, req)

		// 验证响应头中的 TraceID 与客户端提供的一致
		responseTraceID := rec.Header().Get("X-Trace-ID")
		if responseTraceID != clientTraceID {
			t.Errorf("期望响应头 TraceID 为 %s, 得到 %s", clientTraceID, responseTraceID)
		}

		// 验证上下文中的 TraceID 与客户端提供的一致
		if capturedTraceID != clientTraceID {
			t.Errorf("期望上下文 TraceID 为 %s, 得到 %s", clientTraceID, capturedTraceID)
		}
	})

	t.Run("客户端未提供 TraceID 时自动生成", func(t *testing.T) {
		var capturedTraceID string

		// 创建测试请求，不带 TraceID
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		rec := httptest.NewRecorder()

		// 创建处理器，捕获上下文中的 TraceID
		handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTraceID = GetTraceID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		// 执行请求
		handler.ServeHTTP(rec, req)

		// 验证响应头中有 TraceID
		responseTraceID := rec.Header().Get("X-Trace-ID")
		if responseTraceID == "" {
			t.Error("期望自动生成 TraceID")
		}

		// 验证上下文中有 TraceID
		if capturedTraceID == "" {
			t.Error("期望上下文中有 TraceID")
		}

		// 验证响应头和上下文中的 TraceID 一致
		if responseTraceID != capturedTraceID {
			t.Errorf("响应头和上下文中的 TraceID 不一致: %s != %s", responseTraceID, capturedTraceID)
		}

		// 验证 TraceID 格式（应该以 "trace-" 开头）
		if len(capturedTraceID) < 6 || capturedTraceID[:6] != "trace-" {
			t.Errorf("TraceID 格式不正确: %s", capturedTraceID)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("捕获状态码", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
			written:        false,
		}

		// 写入状态码
		rw.WriteHeader(http.StatusCreated)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("期望状态码 %d, 得到 %d", http.StatusCreated, rw.statusCode)
		}

		if !rw.written {
			t.Error("期望 written 标志为 true")
		}
	})

	t.Run("默认状态码为 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
			written:        false,
		}

		// 直接写入内容，不调用 WriteHeader
		rw.Write([]byte("test"))

		if rw.statusCode != http.StatusOK {
			t.Errorf("期望默认状态码 %d, 得到 %d", http.StatusOK, rw.statusCode)
		}

		if !rw.written {
			t.Error("期望 written 标志为 true")
		}
	})

	t.Run("防止重复写入状态码", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
			written:        false,
		}

		// 第一次写入
		rw.WriteHeader(http.StatusCreated)
		// 第二次写入应该被忽略
		rw.WriteHeader(http.StatusBadRequest)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("期望状态码保持为 %d, 得到 %d", http.StatusCreated, rw.statusCode)
		}
	})
}
