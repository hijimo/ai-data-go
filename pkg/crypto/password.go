package crypto

import (
	"errors"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost bcrypt 算法的 cost factor，值越大越安全但计算越慢
	BcryptCost = 12

	// MinPasswordLength 密码最小长度
	MinPasswordLength = 8

	// MaxPasswordLength 密码最大长度
	MaxPasswordLength = 128
)

var (
	// ErrPasswordTooShort 密码太短
	ErrPasswordTooShort = errors.New("密码长度不能少于8个字符")

	// ErrPasswordTooLong 密码太长
	ErrPasswordTooLong = errors.New("密码长度不能超过128个字符")

	// ErrPasswordTooWeak 密码强度不足
	ErrPasswordTooWeak = errors.New("密码强度不足，必须包含大写字母、小写字母、数字和特殊字符")

	// ErrPasswordMismatch 密码不匹配
	ErrPasswordMismatch = errors.New("密码不匹配")
)

// PasswordStrength 密码强度等级
type PasswordStrength int

const (
	// PasswordStrengthWeak 弱密码
	PasswordStrengthWeak PasswordStrength = iota
	// PasswordStrengthMedium 中等强度密码
	PasswordStrengthMedium
	// PasswordStrengthStrong 强密码
	PasswordStrengthStrong
)

// HashPassword 使用 bcrypt 算法对密码进行哈希处理
// 参数:
//   - password: 明文密码
//
// 返回:
//   - string: 密码哈希值
//   - error: 错误信息
func HashPassword(password string) (string, error) {
	// 验证密码长度
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// 使用 bcrypt 生成密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyPassword 验证密码是否与哈希值匹配
// 参数:
//   - hashedPassword: 存储的密码哈希值
//   - password: 待验证的明文密码
//
// 返回:
//   - error: 如果密码匹配返回 nil，否则返回错误
func VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return err
	}
	return nil
}

// ValidatePasswordStrength 验证密码强度
// 参数:
//   - password: 待验证的密码
//
// 返回:
//   - error: 如果密码强度不足返回错误，否则返回 nil
func ValidatePasswordStrength(password string) error {
	// 检查长度
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	// 检查是否包含必需的字符类型
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 必须包含至少三种类型的字符
	typesCount := 0
	if hasUpper {
		typesCount++
	}
	if hasLower {
		typesCount++
	}
	if hasNumber {
		typesCount++
	}
	if hasSpecial {
		typesCount++
	}

	if typesCount < 3 {
		return ErrPasswordTooWeak
	}

	return nil
}

// GetPasswordStrength 获取密码强度等级
// 参数:
//   - password: 待评估的密码
//
// 返回:
//   - PasswordStrength: 密码强度等级
func GetPasswordStrength(password string) PasswordStrength {
	if len(password) < MinPasswordLength {
		return PasswordStrengthWeak
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 计算包含的字符类型数量
	typesCount := 0
	if hasUpper {
		typesCount++
	}
	if hasLower {
		typesCount++
	}
	if hasNumber {
		typesCount++
	}
	if hasSpecial {
		typesCount++
	}

	// 根据长度和字符类型判断强度
	if len(password) >= 12 && typesCount >= 4 {
		return PasswordStrengthStrong
	}
	if len(password) >= 10 && typesCount >= 3 {
		return PasswordStrengthStrong
	}
	if len(password) >= 8 && typesCount >= 3 {
		return PasswordStrengthMedium
	}

	return PasswordStrengthWeak
}

// IsCommonPassword 检查是否为常见弱密码
// 参数:
//   - password: 待检查的密码
//
// 返回:
//   - bool: 如果是常见弱密码返回 true
func IsCommonPassword(password string) bool {
	// 常见弱密码列表
	commonPasswords := []string{
		"password", "12345678", "123456789", "qwerty", "abc123",
		"password123", "admin", "letmein", "welcome", "monkey",
		"1234567890", "password1", "qwerty123", "admin123",
	}

	// 转换为小写进行比较
	lowerPassword := regexp.MustCompile(`\s+`).ReplaceAllString(password, "")
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}

	return false
}

// ValidatePassword 综合验证密码（包含强度验证和常见密码检查）
// 参数:
//   - password: 待验证的密码
//
// 返回:
//   - error: 如果密码不符合要求返回错误
func ValidatePassword(password string) error {
	// 验证强度
	if err := ValidatePasswordStrength(password); err != nil {
		return err
	}

	// 检查是否为常见弱密码
	if IsCommonPassword(password) {
		return errors.New("不能使用常见的弱密码")
	}

	return nil
}
