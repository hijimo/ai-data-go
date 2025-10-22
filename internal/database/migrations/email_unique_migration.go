package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// EmailUniqueMigration 邮箱全局唯一性迁移
// 将邮箱从租户内唯一改为全局唯一
type EmailUniqueMigration struct {
	db *gorm.DB
}

// NewEmailUniqueMigration 创建邮箱唯一性迁移实例
func NewEmailUniqueMigration(db *gorm.DB) *EmailUniqueMigration {
	return &EmailUniqueMigration{
		db: db,
	}
}

// Up 执行迁移（将邮箱改为全局唯一）
func (m *EmailUniqueMigration) Up() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除旧的租户+邮箱联合唯一索引
		if err := tx.Exec("DROP INDEX IF EXISTS idx_tenant_email").Error; err != nil {
			return fmt.Errorf("删除旧索引失败: %w", err)
		}

		// 2. 创建新的邮箱全局唯一索引
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email) WHERE NOT is_deleted").Error; err != nil {
			return fmt.Errorf("创建新索引失败: %w", err)
		}

		// 3. 更新表注释
		if err := tx.Exec("COMMENT ON COLUMN users.email IS '用户邮箱地址（全局唯一）'").Error; err != nil {
			return fmt.Errorf("更新列注释失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移（恢复租户内唯一）
func (m *EmailUniqueMigration) Down() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除全局唯一索引
		if err := tx.Exec("DROP INDEX IF EXISTS idx_users_email_unique").Error; err != nil {
			return fmt.Errorf("删除全局唯一索引失败: %w", err)
		}

		// 2. 恢复租户+邮箱联合唯一索引
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_email ON users(tenant_id, email) WHERE NOT is_deleted").Error; err != nil {
			return fmt.Errorf("恢复旧索引失败: %w", err)
		}

		// 3. 恢复表注释
		if err := tx.Exec("COMMENT ON COLUMN users.email IS '用户邮箱地址（租户内唯一）'").Error; err != nil {
			return fmt.Errorf("恢复列注释失败: %w", err)
		}

		return nil
	})
}

// Name 返回迁移名称
func (m *EmailUniqueMigration) Name() string {
	return "email_unique_migration"
}

// GetName 获取迁移名称（实现 Migration 接口）
func (m *EmailUniqueMigration) GetName() string {
	return m.Name()
}

// RunEmailUniqueMigration 执行邮箱全局唯一性迁移
func RunEmailUniqueMigration(db *gorm.DB) error {
	migration := NewEmailUniqueMigration(db)
	return migration.Up()
}
