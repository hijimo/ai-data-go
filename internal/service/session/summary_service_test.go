package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
)

// mockSummaryRepository 模拟摘要仓库
type mockSummaryRepository struct {
	summaries                map[string][]*model.ChatSummary
	createFunc               func(ctx context.Context, summary *model.ChatSummary) error
	getLatestBySessionIDFunc func(ctx context.Context, sessionID string) (*model.ChatSummary, error)
	getBySessionIDFunc       func(ctx context.Context, sessionID string) ([]*model.ChatSummary, error)
}

func newMockSummaryRepository() *mockSummaryRepository {
	return &mockSummaryRepository{
		summaries: make(map[string][]*model.ChatSummary),
	}
}

func (m *mockSummaryRepository) Create(ctx context.Context, summary *model.ChatSummary) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, summary)
	}
	if summary.ID == "" {
		summary.ID = "test-summary-id"
	}
	m.summaries[summary.SessionID] = append(m.summaries[summary.SessionID], summary)
	return nil
}

func (m *mockSummaryRepository) GetLatestBySessionID(ctx context.Context, sessionID string) (*model.ChatSummary, error) {
	if m.getLatestBySessionIDFunc != nil {
		return m.getLatestBySessionIDFunc(ctx, sessionID)
	}
	summaries := m.summaries[sessionID]
	if len(summaries) == 0 {
		return nil, nil
	}
	return summaries[len(summaries)-1], nil
}

func (m *mockSummaryRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*model.ChatSummary, error) {
	if m.getBySessionIDFunc != nil {
		return m.getBySessionIDFunc(ctx, sessionID)
	}
	return m.summaries[sessionID], nil
}

// mockAIService 模拟AI服务
type mockAIService struct {
	chatFunc       func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error)
	chatStreamFunc func(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error)
	abortChatFunc  func(ctx context.Context, sessionID string) error
}

func (m *mockAIService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &model.ChatResponse{
		Message: "这是一个测试摘要",
		Usage: &model.Usage{
			TotalTokens: 100,
		},
	}, nil
}

func (m *mockAIService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, error) {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAIService) AbortChat(ctx context.Context, sessionID string) error {
	if m.abortChatFunc != nil {
		return m.abortChatFunc(ctx, sessionID)
	}
	return nil
}

// setupSummaryServiceTest 设置测试环境
func setupSummaryServiceTest() (*summaryService, *mockSummaryRepository, *mockMessageRepository, *mockSessionRepository, *mockAIService) {
	summaryRepo := newMockSummaryRepository()
	messageRepo := newMockMessageRepository()
	sessionRepo := newMockSessionRepository()
	aiService := &mockAIService{}

	cfg := &config.Config{
		Session: config.SessionConfig{
			SummaryThreshold: 50,
			DefaultPageSize:  20,
			MaxPageSize:      100,
			MaxTitleLength:   255,
		},
	}

	logger.Init("info", "json")
	log := logger.Default()

	service := &summaryService{
		summaryRepo: summaryRepo,
		messageRepo: messageRepo,
		sessionRepo: sessionRepo,
		aiService:   aiService,
		config:      cfg,
		logger:      log,
	}

	return service, summaryRepo, messageRepo, sessionRepo, aiService
}

