package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"genkit-ai-service/internal/model"
)

// setupSummaryTestDB 创建测试数据库
func setupSummaryTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(&model.ConversationSummary{})
	require.NoError(t, err)

	return db
}

// TestSummaryRepository_Create 测试创建摘要
func TestSummaryRepository_Create(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "这是一个完整的对话摘要",
		TokenCount:   100,
		MessageCount: 20,
	}

	err := repo.Create(ctx, summary)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, summary.ID)
}

// TestSummaryRepository_GetByID 测试根据ID获取摘要
func TestSummaryRepository_GetByID(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "增量摘要",
		TokenCount:   50,
		MessageCount: 10,
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 获取摘要
	retrieved, err := repo.GetByID(ctx, summary.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, summary.ID, retrieved.ID)
	assert.Equal(t, summary.Content, retrieved.Content)
}

// TestSummaryRepository_GetBySessionID 测试根据会话ID获取摘要列表
func TestSummaryRepository_GetBySessionID(t *testing.T) {
	db := setupSummaryTestDB(t)
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
			Content:      "测试摘要",
			TokenCount:   50,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 获取会话的所有摘要
	summaries, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Len(t, summaries, 5)
}

// TestSummaryRepository_GetLatestBySessionID 测试获取最新摘要
func TestSummaryRepository_GetLatestBySessionID(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	var lastSummary *model.ConversationSummary
	for i := 0; i < 3; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "测试摘要",
			TokenCount:   50,
			MessageCount: 10 * (i + 1),
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
		lastSummary = summary
	}

	// 获取最新摘要
	latest, err := repo.GetLatestBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, lastSummary.ID, latest.ID)
	assert.Equal(t, 30, latest.MessageCount) // 最后一个摘要的消息数
}

// TestSummaryRepository_Update 测试更新摘要
func TestSummaryRepository_Update(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "incremental",
		Content:      "原始摘要",
		TokenCount:   50,
		MessageCount: 10,
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 更新摘要
	qualityScore := 0.85
	compressionRate := 0.6
	summary.Content = "更新后的摘要"
	summary.QualityScore = &qualityScore
	summary.CompressionRate = &compressionRate
	summary.KeyTopics = []string{"主题1", "主题2", "主题3"}

	err = repo.Update(ctx, summary)
	assert.NoError(t, err)

	// 验证更新
	updated, err := repo.GetByID(ctx, summary.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, "更新后的摘要", updated.Content)
	assert.NotNil(t, updated.QualityScore)
	assert.Equal(t, 0.85, *updated.QualityScore)
	assert.NotNil(t, updated.CompressionRate)
	assert.Equal(t, 0.6, *updated.CompressionRate)
	assert.Len(t, updated.KeyTopics, 3)
}

// TestSummaryRepository_Delete 测试删除摘要
func TestSummaryRepository_Delete(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "待删除摘要",
		TokenCount:   100,
		MessageCount: 20,
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 软删除
	err = repo.Delete(ctx, summary.ID.String())
	assert.NoError(t, err)

	// 验证软删除后查询不到
	deleted, err := repo.GetByID(ctx, summary.ID.String())
	assert.Error(t, err)
	assert.Nil(t, deleted)
}

// TestSummaryRepository_SummaryTypes 测试不同摘要类型
func TestSummaryRepository_SummaryTypes(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	summaryTypes := []string{"incremental", "full"}

	for _, summaryType := range summaryTypes {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  summaryType,
			Content:      "测试摘要",
			TokenCount:   50,
			MessageCount: 10,
		}

		err := repo.Create(ctx, summary)
		assert.NoError(t, err)

		// 验证类型正确保存
		retrieved, err := repo.GetByID(ctx, summary.ID.String())
		assert.NoError(t, err)
		assert.Equal(t, summaryType, retrieved.SummaryType)
	}
}

// TestSummaryRepository_WithKeyTopics 测试关键主题
func TestSummaryRepository_WithKeyTopics(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建带关键主题的摘要
	summary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "完整摘要",
		TokenCount:   100,
		MessageCount: 20,
		KeyTopics:    []string{"人工智能", "机器学习", "深度学习", "自然语言处理"},
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 获取并验证关键主题
	retrieved, err := repo.GetByID(ctx, summary.ID.String())
	assert.NoError(t, err)
	assert.Len(t, retrieved.KeyTopics, 4)
	assert.Contains(t, retrieved.KeyTopics, "人工智能")
	assert.Contains(t, retrieved.KeyTopics, "机器学习")
}

