package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// InitialMigration 初始迁移结构
type InitialMigration struct {
	db *gorm.DB
}

// NewInitialMigration 创建初始迁移实例
func NewInitialMigration(db *gorm.DB) *InitialMigration {
	return &InitialMigration{
		db: db,
	}
}

// Up 执行迁移（创建所有表）
func (m *InitialMigration) Up() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 1. 启用 UUID 扩展
		if err := m.enableUUIDExtension(tx); err != nil {
			return fmt.Errorf("启用UUID扩展失败: %w", err)
		}

		// 2. 创建 tenants 表
		if err := m.createTenantsTable(tx); err != nil {
			return fmt.Errorf("创建tenants表失败: %w", err)
		}

		// 3. 创建 users 表
		if err := m.createUsersTable(tx); err != nil {
			return fmt.Errorf("创建users表失败: %w", err)
		}

		// 4. 创建 refresh_tokens 表
		if err := m.createRefreshTokensTable(tx); err != nil {
			return fmt.Errorf("创建refresh_tokens表失败: %w", err)
		}

		// 5. 创建 email_verification_tokens 表
		if err := m.createEmailVerificationTokensTable(tx); err != nil {
			return fmt.Errorf("创建email_verification_tokens表失败: %w", err)
		}

		// 6. 创建 auth_audit 表
		if err := m.createAuthAuditTable(tx); err != nil {
			return fmt.Errorf("创建auth_audit表失败: %w", err)
		}

		// 7. 创建 chat_sessions 表
		if err := m.createChatSessionsTable(tx); err != nil {
			return fmt.Errorf("创建chat_sessions表失败: %w", err)
		}

		// 8. 创建 chat_messages 表
		if err := m.createChatMessagesTable(tx); err != nil {
			return fmt.Errorf("创建chat_messages表失败: %w", err)
		}

		// 9. 创建 chat_summaries 表
		if err := m.createChatSummariesTable(tx); err != nil {
			return fmt.Errorf("创建chat_summaries表失败: %w", err)
		}

		return nil
	})
}

