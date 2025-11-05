package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
)

// MockContextRepository 模拟上下文仓库
type MockContextRepository struct {
	mock.Mock
}

func (m *MockContextRepository) Create(ctx context.Context, context *model.ConversationContext) error {
	args := m.Called(ctx, context)
	return args.Error(0)
}

func (m *MockContextRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationContext), args.Error(1)
}

func (m *MockContextRepository) Update(ctx context.Context, context *model.ConversationContext) error {
	args := m.Called(ctx, context)
	return args.Error(0)
}

func (m *MockContextRepository) GetLatestSummary(ctx context.Context, sessionID string) (*model.ConversationSummary, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockContextRepository) UpdateTokenUsage(ctx context.Context, sessionID string, tokens int) error {
	args := m.Called(ctx, sessionID, tokens)
	return args.Error(0)
}

// MockMessageRepository 模拟消息仓库
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]*model.ChatMessage, error) {
	args := m.Called(ctx, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ChatMessage), args.Error(1)
}

// MockTokenManager 模拟Token管理器
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) CalculateContextTokens(messages []*model.ChatMessage, memories []*model.ConversationMemory, summary *model.ConversationSummary) int {
	args := m.Called(messages, memories, summary)
	return args.Int(0)
}

func (m *MockTokenManager) CalculateTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

// TestContextService_BuildContext 测试构建上下文
func TestContextService_BuildContext(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取最近消息
	messages := []*model.ChatMessage{
		{ID: uuid.New(), Role: "user", Content: "消息1"},
		{ID: uuid.New(), Role: "assistant", Content: "回复1"},
	}
	mockMessageRepo.On("GetRecentMessages", ctx, sessionID, 10).Return(messages, nil)

	// 模拟Token计算
	mockTokenMgr.On("CalculateContextTokens", messages, mock.Anything, mock.Anything).Return(500)

	// 执行构建
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sessionID, result.SessionID)
	assert.Len(t, result.ShortTermMessages, 2)
	assert.Equal(t, 500, result.TotalTokens)

	mockMessageRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_BuildContext_WithLongTermMemory 测试构建包含长期记忆的上下文
func TestContextService_BuildContext_WithLongTermMemory(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取最近消息
	messages := []*model.ChatMessage{
		{ID: uuid.New(), Role: "user", Content: "消息1"},
	}
	mockMessageRepo.On("GetRecentMessages", ctx, sessionID, 10).Return(messages, nil)

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试查询").Return(embedding, nil)

	// 模拟向量检索
	memories := []*model.ConversationMemory{
		{ID: uuid.New(), Content: "相关记忆1", Importance: 0.9},
		{ID: uuid.New(), Content: "相关记忆2", Importance: 0.8},
	}
	mockMemoryRepo.On("SearchByVector", ctx, sessionID, embedding, 5, float32(0.7)).Return(memories, nil)

	// 模拟Token计算
	mockTokenMgr.On("CalculateContextTokens", messages, memories, mock.Anything).Return(800)

	// 执行构建
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.LongTermMemories, 2)
	assert.Equal(t, 800, result.TotalTokens)

	mockMessageRepo.AssertExpectations(t)
	mockVectorSvc.AssertExpectations(t)
	mockMemoryRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_BuildContext_WithSummary 测试构建包含摘要的上下文
func TestContextService_BuildContext_WithSummary(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取最近消息
	messages := []*model.ChatMessage{
		{ID: uuid.New(), Role: "user", Content: "消息1"},
	}
	mockMessageRepo.On("GetRecentMessages", ctx, sessionID, 10).Return(messages, nil)

	// 模拟获取摘要
	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		Content:    "对话摘要",
		TokenCount: 100,
	}
	mockContextRepo.On("GetLatestSummary", ctx, sessionID).Return(summary, nil)

	// 模拟Token计算
	mockTokenMgr.On("CalculateContextTokens", messages, mock.Anything, summary).Return(600)

	// 执行构建
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Summary)
	assert.Equal(t, "对话摘要", result.Summary.Content)
	assert.Equal(t, 600, result.TotalTokens)

	mockMessageRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_OptimizeContext 测试优化上下文
func TestContextService_OptimizeContext(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()

	// 创建原始上下文
	originalContext := &ContextResult{
		SessionID: uuid.New().String(),
		ShortTermMessages: []*model.ChatMessage{
			{ID: uuid.New(), Content: "消息1"},
			{ID: uuid.New(), Content: "消息2"},
			{ID: uuid.New(), Content: "消息3"},
			{ID: uuid.New(), Content: "消息4"},
			{ID: uuid.New(), Content: "消息5"},
		},
		LongTermMemories: []*model.ConversationMemory{
			{ID: uuid.New(), Content: "记忆1", Importance: 0.9},
			{ID: uuid.New(), Content: "记忆2", Importance: 0.8},
			{ID: uuid.New(), Content: "记忆3", Importance: 0.7},
		},
		TotalTokens: 5000,
	}

	// 模拟Token计算（优化后）
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(3000)

	// 执行优化
	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    3000,
		Strategy:        "balanced",
		PreserveSummary: true,
	}

	result, err := service.OptimizeContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, result.TotalTokens, 3000)
	assert.Less(t, len(result.ShortTermMessages), len(originalContext.ShortTermMessages))

	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_OptimizeContext_AggressiveStrategy 测试激进优化策略
