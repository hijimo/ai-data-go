package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// GenkitSessionManagementMigration Genkit会话管理模块迁移
type GenkitSessionManagementMigration struct {
	db *gorm.DB
}

// NewGenkitSessionManagementMigration 创建Genkit会话管理迁移实例
func NewGenkitSessionManagementMigration(db *gorm.DB) *GenkitSessionManagementMigration {
	return &GenkitSessionManagementMigration{
		db: db,
	}
}

// Up 执行迁移
func (m *GenkitSessionManagementMigration) Up() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建 conversation_memories 表
		if err := m.createConversationMemoriesTable(tx); err != nil {
			return fmt.Errorf("创建conversation_memories表失败: %w", err)
		}

		// 2. 创建 conversation_contexts 表
		if err := m.createConversationContextsTable(tx); err != nil {
			return fmt.Errorf("创建conversation_contexts表失败: %w", err)
		}

		// 3. 创建 conversation_summaries 表
		if err := m.createConversationSummariesTable(tx); err != nil {
			return fmt.Errorf("创建conversation_summaries表失败: %w", err)
		}

		// 4. 添加 conversation_contexts 表的外键约束（引用 conversation_summaries）
		if err := m.addContextSummaryForeignKey(tx); err != nil {
			return fmt.Errorf("添加conversation_contexts外键约束失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移
func (m *GenkitSessionManagementMigration) Down() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 按逆序删除表
		tables := []string{
			"conversation_summaries",
			"conversation_contexts",
			"conversation_memories",
		}

		for _, table := range tables {
			if err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
				return fmt.Errorf("删除表 %s 失败: %w", table, err)
			}
		}

		return nil
	})
}

// Name 返回迁移名称
func (m *GenkitSessionManagementMigration) Name() string {
	return "genkit_session_management_migration"
}

// GetName 获取迁移名称（实现 Migration 接口）
func (m *GenkitSessionManagementMigration) GetName() string {
	return m.Name()
}

