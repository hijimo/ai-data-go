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
	"genkit-ai-service/internal/genkit/flows"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/auth"
	"genkit-ai-service/internal/service/cleanup"
	"genkit-ai-service/internal/service/health"
	"genkit-ai-service/internal/service/session"
	"genkit-ai-service/internal/storage"
	"genkit-ai-service/pkg/lexiang"

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

// @tag.name Lexiang Knowledge Base
// @tag.description 乐享知识库管理接口

// @tag.name Lexiang Entries
// @tag.description 乐享知识节点管理接口

// @tag.name Lexiang Upload
// @tag.description 乐享文件上传接口

// @tag.name Lexiang Download
// @tag.description 乐享附件下载接口

// @tag.name Lexiang Feedback
// @tag.description 乐享知识反馈接口

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

	// 初始化带文件持久化的日志记录器
	// 日志文件按天存储在 logs 目录下，格式为 app-YYYY-MM-DD.log
	// enableConsole 设置为 true 表示同时输出到控制台和文件
	var log logger.Logger
	if cfg.Log.EnableFile {
		logDir := cfg.Log.LogDir
		if logDir == "" {
			logDir = "logs" // 默认日志目录
		}

		var err error
		log, err = logger.NewWithFile(logLevel, logFormat, logDir, cfg.Log.EnableConsole)
		if err != nil {
			fmt.Fprintf(os.Stderr, "初始化文件日志失败: %v，将使用控制台日志\n", err)
			log = logger.New(logLevel, logFormat, os.Stdout)
		} else {
			// 确保程序退出时关闭日志文件
			defer func() {
				if closer, ok := log.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						fmt.Fprintf(os.Stderr, "关闭日志文件失败: %v\n", err)
					}
				}
			}()
		}
	} else {
		log = logger.New(logLevel, logFormat, os.Stdout)
	}

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
	// 注意：Genkit 客户端需要数据库连接来查询模型配置
	genkitClient, err := initGenkit(db, cfg, log)
	if err != nil {
		log.Warn("初始化 Genkit 客户端失败，AI服务将不可用", logger.Fields{"error": err})
		genkitClient = nil
	}

	// 6. 初始化服务
	var aiService ai.AIService
	var healthService health.Service

	// AI 服务需要 Genkit 客户端和数据库（用于历史消息记忆）
	if genkitClient != nil {
		aiService = initAIService(genkitClient, db, cfg, log)
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

	// 8.1 注册认证路由（如果数据库可用）
	var cleanupSvc cleanup.CleanupService
	var jwtAuthMW func(http.Handler) http.Handler
	if db != nil {
		authHandler, tenantHandler, userHandler, auditHandler, tenantMW, jwtAuthMiddleware, rbacMW := initAuthHandlers(db, cfg, log)
		jwtAuthMW = jwtAuthMiddleware // 保存 JWT 认证中间件供其他路由使用

		routes.RegisterAuthRoutes(serveMux, authHandler, tenantHandler, userHandler, auditHandler, tenantMW, jwtAuthMW, rbacMW)
		log.Info("认证和管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/auth/register",
				"/api/v1/auth/login",
				"/api/v1/auth/refresh",
				"/api/v1/auth/logout",
				"/api/v1/auth/change-password",
				"/api/v1/auth/me",
				"/api/v1/tenants",
				"/api/v1/tenants/{id}",
				"/api/v1/tenants/{id}/status",
				"/api/v1/users",
				"/api/v1/users/{id}",
				"/api/v1/users/{id}/status",
				"/api/v1/audit/auth",
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

	// 8.2 注册模型配置管理路由（如果数据库可用）
	// 模型配置路由需要 JWT 认证中间件和 RBAC 中间件
	var rbacMW func(...string) func(http.Handler) http.Handler
	if db != nil && jwtAuthMW != nil {
		modelConfigHandler, rbacMiddleware := initModelConfigurationHandler(db, genkitClient, cfg, log)
		rbacMW = rbacMiddleware

		routes.RegisterModelConfigurationRoutes(serveMux, modelConfigHandler, jwtAuthMW, rbacMW)
		log.Info("模型配置管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/model-configurations",
				"/api/v1/model-configurations/available",
				"/api/v1/model-configurations/{id}",
				"/api/v1/model-configurations/{id}/status",
				"/api/v1/model-configurations/{id}/validate",
			},
		})
	} else {
		log.Warn("模型配置管理路由未注册（数据库不可用）", nil)
	}

	// 8.2.1 注册乐享知识库路由（如果配置可用）
	lexiangHandler := initLexiangHandler(cfg, log)
	if lexiangHandler != nil && jwtAuthMW != nil && rbacMW != nil {
		routes.RegisterLexiangRoutes(serveMux, lexiangHandler, jwtAuthMW, rbacMW)
		log.Info("乐享知识库路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/lexiang/spaces",
				"/api/v1/lexiang/spaces/{id}",
				"/api/v1/lexiang/entries",
				"/api/v1/lexiang/entries/{id}",
				"/api/v1/lexiang/upload",
				"/api/v1/lexiang/files/{id}",
				"/api/v1/lexiang/feedbacks",
			},
		})
	} else {
		log.Info("乐享知识库路由未注册（未配置或认证中间件不可用）", nil)
	}

	// 8.3 注册会话管理路由（如果数据库可用）
	// 会话管理路由需要 JWT 认证中间件
	var cacheWarmer *storage.CacheWarmer
	if db != nil && aiService != nil && jwtAuthMW != nil {
		sessionHandler, messageHandler, contextHandler, memoryHandler, summaryHandler, warmer, rbacMiddleware := initSessionHandlers(db, aiService, cfg, log)
		cacheWarmer = warmer
		if rbacMW == nil {
			rbacMW = rbacMiddleware
		}

		// 执行启动时缓存预热
		if cacheWarmer != nil {
			ctx := context.Background()
			if err := cacheWarmer.WarmupOnStartup(ctx); err != nil {
				log.Warn("启动时缓存预热失败", logger.Fields{"error": err})
			}

			// 启动定期预热
			cacheWarmer.StartPeriodicWarmup(ctx)
		}

		routes.RegisterSessionRoutes(serveMux, sessionHandler, messageHandler, jwtAuthMW)
		log.Info("会话管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/chat/sessions",
				"/api/v1/chat/sessions/{id}",
				"/api/v1/chat/sessions/{id}/messages",
				"/api/v1/chat/messages/{id}",
			},
		})

		// 注册上下文管理路由
		routes.RegisterContextRoutes(serveMux, contextHandler, jwtAuthMW, rbacMW)
		log.Info("上下文管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/contexts/build",
				"/api/v1/contexts/{sessionId}",
			},
		})

		// 注册记忆管理路由
		routes.RegisterMemoryRoutes(serveMux, memoryHandler, jwtAuthMW, rbacMW)
		log.Info("记忆管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/memories/search",
				"/api/v1/memories",
				"/api/v1/memories/cleanup",
				"/api/v1/memories/{id}",
			},
		})

		// 注册摘要管理路由
		routes.RegisterSummaryRoutes(serveMux, summaryHandler, jwtAuthMW, rbacMW)
		log.Info("摘要管理路由已注册", logger.Fields{
			"routes": []string{
				"/api/v1/summaries",
				"/api/v1/summaries/{id}",
				"/api/v1/summaries/session/{sessionId}",
				"/api/v1/summaries/check-trigger",
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
		"swagger_ui":   fmt.Sprintf("http://%s:%s/swagger/index.html", cfg.Server.Host, cfg.Server.Port),
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
	// 注意：WriteTimeout 设置为 10 分钟以支持长时间的流式输出
	// AI 流式响应可能需要较长时间生成内容，避免连接过早超时
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second, // 读取请求超时：15秒
		WriteTimeout: 10 * time.Minute, // 写入响应超时：10分钟（支持流式输出）
		IdleTimeout:  60 * time.Second, // 空闲连接超时：60秒
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

		// 停止缓存预热器
		if cacheWarmer != nil {
			cacheWarmer.Stop()
			log.Info("缓存预热器已停止", nil)
		}

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

	// 执行所有数据库迁移
	// 包括初始迁移和所有后续的增量迁移（如添加 created_by_name 字段等）
	if err := database.RunAllMigrations(gormDB); err != nil {
		log.Error("数据库迁移失败", logger.Fields{
			"error": err,
		})

		// 提供详细的错误信息和解决建议
		return fmt.Errorf("数据库迁移失败: %w\n\n可能的原因和解决方案:\n"+
			"1. 数据库权限不足 - 确保数据库用户具有 CREATE TABLE、CREATE INDEX、ALTER TABLE 等权限\n"+
			"2. UUID 扩展未启用 - 确保 PostgreSQL 支持 gen_random_uuid() 函数\n"+
			"3. 表已存在但结构不匹配 - 考虑使用 reset_db.go 脚本重置数据库\n"+
			"4. 数据库连接中断 - 检查网络连接和数据库服务状态\n"+
			"5. 磁盘空间不足 - 检查数据库服务器磁盘空间\n\n"+
			"详细错误信息: %v", err)
	}

	log.Info("数据库迁移完成", logger.Fields{
		"migrations": []string{
			"initial_migration",
			"fix_timestamps_migration",
			"add_created_by_name_migration",
		},
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
// 注入 ModelConfigurationRepository 和 EncryptionService 以支持动态模型配置
func initGenkit(db database.Database, cfg *config.Config, log logger.Logger) (genkit.Client, error) {
	log.Info("初始化 Genkit 客户端...", logger.Fields{
		"model": cfg.Genkit.Model,
	})

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()
	if gormDB == nil {
		log.Warn("无法获取数据库实例，Genkit 客户端将以传统模式初始化", nil)
		// 降级到传统模式：不注入 repository
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
		if err := client.InitializeModel(ctx); err != nil {
			return nil, fmt.Errorf("初始化 Genkit 模型失败: %w", err)
		}

		log.Info("Genkit 客户端初始化成功（传统模式）", logger.Fields{
			"model": cfg.Genkit.Model,
		})

		return client, nil
	}

	// 2. 创建 ModelConfigurationRepository
	modelConfigRepo := repository.NewModelConfigurationRepository(gormDB)

	// 3. 创建 EncryptionService
	// 将字符串密钥转换为32字节数组
	secretKeyBytes := []byte(cfg.Encryption.SecretKey)
	// 确保密钥长度为32字节
	if len(secretKeyBytes) < 32 {
		// 如果密钥不足32字节，填充到32字节
		paddedKey := make([]byte, 32)
		copy(paddedKey, secretKeyBytes)
		secretKeyBytes = paddedKey
	} else if len(secretKeyBytes) > 32 {
		// 如果密钥超过32字节，截取前32字节
		secretKeyBytes = secretKeyBytes[:32]
	}

	encryptionService, err := service.NewEncryptionService(secretKeyBytes)
	if err != nil {
		log.Error("创建加密服务失败", logger.Fields{"error": err})
		// 如果加密服务创建失败，使用环境变量方式
		encryptionService, err = service.NewEncryptionServiceFromEnv()
		if err != nil {
			log.Error("从环境变量创建加密服务失败", logger.Fields{"error": err})
			return nil, fmt.Errorf("无法创建加密服务: %w", err)
		}
	}

	log.Info("创建 Genkit 客户端（注入 ModelConfigurationRepository 和 EncryptionService）", logger.Fields{
		"mode": "dynamic_configuration",
	})

	// 4. 创建 Genkit 客户端并注入 repository 和 encryptionService
	client := genkit.NewClientWithServices(modelConfigRepo, encryptionService)

	// 4. 初始化客户端（传统配置，用于向后兼容）
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

	// 5. 初始化并设置 Genkit 模型（用于向后兼容）
	// 注意：这里初始化的是默认模型，实际使用时会根据租户和模型名称动态选择
	if err := client.InitializeModel(ctx); err != nil {
		return nil, fmt.Errorf("初始化 Genkit 模型失败: %w", err)
	}

	log.Info("Genkit 客户端初始化成功（动态配置模式）", logger.Fields{
		"model":                  cfg.Genkit.Model,
		"repository_injected":    true,
		"dynamic_config_enabled": true,
	})

	return client, nil
}

// initAIService 初始化 AI 服务
// 参数:
//   - genkitClient: Genkit 客户端
//   - db: 数据库连接（用于获取历史消息，实现"记忆"功能）
//   - cfg: 配置
//   - log: 日志记录器
func initAIService(genkitClient genkit.Client, db database.Database, cfg *config.Config, log logger.Logger) ai.AIService {
	log.Info("初始化 AI 服务...", logger.Fields{
		"sessionTimeout":         cfg.Session.Timeout,
		"sessionCleanupInterval": cfg.Session.CleanupInterval,
	})

	// 创建上下文管理器
	contextManager := ai.NewContextManager(
		cfg.Session.Timeout,
		cfg.Session.CleanupInterval,
	)

	// 启动上下文管理器的自动清理
	contextManager.Start()

	// 创建会话上下文服务（用于获取历史消息，实现"记忆"功能）
	var conversationContextService ai.ConversationContextService
	if db != nil {
		gormDB := db.GetDB()
		if gormDB != nil {
			messageRepo := repository.NewMessageRepository(gormDB)
			sessionRepo := repository.NewSessionRepository(gormDB)
			userRepo := repository.NewUserRepository(gormDB)
			conversationContextService = service.NewConversationContextService(messageRepo, sessionRepo, userRepo)
			log.Info("会话上下文服务已启用（支持历史消息记忆）", nil)
		}
	}

	// 创建 AI 服务
	var aiService ai.AIService
	if conversationContextService != nil {
		// 使用带上下文服务的构造函数（支持历史消息记忆）
		aiService = ai.NewGenkitServiceWithContext(genkitClient, contextManager, conversationContextService, log)
	} else {
		// 使用基础构造函数（不支持历史消息记忆）
		aiService = ai.NewGenkitService(genkitClient, contextManager, log)
		log.Warn("会话上下文服务未启用，AI 对话将不支持历史消息记忆", nil)
	}

	log.Info("AI 服务初始化成功", nil)

	return aiService
}

// initSessionHandlers 初始化会话管理相关的处理器
func initSessionHandlers(db database.Database, aiService ai.AIService, cfg *config.Config, log logger.Logger) (
	*handler.SessionHandler,
	*handler.MessageHandler,
	*handler.ContextHandler,
	*handler.MemoryHandler,
	*handler.SummaryHandler,
	*storage.CacheWarmer,
	func(...string) func(http.Handler) http.Handler,
) {
	log.Info("初始化会话管理服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository 层实例
	sessionRepo := repository.NewSessionRepository(gormDB)
	messageRepo := repository.NewMessageRepository(gormDB)
	summaryRepo := repository.NewSummaryRepository(gormDB)
	contextRepo := repository.NewContextRepository(gormDB)
	memoryRepo := repository.NewMemoryRepository(gormDB)
	userRepo := repository.NewUserRepository(gormDB)

	// 3. 创建依赖服务
	// 3.1 创建 TokenManager
	tokenManager := service.NewTokenManager(log)

	// 3.2 创建 Genkit Client（如果需要）
	genkitClient := genkit.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 初始化 Genkit 客户端
	if err := genkitClient.Initialize(ctx, &genkit.Config{
		APIKey: cfg.Genkit.APIKey,
		Model:  cfg.Genkit.Model,
	}); err != nil {
		log.Warn("初始化 Genkit 客户端失败，摘要服务可能不可用", logger.Fields{"error": err})
	} else {
		if err := genkitClient.InitializeModel(ctx); err != nil {
			log.Warn("初始化 Genkit 模型失败，摘要服务可能不可用", logger.Fields{"error": err})
		}
	}

	// 3.3 初始化 Redis 客户端和缓存服务（如果启用）
	var cacheService storage.CacheService
	var cacheWarmer *storage.CacheWarmer

	if cfg.Redis.Enabled {
		redisClient, err := database.NewRedisClient(cfg.Redis, log)
		if err != nil {
			log.Warn("Redis 连接失败，缓存功能将被禁用", logger.Fields{"error": err})
		} else {
			// 创建缓存服务
			cacheService = storage.NewCacheService(redisClient, log)
			log.Info("缓存服务已启用", nil)

			// 创建缓存预热器
			cacheWarmerConfig := &storage.CacheWarmerConfig{
				WarmupInterval:    30 * time.Minute, // 30分钟预热一次
				ActiveSessionDays: 7,                // 7天内活跃的会话
			}
			cacheWarmer = storage.NewCacheWarmer(
				cacheService,
				contextRepo,
				summaryRepo,
				sessionRepo,
				gormDB,
				log,
				cacheWarmerConfig,
			)
			log.Info("缓存预热器已创建", logger.Fields{
				"warmup_interval_minutes": cacheWarmerConfig.WarmupInterval.Minutes(),
				"active_session_days":     cacheWarmerConfig.ActiveSessionDays,
			})
		}
	} else {
		log.Info("Redis 已禁用，缓存功能将不可用", nil)
	}

	// 3.4 初始化 Qdrant 客户端（如果配置可用）
	var qdrantClient storage.QdrantClient
	// TODO: 从配置文件读取 Qdrant 配置
	// 暂时使用 nil，后续需要添加配置支持
	if cfg.Redis.Enabled { // 临时使用 Redis 配置作为判断条件
		log.Info("Qdrant 客户端未配置，向量检索功能将不可用", nil)
	}

	// 3.5 初始化向量服务（如果配置可用）
	var vectorService ai.VectorService
	// TODO: 从配置文件读取向量服务配置
	// 暂时使用 nil，后续需要添加配置支持
	if cfg.Genkit.APIKey != "" {
		log.Info("向量服务未配置，向量嵌入功能将不可用", nil)
	}

	// 3.6 注册 Genkit Flows（需要在创建Service之前注册）
	// 注意：Flows需要在Service创建之前注册，因为它们需要Service实例
	// 我们将在创建Service之后再注册Flows

	// 4. 创建 Service 层实例
	// 4.1 创建 SessionService
	sessionService := session.NewSessionService(sessionRepo, messageRepo)

	// 4.2 创建 SummaryService
	summaryService := session.NewSummaryService(
		summaryRepo,
		messageRepo,
		contextRepo,
		sessionRepo,
		genkitClient,
		tokenManager,
		log,
	)

	// 4.3 创建 MessageService
	messageService := session.NewMessageService(gormDB, sessionRepo, messageRepo, aiService, log)

	// 4.4 创建 ContextService
	contextService := service.NewContextService(
		sessionRepo,
		messageRepo,
		memoryRepo,
		contextRepo,
		summaryRepo,
		userRepo,
		vectorService,
		tokenManager,
	)

	// 4.5 创建 MemoryService
	memoryService := service.NewMemoryService(
		memoryRepo,
		sessionRepo,
		userRepo,
		vectorService,
		qdrantClient,
		tokenManager,
	)

	// 4.6 注册 Genkit Flows（在Service创建之后）
	if genkitClient != nil {
		genkitInstance := genkitClient.GetGenkit()

		// 注册上下文管理Flow
		flows.RegisterContextFlows(genkitInstance, contextService)
		log.Info("上下文管理Flow已注册", logger.Fields{
			"flows": []string{"contextBuildFlow"},
		})

		// 注册记忆管理Flow
		flows.RegisterMemoryFlows(genkitInstance, memoryService)
		log.Info("记忆管理Flow已注册", logger.Fields{
			"flows": []string{"memorySearchFlow", "memoryStoreFlow", "memoryCleanupFlow"},
		})

		// 注册摘要管理Flow
		flows.RegisterSummaryFlows(genkitInstance, summaryService)
		log.Info("摘要管理Flow已注册", logger.Fields{
			"flows": []string{"summaryGenerateFlow", "summaryTriggerCheckFlow"},
		})
	} else {
		log.Warn("Genkit客户端未初始化，Flows未注册", nil)
	}

	// 5. 创建 Handler 层实例
	sessionHandler := handler.NewSessionHandler(sessionService, log)
	messageHandler := handler.NewMessageHandler(messageService, log)
	contextHandler := handler.NewContextHandler(contextService, log)
	memoryHandler := handler.NewMemoryHandler(memoryService, log)
	summaryHandler := handler.NewSummaryHandler(summaryService, log)

	// 6. 创建 RBAC 中间件工厂函数
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

	log.Info("会话管理服务初始化成功", logger.Fields{
		"repositories":  []string{"SessionRepository", "MessageRepository", "SummaryRepository", "ContextRepository", "MemoryRepository"},
		"services":      []string{"SessionService", "MessageService", "SummaryService", "ContextService", "MemoryService", "CacheService"},
		"handlers":      []string{"SessionHandler", "MessageHandler", "ContextHandler", "MemoryHandler", "SummaryHandler"},
		"cache_enabled": cacheService != nil,
	})

	return sessionHandler, messageHandler, contextHandler, memoryHandler, summaryHandler, cacheWarmer, rbacMiddleware
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
	authHandler := handler.NewAuthHandler(authService, emailService, userService, log)
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

// initModelConfigurationHandler 初始化模型配置管理处理器
func initModelConfigurationHandler(db database.Database, genkitClient genkit.Client, cfg *config.Config, log logger.Logger) (
	*handler.ModelConfigurationHandler,
	func(...string) func(http.Handler) http.Handler,
) {
	log.Info("初始化模型配置管理服务...", nil)

	// 1. 获取 GORM 数据库实例
	gormDB := db.GetDB()

	// 2. 创建 Repository 层实例
	modelConfigRepo := repository.NewModelConfigurationRepository(gormDB)

	// 3. 创建 EncryptionService
	// 将字符串密钥转换为32字节数组
	secretKeyBytes := []byte(cfg.Encryption.SecretKey)
	// 确保密钥长度为32字节
	if len(secretKeyBytes) < 32 {
		// 如果密钥不足32字节，填充到32字节
		paddedKey := make([]byte, 32)
		copy(paddedKey, secretKeyBytes)
		secretKeyBytes = paddedKey
	} else if len(secretKeyBytes) > 32 {
		// 如果密钥超过32字节，截取前32字节
		secretKeyBytes = secretKeyBytes[:32]
	}

	encryptionService, err := service.NewEncryptionService(secretKeyBytes)
	if err != nil {
		log.Error("创建加密服务失败", logger.Fields{"error": err})
		// 如果加密服务创建失败，使用环境变量方式
		encryptionService, err = service.NewEncryptionServiceFromEnv()
		if err != nil {
			log.Error("从环境变量创建加密服务失败", logger.Fields{"error": err})
			panic(fmt.Sprintf("无法创建加密服务: %v", err))
		}
	}

	// 4. 创建 ModelConfigurationService（注入 genkit client）
	modelConfigService := service.NewModelConfigurationService(
		modelConfigRepo,
		encryptionService,
		genkitClient,
	)

	// 5. 创建 Handler 层实例
	modelConfigHandler := handler.NewModelConfigurationHandler(modelConfigService, log)

	// 6. 创建 RBAC 中间件工厂函数
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

	log.Info("模型配置管理服务初始化成功", logger.Fields{
		"repositories": []string{"ModelConfigurationRepository"},
		"services":     []string{"EncryptionService", "ModelConfigurationService", "GenkitClient"},
		"handlers":     []string{"ModelConfigurationHandler"},
	})

	return modelConfigHandler, rbacMiddleware
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
			"平台租户ID":  result.TenantID,
			"管理员邮箱":   result.AdminEmail,
			"管理员初始密码": result.AdminPassword,
			"重要提示":    "请妥善保管管理员初始密码，建议首次登录后立即修改",
		})
	} else {
		log.Info("系统已初始化，跳过初始化流程", nil)
	}

	return nil
}

// initLexiangHandler 初始化乐享知识库处理器
// 从环境变量读取 LEXIANG_APP_KEY 和 LEXIANG_APP_SECRET
func initLexiangHandler(cfg *config.Config, log logger.Logger) *handler.LexiangHandler {
	// 尝试从环境变量创建乐享客户端
	client, err := lexiang.NewClientFromEnv()
	if err != nil {
		log.Info("乐享客户端未配置，跳过初始化", logger.Fields{
			"reason": err.Error(),
			"hint":   "设置 LEXIANG_APP_KEY 和 LEXIANG_APP_SECRET 环境变量以启用乐享知识库功能",
		})
		return nil
	}

	log.Info("乐享客户端初始化成功", nil)
	return handler.NewLexiangHandler(client, log)
}
