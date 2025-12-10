package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID           string `gorm:"column:id"`
	Email        string `gorm:"column:email"`
	PasswordHash string `gorm:"column:password_hash"`
	IsActive     bool   `gorm:"column:is_active"`
	IsAdmin      bool   `gorm:"column:is_admin"`
}

func main() {
	// 从环境变量读取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "ai_service")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")

	// 从环境变量读取管理员配置
	adminEmail := getEnv("PLATFORM_ADMIN_EMAIL", "admin@system.local")
	adminPassword := getEnv("PLATFORM_ADMIN_PASSWORD", "Admin123456")

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Printf("✅ 成功连接到数据库\n\n")

	// 查询管理员用户
	var user User
	result := db.Table("users").Where("email = ?", adminEmail).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			fmt.Printf("❌ 未找到邮箱为 '%s' 的用户\n", adminEmail)
			fmt.Println("\n可能的原因:")
			fmt.Println("1. 系统尚未初始化")
			fmt.Println("2. 管理员邮箱配置错误")
			return
		}
		log.Fatalf("查询用户失败: %v", result.Error)
	}

	fmt.Printf("📧 邮箱: %s\n", user.Email)
	fmt.Printf("🔑 密码哈希: %s\n", user.PasswordHash[:50]+"...")
	fmt.Printf("✓ 账户激活: %v\n", user.IsActive)
	fmt.Printf("👑 管理员: %v\n", user.IsAdmin)
	fmt.Println()

	// 验证密码
	fmt.Printf("正在验证密码 '%s'...\n", adminPassword)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(adminPassword))

	if err != nil {
		fmt.Printf("\n❌ 密码验证失败!\n\n")
		fmt.Println("问题诊断:")
		fmt.Println("1. 数据库中存储的密码哈希与配置的密码不匹配")
		fmt.Println("2. 可能的原因:")
		fmt.Println("   - 系统初始化时使用了不同的密码")
		fmt.Println("   - 密码已被修改")
		fmt.Println("   - .env 文件中的密码配置错误")
		fmt.Println()
		fmt.Println("解决方案:")
		fmt.Println("1. 检查系统启动日志，查看初始化时生成的密码")
		fmt.Println("2. 使用密码重置功能")
		fmt.Println("3. 重新初始化系统（需要清空数据库）")
	} else {
		fmt.Printf("\n✅ 密码验证成功!\n")
		fmt.Println("密码配置正确，可以正常登录。")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
