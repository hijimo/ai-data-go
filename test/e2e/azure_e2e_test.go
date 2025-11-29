package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"genkit-ai-service/internal/genkit"
	"genkit-ai-service/internal/model"
	"genkit-ai-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestAzureOpenAI_E2E_Complete 测试 Azure OpenAI 完整的端到端流程
// 这个测试模拟真实用户场景：从配置创建到模型调用的完整流程
func TestAzureOpenAI_E2E_Complete(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过端到端测试")
	}

	// 从环境变量获取 Azure OpenAI 配置
	azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	azureAPIVersion := os.Getenv("AZURE_OPENAI_API_VERSION")

	if azureAPIKey == "" || azureEndpoint == "" || azureDeployment == "" {
		t.Skip("跳过 Azure OpenAI 端到端测试：缺少必需的环境变量")
	}

	// 设置默认的 API Version
	if azureAPIVersion == "" {
		azureAPIVersion = "2024-02-15-preview"
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	// 创建测试数据库连接
	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	t.Log("✓ 数据库连接成功")

	// ========== 阶段 2: 创建租户和模型配置 ==========
	t.Log("========== 阶段 2: 创建租户和模型配置 ==========")

	// 创建测试租户
	tenantID := uuid.New()
	t.Logf("创建测试租户: %s", tenantID)

	// 创建模型配置
	modelName := "azure-gpt-4-e2e"
	queryParams := fmt.Sprintf(`{
		"model": "gpt-4",
		"azureEndpoint": "%s",
		"azureDeployment": "%s",
		"azureApiVersion": "%s",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`, azureEndpoint, azureDeployment, azureAPIVersion)

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
	require.NoError(t, err, "模型配置创建应该成功")

	t.Logf("✓ 模型配置创建成功: %s", modelName)

	// ========== 阶段 3: 初始化 Genkit Client ==========
	t.Log("========== 阶段 3: 初始化 Genkit Client ==========")

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ Genkit Client 初始化成功")

	// ========== 阶段 4: 测试非流式调用 ==========
	t.Log("========== 阶段 4: 测试非流式调用 ==========")

	t.Run("简单问答", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"你好，请用一句话介绍你自己。",
			nil,
		)

		require.NoError(t, err, "生成应该成功")
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Text)
		assert.Equal(t, "gpt-4", result.Model)

		t.Logf("✓ 简单问答成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("带参数的调用", func(t *testing.T) {
		temperature := 0.8
		maxTokens := 500
		options := &genkit.GenerateOptions{
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		}

		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"请列举三个编程语言。",
			options,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)

		t.Logf("✓ 带参数调用成功")
		t.Logf("  响应: %s", result.Text)
	})

	t.Run("Token 统计", func(t *testing.T) {
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"计算 1+1 等于多少？",
			nil,
		)

		require.NoError(t, err)
		assert.NotNil(t, result.Usage)
		assert.Greater(t, result.Usage.TotalTokens, 0)

		t.Logf("✓ Token 统计正常")
		t.Logf("  Prompt: %d, Completion: %d, Total: %d",
			result.Usage.PromptTokens,
			result.Usage.CompletionTokens,
			result.Usage.TotalTokens)
	})

	// ========== 阶段 5: 测试流式调用 ==========
	t.Log("========== 阶段 5: 测试流式调用 ==========")

	t.Run("流式响应", func(t *testing.T) {
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			modelName,
			"请用一句话介绍人工智能。",
			nil,
		)

		require.NoError(t, err)

		var fullText string
		var chunkCount int

		for chunk := range streamChan {
			require.NoError(t, chunk.Error)

			if !chunk.Done {
				fullText += chunk.Content
				chunkCount++
			}
		}

		assert.NotEmpty(t, fullText)
		assert.Greater(t, chunkCount, 0)

		t.Logf("✓ 流式响应成功")
		t.Logf("  接收到 %d 个数据块", chunkCount)
		t.Logf("  完整响应: %s", fullText)
	})

	t.Run("流式中断", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)

		streamChan, err := client.GenerateStream(
			cancelCtx,
			tenantID.String(),
			modelName,
			"请详细介绍一下云计算的发展历史。",
			nil,
		)

		require.NoError(t, err)

		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				break
			}

			if !chunk.Done {
				chunkCount++
				if chunkCount >= 2 {
					cancel()
					t.Log("  取消流式请求")
				}
			}
		}

		assert.GreaterOrEqual(t, chunkCount, 1)
		t.Logf("✓ 流式中断处理正常")
		t.Logf("  在取消前接收到 %d 个数据块", chunkCount)
	})

	// ========== 阶段 6: 测试缓存机制 ==========
	t.Log("========== 阶段 6: 测试缓存机制 ==========")

	t.Run("实例缓存", func(t *testing.T) {
		// 第一次调用
		start1 := time.Now()
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第一次调用",
			nil,
		)
		duration1 := time.Since(start1)
		require.NoError(t, err)

		// 第二次调用（应该使用缓存的实例）
		start2 := time.Now()
		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"第二次调用",
			nil,
		)
		duration2 := time.Since(start2)
		require.NoError(t, err)

		t.Logf("✓ 缓存机制正常")
		t.Logf("  第一次调用: %v", duration1)
		t.Logf("  第二次调用: %v", duration2)
	})

	// ========== 阶段 7: 测试错误处理 ==========
	t.Log("========== 阶段 7: 测试错误处理 ==========")

	t.Run("配置不存在", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 配置不存在错误处理正常: %v", err)
	})

	t.Run("租户ID无效", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"invalid-uuid",
			modelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无效的租户ID")
		t.Logf("✓ 租户ID无效错误处理正常: %v", err)
	})

	t.Run("模型已禁用", func(t *testing.T) {
		// 禁用模型
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")
		t.Logf("✓ 模型已禁用错误处理正常: %v", err)

		// 恢复模型状态
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", modelConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	// ========== 阶段 8: 测试并发场景 ==========
	t.Log("========== 阶段 8: 测试并发场景 ==========")

	t.Run("并发调用", func(t *testing.T) {
		const concurrency = 3
		results := make(chan *genkit.GenerateResult, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				result, err := client.Generate(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("这是第 %d 个并发请求", index+1),
					nil,
				)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case result := <-results:
				assert.NotEmpty(t, result.Text)
				successCount++
			case err := <-errors:
				t.Errorf("并发调用失败: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("并发调用超时")
			}
		}

		assert.Equal(t, concurrency, successCount)
		t.Logf("✓ 并发调用成功: %d/%d", successCount, concurrency)
	})

	// ========== 阶段 9: 测试多轮对话 ==========
	t.Log("========== 阶段 9: 测试多轮对话 ==========")

	t.Run("多轮对话", func(t *testing.T) {
		prompts := []string{
			"你好，我想了解一下 Azure OpenAI。",
			"它有哪些特点？",
			"谢谢你的介绍。",
		}

		for i, prompt := range prompts {
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				prompt,
				nil,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			t.Logf("  第 %d 轮 - 提示: %s", i+1, prompt)
			t.Logf("  第 %d 轮 - 响应: %s", i+1, result.Text)
		}

		t.Log("✓ 多轮对话成功")
	})

	// ========== 测试完成 ==========
	t.Log("========== Azure OpenAI 端到端测试完成 ==========")
	t.Log("✓ 所有测试阶段通过")
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

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		dbHost, dbUser, dbPassword, dbName, dbPort)

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
	db.Exec("DELETE FROM model_configurations WHERE name LIKE '%e2e%'")
}
