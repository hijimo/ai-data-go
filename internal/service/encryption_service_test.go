package service

import (
	"os"
	"strings"
	"testing"
)

func TestNewEncryptionService(t *testing.T) {
	tests := []struct {
		name      string
		secretKey []byte
		wantErr   bool
	}{
		{
			name:      "有效的32字节密钥",
			secretKey: []byte("12345678901234567890123456789012"),
			wantErr:   false,
		},
		{
			name:      "密钥长度不足32字节",
			secretKey: []byte("short"),
			wantErr:   true,
		},
		{
			name:      "密钥长度超过32字节",
			secretKey: []byte("123456789012345678901234567890123456"),
			wantErr:   true,
		},
		{
			name:      "空密钥",
			secretKey: []byte(""),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionService(tt.secretKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEncryptionServiceFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		wantErr bool
	}{
		{
			name:    "环境变量已设置",
			envKey:  "test-secret-key-32-bytes-long",
			wantErr: false,
		},
		{
			name:    "环境变量未设置",
			envKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			if tt.envKey != "" {
				os.Setenv("ENCRYPTION_SECRET_KEY", tt.envKey)
				defer os.Unsetenv("ENCRYPTION_SECRET_KEY")
			} else {
				os.Unsetenv("ENCRYPTION_SECRET_KEY")
			}

			_, err := NewEncryptionServiceFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionServiceFromEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionService_EncryptAPIKey(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012")
	service, err := NewEncryptionService(secretKey)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "正常加密",
			plaintext: "sk-test-api-key-12345",
			wantErr:   false,
		},
		{
			name:      "加密空字符串",
			plaintext: "",
			wantErr:   true,
		},
		{
			name:      "加密长字符串",
			plaintext: strings.Repeat("a", 1000),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := service.EncryptAPIKey(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证加密结果不为空
				if encrypted == "" {
					t.Error("加密结果不应为空")
				}

				// 验证加密结果与明文不同
				if encrypted == tt.plaintext {
					t.Error("加密结果不应与明文相同")
				}

				// 验证每次加密结果不同（因为使用随机nonce）
				encrypted2, err := service.EncryptAPIKey(tt.plaintext)
				if err != nil {
					t.Errorf("第二次加密失败: %v", err)
				}
				if encrypted == encrypted2 {
					t.Error("相同明文的两次加密结果应该不同")
				}
			}
		})
	}
}

func TestEncryptionService_DecryptAPIKey(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012")
	service, err := NewEncryptionService(secretKey)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}

	// 先加密一个密钥
	plaintext := "sk-test-api-key-12345"
	encrypted, err := service.EncryptAPIKey(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	tests := []struct {
		name      string
		encrypted string
		want      string
		wantErr   bool
	}{
		{
			name:      "正常解密",
			encrypted: encrypted,
			want:      plaintext,
			wantErr:   false,
		},
		{
			name:      "解密空字符串",
			encrypted: "",
			wantErr:   true,
		},
		{
			name:      "解密无效的Base64字符串",
			encrypted: "invalid-base64!!!",
			wantErr:   true,
		},
		{
			name:      "解密太短的密文",
			encrypted: "YWJj", // "abc" in base64
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.DecryptAPIKey(tt.encrypted)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecryptAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("DecryptAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012")
	service, err := NewEncryptionService(secretKey)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}

	testCases := []string{
		"sk-test-api-key-12345",
		"very-long-api-key-" + strings.Repeat("x", 100),
		"special-chars-!@#$%^&*()",
		"中文密钥测试",
	}

	for _, plaintext := range testCases {
		t.Run("加密解密循环测试: "+plaintext[:min(20, len(plaintext))], func(t *testing.T) {
			// 加密
			encrypted, err := service.EncryptAPIKey(plaintext)
			if err != nil {
				t.Fatalf("加密失败: %v", err)
			}

			// 解密
			decrypted, err := service.DecryptAPIKey(encrypted)
			if err != nil {
				t.Fatalf("解密失败: %v", err)
			}

			// 验证解密结果与原文相同
			if decrypted != plaintext {
				t.Errorf("解密结果不匹配: got %v, want %v", decrypted, plaintext)
			}
		})
	}
}

func TestEncryptionService_MaskAPIKey(t *testing.T) {
	secretKey := []byte("12345678901234567890123456789012")
	service, err := NewEncryptionService(secretKey)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}

	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{
			name:   "正常长度的密钥",
			apiKey: "sk-test-api-key-12345",
			want:   "sk-t****2345",
		},
		{
			name:   "短密钥（8个字符以下）",
			apiKey: "short",
			want:   "****",
		},
		{
			name:   "恰好8个字符",
			apiKey: "12345678",
			want:   "****",
		},
		{
			name:   "9个字符",
			apiKey: "123456789",
			want:   "1234****6789",
		},
		{
			name:   "空字符串",
			apiKey: "",
			want:   "",
		},
		{
			name:   "很长的密钥",
			apiKey: strings.Repeat("a", 100),
			want:   "aaaa****aaaa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.MaskAPIKey(tt.apiKey)
			if got != tt.want {
				t.Errorf("MaskAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptionService_DifferentKeys(t *testing.T) {
	// 使用不同的密钥创建两个服务
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("abcdefghijklmnopqrstuvwxyz123456")

	service1, err := NewEncryptionService(key1)
	if err != nil {
		t.Fatalf("创建服务1失败: %v", err)
	}

	service2, err := NewEncryptionService(key2)
	if err != nil {
		t.Fatalf("创建服务2失败: %v", err)
	}

	plaintext := "sk-test-api-key-12345"

	// 使用服务1加密
	encrypted, err := service1.EncryptAPIKey(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 尝试使用服务2解密（应该失败）
	_, err = service2.DecryptAPIKey(encrypted)
	if err == nil {
		t.Error("使用不同密钥解密应该失败")
	}

	// 使用服务1解密（应该成功）
	decrypted, err := service1.DecryptAPIKey(encrypted)
	if err != nil {
		t.Fatalf("使用正确密钥解密失败: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("解密结果不匹配: got %v, want %v", decrypted, plaintext)
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
