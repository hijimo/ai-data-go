package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/storage"
)

// ========== Mock QdrantClient ==========

type MockQdrantClient struct {
	mock.Mock
}

func (m *MockQdrantClient) SearchVectors(ctx context.Context, req *storage.SearchVectorRequest) ([]*storage.VectorSearchResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.VectorSearchResult), args.Error(1)
}

func (m *MockQdrantClient) UpsertVector(ctx context.Context, req *storage.UpsertVectorRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockQdrantClient) DeleteByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error {
	args := m.Called(ctx, tenantID, filter)
	return args.Error(0)
}

func (m *MockQdrantClient) DeleteVector(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	args := m.Called(ctx, tenantID, memoryID)
	return args.Error(0)
}

// ========== SearchMemories 测试 ==========

func TestMemoryService_SearchMemories_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())
	
	queryVector := []float32{0.1, 0.2, 0.3}
	memoryID1 := uuid.New()
	memoryID2 := uuid.New()
	
	vectorResults := []*storage.VectorSearchResult{
		{MemoryID: memoryID1, Score: 0.9},
		{MemoryID: memoryID2, Score: 0.8},
	}
	
	memories := []*model.ConversationMemory{
		{ID: memoryID1, Content: "记忆1", Importance: 0.8},
		{ID: memoryID2, Content: "记忆2", Importance: 0.7},
	}

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试查询").Return(queryVector, nil)
	mockQdrantClient.On("SearchVectors", ctx, mock.AnythingOfType("*storage.SearchVectorRequest")).Return(vectorResults, nil)
	mockMemoryRepo.On("SearchByVector", ctx, tenantID, sessionID, []uuid.UUID{memoryID1, memoryID2}).Return(memories, nil)
	mockMemoryRepo.On("UpdateAccessStats", mock.Anything, tenantID, memoryID1).Return(nil)
	mockMemoryRepo.On("UpdateAccessStats", mock.Anything, tenantID, memoryID2).Return(nil)

	// 执行测试
	req := &SearchMemoriesRequest{
		SessionID:            sessionID,
		Query:                "测试查询",
		TopK:                 5,
		MinSimilarity:        0.7,
		TimeRangeDays:        0,
		MemoryTypes:          []string{},
		IncludeCrossSessions: false,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, memoryID1, results[0].Memory.ID)
	assert.Equal(t, float32(0.9), results[0].Similarity)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockVectorSvc.AssertExpectations(t)
	mockQdrantClient.AssertExpectations(t)
	mockMemoryRepo.AssertExpectations(t)
}

func TestMemoryService_SearchMemories_CrossSessions(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())
	
	queryVector := []float32{0.1, 0.2, 0.3}
	memoryID1 := uuid.New()
	
	vectorResults := []*storage.VectorSearchResult{
		{MemoryID: memoryID1, Score: 0.9},
	}
	
	memories := []*model.ConversationMemory{
		{ID: memoryID1, Content: "跨会话记忆", Importance: 0.8},
	}

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试查询").Return(queryVector, nil)
	mockQdrantClient.On("SearchVectors", ctx, mock.AnythingOfType("*storage.SearchVectorRequest")).Return(vectorResults, nil)
	mockMemoryRepo.On("SearchByVectorCrossSessions", ctx, tenantID, []uuid.UUID{memoryID1}).Return(memories, nil)
	mockMemoryRepo.On("UpdateAccessStats", mock.Anything, tenantID, memoryID1).Return(nil)

	// 执行测试
	req := &SearchMemoriesRequest{
		SessionID:            sessionID,
		Query:                "测试查询",
		TopK:                 5,
		MinSimilarity:        0.7,
		IncludeCrossSessions: true,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 1, len(results))

	// 验证 Mock 调用
	mockMemoryRepo.AssertCalled(t, "SearchByVectorCrossSessions", ctx, tenantID, []uuid.UUID{memoryID1})
}

func TestMemoryService_SearchMemories_InvalidParams(t *testing.T) {
	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	ctx := context.Background()

	// 测试空会话ID
	req := &SearchMemoriesRequest{
		SessionID: uuid.Nil,
		Query:     "测试查询",
	}
	results, err := service.SearchMemories(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "会话ID不能为空")

	// 测试空查询
	req = &SearchMemoriesRequest{
		SessionID: uuid.New(),
		Query:     "",
	}
	results, err = service.SearchMemories(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "查询文本不能为空")
}


