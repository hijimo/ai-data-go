package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// TenantHandler 租户管理处理器
// 提供租户的创建、查询、更新、删除等功能
type TenantHandler struct {
	tenantService auth.TenantService
	logger        logger.Logger
	validator     *validator.Validator
}

// NewTenantHandler 创建租户处理器实例
// 参数：
//   - tenantService: 租户服务接口
//   - log: 日志记录器
// 返回：
//   - *TenantHandler: 租户处理器实例
func NewTenantHandler(tenantService auth.TenantService, log logger.Logger) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
		logger:        log,
		validator:     validator.New(),
	}
}

// CreateTenantRequest 创建租户请求（用于 Swagger）
type CreateTenantRequest struct {
	Name      string                 `json:"name" validate:"required,min=1,max=255" example:"示例租户"`
	Domain    string                 `json:"domain" validate:"omitempty,max=255" example:"example.com"`
	Metadata  map[string]interface{} `json:"metadata" swaggertype:"object"`
	CreatedBy *string                `json:"createdBy" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateTenantRequest 更新租户请求（用于 Swagger）
type UpdateTenantRequest struct {
	Name     *string                `json:"name" validate:"omitempty,min=1,max=255" example:"更新后的租户名"`
	Domain   *string                `json:"domain" validate:"omitempty,max=255" example:"updated.com"`
	Metadata map[string]interface{} `json:"metadata" swaggertype:"object"`
	Status   *bool                  `json:"status" example:"true"`
}

// HandleCreate 处理创建租户
// @Summary 创建租户
// @Description 创建新的租户（需要管理员权限）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTenantRequest true "创建租户请求"
// @Success 201 {object} model.TenantDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants [post]
func (h *TenantHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证管理员权限
	if !h.isAdmin(ctx) {
		h.logger.Warn("非管理员尝试创建租户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要管理员权限"))
		return
	}

	// 2. 解析请求参数
	var req auth.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建租户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建租户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到创建租户请求", logger.Fields{
		"name":   req.Name,
		"domain": req.Domain,
	})

	// 5. 调用服务层创建租户
	tenant, err := h.tenantService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建租户失败", logger.Fields{
			"error": err,
			"name":  req.Name,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("创建租户成功", logger.Fields{
		"tenantId": tenant.ID,
		"name":     tenant.Name,
	})

	// 7. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessage("创建租户成功", tenant)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleGet 处理获取租户详情
// @Summary 获取租户详情
// @Description 根据租户ID获取租户详细信息（需要管理员权限）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID"
// @Success 200 {object} model.TenantDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [get]
func (h *TenantHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证管理员权限
	if !h.isAdmin(ctx) {
		h.logger.Warn("非管理员尝试获取租户详情")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要管理员权限"))
		return
	}

	// 2. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取租户详情请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 4. 调用服务层获取租户
	tenant, err := h.tenantService.Get(ctx, tenantID)
	if err != nil {
		h.logger.Error("获取租户详情失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewNotFoundError("租户不存在"))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取租户详情成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 6. 返回成功响应
	resp := response.Success(tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleUpdate 处理更新租户
// @Summary 更新租户
// @Description 更新租户信息（需要管理员权限）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID"
// @Param request body UpdateTenantRequest true "更新租户请求"
// @Success 200 {object} model.TenantDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [put]
func (h *TenantHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证管理员权限
	if !h.isAdmin(ctx) {
		h.logger.Warn("非管理员尝试更新租户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要管理员权限"))
		return
	}

	// 2. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 3. 解析请求参数
	var req auth.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新租户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新租户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 5. 记录请求日志
	h.logger.Info("收到更新租户请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 6. 调用服务层更新租户
	tenant, err := h.tenantService.Update(ctx, tenantID, req)
	if err != nil {
		h.logger.Error("更新租户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("更新租户成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 8. 返回成功响应
	resp := response.SuccessWithMessage("更新租户成功", tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleDelete 处理删除租户
// @Summary 删除租户
// @Description 软删除指定的租户（需要管理员权限）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [delete]
func (h *TenantHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证管理员权限
	if !h.isAdmin(ctx) {
		h.logger.Warn("非管理员尝试删除租户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要管理员权限"))
		return
	}

	// 2. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到删除租户请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 4. 调用服务层删除租户
	err := h.tenantService.Delete(ctx, tenantID)
	if err != nil {
		h.logger.Error("删除租户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("删除租户成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("删除租户成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleList 处理获取租户列表
// @Summary 获取租户列表
// @Description 获取租户列表，支持分页（需要管理员权限）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(20)
// @Success 200 {object} model.TenantListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants [get]
func (h *TenantHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证管理员权限
	if !h.isAdmin(ctx) {
		h.logger.Warn("非管理员尝试获取租户列表")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要管理员权限"))
		return
	}

	// 2. 解析查询参数
	pageNo, err := h.parseIntQuery(r, "pageNo", 1)
	if err != nil {
		h.logger.Error("解析页码参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的页码参数"))
		return
	}

	pageSize, err := h.parseIntQuery(r, "pageSize", 20)
	if err != nil {
		h.logger.Error("解析每页大小参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的每页大小参数"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取租户列表请求", logger.Fields{
		"pageNo":   pageNo,
		"pageSize": pageSize,
	})

	// 4. 调用服务层获取租户列表
	tenants, total, err := h.tenantService.List(ctx, pageNo, pageSize)
	if err != nil {
		h.logger.Error("获取租户列表失败", logger.Fields{"error": err})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取租户列表成功", logger.Fields{
		"count": len(tenants),
		"total": total,
	})

	// 6. 返回分页响应
	h.writePaginationResponse(w, tenants, pageNo, pageSize, int(total))
}

// extractTenantID 从URL路径中提取租户ID
// 路径格式: /api/v1/tenants/{id}
func (h *TenantHandler) extractTenantID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "tenants" 后的部分
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// parseIntQuery 解析整数查询参数
func (h *TenantHandler) parseIntQuery(r *http.Request, key string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

// isAdmin 检查当前用户是否为管理员
func (h *TenantHandler) isAdmin(ctx context.Context) bool {
	// 从上下文获取角色信息
	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return false
	}

	// 检查是否包含 admin 角色
	for _, role := range roles {
		if role == "admin" {
			return true
		}
	}

	return false
}

// writeJSONResponse 写入 JSON 响应
func (h *TenantHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *TenantHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
	resp := response.Error[any](appErr.Code, appErr.Message)

	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}

	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *TenantHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
	// 构建验证错误详情
	errorData := map[string]interface{}{
		"errors": validationErrors,
	}

	resp := response.ErrorWithData(
		errors.CodeValidationError,
		errors.MsgValidationError,
		&errorData,
	)

	h.writeJSONResponse(w, http.StatusUnprocessableEntity, resp)
}

// writePaginationResponse 写入分页响应
func (h *TenantHandler) writePaginationResponse(w http.ResponseWriter, data []*model.Tenant, pageNo, pageSize, total int) {
	resp := response.Pagination(data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}
