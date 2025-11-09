package flows

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
)

// TestMemorySearchInput_Validation 测试记忆检索输入验证
func TestMemorySearchInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemorySearchInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: false,
		},
		{
			name: "缺少会话ID",
			input: MemorySearchInput{
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "缺少查询文本",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				TopK:          5,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "TopK为0",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          0,
				MinSimilarity: 0.7,
			},
			wantErr: true,
		},
		{
			name: "相似度超出范围",
			input: MemorySearchInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				Query:         "测试查询",
				TopK:          5,
				MinSimilarity: 1.5,
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

// TestMemoryStoreInput_Validation 测试记忆存储输入验证
func TestMemoryStoreInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemoryStoreInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: false,
		},
		{
			name: "缺少会话ID",
			input: MemoryStoreInput{
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "缺少记忆类型",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				Content:    "测试内容",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "缺少内容",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Importance: 0.8,
			},
			wantErr: true,
		},
		{
			name: "重要性超出范围",
			input: MemoryStoreInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440000",
				MemoryType: "long_term",
				Content:    "测试内容",
				Importance: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemoryStoreInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMemoryCleanupInput_Validation 测试记忆清理输入验证
func TestMemoryCleanupInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   MemoryCleanupInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				Mode:      "soft",
				BatchSize: 100,
			},
			wantErr: false,
		},
		{
			name: "缺少策略",
			input: MemoryCleanupInput{
				Mode:      "soft",
				BatchSize: 100,
			},
			wantErr: true,
		},
		{
			name: "缺少模式",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				BatchSize: 100,
			},
			wantErr: true,
		},
		{
			name: "批量大小为0",
			input: MemoryCleanupInput{
				Strategy:  "expired",
				Mode:      "soft",
				BatchSize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemoryCleanupInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFormatTimePtr 测试时间指针格式化
func TestFormatTimePtr(t *testing.T) {
	tests := []struct {
		name string
		time *time.Time
		want string
	}{
		{
			name: "nil时间",
			time: nil,
			want: "",
		},
		{
			name: "有效时间",
			time: func() *time.Time {
				t := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
				return &t
			}(),
			want: "2024-01-01T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimePtr(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

// MockMemoryService 是MemoryService的mock实现
type MockMemoryService struct {
	mock.Mock
}

func (m *MockMemoryService) SearchMemories(ctx context.Context, req *service.SearchMemoriesRequest) ([]*service.MemorySearchResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.MemorySearchResult), args.Error(1)
}

func (m *MockMemoryService) StoreMemory(ctx context.Context, req *service.StoreMemoryRequest) (*model.ConversationMemory, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationMemory), args.Error(1)
}

func (m *MockMemoryService) CleanupMemories(ctx context.Context, req *service.CleanupMemoriesRequest) (*service.CleanupResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.CleanupResult), args.Error(1)
}

func (m *MockMemoryService) UpdateMemoryAccess(ctx context.Context, tenantID, memoryID uuid.UUID) error {
	args := m.Called(ctx, tenantID, memoryID)
	return args.Error(0)
}

func (m *MockMemoryService) GetMemory(ctx context.Context, tenantID, memoryID uuid.UUID) (*model.ConversationMemory, error) {
	args := m.Called(ctx, tenantID, memoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationMemory), args.Error(1)
}

// TestMemorySearchFlow 测试记忆检索Flow
func TestMemorySearchFlow(t *testing.T) {
	t.Run("成功检索记忆", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()
		memoryID1 := uuid.New()
		memoryID2 := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 准备mock返回数据
		now := time.Now()
		metadata1 := map[string]interface{}{"key": "value1"}
		metadata2 := map[string]interface{}{"key": "value2"}
		metadata1JSON, _ := json.Marshal(metadata1)
		metadata2JSON, _ := json.Marshal(metadata2)

		mockSvc.On("SearchMemories", mock.Anything, mock.MatchedBy(func(req *service.SearchMemoriesRequest) bool {
			return req.SessionID == sessionID && req.Query == "测试查询"
		})).Return([]*service.MemorySearchResult{
			{
				Memory: &model.ConversationMemory{
					ID:          memoryID1,
					SessionID:   sessionID,
					MemoryType:  "long_term",
					Content:     "记忆内容1",
					TokenCount:  50,
					Importance:  0.8,
					AccessCount: 5,
					CreatedAt:   now,
					Metadata:    metadata1JSON,
				},
				Similarity: 0.85,
				Score:      0.68,
			},
			{
				Memory: &model.ConversationMemory{
					ID:          memoryID2,
					SessionID:   sessionID,
					MemoryType:  "long_term",
					Content:     "记忆内容2",
					TokenCount:  60,
					Importance:  0.9,
					AccessCount: 3,
					CreatedAt:   now,
					Metadata:    metadata2JSON,
				},
				Similarity: 0.75,
				Score:      0.675,
			},
		}, nil)

		// 创建Flow函数
		flowFunc := memorySearchFlow(mockSvc)

		// 准备输入
		input := MemorySearchInput{
			SessionID:     sessionID.String(),
			Query:         "测试查询",
			TopK:          5,
			MinSimilarity: 0.7,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.Len(t, output.Memories, 2)
		assert.Equal(t, memoryID1.String(), output.Memories[0].ID)
		assert.Equal(t, "记忆内容1", output.Memories[0].Content)
		assert.Equal(t, float32(0.85), output.Memories[0].Similarity)
		assert.Equal(t, 2, output.TotalFound)
		assert.Equal(t, 2, output.ReturnedCount)
		assert.GreaterOrEqual(t, output.SearchTime, int64(0))
		assert.Greater(t, output.AverageSimilarity, float32(0))
		assert.Equal(t, "session", output.SearchStrategy)

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})

	t.Run("参数验证失败", func(t *testing.T) {
		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 创建Flow函数
		flowFunc := memorySearchFlow(mockSvc)

		// 准备无效输入（缺少会话ID）
		input := MemorySearchInput{
			Query:         "测试查询",
			TopK:          5,
			MinSimilarity: 0.7,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "参数验证失败")
	})

	t.Run("服务层错误处理", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望返回错误
		mockSvc.On("SearchMemories", mock.Anything, mock.Anything).Return(
			nil,
			assert.AnError,
		)

		// 创建Flow函数
		flowFunc := memorySearchFlow(mockSvc)

		// 准备输入
		input := MemorySearchInput{
			SessionID:     sessionID.String(),
			Query:         "测试查询",
			TopK:          5,
			MinSimilarity: 0.7,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "记忆检索失败")

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})
}

// TestMemoryStoreFlow 测试记忆存储Flow
func TestMemoryStoreFlow(t *testing.T) {
	t.Run("成功存储记忆", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()
		memoryID := uuid.New()
		messageID1 := uuid.New()
		messageID2 := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		now := time.Now()
		expiresAt := now.Add(30 * 24 * time.Hour)
		mockSvc.On("StoreMemory", mock.Anything, mock.MatchedBy(func(req *service.StoreMemoryRequest) bool {
			return req.SessionID == sessionID && req.MemoryType == "long_term"
		})).Return(&model.ConversationMemory{
			ID:         memoryID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "测试记忆内容",
			TokenCount: 50,
			Importance: 0.8,
			ExpiresAt:  &expiresAt,
			CreatedAt:  now,
		}, nil)

		// 创建Flow函数
		flowFunc := memoryStoreFlow(mockSvc)

		// 准备输入
		input := MemoryStoreInput{
			SessionID:      sessionID.String(),
			MessageIDs:     []string{messageID1.String(), messageID2.String()},
			MemoryType:     "long_term",
			Content:        "测试记忆内容",
			Importance:     0.8,
			ExpirationDays: 30,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, memoryID.String(), output.MemoryID)
		assert.Equal(t, sessionID.String(), output.SessionID)
		assert.Equal(t, "long_term", output.MemoryType)
		assert.Equal(t, 50, output.TokenCount)
		assert.Equal(t, float32(0.8), output.Importance)
		assert.NotEmpty(t, output.ExpiresAt)
		assert.Equal(t, "generated", output.VectorStatus)
		assert.GreaterOrEqual(t, output.StoreTime, int64(0))

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})

	t.Run("参数验证失败", func(t *testing.T) {
		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 创建Flow函数
		flowFunc := memoryStoreFlow(mockSvc)

		// 准备无效输入（缺少内容）
		input := MemoryStoreInput{
			SessionID:  uuid.New().String(),
			MemoryType: "long_term",
			Importance: 0.8,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "参数验证失败")
	})

	t.Run("服务层错误处理", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望返回错误
		mockSvc.On("StoreMemory", mock.Anything, mock.Anything).Return(
			nil,
			assert.AnError,
		)

		// 创建Flow函数
		flowFunc := memoryStoreFlow(mockSvc)

		// 准备输入
		input := MemoryStoreInput{
			SessionID:  sessionID.String(),
			MemoryType: "long_term",
			Content:    "测试内容",
			Importance: 0.8,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "记忆存储失败")

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})
}

