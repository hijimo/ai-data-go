package vector

import (
	"context"
	"fmt"
	"sync"
)

// Manager 向量存储管理器
type Manager struct {
	providers map[string]VectorProvider
	configs   map[string]*Config
	mu        sync.RWMutex
	factory   ProviderFactory
}

// ProviderFactory 向量提供商工厂接口
type ProviderFactory interface {
	CreateProvider(ctx context.Context, config *Config) (VectorProvider, error)
	SupportedProviders() []ProviderType
}

// NewManager 创建向量存储管理器
func NewManager(factory ProviderFactory) *Manager {
	return &Manager{
		providers: make(map[string]VectorProvider),
		configs:   make(map[string]*Config),
		factory:   factory,
	}
}

// RegisterProvider 注册向量提供商
func (m *Manager) RegisterProvider(ctx context.Context, name string, config *Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	provider, err := m.factory.CreateProvider(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	// 健康检查
	if err := provider.HealthCheck(ctx); err != nil {
		provider.Close()
		return fmt.Errorf("provider health check failed: %w", err)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 如果已存在同名提供商，先关闭旧的
	if oldProvider, exists := m.providers[name]; exists {
		oldProvider.Close()
	}
	
	m.providers[name] = provider
	m.configs[name] = config
	
	return nil
}

// GetProvider 获取向量提供商
func (m *Manager) GetProvider(name string) (VectorProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	
	return provider, nil
}

// RemoveProvider 移除向量提供商
func (m *Manager) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	provider, exists := m.providers[name]
	if !exists {
		return fmt.Errorf("provider not found: %s", name)
	}
	
	if err := provider.Close(); err != nil {
		return fmt.Errorf("failed to close provider: %w", err)
	}
	
	delete(m.providers, name)
	delete(m.configs, name)
	
	return nil
}

// ListProviders 列出所有注册的提供商
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	
	return names
}

// GetProviderConfig 获取提供商配置
func (m *Manager) GetProviderConfig(name string) (*Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	config, exists := m.configs[name]
	if !exists {
		return nil, fmt.Errorf("provider config not found: %s", name)
	}
	
	return config, nil
}

// HealthCheck 检查所有提供商的健康状态
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make(map[string]error)
	for name, provider := range m.providers {
		results[name] = provider.HealthCheck(ctx)
	}
	
	return results
}

// Close 关闭所有提供商连接
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for name, provider := range m.providers {
		if err := provider.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close provider %s: %w", name, err)
		}
	}
	
	// 清空所有提供商
	m.providers = make(map[string]VectorProvider)
	m.configs = make(map[string]*Config)
	
	return lastErr
}

// DefaultProviderFactory 默认提供商工厂
type DefaultProviderFactory struct{}

// CreateProvider 创建向量提供商实例
func (f *DefaultProviderFactory) CreateProvider(ctx context.Context, config *Config) (VectorProvider, error) {
	switch config.Provider {
	case ProviderADBPG:
		return NewADBPGProvider(ctx, config)
	case ProviderPinecone:
		return nil, fmt.Errorf("pinecone provider not implemented yet")
	case ProviderWeaviate:
		return nil, fmt.Errorf("weaviate provider not implemented yet")
	case ProviderChroma:
		return nil, fmt.Errorf("chroma provider not implemented yet")
	case ProviderMilvus:
		return nil, fmt.Errorf("milvus provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// SupportedProviders 返回支持的提供商列表
func (f *DefaultProviderFactory) SupportedProviders() []ProviderType {
	return []ProviderType{
		ProviderADBPG,
		// 其他提供商将在后续实现
	}
}

// NewDefaultProviderFactory 创建默认提供商工厂
func NewDefaultProviderFactory() ProviderFactory {
	return &DefaultProviderFactory{}
}