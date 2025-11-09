package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"genkit-ai-service/internal/model"
)

// TestSummaryRepository_Create 测试创建摘要
func TestSummaryRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	qualityScore := 0.85
	compressionRate := 0.7

	summary := &model.ConversationSummary{
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "incremental",
		Content:         "这是一个测试摘要",
		TokenCount:      100,
		MessageCount:    10,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
		KeyTopics:       []string{"主题1", "主题2"},
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err, "创建摘要失败")
	assert.NotEqual(t, uuid.Nil, summary.ID, "摘要ID应该被自动生成")
}

// TestSummaryRepository_GetByID 测试根据ID获取摘要
func TestSummaryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
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
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 测试获取摘要
	retrieved, err := repo.GetByID(ctx, tenantID, summary.ID)
	require.NoError(t, err, "获取摘要失败")
	assert.Equal(t, summary.ID, retrieved.ID)
	assert.Equal(t, summary.Content, retrieved.Content)
	assert.Equal(t, summary.TenantID, retrieved.TenantID)
}

// TestSummaryRepository_GetByID_TenantIsolation 测试租户隔离
func TestSummaryRepository_GetByID_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的摘要
	summary := &model.ConversationSummary{
		TenantID:     tenant1ID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "租户1的摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 尝试用租户2的ID获取租户1的摘要（应该失败）
	_, err = repo.GetByID(ctx, tenant2ID, summary.ID)
	assert.Error(t, err, "应该无法获取其他租户的摘要")
	assert.Equal(t, ErrNotFound, err, "应该返回 ErrNotFound")
}

// TestSummaryRepository_GetLatestBySessionID 测试获取会话最新摘要
func TestSummaryRepository_GetLatestBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
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

	err := repo.Create(ctx, summary1)
	require.NoError(t, err)
	err = repo.Create(ctx, summary2)
	require.NoError(t, err)

	// 获取最新摘要
	latest, err := repo.GetLatestBySessionID(ctx, tenantID, sessionID)
	require.NoError(t, err, "获取最新摘要失败")
	assert.Equal(t, summary2.ID, latest.ID, "应该返回最新的摘要")
	assert.Equal(t, "第二个摘要", latest.Content)
}

// TestSummaryRepository_ListBySessionID 测试获取会话摘要列表
func TestSummaryRepository_ListBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	for i := 0; i < 5; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "摘要内容",
			TokenCount:   100,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 获取所有摘要
	summaries, err := repo.ListBySessionID(ctx, tenantID, sessionID, 0)
	require.NoError(t, err, "获取摘要列表失败")
	assert.Len(t, summaries, 5, "应该返回5个摘要")

	// 获取限制数量的摘要
	limitedSummaries, err := repo.ListBySessionID(ctx, tenantID, sessionID, 3)
	require.NoError(t, err, "获取限制数量的摘要列表失败")
	assert.Len(t, limitedSummaries, 3, "应该返回3个摘要")
}

// TestSummaryRepository_ListBySessionID_TenantIsolation 测试列表查询的租户隔离
func TestSummaryRepository_ListBySessionID_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的摘要
	summary1 := &model.ConversationSummary{
		TenantID:     tenant1ID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "租户1的摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	err := repo.Create(ctx, summary1)
	require.NoError(t, err)

	// 创建租户2的摘要（相同会话ID）
	summary2 := &model.ConversationSummary{
		TenantID:     tenant2ID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "租户2的摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	err = repo.Create(ctx, summary2)
	require.NoError(t, err)

	// 租户1查询摘要列表
	tenant1Summaries, err := repo.ListBySessionID(ctx, tenant1ID, sessionID, 0)
	require.NoError(t, err)
	assert.Len(t, tenant1Summaries, 1, "租户1应该只能看到自己的摘要")
	assert.Equal(t, "租户1的摘要", tenant1Summaries[0].Content)

	// 租户2查询摘要列表
	tenant2Summaries, err := repo.ListBySessionID(ctx, tenant2ID, sessionID, 0)
	require.NoError(t, err)
	assert.Len(t, tenant2Summaries, 1, "租户2应该只能看到自己的摘要")
	assert.Equal(t, "租户2的摘要", tenant2Summaries[0].Content)
}

// TestSummaryRepository_Update 测试更新摘要
func TestSummaryRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "原始内容",
		TokenCount:   100,
		MessageCount: 10,
	}
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 更新摘要
	newQualityScore := 0.9
	summary.Content = "更新后的内容"
	summary.TokenCount = 150
	summary.QualityScore = &newQualityScore

	err = repo.Update(ctx, summary)
	require.NoError(t, err, "更新摘要失败")

	// 验证更新
	retrieved, err := repo.GetByID(ctx, tenantID, summary.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的内容", retrieved.Content)
	assert.Equal(t, 150, retrieved.TokenCount)
	assert.Equal(t, 0.9, *retrieved.QualityScore)
}

// TestSummaryRepository_Update_TenantIsolation 测试更新时的租户隔离
func TestSummaryRepository_Update_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的摘要
	summary := &model.ConversationSummary{
		TenantID:     tenant1ID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "租户1的摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 尝试用租户2的ID更新租户1的摘要（应该失败）
	summary.TenantID = tenant2ID
	summary.Content = "尝试修改"

	err = repo.Update(ctx, summary)
	assert.Error(t, err, "不应该允许跨租户更新")
	assert.Contains(t, err.Error(), "权限不足", "错误信息应该包含权限不足")
}

