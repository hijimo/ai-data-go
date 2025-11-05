package service

import (
	"context"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheService Mock 缓存服务
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheService) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	args := m.Called(ctx, key, delta)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

func (m *MockCacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockCacheService) HashQuery(query string) string {
	args := m.Called(query)
	return args.String(0)
}

// MockGenkitMemoryRepository Mock 记忆仓储
type MockGenkitMemoryRepository struct {
	mock.Mock
}

func (m *MockGenkitMemoryRepository) SearchByContent(ctx context.Context, sessionID, query string, topK int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, sessionID, query, topK)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

// MockGenkitMessageRepository Mock 消息仓储
type MockGenkitMessageRepository struct {
	mock.Mock
}

// MockLogger Mock 日志记录器
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

// TestDegradeAIService_CacheHit 测试 AI 服务降级 - 缓存命中
func TestDegradeAIService_CacheHit(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	sessionID := uuid.New().String()
	userQuery := "测试查询"

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockCache.On("HashQuery", userQuery).Return("hash123")
	mockCache.On("Get", ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		dest := args.Get(2).(*string)
		*dest = "这是缓存的响应"
	}).Return(nil)

	// 执行
	result, err := svc.DegradeAIService(ctx, sessionID, userQuery)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "这是缓存的响应", result.Response)
	assert.Equal(t, "cache", result.Source)
	assert.True(t, result.CacheHit)
	assert.Greater(t, result.DegradationTime, int64(0))

	mockCache.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestDegradeAIService_DefaultResponse 测试 AI 服务降级 - 默认响应
func TestDegradeAIService_DefaultResponse(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	sessionID := uuid.New().String()
	userQuery := "测试查询"

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockCache.On("HashQuery", userQuery).Return("hash123")
	mockCache.On("Get", ctx, mock.Anything, mock.Anything).Return(ErrCacheNotFound)

	// 执行
	result, err := svc.DegradeAIService(ctx, sessionID, userQuery)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Response)
	assert.Equal(t, "default", result.Source)
	assert.False(t, result.CacheHit)
	assert.Greater(t, result.DegradationTime, int64(0))
	assert.Contains(t, result.Response, "服务暂时不可用")

	mockCache.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestDegradeVectorSearch_FullTextSuccess 测试向量检索降级 - 全文搜索成功
func TestDegradeVectorSearch_FullTextSuccess(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	sessionID := uuid.New().String()
	query := "测试查询"
	topK := 5

	// 准备测试数据
	testMemories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			SessionID:  uuid.MustParse(sessionID),
			Content:    "测试记忆1",
			Importance: 0.8,
		},
		{
			ID:         uuid.New(),
			SessionID:  uuid.MustParse(sessionID),
			Content:    "测试记忆2",
			Importance: 0.7,
		},
	}

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockMemoryRepo.On("SearchByContent", ctx, sessionID, query, topK).Return(testMemories, nil)

	// 执行
	result, err := svc.DegradeVectorSearch(ctx, sessionID, query, topK)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Memories, 2)
	assert.Equal(t, "fulltext", result.Source)
	assert.True(t, result.FullTextUsed)
	assert.Greater(t, result.DegradationTime, int64(0))

	mockMemoryRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestDegradeVectorSearch_EmptyResult 测试向量检索降级 - 空结果
func TestDegradeVectorSearch_EmptyResult(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	sessionID := uuid.New().String()
	query := "测试查询"
	topK := 5

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockMemoryRepo.On("SearchByContent", ctx, sessionID, query, topK).Return([]*model.ConversationMemory{}, nil)

	// 执行
	result, err := svc.DegradeVectorSearch(ctx, sessionID, query, topK)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Memories)
	assert.Equal(t, "empty", result.Source)
	assert.False(t, result.FullTextUsed)
	assert.Greater(t, result.DegradationTime, int64(0))

	mockMemoryRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestDegradeSummaryGeneration_Truncate 测试摘要生成降级 - 截断策略
func TestDegradeSummaryGeneration_Truncate(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	messages := []*model.ConversationMessage{
		{
			ID:      uuid.New(),
			Role:    "user",
			Content: "这是一条很长的用户消息，包含了很多内容，需要被截断处理。这是一条很长的用户消息，包含了很多内容，需要被截断处理。",
		},
		{
			ID:      uuid.New(),
			Role:    "assistant",
			Content: "这是助手的回复，也包含了很多内容。",
		},
	}
	targetLength := 50

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// 执行
	result, err := svc.DegradeSummaryGeneration(ctx, messages, targetLength)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Summary)
	assert.Equal(t, "truncate", result.Method)
	assert.LessOrEqual(t, result.SummaryLength, targetLength+10) // 允许一些误差
	assert.Greater(t, result.OriginalLength, result.SummaryLength)
	assert.Greater(t, result.DegradationTime, int64(0))

	mockLogger.AssertExpectations(t)
}

// TestDegradeSummaryGeneration_Extract 测试摘要生成降级 - 提取策略
func TestDegradeSummaryGeneration_Extract(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	messages := []*model.ConversationMessage{
		{
			ID:      uuid.New(),
			Role:    "user",
			Content: "这是第一个问题。这是第二个问题。这是第三个问题。",
		},
		{
			ID:      uuid.New(),
			Role:    "assistant",
			Content: "这是第一个回答。这是第二个回答。这是第三个回答。",
		},
	}
	targetLength := 300

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// 执行
	result, err := svc.DegradeSummaryGeneration(ctx, messages, targetLength)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Summary)
	assert.Equal(t, "extract", result.Method)
	assert.LessOrEqual(t, result.SummaryLength, targetLength+50) // 允许一些误差
	assert.Greater(t, result.DegradationTime, int64(0))

	mockLogger.AssertExpectations(t)
}

// TestDegradeSummaryGeneration_Direct 测试摘要生成降级 - 直接返回
func TestDegradeSummaryGeneration_Direct(t *testing.T) {
	// 准备
	mockCache := new(MockCacheService)
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockMessageRepo := new(MockGenkitMessageRepository)
	mockLogger := new(MockLogger)

	svc := NewDegradationService(mockCache, mockMemoryRepo, mockMessageRepo, mockLogger)

	ctx := context.Background()
	messages := []*model.ConversationMessage{
		{
			ID:      uuid.New(),
			Role:    "user",
			Content: "短消息",
		},
	}
	targetLength := 1000

	// 设置 Mock 行为
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// 执行
	result, err := svc.DegradeSummaryGeneration(ctx, messages, targetLength)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Summary)
	assert.Equal(t, "direct", result.Method)
	assert.Equal(t, result.OriginalLength, result.SummaryLength)
	assert.Greater(t, result.DegradationTime, int64(0))

	mockLogger.AssertExpectations(t)
}
