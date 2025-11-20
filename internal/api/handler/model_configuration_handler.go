package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"

	"github.com/google/uuid"
)

// ModelConfigurationHandler 模型配置管理处理器
// 提供模型配置的创建、查询、更新、删除、验证等功能
type ModelConfigurationHandler struct {
	modelConfigService service.ModelConfigurationService
	logger             logger.Logger
	validator          *validator.Validator
}

// NewModelConfigurationHandler 创建模型配置处理器实例
// 参数：
//   - modelConfigService: 模型配置服务接口
//   - log: 日志记录器
// 返回：
//   - *ModelConfigurationHandler: 模型配置处理器实例
func NewModelConfigurationHandler(modelConfigService service.ModelConfigurationService, log logger.Logger) *ModelConfigurationHandler {
	return &ModelConfigurationHandler{
		modelConfigService: modelConfigService,
		logger:             log,
		validator:          validator.New(),
	}
}

// HandleCreate 处理创建模型配置
// @Summary 创建模型配置
// @Description 创建新的模型配置，支持可选的租户ID参数
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以在任意租户下创建模型配置
// @Description - 租户管理员（tenant_admin）：只能在自己的租户下创建模型配置
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
// @Description - name: 配置名称（必填）
// @Description - model: 模型标识（必填，如：gpt-4、claude-3-opus）
// @Description - modelProvider: 模型提供商（必填，可选值：openai、anthropic、googlegenai、azureopenai、bianlian、custom_openai）
// @Description - baseUrl: API基础URL（可选，用于自定义端点）
// @Description - apiKey: API密钥（必填，将被加密存储）
// @Description - queryParams: 查询参数（可选，JSON格式）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateModelConfigurationRequest true "创建模型配置请求"
// @Success 201 {object} model.ResponseData[model.ModelConfiguration] "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：租户管理员只能在当前租户下创建模型配置"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations [post]
func (h *ModelConfigurationHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req model.CreateModelConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建模型配置请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建模型配置请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到创建模型配置请求", logger.Fields{
		"tenantId":      req.TenantID,
		"name":          req.Name,
		"modelProvider": req.ModelProvider,
	})

	// 4. 调用服务层创建模型配置（租户ID处理和权限验证在服务层完成）
	config, err := h.modelConfigService.Create(ctx, req)
	if err != nil {
		h.logger.Error("创建模型配置失败", logger.Fields{
			"error":         err.Error(),
			"tenantId":      req.TenantID,
			"name":          req.Name,
			"modelProvider": req.ModelProvider,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("创建模型配置成功", logger.Fields{
		"configId":      config.ID,
		"tenantId":      config.TenantID,
		"name":          config.Name,
		"modelProvider": config.ModelProvider,
	})

	// 6. 返回成功响应（HTTP 201 Created）
	resp := response.SuccessWithMessageContext(ctx, "创建模型配置成功", config)
	h.writeJSONResponse(w, http.StatusCreated, resp)
}

// HandleGet 处理获取模型配置详情
// @Summary 获取模型配置详情
// @Description 根据配置ID获取模型配置详细信息
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看任意租户下的模型配置
// @Description - 租户管理员（tenant_admin）：只能查看自己租户下的模型配置
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标配置所属的租户
// @Description - 租户管理员尝试查看其他租户的配置时，将收到 403 权限不足错误
// @Description - 平台管理员可以查看任意租户的配置，无需额外验证
// @Description
// @Description **参数说明**：
// @Description - id: 配置ID（UUID格式）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "配置ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.ResponseData[model.ModelConfiguration] "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法访问其他租户的模型配置"
// @Failure 404 {object} model.ErrorResponse "模型配置不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/{id} [get]
func (h *ModelConfigurationHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取配置ID
	configIDStr := h.extractConfigID(r.URL.Path)
	if configIDStr == "" {
		h.logger.Warn("配置ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID不能为空"))
		return
	}

	// 2. 解析UUID
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		h.logger.Warn("配置ID格式无效", logger.Fields{"configId": configIDStr})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取模型配置详情请求", logger.Fields{
		"configId": configID,
	})

	// 4. 调用服务层获取配置（权限验证在服务层完成）
	config, err := h.modelConfigService.Get(ctx, configID)
	if err != nil {
		h.logger.Error("获取模型配置详情失败", logger.Fields{
			"error":    err,
			"configId": configID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewNotFoundError("模型配置不存在"))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取模型配置详情成功", logger.Fields{
		"configId": configID,
	})

	// 6. 返回成功响应
	resp := response.SuccessWithContext(ctx, config)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleList 处理获取模型配置列表
