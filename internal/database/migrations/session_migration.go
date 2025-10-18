package migrations

import (
	"fmt"

	"genkit-ai-service/internal/model"

	"gorm.io/gorm"
)

// SessionMigration 会话管理相关表的迁移
type SessionMigration struct {
	db *gorm.DB
}

// NewSessionMigration 创建会话迁移实例
func NewSessionMigration(db *gorm.DB) *SessionMigration {
	return &SessionMigration{
		db: db,
	}
}

// Up 执行迁移（创建表和索引）
func (m *SessionMigration) Up() error {
	// 自动迁移表结构
	if err := m.db.AutoMigrate(
		&model.ChatSession{},
		&model.ChatMessage{},
		&model.ChatSummary{},
	); err != nil {
		return fmt.Errorf("自动迁移表结构失败: %w", err)
	}

	// 创建复合索引
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	return nil
}

// Down 回滚迁移（删除表）
func (m *SessionMigration) Down() error {
	// 删除表（按依赖关系倒序删除）
	if err := m.db.Migrator().DropTable(
		&model.ChatSummary{},
		&model.ChatMessage{},
		&model.ChatSession{},
	); err != nil {
		return fmt.Errorf("删除表失败: %w", err)
	}

	return nil
}

// createIndexes 创建额外的索引
func (m *SessionMigration) createIndexes() error {
	// ChatSession 表索引
	// idx_user_sessions: 用于用户会话列表查询
	if !m.db.Migrator().HasIndex(&model.ChatSession{}, "idx_user_sessions") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_user_sessions 
			ON chat_sessions(user_id, updated_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_user_sessions 索引失败: %w", err)
		}
	}

	// idx_pinned: 用于置顶会话排序
	if !m.db.Migrator().HasIndex(&model.ChatSession{}, "idx_pinned") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_pinned 
			ON chat_sessions(is_pinned, updated_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_pinned 索引失败: %w", err)
		}
	}

	// idx_archived: 用于归档状态过滤
	if !m.db.Migrator().HasIndex(&model.ChatSession{}, "idx_archived") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_archived 
			ON chat_sessions(is_archived)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_archived 索引失败: %w", err)
		}
	}

	// idx_deleted: 用于软删除过滤
	if !m.db.Migrator().HasIndex(&model.ChatSession{}, "idx_deleted") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_deleted 
			ON chat_sessions(is_deleted)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_deleted 索引失败: %w", err)
		}
	}

	// ChatMessage 表索引
	// idx_session_messages: 用于会话消息查询
	if !m.db.Migrator().HasIndex(&model.ChatMessage{}, "idx_session_messages") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_session_messages 
			ON chat_messages(session_id, sequence ASC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_session_messages 索引失败: %w", err)
		}
	}

	// idx_created: 用于时间排序
	if !m.db.Migrator().HasIndex(&model.ChatMessage{}, "idx_created") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_created 
			ON chat_messages(created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_created 索引失败: %w", err)
		}
	}

	// ChatSummary 表索引
	// idx_session_summary: 用于会话摘要查询
	if !m.db.Migrator().HasIndex(&model.ChatSummary{}, "idx_session_summary") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_session_summary 
			ON chat_summaries(session_id, created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_session_summary 索引失败: %w", err)
		}
	}

	return nil
}

// GetName 获取迁移名称
func (m *SessionMigration) GetName() string {
	return "session_migration"
}
