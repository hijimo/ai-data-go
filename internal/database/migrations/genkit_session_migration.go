package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// GenkitSessionMigration Genkit会话管理模块的数据库迁移
type GenkitSessionMigration struct {
	db *gorm.DB
}

// NewGenkitSessionMigration 创建Genkit会话管理迁移实例
func NewGenkitSessionMigration(db *gorm.DB) *GenkitSessionMigration {
	return &GenkitSessionMigration{db: db}
}

// GetName 获取迁移名称
func (m *GenkitSessionMigration) GetName() string {
	return "genkit_session_migration"
}

// Up 执行迁移
func (m *GenkitSessionMigration) Up() error {
	log.Println("开始执行 Genkit 会话管理模块迁移...")

	// 1. 启用 pgvector 扩展
	if err := m.enablePgvectorExtension(); err != nil {
		return fmt.Errorf("启用 pgvector 扩展失败: %w", err)
	}

	// 2. 创建 conversation_memories 表
	if err := m.createConversationMemoriesTable(); err != nil {
		return fmt.Errorf("创建 conversation_memories 表失败: %w", err)
	}

	// 3. 创建 conversation_contexts 表
	if err := m.createConversationContextsTable(); err != nil {
		return fmt.Errorf("创建 conversation_contexts 表失败: %w", err)
	}

	// 4. 创建 conversation_summaries 表
	if err := m.createConversationSummariesTable(); err != nil {
		return fmt.Errorf("创建 conversation_summaries 表失败: %w", err)
	}

	log.Println("Genkit 会话管理模块迁移完成")
	return nil
}

// Down 回滚迁移
func (m *GenkitSessionMigration) Down() error {
	log.Println("开始回滚 Genkit 会话管理模块迁移...")

	// 按相反顺序删除表
	tables := []string{
		"conversation_summaries",
		"conversation_contexts",
		"conversation_memories",
	}

	for _, table := range tables {
		if err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return fmt.Errorf("删除表 %s 失败: %w", table, err)
		}
		log.Printf("已删除表: %s", table)
	}

	log.Println("Genkit 会话管理模块迁移回滚完成")
	return nil
}

// enablePgvectorExtension 启用 pgvector 扩展
func (m *GenkitSessionMigration) enablePgvectorExtension() error {
	log.Println("启用 pgvector 扩展...")
	
	// 检查扩展是否已存在
	var count int64
	err := m.db.Raw("SELECT COUNT(*) FROM pg_extension WHERE extname = 'vector'").Scan(&count).Error
	if err != nil {
		return err
	}

	if count == 0 {
		// 创建扩展
		if err := m.db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
			return err
		}
		log.Println("pgvector 扩展已启用")
	} else {
		log.Println("pgvector 扩展已存在")
	}

	return nil
}

// createConversationMemoriesTable 创建 conversation_memories 表
func (m *GenkitSessionMigration) createConversationMemoriesTable() error {
	log.Println("创建 conversation_memories 表...")

	sql := `
	CREATE TABLE IF NOT EXISTS conversation_memories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		session_id UUID NOT NULL,
		memory_type VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		embedding vector(1536),
		token_count INTEGER NOT NULL DEFAULT 0,
		importance FLOAT NOT NULL DEFAULT 0.5,
		access_count INTEGER NOT NULL DEFAULT 0,
		last_access_at TIMESTAMP,
		metadata JSONB,
		expires_at TIMESTAMP,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_memories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_memories_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_memories_tenant_session 
		ON conversation_memories(tenant_id, session_id) 
		WHERE is_deleted = FALSE;
	
	CREATE INDEX IF NOT EXISTS idx_memories_type 
		ON conversation_memories(memory_type) 
		WHERE is_deleted = FALSE;
	
	CREATE INDEX IF NOT EXISTS idx_memories_expires 
		ON conversation_memories(expires_at) 
		WHERE is_deleted = FALSE AND expires_at IS NOT NULL;
	
	CREATE INDEX IF NOT EXISTS idx_memories_created 
		ON conversation_memories(created_at DESC) 
		WHERE is_deleted = FALSE;

	-- 创建向量索引（IVFFlat 算法）
	-- 注意：需要表中有足够数据后才能创建向量索引，这里先创建表结构
	-- 向量索引将在有数据后通过单独的迁移或手动创建
	`

	if err := m.db.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("conversation_memories 表创建成功")
	return nil
}

