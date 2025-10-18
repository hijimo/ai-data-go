package session

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"

	"gorm.io/gorm"
)

// testMessageRepository 测试用消息仓库
type testMessageRepository struct {
	messages      map[string]*model.ChatMessage
	returnError   error
	nextSequence  int
}

func newTestMessageRepository() *testMessageRepository {
	return &testMessageRepository{
		messages:     make(map[string]*model.ChatMessage),
		nextSequence: 1,
	}
}

func (m *testMessageRepository) Create(ctx context.Context, message *model.ChatMessage) error {
	if m.returnError != nil {
		return m.returnError
	}
	if message.ID == "" {
		message.ID = "test-msg-id"
	}
	m.messages[message.ID] = message
	return nil
}

func (m *testMessageRepository) GetByID(ctx context.Context, messageID string) (*model.ChatMessage, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	msg, exists := m.messages[messageID]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return msg, nil
}

func (m *testMessageRepository) GetBySessionID(ctx context.Context, sessionID string, page, pageSize int) ([]*model.ChatMessage, int, error) {
	if m.returnError != nil {
		return nil, 0, m.returnError
	}
	var result []*model.ChatMessage
	for _, msg := range m.messages {
		if msg.SessionID == sessionID {
			result = append(result, msg)
		}
	}
	return result, len(result), nil
}

func (m *testMessageRepository) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error) {
	return []*model.ChatMessage{}, nil
}

func (m *testMessageRepository) GetNextSequence(ctx context.Context, sessionID string) (int, error) {
	if m.returnError != nil {
		return 0, m.returnError
	}
	seq := m.nextSequence
	m.nextSequence++
	return seq, nil
}

func (m *testMessageRepository) CountBySessionID(ctx context.Context, sessionID string) (int, error) {
	return 0, nil
}

func (m *testMessageRepository) GetMessagesAfter(ctx context.Context, sessionID string, afterMessageID string) ([]*model.ChatMessage, error) {
	return []*model.ChatMessage{}, nil
}

// testAIService 测试用AI服务
type testAIService struct {
	response    *model.ChatResponse
	returnError error
	abortError  error
}

func newTestAIService() *testAIService {
	return &testAIService{
		response: &model.ChatResponse{
			SessionID: "test-session",
			Message:   "AI回复",
			Model:     "test-model",
			Usage: &model.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		},
	}
}

func (m *testAIService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return m.response, nil
}

func (m *testAIService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	return nil, nil
}

func (m *testAIService) AbortChat(ctx context.Context, sessionID string) error {
	return m.abortError
}

