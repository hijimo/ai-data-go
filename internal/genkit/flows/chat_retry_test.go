package flows

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateChatRetryInput 测试重试输入验证
func TestValidateChatRetryInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ChatRetryInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效输入 - simple 策略",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: false,
		},
		{
			name: "有效输入 - exponential 策略",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "exponential",
				MaxRetries:    5,
			},
			wantErr: false,
		},
		{
			name: "有效输入 - adaptive 策略",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "adaptive",
				MaxRetries:    4,
			},
			wantErr: false,
		},
		{
			name: "无效输入 - 空 sessionId",
			input: ChatRetryInput{
				SessionID:     "",
				UserMessage:   "测试消息",
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "sessionId 不能为空",
		},
		{
			name: "无效输入 - 无效的 UUID",
			input: ChatRetryInput{
				SessionID:     "invalid-uuid",
				UserMessage:   "测试消息",
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "sessionId 格式无效",
		},
		{
			name: "无效输入 - 空消息",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "",
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "userMessage 不能为空",
		},
		{
			name: "无效输入 - 消息过长",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   string(make([]byte, 4001)),
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "userMessage 长度不能超过 4000 字符",
		},
		{
			name: "无效输入 - 空重试策略",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "retryStrategy 不能为空",
		},
		{
			name: "无效输入 - 不支持的重试策略",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "invalid",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "不支持的重试策略",
		},
		{
			name: "无效输入 - maxRetries 太小",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "simple",
				MaxRetries:    0,
			},
			wantErr: true,
			errMsg:  "maxRetries 必须在 1-10 之间",
		},
		{
			name: "无效输入 - maxRetries 太大",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: "simple",
				MaxRetries:    11,
			},
			wantErr: true,
			errMsg:  "maxRetries 必须在 1-10 之间",
		},
		{
			name: "无效输入 - systemPrompt 过长",
			input: ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				SystemPrompt:  string(make([]byte, 1001)),
				RetryStrategy: "simple",
				MaxRetries:    3,
			},
			wantErr: true,
			errMsg:  "systemPrompt 长度不能超过 1000 字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChatRetryInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAnalyzeError 测试错误分析
func TestAnalyzeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "速率限制错误",
			err:      errors.New("rate limit exceeded"),
			expected: "rate_limit",
		},
		{
			name:     "速率限制错误 - 另一种表述",
			err:      errors.New("too many requests"),
			expected: "rate_limit",
		},
		{
			name:     "超时错误",
			err:      errors.New("request timeout"),
			expected: "timeout",
		},
		{
			name:     "超时错误 - deadline",
			err:      errors.New("context deadline exceeded"),
			expected: "timeout",
		},
		{
			name:     "上下文长度错误",
			err:      errors.New("context length exceeded"),
			expected: "context_length",
		},
		{
			name:     "Token 限制错误",
			err:      errors.New("token limit exceeded"),
			expected: "context_length",
		},
		{
			name:     "服务器错误",
			err:      errors.New("internal server error"),
			expected: "server_error",
		},
		{
			name:     "无效请求",
			err:      errors.New("invalid request"),
			expected: "invalid_request",
		},
		{
			name:     "未知错误",
			err:      errors.New("some unknown error"),
			expected: "unknown",
		},
		{
			name:     "nil 错误",
			err:      nil,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestOptimizePromptForRetry 测试提示词优化
func TestOptimizePromptForRetry(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		targetTokens int
		expectTrunc  bool
	}{
		{
			name:         "短提示词不需要优化",
			prompt:       "这是一个短提示词",
			targetTokens: 100,
			expectTrunc:  false,
		},
		{
			name:         "长提示词需要优化",
			prompt:       string(make([]byte, 1000)),
			targetTokens: 50,
			expectTrunc:  true,
		},
		{
			name:         "刚好达到目标长度",
			prompt:       string(make([]byte, 400)),
			targetTokens: 100,
			expectTrunc:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizePromptForRetry(tt.prompt, tt.targetTokens)

			if tt.expectTrunc {
				// 优化后的提示词应该更短
				assert.Less(t, len(result), len(tt.prompt))
				// 应该包含省略标记
				assert.Contains(t, result, "[部分内容已省略]")
			} else {
				// 不需要优化时应该保持原样
				assert.Equal(t, tt.prompt, result)
			}
		})
	}
}

// TestRetryInfo 测试重试信息结构
func TestRetryInfo(t *testing.T) {
	retryInfo := RetryInfo{
		Strategy:       "simple",
		TotalAttempts:  3,
		SuccessAttempt: 2,
		FailedAttempts: []RetryAttempt{
			{
				AttemptNumber: 1,
				Error:         "timeout",
				WaitTime:      1000,
				Timestamp:     "2024-01-01T00:00:00Z",
			},
		},
		TotalRetryTime: 5000,
	}

	assert.Equal(t, "simple", retryInfo.Strategy)
	assert.Equal(t, 3, retryInfo.TotalAttempts)
	assert.Equal(t, 2, retryInfo.SuccessAttempt)
	assert.Len(t, retryInfo.FailedAttempts, 1)
	assert.Equal(t, int64(5000), retryInfo.TotalRetryTime)
}

