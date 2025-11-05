package flows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/model"
)

// MockGenkitMemoryRepository 是 GenkitMemoryRepository 的 mock 实现
type MockGenkitMemoryRepository struct {
	mock.Mock
}

func (m *MockGenkitMemoryRepository) Create(ctx context.Context, memory *model.ConversationMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockGenkitMemoryRepository) GetByID(ctx context.Context, id string) (*model.ConversationMemory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) SearchByVector(
	ctx context.Context,
	sessionID string,
	embedding pgvector.Vector,
	topK int,
	minSimilarity float32,
) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, sessionID, embedding, topK, minSimilarity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) SearchByVectorCrossSessions(
	ctx context.Context,
	tenantID string,
	embedding pgvector.Vector,
	topK int,
	minSimilarity float32,
) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, embedding, topK, minSimilarity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) SearchByVectorWithFilters(
	ctx context.Context,
	sessionID string,
	embedding pgvector.Vector,
	filters interface{},
) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, sessionID, embedding, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) UpdateAccessStats(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGenkitMemoryRepository) BatchUpdateAccessStats(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockGenkitMemoryRepository) DeleteByStrategy(
	ctx context.Context,
	tenantID string,
	strategy string,
	mode string,
	batchSize int,
) (int, error) {
	args := m.Called(ctx, tenantID, strategy, mode, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockGenkitMemoryRepository) GetExpiredMemories(ctx context.Context, tenantID string, batchSize int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, batchSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) GetLowQualityMemories(ctx context.Context, tenantID string, batchSize int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, batchSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) GetUnusedMemories(ctx context.Context, tenantID string, days int, batchSize int) ([]*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, days, batchSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ConversationMemory), args.Error(1)
}

func (m *MockGenkitMemoryRepository) CountBySession(ctx context.Context, sessionID string) (int, error) {
	args := m.Called(ctx, sessionID)
	return args.Int(0), args.Error(1)
}

func (m *MockGenkitMemoryRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *MockGenkitMemoryRepository) Update(ctx context.Context, memory *model.ConversationMemory) error {
	args := m.Called(ctx, memory)
	return args.Error(0)
}

func (m *MockGenkitMemoryRepository) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGenkitMemoryRepository) HardDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockVectorService 是 VectorService 的 mock 实现
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

func (m *MockVectorService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]float32), args.Error(1)
}

func (m *MockVectorService) StoreVector(ctx context.Context, req interface{}) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockVectorService) StoreVectors(ctx context.Context, reqs interface{}) error {
	args := m.Called(ctx, reqs)
	return args.Error(0)
}

func (m *MockVectorService) SearchVectors(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockVectorService) DeleteVector(ctx context.Context, tenantID uuid.UUID, pointID string) error {
	args := m.Called(ctx, tenantID, pointID)
	return args.Error(0)
}

func (m *MockVectorService) DeleteVectorsByFilter(ctx context.Context, tenantID uuid.UUID, filter map[string]interface{}) error {
	args := m.Called(ctx, tenantID, filter)
	return args.Error(0)
}

func (m *MockVectorService) GetEmbeddingDimension() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockVectorService) EnsureCollection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockLogger 是 Logger 的 mock 实现
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	m.Called(ctx, msg, fields)
}

