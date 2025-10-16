package genkit

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
