// internal/service/token_manager_test.go
package service

import (
	"testing"

	"genkit-ai-service/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTextTokens(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "空文本",
			text:     "",
			expected: 0,
		},
		{
			name:     "纯英文",
			text:     "Hello World",
			expected: 3, // 11个字符 / 4 ≈ 3
		},
		{
			name:     "纯中文",
			text:     "你好世界",
			expected: 3, // 4个字符 / 1.5 ≈ 3
		},
		{
			name:     "中英混合",
			text:     "Hello 世界",
			expected: 3, // 5个英文字符/4 + 2个中文字符/1.5 ≈ 3
		},
		{
			name:     "长文本",
			text:     "这是一个测试文本，用于验证Token计算功能。This is a test text for token calculation.",
			expected: 24, // 约24个tokens（实际计算结果）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.CalculateTextTokens(tt.text)
			// 允许一定的误差范围
			assert.InDelta(t, tt.expected, result, 2.0, "Token计算结果应该接近预期值")
		})
	}
}

func TestCalculateContextTokens(t *testing.T) {
	tm := &tokenManagerImpl{}

	// 准备测试数据
	messages := []*model.ChatMessage{
		{
			ID:      uuid.New(),
			Content: "这是第一条消息",
			Tokens:  10,
		},
		{
			ID:      uuid.New(),
			Content: "这是第二条消息",
			Tokens:  12,
		},
	}

	memories := []*model.ConversationMemory{
		{
			ID:         uuid.New(),
			Content:    "这是一条记忆",
			TokenCount: 8,
		},
	}

	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		Content:    "这是摘要内容",
		TokenCount: 15,
	}

	// 测试完整上下文
	t.Run("完整上下文", func(t *testing.T) {
		total := tm.CalculateContextTokens(messages, memories, summary)
		expected := 10 + 12 + 8 + 15 // 45
		assert.Equal(t, expected, total, "应该正确计算所有组件的Token总和")
	})

	// 测试无摘要
	t.Run("无摘要", func(t *testing.T) {
		total := tm.CalculateContextTokens(messages, memories, nil)
		expected := 10 + 12 + 8 // 30
		assert.Equal(t, expected, total, "应该正确计算无摘要时的Token总和")
	})

	// 测试空上下文
	t.Run("空上下文", func(t *testing.T) {
		total := tm.CalculateContextTokens(nil, nil, nil)
		assert.Equal(t, 0, total, "空上下文应该返回0")
	})

	// 测试未预计算Token的消息
	t.Run("未预计算Token", func(t *testing.T) {
		messagesNoToken := []*model.ChatMessage{
			{
				ID:      uuid.New(),
				Content: "测试消息",
				Tokens:  0, // 未预计算
			},
		}
		total := tm.CalculateContextTokens(messagesNoToken, nil, nil)
		assert.Greater(t, total, 0, "应该自动计算未预计算的Token")
	})
}

func TestCompressContent(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name     string
		content  string
		target   int
		checkOps bool
	}{
		{
			name:     "移除多余空白",
			content:  "这是   一个    测试。",
			target:   100,
			checkOps: true,
		},
		{
			name:     "移除重复句子",
			content:  "这是测试。这是测试。这是另一个测试。",
			target:   100,
			checkOps: true,
		},
		{
			name:     "简化表达",
			content:  "这非常好。那特别棒。",
			target:   100,
			checkOps: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ops := tm.compressContent(tt.content, tt.target)
			assert.NotEmpty(t, result, "压缩后的内容不应为空")
			if tt.checkOps {
				assert.NotEmpty(t, ops, "应该记录压缩操作")
			}
			// 压缩后的内容应该不长于原内容
			assert.LessOrEqual(t, len(result), len(tt.content), "压缩后的内容不应长于原内容")
		})
	}
}

func TestTruncateContent(t *testing.T) {
	tm := &tokenManagerImpl{}

	content := "这是一个很长的测试文本。它包含多个句子。我们需要截断它。保留前面的部分。"
	targetTokens := 10

	result, ops := tm.truncateContent(content, targetTokens)

	assert.NotEmpty(t, result, "截断后的内容不应为空")
	assert.Less(t, len(result), len(content), "截断后的内容应该短于原内容")
	assert.NotEmpty(t, ops, "应该记录截断操作")
}

