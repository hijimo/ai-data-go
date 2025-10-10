package kms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SecretManager 敏感信息管理器
type SecretManager struct {
	kmsManager *Manager
}

// NewSecretManager 创建敏感信息管理器
func NewSecretManager(kmsManager *Manager) *SecretManager {
	return &SecretManager{
		kmsManager: kmsManager,
	}
}

// SecretType 敏感信息类型
type SecretType string

const (
	SecretTypeAPIKey      SecretType = "api_key"
	SecretTypePassword    SecretType = "password"
	SecretTypeToken       SecretType = "token"
	SecretTypePrivateKey  SecretType = "private_key"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeDatabase    SecretType = "database"
	SecretTypeOther       SecretType = "other"
)

// Secret 敏感信息结构
type Secret struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        SecretType        `json:"type"`
	Description string            `json:"description,omitempty"`
	Value       string            `json:"value"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// EncryptedSecret 加密后的敏感信息
type EncryptedSecret struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Type             SecretType        `json:"type"`
	Description      string            `json:"description,omitempty"`
	EncryptedValue   string            `json:"encrypted_value"`
	KMSProvider      string            `json:"kms_provider"`
	EncryptionMethod string            `json:"encryption_method"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
}

// EncryptSecret 加密敏感信息
func (sm *SecretManager) EncryptSecret(ctx context.Context, secret *Secret) (*EncryptedSecret, error) {
	if secret == nil {
		return nil, errors.New("敏感信息不能为空")
	}
	
	if secret.Value == "" {
		return nil, errors.New("敏感信息值不能为空")
	}
	
	// 使用默认KMS提供商加密
	encryptedValue, err := sm.kmsManager.Encrypt(ctx, secret.Value)
	if err != nil {
		return nil, fmt.Errorf("加密敏感信息失败: %w", err)
	}
	
	// 获取默认提供商信息
	provider, err := sm.kmsManager.GetDefaultProvider()
	if err != nil {
		return nil, fmt.Errorf("获取默认KMS提供商失败: %w", err)
	}
	
	return &EncryptedSecret{
		ID:               secret.ID,
		Name:             secret.Name,
		Type:             secret.Type,
		Description:      secret.Description,
		EncryptedValue:   encryptedValue,
		KMSProvider:      string(provider.GetProviderType()),
		EncryptionMethod: "KMS",
		Metadata:         secret.Metadata,
		Tags:             secret.Tags,
	}, nil
}

// EncryptSecretWithProvider 使用指定提供商加密敏感信息
func (sm *SecretManager) EncryptSecretWithProvider(ctx context.Context, providerName string, secret *Secret) (*EncryptedSecret, error) {
	if secret == nil {
		return nil, errors.New("敏感信息不能为空")
	}
	
	if secret.Value == "" {
		return nil, errors.New("敏感信息值不能为空")
	}
	
	// 使用指定KMS提供商加密
	encryptedValue, err := sm.kmsManager.EncryptWithProvider(ctx, providerName, secret.Value)
	if err != nil {
		return nil, fmt.Errorf("加密敏感信息失败: %w", err)
	}
	
	return &EncryptedSecret{
		ID:               secret.ID,
		Name:             secret.Name,
		Type:             secret.Type,
		Description:      secret.Description,
		EncryptedValue:   encryptedValue,
		KMSProvider:      providerName,
		EncryptionMethod: "KMS",
		Metadata:         secret.Metadata,
		Tags:             secret.Tags,
	}, nil
}

// DecryptSecret 解密敏感信息
func (sm *SecretManager) DecryptSecret(ctx context.Context, encryptedSecret *EncryptedSecret) (*Secret, error) {
	if encryptedSecret == nil {
		return nil, errors.New("加密的敏感信息不能为空")
	}
	
	if encryptedSecret.EncryptedValue == "" {
		return nil, errors.New("加密值不能为空")
	}
	
	var decryptedValue string
	var err error
	
	// 根据KMS提供商类型选择解密方法
	if encryptedSecret.KMSProvider != "" {
		// 使用指定的KMS提供商解密
		decryptedValue, err = sm.kmsManager.DecryptWithProvider(ctx, encryptedSecret.KMSProvider, encryptedSecret.EncryptedValue)
	} else {
		// 使用默认KMS提供商解密
		decryptedValue, err = sm.kmsManager.Decrypt(ctx, encryptedSecret.EncryptedValue)
	}
	
	if err != nil {
		return nil, fmt.Errorf("解密敏感信息失败: %w", err)
	}
	
	return &Secret{
		ID:          encryptedSecret.ID,
		Name:        encryptedSecret.Name,
		Type:        encryptedSecret.Type,
		Description: encryptedSecret.Description,
		Value:       decryptedValue,
		Metadata:    encryptedSecret.Metadata,
		Tags:        encryptedSecret.Tags,
	}, nil
}

