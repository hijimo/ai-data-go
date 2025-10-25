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

// CreateTenantRequestSwagger 创建租户请求（用于 Swagger）
// @name CreateTenantRequest
type CreateTenantRequestSwagger struct {
	Name     string                 `json:"name" validate:"required,min=1,max=255" example:"示例租户"`
	Domain   string                 `json:"domain" validate:"omitempty,max=255" example:"example.com"`
	Metadata map[string]interface{} `json:"metadata,omitempty" swaggertype:"object"`
}

// UpdateTenantRequestSwagger 更新租户请求（用于 Swagger）
// @name UpdateTenantRequest
type UpdateTenantRequestSwagger struct {
	Name     *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255" example:"更新后的租户名"`
	Domain   *string                `json:"domain,omitempty" validate:"omitempty,max=255" example:"updated.com"`
	Metadata map[string]interface{} `json:"metadata,omitempty" swaggertype:"object"`
	Status   *bool                  `json:"status,omitempty" example:"true"`
}

// HandleCreate 处理创建租户
// @Summary 创建租户（仅平台管理员）
// @Description 创建新的租户并自动生成租户管理员账户
// @Description
// @Description **权限要求**：
// @Description - 仅平台管理员（system_admin）可以调用此接口
// @Description - 租户管理员和普通用户将收到 403 权限不足错误
// @Description
// @Description **功能说明**：
// @Description - 创建租户时会自动生成一个租户管理员账户
// @Description - 管理员邮箱默认为 admin@{tenantDomain}，也可以通过 adminEmail 参数自定义
// @Description - 系统会生成16位随机强密码，并在响应中返回（仅此一次）
// @Description - 建议管理员首次登录后立即修改密码
// @Description
// @Description **参数说明**：
// @Description - name: 租户名称（必填，1-255字符）
// @Description - domain: 租户域名（可选，最多255字符）
// @Description - metadata: 租户元数据（可选，JSON对象）
// @Description - adminEmail: 管理员邮箱（可选，默认为 admin@{domain}）
// @Description - adminDisplayName: 管理员显示名称（可选）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTenantRequestSwagger true "创建租户请求"
// @Success 201 {object} model.TenantDataResponse "创建成功，返回租户信息和管理员初始密码"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：需要平台管理员权限"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants [post]
func (h *TenantHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req auth.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建租户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建租户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到创建租户请求", logger.Fields{
		"name":   req.Name,
		"domain": req.Domain,
	})

	// 4. 调用服务层创建租户（权限验证在中间件层完成）
	tenant, err := h.tenantService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建租户失败", logger.Fields{
			"error": err,
			"name":  req.Name,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("创建租户成功", logger.Fields{
		"tenantId": tenant.ID,
		"name":     tenant.Name,
	})

	// 6. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessageContext(ctx, "创建租户成功", tenant)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleGet 处理获取租户详情
