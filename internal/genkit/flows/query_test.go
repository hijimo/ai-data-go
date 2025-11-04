// Package flows 测试查询分类相关的 Flow
package flows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractQueryFeatures 测试查询特征提取
func TestExtractQueryFeatures(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected QueryFeatures
	}{
		{
			name:  "简单问题",
			query: "什么是人工智能？",
			expected: QueryFeatures{
				HasPronouns:      false,
				HasTimeReference: false,
				HasQuestionWords: true,
				Complexity:       "simple",
			},
		},
		{
			name:  "包含指代词",
			query: "它是怎么工作的？",
			expected: QueryFeatures{
				HasPronouns:      true,
				HasTimeReference: false,
				HasQuestionWords: true,
				Complexity:       "simple",
			},
		},
		{
			name:  "包含时间引用",
			query: "昨天我们讨论的问题解决了吗？",
			expected: QueryFeatures{
				HasPronouns:      false,
				HasTimeReference: true,
				HasQuestionWords: true,
				Complexity:       "simple",
			},
		},
		{
			name:  "复杂查询",
			query: "请详细解释一下机器学习中的梯度下降算法的工作原理，包括它的数学推导过程和实际应用场景，以及与其他优化算法的比较。",
			expected: QueryFeatures{
				HasPronouns:      true, // "它的" 被检测为指代词
				HasTimeReference: false,
				HasQuestionWords: false,
				Complexity:       "complex",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := extractQueryFeatures(tt.query)

			assert.Equal(t, tt.expected.HasPronouns, features.HasPronouns, "HasPronouns 不匹配")
			assert.Equal(t, tt.expected.HasTimeReference, features.HasTimeReference, "HasTimeReference 不匹配")
			assert.Equal(t, tt.expected.HasQuestionWords, features.HasQuestionWords, "HasQuestionWords 不匹配")
			assert.Equal(t, tt.expected.Complexity, features.Complexity, "Complexity 不匹配")
		})
	}
}

// TestClassifyQueryWithRules 测试基于规则的查询分类
func TestClassifyQueryWithRules(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected QueryClassification
	}{
		{
			name:  "简单问题",
			query: "什么是人工智能？",
			expected: QueryClassification{
				QueryType:    "simple_question",
				NeedsHistory: false,
			},
		},
		{
			name:  "追问",
			query: "它是怎么工作的？",
			expected: QueryClassification{
				QueryType:    "followup_question",
				NeedsHistory: true,
			},
		},
		{
			name:  "总结请求",
			query: "请总结一下我们刚才讨论的内容",
			expected: QueryClassification{
				QueryType:    "summarization",
				NeedsHistory: true,
			},
		},
		{
			name:  "引用查询",
			query: "你刚才说的那个算法是什么？",
			expected: QueryClassification{
				QueryType:    "reference_query",
				NeedsHistory: true,
			},
		},
		{
			name:  "澄清问题",
			query: "这个是对的吗？",
			expected: QueryClassification{
				QueryType:    "followup_question", // "这个" 被检测为指代词，优先级高于"对吗"
				NeedsHistory: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := extractQueryFeatures(tt.query)
			classification := classifyQueryWithRules(tt.query, features)

			assert.Equal(t, tt.expected.QueryType, classification.QueryType, "QueryType 不匹配")
			assert.Equal(t, tt.expected.NeedsHistory, classification.NeedsHistory, "NeedsHistory 不匹配")
		})
	}
}

// TestRecommendStrategy 测试策略推荐
func TestRecommendStrategy(t *testing.T) {
	tests := []struct {
		name           string
		classification QueryClassification
		features       QueryFeatures
		expected       string
	}{
		{
			name: "简单问题推荐 short",
			classification: QueryClassification{
				QueryType:    "simple_question",
				NeedsHistory: false,
			},
			features: QueryFeatures{
				Complexity: "simple",
			},
			expected: "short",
		},
		{
			name: "追问推荐 auto",
			classification: QueryClassification{
				QueryType:    "followup_question",
				NeedsHistory: true,
			},
			features: QueryFeatures{
				Complexity: "medium",
			},
			expected: "auto",
		},
		{
			name: "复杂查询推荐 full",
			classification: QueryClassification{
				QueryType:    "complex_query",
				NeedsHistory: true,
			},
			features: QueryFeatures{
				Complexity: "complex",
			},
			expected: "full",
		},
		{
			name: "总结请求推荐 full",
			classification: QueryClassification{
				QueryType:    "summarization",
				NeedsHistory: true,
			},
			features: QueryFeatures{
				Complexity: "medium",
			},
			expected: "full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := recommendStrategy(tt.classification, tt.features)
			assert.Equal(t, tt.expected, strategy, "推荐策略不匹配")
		})
	}
}

// TestValidateQueryInput 测试输入验证
func TestValidateQueryInput(t *testing.T) {
	tests := []struct {
		name      string
		input     QueryClassifyInput
		expectErr bool
	}{
		{
			name: "有效输入",
			input: QueryClassifyInput{
				Query:     "什么是人工智能？",
				SessionID: "",
			},
			expectErr: false,
		},
		{
			name: "空查询",
			input: QueryClassifyInput{
				Query:     "",
				SessionID: "",
			},
			expectErr: true,
		},
		{
			name: "查询过长",
			input: QueryClassifyInput{
				Query:     string(make([]byte, 2001)),
				SessionID: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryInput(tt.input)

			if tt.expectErr {
				assert.Error(t, err, "应该返回错误")
			} else {
				assert.NoError(t, err, "不应该返回错误")
			}
		})
	}
}