// TestMemoryCleanupFlow 测试记忆清理Flow
func TestMemoryCleanupFlow(t *testing.T) {
	t.Run("成功清理记忆", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()
		memoryID1 := uuid.New()
		memoryID2 := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		now := time.Now()
		mockSvc.On("CleanupMemories", mock.Anything, mock.MatchedBy(func(req *service.CleanupMemoriesRequest) bool {
			return req.SessionID == sessionID && req.Strategy == "expired"
		})).Return(&service.CleanupResult{
			CleanedCount: 2,
			FreedSpace:   1024,
			Details: []service.CleanupDetail{
				{
					MemoryID:   memoryID1,
					Reason:     "已过期",
					Size:       512,
					CreatedAt:  now.Add(-60 * 24 * time.Hour),
					LastAccess: now.Add(-30 * 24 * time.Hour),
				},
				{
					MemoryID:   memoryID2,
					Reason:     "已过期",
					Size:       512,
					CreatedAt:  now.Add(-90 * 24 * time.Hour),
					LastAccess: now.Add(-60 * 24 * time.Hour),
				},
			},
			Preview: false,
		}, nil)

		// 创建Flow函数
		flowFunc := memoryCleanupFlow(mockSvc)

		// 准备输入
		input := MemoryCleanupInput{
			SessionID: sessionID.String(),
			Strategy:  "expired",
			Mode:      "soft",
			BatchSize: 100,
			Execute:   true,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, 2, output.CleanedCount)
		assert.Equal(t, int64(1024), output.FreedSpace)
		assert.Len(t, output.Details, 2)
		assert.Equal(t, memoryID1.String(), output.Details[0].MemoryID)
		assert.Equal(t, "已过期", output.Details[0].Reason)
		assert.False(t, output.Preview)
		assert.GreaterOrEqual(t, output.CleanupTime, int64(0))

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})

	t.Run("预览模式", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		mockSvc.On("CleanupMemories", mock.Anything, mock.Anything).Return(&service.CleanupResult{
			CleanedCount: 5,
			FreedSpace:   2048,
			Details:      []service.CleanupDetail{},
			Preview:      true,
		}, nil)

		// 创建Flow函数
		flowFunc := memoryCleanupFlow(mockSvc)

		// 准备输入
		input := MemoryCleanupInput{
			SessionID: sessionID.String(),
			Strategy:  "expired",
			Mode:      "soft",
			BatchSize: 100,
			Execute:   false, // 预览模式
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, 5, output.CleanedCount)
		assert.True(t, output.Preview)

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})

	t.Run("参数验证失败", func(t *testing.T) {
		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 创建Flow函数
		flowFunc := memoryCleanupFlow(mockSvc)

		// 准备无效输入（缺少策略）
		input := MemoryCleanupInput{
			Mode:      "soft",
			BatchSize: 100,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "参数验证失败")
	})

	t.Run("服务层错误处理", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望返回错误
		mockSvc.On("CleanupMemories", mock.Anything, mock.Anything).Return(
			nil,
			assert.AnError,
		)

		// 创建Flow函数
		flowFunc := memoryCleanupFlow(mockSvc)

		// 准备输入
		input := MemoryCleanupInput{
			SessionID: sessionID.String(),
			Strategy:  "expired",
			Mode:      "soft",
			BatchSize: 100,
			Execute:   true,
		}

		// 执行Flow
		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "记忆清理失败")

		// 验证mock调用
		mockSvc.AssertExpectations(t)
	})
}

