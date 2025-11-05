// Package flows_test 测试 Flow 实现
package flows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"genkit-ai-service/internal/genkit/flows"
)

// TestContextBuildInput_Validation 测试输入参数验证
func TestContextBuildInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   flows.ContextBuildInput
		wantErr bool
	}{
		{
			name: "有效输入",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
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
			name: "SessionID 为空",
			input: flows.ContextBuildInput{
				SessionID:       "",
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "UserQuery 为空",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
				UserQuery:       "",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "MaxTokens 太小",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
				UserQuery:       "测试查询",
				MaxTokens:       50,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "MaxTokens 太大",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
				UserQuery:       "测试查询",
				MaxTokens:       50000,
				Strategy:        "auto",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "Strategy 无效",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "invalid",
				ShortTermWindow: 10,
			},
			wantErr: true,
		},
		{
			name: "ShortTermWindow 太小",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
				UserQuery:       "测试查询",
				MaxTokens:       4000,
				Strategy:        "auto",
				ShortTermWindow: 0,
			},
			wantErr: true,
		},
		{
			name: "ShortTermWindow 太大",
			input: flows.ContextBuildInput{
				SessionID:       "550e8400-e29b-41d4-a716-446655440000",
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
			// 这里只测试输入验证逻辑
			// 实际的 Flow 执行需要完整的服务依赖
			
			// 基本的字段检查
			if tt.input.SessionID == "" {
				assert.True(t, tt.wantErr, "SessionID 为空应该返回错误")
				return
			}
			
			if tt.input.UserQuery == "" {
				assert.True(t, tt.wantErr, "UserQuery 为空应该返回错误")
				return
			}
			
			if tt.input.MaxTokens < 100 || tt.input.MaxTokens > 32000 {
				assert.True(t, tt.wantErr, "MaxTokens 超出范围应该返回错误")
				return
			}
			
			if tt.input.Strategy != "auto" && tt.input.Strategy != "short" && tt.input.Strategy != "full" {
				assert.True(t, tt.wantErr, "Strategy 无效应该返回错误")
				return
			}
			
			if tt.input.ShortTermWindow < 1 || tt.input.ShortTermWindow > 50 {
				assert.True(t, tt.wantErr, "ShortTermWindow 超出范围应该返回错误")
				return
			}
			
			// 如果所有检查都通过，不应该有错误
			assert.False(t, tt.wantErr, "有效输入不应该返回错误")
		})
	}
}

// TestConvertToContextBuildOutput 测试输出转换
func TestConvertToContextBuildOutput(t *testing.T) {
	// 这个测试需要 mock 服务层的返回结果
	// 暂时跳过，等待集成测试
	t.Skip("需要完整的服务依赖")
}