// TestValidateMemorySearchInput 测试输入参数验证
func TestValidateMemorySearchInput(t *testing.T) {
	tests := []struct {
		name    string
		input   MemorySearchInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
				TimeRangeDays: 30,
				MemoryTypes:   []string{model.MemoryTypeLongTerm},
			},
			wantErr: false,
		},
		{
			name: "会话ID为空",
			input: MemorySearchInput{
				SessionID:     "",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "无效的会话ID格式",
			input: MemorySearchInput{
				SessionID:     "invalid-uuid",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "查询文本为空",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "TopK为0",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          0,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "TopK超过限制",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          25,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "最小相似度小于0",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: -0.1,
			},
			wantErr: true,
		},
		{
			name: "最小相似度大于1",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 1.5,
			},
			wantErr: true,
		},
		{
			name: "无效的记忆类型",
			input: MemorySearchInput{
				SessionID:     uuid.New().String(),
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
				MemoryTypes:   []string{"invalid_type"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemorySearchInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCalculateCosineSimilarity 测试余弦相似度计算
func TestCalculateCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		vec1     []float32
		vec2     []float32
		expected float32
	}{
		{
			name:     "相同向量",
			vec1:     []float32{1, 0, 0},
			vec2:     []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "正交向量",
			vec1:     []float32{1, 0, 0},
			vec2:     []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "相似向量",
			vec1:     []float32{1, 1, 0},
			vec2:     []float32{1, 0.5, 0},
			expected: 0.9486833, // 约等于
		},
		{
			name:     "长度不同",
			vec1:     []float32{1, 0},
			vec2:     []float32{1, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCosineSimilarity(tt.vec1, tt.vec2)
			if tt.expected == 0.0 {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.InDelta(t, tt.expected, result, 0.01)
			}
		})
	}
}

// TestMemorySearchFlow_SingleSession 测试单会话记忆搜索
func TestMemorySearchFlow_SingleSession(t *testing.T) {
	// 准备测试数据
	sessionID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()

	testMemories := []*model.ConversationMemory{
		{
			ID:           uuid.New(),
			TenantID:     tenantID,
			SessionID:    sessionID,
			MemoryType:   model.MemoryTypeLongTerm,
			Content:      "这是一条测试记忆",
			Embedding:    pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
			TokenCount:   10,
			Importance:   0.8,
			AccessCount:  5,
			LastAccessAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	// 创建 mock
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockLogger := new(MockLogger)

	// 设置 mock 期望
	queryEmbedding := []float32{0.1, 0.2, 0.3}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, "测试查询").
		Return(queryEmbedding, nil)

	mockMemoryRepo.On("SearchByVector",
		mock.Anything,
		sessionID.String(),
		mock.AnythingOfType("pgvector.Vector"),
		5,
		float32(0.7),
	).Return(testMemories, nil)

	mockMemoryRepo.On("BatchUpdateAccessStats",
		mock.Anything,
		mock.AnythingOfType("[]string"),
	).Return(nil)

	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("ErrorContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("WarnContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// 创建带有 JWT claims 的上下文
	ctx := context.Background()
	claims := &model.JWTClaims{
		TenantID: tenantID.String(),
		Roles:    []string{model.RoleTenantAdmin},
	}
	claims.Subject = uuid.New().String()
	ctx = context.WithValue(ctx, middleware.JWTClaimsKey, claims)

	// 准备输入
	input := MemorySearchInput{
		SessionID:            sessionID.String(),
		Query:                "测试查询",
		TopK:                 5,
		MinSimilarity:        0.7,
		TimeRangeDays:        0,
		MemoryTypes:          []string{},
		IncludeCrossSessions: false,
	}

	// 注意：这里我们直接测试验证逻辑，而不是完整的 Flow
	// 因为 Flow 需要 Genkit 实例，这在单元测试中比较复杂

	// 测试参数验证
	err := validateMemorySearchInput(input)
	assert.NoError(t, err)

	// 验证 mock 调用（在实际 Flow 执行后）
	// mockVectorSvc.AssertExpectations(t)
	// mockMemoryRepo.AssertExpectations(t)
}

// TestMemorySearchFlow_CrossSessions 测试跨会话记忆搜索
func TestMemorySearchFlow_CrossSessions(t *testing.T) {
	// 准备测试数据
	sessionID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()

	testMemories := []*model.ConversationMemory{
		{
			ID:           uuid.New(),
			TenantID:     tenantID,
			SessionID:    uuid.New(), // 不同的会话
			MemoryType:   model.MemoryTypeLongTerm,
			Content:      "跨会话记忆1",
			Embedding:    pgvector.NewVector([]float32{0.1, 0.2, 0.3}),
			TokenCount:   10,
			Importance:   0.8,
			AccessCount:  3,
			LastAccessAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			TenantID:     tenantID,
			SessionID:    uuid.New(), // 另一个不同的会话
			MemoryType:   model.MemoryTypeLongTerm,
			Content:      "跨会话记忆2",
			Embedding:    pgvector.NewVector([]float32{0.2, 0.3, 0.4}),
			TokenCount:   15,
			Importance:   0.9,
			AccessCount:  7,
			LastAccessAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	// 创建 mock
	mockMemoryRepo := new(MockGenkitMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockLogger := new(MockLogger)

	// 设置 mock 期望
	queryEmbedding := []float32{0.15, 0.25, 0.35}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, "跨会话查询").
		Return(queryEmbedding, nil)

	mockMemoryRepo.On("SearchByVectorCrossSessions",
		mock.Anything,
		tenantID.String(),
		mock.AnythingOfType("pgvector.Vector"),
		10,
		float32(0.6),
	).Return(testMemories, nil)

	mockMemoryRepo.On("BatchUpdateAccessStats",
		mock.Anything,
		mock.AnythingOfType("[]string"),
	).Return(nil)

	mockLogger.On("InfoContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// 创建带有 JWT claims 的上下文
	ctx := context.Background()
	claims := &model.JWTClaims{
		TenantID: tenantID.String(),
		Roles:    []string{model.RoleTenantAdmin},
	}
	claims.Subject = uuid.New().String()
	ctx = context.WithValue(ctx, middleware.JWTClaimsKey, claims)

	// 准备输入
	input := MemorySearchInput{
		SessionID:            sessionID.String(),
		Query:                "跨会话查询",
		TopK:                 10,
		MinSimilarity:        0.6,
		TimeRangeDays:        0,
		MemoryTypes:          []string{},
		IncludeCrossSessions: true,
	}

	// 测试参数验证
	err := validateMemorySearchInput(input)
	assert.NoError(t, err)

	// 在实际场景中，这里会调用 Flow 并验证结果
	// 由于需要 Genkit 实例，我们在集成测试中进行完整测试
}
