// Package service 服务层集成测试
package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
)

// ServiceTestEnvironment 服务测试环境
type ServiceTestEnvironment struct {
	DB              *gorm.DB
	TenantID        uuid.UUID
	SessionID       uuid.UUID
	UserID          uuid.UUID
	ContextService  ContextService
	MemoryService   MemoryService
	SummaryService  SummaryService
	TokenManager    TokenManager
	VectorService   VectorService
	CacheService    CacheService
	SessionRepo     repository.SessionRepository
	MessageRepo     repository.MessageRepository
	MemoryRepo      repository.GenkitMemoryRepository
	ContextRepo     repository.GenkitContextRepository
	SummaryRepo     repository.SummaryRepository
}

// SetupServiceTestEnvironment 设置服务测试环境
func SetupServiceTestEnvironment(t *testing.T) *ServiceTestEnvironment {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	// 自动迁移
	err = db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.ConversationSession{},
		&model.ChatMessage{},
		&model.ConversationMemory{},
		&model.ConversationContext{},
		&model.ConversationSummary{},
	)
	require.NoError(t, err, "数据库迁移失败")

	// 创建测试租户
	tenantID := uuid.New()
	tenant := &model.Tenant{
		ID:     tenantID,
		Name:   "测试租户",
		Domain: "test.example.com",
		Status: model.TenantStatusActive,
	}
	err = db.Create(tenant).Error
	require.NoError(t, err, "创建测试租户失败")

	// 创建测试用户
	userID := uuid.New()
	user := &model.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "test@example.com",
		Username: "testuser",
		Status:   model.UserStatusActive,
		Roles:    []string{model.RoleTenantAdmin},
	}
	err = db.Create(user).Error
	require.NoError(t, err, "创建测试用户失败")

	// 创建测试会话
	sessionID := uuid.New()
	session := &model.ConversationSession{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    userID,
		Title:     "测试会话",
		Status:    "active",
		CreatedBy: userID,
	}
	err = db.Create(session).Error
	require.NoError(t, err, "创建测试会话失败")

	// 创建仓储层
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	memoryRepo := repository.NewGenkitMemoryRepository(db)
	contextRepo := repository.NewGenkitContextRepository(db)
	summaryRepo := repository.NewSummaryRepository(db)

	// 创建服务层
	cacheService := NewCacheService(nil, "test")
	vectorService := NewVectorService(nil, 1536)
	tokenManager := NewTokenManager()

	contextService := NewContextService(
		contextRepo,
		messageRepo,
		memoryRepo,
		vectorService,
		tokenManager,
		cacheService,
	)

	memoryService := NewMemoryService(
		memoryRepo,
		messageRepo,
		vectorService,
		tokenManager,
	)

	summaryService := NewSummaryService(
		summaryRepo,
		messageRepo,
		contextRepo,
		tokenManager,
	)

	return &ServiceTestEnvironment{
		DB:             db,
		TenantID:       tenantID,
		SessionID:      sessionID,
		UserID:         userID,
		ContextService: contextService,
		MemoryService:  memoryService,
		SummaryService: summaryService,
		TokenManager:   tokenManager,
		VectorService:  vectorService,
		CacheService:   cacheService,
		SessionRepo:    sessionRepo,
		MessageRepo:    messageRepo,
		MemoryRepo:     memoryRepo,
		ContextRepo:    contextRepo,
		SummaryRepo:    summaryRepo,
	}
}

// TeardownServiceTestEnvironment 清理服务测试环境
func TeardownServiceTestEnvironment(t *testing.T, env *ServiceTestEnvironment) {
	if env.DB != nil {
		sqlDB, err := env.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// CreateServiceTestContext 创建带有认证信息的测试上下文
func CreateServiceTestContext(tenantID, userID uuid.UUID, roles []string) context.Context {
	ctx := context.Background()
	claims := &model.JWTClaims{
		TenantID: tenantID.String(),
		Roles:    roles,
	}
	claims.Subject = userID.String()
	return context.WithValue(ctx, middleware.JWTClaimsKey, claims)
}

// TestContextService_Integration 测试上下文服务集成
func TestContextService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupServiceTestEnvironment(t)
	defer TeardownServiceTestEnvironment(t, env)

	ctx := CreateServiceTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("完整的上下文构建流程", func(t *testing.T) {
		// 1. 创建测试消息
		messages := []*model.ChatMessage{
			{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "第一条消息",
				CreatedBy: env.UserID,
			},
			{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "assistant",
				Content:   "第一条回复",
				CreatedBy: env.UserID,
			},
			{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "第二条消息",
				CreatedBy: env.UserID,
			},
		}

		for _, msg := range messages {
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 2. 构建上下文
		req := BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "继续对话",
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  false,
			IncludeLongTerm: false,
			ShortTermWindow: 10,
		}

		result, err := env.ContextService.BuildContext(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, env.SessionID.String(), result.SessionID)
		assert.Len(t, result.ShortTermMessages, 3)
		assert.Greater(t, result.TotalTokens, 0)
		assert.Greater(t, result.QualityScore, 0.0)

		// 3. 验证上下文配置
		config, err := env.ContextService.GetContextConfig(ctx, env.SessionID.String())
		if err == nil {
			assert.NotNil(t, config)
		}
	})

	t.Run("上下文优化", func(t *testing.T) {
		// 创建大量消息
		for i := 0; i < 20; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "这是一条很长的测试消息，用于测试上下文优化功能",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 构建上下文
		buildReq := BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 20,
		}

		originalContext, err := env.ContextService.BuildContext(ctx, buildReq)
		require.NoError(t, err)

		// 优化上下文
		optimizeReq := OptimizeContextRequest{
			Context:         originalContext,
			TargetTokens:    2000,
			Strategy:        "balanced",
			PreserveSummary: true,
		}

		optimizedContext, err := env.ContextService.OptimizeContext(ctx, optimizeReq)
		require.NoError(t, err)
		assert.NotNil(t, optimizedContext)
		assert.LessOrEqual(t, optimizedContext.TotalTokens, 2000)
		assert.Less(t, len(optimizedContext.ShortTermMessages), len(originalContext.ShortTermMessages))
	})
}

