package model

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// TestChatOptionsValidation 测试 ChatOptions 的验证规则
func TestChatOptionsValidation(t *testing.T) {
	validate := validator.New()

	tests := []struct {
		name      string
		options   ChatOptions
		wantError bool
		errorTag  string
	}{
		{
			name: "有效的模型名称",
			options: ChatOptions{
				ModelName: stringPtr("gpt-4"),
			},
			wantError: false,
		},
		{
			name: "有效的模型名称 - 包含连字符",
			options: ChatOptions{
				ModelName: stringPtr("gpt-4-turbo"),
			},
			wantError: false,
		},
		{
			name: "有效的模型名称 - 包含下划线",
			options: ChatOptions{
				ModelName: stringPtr("qwen_turbo"),
			},
			wantError: false,
		},
		{
			name: "有效的模型名称 - 最大长度",
			options: ChatOptions{
				ModelName: stringPtr("a" + string(make([]byte, 127))), // 128 字符
			},
			wantError: false,
		},
		{
			name: "空的 ModelName 指针（nil）",
			options: ChatOptions{
				ModelName: nil,
			},
			wantError: false,
		},
		{
			name: "无效的模型名称 - 空字符串",
			options: ChatOptions{
				ModelName: stringPtr(""),
			},
			wantError: true,
			errorTag:  "min",
		},
		{
			name: "无效的模型名称 - 超过最大长度",
			options: ChatOptions{
				ModelName: stringPtr(string(make([]byte, 129))), // 129 字符
			},
			wantError: true,
			errorTag:  "max",
		},
		{
			name: "有效的温度值",
			options: ChatOptions{
				Temperature: float64Ptr(0.7),
			},
			wantError: false,
		},
		{
			name: "无效的温度值 - 小于0",
			options: ChatOptions{
				Temperature: float64Ptr(-0.1),
			},
			wantError: true,
			errorTag:  "gte",
		},
		{
			name: "无效的温度值 - 大于2",
			options: ChatOptions{
				Temperature: float64Ptr(2.1),
			},
			wantError: true,
			errorTag:  "lte",
		},
		{
			name: "有效的 MaxTokens",
			options: ChatOptions{
				MaxTokens: intPtr(2048),
			},
			wantError: false,
		},
		{
			name: "无效的 MaxTokens - 小于等于0",
			options: ChatOptions{
				MaxTokens: intPtr(0),
			},
			wantError: true,
			errorTag:  "gt",
		},
		{
			name: "组合验证 - 所有字段有效",
			options: ChatOptions{
				ModelName:   stringPtr("gpt-4"),
				Temperature: float64Ptr(0.7),
				MaxTokens:   intPtr(2048),
				TopP:        float64Ptr(0.9),
				TopK:        intPtr(40),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.options)

			if tt.wantError {
				assert.Error(t, err, "期望验证失败")
				if err != nil {
					validationErrors := err.(validator.ValidationErrors)
					assert.NotEmpty(t, validationErrors, "应该有验证错误")
					if len(validationErrors) > 0 {
						assert.Equal(t, tt.errorTag, validationErrors[0].Tag(), "错误标签应该匹配")
					}
				}
			} else {
				assert.NoError(t, err, "期望验证成功")
			}
		})
	}
}

// 辅助函数：创建字符串指针
func stringPtr(s string) *string {
	return &s
}

// 辅助函数：创建 float64 指针
func float64Ptr(f float64) *float64 {
	return &f
}

// 辅助函数：创建 int 指针
func intPtr(i int) *int {
	return &i
}