// createConversationMemoriesTable 创建 conversation_memories 表
func (m *GenkitSessionManagementMigration) createConversationMemoriesTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS conversation_memories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		session_id UUID NOT NULL,
		memory_type VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		token_count INTEGER NOT NULL DEFAULT 0,
		importance REAL NOT NULL DEFAULT 0.5,
		access_count INTEGER NOT NULL DEFAULT 0,
		last_access_at TIMESTAMP WITH TIME ZONE,
		metadata JSONB,
		expires_at TIMESTAMP WITH TIME ZONE,
		is_deleted BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_memories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_memories_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
		CONSTRAINT check_memory_type CHECK (memory_type IN ('short_term', 'long_term', 'summary')),
		CONSTRAINT check_importance CHECK (importance >= 0 AND importance <= 1)
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_memories_tenant_session 
	ON conversation_memories(tenant_id, session_id) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_memories_type 
	ON conversation_memories(memory_type) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_memories_expires 
	ON conversation_memories(expires_at) 
	WHERE is_deleted = false AND expires_at IS NOT NULL;

	CREATE INDEX IF NOT EXISTS idx_memories_created 
	ON conversation_memories(created_at DESC) 
	WHERE is_deleted = false;

	-- 添加表注释
	COMMENT ON TABLE conversation_memories IS 'Genkit会话记忆表，存储长期记忆元数据（向量数据存储在 Qdrant 中）';
	
	-- 添加列注释
	COMMENT ON COLUMN conversation_memories.id IS '记忆唯一标识符（UUID）';
	COMMENT ON COLUMN conversation_memories.tenant_id IS '租户ID';
	COMMENT ON COLUMN conversation_memories.session_id IS '会话ID';
	COMMENT ON COLUMN conversation_memories.memory_type IS '记忆类型：short_term=短期记忆，long_term=长期记忆，summary=摘要记忆';
	COMMENT ON COLUMN conversation_memories.content IS '记忆内容（向量数据存储在 Qdrant 中）';
	COMMENT ON COLUMN conversation_memories.token_count IS 'Token数量';
	COMMENT ON COLUMN conversation_memories.importance IS '重要性评分（0-1）';
	COMMENT ON COLUMN conversation_memories.access_count IS '访问次数';
	COMMENT ON COLUMN conversation_memories.last_access_at IS '最后访问时间';
	COMMENT ON COLUMN conversation_memories.metadata IS '元数据（JSONB格式）';
	COMMENT ON COLUMN conversation_memories.expires_at IS '过期时间';
	COMMENT ON COLUMN conversation_memories.is_deleted IS '软删除标记';
	COMMENT ON COLUMN conversation_memories.created_at IS '创建时间';
	COMMENT ON COLUMN conversation_memories.updated_at IS '更新时间';
	`

	return tx.Exec(sql).Error
}

// createConversationContextsTable 创建 conversation_contexts 表
func (m *GenkitSessionManagementMigration) createConversationContextsTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS conversation_contexts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		session_id UUID NOT NULL UNIQUE,
		max_tokens INTEGER NOT NULL DEFAULT 4000,
		strategy VARCHAR(50) NOT NULL DEFAULT 'auto',
		include_summary BOOLEAN NOT NULL DEFAULT true,
		include_long_term BOOLEAN NOT NULL DEFAULT true,
		short_term_window INTEGER NOT NULL DEFAULT 10,
		last_summary_id UUID,
		last_summary_at TIMESTAMP WITH TIME ZONE,
		total_messages INTEGER NOT NULL DEFAULT 0,
		total_tokens_used BIGINT NOT NULL DEFAULT 0,
		is_deleted BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_contexts_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_contexts_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
		CONSTRAINT check_strategy CHECK (strategy IN ('auto', 'short', 'full')),
		CONSTRAINT check_max_tokens CHECK (max_tokens >= 100 AND max_tokens <= 128000),
		CONSTRAINT check_short_term_window CHECK (short_term_window >= 1 AND short_term_window <= 100)
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_contexts_tenant 
	ON conversation_contexts(tenant_id) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_contexts_session 
	ON conversation_contexts(session_id) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_contexts_updated 
	ON conversation_contexts(updated_at DESC) 
	WHERE is_deleted = false;

	-- 添加表注释
	COMMENT ON TABLE conversation_contexts IS 'Genkit会话上下文配置表，存储会话的上下文管理配置';
	
	-- 添加列注释
	COMMENT ON COLUMN conversation_contexts.id IS '上下文配置唯一标识符（UUID）';
	COMMENT ON COLUMN conversation_contexts.tenant_id IS '租户ID';
	COMMENT ON COLUMN conversation_contexts.session_id IS '会话ID（唯一）';
	COMMENT ON COLUMN conversation_contexts.max_tokens IS '最大Token数量';
	COMMENT ON COLUMN conversation_contexts.strategy IS '上下文策略：auto=自动，short=短期，full=完整';
	COMMENT ON COLUMN conversation_contexts.include_summary IS '是否包含摘要';
	COMMENT ON COLUMN conversation_contexts.include_long_term IS '是否包含长期记忆';
	COMMENT ON COLUMN conversation_contexts.short_term_window IS '短期记忆窗口大小';
	COMMENT ON COLUMN conversation_contexts.last_summary_id IS '最后摘要ID';
	COMMENT ON COLUMN conversation_contexts.last_summary_at IS '最后摘要时间';
	COMMENT ON COLUMN conversation_contexts.total_messages IS '总消息数';
	COMMENT ON COLUMN conversation_contexts.total_tokens_used IS '总Token使用量';
	COMMENT ON COLUMN conversation_contexts.is_deleted IS '软删除标记';
	COMMENT ON COLUMN conversation_contexts.created_at IS '创建时间';
	COMMENT ON COLUMN conversation_contexts.updated_at IS '更新时间';
	`

	return tx.Exec(sql).Error
}

