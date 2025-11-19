package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// EncryptionService API密钥加密服务接口
type EncryptionService interface {
	// EncryptAPIKey 加密API密钥
	EncryptAPIKey(plaintext string) (string, error)
	
	// DecryptAPIKey 解密API密钥
	DecryptAPIKey(encrypted string) (string, error)
	
	// MaskAPIKey 脱敏API密钥（显示前4位和后4位）
	MaskAPIKey(apiKey string) string
}

// encryptionService 加密服务实现
type encryptionService struct {
	secretKey []byte
}

// NewEncryptionService 创建新的加密服务实例
// secretKey 必须是32字节（AES-256）
func NewEncryptionService(secretKey []byte) (EncryptionService, error) {
	if len(secretKey) != 32 {
		return nil, errors.New("加密密钥必须是32字节（AES-256）")
	}
	
	return &encryptionService{
		secretKey: secretKey,
	}, nil
}

// NewEncryptionServiceFromEnv 从环境变量创建加密服务实例
func NewEncryptionServiceFromEnv() (EncryptionService, error) {
	secretKeyStr := os.Getenv("ENCRYPTION_SECRET_KEY")
	if secretKeyStr == "" {
		return nil, errors.New("环境变量 ENCRYPTION_SECRET_KEY 未设置")
	}
	
	// 将字符串转换为32字节密钥
	// 如果密钥长度不足32字节，用0填充；如果超过32字节，截断
	secretKey := make([]byte, 32)
	copy(secretKey, []byte(secretKeyStr))
	
	return NewEncryptionService(secretKey)
}

// EncryptAPIKey 使用AES-256-GCM加密API密钥
func (s *encryptionService) EncryptAPIKey(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("明文不能为空")
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}
	
	// 加密数据（nonce会被添加到密文前面）
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// 使用Base64编码返回
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey 解密API密钥
func (s *encryptionService) DecryptAPIKey(encrypted string) (string, error) {
	if encrypted == "" {
		return "", errors.New("密文不能为空")
	}
	
	// Base64解码
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 检查密文长度
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文太短")
	}
	
	// 提取nonce和实际密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}
	
	return string(plaintext), nil
}

// MaskAPIKey 脱敏API密钥（显示前4位和后4位）
func (s *encryptionService) MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	
	// 如果密钥长度小于等于8，全部用星号替换
	if len(apiKey) <= 8 {
		return "****"
	}
	
	// 显示前4位和后4位，中间用星号替换
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}
