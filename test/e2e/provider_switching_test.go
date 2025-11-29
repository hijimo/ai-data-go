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
)

// TestProviderSwitching 测试提供商切换功能
// 验证系统能够在同一租户下使用不同的模型提供商，并正确切换
func TestProviderSwitching(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过端到端测试")
	}

	// 检查环境变量
	azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureDeployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	bailianAPIKey := os.Getenv("BAILIAN_API_KEY")

	// 至少需要两个提供商的配置才能测试切换
	hasAzure := azureAPIKey != "" && azureEndpoint != "" && azureDeployment != ""
	hasBailian := bailianAPIKey != ""

	if !hasAzure && !hasBailian {
		t.Skip("跳过提供商切换测试：至少需要两个提供商的配置")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	t.Log("✓ 数据库连接成功")

	// ========== 阶段 2: 创建租户和多个模型配置 ==========
	t.Log("========== 阶段 2: 创建租户和多个模型配置 ==========")

	tenantID := uuid.New()
	t.Logf("创建测试租户: %s", tenantID)

	var modelConfigs []*model.ModelConfiguration
	var modelNames []string

	// 创建 Azure OpenAI 配置
	if hasAzure {
		azureAPIVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
		if azureAPIVersion == "" {
			azureAPIVersion = "2024-02-15-preview"
		}

		azureModelName := "azure-gpt-4-switch"
		azureQueryParams := fmt.Sprintf(`{
			"model": "gpt-4",
			"azureEndpoint": "%s",
			"azureDeployment": "%s",
			"azureApiVersion": "%s",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`, azureEndpoint, azureDeployment, azureAPIVersion)

		azureConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          azureModelName,
			Model:         "gpt-4",
			ModelProvider: model.ModelProviderAzureOpenAI,
			APIKey:        azureAPIKey,
			QueryParams:   &azureQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err = db.Create(azureConfig).Error
		require.NoError(t, err, "Azure 模型配置创建应该成功")

		modelConfigs = append(modelConfigs, azureConfig)
		modelNames = append(modelNames, azureModelName)
		t.Logf("✓ Azure OpenAI 配置创建成功: %s", azureModelName)
	}

	// 创建百炼配置
	if hasBailian {
		bailianEndpoint := os.Getenv("BAILIAN_ENDPOINT")
		bailianModel := os.Getenv("BAILIAN_MODEL")
		if bailianEndpoint == "" {
			bailianEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		if bailianModel == "" {
			bailianModel = "qwen-plus"
		}

		bailianModelName := "bailian-qwen-switch"
		bailianQueryParams := fmt.Sprintf(`{
			"model": "%s",
			"bailianEndpoint": "%s",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`, bailianModel, bailianEndpoint)

		bailianConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          bailianModelName,
			Model:         bailianModel,
			ModelProvider: "bianlian",
			APIKey:        bailianAPIKey,
			QueryParams:   &bailianQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err = db.Create(bailianConfig).Error
		require.NoError(t, err, "百炼模型配置创建应该成功")

		modelConfigs = append(modelConfigs, bailianConfig)
		modelNames = append(modelNames, bailianModelName)
		t.Logf("✓ 百炼配置创建成功: %s", bailianModelName)
	}

	require.GreaterOrEqual(t, len(modelConfigs), 2, "至少需要两个模型配置才能测试切换")

	// ========== 阶段 3: 初始化 Genkit Client ==========
	t.Log("========== 阶段 3: 初始化 Genkit Client ==========")

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ Genkit Client 初始化成功")

	// ========== 阶段 4: 测试基本的提供商切换 ==========
	t.Log("========== 阶段 4: 测试基本的提供商切换 ==========")

	t.Run("顺序切换提供商", func(t *testing.T) {
		prompt := "请用一句话介绍你自己。"

		for i, modelName := range modelNames {
			t.Logf("  切换到提供商 %d: %s", i+1, modelName)

			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				prompt,
				nil,
			)

			require.NoError(t, err, "提供商 %s 调用应该成功", modelName)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Text)

			t.Logf("  ✓ 提供商 %s 响应: %s", modelName, result.Text)
		}

		t.Log("✓ 顺序切换提供商成功")
	})

	t.Run("快速切换提供商", func(t *testing.T) {
		// 快速在不同提供商之间切换，测试缓存机制
		for round := 1; round <= 3; round++ {
			t.Logf("  第 %d 轮快速切换", round)

			for _, modelName := range modelNames {
				result, err := client.Generate(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("这是第 %d 轮测试", round),
					nil,
				)

				require.NoError(t, err)
				assert.NotEmpty(t, result.Text)
			}
		}

		t.Log("✓ 快速切换提供商成功")
	})

	// ========== 阶段 5: 测试流式调用的提供商切换 ==========
	t.Log("========== 阶段 5: 测试流式调用的提供商切换 ==========")

	t.Run("流式调用切换提供商", func(t *testing.T) {
		prompt := "请用一句话介绍人工智能。"

		for i, modelName := range modelNames {
			t.Logf("  流式调用提供商 %d: %s", i+1, modelName)

			streamChan, err := client.GenerateStream(
				ctx,
				tenantID.String(),
				modelName,
				prompt,
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

			t.Logf("  ✓ 提供商 %s 流式响应成功，接收 %d 个数据块", modelName, chunkCount)
		}

		t.Log("✓ 流式调用切换提供商成功")
	})

	// ========== 阶段 6: 测试并发切换提供商 ==========
	t.Log("========== 阶段 6: 测试并发切换提供商 ==========")

	t.Run("并发使用不同提供商", func(t *testing.T) {
		const requestsPerProvider = 2
		totalRequests := len(modelNames) * requestsPerProvider

		results := make(chan *genkit.GenerateResult, totalRequests)
		errors := make(chan error, totalRequests)

		// 为每个提供商发起多个并发请求
		for _, modelName := range modelNames {
			for i := 0; i < requestsPerProvider; i++ {
				go func(name string, index int) {
					result, err := client.Generate(
						ctx,
						tenantID.String(),
						name,
						fmt.Sprintf("并发请求 %d", index+1),
						nil,
					)
					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}(modelName, i)
			}
		}

		// 收集结果
		successCount := 0
		for i := 0; i < totalRequests; i++ {
			select {
			case result := <-results:
				assert.NotEmpty(t, result.Text)
				successCount++
			case err := <-errors:
				t.Errorf("并发调用失败: %v", err)
			case <-time.After(60 * time.Second):
				t.Fatal("并发调用超时")
			}
		}

		assert.Equal(t, totalRequests, successCount)
		t.Logf("✓ 并发使用不同提供商成功: %d/%d", successCount, totalRequests)
	})

	// ========== 阶段 7: 测试提供商切换的性能 ==========
	t.Log("========== 阶段 7: 测试提供商切换的性能 ==========")

	t.Run("测量切换延迟", func(t *testing.T) {
		// 预热：确保所有提供商都已初始化
		for _, modelName := range modelNames {
			_, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				"预热",
				nil,
			)
			require.NoError(t, err)
		}

		// 测量切换延迟
		var durations []time.Duration

		for i := 0; i < 3; i++ {
			for _, modelName := range modelNames {
				start := time.Now()
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					modelName,
					fmt.Sprintf("性能测试 %d", i+1),
					nil,
				)
				duration := time.Since(start)

				require.NoError(t, err)
				durations = append(durations, duration)

				t.Logf("  提供商 %s 第 %d 次调用耗时: %v", modelName, i+1, duration)
			}
		}

		// 计算平均延迟
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(durations))

		t.Logf("✓ 平均调用延迟: %v", avgDuration)
		t.Logf("  总调用次数: %d", len(durations))
	})

	// ========== 阶段 8: 测试提供商切换的错误处理 ==========
	t.Log("========== 阶段 8: 测试提供商切换的错误处理 ==========")

	t.Run("切换到不存在的提供商", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-provider",
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 不存在的提供商错误处理正常: %v", err)
	})

	t.Run("禁用一个提供商后切换", func(t *testing.T) {
		if len(modelConfigs) < 2 {
			t.Skip("需要至少两个提供商")
		}

		// 禁用第一个提供商
		firstConfig := modelConfigs[0]
		err := db.Model(&model.ModelConfiguration{}).
			Where("id = ?", firstConfig.ID).
			Update("is_enabled", false).Error
		require.NoError(t, err)

		// 尝试使用被禁用的提供商
		_, err = client.Generate(
			ctx,
			tenantID.String(),
			modelNames[0],
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")
		t.Logf("  ✓ 禁用的提供商错误处理正常: %v", err)

		// 切换到另一个提供商应该成功
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			modelNames[1],
			"测试",
			nil,
		)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		t.Log("  ✓ 切换到其他提供商成功")

		// 恢复第一个提供商
		err = db.Model(&model.ModelConfiguration{}).
			Where("id = ?", firstConfig.ID).
			Update("is_enabled", true).Error
		require.NoError(t, err)
	})

	// ========== 阶段 9: 测试不同提供商的参数传递 ==========
	t.Log("========== 阶段 9: 测试不同提供商的参数传递 ==========")

	t.Run("不同提供商使用不同参数", func(t *testing.T) {
		temperatures := []float64{0.5, 0.8, 1.0}

		for i, modelName := range modelNames {
			temperature := temperatures[i%len(temperatures)]
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

			t.Logf("  ✓ 提供商 %s (temperature=%.1f) 响应成功", modelName, temperature)
		}

		t.Log("✓ 不同提供商使用不同参数成功")
	})

	// ========== 阶段 10: 测试提供商切换的一致性 ==========
	t.Log("========== 阶段 10: 测试提供商切换的一致性 ==========")

	t.Run("相同问题不同提供商的响应", func(t *testing.T) {
		prompt := "什么是人工智能？请用一句话回答。"

		var responses []string

		for _, modelName := range modelNames {
			result, err := client.Generate(
				ctx,
				tenantID.String(),
				modelName,
				prompt,
				nil,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Text)

			responses = append(responses, result.Text)
			t.Logf("  提供商 %s 响应: %s", modelName, result.Text)
		}

		// 验证所有提供商都返回了有效响应
		assert.Equal(t, len(modelNames), len(responses))
		for _, response := range responses {
			assert.NotEmpty(t, response)
		}

		t.Log("✓ 所有提供商都返回了有效响应")
	})

	// ========== 测试完成 ==========
	t.Log("========== 提供商切换测试完成 ==========")
	t.Log("✓ 所有测试阶段通过")
	t.Logf("✓ 测试了 %d 个提供商的切换", len(modelNames))
}
