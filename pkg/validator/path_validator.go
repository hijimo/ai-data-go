package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// 路径参数验证规则
const (
	// MaxIDLength ID的最大长度
	MaxIDLength = 100
	// MinIDLength ID的最小长度
	MinIDLength = 1
)

// 路径参数格式正则表达式（只允许字母、数字、下划线、连字符、点号）
var idFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// PathValidationError 路径验证错误
type PathValidationError struct {
	Field   string
	Message string
}

// Error 实现 error 接口
func (e *PathValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateProviderID 验证提供商ID
func ValidateProviderID(providerID string) error {
	if err := validateID(providerID, "providerId"); err != nil {
		return err
	}
	return nil
}

// ValidateModelID 验证模型ID
func ValidateModelID(modelID string) error {
	if err := validateID(modelID, "modelId"); err != nil {
		return err
	}
	return nil
}

// validateID 验证ID的通用方法
func validateID(id, fieldName string) error {
	// 检查是否为空
	if id == "" {
		return &PathValidationError{
			Field:   fieldName,
			Message: "不能为空",
		}
	}

	// 检查长度
	if len(id) < MinIDLength {
		return &PathValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("长度不能少于 %d 个字符", MinIDLength),
		}
	}

	if len(id) > MaxIDLength {
		return &PathValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("长度不能超过 %d 个字符", MaxIDLength),
		}
	}

	// 检查格式（只允许字母、数字、下划线、连字符、点号）
	if !idFormatRegex.MatchString(id) {
		return &PathValidationError{
			Field:   fieldName,
			Message: "格式无效，只允许字母、数字、下划线、连字符和点号",
		}
	}

	// 防止目录遍历攻击
	if containsPathTraversal(id) {
		return &PathValidationError{
			Field:   fieldName,
			Message: "包含非法字符或路径遍历模式",
		}
	}

	return nil
}

// containsPathTraversal 检查是否包含路径遍历模式
func containsPathTraversal(path string) bool {
	// 检查常见的路径遍历模式
	dangerousPatterns := []string{
		"..",      // 父目录引用
		"./",      // 当前目录引用
		"../",     // 父目录引用
		"/..",     // 父目录引用
		"\\",      // Windows 路径分隔符
		"\x00",    // 空字节
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// ValidatePathSafety 验证路径安全性（用于文件系统操作）
// 确保路径在指定的基础目录内
func ValidatePathSafety(basePath, targetPath string) error {
	// 清理路径
	cleanTarget := strings.TrimPrefix(targetPath, "/")

	// 检查目标路径是否包含路径遍历
	if containsPathTraversal(cleanTarget) {
		return &PathValidationError{
			Field:   "path",
			Message: "路径包含非法的遍历模式",
		}
	}

	// 检查是否尝试访问基础目录之外的路径
	// 注意：这里只做简单检查，实际使用时应该配合 filepath.Clean 和 filepath.Abs
	if strings.HasPrefix(cleanTarget, "../") || strings.Contains(cleanTarget, "/../") {
		return &PathValidationError{
			Field:   "path",
			Message: "不允许访问基础目录之外的路径",
		}
	}

	return nil
}
