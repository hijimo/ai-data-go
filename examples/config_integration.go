package main

import (
	"fmt"
	"log"

	"genkit-ai-service/internal/config"
)

// 演示如何在实际应用中集成配置管理
func main() {
	fmt.Println("配置管理集成示例")
	fmt.Println("==================\n")

	// 方式1: 使用MustLoad快速加载（推荐用于main函数）
	fmt.Println("方式1: 使用MustLoad快速加载")
	fmt.Println("------------------------------")
	
	// 这会自动根据APP_ENV加载配置，失败则panic
	// cfg := config.MustLoad()
	// fmt.Printf("✓ 配置加载成功\n\n")

	// 方式2: 使用ConfigLoader（推荐用于需要错误处理的场景）
	fmt.Println("方式2: 使用ConfigLoader")
	fmt.Println("------------------------------")
	
	loader := config.NewConfigLoader()
	cfg, err := loader.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	
	fmt.Printf("✓ 配置加载成功\n")
	fmt.Printf("  环境: %s\n", loader.GetEnv())
	fmt.Printf("  开发环境: %v\n", loader.IsDevelopment())
	fmt.Printf("  生产环境: %v\n", loader.IsProduction())
	fmt.Println()

	// 方式3: 根据环境使用不同的配置
	fmt.Println("方式3: 根据环境使用不同的配置")
	fmt.Println("------------------------------")
	
	if loader.IsDevelopment() {
		fmt.Println("✓ 开发环境配置:")
		fmt.Printf("  - 日志级别: %s (详细)\n", cfg.Log.Level)
		fmt.Printf("  - 数据库连接池: %d (较小)\n", cfg.Database.MaxOpenConns)
		fmt.Printf("  - Bcrypt Cost: %d (较低，快速)\n", cfg.Auth.BcryptCost)
	} else if loader.IsProduction() {
		fmt.Println("✓ 生产环境配置:")
		fmt.Printf("  - 日志级别: %s (精简)\n", cfg.Log.Level)
		fmt.Printf("  - 数据库连接池: %d (较大)\n", cfg.Database.MaxOpenConns)
		fmt.Printf("  - Bcrypt Cost: %d (较高，安全)\n", cfg.Auth.BcryptCost)
		fmt.Printf("  - SSL模式: %s\n", cfg.Database.SSLMode)
	}
	fmt.Println()

	// 方式4: 使用配置初始化服务
	fmt.Println("方式4: 使用配置初始化服务")
	fmt.Println("------------------------------")
	
	// 数据库配置
	fmt.Println("数据库配置:")
	fmt.Printf("  DSN: %s\n", maskDSN(cfg.Database.GetDSN()))
	fmt.Printf("  最大连接数: %d\n", cfg.Database.MaxOpenConns)
	fmt.Printf("  最大空闲连接数: %d\n", cfg.Database.MaxIdleConns)
	fmt.Printf("  连接最大生命周期: %v\n", cfg.Database.ConnMaxLifetime)
	fmt.Println()

	// Redis配置
	if cfg.Redis.Enabled {
		fmt.Println("Redis配置:")
		fmt.Printf("  地址: %s:%s\n", cfg.Redis.Host, cfg.Redis.Port)
		fmt.Printf("  数据库: %d\n", cfg.Redis.DB)
		fmt.Printf("  密码: %s\n", maskPassword(cfg.Redis.Password))
	}
	fmt.Println()

	// Genkit配置
	fmt.Println("Genkit配置:")
	fmt.Printf("  API密钥: %s\n", maskAPIKey(cfg.Genkit.APIKey))
	fmt.Printf("  模型: %s\n", cfg.Genkit.Model)
	fmt.Printf("  默认温度: %.1f\n", cfg.Genkit.DefaultTemperature)
	fmt.Printf("  默认最大Token: %d\n", cfg.Genkit.DefaultMaxTokens)
	fmt.Println()

	// 认证配置
	fmt.Println("认证配置:")
	fmt.Printf("  JWT密钥: %s\n", maskAPIKey(cfg.Auth.JWTSecret))
	fmt.Printf("  Access Token TTL: %v\n", cfg.Auth.AccessTokenTTL)
	fmt.Printf("  Refresh Token TTL: %v\n", cfg.Auth.RefreshTokenTTL)
	fmt.Printf("  最大登录尝试: %d\n", cfg.Auth.MaxLoginAttempts)
	fmt.Printf("  密码最小长度: %d\n", cfg.Auth.PasswordMinLength)
	fmt.Println()

	// 会话配置
	fmt.Println("会话配置:")
	fmt.Printf("  超时时间: %v\n", cfg.Session.Timeout)
	fmt.Printf("  清理间隔: %v\n", cfg.Session.CleanupInterval)
	fmt.Printf("  摘要阈值: %d条消息\n", cfg.Session.SummaryThreshold)
	fmt.Printf("  默认分页: %d\n", cfg.Session.DefaultPageSize)
	fmt.Println()

	// 方式5: 配置验证
	fmt.Println("方式5: 配置验证")
	fmt.Println("------------------------------")
	
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("✗ 配置验证失败: %v\n", err)
	} else {
		fmt.Println("✓ 配置验证通过")
	}
	fmt.Println()

	// 方式6: 在应用中使用配置
	fmt.Println("方式6: 在应用中使用配置")
	fmt.Println("------------------------------")
	
	fmt.Println("示例代码:")
	fmt.Println(`
	// 初始化数据库
	db, err := gorm.Open(postgres.Open(cfg.Database.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(getLogLevel(cfg.Database.LogLevel)),
	})
	
	// 配置连接池
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	
	// 初始化Redis
	if cfg.Redis.Enabled {
		rdb := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}
	
	// 初始化Genkit
	genkitClient := genkit.NewClient(cfg.Genkit.APIKey, cfg.Genkit.Model)
	
	// 启动服务器
	router := gin.Default()
	router.Run(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port))
	`)
	
	fmt.Println("\n==================")
	fmt.Println("集成示例完成")
	fmt.Println("==================")
}

// maskDSN 遮蔽DSN中的敏感信息
func maskDSN(dsn string) string {
	if len(dsn) <= 20 {
		return "****"
	}
	return dsn[:10] + "****" + dsn[len(dsn)-10:]
}

// maskPassword 遮蔽密码
func maskPassword(password string) string {
	if password == "" {
		return "(未设置)"
	}
	return "****"
}

// maskAPIKey 遮蔽API密钥
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
