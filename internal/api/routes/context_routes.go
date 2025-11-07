package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterContextRoutes 注册上下文管理相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
// 所有上下文路由都需要 JWT 认证和租户管理员权限
func RegisterContextRoutes(
	mux *http.ServeMux,
	contextHandler *handler.ContextHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 上下文管理路由（需要认证和租户管理员权限）==========
	
	// POST /api/v1/contexts/build - 构建上下文
	// 根据会话ID和用户查询构建智能对话上下文
	mux.Handle("POST /api/v1/contexts/build",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(contextHandler.HandleBuildContext))))

	// GET /api/v1/contexts/{sessionId} - 获取上下文配置
	// 获取指定会话的上下文配置信息
	mux.Handle("GET /api/v1/contexts/{sessionId}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(contextHandler.HandleGetContextConfig))))

	// PUT /api/v1/contexts/{sessionId} - 更新上下文配置
	// 更新指定会话的上下文配置（如最大Token数、策略等）
	mux.Handle("PUT /api/v1/contexts/{sessionId}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(contextHandler.HandleUpdateContextConfig))))
}
