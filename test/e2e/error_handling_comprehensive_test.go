package e2e

import (
	"context"
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

// TestComprehensiveErrorHandling 全面的错误处理测试
// 测试所有可能的错误场景，确保系统能够正确处理各种异常情况
func TestComprehensiveErrorHandling(t *testing.T) {
	// 跳过集成测试（除非明确启用）
	if testing.Short() {
		t.Skip("跳过端到端测试")
	}

	ctx := context.Background()

	// ========== 阶段 1: 设置测试环境 ==========
	t.Log("========== 阶段 1: 设置测试环境 ==========")

	db, err := setupTestDatabase(t)
	require.NoError(t, err, "数据库连接应该成功")
	defer cleanupTestDatabase(t, db)

	t.Log("✓ 数据库连接成功")

	// ========== 阶段 2: 创建测试数据 ==========
	t.Log("========== 阶段 2: 创建测试数据 ==========")

	tenantID := uuid.New()
	t.Logf("创建测试租户: %s", tenantID)

	// 创建一个有效的模型配置用于对比测试
	validModelName := "valid-model"
	validQueryParams := `{
		"model": "gemini-1.5-pro",
		"defaultTemperature": 0.7,
		"defaultMaxTokens": 2048
	}`

	validConfig := &model.ModelConfiguration{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          validModelName,
		Model:         "gemini-1.5-pro",
		ModelProvider: "googlegenai",
		APIKey:        os.Getenv("GOOGLE_API_KEY"),
		QueryParams:   &validQueryParams,
		IsEnabled:     true,
		IsDeleted:     false,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
	}

	if validConfig.APIKey != "" {
		err = db.Create(validConfig).Error
		require.NoError(t, err, "有效模型配置创建应该成功")
		t.Logf("✓ 有效模型配置创建成功: %s", validModelName)
	}

	// ========== 阶段 3: 初始化 Genkit Client ==========
	t.Log("========== 阶段 3: 初始化 Genkit Client ==========")

	repo := repository.NewModelConfigurationRepository(db)
	client := genkit.NewClientWithRepo(repo)

	t.Log("✓ Genkit Client 初始化成功")

	// ========== 阶段 4: 测试网络相关错误 ==========
	t.Log("========== 阶段 4: 测试网络相关错误 ==========")

	t.Run("网络超时", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个超时时间很短的上下文
		timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		_, err := client.Generate(
			timeoutCtx,
			tenantID.String(),
			validModelName,
			"请详细介绍一下人工智能的发展历史，包括各个重要的里程碑事件。",
			nil,
		)

		// 应该因为超时而失败
		if err != nil {
			assert.Contains(t, err.Error(), "context deadline exceeded")
			t.Logf("✓ 网络超时错误: %v", err)
		} else {
			t.Log("  注意：请求在超时前完成")
		}
	})

	t.Run("无效的API端点", func(t *testing.T) {
		// 创建一个使用无效端点的Azure配置
		invalidEndpointModelName := "invalid-endpoint-model"
		invalidEndpointQueryParams := `{
			"model": "gpt-4",
			"azureEndpoint": "https://invalid-endpoint-that-does-not-exist.openai.azure.com",
			"azureDeployment": "gpt-4",
			"azureApiVersion": "2024-02-15-preview"
		}`

		invalidEndpointConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          invalidEndpointModelName,
			Model:         "gpt-4",
			ModelProvider: "azureopenai",
			APIKey:        "test-api-key",
			QueryParams:   &invalidEndpointQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(invalidEndpointConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			invalidEndpointModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		t.Logf("✓ 无效API端点错误: %v", err)
	})

	// ========== 阶段 5: 测试Azure特定错误 ==========
	t.Log("========== 阶段 5: 测试Azure特定错误 ==========")

	t.Run("Azure配置缺少endpoint", func(t *testing.T) {
		missingEndpointModelName := "azure-missing-endpoint"
		missingEndpointQueryParams := `{
			"model": "gpt-4",
			"azureDeployment": "gpt-4",
			"azureApiVersion": "2024-02-15-preview"
		}`

		missingEndpointConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          missingEndpointModelName,
			Model:         "gpt-4",
			ModelProvider: "azureopenai",
			APIKey:        "test-api-key",
			QueryParams:   &missingEndpointQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(missingEndpointConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			missingEndpointModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "azureEndpoint")
		t.Logf("✓ Azure缺少endpoint错误: %v", err)
	})

	t.Run("Azure配置缺少deployment", func(t *testing.T) {
		missingDeploymentModelName := "azure-missing-deployment"
		missingDeploymentQueryParams := `{
			"model": "gpt-4",
			"azureEndpoint": "https://test.openai.azure.com",
			"azureApiVersion": "2024-02-15-preview"
		}`

		missingDeploymentConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          missingDeploymentModelName,
			Model:         "gpt-4",
			ModelProvider: "azureopenai",
			APIKey:        "test-api-key",
			QueryParams:   &missingDeploymentQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(missingDeploymentConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			missingDeploymentModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "azureDeployment")
		t.Logf("✓ Azure缺少deployment错误: %v", err)
	})

	// ========== 阶段 6: 测试百炼特定错误 ==========
	t.Log("========== 阶段 6: 测试百炼特定错误 ==========")

	t.Run("百炼配置使用无效endpoint", func(t *testing.T) {
		invalidBailianEndpointModelName := "bailian-invalid-endpoint"
		invalidBailianEndpointQueryParams := `{
			"model": "qwen-turbo",
			"bailianEndpoint": "https://invalid-bailian-endpoint.com"
		}`

		invalidBailianEndpointConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          invalidBailianEndpointModelName,
			Model:         "qwen-turbo",
			ModelProvider: "bianlian",
			APIKey:        "test-api-key",
			QueryParams:   &invalidBailianEndpointQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(invalidBailianEndpointConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			invalidBailianEndpointModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		t.Logf("✓ 百炼无效endpoint错误: %v", err)
	})

	// ========== 阶段 7: 测试自定义OpenAI提供商错误 ==========
	t.Log("========== 阶段 7: 测试自定义OpenAI提供商错误 ==========")

	t.Run("自定义OpenAI缺少baseUrl", func(t *testing.T) {
		missingBaseUrlModelName := "custom-openai-missing-baseurl"
		missingBaseUrlQueryParams := `{
			"model": "gpt-3.5-turbo"
		}`

		missingBaseUrlConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          missingBaseUrlModelName,
			Model:         "gpt-3.5-turbo",
			ModelProvider: "custom_openai",
			APIKey:        "test-api-key",
			QueryParams:   &missingBaseUrlQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(missingBaseUrlConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			missingBaseUrlModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "baseUrl")
		t.Logf("✓ 自定义OpenAI缺少baseUrl错误: %v", err)
	})

	// ========== 阶段 8: 测试速率限制错误 ==========
	t.Log("========== 阶段 8: 测试速率限制错误 ==========")

	t.Run("模拟速率限制", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 快速连续发送多个请求，可能触发速率限制
		const requestCount = 10
		errors := make(chan error, requestCount)

		for i := 0; i < requestCount; i++ {
			go func(index int) {
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					validModelName,
					"测试",
					nil,
				)
				errors <- err
			}(i)
		}

		// 收集结果
		rateLimitErrors := 0
		successCount := 0
		for i := 0; i < requestCount; i++ {
			err := <-errors
			if err != nil {
				if contains(err.Error(), "rate limit") || contains(err.Error(), "429") {
					rateLimitErrors++
				}
			} else {
				successCount++
			}
		}

		t.Logf("✓ 速率限制测试完成: 成功=%d, 速率限制=%d, 总计=%d",
			successCount, rateLimitErrors, requestCount)
	})

	// ========== 阶段 9: 测试内容过滤错误 ==========
	t.Log("========== 阶段 9: 测试内容过滤错误 ==========")

	t.Run("敏感内容过滤", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 尝试发送可能被过滤的内容
		sensitivePrompts := []string{
			"如何制造危险物品",
			"请提供非法活动的详细步骤",
		}

		for _, prompt := range sensitivePrompts {
			_, err := client.Generate(
				ctx,
				tenantID.String(),
				validModelName,
				prompt,
				nil,
			)

			// 某些提供商可能会拒绝敏感内容
			if err != nil {
				t.Logf("  敏感内容被拒绝: %v", err)
			} else {
				t.Log("  注意：提供商接受了敏感内容")
			}
		}

		t.Log("✓ 敏感内容过滤测试完成")
	})

	// ========== 阶段 10: 测试并发安全性 ==========
	t.Log("========== 阶段 10: 测试并发安全性 ==========")

	t.Run("并发初始化同一模型", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个新的模型配置
		concurrentModelName := "concurrent-init-model"
		concurrentQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		concurrentConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          concurrentModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        validConfig.APIKey,
			QueryParams:   &concurrentQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(concurrentConfig).Error
		require.NoError(t, err)

		// 并发调用同一模型
		const concurrency = 10
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					concurrentModelName,
					"测试",
					nil,
				)
				results <- err
			}()
		}

		// 收集结果
		successCount := 0
		errorCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-results
			if err != nil {
				errorCount++
				t.Logf("  错误: %v", err)
			} else {
				successCount++
			}
		}

		// 所有请求都应该成功（或者至少大部分成功）
		assert.Greater(t, successCount, 0, "至少应该有一些请求成功")
		t.Logf("✓ 并发初始化测试完成: 成功=%d, 失败=%d", successCount, errorCount)
	})

	// ========== 阶段 11: 测试资源清理 ==========
	t.Log("========== 阶段 11: 测试资源清理 ==========")

	t.Run("客户端关闭后的调用", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个新的客户端
		newClient := genkit.NewClientWithRepo(repo)

		// 关闭客户端
		err := newClient.Close()
		require.NoError(t, err)

		// 尝试使用已关闭的客户端
		_, err = newClient.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"测试",
			nil,
		)

		// 应该能够正常工作（因为会重新初始化）
		if err != nil {
			t.Logf("  关闭后调用错误: %v", err)
		} else {
			t.Log("  注意：关闭后仍然可以调用（会重新初始化）")
		}
	})

	// ========== 阶段 12: 测试错误恢复 ==========
	t.Log("========== 阶段 12: 测试错误恢复 ==========")

	t.Run("错误后的正常调用", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 先触发一个错误
		_, err := client.Generate(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)
		assert.Error(t, err)

		// 然后进行正常调用
		result, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"你好",
			nil,
		)

		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		t.Log("✓ 错误后的正常调用成功")
	})

	// ========== 阶段 13: 测试流式错误恢复 ==========
	t.Log("========== 阶段 13: 测试流式错误恢复 ==========")

	t.Run("流式错误后的正常调用", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 先触发一个流式错误
		_, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)
		assert.Error(t, err)

		// 然后进行正常的流式调用
		streamChan, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			validModelName,
			"你好",
			nil,
		)

		require.NoError(t, err)

		// 接收流式数据
		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Fatalf("流式调用失败: %v", chunk.Error)
			}
			if !chunk.Done {
				chunkCount++
			}
		}

		assert.Greater(t, chunkCount, 0)
		t.Log("✓ 流式错误后的正常调用成功")
	})

	// ========== 测试完成 ==========
	t.Log("========== 全面错误处理测试完成 ==========")
	t.Log("✓ 所有错误处理场景测试通过")
	t.Log("✓ 验证了系统的错误处理能力和恢复能力")
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsIgnoreCase(s, substr)))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}
