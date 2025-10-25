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
	TenantID    string                 `json:"tenantId" validate:"omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email       string                 `json:"email" validate:"required,email" example:"user@example.com"`
	Password    string                 `json:"password" validate:"required,min=8" example:"password123"`
	DisplayName string                 `json:"displayName" example:"张三"`
	Phone       string                 `json:"phone" example:"13800138000"`
	IsAdmin     bool                   `json:"isAdmin" example:"false"`
	Roles       []string               `json:"roles" example:"[\"user\"]"`
	Meta        map[string]interface{} `json:"meta" swaggertype:"object"`
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
// @Description 创建新用户，支持可选的租户ID参数
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以在任意租户下创建用户
// @Description - 租户管理员（tenant_admin）：只能在自己的租户下创建用户
// @Description
// @Description **tenantId 参数使用规则**：
// @Description - 租户管理员：
// @Description   - 如果不提供 tenantId，系统自动使用当前用户的租户ID
// @Description   - 如果提供 tenantId，必须与当前用户的租户ID匹配，否则返回 403 错误
// @Description - 平台管理员：
// @Description   - 必须提供 tenantId 参数
// @Description   - 可以指定任意有效的租户ID
// @Description
// @Description **参数说明**：
// @Description - tenantId: 租户ID（可选，UUID格式）
// @Description - email: 用户邮箱（必填，需符合邮箱格式）
// @Description - password: 用户密码（必填，最少8个字符）
// @Description - displayName: 显示名称（可选）
// @Description - phone: 手机号码（可选）
// @Description - isAdmin: 是否为管理员（可选，默认false）
// @Description - roles: 用户角色列表（可选）
// @Description - meta: 用户元数据（可选，JSON对象）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "创建用户请求"
// @Success 201 {object} model.UserDataResponse "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：租户管理员只能在当前租户下创建用户"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users [post]
func (h *UserHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req auth.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建用户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建用户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到创建用户请求", logger.Fields{
		"tenantId": req.TenantID,
		"email":    req.Email,
	})

	// 4. 调用服务层创建用户（租户ID处理和权限验证在服务层完成）
	user, err := h.userService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建用户失败", logger.Fields{
			"error":    err.Error(),
			"tenantId": req.TenantID,
			"email":    req.Email,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("创建用户成功", logger.Fields{
		"userId":   user.ID,
		"tenantId": user.TenantID,
		"email":    user.Email,
	})

	// 6. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessageContext(ctx, "创建用户成功", user)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleGet 处理获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看任意租户下的用户
// @Description - 租户管理员（tenant_admin）：只能查看自己租户下的用户
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标用户所属的租户
// @Description - 租户管理员尝试查看其他租户的用户时，将收到 403 权限不足错误
// @Description - 平台管理员可以查看任意租户的用户，无需额外验证
// @Description
// @Description **参数说明**：
// @Description - id: 用户ID（UUID格式）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.UserDataResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法访问其他租户的用户"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [get]
func (h *UserHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 记录请求日志
	h.logger.Info("收到获取用户详情请求", logger.Fields{
		"userId": userID,
	})

	// 3. 调用服务层获取用户（权限验证在服务层完成）
	user, err := h.userService.Get(ctx, userID)
	if err != nil {
		h.logger.Error("获取用户详情失败", logger.Fields{
			"error":  err,
			"userId": userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewNotFoundError("用户不存在"))
		}
		return
	}

	// 4. 记录响应日志
	h.logger.Info("获取用户详情成功", logger.Fields{
		"userId": userID,
	})

	// 5. 返回成功响应
	resp := response.SuccessWithContext(ctx, user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleUpdate 处理更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以更新任意租户下的用户
// @Description - 租户管理员（tenant_admin）：只能更新自己租户下的用户
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标用户所属的租户
// @Description - 租户管理员尝试更新其他租户的用户时，将收到 403 权限不足错误
// @Description - 平台管理员可以更新任意租户的用户，无需额外验证
// @Description
// @Description **参数说明**：
// @Description - id: 用户ID（UUID格式）
// @Description - email: 用户邮箱（可选，需符合邮箱格式）
// @Description - displayName: 显示名称（可选）
// @Description - phone: 手机号码（可选）
// @Description - isActive: 是否启用（可选）
// @Description - isAdmin: 是否为管理员（可选）
// @Description - roles: 用户角色列表（可选）
// @Description - meta: 用户元数据（可选，JSON对象）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body UpdateUserRequest true "更新用户请求"
// @Success 200 {object} model.UserDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法更新其他租户的用户"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [put]
func (h *UserHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req auth.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新用户请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新用户请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到更新用户请求", logger.Fields{
		"userId": userID,
	})

	// 5. 调用服务层更新用户（权限验证在服务层完成）
	user, err := h.userService.Update(ctx, userID, req)
	if err != nil {
		h.logger.Error("更新用户失败", logger.Fields{
			"error":  err,
			"userId": userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("更新用户成功", logger.Fields{
		"userId": userID,
	})

	// 7. 返回成功响应
	resp := response.SuccessWithMessageContext(ctx, "更新用户成功", user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleDelete 处理删除用户
// @Summary 删除用户
// @Description 软删除指定的用户
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以删除任意租户下的用户
// @Description - 租户管理员（tenant_admin）：只能删除自己租户下的用户
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标用户所属的租户
// @Description - 租户管理员尝试删除其他租户的用户时，将收到 403 权限不足错误
// @Description - 平台管理员可以删除任意租户的用户，无需额外验证
// @Description
// @Description **功能说明**：
// @Description - 执行软删除操作（设置 is_deleted=true）
// @Description - 删除后的用户数据仍保留在数据库中，但不再可见
// @Description
// @Description **参数说明**：
// @Description - id: 用户ID（UUID格式）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.AnyDataResponse "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法删除其他租户的用户"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id} [delete]
func (h *UserHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 记录请求日志
	h.logger.Info("收到删除用户请求", logger.Fields{
		"userId": userID,
	})

	// 3. 调用服务层删除用户（权限验证在服务层完成）
	err := h.userService.Delete(ctx, userID)
	if err != nil {
		h.logger.Error("删除用户失败", logger.Fields{
			"error":  err,
			"userId": userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 4. 记录响应日志
	h.logger.Info("删除用户成功", logger.Fields{
		"userId": userID,
	})

	// 5. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessageContext(ctx, "删除用户成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// UpdateUserStatusRequest 更新用户状态请求
// @name UpdateUserStatusRequest
type UpdateUserStatusRequest struct {
	IsActive *bool `json:"isActive" validate:"required" example:"true"`
}

// HandleUpdateStatus 处理更新用户状态
// @Summary 更新用户状态
// @Description 启用或禁用用户
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以更新任意租户下的用户状态
// @Description - 租户管理员（tenant_admin）：只能更新自己租户下的用户状态
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标用户所属的租户
// @Description - 租户管理员尝试更新其他租户的用户状态时，将收到 403 权限不足错误
// @Description - 平台管理员可以更新任意租户的用户状态，无需额外验证
// @Description
// @Description **功能说明**：
// @Description - 启用用户（isActive=true）：用户可以正常登录和访问系统
// @Description - 禁用用户（isActive=false）：用户将无法登录系统
// @Description
// @Description **参数说明**：
// @Description - id: 用户ID（UUID格式）
// @Description - isActive: 用户状态（true=启用，false=禁用）
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body UpdateUserStatusRequest true "更新用户状态请求"
// @Success 200 {object} model.UserDataResponse "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法更新其他租户的用户状态"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users/{id}/status [patch]
func (h *UserHandler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取用户ID
	userID := h.extractUserID(r.URL.Path)
	if userID == "" {
		h.logger.Warn("用户ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新用户状态请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新用户状态请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到更新用户状态请求", logger.Fields{
		"userId":   userID,
		"isActive": *req.IsActive,
	})

	// 5. 构建更新请求
	updateReq := auth.UpdateUserRequest{
		IsActive: req.IsActive,
	}

	// 6. 调用服务层更新用户（权限验证在服务层完成）
	user, err := h.userService.Update(ctx, userID, updateReq)
	if err != nil {
		h.logger.Error("更新用户状态失败", logger.Fields{
			"error":  err,
			"userId": userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("更新用户状态成功", logger.Fields{
		"userId":   userID,
		"isActive": user.IsActive,
	})

	// 8. 返回成功响应
	resp := response.SuccessWithMessageContext(ctx, "更新用户状态成功", user)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleList 处理获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页、租户过滤和搜索
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看所有租户的用户或指定租户的用户
// @Description - 租户管理员（tenant_admin）：只能查看自己租户下的用户
// @Description
// @Description **tenantId 查询参数使用规则**：
// @Description - 租户管理员：
// @Description   - tenantId 参数会被忽略
// @Description   - 始终只返回当前用户所属租户下的用户列表
// @Description - 平台管理员：
// @Description   - 如果提供 tenantId，返回指定租户下的用户列表
// @Description   - 如果不提供 tenantId，返回所有租户下的用户列表
// @Description
// @Description **search 查询参数说明**：
// @Description - 支持对 displayName、phone、email 字段进行模糊搜索
// @Description - 搜索不区分大小写
// @Description - 多个字段使用 OR 逻辑连接
// @Description
// @Description **参数说明**：
// @Description - pageNo: 页码（从1开始，默认1）
// @Description - pageSize: 每页大小（1-100，默认20）
// @Description - tenantId: 租户ID（可选，UUID格式，仅平台管理员可用）
// @Description - search: 搜索关键词（可选，支持模糊匹配 displayName、phone、email）
// @Description
// @Description **注意事项**：
// @Description - 租户管理员调用此接口时，tenantId 参数会被忽略
// @Description - 租户管理员始终只能看到自己租户下的用户
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int false "页码" minimum(1) default(1) example:1
// @Param pageSize query int false "每页大小" minimum(1) maximum(100) default(20) example:20
// @Param tenantId query string false "租户ID（仅平台管理员可用）" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param search query string false "搜索关键词（支持模糊匹配 displayName、phone、email）" example:"张三"
// @Success 200 {object} model.UserListResponse "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /users [get]
func (h *UserHandler) HandleList(w http.ResponseWriter, r *http.Request) {
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

	// 2. 获取可选的租户ID和搜索参数
	tenantID := r.URL.Query().Get("tenantId")
	search := r.URL.Query().Get("search")

	// 3. 记录请求日志
	h.logger.Info("收到获取用户列表请求", logger.Fields{
		"tenantId": tenantID,
		"search":   search,
		"pageNo":   pageNo,
		"pageSize": pageSize,
	})

	// 4. 调用服务层获取用户列表（权限验证和租户过滤在服务层完成）
	var users []*model.User
	var total int64
	var listErr error
	if tenantID != "" {
		users, total, listErr = h.userService.List(ctx, pageNo, pageSize, search, tenantID)
	} else {
		users, total, listErr = h.userService.List(ctx, pageNo, pageSize, search)
	}
	if listErr != nil {
		h.logger.Error("获取用户列表失败", logger.Fields{
			"error":    listErr,
			"tenantId": tenantID,
			"search":   search,
		})
		if appErr, ok := listErr.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(listErr))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取用户列表成功", logger.Fields{
		"tenantId": tenantID,
		"search":   search,
		"count":    len(users),
		"total":    total,
	})

	// 6. 返回分页响应
	h.writePaginationResponseWithContext(w, ctx, users, pageNo, pageSize, int(total))
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

// writeJSONResponse 写入 JSON 响应
func (h *UserHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *UserHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
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
func (h *UserHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
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
func (h *UserHandler) writePaginationResponse(w http.ResponseWriter, data []*model.User, pageNo, pageSize, total int) {
	resp := response.Pagination(data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writePaginationResponseWithContext 写入分页响应（带 Context）
func (h *UserHandler) writePaginationResponseWithContext(w http.ResponseWriter, ctx context.Context, data []*model.User, pageNo, pageSize, total int) {
	resp := response.PaginationWithContext(ctx, data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}