// EncryptAPIKey 加密API密钥
func (sm *SecretManager) EncryptAPIKey(ctx context.Context, name, apiKey string, metadata map[string]string) (*EncryptedSecret, error) {
	secret := &Secret{
		ID:          generateSecretID(name, SecretTypeAPIKey),
		Name:        name,
		Type:        SecretTypeAPIKey,
		Description: fmt.Sprintf("API密钥: %s", name),
		Value:       apiKey,
		Metadata:    metadata,
		Tags:        []string{"api", "key"},
	}
	
	return sm.EncryptSecret(ctx, secret)
}

// DecryptAPIKey 解密API密钥
func (sm *SecretManager) DecryptAPIKey(ctx context.Context, encryptedSecret *EncryptedSecret) (string, error) {
	if encryptedSecret.Type != SecretTypeAPIKey {
		return "", errors.New("不是API密钥类型的敏感信息")
	}
	
	secret, err := sm.DecryptSecret(ctx, encryptedSecret)
	if err != nil {
		return "", err
	}
	
	return secret.Value, nil
}

// EncryptDatabaseConfig 加密数据库配置
func (sm *SecretManager) EncryptDatabaseConfig(ctx context.Context, config map[string]string) (*EncryptedSecret, error) {
	// 将数据库配置序列化为JSON
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("序列化数据库配置失败: %w", err)
	}
	
	secret := &Secret{
		ID:          generateSecretID("database", SecretTypeDatabase),
		Name:        "database_config",
		Type:        SecretTypeDatabase,
		Description: "数据库连接配置",
		Value:       string(configBytes),
		Tags:        []string{"database", "config"},
	}
	
	return sm.EncryptSecret(ctx, secret)
}

// DecryptDatabaseConfig 解密数据库配置
func (sm *SecretManager) DecryptDatabaseConfig(ctx context.Context, encryptedSecret *EncryptedSecret) (map[string]string, error) {
	if encryptedSecret.Type != SecretTypeDatabase {
		return nil, errors.New("不是数据库配置类型的敏感信息")
	}
	
	secret, err := sm.DecryptSecret(ctx, encryptedSecret)
	if err != nil {
		return nil, err
	}
	
	var config map[string]string
	if err := json.Unmarshal([]byte(secret.Value), &config); err != nil {
		return nil, fmt.Errorf("反序列化数据库配置失败: %w", err)
	}
	
	return config, nil
}

// BatchEncryptSecrets 批量加密敏感信息
func (sm *SecretManager) BatchEncryptSecrets(ctx context.Context, secrets []*Secret) ([]*EncryptedSecret, error) {
	if len(secrets) == 0 {
		return nil, errors.New("敏感信息列表不能为空")
	}
	
	encryptedSecrets := make([]*EncryptedSecret, len(secrets))
	
	for i, secret := range secrets {
		encrypted, err := sm.EncryptSecret(ctx, secret)
		if err != nil {
			return nil, fmt.Errorf("加密第 %d 个敏感信息失败: %w", i+1, err)
		}
		encryptedSecrets[i] = encrypted
	}
	
	return encryptedSecrets, nil
}

// BatchDecryptSecrets 批量解密敏感信息
func (sm *SecretManager) BatchDecryptSecrets(ctx context.Context, encryptedSecrets []*EncryptedSecret) ([]*Secret, error) {
	if len(encryptedSecrets) == 0 {
		return nil, errors.New("加密的敏感信息列表不能为空")
	}
	
	secrets := make([]*Secret, len(encryptedSecrets))
	
	for i, encryptedSecret := range encryptedSecrets {
		secret, err := sm.DecryptSecret(ctx, encryptedSecret)
		if err != nil {
			return nil, fmt.Errorf("解密第 %d 个敏感信息失败: %w", i+1, err)
		}
		secrets[i] = secret
	}
	
	return secrets, nil
}

// ValidateSecret 验证敏感信息
func (sm *SecretManager) ValidateSecret(secret *Secret) error {
	if secret == nil {
		return errors.New("敏感信息不能为空")
	}
	
	if secret.Name == "" {
		return errors.New("敏感信息名称不能为空")
	}
	
	if secret.Value == "" {
		return errors.New("敏感信息值不能为空")
	}
	
	if secret.Type == "" {
		secret.Type = SecretTypeOther
	}
	
	// 根据类型进行特定验证
	switch secret.Type {
	case SecretTypeAPIKey:
		if len(secret.Value) < 10 {
			return errors.New("API密钥长度不能少于10个字符")
		}
	case SecretTypePassword:
		if len(secret.Value) < 8 {
			return errors.New("密码长度不能少于8个字符")
		}
	}
	
	return nil
}

// generateSecretID 生成敏感信息ID
func generateSecretID(name string, secretType SecretType) string {
	return fmt.Sprintf("%s_%s", strings.ToLower(string(secretType)), strings.ReplaceAll(strings.ToLower(name), " ", "_"))
}