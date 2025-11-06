package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
)

func TestNewTokenManager(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)

	assert.NotNil(t, tm, "TokenManager不应为nil")
}

func TestCalculateTokens(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	ctx := context.Background()

	tests := []struct {
		name      string
		text      string
		modelName string
		wantMin   int
		wantMax   int
	}{
		{
			name:      "空文本",
			text:      "",
			modelName: "gpt-4",
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "纯英文短文本",
			text:      "Hello world",
			modelName: "gpt-4",
			wantMin:   2,
			wantMax:   5,
		},
		{
			name:      "纯中文短文本",
			text:      "你好世界",
			modelName: "gpt-4",
			wantMin:   2,
			wantMax:   6,
		},
		{
			name:      "中英文混合",
			text:      "Hello 世界",
			modelName: "gpt-4",
			wantMin:   2,
			wantMax:   6,
		},
		{
			name:      "长英文文本",
			text:      "This is a longer text that contains multiple words and should result in more tokens being calculated.",
			modelName: "gpt-4",
			wantMin:   15,
			wantMax:   30,
		},
		{
			name:      "长中文文本",
			text:      "这是一段较长的中文文本，包含多个词语，应该会计算出更多的Token数量。",
			modelName: "gpt-4",
			wantMin:   20,
			wantMax:   40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tm.CalculateTokens(ctx, tt.text, tt.modelName)
			assert.NoError(t, err, "计算Token不应返回错误")
			assert.GreaterOrEqual(t, tokens, tt.wantMin, "Token数量应大于等于最小值")
			assert.LessOrEqual(t, tokens, tt.wantMax, "Token数量应小于等于最大值")
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)

	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "空文本",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "短文本",
			text:    "Hello",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "中文文本",
			text:    "你好世界",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "长文本",
			text:    "This is a much longer text that should result in a higher token estimate.",
			wantMin: 15,
			wantMax: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tm.EstimateTokens(tt.text)
			assert.GreaterOrEqual(t, tokens, tt.wantMin, "估算Token数量应大于等于最小值")
			assert.LessOrEqual(t, tokens, tt.wantMax, "估算Token数量应小于等于最大值")
		})
	}
}

func TestCalculateContextTokens(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	ctx := context.Background()

	// 准备测试数据
	messages := []*model.ChatMessage{
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "user",
			Content:   "Hello, how are you?",
			Tokens:    5,
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "assistant",
			Content:   "I'm doing well, thank you!",
			Tokens:    7,
			CreatedAt: time.Now(),
		},
	}

	memories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			TenantID:   uuid.New(),
			SessionID:  uuid.New(),
			MemoryType: "long_term",
			Content:    "User prefers detailed explanations.",
			TokenCount: 6,
			CreatedAt:  time.Now(),
		},
	}

	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		SessionID:  uuid.New(),
		Content:    "Previous conversation about AI and technology.",
		TokenCount: 8,
		CreatedAt:  time.Now(),
	}

	tests := []struct {
		name     string
		messages []*model.ChatMessage
		memories []*model.ConversationMemory
		summary  *model.ConversationSummary
		wantMin  int
		wantMax  int
	}{
		{
			name:     "仅消息",
			messages: messages,
			memories: nil,
			summary:  nil,
			wantMin:  12, // 5 + 7
			wantMax:  20,
		},
		{
			name:     "消息和记忆",
			messages: messages,
			memories: memories,
			summary:  nil,
			wantMin:  18, // 5 + 7 + 6
			wantMax:  30,
		},
		{
			name:     "完整上下文",
			messages: messages,
			memories: memories,
			summary:  summary,
			wantMin:  26, // 5 + 7 + 6 + 8
			wantMax:  40,
		},
		{
			name:     "空上下文",
			messages: nil,
			memories: nil,
			summary:  nil,
			wantMin:  0,
			wantMax:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tm.CalculateContextTokens(ctx, tt.messages, tt.memories, tt.summary)
			assert.NoError(t, err, "计算上下文Token不应返回错误")
			assert.GreaterOrEqual(t, tokens, tt.wantMin, "Token数量应大于等于最小值")
			assert.LessOrEqual(t, tokens, tt.wantMax, "Token数量应小于等于最大值")
		})
	}
}

