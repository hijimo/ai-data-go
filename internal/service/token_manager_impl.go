package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/model"
)

// tokenManagerImpl TokenManager接口的实现
type tokenManagerImpl struct {
	logger logger.Logger
}

// NewTokenManager 创建TokenManager实例
// 参数：
//   - log: 日志记录器
// 返回：
//   - TokenManager: TokenManager接口实例
func NewTokenManager(log logger.Logger) TokenManager {
	return &tokenManagerImpl{
		logger: log,
	}
}

// CalculateTokens 计算文本的Token数量
// 使用基于字符和单词的启发式方法进行计算
// 对于英文文本：约4个字符 = 1个token
// 对于中文文本：约1.5个字符 = 1个token
// 对于混合文本：根据字符类型动态计算
func (tm *tokenManagerImpl) CalculateTokens(ctx context.Context, text string, modelName string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// 使用改进的启发式算法
	tokens := tm.calculateTokensHeuristic(text)

	tm.logger.DebugContext(ctx, "计算Token数量", logger.Fields{
		"text_length": len(text),
		"tokens":      tokens,
		"model":       modelName,
	})

	return tokens, nil
}

// CalculateContextTokens 计算上下文的总Token数量
func (tm *tokenManagerImpl) CalculateContextTokens(
	ctx context.Context,
	messages []*model.ChatMessage,
	memories []*model.ConversationMemory,
	summary *model.ConversationSummary,
) (int, error) {
	totalTokens := 0

	// 计算消息的Token数量
	for _, msg := range messages {
		if msg.Tokens > 0 {
			// 如果消息已经有Token计数，直接使用
			totalTokens += msg.Tokens
		} else {
			// 否则计算Token数量
			tokens, err := tm.CalculateTokens(ctx, msg.Content, "")
			if err != nil {
				tm.logger.WarnContext(ctx, "计算消息Token失败", logger.Fields{
					"message_id": msg.ID.String(),
					"error":      err.Error(),
				})
				// 使用估算值
				tokens = tm.EstimateTokens(msg.Content)
			}
			totalTokens += tokens
			// 添加角色标记的开销（约4个token）
			totalTokens += 4
		}
	}

	// 计算记忆的Token数量
	for _, memory := range memories {
		if memory.TokenCount > 0 {
			totalTokens += memory.TokenCount
		} else {
			tokens, err := tm.CalculateTokens(ctx, memory.Content, "")
			if err != nil {
				tm.logger.WarnContext(ctx, "计算记忆Token失败", logger.Fields{
					"memory_id": memory.ID.String(),
					"error":     err.Error(),
				})
				tokens = tm.EstimateTokens(memory.Content)
			}
			totalTokens += tokens
		}
	}

	// 计算摘要的Token数量
	if summary != nil {
		if summary.TokenCount > 0 {
			totalTokens += summary.TokenCount
		} else {
			tokens, err := tm.CalculateTokens(ctx, summary.Content, "")
			if err != nil {
				tm.logger.WarnContext(ctx, "计算摘要Token失败", logger.Fields{
					"summary_id": summary.ID.String(),
					"error":      err.Error(),
				})
				tokens = tm.EstimateTokens(summary.Content)
			}
			totalTokens += tokens
		}
	}

	tm.logger.DebugContext(ctx, "计算上下文总Token数量", logger.Fields{
		"messages_count": len(messages),
		"memories_count": len(memories),
		"has_summary":    summary != nil,
		"total_tokens":   totalTokens,
	})

	return totalTokens, nil
}

// EstimateTokens 快速估算文本的Token数量
// 使用简单的启发式方法，速度快但精度较低
func (tm *tokenManagerImpl) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// 使用字符数除以平均字符/token比率
	// 对于混合文本，使用中间值：约2.5个字符 = 1个token
	charCount := utf8.RuneCountInString(text)
	estimatedTokens := charCount / 3

	// 至少返回1个token
	if estimatedTokens == 0 && charCount > 0 {
		estimatedTokens = 1
	}

	return estimatedTokens
}

