package model

// Provider 提供商完整信息
// @name Provider
type Provider struct {
	// 提供商ID（从文件夹名称获取）
	ID string `yaml:"-" json:"id" example:"gemini"`
	// 提供商标识
	Provider string `yaml:"provider" json:"provider" example:"gemini"`
	// 多语言标签
	Label map[string]string `yaml:"label" json:"label" example:"en_US:Google Gemini,zh_Hans:谷歌 Gemini"`
	// 背景色
	Background string `yaml:"background" json:"background" example:"#4285F4"`
	// 小图标（多语言）
	IconSmall map[string]string `yaml:"icon_small" json:"icon_small"`
	// 大图标（多语言）
	IconLarge map[string]string `yaml:"icon_large" json:"icon_large"`
	// 帮助信息
	Help ProviderHelp `yaml:"help" json:"help"`
	// 配置方法列表
	ConfigurateMethods []string `yaml:"configurate_methods" json:"configurate_methods" example:"predefined-model,customizable-model"`
	// 支持的模型类型列表
	SupportedModelTypes []string `yaml:"supported_model_types" json:"supported_model_types" example:"llm,text-embedding"`
	// 提供商凭证配置
	ProviderCredentialSchema CredentialSchema `yaml:"provider_credential_schema" json:"provider_credential_schema"`
	// 模型凭证配置
	ModelCredentialSchema CredentialSchema `yaml:"model_credential_schema" json:"model_credential_schema"`
	// 模型类型配置
	Models map[string]ModelTypeInfo `yaml:"models" json:"models"`
}

// ProviderListItem 提供商列表项（用于列表接口）
type ProviderListItem struct {
	// 提供商ID
	ID string `json:"id"`
	// 提供商标识
	Provider string `json:"provider"`
	// 多语言标签
	Label map[string]string `json:"label"`
	// 背景色
	Background string `json:"background"`
	// 小图标（多语言）
	IconSmall map[string]string `json:"icon_small"`
	// 大图标（多语言）
	IconLarge map[string]string `json:"icon_large"`
	// 帮助信息
	Help ProviderHelp `json:"help"`
	// 配置方法列表
	ConfigurateMethods []string `json:"configurate_methods"`
}

// ProviderHelp 提供商帮助信息
type ProviderHelp struct {
	// 帮助标题（多语言）
	Title map[string]string `yaml:"title" json:"title"`
	// 帮助链接（多语言）
	URL map[string]string `yaml:"url" json:"url"`
}

// CredentialSchema 凭证配置
type CredentialSchema struct {
	// 凭证表单配置列表
	CredentialFormSchemas []CredentialFormSchema `yaml:"credential_form_schemas" json:"credential_form_schemas"`
}

// CredentialFormSchema 凭证表单配置项
type CredentialFormSchema struct {
	// 变量名
	Variable string `yaml:"variable" json:"variable"`
	// 标签（多语言）
	Label map[string]string `yaml:"label" json:"label"`
	// 类型
	Type string `yaml:"type" json:"type"`
	// 是否必填
	Required bool `yaml:"required" json:"required"`
	// 默认值
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
	// 占位符（多语言）
	Placeholder map[string]string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	// 选项列表
	Options []FormOption `yaml:"options,omitempty" json:"options,omitempty"`
}

// FormOption 表单选项
type FormOption struct {
	// 选项标签（多语言）
	Label map[string]string `yaml:"label" json:"label"`
	// 选项值
	Value string `yaml:"value" json:"value"`
}

// ModelTypeInfo 模型类型信息
type ModelTypeInfo struct {
	// 位置文件路径
	Position string `yaml:"position,omitempty" json:"position,omitempty"`
	// 预定义模型列表
	Predefined []string `yaml:"predefined" json:"predefined"`
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

// ModelConfiguration 模型配置实体
// @name ModelConfiguration
type ModelConfiguration struct {
	// 主键，UUID类型
	ID string `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// 租户ID，外键关联
	TenantID string `gorm:"type:uuid;not null;index:idx_tenant_provider" json:"tenantId"`

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
	CreatedBy string `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt string `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"createdAt"`

	// 更新信息
	UpdatedBy *string `gorm:"type:uuid" json:"updatedBy,omitempty"`
	UpdatedAt *string `gorm:"type:timestamp" json:"updatedAt,omitempty"`

	// 删除信息
	DeletedBy *string `gorm:"type:uuid" json:"-"`
	DeletedAt *string `gorm:"type:timestamp" json:"-"`
}

// TableName 指定表名
func (ModelConfiguration) TableName() string {
	return "model_configurations"
}
