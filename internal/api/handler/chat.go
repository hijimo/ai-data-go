package handler

import (
	"encoding/json"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// ChatHandler 对话接口处理器
type ChatHandler struct {
	aiService ai.AIService
	logger    logger.Logger
	validator *validator.Validator
}

// NewChatHandler 创建对话处理器实例
func NewChatHandler(aiService ai.AIService, log logger.Logger) *ChatHandler {
	return &ChatHandler{
		aiService: aiService,
		logger:    log,
		validator: validator.New(),
	}
}

// HandleChat 处理对话请求
// @Summary 发送对话消息
// @Description 向 AI 发送消息并获取回复，支持会话上下文管理
// @Tags chat
// @Accept json
// @Produce json
// @Param request body model.ChatRequest true "对话请求"
// @Success 200 {object} model.ResponseData[model.ChatResponse] "成功返回 AI 回复"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Failure 503 {object} model.ErrorResponse "AI 服务不可用"
// @Router /chat [post]
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到对话请求", logger.Fields{
		"message":    req.Message,
		"sessionId":  req.SessionID,
		"hasOptions": req.Options != nil,
	})

	// 4. 调用 AI 服务处理对话
	chatResp, err := h.aiService.Chat(ctx, &req)
	if err != nil {
		h.logger.Error("AI 服务调用失败", logger.Fields{"error": err})
		
		// 判断错误类型并返回相应的错误响应
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewAIServiceError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("对话请求处理成功", logger.Fields{
		"sessionId":     chatResp.SessionID,
		"model":         chatResp.Model,
		"messageLength": len(chatResp.Message),
	})

	// 6. 构建并返回成功响应
	h.writeSuccessResponse(w, chatResp)
}

// writeSuccessResponse 写入成功响应
func (h *ChatHandler) writeSuccessResponse(w http.ResponseWriter, data *model.ChatResponse) {
	resp := response.Success(data)
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeErrorResponse 写入错误响应
func (h *ChatHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
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
	case errors.CodeAIServiceError, errors.CodeContextCancelled:
		statusCode = http.StatusInternalServerError
	}
	
	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *ChatHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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

// writeJSONResponse 写入 JSON 响应
func (h *ChatHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
