package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/health"
)

// mockAIService 模拟 AI 服务
type mockAIService struct{}

func (m *mockAIService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	return &model.ChatResponse{
		SessionID: "test-session",
		Message:   "test response",
		Model:     "test-model",
	}, nil
}

func (m *mockAIService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	ch := make(chan model.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockAIService) AbortChat(ctx context.Context, sessionID string) error {
	return nil
}

// mockHealthService 模拟健康检查服务
type mockHealthService struct{}

func (m *mockHealthService) Check(ctx context.Context) (*health.HealthStatus, error) {
	return &health.HealthStatus{
		Status:  "healthy",
		Version: "1.0.0",
		Uptime:  "1h",
		Dependencies: map[string]string{
			"genkit":   "connected",
			"database": "connected",
		},
	}, nil
}

func TestNewRouter(t *testing.T) {
	// 创建模拟服务
	aiService := &mockAIService{}
	healthService := &mockHealthService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, io.Discard)

	// 创建路由器
	router := NewRouter(aiService, healthService, log)

	// 验证路由器不为空
	if router == nil {
		t.Fatal("路由器不应为空")
	}

	// 验证处理器不为空
	if router.chatHandler == nil {
		t.Error("对话处理器不应为空")
	}
	if router.abortHandler == nil {
		t.Error("中止处理器不应为空")
	}
	if router.healthHandler == nil {
		t.Error("健康检查处理器不应为空")
	}
	if router.corsConfig == nil {
		t.Error("CORS 配置不应为空")
	}
}

func TestRouterSetup(t *testing.T) {
	// 创建模拟服务
	aiService := &mockAIService{}
	healthService := &mockHealthService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, io.Discard)

	// 创建并设置路由器
	router := NewRouter(aiService, healthService, log)
	handler := router.Setup()

	// 验证处理器不为空
	if handler == nil {
		t.Fatal("HTTP 处理器不应为空")
	}
}

func TestRouterHealthEndpoint(t *testing.T) {
	// 创建模拟服务
	aiService := &mockAIService{}
	healthService := &mockHealthService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, io.Discard)

	// 创建路由器
	router := NewRouter(aiService, healthService, log)
	handler := router.Handler()

	// 创建测试请求
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// 执行请求
	handler.ServeHTTP(w, req)

	// 验证响应状态码
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际得到 %d", http.StatusOK, w.Code)
	}

	// 验证响应头
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type 为 application/json，实际得到 %s", contentType)
	}
}

func TestRouterCORSHeaders(t *testing.T) {
	// 创建模拟服务
	aiService := &mockAIService{}
	healthService := &mockHealthService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, io.Discard)

	// 创建路由器
	router := NewRouter(aiService, healthService, log)
	handler := router.Handler()

	// 创建 OPTIONS 预检请求
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/chat", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	// 执行请求
	handler.ServeHTTP(w, req)

	// 验证 CORS 响应头
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("应该设置 Access-Control-Allow-Origin 头")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("应该设置 Access-Control-Allow-Methods 头")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("应该设置 Access-Control-Allow-Headers 头")
	}

	// 验证预检请求返回 204
	if w.Code != http.StatusNoContent {
		t.Errorf("预检请求期望状态码 %d，实际得到 %d", http.StatusNoContent, w.Code)
	}
}

func TestRouterRequestIDHeader(t *testing.T) {
	// 创建模拟服务
	aiService := &mockAIService{}
	healthService := &mockHealthService{}
	log := logger.New(logger.InfoLevel, logger.JSONFormat, io.Discard)

	// 创建路由器
	router := NewRouter(aiService, healthService, log)
	handler := router.Handler()

	// 创建测试请求
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// 执行请求
	handler.ServeHTTP(w, req)

	// 验证 X-Request-ID 响应头存在
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("应该设置 X-Request-ID 响应头")
	}
}
