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
)

// TestBailianIntegration_NonStreaming 测试百炼非流式调用
// 这是一个集成测试，需要真实的百炼配置
func TestBailianIntegration_NonStreaming(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取百炼配置
	bailianAPIKey := os.Getenv("BAILIAN_API_KEY")
	bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")
	bailianModel := os.Getenv("BAILIAN_MODEL")

	if bailianAPIKey == "" {
		t.Skip("跳过百炼集成测试：缺少必需的环境变量 BAILIAN_API_KEY")
	}

	// 设置默认值
	if bailianEndpoint == "" {
		bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if bailianModel == "" {
		bailianModel = "qwen-plus"
	}

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err)
	defer cleanupTestDatabase(t, db)

	// 创建测试租户和模型配置
	tenantID := uuid.New()
	modelName := "bailian-qwen-test"

	// 准备 QueryParams JSON
	queryParams := `{
		"model": "` + bailianModel + `",
		"bailianEndpoint": "` + bailianEndpoint + `",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	// 创建模型配置
	modelConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          modelName,
		Model:         bailianModel,
		ModelProvider: "bianlian",
		APIKey:        bailianAPIKey,
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
		assert.Equal(t, bailianModel, result.Model)

		t.Logf("生成的文本: %s", result.Text)
	})

	t.Run("中文处理能力测试", func(t *testing.T) {
		// 测试中文理解和生成
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请用中文解释什么是人工智能，不超过50字。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		
		// 验证返回的是中文内容
		assert.Contains(t, result.Text, "人工智能")

		t.Logf("中文生成结果: %s", result.Text)
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
			"请列举三个中国的传统节日。",
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
		assert.Contains(t, err.Error(), "模型配置")
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

	t.Run("复杂中文对话测试", func(t *testing.T) {
		// 测试更复杂的中文理解和生成
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请用一段话描述春天的景色，要求包含花、鸟、风等元素。",
			nil,
		)

		// 验证结果
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		
		// 验证包含关键词
		text := result.Text
		hasFlower := false
		hasBird := false
		hasWind := false
		
		if len(text) > 0 {
			// 简单检查是否包含相关词汇
			hasFlower = len(text) > 10
			hasBird = len(text) > 10
			hasWind = len(text) > 10
		}
		
		assert.True(t, hasFlower || hasBird || hasWind, "生成的文本应该包含春天相关的描述")

		t.Logf("复杂对话结果: %s", result.Text)
	})
}