// TestSummaryRepository_WithQualityMetrics 测试质量指标
func TestSummaryRepository_WithQualityMetrics(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	qualityScore := 0.92
	compressionRate := 0.75

	// 创建带质量指标的摘要
	summary := &model.ConversationSummary{
		TenantID:        tenantID,
		SessionID:       sessionID,
		SummaryType:     "full",
		Content:         "高质量摘要",
		TokenCount:      100,
		MessageCount:    20,
		QualityScore:    &qualityScore,
		CompressionRate: &compressionRate,
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 获取并验证质量指标
	retrieved, err := repo.GetByID(ctx, summary.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.QualityScore)
	assert.Equal(t, 0.92, *retrieved.QualityScore)
	assert.NotNil(t, retrieved.CompressionRate)
	assert.Equal(t, 0.75, *retrieved.CompressionRate)
}

// TestSummaryRepository_WithMessageRange 测试消息范围
func TestSummaryRepository_WithMessageRange(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()
	startMessageID := uuid.New()
	endMessageID := uuid.New()

	// 创建带消息范围的摘要
	summary := &model.ConversationSummary{
		TenantID:       tenantID,
		SessionID:      sessionID,
		SummaryType:    "incremental",
		Content:        "增量摘要",
		TokenCount:     50,
		MessageCount:   10,
		StartMessageID: &startMessageID,
		EndMessageID:   &endMessageID,
	}

	err := repo.Create(ctx, summary)
	require.NoError(t, err)

	// 获取并验证消息范围
	retrieved, err := repo.GetByID(ctx, summary.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.StartMessageID)
	assert.Equal(t, startMessageID, *retrieved.StartMessageID)
	assert.NotNil(t, retrieved.EndMessageID)
	assert.Equal(t, endMessageID, *retrieved.EndMessageID)
}

// TestSummaryRepository_WithPreviousSummary 测试前一个摘要引用
func TestSummaryRepository_WithPreviousSummary(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建第一个摘要
	firstSummary := &model.ConversationSummary{
		TenantID:     tenantID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "第一个摘要",
		TokenCount:   100,
		MessageCount: 20,
	}

	err := repo.Create(ctx, firstSummary)
	require.NoError(t, err)

	// 创建第二个摘要，引用第一个
	secondSummary := &model.ConversationSummary{
		TenantID:          tenantID,
		SessionID:         sessionID,
		SummaryType:       "incremental",
		Content:           "第二个摘要",
		TokenCount:        50,
		MessageCount:      10,
		PreviousSummaryID: &firstSummary.ID,
	}

	err = repo.Create(ctx, secondSummary)
	require.NoError(t, err)

	// 获取并验证前一个摘要引用
	retrieved, err := repo.GetByID(ctx, secondSummary.ID.String())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved.PreviousSummaryID)
	assert.Equal(t, firstSummary.ID, *retrieved.PreviousSummaryID)
}

// TestSummaryRepository_TenantIsolation 测试租户隔离
func TestSummaryRepository_TenantIsolation(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	sessionID := uuid.New()

	// 为租户1创建摘要
	summary1 := &model.ConversationSummary{
		TenantID:     tenant1ID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "租户1的摘要",
		TokenCount:   100,
		MessageCount: 20,
	}
	err := repo.Create(ctx, summary1)
	require.NoError(t, err)

	// 为租户2创建摘要
	summary2 := &model.ConversationSummary{
		TenantID:     tenant2ID,
		SessionID:    sessionID,
		SummaryType:  "full",
		Content:      "租户2的摘要",
		TokenCount:   100,
		MessageCount: 20,
	}
	err = repo.Create(ctx, summary2)
	require.NoError(t, err)

	// 验证租户1的摘要
	retrieved1, err := repo.GetByID(ctx, summary1.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, tenant1ID, retrieved1.TenantID)
	assert.Equal(t, "租户1的摘要", retrieved1.Content)

	// 验证租户2的摘要
	retrieved2, err := repo.GetByID(ctx, summary2.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, tenant2ID, retrieved2.TenantID)
	assert.Equal(t, "租户2的摘要", retrieved2.Content)
}

// TestSummaryRepository_CountBySessionID 测试统计会话摘要数量
func TestSummaryRepository_CountBySessionID(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	sessionID := uuid.New()

	// 创建多个摘要
	for i := 0; i < 7; i++ {
		summary := &model.ConversationSummary{
			TenantID:     tenantID,
			SessionID:    sessionID,
			SummaryType:  "incremental",
			Content:      "测试摘要",
			TokenCount:   50,
			MessageCount: 10,
		}
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 统计摘要数量
	count, err := repo.CountBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Equal(t, int64(7), count)
}

// TestSummaryRepository_DeleteBySessionID 测试删除会话的所有摘要
func TestSummaryRepository_DeleteBySessionID(t *testing.T) {
	db := setupSummaryTestDB(t)
	repo := NewSummaryRepository(db)
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
		err := repo.Create(ctx, summary)
		require.NoError(t, err)
	}

	// 删除会话的所有摘要
	err := repo.DeleteBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)

	// 验证所有摘要都被删除
	summaries, err := repo.GetBySessionID(ctx, sessionID.String())
	assert.NoError(t, err)
	assert.Len(t, summaries, 0)
}