// TestSummaryRepository_SoftDelete 测试软删除
func TestSummaryRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
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
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 软删除
	err = repo.SoftDelete(ctx, tenantID, summary.ID)
	require.NoError(t, err, "软删除失败")

	// 验证摘要已被软删除（无法通过GetByID获取）
	_, err = repo.GetByID(ctx, tenantID, summary.ID)
	assert.Error(t, err, "软删除的摘要不应该被查询到")
	assert.Equal(t, ErrNotFound, err)
}

// TestSummaryRepository_HardDelete 测试硬删除
func TestSummaryRepository_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
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
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 硬删除
	err = repo.HardDelete(ctx, tenantID, summary.ID)
	require.NoError(t, err, "硬删除失败")

	// 验证摘要已被物理删除
	var count int64
	db.Model(&model.ConversationSummary{}).Where("id = ?", summary.ID).Count(&count)
	assert.Equal(t, int64(0), count, "摘要应该被物理删除")
}

// TestSummaryRepository_GetByType 测试根据类型获取摘要列表
func TestSummaryRepository_GetByType(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建不同类型的摘要
	incrementalSummary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "增量摘要",
		TokenCount:   100,
		MessageCount: 10,
	}
	fullSummary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "完整摘要",
		TokenCount:   200,
		MessageCount: 20,
	}

	err := repo.Create(ctx, incrementalSummary)
	require.NoError(t, err)
	err = repo.Create(ctx, fullSummary)
	require.NoError(t, err)

	// 获取增量摘要
	incrementalSummaries, err := repo.GetByType(ctx, tenantID, sessionID, "incremental", 0)
	require.NoError(t, err, "获取增量摘要失败")
	assert.Len(t, incrementalSummaries, 1, "应该返回1个增量摘要")
	assert.Equal(t, "incremental", incrementalSummaries[0].SummaryType)

	// 获取完整摘要
	fullSummaries, err := repo.GetByType(ctx, tenantID, sessionID, "full", 0)
	require.NoError(t, err, "获取完整摘要失败")
	assert.Len(t, fullSummaries, 1, "应该返回1个完整摘要")
	assert.Equal(t, "full", fullSummaries[0].SummaryType)
}

// TestSummaryRepository_CountBySessionID 测试统计会话摘要数量
func TestSummaryRepository_CountBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	for i := 0; i < 5; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "摘要内容",
			TokenCount:   100,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 统计摘要数量
	count, err := repo.CountBySessionID(ctx, tenantID, sessionID)
	require.NoError(t, err, "统计摘要数量失败")
	assert.Equal(t, int64(5), count, "应该有5个摘要")
}

// TestSummaryRepository_CountBySessionID_TenantIsolation 测试统计时的租户隔离
func TestSummaryRepository_CountBySessionID_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 创建租户1的摘要
	for i := 0; i < 3; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenant1ID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "租户1的摘要",
			TokenCount:   100,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 创建租户2的摘要（相同会话ID）
	for i := 0; i < 2; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenant2ID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "租户2的摘要",
			TokenCount:   100,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 租户1统计摘要数量
	tenant1Count, err := repo.CountBySessionID(ctx, tenant1ID, sessionID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), tenant1Count, "租户1应该有3个摘要")

	// 租户2统计摘要数量
	tenant2Count, err := repo.CountBySessionID(ctx, tenant2ID, sessionID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), tenant2Count, "租户2应该有2个摘要")
}

// TestSummaryRepository_SoftDeleteFilter 测试软删除过滤
func TestSummaryRepository_SoftDeleteFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	for i := 0; i < 5; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "摘要内容",
			TokenCount:   100,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 软删除其中2个摘要
	summaries, _ := repo.ListBySessionID(ctx, tenantID, sessionID, 0)
	err := repo.SoftDelete(ctx, tenantID, summaries[0].ID)
	require.NoError(t, err)
	err = repo.SoftDelete(ctx, tenantID, summaries[1].ID)
	require.NoError(t, err)

	// 查询摘要列表（应该只返回未删除的）
	activeSummaries, err := repo.ListBySessionID(ctx, tenantID, sessionID, 0)
	require.NoError(t, err)
	assert.Len(t, activeSummaries, 3, "应该只返回3个未删除的摘要")

	// 统计摘要数量（应该只统计未删除的）
	count, err := repo.CountBySessionID(ctx, tenantID, sessionID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "应该只统计3个未删除的摘要")
}

// TestSummaryRepository_KeyTopics 测试关键主题数组
func TestSummaryRepository_KeyTopics(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建带关键主题的摘要
	keyTopics := []string{"人工智能", "机器学习", "深度学习"}
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "关于AI的讨论",
		TokenCount:   100,
		MessageCount: 10,
		KeyTopics:    keyTopics,
	}
	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 验证关键主题已保存
	retrieved, err := repo.GetByID(ctx, tenantID, summary.ID)
	require.NoError(t, err)
	assert.Equal(t, keyTopics, retrieved.KeyTopics, "关键主题应该被正确保存")
}
