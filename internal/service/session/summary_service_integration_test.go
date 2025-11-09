package session

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
)

// ========== Mock 对象定义 ==========

// MockSummaryRepository 摘要仓储 Mock
type MockSummaryRepository struct {
	mock.Mock
}

func (m *MockSummaryRepository) Create(ctx context.Context, summary *model.ConversationSummary) error {
	args := m.Called(ctx, summary)
	return args.Error(0)
}

func (m *MockSummaryRepository) GetByID(ctx context.Context, tenantID, summaryID uuid.UUID) (*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, summaryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockSummaryRepository) GetLatestBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockSummaryRepository) ListBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID, limit int) ([]*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationSummary), args.Error(1)
}

// MockMessageRepository 消息仓储 Mock
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

func (m *MockMessageRepository) GetByID(ctx context.Context, id string) (*model.ChatMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ChatMessage), args.Error(1)
}

func (m *MockMessageRepository) GetMessagesAfter(ctx context.Context, sessionID, messageID string) ([]*model.ChatMessage, error) {
	args := m.Called(ctx, sessionID, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ChatMessage), args.Error(1)
}

// MockContextRepository 上下文仓储 Mock
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

func (m *MockContextRepository) Update(ctx context.Context, context *model.ConversationContext) error {
	args := m.Called(ctx, context)
	return args.Error(0)
}

// MockSessionRepository 会话仓储 Mock
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

// MockGenkitClient Genkit客户端 Mock
type MockGenkitClient struct {
	mock.Mock
}

func (m *MockGenkitClient) Generate(ctx context.Context, prompt string, opts *genkit.GenerateOptions) (*genkit.GenerateResult, error) {
	args := m.Called(ctx, prompt, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*genkit.GenerateResult), args.Error(1)
}

// MockTokenManager Token管理器 Mock
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) CalculateTokens(ctx context.Context, text, model string) (int, error) {
	args := m.Called(ctx, text, model)
	return args.Int(0), args.Error(1)
}

func (m *MockTokenManager) EstimateTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

func (m *MockTokenManager) CalculateMessagesTokens(ctx context.Context, messages []*model.ChatMessage, modelName string) (int, error) {
	args := m.Called(ctx, messages, modelName)
	return args.Int(0), args.Error(1)
}

// ========== 测试辅助函数 ==========

// createTestContextForSummary 创建测试上下文（带JWT声明）
func createTestContextForSummary(tenantID, userID string, roles []string) context.Context {
	ctx := context.Background()
	claims := &model.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userID,
		},
		TenantID: tenantID,
		Roles:    roles,
	}
	return context.WithValue(ctx, "jwt_claims", claims)
}

// createTestMessagesForSummary 创建测试消息列表
func createTestMessagesForSummary(count int) []*model.ChatMessage {
	messages := make([]*model.ChatMessage, count)
	for i := 0; i < count; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    role,
			Content: "测试消息内容 " + string(rune(i)),
		}
	}
	return messages
}

// ========== GenerateSummary 测试 ==========

func TestSummaryService_GenerateSummary_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	messages := createTestMessagesForSummary(10)

	// Mock Genkit 生成摘要
	genkitResult := &genkit.GenerateResult{
		Text:  "这是一个测试摘要，总结了用户与AI的对话内容",
		Model: "gemini-2.0-flash",
	}

	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 20).Return(messages, nil)
	mockGenkitClient.On("Generate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*genkit.GenerateOptions")).Return(genkitResult, nil)
	mockTokenMgr.On("CalculateTokens", ctx, genkitResult.Text, genkitResult.Model).Return(50, nil)
	mockTokenMgr.On("EstimateTokens", mock.AnythingOfType("string")).Return(10)
	mockTokenMgr.On("CalculateMessagesTokens", ctx, messages, genkitResult.Model).Return(500, nil)
	mockSummaryRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationSummary")).Return(nil)
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID: sid,
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockContextRepo.On("Update", ctx, mock.AnythingOfType("*model.ConversationContext")).Return(nil)

	// 执行测试
	req := &GenerateSummaryRequest{
		TenantID:    tenantID,
		SessionID:   sessionID,
		MessageIDs:  []uuid.UUID{},
		SummaryType: "full",
	}

	result, err := service.GenerateSummary(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantID, result.TenantID)
	assert.Equal(t, sessionID, result.SessionID)
	assert.Equal(t, "full", result.SummaryType)
	assert.Equal(t, genkitResult.Text, result.Content)
	assert.Equal(t, 50, result.TokenCount)
	assert.Equal(t, 10, result.MessageCount)

	// 验证 Mock 调用
	mockMessageRepo.AssertExpectations(t)
	mockGenkitClient.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
	mockSummaryRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
}

