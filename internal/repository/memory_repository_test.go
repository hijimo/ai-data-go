package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"genkit-ai-service/internal/model"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用测试数据库连接字符串
	// 注意：需要在测试环境中配置 TEST_DATABASE_URL 环境变量
	// 或者使用默认的测试数据库配置
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=genkit_test sslmode=disable"
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "连接测试数据库失败")

	// 自动迁移表结构
	err = db.AutoMigrate(
		&model.ConversationMemory{},
		&model.ConversationContext{},
		&model.ConversationSummary{},
	)
	require.NoError(t, err, "迁移数据库表结构失败")

	return db
}

// cleanupTestDB 清理测试数据库
func cleanupTestDB(t *testing.T, db *gorm.DB) {
	// 清理测试数据
	db.Exec("TRUNCATE TABLE conversation_memories CASCADE")
	db.Exec("TRUNCATE TABLE conversation_contexts CASCADE")
	db.Exec("TRUNCATE TABLE conversation_summaries CASCADE")
}

// TestMemoryRepository_Create 测试创建记忆
func TestMemoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "这是一条测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err, "创建记忆失败")
	assert.NotEqual(t, uuid.Nil, memory.ID, "记忆ID应该被自动生成")
}

// TestMemoryRepository_GetByID 测试根据ID获取记忆
func TestMemoryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建测试记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试内容",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 测试获取记忆
	retrieved, err := repo.GetByID(ctx, tenantID, memory.ID)
	require.NoError(t, err, "获取记忆失败")
	assert.Equal(t, memory.ID, retrieved.ID)
	assert.Equal(t, memory.Content, retrieved.Content)
	assert.Equal(t, memory.TenantID, retrieved.TenantID)
}

// TestMemoryRepository_GetByID_TenantIsolation 测试租户隔离
func TestMemoryRepository_GetByID_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的记忆
	memory := &model.ConversationMemory{
		TenantID:   tenant1ID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "租户1的记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 尝试用租户2的ID获取租户1的记忆（应该失败）
	_, err = repo.GetByID(ctx, tenant2ID, memory.ID)
	assert.Error(t, err, "应该无法获取其他租户的记忆")
	assert.Equal(t, ErrNotFound, err, "应该返回 ErrNotFound")
}

// TestMemoryRepository_SearchByVector 测试向量检索（会话内）
func TestMemoryRepository_SearchByVector(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多条记忆
	memories := []*model.ConversationMemory{
		{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "记忆1",
			TokenCount: 10,
			Importance: 0.9,
		},
		{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "记忆2",
			TokenCount: 10,
			Importance: 0.7,
		},
		{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "记忆3",
			TokenCount: 10,
			Importance: 0.5,
		},
	}

	for _, mem := range memories {
		err := repo.Create(ctx, mem)
		require.NoError(t, err)
	}

	// 提取记忆ID
	memoryIDs := []uuid.UUID{memories[0].ID, memories[1].ID, memories[2].ID}

	// 测试向量检索
	results, err := repo.SearchByVector(ctx, tenantID, sessionID, memoryIDs)
	require.NoError(t, err, "向量检索失败")
	assert.Len(t, results, 3, "应该返回3条记忆")

	// 验证按重要性降序排列
	assert.Equal(t, memories[0].ID, results[0].ID, "第一条应该是重要性最高的")
	assert.Equal(t, memories[1].ID, results[1].ID, "第二条应该是重要性第二的")
	assert.Equal(t, memories[2].ID, results[2].ID, "第三条应该是重要性最低的")
}

// TestMemoryRepository_SearchByVectorCrossSessions 测试跨会话向量检索
func TestMemoryRepository_SearchByVectorCrossSessions(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	session1ID := uuid.New()
	session2ID := uuid.New()

	// 创建不同会话的记忆
	memory1 := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  session1ID,
		MemoryType: "long_term",
		Content:    "会话1的记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	memory2 := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  session2ID,
		MemoryType: "long_term",
		Content:    "会话2的记忆",
		TokenCount: 10,
		Importance: 0.9,
	}

	err := repo.Create(ctx, memory1)
	require.NoError(t, err)
	err = repo.Create(ctx, memory2)
	require.NoError(t, err)

	// 测试跨会话检索
	memoryIDs := []uuid.UUID{memory1.ID, memory2.ID}
	results, err := repo.SearchByVectorCrossSessions(ctx, tenantID, memoryIDs)
	require.NoError(t, err, "跨会话检索失败")
	assert.Len(t, results, 2, "应该返回2条记忆")
}

// TestMemoryRepository_UpdateAccessStats 测试更新访问统计
func TestMemoryRepository_UpdateAccessStats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 更新访问统计
	err = repo.UpdateAccessStats(ctx, tenantID, memory.ID)
	require.NoError(t, err, "更新访问统计失败")

	// 验证访问统计已更新
	retrieved, err := repo.GetByID(ctx, tenantID, memory.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.AccessCount, "访问次数应该增加1")
	assert.NotNil(t, retrieved.LastAccessAt, "最后访问时间应该被设置")
}

// TestMemoryRepository_DeleteByStrategy_Expired 测试删除过期记忆
func TestMemoryRepository_DeleteByStrategy_Expired(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建已过期的记忆
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "已过期的记忆",
		TokenCount: 10,
		Importance: 0.8,
		ExpiresAt:  &expiredTime,
	}
	err := repo.Create(ctx, expiredMemory)
	require.NoError(t, err)

	// 创建未过期的记忆
	futureTime := time.Now().Add(24 * time.Hour)
	validMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "未过期的记忆",
		TokenCount: 10,
		Importance: 0.8,
		ExpiresAt:  &futureTime,
	}
	err = repo.Create(ctx, validMemory)
	require.NoError(t, err)

	// 软删除过期记忆
	count, err := repo.DeleteByStrategy(ctx, tenantID, DeleteStrategyExpired, DeleteModeSoft)
	require.NoError(t, err, "删除过期记忆失败")
	assert.Equal(t, int64(1), count, "应该删除1条过期记忆")

	// 验证过期记忆已被软删除
	_, err = repo.GetByID(ctx, tenantID, expiredMemory.ID)
	assert.Error(t, err, "已删除的记忆不应该被查询到")

	// 验证未过期记忆仍然存在
	_, err = repo.GetByID(ctx, tenantID, validMemory.ID)
	assert.NoError(t, err, "未过期的记忆应该仍然存在")
}

