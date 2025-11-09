package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service/ai"
)

// ========== Mock 对象定义 ==========

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

func (m *MockSessionRepository) Create(ctx context.Context, session *model.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *model.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	args := m.Called(ctx, tenantID, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) List(ctx context.Context, tenantID, userID uuid.UUID, pageNo, pageSize int) ([]*model.ChatSession, int64, error) {
	args := m.Called(ctx, tenantID, userID, pageNo, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.ChatSession), args.Get(1).(int64), args.Error(2)
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

func (m *MockMessageRepository) Create(ctx context.Context, message *model.ChatMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// MockMemoryRepository 记忆仓储 Mock
type MockMemoryRepository struct {
	mock.Mock
}

func (m *MockMemoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockMemoryRepository) SearchByVector(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, sessionID, memoryIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) SearchByVectorCrossSessions(ctx context.Context, tenantID uuid.UUID, memoryIDs []uuid.UUID) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, memoryIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) UpdateAccessStats(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	args := m.Called(ctx, tenantID, memoryID)
	return args.Error(0)
}

func (m *MockMemoryRepository) DeleteByStrategy(ctx context.Context, tenantID uuid.UUID, strategy repository.DeleteStrategy, mode repository.DeleteMode) (int64, error) {
	args := m.Called(ctx, tenantID, strategy, mode)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMemoryRepository) HardDelete(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	args := m.Called(ctx, tenantID, memoryID)
	return args.Error(0)
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

func (m *MockContextRepository) Create(ctx context.Context, context *model.ConversationContext) error {
	args := m.Called(ctx, context)
	return args.Error(0)
}

// MockSummaryRepository 摘要仓储 Mock
type MockSummaryRepository struct {
	mock.Mock
}

func (m *MockSummaryRepository) GetLatestBySessionID(ctx context.Context, tenantID, sessionID uuid.UUID) (*model.ConversationSummary, error) {
	args := m.Called(ctx, tenantID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationSummary), args.Error(1)
}

func (m *MockSummaryRepository) Create(ctx context.Context, summary *model.ConversationSummary) error {
	args := m.Called(ctx, summary)
	return args.Error(0)
}

// MockUserRepository 用户仓储 Mock
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByIDOnly(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// MockVectorService 向量服务 Mock
type MockVectorService struct {
	mock.Mock
}

func (m *MockVectorService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

// MockTokenManager Token管理器 Mock
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) CalculateContextTokens(ctx context.Context, messages []*model.ChatMessage, memories []*model.ConversationMemory, summary *model.ConversationSummary) (int, error) {
	args := m.Called(ctx, messages, memories, summary)
	return args.Int(0), args.Error(1)
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

// createTestContext 创建测试上下文（带JWT声明）
func createTestContext(tenantID, userID string, roles []string) context.Context {
	ctx := context.Background()
	claims := &model.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userID,
		},
		TenantID: tenantID,
		Roles:    roles,
	}
	// 使用 context.WithValue 存储 JWT 声明
	return context.WithValue(ctx, "jwt_claims", claims)
}

// createTestSession 创建测试会话
func createTestSession(tenantID, userID, sessionID string) *model.ChatSession {
	uid, _ := uuid.Parse(userID)
	sid, _ := uuid.Parse(sessionID)
	
	return &model.ChatSession{
		ID:        sid,
		UserID:    uid,
		CreatedBy: uid,
		Title:     "测试会话",
		ModelName: "gemini-2.0-flash",
	}
}

// createTestUser 创建测试用户
func createTestUser(tenantID, userID string) *model.User {
	tid, _ := uuid.Parse(tenantID)
	uid, _ := uuid.Parse(userID)
	
	return &model.User{
		ID:       uid,
		TenantID: tid,
		Email:    "test@example.com",
	}
}

// createTestMessages 创建测试消息列表
func createTestMessages(count int) []*model.ChatMessage {
	messages := make([]*model.ChatMessage, count)
	for i := 0; i < count; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: "测试消息内容",
		}
	}
	return messages
}

// createTestMemories 创建测试记忆列表
func createTestMemories(count int) []*model.ConversationMemory {
	memories := make([]*model.ConversationMemory, count)
	for i := 0; i < count; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    "测试记忆内容",
			Importance: 0.8,
		}
	}
	return memories
}

