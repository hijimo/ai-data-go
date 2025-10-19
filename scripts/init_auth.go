package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"
	"genkit-ai-service/pkg/crypto"

	"gorm.io/datatypes"
)

const (
	// 默认租户信息
	defaultTenantName   = "默认租户"
	defaultTenantDomain = "default"

	// 默认管理员信息
	defaultAdminEmail       = "admin@example.com"
	defaultAdminPassword    = "Admin@123456"
	defaultAdminDisplayName = "系统管理员"
)

func main() {
	fmt.Println("=== 认证系统初始化工具 ===")
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

	// 创建 Repository
	tenantRepo := repository.NewTenantRepository(db.GetDB())
	userRepo := repository.NewUserRepository(db.GetDB())

	// 检查是否已存在租户
	fmt.Println("🔍 检查现有租户...")
	tenants, _, err := tenantRepo.List(ctx, 1, 1)
	if err != nil {
		log.Fatalf("❌ 查询租户失败: %v", err)
	}

	var tenant *model.Tenant
	if len(tenants) > 0 {
		fmt.Println("⚠️  已存在租户，跳过租户创建")
		tenant = tenants[0]
	} else {
		// 创建默认租户
		fmt.Println("📝 创建默认租户...")
		tenant = &model.Tenant{
			Name:     defaultTenantName,
			Domain:   defaultTenantDomain,
			Metadata: datatypes.JSON([]byte(`{"description":"系统默认租户"}`)),
			Status:   true,
		}

		if err := tenantRepo.Create(ctx, tenant); err != nil {
			log.Fatalf("❌ 创建租户失败: %v", err)
		}

		fmt.Printf("✅ 租户创建成功 (ID: %s)\n", tenant.ID)
	}

	fmt.Println()

	// 检查是否已存在管理员用户
	fmt.Println("🔍 检查现有管理员用户...")
	existingAdmin, err := userRepo.GetByEmail(ctx, tenant.ID, defaultAdminEmail)
	if err == nil && existingAdmin != nil {
		fmt.Println("⚠️  管理员用户已存在，跳过用户创建")
		fmt.Printf("   邮箱: %s\n", existingAdmin.Email)
		fmt.Printf("   用户ID: %s\n", existingAdmin.ID)
		fmt.Println()
		fmt.Println("💡 提示：如需重置密码，请使用密码修改功能")
		os.Exit(0)
	}

	// 创建管理员用户
	fmt.Println("📝 创建管理员用户...")

	// 哈希密码
	passwordHash, err := crypto.HashPassword(defaultAdminPassword)
	if err != nil {
		log.Fatalf("❌ 密码哈希失败: %v", err)
	}

	// 创建管理员角色数组
	roles := datatypes.JSON([]byte(`["admin","user"]`))

	admin := &model.User{
		TenantID:      tenant.ID,
		Email:         defaultAdminEmail,
		EmailVerified: true,
		PasswordHash:  passwordHash,
		DisplayName:   defaultAdminDisplayName,
		IsActive:      true,
		IsAdmin:       true,
		Roles:         roles,
		Meta:          datatypes.JSON([]byte(`{"created_by":"init_script"}`)),
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		log.Fatalf("❌ 创建管理员用户失败: %v", err)
	}

	fmt.Printf("✅ 管理员用户创建成功 (ID: %s)\n", admin.ID)
	fmt.Println()

	// 显示初始化信息
	fmt.Println("=" + string(make([]byte, 50)) + "=")
	fmt.Println("🎉 认证系统初始化完成！")
	fmt.Println("=" + string(make([]byte, 50)) + "=")
	fmt.Println()
	fmt.Println("📋 初始化信息：")
	fmt.Println()
	fmt.Println("租户信息：")
	fmt.Printf("  - 租户ID: %s\n", tenant.ID)
	fmt.Printf("  - 租户名称: %s\n", tenant.Name)
	fmt.Printf("  - 租户域名: %s\n", tenant.Domain)
	fmt.Println()
	fmt.Println("管理员账户：")
	fmt.Printf("  - 用户ID: %s\n", admin.ID)
	fmt.Printf("  - 邮箱: %s\n", admin.Email)
	fmt.Printf("  - 密码: %s\n", defaultAdminPassword)
	fmt.Printf("  - 角色: admin, user\n")
	fmt.Println()
	fmt.Println("⚠️  安全提示：")
	fmt.Println("  1. 请立即登录并修改默认密码")
	fmt.Println("  2. 不要在生产环境使用默认密码")
	fmt.Println("  3. 建议启用多因素认证（MFA）")
	fmt.Println()
	fmt.Println("📖 使用说明：")
	fmt.Println("  - 查看完整文档: docs/AUTH_SETUP.md")
	fmt.Println("  - API 文档: http://localhost:8080/swagger/index.html")
	fmt.Println()

	os.Exit(0)
}
