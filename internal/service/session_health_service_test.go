package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
)

// MockSessionRepository 模拟会话仓储
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id string) (*model.ChatSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ChatSession), args.Error(1)
}

// MockMessageRepository 模拟消息仓储
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error) {
	args := m.Called(ctx, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ChatMessage), args.Error(1)
}

// MockMemoryRepository 模拟记忆仓储
type MockMemoryRepository struct {
	mock.Mock
}

// MockContextRepository 模拟上下文仓储
type MockContextRepository struct {
	mock.Mock
}

func (m *MockContextRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationContext), args.Error(1)
}

func (m *MockContextRepository) GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

// MockTokenManager 模拟Token管理器
type MockTokenManager struct {
	mock.Mock
}

// MockCacheService 模拟缓存服务
type MockCacheService struct {
	mock.Mock
}

func TestCheckSessionHealth_Success(t *testing.T) {
	// 准备
	ctx := context.Background()
	sessionID := uuid.New().String()

	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockTokenMgr := new(MockTokenManager)
	mockCache := new(MockCacheService)

	service := NewSessionHealthService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockTokenMgr,
		mockCache,
	)

	// 设置Mock行为
	session := &model.ChatSession{
		ID:        uuid.MustParse(sessionID),
		UserID:    uuid.New(),
		Title:     "测试会话",
		ModelName: "gemini-1.5-flash",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)

	contextConfig := &model.ConversationContext{
		ID:        uuid.New(),
		SessionID: uuid.MustParse(sessionID),
		MaxTokens: 4000,
		Strategy:  "auto",
		CreatedAt: time.Now(),
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID).Return(contextConfig, nil)

	messages := []*model.ChatMessage{
		{
			ID:        uuid.New(),
			SessionID: uuid.MustParse(sessionID),
			Role:      "user",
			Content:   "测试消息",
			Tokens:    10,
			CreatedAt: time.Now(),
		},
	}
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 20).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 50).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 100).Return(messages, nil)

	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		SessionID:  uuid.MustParse(sessionID),
		Content:    "测试摘要",
		TokenCount: 50,
		CreatedAt:  time.Now(),
	}
	mockContextRepo.On("GetLatestSummary", ctx, sessionID).Return(summary, nil)

	// 执行
	req := SessionHealthCheckRequest{
		SessionID:   sessionID,
		CheckItems:  []string{"context", "token", "memory", "summary", "performance"},
		AutoFix:     false,
		DetailLevel: "detailed",
	}

	result, err := service.CheckSessionHealth(ctx, req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sessionID, result.SessionID)
	assert.Equal(t, "healthy", result.OverallHealth)
	assert.Greater(t, result.OverallScore, 0.7)
	assert.Len(t, result.CheckResults, 5)
	assert.Greater(t, result.CheckTime, int64(0))

	mockSessionRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}

func TestCheckSessionHealth_WithIssues(t *testing.T) {
	// 准备
	ctx := context.Background()
	sessionID := uuid.New().String()

	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockTokenMgr := new(MockTokenManager)
	mockCache := new(MockCacheService)

	service := NewSessionHealthService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockTokenMgr,
		mockCache,
	)

	// 设置Mock行为 - 低Token限制
	session := &model.ChatSession{
		ID:        uuid.MustParse(sessionID),
		UserID:    uuid.New(),
		Title:     "测试会话",
		ModelName: "gemini-1.5-flash",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)

	contextConfig := &model.ConversationContext{
		ID:        uuid.New(),
		SessionID: uuid.MustParse(sessionID),
		MaxTokens: 500, // 低Token限制
		Strategy:  "auto",
		CreatedAt: time.Now(),
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID).Return(contextConfig, nil)

	// 高Token使用
	messages := make([]*model.ChatMessage, 0)
	for i := 0; i < 20; i++ {
		messages = append(messages, &model.ChatMessage{
			ID:        uuid.New(),
			SessionID: uuid.MustParse(sessionID),
			Role:      "user",
			Content:   "测试消息",
			Tokens:    25, // 总共500 tokens
			CreatedAt: time.Now(),
		})
	}
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 20).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 50).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 100).Return(messages, nil)

	// 没有摘要
	mockContextRepo.On("GetLatestSummary", ctx, sessionID).Return(nil, assert.AnError)

	// 执行
	req := SessionHealthCheckRequest{
		SessionID:   sessionID,
		CheckItems:  []string{"context", "token", "summary"},
		AutoFix:     false,
		DetailLevel: "detailed",
	}

	result, err := service.CheckSessionHealth(ctx, req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sessionID, result.SessionID)
	assert.NotEqual(t, "healthy", result.OverallHealth) // 应该有警告或严重问题
	assert.Less(t, result.OverallScore, 0.8)
	assert.Greater(t, len(result.Issues), 0) // 应该有问题
	assert.Greater(t, len(result.Recommendations), 0) // 应该有建议

	mockSessionRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}

func TestCheckSessionHealth_AutoFix(t *testing.T) {
	// 准备
	ctx := context.Background()
	sessionID := uuid.New().String()

	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockTokenMgr := new(MockTokenManager)
	mockCache := new(MockCacheService)

	service := NewSessionHealthService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockTokenMgr,
		mockCache,
	)

	// 设置Mock行为
	session := &model.ChatSession{
		ID:        uuid.MustParse(sessionID),
		UserID:    uuid.New(),
		Title:     "测试会话",
		ModelName: "gemini-1.5-flash",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)

	// 缺少上下文配置
	mockContextRepo.On("GetBySessionID", ctx, sessionID).Return(nil, assert.AnError)

	messages := []*model.ChatMessage{}
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 20).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 50).Return(messages, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 100).Return(messages, nil)

	mockContextRepo.On("GetLatestSummary", ctx, sessionID).Return(nil, assert.AnError)

	// 执行 - 启用自动修复
	req := SessionHealthCheckRequest{
		SessionID:   sessionID,
		CheckItems:  []string{"context"},
		AutoFix:     true,
		DetailLevel: "detailed",
	}

	result, err := service.CheckSessionHealth(ctx, req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result.FixOperations), 0) // 应该有修复操作
	
	// 检查修复操作状态
	hasSuccessfulFix := false
	for _, fix := range result.FixOperations {
		if fix.Status == "success" {
			hasSuccessfulFix = true
			break
		}
	}
	assert.True(t, hasSuccessfulFix, "应该有成功的修复操作")

	mockSessionRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}
