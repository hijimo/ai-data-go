package kms

import (
	"context"
)

// KMSProvider KMS提供商接口
type KMSProvider interface {
	// Encrypt 加密数据
	Encrypt(ctx context.Context, plaintext string) (string, error)
	
	// Decrypt 解密数据
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	
	// GenerateDataKey 生成数据密钥
	GenerateDataKey(ctx context.Context, keySpec string) (*DataKey, error)
	
	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
	
	// GetProviderType 获取提供商类型
	GetProviderType() ProviderType
}

// ProviderType KMS提供商类型
type ProviderType string

const (
	ProviderTypeAlibabaCloud ProviderType = "alibaba_cloud"
	ProviderTypeAWS          ProviderType = "aws"
	ProviderTypeLocal        ProviderType = "local" // 本地加密，用于开发环境
)

// DataKey 数据密钥结构
type DataKey struct {
	KeyID         string `json:"key_id"`
	Plaintext     []byte `json:"plaintext"`
	CiphertextBlob string `json:"ciphertext_blob"`
}

// KMSConfig KMS配置
type KMSConfig struct {
	Provider    ProviderType `json:"provider"`
	Region      string       `json:"region,omitempty"`
	KeyID       string       `json:"key_id"`
	AccessKeyID string       `json:"access_key_id,omitempty"`
	SecretKey   string       `json:"secret_key,omitempty"`
	Endpoint    string       `json:"endpoint,omitempty"`
}

// EncryptionResult 加密结果
type EncryptionResult struct {
	CiphertextBlob string            `json:"ciphertext_blob"`
	KeyID          string            `json:"key_id"`
	Algorithm      string            `json:"algorithm"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DecryptionResult 解密结果
type DecryptionResult struct {
	Plaintext string            `json:"plaintext"`
	KeyID     string            `json:"key_id"`
	Algorithm string            `json:"algorithm"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}