// CalculateMessagesTokens 计算消息列表的Token数量
func (tm *tokenManagerImpl) CalculateMessagesTokens(
	ctx context.Context,
	messages []*model.ChatMessage,
	modelName string,
) (int, error) {
	totalTokens := 0

	for _, msg := range messages {
		if msg.Tokens > 0 {
			// 如果消息已经有Token计数，直接使用
			totalTokens += msg.Tokens
		} else {
			// 计算消息内容的Token数量
			tokens, err := tm.CalculateTokens(ctx, msg.Content, modelName)
			if err != nil {
				return 0, fmt.Errorf("计算消息Token失败: %w", err)
			}
			totalTokens += tokens
			// 添加角色标记的开销（约4个token）
			totalTokens += 4
		}
	}

	return totalTokens, nil
}

// calculateTokensHeuristic 使用启发式方法计算Token数量
// 这是一个改进的算法，考虑了中英文混合文本的特点
func (tm *tokenManagerImpl) calculateTokensHeuristic(text string) int {
	if text == "" {
		return 0
	}

	// 统计不同类型的字符
	var (
		chineseChars = 0
		englishChars = 0
		otherChars   = 0
		words        = 0
	)

	// 分析文本
	inWord := false
	for _, r := range text {
		if isChinese(r) {
			chineseChars++
			inWord = false
		} else if isEnglishLetter(r) {
			englishChars++
			if !inWord {
				words++
				inWord = true
			}
		} else if isWhitespace(r) {
			inWord = false
		} else {
			otherChars++
			inWord = false
		}
	}

	// 计算Token数量
	// 中文：约1.5个字符 = 1个token
	chineseTokens := (chineseChars * 2) / 3
	if chineseChars%3 != 0 {
		chineseTokens++
	}

	// 英文：约4个字符 = 1个token，或者按单词数计算
	// 使用两种方法的平均值
	englishTokensByChars := englishChars / 4
	englishTokensByWords := words
	englishTokens := (englishTokensByChars + englishTokensByWords) / 2
	if englishTokens == 0 && englishChars > 0 {
		englishTokens = 1
	}

	// 其他字符：约3个字符 = 1个token
	otherTokens := otherChars / 3

	totalTokens := chineseTokens + englishTokens + otherTokens

	// 至少返回1个token
	if totalTokens == 0 && len(text) > 0 {
		totalTokens = 1
	}

	return totalTokens
}

// isChinese 判断字符是否为中文字符
func isChinese(r rune) bool {
	// 中文字符的Unicode范围
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK统一表意文字
		(r >= 0x3400 && r <= 0x4DBF) || // CJK扩展A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK扩展B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK扩展C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK扩展D
		(r >= 0x2B820 && r <= 0x2CEAF) || // CJK扩展E
		(r >= 0xF900 && r <= 0xFAFF) || // CJK兼容表意文字
		(r >= 0x2F800 && r <= 0x2FA1F) // CJK兼容表意文字补充
}

// isEnglishLetter 判断字符是否为英文字母
func isEnglishLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isWhitespace 判断字符是否为空白字符
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// CalculateTokensForModel 根据模型名称计算Token数量
// 这是一个辅助方法，可以根据不同的模型使用不同的计算策略
func (tm *tokenManagerImpl) CalculateTokensForModel(text string, modelName string) int {
	// 标准化模型名称
	modelLower := strings.ToLower(modelName)

	// 根据模型类型选择计算策略
	switch {
	case strings.Contains(modelLower, "gpt-4"):
		// GPT-4系列模型
		return tm.calculateTokensHeuristic(text)
	case strings.Contains(modelLower, "gpt-3.5"):
		// GPT-3.5系列模型
		return tm.calculateTokensHeuristic(text)
	case strings.Contains(modelLower, "gemini"):
		// Gemini系列模型
		return tm.calculateTokensHeuristic(text)
	case strings.Contains(modelLower, "claude"):
		// Claude系列模型
		return tm.calculateTokensHeuristic(text)
	default:
		// 默认使用通用算法
		return tm.calculateTokensHeuristic(text)
	}
}
