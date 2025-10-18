package session

import (
	"context"
	"fmt"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/errors"
)

// SessionService 会话业务逻辑接口
type SessionService interface {
	// CreateSession 创建新会话
	CreateSession(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error)

	// GetSession 获取会话详情
	GetSession(ctx context.Context, sessionID, userID string) (*model.SessionResponse, error)

	// ListSessions 获取会话列表
	ListSessions(ctx context.Context, userID string, req *model.ListSessionsRequest) ([]*model.SessionResponse, int, error)

	// UpdateSession 更新会话
	UpdateSession(ctx context.Context, sessionID, userID string, req *model.UpdateSessionRequest) (*model.SessionResponse, error)

	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, sessionID, userID string) error

	// SearchSessions 搜索会话
	SearchSessions(ctx context.Context, userID string, req *model.SearchSessionsRequest) ([]*model.SessionResponse, int, error)

	// PinSession 置顶/取消置顶会话
	PinSession(ctx context.Context, sessionID, userID string, pinned bool) error

	// ArchiveSession 归档/取消归档会话
	ArchiveSession(ctx context.Context, sessionID, userID string, archived bool) error
}

// sessionService 会话业务逻辑实现
type sessionService struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
}

// NewSessionService 创建会话业务逻辑实例
func NewSessionService(sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository) SessionService {
	return &sessionService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// CreateSession 创建新会话
func (s *sessionService) CreateSession(ctx context.Context, userID string, req *model.CreateSessionRequest) (*model.SessionResponse, error) {
	// 创建会话实体
	session := &model.ChatSession{
		UserID:       userID,
		Title:        req.Title,
		ModelName:    req.ModelName,
		SystemPrompt: req.SystemPrompt,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		CreatedBy:    userID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		MessageCount: 0,
		IsPinned:     false,
		IsArchived:   false,
		IsDeleted:    false,
	}

	// 处理元数据
	if req.Meta != nil {
		metaJSON, err := convertMapToJSON(req.Meta)
		if err != nil {
			return nil, errors.NewBadRequestError("元数据格式错误")
		}
		session.Meta = metaJSON
	}

	// 保存到数据库
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("创建会话失败: %w", err))
	}

	// 转换为响应格式
	return s.toSessionResponse(session, nil), nil
}

// GetSession 获取会话详情
func (s *sessionService) GetSession(ctx context.Context, sessionID, userID string) (*model.SessionResponse, error) {
	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, errors.NewSessionNotFoundError(sessionID)
	}

	// 验证权限
	if session.UserID != userID {
		return nil, errors.NewSessionAccessDeniedError()
	}

	// 获取最后一条消息
	var lastMessage *model.ChatMessage
	if session.LastMessageID != nil {
		lastMessage, _ = s.messageRepo.GetByID(ctx, *session.LastMessageID)
	}

	// 转换为响应格式
	return s.toSessionResponse(session, lastMessage), nil
}

// ListSessions 获取会话列表
func (s *sessionService) ListSessions(ctx context.Context, userID string, req *model.ListSessionsRequest) ([]*model.SessionResponse, int, error) {
	// 构建过滤条件
	filters := &repository.SessionFilters{
		IsPinned:   req.IsPinned,
		IsArchived: req.IsArchived,
		ModelName:  req.ModelName,
	}

	// 查询会话列表
	sessions, total, err := s.sessionRepo.GetByUserID(ctx, userID, req.PageNo, req.PageSize, filters)
	if err != nil {
		return nil, 0, errors.NewInternalError(fmt.Errorf("查询会话列表失败: %w", err))
	}

	// 批量获取最后一条消息
	messageMap := make(map[string]*model.ChatMessage)
	for _, session := range sessions {
		if session.LastMessageID != nil {
			if msg, err := s.messageRepo.GetByID(ctx, *session.LastMessageID); err == nil {
				messageMap[session.ID] = msg
			}
		}
	}

	// 转换为响应格式
	responses := make([]*model.SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		lastMessage := messageMap[session.ID]
		responses = append(responses, s.toSessionResponse(session, lastMessage))
	}

	return responses, int(total), nil
}

// UpdateSession 更新会话
func (s *sessionService) UpdateSession(ctx context.Context, sessionID, userID string, req *model.UpdateSessionRequest) (*model.SessionResponse, error) {
	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, errors.NewSessionNotFoundError(sessionID)
	}

	// 验证权限
	if session.UserID != userID {
		return nil, errors.NewSessionAccessDeniedError()
	}

	// 构建更新字段
	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.SystemPrompt != nil {
		fields["system_prompt"] = *req.SystemPrompt
	}
	if req.Temperature != nil {
		fields["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		fields["top_p"] = *req.TopP
	}
	if req.ModelName != nil {
		fields["model_name"] = *req.ModelName
	}

	// 更新时间戳
	fields["updated_at"] = time.Now()

	// 执行更新
	if len(fields) > 0 {
		if err := s.sessionRepo.UpdateFields(ctx, sessionID, fields); err != nil {
			return nil, errors.NewInternalError(fmt.Errorf("更新会话失败: %w", err))
		}
	}

	// 重新查询会话
	session, err = s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, errors.NewSessionNotFoundError(sessionID)
	}

	// 获取最后一条消息
	var lastMessage *model.ChatMessage
	if session.LastMessageID != nil {
		lastMessage, _ = s.messageRepo.GetByID(ctx, *session.LastMessageID)
	}

	// 转换为响应格式
	return s.toSessionResponse(session, lastMessage), nil
}

