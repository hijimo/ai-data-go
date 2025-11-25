package genkit

import (
	"context"
	"testing"

	"genkit-ai-service/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockModelConfigurationRepository 模拟模型配置仓储
type MockModelConfigurationRepository struct {
	mock.Mock
}

func (m *MockModelConfigurationRepository) Create(ctx context.Context, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelConfiguration), args.Error(1)
}

func (m *MockModelConfigurationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelConfiguration), args.Error(1)
}

func (m *MockModelConfigurationRepository) FindByTenant(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error) {
	args := m.Called(ctx, tenantID, pageNo, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.ModelConfiguration), args.Get(1).(int64), args.Error(2)
}

func (m *MockModelConfigurationRepository) Update(ctx context.Context, id uuid.UUID, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	args := m.Called(ctx, id, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelConfiguration), args.Error(1)
}

func (m *MockModelConfigurationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error {
	args := m.Called(ctx, id, enabled)
	return args.Error(0)
}

func (m *MockModelConfigurationRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	args := m.Called(ctx, id, deletedBy)
	return args.Error(0)
}

func (m *MockModelConfigurationRepository) FindAvailableByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ModelConfiguration, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ModelConfiguration), args.Error(1)
}

func (m *MockModelConfigurationRepository) GetByTenantAndModel(ctx context.Context, tenantID uuid.UUID, modelName string) (*model.ModelConfiguration, error) {
	args := m.Called(ctx, tenantID, modelName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelConfiguration), args.Error(1)
}

// TestGetOrInitGenkit_Success 测试成功获取或初始化 Genkit 实例
func TestGetOrInitGenkit_Success(t *testing.T) {
	// 跳过此测试，因为它需要实际的 Google API key
	// 实际的初始化测试应该在集成测试中进行
	t.Skip("需要实际的 Google API key，应在集成测试中进行")
}

// TestGetOrInitGenkit_CachedInstance 测试从缓存获取实例
func TestGetOrInitGenkit_CachedInstance(t *testing.T) {
	// 跳过此测试，因为它需要实际的 Google API key
	t.Skip("需要实际的 Google API key，应在集成测试中进行")
}

// TestGetOrInitGenkit_ModelDisabled 测试模型已禁用的情况
func TestGetOrInitGenkit_ModelDisabled(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "gemini-pro"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 准备测试配置（模型已禁用）
	queryParams := `{"model":"gemini-1.5-pro"}`
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        "test-api-key",
		QueryParams:   &queryParams,
		IsEnabled:     false, // 模型已禁用
	}

	// 设置模拟期望
	mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).Return(modelConfig, nil)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "模型已禁用")

	mockRepo.AssertExpectations(t)
}

// TestGetOrInitGenkit_InvalidTenantID 测试无效的租户ID
func TestGetOrInitGenkit_InvalidTenantID(t *testing.T) {
	ctx := context.Background()
	invalidTenantID := "invalid-uuid"
	modelName := "gemini-pro"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, invalidTenantID, modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "无效的租户ID")
}

// TestGetOrInitGenkit_ConfigNotFound 测试配置不存在的情况
func TestGetOrInitGenkit_ConfigNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "non-existent-model"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 设置模拟期望：返回错误
	mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).Return(nil, assert.AnError)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "获取模型配置失败")

	mockRepo.AssertExpectations(t)
}

// TestGetOrInitGenkit_InvalidConfig 测试无效配置的情况
func TestGetOrInitGenkit_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "azure-gpt4"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 准备测试配置（Azure 配置不完整）
	queryParams := `{"model":"gpt-4"}` // 缺少必需的 Azure 字段
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gpt-4",
		ModelProvider: "azureopenai",
		APIKey:        "test-api-key",
		QueryParams:   &queryParams,
		IsEnabled:     true,
	}

	// 设置模拟期望
	mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).Return(modelConfig, nil)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "配置验证失败")

	mockRepo.AssertExpectations(t)
}

