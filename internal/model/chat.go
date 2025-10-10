package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatSession 对话会话模型
type ChatSession struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID uuid.UUID      `json:"project_id" gorm:"type:uuid;not null"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	AgentID   *uuid.UUID     `json:"agent_id" gorm:"type:uuid"`
	Title     *string        `json:"title" gorm:"size:255"`
	Context   map[string]any `json:"context" gorm:"type:jsonb;default:'{}'"`
	IsDeleted bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt *time.Time     `json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// 关联关系
	Project  *Project      `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Agent    *Agent        `json:"agent,omitempty" gorm:"foreignKey:AgentID"`
	Messages []ChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID"`
}

// ChatMessage 对话消息模型
type ChatMessage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	SessionID uuid.UUID      `json:"session_id" gorm:"type:uuid;not null"`
	Role      string         `json:"role" gorm:"not null;size:20" validate:"required,oneof=user assistant system"`
	Content   string         `json:"content" gorm:"type:text;not null" validate:"required"`
	Metadata  map[string]any `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	IsDeleted bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt *time.Time     `json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`

	// 关联关系
	Session *ChatSession `json:"session,omitempty" gorm:"foreignKey:SessionID"`
}

// 消息角色常量
const (
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
	ChatMessageRoleSystem    = "system"
)

// TableName 指定表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// BeforeCreate GORM钩子 - 创建前
func (cs *ChatSession) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (cm *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除对话会话
func (cs *ChatSession) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	cs.IsDeleted = true
	cs.DeletedAt = &now
	return tx.Save(cs).Error
}

// SoftDelete 软删除对话消息
func (cm *ChatMessage) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	cm.IsDeleted = true
	cm.DeletedAt = &now
	return tx.Save(cm).Error
}