// createConversationContextsTable 创建 conversation_contexts 表
func (m *GenkitSessionMigration) createConversationContextsTable() error {
	log.Println("创建 conversation_contexts 表...")

	sql := `
	CREATE TABLE IF NOT EXISTS conversation_contexts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		session_id UUID NOT NULL UNIQUE,
		max_tokens INTEGER NOT NULL DEFAULT 4000,
		strategy VARCHAR(50) NOT NULL DEFAULT 'auto',
		include_summary BOOLEAN NOT NULL DEFAULT TRUE,
		include_long_term BOOLEAN NOT NULL DEFAULT TRUE,
		short_term_window INTEGER NOT NULL DEFAULT 10,
		last_summary_id UUID,
		last_summary_at TIMESTAMP,
		total_messages INTEGER NOT NULL DEFAULT 0,
		total_tokens_used BIGINT NOT NULL DEFAULT 0,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_contexts_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_contexts_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_contexts_tenant 
		ON conversation_contexts(tenant_id) 
		WHERE is_deleted = FALSE;
	
	CREATE INDEX IF NOT EXISTS idx_contexts_session 
		ON conversation_contexts(session_id) 
		WHERE is_deleted = FALSE;
	`

	if err := m.db.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("conversation_contexts 表创建成功")
	return nil
}

// createConversationSummariesTable 创建 conversation_summaries 表
func (m *GenkitSessionMigration) createConversationSummariesTable() error {
	log.Println("创建 conversation_summaries 表...")

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
		quality_score FLOAT,
		compression_rate FLOAT,
		key_topics TEXT[],
		previous_summary_id UUID,
		is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		CONSTRAINT fk_summaries_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		CONSTRAINT fk_summaries_session FOREIGN KEY (session_id) REFERENCES conversation_sessions(id) ON DELETE CASCADE,
		CONSTRAINT fk_summaries_previous FOREIGN KEY (previous_summary_id) REFERENCES conversation_summaries(id) ON DELETE SET NULL
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_summaries_tenant_session 
		ON conversation_summaries(tenant_id, session_id) 
		WHERE is_deleted = FALSE;
	
	CREATE INDEX IF NOT EXISTS idx_summaries_created 
		ON conversation_summaries(created_at DESC) 
		WHERE is_deleted = FALSE;
	
	CREATE INDEX IF NOT EXISTS idx_summaries_session_latest 
		ON conversation_summaries(session_id, created_at DESC) 
		WHERE is_deleted = FALSE;
	`

	if err := m.db.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("conversation_summaries 表创建成功")
	return nil
}

// CreateVectorIndex 创建向量索引（需要在表中有数据后调用）
func (m *GenkitSessionMigration) CreateVectorIndex() error {
	log.Println("创建向量索引...")

	// 检查表中是否有足够的数据
	var count int64
	if err := m.db.Table("conversation_memories").
		Where("is_deleted = ? AND embedding IS NOT NULL", false).
		Count(&count).Error; err != nil {
		return err
	}

	if count < 100 {
		log.Printf("警告：表中只有 %d 条记录，建议至少有 100 条记录后再创建向量索引", count)
		return fmt.Errorf("数据量不足，无法创建向量索引")
	}

	// 创建 IVFFlat 向量索引
	sql := `
	CREATE INDEX IF NOT EXISTS idx_memories_embedding 
		ON conversation_memories 
		USING ivfflat (embedding vector_cosine_ops) 
		WITH (lists = 100)
		WHERE is_deleted = FALSE AND embedding IS NOT NULL;
	`

	if err := m.db.Exec(sql).Error; err != nil {
		return err
	}

	log.Println("向量索引创建成功")
	return nil
}
