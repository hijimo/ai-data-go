package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/mock"

	"genkit-ai-service/internal/model"
)

// BenchmarkContextService_BuildContext 基准测试：构建上下文
func BenchmarkContextService_BuildContext(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟10条消息
	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息内容 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(1000)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_BuildContext_WithLongTerm 基准测试：构建包含长期记忆的上下文
func BenchmarkContextService_BuildContext_WithLongTerm(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 模拟消息
	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	// 模拟记忆
	memories := make([]*model.ConversationMemory, 5)
	for i := 0; i < 5; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆 %d", i),
			Importance: 0.8,
		}
	}

	embedding := pgvector.Vector{1.0, 2.0, 3.0}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 10).Return(messages, nil)
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)
	mockMemoryRepo.On("SearchByVector", mock.Anything, sessionID, embedding, 5, float32(0.7)).Return(memories, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(1500)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_BuildContext_WithSummary 基准测试：构建包含摘要的上下文
func BenchmarkContextService_BuildContext_WithSummary(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	summary := &model.ConversationSummary{
		ID:         uuid.New(),
		Content:    "这是对话摘要",
		TokenCount: 100,
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 10).Return(messages, nil)
	mockContextRepo.On("GetLatestSummary", mock.Anything, sessionID).Return(summary, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, summary).Return(1200)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		ShortTermWindow: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_OptimizeContext 基准测试：优化上下文
func BenchmarkContextService_OptimizeContext(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()

	// 创建大量消息和记忆
	messages := make([]*model.ChatMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Content: fmt.Sprintf("消息内容 %d", i),
		}
	}

	memories := make([]*model.ConversationMemory, 20)
	for i := 0; i < 20; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆内容 %d", i),
			Importance: float32(0.5 + float64(i)*0.02),
		}
	}

	originalContext := &ContextResult{
		SessionID:         uuid.New().String(),
		ShortTermMessages: messages,
		LongTermMemories:  memories,
		TotalTokens:       8000,
	}

	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(3000)

	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    3000,
		Strategy:        "balanced",
		PreserveSummary: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.OptimizeContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}


