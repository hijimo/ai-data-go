package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/database/migrations"
)

var (
	// 命令行参数
	dbHost     = flag.String("host", "", "数据库主机地址 (覆盖环境变量)")
	dbPort     = flag.String("port", "", "数据库端口 (覆盖环境变量)")
	dbUser     = flag.String("user", "", "数据库用户名 (覆盖环境变量)")
	dbPassword = flag.String("password", "", "数据库密码 (覆盖环境变量)")
	dbName     = flag.String("dbname", "", "数据库名称 (覆盖环境变量)")
	dbSSLMode  = flag.String("sslmode", "", "SSL模式 (覆盖环境变量)")
	verbose    = flag.Bool("verbose", false, "显示详细日志")
	help       = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 显示帮助信息
	if *help {
		printHelp()
		os.Exit(0)
	}

	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("========================================")
	fmt.Println("  数据库初始迁移工具")
	fmt.Println("========================================")
	fmt.Println()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 显示配置信息
	if *verbose {
		printConfig(cfg)
	}

	// 创建数据库配置
	dbConfig := createDBConfig(cfg)

	// 创建数据库实例
	db := database.NewPostgresDatabase(dbConfig)

	// 连接数据库
	fmt.Println("📡 正在连接数据库...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("⚠️  关闭数据库连接时出错: %v", err)
		}
	}()

	fmt.Println("✅ 数据库连接成功")
	fmt.Println()

	// 执行初始迁移
	fmt.Println("🚀 开始执行初始迁移...")
	fmt.Println()

	startTime := time.Now()

	if err := migrations.RunInitialMigration(db.GetDB()); err != nil {
		log.Fatalf("❌ 初始迁移失败: %v", err)
	}

	duration := time.Since(startTime)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Printf("✅ 初始迁移成功完成！耗时: %v\n", duration)
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("已创建的表:")
	fmt.Println("  - tenants (租户表)")
	fmt.Println("  - users (用户表)")
	fmt.Println("  - refresh_tokens (刷新令牌表)")
	fmt.Println("  - email_verification_tokens (邮箱验证令牌表)")
	fmt.Println("  - auth_audit (认证审计表)")
	fmt.Println("  - chat_sessions (聊天会话表)")
	fmt.Println("  - chat_messages (聊天消息表)")
	fmt.Println("  - chat_summaries (聊天摘要表)")
	fmt.Println()

	os.Exit(0)
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 命令行参数覆盖环境变量
	if *dbHost != "" {
		cfg.Database.Host = *dbHost
	}
	if *dbPort != "" {
		cfg.Database.Port = *dbPort
	}
	if *dbUser != "" {
		cfg.Database.User = *dbUser
	}
	if *dbPassword != "" {
		cfg.Database.Password = *dbPassword
	}
	if *dbName != "" {
		cfg.Database.DBName = *dbName
	}
	if *dbSSLMode != "" {
		cfg.Database.SSLMode = *dbSSLMode
	}

	return cfg, nil
}

// createDBConfig 创建数据库配置
func createDBConfig(cfg *config.Config) *database.PostgresConfig {
	logLevel := cfg.Database.LogLevel
	if *verbose {
		logLevel = "info"
	}

	return &database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		LogLevel:        logLevel,
	}
}

// printConfig 打印配置信息
func printConfig(cfg *config.Config) {
	fmt.Println("📋 数据库配置:")
	fmt.Printf("  主机: %s\n", cfg.Database.Host)
	fmt.Printf("  端口: %s\n", cfg.Database.Port)
	fmt.Printf("  用户: %s\n", cfg.Database.User)
	fmt.Printf("  数据库: %s\n", cfg.Database.DBName)
	fmt.Printf("  SSL模式: %s\n", cfg.Database.SSLMode)
	fmt.Printf("  最大连接数: %d\n", cfg.Database.MaxOpenConns)
	fmt.Printf("  最大空闲连接数: %d\n", cfg.Database.MaxIdleConns)
	fmt.Printf("  连接最大生命周期: %v\n", cfg.Database.ConnMaxLifetime)
	fmt.Println()
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("数据库初始迁移工具")
	fmt.Println()
	fmt.Println("用途:")
	fmt.Println("  在新环境中执行初始数据库迁移，创建所有必要的表结构")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  go run scripts/init_migration.go [选项]")
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  # 使用环境变量配置")
	fmt.Println("  go run scripts/init_migration.go")
	fmt.Println()
	fmt.Println("  # 使用命令行参数覆盖配置")
	fmt.Println("  go run scripts/init_migration.go -host localhost -port 5432 -user postgres -dbname mydb")
	fmt.Println()
	fmt.Println("  # 显示详细日志")
	fmt.Println("  go run scripts/init_migration.go -verbose")
	fmt.Println()
	fmt.Println("环境变量:")
	fmt.Println("  DB_HOST         - 数据库主机地址")
	fmt.Println("  DB_PORT         - 数据库端口")
	fmt.Println("  DB_USER         - 数据库用户名")
	fmt.Println("  DB_PASSWORD     - 数据库密码")
	fmt.Println("  DB_NAME         - 数据库名称")
	fmt.Println("  DB_SSLMODE      - SSL模式 (disable, require, verify-ca, verify-full)")
	fmt.Println()
}
