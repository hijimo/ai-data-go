package model

import (
	"time"

	"github.com/google/uuid"
)

// ModelConfiguration 模型配置实体
type ModelConfiguration struct {
	// 主键，UUID类型
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// 租户ID，外键关联
	TenantID uuid.UUID `gorm:"type:uuid;not null;index:idx_tenant_provider" json:"tenantId"`

	// 配置名称
	Name string `gorm:"type:varchar(255);not null" json:"name"`

	// 模型标识（如：gpt-4、claude-3-opus等）
	Model string `gorm:"type:varchar(255);not null" json:"model"`

	// 模型提供商枚举
	ModelProvider string `gorm:"type:varchar(50);not null;index:idx_tenant_provider" json:"modelProvider"`

	// API基础URL（可选，用于自定义端点）
	BaseURL *string `gorm:"type:varchar(500)" json:"baseUrl,omitempty"`

	// API密钥（加密存储）
	APIKey string `gorm:"type:text;not null" json:"-"`

	// 查询参数（JSON格式，可选）
	QueryParams *string `gorm:"type:jsonb" json:"queryParams,omitempty"`

	// 是否启用
	IsEnabled bool `gorm:"default:true;not null" json:"isEnabled"`

	// 软删除标记
	IsDeleted bool `gorm:"default:false;not null;index" json:"-"`

	// 创建信息
	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`

	// 更新信息
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`

	// 删除信息
	DeletedBy *uuid.UUID `gorm:"type:uuid" json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// TableName 指定表名
func (ModelConfiguration) TableName() string {
	return "model_configurations"
}

// ModelProvider 枚举常量
const (
	ModelProviderOpenAI       = "openai"
	ModelProviderAnthropic    = "anthropic"
	ModelProviderGoogleGenAI  = "googlegenai"
	ModelProviderAzureOpenAI  = "azureopenai"
	ModelProviderBianlian     = "bianlian"
	ModelProviderCustomOpenAI = "custom_openai"
)

// ValidModelProviders 有效的模型提供商列表
var ValidModelProviders = []string{
	ModelProviderOpenAI,
	ModelProviderAnthropic,
	ModelProviderGoogleGenAI,
	ModelProviderAzureOpenAI,
	ModelProviderBianlian,
	ModelProviderCustomOpenAI,
}

// IsValidModelProvider 验证模型提供商是否有效
func IsValidModelProvider(provider string) bool {
	for _, valid := range ValidModelProviders {
		if provider == valid {
			return true
		}
	}
	return false
}

// CreateModelConfigurationRequest 创建模型配置请求
type CreateModelConfigurationRequest struct {
	TenantID      *uuid.UUID `json:"tenantId,omitempty"` // 仅system_admin需要
	Name          string     `json:"name" binding:"required"`
	Model         string     `json:"model" binding:"required"`
	ModelProvider string     `json:"modelProvider" binding:"required,oneof=openai anthropic googlegenai azureopenai bianlian custom_openai"`
	BaseURL       *string    `json:"baseUrl,omitempty"`
	APIKey        string     `json:"apiKey" binding:"required"`
	QueryParams   *string    `json:"queryParams,omitempty"`
}

// UpdateModelConfigurationRequest 更新模型配置请求
type UpdateModelConfigurationRequest struct {
	Name        *string `json:"name,omitempty"`
	Model       *string `json:"model,omitempty"`
	BaseURL     *string `json:"baseUrl,omitempty"`
	APIKey      *string `json:"apiKey,omitempty"`
	QueryParams *string `json:"queryParams,omitempty"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled"`
}

// ModelConfigurationResponse 模型配置响应
type ModelConfigurationResponse struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenantId"`
	Name          string     `json:"name"`
	Model         string     `json:"model"`
	ModelProvider string     `json:"modelProvider"`
	BaseURL       *string    `json:"baseUrl,omitempty"`
	APIKey        string     `json:"apiKey"` // 脱敏后的密钥
	QueryParams   *string    `json:"queryParams,omitempty"`
	IsEnabled     bool       `json:"isEnabled"`
	CreatedBy     uuid.UUID  `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedBy     *uuid.UUID `json:"updatedBy,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

// AvailableModelConfigurationResponse 可用模型配置响应
type AvailableModelConfigurationResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Model         string    `json:"model"`
	ModelProvider string    `json:"modelProvider"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
