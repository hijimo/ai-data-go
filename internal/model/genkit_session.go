package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ConversationMemory 会话记忆实体
type ConversationMemory struct {
	// 记忆ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"tenantId"`
	// 会话ID
	SessionID uuid.UUID `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"sessionId"`
	// 记忆类型 (short_term, long_term, summary)
	MemoryType string `gorm:"type:varchar(50);not null;index:idx_memories_type" json:"memoryType"`
	// 记忆内容
	Content string `gorm:"type:text;not null" json:"content"`
	// 注意：向量数据存储在 Qdrant 中，使用 memory_id 作为关联
	// Token数量
	TokenCount int `gorm:"not null;default:0" json:"tokenCount"`
	// 重要性评分 (0-1)
	Importance float32 `gorm:"not null;default:0.5" json:"importance"`
	// 访问次数
	AccessCount int `gorm:"not null;default:0" json:"accessCount"`
	// 最后访问时间
	LastAccessAt *time.Time `json:"lastAccessAt,omitempty"`
	// 元数据
	Metadata datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	// 过期时间
	ExpiresAt *time.Time `gorm:"index:idx_memories_expires" json:"expiresAt,omitempty"`
	// 软删除标记
	IsDeleted bool `gorm:"not null;default:false" json:"-"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_memories_created" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`

	// 关联
	Tenant  *Tenant      `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	Session *ChatSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ConversationMemory) TableName() string {
	return "conversation_memories"
}

// ConversationContext 会话上下文配置实体
type ConversationContext struct {
	// 上下文配置ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_contexts_tenant" json:"tenantId"`
	// 会话ID（唯一）
	SessionID uuid.UUID `gorm:"type:uuid;not null;unique;index:idx_contexts_session" json:"sessionId"`
	// 最大Token数量
	MaxTokens int `gorm:"not null;default:4000" json:"maxTokens"`
	// 上下文策略 (auto, short, full)
	Strategy string `gorm:"type:varchar(50);not null;default:'auto'" json:"strategy"`
	// 是否包含摘要
	IncludeSummary bool `gorm:"not null;default:true" json:"includeSummary"`
	// 是否包含长期记忆
	IncludeLongTerm bool `gorm:"not null;default:true" json:"includeLongTerm"`
	// 短期记忆窗口大小
	ShortTermWindow int `gorm:"not null;default:10" json:"shortTermWindow"`
	// 最后摘要ID
	LastSummaryID *uuid.UUID `json:"lastSummaryId,omitempty"`
	// 最后摘要时间
	LastSummaryAt *time.Time `json:"lastSummaryAt,omitempty"`
	// 总消息数
	TotalMessages int `gorm:"not null;default:0" json:"totalMessages"`
	// 总Token使用量
	TotalTokensUsed int64 `gorm:"not null;default:0" json:"totalTokensUsed"`
	// 软删除标记
	IsDeleted bool `gorm:"not null;default:false" json:"-"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_contexts_updated" json:"updatedAt"`

	// 关联
	Tenant      *Tenant               `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	Session     *ChatSession          `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
	LastSummary *ConversationSummary  `gorm:"foreignKey:LastSummaryID;constraint:OnDelete:SET NULL" json:"-"`
}

// TableName 指定表名
func (ConversationContext) TableName() string {
	return "conversation_contexts"
}

// ConversationSummary 会话摘要实体
type ConversationSummary struct {
	// 摘要ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	// 租户ID
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_summaries_tenant_session" json:"tenantId"`
	// 会话ID
	SessionID uuid.UUID `gorm:"type:uuid;not null;index:idx_summaries_tenant_session,idx_summaries_session_latest" json:"sessionId"`
	// 摘要类型 (incremental, full)
	SummaryType string `gorm:"type:varchar(50);not null" json:"summaryType"`
	// 摘要内容
	Content string `gorm:"type:text;not null" json:"content"`
	// Token数量
	TokenCount int `gorm:"not null" json:"tokenCount"`
	// 包含的消息数量
	MessageCount int `gorm:"not null" json:"messageCount"`
	// 起始消息ID
	StartMessageID *uuid.UUID `json:"startMessageId,omitempty"`
	// 结束消息ID
	EndMessageID *uuid.UUID `json:"endMessageId,omitempty"`
	// 质量评分 (0-1)
	QualityScore *float64 `json:"qualityScore,omitempty"`
	// 压缩率 (0-1)
	CompressionRate *float64 `json:"compressionRate,omitempty"`
	// 关键主题数组
	KeyTopics []string `gorm:"type:text[]" json:"keyTopics,omitempty"`
	// 前一个摘要ID
	PreviousSummaryID *uuid.UUID `json:"previousSummaryId,omitempty"`
	// 软删除标记
	IsDeleted bool `gorm:"not null;default:false" json:"-"`
	// 创建时间
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_summaries_created,idx_summaries_session_latest" json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`

	// 关联
	Tenant          *Tenant              `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	Session         *ChatSession         `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"-"`
	StartMessage    *ChatMessage         `gorm:"foreignKey:StartMessageID;constraint:OnDelete:SET NULL" json:"-"`
	EndMessage      *ChatMessage         `gorm:"foreignKey:EndMessageID;constraint:OnDelete:SET NULL" json:"-"`
	PreviousSummary *ConversationSummary `gorm:"foreignKey:PreviousSummaryID;constraint:OnDelete:SET NULL" json:"-"`
}

// TableName 指定表名
func (ConversationSummary) TableName() string {
	return "conversation_summaries"
}
