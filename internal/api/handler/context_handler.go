// Package handler 提供 HTTP 请求处理器
package handler

import (
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"

	"genkit-ai-service/internal/genkit/flows"
	_ "genkit-ai-service/internal/model" // 用于 Swagger 文档
	"genkit-ai-service/pkg/response"
)

// ContextHandler 上下文处理器
type ContextHandler struct {
	genkit *genkit.Genkit
}

// NewContextHandler 创建新的上下文处理器
func NewContextHandler(g *genkit.Genkit) *ContextHandler {
	return &ContextHandler{genkit: g}
}

// HandleBuildContext 处理构建上下文请求
// @Summary 构建对话上下文
// @Description 根据会话ID和用户查询构建智能对话上下文
// @Tags Context
// @Accept json
// @Produce json
// @Param input body flows.ContextBuildInput true "上下文构建输入"
// @Success 200 {object} model.ContextBuildOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/context/build [post]
// @Security Bearer
func (h *ContextHandler) HandleBuildContext(c *gin.Context) {
	var input flows.ContextBuildInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.ContextBuildInput, flows.ContextBuildOutput](
		h.genkit,
		"contextBuildFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "contextBuildFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		message := "构建上下文失败"

		// 这里可以根据错误类型进行更细粒度的处理
		// 例如：权限错误返回 403，参数错误返回 400 等

		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "上下文构建成功", output)
}
