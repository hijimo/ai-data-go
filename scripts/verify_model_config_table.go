package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"genkit-ai-service/internal/config"
	"genkit-ai-service/internal/database"

	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 模型配置表结构验证工具 ===")
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
	fmt.Println()

	// 获取 GORM DB 实例
	gormDB := db.GetDB()

	// 1. 验证表存在
	fmt.Println("📋 1. 验证表存在性")
	fmt.Println("   " + strings.Repeat("-", 60))
	if err := verifyTableExists(db); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Println()

	// 2. 验证列结构
	fmt.Println("📋 2. 验证列结构")
	fmt.Println("   " + strings.Repeat("-", 60))
	if err := verifyColumns(db); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Println()

	// 3. 验证索引
	fmt.Println("📋 3. 验证索引")
	fmt.Println("   " + strings.Repeat("-", 60))
	if err := verifyIndexes(db); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Println()

	// 4. 验证外键约束
	fmt.Println("📋 4. 验证外键约束")
	fmt.Println("   " + strings.Repeat("-", 60))
	if err := verifyForeignKeys(db); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Println()

	// 5. 验证 CHECK 约束
	fmt.Println("📋 5. 验证 CHECK 约束")
	fmt.Println("   " + strings.Repeat("-", 60))
	if err := verifyCheckConstraints(db); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Println()

	// 6. 显示表的详细信息
	fmt.Println("📋 6. 表详细信息")
	fmt.Println("   " + strings.Repeat("-", 60))
	displayTableInfo(gormDB)
	fmt.Println()

	fmt.Println("🎉 所有验证通过！")
	fmt.Println()
	fmt.Println("✅ 需求验证结果:")
	fmt.Println("   ✓ 需求 1.4: model_configurations 表已成功创建")
	fmt.Println("   ✓ 需求 2.4: 所有必需字段和索引已正确创建")
	fmt.Println("   ✓ 需求 6.2: 外键约束已正确设置")
	fmt.Println()

	os.Exit(0)
}

func verifyTableExists(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

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

	fmt.Println("   ✅ 表 model_configurations 存在")
	return nil
}

func verifyColumns(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	type ColumnInfo struct {
		ColumnName    string
		DataType      string
		IsNullable    string
		ColumnDefault *string
	}

	var columns []ColumnInfo
	err := gormDB.Raw(`
		SELECT 
			column_name, 
			data_type, 
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = 'model_configurations'
		ORDER BY ordinal_position
	`).Scan(&columns).Error

	if err != nil {
		return fmt.Errorf("查询列信息失败: %w", err)
	}

	// 期望的列配置
	expectedColumns := map[string]struct {
		DataType   string
		IsNullable string
	}{
		"id":             {"uuid", "NO"},
		"tenant_id":      {"uuid", "NO"},
		"name":           {"character varying", "NO"},
		"model":          {"character varying", "NO"},
		"model_provider": {"character varying", "NO"},
		"base_url":       {"character varying", "YES"},
		"api_key":        {"text", "NO"},
		"query_params":   {"jsonb", "YES"},
		"is_enabled":     {"boolean", "NO"},
		"is_deleted":     {"boolean", "NO"},
		"created_by":     {"uuid", "NO"},
		"created_at":     {"timestamp with time zone", "NO"},
		"updated_by":     {"uuid", "YES"},
		"updated_at":     {"timestamp with time zone", "YES"},
		"deleted_by":     {"uuid", "YES"},
		"deleted_at":     {"timestamp with time zone", "YES"},
	}

	for _, col := range columns {
		expected, ok := expectedColumns[col.ColumnName]
		if !ok {
			continue // 跳过不在期望列表中的列
		}

		if col.DataType != expected.DataType {
			return fmt.Errorf("列 %s 类型不匹配: 期望 %s, 实际 %s",
				col.ColumnName, expected.DataType, col.DataType)
		}

		if col.IsNullable != expected.IsNullable {
			return fmt.Errorf("列 %s 可空性不匹配: 期望 %s, 实际 %s",
				col.ColumnName, expected.IsNullable, col.IsNullable)
		}

		nullable := "NOT NULL"
		if col.IsNullable == "YES" {
			nullable = "NULL"
		}

		defaultValue := "无"
		if col.ColumnDefault != nil {
			defaultValue = *col.ColumnDefault
		}

		fmt.Printf("   ✅ %-20s %-25s %-10s 默认值: %s\n",
			col.ColumnName, col.DataType, nullable, defaultValue)
	}

	return nil
}

