package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// TestRegisterMemoryRoutes 测试记忆管理路由注册
func TestRegisterMemoryRoutes(t *testing.T) {
	// 创建测试用的 ServeMux
	mux := http.NewServeMux()

	// 创建 mock logger
	log := logger.New(logger.InfoLevel, logger.TextFormat, io.Discard)

	// 创建 mock memory service
	var memoryService service.MemoryService = nil // 在实际测试中应该使用 mock

	// 创建 memory handler
	memoryHandler := handler.NewMemoryHandler(memoryService, log)

	// 创建测试用的中间件
	jwtAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 模拟 JWT 认证
			next.ServeHTTP(w, r)
		})
	}

	rbacMiddleware := func(roles ...string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 模拟 RBAC 权限验证
				next.ServeHTTP(w, r)
			})
		}
	}

	// 注册路由
	RegisterMemoryRoutes(mux, memoryHandler, jwtAuthMiddleware, rbacMiddleware)

	// 测试路由是否正确注册
	tests := []struct {
		name       string
		method     string
		path       string
		shouldFind bool
	}{
		{
			name:       "检索记忆路由",
			method:     "POST",
			path:       "/api/v1/memories/search",
			shouldFind: true,
		},
		{
			name:       "存储记忆路由",
			method:     "POST",
			path:       "/api/v1/memories",
			shouldFind: true,
		},
		{
			name:       "清理记忆路由",
			method:     "POST",
			path:       "/api/v1/memories/cleanup",
			shouldFind: true,
		},
		{
			name:       "获取记忆详情路由",
			method:     "GET",
			path:       "/api/v1/memories/123e4567-e89b-12d3-a456-426614174000",
			shouldFind: true,
		},
		{
			name:       "不存在的路由",
			method:     "GET",
			path:       "/api/v1/memories/invalid",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// 执行请求
			mux.ServeHTTP(w, req)

			// 验证路由是否存在
			if tt.shouldFind {
				// 如果路由应该存在，状态码不应该是 404
				// 注意：由于我们使用的是 mock service（nil），实际会返回 500 或其他错误
				// 但不会是 404 Not Found
				if w.Code == http.StatusNotFound {
					t.Errorf("路由 %s %s 应该存在，但返回了 404", tt.method, tt.path)
				}
			}
		})
	}
}

// TestMemoryRoutesMiddleware 测试记忆管理路由的中间件应用
func TestMemoryRoutesMiddleware(t *testing.T) {
	mux := http.NewServeMux()

	log := logger.New(logger.InfoLevel, logger.TextFormat, io.Discard)

	var memoryService service.MemoryService = nil
	memoryHandler := handler.NewMemoryHandler(memoryService, log)

	// 创建测试中间件，用于验证中间件是否被调用
	jwtCalled := false
	rbacCalled := false

	jwtAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jwtCalled = true
			next.ServeHTTP(w, r)
		})
	}

	rbacMiddleware := func(roles ...string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rbacCalled = true
				// 验证角色参数
				if len(roles) == 0 {
					t.Error("RBAC 中间件应该接收角色参数")
				}
				if len(roles) > 0 && roles[0] != "tenant_admin" {
					t.Errorf("期望角色为 tenant_admin，实际为 %s", roles[0])
				}
				next.ServeHTTP(w, r)
			})
		}
	}

	RegisterMemoryRoutes(mux, memoryHandler, jwtAuthMiddleware, rbacMiddleware)

	// 测试检索记忆路由
	req := httptest.NewRequest("POST", "/api/v1/memories/search", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// 验证中间件是否被调用
	if !jwtCalled {
		t.Error("JWT 认证中间件未被调用")
	}
	if !rbacCalled {
		t.Error("RBAC 权限中间件未被调用")
	}
}

// TestMemoryRoutesPathParameters 测试记忆管理路由的路径参数
func TestMemoryRoutesPathParameters(t *testing.T) {
	mux := http.NewServeMux()

	log := logger.New(logger.InfoLevel, logger.TextFormat, io.Discard)

	var memoryService service.MemoryService = nil
	memoryHandler := handler.NewMemoryHandler(memoryService, log)

	jwtAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	rbacMiddleware := func(roles ...string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	RegisterMemoryRoutes(mux, memoryHandler, jwtAuthMiddleware, rbacMiddleware)

	// 测试不同的记忆ID
	testIDs := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"987fcdeb-51a2-43d7-9876-543210fedcba",
		"00000000-0000-0000-0000-000000000000",
	}

	for _, id := range testIDs {
		t.Run("记忆ID_"+id, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/memories/"+id, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// 验证路由能够匹配
			if w.Code == http.StatusNotFound {
				t.Errorf("路由应该匹配记忆ID %s，但返回了 404", id)
			}
		})
	}
}
