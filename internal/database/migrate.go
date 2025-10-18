package database

import (
	"fmt"

	"genkit-ai-service/internal/database/migrations"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ParseLogLevel 解析日志级别字符串为 GORM 日志级别
func ParseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// Migrator 数据库迁移器
type Migrator struct {
	db Database
}

// NewMigrator 创建新的迁移器
func NewMigrator(db Database) *Migrator {
	return &Migrator{
		db: db,
	}
}

// Migrate 执行数据库迁移
// models 参数接收需要迁移的模型结构体
func (m *Migrator) Migrate(models ...interface{}) error {
	if len(models) == 0 {
		return fmt.Errorf("没有提供需要迁移的模型")
	}

	if err := m.db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("执行数据库迁移失败: %w", err)
	}

	return nil
}

// RunSessionMigrations 执行会话管理相关的数据库迁移
// 这个函数会创建 chat_sessions、chat_messages 和 chat_summaries 表及其索引
func RunSessionMigrations(db *gorm.DB) error {
	return migrations.RunSessionMigrations(db)
}