// ========== BuildContext 测试 ==========

func TestContextService_BuildContext_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID, userID, sessionID)
	user := createTestUser(tenantID, userID)
	messages := createTestMessages(5)

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", ctx, messages, mock.Anything, mock.Anything).Return(1000, nil)

	// 执行测试
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       2000,
		Strategy:        "balanced",
		IncludeSummary:  false,
		IncludeLongTerm: false,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sessionID, result.SessionID)
	assert.Equal(t, 5, len(result.ShortTermMessages))
	assert.Equal(t, 1000, result.TotalTokens)
	assert.Equal(t, "balanced", result.Strategy)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
}

func TestContextService_BuildContext_WithLongTermMemory(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID, userID, sessionID)
	user := createTestUser(tenantID, userID)
	messages := createTestMessages(5)
	embedding := []float32{0.1, 0.2, 0.3}

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 10).Return(messages, nil)
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试查询").Return(embedding, nil)
	mockTokenMgr.On("CalculateContextTokens", ctx, messages, mock.Anything, mock.Anything).Return(1000, nil)

	// 执行测试
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       2000,
		Strategy:        "balanced",
		IncludeSummary:  false,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 验证 Mock 调用
	mockVectorSvc.AssertExpectations(t)
}

func TestContextService_BuildContext_UnauthorizedAccess(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	otherTenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望 - 会话属于其他租户
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(otherTenantID, userID, sessionID)
	user := createTestUser(otherTenantID, userID)

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)

	// 执行测试
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       2000,
		Strategy:        "balanced",
		IncludeSummary:  false,
		IncludeLongTerm: false,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证结果 - 应该返回权限错误
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "权限不足")

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestContextService_BuildContext_SessionNotFound(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望 - 会话不存在
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	mockSessionRepo.On("GetByID", ctx, sessionID).Return(nil, gorm.ErrRecordNotFound)

	// 执行测试
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       2000,
		Strategy:        "balanced",
		IncludeSummary:  false,
		IncludeLongTerm: false,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "会话不存在")

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
}


// ========== OptimizeContext 测试 ==========

func TestContextService_OptimizeContext_Aggressive(t *testing.T) {
	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 准备测试数据
	ctx := context.Background()
	messages := createTestMessages(10)
	memories := createTestMemories(5)

	// 设置 Mock 期望
	mockTokenMgr.On("CalculateContextTokens", ctx, mock.Anything, mock.Anything, mock.Anything).Return(500, nil)

	// 执行测试
	req := OptimizeContextRequest{
		Context: &ContextResult{
			SessionID:         uuid.New().String(),
			ShortTermMessages: messages,
			LongTermMemories:  memories,
			TotalTokens:       3000,
			Strategy:          "aggressive",
		},
		TargetTokens:    1000,
		Strategy:        "aggressive",
		PreserveSummary: false,
	}

	result, err := service.OptimizeContext(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.TotalTokens <= req.TargetTokens)
	assert.Equal(t, 0, len(result.LongTermMemories)) // 激进策略应该清空长期记忆

	// 验证 Mock 调用
	mockTokenMgr.AssertExpectations(t)
}

