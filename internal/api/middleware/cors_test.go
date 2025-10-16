package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCORS(t *testing.T) {
	cors := DefaultCORS()

	if len(cors.AllowOrigins) == 0 {
		t.Error("期望 AllowOrigins 不为空")
	}

	if len(cors.AllowMethods) == 0 {
		t.Error("期望 AllowMethods 不为空")
	}

	if len(cors.AllowHeaders) == 0 {
		t.Error("期望 AllowHeaders 不为空")
	}

	if cors.MaxAge <= 0 {
		t.Error("期望 MaxAge 大于 0")
	}
}

func TestCORSHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		corsConfig     *CORS
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "允许所有来源",
			method:         http.MethodGet,
			origin:         "http://example.com",
			corsConfig:     DefaultCORS(),
			expectedOrigin: "http://example.com", // 当有 origin 头时，返回该 origin
			expectedStatus: http.StatusOK,
		},
		{
			name:   "允许特定来源",
			method: http.MethodGet,
			origin: "http://example.com",
			corsConfig: &CORS{
				AllowOrigins: []string{"http://example.com"},
				AllowMethods: []string{"GET", "POST"},
			},
			expectedOrigin: "http://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:   "预检请求返回 204",
			method: http.MethodOptions,
			origin: "http://example.com",
			corsConfig: &CORS{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{"GET", "POST"},
			},
			expectedOrigin: "http://example.com", // 当有 origin 头时，返回该 origin
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "不允许的来源",
			method: http.MethodGet,
			origin: "http://evil.com",
			corsConfig: &CORS{
				AllowOrigins: []string{"http://example.com"},
				AllowMethods: []string{"GET", "POST"},
			},
			expectedOrigin: "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试处理器
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// 应用 CORS 中间件
			corsHandler := tt.corsConfig.Handler(handler)

			// 创建测试请求
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			// 执行请求
			corsHandler.ServeHTTP(rec, req)

			// 验证状态码
			if rec.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d", tt.expectedStatus, rec.Code)
			}

			// 验证 CORS 头
			allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.expectedOrigin != "" && allowOrigin != tt.expectedOrigin {
				t.Errorf("期望 Access-Control-Allow-Origin 为 %s, 得到 %s", tt.expectedOrigin, allowOrigin)
			}

			// 验证其他 CORS 头
			if tt.method == http.MethodOptions {
				allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
				if allowMethods == "" {
					t.Error("期望设置 Access-Control-Allow-Methods")
				}
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name         string
		allowOrigins []string
		origin       string
		expected     bool
	}{
		{
			name:         "允许所有来源",
			allowOrigins: []string{"*"},
			origin:       "http://example.com",
			expected:     true,
		},
		{
			name:         "允许特定来源",
			allowOrigins: []string{"http://example.com"},
			origin:       "http://example.com",
			expected:     true,
		},
		{
			name:         "不允许的来源",
			allowOrigins: []string{"http://example.com"},
			origin:       "http://evil.com",
			expected:     false,
		},
		{
			name:         "允许多个来源",
			allowOrigins: []string{"http://example.com", "http://test.com"},
			origin:       "http://test.com",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cors := &CORS{
				AllowOrigins: tt.allowOrigins,
			}

			result := cors.isOriginAllowed(tt.origin)
			if result != tt.expected {
				t.Errorf("期望 %v, 得到 %v", tt.expected, result)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "空切片",
			input:    []string{},
			expected: "",
		},
		{
			name:     "单个字符串",
			input:    []string{"GET"},
			expected: "GET",
		},
		{
			name:     "多个字符串",
			input:    []string{"GET", "POST", "PUT"},
			expected: "GET, POST, PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.input)
			if result != tt.expected {
				t.Errorf("期望 %s, 得到 %s", tt.expected, result)
			}
		})
	}
}
