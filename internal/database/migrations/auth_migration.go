package migrations

import (
	"fmt"

	"genkit-ai-service/internal/model"

	"gorm.io/gorm"
)

// AuthMigration 认证相关表的迁移
type AuthMigration struct {
	db *gorm.DB
}

// NewAuthMigration 创建认证迁移实例
func NewAuthMigration(db *gorm.DB) *AuthMigration {
	return &AuthMigration{
		db: db,
	}
}

// Up 执行迁移（创建表和索引）
func (m *AuthMigration) Up() error {
	// 检查数据库类型
	dbType := m.db.Dialector.Name()
	
	// 如果是 PostgreSQL，先确保 UUID 扩展已启用
	if dbType == "postgres" {
		if err := m.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
			return fmt.Errorf("启用 UUID 扩展失败: %w", err)
		}
	}

	// 自动迁移表结构
	if err := m.db.AutoMigrate(
		&model.Tenant{},
		&model.User{},
		&model.RefreshToken{},
		&model.AuthAudit{},
		&model.EmailVerificationToken{},
	); err != nil {
		return fmt.Errorf("自动迁移表结构失败: %w", err)
	}

	// 如果是 PostgreSQL，修改列类型为 UUID 并添加默认值
	if dbType == "postgres" {
		if err := m.alterColumnsForPostgres(); err != nil {
			return fmt.Errorf("修改 PostgreSQL 列类型失败: %w", err)
		}
	}

	// 创建额外的索引和约束
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 仅在 PostgreSQL 中添加注释
	if dbType == "postgres" {
		// 添加表注释
		if err := m.addTableComments(); err != nil {
			return fmt.Errorf("添加表注释失败: %w", err)
		}

		// 添加列注释
		if err := m.addColumnComments(); err != nil {
			return fmt.Errorf("添加列注释失败: %w", err)
		}
	}

	return nil
}

// Down 回滚迁移（删除表）
func (m *AuthMigration) Down() error {
	// 删除表（按依赖关系倒序删除）
	if err := m.db.Migrator().DropTable(
		&model.AuthAudit{},
		&model.EmailVerificationToken{},
		&model.RefreshToken{},
		&model.User{},
		&model.Tenant{},
	); err != nil {
		return fmt.Errorf("删除表失败: %w", err)
	}

	return nil
}

// alterColumnsForPostgres 为 PostgreSQL 修改列类型为 UUID
func (m *AuthMigration) alterColumnsForPostgres() error {
	// Tenant 表
	alterStatements := []string{
		"ALTER TABLE tenants ALTER COLUMN id TYPE uuid USING id::uuid",
		"ALTER TABLE tenants ALTER COLUMN id SET DEFAULT gen_random_uuid()",
		"ALTER TABLE tenants ALTER COLUMN created_by TYPE uuid USING created_by::uuid",
		"ALTER TABLE tenants ALTER COLUMN metadata TYPE jsonb USING metadata::jsonb",
		
		// User 表
		"ALTER TABLE users ALTER COLUMN id TYPE uuid USING id::uuid",
		"ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid()",
		"ALTER TABLE users ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid",
		"ALTER TABLE users ALTER COLUMN created_by TYPE uuid USING created_by::uuid",
		"ALTER TABLE users ALTER COLUMN roles TYPE jsonb USING roles::jsonb",
		"ALTER TABLE users ALTER COLUMN meta TYPE jsonb USING meta::jsonb",
		
		// RefreshToken 表
		"ALTER TABLE refresh_tokens ALTER COLUMN id TYPE uuid USING id::uuid",
		"ALTER TABLE refresh_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid()",
		"ALTER TABLE refresh_tokens ALTER COLUMN user_id TYPE uuid USING user_id::uuid",
		"ALTER TABLE refresh_tokens ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid",
		"ALTER TABLE refresh_tokens ALTER COLUMN replaced_by TYPE uuid USING replaced_by::uuid",
		
		// AuthAudit 表
		"ALTER TABLE auth_audit ALTER COLUMN id TYPE uuid USING id::uuid",
		"ALTER TABLE auth_audit ALTER COLUMN id SET DEFAULT gen_random_uuid()",
		"ALTER TABLE auth_audit ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid",
		"ALTER TABLE auth_audit ALTER COLUMN user_id TYPE uuid USING user_id::uuid",
		"ALTER TABLE auth_audit ALTER COLUMN ip TYPE inet USING ip::inet",
		"ALTER TABLE auth_audit ALTER COLUMN meta TYPE jsonb USING meta::jsonb",
		
		// EmailVerificationToken 表
		"ALTER TABLE email_verification_tokens ALTER COLUMN id TYPE uuid USING id::uuid",
		"ALTER TABLE email_verification_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid()",
		"ALTER TABLE email_verification_tokens ALTER COLUMN user_id TYPE uuid USING user_id::uuid",
		"ALTER TABLE email_verification_tokens ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid",
	}

	for _, stmt := range alterStatements {
		if err := m.db.Exec(stmt).Error; err != nil {
			// 忽略已经是正确类型的错误
			if err.Error() != "column is already of type uuid" && 
			   err.Error() != "column is already of type jsonb" &&
			   err.Error() != "column is already of type inet" {
				return fmt.Errorf("执行 SQL 失败 [%s]: %w", stmt, err)
			}
		}
	}

	return nil
}

