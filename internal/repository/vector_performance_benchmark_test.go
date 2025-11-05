package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// BenchmarkMemoryRepository_SearchByVector 基准测试：向量检索
func BenchmarkMemoryRepository_SearchByVector(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建100条记忆
	for i := 0; i < 100; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    fmt.Sprintf("记忆内容 %d", i),
			TokenCount: 10,
			Importance: 0.8,
			Embedding:  pgvector.NewVector([]float32{float32(i), float32(i + 1), float32(i + 2)}),
		}
		err := repo.Create(ctx, memory)
		require.NoError(b, err)
	}

	queryEmbedding := pgvector.NewVector([]float32{50.0, 51.0, 52.0})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.SearchByVector(ctx, sessionID.String(), queryEmbedding, 5, 0.7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_SearchByVector_TopK10 基准测试：Top-10向量检索
func BenchmarkMemoryRepository_SearchByVector_TopK10(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建200条记忆
	for i := 0; i < 200; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    fmt.Sprintf("记忆内容 %d", i),
			TokenCount: 10,
			Importance: 0.8,
			Embedding:  pgvector.NewVector([]float32{float32(i), float32(i + 1), float32(i + 2)}),
		}
		err := repo.Create(ctx, memory)
		require.NoError(b, err)
	}

	queryEmbedding := pgvector.NewVector([]float32{100.0, 101.0, 102.0})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.SearchByVector(ctx, sessionID.String(), queryEmbedding, 10, 0.7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_SearchByVector_TopK50 基准测试：Top-50向量检索
func BenchmarkMemoryRepository_SearchByVector_TopK50(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建500条记忆
	for i := 0; i < 500; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    fmt.Sprintf("记忆内容 %d", i),
			TokenCount: 10,
			Importance: 0.8,
			Embedding:  pgvector.NewVector([]float32{float32(i), float32(i + 1), float32(i + 2)}),
		}
		err := repo.Create(ctx, memory)
		require.NoError(b, err)
	}

	queryEmbedding := pgvector.NewVector([]float32{250.0, 251.0, 252.0})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.SearchByVector(ctx, sessionID.String(), queryEmbedding, 50, 0.7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_SearchByVectorCrossSessions 基准测试：跨会话向量检索
func BenchmarkMemoryRepository_SearchByVectorCrossSessions(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	// 创建10个会话，每个会话20条记忆
	for s := 0; s < 10; s++ {
		sessionID := uuid.New()
		for i := 0; i < 20; i++ {
			memory := &model.ConversationMemory{
				TenantID:   tenantID,
				SessionID:  sessionID,
				MemoryType: "long_term",
				Content:    fmt.Sprintf("会话%d记忆%d", s, i),
				TokenCount: 10,
				Importance: 0.8,
				Embedding:  pgvector.NewVector([]float32{float32(s*20 + i), float32(s*20 + i + 1), float32(s*20 + i + 2)}),
			}
			err := repo.Create(ctx, memory)
			require.NoError(b, err)
		}
	}

	queryEmbedding := pgvector.NewVector([]float32{100.0, 101.0, 102.0})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.SearchByVectorCrossSessions(ctx, tenantID.String(), queryEmbedding, 10, 0.7)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_Create 基准测试：创建记忆
func BenchmarkMemoryRepository_Create(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    fmt.Sprintf("基准测试记忆 %d", i),
			TokenCount: 10,
			Importance: 0.8,
			Embedding:  pgvector.NewVector([]float32{float32(i), float32(i + 1), float32(i + 2)}),
		}
		err := repo.Create(ctx, memory)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_GetByID 基准测试：根据ID获取记忆
func BenchmarkMemoryRepository_GetByID(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(b, err)

	memoryID := memory.ID.String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.GetByID(ctx, memoryID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_UpdateAccessStats 基准测试：更新访问统计
func BenchmarkMemoryRepository_UpdateAccessStats(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(b, err)

	memoryID := memory.ID.String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := repo.UpdateAccessStats(ctx, memoryID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_DeleteByStrategy 基准测试：按策略删除
func BenchmarkMemoryRepository_DeleteByStrategy(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// 每次迭代前创建过期记忆
		expiredTime := time.Now().Add(-1 * time.Hour)
		for j := 0; j < 10; j++ {
			memory := &model.ConversationMemory{
				TenantID:   tenantID,
				SessionID:  sessionID,
				MemoryType: "long_term",
				Content:    fmt.Sprintf("过期记忆 %d", j),
				TokenCount: 10,
				Importance: 0.8,
				ExpiresAt:  &expiredTime,
			}
			_ = repo.Create(ctx, memory)
		}
		b.StartTimer()

		_, err := repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "soft", 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryRepository_CountBySession 基准测试：统计会话记忆数量
func BenchmarkMemoryRepository_CountBySession(b *testing.B) {
	db := setupBenchmarkDB(b)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 预先创建50条记忆
	for i := 0; i < 50; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    fmt.Sprintf("记忆 %d", i),
			TokenCount: 10,
			Importance: 0.8,
		}
		err := repo.Create(ctx, memory)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.CountBySession(ctx, sessionID.String())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 辅助函数：设置基准测试数据库
func setupBenchmarkDB(b *testing.B) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(b, err)

	// 自动迁移
	err = db.AutoMigrate(&model.ConversationMemory{})
	require.NoError(b, err)

	return db
}
