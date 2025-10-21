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

// PlatformHandler 平台管理处理器
// 提供平台管理员对租户的管理功能，包括创建、查询、启用/禁用、删除租户
type PlatformHandler struct {
	tenantService auth.TenantService
	logger        logger.Logger
	validator     *validator.Validator
}

// NewPlatformHandler 创建平台管理处理器实例
// 参数：
//   - tenantService: 租户服务接口
//   - log: 日志记录器
// 返回：
//   - *PlatformHandler: 平台管理处理器实例
func NewPlatformHandler(tenantService auth.TenantService, log logger.Logger) *PlatformHandler {
	return &PlatformHandler{
		tenantService: tenantService,
		logger:        log,
		validator:     validator.New(),
	}
}

// isSystemAdmin 检查当前用户是否为平台管理员
func (h *PlatformHandler) isSystemAdmin(ctx context.Context) bool {
	// 从上下文获取角色信息
	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return false
	}

	// 检查是否包含 system_admin 角色
	for _, role := range roles {
		if role == model.RoleSystemAdmin {
			return true
		}
	}

	return false
}

// extractIDFromPath 从URL路径中提取ID
// 路径格式: /api/v1/platform/tenants/{id}
func (h *PlatformHandler) extractIDFromPath(path string, resource string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找资源名称后的部分
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// parseIntQuery 解析整数查询参数
func (h *PlatformHandler) parseIntQuery(r *http.Request, key string, defaultValue int) (int, error) {
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
func (h *PlatformHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *PlatformHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
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
func (h *PlatformHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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
func (h *PlatformHandler) writePaginationResponse(w http.ResponseWriter, data []*model.Tenant, pageNo, pageSize, total int) {
	resp := response.Pagination(data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}


// CreateTenantWithAdminRequest 创建租户（带管理员）请求（用于 Swagger）
// @name CreateTenantWithAdminRequest
type CreateTenantWithAdminRequest struct {
	// 租户名称（必填，1-255字符）
	TenantName string `json:"tenantName" validate:"required,min=1,max=255" example:"示例公司"`
	// 租户域名（必填，最多255字符）
	TenantDomain string `json:"tenantDomain" validate:"required,max=255" example:"example.com"`
	// 租户元数据（可选，JSON对象）
	TenantMetadata map[string]interface{} `json:"tenantMetadata" swaggertype:"object"`
	// 管理员邮箱（可选，默认为 admin@{tenantDomain}）
	AdminEmail string `json:"adminEmail" validate:"omitempty,email" example:"admin@example.com"`
	// 管理员显示名称（可选，最多255字符）
	AdminDisplayName string `json:"adminDisplayName" validate:"omitempty,max=255" example:"管理员"`
}

// CreateTenantWithAdminResponse 创建租户（带管理员）响应（用于 Swagger）
// @name CreateTenantWithAdminResponse
type CreateTenantWithAdminResponse struct {
	// 租户信息
	Tenant *model.Tenant `json:"tenant"`
	// 管理员用户信息
	AdminUser *model.User `json:"adminUser"`
	// 管理员初始密码（请妥善保管并建议首次登录后修改）
	AdminPassword string `json:"adminPassword" example:"Xy9#mK2$pL5@qR8!"`
}

// CreateTenantWithAdminDataResponse 创建租户数据响应（用于 Swagger）
// @name CreateTenantWithAdminDataResponse
type CreateTenantWithAdminDataResponse struct {
	// 响应代码
	Code int `json:"code" example:"201"`
	// 响应信息
	Message string `json:"message" example:"租户创建成功"`
	// 创建租户响应数据
	Data *CreateTenantWithAdminResponse `json:"data"`
}

// HandleCreateTenant 处理创建租户（带管理员）
// @Summary 创建租户（带管理员）
// @Description 创建新的业务租户并自动生成租户管理员账户。系统会自动为租户创建一个管理员账户，并生成随机强密码。需要平台管理员权限（system_admin角色）。
// @Description
// @Description **权限要求：** system_admin
// @Description
// @Description **功能说明：**
// @Description - 创建类型为 "tenant" 的业务租户
// @Description - 自动生成租户管理员账户（角色为 tenant_admin）
// @Description - 生成16位随机强密码（包含大小写字母、数字和特殊字符）
// @Description - 返回租户信息和管理员初始密码
// @Description
// @Description **注意事项：**
// @Description - 管理员邮箱如未指定，默认为 admin@{tenantDomain}
// @Description - 请妥善保管返回的初始密码，建议首次登录后立即修改
// @Description - 租户域名必须唯一
// @Tags Platform Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTenantWithAdminRequest true "创建租户请求"
// @Success 201 {object} CreateTenantWithAdminDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足（需要 system_admin 角色）"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /api/v1/platform/tenants [post]
func (h *PlatformHandler) HandleCreateTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证平台管理员权限
	if !h.isSystemAdmin(ctx) {
		h.logger.Warn("非平台管理员尝试创建租户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要平台管理员权限"))
		return
	}

	// 2. 解析请求参数
	var req auth.CreateTenantWithAdminRequest
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
		"tenantName":   req.TenantName,
		"tenantDomain": req.TenantDomain,
		"adminEmail":   req.AdminEmail,
	})

	// 5. 调用服务层创建租户和管理员
	result, err := h.tenantService.CreateWithAdmin(ctx, req)
	if err != nil {
		h.logger.Error("创建租户失败", logger.Fields{
			"error":      err,
			"tenantName": req.TenantName,
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
		"tenantId":    result.Tenant.ID,
		"tenantName":  result.Tenant.Name,
		"adminUserId": result.AdminUser.ID,
		"adminEmail":  result.AdminUser.Email,
	})

	// 7. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessage("租户创建成功", result)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}


// HandleListTenants 处理获取租户列表
// @Summary 获取租户列表
// @Description 获取所有租户列表，支持分页和类型过滤。需要平台管理员权限（system_admin角色）。
// @Description
// @Description **权限要求：** system_admin
// @Description
// @Description **功能说明：**
// @Description - 支持分页查询（默认每页10条，最多100条）
// @Description - 支持按租户类型过滤（system: 平台租户, tenant: 业务租户）
// @Description - 返回租户的完整信息，包括名称、域名、类型、状态等
// @Description
// @Description **租户类型：**
// @Description - system: 平台租户（系统级租户，用于承载平台管理员）
// @Description - tenant: 业务租户（普通租户，用于实际业务使用）
// @Tags Platform Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int false "页码（从1开始）" minimum(1) default(1) example(1)
// @Param pageSize query int false "每页大小（1-100）" minimum(1) maximum(100) default(10) example(10)
// @Param type query string false "租户类型过滤" Enums(system, tenant) example(tenant)
// @Success 200 {object} model.TenantListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足（需要 system_admin 角色）"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /api/v1/platform/tenants [get]
func (h *PlatformHandler) HandleListTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证平台管理员权限
	if !h.isSystemAdmin(ctx) {
		h.logger.Warn("非平台管理员尝试获取租户列表")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要平台管理员权限"))
		return
	}

	// 2. 解析查询参数
	pageNo, err := h.parseIntQuery(r, "pageNo", 1)
	if err != nil || pageNo < 1 {
		h.logger.Error("解析页码参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的页码参数"))
		return
	}

	pageSize, err := h.parseIntQuery(r, "pageSize", 10)
	if err != nil || pageSize < 1 || pageSize > 100 {
		h.logger.Error("解析每页大小参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的每页大小参数，范围应为 1-100"))
		return
	}

	// 获取租户类型过滤参数
	tenantType := r.URL.Query().Get("type")
	if tenantType != "" && tenantType != model.TenantTypeSystem && tenantType != model.TenantTypeBusiness {
		h.logger.Error("无效的租户类型参数", logger.Fields{"type": tenantType})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的租户类型参数，应为 system 或 tenant"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取租户列表请求", logger.Fields{
		"pageNo":   pageNo,
		"pageSize": pageSize,
		"type":     tenantType,
	})

	// 4. 调用服务层获取租户列表
	var tenants []*model.Tenant
	var total int64

	if tenantType != "" {
		// 按类型过滤
		tenants, total, err = h.tenantService.List(ctx, pageNo, pageSize, tenantType)
	} else {
		// 获取所有租户
		tenants, total, err = h.tenantService.List(ctx, pageNo, pageSize)
	}

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


// UpdateTenantStatusRequest 更新租户状态请求（用于 Swagger）
// @name UpdateTenantStatusRequest
type UpdateTenantStatusRequest struct {
	// 租户状态（true: 启用, false: 禁用）
	Status bool `json:"status" example:"true"`
}

// HandleUpdateTenantStatus 处理启用/禁用租户
// @Summary 启用/禁用租户
// @Description 更新租户的启用/禁用状态。需要平台管理员权限（system_admin角色）。
// @Description
// @Description **权限要求：** system_admin
// @Description
// @Description **功能说明：**
// @Description - 启用租户：设置 status = true，该租户下的用户可以正常登录和访问系统
// @Description - 禁用租户：设置 status = false，该租户下的所有用户将无法登录和访问系统
// @Description
// @Description **影响范围：**
// @Description - 禁用租户后，该租户下所有用户的登录请求将被拒绝
// @Description - 禁用租户后，该租户下所有用户的 API 访问请求将被拒绝
// @Description - 启用租户后，该租户下用户恢复正常访问
// @Description
// @Description **注意事项：**
// @Description - 不允许禁用平台租户（type = "system"）
// @Description - 禁用操作会立即生效
// @Tags Platform Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID（UUID格式）" example("550e8400-e29b-41d4-a716-446655440000")
// @Param request body UpdateTenantStatusRequest true "更新租户状态请求"
// @Success 200 {object} model.TenantDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足（需要 system_admin 角色）"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /api/v1/platform/tenants/{id}/status [patch]
func (h *PlatformHandler) HandleUpdateTenantStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证平台管理员权限
	if !h.isSystemAdmin(ctx) {
		h.logger.Warn("非平台管理员尝试更新租户状态")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要平台管理员权限"))
		return
	}

	// 2. 从URL路径中提取租户ID
	tenantID := h.extractIDFromPath(r.URL.Path, "tenants")
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 3. 解析请求参数
	var req UpdateTenantStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新租户状态请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到更新租户状态请求", logger.Fields{
		"tenantId": tenantID,
		"status":   req.Status,
	})

	// 5. 调用服务层启用或禁用租户
	var err error
	if req.Status {
		err = h.tenantService.EnableTenant(ctx, tenantID)
	} else {
		err = h.tenantService.DisableTenant(ctx, tenantID)
	}

	if err != nil {
		h.logger.Error("更新租户状态失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"status":   req.Status,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 获取更新后的租户信息
	tenant, err := h.tenantService.Get(ctx, tenantID)
	if err != nil {
		h.logger.Error("获取更新后的租户信息失败", logger.Fields{
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
	h.logger.Info("更新租户状态成功", logger.Fields{
		"tenantId": tenantID,
		"status":   req.Status,
	})

	// 8. 返回成功响应
	resp := response.SuccessWithMessage("租户状态更新成功", tenant)
	h.writeJSONResponse(w, http.StatusOK, resp)
}


// HandleDeleteTenant 处理删除租户
// @Summary 删除租户
// @Description 软删除指定的业务租户。需要平台管理员权限（system_admin角色）。
// @Description
// @Description **权限要求：** system_admin
// @Description
// @Description **功能说明：**
// @Description - 执行软删除操作，设置 is_deleted = true
// @Description - 不会物理删除数据库记录，保留数据用于审计和恢复
// @Description - 删除后的租户不会出现在租户列表中
// @Description
// @Description **限制条件：**
// @Description - 不允许删除平台租户（type = "system"）
// @Description - 删除租户时会级联处理相关数据（根据数据库外键约束）
// @Description
// @Description **注意事项：**
// @Description - 删除操作不可逆（除非通过数据库直接恢复）
// @Description - 建议在删除前先禁用租户，观察一段时间后再删除
// @Description - 删除租户会影响该租户下的所有用户和数据
// @Tags Platform Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "租户ID（UUID格式）" example("550e8400-e29b-41d4-a716-446655440000")
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误（如尝试删除平台租户）"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足（需要 system_admin 角色）"
// @Failure 404 {object} model.ErrorResponse "租户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /api/v1/platform/tenants/{id} [delete]
func (h *PlatformHandler) HandleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证平台管理员权限
	if !h.isSystemAdmin(ctx) {
		h.logger.Warn("非平台管理员尝试删除租户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要平台管理员权限"))
		return
	}

	// 2. 从URL路径中提取租户ID
	tenantID := h.extractIDFromPath(r.URL.Path, "tenants")
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