func TestContextService_OptimizeContext_AggressiveStrategy(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()

	// 创建原始上下文
	originalContext := &ContextResult{
		SessionID: uuid.New().String(),
		ShortTermMessages: []*model.ChatMessage{
			{ID: uuid.New(), Content: "消息1"},
			{ID: uuid.New(), Content: "消息2"},
			{ID: uuid.New(), Content: "消息3"},
			{ID: uuid.New(), Content: "消息4"},
			{ID: uuid.New(), Content: "消息5"},
			{ID: uuid.New(), Content: "消息6"},
			{ID: uuid.New(), Content: "消息7"},
			{ID: uuid.New(), Content: "消息8"},
		},
		LongTermMemories: []*model.ConversationMemory{
			{ID: uuid.New(), Content: "记忆1", Importance: 0.9},
			{ID: uuid.New(), Content: "记忆2", Importance: 0.8},
			{ID: uuid.New(), Content: "记忆3", Importance: 0.7},
			{ID: uuid.New(), Content: "记忆4", Importance: 0.6},
		},
		Summary: &model.ConversationSummary{
			ID:      uuid.New(),
			Content: "摘要内容",
		},
		TotalTokens: 6000,
	}

	// 模拟Token计算（优化后）
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(2500)

	// 执行激进优化
	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    2500,
		Strategy:        "aggressive",
		PreserveSummary: false, // 激进策略可能移除摘要
	}

	result, err := service.OptimizeContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, result.TotalTokens, 2500)
	// 激进策略应该大幅减少消息和记忆数量
	assert.LessOrEqual(t, len(result.ShortTermMessages), 5)
	assert.LessOrEqual(t, len(result.LongTermMemories), 2)

	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_GetContextConfig 测试获取上下文配置
func TestContextService_GetContextConfig(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取配置
	config := &model.ConversationContext{
		ID:              uuid.New(),
		SessionID:       uuid.MustParse(sessionID),
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID).Return(config, nil)

	// 执行获取
	result, err := service.GetContextConfig(ctx, sessionID)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4000, result.MaxTokens)
	assert.Equal(t, "auto", result.Strategy)

	mockContextRepo.AssertExpectations(t)
}

// TestContextService_UpdateContextConfig 测试更新上下文配置
func TestContextService_UpdateContextConfig(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟更新配置
	config := &model.ConversationContext{
		ID:              uuid.New(),
		SessionID:       uuid.MustParse(sessionID),
		MaxTokens:       8000,
		Strategy:        "full",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 20,
	}
	mockContextRepo.On("Update", ctx, config).Return(nil)

	// 执行更新
	err := service.UpdateContextConfig(ctx, sessionID, config)

	// 验证
	assert.NoError(t, err)

	mockContextRepo.AssertExpectations(t)
}

// TestContextService_BuildContext_TokenExceeded 测试Token超限自动优化
func TestContextService_BuildContext_TokenExceeded(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取大量消息
	messages := make([]*model.ChatMessage, 20)
	for i := 0; i < 20; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: "这是一条很长的消息内容",
		}
	}
	mockMessageRepo.On("GetRecentMessages", ctx, sessionID, 10).Return(messages, nil)

	// 第一次Token计算（超限）
	mockTokenMgr.On("CalculateContextTokens", messages, mock.Anything, mock.Anything).Return(5000).Once()

	// 第二次Token计算（优化后）
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(3500).Once()

	// 执行构建
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, result.TotalTokens, 4000)

	mockMessageRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}

// TestContextService_CalculateQualityScore 测试质量评分计算
func TestContextService_CalculateQualityScore(t *testing.T) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "test")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟获取消息
	messages := []*model.ChatMessage{
		{ID: uuid.New(), Role: "user", Content: "消息1"},
		{ID: uuid.New(), Role: "assistant", Content: "回复1"},
		{ID: uuid.New(), Role: "user", Content: "消息2"},
	}
	mockMessageRepo.On("GetRecentMessages", ctx, sessionID, 10).Return(messages, nil)

	// 模拟Token计算
	mockTokenMgr.On("CalculateContextTokens", messages, mock.Anything, mock.Anything).Return(500)

	// 执行构建
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.QualityScore, 0.0)
	assert.LessOrEqual(t, result.QualityScore, 1.0)

	mockMessageRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}
