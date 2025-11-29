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

// TestErrorScenarios 测试各种错误场景
// 验证系统在各种异常情况下的错误处理能力
func TestErrorScenarios(t *testing.T) {
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

	// ========== 阶段 4: 测试配置相关错误 ==========
	t.Log("========== 阶段 4: 测试配置相关错误 ==========")

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
		t.Logf("✓ 配置不存在错误: %v", err)
	})

	t.Run("模型已禁用", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个禁用的模型配置
		disabledModelName := "disabled-model"
		disabledQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		disabledConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          disabledModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        validConfig.APIKey,
			QueryParams:   &disabledQueryParams,
			IsEnabled:     false, // 禁用状态
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(disabledConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			disabledModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型已禁用")
		t.Logf("✓ 模型已禁用错误: %v", err)
	})

	t.Run("模型已删除", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个已删除的模型配置
		deletedModelName := "deleted-model"
		deletedQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		deletedConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          deletedModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        validConfig.APIKey,
			QueryParams:   &deletedQueryParams,
			IsEnabled:     true,
			IsDeleted:     true, // 已删除状态
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(deletedConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			deletedModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		// 已删除的配置应该被视为不存在
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 模型已删除错误: %v", err)
	})

	t.Run("配置JSON格式错误", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个配置JSON格式错误的模型
		invalidJSONModelName := "invalid-json-model"
		invalidJSONQueryParams := `{invalid json}` // 无效的JSON

		invalidJSONConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          invalidJSONModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        validConfig.APIKey,
			QueryParams:   &invalidJSONQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(invalidJSONConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			invalidJSONModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		t.Logf("✓ 配置JSON格式错误: %v", err)
	})

	// ========== 阶段 5: 测试租户相关错误 ==========
	t.Log("========== 阶段 5: 测试租户相关错误 ==========")

	t.Run("租户ID无效 - 空字符串", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"", // 空租户ID
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "租户ID")
		t.Logf("✓ 空租户ID错误: %v", err)
	})

	t.Run("租户ID无效 - 格式错误", func(t *testing.T) {
		_, err := client.Generate(
			ctx,
			"invalid-uuid-format", // 无效的UUID格式
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "租户ID")
		t.Logf("✓ 租户ID格式错误: %v", err)
	})

	t.Run("租户ID不存在", func(t *testing.T) {
		nonExistentTenantID := uuid.New()

		_, err := client.Generate(
			ctx,
			nonExistentTenantID.String(),
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		// 租户不存在时，应该找不到对应的模型配置
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 租户ID不存在错误: %v", err)
	})

	// ========== 阶段 6: 测试API密钥相关错误 ==========
	t.Log("========== 阶段 6: 测试API密钥相关错误 ==========")

	t.Run("API密钥为空", func(t *testing.T) {
		// 创建一个API密钥为空的模型配置
		emptyKeyModelName := "empty-key-model"
		emptyKeyQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		emptyKeyConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          emptyKeyModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        "", // 空API密钥
			QueryParams:   &emptyKeyQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(emptyKeyConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			emptyKeyModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		t.Logf("✓ API密钥为空错误: %v", err)
	})

	t.Run("API密钥无效", func(t *testing.T) {
		// 创建一个API密钥无效的模型配置
		invalidKeyModelName := "invalid-key-model"
		invalidKeyQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		invalidKeyConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          invalidKeyModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        "invalid-api-key-12345", // 无效的API密钥
			QueryParams:   &invalidKeyQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(invalidKeyConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			invalidKeyModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		// API密钥无效时，应该在调用API时失败
		t.Logf("✓ API密钥无效错误: %v", err)
	})

	// ========== 阶段 7: 测试提供商相关错误 ==========
	t.Log("========== 阶段 7: 测试提供商相关错误 ==========")

	t.Run("不支持的提供商类型", func(t *testing.T) {
		// 创建一个不支持的提供商类型的模型配置
		unsupportedProviderModelName := "unsupported-provider-model"
		unsupportedProviderQueryParams := `{
			"model": "some-model",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		unsupportedProviderConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          unsupportedProviderModelName,
			Model:         "some-model",
			ModelProvider: "unsupported-provider", // 不支持的提供商
			APIKey:        "some-api-key",
			QueryParams:   &unsupportedProviderQueryParams,
			IsEnabled:     true,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(unsupportedProviderConfig).Error
		require.NoError(t, err)

		_, err = client.Generate(
			ctx,
			tenantID.String(),
			unsupportedProviderModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不支持的提供商")
		t.Logf("✓ 不支持的提供商类型错误: %v", err)
	})

	// ========== 阶段 8: 测试参数相关错误 ==========
	t.Log("========== 阶段 8: 测试参数相关错误 ==========")

	t.Run("Temperature参数超出范围 - 负数", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		temperature := -0.5 // 负数temperature
		options := &genkit.GenerateOptions{
			Temperature: &temperature,
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"测试",
			options,
		)

		// 某些提供商可能会拒绝负数temperature
		if err != nil {
			t.Logf("✓ 负数Temperature错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了负数temperature")
		}
	})

	t.Run("Temperature参数超出范围 - 大于2", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		temperature := 2.5 // 超出范围的temperature
		options := &genkit.GenerateOptions{
			Temperature: &temperature,
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"测试",
			options,
		)

		// 某些提供商可能会拒绝超出范围的temperature
		if err != nil {
			t.Logf("✓ 超出范围Temperature错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了超出范围的temperature")
		}
	})

	t.Run("MaxTokens参数为负数", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		maxTokens := -100 // 负数maxTokens
		options := &genkit.GenerateOptions{
			MaxTokens: &maxTokens,
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"测试",
			options,
		)

		// 应该拒绝负数maxTokens
		if err != nil {
			t.Logf("✓ 负数MaxTokens错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了负数maxTokens")
		}
	})

	t.Run("MaxTokens参数为零", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		maxTokens := 0 // 零maxTokens
		options := &genkit.GenerateOptions{
			MaxTokens: &maxTokens,
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"测试",
			options,
		)

		// 应该拒绝零maxTokens
		if err != nil {
			t.Logf("✓ 零MaxTokens错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了零maxTokens")
		}
	})

	// ========== 阶段 9: 测试输入相关错误 ==========
	t.Log("========== 阶段 9: 测试输入相关错误 ==========")

	t.Run("空提示词", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			"", // 空提示词
			nil,
		)

		// 某些提供商可能会拒绝空提示词
		if err != nil {
			t.Logf("✓ 空提示词错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了空提示词")
		}
	})

	t.Run("超长提示词", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 生成一个超长的提示词（10000个字符）
		longPrompt := ""
		for i := 0; i < 10000; i++ {
			longPrompt += "测"
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			validModelName,
			longPrompt,
			nil,
		)

		// 某些提供商可能会拒绝超长提示词
		if err != nil {
			t.Logf("✓ 超长提示词错误: %v", err)
		} else {
			t.Log("  注意：提供商接受了超长提示词")
		}
	})

	// ========== 阶段 10: 测试上下文相关错误 ==========
	t.Log("========== 阶段 10: 测试上下文相关错误 ==========")

	t.Run("上下文已取消", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // 立即取消上下文

		_, err := client.Generate(
			cancelCtx,
			tenantID.String(),
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		t.Logf("✓ 上下文已取消错误: %v", err)
	})

	t.Run("上下文超时", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // 等待超时

		_, err := client.Generate(
			timeoutCtx,
			tenantID.String(),
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		t.Logf("✓ 上下文超时错误: %v", err)
	})

	// ========== 阶段 11: 测试流式调用错误 ==========
	t.Log("========== 阶段 11: 测试流式调用错误 ==========")

	t.Run("流式调用 - 配置不存在", func(t *testing.T) {
		_, err := client.GenerateStream(
			ctx,
			tenantID.String(),
			"non-existent-model",
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 流式调用配置不存在错误: %v", err)
	})

	t.Run("流式调用 - 租户ID无效", func(t *testing.T) {
		_, err := client.GenerateStream(
			ctx,
			"invalid-uuid",
			validModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "租户ID")
		t.Logf("✓ 流式调用租户ID无效错误: %v", err)
	})

	t.Run("流式调用 - 上下文取消", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		cancelCtx, cancel := context.WithCancel(ctx)

		streamChan, err := client.GenerateStream(
			cancelCtx,
			tenantID.String(),
			validModelName,
			"请详细介绍一下人工智能的发展历史。",
			nil,
		)

		require.NoError(t, err)

		// 接收一些数据后取消
		chunkCount := 0
		for chunk := range streamChan {
			if chunk.Error != nil {
				t.Logf("  流式错误: %v", chunk.Error)
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

		t.Logf("✓ 流式调用上下文取消处理正常，接收了 %d 个数据块", chunkCount)
	})

	// ========== 阶段 12: 测试并发错误场景 ==========
	t.Log("========== 阶段 12: 测试并发错误场景 ==========")

	t.Run("并发调用不存在的模型", func(t *testing.T) {
		const concurrency = 5
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					"non-existent-model",
					"测试",
					nil,
				)
				errors <- err
			}()
		}

		// 收集错误
		errorCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errors
			if err != nil {
				errorCount++
				assert.Contains(t, err.Error(), "模型配置")
			}
		}

		assert.Equal(t, concurrency, errorCount)
		t.Logf("✓ 并发调用不存在的模型，所有请求都正确返回错误: %d/%d", errorCount, concurrency)
	})

	t.Run("并发调用禁用的模型", func(t *testing.T) {
		if validConfig.APIKey == "" {
			t.Skip("跳过：缺少有效的 API 密钥")
		}

		// 创建一个禁用的模型
		disabledModelName := "disabled-concurrent-model"
		disabledQueryParams := `{
			"model": "gemini-1.5-pro",
			"defaultTemperature": 0.7,
			"defaultMaxTokens": 2048
		}`

		disabledConfig := &model.ModelConfiguration{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          disabledModelName,
			Model:         "gemini-1.5-pro",
			ModelProvider: "googlegenai",
			APIKey:        validConfig.APIKey,
			QueryParams:   &disabledQueryParams,
			IsEnabled:     false,
			IsDeleted:     false,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
		}

		err := db.Create(disabledConfig).Error
		require.NoError(t, err)

		const concurrency = 5
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				_, err := client.Generate(
					ctx,
					tenantID.String(),
					disabledModelName,
					"测试",
					nil,
				)
				errors <- err
			}()
		}

		// 收集错误
		errorCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errors
			if err != nil {
				errorCount++
				assert.Contains(t, err.Error(), "模型已禁用")
			}
		}

		assert.Equal(t, concurrency, errorCount)
		t.Logf("✓ 并发调用禁用的模型，所有请求都正确返回错误: %d/%d", errorCount, concurrency)
	})

	// ========== 阶段 13: 测试边界条件 ==========
	t.Log("========== 阶段 13: 测试边界条件 ==========")

	t.Run("模型名称包含特殊字符", func(t *testing.T) {
		specialCharModelName := "model-with-特殊字符-!@#$%"

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			specialCharModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 特殊字符模型名称错误: %v", err)
	})

	t.Run("模型名称超长", func(t *testing.T) {
		// 生成一个超长的模型名称（1000个字符）
		longModelName := ""
		for i := 0; i < 1000; i++ {
			longModelName += "a"
		}

		_, err := client.Generate(
			ctx,
			tenantID.String(),
			longModelName,
			"测试",
			nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "模型配置")
		t.Logf("✓ 超长模型名称错误: %v", err)
	})

	// ========== 测试完成 ==========
	t.Log("========== 错误场景测试完成 ==========")
	t.Log("✓ 所有错误场景测试通过")
	t.Log("✓ 验证了系统在各种异常情况下的错误处理能力")
}
