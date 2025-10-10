package vector

import (
	"context"
	"fmt"
	"time"
)

// EmbeddingProvider 定义文本向量化的统一接口
type EmbeddingProvider interface {
	// 生成单个文本的向量
	Embed(ctx context.Context, text string) ([]float32, error)
	
	// 批量生成文本向量
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	
	// 获取向量维度
	GetDimension() int
	
	// 获取模型名称
	GetModelName() string
	
	// 健康检查
	HealthCheck(ctx context.Context) error
}

// EmbeddingRequest 向量化请求
type EmbeddingRequest struct {
	Texts     []string               `json:"texts"`
	Model     string                 `json:"model"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// EmbeddingResponse 向量化响应
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Usage      *Usage      `json:"usage,omitempty"`
}

// Usage API使用统计
type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingProviderType 向量化提供商类型
type EmbeddingProviderType string

const (
	EmbeddingProviderOpenAI    EmbeddingProviderType = "openai"
	EmbeddingProviderAzure     EmbeddingProviderType = "azure"
	EmbeddingProviderQianwen   EmbeddingProviderType = "qianwen"
	EmbeddingProviderBaichuan  EmbeddingProviderType = "baichuan"
	EmbeddingProviderZhipu     EmbeddingProviderType = "zhipu"
	EmbeddingProviderLocal     EmbeddingProviderType = "local"
)

// EmbeddingConfig 向量化配置
type EmbeddingConfig struct {
	Provider   EmbeddingProviderType  `json:"provider" yaml:"provider"`
	Model      string                 `json:"model" yaml:"model"`
	APIKey     string                 `json:"api_key" yaml:"api_key"`
	BaseURL    string                 `json:"base_url" yaml:"base_url"`
	Dimension  int                    `json:"dimension" yaml:"dimension"`
	MaxTokens  int                    `json:"max_tokens" yaml:"max_tokens"`
	BatchSize  int                    `json:"batch_size" yaml:"batch_size"`
	Timeout    time.Duration          `json:"timeout" yaml:"timeout"`
	RetryCount int                    `json:"retry_count" yaml:"retry_count"`
	Settings   map[string]interface{} `json:"settings" yaml:"settings"`
}

// Validate 验证配置
func (c *EmbeddingConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	
	if c.APIKey == "" && c.Provider != EmbeddingProviderLocal {
		return fmt.Errorf("api_key is required for provider %s", c.Provider)
	}
	
	if c.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}
	
	if c.BatchSize <= 0 {
		c.BatchSize = 10 // 默认批次大小
	}
	
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second // 默认超时时间
	}
	
	if c.RetryCount < 0 {
		c.RetryCount = 3 // 默认重试次数
	}
	
	return nil
}

// GetDefaultConfigs 获取默认配置
func GetDefaultConfigs() map[EmbeddingProviderType]*EmbeddingConfig {
	return map[EmbeddingProviderType]*EmbeddingConfig{
		EmbeddingProviderOpenAI: {
			Provider:   EmbeddingProviderOpenAI,
			Model:      "text-embedding-ada-002",
			BaseURL:    "https://api.openai.com/v1",
			Dimension:  1536,
			MaxTokens:  8192,
			BatchSize:  100,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
		EmbeddingProviderAzure: {
			Provider:   EmbeddingProviderAzure,
			Model:      "text-embedding-ada-002",
			Dimension:  1536,
			MaxTokens:  8192,
			BatchSize:  100,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
		EmbeddingProviderQianwen: {
			Provider:   EmbeddingProviderQianwen,
			Model:      "text-embedding-v1",
			BaseURL:    "https://dashscope.aliyuncs.com/api/v1",
			Dimension:  1536,
			MaxTokens:  2048,
			BatchSize:  25,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
		EmbeddingProviderBaichuan: {
			Provider:   EmbeddingProviderBaichuan,
			Model:      "Baichuan-Text-Embedding",
			BaseURL:    "https://api.baichuan-ai.com/v1",
			Dimension:  1024,
			MaxTokens:  512,
			BatchSize:  16,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
		EmbeddingProviderZhipu: {
			Provider:   EmbeddingProviderZhipu,
			Model:      "embedding-2",
			BaseURL:    "https://open.bigmodel.cn/api/paas/v4",
			Dimension:  1024,
			MaxTokens:  512,
			BatchSize:  100,
			Timeout:    30 * time.Second,
			RetryCount: 3,
		},
	}
}

// EmbeddingTask 异步向量化任务
type EmbeddingTask struct {
	ID        string    `json:"id"`
	Texts     []string  `json:"texts"`
	Status    TaskStatus `json:"status"`
	Progress  int       `json:"progress"`
	Result    [][]float32 `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending    TaskStatus = 0 // 等待中
	TaskStatusProcessing TaskStatus = 1 // 处理中
	TaskStatusCompleted  TaskStatus = 2 // 已完成
	TaskStatusFailed     TaskStatus = 3 // 失败
	TaskStatusCancelled  TaskStatus = 4 // 已取消
)

// String 返回任务状态的字符串表示
func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusProcessing:
		return "processing"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}