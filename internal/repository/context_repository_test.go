package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"genkit-ai-service/internal/model"
)

// TestContextRepository_Create 测试创建上下文配置
func TestContextRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}

	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err, "创建上下文配置失败")
	assert.NotEqual(t, uuid.Nil, contextConfig.ID, "上下文配置ID应该被自动生成")
}

// TestContextRepository_GetBySessionID 测试根据会话ID获取上下文配置
func TestContextRepository_GetBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 测试获取上下文配置
	retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
	require.NoError(t, err, "获取上下文配置失败")
	assert.Equal(t, contextConfig.ID, retrieved.ID)
	assert.Equal(t, contextConfig.MaxTokens, retrieved.MaxTokens)
	assert.Equal(t, contextConfig.Strategy, retrieved.Strategy)
}

// TestContextRepository_GetBySessionID_NotFound 测试获取不存在的上下文配置
func TestContextRepository_GetBySessionID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	nonExistentSessionID := uuid.New()

	// 尝试获取不存在的上下文配置
	_, err := repo.GetBySessionID(ctx, nonExistentSessionID.String())
	assert.Error(t, err, "应该返回错误")
}

// TestContextRepository_GetBySessionID_SoftDeleted 测试软删除过滤
func TestContextRepository_GetBySessionID_SoftDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 软删除上下文配置
	db.Model(&model.ConversationContext{}).
		Where("id = ?", contextConfig.ID).
		Update("is_deleted", true)

	// 尝试获取已软删除的上下文配置（应该失败）
	_, err = repo.GetBySessionID(ctx, sessionID.String())
	assert.Error(t, err, "不应该获取到已软删除的上下文配置")
}

// TestContextRepository_Update 测试更新上下文配置
func TestContextRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 更新配置
	contextConfig.MaxTokens = 8000
	contextConfig.Strategy = "full"
	contextConfig.ShortTermWindow = 20

	err = repo.Update(ctx, contextConfig)
	require.NoError(t, err, "更新上下文配置失败")

	// 验证更新
	retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
	require.NoError(t, err)
	assert.Equal(t, 8000, retrieved.MaxTokens)
	assert.Equal(t, "full", retrieved.Strategy)
	assert.Equal(t, 20, retrieved.ShortTermWindow)
}

// TestContextRepository_Update_TenantIsolation 测试更新时的租户隔离
func TestContextRepository_Update_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenant1ID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 尝试用租户2的ID更新租户1的配置（应该失败）
	contextConfig.TenantID = tenant2ID
	contextConfig.MaxTokens = 8000

	err = repo.Update(ctx, contextConfig)
	assert.Error(t, err, "不应该允许跨租户更新")
	assert.Contains(t, err.Error(), "权限不足", "错误信息应该包含权限不足")
}

// TestContextRepository_GetLatestSummary 测试获取最新摘要
func TestContextRepository_GetLatestSummary(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	summary1 := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "第一个摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	summary2 := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "第二个摘要",
		TokenCount:   150,
		MessageCount: 15,
	}

	db.Create(summary1)
	db.Create(summary2)

	// 获取最新摘要
	latest, err := repo.GetLatestSummary(ctx, sessionID.String())
	require.NoError(t, err, "获取最新摘要失败")
	assert.Equal(t, summary2.ID, latest.ID, "应该返回最新的摘要")
	assert.Equal(t, "第二个摘要", latest.Content)
}

// TestContextRepository_GetLatestSummary_NotFound 测试获取不存在的摘要
func TestContextRepository_GetLatestSummary_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	nonExistentSessionID := uuid.New()

	// 尝试获取不存在的摘要
	_, err := repo.GetLatestSummary(ctx, nonExistentSessionID.String())
	assert.Error(t, err, "应该返回错误")
}

// TestContextRepository_GetLatestSummary_SoftDeleted 测试软删除过滤
func TestContextRepository_GetLatestSummary_SoftDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "测试摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	db.Create(summary)

	// 软删除摘要
	db.Model(&model.ConversationSummary{}).
		Where("id = ?", summary.ID).
		Update("is_deleted", true)

	// 尝试获取已软删除的摘要（应该失败）
	_, err := repo.GetLatestSummary(ctx, sessionID.String())
	assert.Error(t, err, "不应该获取到已软删除的摘要")
}

// TestContextRepository_UpdateTokenUsage 测试更新Token使用统计
func TestContextRepository_UpdateTokenUsage(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		TotalMessages:   0,
		TotalTokensUsed: 0,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 更新Token使用统计
	err = repo.UpdateTokenUsage(ctx, sessionID.String(), 100)
	require.NoError(t, err, "更新Token使用统计失败")

	// 验证统计已更新
	retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(100), retrieved.TotalTokensUsed, "Token使用量应该增加100")
	assert.Equal(t, 1, retrieved.TotalMessages, "消息计数应该增加1")

	// 再次更新
	err = repo.UpdateTokenUsage(ctx, sessionID.String(), 200)
	require.NoError(t, err)

	// 验证累计统计
	retrieved, err = repo.GetBySessionID(ctx, sessionID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(300), retrieved.TotalTokensUsed, "Token使用量应该累计到300")
	assert.Equal(t, 2, retrieved.TotalMessages, "消息计数应该累计到2")
}

// TestContextRepository_UpdateTokenUsage_NotFound 测试更新不存在的配置
func TestContextRepository_UpdateTokenUsage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	nonExistentSessionID := uuid.New()

	// 尝试更新不存在的配置
	err := repo.UpdateTokenUsage(ctx, nonExistentSessionID.String(), 100)
	assert.Error(t, err, "应该返回错误")
}

// TestContextRepository_UpdateTokenUsage_SoftDeleted 测试更新已软删除的配置
func TestContextRepository_UpdateTokenUsage_SoftDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 软删除配置
	db.Model(&model.ConversationContext{}).
		Where("id = ?", contextConfig.ID).
		Update("is_deleted", true)

	// 尝试更新已软删除的配置（应该失败）
	err = repo.UpdateTokenUsage(ctx, sessionID.String(), 100)
	assert.Error(t, err, "不应该更新已软删除的配置")
}

// TestContextRepository_ConcurrentUpdates 测试并发更新
func TestContextRepository_ConcurrentUpdates(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewContextRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建上下文配置
	contextConfig := &model.ConversationContext{
		TenantID:        tenantID,
		SessionID:       sessionID,
		MaxTokens:       4000,
		Strategy:        "auto",
		IncludeSummary:  true,
		IncludeLongTerm: true,
		ShortTermWindow: 10,
		TotalMessages:   0,
		TotalTokensUsed: 0,
	}
	err := repo.Create(ctx, contextConfig)
	require.NoError(t, err)

	// 并发更新Token使用统计
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			err := repo.UpdateTokenUsage(ctx, sessionID.String(), 10)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终统计
	retrieved, err := repo.GetBySessionID(ctx, sessionID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(100), retrieved.TotalTokensUsed, "Token使用量应该累计到100")
	assert.Equal(t, 10, retrieved.TotalMessages, "消息计数应该累计到10")
}
