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

// GenkitChatHandler Genkit对话处理器
type GenkitChatHandler struct {
	genkit *genkit.Genkit
}

// NewGenkitChatHandler 创建新的Genkit对话处理器
func NewGenkitChatHandler(g *genkit.Genkit) *GenkitChatHandler {
	return &GenkitChatHandler{genkit: g}
}

// HandleGenerate 处理对话生成请求
// @Summary 生成AI对话响应
// @Description 基于上下文生成AI响应
// @Tags Chat
// @Accept json
// @Produce json
// @Param input body flows.ChatGenerateInput true "对话生成输入"
// @Success 200 {object} model.ChatGenerateOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/chat/generate [post]
// @Security Bearer
func (h *GenkitChatHandler) HandleGenerate(c *gin.Context) {
	var input flows.ChatGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.ChatGenerateInput, flows.ChatGenerateOutput](
		h.genkit,
		"chatGenerateFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "chatGenerateFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "生成对话响应失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "对话生成成功", output)
}

// HandleStream 处理流式对话请求
// @Summary 流式生成AI对话响应
// @Description 以流式方式生成AI响应
// @Tags Chat
// @Accept json
// @Produce text/event-stream
// @Param input body flows.ChatStreamInput true "流式对话输入"
// @Success 200 {string} string "流式返回 AI 回复（Server-Sent Events 格式）"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/chat/stream [post]
// @Security Bearer
func (h *GenkitChatHandler) HandleStream(c *gin.Context) {
	var input flows.ChatStreamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.ChatStreamInput, flows.ChatStreamOutput](
		h.genkit,
		"chatStreamFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "chatStreamFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "流式对话失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "流式对话成功", output)
}
