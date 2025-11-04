package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
)

// MockMemoryRepository 模拟记忆仓库
type MockMemoryRepository struct {
	mock.Mock
}

func (m *MockMemoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockMemoryRepository) GetByID(ctx context.Context, id string) (*model.ConversationMemory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) GetBySessionID(ctx context.Context, sessionID string, limit int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) SearchByVector(ctx context.Context, sessionID string, embedding pgvector.Vector, topK int, minSimilarity float32) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, sessionID, embedding, topK, minSimilarity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) SearchByVectorCrossSessions(ctx context.Context, tenantID string, embedding pgvector.Vector, topK int, minSimilarity float32) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, embedding, topK, minSimilarity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryRepository) UpdateAccessStats(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemoryRepository) DeleteByStrategy(ctx context.Context, tenantID string, strategy string, mode string, batchSize int) (int, error) {
	args := m.Called(ctx, tenantID, strategy, mode, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockMemoryRepository) GetExpiredMemories(ctx context.Context, batchSize int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, batchSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

// MockVectorService 模拟向量服务
type MockVectorService struct {
	mock.Mock
}

func (m *MockVectorService) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return pgvector.Vector{}, args.Error(1)
	}
	return args.Get(0).(pgvector.Vector), args.Error(1)
}

func (m *MockVectorService) GenerateEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	args := m.Called(ctx, texts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pgvector.Vector), args.Error(1)
}

// TestMemoryService_StoreMemory 测试存储记忆
func TestMemoryService_StoreMemory(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "测试内容").Return(embedding, nil)

	// 模拟创建记忆
	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.ConversationMemory")).Return(nil)

	// 执行存储
	req := StoreMemoryRequest{
		TenantID:   tenantID,
		SessionID:  sessionID,
		Content:    "测试内容",
		MemoryType: "long_term",
		Importance: 0.8,
	}

	memory, err := service.StoreMemory(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, memory)
	assert.Equal(t, "测试内容", memory.Content)
	assert.Equal(t, float32(0.8), memory.Importance)

	mockVectorSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// TestMemoryService_SearchMemories 测试搜索记忆
func TestMemoryService_SearchMemories(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "搜索查询").Return(embedding, nil)

	// 模拟搜索结果
	memories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			Content:    "相关记忆1",
			Importance: 0.9,
		},
		{
			ID:         uuid.New(),
			Content:    "相关记忆2",
			Importance: 0.8,
		},
	}
	mockRepo.On("SearchByVector", ctx, sessionID, embedding, 5, float32(0.7)).Return(memories, nil)

	// 模拟更新访问统计
	mockRepo.On("UpdateAccessStats", ctx, mock.AnythingOfType("string")).Return(nil)

	// 执行搜索
	req := SearchMemoriesRequest{
		TenantID:      tenantID,
		SessionID:     sessionID,
		Query:         "搜索查询",
		TopK:          5,
		MinSimilarity: 0.7,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "相关记忆1", results[0].Memory.Content)

	mockVectorSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// TestMemoryService_SearchMemories_CrossSessions 测试跨会话搜索
func TestMemoryService_SearchMemories_CrossSessions(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "跨会话查询").Return(embedding, nil)

	// 模拟跨会话搜索结果
	memories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			SessionID:  uuid.New(),
			Content:    "会话1的记忆",
			Importance: 0.9,
		},
		{
			ID:         uuid.New(),
			SessionID:  uuid.New(),
			Content:    "会话2的记忆",
			Importance: 0.85,
		},
	}
	mockRepo.On("SearchByVectorCrossSessions", ctx, tenantID, embedding, 5, float32(0.7)).Return(memories, nil)

	// 模拟更新访问统计
	mockRepo.On("UpdateAccessStats", ctx, mock.AnythingOfType("string")).Return(nil)

	// 执行跨会话搜索
	req := SearchMemoriesRequest{
		TenantID:             tenantID,
		Query:                "跨会话查询",
		TopK:                 5,
		MinSimilarity:        0.7,
		IncludeCrossSessions: true,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	mockVectorSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// TestMemoryService_CleanupMemories_Expired 测试清理过期记忆
func TestMemoryService_CleanupMemories_Expired(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 模拟删除操作
	mockRepo.On("DeleteByStrategy", ctx, tenantID, "expired", "soft", 100).Return(5, nil)

	// 执行清理
	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "expired",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, result.CleanedCount)

	mockRepo.AssertExpectations(t)
}

