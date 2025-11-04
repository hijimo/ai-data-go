package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&model.ConversationMemory{})
	require.NoError(t, err)

	return db
}

// TestMemoryRepository_Create 测试创建记忆
func TestMemoryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
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
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, memory.ID)
}

// TestMemoryRepository_GetByID 测试根据ID获取记忆
func TestMemoryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建记忆
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "测试内容",
		TokenCount: 10,
		Importance: 0.7,
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 获取记忆
	retrieved, err := repo.GetByID(ctx, memory.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, memory.ID, retrieved.ID)
	assert.Equal(t, memory.Content, retrieved.Content)
}

// TestMemoryRepository_GetByID_NotFound 测试获取不存在的记忆
func TestMemoryRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	nonExistentID := uuid.New().String()
	memory, err := repo.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.Nil(t, memory)
}

// TestMemoryRepository_UpdateAccessStats 测试更新访问统计
func TestMemoryRepository_UpdateAccessStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建记忆
	memory := &model.ConversationMemory{
		TenantID:    tenantID,
		SessionID:   sessionID,
		MemoryType:  "long_term",
		Content:     "测试内容",
		TokenCount:  10,
		Importance:  0.7,
		AccessCount: 0,
	}

	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 更新访问统计
	err = repo.UpdateAccessStats(ctx, memory.ID.String())
	assert.NoError(t, err)

	// 验证访问次数增加
	updated, err := repo.GetByID(ctx, memory.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, 1, updated.AccessCount)
	assert.NotNil(t, updated.LastAccessAt)

	// 再次更新
	err = repo.UpdateAccessStats(ctx, memory.ID.String())
	assert.NoError(t, err)

	updated, err = repo.GetByID(ctx, memory.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, 2, updated.AccessCount)
}

// TestMemoryRepository_CountBySession 测试统计会话记忆数量
func TestMemoryRepository_CountBySession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多条记忆
	for i := 0; i < 5; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "测试内容",
			TokenCount: 10,
			Importance: 0.7,
		}
		err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	// 统计会话的记忆数量
	count, err := repo.CountBySession(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

// TestMemoryRepository_DeleteByStrategy_Expired 测试删除过期记忆
func TestMemoryRepository_DeleteByStrategy_Expired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建过期记忆
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "过期记忆",
		TokenCount: 10,
		Importance: 0.7,
		ExpiresAt:  &expiredTime,
	}
	err := repo.Create(ctx, expiredMemory)
	require.NoError(t, err)

	// 创建未过期记忆
	futureTime := time.Now().Add(1 * time.Hour)
	validMemory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "有效记忆",
		TokenCount: 10,
		Importance: 0.7,
		ExpiresAt:  &futureTime,
	}
	err = repo.Create(ctx, validMemory)
	require.NoError(t, err)

	// 删除过期记忆（软删除）
	count, err := repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "soft", 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证过期记忆被标记为删除
	deleted, err := repo.GetByID(ctx, expiredMemory.ID.String())
	assert.Error(t, err) // 软删除后应该查询不到
	assert.Nil(t, deleted)

	// 验证有效记忆仍然存在
	valid, err := repo.GetByID(ctx, validMemory.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, valid)
}

// TestMemoryRepository_DeleteByStrategy_LowQuality 测试删除低质量记忆
func TestMemoryRepository_DeleteByStrategy_LowQuality(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建低质量记忆（低重要性 + 低访问次数）
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
		AccessCount: 10,
	}
	err = repo.Create(ctx, highQualityMemory)
	require.NoError(t, err)

	// 删除低质量记忆
	count, err := repo.DeleteByStrategy(ctx, tenantID.String(), "low_quality", "soft", 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证低质量记忆被删除
	deleted, err := repo.GetByID(ctx, lowQualityMemory.ID.String())
	assert.Error(t, err)
	assert.Nil(t, deleted)

	// 验证高质量记忆仍然存在
	valid, err := repo.GetByID(ctx, highQualityMemory.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, valid)
}

