package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// 租户类型常量
const (
	// TenantTypeSystem 平台租户类型
	TenantTypeSystem = "system"
	// TenantTypeBusiness 业务租户类型
	TenantTypeBusiness = "tenant"
)

// 角色常量
const (
	// RoleSystemAdmin 平台管理员角色
	RoleSystemAdmin = "system_admin"
	// RoleTenantAdmin 租户管理员角色
	RoleTenantAdmin = "tenant_admin"
	// RoleUser 普通用户角色
	RoleUser = "user"
)

// 审计日志事件类型常量
const (
	// 认证相关事件
	AuditEventLogin        = "login"         // 用户登录
	AuditEventLogout       = "logout"        // 用户登出
	AuditEventRefresh      = "refresh"       // 刷新令牌
	AuditEventRevoke       = "revoke"        // 撤销令牌
	AuditEventFailedLogin  = "failed_login"  // 登录失败
	
	// 租户管理事件
	AuditEventTenantCreated  = "tenant_created"  // 租户创建
	AuditEventTenantDeleted  = "tenant_deleted"  // 租户删除
	AuditEventTenantEnabled  = "tenant_enabled"  // 租户启用
	AuditEventTenantDisabled = "tenant_disabled" // 租户禁用
	
	// 用户管理事件
	AuditEventUserCreated  = "user_created"  // 用户创建
	AuditEventUserDeleted  = "user_deleted"  // 用户删除
	AuditEventUserEnabled  = "user_enabled"  // 用户启用
	AuditEventUserDisabled = "user_disabled" // 用户禁用
)

// Tenant 租户模型
// @Description 租户信息，包含租户的基本信息和状态
// @name Tenant
type Tenant struct {
	// 租户ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 租户名称
	Name string `gorm:"type:varchar(255);not null;uniqueIndex" json:"name" example:"Acme Corporation"`
	// 租户域名，用于子域识别
	Domain string `gorm:"type:varchar(255)" json:"domain" example:"acme.example.com"`
	// 租户类型：system=平台租户，tenant=业务租户
	// 可选值：system（平台租户，系统级租户，只能有一个）, tenant（业务租户，普通租户）
	Type string `gorm:"type:varchar(32);not null;default:'tenant';index" json:"type" example:"tenant" enums:"system,tenant"`
	// 租户元数据，存储租户的自定义信息
	Metadata datatypes.JSON `gorm:"type:jsonb" json:"metadata" swaggertype:"object"`
	// 租户状态：true=启用，false=禁用
	// 禁用的租户下的所有用户将无法登录
	Status bool `gorm:"default:true" json:"status" example:"true"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt" example:"2025-01-20T10:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt" example:"2025-01-20T10:00:00Z"`
	// 创建者用户ID
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"createdBy" example:"660e8400-e29b-41d4-a716-446655440000"`
	// 软删除标记
	IsDeleted bool `gorm:"default:false" json:"isDeleted" example:"false"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}

