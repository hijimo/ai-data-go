package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
)

// RegisterModelConfigurationRoutes 注册模型配置相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
//
// 路由列表：
//   - POST /api/v1/model-configurations - 创建模型配置（需要租户管理员或平台管理员权限）
//   - GET /api/v1/model-configurations - 查询模型配置列表（需要租户管理员或平台管理员权限）
//   - GET /api/v1/model-configurations/available - 查询可用模型列表（需要JWT认证）
//   - GET /api/v1/model-configurations/{id} - 查询单个模型配置（需要租户管理员或平台管理员权限）
//   - PUT /api/v1/model-configurations/{id} - 更新模型配置（需要租户管理员或平台管理员权限）
//   - PATCH /api/v1/model-configurations/{id}/status - 更新模型配置状态（需要租户管理员或平台管理员权限）
//   - DELETE /api/v1/model-configurations/{id} - 删除模型配置（需要租户管理员或平台管理员权限）
//   - POST /api/v1/model-configurations/{id}/validate - 验证模型配置（需要租户管理员或平台管理员权限）
//
// 参数：
//   - mux: HTTP ServeMux 实例
//   - modelConfigHandler: 模型配置处理器
//   - jwtAuthMiddleware: JWT 认证中间件
//   - rbacMiddleware: RBAC 授权中间件工厂函数
func RegisterModelConfigurationRoutes(
	mux *http.ServeMux,
	modelConfigHandler *handler.ModelConfigurationHandler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 模型配置管理路由（需要租户管理员或平台管理员权限）==========

	// POST /api/v1/model-configurations - 创建模型配置
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能在自己的租户下创建配置
	// 平台管理员可以在任意租户下创建配置
	mux.Handle("POST /api/v1/model-configurations",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleCreate))))

	// GET /api/v1/model-configurations - 查询模型配置列表
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能查看自己租户的配置
	// 平台管理员可以查看所有租户的配置（可通过 tenantId 参数过滤）
	mux.Handle("GET /api/v1/model-configurations",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleList))))

	// GET /api/v1/model-configurations/available - 查询可用模型列表
	// 权限：所有已认证用户
	// 返回当前租户下已启用且未删除的模型配置
	// 注意：此路由必须在 GET /api/v1/model-configurations/{id} 之前注册
	mux.Handle("GET /api/v1/model-configurations/available",
		jwtAuthMiddleware(http.HandlerFunc(modelConfigHandler.HandleListAvailable)))

	// GET /api/v1/model-configurations/{id} - 查询单个模型配置
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能查看自己租户的配置
	// 平台管理员可以查看任意租户的配置
	mux.Handle("GET /api/v1/model-configurations/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleGet))))

	// PUT /api/v1/model-configurations/{id} - 更新模型配置
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能更新自己租户的配置
	// 平台管理员可以更新任意租户的配置
	mux.Handle("PUT /api/v1/model-configurations/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleUpdate))))

	// PATCH /api/v1/model-configurations/{id}/status - 更新模型配置状态
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能更新自己租户的配置状态
	// 平台管理员可以更新任意租户的配置状态
	mux.Handle("PATCH /api/v1/model-configurations/{id}/status",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleUpdateStatus))))

	// DELETE /api/v1/model-configurations/{id} - 删除模型配置（软删除）
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能删除自己租户的配置
	// 平台管理员可以删除任意租户的配置
	mux.Handle("DELETE /api/v1/model-configurations/{id}",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleDelete))))

	// POST /api/v1/model-configurations/{id}/validate - 验证模型配置
	// 权限：租户管理员（tenant_admin）或平台管理员（system_admin）
	// 租户管理员只能验证自己租户的配置
	// 平台管理员可以验证任意租户的配置
	// 验证操作会尝试连接到模型提供商，验证配置是否有效
	mux.Handle("POST /api/v1/model-configurations/{id}/validate",
		jwtAuthMiddleware(rbacMiddleware("tenant_admin")(http.HandlerFunc(modelConfigHandler.HandleValidate))))
}
