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
	fmt.Println("=== 模型配置模块数据库迁移工具 ===")
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ 数据库连接成功")
	fmt.Printf("   数据库: %s@%s:%d/%s\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Println()

	// 获取 GORM DB 实例
	gormDB := db.GetDB()

	// 执行模型配置迁移
	fmt.Println("📦 开始执行模型配置迁移...")
	migration := migrations.NewModelConfigurationMigration(gormDB)

	if err := migration.Up(); err != nil {
		log.Fatalf("❌ 迁移失败: %v", err)
	}

	fmt.Println("✅ 模型配置迁移执行成功")
	fmt.Println()

	// 记录迁移状态
	if err := migrations.RecordMigrationStatus(gormDB, migration.GetName()); err != nil {
		fmt.Printf("⚠️  警告: 记录迁移状态失败: %v\n", err)
	} else {
		fmt.Println("✅ 迁移状态已记录")
	}
	fmt.Println()

	// 验证表结构
	fmt.Println("🔍 验证表结构...")
	if err := verifyTableStructure(db); err != nil {
		log.Fatalf("❌ 表结构验证失败: %v", err)
	}

	fmt.Println("✅ 表结构验证通过")
	fmt.Println()

	// 验证索引
	fmt.Println("🔍 验证索引...")
	if err := verifyIndexes(db); err != nil {
		log.Fatalf("❌ 索引验证失败: %v", err)
	}

	fmt.Println("✅ 索引验证通过")
	fmt.Println()

	// 验证外键约束
	fmt.Println("🔍 验证外键约束...")
	if err := verifyForeignKeys(db); err != nil {
		log.Fatalf("❌ 外键约束验证失败: %v", err)
	}

	fmt.Println("✅ 外键约束验证通过")
	fmt.Println()

	fmt.Println("🎉 所有验证通过！模型配置模块迁移完成！")
	os.Exit(0)
}

// verifyTableStructure 验证表结构
func verifyTableStructure(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	// 检查表是否存在
	var exists bool
	err := gormDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'model_configurations'
		)
	`).Scan(&exists).Error

	if err != nil {
		return fmt.Errorf("检查表是否存在失败: %w", err)
	}

	if !exists {
		return fmt.Errorf("表 model_configurations 不存在")
	}

	fmt.Println("   ✓ 表 model_configurations 存在")

	// 验证必需的列
	requiredColumns := []string{
		"id", "tenant_id", "name", "model", "model_provider",
		"base_url", "api_key", "query_params", "is_enabled", "is_deleted",
		"created_by", "created_at", "updated_by", "updated_at",
		"deleted_by", "deleted_at",
	}

	for _, column := range requiredColumns {
		var columnExists bool
		err := gormDB.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.columns 
				WHERE table_schema = 'public' 
				AND table_name = 'model_configurations'
				AND column_name = ?
			)
		`, column).Scan(&columnExists).Error

		if err != nil {
			return fmt.Errorf("检查列 %s 失败: %w", column, err)
		}

		if !columnExists {
			return fmt.Errorf("列 %s 不存在", column)
		}

		fmt.Printf("   ✓ 列 %s 存在\n", column)
	}

	// 验证列类型
	type ColumnInfo struct {
		ColumnName string
		DataType   string
		IsNullable string
	}

	var columns []ColumnInfo
	err = gormDB.Raw(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = 'model_configurations'
		ORDER BY ordinal_position
	`).Scan(&columns).Error

	if err != nil {
		return fmt.Errorf("查询列信息失败: %w", err)
	}

	// 验证关键列的类型
	expectedTypes := map[string]string{
		"id":             "uuid",
		"tenant_id":      "uuid",
		"name":           "character varying",
		"model":          "character varying",
		"model_provider": "character varying",
		"api_key":        "text",
		"query_params":   "jsonb",
		"is_enabled":     "boolean",
		"is_deleted":     "boolean",
		"created_by":     "uuid",
		"created_at":     "timestamp with time zone",
	}

	for columnName, expectedType := range expectedTypes {
		found := false
		for _, col := range columns {
			if col.ColumnName == columnName {
				found = true
				if col.DataType != expectedType {
					return fmt.Errorf("列 %s 类型不匹配: 期望 %s, 实际 %s", columnName, expectedType, col.DataType)
				}
				fmt.Printf("   ✓ 列 %s 类型正确 (%s)\n", columnName, expectedType)
				break
			}
		}
		if !found {
			return fmt.Errorf("列 %s 不存在", columnName)
		}
	}

	return nil
}

// verifyIndexes 验证索引
func verifyIndexes(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	// 期望的索引列表
	expectedIndexes := []string{
		"idx_model_configs_tenant_provider",
		"idx_model_configs_deleted",
		"idx_model_configs_enabled",
		"idx_model_configs_tenant_id",
		"idx_model_configs_created_at",
	}

	for _, indexName := range expectedIndexes {
		var exists bool
		err := gormDB.Raw(`
			SELECT EXISTS (
				SELECT FROM pg_indexes
				WHERE schemaname = 'public'
				AND tablename = 'model_configurations'
				AND indexname = ?
			)
		`, indexName).Scan(&exists).Error

		if err != nil {
			return fmt.Errorf("检查索引 %s 失败: %w", indexName, err)
		}

		if !exists {
			return fmt.Errorf("索引 %s 不存在", indexName)
		}

		fmt.Printf("   ✓ 索引 %s 存在\n", indexName)
	}

	return nil
}

// verifyForeignKeys 验证外键约束
func verifyForeignKeys(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	// 期望的外键约束
	expectedForeignKeys := []string{
		"fk_model_configurations_tenant",
		"fk_model_configurations_created_by",
		"fk_model_configurations_updated_by",
		"fk_model_configurations_deleted_by",
	}

	for _, fkName := range expectedForeignKeys {
		var exists bool
		err := gormDB.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.table_constraints
				WHERE constraint_schema = 'public'
				AND table_name = 'model_configurations'
				AND constraint_name = ?
				AND constraint_type = 'FOREIGN KEY'
			)
		`, fkName).Scan(&exists).Error

		if err != nil {
			return fmt.Errorf("检查外键约束 %s 失败: %w", fkName, err)
		}

		if !exists {
			return fmt.Errorf("外键约束 %s 不存在", fkName)
		}

		fmt.Printf("   ✓ 外键约束 %s 存在\n", fkName)
	}

	// 验证 CHECK 约束
	var checkExists bool
	err := gormDB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.table_constraints
			WHERE constraint_schema = 'public'
			AND table_name = 'model_configurations'
			AND constraint_name = 'chk_model_provider'
			AND constraint_type = 'CHECK'
		)
	`).Scan(&checkExists).Error

	if err != nil {
		return fmt.Errorf("检查 CHECK 约束失败: %w", err)
	}

	if !checkExists {
		return fmt.Errorf("CHECK 约束 chk_model_provider 不存在")
	}

	fmt.Println("   ✓ CHECK 约束 chk_model_provider 存在")

	return nil
}