// @Summary 获取模型配置列表
// @Description 获取模型配置列表，支持分页和租户过滤
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以查看所有租户的配置或指定租户的配置
// @Description - 租户管理员（tenant_admin）：只能查看自己租户下的配置
// @Description
// @Description **tenantId 查询参数使用规则**：
// @Description - 租户管理员：
// @Description   - tenantId 参数会被忽略
// @Description   - 始终只返回当前用户所属租户下的配置列表
// @Description - 平台管理员：
// @Description   - 如果提供 tenantId，返回指定租户下的配置列表
// @Description   - 如果不提供 tenantId，返回所有租户下的配置列表
// @Description
// @Description **参数说明**：
// @Description - pageNo: 页码（从1开始，默认1）
// @Description - pageSize: 每页大小（1-100，默认20）
// @Description - tenantId: 租户ID（可选，UUID格式，仅平台管理员可用）
// @Description
// @Description **注意事项**：
// @Description - 租户管理员调用此接口时，tenantId 参数会被忽略
// @Description - 租户管理员始终只能看到自己租户下的配置
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pageNo query int false "页码" minimum(1) default(1) example:1
// @Param pageSize query int false "每页大小" minimum(1) maximum(100) default(20) example:20
// @Param tenantId query string false "租户ID（仅平台管理员可用）" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.ResponsePaginationData[[]model.ModelConfiguration] "获取成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations [get]
func (h *ModelConfigurationHandler) HandleList(w http.ResponseWriter, r *http.Request) {
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

	// 2. 获取可选的租户ID
	tenantIDStr := r.URL.Query().Get("tenantId")
	var tenantID *uuid.UUID
	if tenantIDStr != "" {
		parsedID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			h.logger.Error("租户ID格式无效", logger.Fields{"tenantId": tenantIDStr})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
			return
		}
		tenantID = &parsedID
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取模型配置列表请求", logger.Fields{
		"tenantId": tenantID,
		"pageNo":   pageNo,
		"pageSize": pageSize,
	})

	// 4. 调用服务层获取配置列表（权限验证和租户过滤在服务层完成）
	configs, total, err := h.modelConfigService.List(ctx, tenantID, pageNo, pageSize)
	if err != nil {
		h.logger.Error("获取模型配置列表失败", logger.Fields{
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

	// 5. 记录响应日志
	h.logger.Info("获取模型配置列表成功", logger.Fields{
		"tenantId": tenantID,
		"count":    len(configs),
		"total":    total,
	})

	// 6. 返回分页响应
	h.writePaginationResponseWithContext(w, ctx, configs, pageNo, pageSize, int(total))
}

// HandleUpdate 处理更新模型配置
// @Summary 更新模型配置
// @Description 更新模型配置信息
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以更新任意租户下的模型配置
// @Description - 租户管理员（tenant_admin）：只能更新自己租户下的模型配置
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标配置所属的租户
// @Description - 租户管理员尝试更新其他租户的配置时，将收到 403 权限不足错误
// @Description - 平台管理员可以更新任意租户的配置，无需额外验证
// @Description
// @Description **参数说明**：
// @Description - id: 配置ID（UUID格式）
// @Description - name: 配置名称（可选）
// @Description - model: 模型标识（可选）
// @Description - baseUrl: API基础URL（可选）
// @Description - apiKey: API密钥（可选，将被加密存储）
// @Description - queryParams: 查询参数（可选，JSON格式）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "配置ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body model.UpdateModelConfigurationRequest true "更新模型配置请求"
// @Success 200 {object} model.ResponseData[model.ModelConfiguration] "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法更新其他租户的模型配置"
// @Failure 404 {object} model.ErrorResponse "模型配置不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/{id} [put]
func (h *ModelConfigurationHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取配置ID
	configIDStr := h.extractConfigID(r.URL.Path)
	if configIDStr == "" {
		h.logger.Warn("配置ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID不能为空"))
		return
	}

	// 2. 解析UUID
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		h.logger.Warn("配置ID格式无效", logger.Fields{"configId": configIDStr})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID格式无效"))
		return
	}

	// 3. 解析请求参数
	var req model.UpdateModelConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新模型配置请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新模型配置请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 5. 记录请求日志
	h.logger.Info("收到更新模型配置请求", logger.Fields{
		"configId": configID,
	})

	// 6. 调用服务层更新配置（权限验证在服务层完成）
	config, err := h.modelConfigService.Update(ctx, configID, req)
	if err != nil {
		h.logger.Error("更新模型配置失败", logger.Fields{
			"error":    err,
			"configId": configID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("更新模型配置成功", logger.Fields{
		"configId": configID,
	})

	// 8. 返回成功响应
	resp := response.SuccessWithMessageContext(ctx, "更新模型配置成功", config)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleUpdateStatus 处理更新模型配置状态
// @Summary 更新模型配置状态
// @Description 启用或禁用模型配置
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以更新任意租户下的模型配置状态
// @Description - 租户管理员（tenant_admin）：只能更新自己租户下的模型配置状态
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标配置所属的租户
// @Description - 租户管理员尝试更新其他租户的配置状态时，将收到 403 权限不足错误
// @Description - 平台管理员可以更新任意租户的配置状态，无需额外验证
// @Description
// @Description **功能说明**：
// @Description - 启用配置（status=enabled）：配置可以被使用
// @Description - 禁用配置（status=disabled）：配置将不可用
// @Description
// @Description **参数说明**：
// @Description - id: 配置ID（UUID格式）
// @Description - status: 配置状态（enabled=启用，disabled=禁用）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "配置ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Param request body model.UpdateStatusRequest true "更新配置状态请求"
// @Success 200 {object} model.ResponseData[any] "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法更新其他租户的模型配置状态"
// @Failure 404 {object} model.ErrorResponse "模型配置不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/{id}/status [patch]
func (h *ModelConfigurationHandler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取配置ID
	configIDStr := h.extractConfigID(r.URL.Path)
	if configIDStr == "" {
		h.logger.Warn("配置ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID不能为空"))
		return
	}

	// 2. 解析UUID
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		h.logger.Warn("配置ID格式无效", logger.Fields{"configId": configIDStr})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID格式无效"))
		return
	}

	// 3. 解析请求参数
	var req model.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新配置状态请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 4. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新配置状态请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 5. 转换状态值
	enabled := req.Status == "enabled"

	// 6. 记录请求日志
	h.logger.Info("收到更新模型配置状态请求", logger.Fields{
		"configId": configID,
		"status":   req.Status,
	})

	// 7. 调用服务层更新状态（权限验证在服务层完成）
	err = h.modelConfigService.UpdateStatus(ctx, configID, enabled)
	if err != nil {
		h.logger.Error("更新模型配置状态失败", logger.Fields{
			"error":    err,
			"configId": configID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 8. 记录响应日志
	h.logger.Info("更新模型配置状态成功", logger.Fields{
		"configId": configID,
		"status":   req.Status,
	})

	// 9. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessageContext(ctx, "更新模型配置状态成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleDelete 处理删除模型配置
// @Summary 删除模型配置
// @Description 软删除指定的模型配置
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以删除任意租户下的模型配置
// @Description - 租户管理员（tenant_admin）：只能删除自己租户下的模型配置
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标配置所属的租户
// @Description - 租户管理员尝试删除其他租户的配置时，将收到 403 权限不足错误
// @Description - 平台管理员可以删除任意租户的配置，无需额外验证
// @Description
// @Description **功能说明**：
// @Description - 执行软删除操作（设置 is_deleted=true）
// @Description - 删除后的配置数据仍保留在数据库中，但不再可见
// @Description
// @Description **参数说明**：
// @Description - id: 配置ID（UUID格式）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "配置ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.ResponseData[any] "删除成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法删除其他租户的模型配置"
// @Failure 404 {object} model.ErrorResponse "模型配置不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/{id} [delete]
func (h *ModelConfigurationHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取配置ID
	configIDStr := h.extractConfigID(r.URL.Path)
	if configIDStr == "" {
		h.logger.Warn("配置ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID不能为空"))
		return
	}

	// 2. 解析UUID
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		h.logger.Warn("配置ID格式无效", logger.Fields{"configId": configIDStr})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到删除模型配置请求", logger.Fields{
		"configId": configID,
	})

	// 4. 调用服务层删除配置（权限验证在服务层完成）
	err = h.modelConfigService.Delete(ctx, configID)
	if err != nil {
		h.logger.Error("删除模型配置失败", logger.Fields{
			"error":    err,
			"configId": configID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("删除模型配置成功", logger.Fields{
		"configId": configID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	resp := response.SuccessWithMessageContext(ctx, "删除模型配置成功", &emptyData)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleValidate 处理验证模型配置
// @Summary 验证模型配置
// @Description 验证模型配置是否可以正确连接到提供商
// @Description
// @Description **权限要求**：
// @Description - 平台管理员（system_admin）：可以验证任意租户下的模型配置
// @Description - 租户管理员（tenant_admin）：只能验证自己租户下的模型配置
// @Description
// @Description **访问权限验证**：
// @Description - 系统会自动查询目标配置所属的租户
// @Description - 租户管理员尝试验证其他租户的配置时，将收到 403 权限不足错误
// @Description - 平台管理员可以验证任意租户的配置，无需额外验证
// @Description
// @Description **功能说明**：
// @Description - 使用配置的参数尝试连接到模型提供商
// @Description - 验证请求设置30秒超时
// @Description - 返回验证结果，包括成功/失败状态和详细信息
// @Description
// @Description **参数说明**：
// @Description - id: 配置ID（UUID格式）
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "配置ID" example:"550e8400-e29b-41d4-a716-446655440000"
// @Success 200 {object} model.ResponseData[model.ValidationResult] "验证完成"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足：无法验证其他租户的模型配置"
// @Failure 404 {object} model.ErrorResponse "模型配置不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/{id}/validate [post]
func (h *ModelConfigurationHandler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取配置ID
	configIDStr := h.extractConfigID(r.URL.Path)
	if configIDStr == "" {
		h.logger.Warn("配置ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID不能为空"))
		return
	}

	// 2. 解析UUID
	configID, err := uuid.Parse(configIDStr)
	if err != nil {
		h.logger.Warn("配置ID格式无效", logger.Fields{"configId": configIDStr})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("配置ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到验证模型配置请求", logger.Fields{
		"configId": configID,
	})

	// 4. 调用服务层验证配置（权限验证在服务层完成）
	result, err := h.modelConfigService.Validate(ctx, configID)
	if err != nil {
		h.logger.Error("验证模型配置失败", logger.Fields{
			"error":    err,
			"configId": configID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 检查验证结果，如果验证失败则返回错误
	if !result.Valid {
		h.logger.Warn("模型配置验证失败", logger.Fields{
			"configId": configID,
			"message":  result.Message,
			"details":  result.Details,
		})
		// 返回400错误，包含验证失败的详细信息
		errMsg := result.Message
		if result.Details != "" {
			errMsg = fmt.Sprintf("%s: %s", result.Message, result.Details)
		}
		h.writeErrorResponse(w, r, errors.NewBadRequestError(errMsg))
		return
	}

	// 6. 记录响应日志
	h.logger.Info("验证模型配置成功", logger.Fields{
		"configId": configID,
	})

	// 7. 返回验证成功结果
	resp := response.SuccessWithMessageContext(ctx, "验证成功", result)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// HandleListAvailable 处理获取可用模型配置列表
// @Summary 获取可用模型配置列表
// @Description 获取当前租户下所有可用的模型配置（已启用且未删除）
// @Description
// @Description **权限要求**：
// @Description - 所有已认证用户都可以调用此接口
// @Description - 自动返回当前用户所属租户下的可用配置
// @Description
// @Description **功能说明**：
// @Description - 仅返回已启用（is_enabled=true）且未删除（is_deleted=false）的配置
// @Description - 返回的配置不包含敏感信息（API密钥、查询参数等）
// @Description - 用于前端展示可选的模型列表
// @Description
// @Description **返回字段**：
// @Description - id: 配置ID
// @Description - name: 配置名称
// @Description - model: 模型标识
// @Description - modelProvider: 模型提供商
// @Tags Model Configuration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.ResponseData[[]model.ModelConfiguration] "获取成功"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /model-configurations/available [get]
func (h *ModelConfigurationHandler) HandleListAvailable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 记录请求日志
	h.logger.Info("收到获取可用模型配置列表请求")

	// 2. 调用服务层获取可用配置列表
	configs, err := h.modelConfigService.ListAvailable(ctx)
	if err != nil {
		h.logger.Error("获取可用模型配置列表失败", logger.Fields{
			"error": err,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 3. 记录响应日志
	h.logger.Info("获取可用模型配置列表成功", logger.Fields{
		"count": len(configs),
	})

	// 4. 返回成功响应
	resp := response.SuccessWithContext(ctx, &configs)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// extractConfigID 从URL路径中提取配置ID
// 路径格式: /api/v1/model-configurations/{id} 或 /api/v1/model-configurations/{id}/validate
func (h *ModelConfigurationHandler) extractConfigID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "model-configurations" 后的部分
	for i, part := range parts {
		if part == "model-configurations" && i+1 < len(parts) {
			// 返回下一个部分（可能是ID或"available"）
			nextPart := parts[i+1]
			// 如果是"available"，返回空字符串
			if nextPart == "available" {
				return ""
			}
			return nextPart
		}
	}

	return ""
}

// parseIntQuery 解析整数查询参数
func (h *ModelConfigurationHandler) parseIntQuery(r *http.Request, key string, defaultValue int) (int, error) {
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
func (h *ModelConfigurationHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}

// writeErrorResponse 写入错误响应
func (h *ModelConfigurationHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
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
func (h *ModelConfigurationHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
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

// writePaginationResponseWithContext 写入分页响应（带 Context）
func (h *ModelConfigurationHandler) writePaginationResponseWithContext(w http.ResponseWriter, ctx context.Context, data []*model.ModelConfiguration, pageNo, pageSize, total int) {
	resp := response.PaginationWithContext(ctx, data, pageNo, pageSize, total)
	h.writeJSONResponse(w, http.StatusOK, resp)
}