// createIndexes 创建额外的索引
func (m *AuthMigration) createIndexes() error {
	// Tenant 表索引
	// idx_tenants_domain: 用于域名查询
	if !m.db.Migrator().HasIndex(&model.Tenant{}, "idx_tenants_domain") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenants_domain 
			ON tenants(domain) 
			WHERE domain IS NOT NULL
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenants_domain 索引失败: %w", err)
		}
	}

	// idx_tenants_status: 用于状态过滤
	if !m.db.Migrator().HasIndex(&model.Tenant{}, "idx_tenants_status") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenants_status 
			ON tenants(status) 
			WHERE NOT is_deleted
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenants_status 索引失败: %w", err)
		}
	}

	// idx_tenants_created_at: 用于时间排序
	if !m.db.Migrator().HasIndex(&model.Tenant{}, "idx_tenants_created_at") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenants_created_at 
			ON tenants(created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenants_created_at 索引失败: %w", err)
		}
	}

	// User 表索引
	// idx_tenant_email: 租户内邮箱唯一性（已由 GORM 自动创建）
	// 确保唯一约束存在（仅在 PostgreSQL 中）
	dbType := m.db.Dialector.Name()
	if dbType == "postgres" && !m.db.Migrator().HasConstraint(&model.User{}, "uq_tenant_email") {
		if err := m.db.Exec(`
			ALTER TABLE users 
			ADD CONSTRAINT uq_tenant_email UNIQUE (tenant_id, email)
		`).Error; err != nil {
			// 如果约束已存在，忽略错误
			if err.Error() != "constraint already exists" {
				return fmt.Errorf("创建 uq_tenant_email 约束失败: %w", err)
			}
		}
	}

	// idx_users_tenant_id: 用于租户用户查询
	if !m.db.Migrator().HasIndex(&model.User{}, "idx_users_tenant_id") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_users_tenant_id 
			ON users(tenant_id) 
			WHERE NOT is_deleted
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_users_tenant_id 索引失败: %w", err)
		}
	}

	// idx_users_email: 用于邮箱查询
	if !m.db.Migrator().HasIndex(&model.User{}, "idx_users_email") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_users_email 
			ON users(email)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_users_email 索引失败: %w", err)
		}
	}

	// idx_users_is_active: 用于活跃状态过滤
	if !m.db.Migrator().HasIndex(&model.User{}, "idx_users_is_active") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_users_is_active 
			ON users(is_active) 
			WHERE NOT is_deleted
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_users_is_active 索引失败: %w", err)
		}
	}

	// idx_users_created_at: 用于时间排序
	if !m.db.Migrator().HasIndex(&model.User{}, "idx_users_created_at") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_users_created_at 
			ON users(created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_users_created_at 索引失败: %w", err)
		}
	}

	// RefreshToken 表索引
	// idx_user_tokens: 用于用户令牌查询
	if !m.db.Migrator().HasIndex(&model.RefreshToken{}, "idx_user_tokens") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_user_tokens 
			ON refresh_tokens(user_id, created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_user_tokens 索引失败: %w", err)
		}
	}

	// idx_tenant_tokens: 用于租户令牌查询
	if !m.db.Migrator().HasIndex(&model.RefreshToken{}, "idx_tenant_tokens") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenant_tokens 
			ON refresh_tokens(tenant_id)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenant_tokens 索引失败: %w", err)
		}
	}

	// idx_revoked: 用于撤销状态过滤
	if !m.db.Migrator().HasIndex(&model.RefreshToken{}, "idx_revoked") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_revoked 
			ON refresh_tokens(revoked)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_revoked 索引失败: %w", err)
		}
	}

	// idx_expires: 用于过期时间查询
	if !m.db.Migrator().HasIndex(&model.RefreshToken{}, "idx_expires") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_expires 
			ON refresh_tokens(expires_at)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_expires 索引失败: %w", err)
		}
	}

	// idx_token_hash: 用于令牌哈希查询（已由 GORM 自动创建唯一索引）

	// AuthAudit 表索引
	// idx_tenant_audit: 用于租户审计日志查询
	if !m.db.Migrator().HasIndex(&model.AuthAudit{}, "idx_tenant_audit") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenant_audit 
			ON auth_audit(tenant_id, created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenant_audit 索引失败: %w", err)
		}
	}

	// idx_user_audit: 用于用户审计日志查询
	if !m.db.Migrator().HasIndex(&model.AuthAudit{}, "idx_user_audit") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_user_audit 
			ON auth_audit(user_id, created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_user_audit 索引失败: %w", err)
		}
	}

	// idx_event: 用于事件类型过滤
	if !m.db.Migrator().HasIndex(&model.AuthAudit{}, "idx_event") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_event 
			ON auth_audit(event)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_event 索引失败: %w", err)
		}
	}

	// idx_created_at: 用于时间排序
	if !m.db.Migrator().HasIndex(&model.AuthAudit{}, "idx_created_at") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_created_at 
			ON auth_audit(created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_created_at 索引失败: %w", err)
		}
	}

	// EmailVerificationToken 表索引
	// idx_user_verification: 用于用户验证令牌查询
	if !m.db.Migrator().HasIndex(&model.EmailVerificationToken{}, "idx_user_verification") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_user_verification 
			ON email_verification_tokens(user_id, created_at DESC)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_user_verification 索引失败: %w", err)
		}
	}

	// idx_tenant_verification: 用于租户验证令牌查询
	if !m.db.Migrator().HasIndex(&model.EmailVerificationToken{}, "idx_tenant_verification") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_tenant_verification 
			ON email_verification_tokens(tenant_id)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_tenant_verification 索引失败: %w", err)
		}
	}

	// idx_used: 用于使用状态过滤
	if !m.db.Migrator().HasIndex(&model.EmailVerificationToken{}, "idx_used") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_used 
			ON email_verification_tokens(used)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_used 索引失败: %w", err)
		}
	}

	// idx_verification_expires: 用于过期时间查询
	if !m.db.Migrator().HasIndex(&model.EmailVerificationToken{}, "idx_verification_expires") {
		if err := m.db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_verification_expires 
			ON email_verification_tokens(expires_at)
		`).Error; err != nil {
			return fmt.Errorf("创建 idx_verification_expires 索引失败: %w", err)
		}
	}

	// idx_verification_token: 用于令牌查询（已由 GORM 自动创建唯一索引）

	return nil
}

// addTableComments 添加表注释
func (m *AuthMigration) addTableComments() error {
	comments := map[string]string{
		"tenants":                    "租户表，存储多租户系统中的租户信息",
		"users":                      "用户表，存储租户下的用户账户信息",
		"refresh_tokens":             "刷新令牌表，存储用户的 Refresh Token 信息",
		"auth_audit":                 "认证审计日志表，记录所有身份认证相关的操作",
		"email_verification_tokens":  "邮箱验证令牌表，存储用户邮箱验证的令牌信息",
	}

	for table, comment := range comments {
		if err := m.db.Exec(fmt.Sprintf(
			"COMMENT ON TABLE %s IS '%s'",
			table, comment,
		)).Error; err != nil {
			return fmt.Errorf("添加表 %s 注释失败: %w", table, err)
		}
	}

	return nil
}

// addColumnComments 添加列注释
func (m *AuthMigration) addColumnComments() error {
	// Tenant 表列注释
	tenantComments := map[string]string{
		"id":         "租户唯一标识符（UUID）",
		"name":       "租户名称",
		"domain":     "租户域名，用于子域名识别",
		"metadata":   "租户元数据（JSONB格式）",
		"status":     "租户状态：true=启用，false=禁用",
		"created_at": "创建时间",
		"updated_at": "更新时间",
		"created_by": "创建者用户ID",
		"is_deleted": "软删除标记",
	}
	if err := m.addColumnCommentsForTable("tenants", tenantComments); err != nil {
		return err
	}

	// User 表列注释
	userComments := map[string]string{
		"id":              "用户唯一标识符（UUID）",
		"tenant_id":       "所属租户ID",
		"email":           "用户邮箱地址",
		"email_verified":  "邮箱是否已验证",
		"phone":           "手机号码",
		"password_hash":   "密码哈希值（使用 bcrypt 算法）",
		"display_name":    "用户显示名称",
		"is_active":       "账户是否激活",
		"is_admin":        "是否为管理员",
		"roles":           "用户角色列表（JSONB格式），如 [\"user\",\"admin\"]",
		"meta":            "用户元数据（JSONB格式）",
		"last_login_at":   "最后登录时间",
		"created_at":      "创建时间",
		"updated_at":      "更新时间",
		"created_by":      "创建者用户ID",
		"is_deleted":      "软删除标记",
	}
	if err := m.addColumnCommentsForTable("users", userComments); err != nil {
		return err
	}

	// RefreshToken 表列注释
	tokenComments := map[string]string{
		"id":          "令牌唯一标识符（UUID）",
		"user_id":     "用户ID",
		"tenant_id":   "租户ID",
		"token_hash":  "Refresh Token 的 SHA256 哈希值",
		"revoked":     "是否已撤销",
		"created_at":  "创建时间",
		"expires_at":  "过期时间",
		"replaced_by": "轮换时指向新 token 的 ID",
	}
	if err := m.addColumnCommentsForTable("refresh_tokens", tokenComments); err != nil {
		return err
	}

	// AuthAudit 表列注释
	auditComments := map[string]string{
		"id":         "审计日志唯一标识符（UUID）",
		"tenant_id":  "租户ID",
		"user_id":    "用户ID",
		"event":      "事件类型：login, logout, refresh, revoke, failed_login",
		"ip":         "客户端IP地址",
		"user_agent": "用户代理字符串",
		"meta":       "事件元数据（JSONB格式）",
		"created_at": "事件发生时间",
	}
	if err := m.addColumnCommentsForTable("auth_audit", auditComments); err != nil {
		return err
	}

	// EmailVerificationToken 表列注释
	verificationComments := map[string]string{
		"id":         "验证令牌唯一标识符（UUID）",
		"user_id":    "用户ID",
		"tenant_id":  "租户ID",
		"token":      "验证令牌（随机生成的UUID）",
		"email":      "邮箱地址",
		"used":       "是否已使用",
		"created_at": "创建时间",
		"expires_at": "过期时间",
	}
	if err := m.addColumnCommentsForTable("email_verification_tokens", verificationComments); err != nil {
		return err
	}

	return nil
}

// addColumnCommentsForTable 为指定表的列添加注释
func (m *AuthMigration) addColumnCommentsForTable(table string, comments map[string]string) error {
	for column, comment := range comments {
		if err := m.db.Exec(fmt.Sprintf(
			"COMMENT ON COLUMN %s.%s IS '%s'",
			table, column, comment,
		)).Error; err != nil {
			return fmt.Errorf("添加表 %s 列 %s 注释失败: %w", table, column, err)
		}
	}
	return nil
}

// GetName 获取迁移名称
func (m *AuthMigration) GetName() string {
	return "auth_migration"
}
