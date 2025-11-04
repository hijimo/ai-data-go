package model_test

import (
	"fmt"
	"time"

	"genkit-ai-service/internal/model"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ExampleConversationMemory 演示如何创建和使用 ConversationMemory
func ExampleConversationMemory() {
	// 创建一个新的对话记忆
	memory := &model.ConversationMemory{
		TenantID:   uuid.New(),
		SessionID:  uuid.New(),
		MemoryType: model.MemoryTypeLongTerm,
		Content:    "用户询问了关于 Go 语言的问题",
		TokenCount: 15,
		Importance: 0.8,
		Metadata: map[string]interface{}{
			"topic":    "programming",
			"language": "go",
		},
	}

	// 设置向量嵌入（示例：1536维向量）
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // 实际应用中应该使用嵌入模型生成
	}
	memory.Embedding = pgvector.NewVector(embedding)

	fmt.Printf("记忆类型: %s\n", memory.MemoryType)
	fmt.Printf("内容: %s\n", memory.Content)
	fmt.Printf("重要性: %.2f\n", memory.Importance)
	// Output:
	// 记忆类型: long_term
	// 内容: 用户询问了关于 Go 语言的问题
	// 重要性: 0.80
}

// ExampleConversationContext 演示如何创建和使用 ConversationContext
func ExampleConversationContext() {
	// 创建一个新的对话上下文配置
	context := &model.ConversationContext{
		TenantID:        uuid.New(),
		SessionID:       uuid.New(),
		MaxTokens:       4000,
		Strategy:        model.ContextStrategyAuto,
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		TotalMessages:   25,
		TotalTokensUsed: 15000,
	}

	fmt.Printf("策略: %s\n", context.Strategy)
	fmt.Printf("最大Token数: %d\n", context.MaxTokens)
	fmt.Printf("短期记忆窗口: %d\n", context.ShortTermWindow)
	// Output:
	// 策略: auto
	// 最大Token数: 4000
	// 短期记忆窗口: 10
}

// ExampleConversationSummary 演示如何创建和使用 ConversationSummary
func ExampleConversationSummary() {
	// 创建一个新的对话摘要
	qualityScore := 0.85
	compressionRate := 0.65

	summary := &model.ConversationSummary{
		TenantID:        uuid.New(),
		SessionID:       uuid.New(),
		SummaryType:     model.SummaryTypeIncremental,
		Content:         "用户询问了多个关于 Go 语言的问题，包括并发、性能优化等主题。",
		TokenCount:      50,
		MessageCount:    20,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"Go语言", "并发", "性能优化"},
	}

	fmt.Printf("摘要类型: %s\n", summary.SummaryType)
	fmt.Printf("消息数量: %d\n", summary.MessageCount)
	fmt.Printf("质量评分: %.2f\n", *summary.QualityScore)
	fmt.Printf("关键主题: %v\n", summary.KeyTopics)
	// Output:
	// 摘要类型: incremental
	// 消息数量: 20
	// 质量评分: 0.85
	// 关键主题: [Go语言 并发 性能优化]
}

// ExampleMemoryType 演示记忆类型常量的使用
func ExampleMemoryType() {
	// 使用预定义的记忆类型常量
	memoryTypes := []string{
		model.MemoryTypeShortTerm,
		model.MemoryTypeLongTerm,
		model.MemoryTypeSummary,
	}

	for _, mt := range memoryTypes {
		fmt.Println(mt)
	}
	// Output:
	// short_term
	// long_term
	// summary
}

// ExampleContextStrategy 演示上下文策略常量的使用
func ExampleContextStrategy() {
	// 使用预定义的上下文策略常量
	strategies := []string{
		model.ContextStrategyAuto,
		model.ContextStrategyShort,
		model.ContextStrategyFull,
	}

	for _, strategy := range strategies {
		fmt.Println(strategy)
	}
	// Output:
	// auto
	// short
	// full
}

// ExampleSummaryType 演示摘要类型常量的使用
func ExampleSummaryType() {
	// 使用预定义的摘要类型常量
	summaryTypes := []string{
		model.SummaryTypeIncremental,
		model.SummaryTypeFull,
	}

	for _, st := range summaryTypes {
		fmt.Println(st)
	}
	// Output:
	// incremental
	// full
}

// ExampleConversationMemory_withExpiration 演示如何创建带过期时间的记忆
func ExampleConversationMemory_withExpiration() {
	// 创建一个7天后过期的记忆
	expiresAt := time.Now().AddDate(0, 0, 7)

	memory := &model.ConversationMemory{
		TenantID:   uuid.New(),
		SessionID:  uuid.New(),
		MemoryType: model.MemoryTypeShortTerm,
		Content:    "临时信息，7天后过期",
		TokenCount: 10,
		Importance: 0.5,
		ExpiresAt:  &expiresAt,
	}

	fmt.Printf("记忆类型: %s\n", memory.MemoryType)
	fmt.Printf("是否设置过期时间: %v\n", memory.ExpiresAt != nil)
	// Output:
	// 记忆类型: short_term
	// 是否设置过期时间: true
}

// ExampleConversationContext_withSummary 演示如何创建关联摘要的上下文
func ExampleConversationContext_withSummary() {
	summaryID := uuid.New()
	summaryTime := time.Now()

	context := &model.ConversationContext{
		TenantID:        uuid.New(),
		SessionID:       uuid.New(),
		MaxTokens:       4000,
		Strategy:        model.ContextStrategyAuto,
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		LastSummaryID:   &summaryID,
		LastSummaryAt:   &summaryTime,
	}

	fmt.Printf("是否包含摘要: %v\n", context.IncludeSummary)
	fmt.Printf("是否有最后摘要ID: %v\n", context.LastSummaryID != nil)
	// Output:
	// 是否包含摘要: true
	// 是否有最后摘要ID: true
}