// TestMemoryRepository_DeleteByStrategy_Unused 测试删除未使用记忆
func TestMemoryRepository_DeleteByStrategy_Unused(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建长时间未访问的记忆
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	unusedMemory := &model.ConversationMemory{
		TenantID:     tenantID,
		SessionID:    sessionID,
		MemoryType:   "long_term",
		Content:      "未使用记忆",
		TokenCount:   10,
		Importance:   0.7,
		LastAccessAt: &oldTime,
	}
	err := repo.Create(ctx, unusedMemory)
	require.NoError(t, err)

	// 创建最近访问的记忆
	recentTime := time.Now()
	recentMemory := &model.ConversationMemory{
		TenantID:     tenantID,
		SessionID:    sessionID,
		MemoryType:   "long_term",
		Content:      "最近使用记忆",
		TokenCount:   10,
		Importance:   0.7,
		LastAccessAt: &recentTime,
	}
	err = repo.Create(ctx, recentMemory)
	require.NoError(t, err)

	// 删除未使用记忆
	count, err := repo.DeleteByStrategy(ctx, tenantID.String(), "unused", "soft", 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证未使用记忆被删除
	deleted, err := repo.GetByID(ctx, unusedMemory.ID.String())
	assert.Error(t, err)
	assert.Nil(t, deleted)

	// 验证最近使用记忆仍然存在
	valid, err := repo.GetByID(ctx, recentMemory.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, valid)
}

// TestMemoryRepository_DeleteByStrategy_HardDelete 测试硬删除
func TestMemoryRepository_DeleteByStrategy_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建过期记忆
	expiredTime := time.Now().Add(-1 * time.Hour)
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "待删除记忆",
		TokenCount: 10,
		Importance: 0.7,
		ExpiresAt:  &expiredTime,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	memoryID := memory.ID.String()

	// 硬删除
	count, err := repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "hard", 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证记录被物理删除（即使查询已删除记录也找不到）
	var deletedMemory model.ConversationMemory
	err = db.Unscoped().Where("id = ?", memoryID).First(&deletedMemory).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// TestMemoryRepository_GetExpiredMemories 测试获取过期记忆
func TestMemoryRepository_GetExpiredMemories(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建过期记忆
	expiredTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 3; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "过期记忆",
			TokenCount: 10,
			Importance: 0.7,
			ExpiresAt:  &expiredTime,
		}
		err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	// 创建未过期记忆
	futureTime := time.Now().Add(1 * time.Hour)
	memory := &model.ConversationMemory{
		TenantID:   tenantID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "有效记忆",
		TokenCount: 10,
		Importance: 0.7,
		ExpiresAt:  &futureTime,
	}
	err := repo.Create(ctx, memory)
	require.NoError(t, err)

	// 获取过期记忆
	filters := &MemoryCleanupFilters{
		TenantID:  tenantID.String(),
		SessionID: sessionID.String(),
		BatchSize: 10,
	}
	expired, err := repo.GetExpiredMemories(ctx, filters)
	assert.NoError(t, err)
	assert.Len(t, expired, 3)
}

// TestMemoryRepository_TenantIsolation 测试租户隔离
func TestMemoryRepository_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 为租户1创建记忆
	memory1 := &model.ConversationMemory{
		TenantID:   tenant1ID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "租户1的记忆",
		TokenCount: 10,
		Importance: 0.7,
	}
	err := repo.Create(ctx, memory1)
	require.NoError(t, err)

	// 为租户2创建记忆
	memory2 := &model.ConversationMemory{
		TenantID:   tenant2ID,
		SessionID:  sessionID,
		MemoryType: "long_term",
		Content:    "租户2的记忆",
		TokenCount: 10,
		Importance: 0.7,
	}
	err = repo.Create(ctx, memory2)
	require.NoError(t, err)

	// 删除租户1的记忆
	count, err := repo.DeleteByStrategy(ctx, tenant1ID.String(), "all", "soft", 100)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证租户1的记忆被删除
	deleted, err := repo.GetByID(ctx, memory1.ID.String())
	assert.Error(t, err)
	assert.Nil(t, deleted)

	// 验证租户2的记忆仍然存在
	valid, err := repo.GetByID(ctx, memory2.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, valid)
	assert.Equal(t, tenant2ID, valid.TenantID)
}

// TestMemoryRepository_BatchSize 测试批量大小限制
func TestMemoryRepository_BatchSize(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGenkitMemoryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建10条过期记忆
	expiredTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 10; i++ {
		memory := &model.ConversationMemory{
			TenantID:   tenantID,
			SessionID:  sessionID,
			MemoryType: "long_term",
			Content:    "过期记忆",
			TokenCount: 10,
			Importance: 0.7,
			ExpiresAt:  &expiredTime,
		}
		err := repo.Create(ctx, memory)
		require.NoError(t, err)
	}

	// 批量删除，限制为5条
	count, err := repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "soft", 5)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)

	// 再次删除剩余的
	count, err = repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "soft", 5)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)

	// 第三次应该没有记录可删除
	count, err = repo.DeleteByStrategy(ctx, tenantID.String(), "expired", "soft", 5)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}