// TestGetOrInitGenkit_UnsupportedProvider 测试不支持的提供商
func TestGetOrInitGenkit_UnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "unknown-model"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 准备测试配置（不支持的提供商）
	queryParams := `{"model":"unknown-model"}`
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "unknown-model",
		ModelProvider: "unknown-provider",
		APIKey:        "test-api-key",
		QueryParams:   &queryParams,
		IsEnabled:     true,
	}

	// 设置模拟期望
	mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).Return(modelConfig, nil)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	// 应该在配置验证阶段就失败
	assert.Contains(t, err.Error(), "配置验证失败")

	mockRepo.AssertExpectations(t)
}

// TestGetOrInitGenkit_NoRepository 测试未注入仓储的情况
func TestGetOrInitGenkit_NoRepository(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "gemini-pro"

	// 创建客户端（不注入仓储）
	client := NewClient().(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "模型配置仓储未初始化")
}

// TestGetOrInitGenkit_InvalidJSON 测试无效的 JSON 配置
func TestGetOrInitGenkit_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	modelName := "gemini-pro"

	// 创建模拟仓储
	mockRepo := new(MockModelConfigurationRepository)

	// 准备测试配置（无效的 JSON）
	invalidJSON := `{invalid json}`
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        "test-api-key",
		QueryParams:   &invalidJSON,
		IsEnabled:     true,
	}

	// 设置模拟期望
	mockRepo.On("GetByTenantAndModel", ctx, tenantID, modelName).Return(modelConfig, nil)

	// 创建客户端
	client := NewClientWithRepo(mockRepo).(*client)

	// 调用方法
	g, genkitConfig, err := client.getOrInitGenkit(ctx, tenantID.String(), modelName)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, g)
	assert.Nil(t, genkitConfig)
	assert.Contains(t, err.Error(), "解析模型配置失败")

	mockRepo.AssertExpectations(t)
}

// TestParseUUID 测试 UUID 解析函数
func TestParseUUID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "有效的 UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "无效的 UUID",
			input:   "invalid-uuid",
			wantErr: true,
		},
		{
			name:    "空字符串",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseUUID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, result)
			}
		})
	}
}

// TestClearCache 测试清理指定缓存
func TestClearCache(t *testing.T) {
	// 跳过此测试，因为它需要实际的 Google API key
	t.Skip("需要实际的 Google API key，应在集成测试中进行")
}

// TestClearAllCache 测试清理所有缓存
func TestClearAllCache(t *testing.T) {
	// 跳过此测试，因为它需要实际的 Google API key
	t.Skip("需要实际的 Google API key，应在集成测试中进行")
}

// TestGetCacheSize 测试获取缓存大小
func TestGetCacheSize(t *testing.T) {
	// 创建客户端
	client := NewClient().(*client)

	// 初始状态应该为 0
	assert.Equal(t, 0, client.GetCacheSize())

	// 手动添加缓存项
	client.mu.Lock()
	client.instances["test_key_1"] = nil
	client.instances["test_key_2"] = nil
	client.mu.Unlock()

	// 验证缓存大小
	assert.Equal(t, 2, client.GetCacheSize())
}

// TestClose 测试关闭客户端
func TestClose(t *testing.T) {
	// 创建客户端
	client := NewClient().(*client)

	// 手动添加缓存项
	client.mu.Lock()
	client.instances["test_key_1"] = nil
	client.instances["test_key_2"] = nil
	client.mu.Unlock()

	// 验证缓存存在
	assert.Equal(t, 2, client.GetCacheSize())

	// 关闭客户端
	err := client.Close()
	assert.NoError(t, err)

	// 验证缓存已清理
	assert.Equal(t, 0, client.GetCacheSize())
}

// TestConcurrentGetOrInitGenkit 测试并发访问缓存的安全性
func TestConcurrentGetOrInitGenkit(t *testing.T) {
	// 跳过此测试，因为它需要实际的 Google API key
	t.Skip("需要实际的 Google API key，应在集成测试中进行")
}
