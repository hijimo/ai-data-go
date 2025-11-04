// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"

	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/response"
)

// MemoryHandler 记忆管理处理器
type MemoryHandler struct {
	genkit *genkit.Genkit
}

// NewMemoryHandler 创建新的记忆管理处理器
func NewMemoryHandler(g *genkit.Genkit) *MemoryHandler {
	return &MemoryHandler{genkit: g}
}

// HandleSearch 处理记忆检索请求
// @Summary 检索记忆
// @Description 基于向量相似度检索相关的历史对话记忆
// @Tags Memory
// @Accept json
// @Produce json
// @Param input body flows.MemorySearchInput true "记忆检索输入"
// @Success 200 {object} model.MemorySearchOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/memory/search [post]
// @Security Bearer
func (h *MemoryHandler) HandleSearch(c *gin.Context) {
	var input flows.MemorySearchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.MemorySearchInput, flows.MemorySearchOutput](
		h.genkit,
		"memorySearchFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "memorySearchFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "检索记忆失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "记忆检索成功", output)
}

// HandleStore 处理记忆存储请求
// @Summary 存储记忆
// @Description 将对话消息转换为长期记忆并存储
// @Tags Memory
// @Accept json
// @Produce json
// @Param input body flows.MemoryStoreInput true "记忆存储输入"
// @Success 200 {object} model.MemoryStoreOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/memory/store [post]
// @Security Bearer
func (h *MemoryHandler) HandleStore(c *gin.Context) {
	var input flows.MemoryStoreInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.MemoryStoreInput, flows.MemoryStoreOutput](
		h.genkit,
		"memoryStoreFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "memoryStoreFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "存储记忆失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "记忆存储成功", output)
}

// HandleCleanup 处理记忆清理请求
// @Summary 清理记忆
// @Description 根据策略清理过期或低质量的记忆
// @Tags Memory
// @Accept json
// @Produce json
// @Param input body flows.MemoryCleanupInput true "记忆清理输入"
// @Success 200 {object} model.MemoryCleanupOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/memory/cleanup [post]
// @Security Bearer
func (h *MemoryHandler) HandleCleanup(c *gin.Context) {
	var input flows.MemoryCleanupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.MemoryCleanupInput, flows.MemoryCleanupOutput](
		h.genkit,
		"memoryCleanupFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "memoryCleanupFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "清理记忆失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "记忆清理成功", output)
}
