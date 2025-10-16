package handler

import (
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/health"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	healthService health.Service
	logger        logger.Logger
}

// NewHealthHandler 创建新的健康检查处理器
func NewHealthHandler(healthService health.Service, logger logger.Logger) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
		logger:        logger,
	}
}

// Handle 处理健康检查请求
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 记录请求日志
	h.logger.Info("收到健康检查请求", map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})

	// 执行健康检查
	healthStatus, err := h.healthService.Check(ctx)
	if err != nil {
		h.logger.Error("健康检查失败", map[string]interface{}{
			"error": err.Error(),
		})

		// 返回错误响应
		resp := response.Error[health.HealthStatus](
			errors.CodeInternalError,
			"健康检查失败",
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 根据健康状态设置 HTTP 状态码
	httpStatus := http.StatusOK
	if healthStatus.Status != "healthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	// 构建成功响应
	resp := response.Success(healthStatus)

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("编码响应失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 记录响应日志
	h.logger.Info("健康检查完成", map[string]interface{}{
		"status":     healthStatus.Status,
		"httpStatus": httpStatus,
	})
}
