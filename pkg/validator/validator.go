package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator 参数验证器
type Validator struct {
	validate *validator.Validate
}

// ValidationError 验证错误详情
type ValidationError struct {
	// 字段名
	Field string `json:"field"`
	// 错误消息
	Message string `json:"message"`
}

// New 创建新的验证器实例
func New() *Validator {
	v := validator.New()
	
	// 注册自定义验证规则
	registerCustomValidations(v)
	
	return &Validator{
		validate: v,
	}
}

// Validate 验证结构体
func (v *Validator) Validate(data interface{}) error {
	return v.validate.Struct(data)
}

// FormatErrors 格式化验证错误
func (v *Validator) FormatErrors(err error) []ValidationError {
	var errors []ValidationError
	
	if err == nil {
		return errors
	}
	
	// 类型断言为 validator.ValidationErrors
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// 如果不是验证错误，返回通用错误
		errors = append(errors, ValidationError{
			Field:   "unknown",
			Message: err.Error(),
		})
		return errors
	}
	
	// 遍历所有验证错误
	for _, fieldError := range validationErrors {
		errors = append(errors, ValidationError{
			Field:   getFieldName(fieldError),
			Message: getErrorMessage(fieldError),
		})
	}
	
	return errors
}

// ValidateStruct 验证结构体并返回格式化的错误
func (v *Validator) ValidateStruct(data interface{}) []ValidationError {
	err := v.Validate(data)
	if err == nil {
		return nil
	}
	return v.FormatErrors(err)
}

// registerCustomValidations 注册自定义验证规则
func registerCustomValidations(v *validator.Validate) {
	// 可以在这里注册自定义验证规则
	// 例如：v.RegisterValidation("custom_rule", customValidationFunc)
}

// getFieldName 获取字段名（使用 JSON 标签名）
func getFieldName(fe validator.FieldError) string {
	// 获取字段的 JSON 标签名
	field := fe.Field()
	
	// 将驼峰命名转换为小驼峰（符合 JSON 命名规范）
	// 例如: SessionID -> sessionId, MaxTokens -> maxTokens
	if len(field) > 0 {
		runes := []rune(field)
		result := make([]rune, 0, len(runes))
		
		// 找到第一个小写字母或数字的位置
		firstLowerIndex := -1
		for i, r := range runes {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				firstLowerIndex = i
				break
			}
		}
		
		if firstLowerIndex == -1 {
			// 全是大写字母，全部转小写
			return strings.ToLower(field)
		}
		
		// 将开头的大写字母转为小写，但保留最后一个大写字母
		// 例如: SessionID -> sessionId (Session 的 S 转小写，ID 的 I 保留大写，D 转小写)
		for i, r := range runes {
			if i == 0 {
				// 首字母转小写
				result = append(result, []rune(strings.ToLower(string(r)))...)
			} else if i < firstLowerIndex-1 {
				// 在第一个小写字母之前的大写字母（除了紧邻的那个）都转小写
				result = append(result, []rune(strings.ToLower(string(r)))...)
			} else {
				// 保留其他字符
				result = append(result, r)
			}
		}
		
		field = string(result)
	}
	
	return field
}

// getErrorMessage 根据验证标签生成错误消息
func getErrorMessage(fe validator.FieldError) string {
	field := getFieldName(fe)
	
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s 是必填字段", field)
	case "gte":
		return fmt.Sprintf("%s 必须大于或等于 %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s 必须小于或等于 %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s 必须大于 %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s 必须小于 %s", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s 长度必须至少为 %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能超过 %s", field, fe.Param())
	case "email":
		return fmt.Sprintf("%s 必须是有效的邮箱地址", field)
	case "url":
		return fmt.Sprintf("%s 必须是有效的 URL", field)
	case "oneof":
		return fmt.Sprintf("%s 必须是以下值之一: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s 验证失败: %s", field, fe.Tag())
	}
}

// DefaultValidator 默认验证器实例
var DefaultValidator = New()

// Validate 使用默认验证器验证结构体
func Validate(data interface{}) error {
	return DefaultValidator.Validate(data)
}

// ValidateStruct 使用默认验证器验证结构体并返回格式化的错误
func ValidateStruct(data interface{}) []ValidationError {
	return DefaultValidator.ValidateStruct(data)
}

// FormatErrors 使用默认验证器格式化验证错误
func FormatErrors(err error) []ValidationError {
	return DefaultValidator.FormatErrors(err)
}
