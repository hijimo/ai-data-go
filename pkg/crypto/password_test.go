package crypto

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "有效密码",
			password: "ValidPass123!",
			wantErr:  false,
		},
		{
			name:        "密码太短",
			password:    "short",
			wantErr:     true,
			expectedErr: ErrPasswordTooShort,
		},
		{
			name:        "密码太长",
			password:    strings.Repeat("a", 129),
			wantErr:     true,
			expectedErr: ErrPasswordTooLong,
		},
		{
			name:     "最小长度密码",
			password: "12345678",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("HashPassword() 期望错误但没有返回错误")
					return
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("HashPassword() 错误 = %v, 期望错误 %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("HashPassword() 意外错误 = %v", err)
				return
			}

			// 验证哈希值不为空
			if hash == "" {
				t.Error("HashPassword() 返回空哈希值")
			}

			// 验证哈希值可以被 bcrypt 识别
			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
			if err != nil {
				t.Errorf("生成的哈希值无法验证原始密码: %v", err)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() 失败: %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
		expectedErr    error
	}{
		{
			name:           "正确密码",
			hashedPassword: hash,
			password:       password,
			wantErr:        false,
		},
		{
			name:           "错误密码",
			hashedPassword: hash,
			password:       "WrongPassword123!",
			wantErr:        true,
			expectedErr:    ErrPasswordMismatch,
		},
		{
			name:           "空密码",
			hashedPassword: hash,
			password:       "",
			wantErr:        true,
			expectedErr:    ErrPasswordMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.hashedPassword, tt.password)
			if tt.wantErr {
				if err == nil {
					t.Error("VerifyPassword() 期望错误但没有返回错误")
					return
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("VerifyPassword() 错误 = %v, 期望错误 %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyPassword() 意外错误 = %v", err)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "强密码 - 包含所有类型",
			password: "StrongPass123!",
			wantErr:  false,
		},
		{
			name:     "强密码 - 大小写数字",
			password: "Password123",
			wantErr:  false,
		},
		{
			name:     "强密码 - 大小写特殊字符",
			password: "Password!@#",
			wantErr:  false,
		},
		{
			name:        "弱密码 - 只有小写",
			password:    "password",
			wantErr:     true,
			expectedErr: ErrPasswordTooWeak,
		},
		{
			name:        "弱密码 - 只有数字",
			password:    "12345678",
			wantErr:     true,
			expectedErr: ErrPasswordTooWeak,
		},
		{
			name:        "弱密码 - 太短",
			password:    "Pass1!",
			wantErr:     true,
			expectedErr: ErrPasswordTooShort,
		},
		{
			name:        "弱密码 - 太长",
			password:    strings.Repeat("a", 129),
			wantErr:     true,
			expectedErr: ErrPasswordTooLong,
		},
		{
			name:     "中等密码 - 小写数字特殊字符",
			password: "password123!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidatePasswordStrength() 期望错误但没有返回错误")
					return
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("ValidatePasswordStrength() 错误 = %v, 期望错误 %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePasswordStrength() 意外错误 = %v", err)
			}
		})
	}
}

func TestGetPasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     PasswordStrength
	}{
		{
			name:     "强密码 - 长且复杂",
			password: "VeryStrongPass123!@#",
			want:     PasswordStrengthStrong,
		},
		{
			name:     "强密码 - 12字符4类型",
			password: "Password123!",
			want:     PasswordStrengthStrong,
		},
		{
			name:     "强密码 - 10字符3类型",
			password: "Password12",
			want:     PasswordStrengthStrong,
		},
		{
			name:     "中等密码 - 8字符3类型",
			password: "Pass123!",
			want:     PasswordStrengthMedium,
		},
		{
			name:     "弱密码 - 只有小写",
			password: "password",
			want:     PasswordStrengthWeak,
		},
		{
			name:     "弱密码 - 太短",
			password: "Pass1!",
			want:     PasswordStrengthWeak,
		},
		{
			name:     "弱密码 - 只有两种类型",
			password: "password123",
			want:     PasswordStrengthWeak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPasswordStrength(tt.password)
			if got != tt.want {
				t.Errorf("GetPasswordStrength() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "常见密码 - password",
			password: "password",
			want:     true,
		},
		{
			name:     "常见密码 - 12345678",
			password: "12345678",
			want:     true,
		},
		{
			name:     "常见密码 - admin",
			password: "admin",
			want:     true,
		},
		{
			name:     "非常见密码",
			password: "MyUniquePass123!",
			want:     false,
		},
		{
			name:     "非常见密码 - 复杂密码",
			password: "Str0ng!P@ssw0rd",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCommonPassword(tt.password)
			if got != tt.want {
				t.Errorf("IsCommonPassword() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr bool
	}{
		{
			name:     "有效密码",
			password: "StrongPass123!",
			wantErr:  false,
		},
		{
			name:     "有效密码 - 无特殊字符但有3种类型",
			password: "Password123",
			wantErr:  false,
		},
		{
			name:     "无效 - 常见密码",
			password: "password",
			wantErr:  true,
		},
		{
			name:     "无效 - 强度不足",
			password: "weakpass",
			wantErr:  true,
		},
		{
			name:     "无效 - 太短",
			password: "Pass1!",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() 错误 = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantMinLen int
	}{
		{
			name:       "默认长度16",
			length:     16,
			wantMinLen: 16,
		},
		{
			name:       "最小长度8",
			length:     8,
			wantMinLen: 8,
		},
		{
			name:       "长度20",
			length:     20,
			wantMinLen: 20,
		},
		{
			name:       "长度小于最小值应使用最小长度",
			length:     5,
			wantMinLen: MinPasswordLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateSecurePassword(tt.length)
			if err != nil {
				t.Errorf("GenerateSecurePassword() 错误 = %v", err)
				return
			}

			// 验证长度
			if len(password) < tt.wantMinLen {
				t.Errorf("GenerateSecurePassword() 生成的密码长度 = %v, 期望至少 %v", len(password), tt.wantMinLen)
			}

			// 验证包含所有必需的字符类型
			var (
				hasUpper   bool
				hasLower   bool
				hasNumber  bool
				hasSpecial bool
			)

			for _, char := range password {
				switch {
				case char >= 'A' && char <= 'Z':
					hasUpper = true
				case char >= 'a' && char <= 'z':
					hasLower = true
				case char >= '0' && char <= '9':
					hasNumber = true
				case strings.ContainsRune("!@#$%^&*", char):
					hasSpecial = true
				}
			}

			if !hasUpper {
				t.Error("GenerateSecurePassword() 生成的密码缺少大写字母")
			}
			if !hasLower {
				t.Error("GenerateSecurePassword() 生成的密码缺少小写字母")
			}
			if !hasNumber {
				t.Error("GenerateSecurePassword() 生成的密码缺少数字")
			}
			if !hasSpecial {
				t.Error("GenerateSecurePassword() 生成的密码缺少特殊字符")
			}

			// 验证密码强度
			err = ValidatePasswordStrength(password)
			if err != nil {
				t.Errorf("GenerateSecurePassword() 生成的密码强度验证失败: %v, 密码: %s", err, password)
			}
		})
	}
}

func TestGenerateSecurePassword_Uniqueness(t *testing.T) {
	// 生成多个密码，验证它们是唯一的
	passwords := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		password, err := GenerateSecurePassword(16)
		if err != nil {
			t.Fatalf("GenerateSecurePassword() 错误 = %v", err)
		}

		if passwords[password] {
			t.Errorf("GenerateSecurePassword() 生成了重复的密码: %s", password)
		}
		passwords[password] = true
	}

	if len(passwords) != count {
		t.Errorf("GenerateSecurePassword() 生成的唯一密码数量 = %v, 期望 %v", len(passwords), count)
	}
}

// 基准测试
func BenchmarkHashPassword(b *testing.B) {
	password := "TestPassword123!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "TestPassword123!"
	hash, _ := HashPassword(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(hash, password)
	}
}

func BenchmarkValidatePasswordStrength(b *testing.B) {
	password := "TestPassword123!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePasswordStrength(password)
	}
}
