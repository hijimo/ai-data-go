package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectPanic    bool
		expectedStatus int
	}{
		{
			name: "正常请求不触发恢复",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			},
			expectPanic:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "panic 被恢复并返回 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			},
			expectPanic:    true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "panic 错误对象被恢复",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(http.ErrAbortHandler)
			},
			expectPanic:    true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// 应用中间件
			handler := Recovery(tt.handler)

			// 执行请求
			handler.ServeHTTP(rec, req)

			// 验证状态码
			if rec.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, rec.Code)
			}

			// 验证响应头
			if tt.expectPanic {
				contentType := rec.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("期望 Content-Type 为 application/json, 得到 %s", contentType)
				}
			}
		})
	}
}
