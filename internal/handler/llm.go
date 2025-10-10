package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"your-project/internal/service"
)

// LLMHandler LLM处理器
type LLMHandler struct {
	llmService service.LLMService
}

// NewLLMHandler 创建LLM处理器
func NewLLMHandler(llmService service.LLMService) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
	}
}

// CreateProvider 创建提供商
// @Summary 创建LLM提供商
// @Description 创建新的LLM提供商配置
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body service.CreateProviderRequest true "创建提供商请求"
// @Success 201 {object} model.LLMProvider
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers [post]
func (h *LLMHandler) CreateProvider(c *gin.Context) {
	var req service.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	provider, err := h.llmService.CreateProvider(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "CREATE_PROVIDER_FAILED",
			Message: "创建提供商失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// GetProvider 获取提供商
// @Summary 获取LLM提供商
// @Description 根据ID获取LLM提供商详情
// @Tags LLM
// @Produce json
// @Param id path string true "提供商ID"
// @Success 200 {object} model.LLMProvider
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id} [get]
func (h *LLMHandler) GetProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	provider, err := h.llmService.GetProvider(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "PROVIDER_NOT_FOUND",
			Message: "提供商未找到: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// ListProviders 列出提供商
// @Summary 列出LLM提供商
// @Description 获取LLM提供商列表
// @Tags LLM
// @Produce json
// @Param is_active query boolean false "是否只返回激活的提供商"
// @Success 200 {array} model.LLMProvider
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers [get]
func (h *LLMHandler) ListProviders(c *gin.Context) {
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if active, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &active
		}
	}

	providers, err := h.llmService.ListProviders(c.Request.Context(), isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "LIST_PROVIDERS_FAILED",
			Message: "获取提供商列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, providers)
}

// UpdateProvider 更新提供商
// @Summary 更新LLM提供商
// @Description 更新LLM提供商配置
// @Tags LLM
// @Accept json
// @Produce json
// @Param id path string true "提供商ID"
// @Param request body service.UpdateProviderRequest true "更新提供商请求"
// @Success 200 {object} model.LLMProvider
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id} [put]
func (h *LLMHandler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	var req service.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	provider, err := h.llmService.UpdateProvider(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "UPDATE_PROVIDER_FAILED",
			Message: "更新提供商失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider 删除提供商
// @Summary 删除LLM提供商
// @Description 删除LLM提供商及其关联的模型
// @Tags LLM
// @Param id path string true "提供商ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id} [delete]
func (h *LLMHandler) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	if err := h.llmService.DeleteProvider(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DELETE_PROVIDER_FAILED",
			Message: "删除提供商失败: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestProviderConnection 测试提供商连接
// @Summary 测试LLM提供商连接
// @Description 测试LLM提供商的连接和配置是否正确
// @Tags LLM
// @Param id path string true "提供商ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id}/test [post]
func (h *LLMHandler) TestProviderConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	if err := h.llmService.TestProviderConnection(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "CONNECTION_TEST_FAILED",
			Message: "连接测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "连接测试成功",
	})
}

// CreateModel 创建模型
// @Summary 创建LLM模型
// @Description 为指定提供商创建新的LLM模型配置
// @Tags LLM
// @Accept json
// @Produce json
// @Param request body service.CreateModelRequest true "创建模型请求"
// @Success 201 {object} model.LLMModel
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/models [post]
func (h *LLMHandler) CreateModel(c *gin.Context) {
	var req service.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	model, err := h.llmService.CreateModel(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "CREATE_MODEL_FAILED",
			Message: "创建模型失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// GetModel 获取模型
// @Summary 获取LLM模型
// @Description 根据ID获取LLM模型详情
// @Tags LLM
// @Produce json
// @Param id path string true "模型ID"
// @Success 200 {object} model.LLMModel
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/llm/models/{id} [get]
func (h *LLMHandler) GetModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的模型ID",
		})
		return
	}

	model, err := h.llmService.GetModel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "MODEL_NOT_FOUND",
			Message: "模型未找到: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model)
}

// ListModels 列出模型
// @Summary 列出LLM模型
// @Description 获取LLM模型列表
// @Tags LLM
// @Produce json
// @Param provider_id query string false "提供商ID"
// @Param model_type query string false "模型类型"
// @Param is_active query boolean false "是否只返回激活的模型"
// @Success 200 {array} model.LLMModel
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/models [get]
func (h *LLMHandler) ListModels(c *gin.Context) {
	var providerID *uuid.UUID
	if providerIDStr := c.Query("provider_id"); providerIDStr != "" {
		if id, err := uuid.Parse(providerIDStr); err == nil {
			providerID = &id
		}
	}

	var modelType *string
	if modelTypeStr := c.Query("model_type"); modelTypeStr != "" {
		modelType = &modelTypeStr
	}

	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if active, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &active
		}
	}

	models, err := h.llmService.ListModels(c.Request.Context(), providerID, modelType, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "LIST_MODELS_FAILED",
			Message: "获取模型列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models)
}

// UpdateModel 更新模型
// @Summary 更新LLM模型
// @Description 更新LLM模型配置
// @Tags LLM
// @Accept json
// @Produce json
// @Param id path string true "模型ID"
// @Param request body service.UpdateModelRequest true "更新模型请求"
// @Success 200 {object} model.LLMModel
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/models/{id} [put]
func (h *LLMHandler) UpdateModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的模型ID",
		})
		return
	}

	var req service.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	model, err := h.llmService.UpdateModel(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "UPDATE_MODEL_FAILED",
			Message: "更新模型失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model)
}

// DeleteModel 删除模型
// @Summary 删除LLM模型
// @Description 删除LLM模型配置
// @Tags LLM
// @Param id path string true "模型ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/models/{id} [delete]
func (h *LLMHandler) DeleteModel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的模型ID",
		})
		return
	}

	if err := h.llmService.DeleteModel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DELETE_MODEL_FAILED",
			Message: "删除模型失败: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// SyncProviderModels 同步提供商模型
// @Summary 同步提供商模型
// @Description 从提供商API同步最新的模型列表
// @Tags LLM
// @Param id path string true "提供商ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id}/sync-models [post]
func (h *LLMHandler) SyncProviderModels(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	if err := h.llmService.SyncProviderModels(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "SYNC_MODELS_FAILED",
			Message: "同步模型失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "模型同步成功",
	})
}

// GetProviderStats 获取提供商统计信息
// @Summary 获取提供商统计信息
// @Description 获取提供商的模型数量等统计信息
// @Tags LLM
// @Param id path string true "提供商ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/providers/{id}/stats [get]
func (h *LLMHandler) GetProviderStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "无效的提供商ID",
		})
		return
	}

	stats, err := h.llmService.GetProviderStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "GET_STATS_FAILED",
			Message: "获取统计信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}