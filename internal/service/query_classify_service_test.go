// Package service 查询分类服务测试
package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
)

// MockAIClient 模拟 AI 客户端
type MockAIClient struct {
	mock.Mock
}

func (m *MockAIClient) Generate(ctx context.Context, prompt string, options *service.GenerateOptions) (*service.GenerateResult, error) {
	args := m.Called(ctx, prompt, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.GenerateResult), args.Error(1)
}

func TestQueryClassifyService_SimpleQuestion(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "什么是 Go 语言？",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "simple_question", result.QueryType)
	assert.False(t, result.NeedsHistory)
	assert.False(t, result.NeedsLongTerm)
	assert.Equal(t, "short", result.RecommendedStrategy)
	assert.Greater(t, result.Confidence, 0.7)
}

func TestQueryClassifyService_FollowupWithPronouns(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query:          "它的主要特性是什么？",
		RecentMessages: []string{"什么是 Go 语言？"},
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "followup_question", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.False(t, result.NeedsLongTerm)
	assert.Equal(t, "short", result.RecommendedStrategy)
}

func TestQueryClassifyService_TimeReference(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "刚才说的那个功能怎么用？",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "followup_question", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.Equal(t, "short", result.RecommendedStrategy)
}

func TestQueryClassifyService_Comparison(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "Go 和 Python 有什么区别？",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "reference_query", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.True(t, result.NeedsLongTerm)
	assert.Equal(t, "full", result.RecommendedStrategy)
}

func TestQueryClassifyService_Summarization(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "请总结一下我们讨论的内容",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "summarization", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.False(t, result.NeedsLongTerm)
	assert.Equal(t, "full", result.RecommendedStrategy)
	assert.Greater(t, result.Confidence, 0.8)
}

func TestQueryClassifyService_Clarification(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "这个是什么意思？",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "clarification", result.QueryType)
	assert.True(t, result.NeedsHistory)
	assert.Equal(t, "short", result.RecommendedStrategy)
}

func TestQueryClassifyService_ComplexQuery(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "请详细解释 Go 语言的并发模型，包括 goroutine、channel 的工作原理，以及它们与传统线程模型的区别。",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "complex_query", result.QueryType)
	assert.False(t, result.NeedsHistory)
	assert.True(t, result.NeedsLongTerm)
	assert.Equal(t, "full", result.RecommendedStrategy)
}

func TestQueryClassifyService_EntityExtraction(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "Go 1.21 版本有什么新特性？",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Entities)
	// 应该提取到 "Go" 和 "1" 或 "21"
	assert.True(t, len(result.Entities) > 0)
}

func TestQueryClassifyService_LowConfidenceUsesDefault(t *testing.T) {
	// 准备
	mockClient := new(MockAIClient)
	mockLogger := logger.NewTestLogger()
	svc := service.NewQueryClassifyService(mockClient, mockLogger)

	// 执行一个模糊的查询
	result, err := svc.Classify(context.Background(), &service.ClassifyRequest{
		Query: "嗯...",
	})

	// 验证
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 低置信度应该使用 auto 策略
	if result.Confidence < 0.7 {
		assert.Equal(t, "auto", result.RecommendedStrategy)
	}
}
