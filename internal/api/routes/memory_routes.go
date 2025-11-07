package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterMemoryRoutes 注册记忆管理相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
// 所有记忆路由都需要 JWT 认证和租户管理员权限
func RegisterMemoryRoutes(
	mux *http.ServeMux,
	memoryHandler *handler.MemoryHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 记忆管理路由（需要认证和租户管理员权限）==========
	
	// POST /api/v1/memories/search - 检索记忆
	// 基于向量相似度检索相关的历史对话记忆
	mux.Handle("POST /api/v1/memories/search",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(memoryHandler.HandleSearchMemories))))

	// POST /api/v1/memories - 存储记忆
	// 将对话消息转换为长期记忆并存储
	mux.Handle("POST /api/v1/memories",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(memoryHandler.HandleStoreMemory))))

	// POST /api/v1/memories/cleanup - 清理记忆
	// 按策略清理过期或低质量的记忆
	mux.Handle("POST /api/v1/memories/cleanup",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(memoryHandler.HandleCleanupMemories))))

	// GET /api/v1/memories/{id} - 获取记忆详情
	// 获取指定ID的记忆详细信息
	mux.Handle("GET /api/v1/memories/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(memoryHandler.HandleGetMemory))))
}
