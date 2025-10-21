package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterPlatformRoutes 注册平台管理相关的API路由
// 所有平台管理路由都需要平台管理员（system_admin）权限
func RegisterPlatformRoutes(
	mux *http.ServeMux,
	platformHandler *handler.PlatformHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 平台管理路由（需要平台管理员权限）==========

	// POST /api/v1/platform/tenants - 创建租户（带管理员）
	mux.Handle("POST /api/v1/platform/tenants",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(platformHandler.HandleCreateTenant))))

	// GET /api/v1/platform/tenants - 列出所有租户（支持分页和类型过滤）
	mux.Handle("GET /api/v1/platform/tenants",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(platformHandler.HandleListTenants))))

	// PATCH /api/v1/platform/tenants/{id}/status - 启用/禁用租户
	mux.Handle("PATCH /api/v1/platform/tenants/{id}/status",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(platformHandler.HandleUpdateTenantStatus))))

	// DELETE /api/v1/platform/tenants/{id} - 删除租户
	mux.Handle("DELETE /api/v1/platform/tenants/{id}",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(platformHandler.HandleDeleteTenant))))
}
