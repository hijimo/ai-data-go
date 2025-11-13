package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/validator"
)

// ChatStreamHandler 流式对话接口处理器
type ChatStreamHandler struct {
	aiService ai.AIService
	logger    logger.Logger
	validator *validator.Validator
}

// NewChatStreamHandler 创建流式对话处理器实例
func NewChatStreamHandler(aiService ai.AIService, log logger.Logger) *ChatStreamHandler {
	return &ChatStreamHandler{
		aiService: aiService,
		logger:    log,
		validator: validator.New(),
	}
}

// HandleChatStream 处理流式对话请求
// @Summary 发送流式对话消息
// @Description 向 AI 发送消息并以流式方式获取回复，支持通过 messageId 继续对话
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Param request body model.ChatRequest true "对话请求"
// @Success 200 {string} string "流式返回 AI 回复（Server-Sent Events 格式）"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Failure 503 {object} model.ErrorResponse "AI 服务不可用"
// @Router /chat/stream [post]
func (h *ChatStreamHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
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

	// 3. 检查是否支持流式传输（在设置响应头之前）
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("响应写入器不支持流式传输", logger.Fields{})
		h.writeErrorResponse(w, errors.NewInternalError(fmt.Errorf("不支持流式传输")))
		return
	}

	// 4. 记录请求日志
	h.logger.Info("收到流式对话请求", logger.Fields{
		"message":    req.Message,
		"messageId":  req.MessageID,
		"hasOptions": req.Options != nil,
	})

	// 5. 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 6. 调用 AI 服务处理流式对话
	streamChan, err := h.aiService.ChatStream(ctx, &req)
	if err != nil {
		h.logger.Error("AI 流式服务调用失败", logger.Fields{"error": err})
		
		// 判断错误类型并返回相应的错误响应
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeSSEError(w, appErr)
		} else {
			h.writeSSEError(w, errors.NewAIServiceError(err))
		}
		flusher.Flush()
		return
	}

	// 7. 流式发送响应（腾讯云格式）
	for chunk := range streamChan {
		// 检查是否是错误消息
		if chunk.FinishReason == "error" {
			h.logger.Error("流式响应出错", logger.Fields{"message": chunk.Processes.Message})
		}

		// 发送数据块
		if err := h.writeSSEChunk(w, chunk); err != nil {
			h.logger.Error("写入流式数据失败", logger.Fields{"error": err})
			return
		}
		flusher.Flush()

		// 如果是停止消息，记录日志
		if chunk.IsStop {
			h.logger.Info("流式对话请求处理成功", logger.Fields{
				"sessionId":    chunk.SessionID,
				"completionId": chunk.CompletionID,
			})
			break
		}
	}
}

// writeSSEChunk 写入 SSE 数据块（腾讯云格式）
func (h *ChatStreamHandler) writeSSEChunk(w http.ResponseWriter, chunk *model.TencentCloudStreamMessage) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// 腾讯云 SSE 格式
	// 如果是 finish 事件，添加 event 行
	if chunk.IsStop && chunk.FinishReason == "stop" {
		_, err = fmt.Fprintf(w, "event: finish\n")
		if err != nil {
			return err
		}
	}

	// data: {json}\n\n
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

// writeSSEError 写入 SSE 错误（腾讯云格式）
func (h *ChatStreamHandler) writeSSEError(w http.ResponseWriter, appErr *errors.AppError) {
	errorChunk := &model.TencentCloudStreamMessage{
		CompletionID: "",
		Processes: model.ProcessInfo{
			Stage:   model.StreamStageOutput,
			Message: appErr.Message,
		},
		FinishReason: "error",
		IsStop:       true,
	}

	data, _ := json.Marshal(errorChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// writeErrorResponse 写入错误响应（用于非流式错误）
func (h *ChatStreamHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
	// 构建错误响应
	errorData := map[string]interface{}{
		"code":    appErr.Code,
		"message": appErr.Message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}
	
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorData)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *ChatStreamHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
	// 构建验证错误详情
	errorData := map[string]interface{}{
		"code":    errors.CodeValidationError,
		"message": errors.MsgValidationError,
		"errors":  validationErrors,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(errorData)
}
