package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/middleware"
	"genkit-ai-service/internal/api/routes"
	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/loader"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/internal/service/cleanup"
	"genkit-ai-service/internal/service/health"
	"genkit-ai-service/internal/service/session"
	"genkit-ai-service/internal/storage"

	_ "genkit-ai-service/docs" // Swagger 文档
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Genkit AI Service API
// @version 1.0.0
// @description AI 模型提供商管理服务 API 文档
// @description 提供模型提供商、模型信息和参数规则的查询接口

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 "Bearer {token}" 格式的 JWT 令牌进行身份认证

// @tag.name Authentication
// @tag.description 用户认证相关接口（注册、登录、Token 刷新、注销等）

// @tag.name Tenant Management
// @tag.description 租户管理接口（需要管理员权限）

// @tag.name User Management
// @tag.description 用户管理接口（需要租户管理员权限）

// @tag.name Tenant User Management
// @tag.description 租户用户管理接口（需要租户管理员或平台管理员权限）

// @tag.name Platform Management
// @tag.description 平台管理接口（需要平台管理员权限）

// @tag.name Audit
// @tag.description 审计日志接口

// @tag.name Monitoring
// @tag.description 监控接口

// @tag.name providers
// @tag.description 模型提供商管理接口

// @tag.name chat
// @tag.description AI 对话接口

// @tag.name sessions
// @tag.description 会话管理接口

// @tag.name messages
// @tag.description 消息管理接口

// @tag.name health
// @tag.description 健康检查接口

