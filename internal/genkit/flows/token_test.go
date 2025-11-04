// internal/genkit/flows/token_test.go
package flows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/service"
)

// MockTokenManager 是 TokenManager 的 mock 实现
type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) CalculateContextTokens(
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
) int {
	args := m.Called(messages, memories, summary)
	return args.Int(0)
}

func (m *MockTokenManager) CalculateTextTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

func (m *MockTokenManager) CountTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

func (m *MockTokenManager) GetBudgetStatus(ctx context.Context, req service.TokenBudgetRequest) (*service.TokenBudgetResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TokenBudgetResult), args.Error(1)
}

func (m *MockTokenManager) OptimizeContent(ctx context.Context, req service.TokenOptimizeRequest) (*service.TokenOptimizeResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TokenOptimizeResult), args.Error(1)
}

func (m *MockTokenManager) AnalyzeUsage(ctx context.Context, req service.TokenAnalysisRequest) (*service.TokenAnalysisResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TokenAnalysisResult), args.Error(1)
}

func TestValidateTokenBudgetInput(t *testing.T) {
	tests := []struct {
		name    string
		input   TokenBudgetInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的每日预算请求",
			input: TokenBudgetInput{
				TenantID:   "550e8400-e29b-41d4-a716-446655440000",
				BudgetType: "daily",
			},
			wantErr: false,
		},
		{
			name: "有效的会话预算请求",
			input: TokenBudgetInput{
				SessionID:  "550e8400-e29b-41d4-a716-446655440001",
				TenantID:   "550e8400-e29b-41d4-a716-446655440000",
				BudgetType: "session",
			},
			wantErr: false,
		},
		{
			name: "缺少租户ID",
			input: TokenBudgetInput{
				BudgetType: "daily",
			},
			wantErr: true,
			errMsg:  "租户ID不能为空",
		},
		{
			name: "缺少预算类型",
			input: TokenBudgetInput{
				TenantID: "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: true,
			errMsg:  "预算类型不能为空",
		},
		{
			name: "无效的预算类型",
			input: TokenBudgetInput{
				TenantID:   "550e8400-e29b-41d4-a716-446655440000",
				BudgetType: "invalid",
			},
			wantErr: true,
			errMsg:  "无效的预算类型",
		},
		{
			name: "会话预算缺少会话ID",
			input: TokenBudgetInput{
				TenantID:   "550e8400-e29b-41d4-a716-446655440000",
				BudgetType: "session",
			},
			wantErr: true,
			errMsg:  "会话级别预算需要提供会话ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokenBudgetInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokenOptimizeInput(t *testing.T) {
	tests := []struct {
		name    string
		input   TokenOptimizeInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的压缩请求",
			input: TokenOptimizeInput{
				Content:          "这是一段需要优化的文本内容",
				TargetTokens:     100,
				Strategy:         "compress",
				QualityThreshold: 0.7,
			},
			wantErr: false,
		},
		{
			name: "缺少内容",
			input: TokenOptimizeInput{
				TargetTokens: 100,
				Strategy:     "compress",
			},
			wantErr: true,
			errMsg:  "内容不能为空",
		},
		{
			name: "内容过长",
			input: TokenOptimizeInput{
				Content:      string(make([]byte, 10001)),
				TargetTokens: 100,
				Strategy:     "compress",
			},
			wantErr: true,
			errMsg:  "内容长度不能超过10000字符",
		},
		{
			name: "目标Token数过小",
			input: TokenOptimizeInput{
				Content:      "测试内容",
				TargetTokens: 5,
				Strategy:     "compress",
			},
			wantErr: true,
			errMsg:  "目标Token数必须在10-8000之间",
		},
		{
			name: "无效的策略",
			input: TokenOptimizeInput{
				Content:      "测试内容",
				TargetTokens: 100,
				Strategy:     "invalid",
			},
			wantErr: true,
			errMsg:  "无效的优化策略",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokenOptimizeInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokenAnalysisInput(t *testing.T) {
	tests := []struct {
		name    string
		input   TokenAnalysisInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的分析请求",
			input: TokenAnalysisInput{
				TenantID:      "550e8400-e29b-41d4-a716-446655440000",
				TimeRangeDays: 7,
				Dimensions:    []string{"usage", "trend"},
			},
			wantErr: false,
		},
		{
			name: "缺少租户ID",
			input: TokenAnalysisInput{
				TimeRangeDays: 7,
			},
			wantErr: true,
			errMsg:  "租户ID不能为空",
		},
		{
			name: "时间范围过小",
			input: TokenAnalysisInput{
				TenantID:      "550e8400-e29b-41d4-a716-446655440000",
				TimeRangeDays: 0,
			},
			wantErr: true,
			errMsg:  "时间范围必须在1-365天之间",
		},
		{
			name: "时间范围过大",
			input: TokenAnalysisInput{
				TenantID:      "550e8400-e29b-41d4-a716-446655440000",
				TimeRangeDays: 400,
			},
			wantErr: true,
			errMsg:  "时间范围必须在1-365天之间",
		},
		{
			name: "无效的分析维度",
			input: TokenAnalysisInput{
				TenantID:      "550e8400-e29b-41d4-a716-446655440000",
				TimeRangeDays: 7,
				Dimensions:    []string{"invalid"},
			},
			wantErr: true,
			errMsg:  "无效的分析维度",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokenAnalysisInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
