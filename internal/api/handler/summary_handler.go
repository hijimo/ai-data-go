package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service/session"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"

	"github.com/google/uuid"
)

// SummaryHandler 摘要管理处理器
// 提供摘要生成、查询和触发检查等功能
type SummaryHandler struct {
	summaryService session.SummaryService
	logger         logger.Logger
	validator      *validator.Validator
}

// NewSummaryHandler 创建摘要管理处理器实例
func NewSummaryHandler(summaryService session.SummaryService, log logger.Logger) *SummaryHandler {
	return &SummaryHandler{
		summaryService: summaryService,
		logger:         log,
		validator:      validator.New(),
	}
}

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	SessionID       string   `json:"sessionId" validate:"required,uuid"`
	MessageIDs      []string `json:"messageIds"`
	StartMessageID  *string  `json:"startMessageId" validate:"omitempty,uuid"`
	EndMessageID    *string  `json:"endMessageId" validate:"omitempty,uuid"`
	PreviousSummary string   `json:"previousSummary"`
	SummaryType     string   `json:"summaryType" validate:"omitempty,oneof=incremental full"`
	TargetLength    int      `json:"targetLength" validate:"omitempty,min=100,max=2000"`
}

// GenerateSummaryResponse 生成摘要响应
type GenerateSummaryResponse struct {
	ID              string                 `json:"id"`
	SessionID       string                 `json:"sessionId"`
	Content         string                 `json:"content"`
	SummaryType     string                 `json:"summaryType"`
	MessageCount    int                    `json:"messageCount"`
	TokenCount      int                    `json:"tokenCount"`
	QualityScore    float64                `json:"qualityScore"`
	KeyPoints       []string               `json:"keyPoints,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GetSummaryResponse 获取摘要详情响应
type GetSummaryResponse struct {
	ID              string                 `json:"id"`
	SessionID       string                 `json:"sessionId"`
	Content         string                 `json:"content"`
	SummaryType     string                 `json:"summaryType"`
	MessageCount    int                    `json:"messageCount"`
	TokenCount      int                    `json:"tokenCount"`
	QualityScore    float64                `json:"qualityScore"`
	KeyPoints       []string               `json:"keyPoints,omitempty"`
	StartMessageID  *string                `json:"startMessageId,omitempty"`
	EndMessageID    *string                `json:"endMessageId,omitempty"`
	PreviousSummary *string                `json:"previousSummaryId,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
	UpdatedAt       string                 `json:"updatedAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ListSummariesResponse 获取摘要列表响应
type ListSummariesResponse struct {
	Summaries  []SummaryItemResponse `json:"summaries"`
	TotalCount int                   `json:"totalCount"`
	SessionID  string                `json:"sessionId"`
}

// SummaryItemResponse 摘要列表项响应
type SummaryItemResponse struct {
	ID           string  `json:"id"`
	SessionID    string  `json:"sessionId"`
	Content      string  `json:"content"`
	SummaryType  string  `json:"summaryType"`
	MessageCount int     `json:"messageCount"`
	TokenCount   int     `json:"tokenCount"`
	QualityScore float64 `json:"qualityScore"`
	CreatedAt    string  `json:"createdAt"`
}

// CheckTriggerResponse 检查摘要触发条件响应
type CheckTriggerResponse struct {
	ShouldSummarize      bool     `json:"shouldSummarize"`
	TriggerReason        string   `json:"triggerReason"`
	MessageCount         int      `json:"messageCount"`
	EstimatedTokenSaving int      `json:"estimatedTokenSaving"`
	Urgency              float64  `json:"urgency"`
	RecommendedType      string   `json:"recommendedType"`
	MessageIDs           []string `json:"messageIds,omitempty"`
}


// HandleGenerateSummary 处理生成摘要请求
func (h *SummaryHandler) HandleGenerateSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req GenerateSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析生成摘要请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("生成摘要请求参数验证失败", logger.Fields{"errors": validationErrors})
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
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 4. 解析会话ID
	sessionUUID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": req.SessionID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
		return
	}

	// 5. 解析消息ID列表
	var messageIDs []uuid.UUID
	for _, msgIDStr := range req.MessageIDs {
		msgID, err := uuid.Parse(msgIDStr)
		if err != nil {
			h.logger.Warn("消息ID格式无效", logger.Fields{"messageId": msgIDStr})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("消息ID格式无效"))
			return
		}
		messageIDs = append(messageIDs, msgID)
	}

	// 6. 解析起始和结束消息ID
	var startMessageID, endMessageID *uuid.UUID
	if req.StartMessageID != nil && *req.StartMessageID != "" {
		startID, err := uuid.Parse(*req.StartMessageID)
		if err != nil {
			h.logger.Warn("起始消息ID格式无效", logger.Fields{"startMessageId": *req.StartMessageID})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("起始消息ID格式无效"))
			return
		}
		startMessageID = &startID
	}
	if req.EndMessageID != nil && *req.EndMessageID != "" {
		endID, err := uuid.Parse(*req.EndMessageID)
		if err != nil {
			h.logger.Warn("结束消息ID格式无效", logger.Fields{"endMessageId": *req.EndMessageID})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("结束消息ID格式无效"))
			return
		}
		endMessageID = &endID
	}

	// 7. 设置默认值
	if req.SummaryType == "" {
		req.SummaryType = "incremental"
	}
	if req.TargetLength == 0 {
		req.TargetLength = 500
	}

	// 8. 记录请求日志
	h.logger.InfoContext(ctx, "收到生成摘要请求", logger.Fields{
		"userId":      userID,
		"tenantId":    tenantID,
		"sessionId":   req.SessionID,
		"summaryType": req.SummaryType,
		"messageCount": len(messageIDs),
	})

	// 9. 调用服务层生成摘要
	generateReq := &session.GenerateSummaryRequest{
		TenantID:        tenantUUID,
		SessionID:       sessionUUID,
		MessageIDs:      messageIDs,
		StartMessageID:  startMessageID,
		EndMessageID:    endMessageID,
		PreviousSummary: req.PreviousSummary,
		SummaryType:     req.SummaryType,
		TargetLength:    req.TargetLength,
	}

	summary, err := h.summaryService.GenerateSummary(ctx, generateReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "生成摘要失败", logger.Fields{
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

	// 10. 转换为响应格式
	resp := h.convertToGenerateResponse(summary)

	// 11. 记录响应日志
	h.logger.InfoContext(ctx, "生成摘要成功", logger.Fields{
		"sessionId":  req.SessionID,
		"summaryId":  summary.ID.String(),
		"tokenCount": summary.TokenCount,
		"userId":     userID,
		"tenantId":   tenantID,
	})

	// 12. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleGetSummary 处理获取摘要详情请求
func (h *SummaryHandler) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取摘要ID
	summaryID := h.extractSummaryID(r.URL.Path)
	if summaryID == "" {
		h.logger.Warn("摘要ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("摘要ID不能为空"))
		return
	}

	// 验证摘要ID格式
	summaryUUID, err := uuid.Parse(summaryID)
	if err != nil {
		h.logger.Warn("摘要ID格式无效", logger.Fields{"summaryId": summaryID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("摘要ID格式无效"))
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

	// 验证租户ID格式
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.InfoContext(ctx, "收到获取摘要详情请求", logger.Fields{
		"summaryId": summaryID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 4. 调用服务层获取摘要
	summary, err := h.summaryService.GetSummary(ctx, tenantUUID, summaryUUID)
	if err != nil {
		h.logger.ErrorContext(ctx, "获取摘要详情失败", logger.Fields{
			"error":     err,
			"summaryId": summaryID,
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
	resp := h.convertToGetResponse(summary)

	// 6. 记录响应日志
	h.logger.InfoContext(ctx, "获取摘要详情成功", logger.Fields{
		"summaryId": summaryID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 7. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleListSummaries 处理获取摘要列表请求
func (h *SummaryHandler) HandleListSummaries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionIDFromPath(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 验证会话ID格式
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
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

	// 验证租户ID格式
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 3. 获取查询参数（limit）
	limit := 10 // 默认值
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := parseInt(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// 4. 记录请求日志
	h.logger.InfoContext(ctx, "收到获取摘要列表请求", logger.Fields{
		"sessionId": sessionID,
		"limit":     limit,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 5. 调用服务层获取摘要列表
	summaries, err := h.summaryService.ListSummaries(ctx, tenantUUID, sessionUUID, limit)
	if err != nil {
		h.logger.ErrorContext(ctx, "获取摘要列表失败", logger.Fields{
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

	// 6. 转换为响应格式
	resp := h.convertToListResponse(summaries, sessionID)

	// 7. 记录响应日志
	h.logger.InfoContext(ctx, "获取摘要列表成功", logger.Fields{
		"sessionId": sessionID,
		"count":     len(summaries),
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 8. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleCheckTrigger 处理检查摘要触发条件请求
func (h *SummaryHandler) HandleCheckTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取会话ID
	sessionID := h.extractSessionIDFromPath(r.URL.Path)
	if sessionID == "" {
		h.logger.Warn("会话ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID不能为空"))
		return
	}

	// 验证会话ID格式
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
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

	// 验证租户ID格式
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.InfoContext(ctx, "收到检查摘要触发条件请求", logger.Fields{
		"sessionId": sessionID,
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 4. 调用服务层检查触发条件
	result, err := h.summaryService.CheckSummaryTrigger(ctx, tenantUUID, sessionUUID)
	if err != nil {
		h.logger.ErrorContext(ctx, "检查摘要触发条件失败", logger.Fields{
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
	resp := h.convertToCheckTriggerResponse(result)

	// 6. 记录响应日志
	h.logger.InfoContext(ctx, "检查摘要触发条件成功", logger.Fields{
		"sessionId":       sessionID,
		"shouldSummarize": result.ShouldSummarize,
		"triggerReason":   result.TriggerReason,
		"userId":          userID,
		"tenantId":        tenantID,
	})

	// 7. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// extractSummaryID 从URL路径中提取摘要ID
// 路径格式: /api/v1/summaries/{summaryId}
func (h *SummaryHandler) extractSummaryID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "summaries" 后的部分
	for i, part := range parts {
		if part == "summaries" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// extractSessionIDFromPath 从URL路径中提取会话ID
// 路径格式: /api/v1/sessions/{sessionId}/summaries 或 /api/v1/sessions/{sessionId}/summaries/check-trigger
func (h *SummaryHandler) extractSessionIDFromPath(path string) string {
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

// parseInt 解析整数字符串
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// convertToGenerateResponse 转换生成摘要结果为响应格式
func (h *SummaryHandler) convertToGenerateResponse(summary *model.ConversationSummary) *GenerateSummaryResponse {
	resp := &GenerateSummaryResponse{
		ID:           summary.ID.String(),
		SessionID:    summary.SessionID.String(),
		Content:      summary.Content,
		SummaryType:  summary.SummaryType,
		MessageCount: summary.MessageCount,
		TokenCount:   summary.TokenCount,
		CreatedAt:    summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 转换质量评分
	if summary.QualityScore != nil {
		resp.QualityScore = *summary.QualityScore
	}

	// 转换关键主题
	if summary.KeyTopics != nil && len(summary.KeyTopics) > 0 {
		resp.KeyPoints = summary.KeyTopics
	}

	return resp
}

// convertToGetResponse 转换摘要实体为详情响应格式
func (h *SummaryHandler) convertToGetResponse(summary *model.ConversationSummary) *GetSummaryResponse {
	resp := &GetSummaryResponse{
		ID:           summary.ID.String(),
		SessionID:    summary.SessionID.String(),
		Content:      summary.Content,
		SummaryType:  summary.SummaryType,
		MessageCount: summary.MessageCount,
		TokenCount:   summary.TokenCount,
		CreatedAt:    summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    summary.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 转换质量评分
	if summary.QualityScore != nil {
		resp.QualityScore = *summary.QualityScore
	}

	// 转换关键主题
	if summary.KeyTopics != nil && len(summary.KeyTopics) > 0 {
		resp.KeyPoints = summary.KeyTopics
	}

	// 转换起始和结束消息ID
	if summary.StartMessageID != nil {
		startID := summary.StartMessageID.String()
		resp.StartMessageID = &startID
	}
	if summary.EndMessageID != nil {
		endID := summary.EndMessageID.String()
		resp.EndMessageID = &endID
	}

	// 转换前一个摘要ID
	if summary.PreviousSummaryID != nil {
		prevID := summary.PreviousSummaryID.String()
		resp.PreviousSummary = &prevID
	}

	return resp
}

// convertToListResponse 转换摘要列表为响应格式
func (h *SummaryHandler) convertToListResponse(summaries []*model.ConversationSummary, sessionID string) *ListSummariesResponse {
	resp := &ListSummariesResponse{
		Summaries:  make([]SummaryItemResponse, 0, len(summaries)),
		TotalCount: len(summaries),
		SessionID:  sessionID,
	}

	for _, summary := range summaries {
		item := SummaryItemResponse{
			ID:           summary.ID.String(),
			SessionID:    summary.SessionID.String(),
			Content:      summary.Content,
			SummaryType:  summary.SummaryType,
			MessageCount: summary.MessageCount,
			TokenCount:   summary.TokenCount,
			CreatedAt:    summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		
		// 转换质量评分
		if summary.QualityScore != nil {
			item.QualityScore = *summary.QualityScore
		}
		
		resp.Summaries = append(resp.Summaries, item)
	}

	return resp
}

// convertToCheckTriggerResponse 转换触发检查结果为响应格式
func (h *SummaryHandler) convertToCheckTriggerResponse(result *session.SummaryTriggerResult) *CheckTriggerResponse {
	resp := &CheckTriggerResponse{
		ShouldSummarize:      result.ShouldSummarize,
		TriggerReason:        result.TriggerReason,
		MessageCount:         result.MessageCount,
		EstimatedTokenSaving: result.EstimatedTokenSaving,
		Urgency:              result.Urgency,
		RecommendedType:      result.RecommendedType,
	}

	// 转换消息ID列表
	if len(result.MessageIDs) > 0 {
		resp.MessageIDs = make([]string, 0, len(result.MessageIDs))
		for _, msgID := range result.MessageIDs {
			resp.MessageIDs = append(resp.MessageIDs, msgID.String())
		}
	}

	return resp
}


// writeSuccessResponseWithContext 写入成功响应（带 Context）
func (h *SummaryHandler) writeSuccessResponseWithContext(ctx context.Context, w http.ResponseWriter, data interface{}) {
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
func (h *SummaryHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
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
func (h *SummaryHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
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
func (h *SummaryHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