func TestGenerateBudgetSuggestions(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name       string
		usageRate  float64
		budgetType string
		minSuggestions int
	}{
		{
			name:       "配额已用尽",
			usageRate:  1.0,
			budgetType: "daily",
			minSuggestions: 2,
		},
		{
			name:       "使用率90%",
			usageRate:  0.9,
			budgetType: "daily",
			minSuggestions: 2,
		},
		{
			name:       "使用率80%",
			usageRate:  0.8,
			budgetType: "daily",
			minSuggestions: 2,
		},
		{
			name:       "使用率70%",
			usageRate:  0.7,
			budgetType: "daily",
			minSuggestions: 1,
		},
		{
			name:       "使用率正常",
			usageRate:  0.5,
			budgetType: "daily",
			minSuggestions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := tm.generateBudgetSuggestions(tt.usageRate, tt.budgetType)
			assert.GreaterOrEqual(t, len(suggestions), tt.minSuggestions, "应该生成足够的建议")
		})
	}
}

func TestAnalyzeTrend(t *testing.T) {
	// 这个测试需要数据库连接，这里只做基本的逻辑测试
	// 实际的集成测试应该在integration_test中进行
	t.Skip("需要数据库连接，跳过单元测试")
}

func TestCalculateEfficiencyScore(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name           string
		totalUsage     int
		avgDailyUsage  int
		peakUsage      int
		expectedRange  [2]float64 // [min, max]
	}{
		{
			name:          "理想比率",
			totalUsage:    10000,
			avgDailyUsage: 1000,
			peakUsage:     1500, // 比率1.5
			expectedRange: [2]float64{0.85, 0.95},
		},
		{
			name:          "峰值过高",
			totalUsage:    10000,
			avgDailyUsage: 1000,
			peakUsage:     3000, // 比率3.0
			expectedRange: [2]float64{0.3, 0.8},
		},
		{
			name:          "使用稳定",
			totalUsage:    10000,
			avgDailyUsage: 1000,
			peakUsage:     1100, // 比率1.1
			expectedRange: [2]float64{0.7, 0.9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tm.calculateEfficiencyScore(tt.totalUsage, tt.avgDailyUsage, tt.peakUsage)
			assert.GreaterOrEqual(t, score, tt.expectedRange[0], "效率评分应该在预期范围内")
			assert.LessOrEqual(t, score, tt.expectedRange[1], "效率评分应该在预期范围内")
		})
	}
}

func TestPredictFutureUsage(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name          string
		avgDailyUsage int
		trend         string
		checkIncrease bool
		checkDecrease bool
	}{
		{
			name:          "增长趋势",
			avgDailyUsage: 1000,
			trend:         "increasing",
			checkIncrease: true,
		},
		{
			name:          "下降趋势",
			avgDailyUsage: 1000,
			trend:         "decreasing",
			checkDecrease: true,
		},
		{
			name:          "稳定趋势",
			avgDailyUsage: 1000,
			trend:         "stable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predictions := tm.predictFutureUsage(tt.avgDailyUsage, tt.trend)

			assert.Greater(t, predictions.NextDay, 0, "次日预测应该大于0")
			assert.Greater(t, predictions.NextWeek, 0, "次周预测应该大于0")
			assert.Greater(t, predictions.NextMonth, 0, "次月预测应该大于0")

			if tt.checkIncrease {
				assert.Greater(t, predictions.NextDay, tt.avgDailyUsage, "增长趋势下次日预测应该更高")
			}

			if tt.checkDecrease {
				assert.Less(t, predictions.NextDay, tt.avgDailyUsage, "下降趋势下次日预测应该更低")
			}
		})
	}
}

func TestGenerateOptimizationSuggestions(t *testing.T) {
	tm := &tokenManagerImpl{}

	tests := []struct {
		name            string
		totalUsage      int
		avgDailyUsage   int
		trend           string
		efficiencyScore float64
		minSuggestions  int
	}{
		{
			name:            "增长趋势+低效率",
			totalUsage:      200000,
			avgDailyUsage:   10000,
			trend:           "increasing",
			efficiencyScore: 0.5,
			minSuggestions:  2,
		},
		{
			name:            "稳定趋势+高效率",
			totalUsage:      50000,
			avgDailyUsage:   2500,
			trend:           "stable",
			efficiencyScore: 0.9,
			minSuggestions:  1,
		},
		{
			name:            "高使用量",
			totalUsage:      500000,
			avgDailyUsage:   25000,
			trend:           "stable",
			efficiencyScore: 0.8,
			minSuggestions:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := tm.generateOptimizationSuggestions(
				tt.totalUsage,
				tt.avgDailyUsage,
				tt.trend,
				tt.efficiencyScore,
			)

			assert.GreaterOrEqual(t, len(suggestions), tt.minSuggestions, "应该生成足够的优化建议")

			// 验证建议结构
			for _, s := range suggestions {
				assert.NotEmpty(t, s.Priority, "建议应该有优先级")
				assert.NotEmpty(t, s.Suggestion, "建议应该有内容")
				assert.GreaterOrEqual(t, s.EstimatedSaving, 0, "预计节省应该非负")
			}
		})
	}
}
