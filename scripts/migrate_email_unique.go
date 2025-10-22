package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/database/migrations"
)

func main() {
	fmt.Println("=== 邮箱全局唯一性迁移工具 ===")
	fmt.Println()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("📡 正在连接数据库...")
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ 数据库连接成功")
	fmt.Println()

	// 执行邮箱全局唯一性迁移
	fmt.Println("🔄 开始执行邮箱全局唯一性迁移...")
	fmt.Println()

	if err := migrations.RunEmailUniqueMigration(db.GetDB()); err != nil {
		log.Fatalf("❌ 迁移失败: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ 邮箱全局唯一性迁移成功完成！")
	fmt.Println()
	fmt.Println("变更内容：")
	fmt.Println("  - 删除了 idx_tenant_email 索引（租户+邮箱联合唯一）")
	fmt.Println("  - 创建了 idx_users_email_unique 索引（邮箱全局唯一）")
	fmt.Println("  - 更新了邮箱字段的注释")
	fmt.Println()
	fmt.Println("💡 提示：现在用户登录时只需要提供邮箱和密码，不再需要租户ID")

	os.Exit(0)
}
