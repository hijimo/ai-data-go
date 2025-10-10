package kms

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Manager KMS管理器
type Manager struct {
	providers map[string]KMSProvider
	default_  KMSProvider
	mu        sync.RWMutex
}

// NewManager 创建KMS管理器
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]KMSProvider),
	}
}

// RegisterProvider 注册KMS提供商
func (m *Manager) RegisterProvider(name string, provider KMSProvider) error {
	if name == "" {
		return errors.New("提供商名称不能为空")
	}
	
	if provider == nil {
		return errors.New("提供商不能为nil")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.providers[name] = provider
	
	// 如果是第一个提供商，设置为默认
	if m.default_ == nil {
		m.default_ = provider
	}
	
	return nil
}

// SetDefaultProvider 设置默认提供商
func (m *Manager) SetDefaultProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return fmt.Errorf("提供商 %s 不存在", name)
	}
	
	m.default_ = provider
	return nil
}

// GetProvider 获取指定提供商
func (m *Manager) GetProvider(name string) (KMSProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("提供商 %s 不存在", name)
	}
	
	return provider, nil
}

// GetDefaultProvider 获取默认提供商
func (m *Manager) GetDefaultProvider() (KMSProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.default_ == nil {
		return nil, errors.New("没有设置默认KMS提供商")
	}
	
	return m.default_, nil
}

// ListProviders 列出所有提供商
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	
	return names
}

// Encrypt 使用默认提供商加密
func (m *Manager) Encrypt(ctx context.Context, plaintext string) (string, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return "", err
	}
	
	return provider.Encrypt(ctx, plaintext)
}

// EncryptWithProvider 使用指定提供商加密
func (m *Manager) EncryptWithProvider(ctx context.Context, providerName, plaintext string) (string, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return "", err
	}
	
	return provider.Encrypt(ctx, plaintext)
}

// Decrypt 使用默认提供商解密
func (m *Manager) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return "", err
	}
	
	return provider.Decrypt(ctx, ciphertext)
}

// DecryptWithProvider 使用指定提供商解密
func (m *Manager) DecryptWithProvider(ctx context.Context, providerName, ciphertext string) (string, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return "", err
	}
	
	return provider.Decrypt(ctx, ciphertext)
}

// GenerateDataKey 使用默认提供商生成数据密钥
func (m *Manager) GenerateDataKey(ctx context.Context, keySpec string) (*DataKey, error) {
	provider, err := m.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	
	return provider.GenerateDataKey(ctx, keySpec)
}

// HealthCheckAll 检查所有提供商的健康状态
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make(map[string]error)
	for name, provider := range m.providers {
		results[name] = provider.HealthCheck(ctx)
	}
	
	return results
}

// HealthCheck 检查指定提供商的健康状态
func (m *Manager) HealthCheck(ctx context.Context, providerName string) error {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return err
	}
	
	return provider.HealthCheck(ctx)
}

// CreateProvider 根据配置创建KMS提供商
func CreateProvider(config *KMSConfig) (KMSProvider, error) {
	if config == nil {
		return nil, errors.New("KMS配置不能为空")
	}
	
	switch config.Provider {
	case ProviderTypeAlibabaCloud:
		return NewAlibabaCloudKMS(config)
	case ProviderTypeLocal:
		return NewLocalKMS(config)
	default:
		return nil, fmt.Errorf("不支持的KMS提供商类型: %s", config.Provider)
	}
}

// InitializeFromConfigs 从配置列表初始化KMS管理器
func (m *Manager) InitializeFromConfigs(configs map[string]*KMSConfig, defaultProvider string) error {
	if len(configs) == 0 {
		return errors.New("KMS配置不能为空")
	}
	
	// 创建并注册所有提供商
	for name, config := range configs {
		provider, err := CreateProvider(config)
		if err != nil {
			return fmt.Errorf("创建KMS提供商 %s 失败: %w", name, err)
		}
		
		if err := m.RegisterProvider(name, provider); err != nil {
			return fmt.Errorf("注册KMS提供商 %s 失败: %w", name, err)
		}
	}
	
	// 设置默认提供商
	if defaultProvider != "" {
		if err := m.SetDefaultProvider(defaultProvider); err != nil {
			return fmt.Errorf("设置默认KMS提供商失败: %w", err)
		}
	}
	
	return nil
}