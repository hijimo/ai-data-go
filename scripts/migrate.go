package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
)

// 在这里导入你的模型
// import "genkit-ai-service/internal/model"

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建数据库配置
	dbConfig := &database.PostgresConfig{
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

	// 创建数据库实例
	db := database.NewPostgresDatabase(dbConfig)

	// 连接数据库
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Println("数据库连接成功")

	// 执行迁移
	// 在这里添加你的模型结构体
	// 例如: db.AutoMigrate(&model.User{}, &model.ChatSession{})
	
	fmt.Println("请在此脚本中添加需要迁移的模型")
	fmt.Println("示例: db.AutoMigrate(&model.User{}, &model.ChatSession{})")
	
	// 取消下面的注释并添加你的模型
	/*
	fmt.Println("开始执行数据库迁移...")
	if err := db.AutoMigrate(
		// &model.User{},
		// &model.ChatSession{},
		// 在这里添加更多模型...
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	
	fmt.Println("数据库迁移成功完成")
	*/
	
	os.Exit(0)
}
