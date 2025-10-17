package model

// Model 模型完整信息
type Model struct {
	// 模型标识
	Model string `yaml:"model" json:"model"`
	// 多语言标签
	Label map[string]string `yaml:"label" json:"label"`
	// 模型类型（llm、tts、text_embedding等）
	ModelType string `yaml:"model_type" json:"model_type"`
	// 特性列表
	Features []string `yaml:"features,omitempty" json:"features,omitempty"`
	// 模型属性
	ModelProperties ModelProperties `yaml:"model_properties" json:"model_properties"`
	// 参数规则
	ParameterRules []ParameterRule `yaml:"parameter_rules,omitempty" json:"parameter_rules,omitempty"`
	// 定价信息
	Pricing Pricing `yaml:"pricing,omitempty" json:"pricing,omitempty"`
	// 是否已弃用
	Deprecated bool `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
}

// ModelListItem 模型列表项（用于列表接口）
type ModelListItem struct {
	// 模型标识
	Model string `json:"model"`
	// 多语言标签
	Label map[string]string `json:"label"`
	// 模型类型
	ModelType string `json:"model_type"`
	// 特性列表
	Features []string `json:"features,omitempty"`
	// 模型属性
	ModelProperties ModelProperties `json:"model_properties"`
	// 参数规则
	ParameterRules []ParameterRule `json:"parameter_rules,omitempty"`
	// 定价信息
	Pricing Pricing `json:"pricing,omitempty"`
}

// ModelProperties 模型属性
type ModelProperties struct {
	// 模式（chat、completion等）
	Mode string `yaml:"mode" json:"mode"`
	// 上下文大小
	ContextSize int `yaml:"context_size" json:"context_size"`
}

// ParameterRule 参数规则
type ParameterRule struct {
	// 参数名称
	Name string `yaml:"name" json:"name"`
	// 使用的模板
	UseTemplate string `yaml:"use_template,omitempty" json:"use_template,omitempty"`
	// 标签（多语言）
	Label map[string]string `yaml:"label,omitempty" json:"label,omitempty"`
	// 类型
	Type string `yaml:"type" json:"type"`
	// 是否必填
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`
	// 默认值
	Default interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	// 最小值
	Min interface{} `yaml:"min,omitempty" json:"min,omitempty"`
	// 最大值
	Max interface{} `yaml:"max,omitempty" json:"max,omitempty"`
	// 帮助信息（多语言）
	Help map[string]string `yaml:"help,omitempty" json:"help,omitempty"`
	// 选项列表
	Options []string `yaml:"options,omitempty" json:"options,omitempty"`
}

// Pricing 定价信息
type Pricing struct {
	// 输入价格
	Input string `yaml:"input" json:"input"`
	// 输出价格
	Output string `yaml:"output" json:"output"`
	// 单位
	Unit string `yaml:"unit" json:"unit"`
	// 货币
	Currency string `yaml:"currency" json:"currency"`
}
