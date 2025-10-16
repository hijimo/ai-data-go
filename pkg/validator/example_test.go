package validator_test

import (
	"fmt"
	"genkit-ai-service/pkg/validator"
)

// ChatRequest 示例请求结构
type ChatRequest struct {
	Message     string       `json:"message" validate:"required"`
	SessionID   string       `json:"sessionId,omitempty"`
	Temperature *float64     `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
	MaxTokens   *int         `json:"maxTokens,omitempty" validate:"omitempty,gt=0"`
}

// Example_validate 演示如何使用验证器验证请求参数
func Example_validate() {
	// 创建验证器实例
	v := validator.New()
	
	// 有效的请求
	temp := 0.7
	maxTokens := 1000
	validRequest := ChatRequest{
		Message:     "你好",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}
	
	err := v.Validate(validRequest)
	if err == nil {
		fmt.Println("验证通过")
	}
	
	// Output:
	// 验证通过
}

// Example_validateStruct 演示如何验证并获取格式化的错误信息
func Example_validateStruct() {
	// 使用默认验证器
	invalidTemp := 3.0 // 超出范围 (0-2)
	invalidRequest := ChatRequest{
		Message:     "", // 必填字段为空
		Temperature: &invalidTemp,
	}
	
	errors := validator.ValidateStruct(invalidRequest)
	if errors != nil {
		for _, err := range errors {
			fmt.Printf("字段: %s, 错误: %s\n", err.Field, err.Message)
		}
	}
	
	// Output:
	// 字段: message, 错误: message 是必填字段
	// 字段: temperature, 错误: temperature 必须小于或等于 2
}

// Example_formatErrors 演示如何格式化验证错误
func Example_formatErrors() {
	v := validator.New()
	
	// 创建一个无效的请求
	invalidTokens := -100
	request := ChatRequest{
		MaxTokens: &invalidTokens,
	}
	
	// 验证请求
	err := v.Validate(request)
	if err != nil {
		// 格式化错误
		errors := v.FormatErrors(err)
		fmt.Printf("发现 %d 个验证错误\n", len(errors))
		for _, e := range errors {
			fmt.Printf("- %s: %s\n", e.Field, e.Message)
		}
	}
	
	// Output:
	// 发现 2 个验证错误
	// - message: message 是必填字段
	// - maxTokens: maxTokens 必须大于 0
}