// TestGenerateSummary_Success 测试成功生成摘要
func TestGenerateSummary_Success(t *testing.T) {
	service, summaryRepo, messageRepo, sessionRepo, aiService := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"
	userID := "test-user-id"

	// 模拟会话存在
	session := &model.ChatSession{
		ID:        sessionID,
		UserID:    userID,
		Title:     "测试会话",
		ModelName: "gemini-2.5-flash",
	}
	sessionRepo.sessions[sessionID] = session

	// 模拟消息列表
	messages := []*model.ChatMessage{
		{
			ID:        "msg-1",
			SessionID: sessionID,
			Role:      "user",
			Content:   "你好",
			Sequence:  1,
		},
		{
			ID:        "msg-2",
			SessionID: sessionID,
			Role:      "assistant",
			Content:   "你好！有什么可以帮助你的吗？",
			Sequence:  2,
		},
	}
	
	messageRepo.getBySessionIDFunc = func(ctx context.Context, sid string, page, pageSize int) ([]*model.ChatMessage, int, error) {
		return messages, len(messages), nil
	}

	// 模拟AI生成摘要
	aiService.chatFunc = func(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
		return &model.ChatResponse{
			Message: "用户与AI进行了简单的问候交流",
			Usage: &model.Usage{
				TotalTokens: 50,
			},
		}, nil
	}

	// 模拟保存摘要
	var savedSummary *model.ChatSummary
	summaryRepo.createFunc = func(ctx context.Context, summary *model.ChatSummary) error {
		savedSummary = summary
		summary.ID = "summary-1"
		summary.CreatedAt = time.Now()
		return nil
	}

	// 执行测试
	summary, err := service.GenerateSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("GenerateSummary() 返回错误: %v", err)
	}

	if summary == nil {
		t.Fatal("GenerateSummary() 返回的摘要为空")
	}

	if savedSummary == nil {
		t.Fatal("摘要未被保存")
	}

	if savedSummary.SessionID != sessionID {
		t.Errorf("摘要的会话ID不正确，期望 %s，实际 %s", sessionID, savedSummary.SessionID)
	}

	if savedSummary.Summary != "用户与AI进行了简单的问候交流" {
		t.Errorf("摘要内容不正确，实际: %s", savedSummary.Summary)
	}

	if savedSummary.LastMessageID != "msg-2" {
		t.Errorf("最后消息ID不正确，期望 msg-2，实际 %s", savedSummary.LastMessageID)
	}

	if savedSummary.TokenCount != 50 {
		t.Errorf("Token数量不正确，期望 50，实际 %d", savedSummary.TokenCount)
	}
}

// TestGenerateSummary_SessionNotFound 测试会话不存在
func TestGenerateSummary_SessionNotFound(t *testing.T) {
	service, _, _, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "non-existent-session"

	// 会话不存在（sessions map中没有该ID）

	// 执行测试
	_, err := service.GenerateSummary(ctx, sessionID)

	// 验证结果
	if err == nil {
		t.Error("GenerateSummary() 应该返回错误")
	}
}

// TestGenerateSummary_NoMessages 测试没有消息
func TestGenerateSummary_NoMessages(t *testing.T) {
	service, _, messageRepo, sessionRepo, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟会话存在
	session := &model.ChatSession{
		ID:        sessionID,
		ModelName: "gemini-2.5-flash",
	}
	sessionRepo.sessions[sessionID] = session

	// 模拟没有消息
	messageRepo.getBySessionIDFunc = func(ctx context.Context, sid string, page, pageSize int) ([]*model.ChatMessage, int, error) {
		return []*model.ChatMessage{}, 0, nil
	}

	// 执行测试
	summary, err := service.GenerateSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("GenerateSummary() 返回错误: %v", err)
	}

	if summary != nil {
		t.Error("没有消息时应该返回nil")
	}
}

// TestGetSummary_Success 测试成功获取摘要
func TestGetSummary_Success(t *testing.T) {
	service, summaryRepo, _, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟摘要存在
	expectedSummary := &model.ChatSummary{
		ID:            "summary-1",
		SessionID:     sessionID,
		Summary:       "这是一个测试摘要",
		LastMessageID: "msg-10",
		TokenCount:    100,
		CreatedAt:     time.Now(),
	}

	summaryRepo.getLatestBySessionIDFunc = func(ctx context.Context, sid string) (*model.ChatSummary, error) {
		return expectedSummary, nil
	}

	// 执行测试
	summary, err := service.GetSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("GetSummary() 返回错误: %v", err)
	}

	if summary == nil {
		t.Fatal("GetSummary() 返回的摘要为空")
	}

	if summary.ID != expectedSummary.ID {
		t.Errorf("摘要ID不正确，期望 %s，实际 %s", expectedSummary.ID, summary.ID)
	}

	if summary.Summary != expectedSummary.Summary {
		t.Errorf("摘要内容不正确，期望 %s，实际 %s", expectedSummary.Summary, summary.Summary)
	}
}

