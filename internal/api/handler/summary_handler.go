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

// SummaryHandler 摘要管理处理器
type SummaryHandler struct {
	genkit *genkit.Genkit
}

// NewSummaryHandler 创建新的摘要管理处理器
func NewSummaryHandler(g *genkit.Genkit) *SummaryHandler {
	return &SummaryHandler{genkit: g}
}

// HandleGenerate 处理摘要生成请求
// @Summary 生成摘要
// @Description 自动生成对话摘要以压缩历史对话
// @Tags Summary
// @Accept json
// @Produce json
// @Param input body flows.SummaryGenerateInput true "摘要生成输入"
// @Success 200 {object} model.SummaryGenerateOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/summary/generate [post]
// @Security Bearer
func (h *SummaryHandler) HandleGenerate(c *gin.Context) {
	var input flows.SummaryGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.SummaryGenerateInput, flows.SummaryGenerateOutput](
		h.genkit,
		"summaryGenerateFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "summaryGenerateFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "生成摘要失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "摘要生成成功", output)
}

// HandleTrigger 处理摘要触发检查请求
// @Summary 检查摘要触发条件
// @Description 智能判断是否需要生成摘要
// @Tags Summary
// @Accept json
// @Produce json
// @Param input body flows.SummaryTriggerInput true "摘要触发检查输入"
// @Success 200 {object} model.SummaryTriggerOutputResponse "成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未认证"
// @Failure 403 {object} model.ErrorResponse "权限不足"
// @Failure 500 {object} model.ErrorResponse "内部服务器错误"
// @Router /api/v1/summary/trigger [post]
// @Security Bearer
func (h *SummaryHandler) HandleTrigger(c *gin.Context) {
	var input flows.SummaryTriggerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 查找并调用 Flow
	flow := genkit.LookupFlow[flows.SummaryTriggerInput, flows.SummaryTriggerOutput](
		h.genkit,
		"summaryTriggerFlow",
	)

	if flow == nil {
		response.Error(c, http.StatusInternalServerError, "Flow 未找到", "summaryTriggerFlow 未注册")
		return
	}

	// 执行 Flow
	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "检查摘要触发条件失败"
		response.Error(c, statusCode, message, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, "摘要触发检查成功", output)
}
