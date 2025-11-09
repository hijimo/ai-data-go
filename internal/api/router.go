package api

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/health"
	"genkit-ai-service/internal/service/session"
)

// Router HTTP 路由器
type Router struct {
	mux               *http.ServeMux
	chatHandler       *handler.ChatHandler
	chatStreamHandler *handler.ChatStreamHandler
	abortHandler      *handler.AbortHandler
	healthHandler     *handler.HealthHandler
	sessionHandler    *handler.SessionHandler
	messageHandler    *handler.MessageHandler
	corsConfig        *middleware.CORS
}

// NewRouter 创建新的路由器
func NewRouter(
	aiService ai.AIService,
	healthService health.Service,
	sessionService session.SessionService,
	messageService session.MessageService,
	log logger.Logger,
) *Router {
	return &Router{
		mux:               http.NewServeMux(),
		chatHandler:       handler.NewChatHandler(aiService, log),
		chatStreamHandler: handler.NewChatStreamHandler(aiService, log),
		abortHandler:      handler.NewAbortHandler(aiService, log),
		healthHandler:     handler.NewHealthHandler(healthService, log),
		sessionHandler:    handler.NewSessionHandler(sessionService, log),
		messageHandler:    handler.NewMessageHandler(messageService, log),
		corsConfig:        middleware.DefaultCORS(),
	}
}

// Setup 配置所有路由
func (r *Router) Setup() http.Handler {
	// 注册旧版 API 路由（保持向后兼容）
	r.mux.HandleFunc("/api/v1/chat", r.chatHandler.HandleChat)
	r.mux.HandleFunc("/api/v1/chat/stream", r.chatStreamHandler.HandleChatStream)
	r.mux.HandleFunc("/api/v1/chat/abort", r.abortHandler.HandleAbort)
	
	// 注册会话管理路由
	r.mux.HandleFunc("/api/chat/sessions", r.handleSessionsRoute)
	r.mux.HandleFunc("/api/chat/sessions/", r.handleSessionsWithIDRoute)
	
	// 注册健康检查路由
	r.mux.HandleFunc("/health", r.healthHandler.Handle)
	
	// 应用中间件（按顺序：Recovery -> Logger -> CORS）
	var handler http.Handler = r.mux
	handler = r.corsConfig.Handler(handler)
	handler = middleware.Logger(handler)
	handler = middleware.Recovery(handler)
	
	return handler
}

// handleSessionsRoute 处理 /api/chat/sessions 路由
func (r *Router) handleSessionsRoute(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// 检查是否是搜索请求
		if req.URL.Query().Get("keyword") != "" {
			r.sessionHandler.SearchSessions(w, req)
		} else {
			r.sessionHandler.ListSessions(w, req)
		}
	case http.MethodPost:
		r.sessionHandler.CreateSession(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionsWithIDRoute 处理 /api/chat/sessions/{id} 及其子路由
func (r *Router) handleSessionsWithIDRoute(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	
	// 处理消息相关路由
	if containsPath(path, "/messages/stream") {
		if req.Method == http.MethodPost {
			r.messageHandler.SendMessageStream(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	if containsPath(path, "/messages") && !containsPath(path, "/messages/") {
		switch req.Method {
		case http.MethodGet:
			r.messageHandler.GetMessages(w, req)
		case http.MethodPost:
			r.messageHandler.SendMessage(w, req)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// 处理会话操作路由
	if containsPath(path, "/pin") {
		if req.Method == http.MethodPost {
			r.sessionHandler.PinSession(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	if containsPath(path, "/archive") {
		if req.Method == http.MethodPost {
			r.sessionHandler.ArchiveSession(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// 处理基本会话路由
	switch req.Method {
	case http.MethodGet:
		r.sessionHandler.GetSession(w, req)
	case http.MethodPatch:
		r.sessionHandler.UpdateSession(w, req)
	case http.MethodDelete:
		r.sessionHandler.DeleteSession(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// containsPath 检查路径是否包含指定的子路径
func containsPath(path, subPath string) bool {
	return len(path) >= len(subPath) && path[len(path)-len(subPath):] == subPath || 
		   len(path) > len(subPath) && path[len(path)-len(subPath)-1:len(path)-len(subPath)] == "/"
}

// Handler 返回配置好的 HTTP 处理器
func (r *Router) Handler() http.Handler {
	return r.Setup()
}