const (
	// Version 服务版本
	Version = "1.0.0"
	
	// ShutdownTimeout 优雅关闭超时时间
	ShutdownTimeout = 30 * time.Second
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	logLevel := logger.ParseLevel(cfg.Log.Level)
	logFormat := logger.JSONFormat
	if cfg.Log.Format == "text" {
		logFormat = logger.TextFormat
	}
	log := logger.New(logLevel, logFormat, os.Stdout)
	log.Info("服务启动中...", logger.Fields{
		"version": Version,
		"port":    cfg.Server.Port,
	})

	// 3. 初始化数据库连接（可选）
	db, err := initDatabase(cfg, log)
	if err != nil {
		// 数据库初始化失败（包括迁移失败）时，记录错误并退出
		// 因为认证、会话管理等核心功能依赖数据库
		log.Error("初始化数据库失败，应用无法启动", logger.Fields{
			"error": err,
			"解决方案": []string{
				"1. 检查数据库连接配置是否正确",
				"2. 确保数据库服务正在运行",
				"3. 验证数据库用户权限是否足够",
				"4. 查看详细错误信息定位问题",
			},
		})
		os.Exit(1)
	}
	
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("关闭数据库连接失败", logger.Fields{"error": err})
		}
	}()

	// 4. 初始化 Genkit 客户端（可选）
	genkitClient, err := initGenkit(cfg, log)
	if err != nil {
		log.Warn("初始化 Genkit 客户端失败，AI服务将不可用", logger.Fields{"error": err})
		genkitClient = nil
	}

	// 5. 初始化模型提供商数据
	providerService, err := initProviderService(cfg, log)
	if err != nil {
		log.Error("初始化模型提供商服务失败", logger.Fields{"error": err})
		os.Exit(1)
	}

	// 6. 初始化服务
	var aiService ai.AIService
	var healthService health.Service
	
	// AI 服务只需要 Genkit 客户端
	if genkitClient != nil {
		aiService = initAIService(genkitClient, cfg, log)
		log.Info("AI服务已启用", nil)
	} else {
		log.Warn("AI服务未启用（Genkit 客户端初始化失败）", nil)
	}
	
	// 健康检查服务需要 Genkit 客户端和数据库
	if genkitClient != nil && db != nil {
		healthService = health.NewService(genkitClient, db, Version)
		log.Info("健康检查服务已启用", nil)
	} else {
		log.Warn("健康检查服务未启用（缺少数据库连接）", nil)
	}

	// 7. 创建基础 ServeMux 并注册所有路由
	serveMux := http.NewServeMux()
	
	// 8. 注册模型提供商API路由
	providerHandler := handler.NewProviderHandler(providerService, log)
	routes.RegisterProviderRoutes(serveMux, providerHandler)
	log.Info("模型提供商API路由已注册", nil)

	// 8.1 注册认证路由（如果数据库可用）
	var cleanupSvc cleanup.CleanupService
	if db != nil {
		authHandler, tenantHandler, userHandler, auditHandler, tenantMW, jwtAuthMW, rbacMW := initAuthHandlers(db, cfg, log)
		routes.RegisterAuthRoutes(serveMux, authHandler, tenantHandler, userHandler, auditHandler, tenantMW, jwtAuthMW, rbacMW)
		log.Info("认证路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/auth/register",
				"/api/v1/auth/login",
				"/api/v1/auth/refresh",
				"/api/v1/auth/logout",
				"/api/v1/auth/change-password",
				"/api/v1/auth/me",
				"/api/v1/tenants",
				"/api/v1/users",
				"/api/v1/audit/auth",
			},
		})
		
		// 注册平台管理路由
		platformHandler := initPlatformHandler(db, log)
		routes.RegisterPlatformRoutes(serveMux, platformHandler, jwtAuthMW, rbacMW)
		log.Info("平台管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/platform/tenants",
				"/api/v1/platform/tenants/{id}/status",
				"/api/v1/platform/tenants/{id}",
			},
		})
		
		// 启动数据库清理服务
		cleanupSvc = initCleanupService(db, cfg, log)
		ctx := context.Background()
		cleanupSvc.Start(ctx)
		log.Info("数据库清理服务已启动", nil)
	} else {
		log.Warn("认证路由未注册（数据库不可用）", nil)
	}

	// 8.2 注册会话管理路由（如果数据库可用）
	if db != nil && aiService != nil {
		sessionHandler, messageHandler := initSessionHandlers(db, aiService, cfg, log)
		routes.RegisterSessionRoutes(serveMux, sessionHandler, messageHandler)
		log.Info("会话管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/chat/sessions",
				"/api/v1/chat/sessions/{id}",
				"/api/v1/chat/sessions/{id}/messages",
				"/api/v1/chat/messages/{id}",
			},
		})
	} else {
		log.Warn("会话管理路由未注册（数据库或AI服务不可用）", nil)
	}

	// 9. 注册 AI 服务路由（如果可用）
	if aiService != nil {
		chatHandler := handler.NewChatHandler(aiService, log)
		chatStreamHandler := handler.NewChatStreamHandler(aiService, log)
		abortHandler := handler.NewAbortHandler(aiService, log)
		
		// 注意：必须先注册更具体的路径，再注册通用路径
		serveMux.HandleFunc("POST /api/v1/chat/stream", chatStreamHandler.HandleChatStream)
		serveMux.HandleFunc("POST /api/v1/chat/abort", abortHandler.HandleAbort)
		serveMux.HandleFunc("POST /api/v1/chat", chatHandler.HandleChat)
		
		log.Info("AI对话路由已注册", logger.Fields{
			"routes": []string{"/api/v1/chat", "/api/v1/chat/stream", "/api/v1/chat/abort"},
		})
	} else {
		log.Warn("AI对话路由未注册（AI服务不可用）", nil)
	}
	
	// 10. 注册健康检查路由（如果可用）
	if healthService != nil {
		healthHandler := handler.NewHealthHandler(healthService, log)
		serveMux.HandleFunc("GET /api/v1/health", healthHandler.Handle)
		log.Info("健康检查路由已注册", logger.Fields{
			"routes": []string{"/api/v1/health"},
		})
	} else {
		log.Warn("健康检查路由未注册（健康检查服务不可用）", nil)
	}

	// 11. 注册 Swagger UI 路由
	serveMux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	
	// 11.1 提供 swagger.yaml 静态文件访问
	serveMux.HandleFunc("GET /swagger/doc.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		http.ServeFile(w, r, "docs/swagger.yaml")
	})
	
	log.Info("Swagger UI 已启用", logger.Fields{
		"swagger_ui": fmt.Sprintf("http://%s:%s/swagger/index.html", cfg.Server.Host, cfg.Server.Port),
		"swagger_json": fmt.Sprintf("http://%s:%s/swagger/doc.json", cfg.Server.Host, cfg.Server.Port),
		"swagger_yaml": fmt.Sprintf("http://%s:%s/swagger/doc.yaml", cfg.Server.Host, cfg.Server.Port),
	})
	
	// 12. 应用中间件（按顺序：Recovery -> Logger -> CORS）
	var mux http.Handler = serveMux
	corsConfig := middleware.DefaultCORS()
	mux = corsConfig.Handler(mux)
	mux = middleware.Logger(mux)
	mux = middleware.Recovery(mux)

	// 13. 创建 HTTP 服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 14. 启动服务器（在 goroutine 中）
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务器启动", logger.Fields{
			"address": server.Addr,
		})
		serverErrors <- server.ListenAndServe()
	}()

	// 15. 监听系统信号以实现优雅关闭
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// 16. 等待关闭信号或服务器错误
	select {
	case err := <-serverErrors:
		log.Error("服务器启动失败", logger.Fields{"error": err})
		os.Exit(1)

	case sig := <-shutdown:
		log.Info("收到关闭信号，开始优雅关闭", logger.Fields{
			"signal": sig.String(),
		})

		// 停止清理服务
		if cleanupSvc != nil {
			cleanupSvc.Stop()
			log.Info("数据库清理服务已停止", nil)
		}

		// 创建关闭超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		// 优雅关闭 HTTP 服务器
		if err := server.Shutdown(ctx); err != nil {
			log.Error("服务器关闭失败，强制关闭", logger.Fields{"error": err})
			if err := server.Close(); err != nil {
				log.Error("强制关闭服务器失败", logger.Fields{"error": err})
			}
		}

		log.Info("服务已成功关闭", logger.Fields{
			"version": Version,
		})
	}
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.Config, log logger.Logger) (database.Database, error) {
	// 记录连接信息（如果使用 DATABASE_URL 则不显示详细信息）
	if cfg.Database.URL != "" {
		log.Info("初始化数据库连接...", logger.Fields{
			"source": "DATABASE_URL",
		})
	} else {
		log.Info("初始化数据库连接...", logger.Fields{
			"host": cfg.Database.Host,
			"port": cfg.Database.Port,
			"name": cfg.Database.DBName,
		})
	}

	postgresConfig := &database.PostgresConfig{
		DSN:             cfg.Database.GetDSN(), // 使用新的 GetDSN 方法
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		LogLevel:        cfg.Database.LogLevel,
	}

	db := database.NewPostgresDatabase(postgresConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 验证连接
	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("数据库连接验证失败: %w", err)
	}

	log.Info("数据库连接成功", logger.Fields{
		"host": cfg.Database.Host,
	})

	// 执行数据库迁移
	if err := runDatabaseMigrations(db, cfg, log); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// runDatabaseMigrations 执行数据库迁移
func runDatabaseMigrations(db database.Database, cfg *config.Config, log logger.Logger) error {
	log.Info("开始执行数据库迁移...", nil)

	// 获取 GORM 数据库实例
	gormDB := db.GetDB()
	if gormDB == nil {
		return fmt.Errorf("无法获取数据库实例")
	}

	// 执行初始迁移
	// 初始迁移包含所有表的创建逻辑（认证表和会话管理表）
	if err := database.RunInitialMigration(gormDB); err != nil {
		log.Error("初始迁移失败", logger.Fields{
			"error": err,
			"迁移名称": "initial_migration",
		})
		
		// 提供详细的错误信息和解决建议
		return fmt.Errorf("数据库迁移失败: %w\n\n可能的原因和解决方案:\n"+
			"1. 数据库权限不足 - 确保数据库用户具有 CREATE TABLE、CREATE INDEX 等权限\n"+
			"2. UUID 扩展未启用 - 确保 PostgreSQL 支持 gen_random_uuid() 函数\n"+
			"3. 表已存在但结构不匹配 - 考虑使用 reset_db.go 脚本重置数据库\n"+
			"4. 数据库连接中断 - 检查网络连接和数据库服务状态\n"+
			"5. 磁盘空间不足 - 检查数据库服务器磁盘空间\n\n"+
			"详细错误信息: %v", err)
	}

	log.Info("数据库迁移完成", logger.Fields{
		"migration": "initial_migration",
		"tables": []string{
			"tenants",
			"users",
			"refresh_tokens",
			"email_verification_tokens",
			"auth_audit",
			"chat_sessions",
			"chat_messages",
			"chat_summaries",
		},
	})

	// 执行系统初始化（创建平台租户和管理员）
	if err := runSystemBootstrap(db, cfg, log); err != nil {
		log.Error("系统初始化失败", logger.Fields{
			"error": err,
		})
		return err
	}

	return nil
}

// initGenkit 初始化 Genkit 客户端
func initGenkit(cfg *config.Config, log logger.Logger) (genkit.Client, error) {
	log.Info("初始化 Genkit 客户端...", logger.Fields{
		"model": cfg.Genkit.Model,
	})

	client := genkit.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	genkitConfig := &genkit.Config{
		APIKey:             cfg.Genkit.APIKey,
		Model:              cfg.Genkit.Model,
		DefaultTemperature: cfg.Genkit.DefaultTemperature,
		DefaultMaxTokens:   cfg.Genkit.DefaultMaxTokens,
	}

	if err := client.Initialize(ctx, genkitConfig); err != nil {
		return nil, fmt.Errorf("初始化 Genkit 客户端失败: %w", err)
	}

	// 初始化并设置 Genkit 模型
	// 注意：这里需要根据实际使用的模型提供者来初始化模型
	// 例如使用 Google AI 的 Gemini 模型
	if err := client.InitializeModel(ctx); err != nil {
		return nil, fmt.Errorf("初始化 Genkit 模型失败: %w", err)
	}

	log.Info("Genkit 客户端初始化成功", logger.Fields{
		"model": cfg.Genkit.Model,
	})

	return client, nil
}

// initAIService 初始化 AI 服务
func initAIService(genkitClient genkit.Client, cfg *config.Config, log logger.Logger) ai.AIService {
	log.Info("初始化 AI 服务...", logger.Fields{
		"sessionTimeout":        cfg.Session.Timeout,
		"sessionCleanupInterval": cfg.Session.CleanupInterval,
	})

	// 创建上下文管理器
	contextManager := ai.NewContextManager(
		cfg.Session.Timeout,
		cfg.Session.CleanupInterval,
	)
	
	// 启动上下文管理器的自动清理
	contextManager.Start()

	// 创建 AI 服务
	aiService := ai.NewGenkitService(genkitClient, contextManager, log)

	log.Info("AI 服务初始化成功", nil)

	return aiService
}

// initProviderService 初始化模型提供商服务
func initProviderService(cfg *config.Config, log logger.Logger) (service.ProviderService, error) {
	log.Info("初始化模型提供商服务...", nil)

	// 1. 创建内存存储实例
	store := storage.NewMemoryStore()

	// 2. 创建数据加载器
	modelLoader := loader.NewModelLoader(store, log)

	// 3. 执行数据加载
	// 使用配置中的模型目录路径（已包含默认值）
	if err := modelLoader.LoadAll(cfg.Models.Dir); err != nil {
		return nil, fmt.Errorf("加载模型数据失败: %w", err)
	}

	// 4. 创建服务层实例
	providerService := service.NewProviderService(store)

	log.Info("模型提供商服务初始化成功", nil)

	return providerService, nil
}

// initSessionHandlers 初始化会话管理相关的处理器
func initSessionHandlers(db database.Database, aiService ai.AIService, cfg *config.Config, log logger.Logger) (*handler.SessionHandler, *handler.MessageHandler) {
	log.Info("初始化会话管理服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository 层实例
	sessionRepo := repository.NewSessionRepository(gormDB)
	messageRepo := repository.NewMessageRepository(gormDB)
	summaryRepo := repository.NewSummaryRepository(gormDB)

	// 3. 创建 Service 层实例
	// 3.1 创建 SessionService
	sessionService := session.NewSessionService(sessionRepo, messageRepo)
	
	// 3.2 创建 SummaryService
	summaryService := session.NewSummaryService(summaryRepo, messageRepo, sessionRepo, aiService, cfg, log)
	
	// 3.3 创建 MessageService
	messageService := session.NewMessageService(gormDB, sessionRepo, messageRepo, aiService, log)
	
	// 注意：SummaryService 已初始化但当前未直接使用，
	// 它可以在未来的功能中被 MessageService 或其他服务调用
	_ = summaryService

	// 4. 创建 Handler 层实例
	sessionHandler := handler.NewSessionHandler(sessionService, log)
	messageHandler := handler.NewMessageHandler(messageService, log)

	log.Info("会话管理服务初始化成功", logger.Fields{
		"repositories": []string{"SessionRepository", "MessageRepository", "SummaryRepository"},
		"services":     []string{"SessionService", "MessageService", "SummaryService"},
		"handlers":     []string{"SessionHandler", "MessageHandler"},
	})

	return sessionHandler, messageHandler
}

// initAuthHandlers 初始化认证相关的处理器和中间件
func initAuthHandlers(db database.Database, cfg *config.Config, log logger.Logger) (
	*handler.AuthHandler,
	*handler.TenantHandler,
	*handler.UserHandler,
	*handler.AuditHandler,
	func(http.Handler) http.Handler,
	func(http.Handler) http.Handler,
	func(...string) func(http.Handler) http.Handler,
) {
	log.Info("初始化认证服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository 层实例
	tenantRepo := repository.NewTenantRepository(gormDB)
	userRepo := repository.NewUserRepository(gormDB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(gormDB)
	auditRepo := repository.NewAuditRepository(gormDB)
	emailVerificationRepo := repository.NewEmailVerificationRepository(gormDB)

	// 3. 创建 Service 层实例
	// 3.1 初始化 Redis 客户端（如果启用）
	var redisClient *database.RedisClient
	if cfg.Redis.Enabled {
		var err error
		redisClient, err = database.NewRedisClient(cfg.Redis, log)
		if err != nil {
			log.Warn("Redis 连接失败，Token 黑名单功能将被禁用", logger.Fields{"error": err})
			redisClient = nil
		}
	} else {
		log.Info("Redis 已禁用，Token 黑名单功能将不可用", nil)
	}

	// 3.2 创建 TokenBlacklistService
	var blacklistService auth.TokenBlacklistService
	if redisClient != nil && cfg.Auth.EnableTokenBlacklist {
		blacklistService = auth.NewTokenBlacklistService(redisClient, log)
		log.Info("Token 黑名单服务已启用", nil)
	} else {
		log.Info("Token 黑名单服务已禁用", nil)
	}

	// 3.3 创建 TokenService
	tokenService := auth.NewTokenService(&cfg.Auth, refreshTokenRepo)

	// 3.4 创建 TenantService
	tenantService := auth.NewTenantService(tenantRepo, userRepo, auditRepo)

	// 3.5 创建 UserService
	userService := auth.NewUserService(userRepo, tenantRepo, refreshTokenRepo, auditRepo)

	// 3.6 创建 EmailService
	// 使用控制台邮件发送器（开发环境）
	// 生产环境应该使用 SMTP 邮件发送器
	emailSender := auth.NewConsoleEmailSender()
	emailService := auth.NewEmailService(
		emailVerificationRepo,
		userRepo,
		emailSender,
		24*time.Hour, // 验证令牌有效期24小时
	)

	// 3.7 创建 AuthService
	authService := auth.NewAuthService(
		userRepo,
		tenantRepo,
		tokenService, 
		auditRepo, 
		refreshTokenRepo,
		blacklistService, // 传入黑名单服务
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.MaxLoginAttempts,
		cfg.Auth.LoginAttemptWindow,
	)

	// 4. 创建 Handler 层实例
	authHandler := handler.NewAuthHandler(authService, emailService, log)
	tenantHandler := handler.NewTenantHandler(tenantService, log)
	userHandler := handler.NewUserHandler(userService, log)
	auditHandler := handler.NewAuditHandler(auditRepo, log)

	// 5. 创建中间件
	// 创建租户识别中间件配置
	tenantConfig := middleware.TenantIdentifierConfig{
		Strategy:   cfg.Auth.TenantIdentifyStrategy,
		TenantRepo: tenantRepo,
		BaseDomain: "",
	}
	tenantMiddleware := middleware.TenantIdentifier(tenantConfig)
	
	// 创建 JWT 认证中间件（传入黑名单服务）
	jwtAuthMiddleware := middleware.JWTAuth(tokenService, blacklistService)
	
	// 创建 RBAC 授权中间件工厂函数
	rbacMiddleware := func(roles ...string) func(http.Handler) http.Handler {
		// 根据角色返回相应的中间件
		if len(roles) == 1 {
			switch roles[0] {
			case "system_admin":
				return middleware.RequireSystemAdmin()
			case "admin", "tenant_admin":
				return middleware.RequireTenantAdmin()
			}
		}
		// 默认返回租户管理员中间件
		return middleware.RequireTenantAdmin()
	}

	log.Info("认证服务初始化成功", logger.Fields{
		"repositories": []string{"TenantRepository", "UserRepository", "RefreshTokenRepository", "AuditRepository"},
		"services":     []string{"TokenService", "TenantService", "UserService", "AuthService"},
		"handlers":     []string{"AuthHandler", "TenantHandler", "UserHandler", "AuditHandler"},
		"middlewares":  []string{"TenantIdentifier", "JWTAuth", "RBACAuthorizer"},
	})

	return authHandler, tenantHandler, userHandler, auditHandler, tenantMiddleware, jwtAuthMiddleware, rbacMiddleware
}

// initCleanupService 初始化数据库清理服务
func initCleanupService(db database.Database, cfg *config.Config, log logger.Logger) cleanup.CleanupService {
	log.Info("初始化数据库清理服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository
	refreshTokenRepo := repository.NewRefreshTokenRepository(gormDB)
	emailVerificationRepo := repository.NewEmailVerificationRepository(gormDB)

	// 3. 创建清理服务配置
	cleanupConfig := cleanup.CleanupConfig{
		TokenCleanupInterval: cfg.Auth.TokenCleanupInterval,
	}

	// 4. 创建清理服务实例
	cleanupService := cleanup.NewCleanupService(refreshTokenRepo, emailVerificationRepo, log, cleanupConfig)

	log.Info("数据库清理服务初始化成功", logger.Fields{
		"interval": cleanupConfig.TokenCleanupInterval.String(),
	})

	return cleanupService
}

// runSystemBootstrap 执行系统初始化（创建平台租户和管理员）
func runSystemBootstrap(db database.Database, cfg *config.Config, log logger.Logger) error {
	log.Info("开始系统初始化检查...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()
	if gormDB == nil {
		return fmt.Errorf("无法获取数据库实例")
	}

	// 2. 创建 Repository 层实例
	tenantRepo := repository.NewTenantRepository(gormDB)
	userRepo := repository.NewUserRepository(gormDB)

	// 3. 创建 BootstrapService 实例
	bootstrapService := auth.NewBootstrapService(tenantRepo, userRepo, &cfg.Bootstrap)

	// 4. 执行初始化
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := bootstrapService.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("系统初始化失败: %w", err)
	}

	// 5. 记录初始化结果
	if result.Initialized {
		log.Info("系统初始化成功", logger.Fields{
			"平台租户ID":   result.TenantID,
			"管理员邮箱":    result.AdminEmail,
			"管理员初始密码": result.AdminPassword,
			"重要提示":     "请妥善保管管理员初始密码，建议首次登录后立即修改",
		})
	} else {
		log.Info("系统已初始化，跳过初始化流程", nil)
	}

	return nil
}

// initPlatformHandler 初始化平台管理处理器
func initPlatformHandler(db database.Database, log logger.Logger) *handler.PlatformHandler {
	log.Info("初始化平台管理服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository 层实例
	tenantRepo := repository.NewTenantRepository(gormDB)
	userRepo := repository.NewUserRepository(gormDB)
	auditRepo := repository.NewAuditRepository(gormDB)

	// 3. 创建 TenantService 实例
	tenantService := auth.NewTenantService(tenantRepo, userRepo, auditRepo)

	// 4. 创建 PlatformHandler 实例
	platformHandler := handler.NewPlatformHandler(tenantService, log)

	log.Info("平台管理服务初始化成功", logger.Fields{
		"repositories": []string{"TenantRepository", "UserRepository"},
		"services":     []string{"TenantService"},
		"handlers":     []string{"PlatformHandler"},
	})

	return platformHandler
}
