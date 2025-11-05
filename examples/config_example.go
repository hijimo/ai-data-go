package main

import (
	"fmt"
	"os"

	"genkit-ai-service/internal/config"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("配置管理示例")
	fmt.Println("========================================\n")

	// 示例1: 使用配置加载器自动加载配置
	fmt.Println("示例1: 自动加载配置（根据APP_ENV环境变量）")
	fmt.Println("----------------------------------------")
	
	loader := config.NewConfigLoader()
	fmt.Printf("当前环境: %s\n", loader.GetEnv())
	fmt.Printf("是否为开发环境: %v\n", loader.IsDevelopment())
	fmt.Printf("是否为生产环境: %v\n", loader.IsProduction())
	fmt.Printf("是否为测试环境: %v\n\n", loader.IsTest())

	// 示例2: 从YAML文件加载配置
	fmt.Println("示例2: 从YAML文件加载配置")
	fmt.Println("----------------------------------------")
	
	// 检查配置文件是否存在
	configPath := "config/dev.yaml"
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := config.LoadFromYAML(configPath)
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
		} else {
			fmt.Printf("✓ 成功加载配置文件: %s\n", configPath)
			fmt.Printf("  服务器: %s:%s\n", cfg.Server.Host, cfg.Server.Port)
			fmt.Printf("  数据库: %s:%s/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
			fmt.Printf("  Redis: %s:%s\n", cfg.Redis.Host, cfg.Redis.Port)
			fmt.Printf("  Genkit模型: %s\n", cfg.Genkit.Model)
			fmt.Printf("  日志级别: %s\n", cfg.Log.Level)
		}
	} else {
		fmt.Printf("配置文件不存在: %s\n", configPath)
	}
	fmt.Println()

	// 示例3: 从环境变量加载配置
	fmt.Println("示例3: 从环境变量加载配置")
	fmt.Println("----------------------------------------")
	
	// 设置必需的环境变量
	os.Setenv("GENKIT_API_KEY", "test-api-key")
	os.Setenv("JWT_SECRET", "test-jwt-secret-key-min-32-characters")
	
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
	} else {
		fmt.Println("✓ 成功从环境变量加载配置")
		fmt.Printf("  服务器端口: %s\n", cfg.Server.Port)
		fmt.Printf("  Genkit API密钥: %s\n", maskString(cfg.Genkit.APIKey))
		fmt.Printf("  JWT密钥: %s\n", maskString(cfg.Auth.JWTSecret))
	}
	fmt.Println()

	// 示例4: 环境变量替换
	fmt.Println("示例4: 环境变量替换示例")
	fmt.Println("----------------------------------------")
	
	os.Setenv("CUSTOM_PORT", "9090")
	os.Setenv("CUSTOM_HOST", "0.0.0.0")
	
	yamlContent := `
server:
  port: "${CUSTOM_PORT:8080}"
  host: "${CUSTOM_HOST:localhost}"
  mode: debug
`
	
	fmt.Println("YAML内容:")
	fmt.Println(yamlContent)
	fmt.Println("环境变量:")
	fmt.Printf("  CUSTOM_PORT=%s\n", os.Getenv("CUSTOM_PORT"))
	fmt.Printf("  CUSTOM_HOST=%s\n", os.Getenv("CUSTOM_HOST"))
	fmt.Println()

	// 示例5: 配置验证
	fmt.Println("示例5: 配置验证")
	fmt.Println("----------------------------------------")
	
	// 创建一个无效的配置
	invalidConfig := &config.Config{
		Server: config.ServerConfig{
			Port: "99999", // 无效端口
			Host: "localhost",
		},
	}
	
	if err := invalidConfig.Validate(); err != nil {
		fmt.Printf("✓ 配置验证正确捕获错误: %v\n", err)
	}
	fmt.Println()

	// 示例6: 不同环境的配置
	fmt.Println("示例6: 不同环境的配置文件")
	fmt.Println("----------------------------------------")
	
	environments := []string{"dev", "prod", "test"}
	for _, env := range environments {
		configPath := fmt.Sprintf("config/%s.yaml", env)
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("✓ %s 环境配置文件存在: %s\n", env, configPath)
		} else {
			fmt.Printf("✗ %s 环境配置文件不存在: %s\n", env, configPath)
		}
	}
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("示例完成")
	fmt.Println("========================================")
}

// maskString 遮蔽字符串中间部分，用于显示敏感信息
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
