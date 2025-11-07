package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"

	"github.com/google/uuid"
)

// ContextHandler 上下文管理处理器
// 提供上下文构建、配置查询和更新等功能
type ContextHandler struct {
	contextService service.ContextService
	logger         logger.Logger
	validator      *validator.Validator
}

// NewContextHandler 创建上下文管理处理器实例
func NewContextHandler(contextService service.ContextService, log logger.Logger) *ContextHandler {
	return &ContextHandler{
		contextService: contextService,
		logger:         log,
		validator:      validator.New(),
	}
}

// BuildContextRequest 构建上下文请求
type BuildContextRequest struct {
	SessionID       string `json:"sessionId" validate:"required,uuid"`
	UserQuery       string `json:"userQuery" validate:"required,max=2000"`
	MaxTokens       int    `json:"maxTokens" validate:"omitempty,min=100,max=32000"`
	Strategy        string `json:"strategy" validate:"omitempty,oneof=auto short full"`
	IncludeSummary  *bool  `json:"includeSummary"`
	IncludeLongTerm *bool  `json:"includeLongTerm"`
	ShortTermWindow int    `json:"shortTermWindow" validate:"omitempty,min=1,max=50"`
}

// BuildContextResponse 构建上下文响应
type BuildContextResponse struct {
	SessionID         string                  `json:"sessionId"`
	Summary           *flows.SummaryContext   `json:"summary,omitempty"`
	LongTermMemories  []flows.MemoryContext   `json:"longTermMemories,omitempty"`
	ShortTermMessages []flows.MessageContext  `json:"shortTermMessages"`
	TotalTokens       int                     `json:"totalTokens"`
	Strategy          string                  `json:"strategy"`
	QualityScore      float64                 `json:"qualityScore"`
	BuildTime         int64                   `json:"buildTime"`
}

