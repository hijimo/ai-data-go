package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LLMProvider LLM提供商模型
type LLMProvider struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name         string         `json:"name" gorm:"not null;size:100" validate:"required,min=1,max=100"`
	ProviderType string         `json:"provider_type" gorm:"not null;size:50" validate:"required,oneof=openai azure qianwen claude baichuan chatglm"`
	Config       map[string]any `json:"config" gorm:"type:jsonb;not null" validate:"required"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	IsDeleted    bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt    *time.Time     `json:"deleted_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// 关联关系
	Models []LLMModel `json:"models,omitempty" gorm:"foreignKey:ProviderID"`
}

// LLMModel LLM模型模型
type LLMModel struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ProviderID  uuid.UUID      `json:"provider_id" gorm:"type:uuid;not null"`
	ModelName   string         `json:"model_name" gorm:"not null;size:100" validate:"required,min=1,max=100"`
	DisplayName string         `json:"display_name" gorm:"not null;size:200" validate:"required,min=1,max=200"`
	ModelType   string         `json:"model_type" gorm:"not null;size:50" validate:"required,oneof=chat completion embedding image audio"`
	Config      map[string]any `json:"config" gorm:"type:jsonb;default:'{}'"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	IsDeleted   bool           `json:"is_deleted" gorm:"default:false"`
	DeletedAt   *time.Time     `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`

	// 关联关系
	Provider *LLMProvider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// LLMProviderType 提供商类型常量
type LLMProviderType string

const (
	LLMProviderOpenAI    LLMProviderType = "openai"
	LLMProviderAzure     LLMProviderType = "azure"
	LLMProviderQianwen   LLMProviderType = "qianwen"
	LLMProviderClaude    LLMProviderType = "claude"
	LLMProviderBaichuan  LLMProviderType = "baichuan"
	LLMProviderChatGLM   LLMProviderType = "chatglm"
)

// LLMModelType 模型类型常量
type LLMModelType string

const (
	LLMModelTypeChat       LLMModelType = "chat"
	LLMModelTypeCompletion LLMModelType = "completion"
	LLMModelTypeEmbedding  LLMModelType = "embedding"
	LLMModelTypeImage      LLMModelType = "image"
	LLMModelTypeAudio      LLMModelType = "audio"
)

// TableName 指定表名
func (LLMProvider) TableName() string {
	return "llm_providers"
}

// TableName 指定表名
func (LLMModel) TableName() string {
	return "llm_models"
}

// BeforeCreate GORM钩子 - 创建前
func (p *LLMProvider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// BeforeCreate GORM钩子 - 创建前
func (m *LLMModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除提供商
func (p *LLMProvider) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	p.IsDeleted = true
	p.DeletedAt = &now
	return tx.Save(p).Error
}

// SoftDelete 软删除模型
func (m *LLMModel) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	m.IsDeleted = true
	m.DeletedAt = &now
	return tx.Save(m).Error
}

// IsValidProviderType 检查是否为有效的提供商类型
func IsValidProviderType(providerType string) bool {
	validTypes := []string{
		string(LLMProviderOpenAI),
		string(LLMProviderAzure),
		string(LLMProviderQianwen),
		string(LLMProviderClaude),
		string(LLMProviderBaichuan),
		string(LLMProviderChatGLM),
	}
	
	for _, validType := range validTypes {
		if providerType == validType {
			return true
		}
	}
	return false
}

// IsValidModelType 检查是否为有效的模型类型
func IsValidModelType(modelType string) bool {
	validTypes := []string{
		string(LLMModelTypeChat),
		string(LLMModelTypeCompletion),
		string(LLMModelTypeEmbedding),
		string(LLMModelTypeImage),
		string(LLMModelTypeAudio),
	}
	
	for _, validType := range validTypes {
		if modelType == validType {
			return true
		}
	}
	return false
}

// GetProviderDisplayName 获取提供商显示名称
func GetProviderDisplayName(providerType string) string {
	displayNames := map[string]string{
		string(LLMProviderOpenAI):   "OpenAI",
		string(LLMProviderAzure):    "Azure OpenAI",
		string(LLMProviderQianwen):  "通义千问",
		string(LLMProviderClaude):   "Claude",
		string(LLMProviderBaichuan): "百川智能",
		string(LLMProviderChatGLM):  "智谱清言",
	}
	
	if displayName, exists := displayNames[providerType]; exists {
		return displayName
	}
	return providerType
}

// GetModelTypeDisplayName 获取模型类型显示名称
func GetModelTypeDisplayName(modelType string) string {
	displayNames := map[string]string{
		string(LLMModelTypeChat):       "对话模型",
		string(LLMModelTypeCompletion): "文本补全模型",
		string(LLMModelTypeEmbedding):  "嵌入模型",
		string(LLMModelTypeImage):      "图像模型",
		string(LLMModelTypeAudio):      "音频模型",
	}
	
	if displayName, exists := displayNames[modelType]; exists {
		return displayName
	}
	return modelType
}

// ValidateConfig 验证提供商配置
func (p *LLMProvider) ValidateConfig() error {
	if !IsValidProviderType(p.ProviderType) {
		return ErrInvalidProviderType
	}
	
	// 检查必需的配置字段
	requiredFields := getRequiredConfigFields(p.ProviderType)
	for _, field := range requiredFields {
		if _, exists := p.Config[field]; !exists {
			return fmt.Errorf("缺少必需的配置字段: %s", field)
		}
	}
	
	return nil
}

// ValidateConfig 验证模型配置
func (m *LLMModel) ValidateConfig() error {
	if !IsValidModelType(m.ModelType) {
		return ErrInvalidModelType
	}
	
	return nil
}

// getRequiredConfigFields 获取提供商类型的必需配置字段
func getRequiredConfigFields(providerType string) []string {
	requiredFields := map[string][]string{
		string(LLMProviderOpenAI):   {"api_key"},
		string(LLMProviderAzure):    {"api_key", "resource_name", "deployment"},
		string(LLMProviderQianwen):  {"api_key"},
		string(LLMProviderClaude):   {"api_key"},
		string(LLMProviderBaichuan): {"api_key"},
		string(LLMProviderChatGLM):  {"api_key"},
	}
	
	if fields, exists := requiredFields[providerType]; exists {
		return fields
	}
	return []string{"api_key"}
}

// 错误定义
var (
	ErrInvalidProviderType = errors.New("无效的提供商类型")
	ErrInvalidModelType    = errors.New("无效的模型类型")
	ErrProviderNotFound    = errors.New("提供商未找到")
	ErrModelNotFound       = errors.New("模型未找到")
	ErrProviderInUse       = errors.New("提供商正在使用中，无法删除")
	ErrModelInUse          = errors.New("模型正在使用中，无法删除")
)