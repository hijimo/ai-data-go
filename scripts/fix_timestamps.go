package main

import (
	"fmt"
	"log"
	"os"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/database/migrations"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("开始修复时间戳字段...")

	// 创建并执行迁移
	migration := migrations.NewFixTimestampsMigration(db)
	if err := migration.Up(); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	fmt.Println("时间戳字段修复成功！")
	fmt.Println("- tenants 表的 created_at 和 updated_at 字段已设置默认值")
	fmt.Println("- users 表的 created_at 和 updated_at 字段已设置默认值")
	fmt.Println("- 已存在的空值记录已更新为当前时间")

	os.Exit(0)
}
