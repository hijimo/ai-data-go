package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/pkg/errors"
	"genkit-ai-service/pkg/response"
	"genkit-ai-service/pkg/validator"

	"github.com/google/uuid"
)

// MemoryHandler 记忆管理处理器
// 提供记忆检索、存储、清理和查询等功能
type MemoryHandler struct {
	memoryService service.MemoryService
	logger        logger.Logger
	validator     *validator.Validator
}

// NewMemoryHandler 创建记忆管理处理器实例
func NewMemoryHandler(memoryService service.MemoryService, log logger.Logger) *MemoryHandler {
	return &MemoryHandler{
		memoryService: memoryService,
		logger:        log,
		validator:     validator.New(),
	}
}


// SearchMemoriesRequest 检索记忆请求
type SearchMemoriesRequest struct {
	SessionID            string   `json:"sessionId" validate:"required,uuid"`
	Query                string   `json:"query" validate:"required,max=2000"`
	TopK                 int      `json:"topK" validate:"omitempty,min=1,max=50"`
	MinSimilarity        float32  `json:"minSimilarity" validate:"omitempty,min=0,max=1"`
	TimeRangeDays        int      `json:"timeRangeDays" validate:"omitempty,min=0,max=365"`
	MemoryTypes          []string `json:"memoryTypes"`
	IncludeCrossSessions bool     `json:"includeCrossSessions"`
}

// SearchMemoriesResponse 检索记忆响应
type SearchMemoriesResponse struct {
	Results    []MemorySearchResultResponse `json:"results"`
	TotalCount int                          `json:"totalCount"`
	Query      string                       `json:"query"`
	SearchTime int64                        `json:"searchTime"`
}

