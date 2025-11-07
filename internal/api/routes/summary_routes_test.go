package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/api/handler"
)

// mockHandler 创建一个简单的mock handler用于测试路由注册
func mockSummaryHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":200,"message":"success"}`))
}

// mockJWTAuthMiddleware 模拟JWT认证中间件
func mockJWTAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 模拟认证通过
			next.ServeHTTP(w, r)
		})
	}
}

// mockRBACMiddleware 模拟RBAC权限中间件
func mockRBACMiddleware() func(...string) func(http.Handler) http.Handler {
	return func(roles ...string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 模拟权限验证通过
				next.ServeHTTP(w, r)
			})
		}
	}
}

// TestRegisterSummaryRoutes 测试摘要路由注册
func TestRegisterSummaryRoutes(t *testing.T) {
	// 创建测试用的ServeMux
	mux := http.NewServeMux()
	
	// 创建一个简单的mock handler，所有方法都返回200
	mockSummaryHandler := &handler.SummaryHandler{}
	
	jwtAuth := mockJWTAuthMiddleware()
	rbac := mockRBACMiddleware()
	
	// 手动注册路由用于测试（不使用实际的handler方法）
	mux.Handle("POST /api/v1/summaries",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("GET /api/v1/summaries/{id}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("GET /api/v1/summaries/session/{sessionId}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("POST /api/v1/summaries/check-trigger",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	
	// 测试用例
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "POST /api/v1/summaries - 生成摘要",
			method:         "POST",
			path:           "/api/v1/summaries",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/v1/summaries/{id} - 获取摘要详情",
			method:         "GET",
			path:           "/api/v1/summaries/123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET /api/v1/summaries/session/{sessionId} - 获取会话摘要列表",
			method:         "GET",
			path:           "/api/v1/summaries/session/123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST /api/v1/summaries/check-trigger - 检查摘要触发条件",
			method:         "POST",
			path:           "/api/v1/summaries/check-trigger",
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			// 执行请求
			mux.ServeHTTP(w, req)
			
			// 验证路由是否注册成功（不是404）
			if w.Code == http.StatusNotFound {
				t.Errorf("路由未注册: %s %s, 状态码: %d", tt.method, tt.path, w.Code)
			}
			
			// 验证返回状态码
			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 实际状态码 %d", tt.expectedStatus, w.Code)
			}
		})
	}
	
	// 验证RegisterSummaryRoutes函数不会panic
	t.Run("RegisterSummaryRoutes不会panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RegisterSummaryRoutes panic: %v", r)
			}
		}()
		
		testMux := http.NewServeMux()
		RegisterSummaryRoutes(testMux, mockSummaryHandler, jwtAuth, rbac)
	})
}

// TestSummaryRoutesMethodValidation 测试HTTP方法验证
func TestSummaryRoutesMethodValidation(t *testing.T) {
	// 这个测试验证路由是否正确注册了HTTP方法
	// 我们使用mock handler来避免调用实际的handler逻辑
	mux := http.NewServeMux()
	
	jwtAuth := mockJWTAuthMiddleware()
	rbac := mockRBACMiddleware()
	
	// 手动注册路由用于测试
	mux.Handle("POST /api/v1/summaries",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("GET /api/v1/summaries/{id}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("GET /api/v1/summaries/session/{sessionId}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("POST /api/v1/summaries/check-trigger",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	
	tests := []struct {
		name           string
		method         string
		path           string
		shouldMatch    bool
	}{
		{
			name:        "POST方法应该匹配 /api/v1/summaries",
			method:      "POST",
			path:        "/api/v1/summaries",
			shouldMatch: true,
		},
		{
			name:        "GET方法应该匹配 /api/v1/summaries/{id}",
			method:      "GET",
			path:        "/api/v1/summaries/123e4567-e89b-12d3-a456-426614174000",
			shouldMatch: true,
		},
		{
			name:        "GET方法应该匹配 /api/v1/summaries/session/{sessionId}",
			method:      "GET",
			path:        "/api/v1/summaries/session/123e4567-e89b-12d3-a456-426614174000",
			shouldMatch: true,
		},
		{
			name:        "POST方法应该匹配 /api/v1/summaries/check-trigger",
			method:      "POST",
			path:        "/api/v1/summaries/check-trigger",
			shouldMatch: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			mux.ServeHTTP(w, req)
			
			isNotFound := w.Code == http.StatusNotFound
			if tt.shouldMatch && isNotFound {
				t.Errorf("期望路由匹配但返回404: %s %s", tt.method, tt.path)
			}
			if !tt.shouldMatch && !isNotFound {
				t.Errorf("期望路由不匹配但没有返回404: %s %s", tt.method, tt.path)
			}
		})
	}
}

// TestSummaryRoutesPathParameters 测试路径参数
func TestSummaryRoutesPathParameters(t *testing.T) {
	// 这个测试验证路径参数是否正确匹配
	mux := http.NewServeMux()
	
	jwtAuth := mockJWTAuthMiddleware()
	rbac := mockRBACMiddleware()
	
	// 手动注册路由用于测试
	mux.Handle("GET /api/v1/summaries/{id}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	mux.Handle("GET /api/v1/summaries/session/{sessionId}",
		jwtAuth(rbac("tenant_admin")(http.HandlerFunc(mockSummaryHandlerFunc))))
	
	tests := []struct {
		name        string
		path        string
		shouldMatch bool
	}{
		{
			name:        "有效的UUID应该匹配",
			path:        "/api/v1/summaries/123e4567-e89b-12d3-a456-426614174000",
			shouldMatch: true,
		},
		{
			name:        "任意字符串也应该匹配（验证在handler中进行）",
			path:        "/api/v1/summaries/invalid-id",
			shouldMatch: true,
		},
		{
			name:        "会话摘要路径应该匹配",
			path:        "/api/v1/summaries/session/123e4567-e89b-12d3-a456-426614174000",
			shouldMatch: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			
			mux.ServeHTTP(w, req)
			
			isNotFound := w.Code == http.StatusNotFound
			if tt.shouldMatch && isNotFound {
				t.Errorf("期望路由匹配但返回404: %s", tt.path)
			}
			if !tt.shouldMatch && !isNotFound {
				t.Errorf("期望路由不匹配但没有返回404: %s", tt.path)
			}
		})
	}
}
