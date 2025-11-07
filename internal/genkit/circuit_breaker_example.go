package genkit

import (
	"context"
	"time"

	"genkit-ai-service/internal/middleware"
)

// 示例：如何在应用中使用带熔断器的 Genkit 客户端

// CreateGenkitClientWithCircuitBreaker 创建带熔断器的 Genkit 客户端
// 这是推荐的创建方式，适用于生产环境
func CreateGenkitClientWithCircuitBreaker(config *Config) (*ClientWithCircuitBreaker, error) {
	// 1. 创建基础客户端
	baseClient := NewClient()

	// 2. 初始化客户端
	ctx := context.Background()
	if err := baseClient.Initialize(ctx, config); err != nil {
		return nil, err
	}

	// 3. 初始化模型
	if err := baseClient.InitializeModel(ctx); err != nil {
		return nil, err
	}

	// 4. 配置熔断器
	breakerConfig := &middleware.CircuitBreakerConfig{
		MaxFailures:         5,                // 5次失败后打开熔断器
		Timeout:             30 * time.Second, // 30秒后进入半开状态
		HalfOpenMaxRequests: 3,                // 半开状态允许3个请求
		SuccessThreshold:    2,                // 连续2次成功后关闭熔断器
		OnStateChange: func(from, to middleware.CircuitState) {
			// 状态变化时的处理逻辑
			// 可以在这里发送告警、记录日志等
			switch to {
			case middleware.StateOpen:
				// 熔断器打开，可以发送告警
				// alertService.SendAlert("AI服务熔断器已打开")
			case middleware.StateClosed:
				// 熔断器关闭，服务恢复正常
				// alertService.SendAlert("AI服务熔断器已关闭，服务恢复")
			case middleware.StateHalfOpen:
				// 熔断器进入半开状态，正在测试服务
				// alertService.SendAlert("AI服务熔断器进入半开状态")
			}
		},
	}

	// 5. 创建带熔断器的客户端
	clientWithBreaker := NewClientWithCircuitBreaker(baseClient, breakerConfig)

	return clientWithBreaker, nil
}

// 使用示例：

/*
// 在 main.go 或服务初始化代码中：

func initGenkitClient() (*genkit.ClientWithCircuitBreaker, error) {
	config := &genkit.Config{
		APIKey: os.Getenv("GENKIT_API_KEY"),
		Model:  os.Getenv("GENKIT_MODEL"),
	}

	client, err := genkit.CreateGenkitClientWithCircuitBreaker(config)
	if err != nil {
		return nil, fmt.Errorf("初始化 Genkit 客户端失败: %w", err)
	}

	return client, nil
}

// 在服务层使用：

func (s *ChatService) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// 调用带熔断保护的生成方法
	result, err := s.genkitClient.Generate(ctx, prompt, nil)
	if err != nil {
		// 如果是熔断器打开导致的错误，可以执行降级逻辑
		if errors.Is(err, middleware.ErrCircuitBreakerOpen) {
			// 执行降级策略
			return s.degradationService.DegradeAIService(ctx, sessionID, prompt)
		}
		return "", err
	}

	return result.Text, nil
}

// 监控熔断器状态：

func (s *MonitoringService) GetCircuitBreakerStatus() middleware.CircuitBreakerStats {
	return s.genkitClient.GetCircuitBreakerStats()
}

// 手动重置熔断器（管理员操作）：

func (s *AdminService) ResetCircuitBreaker(ctx context.Context) error {
	// 验证管理员权限
	if !s.isAdmin(ctx) {
		return errors.New("权限不足")
	}

	s.genkitClient.ResetCircuitBreaker()
	logger.InfoContext(ctx, "熔断器已手动重置")
	return nil
}
*/