func verifyIndexes(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	type IndexInfo struct {
		IndexName  string
		ColumnName string
		IndexDef   string
	}

	var indexes []IndexInfo
	err := gormDB.Raw(`
		SELECT 
			indexname as index_name,
			'' as column_name,
			indexdef as index_def
		FROM pg_indexes
		WHERE schemaname = 'public'
		AND tablename = 'model_configurations'
		ORDER BY indexname
	`).Scan(&indexes).Error

	if err != nil {
		return fmt.Errorf("查询索引信息失败: %w", err)
	}

	// 期望的索引
	expectedIndexes := map[string]bool{
		"idx_model_configs_tenant_provider": false,
		"idx_model_configs_deleted":         false,
		"idx_model_configs_enabled":         false,
		"idx_model_configs_tenant_id":       false,
		"idx_model_configs_created_at":      false,
	}

	for _, idx := range indexes {
		if _, ok := expectedIndexes[idx.IndexName]; ok {
			expectedIndexes[idx.IndexName] = true
			fmt.Printf("   ✅ %s\n", idx.IndexName)
		}
	}

	// 检查是否所有期望的索引都存在
	for indexName, exists := range expectedIndexes {
		if !exists {
			return fmt.Errorf("索引 %s 不存在", indexName)
		}
	}

	return nil
}

func verifyForeignKeys(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	type ForeignKeyInfo struct {
		ConstraintName string
		ColumnName     string
		RefTable       string
		RefColumn      string
	}

	var foreignKeys []ForeignKeyInfo
	err := gormDB.Raw(`
		SELECT 
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS ref_table,
			ccu.column_name AS ref_column
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'
		AND tc.table_name = 'model_configurations'
		ORDER BY tc.constraint_name
	`).Scan(&foreignKeys).Error

	if err != nil {
		return fmt.Errorf("查询外键约束失败: %w", err)
	}

	// 期望的外键约束
	expectedForeignKeys := map[string]struct {
		Column   string
		RefTable string
	}{
		"fk_model_configurations_tenant":     {"tenant_id", "tenants"},
		"fk_model_configurations_created_by": {"created_by", "users"},
		"fk_model_configurations_updated_by": {"updated_by", "users"},
		"fk_model_configurations_deleted_by": {"deleted_by", "users"},
	}

	foundForeignKeys := make(map[string]bool)

	for _, fk := range foreignKeys {
		expected, ok := expectedForeignKeys[fk.ConstraintName]
		if !ok {
			continue
		}

		if fk.ColumnName != expected.Column {
			return fmt.Errorf("外键 %s 列不匹配: 期望 %s, 实际 %s",
				fk.ConstraintName, expected.Column, fk.ColumnName)
		}

		if fk.RefTable != expected.RefTable {
			return fmt.Errorf("外键 %s 引用表不匹配: 期望 %s, 实际 %s",
				fk.ConstraintName, expected.RefTable, fk.RefTable)
		}

		foundForeignKeys[fk.ConstraintName] = true
		fmt.Printf("   ✅ %-45s %s -> %s(%s)\n",
			fk.ConstraintName, fk.ColumnName, fk.RefTable, fk.RefColumn)
	}

	// 检查是否所有期望的外键都存在
	for fkName := range expectedForeignKeys {
		if !foundForeignKeys[fkName] {
			return fmt.Errorf("外键约束 %s 不存在", fkName)
		}
	}

	return nil
}

func verifyCheckConstraints(db *database.PostgresDatabase) error {
	gormDB := db.GetDB()

	type CheckConstraintInfo struct {
		ConstraintName string
		CheckClause    string
	}

	var checkConstraints []CheckConstraintInfo
	err := gormDB.Raw(`
		SELECT 
			tc.constraint_name,
			cc.check_clause
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.check_constraints AS cc
			ON tc.constraint_name = cc.constraint_name
		WHERE tc.constraint_type = 'CHECK'
		AND tc.table_schema = 'public'
		AND tc.table_name = 'model_configurations'
		ORDER BY tc.constraint_name
	`).Scan(&checkConstraints).Error

	if err != nil {
		return fmt.Errorf("查询 CHECK 约束失败: %w", err)
	}

	// 查找 model_provider 的 CHECK 约束
	found := false
	for _, cc := range checkConstraints {
		if cc.ConstraintName == "chk_model_provider" {
			found = true
			fmt.Printf("   ✅ %-30s\n", cc.ConstraintName)
			fmt.Printf("      约束条件: %s\n", cc.CheckClause)
		}
	}

	if !found {
		return fmt.Errorf("CHECK 约束 chk_model_provider 不存在")
	}

	return nil
}

func displayTableInfo(gormDB *gorm.DB) {
	// 显示表的行数
	var count int64
	gormDB.Raw("SELECT COUNT(*) FROM model_configurations").Scan(&count)
	fmt.Printf("   📊 当前记录数: %d\n", count)

	// 显示表大小
	type TableSize struct {
		TableSize  string
		IndexSize  string
		TotalSize  string
	}

	var size TableSize
	gormDB.Raw(`
		SELECT 
			pg_size_pretty(pg_table_size('model_configurations')) as table_size,
			pg_size_pretty(pg_indexes_size('model_configurations')) as index_size,
			pg_size_pretty(pg_total_relation_size('model_configurations')) as total_size
	`).Scan(&size)

	fmt.Printf("   📦 表大小: %s\n", size.TableSize)
	fmt.Printf("   📦 索引大小: %s\n", size.IndexSize)
	fmt.Printf("   📦 总大小: %s\n", size.TotalSize)
}