// TestMemoryService_CleanupMemories_LowQuality 测试清理低质量记忆
func TestMemoryService_CleanupMemories_LowQuality(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 模拟删除操作
	mockRepo.On("DeleteByStrategy", ctx, tenantID, "low_quality", "soft", 100).Return(3, nil)

	// 执行清理
	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "low_quality",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.CleanedCount)

	mockRepo.AssertExpectations(t)
}

// TestMemoryService_CleanupMemories_PreviewMode 测试预览模式
func TestMemoryService_CleanupMemories_PreviewMode(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 预览模式不应该调用删除操作
	// 不设置 mock 期望

	// 执行预览
	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "expired",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   false, // 预览模式
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 预览模式应该返回0
	assert.Equal(t, 0, result.CleanedCount)
}

// TestMemoryService_UpdateMemoryAccess 测试更新记忆访问统计
func TestMemoryService_UpdateMemoryAccess(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	memoryID := uuid.New().String()

	// 模拟更新操作
	mockRepo.On("UpdateAccessStats", ctx, memoryID).Return(nil)

	// 执行更新
	err := service.UpdateMemoryAccess(ctx, memoryID)

	// 验证
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// TestMemoryService_StoreMemory_WithExpiration 测试存储带过期时间的记忆
func TestMemoryService_StoreMemory_WithExpiration(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "临时内容").Return(embedding, nil)

	// 模拟创建记忆
	mockRepo.On("Create", ctx, mock.MatchedBy(func(m *model.ConversationMemory) bool {
		return m.ExpiresAt != nil
	})).Return(nil)

	// 执行存储
	req := StoreMemoryRequest{
		TenantID:       tenantID,
		SessionID:      sessionID,
		Content:        "临时内容",
		MemoryType:     "short_term",
		Importance:     0.5,
		ExpirationDays: 7,
	}

	memory, err := service.StoreMemory(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, memory)
	assert.NotNil(t, memory.ExpiresAt)

	mockVectorSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// TestMemoryService_StoreMemory_WithMetadata 测试存储带元数据的记忆
func TestMemoryService_StoreMemory_WithMetadata(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	// 模拟向量生成
	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", ctx, "带元数据的内容").Return(embedding, nil)

	// 模拟创建记忆
	mockRepo.On("Create", ctx, mock.MatchedBy(func(m *model.ConversationMemory) bool {
		return m.Metadata != nil && m.Metadata["source"] == "user_input"
	})).Return(nil)

	// 执行存储
	req := StoreMemoryRequest{
		TenantID:   tenantID,
		SessionID:  sessionID,
		Content:    "带元数据的内容",
		MemoryType: "long_term",
		Importance: 0.8,
		Metadata: map[string]interface{}{
			"source":    "user_input",
			"timestamp": time.Now().Unix(),
		},
	}

	memory, err := service.StoreMemory(ctx, req)

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, memory)
	assert.NotNil(t, memory.Metadata)
	assert.Equal(t, "user_input", memory.Metadata["source"])

	mockVectorSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// TestMemoryService_SearchMemories_EmptyQuery 测试空查询
func TestMemoryService_SearchMemories_EmptyQuery(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	// 空查询应该返回错误或空结果
	req := SearchMemoriesRequest{
		TenantID:      tenantID,
		SessionID:     sessionID,
		Query:         "",
		TopK:          5,
		MinSimilarity: 0.7,
	}

	results, err := service.SearchMemories(ctx, req)

	// 验证
	assert.Error(t, err)
	assert.Nil(t, results)
}

// TestMemoryService_CleanupMemories_InvalidStrategy 测试无效的清理策略
func TestMemoryService_CleanupMemories_InvalidStrategy(t *testing.T) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "test")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 无效策略应该返回错误
	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "invalid_strategy",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   true,
	}

	result, err := service.CleanupMemories(ctx, req)

	// 验证
	assert.Error(t, err)
	assert.Nil(t, result)
}
