// internal/api/handler/token_handler.go
package handler

import (
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/response"
)

// TokenHandler Token管理处理器
type TokenHandler struct {
	genkit *genkit.Genkit
}

// NewTokenHandler 创建Token处理器
func NewTokenHandler(g *genkit.Genkit) *TokenHandler {
	return &TokenHandler{genkit: g}
}

// HandleBudget 处理Token预算查询
// @Summary 查询Token预算状态
// @Description 查询会话、每日或每月的Token预算使用情况
// @Tags Token管理
// @Accept json
// @Produce json
// @Param request body flows.TokenBudgetInput true "预算查询请求"
// @Success 200 {object} model.TokenBudgetOutputResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/tokens/budget [post]
func (h *TokenHandler) HandleBudget(c *gin.Context) {
	var input flows.TokenBudgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, response.ResponseData[any]{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误",
			Data:    nil,
		})
		return
	}

	// 查找并调用Flow
	flow := genkit.LookupFlow[flows.TokenBudgetInput, flows.TokenBudgetOutput](
		h.genkit,
		"tokenBudgetFlow",
	)

	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ResponseData[any]{
			Code:    http.StatusInternalServerError,
			Message: "查询Token预算失败",
			Data:    nil,
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, response.ResponseData[flows.TokenBudgetOutput]{
		Code:    http.StatusOK,
		Message: "查询Token预算成功",
		Data:    &output,
	})
}

// HandleOptimize 处理Token优化
// @Summary 优化内容以减少Token
// @Description 使用指定策略优化内容，减少Token消耗
// @Tags Token管理
// @Accept json
// @Produce json
// @Param request body flows.TokenOptimizeInput true "优化请求"
// @Success 200 {object} model.TokenOptimizeOutputResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/tokens/optimize [post]
func (h *TokenHandler) HandleOptimize(c *gin.Context) {
	var input flows.TokenOptimizeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, response.ResponseData[any]{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误",
			Data:    nil,
		})
		return
	}

	// 查找并调用Flow
	flow := genkit.LookupFlow[flows.TokenOptimizeInput, flows.TokenOptimizeOutput](
		h.genkit,
		"tokenOptimizeFlow",
	)

	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ResponseData[any]{
			Code:    http.StatusInternalServerError,
			Message: "Token优化失败",
			Data:    nil,
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, response.ResponseData[flows.TokenOptimizeOutput]{
		Code:    http.StatusOK,
		Message: "Token优化成功",
		Data:    &output,
	})
}

// HandleAnalysis 处理Token使用分析
// @Summary 分析Token使用情况
// @Description 分析指定时间范围内的Token使用情况，提供趋势和优化建议
// @Tags Token管理
// @Accept json
// @Produce json
// @Param request body flows.TokenAnalysisInput true "分析请求"
// @Success 200 {object} model.TokenAnalysisOutputResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/v1/tokens/analysis [post]
func (h *TokenHandler) HandleAnalysis(c *gin.Context) {
	var input flows.TokenAnalysisInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, response.ResponseData[any]{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误",
			Data:    nil,
		})
		return
	}

	// 查找并调用Flow
	flow := genkit.LookupFlow[flows.TokenAnalysisInput, flows.TokenAnalysisOutput](
		h.genkit,
		"tokenAnalysisFlow",
	)

	output, err := flow.Run(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ResponseData[any]{
			Code:    http.StatusInternalServerError,
			Message: "Token使用分析失败",
			Data:    nil,
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, response.ResponseData[flows.TokenAnalysisOutput]{
		Code:    http.StatusOK,
		Message: "Token使用分析成功",
		Data:    &output,
	})
}
