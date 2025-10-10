package router

import (
	"net/http"

	"ai-knowledge-platform/internal/auth"
	"ai-knowledge-platform/internal/config"
	"ai-knowledge-platform/internal/handler"
	"ai-knowledge-platform/internal/kms"
	"ai-knowledge-platform/internal/middleware"
	"ai-knowledge-platform/internal/repository"
	"ai-knowledge-platform/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, db *gorm.DB, redisClient *redis.Client) {
	// 获取认证配置
	authConfig := config.GetAuthConfig()
	
	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(authConfig.JWTSecret, authConfig.JWTExpiration)
	
	// 创建角色管理器
	roleManager := auth.NewRoleManager()
	
	// 获取KMS配置并初始化KMS管理器
	kmsConfig := config.GetKMSConfig()
	kmsManager := kms.NewManager()
	if err := kmsManager.InitializeFromConfigs(kmsConfig.Providers, kmsConfig.DefaultProvider); err != nil {
		// 在实际应用中，这里应该记录错误日志
		// 为了演示目的，我们继续执行，但KMS功能可能不可用
	}
	
	// 创建敏感信息管理器
	secretManager := kms.NewSecretManager(kmsManager)
	
	// 创建仓库
	projectRepo := repository.NewProjectRepository(db)
	
	// 创建服务
	migrationService := service.NewMigrationService(projectRepo, db)
	
	// 创建处理器
	authHandler := handler.NewAuthHandler(jwtManager)
	permissionHandler := handler.NewPermissionHandler(roleManager)
	kmsHandler := handler.NewKMSHandler(kmsManager, secretManager)
	migrationHandler := handler.NewMigrationHandler(migrationService)
	// 健康检查端点
	r.GET("/health", handler.HealthCheck(db, redisClient))
	
	// Prometheus指标端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// Swagger文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API版本1路由组
	v1 := r.Group("/api/v1")
	{
		// 认证相关路由（无需认证）
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.POST("/logout", middleware.AuthMiddleware(jwtManager), authHandler.Logout)
			authGroup.GET("/profile", middleware.AuthMiddleware(jwtManager), authHandler.GetProfile)
			authGroup.GET("/validate", middleware.AuthMiddleware(jwtManager), authHandler.ValidateToken)
		}

		// 权限管理路由（需要认证）
		permissions := v1.Group("/permissions")
		permissions.Use(middleware.AuthMiddleware(jwtManager))
		{
			permissions.GET("/roles", permissionHandler.ListRoles)
			permissions.GET("/roles/:role_name", permissionHandler.GetRole)
			permissions.GET("/user", permissionHandler.GetUserPermissions)
			permissions.GET("/check", permissionHandler.CheckPermission)
			permissions.POST("/check-multiple", permissionHandler.CheckMultiplePermissions)
			permissions.GET("/by-resource", permissionHandler.GetPermissionsByResource)
		}

		// KMS管理路由（需要系统管理员权限）
		kmsGroup := v1.Group("/kms")
		kmsGroup.Use(middleware.AuthMiddleware(jwtManager))
		kmsGroup.Use(middleware.SystemAdminMiddleware(roleManager))
		{
			kmsGroup.POST("/encrypt", kmsHandler.Encrypt)
			kmsGroup.POST("/decrypt", kmsHandler.Decrypt)
			kmsGroup.POST("/secrets/encrypt", kmsHandler.EncryptSecret)
			kmsGroup.POST("/secrets/decrypt", kmsHandler.DecryptSecret)
			kmsGroup.GET("/providers", kmsHandler.ListProviders)
			kmsGroup.GET("/health", kmsHandler.HealthCheck)
			kmsGroup.POST("/data-key", kmsHandler.GenerateDataKey)
		}

		// 项目管理路由（需要认证）
		projects := v1.Group("/projects")
		projects.Use(middleware.AuthMiddleware(jwtManager))
		projects.Use(middleware.ProjectIsolationMiddleware())
		{
			projects.GET("", middleware.PermissionMiddleware(roleManager, auth.PermProjectRead), handler.ListProjects)
			projects.POST("", middleware.PermissionMiddleware(roleManager, auth.PermProjectWrite), handler.CreateProject)
			projects.GET("/:id", middleware.PermissionMiddleware(roleManager, auth.PermProjectRead), handler.GetProject)
			projects.PUT("/:id", middleware.PermissionMiddleware(roleManager, auth.PermProjectWrite), handler.UpdateProject)
			projects.DELETE("/:id", middleware.PermissionMiddleware(roleManager, auth.PermProjectDelete), handler.DeleteProject)
			
			// 项目成员管理（需要项目管理权限）
			projects.GET("/:id/members", middleware.PermissionMiddleware(roleManager, auth.PermProjectManage), handler.ListProjectMembers)
			projects.POST("/:id/members", middleware.PermissionMiddleware(roleManager, auth.PermProjectManage), handler.AddProjectMember)
			projects.DELETE("/:id/members/:user_id", middleware.PermissionMiddleware(roleManager, auth.PermProjectManage), handler.RemoveProjectMember)
			
			// 项目数据迁移相关路由
			projects.GET("/:project_id/export", middleware.PermissionMiddleware(roleManager, auth.PermProjectRead), migrationHandler.ExportProject)
			projects.POST("/:project_id/import", middleware.PermissionMiddleware(roleManager, auth.PermProjectWrite), migrationHandler.ImportProject)
			projects.GET("/:project_id/stats", middleware.PermissionMiddleware(roleManager, auth.PermProjectRead), migrationHandler.GetProjectStats)
		}

		// 文档管理路由（需要认证）
		documents := v1.Group("/documents")
		documents.Use(middleware.AuthMiddleware(jwtManager))
		documents.Use(middleware.ProjectIsolationMiddleware())
		{
			documents.GET("", middleware.PermissionMiddleware(roleManager, auth.PermDocumentRead), handler.ListDocuments)
			documents.POST("/upload", middleware.PermissionMiddleware(roleManager, auth.PermDocumentUpload), handler.UploadDocument)
			documents.GET("/:id", middleware.PermissionMiddleware(roleManager, auth.PermDocumentRead), handler.GetDocument)
			documents.DELETE("/:id", middleware.PermissionMiddleware(roleManager, auth.PermDocumentDelete), handler.DeleteDocument)
			documents.POST("/:id/process", middleware.PermissionMiddleware(roleManager, auth.PermDocumentWrite), handler.ProcessDocument)
			documents.GET("/:id/chunks", middleware.PermissionMiddleware(roleManager, auth.PermDocumentRead), handler.GetDocumentChunks)
		}

		// 向量管理路由（需要认证）
		vectors := v1.Group("/vectors")
		vectors.Use(middleware.AuthMiddleware(jwtManager))
		{
			vectors.POST("/search", handler.VectorSearch)
			vectors.GET("/indexes", handler.ListVectorIndexes)
			vectors.POST("/indexes", handler.CreateVectorIndex)
			vectors.DELETE("/indexes/:id", handler.DeleteVectorIndex)
		}

		// LLM管理路由（需要认证）
		llm := v1.Group("/llm")
		llm.Use(middleware.AuthMiddleware(jwtManager))
		{
			llm.GET("/providers", middleware.PermissionMiddleware(roleManager, auth.PermLLMRead), handler.ListLLMProviders)
			llm.POST("/providers", middleware.PermissionMiddleware(roleManager, auth.PermLLMManage), handler.CreateLLMProvider)
			llm.PUT("/providers/:id", middleware.PermissionMiddleware(roleManager, auth.PermLLMManage), handler.UpdateLLMProvider)
			llm.DELETE("/providers/:id", middleware.PermissionMiddleware(roleManager, auth.PermLLMManage), handler.DeleteLLMProvider)
			llm.GET("/models", middleware.PermissionMiddleware(roleManager, auth.PermLLMRead), handler.ListLLMModels)
			llm.POST("/chat", middleware.PermissionMiddleware(roleManager, auth.PermLLMChat), handler.ChatWithLLM)
		}

		// Agent管理路由（需要认证）
		agents := v1.Group("/agents")
		agents.Use(middleware.AuthMiddleware(jwtManager))
		{
			agents.GET("", handler.ListAgents)
			agents.POST("", handler.CreateAgent)
			agents.GET("/:id", handler.GetAgent)
			agents.PUT("/:id", handler.UpdateAgent)
			agents.DELETE("/:id", handler.DeleteAgent)
			agents.POST("/:id/chat", handler.ChatWithAgent)
		}

		// 对话管理路由（需要认证）
		chat := v1.Group("/chat")
		chat.Use(middleware.AuthMiddleware(jwtManager))
		{
			chat.GET("/sessions", handler.ListChatSessions)
			chat.POST("/sessions", handler.CreateChatSession)
			chat.GET("/sessions/:id", handler.GetChatSession)
			chat.DELETE("/sessions/:id", handler.DeleteChatSession)
			chat.POST("/sessions/:id/messages", handler.SendMessage)
			chat.GET("/sessions/:id/messages", handler.GetChatMessages)
		}

		// 问题管理路由（需要认证）
		questions := v1.Group("/questions")
		questions.Use(middleware.AuthMiddleware(jwtManager))
		{
			questions.GET("", handler.ListQuestions)
			questions.POST("/generate", handler.GenerateQuestions)
			questions.GET("/:id", handler.GetQuestion)
			questions.PUT("/:id", handler.UpdateQuestion)
			questions.DELETE("/:id", handler.DeleteQuestion)
		}

		// 答案管理路由（需要认证）
		answers := v1.Group("/answers")
		answers.Use(middleware.AuthMiddleware(jwtManager))
		{
			answers.GET("", handler.ListAnswers)
			answers.POST("/generate", handler.GenerateAnswers)
			answers.GET("/:id", handler.GetAnswer)
			answers.PUT("/:id", handler.UpdateAnswer)
			answers.DELETE("/:id", handler.DeleteAnswer)
		}

		// 任务管理路由（需要认证）
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.AuthMiddleware(jwtManager))
		{
			tasks.GET("", handler.ListTasks)
			tasks.GET("/:id", handler.GetTask)
			tasks.POST("/:id/cancel", handler.CancelTask)
			tasks.GET("/:id/status", handler.GetTaskStatus)
		}

		// 数据集管理路由（需要认证）
		datasets := v1.Group("/datasets")
		datasets.Use(middleware.AuthMiddleware(jwtManager))
		{
			datasets.POST("/export", handler.ExportDataset)
			datasets.GET("/exports", handler.ListDatasetExports)
			datasets.GET("/exports/:id", handler.GetDatasetExport)
		}

		// 训练任务路由（需要认证）
		training := v1.Group("/training")
		training.Use(middleware.AuthMiddleware(jwtManager))
		{
			training.GET("/jobs", handler.ListTrainingJobs)
			training.POST("/jobs", handler.CreateTrainingJob)
			training.GET("/jobs/:id", handler.GetTrainingJob)
			training.POST("/jobs/:id/cancel", handler.CancelTrainingJob)
		}

		// 数据迁移路由（需要认证）
		migration := v1.Group("/migration")
		migration.Use(middleware.AuthMiddleware(jwtManager))
		{
			migration.POST("/tasks", migrationHandler.CreateMigrationTask)
			migration.GET("/tasks", migrationHandler.ListMigrationTasks)
			migration.GET("/tasks/:task_id", migrationHandler.GetMigrationTaskStatus)
			migration.POST("/tasks/:task_id/cancel", migrationHandler.CancelMigrationTask)
		}
	}

	// 404处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "接口不存在",
		})
	})
}