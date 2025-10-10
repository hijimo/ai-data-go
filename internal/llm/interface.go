package llm

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LLMProvider LLM提供商抽象接口
type LLMProvider interface {
	// 文本生成
	GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	
	// 流式生成
	GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamResponse, error)
	
	// 获取模型列表
	ListModels(ctx context.Context) ([]Model, error)
	
	// 健康检查
	HealthCheck(ctx context.Context) error
	
	// 获取提供商类型
	GetProviderType() ProviderType
	
	// 获取提供商名称
	GetProviderName() string
}

// ProviderType 提供商类型
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAzure     ProviderType = "azure"
	ProviderQianwen   ProviderType = "qianwen"
	ProviderClaude    ProviderType = "claude"
	ProviderBaichuan  ProviderType = "baichuan"
	ProviderChatGLM   ProviderType = "chatglm"
)

// GenerateRequest 文本生成请求
type GenerateRequest struct {
	// 基础参数
	Model       string    `json:"model" validate:"required"`
	Messages    []Message `json:"messages" validate:"required,min=1"`
	
	// 生成参数
	Temperature      *float64 `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxTokens        *int     `json:"max_tokens,omitempty" validate:"omitempty,min=1"`
	TopP             *float64 `json:"top_p,omitempty" validate:"omitempty,min=0,max=1"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty" validate:"omitempty,min=-2,max=2"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty" validate:"omitempty,min=-2,max=2"`
	
	// 流式输出
	Stream bool `json:"stream"`
	
	// 停止词
	Stop []string `json:"stop,omitempty"`
	
	// 用户标识
	User string `json:"user,omitempty"`
	
	// 扩展参数
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// GenerateResponse 文本生成响应
type GenerateResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
}

// StreamResponse 流式响应
type StreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
	Done    bool           `json:"done"`
	Error   *StreamError   `json:"error,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role" validate:"required,oneof=system user assistant"`
	Content string `json:"content" validate:"required"`
	Name    string `json:"name,omitempty"`
}

// Choice 选择结果
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// StreamChoice 流式选择结果
type StreamChoice struct {
	Index        int           `json:"index"`
	Delta        MessageDelta  `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
}

// MessageDelta 消息增量
type MessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamError 流式错误
type StreamError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Model 模型信息
type Model struct {
	ID          string            `json:"id"`
	Object      string            `json:"object"`
	Created     int64             `json:"created"`
	OwnedBy     string            `json:"owned_by"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	ModelType   ModelType         `json:"model_type"`
	Capabilities []string         `json:"capabilities"`
	Limits      ModelLimits       `json:"limits"`
	Pricing     *ModelPricing     `json:"pricing,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// ModelType 模型类型
type ModelType string

const (
	ModelTypeChat       ModelType = "chat"
	ModelTypeCompletion ModelType = "completion"
	ModelTypeEmbedding  ModelType = "embedding"
	ModelTypeImage      ModelType = "image"
	ModelTypeAudio      ModelType = "audio"
)

// ModelLimits 模型限制
type ModelLimits struct {
	MaxTokens       int `json:"max_tokens"`
	MaxInputTokens  int `json:"max_input_tokens"`
	MaxOutputTokens int `json:"max_output_tokens"`
	ContextWindow   int `json:"context_window"`
}

// ModelPricing 模型定价
type ModelPricing struct {
	InputPrice  float64 `json:"input_price"`  // 每1K token价格
	OutputPrice float64 `json:"output_price"` // 每1K token价格
	Currency    string  `json:"currency"`     // 货币单位
}

// ProviderConfig 提供商配置接口
type ProviderConfig interface {
	GetProviderType() ProviderType
	GetAPIKey() string
	GetBaseURL() string
	Validate() error
}

// BaseProviderConfig 基础提供商配置
type BaseProviderConfig struct {
	Type    ProviderType `json:"type" validate:"required"`
	Name    string       `json:"name" validate:"required"`
	APIKey  string       `json:"api_key" validate:"required"`
	BaseURL string       `json:"base_url"`
	Timeout time.Duration `json:"timeout"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// GetProviderType 获取提供商类型
func (c *BaseProviderConfig) GetProviderType() ProviderType {
	return c.Type
}

// GetAPIKey 获取API密钥
func (c *BaseProviderConfig) GetAPIKey() string {
	return c.APIKey
}

// GetBaseURL 获取基础URL
func (c *BaseProviderConfig) GetBaseURL() string {
	return c.BaseURL
}

// Validate 验证配置
func (c *BaseProviderConfig) Validate() error {
	if c.Type == "" {
		return ErrInvalidProviderType
	}
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	return nil
}

// ProviderFactory 提供商工厂接口
type ProviderFactory interface {
	CreateProvider(config ProviderConfig) (LLMProvider, error)
	SupportedTypes() []ProviderType
}

// CallMetrics LLM调用指标
type CallMetrics struct {
	ID           uuid.UUID     `json:"id"`
	ProviderType ProviderType  `json:"provider_type"`
	Model        string        `json:"model"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	TokenUsage   Usage         `json:"token_usage"`
	Success      bool          `json:"success"`
	ErrorCode    string        `json:"error_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Cost         float64       `json:"cost,omitempty"`
}

// RateLimiter 速率限制器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Wait(ctx context.Context, key string) error
}

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Call(ctx context.Context, fn func() error) error
	State() string
}