// GetContextConfigResponse 获取上下文配置响应
type GetContextConfigResponse struct {
	ID              string  `json:"id"`
	SessionID       string  `json:"sessionId"`
	MaxTokens       int     `json:"maxTokens"`
	Strategy        string  `json:"strategy"`
	IncludeSummary  bool    `json:"includeSummary"`
	IncludeLongTerm bool    `json:"includeLongTerm"`
	ShortTermWindow int     `json:"shortTermWindow"`
	LastSummaryID   *string `json:"lastSummaryId,omitempty"`
	LastSummaryAt   *string `json:"lastSummaryAt,omitempty"`
	TotalMessages   int     `json:"totalMessages"`
	TotalTokensUsed int64   `json:"totalTokensUsed"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

// UpdateContextConfigRequest 更新上下文配置请求
type UpdateContextConfigRequest struct {
	MaxTokens       *int    `json:"maxTokens" validate:"omitempty,min=100,max=32000"`
	Strategy        *string `json:"strategy" validate:"omitempty,oneof=auto short full"`
	IncludeSummary  *bool   `json:"includeSummary"`
	IncludeLongTerm *bool   `json:"includeLongTerm"`
	ShortTermWindow *int    `json:"shortTermWindow" validate:"omitempty,min=1,max=50"`
}


// HandleBuildContext 处理构建上下文请求
func (h *ContextHandler) HandleBuildContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// 1. 解析请求参数
	var req BuildContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析构建上下文请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("构建上下文请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 3. 从JWT token中获取用户ID和租户ID
	userID, ok := middleware.GetAuthUserID(ctx)
	if !ok || userID == "" {
		h.logger.Warn("缺少用户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少用户认证信息"))
		return
	}

	tenantID, ok := middleware.GetTenantID(ctx)
	if !ok || tenantID == "" {
		h.logger.Warn("缺少租户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少租户信息"))
		return
	}

	// 验证 userID 和 tenantID 是否为有效的 UUID
	if _, err := uuid.Parse(userID); err != nil {
		h.logger.Warn("用户ID格式无效", logger.Fields{"userId": userID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("用户ID格式无效"))
		return
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 4. 设置默认值
	if req.MaxTokens == 0 {
		req.MaxTokens = 4000
	}
	if req.Strategy == "" {
		req.Strategy = "auto"
	}
	if req.ShortTermWindow == 0 {
		req.ShortTermWindow = 10
	}
	includeSummary := true
	if req.IncludeSummary != nil {
		includeSummary = *req.IncludeSummary
	}
	includeLongTerm := true
	if req.IncludeLongTerm != nil {
		includeLongTerm = *req.IncludeLongTerm
	}

	// 5. 记录请求日志
	h.logger.InfoContext(ctx, "收到构建上下文请求", logger.Fields{
		"userId":          userID,
		"tenantId":        tenantID,
		"sessionId":       req.SessionID,
		"maxTokens":       req.MaxTokens,
		"strategy":        req.Strategy,
		"includeSummary":  includeSummary,
		"includeLongTerm": includeLongTerm,
	})

	// 6. 调用服务层构建上下文
	buildReq := service.BuildContextRequest{
		SessionID:       req.SessionID,
		UserQuery:       req.UserQuery,
		MaxTokens:       req.MaxTokens,
		Strategy:        req.Strategy,
		IncludeSummary:  includeSummary,
		IncludeLongTerm: includeLongTerm,
		ShortTermWindow: req.ShortTermWindow,
	}

	result, err := h.contextService.BuildContext(ctx, buildReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "构建上下文失败", logger.Fields{
			"error":     err,
			"sessionId": req.SessionID,
			"userId":    userID,
			"tenantId":  tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 7. 转换为响应格式
	buildTime := time.Since(startTime).Milliseconds()
	resp := h.convertToContextResponse(result, buildTime)

	// 8. 记录响应日志
	h.logger.InfoContext(ctx, "构建上下文成功", logger.Fields{
		"sessionId":   req.SessionID,
		"totalTokens": resp.TotalTokens,
		"buildTime":   resp.BuildTime,
		"userId":      userID,
		"tenantId":    tenantID,
	})

	// 9. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleGetContextConfig 处理获取上下文配置请求
func (h *ContextHandler) HandleGetContextConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 验证会话ID格式
	if _, err := uuid.Parse(sessionID); err != nil {
		h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": sessionID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
		return
	}

	// 2. 从JWT token中获取用户ID和租户ID
	userID, ok := middleware.GetAuthUserID(ctx)
	if !ok || userID == "" {
		h.logger.Warn("缺少用户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少用户认证信息"))
		return
	}

	tenantID, ok := middleware.GetTenantID(ctx)
	if !ok || tenantID == "" {
		h.logger.Warn("缺少租户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少租户信息"))
		return
	}

	// 3. 记录请求日志
	h.logger.InfoContext(ctx, "收到获取上下文配置请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 4. 调用服务层获取配置
	config, err := h.contextService.GetContextConfig(ctx, sessionID)
	if err != nil {
		h.logger.ErrorContext(ctx, "获取上下文配置失败", logger.Fields{
			"error":     err,
			"sessionId": sessionID,
			"userId":    userID,
			"tenantId":  tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 5. 转换为响应格式
	resp := h.convertToConfigResponse(config)

	// 6. 记录响应日志
	h.logger.InfoContext(ctx, "获取上下文配置成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 7. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleUpdateContextConfig 处理更新上下文配置请求
func (h *ContextHandler) HandleUpdateContextConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionID(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 验证会话ID格式
	if _, err := uuid.Parse(sessionID); err != nil {
		h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": sessionID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
		return
	}

	// 2. 解析请求参数
	var req UpdateContextConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析更新上下文配置请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 3. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("更新上下文配置请求参数验证失败", logger.Fields{"errors": validationErrors})
		h.writeValidationErrorResponse(w, r, validationErrors)
		return
	}

	// 4. 从JWT token中获取用户ID和租户ID
	userID, ok := middleware.GetAuthUserID(ctx)
	if !ok || userID == "" {
		h.logger.Warn("缺少用户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少用户认证信息"))
		return
	}

	tenantID, ok := middleware.GetTenantID(ctx)
	if !ok || tenantID == "" {
		h.logger.Warn("缺少租户ID")
		h.writeErrorResponse(w, r, errors.NewUnauthorizedError("缺少租户信息"))
		return
	}

	// 5. 记录请求日志
	h.logger.InfoContext(ctx, "收到更新上下文配置请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 6. 先获取现有配置
	existingConfig, err := h.contextService.GetContextConfig(ctx, sessionID)
	if err != nil {
		h.logger.ErrorContext(ctx, "获取现有配置失败", logger.Fields{
			"error":     err,
			"sessionId": sessionID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 7. 更新配置字段
	if req.MaxTokens != nil {
		existingConfig.MaxTokens = *req.MaxTokens
	}
	if req.Strategy != nil {
		existingConfig.Strategy = *req.Strategy
	}
	if req.IncludeSummary != nil {
		existingConfig.IncludeSummary = *req.IncludeSummary
	}
	if req.IncludeLongTerm != nil {
		existingConfig.IncludeLongTerm = *req.IncludeLongTerm
	}
	if req.ShortTermWindow != nil {
		existingConfig.ShortTermWindow = *req.ShortTermWindow
	}

	// 8. 调用服务层更新配置
	err = h.contextService.UpdateContextConfig(ctx, sessionID, existingConfig)
	if err != nil {
		h.logger.ErrorContext(ctx, "更新上下文配置失败", logger.Fields{
			"error":     err,
			"sessionId": sessionID,
			"userId":    userID,
			"tenantId":  tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 9. 转换为响应格式
	resp := h.convertToConfigResponse(existingConfig)

	// 10. 记录响应日志
	h.logger.InfoContext(ctx, "更新上下文配置成功", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 11. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// extractSessionID 从URL路径中提取会话ID
// 路径格式: /api/v1/contexts/{sessionId}
func (h *ContextHandler) extractSessionID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "contexts" 后的部分
	for i, part := range parts {
		if part == "contexts" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// convertToContextResponse 转换服务层结果为响应格式
func (h *ContextHandler) convertToContextResponse(result *service.ContextResult, buildTime int64) *BuildContextResponse {
	resp := &BuildContextResponse{
		SessionID:         result.SessionID,
		ShortTermMessages: make([]flows.MessageContext, 0),
		LongTermMemories:  make([]flows.MemoryContext, 0),
		TotalTokens:       result.TotalTokens,
		Strategy:          result.Strategy,
		QualityScore:      result.QualityScore,
		BuildTime:         buildTime,
	}

	// 转换摘要
	if result.Summary != nil {
		resp.Summary = &flows.SummaryContext{
			Content:    result.Summary.Content,
			TokenCount: result.Summary.TokenCount,
			CreatedAt:  result.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Coverage:   "", // 可以根据需要添加覆盖范围描述
		}
	}

	// 转换长期记忆
	for _, mem := range result.LongTermMemories {
		resp.LongTermMemories = append(resp.LongTermMemories, flows.MemoryContext{
			ID:         mem.ID.String(),
			Content:    mem.Content,
			TokenCount: mem.TokenCount,
			Importance: mem.Importance,
			Similarity: 0.0, // 需要从向量检索结果中获取
			CreatedAt:  mem.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// 转换短期消息
	for _, msg := range result.ShortTermMessages {
		resp.ShortTermMessages = append(resp.ShortTermMessages, flows.MessageContext{
			ID:         msg.ID.String(),
			Role:       msg.Role,
			Content:    msg.Content,
			TokenCount: msg.Tokens,
			CreatedAt:  msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return resp
}

// convertToConfigResponse 转换配置实体为响应格式
func (h *ContextHandler) convertToConfigResponse(config *model.ConversationContext) *GetContextConfigResponse {
	resp := &GetContextConfigResponse{
		ID:              config.ID.String(),
		SessionID:       config.SessionID.String(),
		MaxTokens:       config.MaxTokens,
		Strategy:        config.Strategy,
		IncludeSummary:  config.IncludeSummary,
		IncludeLongTerm: config.IncludeLongTerm,
		ShortTermWindow: config.ShortTermWindow,
		TotalMessages:   config.TotalMessages,
		TotalTokensUsed: config.TotalTokensUsed,
		CreatedAt:       config.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       config.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if config.LastSummaryID != nil {
		summaryID := config.LastSummaryID.String()
		resp.LastSummaryID = &summaryID
	}

	if config.LastSummaryAt != nil {
		summaryAt := config.LastSummaryAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastSummaryAt = &summaryAt
	}

	return resp
}


// writeSuccessResponseWithContext 写入成功响应（带 Context）
func (h *ContextHandler) writeSuccessResponseWithContext(ctx context.Context, w http.ResponseWriter, data interface{}) {
	// 直接构建响应，避免泛型类型推断问题
	traceID := ""
	if ctx != nil {
		if id, ok := ctx.Value("traceId").(string); ok {
			traceID = id
		}
	}
	
	resp := map[string]interface{}{
		"code":    errors.CodeSuccess,
		"message": errors.MsgSuccess,
		"data":    data,
	}
	
	if traceID != "" {
		resp["traceId"] = traceID
	}
	
	h.writeJSONResponse(w, http.StatusOK, resp)
}

// writeErrorResponse 写入错误响应
func (h *ContextHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
	ctx := r.Context()
	resp := response.ErrorWithContext[any](ctx, appErr.Code, appErr.Message)

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
	}

	h.writeJSONResponse(w, statusCode, resp)
}

// writeValidationErrorResponse 写入验证错误响应
func (h *ContextHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
	ctx := r.Context()
	// 构建验证错误详情
	errorData := map[string]interface{}{
		"errors": validationErrors,
	}

	resp := response.ErrorWithDataContext(
		ctx,
		errors.CodeValidationError,
		errors.MsgValidationError,
		&errorData,
	)

	h.writeJSONResponse(w, http.StatusUnprocessableEntity, resp)
}

// writeJSONResponse 写入 JSON 响应
func (h *ContextHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
