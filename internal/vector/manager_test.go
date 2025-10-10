package vector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProviderFactory 模拟提供商工厂
type MockProviderFactory struct{}

func (f *MockProviderFactory) CreateProvider(ctx context.Context, config *Config) (VectorProvider, error) {
	if config.Provider == ProviderADBPG {
		return NewMockVectorProvider(), nil
	}
	return nil, ErrUnsupportedProvider
}

func (f *MockProviderFactory) SupportedProviders() []ProviderType {
	return []ProviderType{ProviderADBPG}
}

func TestManager(t *testing.T) {
	ctx := context.Background()
	factory := &MockProviderFactory{}
	manager := NewManager(factory)
	defer manager.Close()

	// 测试注册提供商
	t.Run("RegisterProvider", func(t *testing.T) {
		config := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
		}
		
		err := manager.RegisterProvider(ctx, "test_provider", config)
		require.NoError(t, err)
		
		// 验证提供商已注册
		providers := manager.ListProviders()
		assert.Contains(t, providers, "test_provider")
	})

	// 测试获取提供商
	t.Run("GetProvider", func(t *testing.T) {
		provider, err := manager.GetProvider("test_provider")
		require.NoError(t, err)
		assert.NotNil(t, provider)
		
		// 测试获取不存在的提供商
		_, err = manager.GetProvider("nonexistent")
		assert.Error(t, err)
	})

	// 测试获取提供商配置
	t.Run("GetProviderConfig", func(t *testing.T) {
		config, err := manager.GetProviderConfig("test_provider")
		require.NoError(t, err)
		assert.Equal(t, ProviderADBPG, config.Provider)
	})

	// 测试健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		results := manager.HealthCheck(ctx)
		assert.Contains(t, results, "test_provider")
		assert.NoError(t, results["test_provider"])
	})

	// 测试移除提供商
	t.Run("RemoveProvider", func(t *testing.T) {
		err := manager.RemoveProvider("test_provider")
		require.NoError(t, err)
		
		// 验证提供商已移除
		providers := manager.ListProviders()
		assert.NotContains(t, providers, "test_provider")
		
		// 测试移除不存在的提供商
		err = manager.RemoveProvider("nonexistent")
		assert.Error(t, err)
	})

	// 测试无效配置
	t.Run("InvalidConfig", func(t *testing.T) {
		config := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				// 缺少必需字段
			},
		}
		
		err := manager.RegisterProvider(ctx, "invalid_provider", config)
		assert.Error(t, err)
	})

	// 测试不支持的提供商
	t.Run("UnsupportedProvider", func(t *testing.T) {
		config := &Config{
			Provider: ProviderPinecone,
			Settings: map[string]interface{}{
				"api_key":     "test",
				"environment": "test",
			},
		}
		
		err := manager.RegisterProvider(ctx, "unsupported_provider", config)
		assert.Error(t, err)
	})
}

func TestConfig(t *testing.T) {
	// 测试ADBPG配置验证
	t.Run("ValidateADBPGConfig", func(t *testing.T) {
		config := &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
		}
		
		err := config.Validate()
		assert.NoError(t, err)
		
		// 测试获取ADBPG配置
		adbpgConfig, err := config.GetADBPGConfig()
		require.NoError(t, err)
		assert.Equal(t, "localhost", adbpgConfig.Host)
		assert.Equal(t, 5432, adbpgConfig.Port)
		assert.Equal(t, "test", adbpgConfig.Database)
	})

	// 测试配置验证失败
	t.Run("ValidateConfigFailure", func(t *testing.T) {
		// 空提供商
		config := &Config{}
		err := config.Validate()
		assert.Error(t, err)
		
		// 缺少设置
		config = &Config{
			Provider: ProviderADBPG,
		}
		err = config.Validate()
		assert.Error(t, err)
		
		// 缺少必需字段
		config = &Config{
			Provider: ProviderADBPG,
			Settings: map[string]interface{}{
				"host": "localhost",
				// 缺少其他必需字段
			},
		}
		err = config.Validate()
		assert.Error(t, err)
	})

	// 测试Pinecone配置验证
	t.Run("ValidatePineconeConfig", func(t *testing.T) {
		config := &Config{
			Provider: ProviderPinecone,
			Settings: map[string]interface{}{
				"api_key":     "test-key",
				"environment": "test-env",
			},
		}
		
		err := config.Validate()
		assert.NoError(t, err)
	})

	// 测试不支持的提供商
	t.Run("UnsupportedProvider", func(t *testing.T) {
		config := &Config{
			Provider: "unsupported",
			Settings: map[string]interface{}{},
		}
		
		err := config.Validate()
		assert.Error(t, err)
	})
}