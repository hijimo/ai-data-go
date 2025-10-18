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

// SessionHandler 会话处理器
type SessionHandler struct {
	sessionService session.SessionService
	logger         logger.Logger
	validator      *validator.Validator
}

// NewSessionHandler 创建会话处理器实例
func NewSessionHandler(sessionService session.SessionService, log logger.Logger) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		logger:         log,
		validator:      validator.New(),
	}
}

// CreateSession 创建会话
// @Summary 创建新会话
// @Description 创建一个新的聊天会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body model.CreateSessionRequest true "创建会话请求"
// @Success 200 {object} model.ResponseData[model.SessionResponse] "成功创建会话"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions [post]
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req model.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析创建会话请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("创建会话请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 从上下文获取用户ID（TODO: 实际应从认证中间件获取）
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 4. 记录请求日志
	h.logger.Info("收到创建会话请求", logger.Fields{
		"userId":    userID,
		"title":     req.Title,
		"modelName": req.ModelName,
	})

	// 5. 调用服务层创建会话
	sessionResp, err := h.sessionService.CreateSession(ctx, userID, &req)
	if err != nil {
		h.logger.Error("创建会话失败", logger.Fields{"error": err, "userId": userID})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("创建会话成功", logger.Fields{
		"sessionId": sessionResp.ID,
		"userId":    userID,
	})

	// 7. 返回成功响应
	h.writeSuccessResponse(w, sessionResp)
}

// ListSessions 获取会话列表
// @Summary 获取会话列表
// @Description 获取用户的会话列表，支持分页和过滤
// @Tags sessions
// @Accept json
// @Produce json
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(20)
// @Param isPinned query bool false "是否置顶"
// @Param isArchived query bool false "是否归档"
// @Param modelName query string false "模型名称"
// @Success 200 {object} model.ResponsePaginationData[[]model.SessionResponse] "成功返回会话列表"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions [get]
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析查询参数
	req := &model.ListSessionsRequest{}
	if err := h.parseQueryParams(r, req); err != nil {
		h.logger.Error("解析会话列表查询参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的查询参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(req); validationErrors != nil {
		h.logger.Warn("会话列表请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 4. 记录请求日志
	h.logger.Info("收到获取会话列表请求", logger.Fields{
		"userId":   userID,
		"pageNo":   req.PageNo,
		"pageSize": req.PageSize,
	})

	// 5. 调用服务层获取会话列表
	sessions, total, err := h.sessionService.ListSessions(ctx, userID, req)
	if err != nil {
		h.logger.Error("获取会话列表失败", logger.Fields{"error": err, "userId": userID})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("获取会话列表成功", logger.Fields{
		"userId": userID,
		"count":  len(sessions),
		"total":  total,
	})

	// 7. 返回分页响应
	h.writePaginationResponse(w, sessions, req.PageNo, req.PageSize, total)
}

// GetSession 获取会话详情
// @Summary 获取会话详情
// @Description 根据会话ID获取会话的详细信息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} model.ResponseData[model.SessionResponse] "成功返回会话详情"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id} [get]
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 3. 记录请求日志
	h.logger.Info("收到获取会话详情请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 4. 调用服务层获取会话详情
	sessionResp, err := h.sessionService.GetSession(ctx, sessionID, userID)
	if err != nil {
		h.logger.Error("获取会话详情失败", logger.Fields{
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

	// 5. 记录响应日志
	h.logger.Info("获取会话详情成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 6. 返回成功响应
	h.writeSuccessResponse(w, sessionResp)
}

// UpdateSession 更新会话
// @Summary 更新会话
// @Description 更新会话的标题、配置等信息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param request body model.UpdateSessionRequest true "更新会话请求"
// @Success 200 {object} model.ResponseData[model.SessionResponse] "成功更新会话"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id} [patch]
func (h *SessionHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 解析请求参数
	var req model.UpdateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新会话请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新会话请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 4. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 5. 记录请求日志
	h.logger.Info("收到更新会话请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 6. 调用服务层更新会话
	sessionResp, err := h.sessionService.UpdateSession(ctx, sessionID, userID, &req)
	if err != nil {
		h.logger.Error("更新会话失败", logger.Fields{
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
	h.logger.Info("更新会话成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 8. 返回成功响应
	h.writeSuccessResponse(w, sessionResp)
}

// DeleteSession 删除会话
// @Summary 删除会话
// @Description 软删除指定的会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} model.ResponseData[any] "成功删除会话"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id} [delete]
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 3. 记录请求日志
	h.logger.Info("收到删除会话请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 4. 调用服务层删除会话
	err := h.sessionService.DeleteSession(ctx, sessionID, userID)
	if err != nil {
		h.logger.Error("删除会话失败", logger.Fields{
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

	// 5. 记录响应日志
	h.logger.Info("删除会话成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
	})

	// 6. 返回成功响应
	emptyData := struct{}{}
	h.writeSuccessResponse(w, &emptyData)
}

// SearchSessions 搜索会话
// @Summary 搜索会话
// @Description 根据关键词搜索会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Param pageNo query int true "页码" minimum(1) default(1)
// @Param pageSize query int true "每页大小" minimum(1) maximum(100) default(20)
// @Success 200 {object} model.ResponsePaginationData[[]model.SessionResponse] "成功返回搜索结果"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 422 {object} model.ErrorResponse "参数验证失败"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/search [get]
func (h *SessionHandler) SearchSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析查询参数
	req := &model.SearchSessionsRequest{}
	if err := h.parseQueryParams(r, req); err != nil {
		h.logger.Error("解析搜索会话查询参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, errors.NewBadRequestError("无效的查询参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(req); validationErrors != nil {
		h.logger.Warn("搜索会话请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, validationErrors)
		return
	}

	// 3. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 4. 记录请求日志
	h.logger.Info("收到搜索会话请求", logger.Fields{
		"userId":  userID,
		"keyword": req.Keyword,
		"pageNo":  req.PageNo,
	})

	// 5. 调用服务层搜索会话
	sessions, total, err := h.sessionService.SearchSessions(ctx, userID, req)
	if err != nil {
		h.logger.Error("搜索会话失败", logger.Fields{"error": err, "userId": userID})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, appErr)
		} else {
			h.writeErrorResponse(w, errors.NewInternalError(err))
		}
		return
	}

	// 6. 记录响应日志
	h.logger.Info("搜索会话成功", logger.Fields{
		"userId": userID,
		"count":  len(sessions),
		"total":  total,
	})

	// 7. 返回分页响应
	h.writePaginationResponse(w, sessions, req.PageNo, req.PageSize, total)
}

// PinSession 置顶/取消置顶会话
// @Summary 置顶会话
// @Description 置顶或取消置顶指定的会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param pinned query bool true "是否置顶"
// @Success 200 {object} model.ResponseData[any] "成功更新置顶状态"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id}/pin [post]
func (h *SessionHandler) PinSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionIDFromAction(r.URL.Path, "/pin")
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 获取查询参数
	pinnedStr := r.URL.Query().Get("pinned")
	pinned := pinnedStr == "true"

	// 3. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 4. 记录请求日志
	h.logger.Info("收到置顶会话请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"pinned":    pinned,
	})

	// 5. 调用服务层更新置顶状态
	err := h.sessionService.PinSession(ctx, sessionID, userID, pinned)
	if err != nil {
		h.logger.Error("更新置顶状态失败", logger.Fields{
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

	// 6. 记录响应日志
	h.logger.Info("更新置顶状态成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"pinned":    pinned,
	})

	// 7. 返回成功响应
	emptyData := struct{}{}
	h.writeSuccessResponse(w, &emptyData)
}

// ArchiveSession 归档/取消归档会话
// @Summary 归档会话
// @Description 归档或取消归档指定的会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param archived query bool true "是否归档"
// @Success 200 {object} model.ResponseData[any] "成功更新归档状态"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 403 {object} model.ErrorResponse "无权访问"
// @Failure 404 {object} model.ErrorResponse "会话不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /chat/sessions/{id}/archive [post]
func (h *SessionHandler) ArchiveSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionIDFromAction(r.URL.Path, "/archive")
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 2. 获取查询参数
	archivedStr := r.URL.Query().Get("archived")
	archived := archivedStr == "true"

	// 3. 从上下文获取用户ID
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user-id" // 临时默认值
	}

	// 4. 记录请求日志
	h.logger.Info("收到归档会话请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"archived":  archived,
	})

	// 5. 调用服务层更新归档状态
	err := h.sessionService.ArchiveSession(ctx, sessionID, userID, archived)
	if err != nil {
		h.logger.Error("更新归档状态失败", logger.Fields{
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

	// 6. 记录响应日志
	h.logger.Info("更新归档状态成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"archived":  archived,
	})

	// 7. 返回成功响应
	emptyData := struct{}{}
	h.writeSuccessResponse(w, &emptyData)
}

// extractSessionID 从URL路径中提取会话ID
// 路径格式: /api/v1/chat/sessions/{id}
func (h *SessionHandler) extractSessionID(path string) string {
	// 移除尾部斜杠
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

// extractSessionIDFromAction 从带操作的URL路径中提取会话ID
// 路径格式: /api/v1/chat/sessions/{id}/pin 或 /api/v1/chat/sessions/{id}/archive
func (h *SessionHandler) extractSessionIDFromAction(path, action string) string {
	// 移除操作部分
	path = strings.TrimSuffix(path, action)
	path = strings.TrimSuffix(path, "/")
	
	// 使用 extractSessionID 提取ID
	return h.extractSessionID(path)
}

// parseQueryParams 解析查询参数到结构体
func (h *SessionHandler) parseQueryParams(r *http.Request, target interface{}) error {
	query := r.URL.Query()

	switch v := target.(type) {
	case *model.ListSessionsRequest:
		// 解析分页参数
		if err := h.parseIntParam(query.Get("pageNo"), &v.PageNo, 1); err != nil {
			return err
		}
		if err := h.parseIntParam(query.Get("pageSize"), &v.PageSize, 20); err != nil {
			return err
		}

		// 解析可选参数
		if pinnedStr := query.Get("isPinned"); pinnedStr != "" {
			pinned := pinnedStr == "true"
			v.IsPinned = &pinned
		}
		if archivedStr := query.Get("isArchived"); archivedStr != "" {
			archived := archivedStr == "true"
			v.IsArchived = &archived
		}
		v.ModelName = query.Get("modelName")

	case *model.SearchSessionsRequest:
		v.Keyword = query.Get("keyword")
		if err := h.parseIntParam(query.Get("pageNo"), &v.PageNo, 1); err != nil {
			return err
		}
		if err := h.parseIntParam(query.Get("pageSize"), &v.PageSize, 20); err != nil {
			return err
		}
	}

	return nil
}

// parseIntParam 解析整数参数
func (h *SessionHandler) parseIntParam(value string, target *int, defaultValue int) error {
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
func (h *SessionHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	// 直接构建响应，避免泛型类型推断问题
	resp := map[string]interface{}{
		"code":    errors.CodeSuccess,
		"message": errors.MsgSuccess,
		"data":    data,
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writePaginationResponse 写入分页响应
func (h *SessionHandler) writePaginationResponse(w http.ResponseWriter, data []*model.SessionResponse, pageNo, pageSize, total int) {
	totalPage := total / pageSize
	if total%pageSize > 0 {
		totalPage++
	}

	resp := map[string]interface{}{
		"code":    errors.CodeSuccess,
		"message": errors.MsgSuccess,
		"data": map[string]interface{}{
			"data":       data,
			"pageNo":     pageNo,
			"pageSize":   pageSize,
			"totalCount": total,
			"totalPage":  totalPage,
		},
	}
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeErrorResponse 写入错误响应
func (h *SessionHandler) writeErrorResponse(w http.ResponseWriter, appErr *errors.AppError) {
	resp := response.Error[any](appErr.Code, appErr.Message)

	// 根据错误码确定 HTTP 状态码
	statusCode := http.StatusInternalServerError
	switch appErr.Code {
	case errors.CodeBadRequest:
		statusCode = http.StatusBadRequest
	case errors.CodeValidationError:
		statusCode = http.StatusUnprocessableEntity
	case errors.CodeNotFound, errors.CodeSessionNotFound:
		statusCode = http.StatusNotFound
	case errors.CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.CodeForbidden, errors.CodeSessionAccessDenied:
		statusCode = http.StatusForbidden
	case errors.CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	}

	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *SessionHandler) writeValidationErrorResponse(w http.ResponseWriter, validationErrors []validator.ValidationError) {
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
func (h *SessionHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
