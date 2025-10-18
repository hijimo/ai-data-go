package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/session"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	messageService session.MessageService
	logger         logger.Logger
	validator      *validator.Validator
}

// NewMessageHandler 创建消息处理器实例
func NewMessageHandler(messageService session.MessageService, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		logger:         log,
		validator:      validator.New(),
	}
}

// SendMessage 发送消息
// @Summary 发送消息
// @Description 在指定会话中发送消息并获取AI回复
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param request body model.SendMessageRequest true "发送消息请求"
// @Success 200 {object} model.ResponseData[session.MessageResponse] "成功发送消息"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id}/messages [post]
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req model.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析发送消息请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 确保请求中的 SessionID 与 URL 中的一致
	req.SessionID = sessionID

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("发送消息请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 5. 记录请求日志
	h.logger.Info("收到发送消息请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"message":   req.Message,
	})

	// 6. 调用服务层发送消息
	serviceReq := &session.SendMessageRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    userID,
		Options:   req.Options,
	}

	messageResp, err := h.messageService.SendMessage(ctx, serviceReq)
	if err != nil {
		h.logger.Error("发送消息失败", logger.Fields{
			"error":     err,
			"sessionId": sessionID,
			"userId":    userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("发送消息成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"messageId": messageResp.MessageID,
	})

	// 8. 返回成功响应
	h.writeSuccessResponse(w, messageResp)
}

// GetMessages 获取消息历史
// @Summary 获取消息历史
// @Description 获取指定会话的消息历史，支持分页
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(50)
// @Success 200 {object} model.ResponsePaginationData[[]session.MessageDetailResponse] "成功返回消息历史"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id}/messages [get]
func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 解析查询参数
	req := &model.GetMessagesRequest{
		SessionID: sessionID,
	}
	if err := h.parseQueryParams(r, req); err != nil {
		h.logger.Error("解析获取消息查询参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的查询参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(req); validationErrors != nil {
		h.logger.Warn("获取消息请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 5. 记录请求日志
	h.logger.Info("收到获取消息历史请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"pageNo":    req.PageNo,
		"pageSize":  req.PageSize,
	})

	// 6. 调用服务层获取消息历史
	serviceReq := &session.GetMessagesRequest{
		SessionID: req.SessionID,
		UserID:    userID,
		PageNo:    req.PageNo,
		PageSize:  req.PageSize,
	}

	messageList, err := h.messageService.GetMessages(ctx, serviceReq)
	if err != nil {
		h.logger.Error("获取消息历史失败", logger.Fields{
			"error":     err,
			"sessionId": sessionID,
			"userId":    userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 7. 记录响应日志
	h.logger.Info("获取消息历史成功", logger.Fields{
		"sessionId":  sessionID,
		"userId":     userID,
		"totalCount": messageList.TotalCount,
	})

	// 8. 返回分页响应
	h.writeMessagePaginationResponse(w, messageList)
}

// GetMessageByID 获取单条消息详情
// @Summary 获取消息详情
// @Description 根据消息ID获取消息的详细信息
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "消息ID"
// @Success 200 {object} model.ResponseData[session.MessageDetailResponse] "成功返回消息详情"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "消息不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/messages/{id} [get]
func (h *MessageHandler) GetMessageByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取消息ID
	messageID := h.extractMessageID(r.URL.Path)
	if messageID == "" {
		h.logger.Warn("消息ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("消息ID不能为空"))
		return
	}

	// 2. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取消息详情请求", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 4. 调用服务层获取消息详情
	messageDetail, err := h.messageService.GetMessageByID(ctx, messageID, userID)
	if err != nil {
		h.logger.Error("获取消息详情失败", logger.Fields{
			"error":     err,
			"messageId": messageID,
			"userId":    userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("获取消息详情成功", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 6. 返回成功响应
	h.writeSuccessResponse(w, messageDetail)
}

// AbortMessage 中止消息生成
// @Summary 中止消息生成
// @Description 中止指定消息的AI生成过程
// @Tags messages
// @Accept json
// @Produce json
// @Param id path string true "消息ID"
// @Success 200 {object} model.ResponseData[any] "成功中止消息生成"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "消息不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/messages/{id}/abort [post]
func (h *MessageHandler) AbortMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取消息ID
	messageID := h.extractMessageIDFromAction(r.URL.Path, "/abort")
	if messageID == "" {
		h.logger.Warn("消息ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("消息ID不能为空"))
		return
	}

	// 2. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 3. 记录请求日志
	h.logger.Info("收到中止消息生成请求", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 4. 调用服务层中止消息生成
	err := h.messageService.AbortMessage(ctx, messageID, userID)
	if err != nil {
		h.logger.Error("中止消息生成失败", logger.Fields{
			"error":     err,
			"messageId": messageID,
			"userId":    userID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 5. 记录响应日志
	h.logger.Info("中止消息生成成功", logger.Fields{
		"messageId": messageID,
		"userId":    userID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	h.writeSuccessResponse(w, &emptyData)
}

// extractSessionID 从URL路径中提取会话ID
// 路径格式: /api/v1/chat/sessions/{id}/messages
func (h *MessageHandler) extractSessionID(path string) string {
	// 移除尾部的 /messages
	path = strings.TrimSuffix(path, "/messages")
	path = strings.TrimSuffix(path, "/")
	
	// 分割路径
	parts := strings.Split(path, "/")
	
	// 查找 "sessions" 后的部分
	for i, part := range parts {
		if part == "sessions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	
	return ""
}

// extractMessageID 从URL路径中提取消息ID
// 路径格式: /api/v1/chat/messages/{id}
func (h *MessageHandler) extractMessageID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")
	
	// 分割路径
	parts := strings.Split(path, "/")
	
	// 查找 "messages" 后的部分
	for i, part := range parts {
		if part == "messages" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	
	return ""
}

// extractMessageIDFromAction 从带操作的URL路径中提取消息ID
// 路径格式: /api/v1/chat/messages/{id}/abort
func (h *MessageHandler) extractMessageIDFromAction(path, action string) string {
	// 移除操作部分
	path = strings.TrimSuffix(path, action)
	path = strings.TrimSuffix(path, "/")
	
	// 使用 extractMessageID 提取ID
	return h.extractMessageID(path)
}

// parseQueryParams 解析查询参数到结构体
func (h *MessageHandler) parseQueryParams(r *http.Request, target interface{}) error {
	query := r.URL.Query()

	switch v := target.(type) {
	case *model.GetMessagesRequest:
		// 解析分页参数
		if err := h.parseIntParam(query.Get("pageNo"), &v.PageNo, 1); err != nil {
			return err
		}
		if err := h.parseIntParam(query.Get("pageSize"), &v.PageSize, 50); err != nil {
			return err
		}
	}

	return nil
}

// parseIntParam 解析整数参数
func (h *MessageHandler) parseIntParam(value string, target *int, defaultValue int) error {
	if value == "" {
		*target = defaultValue
		return nil
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return fmt.Errorf("无效的整数参数: %s", value)
	}

	*target = parsed
	return nil
}

// writeSuccessResponse 写入成功响应
func (h *MessageHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	// 直接构建响应，避免泛型类型推断问题
	resp := map[string]interface{}{
		"code":    errors.CodeSuccess,
		"message": errors.MsgSuccess,
		"data":    data,
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeMessagePaginationResponse 写入消息分页响应
func (h *MessageHandler) writeMessagePaginationResponse(w http.ResponseWriter, messageList *session.MessageListResponse) {
	resp := map[string]interface{}{
		"code":    errors.CodeSuccess,
		"message": errors.MsgSuccess,
		"data": map[string]interface{}{
			"data":       messageList.Messages,
			"pageNo":     messageList.PageNo,
			"pageSize":   messageList.PageSize,
			"totalCount": messageList.TotalCount,
			"totalPage":  messageList.TotalPage,
		},
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeErrorResponse 写入错误响应
func (h *MessageHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
	resp := response.Error[any](appErr.Code, appErr.Message)

	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound, errors.CodeSessionNotFound, errors.CodeMessageNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden, errors.CodeSessionAccessDenied, errors.CodeMessageAccessDenied:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	case errors.CodeMessageSendFailed:
		statusCode = http.StatusInternalServerError
	}

	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *MessageHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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
func (h *MessageHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
