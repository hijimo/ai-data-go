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

			// 注意：由于测试中使用的是原始请求对象，
			// 而中间件创建了新的请求对象，所以这里无法直接验证上下文
			// 实际使用中，处理器会收到带有请求ID的上下文
		})
	}
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
