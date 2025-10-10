package config

import (
	"os"

	"ai-knowledge-platform/internal/kms"
)

// KMSConfig KMS配置
type KMSConfig struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]*kms.KMSConfig `json:"providers"`
}

// GetKMSConfig 获取KMS配置
func GetKMSConfig() *KMSConfig {
	config := &KMSConfig{
		DefaultProvider: getEnvOrDefault("KMS_DEFAULT_PROVIDER", "local"),
		Providers:       make(map[string]*kms.KMSConfig),
	}
	
	// 本地KMS配置（用于开发环境）
	config.Providers["local"] = &kms.KMSConfig{
		Provider: kms.ProviderTypeLocal,
		KeyID:    getEnvOrDefault("KMS_LOCAL_KEY_ID", "local-development-key-id"),
	}
	
	// 阿里云KMS配置
	if accessKeyID := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID"); accessKeyID != "" {
		config.Providers["alibaba_cloud"] = &kms.KMSConfig{
			Provider:    kms.ProviderTypeAlibabaCloud,
			Region:      getEnvOrDefault("ALIBABA_CLOUD_REGION", "cn-hangzhou"),
			KeyID:       os.Getenv("ALIBABA_CLOUD_KMS_KEY_ID"),
			AccessKeyID: accessKeyID,
			SecretKey:   os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET"),
			Endpoint:    os.Getenv("ALIBABA_CLOUD_KMS_ENDPOINT"),
		}
		
		// 如果配置了阿里云KMS，设置为默认提供商
		if config.Providers["alibaba_cloud"].KeyID != "" {
			config.DefaultProvider = "alibaba_cloud"
		}
	}
	
	return config
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}