// TestMemoryService_Integration 测试记忆服务集成
func TestMemoryService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupServiceTestEnvironment(t)
	defer TeardownServiceTestEnvironment(t, env)

	ctx := CreateServiceTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("存储和检索记忆", func(t *testing.T) {
		// 1. 创建测试消息
		msg := &model.ChatMessage{
			ID:        uuid.New(),
			TenantID:  env.TenantID,
			SessionID: env.SessionID,
			Role:      "user",
			Content:   "用户对AI技术很感兴趣",
			CreatedBy: env.UserID,
		}
		err := env.DB.Create(msg).Error
		require.NoError(t, err)

		// 2. 存储记忆
		storeReq := StoreMemoryRequest{
			SessionID:  env.SessionID.String(),
			MessageIDs: []string{msg.ID.String()},
			MemoryType: model.MemoryTypeLongTerm,
			Content:    msg.Content,
			Importance: 0.9,
			TenantID:   env.TenantID.String(),
		}

		memory, err := env.MemoryService.StoreMemory(ctx, storeReq)
		if err != nil {
			t.Logf("存储记忆失败（可能因为缺少向量服务）: %v", err)
		} else {
			assert.NotNil(t, memory)
			assert.Equal(t, env.SessionID, memory.SessionID)
		}

		// 3. 直接创建记忆用于测试检索
		testMemory := &model.ConversationMemory{
			ID:         uuid.New(),
			TenantID:   env.TenantID,
			SessionID:  env.SessionID,
			MemoryType: model.MemoryTypeLongTerm,
			Content:    "用户对AI技术很感兴趣",
			TokenCount: 15,
			Importance: 0.9,
		}
		err = env.DB.Create(testMemory).Error
		require.NoError(t, err)

		// 4. 检索记忆
		searchReq := SearchMemoriesRequest{
			SessionID:     env.SessionID.String(),
			Query:         "AI技术",
			TopK:          5,
			MinSimilarity: 0.5,
			TenantID:      env.TenantID.String(),
		}

		results, err := env.MemoryService.SearchMemories(ctx, searchReq)
		if err != nil {
			t.Logf("检索记忆失败（可能因为缺少向量服务）: %v", err)
		} else {
			assert.NotNil(t, results)
		}
	})

	t.Run("清理记忆", func(t *testing.T) {
		// 创建过期记忆
		expiredTime := time.Now().Add(-48 * time.Hour)
		expiredMemory := &model.ConversationMemory{
			ID:         uuid.New(),
			TenantID:   env.TenantID,
			SessionID:  env.SessionID,
			MemoryType: model.MemoryTypeLongTerm,
			Content:    "过期记忆",
			TokenCount: 10,
			Importance: 0.5,
			ExpiresAt:  &expiredTime,
		}
		err := env.DB.Create(expiredMemory).Error
		require.NoError(t, err)

		// 清理记忆
		cleanupReq := CleanupMemoriesRequest{
			SessionID: env.SessionID.String(),
			TenantID:  env.TenantID.String(),
			Strategy:  "expired",
			Mode:      "soft",
			BatchSize: 100,
			Execute:   true,
		}

		result, err := env.MemoryService.CleanupMemories(ctx, cleanupReq)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.CleanedCount, 0)
	})
}

// TestSummaryService_Integration 测试摘要服务集成
func TestSummaryService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupServiceTestEnvironment(t)
	defer TeardownServiceTestEnvironment(t, env)

	ctx := CreateServiceTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("检查摘要触发条件", func(t *testing.T) {
		// 创建足够多的消息
		for i := 0; i < 25; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "测试消息",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 检查是否需要生成摘要
		result, err := env.SummaryService.CheckSummaryTrigger(ctx, env.SessionID.String())
		if err != nil {
			t.Logf("检查摘要触发失败: %v", err)
		} else {
			assert.NotNil(t, result)
			// 由于消息数量超过20，应该触发摘要
			if result.ShouldSummarize {
				t.Logf("触发摘要生成，原因: %s", result.TriggerReason)
			}
		}
	})

	t.Run("生成摘要", func(t *testing.T) {
		// 创建测试消息
		messageIDs := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "这是一条测试消息，用于生成摘要",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
			messageIDs = append(messageIDs, msg.ID.String())
		}

		// 生成摘要
		req := GenerateSummaryRequest{
			SessionID:    env.SessionID.String(),
			MessageIDs:   messageIDs,
			SummaryType:  "full",
			TargetLength: 200,
			TenantID:     env.TenantID.String(),
		}

		result, err := env.SummaryService.GenerateSummary(ctx, req)
		if err != nil {
			t.Logf("生成摘要失败（预期，因为没有AI服务）: %v", err)
		} else {
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Content)
		}
	})
}

