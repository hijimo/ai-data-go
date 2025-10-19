package migrations

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestAuthMigration_Up 测试认证迁移的 Up 方法
func TestAuthMigration_Up(t *testing.T) {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	// 创建迁移实例
	migration := NewAuthMigration(db)

	// 执行迁移
	if err := migration.Up(); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	// 验证表是否创建成功
	tables := []string{"tenants", "users", "refresh_tokens", "auth_audit"}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("表 %s 未创建", table)
		}
	}
}

// TestAuthMigration_GetName 测试获取迁移名称
func TestAuthMigration_GetName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	migration := NewAuthMigration(db)
	name := migration.GetName()

	if name != "auth_migration" {
		t.Errorf("期望迁移名称为 'auth_migration'，实际为 '%s'", name)
	}
}

// TestAuthMigration_UpDown 测试迁移的 Up 和 Down 方法
func TestAuthMigration_UpDown(t *testing.T) {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	migration := NewAuthMigration(db)

	// 执行迁移
	if err := migration.Up(); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	// 验证表存在
	if !db.Migrator().HasTable("tenants") {
		t.Error("表 tenants 未创建")
	}

	// 回滚迁移
	if err := migration.Down(); err != nil {
		t.Fatalf("回滚迁移失败: %v", err)
	}

	// 验证表已删除
	if db.Migrator().HasTable("tenants") {
		t.Error("表 tenants 应该已被删除")
	}
	if db.Migrator().HasTable("users") {
		t.Error("表 users 应该已被删除")
	}
	if db.Migrator().HasTable("refresh_tokens") {
		t.Error("表 refresh_tokens 应该已被删除")
	}
	if db.Migrator().HasTable("auth_audit") {
		t.Error("表 auth_audit 应该已被删除")
	}
}

// TestRunAuthMigrations 测试运行认证迁移函数
func TestRunAuthMigrations(t *testing.T) {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	// 运行认证迁移
	if err := RunAuthMigrations(db); err != nil {
		t.Fatalf("运行认证迁移失败: %v", err)
	}

	// 验证表是否创建成功
	tables := []string{"tenants", "users", "refresh_tokens", "auth_audit"}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("表 %s 未创建", table)
		}
	}
}

// TestRunAllMigrations 测试运行所有迁移函数
func TestRunAllMigrations(t *testing.T) {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	// 运行所有迁移
	if err := RunAllMigrations(db); err != nil {
		t.Fatalf("运行所有迁移失败: %v", err)
	}

	// 验证认证相关表是否创建成功
	authTables := []string{"tenants", "users", "refresh_tokens", "auth_audit"}
	for _, table := range authTables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("认证表 %s 未创建", table)
		}
	}

	// 验证会话相关表是否创建成功
	sessionTables := []string{"chat_sessions", "chat_messages", "chat_summaries"}
	for _, table := range sessionTables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("会话表 %s 未创建", table)
		}
	}
}