// addContextSummaryForeignKey 添加 conversation_contexts 表的外键约束
func (m *GenkitSessionManagementMigration) addContextSummaryForeignKey(tx *gorm.DB) error {
	sql := `
	ALTER TABLE conversation_contexts
	ADD CONSTRAINT fk_contexts_last_summary 
	FOREIGN KEY (last_summary_id) 
	REFERENCES conversation_summaries(id) 
	ON DELETE SET NULL;
	`

	return tx.Exec(sql).Error
}

// createConversationSummariesTable 创建 conversation_summaries 表
func (m *GenkitSessionManagementMigration) createConversationSummariesTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS conversation_summaries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		session_id UUID NOT NULL,
		summary_type VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		token_count INTEGER NOT NULL,
		message_count INTEGER NOT NULL,
		start_message_id UUID,
		end_message_id UUID,
		quality_score REAL,
		compression_rate REAL,
		key_topics TEXT[],
		previous_summary_id UUID,
		is_deleted BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_summaries_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_summaries_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
		CONSTRAINT fk_summaries_start_message FOREIGN KEY (start_message_id) REFERENCES chat_messages(id) ON DELETE SET NULL,
		CONSTRAINT fk_summaries_end_message FOREIGN KEY (end_message_id) REFERENCES chat_messages(id) ON DELETE SET NULL,
		CONSTRAINT fk_summaries_previous FOREIGN KEY (previous_summary_id) REFERENCES conversation_summaries(id) ON DELETE SET NULL,
		CONSTRAINT check_summary_type CHECK (summary_type IN ('incremental', 'full')),
		CONSTRAINT check_quality_score CHECK (quality_score IS NULL OR (quality_score >= 0 AND quality_score <= 1)),
		CONSTRAINT check_compression_rate CHECK (compression_rate IS NULL OR (compression_rate >= 0 AND compression_rate <= 1))
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_summaries_tenant_session 
	ON conversation_summaries(tenant_id, session_id) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_summaries_created 
	ON conversation_summaries(created_at DESC) 
	WHERE is_deleted = false;

	CREATE INDEX IF NOT EXISTS idx_summaries_session_latest 
	ON conversation_summaries(session_id, created_at DESC) 
	WHERE is_deleted = false;

	-- 添加表注释
	COMMENT ON TABLE conversation_summaries IS 'Genkit会话摘要表，存储会话的摘要信息';
	
	-- 添加列注释
	COMMENT ON COLUMN conversation_summaries.id IS '摘要唯一标识符（UUID）';
	COMMENT ON COLUMN conversation_summaries.tenant_id IS '租户ID';
	COMMENT ON COLUMN conversation_summaries.session_id IS '会话ID';
	COMMENT ON COLUMN conversation_summaries.summary_type IS '摘要类型：incremental=增量摘要，full=完整摘要';
	COMMENT ON COLUMN conversation_summaries.content IS '摘要内容';
	COMMENT ON COLUMN conversation_summaries.token_count IS 'Token数量';
	COMMENT ON COLUMN conversation_summaries.message_count IS '包含的消息数量';
	COMMENT ON COLUMN conversation_summaries.start_message_id IS '起始消息ID';
	COMMENT ON COLUMN conversation_summaries.end_message_id IS '结束消息ID';
	COMMENT ON COLUMN conversation_summaries.quality_score IS '质量评分（0-1）';
	COMMENT ON COLUMN conversation_summaries.compression_rate IS '压缩率（0-1）';
	COMMENT ON COLUMN conversation_summaries.key_topics IS '关键主题数组';
	COMMENT ON COLUMN conversation_summaries.previous_summary_id IS '前一个摘要ID';
	COMMENT ON COLUMN conversation_summaries.is_deleted IS '软删除标记';
	COMMENT ON COLUMN conversation_summaries.created_at IS '创建时间';
	COMMENT ON COLUMN conversation_summaries.updated_at IS '更新时间';
	`

	return tx.Exec(sql).Error
}