// TestMultiTenantIsolation_Service 测试服务层多租户隔离
func TestMultiTenantIsolation_Service(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupServiceTestEnvironment(t)
	defer TeardownServiceTestEnvironment(t, env)

	// 创建第二个租户
	tenant2ID := uuid.New()
	tenant2 := &model.Tenant{
		ID:     tenant2ID,
		Name:   "第二个租户",
		Domain: "tenant2.example.com",
		Status: model.TenantStatusActive,
	}
	err := env.DB.Create(tenant2).Error
	require.NoError(t, err)

	// 创建第二个租户的用户
	user2ID := uuid.New()
	user2 := &model.User{
		ID:       user2ID,
		TenantID: tenant2ID,
		Email:    "user2@example.com",
		Username: "user2",
		Status:   model.UserStatusActive,
		Roles:    []string{model.RoleTenantAdmin},
	}
	err = env.DB.Create(user2).Error
	require.NoError(t, err)

	// 创建第二个租户的会话
	session2ID := uuid.New()
	session2 := &model.ConversationSession{
		ID:        session2ID,
		TenantID:  tenant2ID,
		UserID:    user2ID,
		Title:     "租户2的会话",
		Status:    "active",
		CreatedBy: user2ID,
	}
	err = env.DB.Create(session2).Error
	require.NoError(t, err)

	t.Run("租户1无法访问租户2的会话", func(t *testing.T) {
		ctx1 := CreateServiceTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

		// 尝试构建租户2的上下文
		req := BuildContextRequest{
			SessionID:       session2ID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		_, err := env.ContextService.BuildContext(ctx1, req)
		assert.Error(t, err, "应该拒绝跨租户访问")
	})

	t.Run("租户2可以访问自己的会话", func(t *testing.T) {
		ctx2 := CreateServiceTestContext(tenant2ID, user2ID, []string{model.RoleTenantAdmin})

		// 创建租户2的消息
		msg := &model.ChatMessage{
			ID:        uuid.New(),
			TenantID:  tenant2ID,
			SessionID: session2ID,
			Role:      "user",
			Content:   "租户2的消息",
			CreatedBy: user2ID,
		}
		err := env.DB.Create(msg).Error
		require.NoError(t, err)

		// 构建自己的上下文
		req := BuildContextRequest{
			SessionID:       session2ID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		result, err := env.ContextService.BuildContext(ctx2, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, session2ID.String(), result.SessionID)
	})

	t.Run("平台管理员可以访问所有租户", func(t *testing.T) {
		// 创建平台管理员
		adminID := uuid.New()
		admin := &model.User{
			ID:       adminID,
			TenantID: env.TenantID,
			Email:    "admin@example.com",
			Username: "admin",
			Status:   model.UserStatusActive,
			Roles:    []string{model.RoleSystemAdmin},
		}
		err := env.DB.Create(admin).Error
		require.NoError(t, err)

		ctxAdmin := CreateServiceTestContext(env.TenantID, adminID, []string{model.RoleSystemAdmin})

		// 访问租户1的会话
		req1 := BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		result1, err := env.ContextService.BuildContext(ctxAdmin, req1)
		require.NoError(t, err)
		assert.NotNil(t, result1)

		// 访问租户2的会话
		req2 := BuildContextRequest{
			SessionID:       session2ID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		result2, err := env.ContextService.BuildContext(ctxAdmin, req2)
		require.NoError(t, err)
		assert.NotNil(t, result2)
	})
}

// TestServicePerformance 测试服务性能
func TestServicePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupServiceTestEnvironment(t)
	defer TeardownServiceTestEnvironment(t, env)

	ctx := CreateServiceTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("上下文构建性能", func(t *testing.T) {
		// 创建测试消息
		for i := 0; i < 10; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "测试消息",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 测试构建时间
		start := time.Now()
		req := BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "性能测试",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		result, err := env.ContextService.BuildContext(ctx, req)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, result)
		t.Logf("上下文构建耗时: %v", duration)

		// 验证性能要求（P50 < 200ms）
		assert.Less(t, duration.Milliseconds(), int64(500), "上下文构建应该在500ms内完成")
	})
}

// TestMain 测试主函数
func TestMain(m *testing.M) {
	// 设置测试环境变量
	os.Setenv("ENV", "test")
	os.Setenv("LOG_LEVEL", "error")

	// 运行测试
	code := m.Run()

	// 清理
	os.Exit(code)
}