// TestMemoryRepository_DeleteByStrategy_LowQuality 测试删除低质量记忆
func TestMemoryRepository_DeleteByStrategy_LowQuality(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建低质量记忆（重要性低于0.3且访问次数少于2）
	lowQualityMemory := &model.ConversationMemory{
		TenantID:    tenantID,
		SessionID:   sessionID,
		MemoryType:  "long_term",
		Content:     "低质量记忆",
		TokenCount:  10,
		Importance:  0.2,
		AccessCount: 1,
	}
	err := repo.Create(ctx, lowQualityMemory)
	require.NoError(t, err)

	// 创建高质量记忆
	highQualityMemory := &model.ConversationMemory{
		TenantID:    tenantID,
		SessionID:   sessionID,
		MemoryType:  "long_term",
		Content:     "高质量记忆",
		TokenCount:  10,
		Importance:  0.8,
		AccessCount: 5,
	}
	err = repo.Create(ctx, highQualityMemory)
	require.NoError(t, err)

	// 删除低质量记忆
	count, err := repo.DeleteByStrategy(ctx, tenantID, DeleteStrategyLowQuality, DeleteModeSoft)
	require.NoError(t, err, "删除低质量记忆失败")
	assert.Equal(t, int64(1), count, "应该删除1条低质量记忆")

	// 验证低质量记忆已被删除
	_, err = repo.GetByID(ctx, tenantID, lowQualityMemory.ID)
	assert.Error(t, err, "低质量记忆应该被删除")

	// 验证高质量记忆仍然存在
	_, err = repo.GetByID(ctx, tenantID, highQualityMemory.ID)
	assert.NoError(t, err, "高质量记忆应该仍然存在")
}

// TestMemoryRepository_GetExpiredMemories 测试获取过期记忆
func TestMemoryRepository_GetExpiredMemories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多条过期记忆
	for i := 0; i < 5; i++ {
		expiredTime := time.Now().Add(-time.Duration(i+1) * time.Hour)
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "过期记忆",
			TokenCount: 10,
			Importance: 0.8,
			ExpiresAt:  &expiredTime,
		}
		err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	// 获取过期记忆（限制3条）
	expired, err := repo.GetExpiredMemories(ctx, tenantID, 3)
	require.NoError(t, err, "获取过期记忆失败")
	assert.Len(t, expired, 3, "应该返回3条过期记忆")
}

// TestMemoryRepository_BatchCreate 测试批量创建记忆
func TestMemoryRepository_BatchCreate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建批量记忆
	memories := make([]*model.ConversationMemory, 10)
	for i := 0; i < 10; i++ {
		memories[i] = &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "批量记忆",
			TokenCount: 10,
			Importance: 0.8,
		}
	}

	// 批量创建
	err := repo.BatchCreate(ctx, memories)
	require.NoError(t, err, "批量创建记忆失败")

	// 验证所有记忆都已创建
	for _, mem := range memories {
		assert.NotEqual(t, uuid.Nil, mem.ID, "记忆ID应该被自动生成")
	}
}

// TestMemoryRepository_GetBySessionID 测试获取会话的所有记忆
func TestMemoryRepository_GetBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建不同类型的记忆
	shortTermMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "short_term",
		Content:    "短期记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	longTermMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "长期记忆",
		TokenCount: 10,
		Importance: 0.8,
	}

	err := repo.Create(ctx, shortTermMemory)
	require.NoError(t, err)
	err = repo.Create(ctx, longTermMemory)
	require.NoError(t, err)

	// 获取所有记忆
	allMemories, err := repo.GetBySessionID(ctx, tenantID, sessionID, "", 0)
	require.NoError(t, err, "获取会话记忆失败")
	assert.Len(t, allMemories, 2, "应该返回2条记忆")

	// 获取指定类型的记忆
	shortTermMemories, err := repo.GetBySessionID(ctx, tenantID, sessionID, "short_term", 0)
	require.NoError(t, err, "获取短期记忆失败")
	assert.Len(t, shortTermMemories, 1, "应该返回1条短期记忆")
	assert.Equal(t, "short_term", shortTermMemories[0].MemoryType)
}

// TestMemoryRepository_SoftDelete 测试软删除
func TestMemoryRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 软删除
	err = repo.SoftDelete(ctx, tenantID, memory.ID)
	require.NoError(t, err, "软删除失败")

	// 验证记忆已被软删除（无法通过GetByID获取）
	_, err = repo.GetByID(ctx, tenantID, memory.ID)
	assert.Error(t, err, "软删除的记忆不应该被查询到")
}

// TestMemoryRepository_HardDelete 测试硬删除
func TestMemoryRepository_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试记忆",
		TokenCount: 10,
		Importance: 0.8,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 硬删除
	err = repo.HardDelete(ctx, tenantID, memory.ID)
	require.NoError(t, err, "硬删除失败")

	// 验证记忆已被物理删除
	var count int64
	db.Model(&model.ConversationMemory{}).Where("id = ?", memory.ID).Count(&count)
	assert.Equal(t, int64(0), count, "记忆应该被物理删除")
}
