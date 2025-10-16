package health_test

import (
	"context"
	"fmt"
	"log"

	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/service/health"
)

// Example_basicUsage 演示基本使用方法
func Example_basicUsage() {
	// 初始化 Genkit 客户端
	genkitClient := genkit.NewClient()
	genkitConfig := &genkit.Config{
		APIKey:             "your-api-key",
		Model:              "gemini-2.5-flash",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2000,
	}
	if err := genkitClient.Initialize(context.Background(), genkitConfig); err != nil {
		log.Fatal(err)
	}

	// 初始化数据库连接
	dbConfig := &database.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "genkit_ai_service",
		SSLMode:  "disable",
	}
	db := database.NewPostgresDatabase(dbConfig)
	if err := db.Connect(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 创建健康检查服务
	healthService := health.NewService(genkitClient, db, "1.0.0")

	// 执行健康检查
	ctx := context.Background()
	status, err := healthService.Check(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 输出健康状态
	fmt.Printf("状态: %s\n", status.Status)
	fmt.Printf("版本: %s\n", status.Version)
	fmt.Printf("运行时间: %s\n", status.Uptime)
	fmt.Printf("Genkit 状态: %s\n", status.Dependencies["genkit"])
	fmt.Printf("数据库状态: %s\n", status.Dependencies["database"])
}

// Example_withoutDependencies 演示不使用依赖的情况
func Example_withoutDependencies() {
	// 创建健康检查服务，不配置依赖
	healthService := health.NewService(nil, nil, "1.0.0")

	// 执行健康检查
	ctx := context.Background()
	status, err := healthService.Check(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 输出健康状态
	fmt.Printf("状态: %s\n", status.Status)
	fmt.Printf("版本: %s\n", status.Version)
	fmt.Printf("Genkit 状态: %s\n", status.Dependencies["genkit"])
	fmt.Printf("数据库状态: %s\n", status.Dependencies["database"])
	// Output:
	// 状态: unhealthy
	// 版本: 1.0.0
	// Genkit 状态: not_configured
	// 数据库状态: not_configured
}

// Example_checkStatus 演示如何判断健康状态
func Example_checkStatus() {
	// 创建健康检查服务
	healthService := health.NewService(nil, nil, "1.0.0")

	// 执行健康检查
	ctx := context.Background()
	status, err := healthService.Check(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 根据状态执行不同操作
	if status.Status == "healthy" {
		fmt.Println("服务运行正常")
	} else {
		fmt.Println("服务存在问题")
		// 检查具体哪个依赖有问题
		for name, state := range status.Dependencies {
			if state != "connected" {
				fmt.Printf("依赖 %s 状态异常: %s\n", name, state)
			}
		}
	}
	// Output:
	// 服务存在问题
	// 依赖 genkit 状态异常: not_configured
	// 依赖 database 状态异常: not_configured
}
