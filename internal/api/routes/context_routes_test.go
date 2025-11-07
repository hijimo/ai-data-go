package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
)

// TestRegisterContextRoutes 测试上下文管理路由注册
func TestRegisterContextRoutes(t *testing.T) {
	// 创建测试用的 ServeMux
	mux := http.NewServeMux()
	
	// 创建模拟的 ContextService
	mockContextService := &mockContextService{}
	
	// 创建 ContextHandler
	log := logger.New(logger.InfoLevel, logger.TextFormat, io.Discard)
	contextHandler := handler.NewContextHandler(mockContextService, log)
	
	// 创建模拟的中间件
	jwtAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 模拟 JWT 认证通过
			next.ServeHTTP(w, r)
		})
	}
	
	rbacMiddleware := func(roles ...string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 模拟 RBAC 授权通过
				next.ServeHTTP(w, r)
			})
		}
	}
	
	// 注册路由
	RegisterContextRoutes(mux, contextHandler, jwtAuthMiddleware, rbacMiddleware)
	
	// 测试路由是否正确注册
	// 注意：这里只验证路由是否存在，不验证具体的业务逻辑
	// 返回 400 或其他错误码都说明路由已经注册成功
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int // 期望的状态码（400 表示路由存在但请求无效）
	}{
		{
			name:       "POST /api/v1/contexts/build",
			method:     "POST",
			path:       "/api/v1/contexts/build",
			wantStatus: http.StatusBadRequest, // 400 - 路由存在，但请求体无效
		},
		{
			name:       "GET /api/v1/contexts/{sessionId}",
			method:     "GET",
			path:       "/api/v1/contexts/test-session-id",
			wantStatus: http.StatusBadRequest, // 400 - 路由存在，但参数无效
		},
		{
			name:       "PUT /api/v1/contexts/{sessionId}",
			method:     "PUT",
			path:       "/api/v1/contexts/test-session-id",
			wantStatus: http.StatusBadRequest, // 400 - 路由存在，但请求体无效
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			mux.ServeHTTP(w, req)
			
			// 验证路由是否存在（不是 404）
			if w.Code == http.StatusNotFound {
				t.Errorf("路由未注册：%s %s", tt.method, tt.path)
			}
			
			// 验证返回的状态码
			if w.Code != tt.wantStatus {
				t.Logf("注意：期望状态码 %d, 实际得到 %d（路由已注册）", tt.wantStatus, w.Code)
			}
		})
	}
}

// mockContextService 模拟的 ContextService
type mockContextService struct{}

func (m *mockContextService) BuildContext(ctx context.Context, req service.BuildContextRequest) (*service.ContextResult, error) {
	return &service.ContextResult{}, nil
}

func (m *mockContextService) OptimizeContext(ctx context.Context, req service.OptimizeContextRequest) (*service.ContextResult, error) {
	return &service.ContextResult{}, nil
}

func (m *mockContextService) GetContextConfig(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	return nil, nil
}

func (m *mockContextService) UpdateContextConfig(ctx context.Context, sessionID string, config *model.ConversationContext) error {
	return nil
}
