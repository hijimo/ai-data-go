package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
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
	force      = flag.Bool("force", false, "跳过确认提示，直接执行重置")
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
	fmt.Println("  数据库重置工具")
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

	// 显示警告信息
	printWarning(cfg)

	// 确认操作
	if !*force {
		if !confirmReset() {
			fmt.Println("❌ 操作已取消")
			os.Exit(0)
		}
	}

	// 创建数据库配置
	dbConfig := createDBConfig(cfg)

	// 创建数据库实例
	db := database.NewPostgresDatabase(dbConfig)

	// 连接数据库
	fmt.Println()
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

	// 执行数据库重置
	if err := resetDatabase(db); err != nil {
		log.Fatalf("❌ 数据库重置失败: %v", err)
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ 数据库重置成功完成！")
	fmt.Println("========================================")
	fmt.Println()

	os.Exit(0)
}

// resetDatabase 重置数据库
func resetDatabase(db *database.PostgresDatabase) error {
	migration := migrations.NewInitialMigration(db.GetDB())

	// 步骤1: 执行 Down 方法清空所有表
	fmt.Println("🗑️  步骤 1/2: 删除所有表...")
	startTime := time.Now()

	if err := migration.Down(); err != nil {
		return fmt.Errorf("删除表失败: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ 所有表已删除 (耗时: %v)\n", duration)
	fmt.Println()

	// 步骤2: 执行 Up 方法重建表结构
	fmt.Println("🏗️  步骤 2/2: 重建表结构...")
	startTime = time.Now()

	if err := migration.Up(); err != nil {
		return fmt.Errorf("重建表结构失败: %w", err)
	}

	duration = time.Since(startTime)
	fmt.Printf("✅ 表结构已重建 (耗时: %v)\n", duration)
	fmt.Println()

	fmt.Println("已重建的表:")
	fmt.Println("  - tenants (租户表)")
	fmt.Println("  - users (用户表)")
	fmt.Println("  - refresh_tokens (刷新令牌表)")
	fmt.Println("  - email_verification_tokens (邮箱验证令牌表)")
	fmt.Println("  - auth_audit (认证审计表)")
	fmt.Println("  - chat_sessions (聊天会话表)")
	fmt.Println("  - chat_messages (聊天消息表)")
	fmt.Println("  - chat_summaries (聊天摘要表)")

	return nil
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

// printWarning 打印警告信息
func printWarning(cfg *config.Config) {
	fmt.Println("⚠️  警告: 此操作将执行以下步骤:")
	fmt.Println()
	fmt.Println("  1. 删除数据库中的所有表")
	fmt.Println("  2. 重新创建所有表结构")
	fmt.Println()
	fmt.Println("⚠️  所有数据将被永久删除且无法恢复！")
	fmt.Println()
	fmt.Printf("目标数据库: %s@%s:%s/%s\n", 
		cfg.Database.User, 
		cfg.Database.Host, 
		cfg.Database.Port, 
		cfg.Database.DBName)
	fmt.Println()
}

// confirmReset 确认重置操作
func confirmReset() bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入数据库名称以确认操作: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("读取输入失败: %v", err)
		return false
	}

	// 去除换行符和空格
	input = strings.TrimSpace(input)

	// 加载配置以获取数据库名称
	cfg, err := loadConfig()
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return false
	}

	if input != cfg.Database.DBName {
		fmt.Println()
		fmt.Printf("❌ 输入的数据库名称 '%s' 与配置不匹配 '%s'\n", input, cfg.Database.DBName)
		return false
	}

	fmt.Println()
	fmt.Print("确认要继续吗？(yes/no): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("读取输入失败: %v", err)
		return false
	}

	confirm = strings.TrimSpace(strings.ToLower(confirm))
	return confirm == "yes" || confirm == "y"
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("数据库重置工具")
	fmt.Println()
	fmt.Println("用途:")
	fmt.Println("  在开发环境中重置数据库，删除所有表并重新创建")
	fmt.Println()
	fmt.Println("⚠️  警告: 此工具会删除所有数据，仅用于开发环境！")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Println("  go run scripts/reset_db.go [选项]")
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  # 交互式重置（需要确认）")
	fmt.Println("  go run scripts/reset_db.go")
	fmt.Println()
	fmt.Println("  # 强制重置（跳过确认）")
	fmt.Println("  go run scripts/reset_db.go -force")
	fmt.Println()
	fmt.Println("  # 使用命令行参数覆盖配置")
	fmt.Println("  go run scripts/reset_db.go -host localhost -port 5432 -user postgres -dbname mydb")
	fmt.Println()
	fmt.Println("  # 显示详细日志")
	fmt.Println("  go run scripts/reset_db.go -verbose")
	fmt.Println()
	fmt.Println("环境变量:")
	fmt.Println("  DB_HOST         - 数据库主机地址")
	fmt.Println("  DB_PORT         - 数据库端口")
	fmt.Println("  DB_USER         - 数据库用户名")
	fmt.Println("  DB_PASSWORD     - 数据库密码")
	fmt.Println("  DB_NAME         - 数据库名称")
	fmt.Println("  DB_SSLMODE      - SSL模式 (disable, require, verify-ca, verify-full)")
	fmt.Println()
	fmt.Println("安全提示:")
	fmt.Println("  - 此工具仅用于开发环境")
	fmt.Println("  - 不要在生产环境中使用")
	fmt.Println("  - 操作前请确保已备份重要数据")
	fmt.Println("  - 使用 -force 参数时请格外小心")
	fmt.Println()
}
