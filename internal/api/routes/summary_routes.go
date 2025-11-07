package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterSummaryRoutes 注册摘要管理相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
// 所有摘要路由都需要 JWT 认证和租户管理员权限
func RegisterSummaryRoutes(
	mux *http.ServeMux,
	summaryHandler *handler.SummaryHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 摘要管理路由（需要认证和租户管理员权限）==========
	
	// POST /api/v1/summaries - 生成摘要
	// 为指定会话生成对话摘要
	mux.Handle("POST /api/v1/summaries",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(summaryHandler.HandleGenerateSummary))))

	// GET /api/v1/summaries/{id} - 获取摘要详情
	// 获取指定ID的摘要详细信息
	mux.Handle("GET /api/v1/summaries/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(summaryHandler.HandleGetSummary))))

	// GET /api/v1/summaries/session/{sessionId} - 获取会话摘要列表
	// 获取指定会话的所有摘要列表
	mux.Handle("GET /api/v1/summaries/session/{sessionId}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(summaryHandler.HandleListSummaries))))

	// POST /api/v1/summaries/check-trigger - 检查摘要触发条件
	// 检查指定会话是否需要生成摘要
	mux.Handle("POST /api/v1/summaries/check-trigger",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(summaryHandler.HandleCheckTrigger))))
}
