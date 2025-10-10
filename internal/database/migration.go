package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
)

// MigrationManager 迁移管理器
type MigrationManager struct {
	migrate     *migrate.Migrate
	db          *sql.DB
	migrationsPath string
}

// NewMigrationManager 创建新的迁移管理器
func NewMigrationManager(databaseURL, migrationsPath string) (*MigrationManager, error) {
	// 连接数据库
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 创建迁移驱动
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建迁移驱动失败: %w", err)
	}

	// 创建迁移实例
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建迁移实例失败: %w", err)
	}

	return &MigrationManager{
		migrate:        m,
		db:            db,
		migrationsPath: migrationsPath,
	}, nil
}

// Close 关闭迁移管理器
func (mm *MigrationManager) Close() error {
	if mm.db != nil {
		return mm.db.Close()
	}
	return nil
}

// Up 执行所有待执行的迁移
func (mm *MigrationManager) Up() error {
	logrus.Info("开始执行数据库迁移...")
	
	if err := mm.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %w", err)
	}

	if err == migrate.ErrNoChange {
		logrus.Info("数据库已是最新版本，无需迁移")
	} else {
		logrus.Info("数据库迁移完成")
	}
	
	return nil
}

// Down 回滚一个迁移版本
func (mm *MigrationManager) Down() error {
	logrus.Info("开始回滚数据库迁移...")
	
	if err := mm.migrate.Steps(-1); err != nil {
		return fmt.Errorf("回滚迁移失败: %w", err)
	}

	logrus.Info("数据库迁移回滚完成")
	return nil
}

// Migrate 迁移到指定版本
func (mm *MigrationManager) Migrate(version uint) error {
	logrus.Infof("迁移数据库到版本 %d...", version)
	
	if err := mm.migrate.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("迁移到版本 %d 失败: %w", version, err)
	}

	logrus.Infof("数据库已迁移到版本 %d", version)
	return nil
}

// Version 获取当前数据库版本
func (mm *MigrationManager) Version() (uint, bool, error) {
	version, dirty, err := mm.migrate.Version()
	if err != nil {
		return 0, false, fmt.Errorf("获取数据库版本失败: %w", err)
	}
	return version, dirty, nil
}

// Force 强制设置数据库版本（用于修复脏状态）
func (mm *MigrationManager) Force(version int) error {
	logrus.Warnf("强制设置数据库版本为 %d", version)
	
	if err := mm.migrate.Force(version); err != nil {
		return fmt.Errorf("强制设置版本失败: %w", err)
	}

	logrus.Infof("数据库版本已强制设置为 %d", version)
	return nil
}

// Drop 删除所有表（危险操作）
func (mm *MigrationManager) Drop() error {
	logrus.Warn("警告：即将删除所有数据库表")
	
	if err := mm.migrate.Drop(); err != nil {
		return fmt.Errorf("删除数据库表失败: %w", err)
	}

	logrus.Info("所有数据库表已删除")
	return nil
}

// RunMigrations 运行数据库迁移（兼容旧接口）
func RunMigrations(databaseURL string) error {
	mm, err := NewMigrationManager(databaseURL, "migrations")
	if err != nil {
		return err
	}
	defer mm.Close()

	return mm.Up()
}

// InitializeDatabase 初始化数据库（包括迁移和种子数据）
func InitializeDatabase(databaseURL string) error {
	// 运行迁移
	if err := RunMigrations(databaseURL); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 运行种子数据
	if err := SeedDatabase(databaseURL); err != nil {
		return fmt.Errorf("种子数据初始化失败: %w", err)
	}

	return nil
}

// CreateMigration 创建新的迁移文件
func CreateMigration(name string) error {
	migrationsDir := "migrations"
	
	// 确保迁移目录存在
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("创建迁移目录失败: %w", err)
	}

	// 生成时间戳
	timestamp := time.Now().Format("20060102150405")
	
	// 创建up文件
	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.up.sql", timestamp, name))
	if err := createMigrationFile(upFile, fmt.Sprintf("-- %s up migration", name)); err != nil {
		return err
	}

	// 创建down文件
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.down.sql", timestamp, name))
	if err := createMigrationFile(downFile, fmt.Sprintf("-- %s down migration", name)); err != nil {
		return err
	}

	logrus.Infof("迁移文件已创建: %s, %s", upFile, downFile)
	return nil
}

// createMigrationFile 创建迁移文件
func createMigrationFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件 %s 失败: %w", filename, err)
	}
	defer file.Close()

	if _, err := file.WriteString(content + "\n"); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", filename, err)
	}

	return nil
}