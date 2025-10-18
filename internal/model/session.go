package model

import (
	"time"

	"gorm.io/datatypes"
)

// ChatSession 会话实体
type ChatSession struct {
	// 会话ID
	ID string `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 用户ID
	UserID string `gorm:"type:uuid;not null;index:idx_user_sessions" json:"userId"`
	// 会话标题
	Title string `gorm:"type:varchar(255);not null" json:"title"`
	// 模型名称
	ModelName string `gorm:"type:varchar(128);not null" json:"modelName"`
	// 系统提示词
	SystemPrompt string `gorm:"type:text" json:"systemPrompt"`
	// 温度参数
	Temperature *float64 `gorm:"type:float" json:"temperature"`
	// TopP参数
	TopP *float64 `gorm:"type:float" json:"topP"`
	// 创建者ID
	CreatedBy string `gorm:"type:uuid;not null" json:"createdBy"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
	// 最后一条消息ID
	LastMessageID *string `gorm:"type:uuid" json:"lastMessageId"`
	// 消息数量
	MessageCount int `gorm:"default:0" json:"messageCount"`
	// 是否置顶
	IsPinned bool `gorm:"default:false;index:idx_pinned" json:"isPinned"`
	// 是否归档
	IsArchived bool `gorm:"default:false;index:idx_archived" json:"isArchived"`
	// 是否删除
	IsDeleted bool `gorm:"default:false;index:idx_deleted" json:"isDeleted"`
	// 元数据
	Meta datatypes.JSON `gorm:"type:jsonb" json:"meta"`
}

// TableName 指定表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage 消息实体
type ChatMessage struct {
	// 消息ID
	ID string `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 会话ID
	SessionID string `gorm:"type:uuid;not null;index:idx_session_messages" json:"sessionId"`
	// 角色 (user, assistant, system, function)
	Role string `gorm:"type:varchar(32);not null" json:"role"`
	// 消息内容
	Content string `gorm:"type:text;not null" json:"content"`
	// Token数量
	Tokens int `gorm:"default:0" json:"tokens"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_created" json:"createdAt"`
	// 消息序列号
	Sequence int `gorm:"not null" json:"sequence"`
	// 工具调用信息
	ToolCalls datatypes.JSON `gorm:"type:jsonb" json:"toolCalls"`
	// 错误信息
	Error string `gorm:"type:text" json:"error"`
	// 父消息ID
	ParentID *string `gorm:"type:uuid" json:"parentId"`
	// 元数据
	Meta datatypes.JSON `gorm:"type:jsonb" json:"meta"`

	// 关联
	Session *ChatSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// ChatSummary 会话摘要实体
type ChatSummary struct {
	// 摘要ID
	ID string `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 会话ID
	SessionID string `gorm:"type:uuid;not null;index:idx_session_summary" json:"sessionId"`
	// 摘要内容
	Summary string `gorm:"type:text;not null" json:"summary"`
	// 最后一条消息ID
	LastMessageID string `gorm:"type:uuid;not null" json:"lastMessageId"`
	// Token数量
	TokenCount int `gorm:"default:0" json:"tokenCount"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`

	// 关联
	Session *ChatSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChatSummary) TableName() string {
	return "chat_summaries"
}
