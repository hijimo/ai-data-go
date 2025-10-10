package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Project 项目模型
type Project struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string         `json:"name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	Description *string        `json:"description" gorm:"type:text"`
	OwnerID     uuid.UUID      `json:"owner_id" gorm:"type:uuid;not null"`
	Settings    map[string]any `json:"settings" gorm:"type:jsonb;default:'{}'"`
	IsDeleted   bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt   *time.Time     `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// 关联关系
	Members []ProjectMember `json:"members,omitempty" gorm:"foreignKey:ProjectID"`
}

// ProjectMember 项目成员模型
type ProjectMember struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID uuid.UUID  `json:"project_id" gorm:"type:uuid;not null"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	Role      string     `json:"role" gorm:"not null;size:50;default:'member'" validate:"required,oneof=owner admin member viewer"`
	IsDeleted bool       `json:"is_deleted" gorm:"default:false"`
	DeletedAt *time.Time `json:"deleted_at"`
	CreatedAt time.Time  `json:"created_at"`

	// 关联关系
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// ProjectRole 项目角色常量
type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "owner"
	ProjectRoleAdmin  ProjectRole = "admin"
	ProjectRoleMember ProjectRole = "member"
	ProjectRoleViewer ProjectRole = "viewer"
)

// TableName 指定表名
func (Project) TableName() string {
	return "projects"
}

// TableName 指定表名
func (ProjectMember) TableName() string {
	return "project_members"
}

// BeforeCreate GORM钩子 - 创建前
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (pm *ProjectMember) BeforeCreate(tx *gorm.DB) error {
	if pm.ID == uuid.Nil {
		pm.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除项目
func (p *Project) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	p.IsDeleted = true
	p.DeletedAt = &now
	return tx.Save(p).Error
}

// SoftDelete 软删除项目成员
func (pm *ProjectMember) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	pm.IsDeleted = true
	pm.DeletedAt = &now
	return tx.Save(pm).Error
}

// IsOwner 检查是否为项目所有者
func (p *Project) IsOwner(userID uuid.UUID) bool {
	return p.OwnerID == userID
}

// HasMember 检查用户是否为项目成员
func (p *Project) HasMember(userID uuid.UUID) bool {
	if p.IsOwner(userID) {
		return true
	}
	
	for _, member := range p.Members {
		if member.UserID == userID && !member.IsDeleted {
			return true
		}
	}
	return false
}

// GetMemberRole 获取用户在项目中的角色
func (p *Project) GetMemberRole(userID uuid.UUID) ProjectRole {
	if p.IsOwner(userID) {
		return ProjectRoleOwner
	}
	
	for _, member := range p.Members {
		if member.UserID == userID && !member.IsDeleted {
			return ProjectRole(member.Role)
		}
	}
	return ""
}

// CanRead 检查用户是否有读取权限
func (p *Project) CanRead(userID uuid.UUID) bool {
	return p.HasMember(userID)
}

// CanWrite 检查用户是否有写入权限
func (p *Project) CanWrite(userID uuid.UUID) bool {
	role := p.GetMemberRole(userID)
	return role == ProjectRoleOwner || role == ProjectRoleAdmin || role == ProjectRoleMember
}

// CanDelete 检查用户是否有删除权限
func (p *Project) CanDelete(userID uuid.UUID) bool {
	return p.IsOwner(userID)
}

// CanManageMembers 检查用户是否可以管理成员
func (p *Project) CanManageMembers(userID uuid.UUID) bool {
	role := p.GetMemberRole(userID)
	return role == ProjectRoleOwner || role == ProjectRoleAdmin
}