func TestSummaryService_GenerateSummary_IncrementalType(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	messages := createTestMessagesForSummary(5)

	// Mock 已有摘要
	previousSummary := &model.ConversationSummary{
		ID:      uuid.New(),
		Content: "之前的摘要内容",
	}

	// Mock Genkit 生成增量摘要
	genkitResult := &genkit.GenerateResult{
		Text:  "增量摘要：新增了一些对话内容",
		Model: "gemini-2.0-flash",
	}

	mockSummaryRepo.On("GetLatestBySessionID", ctx, tenantID, sessionID).Return(previousSummary, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 20).Return(messages, nil)
	mockGenkitClient.On("Generate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*genkit.GenerateOptions")).Return(genkitResult, nil)
	mockTokenMgr.On("CalculateTokens", ctx, genkitResult.Text, genkitResult.Model).Return(30, nil)
	mockTokenMgr.On("EstimateTokens", mock.AnythingOfType("string")).Return(10)
	mockTokenMgr.On("CalculateMessagesTokens", ctx, messages, genkitResult.Model).Return(250, nil)
	mockSummaryRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationSummary")).Return(nil)
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID: sid,
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockContextRepo.On("Update", ctx, mock.AnythingOfType("*model.ConversationContext")).Return(nil)

	// 执行测试
	req := &GenerateSummaryRequest{
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "incremental",
		PreviousSummary: previousSummary.Content,
	}

	result, err := service.GenerateSummary(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "incremental", result.SummaryType)
	assert.NotNil(t, result.PreviousSummaryID)
	assert.Equal(t, previousSummary.ID, *result.PreviousSummaryID)

	// 验证 Mock 调用
	mockSummaryRepo.AssertExpectations(t)
}


// ========== CheckSummaryTrigger 测试 ==========

func TestSummaryService_CheckSummaryTrigger_ShouldTrigger_MessageThreshold(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID:     sid,
		MaxTokens:     2000,
		TotalMessages: 25,
	}

	// 创建25条新消息
	messages := createTestMessagesForSummary(25)

	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockSummaryRepo.On("GetLatestBySessionID", ctx, tenantID, sessionID).Return(nil, repository.ErrNotFound)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 25).Return(messages, nil)

	// 执行测试
	result, err := service.CheckSummaryTrigger(ctx, tenantID, sessionID)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.ShouldSummarize)
	assert.Equal(t, 25, result.MessageCount)
	assert.Contains(t, result.TriggerReason, "消息数量")

	// 验证 Mock 调用
	mockContextRepo.AssertExpectations(t)
	mockSummaryRepo.AssertExpectations(t)
}

