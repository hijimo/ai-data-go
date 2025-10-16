package config_test

import (
	"fmt"
	"os"

	"genkit-ai-service/internal/config"
)

// ExampleLoad 演示如何加载配置
func ExampleLoad() {
	// 设置必需的环境变量
	os.Setenv("GENKIT_API_KEY", "your-api-key")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_NAME", "genkit_ai_service")
	
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}
	
	// 使用配置
	fmt.Printf("服务器将在 %s:%s 上运行\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("使用的 AI 模型: %s\n", cfg.Genkit.Model)
	fmt.Printf("数据库连接: %s:%s/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	
	// Output:
	// 服务器将在 0.0.0.0:8080 上运行
	// 使用的 AI 模型: gemini-2.5-flash
	// 数据库连接: localhost:5432/genkit_ai_service
}
