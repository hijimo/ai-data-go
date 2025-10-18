package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// Migration 迁移接口
type Migration interface {
	// Up 执行迁移
	Up() error
	// Down 回滚迁移
	Down() error
	// GetName 获取迁移名称
	GetName() string
}

// MigrationManager 迁移管理器
type MigrationManager struct {
	db         *gorm.DB
	migrations []Migration
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *gorm.DB) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: make([]Migration, 0),
	}
}

// Register 注册迁移
func (m *MigrationManager) Register(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

// Up 执行所有迁移
func (m *MigrationManager) Up() error {
	for _, migration := range m.migrations {
		if err := migration.Up(); err != nil {
			return fmt.Errorf("迁移 %s 失败: %w", migration.GetName(), err)
		}
	}
	return nil
}

// Down 回滚所有迁移（倒序执行）
func (m *MigrationManager) Down() error {
	// 倒序回滚
	for i := len(m.migrations) - 1; i >= 0; i-- {
		migration := m.migrations[i]
		if err := migration.Down(); err != nil {
			return fmt.Errorf("回滚迁移 %s 失败: %w", migration.GetName(), err)
		}
	}
	return nil
}

// RunSessionMigrations 运行会话管理相关的迁移
func RunSessionMigrations(db *gorm.DB) error {
	manager := NewMigrationManager(db)
	
	// 注册会话管理迁移
	manager.Register(NewSessionMigration(db))
	
	// 执行迁移
	if err := manager.Up(); err != nil {
		return fmt.Errorf("执行会话管理迁移失败: %w", err)
	}
	
	return nil
}