// User 用户模型
// @Description 用户信息，包含用户的基本信息、角色和状态
// @name User
type User struct {
	// 用户ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id" example:"660e8400-e29b-41d4-a716-446655440001"`
	// 所属租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_email" json:"tenantId" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 用户邮箱，全局唯一
	Email string `gorm:"type:varchar(320);not null;uniqueIndex" json:"email" example:"user@example.com"`
	// 邮箱是否已验证
	EmailVerified bool `gorm:"default:false" json:"emailVerified" example:"false"`
	// 手机号码
	Phone string `gorm:"type:varchar(20)" json:"phone" example:"13800138000"`
	// 密码哈希值（bcrypt），不在API响应中返回
	PasswordHash string `gorm:"type:text;not null" json:"-"`
	// 显示名称
	DisplayName string `gorm:"type:varchar(255)" json:"displayName" example:"张三"`
	// 账户是否激活，禁用的用户无法登录
	IsActive bool `gorm:"default:true" json:"isActive" example:"true"`
	// 是否为管理员（租户管理员或平台管理员）
	IsAdmin bool `gorm:"default:false" json:"isAdmin" example:"false"`
	// 用户角色列表，支持多角色
	// 可选值：system_admin（平台管理员）, tenant_admin（租户管理员）, user（普通用户）
	// 示例：["user"], ["tenant_admin"], ["system_admin"]
	Roles datatypes.JSON `gorm:"type:jsonb" json:"roles" swaggertype:"array,string" example:"[\"user\"]"`
	// 用户元数据，存储用户的自定义信息
	Meta datatypes.JSON `gorm:"type:jsonb" json:"meta" swaggertype:"object"`
	// 最后登录时间
	LastLoginAt *time.Time `json:"lastLoginAt" example:"2025-01-20T10:00:00Z"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt" example:"2025-01-20T10:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt" example:"2025-01-20T10:00:00Z"`
	// 创建者用户ID
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"createdBy" example:"550e8400-e29b-41d4-a716-446655440000"`
	// 软删除标记
	IsDeleted bool `gorm:"default:false" json:"isDeleted" example:"false"`
	// 登录失败次数，用于账户锁定策略
	FailedLoginAttempts int `gorm:"default:0" json:"failedLoginAttempts" example:"0"`
	// 账户锁定时间，锁定期间无法登录
	LockedUntil *time.Time `json:"lockedUntil" example:"2025-01-20T10:00:00Z"`

	// 关联
	Tenant *Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// RefreshToken 刷新令牌模型
type RefreshToken struct {
	// 令牌ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 用户ID
	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_user_tokens" json:"userId"`
	// 租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_tokens" json:"tenantId"`
	// Refresh Token 的 SHA256 哈希值
	TokenHash string `gorm:"type:text;not null;uniqueIndex:idx_token_hash" json:"-"`
	// 是否已撤销
	Revoked bool `gorm:"default:false;index:idx_revoked" json:"revoked"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 过期时间
	ExpiresAt time.Time `gorm:"not null;index:idx_expires" json:"expiresAt"`
	// 轮换时指向新 token 的 ID
	ReplacedBy *uuid.UUID `gorm:"type:uuid" json:"replacedBy"`

	// 关联
	User   *User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// AuthAudit 认证审计日志模型
type AuthAudit struct {
	// 审计日志ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 租户ID
	TenantID *uuid.UUID `gorm:"type:uuid;index:idx_tenant_audit" json:"tenantId"`
	// 用户ID
	UserID *uuid.UUID `gorm:"type:uuid;index:idx_user_audit" json:"userId"`
	// 事件类型：login, logout, refresh, revoke, failed_login
	Event string `gorm:"type:varchar(64);not null;index:idx_event" json:"event"`
	// 客户端IP地址
	IP string `gorm:"type:varchar(45)" json:"ip"`
	// 用户代理字符串
	UserAgent string `gorm:"type:text" json:"userAgent"`
	// 事件元数据
	Meta datatypes.JSON `gorm:"type:jsonb" json:"meta"`
	// 事件发生时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;index:idx_created_at" json:"createdAt"`
}

// TableName 指定表名
func (AuthAudit) TableName() string {
	return "auth_audit"
}

// EmailVerificationToken 邮箱验证令牌模型
type EmailVerificationToken struct {
	// 令牌ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 用户ID
	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_user_verification" json:"userId"`
	// 租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_verification" json:"tenantId"`
	// 验证令牌（随机生成的UUID）
	Token string `gorm:"type:varchar(64);not null;uniqueIndex:idx_verification_token" json:"-"`
	// 邮箱地址
	Email string `gorm:"type:varchar(320);not null" json:"email"`
	// 是否已使用
	Used bool `gorm:"default:false;index:idx_used" json:"used"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 过期时间
	ExpiresAt time.Time `gorm:"not null;index:idx_verification_expires" json:"expiresAt"`

	// 关联
	User   *User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}