// TestChatRetryInput 测试重试输入结构
func TestChatRetryInput(t *testing.T) {
	input := ChatRetryInput{
		SessionID:     "550e8400-e29b-41d4-a716-446655440000",
		UserMessage:   "测试消息",
		RetryStrategy: "exponential",
		MaxRetries:    5,
		SaveMessage:   true,
	}

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", input.SessionID)
	assert.Equal(t, "测试消息", input.UserMessage)
	assert.Equal(t, "exponential", input.RetryStrategy)
	assert.Equal(t, 5, input.MaxRetries)
	assert.True(t, input.SaveMessage)
}

// TestChatRetryOutput 测试重试输出结构
func TestChatRetryOutput(t *testing.T) {
	output := ChatRetryOutput{
		MessageID:    "msg-123",
		Response:     "测试响应",
		FallbackUsed: true,
		FallbackReason: "减少上下文",
		RetryInfo: RetryInfo{
			Strategy:       "adaptive",
			TotalAttempts:  4,
			SuccessAttempt: 0,
			TotalRetryTime: 10000,
		},
	}

	assert.Equal(t, "msg-123", output.MessageID)
	assert.Equal(t, "测试响应", output.Response)
	assert.True(t, output.FallbackUsed)
	assert.Equal(t, "减少上下文", output.FallbackReason)
	assert.Equal(t, "adaptive", output.RetryInfo.Strategy)
	assert.Equal(t, 4, output.RetryInfo.TotalAttempts)
}

// BenchmarkAnalyzeError 性能测试：错误分析
func BenchmarkAnalyzeError(b *testing.B) {
	err := errors.New("rate limit exceeded")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = analyzeError(err)
	}
}

// BenchmarkOptimizePromptForRetry 性能测试：提示词优化
func BenchmarkOptimizePromptForRetry(b *testing.B) {
	prompt := string(make([]byte, 1000))
	targetTokens := 50
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = optimizePromptForRetry(prompt, targetTokens)
	}
}

// TestRetryStrategies 测试不同重试策略的配置
func TestRetryStrategies(t *testing.T) {
	strategies := []struct {
		name       string
		strategy   string
		maxRetries int
	}{
		{
			name:       "简单重试策略",
			strategy:   "simple",
			maxRetries: 3,
		},
		{
			name:       "指数退避策略",
			strategy:   "exponential",
			maxRetries: 5,
		},
		{
			name:       "自适应策略",
			strategy:   "adaptive",
			maxRetries: 4,
		},
	}

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			input := ChatRetryInput{
				SessionID:     "550e8400-e29b-41d4-a716-446655440000",
				UserMessage:   "测试消息",
				RetryStrategy: tt.strategy,
				MaxRetries:    tt.maxRetries,
			}

			err := validateChatRetryInput(input)
			assert.NoError(t, err)
		})
	}
}

// TestFallbackOperations 测试回退操作类型
func TestFallbackOperations(t *testing.T) {
	operations := []FallbackOperation{
		{
			Type:        "reduce_context",
			Description: "减少上下文大小",
			Applied:     true,
		},
		{
			Type:        "use_fallback_model",
			Description: "使用备用模型",
			Applied:     false,
		},
		{
			Type:        "return_preset_response",
			Description: "返回预设响应",
			Applied:     true,
		},
	}

	assert.Len(t, operations, 3)
	assert.Equal(t, "reduce_context", operations[0].Type)
	assert.True(t, operations[0].Applied)
	assert.False(t, operations[1].Applied)
}

// TestRetryAttempt 测试重试尝试记录
func TestRetryAttempt(t *testing.T) {
	attempt := RetryAttempt{
		AttemptNumber: 1,
		Error:         "timeout error",
		WaitTime:      1000,
		Timestamp:     "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, 1, attempt.AttemptNumber)
	assert.Equal(t, "timeout error", attempt.Error)
	assert.Equal(t, int64(1000), attempt.WaitTime)
	assert.Equal(t, "2024-01-01T00:00:00Z", attempt.Timestamp)
}

// TestContextOptimization 测试上下文优化逻辑
func TestContextOptimization(t *testing.T) {
	// 测试不同大小的提示词优化
	testCases := []struct {
		name           string
		promptLength   int
		targetTokens   int
		shouldOptimize bool
	}{
		{
			name:           "小提示词",
			promptLength:   100,
			targetTokens:   100,
			shouldOptimize: false,
		},
		{
			name:           "中等提示词",
			promptLength:   500,
			targetTokens:   100,
			shouldOptimize: true,
		},
		{
			name:           "大提示词",
			promptLength:   2000,
			targetTokens:   100,
			shouldOptimize: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prompt := string(make([]byte, tc.promptLength))
			result := optimizePromptForRetry(prompt, tc.targetTokens)

			if tc.shouldOptimize {
				assert.Less(t, len(result), len(prompt))
			} else {
				assert.Equal(t, len(prompt), len(result))
			}
		})
	}
}
