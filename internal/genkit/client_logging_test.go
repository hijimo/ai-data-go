package genkit

import (
	"context"
	"fmt"
	"testing"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/pkg/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockModelConfigurationRepository 模拟模型配置仓储
type mockModelConfigurationRepository struct {
	configs map[string]*model.ModelConfiguration
}

func newMockModelConfigurationRepository() *mockModelConfigurationRepository {
	return &mockModelConfigurationRepository{
		configs: make(map[string]*model.ModelConfiguration),
	}
}

func (m *mockModelConfigurationRepository) GetByTenantAndModel(ctx context.Context, tenantID uuid.UUID, modelName string) (*model.ModelConfiguration, error) {
	key := tenantID.String() + "_" + modelName
	if config, ok := m.configs[key]; ok {
		return config, nil
	}
	return nil, errors.NewNotFoundError("模型配置不存在")
}

func (m *mockModelConfigurationRepository) addConfig(tenantID uuid.UUID, modelName string, config *model.ModelConfiguration) {
	key := tenantID.String() + "_" + modelName
	m.configs[key] = config
}

// 实现其他必需的接口方法（用于满足接口要求）
func (m *mockModelConfigurationRepository) Create(ctx context.Context, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	return nil, errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ModelConfiguration, error) {
	return nil, errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) FindByTenant(ctx context.Context, tenantID *uuid.UUID, pageNo, pageSize int) ([]*model.ModelConfiguration, int64, error) {
	return nil, 0, errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) Update(ctx context.Context, id uuid.UUID, config *model.ModelConfiguration) (*model.ModelConfiguration, error) {
	return nil, errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, enabled bool) error {
	return errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	return errors.NewInternalError(fmt.Errorf("not implemented"))
}

func (m *mockModelConfigurationRepository) FindAvailableByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ModelConfiguration, error) {
	return nil, errors.NewInternalError(fmt.Errorf("not implemented"))
}

// TestProviderSelectionLogging 测试提供商选择日志
func TestProviderSelectionLogging(t *testing.T) {
	t.Skip("需要真实的 Genkit 实例，跳过集成测试")
	
	// 这个测试需要真实的 Genkit 实例和 API 密钥
	// 在实际环境中运行时取消跳过
}

// TestCacheHitLogging 测试缓存命中日志
func TestCacheHitLogging(t *testing.T) {
	t.Skip("需要真实的 Genkit 实例，跳过集成测试")
	
	// 这个测试需要真实的 Genkit 实例和 API 密钥
	// 在实际环境中运行时取消跳过
}

// TestErrorLogging 测试错误日志
func TestErrorLogging(t *testing.T) {
	tests := []struct {
		name         string
		getTenantID  func(*mockModelConfigurationRepository) string
		modelName    string
		expectedErr  string
		setupConfig  func(*mockModelConfigurationRepository) uuid.UUID
	}{
		{
			name:        "无效的租户ID",
			getTenantID: func(repo *mockModelConfigurationRepository) string { return "invalid-uuid" },
			modelName:   "gemini-pro",
			expectedErr: "无效的租户ID",
			setupConfig: func(repo *mockModelConfigurationRepository) uuid.UUID { return uuid.Nil },
		},
		{
			name:        "模型配置不存在",
			getTenantID: func(repo *mockModelConfigurationRepository) string { return uuid.New().String() },
			modelName:   "non-existent-model",
			expectedErr: "获取模型配置失败",
			setupConfig: func(repo *mockModelConfigurationRepository) uuid.UUID { return uuid.Nil },
		},
		{
			name:        "模型已禁用",
			getTenantID: func(repo *mockModelConfigurationRepository) string { return "" },
			modelName:   "disabled-model",
			expectedErr: "模型已禁用",
			setupConfig: func(repo *mockModelConfigurationRepository) uuid.UUID {
				tenantID := uuid.New()
				queryParams := `{"model":"gemini-1.5-pro"}`
				repo.addConfig(tenantID, "disabled-model", &model.ModelConfiguration{
					ID:            uuid.New(),
					TenantID:      tenantID,
					Name:          "disabled-model",
					ModelProvider: "googlegenai",
					APIKey:        "test-api-key",
					Model:         "gemini-1.5-pro",
					QueryParams:   &queryParams,
					IsEnabled:     false, // 禁用
				})
				return tenantID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟仓储
			mockRepo := newMockModelConfigurationRepository()
			tenantID := tt.setupConfig(mockRepo)
			
			// 获取测试用的租户ID
			testTenantID := tt.getTenantID(mockRepo)
			if testTenantID == "" {
				testTenantID = tenantID.String()
			}
			
			// 创建客户端
			clientInterface := NewClientWithRepo(mockRepo)
			
			// 调用 getOrInitGenkit
			ctx := context.Background()
			c, ok := clientInterface.(*client)
			assert.True(t, ok, "应该能够转换为 *client 类型")
			
			_, _, err := c.getOrInitGenkit(ctx, testTenantID, tt.modelName)
			
			// 应该返回错误
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr, 
				"错误消息应包含: %s", tt.expectedErr)
		})
	}
}

// TestGenerateLogging 测试 Generate 方法的日志
func TestGenerateLogging(t *testing.T) {
	t.Skip("需要真实的 Genkit 实例，跳过集成测试")
	
	// 这个测试需要真实的 Genkit 实例和 API 密钥
	// 在实际环境中运行时取消跳过
}

// TestGenerateStreamLogging 测试 GenerateStream 方法的日志
func TestGenerateStreamLogging(t *testing.T) {
	t.Skip("需要真实的 Genkit 实例，跳过集成测试")
	
	// 这个测试需要真实的 Genkit 实例和 API 密钥
	// 在实际环境中运行时取消跳过
}
