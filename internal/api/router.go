package api

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/health"
)

// Router HTTP 路由器
type Router struct {
	mux           *http.ServeMux
	chatHandler   *handler.ChatHandler
	abortHandler  *handler.AbortHandler
	healthHandler *handler.HealthHandler
	corsConfig    *middleware.CORS
}

// NewRouter 创建新的路由器
func NewRouter(
	aiService ai.AIService,
	healthService health.Service,
	log logger.Logger,
) *Router {
	return &Router{
		mux:           http.NewServeMux(),
		chatHandler:   handler.NewChatHandler(aiService, log),
		abortHandler:  handler.NewAbortHandler(aiService, log),
		healthHandler: handler.NewHealthHandler(healthService, log),
		corsConfig:    middleware.DefaultCORS(),
	}
}

// Setup 配置所有路由
func (r *Router) Setup() http.Handler {
	// 注册 API 路由
	r.mux.HandleFunc("/api/v1/chat", r.chatHandler.HandleChat)
	r.mux.HandleFunc("/api/v1/chat/abort", r.abortHandler.HandleAbort)
	
	// 注册健康检查路由
	r.mux.HandleFunc("/health", r.healthHandler.Handle)
	
	// 应用中间件（按顺序：Recovery -> Logger -> CORS）
	var handler http.Handler = r.mux
	handler = r.corsConfig.Handler(handler)
	handler = middleware.Logger(handler)
	handler = middleware.Recovery(handler)
	
	return handler
}

// Handler 返回配置好的 HTTP 处理器
func (r *Router) Handler() http.Handler {
	return r.Setup()
}
