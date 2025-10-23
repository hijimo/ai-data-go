package migrations

import (
	"fmt"
	"time"

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

// RegisterInitialMigration 注册初始迁移
// 确保初始迁移在所有迁移之前执行
func (m *MigrationManager) RegisterInitialMigration() {
	// 将初始迁移插入到迁移列表的最前面
	initialMigration := NewInitialMigration(m.db)
	m.migrations = append([]Migration{initialMigration}, m.migrations...)
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
// 注意：此函数已被 RunInitialMigration 替代，保留用于向后兼容
func RunSessionMigrations(db *gorm.DB) error {
	// 直接使用初始迁移，因为它包含了所有表的创建
	return RunInitialMigration(db)
}

// RunAuthMigrations 运行认证相关的迁移
// 注意：此函数已被 RunInitialMigration 替代，保留用于向后兼容
func RunAuthMigrations(db *gorm.DB) error {
	// 直接使用初始迁移，因为它包含了所有表的创建
	return RunInitialMigration(db)
}

// RunInitialMigration 单独执行初始迁移
// 该函数用于在新环境中快速建立完整的数据库结构
func RunInitialMigration(db *gorm.DB) error {
	// 检查是否需要执行迁移
	needsMigration, err := CheckMigrationNeeded(db)
	if err != nil {
		return fmt.Errorf("检查迁移状态失败: %w", err)
	}
	
	if !needsMigration {
		fmt.Println("数据库已完成初始迁移，跳过迁移步骤")
		return nil
	}
	
	// 创建初始迁移实例
	migration := NewInitialMigration(db)
	
	// 记录开始执行
	fmt.Printf("开始执行初始迁移: %s\n", migration.GetName())
	
	// 执行迁移
	if err := migration.Up(); err != nil {
		// 详细的错误处理和日志记录
		return fmt.Errorf("执行初始迁移 %s 失败: %w", migration.GetName(), err)
	}
	
	// 记录迁移状态
	if err := RecordMigrationStatus(db, migration.GetName()); err != nil {
		// 迁移成功但记录状态失败，记录警告但不返回错误
		fmt.Printf("警告: 记录迁移状态失败: %v\n", err)
	}
	
	// 记录成功完成
	fmt.Printf("初始迁移 %s 执行成功\n", migration.GetName())
	
	return nil
}

// RunAllMigrations 运行所有迁移
// 确保初始迁移首先执行，然后按顺序执行其他迁移
func RunAllMigrations(db *gorm.DB) error {
	manager := NewMigrationManager(db)
	
	// 1. 首先注册并执行初始迁移
	manager.RegisterInitialMigration()
	
	// 2. 按顺序注册其他迁移
	// 注意：由于初始迁移已经包含了所有表的创建，
	// 这里的认证迁移和会话迁移可能需要根据实际情况调整或移除
	// manager.Register(NewAuthMigration(db))
	// manager.Register(NewSessionMigration(db))
	
	// 3. 注册时间戳修复迁移
	manager.Register(NewFixTimestampsMigration(db))
	
	// 4. 注册添加 created_by_name 字段的迁移
	manager.Register(NewAddCreatedByNameMigration(db))
	
	// 验证迁移顺序
	if err := validateMigrationOrder(manager.migrations); err != nil {
		return fmt.Errorf("迁移顺序验证失败: %w", err)
	}
	
	// 执行迁移
	if err := manager.Up(); err != nil {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	
	return nil
}

// validateMigrationOrder 验证迁移顺序
// 确保初始迁移在所有迁移之前
func validateMigrationOrder(migrations []Migration) error {
	if len(migrations) == 0 {
		return fmt.Errorf("没有注册任何迁移")
	}
	
	// 检查第一个迁移是否为初始迁移
	firstMigration := migrations[0]
	if firstMigration.GetName() != "initial_migration" {
		return fmt.Errorf("初始迁移必须是第一个执行的迁移，当前第一个迁移为: %s", firstMigration.GetName())
	}
	
	return nil
}

// MigrationRecord 迁移记录模型
type MigrationRecord struct {
	ID          uint      `gorm:"primarykey"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExecutedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Description string    `gorm:"type:text"`
}

// TableName 指定表名
func (MigrationRecord) TableName() string {
	return "schema_migrations"
}

// CheckMigrationNeeded 检查是否需要执行迁移
// 通过检查关键表是否存在来判断
func CheckMigrationNeeded(db *gorm.DB) (bool, error) {
	// 检查 tenants 表是否存在
	// 如果 tenants 表存在，说明数据库已经初始化
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'tenants'
		)
	`).Scan(&exists).Error
	
	if err != nil {
		return false, fmt.Errorf("检查表是否存在失败: %w", err)
	}
	
	// 如果表不存在，需要执行迁移
	return !exists, nil
}

// RecordMigrationStatus 记录迁移状态
func RecordMigrationStatus(db *gorm.DB, migrationName string) error {
	// 确保迁移记录表存在
	if err := db.AutoMigrate(&MigrationRecord{}); err != nil {
		return fmt.Errorf("创建迁移记录表失败: %w", err)
	}
	
	// 检查是否已经记录过
	var count int64
	if err := db.Model(&MigrationRecord{}).Where("name = ?", migrationName).Count(&count).Error; err != nil {
		return fmt.Errorf("检查迁移记录失败: %w", err)
	}
	
	// 如果已经记录过，跳过
	if count > 0 {
		return nil
	}
	
	// 创建迁移记录
	record := &MigrationRecord{
		Name:        migrationName,
		ExecutedAt:  time.Now(),
		Description: "初始数据库迁移，创建所有基础表",
	}
	
	if err := db.Create(record).Error; err != nil {
		return fmt.Errorf("创建迁移记录失败: %w", err)
	}
	
	return nil
}

// GetMigrationHistory 获取迁移历史
func GetMigrationHistory(db *gorm.DB) ([]MigrationRecord, error) {
	var records []MigrationRecord
	
	// 确保迁移记录表存在
	if err := db.AutoMigrate(&MigrationRecord{}); err != nil {
		return nil, fmt.Errorf("创建迁移记录表失败: %w", err)
	}
	
	if err := db.Order("executed_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("查询迁移历史失败: %w", err)
	}
	
	return records, nil
}
