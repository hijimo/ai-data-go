package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"genkit-ai-service/internal/api"
	"genkit-ai-service/internal/api/handler"
	"genkit-ai-service/internal/api/routes"
	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/loader"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/health"
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

// @tag.name providers
// @tag.description 模型提供商管理接口

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
		log.Warn("初始化数据库失败，AI服务将不可用", logger.Fields{"error": err})
		db = nil
	} else {
		defer func() {
			if err := db.Close(); err != nil {
				log.Error("关闭数据库连接失败", logger.Fields{"error": err})
			}
		}()
	}

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
	
	if genkitClient != nil && db != nil {
		aiService = initAIService(genkitClient, cfg, log)
		healthService = health.NewService(genkitClient, db, Version)
		log.Info("AI服务和健康检查服务已启用", nil)
	} else {
		log.Warn("AI服务和健康检查服务未启用（缺少必要依赖）", nil)
	}

	// 7. 初始化路由和处理器
	var mux http.Handler
	if aiService != nil && healthService != nil {
		router := api.NewRouter(aiService, healthService, log)
		mux = router.Handler()
	} else {
		// 只创建基础的 ServeMux
		mux = http.NewServeMux()
		log.Info("使用基础路由（仅模型提供商API）", nil)
	}

	// 8. 注册模型提供商API路由
	providerHandler := handler.NewProviderHandler(providerService, log)
	
	// 获取 ServeMux 实例
	var serveMux *http.ServeMux
	if sm, ok := mux.(*http.ServeMux); ok {
		serveMux = sm
	} else {
		// 如果 mux 不是 ServeMux，创建一个新的
		serveMux = http.NewServeMux()
		mux = serveMux
	}
	
	routes.RegisterProviderRoutes(serveMux, providerHandler)
	log.Info("模型提供商API路由已注册", nil)

	// 注册 Swagger UI 路由
	serveMux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	log.Info("Swagger UI 已启用", logger.Fields{
		"url": fmt.Sprintf("http://%s:%s/swagger/index.html", cfg.Server.Host, cfg.Server.Port),
	})

	// 9. 创建 HTTP 服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10. 启动服务器（在 goroutine 中）
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务器启动", logger.Fields{
			"address": server.Addr,
		})
		serverErrors <- server.ListenAndServe()
	}()

	// 11. 监听系统信号以实现优雅关闭
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// 12. 等待关闭信号或服务器错误
	select {
	case err := <-serverErrors:
		log.Error("服务器启动失败", logger.Fields{"error": err})
		os.Exit(1)

	case sig := <-shutdown:
		log.Info("收到关闭信号，开始优雅关闭", logger.Fields{
			"signal": sig.String(),
		})

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
	log.Info("初始化数据库连接...", logger.Fields{
		"host": cfg.Database.Host,
		"port": cfg.Database.Port,
		"name": cfg.Database.DBName,
	})

	postgresConfig := &database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
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

	return db, nil
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
