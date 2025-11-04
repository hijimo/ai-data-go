package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// setupContextTestDB 创建测试数据库
func setupContextTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&model.ConversationContext{}, &model.ConversationSummary{})
	require.NoError(t, err)

	return db
}

// TestContextRepository_Create 测试创建上下文配置
func TestContextRepository_Create(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	context := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	err := repo.Create(ctx, context)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, context.ID)
}

// TestContextRepository_GetBySessionID 测试根据会话ID获取上下文配置
func TestContextRepository_GetBySessionID(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 获取上下文配置
	retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sessionID, retrieved.SessionID)
	assert.Equal(t, 4000, retrieved.MaxTokens)
	assert.Equal(t, "auto", retrieved.Strategy)
}

// TestContextRepository_GetBySessionID_NotFound 测试获取不存在的上下文配置
func TestContextRepository_GetBySessionID_NotFound(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	nonExistentSessionID := uuid.New().String()
	context, err := repo.GetBySessionID(ctx, nonExistentSessionID)
	assert.Error(t, err)
	assert.Nil(t, context)
}

// TestContextRepository_Update 测试更新上下文配置
func TestContextRepository_Update(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		TotalMessages:   0,
	}

	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 更新配置
	context.MaxTokens = 8000
	context.Strategy = "full"
	context.TotalMessages = 10

	err = repo.Update(ctx, context)
	assert.NoError(t, err)

	// 验证更新
	updated, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, 8000, updated.MaxTokens)
	assert.Equal(t, "full", updated.Strategy)
	assert.Equal(t, 10, updated.TotalMessages)
}

// TestContextRepository_UpdateTokenUsage 测试更新Token使用统计
func TestContextRepository_UpdateTokenUsage(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		TotalTokensUsed: 0,
	}

	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 更新Token使用量
	err = repo.UpdateTokenUsage(ctx, sessionID.String(), 100)
	assert.NoError(t, err)

	// 验证更新
	updated, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, int64(100), updated.TotalTokensUsed)

	// 再次更新
	err = repo.UpdateTokenUsage(ctx, sessionID.String(), 200)
	assert.NoError(t, err)

	updated, err = repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, int64(300), updated.TotalTokensUsed)
}

// TestContextRepository_GetLatestSummary 测试获取最新摘要
func TestContextRepository_GetLatestSummary(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	for i := 0; i < 3; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "测试摘要",
			TokenCount:   50,
			MessageCount: 10,
		}
		err := db.Create(summary).Error
		require.NoError(t, err)

		// 等待一小段时间确保创建时间不同
		time.Sleep(10 * time.Millisecond)
	}

	// 获取最新摘要
	latest, err := repo.GetLatestSummary(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, sessionID, latest.SessionID)
}

// TestContextRepository_GetLatestSummary_NotFound 测试获取不存在的摘要
func TestContextRepository_GetLatestSummary_NotFound(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	nonExistentSessionID := uuid.New().String()
	summary, err := repo.GetLatestSummary(ctx, nonExistentSessionID)
	assert.Error(t, err)
	assert.Nil(t, summary)
}

// TestContextRepository_TenantIsolation 测试租户隔离
func TestContextRepository_TenantIsolation(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	session1ID := uuid.New()
	session2ID := uuid.New()

	// 为租户1创建上下文配置
	context1 := &model.ConversationContext{
		TenantID:  tenant1ID,
		SessionID: session1ID,
		MaxTokens: 4000,
		Strategy:  "auto",
	}
	err := repo.Create(ctx, context1)
	require.NoError(t, err)

	// 为租户2创建上下文配置
	context2 := &model.ConversationContext{
		TenantID:  tenant2ID,
		SessionID: session2ID,
		MaxTokens: 8000,
		Strategy:  "full",
	}
	err = repo.Create(ctx, context2)
	require.NoError(t, err)

	// 获取租户1的配置
	retrieved1, err := repo.GetBySessionID(ctx, session1ID.String())
	assert.NoError(t, err)
	assert.Equal(t, tenant1ID, retrieved1.TenantID)
	assert.Equal(t, 4000, retrieved1.MaxTokens)

	// 获取租户2的配置
	retrieved2, err := repo.GetBySessionID(ctx, session2ID.String())
	assert.NoError(t, err)
	assert.Equal(t, tenant2ID, retrieved2.TenantID)
	assert.Equal(t, 8000, retrieved2.MaxTokens)
}

// TestContextRepository_UpdateLastSummary 测试更新最后摘要信息
func TestContextRepository_UpdateLastSummary(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:  tenantID,
		SessionID: sessionID,
		MaxTokens: 4000,
		Strategy:  "auto",
	}
	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "完整摘要",
		TokenCount:   100,
		MessageCount: 20,
	}
	err = db.Create(summary).Error
	require.NoError(t, err)

	// 更新上下文配置的最后摘要信息
	context.LastSummaryID = &summary.ID
	now := time.Now()
	context.LastSummaryAt = &now

	err = repo.Update(ctx, context)
	assert.NoError(t, err)

	// 验证更新
	updated, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.NotNil(t, updated.LastSummaryID)
	assert.Equal(t, summary.ID, *updated.LastSummaryID)
	assert.NotNil(t, updated.LastSummaryAt)
}

// TestContextRepository_IncrementMessageCount 测试增加消息计数
func TestContextRepository_IncrementMessageCount(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:      tenantID,
		SessionID:     sessionID,
		MaxTokens:     4000,
		Strategy:      "auto",
		TotalMessages: 0,
	}
	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 增加消息计数
	context.TotalMessages++
	err = repo.Update(ctx, context)
	assert.NoError(t, err)

	// 验证更新
	updated, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, 1, updated.TotalMessages)

	// 再次增加
	updated.TotalMessages++
	err = repo.Update(ctx, updated)
	assert.NoError(t, err)

	updated, err = repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, 2, updated.TotalMessages)
}

// TestContextRepository_SoftDelete 测试软删除
func TestContextRepository_SoftDelete(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	context := &model.ConversationContext{
		TenantID:  tenantID,
		SessionID: sessionID,
		MaxTokens: 4000,
		Strategy:  "auto",
	}
	err := repo.Create(ctx, context)
	require.NoError(t, err)

	// 软删除
	context.IsDeleted = true
	err = repo.Update(ctx, context)
	assert.NoError(t, err)

	// 验证软删除后查询不到
	deleted, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.Error(t, err)
	assert.Nil(t, deleted)
}

// TestContextRepository_MultipleStrategies 测试不同策略
func TestContextRepository_MultipleStrategies(t *testing.T) {
	db := setupContextTestDB(t)
	repo := NewGenkitContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()

	strategies := []string{"auto", "short", "full"}

	for _, strategy := range strategies {
		sessionID := uuid.New()
		context := &model.ConversationContext{
			TenantID:  tenantID,
			SessionID: sessionID,
			MaxTokens: 4000,
			Strategy:  strategy,
		}

		err := repo.Create(ctx, context)
		assert.NoError(t, err)

		// 验证策略正确保存
		retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
		assert.NoError(t, err)
		assert.Equal(t, strategy, retrieved.Strategy)
	}
}
