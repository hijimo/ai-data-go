// Package flows 实现查询分类相关的 Genkit Flow
package flows

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/firebase/genkit/go/genkit"
)

// RegisterQueryFlows 注册查询相关的 Flow
// 注意：此 Flow 使用基于规则的分类方法，不依赖 AI 服务以避免循环依赖
func RegisterQueryFlows(g *genkit.Genkit) {
	genkit.DefineFlow(
		g,
		"queryClassifyFlow",
		func(ctx context.Context, input QueryClassifyInput) (QueryClassifyOutput, error) {
			// 1. 参数验证
			if err := validateQueryInput(input); err != nil {
				return QueryClassifyOutput{}, fmt.Errorf("参数验证失败: %w", err)
			}

			// 2. 提取查询特征
			features := extractQueryFeatures(input.Query)

			// 3. 使用规则进行分类
			// 注意：为避免循环依赖，这里使用基于规则的分类
			// 如果需要 AI 分类，可以在调用此 Flow 后再调用 AI 服务进行增强
			classification := classifyQueryWithRules(input.Query, features)

			// 4. 推荐策略
			strategy := recommendStrategy(classification, features)

			// 5. 构建输出
			output := QueryClassifyOutput{
				QueryType:           classification.QueryType,
				NeedsHistory:        classification.NeedsHistory,
				KeyEntities:         features.Entities,
				RecommendedStrategy: strategy,
				Confidence:          classification.Confidence,
				Reasoning:           classification.Reasoning,
			}

			return output, nil
		},
	)
}

// validateQueryInput 验证输入参数
func validateQueryInput(input QueryClassifyInput) error {
	if err := validate.Struct(input); err != nil {
		return err
	}

	if len(input.Query) == 0 {
		return fmt.Errorf("查询内容不能为空")
	}

	if len(input.Query) > 2000 {
		return fmt.Errorf("查询内容不能超过 2000 个字符")
	}

	return nil
}

// QueryFeatures 查询特征
type QueryFeatures struct {
	HasPronouns      bool     // 是否包含指代词
	HasTimeReference bool     // 是否包含时间引用
	HasQuestionWords bool     // 是否包含疑问词
	Entities         []string // 关键实体
	Length           int      // 查询长度
	Complexity       string   // 复杂度：simple, medium, complex
}

// extractQueryFeatures 提取查询特征
func extractQueryFeatures(query string) QueryFeatures {
	features := QueryFeatures{
		Entities: make([]string, 0),
		Length:   len([]rune(query)),
	}

	// 检查指代词
	pronouns := []string{"它", "他", "她", "这个", "那个", "这些", "那些", "上面", "前面", "之前"}
	for _, pronoun := range pronouns {
		if strings.Contains(query, pronoun) {
			features.HasPronouns = true
			break
		}
	}

	// 检查时间引用
	timeWords := []string{"昨天", "今天", "明天", "上次", "之前", "刚才", "最近", "以前", "后来"}
	for _, timeWord := range timeWords {
		if strings.Contains(query, timeWord) {
			features.HasTimeReference = true
			break
		}
	}

	// 检查疑问词
	questionWords := []string{"什么", "为什么", "怎么", "如何", "哪", "谁", "吗", "呢", "？"}
	for _, qWord := range questionWords {
		if strings.Contains(query, qWord) {
			features.HasQuestionWords = true
			break
		}
	}

	// 提取关键实体（简单实现：提取名词性短语）
	features.Entities = extractEntities(query)

	// 判断复杂度
	if features.Length < 20 {
		features.Complexity = "simple"
	} else if features.Length < 50 {
		features.Complexity = "medium"
	} else {
		features.Complexity = "complex"
	}

	return features
}

// extractEntities 提取关键实体（简化版本）
func extractEntities(query string) []string {
	entities := make([]string, 0)

	// 提取引号中的内容
	quotedPattern := regexp.MustCompile(`["'](.*?)["']`)
	matches := quotedPattern.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 && len(match[1]) > 0 {
			entities = append(entities, match[1])
		}
	}

	// 提取专有名词（简单规则：连续的大写字母或数字）
	// 这里只是示例，实际应该使用 NER 模型
	wordPattern := regexp.MustCompile(`[A-Z][a-z]+|[0-9]+`)
	words := wordPattern.FindAllString(query, -1)
	entities = append(entities, words...)

	return entities
}

// QueryClassification 查询分类结果
type QueryClassification struct {
	QueryType    string  // 查询类型
	NeedsHistory bool    // 是否需要历史上下文
	Confidence   float64 // 置信度
	Reasoning    string  // 推理过程
}

// classifyQueryWithRules 使用规则进行查询分类
func classifyQueryWithRules(query string, features QueryFeatures) QueryClassification {
	classification := QueryClassification{
		QueryType:    "simple_question",
		NeedsHistory: false,
		Confidence:   0.6,
		Reasoning:    "基于规则的分类",
	}

	// 规则 1：包含"总结"、"概括"等关键词 -> 摘要请求（优先级最高）
	summarizationKeywords := []string{"总结", "概括", "归纳", "汇总", "梳理"}
	for _, keyword := range summarizationKeywords {
		if strings.Contains(query, keyword) {
			classification.QueryType = "summarization"
			classification.NeedsHistory = true
			classification.Reasoning = "查询要求总结或概括"
			return classification
		}
	}

	// 规则 2：包含"刚才"、"上面"等明确引用 -> 引用查询
	referenceKeywords := []string{"刚才", "上面", "前面", "之前说的", "你说的"}
	for _, keyword := range referenceKeywords {
		if strings.Contains(query, keyword) {
			classification.QueryType = "reference_query"
			classification.NeedsHistory = true
			classification.Reasoning = "查询明确引用之前的内容"
			return classification
		}
	}

	// 规则 3：包含"是吗"、"对吗"等确认词 -> 澄清问题
	clarificationKeywords := []string{"是吗", "对吗", "确认", "是不是", "对不对"}
	for _, keyword := range clarificationKeywords {
		if strings.Contains(query, keyword) {
			classification.QueryType = "clarification"
			classification.NeedsHistory = true
			classification.Reasoning = "查询需要确认或澄清"
			return classification
		}
	}

	// 规则 4：包含指代词或时间引用 -> 需要历史上下文
	if features.HasPronouns || features.HasTimeReference {
		classification.NeedsHistory = true
		classification.QueryType = "followup_question"
		classification.Reasoning = "查询包含指代词或时间引用，需要历史上下文"
		return classification
	}

	// 规则 5：复杂查询 -> 需要更多上下文
	if features.Complexity == "complex" {
		classification.QueryType = "complex_query"
		classification.NeedsHistory = true
		classification.Reasoning = "查询较为复杂，可能需要历史上下文"
		return classification
	}

	// 默认：简单问题
	classification.Reasoning = "简单问题，不需要历史上下文"
	return classification
}

// recommendStrategy 推荐上下文策略
func recommendStrategy(classification QueryClassification, features QueryFeatures) string {
	// 如果不需要历史，使用 short 策略
	if !classification.NeedsHistory {
		return "short"
	}

	// 根据查询类型推荐策略
	switch classification.QueryType {
	case "simple_question":
		return "short"

	case "followup_question":
		return "auto"

	case "complex_query":
		return "full"

	case "reference_query":
		return "full"

	case "summarization":
		return "full"

	case "clarification":
		return "auto"

	default:
		return "auto"
	}
}
