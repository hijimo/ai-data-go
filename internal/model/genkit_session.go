package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ConversationMemory 对话记忆模型
// 存储长期记忆，支持向量检索
type ConversationMemory struct {
	ID           uuid.UUID              `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID              `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"tenantId"`
	SessionID    uuid.UUID              `gorm:"type:uuid;not null;index:idx_memories_tenant_session" json:"sessionId"`
	MemoryType   string                 `gorm:"type:varchar(50);not null;index:idx_memories_type" json:"memoryType"` // 'short_term', 'long_term', 'summary'
	Content      string                 `gorm:"type:text;not null" json:"content"`
	Embedding    pgvector.Vector        `gorm:"type:vector(1536)" json:"-"` // 向量维度根据嵌入模型确定
	TokenCount   int                    `gorm:"not null;default:0" json:"tokenCount"`
	Importance   float32                `gorm:"not null;default:0.5" json:"importance"` // 0-1
	AccessCount  int                    `gorm:"not null;default:0" json:"accessCount"`
	LastAccessAt *time.Time             `json:"lastAccessAt,omitempty"`
	Metadata     map[string]interface{} `gorm:"type:jsonb" json:"metadata,omitempty"`
	ExpiresAt    *time.Time             `gorm:"index:idx_memories_expires" json:"expiresAt,omitempty"`
	IsDeleted    bool                   `gorm:"not null;default:false" json:"-"`
	CreatedAt    time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_memories_created" json:"createdAt"`
	UpdatedAt    time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 指定表名
func (ConversationMemory) TableName() string {
	return "conversation_memories"
}

// ConversationContext 对话上下文配置模型
// 存储会话的上下文配置和统计信息
type ConversationContext struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_contexts_tenant" json:"tenantId"`
	SessionID       uuid.UUID  `gorm:"type:uuid;not null;unique;index:idx_contexts_session" json:"sessionId"`
	MaxTokens       int        `gorm:"not null;default:4000" json:"maxTokens"`
	Strategy        string     `gorm:"type:varchar(50);not null;default:'auto'" json:"strategy"` // 'auto', 'short', 'full'
	IncludeSummary  bool       `gorm:"not null;default:true" json:"includeSummary"`
	IncludeLongTerm bool       `gorm:"not null;default:true" json:"includeLongTerm"`
	ShortTermWindow int        `gorm:"not null;default:10" json:"shortTermWindow"`
	LastSummaryID   *uuid.UUID `json:"lastSummaryId,omitempty"`
	LastSummaryAt   *time.Time `json:"lastSummaryAt,omitempty"`
	TotalMessages   int        `gorm:"not null;default:0" json:"totalMessages"`
	TotalTokensUsed int64      `gorm:"not null;default:0" json:"totalTokensUsed"`
	IsDeleted       bool       `gorm:"not null;default:false" json:"-"`
	CreatedAt       time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 指定表名
func (ConversationContext) TableName() string {
	return "conversation_contexts"
}

// ConversationSummary 对话摘要模型
// 存储会话摘要，用于压缩历史对话
type ConversationSummary struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_summaries_tenant_session" json:"tenantId"`
	SessionID         uuid.UUID  `gorm:"type:uuid;not null;index:idx_summaries_tenant_session,idx_summaries_session_latest" json:"sessionId"`
	SummaryType       string     `gorm:"type:varchar(50);not null" json:"summaryType"` // 'incremental', 'full'
	Content           string     `gorm:"type:text;not null" json:"content"`
	TokenCount        int        `gorm:"not null" json:"tokenCount"`
	MessageCount      int        `gorm:"not null" json:"messageCount"`
	StartMessageID    *uuid.UUID `json:"startMessageId,omitempty"`
	EndMessageID      *uuid.UUID `json:"endMessageId,omitempty"`
	QualityScore      *float64   `json:"qualityScore,omitempty"`      // 0-1
	CompressionRate   *float64   `json:"compressionRate,omitempty"`   // 0-1
	KeyTopics         []string   `gorm:"type:text[]" json:"keyTopics,omitempty"`
	PreviousSummaryID *uuid.UUID `json:"previousSummaryId,omitempty"`
	IsDeleted         bool       `gorm:"not null;default:false" json:"-"`
	CreatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_summaries_created,idx_summaries_session_latest" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 指定表名
func (ConversationSummary) TableName() string {
	return "conversation_summaries"
}

// MemoryType 记忆类型常量
const (
	MemoryTypeShortTerm = "short_term"
	MemoryTypeLongTerm  = "long_term"
	MemoryTypeSummary   = "summary"
)

// ContextStrategy 上下文策略常量
const (
	ContextStrategyAuto  = "auto"
	ContextStrategyShort = "short"
	ContextStrategyFull  = "full"
)

// SummaryType 摘要类型常量
const (
	SummaryTypeIncremental = "incremental"
	SummaryTypeFull        = "full"
)
