package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// PlatformTenantMigration 平台租户迁移
type PlatformTenantMigration struct {
	db *gorm.DB
}

// NewPlatformTenantMigration 创建平台租户迁移实例
func NewPlatformTenantMigration(db *gorm.DB) *PlatformTenantMigration {
	return &PlatformTenantMigration{
		db: db,
	}
}

// Up 执行迁移（添加 type 字段和相关约束）
func (m *PlatformTenantMigration) Up() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 添加 type 字段
		sql := `
		-- 添加 type 字段，默认值为 'tenant'
		ALTER TABLE tenants ADD COLUMN IF NOT EXISTS type VARCHAR(32) NOT NULL DEFAULT 'tenant';
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("添加 type 字段失败: %w", err)
		}

		// 2. 为 type 字段添加检查约束
		sql = `
		-- 添加 type 字段的检查约束
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'tenants_type_check'
			) THEN
				ALTER TABLE tenants ADD CONSTRAINT tenants_type_check 
				CHECK (type IN ('system', 'tenant'));
			END IF;
		END $$;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("添加 type 检查约束失败: %w", err)
		}

		// 3. 为 type 字段创建索引
		sql = `
		-- 创建 type 字段索引
		CREATE INDEX IF NOT EXISTS idx_tenants_type ON tenants(type);
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("创建 type 索引失败: %w", err)
		}

		// 4. 添加唯一约束确保只能有一个平台租户
		sql = `
		-- 添加唯一约束确保只能有一个 type='system' 的租户
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_system_tenant 
		ON tenants(type) 
		WHERE type = 'system' AND is_deleted = false;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("添加平台租户唯一约束失败: %w", err)
		}

		// 5. 为 type 字段添加注释
		sql = `
		-- 添加 type 字段注释
		COMMENT ON COLUMN tenants.type IS '租户类型：system=平台租户，tenant=业务租户';
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("添加 type 字段注释失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移（删除 type 字段和相关约束）
func (m *PlatformTenantMigration) Down() error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除唯一约束
		sql := `
		-- 删除平台租户唯一约束
		DROP INDEX IF EXISTS idx_unique_system_tenant;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("删除平台租户唯一约束失败: %w", err)
		}

		// 2. 删除 type 字段索引
		sql = `
		-- 删除 type 字段索引
		DROP INDEX IF EXISTS idx_tenants_type;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("删除 type 索引失败: %w", err)
		}

		// 3. 删除 type 字段的检查约束
		sql = `
		-- 删除 type 字段的检查约束
		ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_type_check;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("删除 type 检查约束失败: %w", err)
		}

		// 4. 删除 type 字段
		sql = `
		-- 删除 type 字段
		ALTER TABLE tenants DROP COLUMN IF EXISTS type;
		`
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("删除 type 字段失败: %w", err)
		}

		return nil
	})
}

// Name 返回迁移名称
func (m *PlatformTenantMigration) Name() string {
	return "platform_tenant_migration"
}

// GetName 获取迁移名称（实现 Migration 接口）
func (m *PlatformTenantMigration) GetName() string {
	return m.Name()
}
