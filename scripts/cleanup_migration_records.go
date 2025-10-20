package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"

	"gorm.io/gorm"
)

// 常见的迁移记录表名
var commonMigrationTables = []string{
	"schema_migrations",
	"migrations",
	"gorm_migrations",
	"db_migrations",
	"_migrations",
}

func main() {
	// 命令行参数
	dryRun := flag.Bool("dry-run", false, "仅检查不执行删除操作")
	force := flag.Bool("force", false, "强制删除，不需要确认")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建数据库连接
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

	db := database.NewPostgresDatabase(dbConfig)
	ctx := context.Background()

	// 连接数据库
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	gormDB := db.GetDB()

	fmt.Println("=== 数据库迁移记录清理工具 ===")
	fmt.Printf("数据库: %s@%s:%s/%s\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Println()

	// 检查是否存在迁移记录表
	foundTables := checkMigrationTables(gormDB)

	if len(foundTables) == 0 {
		fmt.Println("✓ 未发现任何迁移记录表")
		fmt.Println("数据库状态良好，无需清理")
		return
	}

	// 显示发现的表
	fmt.Printf("发现 %d 个迁移记录表:\n", len(foundTables))
	for _, table := range foundTables {
		fmt.Printf("  - %s\n", table)
		
		// 显示表的记录数
		var count int64
		if err := gormDB.Table(table).Count(&count).Error; err != nil {
			fmt.Printf("    警告: 无法获取记录数: %v\n", err)
		} else {
			fmt.Printf("    记录数: %d\n", count)
		}
	}
	fmt.Println()

	// 如果是 dry-run 模式，只显示信息不执行删除
	if *dryRun {
		fmt.Println("--- Dry Run 模式 ---")
		fmt.Println("以下表将被删除（实际未执行）:")
		for _, table := range foundTables {
			fmt.Printf("  DROP TABLE IF EXISTS %s CASCADE;\n", table)
		}
		fmt.Println()
		fmt.Println("提示: 移除 --dry-run 参数以执行实际删除操作")
		return
	}

	// 确认删除操作
	if !*force {
		fmt.Println("警告: 此操作将删除上述迁移记录表")
		fmt.Println("这些表通常用于跟踪数据库迁移历史")
		fmt.Println("删除后将无法回滚到之前的迁移状态")
		fmt.Println()
		fmt.Print("确认删除这些表吗? (yes/no): ")
		
		var confirm string
		fmt.Scanln(&confirm)
		
		if confirm != "yes" && confirm != "y" && confirm != "YES" && confirm != "Y" {
			fmt.Println("操作已取消")
			return
		}
	}

	// 执行删除操作
	fmt.Println()
	fmt.Println("开始清理迁移记录表...")
	
	successCount := 0
	failCount := 0
	
	for _, table := range foundTables {
		fmt.Printf("删除表 %s... ", table)
		
		if err := dropTable(gormDB, table); err != nil {
			fmt.Printf("失败: %v\n", err)
			failCount++
		} else {
			fmt.Println("成功")
			successCount++
		}
	}

	// 显示结果
	fmt.Println()
	fmt.Println("=== 清理完成 ===")
	fmt.Printf("成功删除: %d 个表\n", successCount)
	if failCount > 0 {
		fmt.Printf("删除失败: %d 个表\n", failCount)
	}
	
	// 验证清理结果
	fmt.Println()
	fmt.Println("验证清理结果...")
	remainingTables := checkMigrationTables(gormDB)
	if len(remainingTables) == 0 {
		fmt.Println("✓ 所有迁移记录表已成功清理")
	} else {
		fmt.Printf("⚠ 仍有 %d 个表未能删除:\n", len(remainingTables))
		for _, table := range remainingTables {
			fmt.Printf("  - %s\n", table)
		}
	}
}

// checkMigrationTables 检查数据库中是否存在迁移记录表
func checkMigrationTables(db *gorm.DB) []string {
	var foundTables []string
	
	for _, tableName := range commonMigrationTables {
		if db.Migrator().HasTable(tableName) {
			foundTables = append(foundTables, tableName)
		}
	}
	
	return foundTables
}

// dropTable 删除指定的表
func dropTable(db *gorm.DB, tableName string) error {
	// 使用 CASCADE 确保删除依赖关系
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
	
	// 执行删除操作
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("执行 SQL 失败: %w", err)
	}
	
	// 等待一小段时间确保操作完成
	time.Sleep(100 * time.Millisecond)
	
	// 验证表是否已删除
	if db.Migrator().HasTable(tableName) {
		return fmt.Errorf("表仍然存在")
	}
	
	return nil
}