// @Summary 获取租户详情
// @Description 根据租户ID获取租户详细信息
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看任意租户的详细信息
// @Description - 租户管理员（tenant_admin）：只能查看自己所属租户的信息
// @Description
// @Description **访问控制**：
// @Description - 租户管理员尝试访问其他租户时，将收到 403 权限不足错误
// @Description - 系统会自动验证目标租户ID是否与当前用户的租户ID匹配
// @Description
// @Description **参数说明**：
// @Description - id: 租户ID（UUID格式）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.TenantDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法访问其他租户的数据"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [get]
func (h *TenantHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 记录请求日志
	h.logger.Info("收到获取租户详情请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 3. 调用服务层获取租户（权限验证在服务层完成）
	tenant, err := h.tenantService.Get(ctx, tenantID)
	if err != nil {
		h.logger.Error("获取租户详情失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewNotFoundError("租户不存在"))
		}
		return
	}

	// 4. 记录响应日志
	h.logger.Info("获取租户详情成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 5. 返回成功响应
	resp := response.SuccessWithContext(ctx, tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleUpdate 处理更新租户
// @Summary 更新租户
// @Description 更新租户信息，支持字段级权限控制
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以修改任意租户的所有字段
// @Description - 租户管理员（tenant_admin）：只能修改自己租户的 name 字段
// @Description
// @Description **字段级权限控制**：
// @Description - 租户管理员可修改字段：name（租户名称）
// @Description - 租户管理员不可修改字段：domain（域名）、metadata（元数据）、status（状态）
// @Description - 租户管理员尝试修改受限字段时，将收到 403 权限不足错误
// @Description - 平台管理员可以修改所有字段
// @Description
// @Description **访问控制**：
// @Description - 租户管理员只能更新自己所属的租户
// @Description - 租户管理员尝试更新其他租户时，将收到 403 权限不足错误
// @Description
// @Description **参数说明**：
// @Description - id: 租户ID（UUID格式）
// @Description - name: 租户名称（可选，1-255字符）
// @Description - domain: 租户域名（可选，最多255字符，仅平台管理员可修改）
// @Description - metadata: 租户元数据（可选，JSON对象，仅平台管理员可修改）
// @Description - status: 租户状态（可选，布尔值，仅平台管理员可修改）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body UpdateTenantRequestSwagger true "更新租户请求"
// @Success 200 {object} model.TenantDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：租户管理员只能修改租户名称，或无法访问其他租户"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [put]
func (h *TenantHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req auth.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新租户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新租户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到更新租户请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 5. 调用服务层更新租户（权限验证和字段级权限控制在服务层完成）
	tenant, err := h.tenantService.Update(ctx, tenantID, req)
	if err != nil {
		h.logger.Error("更新租户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("更新租户成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 7. 返回成功响应
	resp := response.SuccessWithMessageContext(ctx, "更新租户成功", tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleDelete 处理删除租户
// @Summary 删除租户（仅平台管理员）
// @Description 软删除指定的租户
// @Description
// @Description **权限要求**：
// @Description - 仅平台管理员（system_admin）可以调用此接口
// @Description - 租户管理员和普通用户将收到 403 权限不足错误
// @Description
// @Description **功能说明**：
// @Description - 执行软删除操作（设置 is_deleted=true）
// @Description - 不允许删除平台租户（type="system"）
// @Description - 删除后的租户数据仍保留在数据库中，但不再可见
// @Description
// @Description **参数说明**：
// @Description - id: 租户ID（UUID格式）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：需要平台管理员权限"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id} [delete]
func (h *TenantHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 记录请求日志
	h.logger.Info("收到删除租户请求", logger.Fields{
		"tenantId": tenantID,
	})

	// 3. 调用服务层删除租户（权限验证在中间件层完成）
	err := h.tenantService.Delete(ctx, tenantID)
	if err != nil {
		h.logger.Error("删除租户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 4. 记录响应日志
	h.logger.Info("删除租户成功", logger.Fields{
		"tenantId": tenantID,
	})

	// 5. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessageContext(ctx, "删除租户成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// UpdateTenantStatusRequest 更新租户状态请求
// @name UpdateTenantStatusRequest
type UpdateTenantStatusRequest struct {
	Status *bool `json:"status" validate:"required" example:"true"`
}

// HandleUpdateStatus 处理更新租户状态
// @Summary 更新租户状态（仅平台管理员）
// @Description 启用或禁用租户
// @Description
// @Description **权限要求**：
// @Description - 仅平台管理员（system_admin）可以调用此接口
// @Description - 租户管理员和普通用户将收到 403 权限不足错误
// @Description
// @Description **功能说明**：
// @Description - 启用租户（status=true）：租户下的所有用户可以正常访问系统
// @Description - 禁用租户（status=false）：租户下的所有用户将无法访问系统
// @Description - 不允许禁用平台租户（type="system"）
// @Description
// @Description **参数说明**：
// @Description - id: 租户ID（UUID格式）
// @Description - status: 租户状态（true=启用，false=禁用）
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body UpdateTenantStatusRequest true "更新租户状态请求"
// @Success 200 {object} model.TenantDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：需要平台管理员权限"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{id}/status [patch]
func (h *TenantHandler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取租户ID
	tenantID := h.extractTenantID(r.URL.Path)
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req UpdateTenantStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新租户状态请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新租户状态请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到更新租户状态请求", logger.Fields{
		"tenantId": tenantID,
		"status":   *req.Status,
	})

	// 5. 构建更新请求
	updateReq := auth.UpdateTenantRequest{
		Status: req.Status,
	}

	// 6. 调用服务层更新租户（权限验证在中间件层完成）
	tenant, err := h.tenantService.Update(ctx, tenantID, updateReq)
	if err != nil {
		h.logger.Error("更新租户状态失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("更新租户状态成功", logger.Fields{
		"tenantId": tenantID,
		"status":   tenant.Status,
	})

	// 8. 返回成功响应
	resp := response.SuccessWithMessageContext(ctx, "更新租户状态成功", tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleList 处理获取租户列表
// @Summary 获取租户列表
// @Description 获取租户列表，支持分页和过滤，不同角色返回不同的数据
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看所有租户列表
// @Description - 租户管理员（tenant_admin）：只能查看自己所属的租户信息
// @Description
// @Description **返回数据差异**：
// @Description - 平台管理员：返回所有租户的分页列表，可能包含多条记录
// @Description - 租户管理员：只返回当前用户所属租户的信息（单条记录），忽略分页和过滤参数
// @Description
// @Description **参数说明**：
// @Description - pageNo: 页码（从1开始，默认1）
// @Description - pageSize: 每页大小（1-100，默认20）
// @Description - name: 租户名称模糊搜索（可选）
// @Description - status: 租户状态过滤（可选，true=启用，false=禁用）
// @Description
// @Description **注意事项**：
// @Description - 租户管理员调用此接口时，所有过滤参数会被忽略
// @Description - 租户管理员始终只能看到自己的租户信息
// @Tags Tenant Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int false "页码" minimum(1) default(1) example:1
// @Param pageSize query int false "每页大小" minimum(1) maximum(100) default(20) example:20
// @Param name query string false "租户名称模糊搜索" example:"示例"
// @Param status query boolean false "租户状态过滤" example:true
// @Success 200 {object} model.TenantListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants [get]
func (h *TenantHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析查询参数
	pageNo, err := h.parseIntQuery(r, "pageNo", 1)
	if err != nil {
		h.logger.Error("解析页码参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的页码参数"))
		return
	}

	pageSize, err := h.parseIntQuery(r, "pageSize", 20)
	if err != nil {
		h.logger.Error("解析每页大小参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的每页大小参数"))
		return
	}

	// 解析过滤参数
	name := r.URL.Query().Get("name")
	statusStr := r.URL.Query().Get("status")
	
	var status *bool
	if statusStr != "" {
		if statusStr == "true" {
			trueVal := true
			status = &trueVal
		} else if statusStr == "false" {
			falseVal := false
			status = &falseVal
		} else {
			h.logger.Error("解析状态参数失败", logger.Fields{"status": statusStr})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的状态参数，必须是 true 或 false"))
			return
		}
	}

	// 2. 记录请求日志
	h.logger.Info("收到获取租户列表请求", logger.Fields{
		"pageNo":   pageNo,
		"pageSize": pageSize,
		"name":     name,
		"status":   status,
	})

	// 3. 构建过滤条件
	filter := auth.TenantListFilter{
		Name:   name,
		Status: status,
	}

	// 4. 调用服务层获取租户列表（权限验证和角色过滤在服务层完成）
	tenants, total, err := h.tenantService.ListWithFilter(ctx, pageNo, pageSize, filter)
	if err != nil {
		h.logger.Error("获取租户列表失败", logger.Fields{"error": err})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取租户列表成功", logger.Fields{
		"count": len(tenants),
		"total": total,
	})

	// 6. 返回分页响应
	h.writePaginationResponseWithContext(w, ctx, tenants, pageNo, pageSize, int(total))
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

// writeJSONResponse 写入 JSON 响应
func (h *TenantHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *TenantHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
	ctx := r.Context()

	resp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)

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
func (h *TenantHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
	// 构建验证错误详情
	errorData := map[string]interface{}{
		"errors": validationErrors,
	}

	ctx := r.Context()


	resp := response.ErrorWithDataContext(


		ctx,
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

// writePaginationResponseWithContext 写入分页响应（带 Context）
func (h *TenantHandler) writePaginationResponseWithContext(w http.ResponseWriter, ctx context.Context, data []*model.Tenant, pageNo, pageSize, total int) {
	resp := response.PaginationWithContext(ctx, data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}
