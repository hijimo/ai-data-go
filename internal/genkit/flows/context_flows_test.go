package flows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
)

// MockContextService 是 ContextService 的 mock 实现
type MockContextService struct {
	mock.Mock
}

func (m *MockContextService) BuildContext(ctx context.Context, req service.BuildContextRequest) (*service.ContextResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ContextResult), args.Error(1)
}

func (m *MockContextService) OptimizeContext(ctx context.Context, req service.OptimizeContextRequest) (*service.ContextResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ContextResult), args.Error(1)
}

func (m *MockContextService) GetContextConfig(ctx context.Context, sessionID string) (*model.ConversationContext, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ConversationContext), args.Error(1)
}

func (m *MockContextService) UpdateContextConfig(ctx context.Context, sessionID string, config *model.ConversationContext) error {
	args := m.Called(ctx, sessionID, config)
	return args.Error(0)
}

// TestValidateContextInput 测试输入参数验证
func TestValidateContextInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ContextBuildInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				IncludeSummary:  true,
				IncludeLongTerm: true,
				ShortTermWindow: 10,
			},
			wantErr: false,
		},
		{
			name: "会话ID为空",
			input: ContextBuildInput{
				SessionID:       "",
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "用户查询为空",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "MaxTokens过小",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       50,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "MaxTokens过大",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       50000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "无效的策略",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "invalid",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "ShortTermWindow过小",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 0,
			},
			wantErr: true,
		},
		{
			name: "ShortTermWindow过大",
			input: ContextBuildInput{
				SessionID:       uuid.New().String(),
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContextInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConvertSummary 测试摘要转换
func TestConvertSummary(t *testing.T) {
	t.Run("nil摘要", func(t *testing.T) {
		result := convertSummary(nil)
		assert.Nil(t, result)
	})

	t.Run("有效摘要", func(t *testing.T) {
		now := time.Now()
		summary := &model.ConversationSummary{
			Content:      "这是一个测试摘要",
			TokenCount:   100,
			MessageCount: 10,
			CreatedAt:    now,
		}

		result := convertSummary(summary)
		assert.NotNil(t, result)
		assert.Equal(t, "这是一个测试摘要", result.Content)
		assert.Equal(t, 100, result.TokenCount)
		assert.Equal(t, now.Format(time.RFC3339), result.CreatedAt)
		assert.Equal(t, "10条消息", result.Coverage)
	})
}

// TestConvertMemories 测试记忆列表转换
func TestConvertMemories(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		result := convertMemories(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("有效记忆列表", func(t *testing.T) {
		now := time.Now()
		memories := []*model.ConversationMemory{
			{
				ID:         uuid.New(),
				Content:    "记忆1",
				TokenCount: 50,
				Importance: 0.8,
				CreatedAt:  now,
			},
			{
				ID:         uuid.New(),
				Content:    "记忆2",
				TokenCount: 60,
				Importance: 0.9,
				CreatedAt:  now,
			},
		}

		result := convertMemories(memories)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "记忆1", result[0].Content)
		assert.Equal(t, 50, result[0].TokenCount)
		assert.Equal(t, float32(0.8), result[0].Importance)
		assert.Equal(t, "记忆2", result[1].Content)
	})
}

// TestConvertMessages 测试消息列表转换
func TestConvertMessages(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		result := convertMessages(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("有效消息列表", func(t *testing.T) {
		now := time.Now()
		messages := []*model.ChatMessage{
			{
				ID:        uuid.New(),
				Role:      "user",
				Content:   "用户消息",
				Tokens:    10,
				CreatedAt: now,
			},
			{
				ID:        uuid.New(),
				Role:      "assistant",
				Content:   "助手回复",
				Tokens:    20,
				CreatedAt: now,
			},
		}

		result := convertMessages(messages)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "user", result[0].Role)
		assert.Equal(t, "用户消息", result[0].Content)
		assert.Equal(t, 10, result[0].TokenCount)
		assert.Equal(t, "assistant", result[1].Role)
	})
}

// TestContextBuildFlow 测试上下文构建Flow的基本逻辑
// 注意：由于Flow包含权限验证，这个测试会因为缺少JWT上下文而失败
// 实际的Flow测试应该在集成测试中进行，或者需要模拟完整的认证上下文
func TestContextBuildFlow(t *testing.T) {
	t.Run("权限验证失败", func(t *testing.T) {
		// 准备 Mock
		mockContextSvc := new(MockContextService)
		sessionID := uuid.New().String()

		// 创建 Flow 函数
		flowFunc := contextBuildFlow(mockContextSvc)

		// 执行 Flow（没有JWT上下文）
		input := ContextBuildInput{
			SessionID:       sessionID,
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  true,
			IncludeLongTerm: true,
			ShortTermWindow: 10,
		}

		ctx := context.Background()
		_, err := flowFunc(ctx, input)

		// 断言：应该返回未认证错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "未认证")
	})
}
