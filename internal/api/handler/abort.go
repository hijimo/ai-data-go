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

// AbortHandler 中止接口处理器
type AbortHandler struct {
	aiService ai.AIService
	logger    logger.Logger
	validator *validator.Validator
}

// NewAbortHandler 创建中止处理器实例
func NewAbortHandler(aiService ai.AIService, log logger.Logger) *AbortHandler {
	return &AbortHandler{
		aiService: aiService,
		logger:    log,
		validator: validator.New(),
	}
}

// HandleAbort 处理中止对话请求
// POST /api/v1/chat/abort
func (h *AbortHandler) HandleAbort(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req model.AbortRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析中止请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("中止请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 记录请求日志
	h.logger.Info("收到中止对话请求", logger.Fields{
		"sessionId": req.SessionID,
	})

	// 4. 调用 AI 服务中止对话
	err := h.aiService.AbortChat(ctx, req.SessionID)
	if err != nil {
		h.logger.Error("中止对话失败", logger.Fields{
			"sessionId": req.SessionID,
			"error":     err,
		})

		// 判断错误类型并返回相应的错误响应
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录成功日志
	h.logger.Info("对话已成功中止", logger.Fields{
		"sessionId": req.SessionID,
	})

	// 6. 构建并返回成功响应
	h.writeSuccessResponse(w)
}

// writeSuccessResponse 写入成功响应
func (h *AbortHandler) writeSuccessResponse(w http.ResponseWriter) {
	resp := response.Success[any](nil)
	resp.Message = "对话已成功中止"
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeErrorResponse 写入错误响应
func (h *AbortHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
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
func (h *AbortHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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
func (h *AbortHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
