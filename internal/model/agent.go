package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Agent Agent模型
type Agent struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProjectID    uuid.UUID      `json:"project_id" gorm:"type:uuid;not null"`
	Name         string         `json:"name" gorm:"not null;size:255" validate:"required,min=1,max=255"`
	Description  *string        `json:"description" gorm:"type:text"`
	SystemPrompt *string        `json:"system_prompt" gorm:"type:text"`
	LLMModelID   uuid.UUID      `json:"llm_model_id" gorm:"type:uuid;not null"`
	Tools        []interface{}  `json:"tools" gorm:"type:jsonb;default:'[]'"`
	Config       map[string]any `json:"config" gorm:"type:jsonb;default:'{}'"`
	CreatedBy    uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
	IsDeleted    bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time     `json:"deleted_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联关系
	Project  *Project  `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	LLMModel *LLMModel `json:"llm_model,omitempty" gorm:"foreignKey:LLMModelID"`
}

// LLMProvider LLM提供商模型
type LLMProvider struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name         string         `json:"name" gorm:"not null;size:100" validate:"required,min=1,max=100"`
	ProviderType string         `json:"provider_type" gorm:"not null;size:50" validate:"required"`
	Config       map[string]any `json:"config" gorm:"type:jsonb;not null"` // 加密存储的配置
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	IsDeleted    bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time     `json:"deleted_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联关系
	Models []LLMModel `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
}

// LLMModel LLM模型
type LLMModel struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProviderID  uuid.UUID      `json:"provider_id" gorm:"type:uuid;not null"`
	ModelName   string         `json:"model_name" gorm:"not null;size:100" validate:"required"`
	DisplayName string         `json:"display_name" gorm:"not null;size:200" validate:"required"`
	ModelType   string         `json:"model_type" gorm:"not null;size:50" validate:"required"` // chat, completion, embedding
	Config      map[string]any `json:"config" gorm:"type:jsonb;default:'{}'"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	IsDeleted   bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt   *time.Time     `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`

	// 关联关系
	Provider *LLMProvider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// LLM提供商类型常量
const (
	LLMProviderOpenAI  = "openai"
	LLMProviderAzure   = "azure"
	LLMProviderQianwen = "qianwen"
	LLMProviderClaude  = "claude"
)

// LLM模型类型常量
const (
	LLMModelTypeChat       = "chat"
	LLMModelTypeCompletion = "completion"
	LLMModelTypeEmbedding  = "embedding"
)

// TableName 指定表名
func (Agent) TableName() string {
	return "agents"
}

// TableName 指定表名
func (LLMProvider) TableName() string {
	return "llm_providers"
}

// TableName 指定表名
func (LLMModel) TableName() string {
	return "llm_models"
}

// BeforeCreate GORM钩子 - 创建前
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (lp *LLMProvider) BeforeCreate(tx *gorm.DB) error {
	if lp.ID == uuid.Nil {
		lp.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (lm *LLMModel) BeforeCreate(tx *gorm.DB) error {
	if lm.ID == uuid.Nil {
		lm.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除Agent
func (a *Agent) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	a.IsDeleted = true
	a.DeletedAt = &now
	return tx.Save(a).Error
}

// SoftDelete 软删除LLM提供商
func (lp *LLMProvider) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	lp.IsDeleted = true
	lp.DeletedAt = &now
	return tx.Save(lp).Error
}

// SoftDelete 软删除LLM模型
func (lm *LLMModel) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	lm.IsDeleted = true
	lm.DeletedAt = &now
	return tx.Save(lm).Error
}