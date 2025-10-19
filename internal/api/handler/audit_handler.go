package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"

	"github.com/google/uuid"
)

// AuditHandler 审计日志处理器
// 提供审计日志查询功能
type AuditHandler struct {
	auditRepo repository.AuditRepository
	logger    logger.Logger
}

// NewAuditHandler 创建审计日志处理器实例
// 参数：
//   - auditRepo: 审计日志仓储接口
//   - log: 日志记录器
// 返回：
//   - *AuditHandler: 审计日志处理器实例
func NewAuditHandler(auditRepo repository.AuditRepository, log logger.Logger) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
		logger:    log,
	}
}

// AuditQueryRequest 审计日志查询请求（用于 Swagger）
type AuditQueryRequest struct {
	TenantID  string `json:"tenantId" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Event     string `json:"event" example:"login"`
	StartTime string `json:"startTime" example:"2024-01-01T00:00:00Z"`
	EndTime   string `json:"endTime" example:"2024-12-31T23:59:59Z"`
	Page      int    `json:"page" example:"1"`
	PageSize  int    `json:"pageSize" example:"10"`
}

// HandleListAuditLogs 处理审计日志查询
// @Summary 查询审计日志
// @Description 查询认证审计日志，支持多条件过滤和分页
// @Tags 审计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenantId query string false "租户ID"
// @Param userId query string false "用户ID"
// @Param event query string false "事件类型（login, logout, refresh, revoke, failed_login）"
// @Param startTime query string false "开始时间（RFC3339格式）"
// @Param endTime query string false "结束时间（RFC3339格式）"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Success 200 {object} model.ResponsePaginationData[[]model.AuthAudit] "查询成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /api/v1/audit/auth [get]
func (h *AuditHandler) HandleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析查询参数
	query := r.URL.Query()

	// 解析分页参数
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 10
	if pageSizeStr := query.Get("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// 构建过滤条件
	filter := repository.AuditFilter{}

	// 解析租户 ID
	if tenantIDStr := query.Get("tenantId"); tenantIDStr != "" {
		if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
			filter.TenantID = &tenantID
		} else {
			h.logger.Warn("无效的租户ID", logger.Fields{"tenantId": tenantIDStr})
			h.writeErrorResponse(w, errors.NewBadRequestError("无效的租户ID"))
			return
		}
	}

	// 解析用户 ID
	if userIDStr := query.Get("userId"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		} else {
			h.logger.Warn("无效的用户ID", logger.Fields{"userId": userIDStr})
			h.writeErrorResponse(w, errors.NewBadRequestError("无效的用户ID"))
			return
		}
	}

	// 解析事件类型
	if event := query.Get("event"); event != "" {
		filter.Event = event
	}

	// 解析开始时间
	if startTimeStr := query.Get("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		} else {
			h.logger.Warn("无效的开始时间", logger.Fields{"startTime": startTimeStr})
			h.writeErrorResponse(w, errors.NewBadRequestError("无效的开始时间格式，请使用RFC3339格式"))
			return
		}
	}

	// 解析结束时间
	if endTimeStr := query.Get("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		} else {
			h.logger.Warn("无效的结束时间", logger.Fields{"endTime": endTimeStr})
			h.writeErrorResponse(w, errors.NewBadRequestError("无效的结束时间格式，请使用RFC3339格式"))
			return
		}
	}

	// 2. 查询审计日志
	audits, total, err := h.auditRepo.List(ctx, filter, page, pageSize)
	if err != nil {
		h.logger.Error("查询审计日志失败", logger.Fields{"error": err, "filter": filter})
		h.writeErrorResponse(w, errors.NewInternalError(err))
		return
	}

	// 3. 返回分页响应
	h.logger.Info("查询审计日志成功", logger.Fields{
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
		"count":    len(audits),
	})

	resp := response.PaginationWithMessage(
		"查询审计日志成功",
		audits,
		page,
		pageSize,
		int(total),
	)

	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeJSONResponse 写入 JSON 响应
func (h *AuditHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("编码响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *AuditHandler) writeErrorResponse(w http.ResponseWriter, err *errors.AppError) {
	resp := response.Error[any](err.Code, err.Message)
	// 根据错误码确定 HTTP 状态码
	httpStatus := http.StatusInternalServerError
	if err.Code >= 400 && err.Code < 500 {
		httpStatus = err.Code
	} else if err.Code >= 500 && err.Code < 600 {
		httpStatus = http.StatusInternalServerError
	}
	h.writeJSONResponse(w, httpStatus, resp)
}
