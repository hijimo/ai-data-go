package routes

import (
	"net/http"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/middleware"
)

// RegisterMonitoringRoutes 注册监控路由
func RegisterMonitoringRoutes(mux *http.ServeMux, h *handler.MonitoringHandler, jwtAuth func(http.Handler) http.Handler, rbac func(...string) func(http.Handler) http.Handler) {
	// 监控路由需要管理员权限
	
	// GET /api/v1/monitoring/metrics - 获取性能指标
	mux.Handle("/api/v1/monitoring/metrics", 
		jwtAuth(rbac("admin")(http.HandlerFunc(h.HandleMetrics))))
	
	// GET /api/v1/monitoring/alerts - 获取活跃告警
	mux.Handle("/api/v1/monitoring/alerts", 
		jwtAuth(rbac("admin")(http.HandlerFunc(h.HandleAlerts))))
	
	// DELETE /api/v1/monitoring/alerts - 清空告警
	mux.Handle("/api/v1/monitoring/alerts", 
		middleware.MethodFilter("DELETE", 
			jwtAuth(rbac("admin")(http.HandlerFunc(h.HandleClearAlerts)))))
	
	// POST /api/v1/monitoring/metrics/reset - 重置指标
	mux.Handle("/api/v1/monitoring/metrics/reset", 
		middleware.MethodFilter("POST", 
			jwtAuth(rbac("admin")(http.HandlerFunc(h.HandleResetMetrics)))))
	
	// GET /api/v1/monitoring/health - 健康检查（含监控信息）
	// 健康检查端点不需要认证，但需要管理员权限才能看到详细信息
	mux.Handle("/api/v1/monitoring/health", 
		http.HandlerFunc(h.HandleHealthCheck))
}