// Down 回滚迁移（删除所有表）
func (m *InitialMigration) Down() error {
	// 使用事务确保原子性
	return m.db.Transaction(func(tx *gorm.DB) error {
		// 按逆序删除表，使用 CASCADE 确保依赖关系正确处理
		tables := []string{
			"chat_summaries",
			"chat_messages",
			"chat_sessions",
			"auth_audit",
			"email_verification_tokens",
			"refresh_tokens",
			"users",
			"tenants",
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
func (m *InitialMigration) Name() string {
	return "initial_migration"
}

// GetName 获取迁移名称（实现 Migration 接口）
func (m *InitialMigration) GetName() string {
	return m.Name()
}

// enableUUIDExtension 启用 PostgreSQL UUID 扩展
func (m *InitialMigration) enableUUIDExtension(tx *gorm.DB) error {
	// 检查数据库类型
	dbType := tx.Dialector.Name()
	if dbType != "postgres" {
		return nil
	}

	// 尝试启用 pgcrypto 扩展（PostgreSQL 13+ 推荐）
	if err := tx.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"").Error; err != nil {
		// 如果 pgcrypto 失败，尝试 uuid-ossp
		if err := tx.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
			return fmt.Errorf("启用UUID扩展失败: %w", err)
		}
	}

	return nil
}

// createTenantsTable 创建 tenants 表
func (m *InitialMigration) createTenantsTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS tenants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		domain VARCHAR(255),
		type VARCHAR(32) NOT NULL DEFAULT 'tenant',
		metadata JSONB,
		status BOOLEAN DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE,
		updated_at TIMESTAMP WITH TIME ZONE,
		created_by UUID,
		is_deleted BOOLEAN DEFAULT false,
		CONSTRAINT tenants_type_check CHECK (type IN ('system', 'tenant'))
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_tenants_domain ON tenants(domain) WHERE domain IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status) WHERE NOT is_deleted;
	CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tenants_type ON tenants(type);
	
	-- 创建唯一约束确保只能有一个平台租户
	CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_system_tenant 
	ON tenants(type) 
	WHERE type = 'system' AND is_deleted = false;

	-- 添加表注释
	COMMENT ON TABLE tenants IS '租户表，存储多租户系统中的租户信息';
	
	-- 添加列注释
	COMMENT ON COLUMN tenants.id IS '租户唯一标识符（UUID）';
	COMMENT ON COLUMN tenants.name IS '租户名称';
	COMMENT ON COLUMN tenants.domain IS '租户域名，用于子域名识别';
	COMMENT ON COLUMN tenants.type IS '租户类型：system=平台租户，tenant=业务租户';
	COMMENT ON COLUMN tenants.metadata IS '租户元数据（JSONB格式）';
	COMMENT ON COLUMN tenants.status IS '租户状态：true=启用，false=禁用';
	COMMENT ON COLUMN tenants.created_at IS '创建时间';
	COMMENT ON COLUMN tenants.updated_at IS '更新时间';
	COMMENT ON COLUMN tenants.created_by IS '创建者用户ID';
	COMMENT ON COLUMN tenants.is_deleted IS '软删除标记';
	`

	return tx.Exec(sql).Error
}

// createUsersTable 创建 users 表
func (m *InitialMigration) createUsersTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		email VARCHAR(320) NOT NULL,
		email_verified BOOLEAN DEFAULT false,
		phone VARCHAR(20),
		password_hash TEXT NOT NULL,
		display_name VARCHAR(255),
		is_active BOOLEAN DEFAULT true,
		is_admin BOOLEAN DEFAULT false,
		roles JSONB,
		meta JSONB,
		last_login_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE,
		updated_at TIMESTAMP WITH TIME ZONE,
		created_by UUID,
		is_deleted BOOLEAN DEFAULT false,
		failed_login_attempts BIGINT DEFAULT 0,
		locked_until TIMESTAMP WITH TIME ZONE,
		CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
	);

	-- 创建唯一约束和索引
	CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_email ON users(tenant_id, email);
	CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id) WHERE NOT is_deleted;
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active) WHERE NOT is_deleted;
	CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

	-- 添加表注释
	COMMENT ON TABLE users IS '用户表，存储租户下的用户账户信息';
	
	-- 添加列注释
	COMMENT ON COLUMN users.id IS '用户唯一标识符（UUID）';
	COMMENT ON COLUMN users.tenant_id IS '所属租户ID';
	COMMENT ON COLUMN users.email IS '用户邮箱地址';
	COMMENT ON COLUMN users.email_verified IS '邮箱是否已验证';
	COMMENT ON COLUMN users.phone IS '手机号码';
	COMMENT ON COLUMN users.password_hash IS '密码哈希值（使用 bcrypt 算法）';
	COMMENT ON COLUMN users.display_name IS '用户显示名称';
	COMMENT ON COLUMN users.is_active IS '账户是否激活';
	COMMENT ON COLUMN users.is_admin IS '是否为管理员';
	COMMENT ON COLUMN users.roles IS '用户角色列表（JSONB格式），如 ["user","admin"]';
	COMMENT ON COLUMN users.meta IS '用户元数据（JSONB格式）';
	COMMENT ON COLUMN users.last_login_at IS '最后登录时间';
	COMMENT ON COLUMN users.created_at IS '创建时间';
	COMMENT ON COLUMN users.updated_at IS '更新时间';
	COMMENT ON COLUMN users.created_by IS '创建者用户ID';
	COMMENT ON COLUMN users.is_deleted IS '软删除标记';
	COMMENT ON COLUMN users.failed_login_attempts IS '失败登录次数';
	COMMENT ON COLUMN users.locked_until IS '锁定截止时间';
	`

	return tx.Exec(sql).Error
}

