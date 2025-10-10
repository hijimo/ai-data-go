package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-knowledge-platform/internal/config"
	"ai-knowledge-platform/internal/database"
	"ai-knowledge-platform/internal/middleware"
	"ai-knowledge-platform/internal/router"
	"ai-knowledge-platform/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// @title AI知识管理平台 API
// @version 1.0
// @description 支持RAG、模型蒸馏、SFT等功能的大模型知识管理平台
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		logrus.Warn("未找到 .env 文件，使用系统环境变量")
	}

	// 初始化配置
	cfg := config.Load()

	// 设置日志级别
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// 初始化数据库连接
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logrus.Fatalf("数据库连接失败: %v", err)
	}

	// 构建数据库URL用于迁移
	databaseURL := cfg.Database.URL
	if databaseURL == "" {
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
			cfg.Database.SSLMode,
		)
	}

	// 运行数据库迁移
	if err := database.RunMigrations(databaseURL); err != nil {
		logrus.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化Redis连接
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		logrus.Fatalf("Redis连接失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	r := gin.New()

	// 添加中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.Metrics())

	// 设置路由
	router.SetupRoutes(r, db, redisClient)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	// 启动服务器
	go func() {
		logrus.Infof("服务器启动在端口 %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("正在关闭服务器...")

	// 设置5秒的超时时间来关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("服务器强制关闭: %v", err)
	}

	logrus.Info("服务器已退出")
}