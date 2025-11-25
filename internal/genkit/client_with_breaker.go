package genkit

import (
	"context"
	"time"

	"github.com/firebase/genkit/go/genkit"

	"genkit-ai-service/internal/logger"
	"genkit-ai-service/internal/middleware"
	"genkit-ai-service/internal/model"
)

// ClientWithCircuitBreaker 带熔断器的 Genkit 客户端
type ClientWithCircuitBreaker struct {
	client         Client
	circuitBreaker *middleware.CircuitBreaker
}

// NewClientWithCircuitBreaker 创建带熔断器的 Genkit 客户端
// 参数:
//   - client: 底层的 Genkit 客户端
//   - config: 熔断器配置，如果为 nil 则使用默认配置
func NewClientWithCircuitBreaker(client Client, config *middleware.CircuitBreakerConfig) *ClientWithCircuitBreaker {
	if config == nil {
		config = &middleware.CircuitBreakerConfig{
			MaxFailures:         5,                // 5次失败后打开熔断器
			Timeout:             30 * time.Second, // 30秒后进入半开状态
			HalfOpenMaxRequests: 3,                // 半开状态允许3个请求
			SuccessThreshold:    2,                // 连续2次成功后关闭熔断器
			OnStateChange: func(from, to middleware.CircuitState) {
				logger.Info("AI服务熔断器状态变化", logger.Fields{
					"from": from.String(),
					"to":   to.String(),
				})
			},
		}
	}

	return &ClientWithCircuitBreaker{
		client:         client,
		circuitBreaker: middleware.NewCircuitBreaker("genkit-ai-service", config),
	}
}

// Initialize 初始化客户端
func (c *ClientWithCircuitBreaker) Initialize(ctx context.Context, config *Config) error {
	return c.client.Initialize(ctx, config)
}

// InitializeModel 初始化并设置模型
func (c *ClientWithCircuitBreaker) InitializeModel(ctx context.Context) error {
	return c.client.InitializeModel(ctx)
}

// Generate 生成内容（带熔断保护）
func (c *ClientWithCircuitBreaker) Generate(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (*GenerateResult, error) {
	// 使用熔断器执行
	result, err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return c.client.Generate(ctx, tenantID, modelName, prompt, options)
	})

	if err != nil {
		// 检查是否是熔断器打开导致的错误
		if err == middleware.ErrCircuitBreakerOpen {
			logger.WarnContext(ctx, "AI服务熔断器已打开，请求被拒绝", logger.Fields{
				"tenant_id":     tenantID,
				"model_name":    modelName,
				"prompt_length": len(prompt),
			})
			return nil, model.NewAIServiceError(err)
		}

		// 其他错误
		logger.ErrorContext(ctx, "AI服务生成内容失败", logger.Fields{
			"error":         err.Error(),
			"tenant_id":     tenantID,
			"model_name":    modelName,
			"prompt_length": len(prompt),
		})
		return nil, err
	}

	// 类型断言
	if result == nil {
		return nil, model.NewAIServiceError(nil)
	}

	generateResult, ok := result.(*GenerateResult)
	if !ok {
		return nil, model.NewAIServiceError(nil)
	}

	return generateResult, nil
}

// GenerateStream 流式生成内容（带熔断保护）
func (c *ClientWithCircuitBreaker) GenerateStream(ctx context.Context, tenantID, modelName, prompt string, options *GenerateOptions) (<-chan StreamChunk, error) {
	// 检查熔断器状态
	state := c.circuitBreaker.GetState()
	if state == middleware.StateOpen {
		logger.WarnContext(ctx, "AI服务熔断器已打开，流式请求被拒绝", logger.Fields{
			"tenant_id":     tenantID,
			"model_name":    modelName,
			"prompt_length": len(prompt),
		})
		
		// 创建错误通道
		errChan := make(chan StreamChunk, 1)
		errChan <- StreamChunk{
			Content: "",
			Done:    true,
			Error:   middleware.ErrCircuitBreakerOpen,
		}
		close(errChan)
		return errChan, model.NewAIServiceError(middleware.ErrCircuitBreakerOpen)
	}

	// 调用底层客户端
	streamChan, err := c.client.GenerateStream(ctx, tenantID, modelName, prompt, options)
	if err != nil {
		// 记录失败
		c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, err
		})

		logger.ErrorContext(ctx, "AI服务流式生成失败", logger.Fields{
			"error":         err.Error(),
			"tenant_id":     tenantID,
			"model_name":    modelName,
			"prompt_length": len(prompt),
		})
		return nil, err
	}

	// 包装流式通道以监控结果
	wrappedChan := make(chan StreamChunk, 10)
	go func() {
		defer close(wrappedChan)

		var streamErr error
		for chunk := range streamChan {
			wrappedChan <- chunk
			if chunk.Error != nil {
				streamErr = chunk.Error
			}
		}

		// 记录流式请求的结果
		c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return nil, streamErr
		})
	}()

	return wrappedChan, nil
}

// GetGenkit 获取底层的Genkit实例
func (c *ClientWithCircuitBreaker) GetGenkit() *genkit.Genkit {
	return c.client.GetGenkit()
}

// Close 关闭客户端
func (c *ClientWithCircuitBreaker) Close() error {
	return c.client.Close()
}

// GetCircuitBreakerStats 获取熔断器统计信息
func (c *ClientWithCircuitBreaker) GetCircuitBreakerStats() middleware.CircuitBreakerStats {
	return c.circuitBreaker.GetStats()
}

// ResetCircuitBreaker 重置熔断器
func (c *ClientWithCircuitBreaker) ResetCircuitBreaker() {
	c.circuitBreaker.Reset()
}
