package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/service/auth"
)

// RegisterAuthRoutes 注册认证相关的API路由
// 使用 Go 1.22+ 的新路由模式定义路径参数
func RegisterAuthRoutes(
	mux *http.ServeMux,
	authHandler *handler.AuthHandler,
	tenantHandler *handler.TenantHandler,
	userHandler *handler.UserHandler,
	auditHandler *handler.AuditHandler,
	tenantMiddleware func(http.Handler) http.Handler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// ========== 公开认证路由（不需要认证）==========
	
	// POST /api/v1/auth/register - 用户注册
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.HandleRegister)

	// POST /api/v1/auth/login - 用户登录
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.HandleLogin)

	// POST /api/v1/auth/refresh - 刷新 Token
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.HandleRefresh)

	// POST /api/v1/auth/verify-email - 验证邮箱
	mux.HandleFunc("POST /api/v1/auth/verify-email", authHandler.HandleVerifyEmail)

	// POST /api/v1/auth/resend-verification - 重新发送验证邮件
	mux.HandleFunc("POST /api/v1/auth/resend-verification", authHandler.HandleResendVerification)

	// ========== 需要认证的路由 ==========
	
	// POST /api/v1/auth/logout - 用户注销（需要认证）
	mux.Handle("POST /api/v1/auth/logout",
		tenantMiddleware(jwtAuthMiddleware(http.HandlerFunc(authHandler.HandleLogout))))

	// POST /api/v1/auth/change-password - 修改密码（需要认证）
	mux.Handle("POST /api/v1/auth/change-password",
		tenantMiddleware(jwtAuthMiddleware(http.HandlerFunc(authHandler.HandleChangePassword))))

	// GET /api/v1/auth/me - 获取当前用户信息（需要认证）
	mux.Handle("GET /api/v1/auth/me",
		tenantMiddleware(jwtAuthMiddleware(http.HandlerFunc(authHandler.HandleMe))))

	// POST /api/v1/auth/unlock-account - 解锁账户（需要管理员权限）
	mux.Handle("POST /api/v1/auth/unlock-account",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(authHandler.HandleUnlockAccount)))))

	// ========== 租户管理路由（需要系统管理员权限）==========
	
	// POST /api/v1/tenants - 创建租户（需要系统管理员权限）
	mux.Handle("POST /api/v1/tenants",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(tenantHandler.HandleCreate))))

	// GET /api/v1/tenants - 列出租户（需要系统管理员权限）
	mux.Handle("GET /api/v1/tenants",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(tenantHandler.HandleList))))

	// GET /api/v1/tenants/{id} - 获取租户详情（需要系统管理员权限）
	mux.Handle("GET /api/v1/tenants/{id}",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(tenantHandler.HandleGet))))

	// PUT /api/v1/tenants/{id} - 更新租户（需要系统管理员权限）
	mux.Handle("PUT /api/v1/tenants/{id}",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(tenantHandler.HandleUpdate))))

	// DELETE /api/v1/tenants/{id} - 删除租户（需要系统管理员权限）
	mux.Handle("DELETE /api/v1/tenants/{id}",
		jwtAuthMiddleware(rbacMiddleware("system_admin")(http.HandlerFunc(tenantHandler.HandleDelete))))

	// ========== 用户管理路由（需要租户管理员权限）==========
	
	// POST /api/v1/users - 创建用户（需要租户管理员权限）
	mux.Handle("POST /api/v1/users",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(userHandler.HandleCreate)))))

	// GET /api/v1/users - 列出用户（需要租户管理员权限）
	mux.Handle("GET /api/v1/users",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(userHandler.HandleList)))))

	// GET /api/v1/users/{id} - 获取用户详情（需要租户管理员权限）
	mux.Handle("GET /api/v1/users/{id}",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(userHandler.HandleGet)))))

	// PUT /api/v1/users/{id} - 更新用户（需要租户管理员权限）
	mux.Handle("PUT /api/v1/users/{id}",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(userHandler.HandleUpdate)))))

	// DELETE /api/v1/users/{id} - 删除用户（需要租户管理员权限）
	mux.Handle("DELETE /api/v1/users/{id}",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(userHandler.HandleDelete)))))

	// ========== 审计日志路由（需要管理员权限）==========
	
	// GET /api/v1/audit/auth - 查询审计日志（需要管理员权限）
	mux.Handle("GET /api/v1/audit/auth",
		tenantMiddleware(jwtAuthMiddleware(rbacMiddleware("admin")(http.HandlerFunc(auditHandler.HandleListAuditLogs)))))
}

// WrapAuthMiddleware 包装认证中间件，用于简化路由注册
// 这个辅助函数可以让路由注册更简洁
func WrapAuthMiddleware(
	tokenService auth.TokenService,
	blacklistService auth.TokenBlacklistService,
	tenantRepo interface{},
	tenantStrategy string,
) (
	tenantMiddleware func(http.Handler) http.Handler,
	jwtAuthMiddleware func(http.Handler) http.Handler,
	rbacMiddleware func(...string) func(http.Handler) http.Handler,
) {
	// 创建租户识别中间件配置
	tenantConfig := middleware.TenantIdentifierConfig{
		Strategy:   tenantStrategy,
		TenantRepo: nil, // 暂时不使用 TenantRepo 验证
		BaseDomain: "",
	}
	tenantMiddleware = middleware.TenantIdentifier(tenantConfig)
	
	// 创建 JWT 认证中间件（传入黑名单服务）
	jwtAuthMiddleware = middleware.JWTAuth(tokenService, blacklistService)
	
	// 创建 RBAC 授权中间件工厂函数
	rbacMiddleware = func(roles ...string) func(http.Handler) http.Handler {
		config := middleware.RBACConfig{
			RequiredRoles: roles,
			RequireAll:    false,
		}
		return middleware.RBACAuthorizer(config)
	}

	return
}
