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
	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/service/ai"
	"genkit-ai-service/internal/service/health"
)

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

	// 3. 初始化数据库连接
	db, err := initDatabase(cfg, log)
	if err != nil {
		log.Error("初始化数据库失败", logger.Fields{"error": err})
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("关闭数据库连接失败", logger.Fields{"error": err})
		}
	}()

	// 4. 初始化 Genkit 客户端
	genkitClient, err := initGenkit(cfg, log)
	if err != nil {
		log.Error("初始化 Genkit 客户端失败", logger.Fields{"error": err})
		os.Exit(1)
	}

	// 5. 初始化服务
	aiService := initAIService(genkitClient, cfg, log)
	healthService := health.NewService(genkitClient, db, Version)

	// 6. 初始化路由和处理器
	router := api.NewRouter(aiService, healthService, log)
	handler := router.Handler()

	// 7. 创建 HTTP 服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 启动服务器（在 goroutine 中）
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务器启动", logger.Fields{
			"address": server.Addr,
		})
		serverErrors <- server.ListenAndServe()
	}()

	// 9. 监听系统信号以实现优雅关闭
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// 10. 等待关闭信号或服务器错误
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