func TestContextService_OptimizeContext_Balanced(t *testing.T) {
	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 准备测试数据
	ctx := context.Background()
	messages := createTestMessages(10)
	memories := createTestMemories(5)

	// 设置 Mock 期望
	mockTokenMgr.On("CalculateContextTokens", ctx, mock.Anything, mock.Anything, mock.Anything).Return(800, nil)

	// 执行测试
	req := OptimizeContextRequest{
		Context: &ContextResult{
			SessionID:         uuid.New().String(),
			ShortTermMessages: messages,
			LongTermMemories:  memories,
			TotalTokens:       3000,
			Strategy:          "balanced",
		},
		TargetTokens:    1000,
		Strategy:        "balanced",
		PreserveSummary: false,
	}

	result, err := service.OptimizeContext(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.TotalTokens <= req.TargetTokens)

	// 验证 Mock 调用
	mockTokenMgr.AssertExpectations(t)
}

// ========== GetContextConfig 测试 ==========

func TestContextService_GetContextConfig_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID, userID, sessionID)
	user := createTestUser(tenantID, userID)
	
	sid, _ := uuid.Parse(sessionID)
	contextConfig := &model.ConversationContext{
		SessionID:       sid,
		MaxTokens:       2000,
		Strategy:        "balanced",
		ShortTermWindow: 10,
	}

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)
	mockContextRepo.On("GetBySessionID", ctx, sessionID).Return(contextConfig, nil)

	// 执行测试
	result, err := service.GetContextConfig(ctx, sessionID)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2000, result.MaxTokens)
	assert.Equal(t, "balanced", result.Strategy)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
}

// ========== UpdateContextConfig 测试 ==========

func TestContextService_UpdateContextConfig_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID, userID, sessionID)
	user := createTestUser(tenantID, userID)
	
	sid, _ := uuid.Parse(sessionID)
	contextConfig := &model.ConversationContext{
		SessionID:       sid,
		MaxTokens:       3000,
		Strategy:        "aggressive",
		ShortTermWindow: 15,
	}

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)
	mockContextRepo.On("Update", ctx, contextConfig).Return(nil)

	// 执行测试
	err := service.UpdateContextConfig(ctx, sessionID, contextConfig)

	// 验证结果
	assert.NoError(t, err)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockContextRepo.AssertExpectations(t)
}

func TestContextService_UpdateContextConfig_InvalidParams(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID, userID, []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID, userID, sessionID)
	user := createTestUser(tenantID, userID)
	
	sid, _ := uuid.Parse(sessionID)
	contextConfig := &model.ConversationContext{
		SessionID:       sid,
		MaxTokens:       -100, // 无效值
		Strategy:        "balanced",
		ShortTermWindow: 10,
	}

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID).Return(user, nil)

	// 执行测试
	err := service.UpdateContextConfig(ctx, sessionID, contextConfig)

	// 验证结果 - 应该返回参数错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxTokens")

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// ========== 权限验证测试 ==========

func TestContextService_SystemAdminCanAccessAllSessions(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New().String()
	otherTenantID := uuid.New().String()
	userID := uuid.New().String()
	sessionID := uuid.New().String()

	// 创建 Mock 对象
	mockSessionRepo := new(MockSessionRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockContextRepo := new(MockContextRepository)
	mockSummaryRepo := new(MockSummaryRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewContextService(
		mockSessionRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockContextRepo,
		mockSummaryRepo,
		mockUserRepo,
		mockVectorSvc,
		mockTokenMgr,
	)

	// 设置 Mock 期望 - 平台管理员访问其他租户的会话
	ctx := createTestContext(tenantID, userID, []string{model.RoleSystemAdmin})
	session := createTestSession(otherTenantID, userID, sessionID)
	messages := createTestMessages(5)

	mockSessionRepo.On("GetByID", ctx, sessionID).Return(session, nil)
	mockMessageRepo.On("GetLatestMessages", ctx, sessionID, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", ctx, messages, mock.Anything, mock.Anything).Return(1000, nil)

	// 执行测试
	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       2000,
		Strategy:        "balanced",
		IncludeSummary:  false,
		IncludeLongTerm: false,
		ShortTermWindow: 10,
	}

	result, err := service.BuildContext(ctx, req)

	// 验证结果 - 平台管理员应该可以访问
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}