// TestMemoryFlowsMonitoring 测试Flow监控指标记录
func TestMemoryFlowsMonitoring(t *testing.T) {
	t.Run("检索Flow记录监控指标", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		mockSvc.On("SearchMemories", mock.Anything, mock.Anything).Return(
			[]*service.MemorySearchResult{},
			nil,
		)

		// 创建Flow函数
		flowFunc := memorySearchFlow(mockSvc)

		// 准备输入
		input := MemorySearchInput{
			SessionID:     sessionID.String(),
			Query:         "测试查询",
			TopK:          5,
			MinSimilarity: 0.7,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, output.SearchTime, int64(0))

		// 注意：实际的监控指标验证需要在集成测试中进行
	})

	t.Run("存储Flow记录监控指标", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()
		memoryID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		now := time.Now()
		mockSvc.On("StoreMemory", mock.Anything, mock.Anything).Return(&model.ConversationMemory{
			ID:         memoryID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "测试内容",
			TokenCount: 50,
			Importance: 0.8,
			CreatedAt:  now,
		}, nil)

		// 创建Flow函数
		flowFunc := memoryStoreFlow(mockSvc)

		// 准备输入
		input := MemoryStoreInput{
			SessionID:  sessionID.String(),
			MemoryType: "long_term",
			Content:    "测试内容",
			Importance: 0.8,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, output.StoreTime, int64(0))

		// 注意：实际的监控指标验证需要在集成测试中进行
	})

	t.Run("清理Flow记录监控指标", func(t *testing.T) {
		// 准备测试数据
		sessionID := uuid.New()

		// 创建mock服务
		mockSvc := new(MockMemoryService)

		// 设置mock期望
		mockSvc.On("CleanupMemories", mock.Anything, mock.Anything).Return(&service.CleanupResult{
			CleanedCount: 0,
			FreedSpace:   0,
			Details:      []service.CleanupDetail{},
			Preview:      false,
		}, nil)

		// 创建Flow函数
		flowFunc := memoryCleanupFlow(mockSvc)

		// 准备输入
		input := MemoryCleanupInput{
			SessionID: sessionID.String(),
			Strategy:  "expired",
			Mode:      "soft",
			BatchSize: 100,
			Execute:   true,
		}

		// 执行Flow
		ctx := context.Background()
		output, err := flowFunc(ctx, input)

		// 验证结果
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, output.CleanupTime, int64(0))

		// 注意：实际的监控指标验证需要在集成测试中进行
	})
}
