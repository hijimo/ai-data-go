package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// AddCreatedByNameMigration 添加 created_by_name 字段的迁移
type AddCreatedByNameMigration struct {
	db *gorm.DB
}

// NewAddCreatedByNameMigration 创建迁移实例
func NewAddCreatedByNameMigration(db *gorm.DB) *AddCreatedByNameMigration {
	return &AddCreatedByNameMigration{db: db}
}

// Up 执行迁移（添加 created_by_name 字段）
func (m *AddCreatedByNameMigration) Up() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 为 tenants 表添加 created_by_name 字段
		if err := m.addCreatedByNameToTenants(tx); err != nil {
			return fmt.Errorf("为 tenants 表添加 created_by_name 字段失败: %w", err)
		}

		// 2. 为 users 表添加 created_by_name 字段
		if err := m.addCreatedByNameToUsers(tx); err != nil {
			return fmt.Errorf("为 users 表添加 created_by_name 字段失败: %w", err)
		}

		// 3. 为 chat_sessions 表添加 created_by_name 字段
		if err := m.addCreatedByNameToChatSessions(tx); err != nil {
			return fmt.Errorf("为 chat_sessions 表添加 created_by_name 字段失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移（删除 created_by_name 字段）
func (m *AddCreatedByNameMigration) Down() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 按逆序删除字段
		sql := `
		-- 删除 chat_sessions 表的 created_by_name 字段
		ALTER TABLE chat_sessions DROP COLUMN IF EXISTS created_by_name;

		-- 删除 users 表的 created_by_name 字段
		ALTER TABLE users DROP COLUMN IF EXISTS created_by_name;

		-- 删除 tenants 表的 created_by_name 字段
		ALTER TABLE tenants DROP COLUMN IF EXISTS created_by_name;
		`

		return tx.Exec(sql).Error
	})
}

// GetName 返回迁移名称
func (m *AddCreatedByNameMigration) GetName() string {
	return "add_created_by_name_migration"
}

// addCreatedByNameToTenants 为 tenants 表添加 created_by_name 字段
func (m *AddCreatedByNameMigration) addCreatedByNameToTenants(tx *gorm.DB) error {
	sql := `
	-- 为 tenants 表添加 created_by_name 字段
	ALTER TABLE tenants 
	ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255);

	-- 添加字段注释
	COMMENT ON COLUMN tenants.created_by_name IS '创建者显示名称';
	`

	return tx.Exec(sql).Error
}

// addCreatedByNameToUsers 为 users 表添加 created_by_name 字段
func (m *AddCreatedByNameMigration) addCreatedByNameToUsers(tx *gorm.DB) error {
	sql := `
	-- 为 users 表添加 created_by_name 字段
	ALTER TABLE users 
	ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255);

	-- 添加字段注释
	COMMENT ON COLUMN users.created_by_name IS '创建者显示名称';
	`

	return tx.Exec(sql).Error
}

// addCreatedByNameToChatSessions 为 chat_sessions 表添加 created_by_name 字段
func (m *AddCreatedByNameMigration) addCreatedByNameToChatSessions(tx *gorm.DB) error {
	sql := `
	-- 为 chat_sessions 表添加 created_by_name 字段
	ALTER TABLE chat_sessions 
	ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(255);

	-- 添加字段注释
	COMMENT ON COLUMN chat_sessions.created_by_name IS '创建者显示名称';
	`

	return tx.Exec(sql).Error
}