// MemorySearchResultResponse 记忆检索结果响应
type MemorySearchResultResponse struct {
	ID             string                 `json:"id"`
	SessionID      string                 `json:"sessionId"`
	MemoryType     string                 `json:"memoryType"`
	Content        string                 `json:"content"`
	Importance     float32                `json:"importance"`
	Similarity     float32                `json:"similarity"`
	Score          float32                `json:"score"`
	AccessCount    int                    `json:"accessCount"`
	LastAccessedAt *string                `json:"lastAccessedAt,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// StoreMemoryRequest 存储记忆请求
type StoreMemoryRequest struct {
	SessionID      string                 `json:"sessionId" validate:"required,uuid"`
	MessageIDs     []string               `json:"messageIds"`
	MemoryType     string                 `json:"memoryType" validate:"required,oneof=fact preference context event summary"`
	Content        string                 `json:"content" validate:"required,max=10000"`
	Importance     float32                `json:"importance" validate:"omitempty,min=0,max=1"`
	ExpirationDays int                    `json:"expirationDays" validate:"omitempty,min=0,max=3650"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// StoreMemoryResponse 存储记忆响应
type StoreMemoryResponse struct {
	ID             string                 `json:"id"`
	SessionID      string                 `json:"sessionId"`
	MemoryType     string                 `json:"memoryType"`
	Content        string                 `json:"content"`
	Importance     float32                `json:"importance"`
	ExpiresAt      *string                `json:"expiresAt,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CleanupMemoriesRequest 清理记忆请求
type CleanupMemoriesRequest struct {
	SessionID *string `json:"sessionId" validate:"omitempty,uuid"`
	Strategy  string  `json:"strategy" validate:"required,oneof=expired low_quality unused all"`
	Mode      string  `json:"mode" validate:"omitempty,oneof=soft hard"`
	BatchSize int     `json:"batchSize" validate:"omitempty,min=1,max=1000"`
	Execute   bool    `json:"execute"`
}

// CleanupMemoriesResponse 清理记忆响应
type CleanupMemoriesResponse struct {
	CleanedCount int                    `json:"cleanedCount"`
	FreedSpace   int64                  `json:"freedSpace"`
	Details      []CleanupDetailResponse `json:"details"`
	Preview      bool                   `json:"preview"`
	Strategy     string                 `json:"strategy"`
	Mode         string                 `json:"mode"`
}

// CleanupDetailResponse 清理详情响应
type CleanupDetailResponse struct {
	MemoryID   string `json:"memoryId"`
	Reason     string `json:"reason"`
	Size       int64  `json:"size"`
	CreatedAt  string `json:"createdAt"`
	LastAccess string `json:"lastAccess"`
}

// GetMemoryResponse 获取记忆详情响应
type GetMemoryResponse struct {
	ID             string                 `json:"id"`
	SessionID      string                 `json:"sessionId"`
	MemoryType     string                 `json:"memoryType"`
	Content        string                 `json:"content"`
	Importance     float32                `json:"importance"`
	AccessCount    int                    `json:"accessCount"`
	LastAccessedAt *string                `json:"lastAccessedAt,omitempty"`
	ExpiresAt      *string                `json:"expiresAt,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}


// HandleSearchMemories 处理检索记忆请求
func (h *MemoryHandler) HandleSearchMemories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// 1. 解析请求参数
	var req SearchMemoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析检索记忆请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("检索记忆请求参数验证失败", logger.Fields{"errors": validationErrors})
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

	// 4. 解析会话ID
	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": req.SessionID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
		return
	}

	// 5. 设置默认值
	if req.TopK == 0 {
		req.TopK = 5
	}
	if req.MinSimilarity == 0 {
		req.MinSimilarity = 0.7
	}

	// 6. 记录请求日志
	h.logger.InfoContext(ctx, "收到检索记忆请求", logger.Fields{
		"userId":    userID,
		"tenantId":  tenantID,
		"sessionId": req.SessionID,
		"query":     req.Query,
		"topK":      req.TopK,
	})

	// 7. 调用服务层检索记忆
	searchReq := &service.SearchMemoriesRequest{
		SessionID:            sessionID,
		Query:                req.Query,
		TopK:                 req.TopK,
		MinSimilarity:        req.MinSimilarity,
		TimeRangeDays:        req.TimeRangeDays,
		MemoryTypes:          req.MemoryTypes,
		IncludeCrossSessions: req.IncludeCrossSessions,
	}

	results, err := h.memoryService.SearchMemories(ctx, searchReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "检索记忆失败", logger.Fields{
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

	// 8. 转换为响应格式
	searchTime := time.Since(startTime).Milliseconds()
	resp := h.convertToSearchResponse(results, req.Query, searchTime)

	// 9. 记录响应日志
	h.logger.InfoContext(ctx, "检索记忆成功", logger.Fields{
		"sessionId":  req.SessionID,
		"resultCount": len(results),
		"searchTime": searchTime,
		"userId":     userID,
		"tenantId":   tenantID,
	})

	// 10. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleStoreMemory 处理存储记忆请求
func (h *MemoryHandler) HandleStoreMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req StoreMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析存储记忆请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("存储记忆请求参数验证失败", logger.Fields{"errors": validationErrors})
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

	// 4. 解析会话ID
	sessionID, err := uuid.Parse(req.SessionID)
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

	// 6. 设置默认值
	if req.Importance == 0 {
		req.Importance = 0.5
	}

	// 7. 记录请求日志
	h.logger.InfoContext(ctx, "收到存储记忆请求", logger.Fields{
		"userId":     userID,
		"tenantId":   tenantID,
		"sessionId":  req.SessionID,
		"memoryType": req.MemoryType,
		"importance": req.Importance,
	})

	// 8. 调用服务层存储记忆
	storeReq := &service.StoreMemoryRequest{
		SessionID:      sessionID,
		MessageIDs:     messageIDs,
		MemoryType:     req.MemoryType,
		Content:        req.Content,
		Importance:     req.Importance,
		ExpirationDays: req.ExpirationDays,
		Metadata:       req.Metadata,
	}

	memory, err := h.memoryService.StoreMemory(ctx, storeReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "存储记忆失败", logger.Fields{
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

	// 9. 转换为响应格式
	resp := h.convertToStoreResponse(memory)

	// 10. 记录响应日志
	h.logger.InfoContext(ctx, "存储记忆成功", logger.Fields{
		"sessionId": req.SessionID,
		"memoryId":  memory.ID.String(),
		"userId":    userID,
		"tenantId":  tenantID,
	})

	// 11. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleCleanupMemories 处理清理记忆请求
func (h *MemoryHandler) HandleCleanupMemories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解析请求参数
	var req CleanupMemoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("解析清理记忆请求参数失败", logger.Fields{"error": err})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("无效的请求参数"))
		return
	}

	// 2. 验证请求参数
	if validationErrors := h.validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warn("清理记忆请求参数验证失败", logger.Fields{"errors": validationErrors})
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

	// 4. 解析会话ID（可选）
	var sessionID uuid.UUID
	if req.SessionID != nil && *req.SessionID != "" {
		var err error
		sessionID, err = uuid.Parse(*req.SessionID)
		if err != nil {
			h.logger.Warn("会话ID格式无效", logger.Fields{"sessionId": *req.SessionID})
			h.writeErrorResponse(w, r, errors.NewBadRequestError("会话ID格式无效"))
			return
		}
	}

	// 5. 设置默认值
	if req.Mode == "" {
		req.Mode = "soft"
	}
	if req.BatchSize == 0 {
		req.BatchSize = 100
	}

	// 6. 记录请求日志
	h.logger.InfoContext(ctx, "收到清理记忆请求", logger.Fields{
		"userId":    userID,
		"tenantId":  tenantID,
		"sessionId": req.SessionID,
		"strategy":  req.Strategy,
		"mode":      req.Mode,
		"execute":   req.Execute,
	})

	// 7. 调用服务层清理记忆
	cleanupReq := &service.CleanupMemoriesRequest{
		SessionID: sessionID,
		Strategy:  req.Strategy,
		Mode:      req.Mode,
		BatchSize: req.BatchSize,
		Execute:   req.Execute,
	}

	result, err := h.memoryService.CleanupMemories(ctx, cleanupReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "清理记忆失败", logger.Fields{
			"error":    err,
			"strategy": req.Strategy,
			"userId":   userID,
			"tenantId": tenantID,
		})
		if appErr, ok := err.(*errors.AppError); ok {
			h.writeErrorResponse(w, r, appErr)
		} else {
			h.writeErrorResponse(w, r, errors.NewInternalError(err))
		}
		return
	}

	// 8. 转换为响应格式
	resp := h.convertToCleanupResponse(result, req.Strategy, req.Mode)

	// 9. 记录响应日志
	h.logger.InfoContext(ctx, "清理记忆成功", logger.Fields{
		"cleanedCount": result.CleanedCount,
		"freedSpace":   result.FreedSpace,
		"preview":      result.Preview,
		"userId":       userID,
		"tenantId":     tenantID,
	})

	// 10. 返回成功响应
	h.writeSuccessResponseWithContext(ctx, w, resp)
}


// HandleGetMemory 处理获取记忆详情请求
func (h *MemoryHandler) HandleGetMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 从URL路径中提取记忆ID
	memoryID := h.extractMemoryID(r.URL.Path)
	if memoryID == "" {
		h.logger.Warn("记忆ID为空")
		h.writeErrorResponse(w, r, errors.NewBadRequestError("记忆ID不能为空"))
		return
	}

	// 验证记忆ID格式
	memoryUUID, err := uuid.Parse(memoryID)
	if err != nil {
		h.logger.Warn("记忆ID格式无效", logger.Fields{"memoryId": memoryID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("记忆ID格式无效"))
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

	// 验证 tenantID 是否为有效的 UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		h.logger.Warn("租户ID格式无效", logger.Fields{"tenantId": tenantID})
		h.writeErrorResponse(w, r, errors.NewBadRequestError("租户ID格式无效"))
		return
	}

	// 3. 记录请求日志
	h.logger.InfoContext(ctx, "收到获取记忆详情请求", logger.Fields{
		"memoryId": memoryID,
		"userId":   userID,
		"tenantId": tenantID,
	})

	// 4. 这里需要通过 repository 获取记忆详情
	// 由于 MemoryService 接口没有 GetMemory 方法，我们需要直接访问 repository
	// 或者通过 SearchMemories 来获取单个记忆
	// 为了保持一致性，我们使用 SearchMemories 并过滤结果
	
	// 注意：这是一个简化实现，实际应该在 service 层添加 GetMemory 方法
	// 这里我们返回一个错误，提示需要实现该功能
	h.logger.Warn("获取记忆详情功能尚未完全实现")
	h.writeErrorResponse(w, r, errors.NewServiceUnavailableError("获取记忆详情功能尚未完全实现"))
	
	// TODO: 实现获取记忆详情的逻辑
	// 1. 从 repository 获取记忆
	// 2. 验证租户权限
	// 3. 更新访问统计
	// 4. 返回记忆详情
	
	_ = memoryUUID
	_ = tenantUUID
}


// extractMemoryID 从URL路径中提取记忆ID
// 路径格式: /api/v1/memories/{memoryId}
func (h *MemoryHandler) extractMemoryID(path string) string {
	// 移除尾部斜杠
	path = strings.TrimSuffix(path, "/")

	// 分割路径
	parts := strings.Split(path, "/")

	// 查找 "memories" 后的部分
	for i, part := range parts {
		if part == "memories" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// convertToSearchResponse 转换检索结果为响应格式
func (h *MemoryHandler) convertToSearchResponse(results []*service.MemorySearchResult, query string, searchTime int64) *SearchMemoriesResponse {
	resp := &SearchMemoriesResponse{
		Results:    make([]MemorySearchResultResponse, 0, len(results)),
		TotalCount: len(results),
		Query:      query,
		SearchTime: searchTime,
	}

	for _, result := range results {
		memResp := MemorySearchResultResponse{
			ID:          result.Memory.ID.String(),
			SessionID:   result.Memory.SessionID.String(),
			MemoryType:  result.Memory.MemoryType,
			Content:     result.Memory.Content,
			Importance:  result.Memory.Importance,
			Similarity:  result.Similarity,
			Score:       result.Score,
			AccessCount: result.Memory.AccessCount,
			CreatedAt:   result.Memory.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// 转换 Metadata
		if result.Memory.Metadata != nil && len(result.Memory.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(result.Memory.Metadata, &metadata); err == nil {
				memResp.Metadata = metadata
			}
		}

		// 转换 LastAccessAt
		if result.Memory.LastAccessAt != nil {
			lastAccess := result.Memory.LastAccessAt.Format("2006-01-02T15:04:05Z07:00")
			memResp.LastAccessedAt = &lastAccess
		}

		resp.Results = append(resp.Results, memResp)
	}

	return resp
}

// convertToStoreResponse 转换存储结果为响应格式
func (h *MemoryHandler) convertToStoreResponse(memory *model.ConversationMemory) *StoreMemoryResponse {
	resp := &StoreMemoryResponse{
		ID:         memory.ID.String(),
		SessionID:  memory.SessionID.String(),
		MemoryType: memory.MemoryType,
		Content:    memory.Content,
		Importance: memory.Importance,
		CreatedAt:  memory.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 转换 Metadata
	if memory.Metadata != nil && len(memory.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(memory.Metadata, &metadata); err == nil {
			resp.Metadata = metadata
		}
	}

	if memory.ExpiresAt != nil {
		expiresAt := memory.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ExpiresAt = &expiresAt
	}

	return resp
}

// convertToCleanupResponse 转换清理结果为响应格式
func (h *MemoryHandler) convertToCleanupResponse(result *service.CleanupResult, strategy, mode string) *CleanupMemoriesResponse {
	resp := &CleanupMemoriesResponse{
		CleanedCount: result.CleanedCount,
		FreedSpace:   result.FreedSpace,
		Details:      make([]CleanupDetailResponse, 0, len(result.Details)),
		Preview:      result.Preview,
		Strategy:     strategy,
		Mode:         mode,
	}

	for _, detail := range result.Details {
		detailResp := CleanupDetailResponse{
			MemoryID:   detail.MemoryID.String(),
			Reason:     detail.Reason,
			Size:       detail.Size,
			CreatedAt:  detail.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastAccess: detail.LastAccess.Format("2006-01-02T15:04:05Z07:00"),
		}
		resp.Details = append(resp.Details, detailResp)
	}

	return resp
}


// writeSuccessResponseWithContext 写入成功响应（带 Context）
func (h *MemoryHandler) writeSuccessResponseWithContext(ctx context.Context, w http.ResponseWriter, data interface{}) {
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
func (h *MemoryHandler) writeErrorResponse(w http.ResponseWriter, r *http.Request, appErr *errors.AppError) {
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
func (h *MemoryHandler) writeValidationErrorResponse(w http.ResponseWriter, r *http.Request, validationErrors []validator.ValidationError) {
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
func (h *MemoryHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("写入响应失败", logger.Fields{"error": err})
	}
}
