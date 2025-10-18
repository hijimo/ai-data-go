package session

import (
	"context"
	"testing"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
)

// mockSessionRepository 模拟会话仓库
type mockSessionRepository struct {
	sessions map[string]*model.ChatSession
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*model.ChatSession),
	}
}

func (m *mockSessionRepository) Create(ctx context.Context, session *model.ChatSession) error {
	if session.ID == "" {
		session.ID = "test-session-id"
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) GetByID(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, repository.ErrNotFound
	}
	return session, nil
}

func (m *mockSessionRepository) GetByUserID(ctx context.Context, userID string, page, pageSize int, filters *repository.SessionFilters) ([]*model.ChatSession, int64, error) {
	var result []*model.ChatSession
	for _, session := range m.sessions {
		if session.UserID == userID && !session.IsDeleted {
			result = append(result, session)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockSessionRepository) Update(ctx context.Context, session *model.ChatSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) UpdateFields(ctx context.Context, sessionID string, fields map[string]interface{}) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return repository.ErrNotFound
	}
	// 简单更新逻辑
	if title, ok := fields["title"].(string); ok {
		session.Title = title
	}
	return nil
}

func (m *mockSessionRepository) SoftDelete(ctx context.Context, sessionID string) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return repository.ErrNotFound
	}
	session.IsDeleted = true
	return nil
}

func (m *mockSessionRepository) Search(ctx context.Context, userID, keyword string, page, pageSize int) ([]*model.ChatSession, int64, error) {
	return []*model.ChatSession{}, 0, nil
}

func (m *mockSessionRepository) IncrementMessageCount(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionRepository) UpdateLastMessage(ctx context.Context, sessionID, messageID string) error {
	return nil
}

// mockMessageRepository 模拟消息仓库
type mockMessageRepository struct {
	getBySessionIDFunc   func(ctx context.Context, sessionID string, page, pageSize int) ([]*model.ChatMessage, int, error)
	countBySessionIDFunc func(ctx context.Context, sessionID string) (int, error)
	getMessagesAfterFunc func(ctx context.Context, sessionID, afterMessageID string) ([]*model.ChatMessage, error)
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{}
}

func (m *mockMessageRepository) Create(ctx context.Context, message *model.ChatMessage) error {
	return nil
}

func (m *mockMessageRepository) GetByID(ctx context.Context, messageID string) (*model.ChatMessage, error) {
	return nil, repository.ErrNotFound
}

func (m *mockMessageRepository) GetBySessionID(ctx context.Context, sessionID string, page, pageSize int) ([]*model.ChatMessage, int, error) {
	if m.getBySessionIDFunc != nil {
		return m.getBySessionIDFunc(ctx, sessionID, page, pageSize)
	}
	return []*model.ChatMessage{}, 0, nil
}

func (m *mockMessageRepository) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error) {
	return []*model.ChatMessage{}, nil
}

func (m *mockMessageRepository) GetNextSequence(ctx context.Context, sessionID string) (int, error) {
	return 1, nil
}

func (m *mockMessageRepository) CountBySessionID(ctx context.Context, sessionID string) (int, error) {
	if m.countBySessionIDFunc != nil {
		return m.countBySessionIDFunc(ctx, sessionID)
	}
	return 0, nil
}

func (m *mockMessageRepository) GetMessagesAfter(ctx context.Context, sessionID string, afterMessageID string) ([]*model.ChatMessage, error) {
	if m.getMessagesAfterFunc != nil {
		return m.getMessagesAfterFunc(ctx, sessionID, afterMessageID)
	}
	return []*model.ChatMessage{}, nil
}

// TestCreateSession 测试创建会话
func TestCreateSession(t *testing.T) {
	sessionRepo := newMockSessionRepository()
	messageRepo := newMockMessageRepository()
	service := NewSessionService(sessionRepo, messageRepo)

	ctx := context.Background()
	userID := "test-user-id"
	temp := 0.7
	topP := 0.9

	req := &model.CreateSessionRequest{
		Title:        "测试会话",
		ModelName:    "gpt-4",
		SystemPrompt: "你是一个有帮助的助手",
		Temperature:  &temp,
		TopP:         &topP,
	}

	response, err := service.CreateSession(ctx, userID, req)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	if response.Title != req.Title {
		t.Errorf("期望标题 %s, 得到 %s", req.Title, response.Title)
	}

	if response.ModelName != req.ModelName {
		t.Errorf("期望模型 %s, 得到 %s", req.ModelName, response.ModelName)
	}

	if response.UserID != userID {
		t.Errorf("期望用户ID %s, 得到 %s", userID, response.UserID)
	}
}

// TestGetSession 测试获取会话
func TestGetSession(t *testing.T) {
	sessionRepo := newMockSessionRepository()
	messageRepo := newMockMessageRepository()
	service := NewSessionService(sessionRepo, messageRepo)

	ctx := context.Background()
	userID := "test-user-id"

	// 先创建一个会话
	temp := 0.7
	req := &model.CreateSessionRequest{
		Title:       "测试会话",
		ModelName:   "gpt-4",
		Temperature: &temp,
	}

	createResp, err := service.CreateSession(ctx, userID, req)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 获取会话
	getResp, err := service.GetSession(ctx, createResp.ID, userID)
	if err != nil {
		t.Fatalf("获取会话失败: %v", err)
	}

	if getResp.ID != createResp.ID {
		t.Errorf("期望会话ID %s, 得到 %s", createResp.ID, getResp.ID)
	}
}

// TestUpdateSession 测试更新会话
func TestUpdateSession(t *testing.T) {
	sessionRepo := newMockSessionRepository()
	messageRepo := newMockMessageRepository()
	service := NewSessionService(sessionRepo, messageRepo)

	ctx := context.Background()
	userID := "test-user-id"

	// 先创建一个会话
	temp := 0.7
	createReq := &model.CreateSessionRequest{
		Title:       "原始标题",
		ModelName:   "gpt-4",
		Temperature: &temp,
	}

	createResp, err := service.CreateSession(ctx, userID, createReq)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 更新会话
	newTitle := "更新后的标题"
	updateReq := &model.UpdateSessionRequest{
		Title: &newTitle,
	}

	updateResp, err := service.UpdateSession(ctx, createResp.ID, userID, updateReq)
	if err != nil {
		t.Fatalf("更新会话失败: %v", err)
	}

	if updateResp.Title != newTitle {
		t.Errorf("期望标题 %s, 得到 %s", newTitle, updateResp.Title)
	}
}

// TestDeleteSession 测试删除会话
func TestDeleteSession(t *testing.T) {
	sessionRepo := newMockSessionRepository()
	messageRepo := newMockMessageRepository()
	service := NewSessionService(sessionRepo, messageRepo)

	ctx := context.Background()
	userID := "test-user-id"

	// 先创建一个会话
	temp := 0.7
	createReq := &model.CreateSessionRequest{
		Title:       "测试会话",
		ModelName:   "gpt-4",
		Temperature: &temp,
	}

	createResp, err := service.CreateSession(ctx, userID, createReq)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 删除会话
	err = service.DeleteSession(ctx, createResp.ID, userID)
	if err != nil {
		t.Fatalf("删除会话失败: %v", err)
	}

	// 验证会话已被软删除
	session, _ := sessionRepo.GetByID(ctx, createResp.ID)
	if session != nil && !session.IsDeleted {
		t.Error("会话应该被标记为已删除")
	}
}