func TestSummaryService_CheckSummaryTrigger_ShouldTrigger_TokenLimit(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID:     sid,
		MaxTokens:     2000,
		TotalMessages: 15,
	}

	messages := createTestMessagesForSummary(15)

	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockSummaryRepo.On("GetLatestBySessionID", ctx, tenantID, sessionID).Return(nil, repository.ErrNotFound)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 15).Return(messages, nil)
	// Token使用量超过80%
	mockTokenMgr.On("CalculateMessagesTokens", ctx, messages, "").Return(1700, nil)

	// 执行测试
	result, err := service.CheckSummaryTrigger(ctx, tenantID, sessionID)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.ShouldSummarize)
	assert.Contains(t, result.TriggerReason, "Token")
	assert.Equal(t, 1.0, result.Urgency) // Token超限应该是最高紧急程度

	// 验证 Mock 调用
	mockTokenMgr.AssertExpectations(t)
}

func TestSummaryService_CheckSummaryTrigger_ShouldNotTrigger(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID:     sid,
		MaxTokens:     2000,
		TotalMessages: 5, // 消息数量少
	}

	messages := createTestMessagesForSummary(5)

	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockSummaryRepo.On("GetLatestBySessionID", ctx, tenantID, sessionID).Return(nil, repository.ErrNotFound)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 5).Return(messages, nil)
	mockTokenMgr.On("CalculateMessagesTokens", ctx, messages, "").Return(500, nil)

	// 执行测试
	result, err := service.CheckSummaryTrigger(ctx, tenantID, sessionID)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.ShouldSummarize)
	assert.Equal(t, 5, result.MessageCount)
}

// ========== EvaluateSummaryQuality 测试 ==========

func TestSummaryService_EvaluateSummaryQuality_HighQuality(t *testing.T) {
	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	ctx := context.Background()

	// 准备测试数据 - 高质量摘要
	summary := "用户询问了关于人工智能的基本概念，AI助手详细解释了机器学习、深度学习和神经网络的区别，并提供了实际应用案例。"
	originalMessages := []*model.ChatMessage{
		{Content: "什么是人工智能？"},
		{Content: "人工智能是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的系统。"},
		{Content: "机器学习和深度学习有什么区别？"},
		{Content: "机器学习是AI的一个子集，而深度学习是机器学习的一个子集，使用神经网络进行学习。"},
	}

	// 执行测试
	req := &EvaluateSummaryRequest{
		Summary:          summary,
		OriginalMessages: []string{originalMessages[0].Content, originalMessages[1].Content, originalMessages[2].Content, originalMessages[3].Content},
		Dimensions:       []string{"completeness", "conciseness", "coherence", "accuracy"},
	}

	mockTokenMgr.On("EstimateTokens", mock.AnythingOfType("string")).Return(50)

	result, err := service.EvaluateSummaryQuality(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Greater(t, result.OverallScore, 0.7)
	assert.Equal(t, 4, len(result.DimensionScores))
}

func TestSummaryService_EvaluateSummaryQuality_LowQuality(t *testing.T) {
	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	ctx := context.Background()

	// 准备测试数据 - 低质量摘要（过于简短）
	summary := "聊天"
	originalMessages := []*model.ChatMessage{
		{Content: "什么是人工智能？"},
		{Content: "人工智能是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的系统。"},
		{Content: "机器学习和深度学习有什么区别？"},
		{Content: "机器学习是AI的一个子集，而深度学习是机器学习的一个子集，使用神经网络进行学习。"},
	}

	// 执行测试
	req := &EvaluateSummaryRequest{
		Summary:          summary,
		OriginalMessages: []string{originalMessages[0].Content, originalMessages[1].Content, originalMessages[2].Content, originalMessages[3].Content},
		Dimensions:       []string{"completeness", "conciseness"},
	}

	mockTokenMgr.On("EstimateTokens", mock.AnythingOfType("string")).Return(5)

	result, err := service.EvaluateSummaryQuality(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	assert.Less(t, result.OverallScore, 0.7)
	assert.Greater(t, len(result.Issues), 0)
	assert.Greater(t, len(result.Suggestions), 0)
}

// ========== GetSummary 测试 ==========

func TestSummaryService_GetSummary_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()
	summaryID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	
	summary := &model.ConversationSummary{
		ID:        summaryID,
		TenantID:  tenantID,
		SessionID: sessionID,
		Content:   "测试摘要内容",
	}

	mockSummaryRepo.On("GetByID", ctx, tenantID, summaryID).Return(summary, nil)

	// 执行测试
	result, err := service.GetSummary(ctx, tenantID, summaryID)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, summaryID, result.ID)
	assert.Equal(t, "测试摘要内容", result.Content)

	// 验证 Mock 调用
	mockSummaryRepo.AssertExpectations(t)
}