// ========== StoreMemory 测试 ==========

func TestMemoryService_StoreMemory_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())
	
	embedding := []float32{0.1, 0.2, 0.3}

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试记忆内容").Return(embedding, nil)
	mockTokenMgr.On("CalculateTokens", ctx, "测试记忆内容", "").Return(100, nil)
	mockMemoryRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationMemory")).Return(nil)
	mockQdrantClient.On("UpsertVector", ctx, mock.AnythingOfType("*storage.UpsertVectorRequest")).Return(nil)

	// 执行测试
	req := &StoreMemoryRequest{
		SessionID:      sessionID,
		Content:        "测试记忆内容",
		MemoryType:     "conversation",
		Importance:     0.8,
		ExpirationDays: 30,
		Metadata:       map[string]interface{}{"key": "value"},
	}

	result, err := service.StoreMemory(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "测试记忆内容", result.Content)
	assert.Equal(t, "conversation", result.MemoryType)
	assert.Equal(t, float32(0.8), result.Importance)
	assert.Equal(t, 100, result.TokenCount)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockVectorSvc.AssertExpectations(t)
	mockTokenMgr.AssertExpectations(t)
	mockMemoryRepo.AssertExpectations(t)
	mockQdrantClient.AssertExpectations(t)
}

func TestMemoryService_StoreMemory_VectorGenerationFailed(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())
	
	embedding := []float32{0.1, 0.2, 0.3}

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	// 第一次失败，第二次成功（重试机制）
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试记忆内容").Return(nil, assert.AnError).Once()
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试记忆内容").Return(embedding, nil).Once()
	mockTokenMgr.On("CalculateTokens", ctx, "测试记忆内容", "").Return(100, nil)
	mockMemoryRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationMemory")).Return(nil)
	mockQdrantClient.On("UpsertVector", ctx, mock.AnythingOfType("*storage.UpsertVectorRequest")).Return(nil)

	// 执行测试
	req := &StoreMemoryRequest{
		SessionID:  sessionID,
		Content:    "测试记忆内容",
		MemoryType: "conversation",
		Importance: 0.8,
	}

	result, err := service.StoreMemory(ctx, req)

	// 验证结果 - 重试后应该成功
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 验证 Mock 调用
	mockVectorSvc.AssertNumberOfCalls(t, "GenerateEmbedding", 2)
}

func TestMemoryService_StoreMemory_InvalidParams(t *testing.T) {
	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	ctx := context.Background()

	// 测试空会话ID
	req := &StoreMemoryRequest{
		SessionID:  uuid.Nil,
		Content:    "测试内容",
		MemoryType: "conversation",
	}
	result, err := service.StoreMemory(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "会话ID不能为空")

	// 测试空内容
	req = &StoreMemoryRequest{
		SessionID:  uuid.New(),
		Content:    "",
		MemoryType: "conversation",
	}
	result, err = service.StoreMemory(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "记忆内容不能为空")

	// 测试空记忆类型
	req = &StoreMemoryRequest{
		SessionID:  uuid.New(),
		Content:    "测试内容",
		MemoryType: "",
	}
	result, err = service.StoreMemory(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "记忆类型不能为空")
}

// ========== CleanupMemories 测试 ==========

func TestMemoryService_CleanupMemories_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockMemoryRepo.On("DeleteByStrategy", ctx, tenantID, repository.DeleteStrategyExpired, repository.DeleteModeSoft).Return(int64(10), nil)

	// 执行测试
	req := &CleanupMemoriesRequest{
		SessionID: sessionID,
		Strategy:  "expired",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, result.CleanedCount)
	assert.False(t, result.Preview)

	// 验证 Mock 调用
	mockMemoryRepo.AssertExpectations(t)
}

