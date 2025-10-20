package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Tenant 租户模型
type Tenant struct {
	// 租户ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 租户名称
	Name string `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	// 租户域名，用于子域识别
	Domain string `gorm:"type:varchar(255)" json:"domain"`
	// 租户元数据
	Metadata datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	// 租户状态：true=启用，false=禁用
	Status bool `gorm:"default:true" json:"status"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
	// 创建者用户ID
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	// 软删除标记
	IsDeleted bool `gorm:"default:false" json:"isDeleted"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}

// User 用户模型
type User struct {
	// 用户ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 所属租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_tenant_email" json:"tenantId"`
	// 用户邮箱
	Email string `gorm:"type:varchar(320);not null;uniqueIndex:idx_tenant_email" json:"email"`
	// 邮箱是否已验证
	EmailVerified bool `gorm:"default:false" json:"emailVerified"`
	// 手机号码
	Phone string `gorm:"type:varchar(20)" json:"phone"`
	// 密码哈希值（bcrypt）
	PasswordHash string `gorm:"type:text;not null" json:"-"`
	// 显示名称
	DisplayName string `gorm:"type:varchar(255)" json:"displayName"`
	// 账户是否激活
	IsActive bool `gorm:"default:true" json:"isActive"`
	// 是否为管理员
	IsAdmin bool `gorm:"default:false" json:"isAdmin"`
	// 用户角色列表，如 ["user","admin"]
	Roles datatypes.JSON `gorm:"type:jsonb" json:"roles"`
	// 用户元数据
	Meta datatypes.JSON `gorm:"type:jsonb" json:"meta"`
	// 最后登录时间
	LastLoginAt *time.Time `json:"lastLoginAt"`
	// 创建时间
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
	// 创建者用户ID
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"createdBy"`
	// 软删除标记
	IsDeleted bool `gorm:"default:false" json:"isDeleted"`
	// 登录失败次数
	FailedLoginAttempts int `gorm:"default:0" json:"failedLoginAttempts"`
	// 账户锁定时间
	LockedUntil *time.Time `json:"lockedUntil"`

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
