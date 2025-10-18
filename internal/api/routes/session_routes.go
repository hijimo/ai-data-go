package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterSessionRoutes 注册会话管理相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
func RegisterSessionRoutes(mux *http.ServeMux, sessionHandler *handler.SessionHandler, messageHandler *handler.MessageHandler) {
	// ========== 会话管理路由 ==========
	
	// POST /api/v1/chat/sessions - 创建新会话
	mux.HandleFunc("POST /api/v1/chat/sessions", sessionHandler.CreateSession)

	// GET /api/v1/chat/sessions - 获取会话列表（支持分页和过滤）
	mux.HandleFunc("GET /api/v1/chat/sessions", sessionHandler.ListSessions)

	// GET /api/v1/chat/sessions/search - 搜索会话
	// 注意：这个路由必须在 /api/v1/chat/sessions/{id} 之前注册，避免 "search" 被当作 ID
	mux.HandleFunc("GET /api/v1/chat/sessions/search", sessionHandler.SearchSessions)

	// GET /api/v1/chat/sessions/{id} - 获取会话详情
	mux.HandleFunc("GET /api/v1/chat/sessions/{id}", sessionHandler.GetSession)

	// PATCH /api/v1/chat/sessions/{id} - 更新会话
	mux.HandleFunc("PATCH /api/v1/chat/sessions/{id}", sessionHandler.UpdateSession)

	// DELETE /api/v1/chat/sessions/{id} - 删除会话（软删除）
	mux.HandleFunc("DELETE /api/v1/chat/sessions/{id}", sessionHandler.DeleteSession)

	// POST /api/v1/chat/sessions/{id}/pin - 置顶/取消置顶会话
	mux.HandleFunc("POST /api/v1/chat/sessions/{id}/pin", sessionHandler.PinSession)

	// POST /api/v1/chat/sessions/{id}/archive - 归档/取消归档会话
	mux.HandleFunc("POST /api/v1/chat/sessions/{id}/archive", sessionHandler.ArchiveSession)

	// ========== 消息管理路由 ==========

	// POST /api/v1/chat/sessions/{id}/messages - 在会话中发送消息
	mux.HandleFunc("POST /api/v1/chat/sessions/{id}/messages", messageHandler.SendMessage)

	// GET /api/v1/chat/sessions/{id}/messages - 获取会话的消息历史（支持分页）
	mux.HandleFunc("GET /api/v1/chat/sessions/{id}/messages", messageHandler.GetMessages)

	// GET /api/v1/chat/messages/{id} - 获取单条消息详情
	mux.HandleFunc("GET /api/v1/chat/messages/{id}", messageHandler.GetMessageByID)

	// POST /api/v1/chat/messages/{id}/abort - 中止消息生成
	mux.HandleFunc("POST /api/v1/chat/messages/{id}/abort", messageHandler.AbortMessage)
}