// TestShouldGenerateSummary_BelowThreshold 测试消息数量未达到阈值
func TestShouldGenerateSummary_BelowThreshold(t *testing.T) {
	service, _, messageRepo, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟消息数量为30（低于阈值50）
	messageRepo.countBySessionIDFunc = func(ctx context.Context, sid string) (int, error) {
		return 30, nil
	}

	// 执行测试
	shouldGenerate, err := service.ShouldGenerateSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("ShouldGenerateSummary() 返回错误: %v", err)
	}

	if shouldGenerate {
		t.Error("消息数量未达到阈值时不应该生成摘要")
	}
}

// TestShouldGenerateSummary_AboveThreshold_NoSummary 测试消息数量达到阈值且无摘要
func TestShouldGenerateSummary_AboveThreshold_NoSummary(t *testing.T) {
	service, summaryRepo, messageRepo, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟消息数量为60（高于阈值50）
	messageRepo.countBySessionIDFunc = func(ctx context.Context, sid string) (int, error) {
		return 60, nil
	}

	// 模拟没有摘要
	summaryRepo.getLatestBySessionIDFunc = func(ctx context.Context, sid string) (*model.ChatSummary, error) {
		return nil, nil
	}

	// 执行测试
	shouldGenerate, err := service.ShouldGenerateSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("ShouldGenerateSummary() 返回错误: %v", err)
	}

	if !shouldGenerate {
		t.Error("消息数量达到阈值且无摘要时应该生成摘要")
	}
}

// TestShouldGenerateSummary_WithExistingSummary 测试已有摘要的情况
func TestShouldGenerateSummary_WithExistingSummary(t *testing.T) {
	service, summaryRepo, messageRepo, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟消息数量为100
	messageRepo.countBySessionIDFunc = func(ctx context.Context, sid string) (int, error) {
		return 100, nil
	}

	// 模拟已有摘要
	summaryRepo.getLatestBySessionIDFunc = func(ctx context.Context, sid string) (*model.ChatSummary, error) {
		return &model.ChatSummary{
			ID:            "summary-1",
			SessionID:     sessionID,
			LastMessageID: "msg-50",
		}, nil
	}

	// 模拟摘要后有60条新消息（超过阈值50）
	messageRepo.getMessagesAfterFunc = func(ctx context.Context, sid, afterMsgID string) ([]*model.ChatMessage, error) {
		messages := make([]*model.ChatMessage, 60)
		for i := 0; i < 60; i++ {
			messages[i] = &model.ChatMessage{
				ID:        "msg-" + string(rune(51+i)),
				SessionID: sessionID,
				Role:      "user",
				Content:   "测试消息",
			}
		}
		return messages, nil
	}

	// 执行测试
	shouldGenerate, err := service.ShouldGenerateSummary(ctx, sessionID)

	// 验证结果
	if err != nil {
		t.Errorf("ShouldGenerateSummary() 返回错误: %v", err)
	}

	if !shouldGenerate {
		t.Error("摘要后的新消息数量达到阈值时应该生成摘要")
	}
}

// TestShouldGenerateSummary_CountError 测试统计消息数量失败
func TestShouldGenerateSummary_CountError(t *testing.T) {
	service, _, messageRepo, _, _ := setupSummaryServiceTest()
	ctx := context.Background()

	sessionID := "test-session-id"

	// 模拟统计失败
	messageRepo.countBySessionIDFunc = func(ctx context.Context, sid string) (int, error) {
		return 0, errors.New("数据库错误")
	}

	// 执行测试
	_, err := service.ShouldGenerateSummary(ctx, sessionID)

	// 验证结果
	if err == nil {
		t.Error("ShouldGenerateSummary() 应该返回错误")
	}
}