// createRefreshTokensTable 创建 refresh_tokens 表
func (m *InitialMigration) createRefreshTokensTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		tenant_id UUID NOT NULL,
		token_hash TEXT NOT NULL,
		revoked BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		replaced_by UUID,
		CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		CONSTRAINT fk_refresh_tokens_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE UNIQUE INDEX IF NOT EXISTS idx_token_hash ON refresh_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_user_tokens ON refresh_tokens(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tenant_tokens ON refresh_tokens(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_revoked ON refresh_tokens(revoked);
	CREATE INDEX IF NOT EXISTS idx_expires ON refresh_tokens(expires_at);

	-- 添加表注释
	COMMENT ON TABLE refresh_tokens IS '刷新令牌表，存储用户的 Refresh Token 信息';
	
	-- 添加列注释
	COMMENT ON COLUMN refresh_tokens.id IS '令牌唯一标识符（UUID）';
	COMMENT ON COLUMN refresh_tokens.user_id IS '用户ID';
	COMMENT ON COLUMN refresh_tokens.tenant_id IS '租户ID';
	COMMENT ON COLUMN refresh_tokens.token_hash IS 'Refresh Token 的 SHA256 哈希值';
	COMMENT ON COLUMN refresh_tokens.revoked IS '是否已撤销';
	COMMENT ON COLUMN refresh_tokens.created_at IS '创建时间';
	COMMENT ON COLUMN refresh_tokens.expires_at IS '过期时间';
	COMMENT ON COLUMN refresh_tokens.replaced_by IS '轮换时指向新 token 的 ID';
	`

	return tx.Exec(sql).Error
}

// createEmailVerificationTokensTable 创建 email_verification_tokens 表
func (m *InitialMigration) createEmailVerificationTokensTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS email_verification_tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		tenant_id UUID NOT NULL,
		token VARCHAR(64) NOT NULL,
		email VARCHAR(320) NOT NULL,
		used BOOLEAN DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		CONSTRAINT fk_email_verification_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		CONSTRAINT fk_email_verification_tokens_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE UNIQUE INDEX IF NOT EXISTS idx_verification_token ON email_verification_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_user_verification ON email_verification_tokens(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tenant_verification ON email_verification_tokens(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_used ON email_verification_tokens(used);
	CREATE INDEX IF NOT EXISTS idx_verification_expires ON email_verification_tokens(expires_at);

	-- 添加表注释
	COMMENT ON TABLE email_verification_tokens IS '邮箱验证令牌表，存储用户邮箱验证的令牌信息';
	
	-- 添加列注释
	COMMENT ON COLUMN email_verification_tokens.id IS '验证令牌唯一标识符（UUID）';
	COMMENT ON COLUMN email_verification_tokens.user_id IS '用户ID';
	COMMENT ON COLUMN email_verification_tokens.tenant_id IS '租户ID';
	COMMENT ON COLUMN email_verification_tokens.token IS '验证令牌（随机生成的UUID）';
	COMMENT ON COLUMN email_verification_tokens.email IS '邮箱地址';
	COMMENT ON COLUMN email_verification_tokens.used IS '是否已使用';
	COMMENT ON COLUMN email_verification_tokens.created_at IS '创建时间';
	COMMENT ON COLUMN email_verification_tokens.expires_at IS '过期时间';
	`

	return tx.Exec(sql).Error
}

// createAuthAuditTable 创建 auth_audit 表
func (m *InitialMigration) createAuthAuditTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS auth_audit (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID,
		user_id UUID,
		event VARCHAR(64) NOT NULL,
		ip VARCHAR(45),
		user_agent TEXT,
		meta JSONB,
		created_at TIMESTAMP WITH TIME ZONE
	);

	-- 创建索引（不需要外键约束）
	CREATE INDEX IF NOT EXISTS idx_tenant_audit ON auth_audit(tenant_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_user_audit ON auth_audit(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_event ON auth_audit(event);
	CREATE INDEX IF NOT EXISTS idx_created_at ON auth_audit(created_at DESC);

	-- 添加表注释
	COMMENT ON TABLE auth_audit IS '认证审计日志表，记录所有身份认证相关的操作';
	
	-- 添加列注释
	COMMENT ON COLUMN auth_audit.id IS '审计日志唯一标识符（UUID）';
	COMMENT ON COLUMN auth_audit.tenant_id IS '租户ID';
	COMMENT ON COLUMN auth_audit.user_id IS '用户ID';
	COMMENT ON COLUMN auth_audit.event IS '事件类型：login, logout, refresh, revoke, failed_login';
	COMMENT ON COLUMN auth_audit.ip IS '客户端IP地址';
	COMMENT ON COLUMN auth_audit.user_agent IS '用户代理字符串';
	COMMENT ON COLUMN auth_audit.meta IS '事件元数据（JSONB格式）';
	COMMENT ON COLUMN auth_audit.created_at IS '事件发生时间';
	`

	return tx.Exec(sql).Error
}

// createChatSessionsTable 创建 chat_sessions 表
func (m *InitialMigration) createChatSessionsTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS chat_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		title VARCHAR(255) NOT NULL,
		model_name VARCHAR(128) NOT NULL,
		system_prompt TEXT,
		temperature NUMERIC,
		top_p NUMERIC,
		created_by UUID NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_message_id UUID,
		message_count BIGINT DEFAULT 0,
		is_pinned BOOLEAN DEFAULT false,
		is_archived BOOLEAN DEFAULT false,
		is_deleted BOOLEAN DEFAULT false,
		meta JSONB,
		CONSTRAINT fk_chat_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_user_sessions ON chat_sessions(user_id, updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_pinned ON chat_sessions(is_pinned, updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_archived ON chat_sessions(is_archived);
	CREATE INDEX IF NOT EXISTS idx_deleted ON chat_sessions(is_deleted);

	-- 添加表注释
	COMMENT ON TABLE chat_sessions IS '聊天会话表，存储用户的聊天会话信息';
	
	-- 添加列注释
	COMMENT ON COLUMN chat_sessions.id IS '会话唯一标识符（UUID）';
	COMMENT ON COLUMN chat_sessions.user_id IS '用户ID';
	COMMENT ON COLUMN chat_sessions.title IS '会话标题';
	COMMENT ON COLUMN chat_sessions.model_name IS '模型名称';
	COMMENT ON COLUMN chat_sessions.system_prompt IS '系统提示词';
	COMMENT ON COLUMN chat_sessions.temperature IS '温度参数';
	COMMENT ON COLUMN chat_sessions.top_p IS 'Top-P参数';
	COMMENT ON COLUMN chat_sessions.created_by IS '创建者ID';
	COMMENT ON COLUMN chat_sessions.created_at IS '创建时间';
	COMMENT ON COLUMN chat_sessions.updated_at IS '更新时间';
	COMMENT ON COLUMN chat_sessions.last_message_id IS '最后消息ID';
	COMMENT ON COLUMN chat_sessions.message_count IS '消息数量';
	COMMENT ON COLUMN chat_sessions.is_pinned IS '是否置顶';
	COMMENT ON COLUMN chat_sessions.is_archived IS '是否归档';
	COMMENT ON COLUMN chat_sessions.is_deleted IS '软删除标记';
	COMMENT ON COLUMN chat_sessions.meta IS '元数据（JSONB格式）';
	`

	return tx.Exec(sql).Error
}

// createChatMessagesTable 创建 chat_messages 表
func (m *InitialMigration) createChatMessagesTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS chat_messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		session_id UUID NOT NULL,
		role VARCHAR(32) NOT NULL,
		content TEXT NOT NULL,
		tokens BIGINT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sequence BIGINT NOT NULL,
		tool_calls JSONB,
		error TEXT,
		parent_id UUID,
		meta JSONB,
		CONSTRAINT fk_chat_messages_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_session_messages ON chat_messages(session_id, sequence ASC);
	CREATE INDEX IF NOT EXISTS idx_created ON chat_messages(created_at DESC);

	-- 添加表注释
	COMMENT ON TABLE chat_messages IS '聊天消息表，存储会话中的消息内容';
	
	-- 添加列注释
	COMMENT ON COLUMN chat_messages.id IS '消息唯一标识符（UUID）';
	COMMENT ON COLUMN chat_messages.session_id IS '会话ID';
	COMMENT ON COLUMN chat_messages.role IS '角色（user/assistant/system）';
	COMMENT ON COLUMN chat_messages.content IS '消息内容';
	COMMENT ON COLUMN chat_messages.tokens IS 'Token数量';
	COMMENT ON COLUMN chat_messages.created_at IS '创建时间';
	COMMENT ON COLUMN chat_messages.sequence IS '消息序号';
	COMMENT ON COLUMN chat_messages.tool_calls IS '工具调用信息（JSONB格式）';
	COMMENT ON COLUMN chat_messages.error IS '错误信息';
	COMMENT ON COLUMN chat_messages.parent_id IS '父消息ID';
	COMMENT ON COLUMN chat_messages.meta IS '元数据（JSONB格式）';
	`

	return tx.Exec(sql).Error
}

// createChatSummariesTable 创建 chat_summaries 表
func (m *InitialMigration) createChatSummariesTable(tx *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS chat_summaries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		session_id UUID NOT NULL,
		summary TEXT NOT NULL,
		last_message_id UUID NOT NULL,
		token_count BIGINT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_chat_summaries_session FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_session_summary ON chat_summaries(session_id, created_at DESC);

	-- 添加表注释
	COMMENT ON TABLE chat_summaries IS '聊天摘要表，存储会话的摘要信息';
	
	-- 添加列注释
	COMMENT ON COLUMN chat_summaries.id IS '摘要唯一标识符（UUID）';
	COMMENT ON COLUMN chat_summaries.session_id IS '会话ID';
	COMMENT ON COLUMN chat_summaries.summary IS '摘要内容';
	COMMENT ON COLUMN chat_summaries.last_message_id IS '最后消息ID';
	COMMENT ON COLUMN chat_summaries.token_count IS 'Token数量';
	COMMENT ON COLUMN chat_summaries.created_at IS '创建时间';
	`

	return tx.Exec(sql).Error
}
