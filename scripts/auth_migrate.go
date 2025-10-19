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
	fmt.Println("=== 认证系统数据库迁移工具 ===")
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

	// 执行认证系统迁移
	fmt.Println("🔄 开始执行认证系统数据库迁移...")
	fmt.Println()

	// 执行认证表迁移
	if err := migrations.RunAuthMigrations(db.GetDB()); err != nil {
		log.Fatalf("❌ 认证表迁移失败: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ 认证系统数据库迁移成功完成！")
	fmt.Println()
	fmt.Println("已创建以下表：")
	fmt.Println("  - tenants (租户表)")
	fmt.Println("  - users (用户表)")
	fmt.Println("  - refresh_tokens (刷新令牌表)")
	fmt.Println("  - auth_audit (认证审计日志表)")
	fmt.Println()
	fmt.Println("💡 提示：运行 'go run scripts/init_auth.go' 来创建初始租户和管理员用户")

	os.Exit(0)
}