// TestGetMessageByID 测试获取单条消息
func TestGetMessageByID(t *testing.T) {
	ctx := context.Background()

	t.Run("成功获取消息", func(t *testing.T) {
		// 准备测试数据
		messageID := "msg-123"
		userID := "user-123"
		sessionID := "session-123"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		// 创建会话
		session := &model.ChatSession{
			ID:     sessionID,
			UserID: userID,
			Title:  "测试会话",
		}
		sessionRepo.sessions[sessionID] = session

		// 创建消息
		message := &model.ChatMessage{
			ID:        messageID,
			SessionID: sessionID,
			Role:      "user",
			Content:   "测试消息",
			Sequence:  1,
			CreatedAt: time.Now(),
		}
		messageRepo.messages[messageID] = message

		// 创建服务
		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		// 执行测试
		result, err := service.GetMessageByID(ctx, messageID, userID)

		// 验证结果
		if err != nil {
			t.Fatalf("获取消息失败: %v", err)
		}
		if result.ID != messageID {
			t.Errorf("期望消息ID %s, 得到 %s", messageID, result.ID)
		}
		if result.SessionID != sessionID {
			t.Errorf("期望会话ID %s, 得到 %s", sessionID, result.SessionID)
		}
		if result.Role != "user" {
			t.Errorf("期望角色 user, 得到 %s", result.Role)
		}
	})

	t.Run("消息不存在", func(t *testing.T) {
		messageID := "msg-not-exist"
		userID := "user-123"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		result, err := service.GetMessageByID(ctx, messageID, userID)

		if err == nil {
			t.Error("期望返回错误，但没有错误")
		}
		if result != nil {
			t.Error("期望结果为nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Error("期望返回AppError类型")
		}
		if appErr.Code != errors.CodeMessageNotFound {
			t.Errorf("期望错误码 %d, 得到 %d", errors.CodeMessageNotFound, appErr.Code)
		}
	})

	t.Run("无权访问消息", func(t *testing.T) {
		messageID := "msg-123"
		userID := "user-123"
		sessionID := "session-123"
		otherUserID := "user-456"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		// 创建会话（属于其他用户）
		session := &model.ChatSession{
			ID:     sessionID,
			UserID: otherUserID,
			Title:  "测试会话",
		}
		sessionRepo.sessions[sessionID] = session

		// 创建消息
		message := &model.ChatMessage{
			ID:        messageID,
			SessionID: sessionID,
			Role:      "user",
			Content:   "测试消息",
		}
		messageRepo.messages[messageID] = message

		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		result, err := service.GetMessageByID(ctx, messageID, userID)

		if err == nil {
			t.Error("期望返回错误，但没有错误")
		}
		if result != nil {
			t.Error("期望结果为nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Error("期望返回AppError类型")
		}
		if appErr.Code != errors.CodeMessageAccessDenied {
			t.Errorf("期望错误码 %d, 得到 %d", errors.CodeMessageAccessDenied, appErr.Code)
		}
	})
}

// TestAbortMessage 测试中止消息生成
func TestAbortMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("成功中止消息", func(t *testing.T) {
		messageID := "msg-123"
		userID := "user-123"
		sessionID := "session-123"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		// 创建会话
		session := &model.ChatSession{
			ID:     sessionID,
			UserID: userID,
			Title:  "测试会话",
		}
		sessionRepo.sessions[sessionID] = session

		// 创建消息
		message := &model.ChatMessage{
			ID:        messageID,
			SessionID: sessionID,
			Role:      "user",
			Content:   "测试消息",
		}
		messageRepo.messages[messageID] = message

		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		err := service.AbortMessage(ctx, messageID, userID)

		if err != nil {
			t.Fatalf("中止消息失败: %v", err)
		}
	})

	t.Run("消息不存在时幂等返回成功", func(t *testing.T) {
		messageID := "msg-not-exist"
		userID := "user-123"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		err := service.AbortMessage(ctx, messageID, userID)

		if err != nil {
			t.Errorf("期望成功，但返回错误: %v", err)
		}
	})

	t.Run("无权中止其他用户的消息", func(t *testing.T) {
		messageID := "msg-123"
		userID := "user-123"
		sessionID := "session-123"
		otherUserID := "user-456"

		sessionRepo := newMockSessionRepository()
		messageRepo := newTestMessageRepository()
		aiService := newTestAIService()

		// 创建会话（属于其他用户）
		session := &model.ChatSession{
			ID:     sessionID,
			UserID: otherUserID,
			Title:  "测试会话",
		}
		sessionRepo.sessions[sessionID] = session

		// 创建消息
		message := &model.ChatMessage{
			ID:        messageID,
			SessionID: sessionID,
			Role:      "user",
			Content:   "测试消息",
		}
		messageRepo.messages[messageID] = message

		service := NewMessageService(nil, sessionRepo, messageRepo, aiService, nil)

		err := service.AbortMessage(ctx, messageID, userID)

		if err == nil {
			t.Error("期望返回错误，但没有错误")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Error("期望返回AppError类型")
		}
		if appErr.Code != errors.CodeMessageAccessDenied {
			t.Errorf("期望错误码 %d, 得到 %d", errors.CodeMessageAccessDenied, appErr.Code)
		}
	})
}
