package main

import (
	"fmt"
	"log"
	"os"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/database/migrations"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 无法加载 .env 文件: %v", err)
	}

	// 加载配置
	cfg, err := config.Load()
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

	fmt.Println("=== 开始测试 Genkit 会话管理迁移 ===")

	// 创建迁移实例
	migration := migrations.NewGenkitSessionManagementMigration(db)

	// 执行迁移
	fmt.Println("\n1. 执行迁移...")
	if err := migration.Up(); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}
	fmt.Println("✓ 迁移执行成功")

	// 验证表是否创建成功
	fmt.Println("\n2. 验证表创建...")
	tables := []string{
		"conversation_memories",
		"conversation_contexts",
		"conversation_summaries",
	}

	for _, table := range tables {
		var exists bool
		err := db.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = ?
			)
		`, table).Scan(&exists).Error

		if err != nil {
			log.Fatalf("检查表 %s 失败: %v", table, err)
		}

		if !exists {
			log.Fatalf("表 %s 不存在", table)
		}

		fmt.Printf("✓ 表 %s 创建成功\n", table)
	}

	// 注意：由于使用 Qdrant 作为向量数据库，不再需要验证 pgvector 扩展和向量索引
	fmt.Println("\n3. 跳过 pgvector 验证（使用 Qdrant 作为向量数据库）")

	// 验证外键约束
	fmt.Println("\n4. 验证外键约束...")
	constraints := map[string][]string{
		"conversation_memories":  {"fk_memories_tenant", "fk_memories_session"},
		"conversation_contexts":  {"fk_contexts_tenant", "fk_contexts_session", "fk_contexts_last_summary"},
		"conversation_summaries": {"fk_summaries_tenant", "fk_summaries_session"},
	}

	for table, constraintList := range constraints {
		for _, constraint := range constraintList {
			var constraintExists bool
			err = db.Raw(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.table_constraints 
					WHERE table_name = ? 
					AND constraint_name = ?
				)
			`, table, constraint).Scan(&constraintExists).Error

			if err != nil {
				log.Fatalf("检查外键约束 %s.%s 失败: %v", table, constraint, err)
			}

			if !constraintExists {
				log.Fatalf("外键约束 %s.%s 不存在", table, constraint)
			}

			fmt.Printf("✓ 外键约束 %s.%s 创建成功\n", table, constraint)
		}
	}

	// 可选：回滚测试（如果设置了环境变量）
	if os.Getenv("TEST_ROLLBACK") == "true" {
		fmt.Println("\n5. 测试回滚...")
		if err := migration.Down(); err != nil {
			log.Fatalf("回滚迁移失败: %v", err)
		}
		fmt.Println("✓ 回滚成功")

		// 验证表是否已删除
		for _, table := range tables {
			var exists bool
			err := db.Raw(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables 
					WHERE table_schema = 'public' 
					AND table_name = ?
				)
			`, table).Scan(&exists).Error

			if err != nil {
				log.Fatalf("检查表 %s 失败: %v", table, err)
			}

			if exists {
				log.Fatalf("表 %s 仍然存在", table)
			}

			fmt.Printf("✓ 表 %s 已删除\n", table)
		}
	}

	fmt.Println("\n=== Genkit 会话管理迁移测试完成 ===")
}
