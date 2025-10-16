package logger_test

import (
	"context"
	"genkit-ai-service/internal/logger"
)

// ExampleLogger_basic 演示基本日志记录
func ExampleLogger_basic() {
	// 初始化日志记录器
	logger.Init("info", "json")

	// 记录不同级别的日志
	logger.Info("应用启动")
	logger.Warn("这是一个警告")
	logger.Error("发生错误")
}

// ExampleLogger_withFields 演示带字段的日志记录
func ExampleLogger_withFields() {
	logger.Init("info", "json")

	// 记录带有额外字段的日志
	logger.Info("用户登录", logger.Fields{
		"userId":   "user-123",
		"username": "john_doe",
		"ip":       "192.168.1.1",
	})
}

// ExampleLogger_withContext 演示使用上下文的日志记录
func ExampleLogger_withContext() {
	logger.Init("info", "json")

	// 创建带有会话ID的上下文
	ctx := context.WithValue(context.Background(), logger.SessionIDKey, "session-abc123")
	ctx = context.WithValue(ctx, logger.RequestIDKey, "request-xyz789")

	// 使用上下文记录日志，会自动包含 sessionId 和 requestId
	logger.InfoContext(ctx, "处理AI对话请求")
}

// ExampleLogger_withFieldsChaining 演示字段链式调用
func ExampleLogger_withFieldsChaining() {
	logger.Init("info", "json")

	// 创建带有预设字段的日志记录器
	serviceLogger := logger.WithFields(logger.Fields{
		"service": "ai-service",
		"version": "1.0.0",
	})

	// 使用预设字段的日志记录器
	serviceLogger.Info("服务初始化完成")
	serviceLogger.Info("开始处理请求", logger.Fields{
		"requestId": "req-123",
	})
}

// ExampleLogger_textFormat 演示文本格式日志
func ExampleLogger_textFormat() {
	// 使用文本格式
	logger.Init("info", "text")

	logger.Info("这是文本格式的日志", logger.Fields{
		"key": "value",
	})
}

// ExampleLogger_debugLevel 演示调试级别日志
func ExampleLogger_debugLevel() {
	// 设置为调试级别
	logger.Init("debug", "json")

	logger.Debug("调试信息", logger.Fields{
		"variable": "value",
		"state":    "processing",
	})
}

// ExampleLogger_aiService 演示AI服务中的日志使用
func ExampleLogger_aiService() {
	logger.Init("info", "json")

	// 创建带有会话信息的上下文
	ctx := context.WithValue(context.Background(), logger.SessionIDKey, "session-123")

	// 记录AI对话开始
	logger.InfoContext(ctx, "AI对话开始", logger.Fields{
		"model":       "gemini-2.5-flash",
		"temperature": 0.7,
		"maxTokens":   2000,
	})

	// 记录AI响应
	logger.InfoContext(ctx, "AI对话完成", logger.Fields{
		"model":            "gemini-2.5-flash",
		"promptTokens":     10,
		"completionTokens": 50,
		"totalTokens":      60,
		"duration":         "1.5s",
	})
}
