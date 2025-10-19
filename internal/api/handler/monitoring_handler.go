package handler

import (
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/monitoring"
)

// MonitoringHandler 监控处理器
type MonitoringHandler struct {
	alertManager *monitoring.AlertManager
}

// NewMonitoringHandler 创建监控处理器
func NewMonitoringHandler(alertManager *monitoring.AlertManager) *MonitoringHandler {
	return &MonitoringHandler{
		alertManager: alertManager,
	}
}

// HandleMetrics 处理 metrics 请求
// @Summary 获取性能指标
// @Description 获取认证系统的性能监控指标
// @Tags 监控
// @Produce json
// @Success 200 {object} model.MetricsDataResponse
// @Router /monitoring/metrics [get]
func (h *MonitoringHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := monitoring.GetMetrics().GetSnapshot()
	
	resp := model.ResponseData[monitoring.MetricsSnapshot]{
		Code:    http.StatusOK,
		Message: "获取指标成功",
		Data:    &snapshot,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleAlerts 处理告警查询请求
// @Summary 获取活跃告警
// @Description 获取当前活跃的告警列表
// @Tags 监控
// @Produce json
// @Success 200 {object} model.AlertListDataResponse
// @Router /monitoring/alerts [get]
func (h *MonitoringHandler) HandleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := h.alertManager.GetActiveAlerts()
	
	resp := model.ResponseData[[]monitoring.Alert]{
		Code:    http.StatusOK,
		Message: "获取告警成功",
		Data:    &alerts,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleClearAlerts 处理清空告警请求
// @Summary 清空告警
// @Description 清空所有活跃告警
// @Tags 监控
// @Produce json
// @Success 200 {object} model.AnyDataResponse
// @Router /monitoring/alerts [delete]
func (h *MonitoringHandler) HandleClearAlerts(w http.ResponseWriter, r *http.Request) {
	h.alertManager.ClearAlerts()
	
	resp := model.ResponseData[any]{
		Code:    http.StatusOK,
		Message: "清空告警成功",
		Data:    nil,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleResetMetrics 处理重置指标请求
// @Summary 重置指标
// @Description 重置所有性能监控指标
// @Tags 监控
// @Produce json
// @Success 200 {object} model.AnyDataResponse
// @Router /monitoring/metrics/reset [post]
func (h *MonitoringHandler) HandleResetMetrics(w http.ResponseWriter, r *http.Request) {
	monitoring.GetMetrics().Reset()
	
	resp := model.ResponseData[any]{
		Code:    http.StatusOK,
		Message: "重置指标成功",
		Data:    nil,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleHealthCheck 处理健康检查请求（包含监控信息）
// @Summary 健康检查（含监控）
// @Description 获取系统健康状态和关键监控指标
// @Tags 监控
// @Produce json
// @Success 200 {object} model.HealthDataResponse
// @Router /monitoring/health [get]
func (h *MonitoringHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	snapshot := monitoring.GetMetrics().GetSnapshot()
	alerts := h.alertManager.GetActiveAlerts()
	
	// 判断健康状态
	status := "healthy"
	criticalAlerts := 0
	for _, alert := range alerts {
		if alert.Level == monitoring.AlertLevelCritical {
			criticalAlerts++
		}
	}
	
	if criticalAlerts > 0 {
		status = "unhealthy"
	} else if len(alerts) > 0 {
		status = "degraded"
	}
	
	healthData := map[string]interface{}{
		"status":          status,
		"timestamp":       snapshot.Timestamp,
		"active_alerts":   len(alerts),
		"critical_alerts": criticalAlerts,
		"metrics": map[string]interface{}{
			"login_success_rate":    snapshot.LoginSuccessRate,
			"avg_login_duration_ms": snapshot.AvgLoginDuration,
			"slow_queries":          snapshot.SlowQueries,
			"db_errors":             snapshot.DBErrors,
			"active_tenants":        snapshot.ActiveTenants,
			"active_users":          snapshot.ActiveUsers,
		},
	}
	
	resp := model.ResponseData[map[string]interface{}]{
		Code:    http.StatusOK,
		Message: "健康检查成功",
		Data:    &healthData,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