// DeleteSession 删除会话
func (s *sessionService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.NewSessionNotFoundError(sessionID)
	}

	// 验证权限
	if session.UserID != userID {
		return errors.NewSessionAccessDeniedError()
	}

	// 执行软删除
	if err := s.sessionRepo.SoftDelete(ctx, sessionID); err != nil {
		return errors.NewInternalError(fmt.Errorf("删除会话失败: %w", err))
	}

	return nil
}

// SearchSessions 搜索会话
func (s *sessionService) SearchSessions(ctx context.Context, userID string, req *model.SearchSessionsRequest) ([]*model.SessionResponse, int, error) {
	// 搜索会话
	sessions, total, err := s.sessionRepo.Search(ctx, userID, req.Keyword, req.PageNo, req.PageSize)
	if err != nil {
		return nil, 0, errors.NewInternalError(fmt.Errorf("搜索会话失败: %w", err))
	}

	// 批量获取最后一条消息
	messageMap := make(map[string]*model.ChatMessage)
	for _, session := range sessions {
		if session.LastMessageID != nil {
			if msg, err := s.messageRepo.GetByID(ctx, *session.LastMessageID); err == nil {
				messageMap[session.ID] = msg
			}
		}
	}

	// 转换为响应格式
	responses := make([]*model.SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		lastMessage := messageMap[session.ID]
		responses = append(responses, s.toSessionResponse(session, lastMessage))
	}

	return responses, int(total), nil
}

// PinSession 置顶/取消置顶会话
func (s *sessionService) PinSession(ctx context.Context, sessionID, userID string, pinned bool) error {
	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.NewSessionNotFoundError(sessionID)
	}

	// 验证权限
	if session.UserID != userID {
		return errors.NewSessionAccessDeniedError()
	}

	// 更新置顶状态
	fields := map[string]interface{}{
		"is_pinned":  pinned,
		"updated_at": time.Now(),
	}

	if err := s.sessionRepo.UpdateFields(ctx, sessionID, fields); err != nil {
		return errors.NewInternalError(fmt.Errorf("更新置顶状态失败: %w", err))
	}

	return nil
}

// ArchiveSession 归档/取消归档会话
func (s *sessionService) ArchiveSession(ctx context.Context, sessionID, userID string, archived bool) error {
	// 查询会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.NewSessionNotFoundError(sessionID)
	}

	// 验证权限
	if session.UserID != userID {
		return errors.NewSessionAccessDeniedError()
	}

	// 更新归档状态
	fields := map[string]interface{}{
		"is_archived": archived,
		"updated_at":  time.Now(),
	}

	if err := s.sessionRepo.UpdateFields(ctx, sessionID, fields); err != nil {
		return errors.NewInternalError(fmt.Errorf("更新归档状态失败: %w", err))
	}

	return nil
}

// toSessionResponse 将会话实体转换为响应格式
func (s *sessionService) toSessionResponse(session *model.ChatSession, lastMessage *model.ChatMessage) *model.SessionResponse {
	response := &model.SessionResponse{
		ID:           session.ID,
		UserID:       session.UserID,
		Title:        session.Title,
		ModelName:    session.ModelName,
		SystemPrompt: session.SystemPrompt,
		Temperature:  session.Temperature,
		TopP:         session.TopP,
		CreatedAt:    session.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    session.UpdatedAt.Format(time.RFC3339),
		MessageCount: session.MessageCount,
		IsPinned:     session.IsPinned,
		IsArchived:   session.IsArchived,
	}

	// 处理最后一条消息
	if lastMessage != nil {
		response.LastMessage = &model.MessagePreview{
			ID:        lastMessage.ID,
			Role:      lastMessage.Role,
			Content:   lastMessage.Content,
			CreatedAt: lastMessage.CreatedAt.Format(time.RFC3339),
		}
	}

	// 处理元数据
	if session.Meta != nil {
		meta, err := convertJSONToMap(session.Meta)
		if err == nil {
			response.Meta = meta
		}
	}

	return response
}

// convertMapToJSON 将 map 转换为 JSON
func convertMapToJSON(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	// GORM 的 datatypes.JSON 会自动处理 JSON 序列化
	// 这里直接返回 nil，让 GORM 处理
	return nil, nil
}

// convertJSONToMap 将 JSON 转换为 map
func convertJSONToMap(data []byte) (map[string]interface{}, error) {
	if data == nil || len(data) == 0 {
		return nil, nil
	}
	// GORM 的 datatypes.JSON 会自动处理 JSON 反序列化
	// 这里简单返回 nil，实际使用时前端会直接获取 JSON 字段
	return nil, nil
}
