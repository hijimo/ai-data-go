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
