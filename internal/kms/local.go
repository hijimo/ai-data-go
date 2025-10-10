package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// LocalKMS 本地KMS实现（用于开发环境）
type LocalKMS struct {
	config *KMSConfig
	key    []byte
}

// NewLocalKMS 创建本地KMS客户端
func NewLocalKMS(config *KMSConfig) (*LocalKMS, error) {
	if config.KeyID == "" {
		return nil, errors.New("本地KMS密钥ID不能为空")
	}
	
	// 使用KeyID生成32字节的密钥
	hash := sha256.Sum256([]byte(config.KeyID))
	
	return &LocalKMS{
		config: config,
		key:    hash[:],
	}, nil
}

// Encrypt 加密数据
func (k *LocalKMS) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("明文不能为空")
	}
	
	// 创建AES加密器
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", fmt.Errorf("创建AES加密器失败: %w", err)
	}
	
	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}
	
	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// 创建加密结果
	result := &EncryptionResult{
		CiphertextBlob: base64.StdEncoding.EncodeToString(ciphertext),
		KeyID:          k.config.KeyID,
		Algorithm:      "AES_256_GCM",
		Metadata: map[string]string{
			"provider": string(ProviderTypeLocal),
		},
	}
	
	// 序列化结果
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化加密结果失败: %w", err)
	}
	
	return base64.StdEncoding.EncodeToString(resultBytes), nil
}

// Decrypt 解密数据
func (k *LocalKMS) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("密文不能为空")
	}
	
	// Base64解码
	resultBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}
	
	// 反序列化加密结果
	var result EncryptionResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return "", fmt.Errorf("反序列化加密结果失败: %w", err)
	}
	
	// 验证密钥ID
	if result.KeyID != k.config.KeyID {
		return "", errors.New("密钥ID不匹配")
	}
	
	// Base64解码密文
	ciphertextBytes, err := base64.StdEncoding.DecodeString(result.CiphertextBlob)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	
	// 创建AES解密器
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", fmt.Errorf("创建AES解密器失败: %w", err)
	}
	
	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}
	
	// 检查密文长度
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", errors.New("密文长度不足")
	}
	
	// 提取nonce和密文
	nonce, ciphertextData := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]
	
	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}
	
	return string(plaintext), nil
}

// GenerateDataKey 生成数据密钥
func (k *LocalKMS) GenerateDataKey(ctx context.Context, keySpec string) (*DataKey, error) {
	if keySpec == "" {
		keySpec = "AES_256"
	}
	
	// 生成32字节的随机数据密钥
	plaintext := make([]byte, 32)
	if _, err := rand.Read(plaintext); err != nil {
		return nil, fmt.Errorf("生成随机数据密钥失败: %w", err)
	}
	
	// 使用主密钥加密数据密钥
	ciphertextBlob, err := k.Encrypt(ctx, string(plaintext))
	if err != nil {
		return nil, fmt.Errorf("加密数据密钥失败: %w", err)
	}
	
	return &DataKey{
		KeyID:          k.config.KeyID,
		Plaintext:      plaintext,
		CiphertextBlob: ciphertextBlob,
	}, nil
}

// HealthCheck 健康检查
func (k *LocalKMS) HealthCheck(ctx context.Context) error {
	if k.config.KeyID == "" {
		return errors.New("本地KMS密钥ID未配置")
	}
	
	if len(k.key) != 32 {
		return errors.New("本地KMS密钥长度不正确")
	}
	
	// 测试加密解密
	testData := "health-check-test"
	encrypted, err := k.Encrypt(ctx, testData)
	if err != nil {
		return fmt.Errorf("健康检查加密失败: %w", err)
	}
	
	decrypted, err := k.Decrypt(ctx, encrypted)
	if err != nil {
		return fmt.Errorf("健康检查解密失败: %w", err)
	}
	
	if decrypted != testData {
		return errors.New("健康检查数据不匹配")
	}
	
	return nil
}

// GetProviderType 获取提供商类型
func (k *LocalKMS) GetProviderType() ProviderType {
	return ProviderTypeLocal
}