// ========== ListSummaries 测试 ==========

func TestSummaryService_ListSummaries_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	
	summaries := []*model.ConversationSummary{
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SessionID: sessionID,
			Content:   "摘要1",
		},
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SessionID: sessionID,
			Content:   "摘要2",
		},
	}

	mockSummaryRepo.On("ListBySessionID", ctx, tenantID, sessionID, 10).Return(summaries, nil)

	// 执行测试
	results, err := service.ListSummaries(ctx, tenantID, sessionID, 10)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 2, len(results))

	// 验证 Mock 调用
	mockSummaryRepo.AssertExpectations(t)
}

// ========== 权限验证测试 ==========

func TestSummaryService_UnauthorizedAccess(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望 - 租户管理员尝试访问其他租户的会话
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})

	// 执行测试
	req := &GenerateSummaryRequest{
		TenantID:    otherTenantID, // 不同的租户ID
		SessionID:   sessionID,
		SummaryType: "full",
	}

	result, err := service.GenerateSummary(ctx, req)

	// 验证结果 - 应该返回权限错误
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "权限不足")
}

func TestSummaryService_SystemAdminCanAccessAllTenants(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockSummaryRepo := new(MockSummaryRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockContextRepo := new(MockContextRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockGenkitClient := new(MockGenkitClient)
	mockTokenMgr := new(MockTokenManager)
	mockLogger := logger.Default()

	// 创建服务实例
	service := NewSummaryService(
		mockSummaryRepo,
		mockMessageRepo,
		mockContextRepo,
		mockSessionRepo,
		mockGenkitClient,
		mockTokenMgr,
		mockLogger,
	)

	// 设置 Mock 期望 - 平台管理员访问其他租户的会话
	ctx := createTestContextForSummary(tenantID.String(), userID.String(), []string{model.RoleSystemAdmin})
	messages := createTestMessagesForSummary(10)

	genkitResult := &genkit.GenerateResult{
		Text:  "测试摘要",
		Model: "gemini-2.0-flash",
	}

	mockMessageRepo.On("GetLatestMessages", ctx, sessionID.String(), 20).Return(messages, nil)
	mockGenkitClient.On("Generate", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*genkit.GenerateOptions")).Return(genkitResult, nil)
	mockTokenMgr.On("CalculateTokens", ctx, genkitResult.Text, genkitResult.Model).Return(50, nil)
	mockTokenMgr.On("EstimateTokens", mock.AnythingOfType("string")).Return(10)
	mockTokenMgr.On("CalculateMessagesTokens", ctx, messages, genkitResult.Model).Return(500, nil)
	mockSummaryRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationSummary")).Return(nil)
	
	sid := sessionID
	contextConfig := &model.ConversationContext{
		SessionID: sid,
	}
	mockContextRepo.On("GetBySessionID", ctx, sessionID.String()).Return(contextConfig, nil)
	mockContextRepo.On("Update", ctx, mock.AnythingOfType("*model.ConversationContext")).Return(nil)

	// 执行测试
	req := &GenerateSummaryRequest{
		TenantID:    otherTenantID, // 不同的租户ID
		SessionID:   sessionID,
		SummaryType: "full",
	}

	result, err := service.GenerateSummary(ctx, req)

	// 验证结果 - 平台管理员应该可以访问
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 验证 Mock 调用
	mockMessageRepo.AssertExpectations(t)
	mockGenkitClient.AssertExpectations(t)
}
