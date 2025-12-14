package genkit

import (
	"encoding/json"
	"fmt"
)

// Config Genkit 配置结构
type Config struct {
	// API 密钥
	APIKey string
	// 模型名称
	Model string
	// 默认温度值
	DefaultTemperature float64
	// 默认最大 token 数
	DefaultMaxTokens int
}

// GenkitConfig 用于解析 model_configurations.configuration 的配置结构
// 这个结构体包含了所有提供商可能需要的配置字段
type GenkitConfig struct {
	// Azure OpenAI 特定配置
	AzureEndpoint     string `json:"azureEndpoint,omitempty"`
	AzureDeployment   string `json:"azureDeployment,omitempty"`
	AzureAPIVersion   string `json:"azureApiVersion,omitempty"`
	AzureOrganization string `json:"azureOrganization,omitempty"` // Azure 组织 ID

	// 百炼特定配置
	BailianEndpoint  string `json:"bailianEndpoint,omitempty"`
	BailianWorkspace string `json:"bailianWorkspace,omitempty"`
	BailianRegion    string `json:"bailianRegion,omitempty"` // 百炼地域（beijing, singapore, finance）

	// 通用配置
	Model              string            `json:"model"`
	DefaultTemperature float64           `json:"defaultTemperature,omitempty"`
	DefaultMaxTokens   int               `json:"defaultMaxTokens,omitempty"`
	CustomHeaders      map[string]string `json:"customHeaders,omitempty"` // 自定义请求头
}

// ParseGenkitConfig 从 JSON 字符串解析 GenkitConfig
func ParseGenkitConfig(configJSON string) (*GenkitConfig, error) {
	if configJSON == "" {
		return nil, fmt.Errorf("配置JSON不能为空")
	}

	var config GenkitConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置JSON失败: %w", err)
	}

	return &config, nil
}

// Validate 验证配置的有效性
func (c *GenkitConfig) Validate(providerType string) error {
	// 验证通用字段
	if c.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}

	// 根据提供商类型验证特定字段
	switch providerType {
	case "azureopenai":
		return c.validateAzureConfig()
	case "bianlian":
		return c.validateBailianConfig()
	case "googlegenai", "openai", "anthropic", "custom_openai":
		// 这些提供商只需要通用配置
		return nil
	default:
		return fmt.Errorf("不支持的提供商类型: %s", providerType)
	}
}

// validateAzureConfig 验证 Azure OpenAI 特定配置
// 注意：这里只做基本验证，详细的配置验证已在 ModelConfigurationService 层完成
// Service 层通过实际 HTTP 请求验证配置的有效性，更加可靠
func (c *GenkitConfig) validateAzureConfig() error {
	// Azure OpenAI 的配置验证已在 Service 层完成
	// 这里只需要确保模型名称存在即可
	// azureEndpoint、azureDeployment、azureAPIVersion 等字段是可选的
	// 因为它们可能通过 BaseURL 和 QueryParams 的方式提供
	return nil
}

// validateBailianConfig 验证百炼特定配置
// 注意：这里只做基本验证，详细的配置验证已在 ModelConfigurationService 层完成
// Service 层通过实际 HTTP 请求验证配置的有效性，更加可靠
func (c *GenkitConfig) validateBailianConfig() error {
	// 百炼的配置验证已在 Service 层完成
	// 这里只需要确保模型名称存在即可
	// bailianEndpoint、bailianWorkspace 等字段是可选的
	// 因为它们可能通过 BaseURL 的方式提供
	return nil
}

// ToJSON 将配置转换为 JSON 字符串
func (c *GenkitConfig) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("序列化配置失败: %w", err)
	}
	return string(data), nil
}

// GenerateOptions 生成选项
type GenerateOptions struct {
	// 温度值，控制输出的随机性 (0-2)
	Temperature *float64
	// 最大 token 数
	MaxTokens *int
	// Top-p 采样参数 (0-1)
	TopP *float64
	// Top-k 采样参数
	TopK *int
	// 历史消息（用于多轮对话）
	History []*HistoryMessage
}

// HistoryMessage 历史对话消息
type HistoryMessage struct {
	// 角色：user（用户）或 assistant（AI助手）或 system（系统）
	Role string
	// 消息内容
	Content string
}

// GenerateResult 生成结果
type GenerateResult struct {
	// 生成的文本内容
	Text string
	// 使用的模型
	Model string
	// Token 使用情况
	Usage *Usage
}

// Usage Token 使用情况
type Usage struct {
	// 提示词 token 数
	PromptTokens int
	// 生成内容 token 数
	CompletionTokens int
	// 总 token 数
	TotalTokens int
}

// StreamChunk 流式响应块
type StreamChunk struct {
	// 内容片段
	Content string
	// 是否完成
	Done bool
	// 使用的模型（仅在完成时提供）
	Model string
	// Token 使用情况（仅在完成时提供）
	Usage *Usage
	// 错误信息
	Error error
}

// MaskAPIKey 脱敏 API 密钥
// 只保留前4位和后4位，中间用星号替换
// 如果密钥长度小于等于8位，全部用星号替换
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	length := len(apiKey)
	if length <= 8 {
		return "****"
	}

	return apiKey[:4] + "****" + apiKey[length-4:]
}

// MaskSensitiveConfig 脱敏配置中的敏感信息
// 返回一个可以安全记录到日志的配置副本
func MaskSensitiveConfig(config interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 使用 JSON 序列化/反序列化来转换
	data, err := json.Marshal(config)
	if err != nil {
		return result
	}

	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return result
	}

	// 脱敏敏感字段
	for key, value := range temp {
		switch key {
		case "apiKey", "APIKey", "api_key":
			// API 密钥脱敏
			if strValue, ok := value.(string); ok {
				result[key] = MaskAPIKey(strValue)
			} else {
				result[key] = "****"
			}
		default:
			// 其他字段保持不变
			result[key] = value
		}
	}

	return result
}
