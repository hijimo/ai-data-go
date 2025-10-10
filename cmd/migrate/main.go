package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"ai-knowledge-platform/internal/config"
	"ai-knowledge-platform/internal/database"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		logrus.Warn("未找到.env文件，使用系统环境变量")
	}

	// 解析命令行参数
	var (
		action = flag.String("action", "up", "迁移操作: up, down, version, force, drop, create, init, seed, clean")
		name   = flag.String("name", "", "迁移名称（用于create操作）")
		ver    = flag.String("version", "", "目标版本（用于force操作）")
	)
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 构建数据库URL
	databaseURL := cfg.Database.URL
	if databaseURL == "" {
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
			cfg.Database.SSLMode,
		)
	}

	// 执行相应的操作
	switch *action {
	case "up":
		if err := runUp(databaseURL); err != nil {
			log.Fatalf("执行迁移失败: %v", err)
		}
	case "down":
		if err := runDown(databaseURL); err != nil {
			log.Fatalf("回滚迁移失败: %v", err)
		}
	case "version":
		if err := showVersion(databaseURL); err != nil {
			log.Fatalf("获取版本失败: %v", err)
		}
	case "force":
		if *ver == "" {
			log.Fatal("force操作需要指定版本号")
		}
		version, err := strconv.Atoi(*ver)
		if err != nil {
			log.Fatalf("无效的版本号: %v", err)
		}
		if err := forceVersion(databaseURL, version); err != nil {
			log.Fatalf("强制设置版本失败: %v", err)
		}
	case "drop":
		if err := dropTables(databaseURL); err != nil {
			log.Fatalf("删除表失败: %v", err)
		}
	case "create":
		if *name == "" {
			log.Fatal("create操作需要指定迁移名称")
		}
		if err := database.CreateMigration(*name); err != nil {
			log.Fatalf("创建迁移文件失败: %v", err)
		}
	case "init":
		if err := database.InitializeDatabase(databaseURL); err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}
	case "seed":
		if err := database.SeedDatabase(databaseURL); err != nil {
			log.Fatalf("初始化种子数据失败: %v", err)
		}
	case "clean":
		if err := database.CleanSeedData(databaseURL); err != nil {
			log.Fatalf("清理种子数据失败: %v", err)
		}
	default:
		log.Fatalf("未知操作: %s", *action)
	}
}

func runUp(databaseURL string) error {
	mm, err := database.NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	return mm.Up()
}

func runDown(databaseURL string) error {
	mm, err := database.NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	return mm.Down()
}

func showVersion(databaseURL string) error {
	mm, err := database.NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	version, dirty, err := mm.Version()
	if err != nil {
		return err
	}

	status := "clean"
	if dirty {
		status = "dirty"
	}

	fmt.Printf("当前数据库版本: %d (%s)\n", version, status)
	return nil
}

func forceVersion(databaseURL string, version int) error {
	mm, err := database.NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	return mm.Force(version)
}

func dropTables(databaseURL string) error {
	fmt.Print("警告：此操作将删除所有数据库表。确认继续？(y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	
	if confirm != "y" && confirm != "Y" {
		fmt.Println("操作已取消")
		return nil
	}

	mm, err := database.NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	return mm.Drop()
}