func TestCalculateMessagesTokens(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	ctx := context.Background()

	messages := []*model.ChatMessage{
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "user",
			Content:   "What is AI?",
			Tokens:    0, // 未计算
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "assistant",
			Content:   "AI stands for Artificial Intelligence.",
			Tokens:    8, // 已计算
			CreatedAt: time.Now(),
		},
	}

	tokens, err := tm.CalculateMessagesTokens(ctx, messages, "gpt-4")
	assert.NoError(t, err, "计算消息Token不应返回错误")
	assert.Greater(t, tokens, 0, "Token数量应大于0")
	// 第一条消息需要计算（约2-4个token + 4个角色开销）
	// 第二条消息已有token（8 + 4个角色开销）
	assert.GreaterOrEqual(t, tokens, 14, "Token数量应至少为14")
}

func TestCalculateTokensHeuristic(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log).(*tokenManagerImpl)

	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "纯英文",
			text:    "Hello world this is a test",
			wantMin: 4,
			wantMax: 10,
		},
		{
			name:    "纯中文",
			text:    "这是一个测试文本",
			wantMin: 4,
			wantMax: 10,
		},
		{
			name:    "中英文混合",
			text:    "Hello 世界 this is 测试",
			wantMin: 4,
			wantMax: 12,
		},
		{
			name:    "包含标点符号",
			text:    "Hello, world! How are you?",
			wantMin: 4,
			wantMax: 10,
		},
		{
			name:    "包含数字",
			text:    "The year is 2024 and the temperature is 25 degrees",
			wantMin: 8,
			wantMax: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tm.calculateTokensHeuristic(tt.text)
			assert.GreaterOrEqual(t, tokens, tt.wantMin, "Token数量应大于等于最小值")
			assert.LessOrEqual(t, tokens, tt.wantMax, "Token数量应小于等于最大值")
		})
	}
}

func TestIsChinese(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"中文字符", '中', true},
		{"英文字符", 'A', false},
		{"数字", '1', false},
		{"标点符号", '。', false},
		{"空格", ' ', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChinese(tt.r)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsEnglishLetter(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"大写字母", 'A', true},
		{"小写字母", 'a', true},
		{"数字", '1', false},
		{"中文", '中', false},
		{"标点符号", '.', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEnglishLetter(tt.r)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsWhitespace(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"空格", ' ', true},
		{"制表符", '\t', true},
		{"换行符", '\n', true},
		{"回车符", '\r', true},
		{"字母", 'A', false},
		{"数字", '1', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWhitespace(tt.r)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateTokensForModel(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log).(*tokenManagerImpl)

	text := "This is a test message for token calculation."

	models := []string{
		"gpt-4",
		"gpt-3.5-turbo",
		"gemini-pro",
		"claude-3",
		"unknown-model",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			tokens := tm.CalculateTokensForModel(text, model)
			assert.Greater(t, tokens, 0, "Token数量应大于0")
			assert.Less(t, tokens, 100, "Token数量应小于100")
		})
	}
}

func TestCalculateContextTokensWithoutPrecomputedTokens(t *testing.T) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	ctx := context.Background()

	// 准备没有预计算Token的测试数据
	messages := []*model.ChatMessage{
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "user",
			Content:   "Hello, how are you?",
			Tokens:    0, // 未预计算
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			Role:      "assistant",
			Content:   "I'm doing well, thank you!",
			Tokens:    0, // 未预计算
			CreatedAt: time.Now(),
		},
	}

	memories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			TenantID:   uuid.New(),
			SessionID:  uuid.New(),
			MemoryType: "long_term",
			Content:    "User prefers detailed explanations.",
			TokenCount: 0, // 未预计算
			CreatedAt:  time.Now(),
		},
	}

	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		SessionID:  uuid.New(),
		Content:    "Previous conversation about AI and technology.",
		TokenCount: 0, // 未预计算
		CreatedAt:  time.Now(),
	}

	tokens, err := tm.CalculateContextTokens(ctx, messages, memories, summary)
	assert.NoError(t, err, "计算上下文Token不应返回错误")
	assert.Greater(t, tokens, 0, "Token数量应大于0")
	// 应该动态计算所有内容的Token
	assert.GreaterOrEqual(t, tokens, 20, "Token数量应至少为20")
}

func BenchmarkCalculateTokens(b *testing.B) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	ctx := context.Background()
	text := "This is a benchmark test for token calculation performance. It contains multiple words and should be representative of typical usage."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tm.CalculateTokens(ctx, text, "gpt-4")
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	log := logger.New(logger.InfoLevel, logger.TextFormat, os.Stdout)
	tm := NewTokenManager(log)
	text := "This is a benchmark test for token estimation performance. It contains multiple words and should be representative of typical usage."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tm.EstimateTokens(text)
	}
}
