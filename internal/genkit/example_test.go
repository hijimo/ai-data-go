package genkit_test

import (
	"context"
	"fmt"
	"log"

	"genkit-ai-service/internal/genkit"
)

// ExampleClient_Initialize 演示如何初始化 Genkit 客户端
func ExampleClient_Initialize() {
	// 创建客户端
	client := genkit.NewClient()

	// 配置客户端
	config := &genkit.Config{
		APIKey:             "your-api-key",
		Model:              "gemini-2.5-flash",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2000,
	}

	// 初始化客户端
	err := client.Initialize(context.Background(), config)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	fmt.Println("客户端初始化成功")
	// Output: 客户端初始化成功
}

// ExampleClient_Generate 演示如何使用 Genkit 客户端生成内容
func ExampleClient_Generate() {
	// 注意：这是一个示例，实际使用时需要配置真实的模型配置仓储
	// 创建客户端
	client := genkit.NewClient()

	// 配置客户端
	config := &genkit.Config{
		APIKey:             "your-api-key",
		Model:              "gemini-2.5-flash",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2000,
	}

	// 初始化客户端
	err := client.Initialize(context.Background(), config)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 设置生成选项
	temp := 0.9
	maxTokens := 1000
	options := &genkit.GenerateOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	// 生成内容（需要提供租户ID和模型名称）
	// 这里会失败，因为没有配置模型配置仓储
	_, err = client.Generate(context.Background(), "tenant-123", "gemini-pro", "你好，请介绍一下 Firebase", options)
	if err != nil {
		fmt.Println("需要配置模型配置仓储")
	}

	// Output: 需要配置模型配置仓储
}
