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

// UserHandler 用户管理处理器
// 提供用户的创建、查询、更新、删除等功能
type UserHandler struct {
	userService auth.UserService
	logger      logger.Logger
	validator   *validator.Validator
}

// NewUserHandler 创建用户处理器实例
// 参数：
//   - userService: 用户服务接口
//   - log: 日志记录器
// 返回：
//   - *UserHandler: 用户处理器实例
func NewUserHandler(userService auth.UserService, log logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      log,
		validator:   validator.New(),
	}
}

// CreateUserRequest 创建用户请求（用于 Swagger）
// @name CreateUserRequest
type CreateUserRequest struct {
	TenantID    string                 `json:"tenantId" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email       string                 `json:"email" validate:"required,email" example:"user@example.com"`
	Password    string                 `json:"password" validate:"required,min=8" example:"password123"`
	DisplayName string                 `json:"displayName" example:"张三"`
	Phone       string                 `json:"phone" example:"13800138000"`
	IsAdmin     bool                   `json:"isAdmin" example:"false"`
	Roles       []string               `json:"roles" example:"[\"user\"]"`
	Meta        map[string]interface{} `json:"meta" swaggertype:"object"`
	CreatedBy   *string                `json:"createdBy" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateUserRequest 更新用户请求（用于 Swagger）
// @name UpdateUserRequest
type UpdateUserRequest struct {
	Email       *string                `json:"email" validate:"omitempty,email" example:"newemail@example.com"`
	DisplayName *string                `json:"displayName" example:"李四"`
	Phone       *string                `json:"phone" example:"13900139000"`
	IsActive    *bool                  `json:"isActive" example:"true"`
	IsAdmin     *bool                  `json:"isAdmin" example:"false"`
	Roles       []string               `json:"roles" example:"[\"user\",\"moderator\"]"`
	Meta        map[string]interface{} `json:"meta" swaggertype:"object"`
}

// HandleCreate 处理创建用户
// @Summary 创建用户
// @Description 在指定租户下创建新用户（需要租户管理员权限）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "创建用户请求"
// @Success 201 {object} model.UserDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users [post]
func (h *UserHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证租户管理员权限
	if !h.isTenantAdmin(ctx) {
		h.logger.Warn("非租户管理员尝试创建用户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要租户管理员权限"))
		return
	}

	// 2. 解析请求参数
	var req auth.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建用户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证租户ID是否与当前用户的租户ID匹配
	currentTenantID, _ := ctx.Value("tenant_id").(string)
	if req.TenantID != currentTenantID {
		h.logger.Warn("尝试在其他租户下创建用户", logger.Fields{
			"requestTenantId": req.TenantID,
			"currentTenantId": currentTenantID,
		})
		h.writeErrorResponse(w, errors.NewForbiddenError("只能在当前租户下创建用户"))
		return
	}

	// 4. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建用户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 5. 记录请求日志
	h.logger.Info("收到创建用户请求", logger.Fields{
		"tenantId": req.TenantID,
		"email":    req.Email,
	})

	// 6. 调用服务层创建用户
	user, err := h.userService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建用户失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"email":    req.Email,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("创建用户成功", logger.Fields{
		"userId":   user.ID,
		"tenantId": user.TenantID,
		"email":    user.Email,
	})

	// 8. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessage("创建用户成功", user)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleGet 处理获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息（需要租户管理员权限）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Success 200 {object} model.UserDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [get]
func (h *UserHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证租户管理员权限
	if !h.isTenantAdmin(ctx) {
		h.logger.Warn("非租户管理员尝试获取用户详情")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要租户管理员权限"))
		return
	}

	// 2. 从上下文获取租户ID
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 3. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到获取用户详情请求", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 5. 调用服务层获取用户
	user, err := h.userService.Get(ctx, tenantID, userID)
	if err != nil {
		h.logger.Error("获取用户详情失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"userId":   userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewNotFoundError("用户不存在"))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("获取用户详情成功", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 7. 返回成功响应
	resp := response.Success(user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleUpdate 处理更新用户
// @Summary 更新用户
// @Description 更新用户信息（需要租户管理员权限）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Param request body UpdateUserRequest true "更新用户请求"
// @Success 200 {object} model.UserDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [put]
func (h *UserHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证租户管理员权限
	if !h.isTenantAdmin(ctx) {
		h.logger.Warn("非租户管理员尝试更新用户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要租户管理员权限"))
		return
	}

	// 2. 从上下文获取租户ID
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 3. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 4. 解析请求参数
	var req auth.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新用户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 5. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新用户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 6. 记录请求日志
	h.logger.Info("收到更新用户请求", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 7. 调用服务层更新用户
	user, err := h.userService.Update(ctx, tenantID, userID, req)
	if err != nil {
		h.logger.Error("更新用户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"userId":   userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 8. 记录响应日志
	h.logger.Info("更新用户成功", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 9. 返回成功响应
	resp := response.SuccessWithMessage("更新用户成功", user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleDelete 处理删除用户
// @Summary 删除用户
// @Description 软删除指定的用户（需要租户管理员权限）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [delete]
func (h *UserHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证租户管理员权限
	if !h.isTenantAdmin(ctx) {
		h.logger.Warn("非租户管理员尝试删除用户")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要租户管理员权限"))
		return
	}

	// 2. 从上下文获取租户ID
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 3. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到删除用户请求", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 5. 调用服务层删除用户
	err := h.userService.Delete(ctx, tenantID, userID)
	if err != nil {
		h.logger.Error("删除用户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"userId":   userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("删除用户成功", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 7. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("删除用户成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleList 处理获取用户列表
// @Summary 获取用户列表
// @Description 获取当前租户下的用户列表，支持分页（需要租户管理员权限）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(20)
// @Success 200 {object} model.UserListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users [get]
func (h *UserHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 验证租户管理员权限
	if !h.isTenantAdmin(ctx) {
		h.logger.Warn("非租户管理员尝试获取用户列表")
		h.writeErrorResponse(w, errors.NewForbiddenError("需要租户管理员权限"))
		return
	}

	// 2. 从上下文获取租户ID
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		h.logger.Warn("未找到租户ID")
		h.writeErrorResponse(w, errors.NewUnauthorizedError("未认证"))
		return
	}

	// 3. 解析查询参数
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

	// 4. 记录请求日志
	h.logger.Info("收到获取用户列表请求", logger.Fields{
		"tenantId": tenantID,
		"pageNo":   pageNo,
		"pageSize": pageSize,
	})

	// 5. 调用服务层获取用户列表
	users, total, err := h.userService.List(ctx, tenantID, pageNo, pageSize)
	if err != nil {
		h.logger.Error("获取用户列表失败", logger.Fields{
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

	// 6. 记录响应日志
	h.logger.Info("获取用户列表成功", logger.Fields{
		"tenantId": tenantID,
		"count":    len(users),
		"total":    total,
	})

	// 7. 返回分页响应
	h.writePaginationResponse(w, users, pageNo, pageSize, int(total))
}

// extractUserID 从URL路径中提取用户ID
// 路径格式: /api/v1/users/{id}
func (h *UserHandler) extractUserID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "users" 后的部分
	for i, part := range parts {
		if part == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// parseIntQuery 解析整数查询参数
func (h *UserHandler) parseIntQuery(r *http.Request, key string, defaultValue int) (int, error) {
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

// isTenantAdmin 检查当前用户是否为租户管理员
func (h *UserHandler) isTenantAdmin(ctx context.Context) bool {
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
func (h *UserHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *UserHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
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
func (h *UserHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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
func (h *UserHandler) writePaginationResponse(w http.ResponseWriter, data []*model.User, pageNo, pageSize, total int) {
	resp := response.Pagination(data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// ========== 租户管理 API 方法 ==========

// HandleTenantCreateUser 处理在指定租户下创建用户
// @Summary 在租户下创建用户
// @Description 在指定租户下创建新用户（需要租户管理员或平台管理员权限）
// @Tags Tenant User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenantId path string true "租户ID"
// @Param request body CreateUserRequest true "创建用户请求"
// @Success 201 {object} model.UserDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{tenantId}/users [post]
func (h *UserHandler) HandleTenantCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从路径参数中提取租户ID
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 验证权限：租户管理员或平台管理员
	if !h.canManageTenant(ctx, tenantID) {
		h.logger.Warn("权限不足：无法在该租户下创建用户", logger.Fields{
			"tenantId": tenantID,
		})
		h.writeErrorResponse(w, errors.NewForbiddenError("权限不足：无法在该租户下创建用户"))
		return
	}

	// 3. 解析请求参数
	var req auth.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建用户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 设置租户ID（使用路径参数中的租户ID）
	req.TenantID = tenantID

	// 5. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建用户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 6. 记录请求日志
	h.logger.Info("收到在租户下创建用户请求", logger.Fields{
		"tenantId": req.TenantID,
		"email":    req.Email,
	})

	// 7. 调用服务层创建用户
	user, err := h.userService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建用户失败", logger.Fields{
			"error":    err,
			"tenantId": req.TenantID,
			"email":    req.Email,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 8. 记录响应日志
	h.logger.Info("创建用户成功", logger.Fields{
		"userId":   user.ID,
		"tenantId": user.TenantID,
		"email":    user.Email,
	})

	// 9. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessage("创建用户成功", user)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleTenantListUsers 处理获取指定租户下的用户列表
// @Summary 获取租户用户列表
// @Description 获取指定租户下的用户列表，支持分页（需要租户管理员或平台管理员权限）
// @Tags Tenant User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenantId path string true "租户ID"
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(20)
// @Success 200 {object} model.UserListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{tenantId}/users [get]
func (h *UserHandler) HandleTenantListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从路径参数中提取租户ID
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	// 2. 验证权限：租户管理员或平台管理员
	if !h.canManageTenant(ctx, tenantID) {
		h.logger.Warn("权限不足：无法查看该租户的用户列表", logger.Fields{
			"tenantId": tenantID,
		})
		h.writeErrorResponse(w, errors.NewForbiddenError("权限不足：无法查看该租户的用户列表"))
		return
	}

	// 3. 解析查询参数
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

	// 4. 记录请求日志
	h.logger.Info("收到获取租户用户列表请求", logger.Fields{
		"tenantId": tenantID,
		"pageNo":   pageNo,
		"pageSize": pageSize,
	})

	// 5. 调用服务层获取用户列表
	users, total, err := h.userService.List(ctx, tenantID, pageNo, pageSize)
	if err != nil {
		h.logger.Error("获取用户列表失败", logger.Fields{
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

	// 6. 记录响应日志
	h.logger.Info("获取用户列表成功", logger.Fields{
		"tenantId": tenantID,
		"count":    len(users),
		"total":    total,
	})

	// 7. 返回分页响应
	h.writePaginationResponse(w, users, pageNo, pageSize, int(total))
}

// UpdateUserStatusRequest 更新用户状态请求
// @name UpdateUserStatusRequest
type UpdateUserStatusRequest struct {
	IsActive bool `json:"isActive" validate:"required" example:"true"`
}

// HandleTenantUpdateUserStatus 处理更新指定租户下用户的状态
// @Summary 更新用户状态
// @Description 启用或禁用指定租户下的用户（需要租户管理员或平台管理员权限）
// @Tags Tenant User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenantId path string true "租户ID"
// @Param userId path string true "用户ID"
// @Param request body UpdateUserStatusRequest true "更新用户状态请求"
// @Success 200 {object} model.UserDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{tenantId}/users/{userId}/status [patch]
func (h *UserHandler) HandleTenantUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从路径参数中提取租户ID和用户ID
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	userID := r.PathValue("userId")
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 验证权限：租户管理员或平台管理员
	if !h.canManageTenant(ctx, tenantID) {
		h.logger.Warn("权限不足：无法修改该租户下的用户", logger.Fields{
			"tenantId": tenantID,
			"userId":   userID,
		})
		h.writeErrorResponse(w, errors.NewForbiddenError("权限不足：无法修改该租户下的用户"))
		return
	}

	// 3. 解析请求参数
	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新用户状态请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新用户状态请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 5. 记录请求日志
	h.logger.Info("收到更新用户状态请求", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
		"isActive": req.IsActive,
	})

	// 6. 构建更新请求
	updateReq := auth.UpdateUserRequest{
		IsActive: &req.IsActive,
	}

	// 7. 调用服务层更新用户
	user, err := h.userService.Update(ctx, tenantID, userID, updateReq)
	if err != nil {
		h.logger.Error("更新用户状态失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"userId":   userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 8. 记录响应日志
	h.logger.Info("更新用户状态成功", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
		"isActive": user.IsActive,
	})

	// 9. 返回成功响应
	resp := response.SuccessWithMessage("更新用户状态成功", user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleTenantDeleteUser 处理删除指定租户下的用户
// @Summary 删除用户
// @Description 软删除指定租户下的用户（需要租户管理员或平台管理员权限）
// @Tags Tenant User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenantId path string true "租户ID"
// @Param userId path string true "用户ID"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /tenants/{tenantId}/users/{userId} [delete]
func (h *UserHandler) HandleTenantDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从路径参数中提取租户ID和用户ID
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		h.logger.Warn("租户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("租户ID不能为空"))
		return
	}

	userID := r.PathValue("userId")
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 验证权限：租户管理员或平台管理员
	if !h.canManageTenant(ctx, tenantID) {
		h.logger.Warn("权限不足：无法删除该租户下的用户", logger.Fields{
			"tenantId": tenantID,
			"userId":   userID,
		})
		h.writeErrorResponse(w, errors.NewForbiddenError("权限不足：无法删除该租户下的用户"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到删除用户请求", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 4. 调用服务层删除用户
	err := h.userService.Delete(ctx, tenantID, userID)
	if err != nil {
		h.logger.Error("删除用户失败", logger.Fields{
			"error":    err,
			"tenantId": tenantID,
			"userId":   userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("删除用户成功", logger.Fields{
		"tenantId": tenantID,
		"userId":   userID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessage("删除用户成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// canManageTenant 检查当前用户是否有权限管理指定租户
// 平台管理员可以管理所有租户，租户管理员只能管理自己的租户
func (h *UserHandler) canManageTenant(ctx context.Context, targetTenantID string) bool {
	// 从上下文获取角色信息
	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return false
	}

	// 检查是否为平台管理员
	for _, role := range roles {
		if role == model.RoleSystemAdmin {
			return true
		}
	}

	// 检查是否为租户管理员，并且租户ID匹配
	currentTenantID, ok := ctx.Value("tenant_id").(string)
	if !ok {
		return false
	}

	// 租户管理员只能管理自己的租户
	for _, role := range roles {
		if role == model.RoleTenantAdmin && currentTenantID == targetTenantID {
			return true
		}
	}

	return false
}
