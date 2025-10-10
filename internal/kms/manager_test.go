package kms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_RegisterProvider(t *testing.T) {
	manager := NewManager()
	
	// 创建本地KMS提供商
	config := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-id",
	}
	
	provider, err := NewLocalKMS(config)
	assert.NoError(t, err)
	
	// 注册提供商
	err = manager.RegisterProvider("test", provider)
	assert.NoError(t, err)
	
	// 验证提供商已注册
	retrievedProvider, err := manager.GetProvider("test")
	assert.NoError(t, err)
	assert.Equal(t, provider, retrievedProvider)
}

func TestManager_SetDefaultProvider(t *testing.T) {
	manager := NewManager()
	
	// 创建并注册提供商
	config := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-id",
	}
	
	provider, err := NewLocalKMS(config)
	assert.NoError(t, err)
	
	err = manager.RegisterProvider("test", provider)
	assert.NoError(t, err)
	
	// 设置默认提供商
	err = manager.SetDefaultProvider("test")
	assert.NoError(t, err)
	
	// 验证默认提供商
	defaultProvider, err := manager.GetDefaultProvider()
	assert.NoError(t, err)
	assert.Equal(t, provider, defaultProvider)
}

func TestManager_EncryptDecrypt(t *testing.T) {
	manager := NewManager()
	
	// 创建并注册本地KMS提供商
	config := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-id",
	}
	
	provider, err := NewLocalKMS(config)
	assert.NoError(t, err)
	
	err = manager.RegisterProvider("local", provider)
	assert.NoError(t, err)
	
	err = manager.SetDefaultProvider("local")
	assert.NoError(t, err)
	
	ctx := context.Background()
	plaintext := "这是一个测试消息"
	
	// 加密
	ciphertext, err := manager.Encrypt(ctx, plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)
	
	// 解密
	decrypted, err := manager.Decrypt(ctx, ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestManager_HealthCheckAll(t *testing.T) {
	manager := NewManager()
	
	// 创建并注册多个提供商
	config1 := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-1",
	}
	
	provider1, err := NewLocalKMS(config1)
	assert.NoError(t, err)
	
	config2 := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-2",
	}
	
	provider2, err := NewLocalKMS(config2)
	assert.NoError(t, err)
	
	err = manager.RegisterProvider("local1", provider1)
	assert.NoError(t, err)
	
	err = manager.RegisterProvider("local2", provider2)
	assert.NoError(t, err)
	
	// 健康检查
	ctx := context.Background()
	results := manager.HealthCheckAll(ctx)
	
	assert.Len(t, results, 2)
	assert.NoError(t, results["local1"])
	assert.NoError(t, results["local2"])
}

func TestCreateProvider(t *testing.T) {
	// 测试创建本地KMS提供商
	config := &KMSConfig{
		Provider: ProviderTypeLocal,
		KeyID:    "test-key-id",
	}
	
	provider, err := CreateProvider(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTypeLocal, provider.GetProviderType())
	
	// 测试创建阿里云KMS提供商
	config = &KMSConfig{
		Provider:    ProviderTypeAlibabaCloud,
		KeyID:       "test-key-id",
		AccessKeyID: "test-access-key",
		SecretKey:   "test-secret-key",
	}
	
	provider, err = CreateProvider(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTypeAlibabaCloud, provider.GetProviderType())
	
	// 测试不支持的提供商类型
	config = &KMSConfig{
		Provider: ProviderType("unsupported"),
		KeyID:    "test-key-id",
	}
	
	provider, err = CreateProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "不支持的KMS提供商类型")
}

func TestManager_InitializeFromConfigs(t *testing.T) {
	manager := NewManager()
	
	configs := map[string]*KMSConfig{
		"local": {
			Provider: ProviderTypeLocal,
			KeyID:    "local-key-id",
		},
		"alibaba": {
			Provider:    ProviderTypeAlibabaCloud,
			KeyID:       "alibaba-key-id",
			AccessKeyID: "test-access-key",
			SecretKey:   "test-secret-key",
		},
	}
	
	err := manager.InitializeFromConfigs(configs, "local")
	assert.NoError(t, err)
	
	// 验证提供商已注册
	providers := manager.ListProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "local")
	assert.Contains(t, providers, "alibaba")
	
	// 验证默认提供商
	defaultProvider, err := manager.GetDefaultProvider()
	assert.NoError(t, err)
	assert.Equal(t, ProviderTypeLocal, defaultProvider.GetProviderType())
}