// BenchmarkMemoryService_StoreMemory 基准测试：存储记忆
func BenchmarkMemoryService_StoreMemory(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.ConversationMemory")).Return(nil)

	req := StoreMemoryRequest{
		TenantID:   tenantID,
		SessionID:  sessionID,
		Content:    "这是一条测试记忆内容",
		MemoryType: "long_term",
		Importance: 0.8,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.StoreMemory(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryService_SearchMemories 基准测试：搜索记忆
func BenchmarkMemoryService_SearchMemories(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)

	// 模拟搜索结果
	memories := make([]*model.ConversationMemory, 5)
	for i := 0; i < 5; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("相关记忆 %d", i),
			Importance: 0.8,
		}
	}

	mockRepo.On("SearchByVector", mock.Anything, sessionID, embedding, 5, float32(0.7)).Return(memories, nil)
	mockRepo.On("UpdateAccessStats", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	req := SearchMemoriesRequest{
		TenantID:      tenantID,
		SessionID:     sessionID,
		Query:         "搜索查询",
		TopK:          5,
		MinSimilarity: 0.7,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.SearchMemories(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryService_SearchMemories_CrossSessions 基准测试：跨会话搜索
func BenchmarkMemoryService_SearchMemories_CrossSessions(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)

	// 模拟跨会话搜索结果
	memories := make([]*model.ConversationMemory, 10)
	for i := 0; i < 10; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			SessionID:  uuid.New(),
			Content:    fmt.Sprintf("跨会话记忆 %d", i),
			Importance: 0.8,
		}
	}

	mockRepo.On("SearchByVectorCrossSessions", mock.Anything, tenantID, embedding, 10, float32(0.7)).Return(memories, nil)
	mockRepo.On("UpdateAccessStats", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	req := SearchMemoriesRequest{
		TenantID:             tenantID,
		Query:                "跨会话查询",
		TopK:                 10,
		MinSimilarity:        0.7,
		IncludeCrossSessions: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.SearchMemories(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryService_CleanupMemories 基准测试：清理记忆
func BenchmarkMemoryService_CleanupMemories(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	mockRepo.On("DeleteByStrategy", mock.Anything, tenantID, "expired", "soft", 100).Return(10, nil)

	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "expired",
		Mode:      "soft",
		BatchSize: 100,
		Execute:   true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.CleanupMemories(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVectorService_GenerateEmbedding 基准测试：生成向量
func BenchmarkVectorService_GenerateEmbedding(b *testing.B) {
	mockVectorSvc := new(MockVectorService)

	ctx := context.Background()
	text := "这是一段需要生成向量的文本内容"

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, text).Return(embedding, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mockVectorSvc.GenerateEmbedding(ctx, text)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVectorService_GenerateEmbeddings_Batch 基准测试：批量生成向量
func BenchmarkVectorService_GenerateEmbeddings_Batch(b *testing.B) {
	mockVectorSvc := new(MockVectorService)

	ctx := context.Background()
	
	// 准备10条文本
	texts := make([]string, 10)
	for i := 0; i < 10; i++ {
		texts[i] = fmt.Sprintf("文本内容 %d", i)
	}

	// 准备10个向量
	embeddings := make([]pgvector.Vector, 10)
	for i := 0; i < 10; i++ {
		embeddings[i] = pgvector.Vector{float32(i), float32(i + 1), float32(i + 2)}
	}

	mockVectorSvc.On("GenerateEmbeddings", mock.Anything, texts).Return(embeddings, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mockVectorSvc.GenerateEmbeddings(ctx, texts)
		if err != nil {
			b.Fatal(err)
		}
	}
}


// BenchmarkContextService_BuildContext_Parallel 并发基准测试：构建上下文
func BenchmarkContextService_BuildContext_Parallel(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, mock.Anything, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(1000)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		sessionID := uuid.New().String()

		req := BuildContextRequest{
			SessionID:       sessionID,
			UserQuery:       "并发测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		for pb.Next() {
			_, err := service.BuildContext(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryService_StoreMemory_Parallel 并发基准测试：存储记忆
func BenchmarkMemoryService_StoreMemory_Parallel(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.ConversationMemory")).Return(nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		tenantID := uuid.New().String()
		sessionID := uuid.New().String()

		req := StoreMemoryRequest{
			TenantID:   tenantID,
			SessionID:  sessionID,
			Content:    "并发测试记忆",
			MemoryType: "long_term",
			Importance: 0.8,
		}

		for pb.Next() {
			_, err := service.StoreMemory(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryService_SearchMemories_Parallel 并发基准测试：搜索记忆
func BenchmarkMemoryService_SearchMemories_Parallel(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)

	memories := make([]*model.ConversationMemory, 5)
	for i := 0; i < 5; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆 %d", i),
			Importance: 0.8,
		}
	}

	mockRepo.On("SearchByVector", mock.Anything, mock.Anything, embedding, 5, float32(0.7)).Return(memories, nil)
	mockRepo.On("UpdateAccessStats", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		tenantID := uuid.New().String()
		sessionID := uuid.New().String()

		req := SearchMemoriesRequest{
			TenantID:      tenantID,
			SessionID:     sessionID,
			Query:         "并发搜索查询",
			TopK:          5,
			MinSimilarity: 0.7,
		}

		for pb.Next() {
			_, err := service.SearchMemories(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTokenManager_CalculateTokens 基准测试：Token计算
func BenchmarkTokenManager_CalculateTokens(b *testing.B) {
	mockTokenMgr := new(MockTokenManager)

	text := "这是一段需要计算Token数量的文本内容，包含中文和English混合内容。"
	mockTokenMgr.On("CalculateTokens", text).Return(50)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mockTokenMgr.CalculateTokens(text)
	}
}

// BenchmarkTokenManager_CalculateContextTokens 基准测试：上下文Token计算
func BenchmarkTokenManager_CalculateContextTokens(b *testing.B) {
	mockTokenMgr := new(MockTokenManager)

	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Content: fmt.Sprintf("消息内容 %d", i),
		}
	}

	memories := make([]*model.ConversationMemory, 5)
	for i := 0; i < 5; i++ {
		memories[i] = &model.ConversationMemory{
			ID:      uuid.New(),
			Content: fmt.Sprintf("记忆内容 %d", i),
		}
	}

	summary := &model.ConversationSummary{
		ID:      uuid.New(),
		Content: "对话摘要内容",
	}

	mockTokenMgr.On("CalculateContextTokens", messages, memories, summary).Return(1500)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mockTokenMgr.CalculateContextTokens(messages, memories, summary)
	}
}


// BenchmarkCacheService_Set 基准测试：缓存写入
func BenchmarkCacheService_Set(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &ContextResult{
		SessionID:   uuid.New().String(),
		TotalTokens: 1000,
		Strategy:    "auto",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("context:%d", i)
		_ = cache.Set(ctx, key, data, 5*time.Minute)
	}
}

// BenchmarkCacheService_Get 基准测试：缓存读取
func BenchmarkCacheService_Get(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	// 预先设置缓存
	data := &ContextResult{
		SessionID:   uuid.New().String(),
		TotalTokens: 1000,
		Strategy:    "auto",
	}
	key := "context:test"
	_ = cache.Set(ctx, key, data, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result ContextResult
		_ = cache.Get(ctx, key, &result)
	}
}

// BenchmarkCacheService_SetGet 基准测试：缓存读写混合
func BenchmarkCacheService_SetGet(b *testing.B) {
	cache := NewCacheService(nil, "bench")
	ctx := context.Background()

	data := &ContextResult{
		SessionID:   uuid.New().String(),
		TotalTokens: 1000,
		Strategy:    "auto",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("context:%d", i%100) // 使用100个不同的key循环

		if i%2 == 0 {
			// 写入
			_ = cache.Set(ctx, key, data, 5*time.Minute)
		} else {
			// 读取
			var result ContextResult
			_ = cache.Get(ctx, key, &result)
		}
	}
}

// BenchmarkCacheService_Parallel 并发基准测试：缓存操作
func BenchmarkCacheService_Parallel(b *testing.B) {
	cache := NewCacheService(nil, "bench")

	data := &ContextResult{
		SessionID:   uuid.New().String(),
		TotalTokens: 1000,
		Strategy:    "auto",
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		i := 0

		for pb.Next() {
			key := fmt.Sprintf("context:%d", i%100)

			if i%2 == 0 {
				_ = cache.Set(ctx, key, data, 5*time.Minute)
			} else {
				var result ContextResult
				_ = cache.Get(ctx, key, &result)
			}

			i++
		}
	})
}

// BenchmarkContextService_WithCache 基准测试：带缓存的上下文构建
func BenchmarkContextService_WithCache(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(1000)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "缓存测试查询",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_CacheHitRate 基准测试：缓存命中率
func BenchmarkContextService_CacheHitRate(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()

	messages := make([]*model.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, mock.Anything, 10).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(1000)

	// 使用10个不同的会话ID，模拟缓存命中
	sessionIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		sessionIDs[i] = uuid.New().String()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sessionID := sessionIDs[i%10] // 循环使用10个会话ID

		req := BuildContextRequest{
			SessionID:       sessionID,
			UserQuery:       "缓存命中测试",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}


// BenchmarkContextService_LargeContext 基准测试：大规模上下文
func BenchmarkContextService_LargeContext(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	// 创建100条消息
	messages := make([]*model.ChatMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("这是一条很长的消息内容，包含大量文本数据用于测试性能 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 100).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(5000)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "大规模测试查询",
		MaxTokens:       8000,
		Strategy:        "auto",
		ShortTermWindow: 100,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryService_LargeVectorSearch 基准测试：大规模向量检索
func BenchmarkMemoryService_LargeVectorSearch(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()
	sessionID := uuid.New().String()

	embedding := pgvector.Vector{1.0, 2.0, 3.0}
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)

	// 模拟50条搜索结果
	memories := make([]*model.ConversationMemory, 50)
	for i := 0; i < 50; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("大规模记忆内容 %d", i),
			Importance: 0.8,
		}
	}

	mockRepo.On("SearchByVector", mock.Anything, sessionID, embedding, 50, float32(0.7)).Return(memories, nil)
	mockRepo.On("UpdateAccessStats", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	req := SearchMemoriesRequest{
		TenantID:      tenantID,
		SessionID:     sessionID,
		Query:         "大规模搜索查询",
		TopK:          50,
		MinSimilarity: 0.7,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.SearchMemories(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_TokenOptimization 基准测试：Token优化性能
func BenchmarkContextService_TokenOptimization(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()

	// 创建大量消息和记忆，模拟需要优化的场景
	messages := make([]*model.ChatMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Content: fmt.Sprintf("消息内容 %d，包含大量文本数据", i),
		}
	}

	memories := make([]*model.ConversationMemory, 50)
	for i := 0; i < 50; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆内容 %d，包含详细信息", i),
			Importance: float32(0.5 + float64(i)*0.01),
		}
	}

	summary := &model.ConversationSummary{
		ID:      uuid.New(),
		Content: "这是一个详细的对话摘要，包含关键信息",
	}

	originalContext := &ContextResult{
		SessionID:         uuid.New().String(),
		ShortTermMessages: messages,
		LongTermMemories:  memories,
		Summary:           summary,
		TotalTokens:       15000,
	}

	// 第一次计算返回超限
	mockTokenMgr.On("CalculateContextTokens", messages, memories, summary).Return(15000).Once()
	// 优化后的计算
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(4000)

	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    4000,
		Strategy:        "aggressive",
		PreserveSummary: false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.OptimizeContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryService_BatchCleanup 基准测试：批量清理性能
func BenchmarkMemoryService_BatchCleanup(b *testing.B) {
	mockRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockCache := NewCacheService(nil, "bench")

	service := NewMemoryService(mockRepo, mockVectorSvc, mockCache)
	ctx := context.Background()

	tenantID := uuid.New().String()

	// 模拟每次清理100条记录
	mockRepo.On("DeleteByStrategy", mock.Anything, tenantID, "expired", "soft", 1000).Return(100, nil)

	req := CleanupMemoriesRequest{
		TenantID:  tenantID,
		Strategy:  "expired",
		Mode:      "soft",
		BatchSize: 1000,
		Execute:   true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.CleanupMemories(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_QualityScoreCalculation 基准测试：质量评分计算
func BenchmarkContextService_QualityScoreCalculation(b *testing.B) {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	service := NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)

	ctx := context.Background()
	sessionID := uuid.New().String()

	messages := make([]*model.ChatMessage, 20)
	for i := 0; i < 20; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, sessionID, 20).Return(messages, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(2000)

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "质量评分测试",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 20,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		// 验证质量评分被计算
		if result.QualityScore <= 0 {
			b.Fatal("质量评分未计算")
		}
	}
}


// BenchmarkContextService_Strategy_Auto 基准测试：自动策略
func BenchmarkContextService_Strategy_Auto(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()
	sessionID := uuid.New().String()

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "自动策略测试",
		MaxTokens:       4000,
		Strategy:        "auto",
		ShortTermWindow: 10,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_Strategy_Full 基准测试：完整策略
func BenchmarkContextService_Strategy_Full(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()
	sessionID := uuid.New().String()

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "完整策略测试",
		MaxTokens:       8000,
		Strategy:        "full",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 50,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextService_Strategy_Minimal 基准测试：最小策略
func BenchmarkContextService_Strategy_Minimal(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()
	sessionID := uuid.New().String()

	req := BuildContextRequest{
		SessionID:       sessionID,
		UserQuery:       "最小策略测试",
		MaxTokens:       1000,
		Strategy:        "minimal",
		IncludeSummary:  false,
		IncludeLongTerm: false,
		ShortTermWindow: 3,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.BuildContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptimizeContext_Strategy_Balanced 基准测试：平衡优化策略
func BenchmarkOptimizeContext_Strategy_Balanced(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()

	originalContext := createLargeContext()

	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    3000,
		Strategy:        "balanced",
		PreserveSummary: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.OptimizeContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptimizeContext_Strategy_Aggressive 基准测试：激进优化策略
func BenchmarkOptimizeContext_Strategy_Aggressive(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()

	originalContext := createLargeContext()

	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    2000,
		Strategy:        "aggressive",
		PreserveSummary: false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.OptimizeContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptimizeContext_Strategy_Conservative 基准测试：保守优化策略
func BenchmarkOptimizeContext_Strategy_Conservative(b *testing.B) {
	service := setupContextServiceForBenchmark()
	ctx := context.Background()

	originalContext := createLargeContext()

	req := OptimizeContextRequest{
		Context:         originalContext,
		TargetTokens:    4000,
		Strategy:        "conservative",
		PreserveSummary: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := service.OptimizeContext(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 辅助函数：设置用于基准测试的上下文服务
func setupContextServiceForBenchmark() ContextService {
	mockContextRepo := new(MockContextRepository)
	mockMessageRepo := new(MockMessageRepository)
	mockMemoryRepo := new(MockMemoryRepository)
	mockVectorSvc := new(MockVectorService)
	mockTokenMgr := new(MockTokenManager)
	mockCache := NewCacheService(nil, "bench")

	// 设置通用的mock行为
	messages := make([]*model.ChatMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Role:    "user",
			Content: fmt.Sprintf("消息 %d", i),
		}
	}

	memories := make([]*model.ConversationMemory, 10)
	for i := 0; i < 10; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆 %d", i),
			Importance: 0.8,
		}
	}

	summary := &model.ConversationSummary{
		ID:      uuid.New(),
		Content: "对话摘要",
	}

	embedding := pgvector.Vector{1.0, 2.0, 3.0}

	mockMessageRepo.On("GetRecentMessages", mock.Anything, mock.Anything, mock.Anything).Return(messages, nil)
	mockMemoryRepo.On("SearchByVector", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(memories, nil)
	mockContextRepo.On("GetLatestSummary", mock.Anything, mock.Anything).Return(summary, nil)
	mockVectorSvc.On("GenerateEmbedding", mock.Anything, mock.Anything).Return(embedding, nil)
	mockTokenMgr.On("CalculateContextTokens", mock.Anything, mock.Anything, mock.Anything).Return(2000)

	return NewContextService(
		mockContextRepo,
		mockMessageRepo,
		mockMemoryRepo,
		mockVectorSvc,
		mockTokenMgr,
		mockCache,
	)
}

// 辅助函数：创建大规模上下文用于优化测试
func createLargeContext() *ContextResult {
	messages := make([]*model.ChatMessage, 100)
	for i := 0; i < 100; i++ {
		messages[i] = &model.ChatMessage{
			ID:      uuid.New(),
			Content: fmt.Sprintf("消息内容 %d", i),
		}
	}

	memories := make([]*model.ConversationMemory, 50)
	for i := 0; i < 50; i++ {
		memories[i] = &model.ConversationMemory{
			ID:         uuid.New(),
			Content:    fmt.Sprintf("记忆内容 %d", i),
			Importance: float32(0.5 + float64(i)*0.01),
		}
	}

	summary := &model.ConversationSummary{
		ID:      uuid.New(),
		Content: "详细的对话摘要内容",
	}

	return &ContextResult{
		SessionID:         uuid.New().String(),
		ShortTermMessages: messages,
		LongTermMemories:  memories,
		Summary:           summary,
		TotalTokens:       10000,
	}
}