func TestMemoryService_CleanupMemories_HardDelete(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	session := createTestSession(tenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(tenantID.String(), userID.String())

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockMemoryRepo.On("DeleteByStrategy", ctx, tenantID, repository.DeleteStrategyLowQuality, repository.DeleteModeHard).Return(int64(5), nil)
	mockQdrantClient.On("DeleteByFilter", ctx, tenantID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// 执行测试
	req := &CleanupMemoriesRequest{
		SessionID: sessionID,
		Strategy:  "low_quality",
		Mode:      "hard",
		BatchSize: 100,
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.CleanedCount)

	// 验证 Mock 调用 - 硬删除应该同时删除 Qdrant 向量
	mockQdrantClient.AssertExpectations(t)
}

func TestMemoryService_CleanupMemories_InvalidStrategy(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})

	// 执行测试 - 无效的策略
	req := &CleanupMemoriesRequest{
		SessionID: uuid.New(),
		Strategy:  "invalid_strategy",
		Mode:      "soft",
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "无效的清理策略")
}

// ========== UpdateMemoryAccess 测试 ==========

func TestMemoryService_UpdateMemoryAccess_Success(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	userID := uuid.New()
	memoryID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})
	mockMemoryRepo.On("UpdateAccessStats", ctx, tenantID, memoryID).Return(nil)

	// 执行测试
	err := service.UpdateMemoryAccess(ctx, tenantID, memoryID)

	// 验证结果
	assert.NoError(t, err)

	// 验证 Mock 调用
	mockMemoryRepo.AssertExpectations(t)
}

func TestMemoryService_UpdateMemoryAccess_UnauthorizedTenant(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	userID := uuid.New()
	memoryID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望 - 租户管理员尝试访问其他租户的记忆
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleTenantAdmin})

	// 执行测试
	err := service.UpdateMemoryAccess(ctx, otherTenantID, memoryID)

	// 验证结果 - 应该返回权限错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "权限不足")
}

// ========== 权限验证测试 ==========

func TestMemoryService_SystemAdminCanAccessAllTenants(t *testing.T) {
	// 准备测试数据
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	// 创建 Mock 对象
	mockMemoryRepo := new(MockMemoryRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockUserRepo := new(MockUserRepository)
	mockVectorSvc := new(MockVectorService)
	mockQdrantClient := new(MockQdrantClient)
	mockTokenMgr := new(MockTokenManager)

	// 创建服务实例
	service := NewMemoryService(
		mockMemoryRepo,
		mockSessionRepo,
		mockUserRepo,
		mockVectorSvc,
		mockQdrantClient,
		mockTokenMgr,
	)

	// 设置 Mock 期望 - 平台管理员访问其他租户的会话
	ctx := createTestContext(tenantID.String(), userID.String(), []string{model.RoleSystemAdmin})
	session := createTestSession(otherTenantID.String(), userID.String(), sessionID.String())
	user := createTestUser(otherTenantID.String(), userID.String())
	
	queryVector := []float32{0.1, 0.2, 0.3}
	memoryID1 := uuid.New()
	
	vectorResults := []*storage.VectorSearchResult{
		{MemoryID: memoryID1, Score: 0.9},
	}
	
	memories := []*model.ConversationMemory{
		{ID: memoryID1, Content: "记忆1", Importance: 0.8},
	}

	mockSessionRepo.On("GetByID", ctx, sessionID.String()).Return(session, nil)
	mockUserRepo.On("GetByIDOnly", ctx, userID.String()).Return(user, nil)
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试查询").Return(queryVector, nil)
	mockQdrantClient.On("SearchVectors", ctx, mock.AnythingOfType("*storage.SearchVectorRequest")).Return(vectorResults, nil)
	mockMemoryRepo.On("SearchByVector", ctx, otherTenantID, sessionID, []uuid.UUID{memoryID1}).Return(memories, nil)
	mockMemoryRepo.On("UpdateAccessStats", mock.Anything, otherTenantID, memoryID1).Return(nil)

	// 执行测试
	req := &SearchMemoriesRequest{
		SessionID:     sessionID,
		Query:         "测试查询",
		TopK:          5,
		MinSimilarity: 0.7,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证结果 - 平台管理员应该可以访问
	assert.NoError(t, err)
	assert.NotNil(t, results)

	// 验证 Mock 调用
	mockSessionRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}