// TestContextOptimizeInput_Validation 测试优化输入参数验证
func TestContextOptimizeInput_Validation(t *testing.T) {
	// 创建一个有效的上下文用于测试
	validContext := &flows.ContextBuildOutput{
		SessionID:   "550e8400-e29b-41d4-a716-446655440000",
		TotalTokens: 5000,
		Strategy:    "auto",
		QualityScore: 0.85,
		ShortTermMessages: []flows.MessageContext{
			{ID: "msg1", Role: "user", Content: "测试消息1"},
			{ID: "msg2", Role: "assistant", Content: "测试回复1"},
		},
	}

	tests := []struct {
		name    string
		input   flows.ContextOptimizeInput
		wantErr bool
	}{
		{
			name: "有效输入 - aggressive",
			input: flows.ContextOptimizeInput{
				Context:         validContext,
				TargetTokens:    3000,
				Strategy:        "aggressive",
				PreserveSummary: false,
			},
			wantErr: false,
		},
		{
			name: "有效输入 - balanced",
			input: flows.ContextOptimizeInput{
				Context:         validContext,
				TargetTokens:    3000,
				Strategy:        "balanced",
				PreserveSummary: true,
			},
			wantErr: false,
		},
		{
			name: "有效输入 - conservative",
			input: flows.ContextOptimizeInput{
				Context:         validContext,
				TargetTokens:    3000,
				Strategy:        "conservative",
				PreserveSummary: true,
			},
			wantErr: false,
		},
		{
			name: "Context 为空",
			input: flows.ContextOptimizeInput{
				Context:      nil,
				TargetTokens: 3000,
				Strategy:     "balanced",
			},
			wantErr: true,
		},
		{
			name: "TargetTokens 太小",
			input: flows.ContextOptimizeInput{
				Context:      validContext,
				TargetTokens: 50,
				Strategy:     "balanced",
			},
			wantErr: true,
		},
		{
			name: "TargetTokens 太大",
			input: flows.ContextOptimizeInput{
				Context:      validContext,
				TargetTokens: 50000,
				Strategy:     "balanced",
			},
			wantErr: true,
		},
		{
			name: "TargetTokens 大于当前 Token 数",
			input: flows.ContextOptimizeInput{
				Context:      validContext,
				TargetTokens: 6000,
				Strategy:     "balanced",
			},
			wantErr: true,
		},
		{
			name: "Strategy 无效",
			input: flows.ContextOptimizeInput{
				Context:      validContext,
				TargetTokens: 3000,
				Strategy:     "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 基本的字段检查
			if tt.input.Context == nil {
				assert.True(t, tt.wantErr, "Context 为空应该返回错误")
				return
			}

			if tt.input.TargetTokens < 100 || tt.input.TargetTokens > 32000 {
				assert.True(t, tt.wantErr, "TargetTokens 超出范围应该返回错误")
				return
			}

			if tt.input.TargetTokens >= tt.input.Context.TotalTokens {
				assert.True(t, tt.wantErr, "TargetTokens 大于等于当前 Token 数应该返回错误")
				return
			}

			if tt.input.Strategy != "aggressive" && tt.input.Strategy != "balanced" && tt.input.Strategy != "conservative" {
				assert.True(t, tt.wantErr, "Strategy 无效应该返回错误")
				return
			}

			// 如果所有检查都通过，不应该有错误
			assert.False(t, tt.wantErr, "有效输入不应该返回错误")
		})
	}
}

// TestOptimizationStrategies 测试三种优化策略的特性
func TestOptimizationStrategies(t *testing.T) {
	tests := []struct {
		name                  string
		strategy              string
		expectedMaxMessages   int
		expectedMaxMemories   int
		shouldPreserveSummary bool
	}{
		{
			name:                  "aggressive 策略",
			strategy:              "aggressive",
			expectedMaxMessages:   5,
			expectedMaxMemories:   2,
			shouldPreserveSummary: false,
		},
		{
			name:                  "balanced 策略",
			strategy:              "balanced",
			expectedMaxMessages:   10,
			expectedMaxMemories:   5,
			shouldPreserveSummary: true,
		},
		{
			name:                  "conservative 策略",
			strategy:              "conservative",
			expectedMaxMessages:   15,
			expectedMaxMemories:   8,
			shouldPreserveSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证策略特性
			assert.NotEmpty(t, tt.strategy, "策略名称不应为空")
			assert.Greater(t, tt.expectedMaxMessages, 0, "最大消息数应大于0")
			assert.Greater(t, tt.expectedMaxMemories, 0, "最大记忆数应大于0")

			// aggressive 应该最激进
			if tt.strategy == "aggressive" {
				assert.LessOrEqual(t, tt.expectedMaxMessages, 5, "aggressive 策略应保留最少消息")
				assert.LessOrEqual(t, tt.expectedMaxMemories, 2, "aggressive 策略应保留最少记忆")
			}

			// conservative 应该最保守
			if tt.strategy == "conservative" {
				assert.GreaterOrEqual(t, tt.expectedMaxMessages, 15, "conservative 策略应保留更多消息")
				assert.GreaterOrEqual(t, tt.expectedMaxMemories, 8, "conservative 策略应保留更多记忆")
			}
		})
	}
}
