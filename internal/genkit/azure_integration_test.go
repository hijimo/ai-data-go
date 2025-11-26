package genkit

import (
	"context"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestAzureOpenAIIntegration_NonStreaming 测试 Azure OpenAI 非流式调用
// 这是一个集成测试，需要真实的 Azure OpenAI 配置
func TestAzureOpenAIIntegration_NonStreaming(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取 Azure OpenAI 配置
	azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	azureAPIVersion := os.Getenv("AZURE_OPENAI_API_VERSION")

	if azureAPIKey == "" || azureEndpoint == "" || azureDeployment == "" {
		t.Skip("跳过 Azure OpenAI 集成测试：缺少必需的环境变量")
	}

	// 设置默认的 API Version
	if azureAPIVersion == "" {
		azureAPIVersion = "2024-02-15-preview"
	}

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err)
	defer cleanupTestDatabase(t, db)

	// 创建测试租户和模型配置
	tenantID := uuid.New()
	modelName := "azure-gpt-4-test"

	// 准备 QueryParams JSON
	queryParams := `{
		"model": "gpt-4",
		"azureEndpoint": "` + azureEndpoint + `",
		"azureDeployment": "` + azureDeployment + `",
		"azureApiVersion": "` + azureAPIVersion + `",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	// 创建模型配置
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         "gpt-4",
		ModelProvider: model.ModelProviderAzureOpenAI,
		APIKey:        azureAPIKey,
		QueryParams:   &queryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	err = db.Create(modelConfig).Error
	require.NoError(t, err)

	// 创建 repository 和 client
	repo := repository.NewModelConfigurationRepository(db)
	client := NewClientWithRepo(repo)

	ctx := context.Background()

	t.Run("基本文本生成", func(t *testing.T) {
		// 调用 Generate 方法
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"你好，请用一句话介绍你自己。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		assert.Equal(t, "gpt-4", result.Model)

		t.Logf("生成的文本: %s", result.Text)
	})

	t.Run("参数传递测试", func(t *testing.T) {
		// 使用自定义选项
		temperature := 0.8
		maxTokens := 1000
		options := &GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个编程语言的名称。",
			options,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		t.Logf("生成的文本: %s", result.Text)
	})

	t.Run("Token 统计测试", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"计算 1+1 等于多少？",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)

		// 验证 Token 统计
		if result.Usage != nil {
			assert.Greater(t, result.Usage.PromptTokens, 0)
			assert.Greater(t, result.Usage.CompletionTokens, 0)
			assert.Greater(t, result.Usage.TotalTokens, 0)
			assert.Equal(t, result.Usage.PromptTokens+result.Usage.CompletionTokens, result.Usage.TotalTokens)

			t.Logf("Token 使用情况: Prompt=%d, Completion=%d, Total=%d",
				result.Usage.PromptTokens,
				result.Usage.CompletionTokens,
				result.Usage.TotalTokens)
		}
	})

	t.Run("错误处理 - 配置不存在", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置不存在")
	})

	t.Run("错误处理 - 租户ID无效", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"invalid-uuid",
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的租户ID")
	})

	t.Run("错误处理 - 模型已禁用", func(t *testing.T) {
		// 禁用模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"测试提示词",
			nil,
		)

		// 应该返回错误
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")

		// 恢复模型状态
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	t.Run("响应格式验证", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请说'你好'。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)

		// 验证响应格式
		assert.NotEmpty(t, result.Text)
		assert.NotEmpty(t, result.Model)
		assert.Contains(t, result.Text, "你好")

		t.Logf("响应格式验证通过: Text=%s, Model=%s", result.Text, result.Model)
	})

	t.Run("缓存机制测试", func(t *testing.T) {
		// 第一次调用（初始化实例）
		start1 := time.Now()
		result1, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第一次调用",
			nil,
		)
		duration1 := time.Since(start1)
		require.NoError(t, err)
		assert.NotNil(t, result1)

		// 第二次调用（使用缓存的实例）
		start2 := time.Now()
		result2, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第二次调用",
			nil,
		)
		duration2 := time.Since(start2)
		require.NoError(t, err)
		assert.NotNil(t, result2)

		t.Logf("第一次调用耗时: %v", duration1)
		t.Logf("第二次调用耗时: %v", duration2)

		// 注意：由于网络延迟等因素，第二次调用不一定更快
		// 这里只是记录日志，不做断言
	})
}

// setupTestDatabase 设置测试数据库
func setupTestDatabase(t *testing.T) (*gorm.DB, error) {
	// 从环境变量获取数据库连接信息
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// 使用默认值
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "genkit_test"
	}

	dsn := "host=" + dbHost + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " port=" + dbPort + " sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&model.ModelConfiguration{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// cleanupTestDatabase 清理测试数据库
func cleanupTestDatabase(t *testing.T, db *gorm.DB) {
	// 清理测试数据
	db.Exec("DELETE FROM model_configurations WHERE name LIKE '%test%'")
}
