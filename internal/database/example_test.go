package database_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"genkit-ai-service/internal/database"
)

// Example_postgresDatabase 演示如何使用 PostgreSQL 数据库连接管理
func Example_postgresDatabase() {
	// 创建数据库配置
	config := &database.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "postgres",
		Password:        "your-password",
		DBName:          "genkit_ai_service",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// 创建数据库实例
	db := database.NewPostgresDatabase(config)

	// 连接数据库
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 检查数据库连接
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("数据库连接检查失败: %v", err)
	}

	fmt.Println("数据库连接成功")

	// 获取数据库实例进行查询
	sqlDB := db.GetDB()
	if sqlDB != nil {
		// 可以使用 sqlDB 执行查询
		// rows, err := sqlDB.QueryContext(ctx, "SELECT * FROM users")
		fmt.Println("可以使用数据库实例执行查询")
	}

	// 优雅关闭
	if err := db.Close(); err != nil {
		log.Fatalf("关闭数据库连接失败: %v", err)
	}

	fmt.Println("数据库连接已关闭")
}

// Example_postgresDatabase_healthCheck 演示如何在健康检查中使用数据库连接
func Example_postgresDatabase_healthCheck() {
	config := &database.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "postgres",
		Password:        "your-password",
		DBName:          "genkit_ai_service",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db := database.NewPostgresDatabase(config)
	ctx := context.Background()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 健康检查
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		fmt.Println("数据库健康检查失败")
		return
	}

	fmt.Println("数据库健康检查通过")
}
