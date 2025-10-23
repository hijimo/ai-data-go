package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// FixTimestampsMigration 修复时间戳字段的默认值
type FixTimestampsMigration struct {
	db *gorm.DB
}

// NewFixTimestampsMigration 创建时间戳修复迁移实例
func NewFixTimestampsMigration(db *gorm.DB) *FixTimestampsMigration {
	return &FixTimestampsMigration{db: db}
}

// Up 执行迁移
func (m *FixTimestampsMigration) Up() error {
	// 修复 tenants 表的时间戳字段
	if err := m.fixTenantsTimestamps(); err != nil {
		return fmt.Errorf("修复 tenants 表时间戳失败: %w", err)
	}

	// 修复 users 表的时间戳字段
	if err := m.fixUsersTimestamps(); err != nil {
		return fmt.Errorf("修复 users 表时间戳失败: %w", err)
	}

	return nil
}

// Down 回滚迁移
func (m *FixTimestampsMigration) Down() error {
	// 移除默认值
	sql := `
	-- 移除 tenants 表的时间戳默认值
	ALTER TABLE tenants 
		ALTER COLUMN created_at DROP DEFAULT,
		ALTER COLUMN updated_at DROP DEFAULT;

	-- 移除 users 表的时间戳默认值
	ALTER TABLE users 
		ALTER COLUMN created_at DROP DEFAULT,
		ALTER COLUMN updated_at DROP DEFAULT;
	`

	return m.db.Exec(sql).Error
}

// fixTenantsTimestamps 修复 tenants 表的时间戳字段
func (m *FixTimestampsMigration) fixTenantsTimestamps() error {
	sql := `
	-- 为 tenants 表的时间戳字段添加默认值
	ALTER TABLE tenants 
		ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP,
		ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

	-- 为已存在但时间戳为空的记录设置当前时间
	UPDATE tenants 
	SET created_at = CURRENT_TIMESTAMP 
	WHERE created_at IS NULL;

	UPDATE tenants 
	SET updated_at = CURRENT_TIMESTAMP 
	WHERE updated_at IS NULL;
	`

	return m.db.Exec(sql).Error
}

// fixUsersTimestamps 修复 users 表的时间戳字段
func (m *FixTimestampsMigration) fixUsersTimestamps() error {
	sql := `
	-- 为 users 表的时间戳字段添加默认值
	ALTER TABLE users 
		ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP,
		ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

	-- 为已存在但时间戳为空的记录设置当前时间
	UPDATE users 
	SET created_at = CURRENT_TIMESTAMP 
	WHERE created_at IS NULL;

	UPDATE users 
	SET updated_at = CURRENT_TIMESTAMP 
	WHERE updated_at IS NULL;
	`

	return m.db.Exec(sql).Error
}

// GetName 返回迁移名称
func (m *FixTimestampsMigration) GetName() string {
	return "fix_timestamps_migration"
}
