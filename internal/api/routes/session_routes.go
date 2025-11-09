package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterSessionRoutes 注册会话管理相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
// 所有会话和消息路由都需要 JWT 认证
func RegisterSessionRoutes(
	mux *http.ServeMux,
	sessionHandler *handler.SessionHandler,
	messageHandler *handler.MessageHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
) {
	// ========== 会话管理路由（需要认证）==========
	
	// POST /api/v1/chat/sessions - 创建新会话
	mux.Handle("POST /api/v1/chat/sessions",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.CreateSession)))

	// GET /api/v1/chat/sessions - 获取会话列表（支持分页和过滤）
	mux.Handle("GET /api/v1/chat/sessions",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.ListSessions)))

	// GET /api/v1/chat/sessions/search - 搜索会话
	// 注意：这个路由必须在 /api/v1/chat/sessions/{id} 之前注册，避免 "search" 被当作 ID
	mux.Handle("GET /api/v1/chat/sessions/search",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.SearchSessions)))

	// GET /api/v1/chat/sessions/{id} - 获取会话详情
	mux.Handle("GET /api/v1/chat/sessions/{id}",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.GetSession)))

	// PATCH /api/v1/chat/sessions/{id} - 更新会话
	mux.Handle("PATCH /api/v1/chat/sessions/{id}",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.UpdateSession)))

	// DELETE /api/v1/chat/sessions/{id} - 删除会话（软删除）
	mux.Handle("DELETE /api/v1/chat/sessions/{id}",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.DeleteSession)))

	// POST /api/v1/chat/sessions/{id}/pin - 置顶/取消置顶会话
	mux.Handle("POST /api/v1/chat/sessions/{id}/pin",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.PinSession)))

	// POST /api/v1/chat/sessions/{id}/archive - 归档/取消归档会话
	mux.Handle("POST /api/v1/chat/sessions/{id}/archive",
		jwtAuthMiddleware(http.HandlerFunc(sessionHandler.ArchiveSession)))

	// ========== 消息管理路由（需要认证）==========

	// POST /api/v1/chat/sessions/{id}/messages/stream - 在会话中发送消息（流式返回）
	// 注意：这个路由必须在 /api/v1/chat/sessions/{id}/messages 之前注册
	mux.Handle("POST /api/v1/chat/sessions/{id}/messages/stream",
		jwtAuthMiddleware(http.HandlerFunc(messageHandler.SendMessageStream)))

	// POST /api/v1/chat/sessions/{id}/messages - 在会话中发送消息
	mux.Handle("POST /api/v1/chat/sessions/{id}/messages",
		jwtAuthMiddleware(http.HandlerFunc(messageHandler.SendMessage)))

	// GET /api/v1/chat/sessions/{id}/messages - 获取会话的消息历史（支持分页）
	mux.Handle("GET /api/v1/chat/sessions/{id}/messages",
		jwtAuthMiddleware(http.HandlerFunc(messageHandler.GetMessages)))

	// GET /api/v1/chat/messages/{id} - 获取单条消息详情
	mux.Handle("GET /api/v1/chat/messages/{id}",
		jwtAuthMiddleware(http.HandlerFunc(messageHandler.GetMessageByID)))

	// POST /api/v1/chat/messages/{id}/abort - 中止消息生成
	mux.Handle("POST /api/v1/chat/messages/{id}/abort",
		jwtAuthMiddleware(http.HandlerFunc(messageHandler.AbortMessage)))
}
