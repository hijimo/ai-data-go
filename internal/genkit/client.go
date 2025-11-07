package genkit

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// Client Genkit 客户端接口
type Client interface {
	// Initialize 初始化客户端
	Initialize(ctx context.Context, config *Config) error

	// InitializeModel 初始化并设置模型
	InitializeModel(ctx context.Context) error

	// Generate 生成内容
	Generate(ctx context.Context, prompt string, options *GenerateOptions) (*GenerateResult, error)

	// GenerateStream 流式生成内容
	GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan StreamChunk, error)

	// GetGenkit 获取底层的Genkit实例（用于注册Flows）
	GetGenkit() *genkit.Genkit

	// Close 关闭客户端
	Close() error
}

// client Genkit 客户端实现
type client struct {
	config *Config
	g      *genkit.Genkit
}

// NewClient 创建新的 Genkit 客户端
func NewClient() Client {
	return &client{}
}

// Initialize 初始化客户端
func (c *client) Initialize(ctx context.Context, config *Config) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API 密钥不能为空")
	}

	if config.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}

	c.config = config

	return nil
}

// InitializeModel 初始化并设置模型
func (c *client) InitializeModel(ctx context.Context) error {
	if c.config == nil {
		return fmt.Errorf("客户端未初始化，请先调用 Initialize")
	}

	// 初始化 Genkit，配置 Google AI 插件和默认模型
	c.g = genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: c.config.APIKey,
		}),
		genkit.WithDefaultModel("googleai/"+c.config.Model),
	)

	return nil
}

// Generate 生成内容
func (c *client) Generate(ctx context.Context, prompt string, options *GenerateOptions) (*GenerateResult, error) {
	if c.config == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	if c.g == nil {
		return nil, fmt.Errorf("模型未初始化，请先通过 InitializeModel 设置模型")
	}

	if prompt == "" {
		return nil, fmt.Errorf("提示词不能为空")
	}

	// 调用 Genkit 生成
	// 注意：当前简化实现，暂不支持自定义 temperature、maxTokens 等参数
	// 这些参数可以通过 genkit.WithDefaultModel 在初始化时设置
	resp, err := genkit.Generate(ctx, c.g, ai.WithPrompt(prompt))
	if err != nil {
		return nil, fmt.Errorf("生成内容失败: %w", err)
	}

	// 构建结果
	result := &GenerateResult{
		Text:  resp.Text(),
		Model: c.config.Model,
	}

	// 提取 token 使用情况
	if resp.Usage != nil {
		result.Usage = &Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}

	return result, nil
}

// GenerateStream 流式生成内容
func (c *client) GenerateStream(ctx context.Context, prompt string, options *GenerateOptions) (<-chan StreamChunk, error) {
	if c.config == nil {
		return nil, fmt.Errorf("客户端未初始化")
	}

	if c.g == nil {
		return nil, fmt.Errorf("模型未初始化，请先通过 InitializeModel 设置模型")
	}

	if prompt == "" {
		return nil, fmt.Errorf("提示词不能为空")
	}

	// 创建流式响应通道
	streamChan := make(chan StreamChunk, 10)

	// 在 goroutine 中处理流式响应
	go func() {
		defer close(streamChan)

		// 调用 Genkit 流式生成
		resp, err := genkit.Generate(ctx, c.g,
			ai.WithPrompt(prompt),
			ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
				// 发送流式数据块
				select {
				case streamChan <- StreamChunk{
					Content: chunk.Text(),
					Done:    false,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}),
		)

		if err != nil {
			// 发送错误
			streamChan <- StreamChunk{
				Content: "",
				Done:    true,
				Error:   err,
			}
			return
		}

		// 发送完成标记，包含最终的使用统计
		var usage *Usage
		if resp.Usage != nil {
			usage = &Usage{
				PromptTokens:     int(resp.Usage.InputTokens),
				CompletionTokens: int(resp.Usage.OutputTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			}
		}

		streamChan <- StreamChunk{
			Content: "",
			Done:    true,
			Model:   c.config.Model,
			Usage:   usage,
		}
	}()

	return streamChan, nil
}

// GetGenkit 获取底层的Genkit实例
func (c *client) GetGenkit() *genkit.Genkit {
	return c.g
}

// Close 关闭客户端
func (c *client) Close() error {
	// Genkit 客户端通常不需要显式关闭
	// 这里预留接口以便未来扩展
	return nil
}
