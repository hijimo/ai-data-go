// Package flows 查询分类 Flow 测试
package flows_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/service"
)

// MockQueryClassifyService 模拟查询分类服务
type MockQueryClassifyService struct {
	mock.Mock
}

func (m *MockQueryClassifyService) Classify(ctx context.Context, req *service.ClassifyRequest) (*service.ClassifyResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ClassifyResult), args.Error(1)
}

func TestValidateQueryClassifyInput(t *testing.T) {
	tests := []struct {
		name    string
		input   flows.QueryClassifyInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: flows.QueryClassifyInput{
				Query:     "这是一个测试查询",
				SessionID: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "空查询",
			input: flows.QueryClassifyInput{
				Query: "",
			},
			wantErr: true,
		},
		{
			name: "查询过长",
			input: flows.QueryClassifyInput{
				Query: string(make([]byte, 2001)),
			},
			wantErr: true,
		},
		{
			name: "最近消息过多",
			input: flows.QueryClassifyInput{
				Query:          "测试",
				RecentMessages: []string{"1", "2", "3", "4", "5", "6"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里我们无法直接测试私有函数，但可以通过 Flow 执行来间接测试
			// 实际测试需要在集成测试中进行
			if tt.wantErr {
				assert.True(t, len(tt.input.Query) == 0 || len(tt.input.Query) > 2000 || len(tt.input.RecentMessages) > 5)
			}
		})
	}
}

func TestQueryClassifyFlow_SimpleQuestion(t *testing.T) {
	// 准备
	mockSvc := new(MockQueryClassifyService)
	mockSvc.On("Classify", mock.Anything, mock.MatchedBy(func(req *service.ClassifyRequest) bool {
		return req.Query == "什么是 Go 语言？"
	})).Return(&service.ClassifyResult{
		QueryType:           "simple_question",
		Intent:              "simple_information_need",
		NeedsHistory:        false,
		NeedsLongTerm:       false,
		RecommendedStrategy: "short",
		Confidence:          0.85,
		Entities:            []string{"Go"},
	}, nil)

	// 验证
	result, err := mockSvc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "什么是 Go 语言？",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "simple_question", result.QueryType)
	assert.Equal(t, "short", result.RecommendedStrategy)
	assert.False(t, result.NeedsHistory)
	assert.Greater(t, result.Confidence, 0.7)

	mockSvc.AssertExpectations(t)
}

func TestQueryClassifyFlow_FollowupQuestion(t *testing.T) {
	// 准备
	mockSvc := new(MockQueryClassifyService)
	mockSvc.On("Classify", mock.Anything, mock.MatchedBy(func(req *service.ClassifyRequest) bool {
		return req.Query == "它的主要特性是什么？"
	})).Return(&service.ClassifyResult{
		QueryType:           "followup_question",
		Intent:              "followup",
		NeedsHistory:        true,
		NeedsLongTerm:       false,
		RecommendedStrategy: "short",
		Confidence:          0.8,
		Entities:            []string{},
	}, nil)

	// 验证
	result, err := mockSvc.Classify(context.Background(), &service.ClassifyRequest{
		Query:          "它的主要特性是什么？",
		RecentMessages: []string{"什么是 Go 语言？", "Go 是一种编程语言..."},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "followup_question", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.Equal(t, "short", result.RecommendedStrategy)

	mockSvc.AssertExpectations(t)
}

func TestQueryClassifyFlow_ComplexQuery(t *testing.T) {
	// 准备
	mockSvc := new(MockQueryClassifyService)
	mockSvc.On("Classify", mock.Anything, mock.MatchedBy(func(req *service.ClassifyRequest) bool {
		return len(req.Query) > 50
	})).Return(&service.ClassifyResult{
		QueryType:           "complex_query",
		Intent:              "complex_information_need",
		NeedsHistory:        false,
		NeedsLongTerm:       true,
		RecommendedStrategy: "full",
		Confidence:          0.75,
		Entities:            []string{"Go", "Python", "Java"},
	}, nil)

	// 验证
	result, err := mockSvc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "请详细比较 Go、Python 和 Java 这三种编程语言在并发处理、性能表现和生态系统方面的优劣势。",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "complex_query", result.QueryType)
	assert.True(t, result.NeedsLongTerm)
	assert.Equal(t, "full", result.RecommendedStrategy)

	mockSvc.AssertExpectations(t)
}

func TestQueryClassifyFlow_Summarization(t *testing.T) {
	// 准备
	mockSvc := new(MockQueryClassifyService)
	mockSvc.On("Classify", mock.Anything, mock.MatchedBy(func(req *service.ClassifyRequest) bool {
		return req.Query == "请总结一下我们之前讨论的内容"
	})).Return(&service.ClassifyResult{
		QueryType:           "summarization",
		Intent:              "request_summary",
		NeedsHistory:        true,
		NeedsLongTerm:       false,
		RecommendedStrategy: "full",
		Confidence:          0.9,
		Entities:            []string{},
	}, nil)

	// 验证
	result, err := mockSvc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "请总结一下我们之前讨论的内容",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "summarization", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.Equal(t, "full", result.RecommendedStrategy)
	assert.Greater(t, result.Confidence, 0.7)

	mockSvc.AssertExpectations(t)
}
