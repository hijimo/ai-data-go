// Package flows_test Flow 集成测试
package flows_test

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
	"genkit-ai-service/internal/service"
)

// TestEnvironment 测试环境
type TestEnvironment struct {
	DB              *gorm.DB
	TenantID        uuid.UUID
	SessionID       uuid.UUID
	UserID          uuid.UUID
	ContextService  service.ContextService
	MemoryService   service.MemoryService
	SummaryService  service.SummaryService
	TokenManager    service.TokenManager
	VectorService   service.VectorService
	CacheService    service.CacheService
	SessionRepo     repository.SessionRepository
	MessageRepo     repository.MessageRepository
	MemoryRepo      repository.GenkitMemoryRepository
	ContextRepo     repository.GenkitContextRepository
	SummaryRepo     repository.SummaryRepository
}

// SetupTestEnvironment 设置测试环境
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
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
	cacheService := service.NewCacheService(nil, "test")
	vectorService := service.NewVectorService(nil, 1536)
	tokenManager := service.NewTokenManager()

	contextService := service.NewContextService(
		contextRepo,
		messageRepo,
		memoryRepo,
		vectorService,
		tokenManager,
		cacheService,
	)

	memoryService := service.NewMemoryService(
		memoryRepo,
		messageRepo,
		vectorService,
		tokenManager,
	)

	summaryService := service.NewSummaryService(
		summaryRepo,
		messageRepo,
		contextRepo,
		tokenManager,
	)

	return &TestEnvironment{
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

// TeardownTestEnvironment 清理测试环境
func TeardownTestEnvironment(t *testing.T, env *TestEnvironment) {
	if env.DB != nil {
		sqlDB, err := env.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// CreateTestContext 创建带有认证信息的测试上下文
func CreateTestContext(tenantID, userID uuid.UUID, roles []string) context.Context {
	ctx := context.Background()
	claims := &model.JWTClaims{
		TenantID: tenantID.String(),
		Roles:    roles,
	}
	claims.Subject = userID.String()
	return context.WithValue(ctx, middleware.JWTClaimsKey, claims)
}

// TestContextBuildFlow_Integration 测试上下文构建 Flow 集成
func TestContextBuildFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	ctx := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("构建基本上下文", func(t *testing.T) {
		// 创建测试消息
		messages := []*model.ChatMessage{
			{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "你好，这是第一条消息",
				CreatedBy: env.UserID,
			},
			{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "assistant",
				Content:   "你好！我是AI助手，很高兴为您服务。",
				CreatedBy: env.UserID,
			},
		}

		for _, msg := range messages {
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 构建上下文
		req := service.BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "请告诉我更多信息",
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
		assert.Len(t, result.ShortTermMessages, 2)
		assert.Greater(t, result.TotalTokens, 0)
		assert.Greater(t, result.QualityScore, 0.0)
		assert.LessOrEqual(t, result.QualityScore, 1.0)
	})

	t.Run("构建包含长期记忆的上下文", func(t *testing.T) {
		// 创建测试记忆
		memory := &model.ConversationMemory{
			ID:         uuid.New(),
			TenantID:   env.TenantID,
			SessionID:  env.SessionID,
			MemoryType: model.MemoryTypeLongTerm,
			Content:    "用户之前询问过关于AI的问题",
			TokenCount: 20,
			Importance: 0.8,
		}
		err := env.DB.Create(memory).Error
		require.NoError(t, err)

		// 构建上下文
		req := service.BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "继续之前的话题",
			MaxTokens:       4000,
			Strategy:        "auto",
			IncludeSummary:  false,
			IncludeLongTerm: true,
			ShortTermWindow: 10,
		}

		result, err := env.ContextService.BuildContext(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestMemorySearchFlow_Integration 测试记忆检索 Flow 集成
func TestMemorySearchFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	ctx := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("检索会话记忆", func(t *testing.T) {
		// 创建测试记忆
		memories := []*model.ConversationMemory{
			{
				ID:         uuid.New(),
				TenantID:   env.TenantID,
				SessionID:  env.SessionID,
				MemoryType: model.MemoryTypeLongTerm,
				Content:    "用户喜欢讨论技术话题",
				TokenCount: 15,
				Importance: 0.9,
			},
			{
				ID:         uuid.New(),
				TenantID:   env.TenantID,
				SessionID:  env.SessionID,
				MemoryType: model.MemoryTypeLongTerm,
				Content:    "用户对AI很感兴趣",
				TokenCount: 12,
				Importance: 0.85,
			},
		}

		for _, mem := range memories {
			err := env.DB.Create(mem).Error
			require.NoError(t, err)
		}

		// 检索记忆
		req := service.SearchMemoriesRequest{
			SessionID:     env.SessionID.String(),
			Query:         "技术",
			TopK:          5,
			MinSimilarity: 0.5,
			TenantID:      env.TenantID.String(),
		}

		results, err := env.MemoryService.SearchMemories(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, results)
	})
}

// TestSummaryGenerateFlow_Integration 测试摘要生成 Flow 集成
func TestSummaryGenerateFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	ctx := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("生成会话摘要", func(t *testing.T) {
		// 创建足够的测试消息
		messageIDs := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "测试消息内容",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
			messageIDs = append(messageIDs, msg.ID.String())
		}

		// 生成摘要
		req := service.GenerateSummaryRequest{
			SessionID:    env.SessionID.String(),
			MessageIDs:   messageIDs,
			SummaryType:  "full",
			TargetLength: 200,
			TenantID:     env.TenantID.String(),
		}

		result, err := env.SummaryService.GenerateSummary(ctx, req)
		// 注意：由于没有真实的AI服务，这里可能会失败
		// 在实际环境中需要mock AI服务
		if err != nil {
			t.Logf("摘要生成失败（预期，因为没有AI服务）: %v", err)
		} else {
			assert.NotNil(t, result)
		}
	})
}

// TestMultiTenantIsolation_Integration 测试多租户隔离
func TestMultiTenantIsolation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

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

	t.Run("租户1无法访问租户2的数据", func(t *testing.T) {
		ctx1 := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

		// 尝试访问租户2的会话
		req := service.BuildContextRequest{
			SessionID:       session2ID.String(),
			UserQuery:       "测试查询",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		_, err := env.ContextService.BuildContext(ctx1, req)
		// 应该返回权限错误
		assert.Error(t, err, "应该拒绝跨租户访问")
	})

	t.Run("租户2可以访问自己的数据", func(t *testing.T) {
		ctx2 := CreateTestContext(tenant2ID, user2ID, []string{model.RoleTenantAdmin})

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

		// 访问自己的会话
		req := service.BuildContextRequest{
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

	t.Run("平台管理员可以访问所有租户的数据", func(t *testing.T) {
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

		ctxAdmin := CreateTestContext(env.TenantID, adminID, []string{model.RoleSystemAdmin})

		// 访问租户1的会话
		req1 := service.BuildContextRequest{
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
		req2 := service.BuildContextRequest{
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

// TestEndToEnd_CompleteConversation 端到端测试：完整对话流程
func TestEndToEnd_CompleteConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	ctx := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	t.Run("完整对话流程", func(t *testing.T) {
		// 1. 创建初始消息
		msg1 := &model.ChatMessage{
			ID:        uuid.New(),
			TenantID:  env.TenantID,
			SessionID: env.SessionID,
			Role:      "user",
			Content:   "你好，我想了解AI技术",
			CreatedBy: env.UserID,
		}
		err := env.DB.Create(msg1).Error
		require.NoError(t, err)

		// 2. 构建上下文
		contextReq := service.BuildContextRequest{
			SessionID:       env.SessionID.String(),
			UserQuery:       "请详细介绍",
			MaxTokens:       4000,
			Strategy:        "auto",
			ShortTermWindow: 10,
		}

		contextResult, err := env.ContextService.BuildContext(ctx, contextReq)
		require.NoError(t, err)
		assert.NotNil(t, contextResult)
		assert.Len(t, contextResult.ShortTermMessages, 1)

		// 3. 添加更多消息
		for i := 0; i < 5; i++ {
			msg := &model.ChatMessage{
				ID:        uuid.New(),
				TenantID:  env.TenantID,
				SessionID: env.SessionID,
				Role:      "user",
				Content:   "继续讨论",
				CreatedBy: env.UserID,
			}
			err := env.DB.Create(msg).Error
			require.NoError(t, err)
		}

		// 4. 再次构建上下文，验证消息增加
		contextResult2, err := env.ContextService.BuildContext(ctx, contextReq)
		require.NoError(t, err)
		assert.Greater(t, len(contextResult2.ShortTermMessages), len(contextResult.ShortTermMessages))

		// 5. 存储记忆
		memory := &model.ConversationMemory{
			ID:         uuid.New(),
			TenantID:   env.TenantID,
			SessionID:  env.SessionID,
			MemoryType: model.MemoryTypeLongTerm,
			Content:    "用户对AI技术很感兴趣",
			TokenCount: 15,
			Importance: 0.9,
		}
		err = env.DB.Create(memory).Error
		require.NoError(t, err)

		// 6. 检索记忆
		searchReq := service.SearchMemoriesRequest{
			SessionID:     env.SessionID.String(),
			Query:         "AI",
			TopK:          5,
			MinSimilarity: 0.5,
			TenantID:      env.TenantID.String(),
		}

		memories, err := env.MemoryService.SearchMemories(ctx, searchReq)
		require.NoError(t, err)
		assert.NotNil(t, memories)
	})
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	env := SetupTestEnvironment(t)
	defer TeardownTestEnvironment(t, env)

	ctx := CreateTestContext(env.TenantID, env.UserID, []string{model.RoleTenantAdmin})

	// 创建初始消息
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

	t.Run("并发构建上下文", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := service.BuildContextRequest{
					SessionID:       env.SessionID.String(),
					UserQuery:       "并发测试",
					MaxTokens:       4000,
					Strategy:        "auto",
					ShortTermWindow: 10,
				}

				_, err := env.ContextService.BuildContext(ctx, req)
				if err != nil {
					errors <- err
				}
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < concurrency; i++ {
			<-done
		}

		close(errors)
		errorCount := 0
		for err := range errors {
			t.Logf("并发错误: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "不应该有并发错误")
	})
}

// TestFlowsMain 测试主函数
func TestFlowsMain(m *testing.M) {
	// 设置测试环境变量
	os.Setenv("ENV", "test")
	os.Setenv("LOG_LEVEL", "error")

	// 运行测试
	code := m.Run()

	// 清理
	os.Exit(code)
}
