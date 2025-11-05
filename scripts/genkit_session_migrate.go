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
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}
	defer sqlDB.Close()

	// 检查命令行参数
	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	// 创建迁移实例
	migration := migrations.NewGenkitSessionMigration(db)

	// 执行迁移
	switch action {
	case "up":
		fmt.Println("开始执行 Genkit 会话管理模块迁移...")
		if err := migration.Up(); err != nil {
			log.Fatalf("迁移失败: %v", err)
		}
		fmt.Println("迁移成功完成！")

		// 记录迁移状态
		if err := migrations.RecordMigrationStatus(db, migration.GetName()); err != nil {
			log.Printf("警告: 记录迁移状态失败: %v", err)
		}

	case "down":
		fmt.Println("开始回滚 Genkit 会话管理模块迁移...")
		if err := migration.Down(); err != nil {
			log.Fatalf("回滚失败: %v", err)
		}
		fmt.Println("回滚成功完成！")

	case "create-vector-index":
		fmt.Println("开始创建向量索引...")
		if err := migration.CreateVectorIndex(); err != nil {
			log.Fatalf("创建向量索引失败: %v", err)
		}
		fmt.Println("向量索引创建成功！")

	default:
		fmt.Printf("未知操作: %s\n", action)
		fmt.Println("用法:")
		fmt.Println("  go run scripts/genkit_session_migrate.go [up|down|create-vector-index]")
		fmt.Println("")
		fmt.Println("操作说明:")
		fmt.Println("  up                  - 执行迁移（默认）")
		fmt.Println("  down                - 回滚迁移")
		fmt.Println("  create-vector-index - 创建向量索引（需要表中有足够数据）")
		os.Exit(1)
	}
}
